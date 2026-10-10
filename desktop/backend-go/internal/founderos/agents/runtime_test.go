package agents

import (
	"context"
	"errors"
	"math"
	"testing"
	"time"
)

type fakeAgent struct {
	id   string
	res  Result
	err  error
	pan  bool
	said string
}

func (f *fakeAgent) Meta() Meta { return Meta{ID: f.id, Name: "Agent " + f.id, DepartmentID: "tech"} }
func (f *fakeAgent) Run(context.Context) (Result, error) {
	if f.pan {
		panic("boom")
	}
	return f.res, f.err
}

type responder struct{ fakeAgent }

func (r *responder) Respond(_ context.Context, msg string) (Result, error) {
	r.said = msg
	return Result{OK: true, Summary: "heard: " + msg}, nil
}

func TestRunPersistsEveryOutcomeWithCost(t *testing.T) {
	store := NewMemStore()
	in, out := 1_000_000, 100_000
	rt := New(store,
		&fakeAgent{id: "ok", res: Result{OK: true, Summary: "done", Model: "claude-sonnet-5", TokensIn: &in, TokensOut: &out}},
		&fakeAgent{id: "fail", err: errors.New("no creds")},
		&fakeAgent{id: "panics", pan: true},
	)
	ctx := context.Background()

	run, err := rt.Run(ctx, "ok")
	if err != nil || !run.OK || run.Summary != "done" || run.CostUSD == nil || math.Abs(*run.CostUSD-4.5) > 1e-9 {
		t.Fatalf("ok run = %+v err=%v (1M in @3 + 0.1M out @15 = $4.50)", run, err)
	}
	run, _ = rt.Run(ctx, "fail")
	if run.OK || run.Summary != "no creds" || run.CostUSD != nil {
		t.Fatalf("failed run = %+v (unpriced runs keep cost null)", run)
	}
	run, _ = rt.Run(ctx, "panics")
	if run.OK || run.Summary == "" {
		t.Fatalf("a panicking agent must become a failed run, not crash the server: %+v", run)
	}
	if _, err := rt.Run(ctx, "nope"); !errors.Is(err, ErrUnknownAgent) {
		t.Fatalf("unknown agent: %v", err)
	}
	runs, _ := store.Recent(ctx, "", 10)
	if len(runs) != 3 {
		t.Fatalf("stored %d runs, want 3", len(runs))
	}
}

func TestBroadcastUsesRespondWhenAvailable(t *testing.T) {
	r := &responder{fakeAgent{id: "data"}}
	rt := New(NewMemStore(), r, &fakeAgent{id: "plain", res: Result{OK: true, Summary: "status"}})
	b, err := rt.Broadcast(context.Background(), "ship it?")
	if err != nil {
		t.Fatal(err)
	}
	if r.said != "ship it?" || len(b.Replies) != 2 {
		t.Fatalf("broadcast = %+v said=%q", b, r.said)
	}
}

func TestRegistryRejectsDuplicates(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("duplicate agent ids must panic at construction")
		}
	}()
	New(NewMemStore(), &fakeAgent{id: "x"}, &fakeAgent{id: "x"})
}

func TestPricingMatchesFounderosOS(t *testing.T) {
	cases := []struct {
		model    string
		in, out  int
		expected float64
	}{
		{"claude-sonnet-5", 1_000_000, 1_000_000, 18},
		{"anthropic/haiku-4.5", 1_000_000, 0, 0.8},
		{"opus-4.8", 0, 1_000_000, 75},
		{"", 1_000_000, 0, 3},          // unknown → Sonnet
		{"mystery", -5, 1_000_000, 15}, // negatives clamp to 0
	}
	for _, c := range cases {
		if got := RunCostUSD(c.in, c.out, c.model); math.Abs(got-c.expected) > 1e-9 {
			t.Errorf("%s: %v, want %v", c.model, got, c.expected)
		}
	}
}

func TestCronMatchingAndDue(t *testing.T) {
	loc := CronZone // crons read the operator's wall clock
	at := func(s string) time.Time { tm, _ := time.ParseInLocation("2006-01-02 15:04", s, loc); return tm }
	if !ValidCron("*/30 * * * *") || ValidCron("0 9 * *") || ValidCron("0 9 * * mon") {
		t.Fatal("ValidCron")
	}
	if !MatchesCron("0 9 * * 1-5", at("2026-09-30 09:00")) || MatchesCron("0 9 * * 1-5", at("2026-10-03 09:00")) {
		t.Fatal("weekday 09:00 (Wed yes, Sat no)")
	}
	if !MatchesCron("*/30 * * * *", at("2026-09-30 10:30")) || MatchesCron("*/30 * * * *", at("2026-09-30 10:31")) {
		t.Fatal("every 30 min")
	}
	if got := DescribeCron("0 9 * * 1-5"); got != "at 09:00, Mon-Fri" {
		t.Fatalf("describe = %q", got)
	}
	now := at("2026-09-30 09:05")
	last := at("2026-09-29 09:00")
	crons := []Cron{
		{ID: "digest", AgentID: "comms-digest", Schedule: "0 9 * * *", Enabled: true, LastRunAt: &last},
		{ID: "fresh", AgentID: "x", Schedule: "0 9 * * *", Enabled: true, LastRunAt: ptr(at("2026-09-30 09:01"))},
		{ID: "off", AgentID: "y", Schedule: "0 9 * * *", Enabled: false},
		{ID: "never", AgentID: "z", Schedule: "0 8 * * *", Enabled: true},
	}
	due := DueCrons(crons, now)
	if len(due) != 2 || due[0].ID != "digest" || due[1].ID != "never" {
		t.Fatalf("due = %+v", due)
	}
}

func ptr[T any](v T) *T { return &v }
