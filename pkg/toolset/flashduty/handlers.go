package flashduty

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// getClient validates and returns the FlashDuty client from the generic client.
func getClient(client any) (*Client, error) {
	c, ok := client.(*Client)
	if !ok || c == nil {
		return nil, fmt.Errorf("FlashDuty client is not configured")
	}
	return c, nil
}

func getStringParam(m map[string]any, key string) string {
	v, ok := m[key]
	if !ok {
		return ""
	}
	s, _ := v.(string)
	return s
}

func getNumberParam(m map[string]any, key string) (float64, bool) {
	v, ok := m[key]
	if !ok {
		return 0, false
	}
	switch n := v.(type) {
	case float64:
		return n, true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	default:
		return 0, false
	}
}

func getIntParam(m map[string]any, key string) (int, bool) {
	n, ok := getNumberParam(m, key)
	if !ok {
		return 0, false
	}
	return int(n), true
}

func getStringSliceParam(m map[string]any, key string) []string {
	v, ok := m[key]
	if !ok {
		return nil
	}

	switch arr := v.(type) {
	case []any:
		result := make([]string, 0, len(arr))
		for _, item := range arr {
			if s, ok := item.(string); ok {
				result = append(result, s)
			}
		}
		return result
	case []string:
		return arr
	default:
		return nil
	}
}

func getIntSliceParam(m map[string]any, key string) []int {
	v, ok := m[key]
	if !ok {
		return nil
	}

	arr, ok := v.([]any)
	if !ok {
		return nil
	}

	result := make([]int, 0, len(arr))
	for _, item := range arr {
		switch n := item.(type) {
		case float64:
			result = append(result, int(n))
		case int:
			result = append(result, n)
		case int64:
			result = append(result, int(n))
		}
	}
	return result
}

func getBoolParam(m map[string]any, key string) (bool, bool) {
	v, ok := m[key]
	if !ok {
		return false, false
	}
	b, ok := v.(bool)
	return b, ok
}

func getMapParam(m map[string]any, key string) map[string]any {
	v, ok := m[key]
	if !ok {
		return nil
	}
	result, _ := v.(map[string]any)
	return result
}

// setOptionalString copies a non-empty string param from params to body.
func setOptionalString(params, body map[string]any, key string) {
	if v := getStringParam(params, key); v != "" {
		body[key] = v
	}
}

// setOptionalIntSlice copies a non-empty int slice param from params to body.
func setOptionalIntSlice(params, body map[string]any, key string) {
	if v := getIntSliceParam(params, key); len(v) > 0 {
		body[key] = v
	}
}

// setOptionalStringSlice copies a non-empty string slice param from params to body.
func setOptionalStringSlice(params, body map[string]any, key string) {
	if v := getStringSliceParam(params, key); len(v) > 0 {
		body[key] = v
	}
}

// setOptionalMap copies a non-empty map param from params to body.
func setOptionalMap(params, body map[string]any, key string) {
	if v := getMapParam(params, key); len(v) > 0 {
		body[key] = v
	}
}

// lastDayRange returns (00:00:00, 23:59:59) of yesterday in local time.
func lastDayRange() (int64, int64) {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	start := today.AddDate(0, 0, -1)
	end := today.Add(-time.Second)
	return start.Unix(), end.Unix()
}

// currentWeekMonday returns the Monday 00:00:00 of the current ISO week in local time.
func currentWeekMonday() time.Time {
	now := time.Now()
	wd := int(now.Weekday())
	if wd == 0 {
		wd = 7 // ISO: Sunday = 7
	}
	return time.Date(now.Year(), now.Month(), now.Day()-wd+1, 0, 0, 0, 0, now.Location())
}

// lastWeekRange returns (Monday 00:00:00, Sunday 23:59:59) of the previous week in local time.
func lastWeekRange() (int64, int64) {
	thisMonday := currentWeekMonday()
	start := thisMonday.AddDate(0, 0, -7)
	end := thisMonday.Add(-time.Second)
	return start.Unix(), end.Unix()
}

// weekBeforeLastRange returns (Monday 00:00:00, Sunday 23:59:59) of the week before last week in local time.
func weekBeforeLastRange() (int64, int64) {
	thisMonday := currentWeekMonday()
	start := thisMonday.AddDate(0, 0, -14)
	end := thisMonday.AddDate(0, 0, -7).Add(-time.Second)
	return start.Unix(), end.Unix()
}

// parseTimeRange parses a relative time range string and returns (startTime, endTime) as unix seconds.
// Supported formats:
//   - Duration: "1h", "24h", "7d", "30d", "1w", "6M" (end time = now)
//   - Named:    "last_day" (yesterday 00:00:00 to 23:59:59)
//     "last_week" (Monday 00:00:00 to Sunday 23:59:59 of previous week)
//     "week_before_last" (Monday 00:00:00 to Sunday 23:59:59 of the week before last week)
func parseTimeRange(tr string) (int64, int64, error) {
	tr = strings.TrimSpace(tr)
	if tr == "" {
		return 0, 0, fmt.Errorf("time_range is empty")
	}

	// Named ranges
	if start, end, ok := parseNamedTimeRange(tr); ok {
		return start, end, nil
	}

	// Duration-based ranges
	return parseDurationTimeRange(tr)
}

// parseNamedTimeRange handles named time ranges like "last_day", "last_week".
func parseNamedTimeRange(tr string) (int64, int64, bool) {
	switch strings.ToLower(tr) {
	case "last_day":
		s, e := lastDayRange()
		return s, e, true
	case "last_week":
		s, e := lastWeekRange()
		return s, e, true
	case "week_before_last":
		s, e := weekBeforeLastRange()
		return s, e, true
	}
	return 0, 0, false
}

// parseDurationTimeRange handles duration-based ranges like "24h", "7d".
func parseDurationTimeRange(tr string) (int64, int64, error) {
	unit := tr[len(tr)-1]
	numStr := tr[:len(tr)-1]
	n, err := strconv.Atoi(numStr)
	if err != nil || n <= 0 {
		return 0, 0, fmt.Errorf("invalid time_range %q: must be a duration (e.g., '24h', '7d') or a named range ('last_day', 'last_week', 'week_before_last')", tr)
	}

	dur, err := durationFromUnit(n, unit)
	if err != nil {
		return 0, 0, err
	}

	now := time.Now()
	return now.Add(-dur).Unix(), now.Unix(), nil
}

// durationFromUnit converts a numeric value and unit character to a time.Duration.
func durationFromUnit(n int, unit byte) (time.Duration, error) {
	var unitDuration time.Duration
	switch unit {
	case 'h':
		unitDuration = time.Hour
	case 'd':
		unitDuration = 24 * time.Hour
	case 'w':
		unitDuration = 7 * 24 * time.Hour
	case 'M':
		unitDuration = 30 * 24 * time.Hour
	default:
		return 0, fmt.Errorf("invalid time_range unit %q: must be h, d, w, or M", string(unit))
	}

	const maxDuration = time.Duration(1<<63 - 1)
	if int64(n) > int64(maxDuration/unitDuration) {
		return 0, fmt.Errorf("time_range %d%c is too large", n, unit)
	}
	return time.Duration(n) * unitDuration, nil
}

// resolveTimeParams extracts start/end time from params, supporting both
// time_range (e.g., "24h") and explicit start_time/end_time (unix seconds or ISO 8601 string).
func resolveTimeParams(params map[string]any) (int64, int64, error) {
	if tr := getStringParam(params, "time_range"); tr != "" {
		return parseTimeRange(tr)
	}
	sv, hasStart := params["start_time"]
	if !hasStart {
		return 0, 0, fmt.Errorf("either time_range or start_time+end_time is required")
	}
	startTime, err := ParseFlexibleTime(sv)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid start_time: %w", err)
	}
	ev, hasEnd := params["end_time"]
	if !hasEnd {
		return 0, 0, fmt.Errorf("either time_range or start_time+end_time is required")
	}
	endTime, err := ParseFlexibleTime(ev)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid end_time: %w", err)
	}
	return startTime, endTime, nil
}

const maxListTimeWindow = 31 * 24 * time.Hour

func resolveListTimeParams(params map[string]any) (int64, int64, error) {
	startTime, endTime, err := resolveTimeParams(params)
	if err != nil {
		return 0, 0, err
	}
	if endTime <= startTime {
		return 0, 0, fmt.Errorf("end_time must be after start_time")
	}
	if endTime-startTime > int64(maxListTimeWindow/time.Second) {
		return 0, 0, fmt.Errorf("time window must not exceed 31 days")
	}
	return startTime, endTime, nil
}

// buildPaginationParams adds common pagination parameters to the request body.
func buildPaginationParams(params map[string]any, body map[string]any) {
	if limit, ok := getIntParam(params, "limit"); ok {
		body["limit"] = limit
	}
	if p, ok := getIntParam(params, "p"); ok {
		body["p"] = p
	}
}

// doAction executes an API action and returns a friendly success message when the API returns "OK".
func doAction(c *Client, path string, body map[string]any, successMsg string) (string, error) {
	result, err := c.DoRequest(path, body)
	if err != nil {
		return "", err
	}
	if result == "OK" {
		return successMsg, nil
	}
	return result, nil
}

// doQueryList executes a list API call with optional query string and pagination.
func doQueryList(c *Client, path string, params map[string]any) (string, error) {
	body := map[string]any{}
	if query := getStringParam(params, "query"); query != "" {
		body["query"] = query
	}
	buildPaginationParams(params, body)
	return c.DoRequest(path, body)
}

// getClientLocation safely extracts the Location from a client, returning time.UTC on failure.
func getClientLocation(client any) *time.Location {
	if c, ok := client.(*Client); ok && c != nil && c.Location != nil {
		return c.Location
	}
	return time.UTC
}

// injectTimestampDisplayToJSON parses jsonStr, injects _display fields for timestamps,
// and re-serializes. Falls back to the original string on any error.
func injectTimestampDisplayToJSON(jsonStr string, loc *time.Location) string {
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		return jsonStr
	}
	if items, ok := data["items"].([]interface{}); ok {
		AddTimestampDisplayToList(items, loc)
	}
	AddTimestampDisplay(data, loc)
	out, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return jsonStr
	}
	return string(out)
}

// ===== Incident Handlers =====

// incidentBriefFields defines the fields to include in brief mode for incidents.
var incidentBriefFields = []string{
	"incident_id", "title", "incident_severity", "progress",
	"created_at", "channel_name", "channel_id", "num",
}

// alertBriefFields defines the fields to include in brief mode for alerts.
var alertBriefFields = []string{
	"alert_id", "title", "alert_severity", "alert_status",
	"created_at", "channel_name", "channel_id",
}

func handleListIncidents(client any, params map[string]any) (string, error) {
	c, err := getClient(client)
	if err != nil {
		return "", err
	}

	startTime, endTime, err := resolveListTimeParams(params)
	if err != nil {
		return "", err
	}

	body := map[string]any{
		"start_time": startTime,
		"end_time":   endTime,
	}

	setOptionalString(params, body, "progress")
	setOptionalString(params, body, "incident_severity")
	setOptionalString(params, body, "query")
	setOptionalIntSlice(params, body, "channel_ids")
	setOptionalIntSlice(params, body, "responder_ids")
	setOptionalIntSlice(params, body, "acker_ids")
	setOptionalIntSlice(params, body, "creator_ids")
	buildPaginationParams(params, body)

	result, err := c.DoRequest("/incident/list", body)
	if err != nil {
		return "", err
	}

	if brief, ok := getBoolParam(params, "brief"); ok && brief {
		return injectTimestampDisplayToJSON(filterBriefList(result, incidentBriefFields), c.Location), nil
	}
	return enrichIncidentList(c, result), nil
}

func handleGetIncident(client any, params map[string]any) (string, error) {
	c, err := getClient(client)
	if err != nil {
		return "", err
	}

	incidentID := getStringParam(params, "incident_id")
	if incidentID == "" {
		return "", fmt.Errorf("incident_id is required")
	}

	body := map[string]any{
		"incident_id": incidentID,
	}

	result, err := c.DoRequest("/incident/info", body)
	if err != nil {
		return "", err
	}
	return enrichIncidentDetail(c, result), nil
}

func handleCreateIncident(client any, params map[string]any) (string, error) {
	c, err := getClient(client)
	if err != nil {
		return "", err
	}

	severity := getStringParam(params, "incident_severity")
	if severity == "" {
		return "", fmt.Errorf("incident_severity is required")
	}

	body := map[string]any{
		"incident_severity": severity,
	}

	setOptionalString(params, body, "title")
	setOptionalString(params, body, "description")
	if channelID, ok := getIntParam(params, "channel_id"); ok {
		body["channel_id"] = channelID
	}

	return c.DoRequest("/incident/create", body)
}

func handleAckIncidents(client any, params map[string]any) (string, error) {
	c, err := getClient(client)
	if err != nil {
		return "", err
	}

	incidentIDs := getStringSliceParam(params, "incident_ids")
	if len(incidentIDs) == 0 {
		return "", fmt.Errorf("incident_ids is required and must not be empty")
	}

	body := map[string]any{"incident_ids": incidentIDs}
	return doAction(c, "/incident/ack", body,
		fmt.Sprintf("Successfully acknowledged %d incident(s)", len(incidentIDs)))
}

func handleResolveIncidents(client any, params map[string]any) (string, error) {
	c, err := getClient(client)
	if err != nil {
		return "", err
	}

	incidentIDs := getStringSliceParam(params, "incident_ids")
	if len(incidentIDs) == 0 {
		return "", fmt.Errorf("incident_ids is required and must not be empty")
	}

	body := map[string]any{"incident_ids": incidentIDs}
	setOptionalString(params, body, "root_cause")
	setOptionalString(params, body, "resolution")

	return doAction(c, "/incident/resolve", body,
		fmt.Sprintf("Successfully resolved %d incident(s)", len(incidentIDs)))
}

func handleReopenIncidents(client any, params map[string]any) (string, error) {
	c, err := getClient(client)
	if err != nil {
		return "", err
	}

	incidentIDs := getStringSliceParam(params, "incident_ids")
	if len(incidentIDs) == 0 {
		return "", fmt.Errorf("incident_ids is required and must not be empty")
	}
	body := map[string]any{
		"incident_ids": incidentIDs,
	}
	setOptionalString(params, body, "reason")
	return doAction(c, "/incident/reopen", body,
		fmt.Sprintf("Successfully reopened %d incident(s)", len(incidentIDs)))
}

func handleSnoozeIncidents(client any, params map[string]any) (string, error) {
	c, err := getClient(client)
	if err != nil {
		return "", err
	}

	incidentIDs := getStringSliceParam(params, "incident_ids")
	if len(incidentIDs) == 0 {
		return "", fmt.Errorf("incident_ids is required and must not be empty")
	}
	minutes, ok := getIntParam(params, "minutes")
	if !ok {
		return "", fmt.Errorf("minutes is required")
	}
	if minutes < 1 || minutes > 1440 {
		return "", fmt.Errorf("minutes must be between 1 and 1440")
	}

	body := map[string]any{
		"incident_ids": incidentIDs,
		"minutes":      minutes,
	}
	return doAction(c, "/incident/snooze", body,
		fmt.Sprintf("Successfully snoozed %d incident(s) for %d minutes", len(incidentIDs), minutes))
}

func handleCommentIncidents(client any, params map[string]any) (string, error) {
	c, err := getClient(client)
	if err != nil {
		return "", err
	}

	incidentIDs := getStringSliceParam(params, "incident_ids")
	if len(incidentIDs) == 0 {
		return "", fmt.Errorf("incident_ids is required and must not be empty")
	}
	body := map[string]any{
		"incident_ids": incidentIDs,
	}
	setOptionalString(params, body, "comment")
	return doAction(c, "/incident/comment", body,
		fmt.Sprintf("Successfully commented on %d incident(s)", len(incidentIDs)))
}

// ===== Alert Handlers =====

func handleListAlerts(client any, params map[string]any) (string, error) {
	c, err := getClient(client)
	if err != nil {
		return "", err
	}

	startTime, endTime, err := resolveListTimeParams(params)
	if err != nil {
		return "", err
	}

	body := map[string]any{
		"start_time": startTime,
		"end_time":   endTime,
	}

	setOptionalString(params, body, "alert_severity")
	if isActive, ok := getBoolParam(params, "is_active"); ok {
		body["is_active"] = isActive
	}
	setOptionalIntSlice(params, body, "channel_ids")
	buildPaginationParams(params, body)

	result, err := c.DoRequest("/alert/list", body)
	if err != nil {
		return "", err
	}

	if brief, ok := getBoolParam(params, "brief"); ok && brief {
		return injectTimestampDisplayToJSON(filterBriefList(result, alertBriefFields), c.Location), nil
	}
	return injectTimestampDisplayToJSON(result, c.Location), nil
}

func handleGetAlert(client any, params map[string]any) (string, error) {
	c, err := getClient(client)
	if err != nil {
		return "", err
	}

	alertID := getStringParam(params, "alert_id")
	if alertID == "" {
		return "", fmt.Errorf("alert_id is required")
	}

	body := map[string]any{
		"alert_id": alertID,
	}

	result, err := c.DoRequest("/alert/info", body)
	if err != nil {
		return "", err
	}
	return injectTimestampDisplayToJSON(result, c.Location), nil
}

// ===== Channel Handlers =====

func handleListChannels(client any, params map[string]any) (string, error) {
	c, err := getClient(client)
	if err != nil {
		return "", err
	}
	return doQueryList(c, "/channel/list", params)
}

func handleGetChannel(client any, params map[string]any) (string, error) {
	c, err := getClient(client)
	if err != nil {
		return "", err
	}

	channelID, ok := getIntParam(params, "channel_id")
	if !ok {
		return "", fmt.Errorf("channel_id is required")
	}

	body := map[string]any{
		"channel_id": channelID,
	}

	return c.DoRequest("/channel/info", body)
}

// ===== Team Handlers =====

func handleListTeams(client any, params map[string]any) (string, error) {
	c, err := getClient(client)
	if err != nil {
		return "", err
	}
	return doQueryList(c, "/team/list", params)
}

// ===== Member Handlers =====

func handleListMembers(client any, params map[string]any) (string, error) {
	c, err := getClient(client)
	if err != nil {
		return "", err
	}
	return doQueryList(c, "/member/list", params)
}

// ===== Schedule Handlers =====

func handleListSchedules(client any, params map[string]any) (string, error) {
	c, err := getClient(client)
	if err != nil {
		return "", err
	}

	body := map[string]any{}

	setOptionalIntSlice(params, body, "team_ids")
	setOptionalString(params, body, "query")
	buildPaginationParams(params, body)

	return c.DoRequest("/schedule/list", body)
}

// ===== Insight Handlers =====

func handleGetIncidentStats(client any, params map[string]any) (string, error) {
	c, err := getClient(client)
	if err != nil {
		return "", err
	}

	startTime, endTime, err := resolveTimeParams(params)
	if err != nil {
		return "", err
	}

	body := map[string]any{
		"start_time": startTime,
		"end_time":   endTime,
		"query":      "",
		"labels":     map[string]any{},
	}

	setOptionalIntSlice(params, body, "channel_ids")
	setOptionalIntSlice(params, body, "team_ids")
	setOptionalStringSlice(params, body, "severities")
	setOptionalString(params, body, "aggregate_unit")
	setOptionalString(params, body, "query")
	setOptionalMap(params, body, "labels")

	raw, err := c.DoRequest("/insight/account", body)
	if err != nil {
		return "", err
	}

	// Inject time_range_display and timestamp display fields into response.
	loc := getClientLocation(client)
	var result map[string]interface{}
	if jsonErr := json.Unmarshal([]byte(raw), &result); jsonErr != nil {
		// Fallback: return raw response if JSON parsing fails.
		return raw, nil
	}
	AddTimestampDisplay(result, loc)
	result["time_range_display"] = map[string]interface{}{
		"start": FormatTimestamp(startTime, loc),
		"end":   FormatTimestamp(endTime, loc),
	}
	out, marshalErr := json.MarshalIndent(result, "", "  ")
	if marshalErr != nil {
		return raw, nil
	}
	return string(out), nil
}

// ===== Incident Timeline Handler =====

func handleGetIncidentTimeline(client any, params map[string]any) (string, error) {
	c, err := getClient(client)
	if err != nil {
		return "", err
	}

	incidentID := getStringParam(params, "incident_id")
	if incidentID == "" {
		return "", fmt.Errorf("incident_id is required")
	}

	body := map[string]any{
		"incident_id": incidentID,
	}

	setOptionalStringSlice(params, body, "types")
	buildPaginationParams(params, body)

	result, err := c.DoRequest("/incident/feed", body)
	if err != nil {
		return "", err
	}
	enriched := enrichTimeline(c, result)
	return injectTimestampDisplayToJSON(enriched, c.Location), nil
}

// ===== Incident-Alert Association Handler =====

func handleListIncidentAlerts(client any, params map[string]any) (string, error) {
	c, err := getClient(client)
	if err != nil {
		return "", err
	}

	incidentID := getStringParam(params, "incident_id")
	if incidentID == "" {
		return "", fmt.Errorf("incident_id is required")
	}

	body := map[string]any{
		"incident_id": incidentID,
	}

	if isActive, ok := getBoolParam(params, "is_active"); ok {
		body["is_active"] = isActive
	}
	buildPaginationParams(params, body)

	result, err := c.DoRequest("/incident/alert/list", body)
	if err != nil {
		return "", err
	}
	return injectTimestampDisplayToJSON(result, c.Location), nil
}

// ===== Similar Incidents Handler =====

func handleListSimilarIncidents(client any, params map[string]any) (string, error) {
	c, err := getClient(client)
	if err != nil {
		return "", err
	}

	incidentID := getStringParam(params, "incident_id")
	if incidentID == "" {
		return "", fmt.Errorf("incident_id is required")
	}

	body := map[string]any{
		"incident_id": incidentID,
	}

	if limit, ok := getIntParam(params, "limit"); ok {
		body["limit"] = min(limit, 20)
	}

	result, err := c.DoRequest("/incident/past/list", body)
	if err != nil {
		return "", err
	}
	return enrichIncidentList(c, result), nil
}

// ===== Update Incident Handler =====

func handleUpdateIncident(client any, params map[string]any) (string, error) {
	c, err := getClient(client)
	if err != nil {
		return "", err
	}

	incidentID := getStringParam(params, "incident_id")
	if incidentID == "" {
		return "", fmt.Errorf("incident_id is required")
	}

	body := map[string]any{
		"incident_id": incidentID,
	}

	setOptionalString(params, body, "title")
	setOptionalString(params, body, "description")
	setOptionalString(params, body, "impact")
	setOptionalString(params, body, "root_cause")
	setOptionalString(params, body, "resolution")
	setOptionalString(params, body, "incident_severity")

	if len(body) == 1 {
		return "", fmt.Errorf("at least one field besides incident_id must be provided")
	}

	return doAction(c, "/incident/reset", body, "Successfully updated incident")
}

// ===== Assign Incident Handler =====

func handleAssignIncident(client any, params map[string]any) (string, error) {
	c, err := getClient(client)
	if err != nil {
		return "", err
	}

	incidentIDs := getStringSliceParam(params, "incident_ids")
	if len(incidentIDs) == 0 {
		return "", fmt.Errorf("incident_ids is required and must not be empty")
	}

	assignType := getStringParam(params, "type")
	if assignType == "" {
		assignType = "assign"
	}

	assignedTo := map[string]any{
		"type": assignType,
	}

	personIDs := getIntSliceParam(params, "person_ids")
	escalateRuleID := getStringParam(params, "escalate_rule_id")
	if len(personIDs) == 0 && escalateRuleID == "" {
		return "", fmt.Errorf("one of person_ids or escalate_rule_id is required")
	}
	if len(personIDs) > 0 {
		assignedTo["person_ids"] = personIDs
	}
	if escalateRuleID != "" {
		assignedTo["escalate_rule_id"] = escalateRuleID
	}

	body := map[string]any{
		"incident_ids": incidentIDs,
		"assigned_to":  assignedTo,
	}

	return doAction(c, "/incident/assign", body,
		fmt.Sprintf("Successfully assigned %d incident(s)", len(incidentIDs)))
}

// ===== Change Handlers =====

func handleQueryChanges(client any, params map[string]any) (string, error) {
	c, err := getClient(client)
	if err != nil {
		return "", err
	}

	body := map[string]any{}

	// Time range is optional for changes
	if tr := getStringParam(params, "time_range"); tr != "" {
		startTime, endTime, err := parseTimeRange(tr)
		if err != nil {
			return "", err
		}
		body["start_time"] = startTime
		body["end_time"] = endTime
	} else {
		if startTime, ok := getNumberParam(params, "start_time"); ok {
			body["start_time"] = int64(startTime)
		}
		if endTime, ok := getNumberParam(params, "end_time"); ok {
			body["end_time"] = int64(endTime)
		}
	}

	setOptionalString(params, body, "query")
	setOptionalIntSlice(params, body, "channel_ids")
	setOptionalIntSlice(params, body, "integration_ids")
	if orderBy := getStringParam(params, "order_by"); orderBy != "" {
		body["orderby"] = orderBy
	}
	if asc, ok := getBoolParam(params, "asc"); ok {
		body["asc"] = asc
	}
	if includeEvents, ok := getBoolParam(params, "include_events"); ok {
		body["include_events"] = includeEvents
	}
	buildPaginationParams(params, body)

	return c.DoRequest("/change/list", body)
}

// ===== Escalation Rules Handler =====

func handleQueryEscalationRules(client any, params map[string]any) (string, error) {
	c, err := getClient(client)
	if err != nil {
		return "", err
	}

	channelID, ok := getIntParam(params, "channel_id")
	if !ok {
		return "", fmt.Errorf("channel_id is required")
	}

	body := map[string]any{
		"channel_id": channelID,
	}

	return c.DoRequest("/channel/escalate/rule/list", body)
}

// ===== Custom Fields Handler =====

func handleQueryFields(client any, _ map[string]any) (string, error) {
	c, err := getClient(client)
	if err != nil {
		return "", err
	}

	return c.DoRequest("/field/list", map[string]any{})
}

// ===== Aggregate Incidents Handler =====

func handleAggregateIncidents(client any, params map[string]any) (string, error) {
	c, err := getClient(client)
	if err != nil {
		return "", err
	}

	startTime, endTime, err := resolveListTimeParams(params)
	if err != nil {
		return "", err
	}

	req := &AggregateRequest{
		StartTime: startTime,
		EndTime:   endTime,
	}

	// group_by is required.
	req.GroupBy = getStringSliceParam(params, "group_by")
	if len(req.GroupBy) == 0 {
		return "", fmt.Errorf("group_by is required and must not be empty")
	}

	// channel_id and channel_name are mutually exclusive optional filters.
	if channelID, ok := getIntParam(params, "channel_id"); ok && channelID != 0 {
		req.ChannelID = channelID
	} else if channelName := getStringParam(params, "channel_name"); channelName != "" {
		req.ChannelName = channelName
		resolvedID, resolveErr := resolveChannelIDByName(c, channelName)
		if resolveErr != nil {
			return "", resolveErr
		}
		req.ChannelID = resolvedID
	}

	req.Severities = getStringSliceParam(params, "severities")
	req.Query = getStringParam(params, "query")

	if includeDetails, ok := getBoolParam(params, "include_details"); ok {
		req.IncludeDetails = includeDetails
	}
	req.DetailFields = getStringSliceParam(params, "detail_fields")

	if maxIncidents, ok := getIntParam(params, "max_incidents"); ok && maxIncidents > 0 {
		req.MaxIncidents = maxIncidents
	}

	result, err := AggregateIncidents(c, req)
	if err != nil {
		return "", err
	}

	return marshalOrFallback(result, "{}"), nil
}
