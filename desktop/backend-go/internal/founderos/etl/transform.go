package etl

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"
)

func jsonUnmarshal(s string, v any) error { return json.Unmarshal([]byte(s), v) }

func jsonMarshal(v any) (string, error) {
	b, err := json.Marshal(v)
	return string(b), err
}

// convStats counts the conversions the report must show per table.
type convStats struct {
	d2, d6, noOffset int
}

var noOffsetLayouts = []string{
	"2006-01-02T15:04:05.999999999",
	"2006-01-02T15:04:05",
	"2006-01-02T15:04",
	"2006-01-02 15:04:05.999999999",
	"2006-01-02 15:04:05",
}

// ParseInstant applies the table map's instant rules: D1 (ISO with Z or an
// offset) as-is, D2 (date only) as midnight UTC, and an ISO string without an
// offset read as UTC. It reports which rule fired.
func ParseInstant(s string) (t time.Time, rule string, err error) {
	s = strings.TrimSpace(s)
	if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
		return t.UTC(), "D1", nil
	}
	if t, err := time.Parse("2006-01-02", s); err == nil {
		return t.UTC(), "D2", nil
	}
	for _, l := range noOffsetLayouts {
		if t, err := time.Parse(l, s); err == nil {
			return t.UTC(), "no-offset", nil
		}
	}
	return time.Time{}, "", fmt.Errorf("unparseable instant %q", s)
}

// ParseDay is D3: a calendar-day key, strictly YYYY-MM-DD.
func ParseDay(s string) (time.Time, error) {
	t, err := time.Parse("2006-01-02", strings.TrimSpace(s))
	if err != nil {
		return time.Time{}, fmt.Errorf("not a YYYY-MM-DD day: %q", s)
	}
	return t, nil
}

// ParseMonth is D4: YYYY-MM becomes the first of that month.
func ParseMonth(s string) (time.Time, error) {
	t, err := time.Parse("2006-01", strings.TrimSpace(s))
	if err != nil {
		return time.Time{}, fmt.Errorf("not a YYYY-MM month: %q", s)
	}
	return t, nil
}

func asString(v any) (string, bool) {
	switch x := v.(type) {
	case string:
		return x, true
	case []byte:
		return string(x), true
	}
	return "", false
}

func asInt(v any) (int64, bool) {
	switch x := v.(type) {
	case int64:
		return x, true
	case float64:
		if x == math.Trunc(x) {
			return int64(x), true
		}
	case string:
		var n int64
		if _, err := fmt.Sscan(x, &n); err == nil {
			return n, true
		}
	}
	return 0, false
}

func asFloat(v any) (float64, bool) {
	switch x := v.(type) {
	case float64:
		return x, true
	case int64:
		return float64(x), true
	}
	return 0, false
}

// convert turns one SQLite value into the Go value bound for the target
// column, applying the date and JSON rules.
func convert(v any, k kind, cs *convStats) (any, error) {
	if v == nil {
		if k.nullable() {
			return nil, nil
		}
		return nil, fmt.Errorf("NULL in a NOT NULL column")
	}
	switch k {
	case kText, kTextN:
		s, ok := asString(v)
		if !ok {
			return fmt.Sprint(v), nil
		}
		return s, nil
	case kInt, kIntN, kBig:
		n, ok := asInt(v)
		if !ok {
			return nil, fmt.Errorf("not an integer: %v", v)
		}
		if k != kBig && (n > math.MaxInt32 || n < math.MinInt32) {
			return nil, fmt.Errorf("integer %d overflows INTEGER", n)
		}
		return n, nil
	case kFloat, kFloatN:
		f, ok := asFloat(v)
		if !ok {
			return nil, fmt.Errorf("not a number: %v", v)
		}
		return f, nil
	case kBool:
		n, ok := asInt(v)
		if !ok || (n != 0 && n != 1) {
			return nil, fmt.Errorf("not a 0/1 flag: %v", v)
		}
		return n == 1, nil
	case kJSONArr, kJSONStrs, kJSONObj:
		s, ok := asString(v)
		if !ok {
			return nil, fmt.Errorf("JSON column holds a non-text value")
		}
		if err := checkJSON(s, k); err != nil {
			return nil, err
		}
		return s, nil
	case kTS, kTSN:
		s, _ := asString(v)
		if strings.TrimSpace(s) == "" {
			if k == kTSN {
				cs.d6++
				return nil, nil
			}
			return nil, fmt.Errorf("blank instant in a NOT NULL column")
		}
		t, rule, err := ParseInstant(s)
		if err != nil {
			return nil, err
		}
		switch rule {
		case "D2":
			cs.d2++
		case "no-offset":
			cs.noOffset++
		}
		return t, nil
	case kDate:
		s, _ := asString(v)
		return ParseDay(s)
	case kMonth:
		s, _ := asString(v)
		return ParseMonth(s)
	}
	return nil, fmt.Errorf("unknown column kind %d", k)
}

// checkJSON parses the value and checks the structural shape the Zod schema
// requires (array, array of strings, or object).
func checkJSON(s string, k kind) error {
	var v any
	if err := json.Unmarshal([]byte(s), &v); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}
	switch k {
	case kJSONObj:
		if _, ok := v.(map[string]any); !ok {
			return fmt.Errorf("JSON is not an object")
		}
	case kJSONArr, kJSONStrs:
		arr, ok := v.([]any)
		if !ok {
			return fmt.Errorf("JSON is not an array")
		}
		if k == kJSONStrs {
			for i, e := range arr {
				if _, ok := e.(string); !ok {
					return fmt.Errorf("JSON array element %d is not a string", i)
				}
			}
		}
	}
	return nil
}

// keyString renders a transformed key tuple for collision detection.
func keyString(vals []any) string {
	parts := make([]string, len(vals))
	for i, v := range vals {
		if t, ok := v.(time.Time); ok {
			parts[i] = t.UTC().Format(time.RFC3339Nano)
			continue
		}
		parts[i] = fmt.Sprint(v)
	}
	return strings.Join(parts, "\x1f")
}
