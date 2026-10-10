package tech

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/rhl/businessos-backend/internal/founderos/agents"
	"github.com/rhl/businessos-backend/internal/founderos/memory"
)

func healthyHub() *fakeEngine {
	f := newFakeEngine()
	f.workspaces = []string{"founderos", "vantage"}
	f.counts = map[string]int{"contexts": 3, "vectors": 3, "chunk_embeddings": 12, "claims": 2, "facts": 1}
	return f
}

func healthyMacbook() *fakeEngine {
	f := newFakeEngine()
	f.workspaces = []string{"personal"}
	f.counts = map[string]int{"contexts": 1, "vectors": 1, "chunk_embeddings": 4, "claims": 0, "facts": 0}
	return f
}

type fakeSearcher struct {
	hits       []memory.Hit
	err        error
	query      string
	workspaces []string
}

func (s *fakeSearcher) Search(_ context.Context, q string, ws []string, limit int) ([]memory.Hit, error) {
	s.query, s.workspaces = q, ws
	return s.hits, s.err
}

func TestDataAgentMeta(t *testing.T) {
	m := (&DataAgent{}).Meta()
	if m.ID != "data-agent" || m.Name != "Data Agent" || m.DepartmentID != "dept-tech" || m.Description == "" {
		t.Fatalf("meta = %+v", m)
	}
	var _ agents.Responder = &DataAgent{}
}

func TestDataAgentRunHealthy(t *testing.T) {
	a := &DataAgent{Engines: source(t, map[string]*fakeEngine{"hub": healthyHub(), "macbook": healthyMacbook()})}
	res, err := a.Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !res.OK {
		t.Fatalf("healthy engines should be ok: %s", res.Summary)
	}
	for _, want := range []string{"hub up", "macbook up", "3 contexts", "storage healthy", "mini not staged (hermes)"} {
		if !strings.Contains(res.Summary, want) {
			t.Errorf("summary missing %q: %s", want, res.Summary)
		}
	}
	if res.Model != "" || res.TokensIn != nil {
		t.Errorf("data agent makes no LLM call: %+v", res)
	}
}

func TestDataAgentRunSurfacesIdeas(t *testing.T) {
	hub := healthyHub()
	hub.workspaces = []string{"founderos"} // vantage never created on its home
	hub.counts = map[string]int{"contexts": 10, "vectors": 7, "claims": 5, "facts": 0}
	hub.checks = append(hub.checks, fakeCheck{"verified_backup", false, "no_verified_backup"})
	a := &DataAgent{Engines: source(t, map[string]*fakeEngine{"hub": hub, "macbook": healthyMacbook()})}
	res, _ := a.Run(context.Background())
	if !res.OK {
		t.Fatalf("reachable engines with findings are still an ok run: %s", res.Summary)
	}
	for _, want := range []string{"vantage", "verified_backup", "3 context(s) on hub have no embedding", "5 claim(s) on hub, 0 facts"} {
		if !strings.Contains(res.Summary, want) {
			t.Errorf("summary missing idea %q: %s", want, res.Summary)
		}
	}
	if strings.Contains(res.Summary, "storage healthy") {
		t.Errorf("must not call storage healthy with ideas pending: %s", res.Summary)
	}
}

func TestDataAgentRunEngineDown(t *testing.T) {
	hub := healthyHub()
	hub.down = true
	a := &DataAgent{Engines: source(t, map[string]*fakeEngine{"hub": hub, "macbook": healthyMacbook()})}
	res, _ := a.Run(context.Background())
	if res.OK {
		t.Fatalf("an unreachable engine is not ok: %s", res.Summary)
	}
	if !strings.Contains(res.Summary, "hub unreachable") {
		t.Errorf("summary should name the unreachable engine: %s", res.Summary)
	}
	if strings.Contains(res.Summary, "0 contexts") {
		t.Errorf("an unreachable engine must not read as empty: %s", res.Summary)
	}
}

func TestDataAgentRunNoEngines(t *testing.T) {
	res, _ := (&DataAgent{Engines: noEngines(t)}).Run(context.Background())
	if res.OK || !strings.Contains(res.Summary, "no Optimal Engine is staged") {
		t.Fatalf("no engines must fail honestly: %+v", res)
	}
}

func TestDataAgentRunTopologyMissing(t *testing.T) {
	res, _ := (&DataAgent{Engines: failingSource(errors.New("topology not found"))}).Run(context.Background())
	if res.OK || !strings.Contains(res.Summary, "topology not found") {
		t.Fatalf("topology error must surface: %+v", res)
	}
}

func TestDataAgentRespondSearchesStagedWorkspaces(t *testing.T) {
	s := &fakeSearcher{hits: []memory.Hit{
		{Title: "Pricing", Abstract: strings.Repeat("p", 150)},
		{Title: "Offer", Abstract: "the vantage offer"},
		{Title: "Scripts", Abstract: "call scripts"},
		{Title: "Fourth", Abstract: "not shown"},
	}}
	a := &DataAgent{Engines: source(t, map[string]*fakeEngine{"hub": healthyHub(), "macbook": healthyMacbook()}), Search: s}
	res, err := a.Respond(context.Background(), "what is our pricing?")
	if err != nil {
		t.Fatal(err)
	}
	if !res.OK {
		t.Fatalf("hits should answer: %s", res.Summary)
	}
	if s.query != "what is our pricing?" {
		t.Errorf("query = %q", s.query)
	}
	if got := strings.Join(s.workspaces, ","); got != "founderos,vantage,personal" {
		t.Errorf("should search every staged workspace and skip unstaged ones, got %s", got)
	}
	if !strings.Contains(res.Summary, "Pricing: "+strings.Repeat("p", 100)+" · Offer: the vantage offer · Scripts: call scripts") {
		t.Errorf("summary = %s", res.Summary)
	}
	if strings.Contains(res.Summary, "Fourth") || strings.Contains(res.Summary, strings.Repeat("p", 101)) {
		t.Errorf("top 3, snippets capped at 100: %s", res.Summary)
	}
}

func TestDataAgentRespondNoMatch(t *testing.T) {
	a := &DataAgent{Engines: source(t, map[string]*fakeEngine{"hub": healthyHub()}), Search: &fakeSearcher{}}
	res, _ := a.Respond(context.Background(), "unicorns")
	if res.OK || !strings.Contains(res.Summary, `Nothing in memory matches "unicorns"`) {
		t.Fatalf("no hits = honest miss: %+v", res)
	}
}

func TestDataAgentRespondSearchError(t *testing.T) {
	a := &DataAgent{Engines: source(t, map[string]*fakeEngine{"hub": healthyHub()}),
		Search: &fakeSearcher{err: &memory.UnreachableError{Engine: "hub", Err: errors.New("refused")}}}
	res, _ := a.Respond(context.Background(), "pricing")
	if res.OK || !strings.Contains(res.Summary, "memory search failed") || strings.Contains(res.Summary, "Nothing") {
		t.Fatalf("a failed search must not look like an empty one: %+v", res)
	}
}

func TestDataAgentRespondNoMemory(t *testing.T) {
	a := &DataAgent{Engines: source(t, map[string]*fakeEngine{"hub": healthyHub()})}
	res, _ := a.Respond(context.Background(), "pricing")
	if res.OK || !strings.Contains(res.Summary, "memory router is not configured") {
		t.Fatalf("missing memory must fail honestly: %+v", res)
	}
}

// One engine down is a partial answer (memory.PartialError), as every other
// brain read reports it: the hits that came back, plus the unsearched
// workspaces named, never "memory search failed".
func TestDataAgentAnswersFromAPartialSearchAndSaysWhatIsMissing(t *testing.T) {
	s := &fakeSearcher{
		hits: []memory.Hit{{Title: "Pricing", Abstract: "the price list"}},
		err:  &memory.PartialError{Failed: []string{"personal"}, Errs: []error{errors.New("engine macbook unreachable")}},
	}
	a := &DataAgent{Engines: source(t, map[string]*fakeEngine{"hub": healthyHub(), "macbook": healthyMacbook()}), Search: s}
	res, _ := a.Respond(context.Background(), "pricing")
	if !res.OK || !strings.Contains(res.Summary, "Pricing") || !strings.Contains(res.Summary, "partial: personal not searched") {
		t.Fatalf("%+v", res)
	}
}
