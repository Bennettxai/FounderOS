package agents

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/rhl/businessos-backend/internal/founderos/guard"
)

func TestTickReportNamesWhatRanAndWhatTheGuardRefused(t *testing.T) {
	now := time.Date(2026, 9, 30, 9, 1, 0, 0, CronZone)
	src := &memCrons{crons: []Cron{
		{ID: "cron-digest", AgentID: "comms-digest", Schedule: "0 9 * * *", Enabled: true},
		{ID: "cron-ghost", AgentID: "ghost", Schedule: "0 9 * * *", Enabled: true},
		{ID: "cron-later", AgentID: "comms-digest", Schedule: "0 18 * * *", Enabled: true, LastRunAt: ptr(now.Add(-time.Hour))},
	}}
	rt := New(NewMemStore(), &fakeAgent{id: "comms-digest", res: Result{OK: true, Summary: "digest"}})
	s := NewScheduler(rt, src)

	t.Setenv("FOUNDEROS_CRONS", "0")
	guard.Reset()
	rep, err := s.TickReport(context.Background(), now)
	if err != nil {
		t.Fatal(err)
	}
	// Off: every due cron is refused, the unknown-agent one included.
	if rep.Due != 2 || len(rep.Ran) != 0 || len(rep.Refused) != 2 || rep.Refused[0] != "cron-digest" || rep.Refused[1] != "cron-ghost" {
		t.Fatalf("off: %+v", rep)
	}

	t.Setenv("FOUNDEROS_CRONS", "1")
	rep, _ = s.TickReport(context.Background(), now)
	if rep.Due != 2 || len(rep.Ran) != 2 || rep.Ran[0].CronID != "cron-digest" || !rep.Ran[0].OK || rep.Ran[0].Summary != "digest" {
		t.Fatalf("on: %+v", rep)
	}
	// On: the unknown agent's firing is recorded as failed (TS tick), so the
	// cron reads failing rather than overdue forever.
	if g := rep.Ran[1]; g.CronID != "cron-ghost" || g.OK {
		t.Fatalf("ghost: %+v", g)
	}
	if n := len(src.recorded); n != 2 {
		t.Fatalf("recorded %d", n)
	}
}

type failingRuns struct{ *MemStore }

func (failingRuns) InsertRun(context.Context, Run) error { return errors.New("pool closed") }

type summaryCrons struct {
	memCrons
	runs []Run
}

func (m *summaryCrons) RecordCronRun(_ context.Context, c Cron, r Run) error {
	m.runs = append(m.runs, r)
	return nil
}

// app/api/cron/tick: the firing is recorded even when the run failed, so a
// cron that keeps erroring shows in the stats instead of re-firing every
// minute; the summary is cut to 2000 characters.
func TestTickRecordsTheFiringEvenWhenTheRunCannotBeSaved(t *testing.T) {
	t.Setenv("FOUNDEROS_CRONS", "1")
	guard.Reset()
	now := time.Date(2026, 9, 30, 9, 1, 0, 0, CronZone)
	src := &summaryCrons{memCrons: memCrons{crons: []Cron{{ID: "cron-digest", AgentID: "comms-digest", Schedule: "0 9 * * *", Enabled: true}}}}
	rt := New(failingRuns{NewMemStore()}, &fakeAgent{id: "comms-digest", res: Result{OK: true, Summary: strings.Repeat("x", 5000)}})
	rep, err := NewScheduler(rt, src).TickReport(context.Background(), now)
	if err != nil {
		t.Fatal(err)
	}
	if len(src.runs) != 1 {
		t.Fatalf("the failed firing was not recorded: %+v", rep)
	}
	r := src.runs[0]
	if r.OK || !strings.Contains(r.Summary, "pool closed") || len(r.Summary) > 2000 {
		t.Fatalf("recorded %+v (len %d)", r.OK, len(r.Summary))
	}

	src.runs = nil
	ok := New(NewMemStore(), &fakeAgent{id: "comms-digest", res: Result{OK: true, Summary: strings.Repeat("y", 5000)}})
	if _, err := NewScheduler(ok, src).TickReport(context.Background(), now); err != nil {
		t.Fatal(err)
	}
	if len(src.runs) != 1 || len(src.runs[0].Summary) != 2000 {
		t.Fatalf("summary not cut to 2000: %d", len(src.runs[0].Summary))
	}
}
