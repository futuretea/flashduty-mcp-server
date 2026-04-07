package flashduty

import (
	"testing"
	"time"
)

// fixedUnix is used as a stable timestamp across tests.
// 1773684246 → 2026-03-16 00:04:06 UTC  /  2026-03-16 08:04:06 CST
const fixedUnix int64 = 1773684246

func shanghaiLoc(t *testing.T) *time.Location {
	t.Helper()
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		t.Fatalf("failed to load Asia/Shanghai: %v", err)
	}
	return loc
}

// ===== FormatTimestamp =====

func TestFormatTimestamp_Shanghai(t *testing.T) {
	loc := shanghaiLoc(t)
	got := FormatTimestamp(fixedUnix, loc)
	want := "2026-03-17 02:04:06 CST"
	if got != want {
		t.Errorf("FormatTimestamp shanghai = %q, want %q", got, want)
	}
}

func TestFormatTimestamp_UTC(t *testing.T) {
	got := FormatTimestamp(fixedUnix, time.UTC)
	want := "2026-03-16 18:04:06 UTC"
	if got != want {
		t.Errorf("FormatTimestamp UTC = %q, want %q", got, want)
	}
}

func TestFormatTimestamp_NilLocUsesUTC(t *testing.T) {
	got := FormatTimestamp(fixedUnix, nil)
	want := "2026-03-16 18:04:06 UTC"
	if got != want {
		t.Errorf("FormatTimestamp nil loc = %q, want %q", got, want)
	}
}

func TestFormatTimestamp_Zero(t *testing.T) {
	got := FormatTimestamp(0, time.UTC)
	want := "1970-01-01 00:00:00 UTC"
	if got != want {
		t.Errorf("FormatTimestamp zero = %q, want %q", got, want)
	}
}

// ===== AddTimestampDisplay =====

func TestAddTimestampDisplay_BasicFloat64(t *testing.T) {
	data := map[string]interface{}{
		"created_at": float64(fixedUnix),
	}
	AddTimestampDisplay(data, time.UTC)

	display, ok := data["created_at_display"]
	if !ok {
		t.Fatal("expected created_at_display to be set")
	}
	want := "2026-03-16 18:04:06 UTC"
	if display != want {
		t.Errorf("created_at_display = %q, want %q", display, want)
	}
}

func TestAddTimestampDisplay_MultipleFields(t *testing.T) {
	data := map[string]interface{}{
		"created_at":  float64(fixedUnix),
		"updated_at":  float64(fixedUnix + 60),
		"resolved_at": float64(fixedUnix + 120),
	}
	AddTimestampDisplay(data, time.UTC)

	for _, field := range []string{"created_at_display", "updated_at_display", "resolved_at_display"} {
		if _, ok := data[field]; !ok {
			t.Errorf("expected %s to be set", field)
		}
	}
}

func TestAddTimestampDisplay_NestedMap(t *testing.T) {
	nested := map[string]interface{}{
		"created_at": float64(fixedUnix),
	}
	data := map[string]interface{}{
		"incident_id": "INC-001",
		"detail":      nested,
	}
	AddTimestampDisplay(data, time.UTC)

	display, ok := nested["created_at_display"]
	if !ok {
		t.Fatal("expected nested created_at_display to be set")
	}
	want := "2026-03-16 18:04:06 UTC"
	if display != want {
		t.Errorf("nested created_at_display = %q, want %q", display, want)
	}
}

func TestAddTimestampDisplay_ListValue(t *testing.T) {
	item := map[string]interface{}{
		"created_at": float64(fixedUnix),
	}
	data := map[string]interface{}{
		"items": []interface{}{item},
	}
	AddTimestampDisplay(data, time.UTC)

	display, ok := item["created_at_display"]
	if !ok {
		t.Fatal("expected list item created_at_display to be set")
	}
	want := "2026-03-16 18:04:06 UTC"
	if display != want {
		t.Errorf("list item created_at_display = %q, want %q", display, want)
	}
}

func TestAddTimestampDisplay_ZeroValueNotAdded(t *testing.T) {
	data := map[string]interface{}{
		"created_at": float64(0),
	}
	AddTimestampDisplay(data, time.UTC)

	if _, ok := data["created_at_display"]; ok {
		t.Error("created_at_display should not be set when value is 0")
	}
}

func TestAddTimestampDisplay_NonTimestampFieldUnchanged(t *testing.T) {
	data := map[string]interface{}{
		"incident_id": "INC-999",
		"title":       "disk full",
		"created_at":  float64(fixedUnix),
	}
	AddTimestampDisplay(data, time.UTC)

	if _, ok := data["incident_id_display"]; ok {
		t.Error("incident_id_display should not be added")
	}
	if _, ok := data["title_display"]; ok {
		t.Error("title_display should not be added")
	}
	if _, ok := data["created_at_display"]; !ok {
		t.Error("created_at_display should be added")
	}
}

// ===== AddTimestampDisplayToList =====

func TestAddTimestampDisplayToList_MultipleItems(t *testing.T) {
	item1 := map[string]interface{}{"created_at": float64(fixedUnix)}
	item2 := map[string]interface{}{"created_at": float64(fixedUnix + 3600)}
	AddTimestampDisplayToList([]interface{}{item1, item2}, time.UTC)

	for i, item := range []map[string]interface{}{item1, item2} {
		if _, ok := item["created_at_display"]; !ok {
			t.Errorf("item[%d] missing created_at_display", i)
		}
	}
}

func TestAddTimestampDisplayToList_EmptyList(t *testing.T) {
	// Should not panic.
	AddTimestampDisplayToList([]interface{}{}, time.UTC)
}

// ===== ParseFlexibleTime =====

func TestParseFlexibleTime_Float64(t *testing.T) {
	got, err := ParseFlexibleTime(float64(fixedUnix))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != fixedUnix {
		t.Errorf("got %d, want %d", got, fixedUnix)
	}
}

func TestParseFlexibleTime_Int(t *testing.T) {
	got, err := ParseFlexibleTime(int(fixedUnix))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != fixedUnix {
		t.Errorf("got %d, want %d", got, fixedUnix)
	}
}

func TestParseFlexibleTime_Int64(t *testing.T) {
	got, err := ParseFlexibleTime(fixedUnix)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != fixedUnix {
		t.Errorf("got %d, want %d", got, fixedUnix)
	}
}

func TestParseFlexibleTime_IntegerString(t *testing.T) {
	got, err := ParseFlexibleTime("1773684246")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != fixedUnix {
		t.Errorf("got %d, want %d", got, fixedUnix)
	}
}

func TestParseFlexibleTime_RFC3339WithOffset(t *testing.T) {
	// "2026-03-16T00:00:00+08:00" == 2026-03-15T16:00:00Z
	got, err := ParseFlexibleTime("2026-03-16T00:00:00+08:00")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := time.Date(2026, 3, 15, 16, 0, 0, 0, time.UTC).Unix()
	if got != want {
		t.Errorf("got %d, want %d", got, want)
	}
}

func TestParseFlexibleTime_RFC3339UTC(t *testing.T) {
	got, err := ParseFlexibleTime("2026-03-16T00:00:00Z")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := time.Date(2026, 3, 16, 0, 0, 0, 0, time.UTC).Unix()
	if got != want {
		t.Errorf("got %d, want %d", got, want)
	}
}

func TestParseFlexibleTime_InvalidString(t *testing.T) {
	_, err := ParseFlexibleTime("not-a-time")
	if err == nil {
		t.Error("expected error for invalid string, got nil")
	}
}

func TestParseFlexibleTime_UnsupportedType(t *testing.T) {
	_, err := ParseFlexibleTime([]byte("12345"))
	if err == nil {
		t.Error("expected error for unsupported type, got nil")
	}
}
