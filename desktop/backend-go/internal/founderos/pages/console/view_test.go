package console

import (
	"strings"
	"testing"
	"time"

	"github.com/rhl/businessos-backend/internal/founderos/connectors"
)

func viewInput() Input {
	loc := time.Local
	now := time.Date(2026, 9, 24, 18, 0, 0, 0, loc)
	at := func(h int) time.Time { return time.Date(2026, 9, 24, h, 0, 0, 0, loc) }
	ws := 5
	return Input{
		Now: now,
		Connections: []connectors.Status{
			{ID: "optimal-engine", State: connectors.StateConnected},
			{ID: "slack", State: connectors.StateError},
			{ID: "loom", State: connectors.StateNotConfigured},
		},
		Brain:  Brain{Connected: true, EnginesUp: 2, EnginesTotal: 2, Workspaces: &ws, Health: ptr(90), Status: "warnings"},
		Roster: &Roster{Active: 3, Total: 5},
		Recent: []Run{
			{ID: "r4", AgentID: "plaud", Summary: "20 recordings", OK: true, StartedAt: at(12), FinishedAt: at(12)},
			{ID: "r3", AgentID: "plaud", Summary: "20 recordings", OK: true, StartedAt: at(11), FinishedAt: at(11)},
			{ID: "r2", AgentID: "comms", Summary: "boom", OK: false, StartedAt: at(10), FinishedAt: at(10)},
			{ID: "r1", AgentID: "old", Summary: "yesterday", OK: true, StartedAt: now.Add(-30 * time.Hour), FinishedAt: now.Add(-30 * time.Hour)},
		},
		Feed: []Item{
			{Source: "email", Title: "a", TS: now.Add(-time.Hour).UTC().Format(time.RFC3339)},
			{Source: "whatsapp", Title: "b", TS: now.Add(-2 * time.Hour).UTC().Format(time.RFC3339)},
			{Source: "email", Title: "c", TS: now.Add(-48 * time.Hour).UTC().Format(time.RFC3339)},
		},
		Sources: []SourceState{{Source: "email", State: "ok"}, {Source: "whatsapp", State: "ok"}, {Source: "slack", State: "error", Detail: "HTTP 500"}},
		Charges: []Charge{
			{Amount: 600000, Currency: "usd", Description: "August retainer", Created: at(9).Unix()},
			{Amount: 100, Currency: "usd", Description: "old", Created: now.Add(-48 * time.Hour).Unix()},
		},
	}
}

func TestBuildComposesTheConsole(t *testing.T) {
	in := viewInput()
	in.Window = in.Recent
	v := Build(in)

	if v.Systems.Connected != 1 || v.Systems.Total != 3 || len(v.Systems.Bars) != 3 || v.Systems.Bars[1] != "error" {
		t.Fatalf("systems = %+v", v.Systems)
	}
	if v.Agents.Active == nil || *v.Agents.Active != 3 || *v.Agents.Total != 5 || len(v.Agents.Spark) != 7 {
		t.Fatalf("agents tile = %+v", v.Agents)
	}
	if v.Comms.Inbound != 2 || len(v.Comms.Spark) != 7 || v.FeedCount != 3 {
		t.Fatalf("comms = %+v feed %d", v.Comms, v.FeedCount)
	}
	if v.Volume.RunsToday != 3 || v.Volume.FailedToday != 1 {
		t.Fatalf("volume = %+v", v.Volume)
	}
	// Done: the plaud pair folds into one line ×2, plus today's charge; the
	// failed run and yesterday's run are not "done today".
	// Newest first: the 12:00 run, then the 09:00 charge.
	if len(v.Done) != 2 || v.Done[1].Head != "$" || v.Done[1].Tone != "accent" || !strings.Contains(v.Done[1].Body, "$6,000 · August retainer") {
		t.Fatalf("done = %+v", v.Done)
	}
	if v.Done[0].Head != "✓" || v.Done[0].Tone != "ok" || v.Done[0].Body != "plaud · 20 recordings ×2" {
		t.Fatalf("done run = %+v", v.Done[0])
	}
	// Count is every OK run today plus today's charges, before the ledger trims.
	if v.DoneCount != 3 {
		t.Fatalf("done count = %d", v.DoneCount)
	}
	if v.ChargedTodayCents == nil || *v.ChargedTodayCents != 600000 {
		t.Fatalf("charged today = %v", v.ChargedTodayCents)
	}
	// Waiting: 2 inbound + 1 failed run today + 1 connector down.
	if v.Attention.Count != 4 || v.Attention.Headline != "2 inbound · 1 failed run · 1 connector down" {
		t.Fatalf("attention = %+v", v.Attention)
	}
	if v.Hero[0] != (Segment{"1 run failed", "err"}) {
		t.Fatalf("hero = %+v", v.Hero)
	}
	// Prod's "brain 90/100" and "G-Brain health · 90 / 100", from the engine's score.
	if v.Hero[len(v.Hero)-1] != (Segment{"brain 90/100", "ok"}) {
		t.Fatalf("hero brain = %+v", v.Hero)
	}
	if m := v.Volume.Meters[3]; m.Label != "Optimal Engine health" || m.Display != "90 / 100" || *m.Frac != 0.9 {
		t.Fatalf("brain meter = %+v", m)
	}
	if len(v.Activity) != 14 || v.ActivityTotal != 4 {
		t.Fatalf("activity = %d points, total %d", len(v.Activity), v.ActivityTotal)
	}
	if v.Mix[0] != (Point{"email", 2}) || v.Mix[2] != (Point{"slack", 0}) {
		t.Fatalf("mix = %+v", v.Mix)
	}
}

func TestBuildKeepsUnknownsUnknown(t *testing.T) {
	in := viewInput()
	in.Roster, in.Recent, in.Window = nil, nil, nil
	in.RosterErr = "postgres unreachable"
	in.Charges, in.ChargesErr = nil, "stripe: no key"
	v := Build(in)
	if v.Agents.Active != nil || v.Agents.Total != nil || v.Agents.Spark != nil {
		t.Fatalf("unknown roster reads as numbers: %+v", v.Agents)
	}
	if v.ChargedTodayCents != nil {
		t.Fatal("unknown charges read as $0")
	}
	if v.Errors["roster"] != "postgres unreachable" || v.Errors["charges"] != "stripe: no key" {
		t.Fatalf("errors = %+v", v.Errors)
	}
	if v.Volume.Meters[1].Frac != nil {
		t.Fatalf("roster meter = %+v", v.Volume.Meters[1])
	}
}

// An unkeyed Stripe is not an error (FounderOS v1 home: stripeSnapshot().catch(() => null)):
// no error line, and the charged total stays unknown, never $0.
func TestBuildUnconfiguredStripeIsQuietAndUnknown(t *testing.T) {
	in := viewInput()
	in.Charges, in.ChargesErr, in.ChargesNotConfigured = nil, "", true
	v := Build(in)
	if _, ok := v.Errors["charges"]; ok {
		t.Fatalf("not configured surfaced as an error: %+v", v.Errors)
	}
	if v.ChargedTodayCents != nil {
		t.Fatal("unconfigured charges read as $0")
	}
}

func TestFormatMoney(t *testing.T) {
	for cents, want := range map[int64]string{600000: "$6,000", 123456789: "$1,234,568", 50: "$1", 0: "$0"} {
		if got := FormatMoney(cents, "usd"); got != want {
			t.Errorf("FormatMoney(%d) = %q, want %q", cents, got, want)
		}
	}
	if got := FormatMoney(250000, "eur"); got != "2,500 EUR" {
		t.Errorf("eur = %q", got)
	}
}
