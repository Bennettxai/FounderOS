package tech

import (
	"context"
	"strings"
	"testing"
)

func vectorHub() *fakeEngine {
	f := healthyHub()
	f.pending = map[string]int{"founderos": 1, "vantage": 1}
	return f
}

func TestVectorAuditorMeta(t *testing.T) {
	m := (&VectorAuditor{}).Meta()
	if m.ID != "vector-auditor" || m.Name != "Vector Auditor" || m.DepartmentID != "dept-tech" {
		t.Fatalf("meta = %+v", m)
	}
}

func TestVectorAuditorHealthy(t *testing.T) {
	a := &VectorAuditor{Engines: source(t, map[string]*fakeEngine{"hub": vectorHub(), "macbook": healthyMacbook()})}
	res, err := a.Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !res.OK {
		t.Fatalf("healthy: %s", res.Summary)
	}
	for _, want := range []string{
		"hub: health 100/100 · 4 checks, 0 failing",
		"embeddings 3/3 contexts",
		"12 chunk vectors",
		"claims 2 / facts 1 (pending: founderos 1, vantage 1)",
		"macbook: health 100/100",
		"mini not staged",
	} {
		if !strings.Contains(res.Summary, want) {
			t.Errorf("summary missing %q: %s", want, res.Summary)
		}
	}
}

func TestVectorAuditorNonCriticalFailureWarns(t *testing.T) {
	hub := vectorHub()
	hub.checks = append(hub.checks, fakeCheck{"verified_backup", false, ":no_verified_backup"})
	a := &VectorAuditor{Engines: source(t, map[string]*fakeEngine{"hub": hub})}
	res, _ := a.Run(context.Background())
	if !res.OK {
		t.Fatalf("a missing backup is a warning for the vector auditor: %s", res.Summary)
	}
	if !strings.Contains(res.Summary, "health 80/100 · 5 checks, 1 failing: verified_backup") {
		t.Errorf("summary = %s", res.Summary)
	}
}

func TestVectorAuditorCriticalFailureFails(t *testing.T) {
	hub := vectorHub()
	hub.checks[2] = fakeCheck{"vector_integrity", false, map[string]any{"bad_counts": []int{1, 0, 0, 0}}}
	a := &VectorAuditor{Engines: source(t, map[string]*fakeEngine{"hub": hub})}
	res, _ := a.Run(context.Background())
	if res.OK || !strings.Contains(res.Summary, "vector_integrity") {
		t.Fatalf("broken vectors must fail: %s", res.Summary)
	}
}

func TestVectorAuditorWarnsOnGaps(t *testing.T) {
	hub := vectorHub()
	hub.counts = map[string]int{"contexts": 10, "vectors": 7, "chunk_embeddings": 30, "claims": 4, "facts": 0}
	a := &VectorAuditor{Engines: source(t, map[string]*fakeEngine{"hub": hub})}
	res, _ := a.Run(context.Background())
	for _, want := range []string{"embeddings 7/10 contexts", "3 context(s) without an embedding", "no claim promoted to a fact"} {
		if !strings.Contains(res.Summary, want) {
			t.Errorf("summary missing %q: %s", want, res.Summary)
		}
	}
}

func TestVectorAuditorEngineDown(t *testing.T) {
	hub := vectorHub()
	hub.down = true
	a := &VectorAuditor{Engines: source(t, map[string]*fakeEngine{"hub": hub, "macbook": healthyMacbook()})}
	res, _ := a.Run(context.Background())
	if res.OK || !strings.Contains(res.Summary, "hub: offline") {
		t.Fatalf("down engine: %s", res.Summary)
	}
	if strings.Contains(res.Summary, "hub: health") {
		t.Errorf("no health for an engine that did not answer: %s", res.Summary)
	}
}

func TestVectorAuditorHealthNotUp(t *testing.T) {
	hub := vectorHub()
	hub.health = "degraded"
	a := &VectorAuditor{Engines: source(t, map[string]*fakeEngine{"hub": hub})}
	res, _ := a.Run(context.Background())
	if res.OK || !strings.Contains(res.Summary, `health "degraded"`) {
		t.Fatalf("degraded engine: %s", res.Summary)
	}
}

func TestVectorAuditorNoEngines(t *testing.T) {
	res, _ := (&VectorAuditor{Engines: noEngines(t)}).Run(context.Background())
	if res.OK || !strings.Contains(res.Summary, "no Optimal Engine is staged") {
		t.Fatalf("res = %+v", res)
	}
}
