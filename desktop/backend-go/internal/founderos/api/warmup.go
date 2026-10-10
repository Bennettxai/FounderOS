package api

import (
	"context"
	"log/slog"
	"os"
	"sync"
	"time"
)

// WarmOnBoot primes the shared comms and brand-deal caches once, delay after
// start (FounderOS v1 instrumentation.ts), so the first /comms or funnel lead
// lookup is not a cold 4-inbox IMAP scan. It runs only the heartbeat's warm
// list: reads into the shared per-Deps connectors, never a write. The
// snapshot sweep stays on the 15-minute heartbeat. FOUNDEROS_WARMUP=0 skips it.
// The returned WaitGroup lets tests wait for it.
func WarmOnBoot(ctx context.Context, d *Deps, delay time.Duration) *sync.WaitGroup {
	var wg sync.WaitGroup
	if os.Getenv("FOUNDEROS_WARMUP") == "0" {
		return &wg
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		select {
		case <-ctx.Done():
			return
		case <-time.After(delay):
		}
		start := time.Now()
		var inner sync.WaitGroup
		for _, w := range refreshSourcesFor(d).Warm {
			inner.Add(1)
			go func(w func(context.Context)) {
				defer inner.Done()
				defer func() {
					if r := recover(); r != nil {
						slog.Error("founderos warm-up panicked", "panic", r)
					}
				}()
				w(ctx)
			}(w)
		}
		inner.Wait()
		slog.Info("founderos caches warmed", "took", time.Since(start).Round(time.Millisecond))
	}()
	return &wg
}
