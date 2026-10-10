package clients

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/rhl/businessos-backend/internal/founderos/connectors/superset"
)

// Ported from FounderOS v1 tests/client-work.test.ts (launchClientDraft).

type fakeLauncher struct {
	projects []superset.Project
	listErr  error
	specs    []superset.WorkspaceSpec
	id       string
	err      error
}

func (f *fakeLauncher) ListProjects(context.Context) ([]superset.Project, error) {
	return f.projects, f.listErr
}

func (f *fakeLauncher) CreateWorkspace(_ context.Context, s superset.WorkspaceSpec) (string, error) {
	f.specs = append(f.specs, s)
	return f.id, f.err
}

func TestLaunchRefusesWhenTheProjectIsAbsent(t *testing.T) {
	f := &fakeLauncher{}
	if _, err := Launch(context.Background(), f, "silvio-big-mamas", "draft-1", "Write copy"); err == nil || !strings.Contains(err.Error(), "not available") {
		t.Fatalf("err = %v", err)
	}
	if len(f.specs) != 0 {
		t.Fatal("must not create a workspace")
	}
}

func TestLaunchRefusesAProjectWithTheRightIDButWrongName(t *testing.T) {
	f := &fakeLauncher{projects: []superset.Project{{ID: "2ffe9fec-9041-4577-83c3-72902a6aae98", Name: "Other"}}}
	if _, err := Launch(context.Background(), f, "silvio-big-mamas", "draft-1", "Write copy"); err == nil {
		t.Fatal("name must match too")
	}
}

func TestLaunchRefusesUnknownClientOrUnsafeRequestID(t *testing.T) {
	f := &fakeLauncher{}
	if _, err := Launch(context.Background(), f, "nobody", "draft-1", "x"); err == nil {
		t.Fatal("unknown client")
	}
	if _, err := Launch(context.Background(), f, "silvio-big-mamas", "../evil", "x"); err == nil {
		t.Fatal("unsafe id")
	}
}

func TestLaunchUsesTheRegisteredProjectAndAnIsolatedBranch(t *testing.T) {
	f := &fakeLauncher{projects: []superset.Project{{ID: "2ffe9fec-9041-4577-83c3-72902a6aae98", Name: "Silvio-Big Mamas"}}, id: "workspace-1"}
	id, err := Launch(context.Background(), f, "silvio-big-mamas", "draft-1", "Write copy")
	if err != nil || id != "workspace-1" {
		t.Fatalf("id %q err %v", id, err)
	}
	s := f.specs[0]
	if s.ProjectID != "2ffe9fec-9041-4577-83c3-72902a6aae98" || s.Branch != "client-drafts/silvio-big-mamas-draft-1" || s.Agent != "codex" || s.Name != "Silvio / Big Mama's draft" {
		t.Fatalf("spec %+v", s)
	}
	if !strings.Contains(s.Prompt, "Do not publish") || !strings.HasSuffix(s.Prompt, "Requested draft:\nWrite copy") {
		t.Fatalf("prompt %q", s.Prompt)
	}
}

func TestLaunchSurfacesACreateFailure(t *testing.T) {
	f := &fakeLauncher{projects: []superset.Project{{ID: "2ffe9fec-9041-4577-83c3-72902a6aae98", Name: "Silvio-Big Mamas"}}, err: errors.New("boom")}
	if _, err := Launch(context.Background(), f, "silvio-big-mamas", "draft-1", "Write copy"); err == nil {
		t.Fatal("must fail")
	}
	f.err, f.id = nil, ""
	if _, err := Launch(context.Background(), f, "silvio-big-mamas", "draft-1", "Write copy"); err == nil {
		t.Fatal("an empty workspace id is not a launch")
	}
}

func TestDraftPromptCarriesTheGuardrails(t *testing.T) {
	p, err := DraftPrompt("silvio-big-mamas", "Write copy")
	if err != nil || !strings.HasPrefix(p, "Create a marketing DRAFT for Silvio / Big Mama's") || !strings.Contains(p, "Do not invent prices") {
		t.Fatalf("%q %v", p, err)
	}
	if _, err := DraftPrompt("nobody", "x"); err == nil {
		t.Fatal("unknown client")
	}
}
