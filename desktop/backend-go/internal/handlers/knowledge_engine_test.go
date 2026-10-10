package handlers

import "testing"

func TestEngineWorkspaceSearchAnchorNormalizesSlugSeparators(t *testing.T) {
	got := engineWorkspaceSearchAnchor("default:agency-miosa_workspace")
	want := "default agency miosa workspace"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestEngineWorkspaceSearchAnchorDefaultsWhenEmpty(t *testing.T) {
	got := engineWorkspaceSearchAnchor("  ")
	if got != "default" {
		t.Fatalf("expected default, got %q", got)
	}
}
