package agents

import (
	"context"
	"log/slog"
	"time"

	"github.com/rhl/businessos-backend/internal/founderos/guard"
)

// Fired is one cron that ran on a tick (the TS cron/tick "ran" row).
type Fired struct {
	CronID  string `json:"cronId"`
	AgentID string `json:"agentId"`
	OK      bool   `json:"ok"`
	Summary string `json:"summary"`
}

// TickReport is what one tick did: how many crons were due, which ran,
// which the bridge guard refused (FOUNDEROS_CRONS=0) and which were skipped
// because no agent implements them yet.
type TickReport struct {
	Due     int      `json:"due"`
	Ran     []Fired  `json:"ran"`
	Refused []string `json:"refused"`
	Skipped []string `json:"skipped"`
}

// Crons lists the cron definitions with their last run.
func (s *Scheduler) Crons(ctx context.Context) ([]Cron, error) { return s.src.Crons(ctx) }

// TickReport runs every due cron once, through guard.RunCron, and reports
// each outcome. A run's failure is recorded as a failed run, never dropped.
// cronSummaryMax is the TS summary.slice(0, 2000) on a recorded firing.
const cronSummaryMax = 2000

func (s *Scheduler) TickReport(ctx context.Context, now time.Time) (TickReport, error) {
	rep := TickReport{Ran: []Fired{}, Refused: []string{}, Skipped: []string{}}
	crons, err := s.src.Crons(ctx)
	if err != nil {
		return rep, err
	}
	due := DueCrons(crons, now)
	rep.Due = len(due)
	for _, c := range due {
		if !s.rt.Has(c.AgentID) {
			// Still fired and recorded below (the run fails with the unknown
			// agent), as the TS tick does, so the cron reads failing rather
			// than overdue forever.
			slog.Warn("founderos cron: no agent for cron", "cron", c.ID, "agent", c.AgentID)
		}
		c := c
		// While crons are off, refuse (and log) each occurrence once, not on
		// every one-minute tick.
		if !guard.CronsEnabled() {
			occ, _ := LastOccurrence(c.Schedule, now)
			s.mu.Lock()
			seen := s.refusedAt[c.ID].Equal(occ)
			s.refusedAt[c.ID] = occ
			s.mu.Unlock()
			if seen {
				rep.Refused = append(rep.Refused, c.ID)
				continue
			}
		}
		var fired Fired
		ran := guard.RunCron(c.ID, func() error {
			run, err := s.rt.Run(ctx, c.AgentID)
			if err != nil {
				run.OK, run.Summary = false, err.Error()
			}
			fired = Fired{CronID: c.ID, AgentID: c.AgentID, OK: run.OK, Summary: run.Summary}
			// Record the firing even on failure (app/api/cron/tick): a cron
			// that keeps erroring must show in the stats, not re-fire every
			// minute. The stored summary is cut to 2000 characters.
			if run.AgentID == "" {
				run.AgentID = c.AgentID
			}
			if run.FinishedAt.IsZero() {
				run.StartedAt, run.FinishedAt = now, time.Now().UTC()
			}
			if r := []rune(run.Summary); len(r) > cronSummaryMax {
				run.Summary = string(r[:cronSummaryMax])
			}
			if rerr := s.src.RecordCronRun(ctx, c, run); rerr != nil && err == nil {
				err = rerr
			}
			return err
		})
		if ran {
			rep.Ran = append(rep.Ran, fired)
		} else {
			rep.Refused = append(rep.Refused, c.ID)
		}
	}
	return rep, nil
}
