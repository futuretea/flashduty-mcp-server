package flashduty

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	httpclient "github.com/futuretea/go-http-client"
)

// Client is an HTTP client for the FlashDuty API.
type Client struct {
	httpClient httpclient.Client
}

// NewClient creates a new FlashDuty API client.
func NewClient(baseURL, appKey string) *Client {
	c := httpclient.NewClient(
		&httpclient.Config{
			BaseURL: baseURL,
			Timeout: 30 * time.Second,
		},
		httpclient.WithMiddleware(func(req *http.Request) error {
			// Add app_key as query parameter for authentication
			q := req.URL.Query()
			q.Set("app_key", appKey)
			req.URL.RawQuery = q.Encode()
			return nil
		}),
	)
	return &Client{httpClient: c}
}

// apiResponse represents the standard FlashDuty API response envelope.
type apiResponse struct {
	Error *apiError       `json:"error,omitempty"`
	Data  json.RawMessage `json:"data,omitempty"`
}

// apiError represents the error object in a FlashDuty API response.
type apiError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// parseResponse checks for API-level errors and formats the data portion.
func parseResponse(resp *apiResponse) (string, error) {
	if resp.Error != nil && resp.Error.Code != "" {
		return "", fmt.Errorf("API error [%s]: %s", resp.Error.Code, resp.Error.Message)
	}
	if resp.Data != nil {
		return marshalOrFallback(resp.Data, string(resp.Data)), nil
	}
	return "OK", nil
}

// doPost sends a POST request and returns the parsed apiResponse.
func (c *Client) doPost(path string, body map[string]any) (*apiResponse, error) {
	if body == nil {
		body = map[string]any{}
	}
	var resp apiResponse
	if err := c.httpClient.POST(path).WithJSON(body).Do(&resp); err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	return &resp, nil
}

// DoRequest sends a POST request to the FlashDuty API and returns the formatted JSON data.
func (c *Client) DoRequest(path string, body map[string]any) (string, error) {
	resp, err := c.doPost(path, body)
	if err != nil {
		return "", err
	}
	return parseResponse(resp)
}

// DoGet sends a GET request to the FlashDuty API and returns the formatted JSON data.
func (c *Client) DoGet(path string) (string, error) {
	var resp apiResponse
	if err := c.httpClient.GET(path).Do(&resp); err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	return parseResponse(&resp)
}

// DoRequestRaw sends a POST request and returns the parsed data as a map.
func (c *Client) DoRequestRaw(path string, body map[string]any) (map[string]any, error) {
	resp, err := c.doPost(path, body)
	if err != nil {
		return nil, err
	}
	if resp.Error != nil && resp.Error.Code != "" {
		return nil, fmt.Errorf("API error [%s]: %s", resp.Error.Code, resp.Error.Message)
	}
	if resp.Data == nil {
		return nil, nil
	}
	var result map[string]any
	if err := json.Unmarshal(resp.Data, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal data: %w", err)
	}
	return result, nil
}

// FetchPersonInfos fetches person names by IDs. Returns map[personID]personName.
func (c *Client) FetchPersonInfos(personIDs []int) (map[int]string, error) {
	if len(personIDs) == 0 {
		return nil, nil
	}
	data, err := c.DoRequestRaw("/person/infos", map[string]any{"person_ids": personIDs})
	if err != nil {
		return nil, err
	}
	return extractIntNameMap(data, "items", "person_id", "person_name"), nil
}

// FetchChannelInfos fetches channel names by IDs. Returns map[channelID]channelName.
func (c *Client) FetchChannelInfos(channelIDs []int) (map[int]string, error) {
	if len(channelIDs) == 0 {
		return nil, nil
	}
	data, err := c.DoRequestRaw("/channel/infos", map[string]any{"channel_ids": channelIDs})
	if err != nil {
		return nil, err
	}
	return extractIntNameMap(data, "items", "channel_id", "channel_name"), nil
}

// FetchTeamInfos fetches team names by IDs. Returns map[teamID]teamName.
func (c *Client) FetchTeamInfos(teamIDs []int) (map[int]string, error) {
	if len(teamIDs) == 0 {
		return nil, nil
	}
	data, err := c.DoRequestRaw("/team/infos", map[string]any{"team_ids": teamIDs})
	if err != nil {
		return nil, err
	}
	return extractIntNameMap(data, "items", "team_id", "team_name"), nil
}

// FetchScheduleInfos fetches schedule names by IDs. Returns map[scheduleID]scheduleName.
func (c *Client) FetchScheduleInfos(scheduleIDs []string) (map[string]string, error) {
	if len(scheduleIDs) == 0 {
		return nil, nil
	}
	data, err := c.DoRequestRaw("/schedule/infos", map[string]any{"schedule_ids": scheduleIDs})
	if err != nil {
		return nil, err
	}
	return extractStringNameMap(data, "items", "schedule_id", "schedule_name"), nil
}

// extractIntNameMap extracts a map[int]string from a list of objects in data[listKey].
func extractIntNameMap(data map[string]any, listKey, idField, nameField string) map[int]string {
	result := make(map[int]string)
	if data == nil {
		return result
	}
	items, ok := data[listKey].([]any)
	if !ok {
		return result
	}
	for _, item := range items {
		obj, ok := item.(map[string]any)
		if !ok {
			continue
		}
		id := toInt(obj[idField])
		name, _ := obj[nameField].(string)
		if id != 0 && name != "" {
			result[id] = name
		}
	}
	return result
}

// extractStringNameMap extracts a map[string]string from a list of objects in data[listKey].
func extractStringNameMap(data map[string]any, listKey, idField, nameField string) map[string]string {
	result := make(map[string]string)
	if data == nil {
		return result
	}
	items, ok := data[listKey].([]any)
	if !ok {
		return result
	}
	for _, item := range items {
		obj, ok := item.(map[string]any)
		if !ok {
			continue
		}
		id, _ := obj[idField].(string)
		name, _ := obj[nameField].(string)
		if id != "" && name != "" {
			result[id] = name
		}
	}
	return result
}

// toInt converts a numeric interface value to int.
func toInt(v any) int {
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	case int64:
		return int(n)
	default:
		return 0
	}
}

// filterBriefList filters a list response to only include specified fields per item,
// while preserving pagination metadata (total, has_next_page, search_after_ctx).
// On any parse error, it returns the original data unchanged.
func filterBriefList(jsonData string, briefFields []string) string {
	var data map[string]any
	if err := json.Unmarshal([]byte(jsonData), &data); err != nil {
		return jsonData
	}

	items, ok := data["items"].([]any)
	if !ok {
		return jsonData
	}

	fieldSet := make(map[string]struct{}, len(briefFields))
	for _, f := range briefFields {
		fieldSet[f] = struct{}{}
	}

	filtered := make([]any, 0, len(items))
	for _, item := range items {
		obj, ok := item.(map[string]any)
		if !ok {
			continue
		}
		brief := make(map[string]any, len(briefFields))
		for k, v := range obj {
			if _, ok := fieldSet[k]; ok {
				brief[k] = v
			}
		}
		filtered = append(filtered, brief)
	}

	result := map[string]any{"items": filtered}
	for _, key := range []string{"total", "has_next_page", "search_after_ctx"} {
		if v, exists := data[key]; exists {
			result[key] = v
		}
	}

	return marshalOrFallback(result, jsonData)
}
