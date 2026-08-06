package flashduty

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

type toolHandler func(any, map[string]any) (string, error)

func registeredToolHandler(t *testing.T, name string) toolHandler {
	t.Helper()
	for _, serverTool := range (&Toolset{}).GetTools(nil) {
		if serverTool.Tool.Name == name {
			return toolHandler(serverTool.Handler)
		}
	}
	t.Fatalf("tool %q not found", name)
	return nil
}

func captureHandlerRequest(t *testing.T, handler toolHandler, params map[string]any) (string, map[string]any, error) {
	t.Helper()

	var path string
	var body map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode request body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"data": nil})
	}))
	defer server.Close()

	_, err := handler(makeClient(server.URL), params)
	return path, body, err
}

func TestRegisteredTools_RejectInvalidIntegerInputs(t *testing.T) {
	tests := []struct {
		name   string
		tool   string
		params map[string]any
	}{
		{
			name:   "scalar",
			tool:   "get_channel",
			params: map[string]any{"channel_id": float64(17.9)},
		},
		{
			name:   "array",
			tool:   "list_schedules",
			params: map[string]any{"team_ids": []any{float64(7.5)}},
		},
		{
			name:   "out of range scalar",
			tool:   "get_channel",
			params: map[string]any{"channel_id": float64(1e20)},
		},
		{
			name:   "out of range array",
			tool:   "list_schedules",
			params: map[string]any{"team_ids": []any{float64(1e20)}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requestStarted := false
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				requestStarted = true
				_ = json.NewEncoder(w).Encode(map[string]any{"data": nil})
			}))
			defer server.Close()

			_, err := registeredToolHandler(t, tt.tool)(makeClient(server.URL), tt.params)
			if err == nil {
				t.Fatal("expected invalid integer input to return an error")
			}
			if requestStarted {
				t.Fatal("fractional integer input must be rejected before the HTTP request")
			}
		})
	}
}

func TestListHandlers_RejectOverflowingTimeRangeBeforeRequest(t *testing.T) {
	for _, tool := range []string{"list_incidents", "list_alerts", "aggregate_incidents"} {
		t.Run(tool, func(t *testing.T) {
			requestStarted := false
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				requestStarted = true
				_ = json.NewEncoder(w).Encode(map[string]any{"data": nil})
			}))
			defer server.Close()

			_, err := registeredToolHandler(t, tool)(makeClient(server.URL), map[string]any{"time_range": "5124096h"})
			if err == nil {
				t.Fatal("expected overflowing time_range to return an error")
			}
			if requestStarted {
				t.Fatal("overflowing time_range must be rejected before the HTTP request")
			}
		})
	}
}

func TestHandleListIncidents_UsesPublishedQueryField(t *testing.T) {
	path, body, err := captureHandlerRequest(t, handleListIncidents, map[string]any{
		"start_time": float64(1_000),
		"end_time":   float64(2_000),
		"query":      "database",
		"title":      "legacy title",
		"labels":     map[string]any{"env": "prod"},
	})
	if err != nil {
		t.Fatalf("handleListIncidents() error = %v", err)
	}
	if path != "/incident/list" {
		t.Errorf("request path = %q, want /incident/list", path)
	}
	if body["query"] != "database" {
		t.Errorf("query = %v, want database", body["query"])
	}
	for _, unsupported := range []string{"title", "labels"} {
		if _, ok := body[unsupported]; ok {
			t.Errorf("request must not include unsupported %q", unsupported)
		}
	}
}

func TestHandleListAlerts_OmitsUnsupportedFilters(t *testing.T) {
	_, body, err := captureHandlerRequest(t, handleListAlerts, map[string]any{
		"start_time": float64(1_000),
		"end_time":   float64(2_000),
		"title":      "legacy title",
		"labels":     map[string]any{"env": "prod"},
	})
	if err != nil {
		t.Fatalf("handleListAlerts() error = %v", err)
	}
	for _, unsupported := range []string{"title", "labels"} {
		if _, ok := body[unsupported]; ok {
			t.Errorf("request must not include unsupported %q", unsupported)
		}
	}
}

func TestHandleListAlerts_BriefPreservesAlertStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"items": []map[string]any{{
					"alert_id":       "0123456789abcdef01234567",
					"title":          "Database latency",
					"alert_severity": "Warning",
					"alert_status":   "Warning",
					"is_active":      true,
				}},
			},
		})
	}))
	defer server.Close()

	result, err := handleListAlerts(makeClient(server.URL), map[string]any{
		"start_time": float64(1_000),
		"end_time":   float64(2_000),
		"brief":      true,
	})
	if err != nil {
		t.Fatalf("handleListAlerts() error = %v", err)
	}

	var response struct {
		Items []map[string]any `json:"items"`
	}
	if err := json.Unmarshal([]byte(result), &response); err != nil {
		t.Fatalf("unmarshal brief response: %v", err)
	}
	if len(response.Items) != 1 {
		t.Fatalf("brief item count = %d, want 1", len(response.Items))
	}
	if got := response.Items[0]["alert_status"]; got != "Warning" {
		t.Errorf("alert_status = %v, want Warning", got)
	}
	if _, ok := response.Items[0]["is_active"]; ok {
		t.Error("brief response must not promise is_active")
	}
}

func TestHandleQueryChanges_MapsOrderByToPublishedField(t *testing.T) {
	_, body, err := captureHandlerRequest(t, handleQueryChanges, map[string]any{"order_by": "start_time"})
	if err != nil {
		t.Fatalf("handleQueryChanges() error = %v", err)
	}
	if body["orderby"] != "start_time" {
		t.Errorf("orderby = %v, want start_time", body["orderby"])
	}
	if _, ok := body["order_by"]; ok {
		t.Error("request must not include order_by")
	}
}

func TestHandleAssignIncident_UsesStringEscalationRuleID(t *testing.T) {
	_, body, err := captureHandlerRequest(t, handleAssignIncident, map[string]any{
		"incident_ids":     []string{"0123456789abcdef01234567"},
		"type":             "escalate",
		"escalate_rule_id": "89abcdef0123456701234567",
	})
	if err != nil {
		t.Fatalf("handleAssignIncident() error = %v", err)
	}
	assignedTo, ok := body["assigned_to"].(map[string]any)
	if !ok {
		t.Fatalf("assigned_to = %T, want object", body["assigned_to"])
	}
	if assignedTo["escalate_rule_id"] != "89abcdef0123456701234567" {
		t.Errorf("escalate_rule_id = %v, want string ObjectID", assignedTo["escalate_rule_id"])
	}
}

func TestHandleAssignIncident_RejectsMissingAssignmentTarget(t *testing.T) {
	_, _, err := captureHandlerRequest(t, handleAssignIncident, map[string]any{
		"incident_ids": []string{"0123456789abcdef01234567"},
	})
	if err == nil {
		t.Fatal("expected an error when person_ids and escalate_rule_id are both absent")
	}
}

func TestWriteHandlers_PreservePublishedOptionalFields(t *testing.T) {
	tests := []struct {
		name    string
		handler toolHandler
		params  map[string]any
		path    string
		field   string
		want    any
	}{
		{
			name:    "create incident title",
			handler: handleCreateIncident,
			params:  map[string]any{"incident_severity": "Warning", "title": "Database latency"},
			path:    "/incident/create",
			field:   "title",
			want:    "Database latency",
		},
		{
			name:    "reopen reason",
			handler: handleReopenIncidents,
			params:  map[string]any{"incident_ids": []string{"0123456789abcdef01234567"}, "reason": "Investigation resumed"},
			path:    "/incident/reopen",
			field:   "reason",
			want:    "Investigation resumed",
		},
		{
			name:    "comment content",
			handler: handleCommentIncidents,
			params:  map[string]any{"incident_ids": []string{"0123456789abcdef01234567"}, "comment": "Investigating the database"},
			path:    "/incident/comment",
			field:   "comment",
			want:    "Investigating the database",
		},
		{
			name:    "schedule team IDs",
			handler: handleListSchedules,
			params:  map[string]any{"team_ids": []any{float64(7), float64(9)}},
			path:    "/schedule/list",
			field:   "team_ids",
			want:    []any{float64(7), float64(9)},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path, body, err := captureHandlerRequest(t, tt.handler, tt.params)
			if err != nil {
				t.Fatalf("handler returned error: %v", err)
			}
			if path != tt.path {
				t.Errorf("request path = %q, want %s", path, tt.path)
			}
			if got := body[tt.field]; !reflect.DeepEqual(got, tt.want) {
				t.Errorf("request body %s = %#v, want %#v", tt.field, got, tt.want)
			}
		})
	}
}

func TestWriteHandlers_AcceptOmittedPublishedOptionalFields(t *testing.T) {
	tests := []struct {
		name    string
		handler toolHandler
		params  map[string]any
		path    string
		omitted string
	}{
		{
			name:    "create incident title",
			handler: handleCreateIncident,
			params:  map[string]any{"incident_severity": "Warning"},
			path:    "/incident/create",
			omitted: "title",
		},
		{
			name:    "reopen reason",
			handler: handleReopenIncidents,
			params:  map[string]any{"incident_ids": []string{"0123456789abcdef01234567"}},
			path:    "/incident/reopen",
			omitted: "reason",
		},
		{
			name:    "comment content",
			handler: handleCommentIncidents,
			params:  map[string]any{"incident_ids": []string{"0123456789abcdef01234567"}},
			path:    "/incident/comment",
			omitted: "comment",
		},
		{
			name:    "schedule team IDs",
			handler: handleListSchedules,
			params:  map[string]any{},
			path:    "/schedule/list",
			omitted: "team_ids",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path, body, err := captureHandlerRequest(t, tt.handler, tt.params)
			if err != nil {
				t.Fatalf("handler rejected omitted optional %s: %v", tt.omitted, err)
			}
			if path != tt.path {
				t.Errorf("request path = %q, want %s", path, tt.path)
			}
			if _, ok := body[tt.omitted]; ok {
				t.Errorf("request must omit optional %s when it is not supplied", tt.omitted)
			}
		})
	}
}
