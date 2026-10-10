package agents

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/rhl/businessos-backend/internal/founderos/guard"
)

type memCrons struct {
	crons    []Cron
	recorded []string
}

func (m *memCrons) Crons(context.Context) ([]Cron, error) { return m.crons, nil }
func (m *memCrons) RecordCronRun(_ context.Context, c Cron, r Run) error {
	m.recorded = append(m.recorded, c.ID+":"+r.AgentID)
	return nil
}

func TestTickRunsDueCronsOnlyWhenCronsAreOn(t *testing.T) {
	now := time.Date(2026, 9, 30, 9, 1, 0, 0, CronZone)
	src := &memCrons{crons: []Cron{
		{ID: "cron-digest", AgentID: "comms-digest", Schedule: "0 9 * * *", Enabled: true},
		{ID: "cron-later", AgentID: "comms-digest", Schedule: "0 18 * * *", Enabled: true, LastRunAt: ptr(now.Add(-time.Hour))},
	}}
	store := NewMemStore()
	rt := New(store, &fakeAgent{id: "comms-digest", res: Result{OK: true, Summary: "digest"}})
	s := NewScheduler(rt, src)

	t.Setenv("FOUNDEROS_CRONS", "0")
	guard.Reset()
	if n := s.Tick(context.Background(), now); n != 0 {
		t.Fatalf("ran %d crons with FOUNDEROS_CRONS=0", n)
	}
	if runs, _ := store.Recent(context.Background(), "", 10); len(runs) != 0 {
		t.Fatalf("runs stored while off: %d", len(runs))
	}
	if r := guard.Refused(); len(r) != 1 || r[0].Action != "cron:cron-digest" {
		t.Fatalf("refusals = %+v", r)
	}

	t.Setenv("FOUNDEROS_CRONS", "1")
	if n := s.Tick(context.Background(), now); n != 1 {
		t.Fatalf("ran %d, want 1 (only the 09:00 job is due)", n)
	}
	if len(src.recorded) != 1 || src.recorded[0] != "cron-digest:comms-digest" {
		t.Fatalf("recorded = %v", src.recorded)
	}
}

// A cron pointing at an agent that does not exist is recorded as a failed
// firing (app/api/cron/tick: runtime.run throws, the row is written with the
// error), so it shows as failing instead of sitting overdue forever.
func TestTickRecordsCronsForUnknownAgentsAsFailed(t *testing.T) {
	t.Setenv("FOUNDEROS_CRONS", "1")
	guard.Reset()
	src := &summaryCrons{memCrons: memCrons{crons: []Cron{{ID: "c", AgentID: "ghost", Schedule: "* * * * *", Enabled: true}}}}
	rep, err := NewScheduler(New(NewMemStore()), src).TickReport(context.Background(), time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if len(src.runs) != 1 || src.runs[0].OK || !strings.Contains(src.runs[0].Summary, "ghost") || src.runs[0].AgentID != "ghost" {
		t.Fatalf("recorded %+v (report %+v)", src.runs, rep)
	}
}

// With crons off, a due cron is refused once per scheduled occurrence, not
// on every one-minute tick (the refusal log must not flood out real ones).
func TestRefusedCronIsLoggedOncePerOccurrence(t *testing.T) {
	t.Setenv("FOUNDEROS_CRONS", "0")
	guard.Reset()
	src := &memCrons{crons: []Cron{{ID: "cron-digest", AgentID: "comms-digest", Schedule: "0 9 * * *", Enabled: true}}}
	s := NewScheduler(New(NewMemStore(), &fakeAgent{id: "comms-digest"}), src)
	base := time.Date(2026, 9, 30, 9, 1, 0, 0, CronZone)
	for i := 0; i < 5; i++ {
		s.Tick(context.Background(), base.Add(time.Duration(i)*time.Minute))
	}
	if n := len(guard.Refused()); n != 1 {
		t.Fatalf("refusals = %d, want 1 for one occurrence", n)
	}
	s.Tick(context.Background(), base.Add(24*time.Hour)) // next day's window
	if n := len(guard.Refused()); n != 2 {
		t.Fatalf("refusals = %d, want 2 after the next occurrence", n)
	}
}
