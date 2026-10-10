// Package pagekit holds what the /os page ports share on the Go side: the
// slab view-model shapes (meters, series points, chips) that the Svelte kit
// renders, the hue strings they carry, en-US number and date formatting
// matching FounderOS v1's toLocaleString calls, and the workspace lookup every
// founderos_* read is scoped by.
package pagekit

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Hues. FounderOS v1's slab pages color meters with --ramp-*/--send-activity,
// which in the Monolith theme resolve to the grey ramp (brain-1 #f2f2f2 =
// text, brain-2 #9c9c9c = text-2). The bridge theme only has --bn-* tokens,
// so the same mixes are written over them here. Status hues stay status.
const (
	Ramp1        = "var(--bn-text-2)"
	Ramp2        = "color-mix(in oklab, var(--bn-text-2) 35%, var(--bn-text))"
	Ramp3        = "var(--bn-text)"
	Ramp4        = "color-mix(in oklab, var(--bn-text) 70%, var(--bn-accent))"
	SendActivity = "color-mix(in oklab, var(--bn-text-2) 70%, var(--bn-text))"
	Accent       = "var(--bn-accent)"
	OK           = "var(--bn-ok)"
	Warn         = "var(--bn-warn)"
	Err          = "var(--bn-err)"
)

// Meter is one MeterStack row. Frac nil means unknown: the kit says so and
// draws no bar (never a fake zero).
type Meter struct {
	Label   string   `json:"label"`
	Frac    *float64 `json:"frac"`
	Display string   `json:"display"`
	Hue     string   `json:"hue"`
}

func Known(label string, frac float64, display, hue string) Meter {
	return Meter{Label: label, Frac: &frac, Display: display, Hue: hue}
}

func Unknown(label, display, hue string) Meter {
	return Meter{Label: label, Display: display, Hue: hue}
}

// Point is a StepLine / DotMatrix column.
type Point struct {
	Label string  `json:"label"`
	Count float64 `json:"count"`
}

// Chip is a BigStat chip. Tone is ok|warn|err|accent or empty.
type Chip struct {
	Tone string `json:"tone,omitempty"`
	Text string `json:"text"`
}

// Plural is FounderOS v1's plural(n, word, many = word+'s').
func Plural(n int, one, many string) string {
	if many == "" {
		many = one + "s"
	}
	if n == 1 {
		return fmt.Sprintf("%d %s", n, one)
	}
	return fmt.Sprintf("%d %s", n, many)
}

// Frac is n/d, 0 when d is not positive.
func Frac(n, d float64) float64 {
	if d > 0 {
		return n / d
	}
	return 0
}

// Clamp01 bounds a fraction to [0, 1].
func Clamp01(f float64) float64 {
	if f != f || f < 0 {
		return 0
	}
	if f > 1 {
		return 1
	}
	return f
}

// Thousands is n.toLocaleString('en-US') for an integer.
func Thousands(n int) string {
	s := strconv.Itoa(n)
	neg := strings.HasPrefix(s, "-")
	if neg {
		s = s[1:]
	}
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

// Fixed is Number.prototype.toFixed.
func Fixed(f float64, digits int) string { return strconv.FormatFloat(f, 'f', digits, 64) }

// ShortDate is toLocaleDateString('en-US', {month:'short', day:'numeric'}) in t's zone.
func ShortDate(t time.Time) string { return t.Format("Jan 2") }

// Weekday is toLocaleDateString('en-US', {weekday:'short'}) in t's zone.
func Weekday(t time.Time) string { return t.Format("Mon") }

// ISODay is a UTC YYYY-MM-DD key.
func ISODay(t time.Time) string { return t.UTC().Format("2006-01-02") }

// ErrNoWorkspace: the founderos_* tables are workspace-scoped; without the
// workspace a page can read nothing, and says so instead of reading empty.
var ErrNoWorkspace = errors.New("workspace not bootstrapped on the bridge (run founderos-bootstrap)")

// WorkspaceID resolves a FounderOS workspace slug to its BusinessOS id.
func WorkspaceID(ctx context.Context, pool *pgxpool.Pool, slug string) (string, error) {
	if pool == nil {
		return "", ErrNoWorkspace
	}
	var id string
	err := pool.QueryRow(ctx, `SELECT id::text FROM workspaces WHERE slug = $1`, slug).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", fmt.Errorf("%w: %s", ErrNoWorkspace, slug)
	}
	return id, err
}

// UniqueLabels numbers repeated labels ("Sep 3", "Sep 3 2"), FounderOS v1's
// shortLabels rule: the kit's DotMatrix keys its columns by label.
func UniqueLabels(points []Point) []Point {
	seen := map[string]int{}
	out := make([]Point, len(points))
	for i, p := range points {
		seen[p.Label]++
		if n := seen[p.Label]; n > 1 {
			p.Label = p.Label + " " + strconv.Itoa(n)
		}
		out[i] = p
	}
	return out
}
