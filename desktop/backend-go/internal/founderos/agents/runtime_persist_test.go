package agents

import (
	"context"
	"testing"
	"time"
)

// ctxStore refuses a cancelled context the way pgx does, and records the
// persist deadline so a detached save is still bounded.
type ctxStore struct {
	*MemStore
	deadlines []time.Duration
}

func (s *ctxStore) check(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if dl, ok := ctx.Deadline(); ok {
		s.deadlines = append(s.deadlines, time.Until(dl))
	} else {
		s.deadlines = append(s.deadlines, -1)
	}
	return nil
}

func (s *ctxStore) InsertRun(ctx context.Context, r Run) error {
	if err := s.check(ctx); err != nil {
		return err
	}
	return s.MemStore.InsertRun(ctx, r)
}

func (s *ctxStore) InsertBroadcast(ctx context.Context, b Broadcast) error {
	if err := s.check(ctx); err != nil {
		return err
	}
	return s.MemStore.InsertBroadcast(ctx, b)
}

// hangsUp is an agent whose client disconnects while it works.
type hangsUp struct {
	id     string
	cancel context.CancelFunc
}

func (h *hangsUp) Meta() Meta { return Meta{ID: h.id} }
func (h *hangsUp) Run(ctx context.Context) (Result, error) {
	h.cancel()
	<-ctx.Done()
	return Result{OK: true, Summary: "finished as the client left"}, nil
}

func assertBoundedDetached(t *testing.T, s *ctxStore) {
	t.Helper()
	for _, d := range s.deadlines {
		if d <= 0 || d > 10*time.Second {
			t.Errorf("persist ctx deadline = %v, want a short bound (<=10s)", d)
		}
	}
}

// FounderOS v1 always inserts the run; a client hanging up mid-run must not
// lose it (the request ctx is cancelled by then).
func TestRunIsStoredWhenTheClientDisconnectsMidRun(t *testing.T) {
	store := &ctxStore{MemStore: NewMemStore()}
	ctx, cancel := context.WithCancel(context.Background())
	rt := New(store, &hangsUp{id: "a", cancel: cancel})
	run, err := rt.Run(ctx, "a")
	if err != nil {
		t.Fatalf("run lost: %v", err)
	}
	runs, _ := store.Recent(context.Background(), "a", 10)
	if len(runs) != 1 || runs[0].ID != run.ID {
		t.Fatalf("stored runs = %+v", runs)
	}
	assertBoundedDetached(t, store)
}

func TestBroadcastIsStoredWhenTheClientDisconnects(t *testing.T) {
	store := &ctxStore{MemStore: NewMemStore()}
	ctx, cancel := context.WithCancel(context.Background())
	rt := New(store, &hangsUp{id: "a", cancel: cancel}, &fakeAgent{id: "b", res: Result{OK: true, Summary: "hi"}})
	b, err := rt.Broadcast(ctx, "status?")
	if err != nil {
		t.Fatalf("broadcast lost: %v", err)
	}
	if len(store.broadcasts) != 1 || store.broadcasts[0].ID != b.ID || len(store.broadcasts[0].Replies) != 2 {
		t.Fatalf("stored broadcasts = %+v", store.broadcasts)
	}
	assertBoundedDetached(t, store)
}
