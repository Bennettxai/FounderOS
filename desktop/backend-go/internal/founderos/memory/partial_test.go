package memory

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

// One engine asleep (the MacBook) must not sink the whole read: the hits of
// the workspaces that answered come back, with the ones that did not named
// (lib/brain-federated.ts tolerates one half down).
func TestSearchReturnsPartialHitsWhenOneEngineIsDown(t *testing.T) {
	hub := &engine{results: map[string]string{"default:founderos": `[{"id":"f1","title":"SOP","uri":"u1"}]`}}
	mac := &engine{down: true}
	r := newRouter(t, hub, mac)
	hits, err := r.Search(context.Background(), "sop", []string{"founderos", "personal"}, 5)
	var pe *PartialError
	if !errors.As(err, &pe) || !reflect.DeepEqual(pe.Failed, []string{"personal"}) {
		t.Fatalf("err = %v, want a PartialError naming personal", err)
	}
	var un *UnreachableError
	if !errors.As(err, &un) || un.Engine != "macbook" {
		t.Fatalf("the partial error must still carry the unreachable engine: %v", err)
	}
	if len(hits) != 1 || hits[0].ID != "f1" {
		t.Fatalf("hits = %+v, want the hub's answer", hits)
	}
}

// Unreachable-all is still an error, never an empty or partial answer.
func TestSearchWithEveryEngineDownIsAnErrorNotPartial(t *testing.T) {
	hub, mac := &engine{down: true}, &engine{down: true}
	r := newRouter(t, hub, mac)
	hits, err := r.Search(context.Background(), "q", []string{"founderos", "personal"}, 5)
	var pe *PartialError
	if err == nil || errors.As(err, &pe) || hits != nil {
		t.Fatalf("hits=%v err=%v, want a plain error", hits, err)
	}
}

func TestRetrieveReportsFailedWorkspacesInsteadOfFailing(t *testing.T) {
	hub := &engine{results: map[string]string{"default:founderos": `[{"id":"f1","title":"SOP"}]`}}
	mac := &engine{down: true}
	r := newRouter(t, hub, mac)
	got := r.Retrieve(context.Background(), "sop", []string{"founderos", "personal"}, RetrieveOptions{Rerank: func(context.Context, string, []string) ([]float64, bool) { return nil, false }})
	if got.Error != "" || len(got.Hits) != 1 || !reflect.DeepEqual(got.FailedWorkspaces, []string{"personal"}) || got.Degraded == "" {
		t.Fatalf("retrieval = %+v", got)
	}

	// A genuine no-match from the hub is still a clean, complete answer.
	empty := r.Retrieve(context.Background(), "nothing", []string{"vantage"}, RetrieveOptions{})
	if empty.Error != "" || len(empty.Hits) != 0 || empty.Degraded != "" || empty.FailedWorkspaces != nil {
		t.Fatalf("no-match = %+v", empty)
	}

	hub.down = true
	all := r.Retrieve(context.Background(), "sop", []string{"founderos", "personal"}, RetrieveOptions{})
	if all.Error == "" || len(all.Hits) != 0 {
		t.Fatalf("every engine down must stay an error: %+v", all)
	}
}
