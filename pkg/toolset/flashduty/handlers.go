package flashduty

import (
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

// getStringParam extracts a string parameter from the map.
func getStringParam(m map[string]any, key string) string {
	v, ok := m[key]
	if !ok {
		return ""
	}
	s, _ := v.(string)
	return s
}

// getNumberParam extracts a numeric parameter from the map as float64.
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

// getIntParam extracts an integer parameter from the map.
func getIntParam(m map[string]any, key string) (int, bool) {
	n, ok := getNumberParam(m, key)
	if !ok {
		return 0, false
	}
	return int(n), true
}

// getStringSliceParam extracts a string slice parameter from the map.
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

// getIntSliceParam extracts an integer slice parameter from the map.
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

// getBoolParam extracts a boolean parameter from the map.
func getBoolParam(m map[string]any, key string) (bool, bool) {
	v, ok := m[key]
	if !ok {
		return false, false
	}
	b, ok := v.(bool)
	return b, ok
}

// getMapParam extracts a map parameter from the map.
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

// lastWeekRange returns (Monday 00:00:00, Sunday 23:59:59) of the previous week in local time.
func lastWeekRange() (int64, int64) {
	now := time.Now()
	wd := int(now.Weekday())
	if wd == 0 {
		wd = 7 // ISO: Sunday = 7
	}
	thisMonday := time.Date(now.Year(), now.Month(), now.Day()-wd+1, 0, 0, 0, 0, now.Location())
	start := thisMonday.AddDate(0, 0, -7)
	end := thisMonday.Add(-time.Second)
	return start.Unix(), end.Unix()
}

// parseTimeRange parses a relative time range string and returns (startTime, endTime) as unix seconds.
// Supported formats:
//   - Duration: "1h", "24h", "7d", "30d", "1w", "6M" (end time = now)
//   - Named:    "last_day" (yesterday 00:00:00 to 23:59:59)
//               "last_week" (Monday 00:00:00 to Sunday 23:59:59 of previous week)
func parseTimeRange(tr string) (int64, int64, error) {
	tr = strings.TrimSpace(tr)
	if tr == "" {
		return 0, 0, fmt.Errorf("time_range is empty")
	}

	// Named ranges
	switch strings.ToLower(tr) {
	case "last_day":
		s, e := lastDayRange()
		return s, e, nil
	case "last_week":
		s, e := lastWeekRange()
		return s, e, nil
	}

	// Duration-based ranges
	unit := tr[len(tr)-1]
	numStr := tr[:len(tr)-1]
	n, err := strconv.Atoi(numStr)
	if err != nil || n <= 0 {
		return 0, 0, fmt.Errorf("invalid time_range %q: must be a duration (e.g., '24h', '7d') or a named range ('last_day', 'last_week')", tr)
	}
	now := time.Now()
	var dur time.Duration
	switch unit {
	case 'h':
		dur = time.Duration(n) * time.Hour
	case 'd':
		dur = time.Duration(n) * 24 * time.Hour
	case 'w':
		dur = time.Duration(n) * 7 * 24 * time.Hour
	case 'M':
		dur = time.Duration(n) * 30 * 24 * time.Hour
	default:
		return 0, 0, fmt.Errorf("invalid time_range unit %q: must be h, d, w, or M", string(unit))
	}
	return now.Add(-dur).Unix(), now.Unix(), nil
}

// resolveTimeParams extracts start/end time from params, supporting both
// time_range (e.g., "24h") and explicit start_time/end_time unix seconds.
func resolveTimeParams(params map[string]any) (int64, int64, error) {
	if tr := getStringParam(params, "time_range"); tr != "" {
		return parseTimeRange(tr)
	}
	startTime, ok := getNumberParam(params, "start_time")
	if !ok {
		return 0, 0, fmt.Errorf("either time_range or start_time+end_time is required")
	}
	endTime, ok := getNumberParam(params, "end_time")
	if !ok {
		return 0, 0, fmt.Errorf("either time_range or start_time+end_time is required")
	}
	return int64(startTime), int64(endTime), nil
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

// ===== Incident Handlers =====

// incidentBriefFields defines the fields to include in brief mode for incidents.
var incidentBriefFields = []string{
	"incident_id", "title", "incident_severity", "progress",
	"created_at", "channel_name", "channel_id", "num",
}

// alertBriefFields defines the fields to include in brief mode for alerts.
var alertBriefFields = []string{
	"alert_id", "title", "alert_severity", "is_active",
	"created_at", "channel_name", "channel_id",
}

// handleListIncidents handles the list_incidents tool call.
func handleListIncidents(client any, params map[string]any) (string, error) {
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
	}

	setOptionalString(params, body, "progress")
	setOptionalString(params, body, "incident_severity")
	setOptionalString(params, body, "title")
	setOptionalIntSlice(params, body, "channel_ids")
	setOptionalIntSlice(params, body, "responder_ids")
	setOptionalIntSlice(params, body, "acker_ids")
	setOptionalIntSlice(params, body, "creator_ids")
	setOptionalMap(params, body, "labels")
	buildPaginationParams(params, body)

	result, err := c.DoRequest("/incident/list", body)
	if err != nil {
		return "", err
	}

	if brief, ok := getBoolParam(params, "brief"); ok && brief {
		return filterBriefList(result, incidentBriefFields), nil
	}
	return result, nil
}

// handleGetIncident handles the get_incident tool call.
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

	return c.DoRequest("/incident/info", body)
}

// handleCreateIncident handles the create_incident tool call.
func handleCreateIncident(client any, params map[string]any) (string, error) {
	c, err := getClient(client)
	if err != nil {
		return "", err
	}

	title := getStringParam(params, "title")
	if title == "" {
		return "", fmt.Errorf("title is required")
	}
	severity := getStringParam(params, "incident_severity")
	if severity == "" {
		return "", fmt.Errorf("incident_severity is required")
	}

	body := map[string]any{
		"title":             title,
		"incident_severity": severity,
	}

	setOptionalString(params, body, "description")
	if channelID, ok := getIntParam(params, "channel_id"); ok {
		body["channel_id"] = channelID
	}

	return c.DoRequest("/incident/create", body)
}

// handleAckIncidents handles the ack_incidents tool call.
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

// handleResolveIncidents handles the resolve_incidents tool call.
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

// handleReopenIncidents handles the reopen_incidents tool call.
func handleReopenIncidents(client any, params map[string]any) (string, error) {
	c, err := getClient(client)
	if err != nil {
		return "", err
	}

	incidentIDs := getStringSliceParam(params, "incident_ids")
	if len(incidentIDs) == 0 {
		return "", fmt.Errorf("incident_ids is required and must not be empty")
	}
	reason := getStringParam(params, "reason")
	if reason == "" {
		return "", fmt.Errorf("reason is required")
	}

	body := map[string]any{
		"incident_ids": incidentIDs,
		"reason":       reason,
	}
	return doAction(c, "/incident/reopen", body,
		fmt.Sprintf("Successfully reopened %d incident(s)", len(incidentIDs)))
}

// handleSnoozeIncidents handles the snooze_incidents tool call.
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

// handleCommentIncidents handles the comment_incidents tool call.
func handleCommentIncidents(client any, params map[string]any) (string, error) {
	c, err := getClient(client)
	if err != nil {
		return "", err
	}

	incidentIDs := getStringSliceParam(params, "incident_ids")
	if len(incidentIDs) == 0 {
		return "", fmt.Errorf("incident_ids is required and must not be empty")
	}
	comment := getStringParam(params, "comment")
	if comment == "" {
		return "", fmt.Errorf("comment is required")
	}

	body := map[string]any{
		"incident_ids": incidentIDs,
		"comment":      comment,
	}
	return doAction(c, "/incident/comment", body,
		fmt.Sprintf("Successfully commented on %d incident(s)", len(incidentIDs)))
}

// ===== Alert Handlers =====

// handleListAlerts handles the list_alerts tool call.
func handleListAlerts(client any, params map[string]any) (string, error) {
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
	}

	setOptionalString(params, body, "alert_severity")
	setOptionalString(params, body, "title")
	if isActive, ok := getBoolParam(params, "is_active"); ok {
		body["is_active"] = isActive
	}
	setOptionalIntSlice(params, body, "channel_ids")
	setOptionalMap(params, body, "labels")
	buildPaginationParams(params, body)

	result, err := c.DoRequest("/alert/list", body)
	if err != nil {
		return "", err
	}

	if brief, ok := getBoolParam(params, "brief"); ok && brief {
		return filterBriefList(result, alertBriefFields), nil
	}
	return result, nil
}

// handleGetAlert handles the get_alert tool call.
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

	return c.DoRequest("/alert/info", body)
}

// handleCloseAlerts handles the close_alerts tool call.
func handleCloseAlerts(client any, params map[string]any) (string, error) {
	c, err := getClient(client)
	if err != nil {
		return "", err
	}

	alertIDs := getStringSliceParam(params, "alert_ids")
	if len(alertIDs) == 0 {
		return "", fmt.Errorf("alert_ids is required and must not be empty")
	}

	body := map[string]any{"alert_ids": alertIDs}
	return doAction(c, "/alert/close", body,
		fmt.Sprintf("Successfully closed %d alert(s)", len(alertIDs)))
}

// ===== Channel Handlers =====

// handleListChannels handles the list_channels tool call.
func handleListChannels(client any, params map[string]any) (string, error) {
	c, err := getClient(client)
	if err != nil {
		return "", err
	}
	return doQueryList(c, "/channel/list", params)
}

// handleGetChannel handles the get_channel tool call.
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

// handleListTeams handles the list_teams tool call.
func handleListTeams(client any, params map[string]any) (string, error) {
	c, err := getClient(client)
	if err != nil {
		return "", err
	}
	return doQueryList(c, "/team/list", params)
}

// ===== Member Handlers =====

// handleListMembers handles the list_members tool call.
func handleListMembers(client any, params map[string]any) (string, error) {
	c, err := getClient(client)
	if err != nil {
		return "", err
	}
	return doQueryList(c, "/member/list", params)
}

// ===== Schedule Handlers =====

// handleListSchedules handles the list_schedules tool call.
func handleListSchedules(client any, params map[string]any) (string, error) {
	c, err := getClient(client)
	if err != nil {
		return "", err
	}

	teamIDs := getIntSliceParam(params, "team_ids")
	if len(teamIDs) == 0 {
		return "", fmt.Errorf("team_ids is required and must not be empty")
	}

	body := map[string]any{
		"team_ids": teamIDs,
	}

	setOptionalString(params, body, "query")
	buildPaginationParams(params, body)

	return c.DoRequest("/schedule/list", body)
}

// ===== Insight Handlers =====

// handleGetIncidentStats handles the get_incident_stats tool call.
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
		"fields":     map[string]any{},
	}

	setOptionalIntSlice(params, body, "channel_ids")
	setOptionalIntSlice(params, body, "team_ids")
	setOptionalStringSlice(params, body, "severities")
	setOptionalString(params, body, "aggregate_unit")
	setOptionalString(params, body, "query")
	setOptionalMap(params, body, "labels")

	return c.DoRequest("/insight/account", body)
}

// ===== Incident Timeline Handler =====

// handleGetIncidentTimeline handles the get_incident_timeline tool call.
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

	return c.DoRequest("/incident/feed", body)
}

// ===== Incident-Alert Association Handler =====

// handleListIncidentAlerts handles the list_incident_alerts tool call.
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

	return c.DoRequest("/incident/alert/list", body)
}
