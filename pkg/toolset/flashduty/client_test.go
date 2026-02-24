package flashduty

import (
	"encoding/json"
	"testing"
)

func TestFilterBriefList_ValidJSON(t *testing.T) {
	input := `{
		"items": [
			{"incident_id": "abc", "title": "Test", "extra_field": "remove_me", "num": 1},
			{"incident_id": "def", "title": "Test2", "extra_field": "remove_me2", "num": 2}
		],
		"total": 2,
		"has_next_page": false
	}`

	briefFields := []string{"incident_id", "title", "num"}
	result := filterBriefList(input, briefFields)

	var data map[string]any
	if err := json.Unmarshal([]byte(result), &data); err != nil {
		t.Fatalf("failed to parse result: %v", err)
	}

	items, ok := data["items"].([]any)
	if !ok {
		t.Fatal("expected items array")
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}

	// Check first item only has brief fields
	first := items[0].(map[string]any)
	if _, exists := first["extra_field"]; exists {
		t.Error("extra_field should have been filtered out")
	}
	if first["incident_id"] != "abc" {
		t.Errorf("incident_id: got %v, want abc", first["incident_id"])
	}
	if first["title"] != "Test" {
		t.Errorf("title: got %v, want Test", first["title"])
	}

	// Check pagination metadata is preserved
	if data["total"] != float64(2) {
		t.Errorf("total: got %v, want 2", data["total"])
	}
	if data["has_next_page"] != false {
		t.Errorf("has_next_page: got %v, want false", data["has_next_page"])
	}
}

func TestFilterBriefList_PreservesSearchAfterCtx(t *testing.T) {
	input := `{
		"items": [{"id": "1", "name": "test"}],
		"total": 100,
		"has_next_page": true,
		"search_after_ctx": "cursor123"
	}`

	result := filterBriefList(input, []string{"id"})

	var data map[string]any
	if err := json.Unmarshal([]byte(result), &data); err != nil {
		t.Fatalf("failed to parse result: %v", err)
	}

	if data["search_after_ctx"] != "cursor123" {
		t.Errorf("search_after_ctx: got %v, want cursor123", data["search_after_ctx"])
	}
	if data["has_next_page"] != true {
		t.Errorf("has_next_page: got %v, want true", data["has_next_page"])
	}
}

func TestFilterBriefList_EmptyItems(t *testing.T) {
	input := `{"items": [], "total": 0}`
	result := filterBriefList(input, []string{"id"})

	var data map[string]any
	if err := json.Unmarshal([]byte(result), &data); err != nil {
		t.Fatalf("failed to parse result: %v", err)
	}

	items := data["items"].([]any)
	if len(items) != 0 {
		t.Errorf("expected empty items, got %d", len(items))
	}
}

func TestFilterBriefList_InvalidJSON(t *testing.T) {
	input := "not valid json"
	result := filterBriefList(input, []string{"id"})
	if result != input {
		t.Errorf("expected original string on invalid JSON, got %q", result)
	}
}

func TestFilterBriefList_MissingItemsKey(t *testing.T) {
	input := `{"data": [{"id": "1"}]}`
	result := filterBriefList(input, []string{"id"})
	if result != input {
		t.Errorf("expected original string when items key is missing, got %q", result)
	}
}

// ===== extractIntNameMap / extractStringNameMap Tests =====

func TestExtractIntNameMap(t *testing.T) {
	data := map[string]any{
		"items": []any{
			map[string]any{"person_id": float64(1), "person_name": "Alice"},
			map[string]any{"person_id": float64(2), "person_name": "Bob"},
			map[string]any{"person_id": float64(0), "person_name": "Zero"}, // id=0 skipped
			map[string]any{"person_id": float64(3), "person_name": ""},     // empty name skipped
		},
	}

	result := extractIntNameMap(data, "items", "person_id", "person_name")
	if len(result) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(result))
	}
	if result[1] != "Alice" {
		t.Errorf("id 1: got %q, want Alice", result[1])
	}
	if result[2] != "Bob" {
		t.Errorf("id 2: got %q, want Bob", result[2])
	}
}

func TestExtractIntNameMap_NilData(t *testing.T) {
	result := extractIntNameMap(nil, "items", "id", "name")
	if len(result) != 0 {
		t.Errorf("expected empty map for nil data, got %v", result)
	}
}

func TestExtractStringNameMap(t *testing.T) {
	data := map[string]any{
		"items": []any{
			map[string]any{"schedule_id": "s1", "schedule_name": "On-call A"},
			map[string]any{"schedule_id": "s2", "schedule_name": "On-call B"},
			map[string]any{"schedule_id": "", "schedule_name": "No ID"}, // empty id skipped
		},
	}

	result := extractStringNameMap(data, "items", "schedule_id", "schedule_name")
	if len(result) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(result))
	}
	if result["s1"] != "On-call A" {
		t.Errorf("s1: got %q, want On-call A", result["s1"])
	}
}

// ===== toInt Tests =====

func TestToInt(t *testing.T) {
	tests := []struct {
		name  string
		input any
		want  int
	}{
		{"float64", float64(42), 42},
		{"int", 7, 7},
		{"int64", int64(99), 99},
		{"string", "nope", 0},
		{"nil", nil, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := toInt(tt.input); got != tt.want {
				t.Errorf("toInt(%v) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}
