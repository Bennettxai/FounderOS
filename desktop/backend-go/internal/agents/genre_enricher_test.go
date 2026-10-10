package agents

import (
	"strings"
	"testing"

	"github.com/rhl/businessos-backend/internal/services"
	"github.com/rhl/businessos-backend/internal/signal"
)

func TestBuildSignalAnnotation_NilEnvelope(t *testing.T) {
	result := BuildSignalAnnotation(nil, nil)
	if result != "" {
		t.Errorf("expected empty for nil envelope, got %q", result)
	}
}

func TestBuildSignalAnnotation_DirectWithDocType(t *testing.T) {
	env := &signal.SignalEnvelope{
		Mode:    signal.ModeExecute,
		Genre:   signal.GenreDirect,
		DocType: "proposal",
		Weight:  0.8,
	}
	result := BuildSignalAnnotation(env, nil)

	if !strings.Contains(result, "SIGNAL DETECTION") {
		t.Error("expected SIGNAL DETECTION header")
	}
	if !strings.Contains(result, "DIRECT") {
		t.Error("expected DIRECT genre in output")
	}
	if !strings.Contains(result, "proposal") {
		t.Error("expected proposal in output")
	}
	if !strings.Contains(result, "Executive Summary") {
		t.Error("expected proposal structure template")
	}
	// Should include writing style
	if !strings.Contains(result, "Writing Style") {
		t.Error("expected Writing Style section")
	}
	if !strings.Contains(result, "confident") {
		t.Error("expected 'confident' in proposal writing style")
	}
}

func TestBuildSignalAnnotation_WithContext(t *testing.T) {
	env := &signal.SignalEnvelope{
		Mode:    signal.ModeExecute,
		Genre:   signal.GenreDirect,
		DocType: "proposal",
		Weight:  0.8,
	}
	ctx := &services.TieredContext{
		Level1: &services.FullContext{
			Project: &services.ProjectFullContext{
				Name:     "Q3 Expansion",
				Status:   "active",
				Priority: "high",
			},
			LinkedClient: &services.ClientFullContext{
				Name:     "Acme Corp",
				Industry: "Technology",
			},
			Tasks: []services.TaskFullContext{
				{Title: "Finalize pricing"},
				{Title: "Competitor review"},
			},
			TeamMembers: []services.TeamMemberContext{
				{Name: "Roberto", Role: "CTO"},
			},
		},
	}
	result := BuildSignalAnnotation(env, ctx)

	// Context inventory should list all available sources
	if !strings.Contains(result, "Q3 Expansion") {
		t.Error("expected project name in context inventory")
	}
	if !strings.Contains(result, "Acme Corp") {
		t.Error("expected client name in context inventory")
	}
	if !strings.Contains(result, "2 active tasks") {
		t.Error("expected task count in inventory")
	}
	if !strings.Contains(result, "1 members") {
		t.Error("expected team count in inventory")
	}
}

func TestBuildSignalAnnotation_InformNoDocType(t *testing.T) {
	env := &signal.SignalEnvelope{
		Mode:   signal.ModeAssist,
		Genre:  signal.GenreInform,
		Weight: 0.5,
	}
	result := BuildSignalAnnotation(env, nil)

	if !strings.Contains(result, "INFORM") {
		t.Error("expected INFORM genre")
	}
	if !strings.Contains(result, "asking a question") {
		t.Error("expected plain-English genre description")
	}
	// Should have genre-level writing style, not document style
	if !strings.Contains(result, "Answer the question directly") {
		t.Error("expected INFORM writing style guidance")
	}
	// Should NOT have document structure
	if strings.Contains(result, "Document Structure") {
		t.Error("expected no document structure for INFORM without doctype")
	}
}

func TestBuildSignalAnnotation_SOPTemplate(t *testing.T) {
	env := &signal.SignalEnvelope{
		Mode:    signal.ModeExecute,
		Genre:   signal.GenreDirect,
		DocType: "sop",
		Weight:  0.7,
	}
	result := BuildSignalAnnotation(env, nil)
	if !strings.Contains(result, "Purpose") {
		t.Error("expected SOP structure")
	}
	if !strings.Contains(result, "Procedure") {
		t.Error("expected Procedure section in SOP")
	}
	if !strings.Contains(result, "Imperative voice") {
		t.Error("expected SOP writing style")
	}
}

func TestBuildSignalAnnotation_DecideGenre(t *testing.T) {
	env := &signal.SignalEnvelope{
		Mode:   signal.ModeAssist,
		Genre:  signal.GenreDecide,
		Weight: 0.6,
	}
	result := BuildSignalAnnotation(env, nil)
	if !strings.Contains(result, "choosing between options") {
		t.Error("expected DECIDE genre description")
	}
	if !strings.Contains(result, "recommendation") {
		t.Error("expected decision writing style")
	}
}

func TestBuildSignalAnnotation_ExpressGenre(t *testing.T) {
	env := &signal.SignalEnvelope{
		Mode:   signal.ModeAssist,
		Genre:  signal.GenreExpress,
		Weight: 0.3,
	}
	result := BuildSignalAnnotation(env, nil)
	if !strings.Contains(result, "Acknowledge") {
		t.Error("expected empathetic guidance for EXPRESS")
	}
}

func TestBuildSignalAnnotation_EmptyContext(t *testing.T) {
	env := &signal.SignalEnvelope{
		Mode:   signal.ModeAssist,
		Genre:  signal.GenreInform,
		Weight: 0.5,
	}
	// Empty L1 — no available context
	ctx := &services.TieredContext{
		Level1: &services.FullContext{},
	}
	result := BuildSignalAnnotation(env, ctx)
	// Should NOT have "Available Context" section when nothing loaded
	if strings.Contains(result, "Available Context") {
		t.Error("expected no context section when nothing loaded")
	}
}

func TestGetGenreStructureHint_AllGenres(t *testing.T) {
	genres := []signal.Genre{
		signal.GenreDirect, signal.GenreInform, signal.GenreCommit,
		signal.GenreDecide, signal.GenreExpress,
	}
	for _, g := range genres {
		hint := GetGenreStructureHint(g)
		if hint == "" {
			t.Errorf("expected non-empty hint for genre %s", g)
		}
	}
}

func TestGenreStructureTemplates_AllDocTypes(t *testing.T) {
	expectedTypes := []string{"proposal", "sop", "report", "brief", "framework", "guide", "plan"}
	for _, dt := range expectedTypes {
		if _, ok := GenreStructureTemplates[dt]; !ok {
			t.Errorf("expected template for doc type %q", dt)
		}
	}
}
