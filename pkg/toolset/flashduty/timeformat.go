package flashduty

import (
	"fmt"
	"strconv"
	"time"
)

// knownTimestampFields defines the set of field names that are treated as Unix second timestamps.
var knownTimestampFields = map[string]struct{}{
	"created_at":      {},
	"updated_at":      {},
	"resolved_at":     {},
	"acknowledged_at": {},
	"recovered_at":    {},
	"start_time":      {},
	"end_time":        {},
}

// FormatTimestamp converts a Unix second timestamp to "2006-01-02 15:04:05 MST".
// If loc is nil, UTC is used.
func FormatTimestamp(unix int64, loc *time.Location) string {
	if loc == nil {
		loc = time.UTC
	}
	return time.Unix(unix, 0).In(loc).Format("2006-01-02 15:04:05 MST")
}

// AddTimestampDisplay enriches data in-place by adding "<field>_display" keys for every
// known timestamp field whose value is a positive number.
// Nested maps and slices are processed recursively.
func AddTimestampDisplay(data map[string]interface{}, loc *time.Location) {
	for key, val := range data {
		// Recurse into nested maps.
		if nested, ok := val.(map[string]interface{}); ok {
			AddTimestampDisplay(nested, loc)
			continue
		}
		// Recurse into slices.
		if list, ok := val.([]interface{}); ok {
			AddTimestampDisplayToList(list, loc)
			continue
		}
		// Only process known timestamp fields.
		if _, isKnown := knownTimestampFields[key]; !isKnown {
			continue
		}
		var unix int64
		switch n := val.(type) {
		case float64:
			unix = int64(n)
		case int:
			unix = int64(n)
		case int64:
			unix = n
		default:
			continue
		}
		if unix > 0 {
			data[key+"_display"] = FormatTimestamp(unix, loc)
		}
	}
}

// AddTimestampDisplayToList calls AddTimestampDisplay on every map element in items.
func AddTimestampDisplayToList(items []interface{}, loc *time.Location) {
	for _, item := range items {
		if m, ok := item.(map[string]interface{}); ok {
			AddTimestampDisplay(m, loc)
		}
	}
}

// ParseFlexibleTime converts value to a Unix second timestamp (int64).
// Supported input types:
//   - float64, int, int64  – returned directly
//   - string               – parsed as a plain integer, or RFC3339 datetime
func ParseFlexibleTime(value interface{}) (int64, error) {
	switch v := value.(type) {
	case float64:
		return int64(v), nil
	case int:
		return int64(v), nil
	case int64:
		return v, nil
	case string:
		// Try plain integer first.
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			return n, nil
		}
		// Try RFC3339 / ISO 8601.
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			return t.Unix(), nil
		}
		return 0, fmt.Errorf("cannot parse time string %q: not a valid integer or RFC3339 datetime", v)
	default:
		return 0, fmt.Errorf("unsupported type %T for time value", value)
	}
}
