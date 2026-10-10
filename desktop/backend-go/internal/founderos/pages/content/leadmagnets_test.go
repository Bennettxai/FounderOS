package content

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/rhl/businessos-backend/internal/founderos/pages/pagekit"
)

// Ported from FounderOS v1 tests/lead-magnet-volume.test.ts.
var lmRows = []LeadMagnet{
	{Name: "Agent Stack", Status: "live", Captures: "email", Destination: "Beehiiv · newsletter + Cohort 2", Source: "IG reel", LaunchedAt: "2026-08-12"},
	{Name: "Call Kit", Status: "live", Captures: "booking", Destination: "Calendar · discovery", Source: "TikTok", LaunchedAt: "2026-09-20"},
	{Name: "Offer Doc", Status: "draft", Captures: "none", Destination: "Beehiiv", Source: "IG reel", LaunchedAt: "2026-09-01"},
	{Name: "Old Page", Status: "archived", Captures: "email", Destination: "Beehiiv · list", Source: "X thread", LaunchedAt: "2026-03-01"},
}

func TestMagnetVolumeCard(t *testing.T) {
	v := MagnetVolume(lmRows, today, 0)
	if v.Headline != 2 || v.Total != 4 || v.Counts != (StatusCounts{Live: 2, Draft: 1, Archived: 1}) {
		t.Fatalf("%+v", v)
	}
	if len(v.Chips) != 2 || v.Chips[0] != (pagekit.Chip{Tone: "warn", Text: "1 draft"}) || v.Chips[1] != (pagekit.Chip{Text: "1 archived"}) {
		t.Fatalf("%+v", v.Chips)
	}
	if v.Caption != "of 4 landing pages shipped" {
		t.Fatal(v.Caption)
	}
	raw, _ := json.Marshal(v.Meters)
	want := `[{"label":"Live (2/4)","frac":0.5,"display":"2 of 4","hue":"var(--bn-ok)"},{"label":"Capturing email (2/4)","frac":0.5,"display":"2 of 4","hue":"var(--bn-text-2)"},{"label":"Capturing bookings (1/4)","frac":0.25,"display":"1 of 4","hue":"var(--bn-text)"},{"label":"No capture yet (1/4)","frac":0.25,"display":"1 of 4","hue":"var(--bn-warn)"}]`
	if string(raw) != want {
		t.Fatalf("%s", raw)
	}
	if v.Foot != "2 destinations · 3 campaigns" {
		t.Fatal(v.Foot)
	}
}

func TestMagnetVolumeWeeklyLine(t *testing.T) {
	v := MagnetVolume(lmRows, today, 12)
	n := len(v.Series)
	if n != 12 || v.Series[n-1] != (pagekit.Point{Label: "Sep 24", Count: 4}) || v.Series[n-2] != (pagekit.Point{Label: "Sep 17", Count: 3}) || v.Series[0] != (pagekit.Point{Label: "Jul 9", Count: 1}) {
		t.Fatalf("%+v", v.Series)
	}
	if v.ShippedInWindow != 3 {
		t.Fatal(v.ShippedInWindow)
	}
}

func TestMagnetVolumeCapturesAndDestinations(t *testing.T) {
	v := MagnetVolume(lmRows, today, 12)
	if len(v.Captures) != 3 || v.Captures[0] != (pagekit.Point{Label: "Email", Count: 2}) || v.Captures[1] != (pagekit.Point{Label: "Booking", Count: 1}) || v.Captures[2] != (pagekit.Point{Label: "None", Count: 1}) {
		t.Fatalf("%+v", v.Captures)
	}
	if len(v.Destinations) != 2 || v.Destinations[0] != (pagekit.Point{Label: "Beehiiv", Count: 3}) || v.Destinations[1] != (pagekit.Point{Label: "Calendar", Count: 1}) {
		t.Fatalf("%+v", v.Destinations)
	}
}

func TestMagnetVolumeInsight(t *testing.T) {
	v := MagnetVolume(lmRows, today, 12)
	if v.Insight != (Insight{Value: 4, Headline: "4 days since Call Kit shipped.", Body: "2 of 4 live · ticks light with the live share.", Frac: 0.5}) {
		t.Fatalf("%+v", v.Insight)
	}
	a, b := lmRows[0], lmRows[1]
	a.LaunchedAt, b.Name, b.LaunchedAt = today, "Soon", "2026-10-01"
	v = MagnetVolume([]LeadMagnet{a, b}, today, 12)
	if v.Insight.Value != 0 || v.Insight.Headline != "Agent Stack went live today." {
		t.Fatalf("%+v", v.Insight)
	}
}

func TestMagnetVolumeEmpty(t *testing.T) {
	e := MagnetVolume(nil, today, 12)
	if e.Headline != 0 || len(e.Chips) != 0 || len(e.Meters) != 0 || len(e.Destinations) != 0 || e.Caption != "no landing pages recorded yet" {
		t.Fatalf("%+v", e)
	}
	for _, s := range e.Series {
		if s.Count != 0 {
			t.Fatal("flat line expected")
		}
	}
	if e.Insight.Display != "none" || e.Insight.Headline != "Nothing shipped yet." || e.Insight.Frac != 0 {
		t.Fatalf("%+v", e.Insight)
	}
}

func TestFilterLeadMagnets(t *testing.T) {
	if strings.Join(Filters, ",") != "all,live,draft,paused,archived" {
		t.Fatal(Filters)
	}
	names := func(rs []LeadMagnet) []string {
		out := []string{}
		for _, r := range rs {
			out = append(out, r.Name)
		}
		return out
	}
	if got := names(FilterRows(lmRows, "live")); strings.Join(got, ",") != "Agent Stack,Call Kit" {
		t.Fatal(got)
	}
	if len(FilterRows(lmRows, "draft")) != 1 || len(FilterRows(lmRows, "")) != 4 || len(FilterRows(lmRows, "bogus")) != 4 {
		t.Fatal("filter")
	}
	if FilterOf("bogus") != "all" || FilterOf("paused") != "paused" {
		t.Fatal("FilterOf")
	}
}

// Ported from tests/lead-magnets-route.test.ts and lead-magnet-actions.test.ts
// (the validation half; the Postgres half is in the api package tests).
func TestSlugify(t *testing.T) {
	for in, want := range map[string]string{
		"The Claude Trading Setup": "the-claude-trading-setup",
		"  --Hello, World!! ":      "hello-world",
		"!!!":                      "",
	} {
		if got := Slugify(in); got != want {
			t.Errorf("Slugify(%q) = %q, want %q", in, got, want)
		}
	}
	if len(Slugify(strings.Repeat("abc ", 40))) > 60 {
		t.Fatal("slug must cap at 60")
	}
}

func TestUniqueID(t *testing.T) {
	taken := map[string]bool{"the-claude-trading-setup": true, "the-claude-trading-setup-2": true}
	if got := UniqueID("The Claude Trading Setup", taken); got != "the-claude-trading-setup-3" {
		t.Fatal(got)
	}
	if got := UniqueID("!!!", nil); got != "lead-magnet" {
		t.Fatal(got)
	}
}

func TestParseCreateDefaultsAndRejects(t *testing.T) {
	m, err := ParseCreate([]byte(`{"name":"The Claude Trading Setup","url":"https://founderos-claude-trading.vercel.app","offer":"x","source":"IG reel"}`), today)
	if err != nil {
		t.Fatal(err)
	}
	if m.Status != "live" || m.Captures != "email" || m.LaunchedAt != today || m.Origin != "os" || m.Destination != "" {
		t.Fatalf("%+v", m)
	}
	for _, bad := range []string{
		`{"name":"Broken","url":"not-a-url"}`,
		`{"url":"https://example.com"}`,
		`{"name":"","url":"https://example.com"}`,
		`{"name":"x","url":"https://example.com","status":"sideways"}`,
		`{"name":"x","url":"https://example.com","launchedAt":"Sept 1"}`,
		`{"name":"x","url":"https://example.com","captures":"sms"}`,
		`not json`,
		`[]`,
	} {
		if _, err := ParseCreate([]byte(bad), today); err == nil {
			t.Errorf("accepted %s", bad)
		}
	}
}

func TestApplyPatchEditsOnlyWhatWasSent(t *testing.T) {
	cur := LeadMagnet{ID: "retire-me", Name: "Retire Me", URL: "https://example.com/x", Status: "live", Captures: "email", LaunchedAt: "2026-08-14", Origin: "os"}
	got, err := ApplyPatch(cur, []byte(`{"status":"archived","id":"hijack","origin":"seed"}`))
	if err != nil || got.Status != "archived" || got.Name != "Retire Me" || got.ID != "retire-me" || got.Origin != "os" {
		t.Fatalf("%+v %v", got, err)
	}
	got, err = ApplyPatch(cur, []byte(`{"name":"Typo Name","url":"https://example.com/fixed","notes":"renamed"}`))
	if err != nil || got.Name != "Typo Name" || got.URL != "https://example.com/fixed" || got.Notes != "renamed" {
		t.Fatalf("%+v %v", got, err)
	}
	for _, bad := range []string{`{"status":"sideways"}`, `{"url":"nope"}`, `{"name":null}`, `{"launchedAt":"2026"}`, `nope`} {
		if _, err := ApplyPatch(cur, []byte(bad)); err == nil {
			t.Errorf("accepted %s", bad)
		}
	}
}
