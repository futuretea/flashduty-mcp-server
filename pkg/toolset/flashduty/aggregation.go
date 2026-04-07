package flashduty

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/futuretea/flashduty-mcp-server/pkg/core/logging"
)

// AggregateRequest holds parameters for an incident aggregation query.
type AggregateRequest struct {
	ChannelID      int      `json:"channel_id"`
	ChannelName    string   `json:"channel_name"`
	StartTime      int64    `json:"start_time"`
	EndTime        int64    `json:"end_time"`
	Severities     []string `json:"severities"`
	Query          string   `json:"query"`
	GroupBy        []string `json:"group_by"`
	IncludeDetails bool     `json:"include_details"`
	DetailFields   []string `json:"detail_fields"`
	MaxIncidents   int      `json:"max_incidents"`
}

// AggregateResult is the output of an incident aggregation.
type AggregateResult struct {
	TotalIncidents int              `json:"total_incidents"`
	TotalGroups    int              `json:"total_groups"`
	GroupBy        []string         `json:"group_by"`
	Groups         []AggregateGroup `json:"groups"`
}

// AggregateGroup represents a single dimension-grouped result bucket.
type AggregateGroup struct {
	Key     map[string]string        `json:"key"`
	Count   int                      `json:"count"`
	Details []map[string]interface{} `json:"details,omitempty"`
}

// aggregateGroupEntry is an internal accumulator that pairs a group with its composite key.
type aggregateGroupEntry struct {
	group        *AggregateGroup
	compositeKey string
}

// defaultDetailFields are the incident fields returned when include_details=true and no detail_fields are specified.
var defaultDetailFields = []string{
	"incident_id", "title", "incident_severity", "progress",
	"created_at", "channel_name", "channel_id",
}

// defaultMaxIncidents is the upper bound on incidents fetched per aggregation to prevent excessive memory usage.
const defaultMaxIncidents = 500

// extractDimensionValue extracts the value of a grouping dimension from an incident object.
// Built-in dimensions: "severity"/"incident_severity", "channel", "progress".
// Custom label dimensions use the "labels.<key>" format (e.g., "labels.env").
// Returns "(empty)" when the value is absent or blank.
func extractDimensionValue(incident map[string]any, dim string) string {
	switch dim {
	case "severity", "incident_severity":
		if v, ok := incident["incident_severity"].(string); ok && v != "" {
			return v
		}
		return "(empty)"
	case "channel":
		// Prefer channel_name; fall back to a formatted channel_id.
		if v, ok := incident["channel_name"].(string); ok && v != "" {
			return v
		}
		if id := toInt(incident["channel_id"]); id != 0 {
			return fmt.Sprintf("channel_%d", id)
		}
		return "(empty)"
	case "progress":
		if v, ok := incident["progress"].(string); ok && v != "" {
			return v
		}
		return "(empty)"
	default:
		// Support "labels.<key>" for custom label dimensions.
		if strings.HasPrefix(dim, "labels.") {
			labelKey := dim[len("labels."):]
			labels, ok := incident["labels"].(map[string]any)
			if !ok {
				return "(empty)"
			}
			if v, ok := labels[labelKey].(string); ok && v != "" {
				return v
			}
			return "(empty)"
		}
		// Generic string field lookup.
		if v, ok := incident[dim].(string); ok && v != "" {
			return v
		}
		return "(empty)"
	}
}

// buildCompositeKey joins the values of all grouping dimensions with a NUL separator
// to produce a unique map key for the given incident.
func buildCompositeKey(incident map[string]any, groupBy []string) string {
	parts := make([]string, len(groupBy))
	for i, dim := range groupBy {
		parts[i] = extractDimensionValue(incident, dim)
	}
	return strings.Join(parts, "\x00")
}

// buildGroupKey constructs the key map that appears in the result output for a group.
func buildGroupKey(incident map[string]any, groupBy []string) map[string]string {
	key := make(map[string]string, len(groupBy))
	for _, dim := range groupBy {
		key[dim] = extractDimensionValue(incident, dim)
	}
	return key
}

// extractDetailFields picks the requested fields from an incident and returns them
// as a detail record. Falls back to defaultDetailFields when fields is empty.
// Fields using the "labels.<key>" format are extracted from the incident's nested
// labels map, mirroring the behaviour of extractDimensionValue.
func extractDetailFields(incident map[string]any, fields []string) map[string]interface{} {
	if len(fields) == 0 {
		fields = defaultDetailFields
	}
	detail := make(map[string]interface{}, len(fields))
	for _, f := range fields {
		if strings.HasPrefix(f, "labels.") {
			// Extract from the nested labels map.
			labelKey := f[len("labels."):]
			if labels, ok := incident["labels"].(map[string]any); ok {
				if v, ok := labels[labelKey]; ok {
					detail[f] = v
				}
			}
		} else {
			if v, ok := incident[f]; ok {
				detail[f] = v
			}
		}
	}
	return detail
}

// fetchAllIncidents fetches all incidents matching req using cursor-based pagination
// (100 items per page). It stops after collecting req.MaxIncidents incidents.
func fetchAllIncidents(c *Client, req *AggregateRequest) ([]map[string]any, error) {
	maxIncidents := req.MaxIncidents
	if maxIncidents <= 0 {
		maxIncidents = defaultMaxIncidents
	}

	var (
		allIncidents   []map[string]any
		searchAfterCtx string
	)

	for {
		body := buildIncidentListRequest(req, searchAfterCtx)
		items, nextCtx, err := fetchIncidentPage(c, body)
		if err != nil {
			return nil, err
		}
		if len(items) == 0 {
			break
		}

		allIncidents = appendIncidentItems(allIncidents, items, maxIncidents)
		if len(allIncidents) >= maxIncidents {
			logging.Info("Reached max incident limit (%d), stopping pagination", maxIncidents)
			return allIncidents, nil
		}

		if nextCtx == "" {
			break
		}
		searchAfterCtx = nextCtx
	}

	return allIncidents, nil
}

// buildIncidentListRequest constructs the request body for incident list API.
func buildIncidentListRequest(req *AggregateRequest, searchAfterCtx string) map[string]any {
	body := map[string]any{
		"start_time": req.StartTime,
		"end_time":   req.EndTime,
		"limit":      100,
	}
	if req.ChannelID != 0 {
		body["channel_ids"] = []int{req.ChannelID}
	}
	if len(req.Severities) > 0 {
		body["incident_severity"] = strings.Join(req.Severities, ",")
	}
	if req.Query != "" {
		body["title"] = req.Query
	}
	if searchAfterCtx != "" {
		body["search_after_ctx"] = searchAfterCtx
	}
	return body
}

// fetchIncidentPage fetches a single page of incidents and returns items with next cursor.
func fetchIncidentPage(c *Client, body map[string]any) ([]map[string]any, string, error) {
	resp, err := c.doPost("/incident/list", body)
	if err != nil {
		return nil, "", fmt.Errorf("failed to fetch incidents page: %w", err)
	}
	if resp.Error != nil && resp.Error.Code != "" {
		return nil, "", fmt.Errorf("API error [%s]: %s", resp.Error.Code, resp.Error.Message)
	}
	if resp.Data == nil {
		return nil, "", nil
	}

	var data map[string]any
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		return nil, "", fmt.Errorf("failed to parse response: %w", err)
	}

	items, ok := data["items"].([]any)
	if !ok || len(items) == 0 {
		return nil, "", nil
	}

	result := make([]map[string]any, 0, len(items))
	for _, item := range items {
		if obj, ok := item.(map[string]any); ok {
			result = append(result, obj)
		}
	}

	nextCtx, _ := data["search_after_ctx"].(string)
	hasNextPage, _ := data["has_next_page"].(bool)
	if !hasNextPage {
		nextCtx = ""
	}

	return result, nextCtx, nil
}

// appendIncidentItems appends items to the collection, respecting the max limit.
func appendIncidentItems(all []map[string]any, items []map[string]any, max int) []map[string]any {
	for _, item := range items {
		all = append(all, item)
		if len(all) >= max {
			break
		}
	}
	return all
}

// AggregateIncidents is the core aggregation function.
// It fetches all matching incidents via pagination, groups them by the requested
// dimensions, optionally collects per-incident detail records, and returns the
// groups sorted by their composite key in lexicographic order.
func AggregateIncidents(c *Client, req *AggregateRequest) (*AggregateResult, error) {
	if req.StartTime == 0 || req.EndTime == 0 {
		return nil, fmt.Errorf("start_time and end_time are required")
	}
	if len(req.GroupBy) == 0 {
		return nil, fmt.Errorf("group_by must contain at least one dimension")
	}

	incidents, err := fetchAllIncidents(c, req)
	if err != nil {
		return nil, err
	}

	logging.Info("Fetched %d incidents, starting aggregation (group_by=%v)", len(incidents), req.GroupBy)

	// Group incidents by composite key, preserving insertion order for stable sorting.
	groupMap := make(map[string]*aggregateGroupEntry)
	var compositeKeys []string

	for _, incident := range incidents {
		ck := buildCompositeKey(incident, req.GroupBy)

		entry, exists := groupMap[ck]
		if !exists {
			group := &AggregateGroup{
				Key: buildGroupKey(incident, req.GroupBy),
			}
			if req.IncludeDetails {
				group.Details = make([]map[string]interface{}, 0)
			}
			entry = &aggregateGroupEntry{group: group, compositeKey: ck}
			groupMap[ck] = entry
			compositeKeys = append(compositeKeys, ck)
		}

		entry.group.Count++
		if req.IncludeDetails {
			detail := extractDetailFields(incident, req.DetailFields)
			AddTimestampDisplay(detail, c.Location)
			entry.group.Details = append(entry.group.Details, detail)
		}
	}

	// Sort groups lexicographically by composite key for deterministic output.
	sort.Strings(compositeKeys)

	groups := make([]AggregateGroup, 0, len(compositeKeys))
	for _, ck := range compositeKeys {
		groups = append(groups, *groupMap[ck].group)
	}

	return &AggregateResult{
		TotalIncidents: len(incidents),
		TotalGroups:    len(groups),
		GroupBy:        req.GroupBy,
		Groups:         groups,
	}, nil
}

// resolveChannelIDByName looks up a channel ID by name using a fuzzy API search.
// It first tries a case-insensitive exact match; if none is found and there is
// exactly one result, that result is used. Returns an error when the name is
// ambiguous or not found.
func resolveChannelIDByName(c *Client, channelName string) (int, error) {
	data, err := c.DoRequestRaw("/channel/list", map[string]any{
		"query": channelName,
		"limit": 50,
	})
	if err != nil {
		return 0, fmt.Errorf("failed to query channel: %w", err)
	}

	if data == nil {
		return 0, fmt.Errorf("channel not found with name %q", channelName)
	}
	items, ok := data["items"].([]any)
	if !ok || len(items) == 0 {
		return 0, fmt.Errorf("channel not found with name %q", channelName)
	}

	// Prefer a case-insensitive exact match.
	lowerName := strings.ToLower(channelName)
	for _, item := range items {
		obj, ok := item.(map[string]any)
		if !ok {
			continue
		}
		name, _ := obj["channel_name"].(string)
		if strings.ToLower(name) == lowerName {
			if id := toInt(obj["channel_id"]); id != 0 {
				return id, nil
			}
		}
	}

	// Fall back to the single result when there is no exact match.
	if len(items) == 1 {
		obj, ok := items[0].(map[string]any)
		if !ok {
			return 0, fmt.Errorf("channel not found with name %q", channelName)
		}
		if id := toInt(obj["channel_id"]); id != 0 {
			return id, nil
		}
	}

	return 0, fmt.Errorf("channel name %q matched multiple results; provide channel_id or a more specific name", channelName)
}
