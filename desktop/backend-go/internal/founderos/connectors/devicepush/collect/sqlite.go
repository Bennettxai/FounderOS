// Package collect is the device half of spec §4.18: it runs on a Mac (via
// cmd/founderos-collector), reads the device-local sources exactly as the operator
// OS's lib/connectors modules do, and builds the devicepush.Payload the
// backend receives. Everything here is read-only: SQLite databases are opened
// with mode=ro and nothing is ever written to a source.
//
// Only the collector imports this package. The backend (devicepush) never
// reads a device path.
package collect

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

// errTimedOut marks a read that did not finish in time.
var errTimedOut = errors.New("read timed out")

// openReadOnly opens a SQLite file read-only with a busy timeout, and checks
// it is actually a database (a lazy open would otherwise succeed on junk).
func openReadOnly(ctx context.Context, path string) (*sql.DB, error) {
	dsn := "file:" + (&url.URL{Path: path}).EscapedPath() + "?mode=ro&_pragma=busy_timeout(2000)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	var n int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM sqlite_master").Scan(&n); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}

// bounded runs fn with a hard deadline. Reading WhatsApp's live DB blocks
// when macOS withholds Full Disk Access; the TS forks a child it can SIGKILL.
// Here the read runs on a goroutine that is abandoned at the deadline (the
// collector is a short-lived process per tick, so the leak ends with it), and
// the caller gets an honest timeout instead of a hang.
func bounded[T any](ctx context.Context, d time.Duration, fn func(ctx context.Context) (T, error)) (T, error) {
	ctx, cancel := context.WithTimeout(ctx, d)
	defer cancel()
	type result struct {
		v   T
		err error
	}
	done := make(chan result, 1)
	go func() {
		v, err := fn(ctx)
		done <- result{v, err}
	}()
	select {
	case r := <-done:
		if r.err != nil && ctx.Err() != nil {
			var zero T
			return zero, errTimedOut
		}
		return r.v, r.err
	case <-ctx.Done():
		var zero T
		return zero, errTimedOut
	}
}

// rowMaps runs a query and returns each row as column → value.
func rowMaps(ctx context.Context, db *sql.DB, query string, args ...any) ([]map[string]any, error) {
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	var out []map[string]any
	for rows.Next() {
		vals := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return nil, err
		}
		m := make(map[string]any, len(cols))
		for i, c := range cols {
			if b, ok := vals[i].([]byte); ok {
				m[c] = string(b)
			} else {
				m[c] = vals[i]
			}
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// ---- small value helpers ------------------------------------------------------

func asString(v any) (string, bool) {
	s, ok := v.(string)
	return s, ok
}

func asFloat(v any) (float64, bool) {
	switch x := v.(type) {
	case int64:
		return float64(x), true
	case float64:
		return x, true
	case int:
		return float64(x), true
	}
	return 0, false
}

// jsString is JavaScript's String(v ?? ”).
func jsString(v any) string {
	switch x := v.(type) {
	case nil:
		return ""
	case string:
		return x
	case int64:
		return fmt.Sprint(x)
	case float64:
		return fmt.Sprint(x)
	default:
		return fmt.Sprint(x)
	}
}

// isoMillis is JavaScript's Date.toISOString().
func isoMillis(t time.Time) string { return t.UTC().Format("2006-01-02T15:04:05.000Z") }

// localLayouts are the timestamp shapes Wispr and friends write. Ones without
// a zone are local time, as JavaScript's Date parses them.
var zonedLayouts = []string{
	time.RFC3339Nano,
	"2006-01-02 15:04:05.999999999 -07:00",
	"2006-01-02 15:04:05.999999999 Z07:00",
	"2006-01-02 15:04:05.999999999Z07:00",
	"2006-01-02 15:04:05.999999999 -0700",
}

var localLayouts = []string{
	"2006-01-02T15:04:05.999999999",
	"2006-01-02 15:04:05.999999999",
	"2006-01-02T15:04",
}

// toISO turns a SQLite DATETIME (string, or epoch millis as a number) into an
// ISO timestamp; ok is false when it cannot be parsed.
func toISO(v any) (string, bool) {
	if f, isNum := asFloat(v); isNum {
		return isoMillis(time.UnixMilli(int64(f))), true
	}
	s, isStr := asString(v)
	if !isStr {
		return "", false
	}
	s = strings.TrimSpace(s)
	for _, l := range zonedLayouts {
		if t, err := time.Parse(l, s); err == nil {
			return isoMillis(t), true
		}
	}
	for _, l := range localLayouts {
		if t, err := time.ParseInLocation(l, s, local); err == nil {
			return isoMillis(t), true
		}
	}
	if t, err := time.ParseInLocation("2006-01-02", s, time.UTC); err == nil { // date-only is UTC in JS
		return isoMillis(t), true
	}
	return "", false
}

// firstText is the first non-blank string, trimmed.
func firstText(vals ...any) string {
	for _, v := range vals {
		if s, ok := v.(string); ok {
			if s = strings.TrimSpace(s); s != "" {
				return s
			}
		}
	}
	return ""
}

// commas formats an integer the way toLocaleString('en-US') does.
func commas(n int) string {
	s := fmt.Sprint(n)
	neg := strings.HasPrefix(s, "-")
	s = strings.TrimPrefix(s, "-")
	var b strings.Builder
	for i, r := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(r)
	}
	if neg {
		return "-" + b.String()
	}
	return b.String()
}

// truncRunes caps a string at n characters without splitting a rune.
func truncRunes(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n])
}
