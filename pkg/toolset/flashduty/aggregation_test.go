package flashduty

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// ===== extractDimensionValue tests =====

func TestExtractDimensionValue_Severity(t *testing.T) {
	// Built-in dimension severity/incident_severity extraction.
	incident := map[string]any{
		"incident_severity": "critical",
	}

	// severity alias
	if got := extractDimensionValue(incident, "severity"); got != "critical" {
		t.Errorf("severity: got %q, want %q", got, "critical")
	}
	// incident_severity canonical name
	if got := extractDimensionValue(incident, "incident_severity"); got != "critical" {
		t.Errorf("incident_severity: got %q, want %q", got, "critical")
	}
}

func TestExtractDimensionValue_SeverityEmpty(t *testing.T) {
	// Returns "(empty)" when the severity field is absent.
	incident := map[string]any{}
	if got := extractDimensionValue(incident, "severity"); got != "(empty)" {
		t.Errorf("severity missing: got %q, want (empty)", got)
	}
}

func TestExtractDimensionValue_Channel(t *testing.T) {
	// Built-in dimension channel: prefers channel_name when present.
	incident := map[string]any{
		"channel_name": "ops-alerts",
		"channel_id":   float64(42),
	}
	if got := extractDimensionValue(incident, "channel"); got != "ops-alerts" {
		t.Errorf("channel with name: got %q, want ops-alerts", got)
	}
}

func TestExtractDimensionValue_ChannelFallbackToID(t *testing.T) {
	// Falls back to "channel_<id>" format when channel_name is absent.
	incident := map[string]any{
		"channel_id": float64(42),
	}
	if got := extractDimensionValue(incident, "channel"); got != "channel_42" {
		t.Errorf("channel fallback to id: got %q, want channel_42", got)
	}
}

func TestExtractDimensionValue_ChannelEmpty(t *testing.T) {
	// Returns "(empty)" when both channel_name and channel_id are absent.
	incident := map[string]any{}
	if got := extractDimensionValue(incident, "channel"); got != "(empty)" {
		t.Errorf("channel empty: got %q, want (empty)", got)
	}
}

func TestExtractDimensionValue_Progress(t *testing.T) {
	// Built-in dimension progress extraction.
	incident := map[string]any{
		"progress": "triggered",
	}
	if got := extractDimensionValue(incident, "progress"); got != "triggered" {
		t.Errorf("progress: got %q, want triggered", got)
	}
}

func TestExtractDimensionValue_ProgressEmpty(t *testing.T) {
	// Returns "(empty)" when the progress field is absent.
	incident := map[string]any{}
	if got := extractDimensionValue(incident, "progress"); got != "(empty)" {
		t.Errorf("progress missing: got %q, want (empty)", got)
	}
}

func TestExtractDimensionValue_Labels(t *testing.T) {
	// Extracts labels.<key> dimension values.
	incident := map[string]any{
		"labels": map[string]any{
			"datacenter": "us-east-1",
			"team":       "sre",
		},
	}
	if got := extractDimensionValue(incident, "labels.datacenter"); got != "us-east-1" {
		t.Errorf("labels.datacenter: got %q, want us-east-1", got)
	}
	if got := extractDimensionValue(incident, "labels.team"); got != "sre" {
		t.Errorf("labels.team: got %q, want sre", got)
	}
}

func TestExtractDimensionValue_LabelsMissingKey(t *testing.T) {
	// Returns "(empty)" when the requested label key does not exist.
	incident := map[string]any{
		"labels": map[string]any{
			"datacenter": "us-east-1",
		},
	}
	if got := extractDimensionValue(incident, "labels.env"); got != "(empty)" {
		t.Errorf("labels missing key: got %q, want (empty)", got)
	}
}

func TestExtractDimensionValue_LabelsNoLabelsField(t *testing.T) {
	// Returns "(empty)" when the incident has no labels field at all.
	incident := map[string]any{}
	if got := extractDimensionValue(incident, "labels.datacenter"); got != "(empty)" {
		t.Errorf("no labels field: got %q, want (empty)", got)
	}
}

func TestExtractDimensionValue_InvalidDimension(t *testing.T) {
	// Unknown dimension: reads the string field from the incident, or returns "(empty)".
	incident := map[string]any{}
	if got := extractDimensionValue(incident, "unknown_field"); got != "(empty)" {
		t.Errorf("invalid dim: got %q, want (empty)", got)
	}
	// Field exists: returns its string value.
	incident2 := map[string]any{"unknown_field": "some_value"}
	if got := extractDimensionValue(incident2, "unknown_field"); got != "some_value" {
		t.Errorf("unknown field with value: got %q, want some_value", got)
	}
}

// ===== buildCompositeKey tests =====

func TestBuildCompositeKey_SingleDimension(t *testing.T) {
	// Single-dimension composite key.
	incident := map[string]any{"incident_severity": "warning"}
	got := buildCompositeKey(incident, []string{"severity"})
	if got != "warning" {
		t.Errorf("single dim: got %q, want warning", got)
	}
}

func TestBuildCompositeKey_MultiDimension(t *testing.T) {
	// Multi-dimension composite key joined with "\x00".
	incident := map[string]any{
		"incident_severity": "critical",
		"labels":            map[string]any{"datacenter": "us-east-1"},
	}
	got := buildCompositeKey(incident, []string{"severity", "labels.datacenter"})
	want := "critical\x00us-east-1"
	if got != want {
		t.Errorf("multi dim: got %q, want %q", got, want)
	}
}

func TestBuildCompositeKey_EmptyValues(t *testing.T) {
	// Missing fields are represented as "(empty)" in the composite key.
	incident := map[string]any{}
	got := buildCompositeKey(incident, []string{"severity", "labels.datacenter"})
	want := "(empty)\x00(empty)"
	if got != want {
		t.Errorf("empty values: got %q, want %q", got, want)
	}
}

// ===== AggregateIncidents test helpers =====

// newMockServer creates an HTTP test server that returns the given incidents as a
// single-page response (has_next_page=false) in the standard FlashDuty API envelope.
func newMockServer(t *testing.T, incidents []map[string]any) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data := map[string]any{
			"items":         incidents,
			"has_next_page": false,
			"total":         len(incidents),
		}
		dataBytes, _ := json.Marshal(data)

		resp := map[string]any{
			"data": json.RawMessage(dataBytes),
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
}

// makeClient constructs a test Client pointed at the given base URL.
func makeClient(baseURL string) *Client {
	return NewClient(baseURL, "test-app-key", "")
}

// ===== AggregateIncidents core tests =====

func TestAggregateIncidents_SingleDimension(t *testing.T) {
	// Single-dimension grouping (severity only).
	incidents := []map[string]any{
		{"incident_severity": "critical", "title": "DB down"},
		{"incident_severity": "critical", "title": "API down"},
		{"incident_severity": "warning", "title": "Latency high"},
	}
	server := newMockServer(t, incidents)
	defer server.Close()

	client := makeClient(server.URL)
	req := &AggregateRequest{
		StartTime: 1000,
		EndTime:   2000,
		GroupBy:   []string{"severity"},
	}

	result, err := AggregateIncidents(client, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.TotalIncidents != 3 {
		t.Errorf("TotalIncidents: got %d, want 3", result.TotalIncidents)
	}
	if result.TotalGroups != 2 {
		t.Errorf("TotalGroups: got %d, want 2", result.TotalGroups)
	}

	// Verify the critical group count.
	var criticalCount int
	for _, g := range result.Groups {
		if g.Key["severity"] == "critical" {
			criticalCount = g.Count
		}
	}
	if criticalCount != 2 {
		t.Errorf("critical group count: got %d, want 2", criticalCount)
	}
}

func TestAggregateIncidents_MultiDimensionCrossGroup(t *testing.T) {
	// Multi-dimension cross grouping (severity x labels.datacenter).
	incidents := []map[string]any{
		{"incident_severity": "critical", "labels": map[string]any{"datacenter": "us-east-1"}},
		{"incident_severity": "critical", "labels": map[string]any{"datacenter": "eu-west-1"}},
		{"incident_severity": "warning", "labels": map[string]any{"datacenter": "us-east-1"}},
		{"incident_severity": "critical", "labels": map[string]any{"datacenter": "us-east-1"}},
	}
	server := newMockServer(t, incidents)
	defer server.Close()

	client := makeClient(server.URL)
	req := &AggregateRequest{
		StartTime: 1000,
		EndTime:   2000,
		GroupBy:   []string{"severity", "labels.datacenter"},
	}

	result, err := AggregateIncidents(client, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Expected groups: critical/us-east-1, critical/eu-west-1, warning/us-east-1.
	if result.TotalGroups != 3 {
		t.Errorf("TotalGroups: got %d, want 3", result.TotalGroups)
	}

	// The critical/us-east-1 group should contain 2 incidents.
	for _, g := range result.Groups {
		if g.Key["severity"] == "critical" && g.Key["labels.datacenter"] == "us-east-1" {
			if g.Count != 2 {
				t.Errorf("critical/us-east-1 count: got %d, want 2", g.Count)
			}
			return
		}
	}
	t.Error("critical/us-east-1 group not found")
}

func TestAggregateIncidents_IncludeDetailsTrue(t *testing.T) {
	// include_details=true returns detail fields per incident.
	incidents := []map[string]any{
		{
			"incident_severity": "critical",
			"title":             "DB down",
			"incident_id":       "INC001",
			"progress":          "triggered",
		},
	}
	server := newMockServer(t, incidents)
	defer server.Close()

	client := makeClient(server.URL)
	req := &AggregateRequest{
		StartTime:      1000,
		EndTime:        2000,
		GroupBy:        []string{"severity"},
		IncludeDetails: true,
		DetailFields:   []string{"incident_id", "title"},
	}

	result, err := AggregateIncidents(client, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Groups) == 0 {
		t.Fatal("expected at least one group")
	}
	group := result.Groups[0]
	if group.Details == nil {
		t.Fatal("Details should not be nil when include_details=true")
	}
	if len(group.Details) != 1 {
		t.Fatalf("Details length: got %d, want 1", len(group.Details))
	}
	detail := group.Details[0]
	if detail["incident_id"] != "INC001" {
		t.Errorf("detail incident_id: got %v, want INC001", detail["incident_id"])
	}
	if detail["title"] != "DB down" {
		t.Errorf("detail title: got %v, want DB down", detail["title"])
	}
}

func TestAggregateIncidents_IncludeDetailsFalse(t *testing.T) {
	// include_details=false omits details from the result.
	incidents := []map[string]any{
		{"incident_severity": "critical", "title": "DB down"},
	}
	server := newMockServer(t, incidents)
	defer server.Close()

	client := makeClient(server.URL)
	req := &AggregateRequest{
		StartTime:      1000,
		EndTime:        2000,
		GroupBy:        []string{"severity"},
		IncludeDetails: false,
	}

	result, err := AggregateIncidents(client, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Groups) == 0 {
		t.Fatal("expected at least one group")
	}
	if result.Groups[0].Details != nil {
		t.Errorf("Details should be nil when include_details=false, got %v", result.Groups[0].Details)
	}
}

func TestAggregateIncidents_MaxIncidentsLimit(t *testing.T) {
	// max_incidents cap is respected: only the first N incidents are processed.
	// The server returns 10 incidents; with max_incidents=3, at most 3 should be counted.
	incidents := make([]map[string]any, 10)
	for i := range incidents {
		incidents[i] = map[string]any{"incident_severity": "warning"}
	}

	server := newMockServer(t, incidents)
	defer server.Close()

	client := makeClient(server.URL)
	req := &AggregateRequest{
		StartTime:    1000,
		EndTime:      2000,
		GroupBy:      []string{"severity"},
		MaxIncidents: 3,
	}

	result, err := AggregateIncidents(client, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.TotalIncidents > 3 {
		t.Errorf("TotalIncidents: got %d, want ≤ 3 (max_incidents=3)", result.TotalIncidents)
	}
}

func TestAggregateIncidents_EmptyResult(t *testing.T) {
	// Empty incident list returns zero groups and zero total.
	server := newMockServer(t, []map[string]any{})
	defer server.Close()

	client := makeClient(server.URL)
	req := &AggregateRequest{
		StartTime: 1000,
		EndTime:   2000,
		GroupBy:   []string{"severity"},
	}

	result, err := AggregateIncidents(client, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.TotalIncidents != 0 {
		t.Errorf("TotalIncidents: got %d, want 0", result.TotalIncidents)
	}
	if result.TotalGroups != 0 {
		t.Errorf("TotalGroups: got %d, want 0", result.TotalGroups)
	}
	if len(result.Groups) != 0 {
		t.Errorf("Groups: got %d, want 0", len(result.Groups))
	}
}

func TestAggregateIncidents_LabelKeyMissingGoesToEmpty(t *testing.T) {
	// Incidents without the requested label key are placed in the "(empty)" group.
	incidents := []map[string]any{
		{"incident_severity": "critical", "labels": map[string]any{"datacenter": "us-east-1"}},
		{"incident_severity": "critical"},                             // no labels field — should fall into (empty)
		{"incident_severity": "critical", "labels": map[string]any{}}, // empty labels map — should fall into (empty)
	}
	server := newMockServer(t, incidents)
	defer server.Close()

	client := makeClient(server.URL)
	req := &AggregateRequest{
		StartTime: 1000,
		EndTime:   2000,
		GroupBy:   []string{"labels.datacenter"},
	}

	result, err := AggregateIncidents(client, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Expected groups: us-east-1 and (empty).
	if result.TotalGroups != 2 {
		t.Errorf("TotalGroups: got %d, want 2", result.TotalGroups)
	}

	var emptyGroupCount int
	for _, g := range result.Groups {
		if g.Key["labels.datacenter"] == "(empty)" {
			emptyGroupCount = g.Count
		}
	}
	if emptyGroupCount != 2 {
		t.Errorf("(empty) group count: got %d, want 2", emptyGroupCount)
	}
}

func TestAggregateIncidents_ResultsSortedByKey(t *testing.T) {
	// Groups are sorted by composite key in lexicographic order.
	incidents := []map[string]any{
		{"incident_severity": "warning"},
		{"incident_severity": "critical"},
		{"incident_severity": "info"},
	}
	server := newMockServer(t, incidents)
	defer server.Close()

	client := makeClient(server.URL)
	req := &AggregateRequest{
		StartTime: 1000,
		EndTime:   2000,
		GroupBy:   []string{"severity"},
	}

	result, err := AggregateIncidents(client, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Lexicographic order: critical < info < warning.
	keys := make([]string, len(result.Groups))
	for i, g := range result.Groups {
		keys[i] = g.Key["severity"]
	}

	for i := 1; i < len(keys); i++ {
		if strings.Compare(keys[i-1], keys[i]) > 0 {
			t.Errorf("groups not sorted: %q > %q at index %d", keys[i-1], keys[i], i)
		}
	}

	if len(keys) > 0 && keys[0] != "critical" {
		t.Errorf("first group (alphabetically): got %q, want critical", keys[0])
	}
}

func TestAggregateIncidents_DetailFieldsLabels(t *testing.T) {
	// detail_fields containing "labels.xxx" entries must be extracted from the
	// nested labels map, not looked up as a top-level key.
	incidents := []map[string]any{
		{
			"incident_severity": "critical",
			"title":             "Pod crash loop",
			"created_at":        int64(1700000000),
			"labels": map[string]any{
				"workload": "nginx",
				"node":     "node-1",
			},
		},
	}
	server := newMockServer(t, incidents)
	defer server.Close()

	client := makeClient(server.URL)
	req := &AggregateRequest{
		StartTime:      1000,
		EndTime:        2000,
		GroupBy:        []string{"severity"},
		IncludeDetails: true,
		DetailFields:   []string{"title", "incident_severity", "created_at", "labels.workload", "labels.node"},
	}

	result, err := AggregateIncidents(client, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Groups) == 0 {
		t.Fatal("expected at least one group")
	}
	detail := result.Groups[0].Details[0]

	// Verify plain fields.
	if detail["title"] != "Pod crash loop" {
		t.Errorf("detail title: got %v, want Pod crash loop", detail["title"])
	}
	if detail["incident_severity"] != "critical" {
		t.Errorf("detail incident_severity: got %v, want critical", detail["incident_severity"])
	}

	// Verify labels.xxx fields are returned under their dotted key.
	if detail["labels.workload"] != "nginx" {
		t.Errorf("detail labels.workload: got %v, want nginx", detail["labels.workload"])
	}
	if detail["labels.node"] != "node-1" {
		t.Errorf("detail labels.node: got %v, want node-1", detail["labels.node"])
	}
}

func TestAggregateIncidents_DetailFieldsLabels_MissingKey(t *testing.T) {
	// When a requested labels.xxx key is absent from the incident, the field
	// should be omitted from the detail record (not set to nil or "(empty)").
	incidents := []map[string]any{
		{
			"incident_severity": "warning",
			"labels":            map[string]any{"workload": "redis"},
		},
	}
	server := newMockServer(t, incidents)
	defer server.Close()

	client := makeClient(server.URL)
	req := &AggregateRequest{
		StartTime:      1000,
		EndTime:        2000,
		GroupBy:        []string{"severity"},
		IncludeDetails: true,
		DetailFields:   []string{"labels.workload", "labels.node"},
	}

	result, err := AggregateIncidents(client, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Groups) == 0 || len(result.Groups[0].Details) == 0 {
		t.Fatal("expected a detail entry")
	}
	detail := result.Groups[0].Details[0]

	// labels.workload is present.
	if detail["labels.workload"] != "redis" {
		t.Errorf("detail labels.workload: got %v, want redis", detail["labels.workload"])
	}
	// labels.node is absent — key must not appear in the detail map.
	if _, exists := detail["labels.node"]; exists {
		t.Errorf("detail labels.node should be absent, got %v", detail["labels.node"])
	}
}

func TestAggregateIncidents_MissingStartTime(t *testing.T) {
	// Validation error when start_time is zero.
	client := makeClient("http://localhost")
	req := &AggregateRequest{
		EndTime: 2000,
		GroupBy: []string{"severity"},
	}
	_, err := AggregateIncidents(client, req)
	if err == nil {
		t.Error("expected error for missing start_time")
	}
}

func TestAggregateIncidents_MissingGroupBy(t *testing.T) {
	// Validation error when group_by is empty.
	client := makeClient("http://localhost")
	req := &AggregateRequest{
		StartTime: 1000,
		EndTime:   2000,
		GroupBy:   []string{},
	}
	_, err := AggregateIncidents(client, req)
	if err == nil {
		t.Error("expected error for empty group_by")
	}
}
