package flashduty

import (
	"bytes"
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
		var buf bytes.Buffer
		if err := json.Indent(&buf, resp.Data, "", "  "); err != nil {
			return string(resp.Data), nil
		}
		return buf.String(), nil
	}
	return "OK", nil
}

// DoRequest sends a POST request to the FlashDuty API and returns the formatted JSON data.
func (c *Client) DoRequest(path string, body map[string]any) (string, error) {
	if body == nil {
		body = map[string]any{}
	}
	var resp apiResponse
	if err := c.httpClient.POST(path).WithJSON(body).Do(&resp); err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	return parseResponse(&resp)
}

// DoGet sends a GET request to the FlashDuty API and returns the formatted JSON data.
func (c *Client) DoGet(path string) (string, error) {
	var resp apiResponse
	if err := c.httpClient.GET(path).Do(&resp); err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	return parseResponse(&resp)
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

	fieldSet := make(map[string]bool, len(briefFields))
	for _, f := range briefFields {
		fieldSet[f] = true
	}

	filtered := make([]any, 0, len(items))
	for _, item := range items {
		obj, ok := item.(map[string]any)
		if !ok {
			continue
		}
		brief := make(map[string]any, len(briefFields))
		for k, v := range obj {
			if fieldSet[k] {
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

	out, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return jsonData
	}
	return string(out)
}
