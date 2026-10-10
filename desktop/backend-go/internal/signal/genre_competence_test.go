package signal

import "testing"

func TestCompetenceRegistry_OrchestratorDirect(t *testing.T) {
	cr := NewCompetenceRegistry()
	comp := cr.Lookup("orchestrator", GenreDirect)
	if comp == nil {
		t.Fatal("expected competence for orchestrator/DIRECT")
	}
	if len(comp.ContextHints) == 0 {
		t.Error("expected non-empty context hints")
	}
	// Should include "project" and "client"
	found := false
	for _, h := range comp.ContextHints {
		if h == "project" {
			found = true
		}
	}
	if !found {
		t.Error("expected 'project' in context hints for DIRECT")
	}
}

func TestCompetenceRegistry_FallbackToOrchestrator(t *testing.T) {
	cr := NewCompetenceRegistry()
	// Unknown agent type should fall back to orchestrator
	comp := cr.Lookup("unknown_agent", GenreInform)
	if comp == nil {
		t.Fatal("expected fallback to orchestrator competence")
	}
	if comp.Agent != "orchestrator" {
		t.Errorf("expected orchestrator fallback, got %s", comp.Agent)
	}
}

func TestCompetenceRegistry_DocumentAgent(t *testing.T) {
	cr := NewCompetenceRegistry()
	comp := cr.Lookup("document", GenreDirect)
	if comp == nil {
		t.Fatal("expected competence for document/DIRECT")
	}
	if len(comp.DocTypes) == 0 {
		t.Error("document agent should have doc types")
	}
}

func TestCompetenceRegistry_Register(t *testing.T) {
	cr := NewCompetenceRegistry()
	cr.Register(GenreCompetence{
		Agent:        "custom",
		Genre:        GenreDirect,
		ContextHints: []string{"custom_context"},
		DocTypes:     []string{"custom_doc"},
	})
	comp := cr.Lookup("custom", GenreDirect)
	if comp == nil {
		t.Fatal("expected registered competence")
	}
	if comp.ContextHints[0] != "custom_context" {
		t.Errorf("expected custom_context, got %s", comp.ContextHints[0])
	}
}

func TestCompetenceRegistry_ExpressNoDocTypes(t *testing.T) {
	cr := NewCompetenceRegistry()
	comp := cr.Lookup("orchestrator", GenreExpress)
	if comp == nil {
		t.Fatal("expected competence for EXPRESS")
	}
	if len(comp.DocTypes) != 0 {
		t.Errorf("EXPRESS should have no doc types, got %v", comp.DocTypes)
	}
}

func TestCompetenceRegistry_NilForUnknownGenre(t *testing.T) {
	cr := NewCompetenceRegistry()
	// Project agent with a genre it doesn't have
	comp := cr.Lookup("project", GenreExpress)
	// Should fall back to orchestrator
	if comp == nil {
		t.Fatal("expected fallback")
	}
	if comp.Agent != "orchestrator" {
		t.Errorf("expected orchestrator fallback, got %s", comp.Agent)
	}
}
