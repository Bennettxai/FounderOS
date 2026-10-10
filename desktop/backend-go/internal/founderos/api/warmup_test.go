package api

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

// FounderOS v1 primes its comms caches on boot (instrumentation.ts) so the first
// /comms after a restart is not an 18s cold IMAP scan. The bridge runs the
// heartbeat's warm list (reads into the shared connectors, no writes) once,
// shortly after start; FOUNDEROS_WARMUP=0 skips it.
func TestWarmOnBootRunsTheWarmListOnceAndCanBeSkipped(t *testing.T) {
	var ran int32
	prev := refreshSourcesFor
	t.Cleanup(func() { refreshSourcesFor = prev })
	refreshSourcesFor = func(*Deps) refreshSources {
		w := func(context.Context) { atomic.AddInt32(&ran, 1) }
		return refreshSources{Warm: []func(context.Context){w, w, w}}
	}
	t.Setenv("FOUNDEROS_WARMUP", "")
	WarmOnBoot(context.Background(), &Deps{}, 0).Wait()
	if n := atomic.LoadInt32(&ran); n != 3 {
		t.Fatalf("warmed %d, want 3", n)
	}
	t.Setenv("FOUNDEROS_WARMUP", "0")
	WarmOnBoot(context.Background(), &Deps{}, 0).Wait()
	if n := atomic.LoadInt32(&ran); n != 3 {
		t.Fatalf("FOUNDEROS_WARMUP=0 still warmed (%d)", n)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	WarmOnBoot(ctx, &Deps{}, time.Hour).Wait() // a cancelled start returns at once
}
