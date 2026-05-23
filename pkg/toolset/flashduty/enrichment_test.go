package flashduty

import (
	"testing"
)

// ===== marshalOrFallback Tests =====

func TestMarshalOrFallback_Valid(t *testing.T) {
	data := map[string]any{"key": "value"}
	result := marshalOrFallback(data, "fallback")
	if result == "fallback" {
		t.Error("expected valid JSON, got fallback")
	}
	if result == "" {
		t.Error("expected non-empty result")
	}
}

func TestMarshalOrFallback_Fallback(t *testing.T) {
	// Channels cannot be marshaled to JSON
	data := make(chan int)
	result := marshalOrFallback(data, "fallback_value")
	if result != "fallback_value" {
		t.Errorf("expected fallback_value, got %q", result)
	}
}

// ===== intSetToSlice Tests =====

func TestIntSetToSlice_NonEmpty(t *testing.T) {
	set := map[int]struct{}{1: {}, 2: {}, 3: {}}
	result := intSetToSlice(set)
	if len(result) != 3 {
		t.Fatalf("expected 3 elements, got %d", len(result))
	}

	// Check all elements are present (order not guaranteed)
	found := make(map[int]bool)
	for _, v := range result {
		found[v] = true
	}
	for _, want := range []int{1, 2, 3} {
		if !found[want] {
			t.Errorf("missing element %d in result %v", want, result)
		}
	}
}

func TestIntSetToSlice_Empty(t *testing.T) {
	set := map[int]struct{}{}
	result := intSetToSlice(set)
	if len(result) != 0 {
		t.Errorf("expected empty slice, got %v", result)
	}
}

// ===== collectIntID Tests =====

func TestCollectIntID(t *testing.T) {
	obj := map[string]any{"creator_id": float64(42), "other": "ignore"}
	set := make(map[int]struct{})

	collectIntID(obj, "creator_id", set)
	if _, ok := set[42]; !ok {
		t.Error("expected 42 in set")
	}

	// Zero ID should not be collected
	obj2 := map[string]any{"creator_id": float64(0)}
	set2 := make(map[int]struct{})
	collectIntID(obj2, "creator_id", set2)
	if len(set2) != 0 {
		t.Error("zero ID should not be collected")
	}

	// Missing field should not panic or add anything
	set3 := make(map[int]struct{})
	collectIntID(obj, "missing_field", set3)
	if len(set3) != 0 {
		t.Error("missing field should not add to set")
	}
}

// ===== collectIncidentIDs Tests =====

func TestCollectIncidentIDs(t *testing.T) {
	obj := map[string]any{
		"creator_id": float64(1),
		"closer_id":  float64(2),
		"channel_id": float64(100),
		"responders": []any{
			map[string]any{"person_id": float64(3)},
			map[string]any{"person_id": float64(4)},
		},
		"assigned_to": map[string]any{
			"person_ids": []any{float64(5), float64(6)},
		},
	}

	personIDs := make(map[int]struct{})
	channelIDs := make(map[int]struct{})
	collectIncidentIDs(obj, personIDs, channelIDs)

	expectedPersons := []int{1, 2, 3, 4, 5, 6}
	for _, id := range expectedPersons {
		if _, ok := personIDs[id]; !ok {
			t.Errorf("expected person ID %d in set", id)
		}
	}

	if _, ok := channelIDs[100]; !ok {
		t.Error("expected channel ID 100 in set")
	}
}

// ===== injectPersonName Tests =====

func TestInjectPersonName(t *testing.T) {
	obj := map[string]any{"creator_id": float64(1)}
	names := map[int]string{1: "Alice", 2: "Bob"}

	injectPersonName(obj, "creator_id", "creator_name", names)
	if obj["creator_name"] != "Alice" {
		t.Errorf("expected creator_name=Alice, got %v", obj["creator_name"])
	}

	// Unknown ID - no name injected
	obj2 := map[string]any{"creator_id": float64(99)}
	injectPersonName(obj2, "creator_id", "creator_name", names)
	if _, exists := obj2["creator_name"]; exists {
		t.Error("should not inject name for unknown ID")
	}

	// Zero ID - no name injected
	obj3 := map[string]any{"creator_id": float64(0)}
	injectPersonName(obj3, "creator_id", "creator_name", names)
	if _, exists := obj3["creator_name"]; exists {
		t.Error("should not inject name for zero ID")
	}
}

// ===== injectChannelName Tests =====

func TestInjectChannelName(t *testing.T) {
	names := map[int]string{100: "Alerts Channel"}

	// Normal injection
	obj := map[string]any{"channel_id": float64(100)}
	injectChannelName(obj, "channel_id", "channel_name", names)
	if obj["channel_name"] != "Alerts Channel" {
		t.Errorf("expected channel_name=Alerts Channel, got %v", obj["channel_name"])
	}

	// Existing name should NOT be overwritten
	obj2 := map[string]any{"channel_id": float64(100), "channel_name": "Already Set"}
	injectChannelName(obj2, "channel_id", "channel_name", names)
	if obj2["channel_name"] != "Already Set" {
		t.Errorf("existing name should not be overwritten, got %v", obj2["channel_name"])
	}
}

// ===== enrichResponders Tests =====

func TestEnrichResponders(t *testing.T) {
	obj := map[string]any{
		"responders": []any{
			map[string]any{"person_id": float64(1)},
			map[string]any{"person_id": float64(2)},
		},
	}
	names := map[int]string{1: "Alice", 2: "Bob"}

	enrichResponders(obj, names)

	responders := obj["responders"].([]any)
	first := responders[0].(map[string]any)
	if first["person_name"] != "Alice" {
		t.Errorf("first responder: got %v, want Alice", first["person_name"])
	}
	second := responders[1].(map[string]any)
	if second["person_name"] != "Bob" {
		t.Errorf("second responder: got %v, want Bob", second["person_name"])
	}
}

func TestEnrichResponders_NoResponders(_ *testing.T) {
	obj := map[string]any{}
	names := map[int]string{1: "Alice"}
	// Should not panic
	enrichResponders(obj, names)
}

// ===== enrichAssignedTo Tests =====

func TestEnrichAssignedTo(t *testing.T) {
	obj := map[string]any{
		"assigned_to": map[string]any{
			"person_ids": []any{float64(1), float64(99)},
		},
	}
	names := map[int]string{1: "Alice"}

	enrichAssignedTo(obj, names)

	assignedTo := obj["assigned_to"].(map[string]any)
	personNames := assignedTo["person_names"].([]string)
	if len(personNames) != 2 {
		t.Fatalf("expected 2 names, got %d", len(personNames))
	}
	if personNames[0] != "Alice" {
		t.Errorf("first name: got %q, want Alice", personNames[0])
	}
	if personNames[1] != "person_99" {
		t.Errorf("second name: got %q, want person_99", personNames[1])
	}
}

func TestEnrichAssignedTo_NoAssignedTo(_ *testing.T) {
	obj := map[string]any{}
	names := map[int]string{1: "Alice"}
	// Should not panic
	enrichAssignedTo(obj, names)
}
