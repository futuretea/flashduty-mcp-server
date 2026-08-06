package flashduty

import (
	"testing"
	"time"
)

// ===== parseTimeRange Tests =====

func TestParseTimeRange_ValidDurations(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name      string
		input     string
		wantDur   time.Duration
		tolerance time.Duration
	}{
		{"1 hour", "1h", 1 * time.Hour, 2 * time.Second},
		{"24 hours", "24h", 24 * time.Hour, 2 * time.Second},
		{"7 days", "7d", 7 * 24 * time.Hour, 2 * time.Second},
		{"30 days", "30d", 30 * 24 * time.Hour, 2 * time.Second},
		{"1 week", "1w", 7 * 24 * time.Hour, 2 * time.Second},
		{"6 months", "6M", 6 * 30 * 24 * time.Hour, 2 * time.Second},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start, end, err := parseTimeRange(tt.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			expectedStart := now.Add(-tt.wantDur).Unix()
			expectedEnd := now.Unix()

			if abs(start-expectedStart) > int64(tt.tolerance.Seconds()) {
				t.Errorf("start time: got %d, want ~%d (diff %ds)", start, expectedStart, start-expectedStart)
			}
			if abs(end-expectedEnd) > int64(tt.tolerance.Seconds()) {
				t.Errorf("end time: got %d, want ~%d (diff %ds)", end, expectedEnd, end-expectedEnd)
			}
		})
	}
}

func TestParseTimeRange_NamedRanges(t *testing.T) {
	t.Run("last_day", func(t *testing.T) {
		start, end, err := parseTimeRange("last_day")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if start >= end {
			t.Errorf("start (%d) should be before end (%d)", start, end)
		}
		// Verify it covers exactly one day (86399 seconds apart)
		diff := end - start
		if diff != 86399 {
			t.Errorf("last_day range should be 86399 seconds, got %d", diff)
		}
	})

	t.Run("last_week", func(t *testing.T) {
		start, end, err := parseTimeRange("last_week")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if start >= end {
			t.Errorf("start (%d) should be before end (%d)", start, end)
		}
		// Verify it covers exactly one week (7*86400 - 1 seconds apart)
		diff := end - start
		expected := int64(7*86400 - 1)
		if diff != expected {
			t.Errorf("last_week range should be %d seconds, got %d", expected, diff)
		}
	})

	t.Run("last_day_case_insensitive", func(t *testing.T) {
		_, _, err := parseTimeRange("Last_Day")
		if err != nil {
			t.Fatalf("should accept case-insensitive named ranges: %v", err)
		}
	})

	t.Run("week_before_last", func(t *testing.T) {
		start, end, err := parseTimeRange("week_before_last")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if start >= end {
			t.Errorf("start (%d) should be before end (%d)", start, end)
		}
		// Verify it covers exactly one week (7*86400 - 1 seconds apart)
		diff := end - start
		expected := int64(7*86400 - 1)
		if diff != expected {
			t.Errorf("week_before_last range should be %d seconds, got %d", expected, diff)
		}
		// Verify end time is exactly 1 second before lastWeekRange start
		lastWeekStart, _, _ := parseTimeRange("last_week")
		if end != lastWeekStart-1 {
			t.Errorf("week_before_last end (%d) should be exactly 1 second before last_week start (%d)", end, lastWeekStart)
		}
	})
}

// ===== weekBeforeLastRange Tests =====

func TestWeekBeforeLastRange(t *testing.T) {
	start, end := weekBeforeLastRange()

	// Verify start < end
	if start >= end {
		t.Errorf("start (%d) should be before end (%d)", start, end)
	}

	// Verify range is exactly 7 days - 1 second
	diff := end - start
	expected := int64(7*86400 - 1)
	if diff != expected {
		t.Errorf("weekBeforeLastRange should span %d seconds, got %d", expected, diff)
	}

	// Verify start is a Monday 00:00:00 in local time
	startTime := time.Unix(start, 0)
	if startTime.Weekday() != time.Monday {
		t.Errorf("weekBeforeLastRange start should be Monday, got %s", startTime.Weekday())
	}
	if startTime.Hour() != 0 || startTime.Minute() != 0 || startTime.Second() != 0 {
		t.Errorf("weekBeforeLastRange start should be 00:00:00, got %02d:%02d:%02d",
			startTime.Hour(), startTime.Minute(), startTime.Second())
	}

	// Verify end is a Sunday 23:59:59 in local time
	endTime := time.Unix(end, 0)
	if endTime.Weekday() != time.Sunday {
		t.Errorf("weekBeforeLastRange end should be Sunday, got %s", endTime.Weekday())
	}
	if endTime.Hour() != 23 || endTime.Minute() != 59 || endTime.Second() != 59 {
		t.Errorf("weekBeforeLastRange end should be 23:59:59, got %02d:%02d:%02d",
			endTime.Hour(), endTime.Minute(), endTime.Second())
	}

	// Verify end is exactly 1 second before lastWeekRange start
	lastWeekStart, _ := lastWeekRange()
	if end != lastWeekStart-1 {
		t.Errorf("weekBeforeLastRange end (%d) should be exactly 1 second before lastWeekRange start (%d)", end, lastWeekStart)
	}
}

func TestParseTimeRange_Invalid(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"empty", ""},
		{"non-numeric", "abc"},
		{"zero duration", "0h"},
		{"negative duration", "-1d"},
		{"unknown unit", "24x"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := parseTimeRange(tt.input)
			if err == nil {
				t.Error("expected error, got nil")
			}
		})
	}
}

// ===== resolveTimeParams Tests =====

func TestResolveTimeParams_WithTimeRange(t *testing.T) {
	params := map[string]any{"time_range": "24h"}
	start, end, err := resolveTimeParams(params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if start >= end {
		t.Errorf("start (%d) should be before end (%d)", start, end)
	}
}

func TestResolveTimeParams_WithExplicitTimes(t *testing.T) {
	params := map[string]any{
		"start_time": float64(1000),
		"end_time":   float64(2000),
	}
	start, end, err := resolveTimeParams(params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if start != 1000 || end != 2000 {
		t.Errorf("got start=%d end=%d, want start=1000 end=2000", start, end)
	}
}

func TestResolveTimeParams_MissingBoth(t *testing.T) {
	params := map[string]any{}
	_, _, err := resolveTimeParams(params)
	if err == nil {
		t.Error("expected error when both time_range and start/end are missing")
	}
}

func TestResolveTimeParams_MissingEndTime(t *testing.T) {
	params := map[string]any{"start_time": float64(1000)}
	_, _, err := resolveTimeParams(params)
	if err == nil {
		t.Error("expected error when end_time is missing")
	}
}

func TestResolveListTimeParams_EnforcesPublishedWindow(t *testing.T) {
	start := int64(1_000_000)
	_, _, err := resolveListTimeParams(map[string]any{
		"start_time": float64(start),
		"end_time":   float64(start + int64((31*24*time.Hour)/time.Second)),
	})
	if err != nil {
		t.Fatalf("31-day window returned error: %v", err)
	}

	_, _, err = resolveListTimeParams(map[string]any{
		"start_time": float64(start),
		"end_time":   float64(start + int64((31*24*time.Hour)/time.Second) + 1),
	})
	if err == nil {
		t.Fatal("expected error for a window longer than 31 days")
	}
}

// ===== Parameter Extraction Tests =====

func TestGetStringParam(t *testing.T) {
	m := map[string]any{"key": "value", "num": 42}

	if got := getStringParam(m, "key"); got != "value" {
		t.Errorf("got %q, want %q", got, "value")
	}
	if got := getStringParam(m, "missing"); got != "" {
		t.Errorf("got %q, want empty", got)
	}
	if got := getStringParam(m, "num"); got != "" {
		t.Errorf("got %q for non-string, want empty", got)
	}
}

func TestGetNumberParam(t *testing.T) {
	m := map[string]any{
		"float":  float64(3.14),
		"int":    42,
		"int64":  int64(100),
		"string": "not a number",
	}

	if n, ok := getNumberParam(m, "float"); !ok || n != 3.14 {
		t.Errorf("float: got (%f, %v), want (3.14, true)", n, ok)
	}
	if n, ok := getNumberParam(m, "int"); !ok || n != 42 {
		t.Errorf("int: got (%f, %v), want (42, true)", n, ok)
	}
	if n, ok := getNumberParam(m, "int64"); !ok || n != 100 {
		t.Errorf("int64: got (%f, %v), want (100, true)", n, ok)
	}
	if _, ok := getNumberParam(m, "string"); ok {
		t.Error("string: expected ok=false")
	}
	if _, ok := getNumberParam(m, "missing"); ok {
		t.Error("missing: expected ok=false")
	}
}

func TestGetIntParam(t *testing.T) {
	m := map[string]any{"val": float64(42)}
	if n, ok := getIntParam(m, "val"); !ok || n != 42 {
		t.Errorf("got (%d, %v), want (42, true)", n, ok)
	}
	if _, ok := getIntParam(m, "missing"); ok {
		t.Error("expected ok=false for missing key")
	}
}

func TestGetBoolParam(t *testing.T) {
	m := map[string]any{"flag": true, "str": "true"}

	if b, ok := getBoolParam(m, "flag"); !ok || !b {
		t.Errorf("got (%v, %v), want (true, true)", b, ok)
	}
	if _, ok := getBoolParam(m, "str"); ok {
		t.Error("string 'true' should not be accepted as bool")
	}
	if _, ok := getBoolParam(m, "missing"); ok {
		t.Error("expected ok=false for missing key")
	}
}

func TestGetStringSliceParam(t *testing.T) {
	m := map[string]any{
		"from_any":    []any{"a", "b", "c"},
		"from_string": []string{"x", "y"},
		"not_slice":   "single",
	}

	got := getStringSliceParam(m, "from_any")
	if len(got) != 3 || got[0] != "a" || got[1] != "b" || got[2] != "c" {
		t.Errorf("from_any: got %v, want [a b c]", got)
	}

	got = getStringSliceParam(m, "from_string")
	if len(got) != 2 || got[0] != "x" || got[1] != "y" {
		t.Errorf("from_string: got %v, want [x y]", got)
	}

	got = getStringSliceParam(m, "not_slice")
	if got != nil {
		t.Errorf("not_slice: got %v, want nil", got)
	}

	got = getStringSliceParam(m, "missing")
	if got != nil {
		t.Errorf("missing: got %v, want nil", got)
	}
}

func TestGetIntSliceParam(t *testing.T) {
	m := map[string]any{
		"ids":       []any{float64(1), float64(2), float64(3)},
		"mixed":     []any{float64(1), "not_int", int(3)},
		"not_slice": "single",
	}

	got := getIntSliceParam(m, "ids")
	if len(got) != 3 || got[0] != 1 || got[1] != 2 || got[2] != 3 {
		t.Errorf("ids: got %v, want [1 2 3]", got)
	}

	got = getIntSliceParam(m, "mixed")
	if len(got) != 2 || got[0] != 1 || got[1] != 3 {
		t.Errorf("mixed: got %v, want [1 3] (skipping non-numeric)", got)
	}

	got = getIntSliceParam(m, "missing")
	if got != nil {
		t.Errorf("missing: got %v, want nil", got)
	}
}

func TestGetMapParam(t *testing.T) {
	inner := map[string]any{"nested": true}
	m := map[string]any{"obj": inner, "str": "not_map"}

	got := getMapParam(m, "obj")
	if got == nil || got["nested"] != true {
		t.Errorf("obj: got %v, want map with nested=true", got)
	}
	if got := getMapParam(m, "str"); got != nil {
		t.Errorf("str: got %v, want nil", got)
	}
	if got := getMapParam(m, "missing"); got != nil {
		t.Errorf("missing: got %v, want nil", got)
	}
}

// ===== setOptional* Tests =====

func TestSetOptionalString(t *testing.T) {
	params := map[string]any{"title": "hello", "empty": ""}
	body := map[string]any{}

	setOptionalString(params, body, "title")
	if body["title"] != "hello" {
		t.Errorf("expected title=hello, got %v", body["title"])
	}

	setOptionalString(params, body, "empty")
	if _, exists := body["empty"]; exists {
		t.Error("empty string should not be set")
	}

	setOptionalString(params, body, "missing")
	if _, exists := body["missing"]; exists {
		t.Error("missing key should not be set")
	}
}

func TestSetOptionalIntSlice(t *testing.T) {
	params := map[string]any{"ids": []any{float64(1), float64(2)}}
	body := map[string]any{}

	setOptionalIntSlice(params, body, "ids")
	ids, ok := body["ids"].([]int)
	if !ok || len(ids) != 2 {
		t.Errorf("expected [1 2], got %v", body["ids"])
	}

	setOptionalIntSlice(params, body, "missing")
	if _, exists := body["missing"]; exists {
		t.Error("missing key should not be set")
	}
}

func TestSetOptionalStringSlice(t *testing.T) {
	params := map[string]any{"tags": []any{"a", "b"}}
	body := map[string]any{}

	setOptionalStringSlice(params, body, "tags")
	tags, ok := body["tags"].([]string)
	if !ok || len(tags) != 2 {
		t.Errorf("expected [a b], got %v", body["tags"])
	}

	setOptionalStringSlice(params, body, "missing")
	if _, exists := body["missing"]; exists {
		t.Error("missing key should not be set")
	}
}

func TestSetOptionalMap(t *testing.T) {
	params := map[string]any{"labels": map[string]any{"env": "prod"}}
	body := map[string]any{}

	setOptionalMap(params, body, "labels")
	labels, ok := body["labels"].(map[string]any)
	if !ok || labels["env"] != "prod" {
		t.Errorf("expected labels with env=prod, got %v", body["labels"])
	}

	setOptionalMap(params, body, "missing")
	if _, exists := body["missing"]; exists {
		t.Error("missing key should not be set")
	}
}

// ===== buildPaginationParams Tests =====

func TestBuildPaginationParams(t *testing.T) {
	t.Run("with both", func(t *testing.T) {
		params := map[string]any{"limit": float64(50), "p": float64(3)}
		body := map[string]any{}
		buildPaginationParams(params, body)
		if body["limit"] != 50 {
			t.Errorf("limit: got %v, want 50", body["limit"])
		}
		if body["p"] != 3 {
			t.Errorf("p: got %v, want 3", body["p"])
		}
	})

	t.Run("with none", func(t *testing.T) {
		params := map[string]any{}
		body := map[string]any{}
		buildPaginationParams(params, body)
		if _, exists := body["limit"]; exists {
			t.Error("limit should not be set")
		}
		if _, exists := body["p"]; exists {
			t.Error("p should not be set")
		}
	})
}

// ===== resolveTimeParams ISO 8601 Tests =====

func TestResolveTimeParams_ISO8601(t *testing.T) {
	params := map[string]any{
		"start_time": "2026-03-16T00:00:00+08:00",
		"end_time":   "2026-03-22T23:59:59+08:00",
	}
	startTime, endTime, err := resolveTimeParams(params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// 2026-03-16 00:00:00 +08:00 => Unix 1773590400
	wantStart := int64(1773590400)
	// 2026-03-22 23:59:59 +08:00 => Unix 1774195199
	wantEnd := int64(1774195199)
	if startTime != wantStart {
		t.Errorf("start_time: got %d, want %d", startTime, wantStart)
	}
	if endTime != wantEnd {
		t.Errorf("end_time: got %d, want %d", endTime, wantEnd)
	}
}

func TestResolveTimeParams_MixedTypes(t *testing.T) {
	params := map[string]any{
		"start_time": float64(1773590400),
		"end_time":   "2026-03-22T23:59:59+08:00",
	}
	startTime, endTime, err := resolveTimeParams(params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	wantStart := int64(1773590400)
	wantEnd := int64(1774195199)
	if startTime != wantStart {
		t.Errorf("start_time: got %d, want %d", startTime, wantStart)
	}
	if endTime != wantEnd {
		t.Errorf("end_time: got %d, want %d", endTime, wantEnd)
	}
}

func TestResolveTimeParams_InvalidISO8601(t *testing.T) {
	params := map[string]any{
		"start_time": "not-a-date",
		"end_time":   "2026-03-22T23:59:59+08:00",
	}
	_, _, err := resolveTimeParams(params)
	if err == nil {
		t.Error("expected error for invalid start_time, got nil")
	}
}

// ===== Helper =====

func abs(n int64) int64 {
	if n < 0 {
		return -n
	}
	return n
}
