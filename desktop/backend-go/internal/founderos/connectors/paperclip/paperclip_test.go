package paperclip

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/rhl/businessos-backend/internal/founderos/connectors"
	"github.com/rhl/businessos-backend/internal/founderos/guard"
)

// Shapes captured from the live API (GET /companies/:id/agents and /org).
const rawAgents = `[
  {"id":"a1","name":"Conductor","status":"running","adapterType":"claude_local","model":"claude-fable-5","lastHeartbeatAt":"2026-09-29T10:00:00Z"},
  {"id":"a2","name":"Hermes Workers","status":"idle","adapterType":"hermes_gateway"},
  {"id":"a3","name":"TECH","status":"exploded","adapterType":"codex_local","adapterConfig":{"model":"gpt-5.6"}},
  {"id":"bad"},
  "not-an-object"
]`

const rawOrg = `[{"id":"a1","name":"Conductor","role":"ceo","status":"running","reports":[
  {"id":"a2","name":"Hermes Workers","role":"general","status":"idle","reports":[]},
  {"id":"a3","name":"TECH","role":"cto","status":"idle","reports":[{"id":"a4","name":"Sub","status":"idle","reports":[]}]}
]}, {"nonsense":true}, null]`

func resolver(t *testing.T, lines ...string) connectors.Resolver {
	t.Helper()
	p := filepath.Join(t.TempDir(), "env.local")
	if err := os.WriteFile(p, []byte(strings.Join(lines, "\n")+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, k := range []string{"PAPERCLIP_API_URL", "PAPERCLIP_BOARD_KEY", "PAPERCLIP_COMPANY_ID", "PAPERCLIP_COCKPIT_ISSUE_ID"} {
		t.Setenv(k, "")
	}
	return connectors.Resolver{EnvLocal: p}
}

type board struct {
	mu       sync.Mutex
	hits     []string
	bodies   map[string]string
	agents   string
	status   int
	issues   []map[string]any
	comments string
}

func (b *board) handler(t *testing.T) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		b.mu.Lock()
		defer b.mu.Unlock()
		b.hits = append(b.hits, r.Method+" "+r.URL.Path)
		if got := r.Header.Get("Authorization"); got != "Bearer board-key" {
			t.Errorf("auth header = %q", got)
		}
		if r.Body != nil {
			raw, _ := io.ReadAll(r.Body)
			if b.bodies == nil {
				b.bodies = map[string]string{}
			}
			b.bodies[r.Method+" "+r.URL.Path] = string(raw)
		}
		if b.status != 0 {
			w.WriteHeader(b.status)
			_, _ = io.WriteString(w, `{"error":"nope"}`)
			return
		}
		switch {
		case r.URL.Path == "/api/companies/c1/agents":
			_, _ = io.WriteString(w, b.agents)
		case r.URL.Path == "/api/companies/c1/org":
			_, _ = io.WriteString(w, rawOrg)
		case r.URL.Path == "/api/companies/c1/issues" && r.Method == http.MethodGet:
			limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
			offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
			end := offset + limit
			if end > len(b.issues) {
				end = len(b.issues)
			}
			page := []map[string]any{}
			if offset < len(b.issues) {
				page = b.issues[offset:end]
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"issues": page})
		case r.URL.Path == "/api/companies/c1/issues" && r.Method == http.MethodPost:
			b.issues = append(b.issues, map[string]any{"id": "new-1", "title": CockpitTitle, "status": "backlog", "issueNumber": 900})
			_, _ = io.WriteString(w, `{"issue":{"id":"new-1","identifier":"FOS-900","title":"FounderOS v1 Cockpit","status":"backlog"}}`)
		case r.URL.Path == "/api/companies/c1/heartbeat-runs":
			_, _ = io.WriteString(w, `{"runs":[
			  {"id":"r1","agentId":"a1","agentName":"Conductor","status":"succeeded","startedAt":"2026-09-29T09:00:00Z","finishedAt":"2026-09-29T09:04:00Z","usageJson":{"model":"claude-fable-5"}},
			  {"id":"r2","agent":{"id":"a2","name":"Hermes"},"status":"failed","error":"Internal error","resultJson":{"summary":"usage limit reached"}},
			  {"broken":true}]}`)
		case strings.HasPrefix(r.URL.Path, "/api/issues/") && strings.HasSuffix(r.URL.Path, "/comments") && r.Method == http.MethodGet:
			_, _ = io.WriteString(w, b.comments)
		case strings.HasPrefix(r.URL.Path, "/api/issues/") && strings.HasSuffix(r.URL.Path, "/comments") && r.Method == http.MethodPost:
			_, _ = io.WriteString(w, `{"comment":{"id":"c9","body":"hello","authorType":"user","createdAt":"2026-09-29T12:00:00Z"}}`)
		case strings.HasPrefix(r.URL.Path, "/api/issues/") && r.Method == http.MethodGet:
			id := strings.TrimPrefix(r.URL.Path, "/api/issues/")
			_, _ = fmt.Fprintf(w, `{"id":%q,"status":"in_progress","assigneeAgentId":"a9","assigneeUserId":"u1"}`, id)
		case strings.HasPrefix(r.URL.Path, "/api/issues/") && r.Method == http.MethodPatch:
			_, _ = io.WriteString(w, `{}`)
		case strings.HasPrefix(r.URL.Path, "/api/agents/"):
			_, _ = io.WriteString(w, `{}`)
		default:
			w.WriteHeader(404)
		}
	}
}

func (b *board) hitCount(prefix string) int {
	b.mu.Lock()
	defer b.mu.Unlock()
	n := 0
	for _, h := range b.hits {
		if strings.HasPrefix(h, prefix) {
			n++
		}
	}
	return n
}

func newBoard(t *testing.T, extra ...string) (*Connector, *board, *httptest.Server) {
	t.Helper()
	b := &board{agents: rawAgents}
	srv := httptest.NewServer(b.handler(t))
	t.Cleanup(srv.Close)
	lines := append([]string{"PAPERCLIP_API_URL=" + srv.URL + "/", "PAPERCLIP_BOARD_KEY=board-key", "PAPERCLIP_COMPANY_ID=c1"}, extra...)
	return New(resolver(t, lines...)), b, srv
}

func TestMeta(t *testing.T) {
	if Meta != (connectors.Meta{ID: "paperclip", Name: "Paperclip (agent harness)", Kind: connectors.KindOrchestration}) {
		t.Fatalf("Meta = %+v", Meta)
	}
}

func TestStatusNotConfiguredWithoutCreds(t *testing.T) {
	s := New(resolver(t, "PAPERCLIP_API_URL=http://x")).Status(context.Background())
	if s.State != connectors.StateNotConfigured || !strings.Contains(s.Detail, "PAPERCLIP_BOARD_KEY") {
		t.Fatalf("status = %+v", s)
	}
}

func TestStatusConnectedCountsAgentsAndRunning(t *testing.T) {
	c, _, _ := newBoard(t)
	s := c.Status(context.Background())
	if s.State != connectors.StateConnected || s.Detail != "Board live · 3 agents · 1 running" {
		t.Fatalf("status = %+v", s)
	}
	if s.Meta["agents"] != 3 || s.Meta["running"] != 1 {
		t.Errorf("meta = %v", s.Meta)
	}
}

func TestStatusErrorOnEmptyBoardAndOnRejectedKey(t *testing.T) {
	c, b, _ := newBoard(t)
	b.agents = `{"agents":[]}`
	if s := c.Status(context.Background()); s.State != connectors.StateError || !strings.Contains(s.Detail, "returned no agents") {
		t.Fatalf("empty board: %+v", s)
	}
	b.status = 401
	s := c.Status(context.Background())
	if s.State != connectors.StateError || !strings.Contains(s.Detail, "HTTP 401") {
		t.Fatalf("401: %+v", s)
	}
}

func TestAgentsMapsAndSkipsMalformedRows(t *testing.T) {
	c, _, _ := newBoard(t)
	agents, err := c.Agents(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(agents) != 3 || agents[0].Name != "Conductor" || *agents[0].Model != "claude-fable-5" {
		t.Fatalf("agents = %+v", agents)
	}
	if agents[1].Model != nil || agents[2].Status != "idle" || *agents[2].Model != "gpt-5.6" {
		t.Errorf("gateway model must be nil, unknown status idle, adapterConfig model used: %+v %+v", agents[1], agents[2])
	}
	if agents[0].LastHeartbeatAt == nil || *agents[0].LastHeartbeatAt != "2026-09-29T10:00:00Z" {
		t.Errorf("heartbeat = %v", agents[0].LastHeartbeatAt)
	}
}

func TestOrgFlattensPreorderWithDepthAndParent(t *testing.T) {
	c, _, _ := newBoard(t)
	nodes, err := c.Org(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	got := []string{}
	for _, n := range nodes {
		p := "-"
		if n.ParentID != nil {
			p = *n.ParentID
		}
		got = append(got, fmt.Sprintf("%s/%d/%s/%s", n.Name, n.Depth, p, n.Role))
	}
	want := "Conductor/0/-/ceo Hermes Workers/1/a1/general TECH/1/a1/cto Sub/2/a3/general"
	if strings.Join(got, " ") != want {
		t.Fatalf("org = %v", got)
	}
}

func TestIssuesAndRuns(t *testing.T) {
	c, b, _ := newBoard(t)
	b.issues = []map[string]any{
		{"id": "i1", "identifier": "FOS-3", "title": "Dashboard v0", "status": "done", "updatedAt": "2026-08-03T10:00:00Z"},
		{"id": "i2", "title": "New thing", "status": "in_progress", "assignee": map[string]any{"name": "Conductor"}},
		{"id": "nope"},
	}
	issues, err := c.Issues(context.Background(), 30)
	if err != nil {
		t.Fatal(err)
	}
	if len(issues) != 2 || issues[0].Identifier != "FOS-3" || issues[1].Identifier != "i2" || *issues[1].AssigneeName != "Conductor" {
		t.Fatalf("issues = %+v", issues)
	}
	runs, err := c.Runs(context.Background(), 30)
	if err != nil {
		t.Fatal(err)
	}
	if len(runs) != 2 || *runs[0].AgentName != "Conductor" || runs[1].AgentID != "a2" || *runs[1].AgentName != "Hermes" || runs[1].FinishedAt != nil {
		t.Fatalf("runs = %+v", runs)
	}
	fr, err := c.FailoverRuns(context.Background(), 40)
	if err != nil {
		t.Fatal(err)
	}
	if len(fr) != 1 || *fr[0].Model != "claude-fable-5" {
		t.Fatalf("failover runs (rows without agentId skipped) = %+v", fr)
	}
}

func TestReadsReturnAnErrorNotAnEmptyListWhenTheBoardRefuses(t *testing.T) {
	c, b, _ := newBoard(t)
	b.status = 403
	if _, err := c.Agents(context.Background()); err == nil || !strings.Contains(err.Error(), "403") {
		t.Fatalf("err = %v", err)
	}
	if _, err := c.Issues(context.Background(), 5); err == nil {
		t.Fatal("a refused read must not read as an empty board")
	}
}

func TestBreakerProbesOnceThenFailsFastAndRecovers(t *testing.T) {
	c, _, srv := newBoard(t)
	clock := time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC)
	c.now = func() time.Time { return clock }
	srv.Close() // network-level failure

	first := c.Status(context.Background())
	second := c.Status(context.Background())
	if first.State != connectors.StateError || !strings.Contains(second.Detail, "unreachable") || !strings.Contains(second.Detail, "not re-probed") {
		t.Fatalf("first=%+v second=%+v", first, second)
	}
	if strings.Contains(second.Detail, "returned no agents") {
		t.Fatal("unreachable must never read as an empty board")
	}
	clock = clock.Add(29 * time.Second)
	if s := c.Status(context.Background()); !strings.Contains(s.Detail, "not re-probed") {
		t.Fatalf("still inside the window: %+v", s)
	}
	clock = clock.Add(2 * time.Second)
	if s := c.Status(context.Background()); strings.Contains(s.Detail, "not re-probed") {
		t.Fatalf("window over, must probe again: %+v", s)
	}
}

func TestHTTPErrorDoesNotOpenTheBreaker(t *testing.T) {
	c, b, _ := newBoard(t)
	b.status = 500
	c.Status(context.Background())
	c.Status(context.Background())
	if n := b.hitCount("GET /api/companies/c1/agents"); n != 2 {
		t.Fatalf("hits = %d, want 2", n)
	}
}

func cockpitIssues(n int) []map[string]any {
	out := make([]map[string]any, 0, n)
	for i := 0; i < n; i++ {
		out = append(out, map[string]any{"id": fmt.Sprintf("x%d", i), "title": "other", "status": "todo"})
	}
	return out
}

func TestCockpitScanPaginatesAndPicksLowestOpenIssueNumber(t *testing.T) {
	c, b, _ := newBoard(t)
	b.issues = cockpitIssues(130)
	b.issues[5] = map[string]any{"id": "closed", "title": CockpitTitle, "status": "done", "issueNumber": 1}
	b.issues[110] = map[string]any{"id": "late", "title": CockpitTitle, "status": "backlog", "issueNumber": 40}
	b.issues[120] = map[string]any{"id": "early", "title": CockpitTitle, "status": "in_progress", "issueNumber": 12}
	b.comments = `{"comments":[
	  {"id":"c2","body":"On it.","authorType":"agent","authorAgentId":"a1","createdAt":"2026-09-29T10:01:00Z"},
	  {"id":"c1","body":"Status?","authorType":"user","createdAt":"2026-09-29T10:00:00Z"},
	  {"id":"c3","body":"gone","deletedAt":"2026-09-29T10:02:00Z","createdAt":"2026-09-29T10:02:00Z"},
	  {"id":"c4"}]}`

	thread, err := c.CockpitThread(context.Background(), 50)
	if err != nil {
		t.Fatal(err)
	}
	if len(thread) != 2 || thread[0].ID != "c1" || thread[1].AuthorType != "agent" || *thread[1].AuthorAgentID != "a1" {
		t.Fatalf("thread = %+v", thread)
	}
	if b.hitCount("GET /api/issues/early/comments") != 1 {
		t.Fatalf("hits = %v", b.hits)
	}
	if n := b.hitCount("GET /api/companies/c1/issues"); n != 2 {
		t.Fatalf("scan pages = %d, want 2", n)
	}
}

func TestPinnedCockpitIssueSkipsTheScan(t *testing.T) {
	c, b, _ := newBoard(t, "PAPERCLIP_COCKPIT_ISSUE_ID=pinned-7")
	b.comments = `[]`
	if _, err := c.CockpitThread(context.Background(), 10); err != nil {
		t.Fatal(err)
	}
	if b.hitCount("GET /api/companies/c1/issues") != 0 || b.hitCount("GET /api/issues/pinned-7/comments") != 1 {
		t.Fatalf("hits = %v", b.hits)
	}
}

func TestWritesAreRefusedWhileBridgeWritesIsOff(t *testing.T) {
	t.Setenv("FOUNDEROS_WRITES", "0")
	guard.Reset()
	c, b, _ := newBoard(t, "PAPERCLIP_COCKPIT_ISSUE_ID=pinned-7")
	ctx := context.Background()

	checks := map[string]error{}
	_, checks["create"] = c.CreateIssue(ctx, "t", "d")
	_, checks["heartbeat"] = c.InvokeHeartbeat(ctx, "a1")
	_, checks["comment"] = c.PostCockpitMessage(ctx, "hi")
	_, checks["model"] = c.SetAgentModel(ctx, "a1", "m")
	_, checks["reassign"] = c.ReassignCockpitIssue(ctx, "a2")
	_, checks["clear"] = c.ClearAgentError(ctx, "a1")
	for name, err := range checks {
		if !errors.Is(err, guard.ErrWritesDisabled) {
			t.Errorf("%s: err = %v, want ErrWritesDisabled", name, err)
		}
	}
	if got := c.RepairCockpitIssue(ctx, "a1"); got != "failed" {
		t.Errorf("repair = %q, want failed", got)
	}
	for _, h := range b.hits {
		if !strings.HasPrefix(h, "GET ") {
			t.Errorf("a write reached the board: %s", h)
		}
	}
	if len(guard.Refused()) < 7 {
		t.Errorf("refusals not recorded: %v", guard.Refused())
	}
}

func TestEnsureCockpitCreateIsGuardedWhenNoneExists(t *testing.T) {
	t.Setenv("FOUNDEROS_WRITES", "0")
	c, b, _ := newBoard(t)
	b.issues = cockpitIssues(3)
	if _, err := c.CockpitThread(context.Background(), 10); !errors.Is(err, guard.ErrWritesDisabled) {
		t.Fatalf("err = %v", err)
	}
	if b.hitCount("POST") != 0 {
		t.Fatalf("hits = %v", b.hits)
	}
}

func TestWritesGoThroughWhenEnabled(t *testing.T) {
	t.Setenv("FOUNDEROS_WRITES", "1")
	c, b, _ := newBoard(t)
	b.issues = cockpitIssues(3)
	ctx := context.Background()

	id, err := c.EnsureCockpitIssue(ctx)
	if err != nil || id != "new-1" {
		t.Fatalf("ensure = %q, %v", id, err)
	}
	var created map[string]any
	_ = json.Unmarshal([]byte(b.bodies["POST /api/companies/c1/issues"]), &created)
	if created["title"] != CockpitTitle || created["status"] != "backlog" || created["assigneeAgentId"] != "a1" {
		t.Errorf("create body = %v", created)
	}
	msg, err := c.PostCockpitMessage(ctx, "hello")
	if err != nil || msg == nil || msg.ID != "c9" {
		t.Fatalf("post = %+v, %v", msg, err)
	}
	if got := c.RepairCockpitIssue(ctx, "a1"); got != "patched" {
		t.Fatalf("repair = %q", got)
	}
	var patch map[string]any
	_ = json.Unmarshal([]byte(b.bodies["PATCH /api/issues/new-1"]), &patch)
	if patch["status"] != "backlog" || patch["assigneeAgentId"] != "a1" || patch["assigneeUserId"] != nil {
		t.Errorf("patch = %v", patch)
	}
	if _, ok := patch["assigneeUserId"]; !ok {
		t.Errorf("patch must clear the human assignee: %v", patch)
	}
	if ok, err := c.SetAgentModel(ctx, "a1", "gpt-5.6"); !ok || err != nil {
		t.Fatalf("set model = %v %v", ok, err)
	}
	if b.bodies["PATCH /api/agents/a1"] != `{"adapterConfig":{"model":"gpt-5.6"}}` {
		t.Errorf("model body = %s", b.bodies["PATCH /api/agents/a1"])
	}
}

func TestPostOnceDedupesOnTheMarker(t *testing.T) {
	t.Setenv("FOUNDEROS_WRITES", "1")
	c, b, _ := newBoard(t, "PAPERCLIP_COCKPIT_ISSUE_ID=pinned-7")
	b.comments = `[{"id":"c1","body":"[failover] Conductor quota 2026-09-29T09:00:00Z\nmsg","authorType":"agent","createdAt":"2026-09-29T10:00:00Z"}]`
	if got := c.PostOnce(context.Background(), "[failover] Conductor quota 2026-09-29T09:00:00Z", "x"); got != "already" {
		t.Fatalf("got %q", got)
	}
	if got := c.PostOnce(context.Background(), "[failover] TECH quota 2026-09-29T09:05:00Z", "y"); got != "posted" {
		t.Fatalf("got %q", got)
	}
}

func TestCockpitRepairPatch(t *testing.T) {
	if p := CockpitRepairPatch("done", "", "", "a1"); p != nil {
		t.Errorf("closed issue must be left alone: %v", p)
	}
	if p := CockpitRepairPatch("backlog", "a1", "", "a1"); p != nil {
		t.Errorf("safe shape needs no patch: %v", p)
	}
	p := CockpitRepairPatch("in_progress", "a1", "", "")
	if len(p) != 1 || p["status"] != "backlog" {
		t.Errorf("patch = %v", p)
	}
}
