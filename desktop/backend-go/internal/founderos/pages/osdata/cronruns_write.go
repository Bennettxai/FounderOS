package osdata

import (
	"context"
	"time"
)

// InsertCronRun records one scheduled-task run (cronRuns.insert), whether the
// tick fired it or the operator pressed "run now".
func (s *Store) InsertCronRun(ctx context.Context, r CronRun) error {
	ws, err := s.founder(ctx)
	if err != nil {
		return err
	}
	st, err := ParseTime(r.StartedAt)
	if err != nil {
		return err
	}
	var fin *time.Time
	if r.FinishedAt != nil {
		t, err := ParseTime(*r.FinishedAt)
		if err != nil {
			return err
		}
		fin = &t
	}
	_, err = s.pool.Exec(ctx, `INSERT INTO founderos_cron_runs (id, workspace_id, cron_id, agent_id, started_at, finished_at, ok, summary) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
		r.ID, ws, r.CronID, r.AgentID, st, fin, r.OK, r.Summary)
	return err
}
