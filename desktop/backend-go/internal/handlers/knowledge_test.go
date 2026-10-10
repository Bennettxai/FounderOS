package handlers

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewKnowledgeHandlerUsesParentWhenOptimalEngineRootIsWorkspace(t *testing.T) {
	workspacesRoot := t.TempDir()
	workspaceRoot := filepath.Join(workspacesRoot, "businessos")
	if err := os.MkdirAll(filepath.Join(workspaceRoot, ".optimal"), 0o755); err != nil {
		t.Fatal(err)
	}

	t.Setenv("KNOWLEDGE_WORKSPACES_ROOT", "")
	t.Setenv("OPTIMAL_ENGINE_ROOT", workspaceRoot)

	handler := NewKnowledgeHandler(nil)

	if handler.root != workspacesRoot {
		t.Fatalf("expected %q, got %q", workspacesRoot, handler.root)
	}
}

func TestNewKnowledgeHandlerRespectsExplicitKnowledgeWorkspacesRoot(t *testing.T) {
	workspacesRoot := t.TempDir()
	optimalRoot := filepath.Join(t.TempDir(), "businessos")
	if err := os.MkdirAll(filepath.Join(optimalRoot, ".optimal"), 0o755); err != nil {
		t.Fatal(err)
	}

	t.Setenv("KNOWLEDGE_WORKSPACES_ROOT", workspacesRoot)
	t.Setenv("OPTIMAL_ENGINE_ROOT", optimalRoot)

	handler := NewKnowledgeHandler(nil)

	if handler.root != workspacesRoot {
		t.Fatalf("expected %q, got %q", workspacesRoot, handler.root)
	}
}

func TestIsWorkspaceProjectionRejectsArbitraryAssetDirectory(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "assets"), 0o755); err != nil {
		t.Fatal(err)
	}

	if isWorkspaceProjection(dir) {
		t.Fatal("asset-only directory must not be treated as a workspace")
	}
}

func TestIsWorkspaceProjectionAcceptsDurableWorkspaceMarker(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".wiki"), 0o755); err != nil {
		t.Fatal(err)
	}

	if !isWorkspaceProjection(dir) {
		t.Fatal("workspace directory with .wiki marker must be discovered")
	}
}

func TestInternalWorkspaceProjection(t *testing.T) {
	for _, slug := range []string{"inbox", "team", "knowledge-intake", "default-businessos", "benchmark-truememory-locomo-r1-c0"} {
		if !isInternalWorkspaceProjection(slug) {
			t.Fatalf("%q should be internal", slug)
		}
	}
	for _, slug := range []string{"businessos", "miosa", "agency-miosa", "clinic-iq", "personal-brand"} {
		if isInternalWorkspaceProjection(slug) {
			t.Fatalf("%q should be selectable", slug)
		}
	}
}
