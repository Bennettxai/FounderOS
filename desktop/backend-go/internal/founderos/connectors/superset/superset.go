// Package superset ports FounderOS v1's lib/connectors/superset.ts: the
// Conductor's hands for code changes. The OS never edits itself in-process;
// it shells the local Superset CLI to spin up an isolated workspace (own
// worktree and branch) with a real coding agent inside.
//
// Status only ever runs the read-only `projects list`. Creating a workspace
// starts an agent, so it is a side effect and goes through guard.Outbound.
// Exec is injectable so tests stay offline; failures carry the CLI's own
// stderr, never a fake success.
package superset

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"golang.org/x/text/unicode/norm"

	"github.com/rhl/businessos-backend/internal/founderos/connectors"
	"github.com/rhl/businessos-backend/internal/founderos/guard"
)

// Meta: FounderOS v1 has no superset row in lib/connectors/index.ts; this is
// the bridge's own id for the CLI dependency.
var Meta = connectors.Meta{ID: "superset", Name: "Superset", Kind: connectors.KindOrchestration}

const (
	projectName = "FounderOS-v1"
	branchMax   = 80
	execTimeout = 60 * time.Second
)

type ExecResult struct {
	Stdout string
	Stderr string
	Code   int
}

type ExecFn func(ctx context.Context, cmd string, args []string) ExecResult

type Project struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Repo string `json:"repo,omitempty"`
}

type DispatchResult struct {
	WorkspaceID string `json:"workspaceId"`
	Branch      string `json:"branch"`
}

type Connector struct {
	res  connectors.Resolver
	Exec ExecFn
	now  func() time.Time
}

func New(res connectors.Resolver) *Connector {
	return &Connector{res: res, Exec: defaultExec, now: time.Now}
}

func defaultExec(ctx context.Context, cmd string, args []string) ExecResult {
	ctx, cancel := context.WithTimeout(ctx, execTimeout)
	defer cancel()
	var stdout, stderr bytes.Buffer
	c := exec.CommandContext(ctx, cmd, args...)
	c.Stdout, c.Stderr = &stdout, &stderr
	err := c.Run()
	r := ExecResult{Stdout: stdout.String(), Stderr: stderr.String()}
	if err != nil {
		r.Code = 1
		var exit *exec.ExitError
		if errors.As(err, &exit) && exit.ExitCode() > 0 {
			r.Code = exit.ExitCode()
		}
		if r.Stderr == "" { // spawn failure: keep the real error
			r.Stderr = err.Error()
		}
	}
	return r
}

// Bin is SUPERSET_BIN, else ~/.superset/bin/superset.
func (c *Connector) Bin() string {
	if b := c.res.Resolve("SUPERSET_BIN"); b != "" {
		return b
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".superset", "bin", "superset")
}

func (c *Connector) run(ctx context.Context, args ...string) (string, error) {
	r := c.Exec(ctx, c.Bin(), args)
	if r.Code != 0 {
		out := r.Stderr
		if out == "" {
			out = r.Stdout
		}
		if len(out) > 300 {
			out = out[:300]
		}
		second := ""
		if len(args) > 1 {
			second = args[1]
		}
		return "", errors.New(strings.TrimSpace(fmt.Sprintf("superset %s %s failed: %s", args[0], second, out)))
	}
	return r.Stdout, nil
}

// ListProjects is the read-only `superset projects list --local --json`.
func (c *Connector) ListProjects(ctx context.Context) ([]Project, error) {
	out, err := c.run(ctx, "projects", "list", "--local", "--json")
	if err != nil {
		return nil, err
	}
	var raw json.RawMessage = []byte(strings.TrimSpace(out))
	var list []Project
	if json.Unmarshal(raw, &list) != nil {
		var wrapped struct {
			Projects []Project `json:"projects"`
		}
		if err := json.Unmarshal(raw, &wrapped); err != nil {
			return nil, errors.New("superset projects list returned unparseable JSON")
		}
		list = wrapped.Projects
	}
	return list, nil
}

func (c *Connector) resolveProjectID(ctx context.Context) (string, error) {
	if id := c.res.Resolve("SUPERSET_PROJECT_ID"); id != "" {
		return id, nil
	}
	projects, err := c.ListProjects(ctx)
	if err != nil {
		return "", err
	}
	for _, p := range projects {
		if p.ID != "" && strings.EqualFold(p.Name, projectName) {
			return p.ID, nil
		}
	}
	return "", fmt.Errorf("no local Superset project named %q — set SUPERSET_PROJECT_ID in ~/.founderos/.env", projectName)
}

func (c *Connector) Status(ctx context.Context) connectors.Status {
	s := connectors.Status{ID: Meta.ID, Name: Meta.Name, Kind: Meta.Kind}
	bin := c.Bin()
	if _, err := os.Stat(bin); err != nil {
		s.State = connectors.StateNotConfigured
		s.Detail = fmt.Sprintf("Superset CLI not found at %s — set SUPERSET_BIN to override.", bin)
		return s
	}
	projects, err := c.ListProjects(ctx)
	if err != nil {
		s.State = connectors.StateError
		s.Detail = err.Error()
		return s
	}
	found := c.res.Resolve("SUPERSET_PROJECT_ID") != ""
	for _, p := range projects {
		if strings.EqualFold(p.Name, projectName) {
			found = true
		}
	}
	project := projectName + " found"
	if !found {
		project = projectName + " project missing (set SUPERSET_PROJECT_ID)"
	}
	s.State = connectors.StateConnected
	s.Detail = fmt.Sprintf("CLI ready · %d local projects · %s", len(projects), project)
	s.Meta = map[string]any{"projects": len(projects)}
	return s
}

var (
	nonWord   = regexp.MustCompile(`[^\w\s\v-]`)
	spaceRun  = regexp.MustCompile(`[\s\v_]+`)
	dashRun   = regexp.MustCompile(`-+`)
	edgeDash  = regexp.MustCompile(`^-|-$`)
	trailDash = regexp.MustCompile(`-$`)
)

// BranchFor slugs a request under conductor/ui- with a UTC date stamp.
func BranchFor(request string, now time.Time) string {
	slug := norm.NFKD.String(strings.ToLower(request))
	slug = strings.TrimSpace(nonWord.ReplaceAllString(slug, ""))
	slug = spaceRun.ReplaceAllString(slug, "-")
	slug = dashRun.ReplaceAllString(slug, "-")
	slug = edgeDash.ReplaceAllString(slug, "")
	if len(slug) > 48 {
		slug = slug[:48]
	}
	slug = trailDash.ReplaceAllString(slug, "")
	if slug == "" {
		slug = "change"
	}
	branch := fmt.Sprintf("conductor/ui-%s-%s", slug, now.UTC().Format("20060102"))
	if len(branch) > branchMax {
		branch = branch[:branchMax]
	}
	return branch
}

// WorkerPromptFor is the coding agent's prompt: the request plus house rules.
func WorkerPromptFor(request, branch string) string {
	return strings.Join([]string{
		fmt.Sprintf("You are a coding agent working the FounderOS v1 repo on branch `%s` in an isolated Superset workspace.", branch),
		"",
		"Task from the Conductor (UI change requested by the operator):",
		request,
		"",
		"House rules (non-negotiable):",
		"- Read AGENTS.md and CLAUDE.md first and follow them.",
		"- TDD: write the failing test first, then the minimal implementation.",
		"- `npm test` and `npm run typecheck` must both be green before you claim done.",
		"- Commit locally on this branch, small checkpoints. NEVER push to any remote.",
		"- Never touch the dev servers on ports 4100 or 4101 — other sessions use them.",
		"- Stay on scope: only the change requested above.",
	}, "\n")
}

// DispatchCodingTask spins up a workspace with a Claude agent working the
// request. Resolving the project is a read; creating the workspace is
// guarded as superset.workspace.create.
func (c *Connector) DispatchCodingTask(ctx context.Context, request string) (DispatchResult, error) {
	request = strings.TrimSpace(request)
	if request == "" {
		return DispatchResult{}, errors.New("dispatch request is empty")
	}
	projectID, err := c.resolveProjectID(ctx)
	if err != nil {
		return DispatchResult{}, err
	}
	branch := BranchFor(request, c.now())
	id, err := c.CreateWorkspace(ctx, WorkspaceSpec{
		ProjectID: projectID, Branch: branch, Agent: "claude", Prompt: WorkerPromptFor(request, branch),
	})
	if err != nil {
		return DispatchResult{}, err
	}
	return DispatchResult{WorkspaceID: id, Branch: branch}, nil
}

// WorkspaceSpec is one `superset workspaces create --local` call.
type WorkspaceSpec struct {
	ProjectID string
	Branch    string
	Name      string
	Agent     string
	Prompt    string
}

// CreateWorkspace starts an agent in a new workspace and returns its id.
func (c *Connector) CreateWorkspace(ctx context.Context, spec WorkspaceSpec) (string, error) {
	args := []string{"workspaces", "create", "--local", "--project", spec.ProjectID, "--branch", spec.Branch}
	if spec.Name != "" {
		args = append(args, "--name", spec.Name)
	}
	args = append(args, "--agent", spec.Agent, "--prompt", spec.Prompt, "--json")
	var id string
	err := guard.Outbound("superset.workspace.create", func() error {
		out, err := c.run(ctx, args...)
		if err != nil {
			return err
		}
		var created map[string]any
		if json.Unmarshal([]byte(out), &created) != nil {
			return errors.New("superset workspaces create returned unparseable JSON")
		}
		ws := created
		if inner, ok := created["workspace"].(map[string]any); ok {
			ws = inner
		}
		s, ok := ws["id"].(string)
		if !ok {
			return errors.New("superset workspaces create returned no workspace id")
		}
		id = s
		return nil
	})
	return id, err
}
