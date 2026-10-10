package agents

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

// CronSource lists cron definitions with their last run, and records runs.
type CronSource interface {
	Crons(ctx context.Context) ([]Cron, error)
	RecordCronRun(ctx context.Context, c Cron, r Run) error
}

// Scheduler fires due crons (FounderOS v1 lib/cron-scheduler.ts). Every job
// goes through guard.RunCron, so nothing fires on the bridge until
// FOUNDEROS_CRONS=1 at cutover.
type Scheduler struct {
	rt  *Runtime
	src CronSource
	// refusedAt remembers the occurrence a cron was last refused for, so a
	// refused cron is logged once per window rather than every tick.
	mu        sync.Mutex
	refusedAt map[string]time.Time
}

func NewScheduler(rt *Runtime, src CronSource) *Scheduler {
	return &Scheduler{rt: rt, src: src, refusedAt: map[string]time.Time{}}
}

// Tick runs every due cron once and returns how many ran.
func (s *Scheduler) Tick(ctx context.Context, now time.Time) int {
	rep, err := s.TickReport(ctx, now)
	if err != nil {
		slog.Error("founderos cron: list failed", "err", err)
		return 0
	}
	return len(rep.Ran)
}

// Start ticks every minute until ctx ends.
func (s *Scheduler) Start(ctx context.Context) {
	go func() {
		t := time.NewTicker(time.Minute)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case now := <-t.C:
				s.Tick(ctx, now)
			}
		}
	}()
}
