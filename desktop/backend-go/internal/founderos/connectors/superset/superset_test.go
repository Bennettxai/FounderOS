package superset

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/rhl/businessos-backend/internal/founderos/connectors"
	"github.com/rhl/businessos-backend/internal/founderos/guard"
)

var fixed = time.Date(2026, 8, 6, 12, 0, 0, 0, time.UTC)

const projectsJSON = `[{"name":"FounderOS-v1","repo":"https://github.com/example-org/FounderOS-v1","id":"proj-1"},{"name":"BusinessOS","id":"proj-2"}]`

func resolver(t *testing.T, lines ...string) connectors.Resolver {
	t.Helper()
	p := filepath.Join(t.TempDir(), "env.local")
	if err := os.WriteFile(p, []byte(strings.Join(lines, "\n")+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("SUPERSET_BIN", "")
	t.Setenv("SUPERSET_PROJECT_ID", "")
	return connectors.Resolver{EnvLocal: p}
}

// fakeBin writes a shell script standing in for the Superset CLI.
func fakeBin(t *testing.T, script string) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "superset")
	if err := os.WriteFile(bin, []byte("#!/bin/sh\n"+script+"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	return bin
}

type call struct {
	cmd  string
	args []string
}

func stub(handler func(args []string) ExecResult) (*[]call, ExecFn) {
	calls := &[]call{}
	return calls, func(_ context.Context, cmd string, args []string) ExecResult {
		*calls = append(*calls, call{cmd, args})
		return handler(args)
	}
}

func after(args []string, flag string) string {
	for i, a := range args {
		if a == flag && i+1 < len(args) {
			return args[i+1]
		}
	}
	return ""
}

func TestMeta(t *testing.T) {
	if Meta != (connectors.Meta{ID: "superset", Name: "Superset", Kind: connectors.KindOrchestration}) {
		t.Fatalf("Meta = %+v", Meta)
	}
}

func TestBranchFor(t *testing.T) {
	if got := BranchFor("Make the sidebar blue!", fixed); got != "conductor/ui-make-the-sidebar-blue-20260806" {
		t.Errorf("got %q", got)
	}
	if got := BranchFor("???", fixed); got != "conductor/ui-change-20260806" {
		t.Errorf("got %q", got)
	}
	if got := BranchFor("Café  au_lait -- menü", fixed); got != "conductor/ui-cafe-au-lait-menu-20260806" {
		t.Errorf("accents fold, runs collapse: got %q", got)
	}
	long := BranchFor("one two three four five six seven eight nine ten eleven twelve", fixed)
	if len(long) > 80 || strings.Contains(long, "--") || !strings.HasSuffix(long, "-20260806") {
		t.Errorf("got %q", long)
	}
}

func TestWorkerPromptCarriesRequestAndHouseRules(t *testing.T) {
	p := WorkerPromptFor("Make the sidebar blue", "conductor/ui-make-the-sidebar-blue-20260806")
	for _, want := range []string{"Make the sidebar blue", "failing test", "npm test", "npm run typecheck", "NEVER push", "4100", "conductor/ui-make-the-sidebar-blue-20260806"} {
		if !strings.Contains(p, want) {
			t.Errorf("prompt missing %q", want)
		}
	}
}

func TestStatusNotConfiguredWhenTheCLIIsMissing(t *testing.T) {
	c := New(resolver(t, "SUPERSET_BIN="+filepath.Join(t.TempDir(), "nope")))
	s := c.Status(context.Background())
	if s.State != connectors.StateNotConfigured || !strings.Contains(s.Detail, "not found") {
		t.Fatalf("status = %+v", s)
	}
}

func TestStatusConnectedListsProjectsReadOnly(t *testing.T) {
	log := filepath.Join(t.TempDir(), "args")
	bin := fakeBin(t, `echo "$@" >> `+log+`
if [ "$1" = "projects" ]; then echo '`+projectsJSON+`'; else echo "unexpected" >&2; exit 1; fi`)
	c := New(resolver(t, "SUPERSET_BIN="+bin))
	s := c.Status(context.Background())
	if s.State != connectors.StateConnected || s.Detail != "CLI ready · 2 local projects · FounderOS-v1 found" {
		t.Fatalf("status = %+v", s)
	}
	raw, _ := os.ReadFile(log)
	if strings.TrimSpace(string(raw)) != "projects list --local --json" {
		t.Errorf("status ran %q; only the read-only projects list is allowed", raw)
	}
}

func TestStatusErrorCarriesTheCLIsOwnWords(t *testing.T) {
	bin := fakeBin(t, `echo "Error: Not logged in" >&2; exit 1`)
	s := New(resolver(t, "SUPERSET_BIN="+bin)).Status(context.Background())
	if s.State != connectors.StateError || !strings.Contains(s.Detail, "Not logged in") {
		t.Fatalf("status = %+v", s)
	}
}

func TestListProjectsAcceptsWrappedJSON(t *testing.T) {
	c := New(resolver(t))
	_, c.Exec = stub(func([]string) ExecResult { return ExecResult{Stdout: `{"projects":` + projectsJSON + `}`} })
	ps, err := c.ListProjects(context.Background())
	if err != nil || len(ps) != 2 || ps[0].ID != "proj-1" {
		t.Fatalf("projects = %+v, %v", ps, err)
	}
	_, c.Exec = stub(func([]string) ExecResult { return ExecResult{Stdout: "not json"} })
	if _, err := c.ListProjects(context.Background()); err == nil || !strings.Contains(err.Error(), "unparseable") {
		t.Fatalf("err = %v", err)
	}
}

func TestDispatchIsRefusedWhileWritesAreOff(t *testing.T) {
	t.Setenv("FOUNDEROS_WRITES", "0")
	c := New(resolver(t))
	calls, exec := stub(func(args []string) ExecResult {
		if args[0] == "projects" {
			return ExecResult{Stdout: projectsJSON}
		}
		return ExecResult{Stdout: `{"id":"ws-1"}`}
	})
	c.Exec = exec
	_, err := c.DispatchCodingTask(context.Background(), "Make the sidebar blue")
	if !errors.Is(err, guard.ErrWritesDisabled) {
		t.Fatalf("err = %v", err)
	}
	for _, cl := range *calls {
		if cl.args[0] == "workspaces" {
			t.Fatalf("a workspace was created with writes off: %v", cl.args)
		}
	}
}

func TestDispatchResolvesProjectThenCreatesWorkspace(t *testing.T) {
	t.Setenv("FOUNDEROS_WRITES", "1")
	c := New(resolver(t))
	c.now = func() time.Time { return fixed }
	calls, exec := stub(func(args []string) ExecResult {
		if args[0] == "projects" {
			return ExecResult{Stdout: projectsJSON}
		}
		return ExecResult{Stdout: `{"workspace":{"id":"ws-1"}}`}
	})
	c.Exec = exec
	res, err := c.DispatchCodingTask(context.Background(), "  Make the sidebar blue ")
	if err != nil {
		t.Fatal(err)
	}
	if res.WorkspaceID != "ws-1" || res.Branch != "conductor/ui-make-the-sidebar-blue-20260806" {
		t.Fatalf("res = %+v", res)
	}
	if got := strings.Join((*calls)[0].args, " "); got != "projects list --local --json" {
		t.Errorf("first call = %q", got)
	}
	create := (*calls)[1].args
	if strings.Join(create[:3], " ") != "workspaces create --local" || after(create, "--project") != "proj-1" ||
		after(create, "--branch") != res.Branch || after(create, "--agent") != "claude" ||
		!strings.Contains(after(create, "--prompt"), "Make the sidebar blue") || create[len(create)-1] != "--json" {
		t.Errorf("create args = %v", create)
	}
}

func TestDispatchHonestFailures(t *testing.T) {
	t.Setenv("FOUNDEROS_WRITES", "1")
	ctx := context.Background()

	c := New(resolver(t, "SUPERSET_PROJECT_ID=proj-env"))
	calls, exec := stub(func([]string) ExecResult { return ExecResult{Stdout: `{"ok":true}`} })
	c.Exec = exec
	if _, err := c.DispatchCodingTask(ctx, "x"); err == nil || !strings.Contains(err.Error(), "no workspace id") {
		t.Errorf("err = %v", err)
	}
	if len(*calls) != 1 || after((*calls)[0].args, "--project") != "proj-env" {
		t.Errorf("SUPERSET_PROJECT_ID must skip discovery: %v", *calls)
	}

	c = New(resolver(t))
	_, c.Exec = stub(func([]string) ExecResult { return ExecResult{Stderr: "Error: Not logged in", Code: 1} })
	if _, err := c.DispatchCodingTask(ctx, "x"); err == nil || !strings.Contains(err.Error(), "Not logged in") {
		t.Errorf("err = %v", err)
	}

	_, c.Exec = stub(func([]string) ExecResult { return ExecResult{Stdout: "[]"} })
	if _, err := c.DispatchCodingTask(ctx, "x"); err == nil || !strings.Contains(err.Error(), "project") {
		t.Errorf("err = %v", err)
	}

	if _, err := c.DispatchCodingTask(ctx, "   "); err == nil || !strings.Contains(err.Error(), "empty") {
		t.Errorf("err = %v", err)
	}
}

func TestDefaultExecReportsASpawnFailure(t *testing.T) {
	r := defaultExec(context.Background(), filepath.Join(t.TempDir(), "missing"), []string{"x"})
	if r.Code == 0 || r.Stderr == "" {
		t.Fatalf("result = %+v", r)
	}
}
