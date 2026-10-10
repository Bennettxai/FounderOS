package api

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/rhl/businessos-backend/internal/founderos/connectors/paperclip"
	agentspage "github.com/rhl/businessos-backend/internal/founderos/pages/agents"
	"github.com/rhl/businessos-backend/internal/founderos/pages/osdata"
)

func liveFake() *fakeBoard {
	started := testNow.Add(-time.Hour).Format(time.RFC3339)
	return &fakeBoard{
		agents: []paperclip.Agent{
			{ID: "a1", Name: "Conductor", Status: "running", Model: sp("claude-fable-5")},
			{ID: "a2", Name: "Tech", Status: "error", Model: sp("claude-fable-5")},
		},
		issues: []paperclip.Issue{{ID: "i1", Identifier: "FOS-1", Title: "t", Status: "in_review"}},
		runs:   []paperclip.Run{{ID: "r1", AgentID: "a1", Status: "succeeded", StartedAt: &started}},
		failRuns: []paperclip.FailoverRun{{AgentID: "a2", Status: "failed", Model: sp("claude-fable-5"),
			Error: sp("You're out of usage credits."), FinishedAt: sp(testNow.Add(-time.Minute).Format(time.RFC3339))}},
	}
}

func TestAgentsRosterAndActivity(t *testing.T) {
	d := pageDeps(t, liveFake())
	var roster struct{ Agents []osdata.Agent }
	if code := getJSON(t, d, "GET", "/api/founderos/pages/agents", nil, &roster); code != 200 || len(roster.Agents) != 4 {
		t.Fatalf("roster %d %+v", code, roster)
	}
	var act struct{ Events []agentspage.ActivityEvent }
	if code := getJSON(t, d, "GET", "/api/founderos/pages/agents/activity?limit=1", nil, &act); code != 200 || len(act.Events) != 1 || act.Events[0].Kind != "run" {
		t.Fatalf("activity %d %+v", code, act)
	}
}

func TestAgentsRoutesNeedASession(t *testing.T) {
	d := &Deps{}
	for _, p := range []string{"/api/founderos/pages/agents", "/api/founderos/pages/board/live", "/api/founderos/pages/agents/view"} {
		w := httptest.NewRecorder()
		router(t, d).ServeHTTP(w, httptest.NewRequest("GET", p, nil))
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("%s without session: %d", p, w.Code)
		}
	}
}

func TestBoardLiveUpAndDown(t *testing.T) {
	fb := liveFake()
	d := pageDeps(t, fb)
	var live struct {
		agentspage.Board
		Volume agentspage.Volume     `json:"volume"`
		Stats  agentspage.BoardStats `json:"stats"`
	}
	if code := getJSON(t, d, "GET", "/api/founderos/pages/board/live", nil, &live); code != 200 || !live.Connected || len(live.Agents) != 2 || live.Volume.Headline != 2 || live.Stats.Running != 1 || live.Decisions == nil {
		t.Fatalf("live %d %+v", code, live)
	}
	fb.down = true
	live.Board = agentspage.Board{}
	if code := getJSON(t, d, "GET", "/api/founderos/pages/board/live", nil, &live); code != 200 || live.Connected || live.Error == "" || len(live.Agents) != 0 || live.Volume.Chips[0].Text != "board unreachable" {
		t.Fatalf("down %d %+v", code, live)
	}
	var view struct {
		BoardURL  *string          `json:"boardUrl"`
		HermesURL string           `json:"hermesUrl"`
		Board     agentspage.Board `json:"board"`
	}
	t.Setenv("HERMES_DASH_URL", "https://hermes.example:9119")
	t.Setenv("PAPERCLIP_API_URL", "")
	if code := getJSON(t, d, "GET", "/api/founderos/pages/agents/view", nil, &view); code != 200 || view.HermesURL != "https://hermes.example:9119" || view.BoardURL != nil || view.Board.Connected {
		t.Fatalf("view %d %+v", code, view)
	}
	// unset: v1's generic placeholder host, never a private machine name
	t.Setenv("HERMES_DASH_URL", "")
	if code := getJSON(t, d, "GET", "/api/founderos/pages/agents/view", nil, &view); code != 200 || view.HermesURL != "https://os.example.internal:9000" {
		t.Fatalf("default hermes %d %+v", code, view.HermesURL)
	}
}

func TestBoardRunIsGuarded(t *testing.T) {
	fb := liveFake()
	d := pageDeps(t, fb)
	t.Setenv("FOUNDEROS_WRITES", "0")
	var body map[string]any
	if code := getJSON(t, d, "POST", "/api/founderos/pages/board/agents/a1/run", nil, &body); code != http.StatusForbidden || body["guarded"] != true {
		t.Fatalf("guarded run %d %v", code, body)
	}
	if len(fb.calls) != 0 {
		t.Fatalf("a write reached the board: %v", fb.calls)
	}
	t.Setenv("FOUNDEROS_WRITES", "1")
	if code := getJSON(t, d, "POST", "/api/founderos/pages/board/agents/a1/run", nil, &body); code != http.StatusAccepted || body["ok"] != true {
		t.Fatalf("open run %d %v", code, body)
	}
}

func TestFailoverDryRunAndGuardedApply(t *testing.T) {
	fb := liveFake()
	d := pageDeps(t, fb)
	var dry struct {
		OK        bool                        `json:"ok"`
		Applied   bool                        `json:"applied"`
		Inspected int                         `json:"inspected"`
		Actions   []agentspage.FailoverAction `json:"actions"`
	}
	if code := getJSON(t, d, "GET", "/api/founderos/pages/agents/failover", nil, &dry); code != 200 || dry.Applied || dry.Inspected != 2 || len(dry.Actions) != 2 {
		t.Fatalf("dry run %d %+v", code, dry)
	}
	if len(fb.calls) != 0 {
		t.Fatalf("dry run wrote: %v", fb.calls)
	}
	t.Setenv("FOUNDEROS_WRITES", "0")
	var applied struct {
		Applied bool `json:"applied"`
		Actions []struct {
			AgentName string `json:"agentName"`
			Moved     bool   `json:"moved"`
		} `json:"actions"`
		Cockpit string `json:"cockpit"`
	}
	if code := getJSON(t, d, "POST", "/api/founderos/pages/agents/failover", nil, &applied); code != 200 || !applied.Applied || len(applied.Actions) != 2 || applied.Actions[0].Moved || applied.Cockpit != "failed" {
		t.Fatalf("guarded apply %d %+v", code, applied)
	}
	if len(fb.calls) != 0 {
		t.Fatalf("guarded apply reached the board: %v", fb.calls)
	}
	fb.down = true
	var down struct {
		Inspected int      `json:"inspected"`
		Notes     []string `json:"notes"`
	}
	if code := getJSON(t, d, "GET", "/api/founderos/pages/agents/failover", nil, &down); code != 200 || down.Inspected != 0 || !strings.Contains(strings.Join(down.Notes, " "), "unreachable") {
		t.Fatalf("down %d %+v", code, down)
	}
}

func TestDeliverablesListViewDownloadAndDecisions(t *testing.T) {
	base := t.TempDir()
	dir := filepath.Join(base, "ws1", "deliverables")
	_ = os.MkdirAll(dir, 0o755)
	_ = os.WriteFile(filepath.Join(dir, "STAGED-reply.md"), []byte("# Reply to Sam\n\nThanks."), 0o644)
	_ = os.WriteFile(filepath.Join(dir, "chart.png"), []byte("\x89PNG"), 0o644)
	d := pageDeps(t, liveFake())

	// Unset, the bridge looks where prod looks (lib/board-deliverables
	// WORKSPACES_DIR): ~/.paperclip/instances/default/workspaces. A box
	// without that directory is an honest empty, never an error.
	t.Setenv("PAPERCLIP_WORKSPACES_DIR", "")
	home := t.TempDir()
	t.Setenv("HOME", home)
	var none struct {
		Available bool   `json:"available"`
		Reason    string `json:"reason"`
		Dir       string `json:"dir"`
		Groups    []any  `json:"groups"`
	}
	if code := getJSON(t, d, "GET", "/api/founderos/pages/board/deliverables", nil, &none); code != 200 || none.Available || none.Reason == "" || none.Groups == nil {
		t.Fatalf("no dir %d %+v", code, none)
	}
	if want := filepath.Join(home, ".paperclip", "instances", "default", "workspaces"); none.Dir != want {
		t.Fatalf("default dir = %q, want %q", none.Dir, want)
	}
	defDir := filepath.Join(home, ".paperclip", "instances", "default", "workspaces", "wsd", "deliverables")
	_ = os.MkdirAll(defDir, 0o755)
	_ = os.WriteFile(filepath.Join(defDir, "STAGED-default.md"), []byte("# Default\n"), 0o644)
	var def struct {
		Available bool `json:"available"`
		NeedsYou  struct {
			Open []agentspage.Classified `json:"open"`
		} `json:"needsYou"`
	}
	if code := getJSON(t, d, "GET", "/api/founderos/pages/board/deliverables", nil, &def); code != 200 || !def.Available || len(def.NeedsYou.Open) != 1 {
		t.Fatalf("default dir list %d %+v", code, def)
	}

	t.Setenv("PAPERCLIP_WORKSPACES_DIR", base)
	// Home's Needs you links "open board" to the board itself (prod HomeNeedsYou boardUrl).
	t.Setenv("PAPERCLIP_API_URL", "http://board.test:3100")
	var list struct {
		BoardURL  *string                       `json:"boardUrl"`
		Available bool                          `json:"available"`
		Groups    []agentspage.DeliverableGroup `json:"groups"`
		NeedsYou  struct {
			Open    []agentspage.Classified `json:"open"`
			Decided []any                   `json:"decided"`
		} `json:"needsYou"`
	}
	if code := getJSON(t, d, "GET", "/api/founderos/pages/board/deliverables", nil, &list); code != 200 || !list.Available || len(list.Groups) != 1 || len(list.NeedsYou.Open) != 1 {
		t.Fatalf("list %d %+v", code, list)
	}
	if list.BoardURL == nil || *list.BoardURL != "http://board.test:3100" {
		t.Fatalf("boardUrl = %v", list.BoardURL)
	}
	item := list.NeedsYou.Open[0]

	var view struct {
		Kind      string `json:"kind"`
		Text      string `json:"text"`
		Truncated bool   `json:"truncated"`
	}
	if code := getJSON(t, d, "GET", "/api/founderos/pages/board/deliverables?file=ws1/STAGED-reply.md&mode=view", nil, &view); code != 200 || view.Kind != "text" || !strings.Contains(view.Text, "Reply to Sam") {
		t.Fatalf("view %d %+v", code, view)
	}
	w := httptest.NewRecorder()
	router(t, d).ServeHTTP(w, authed("GET", "/api/founderos/pages/board/deliverables?file=ws1/chart.png&inline=1", nil))
	if w.Code != 200 || w.Header().Get("Content-Type") != "image/png" || !strings.HasPrefix(w.Header().Get("Content-Disposition"), "inline") {
		t.Fatalf("inline %d %v", w.Code, w.Header())
	}
	if code := getJSON(t, d, "GET", "/api/founderos/pages/board/deliverables?file=../etc/passwd", nil, nil); code != 400 {
		t.Fatalf("traversal %d", code)
	}

	post := func(body string) (int, map[string]any) {
		var out map[string]any
		code := getJSON(t, d, "POST", "/api/founderos/pages/board/deliverables/decision", []byte(body), &out)
		return code, out
	}
	if code, _ := post(`{"decision":"approved"}`); code != 400 {
		t.Fatalf("no id %d", code)
	}
	if code, _ := post(`{"id":"x","decision":"maybe"}`); code != 400 {
		t.Fatalf("bad kind %d", code)
	}
	if code, _ := post(`{"items":[{"id":"a"}],"decision":"approved"}`); code != 400 {
		t.Fatalf("bulk approve must be refused: %d", code)
	}
	// Approve means send/publish/go ahead: an outward effect, so with
	// FOUNDEROS_WRITES=0 the guard refuses it and the item stays in front of him.
	t.Setenv("FOUNDEROS_WRITES", "0")
	if code, out := post(`{"id":"` + item.ID + `","decision":"approved","decidedRevision":"` + item.Revision + `"}`); code != 403 || out["guarded"] != true {
		t.Fatalf("approve with writes off must be guarded: %d %v", code, out)
	}
	if code := getJSON(t, d, "GET", "/api/founderos/pages/board/deliverables", nil, &list); code != 200 || len(list.NeedsYou.Open) != 1 {
		t.Fatalf("a refused approve must not record: %+v", list.NeedsYou)
	}
	// Dismiss is his own view's state, never outward: allowed with writes off.
	if code, _ := post(`{"id":"` + item.ID + `","decision":"dismissed","decidedRevision":"` + item.Revision + `"}`); code != 200 {
		t.Fatalf("dismiss with writes off %d", code)
	}
	if code, _ := post(`{"id":"` + item.ID + `","decision":null}`); code != 200 {
		t.Fatalf("undo dismiss %d", code)
	}
	t.Setenv("FOUNDEROS_WRITES", "1")
	if code, _ := post(`{"id":"` + item.ID + `","decision":"approved","decidedRevision":"` + item.Revision + `"}`); code != 200 {
		t.Fatalf("decide %d", code)
	}
	if code := getJSON(t, d, "GET", "/api/founderos/pages/board/deliverables", nil, &list); code != 200 || len(list.NeedsYou.Open) != 0 || len(list.NeedsYou.Decided) != 1 {
		t.Fatalf("after decide %+v", list.NeedsYou)
	}
	if code, out := post(`{"items":[{"id":"a"},{"id":"b"}],"decision":"dismissed"}`); code != 200 || out["count"].(float64) != 2 {
		t.Fatalf("bulk dismiss %d %v", code, out)
	}
	if code, _ := post(`{"id":"` + item.ID + `","decision":null}`); code != 200 {
		t.Fatalf("undo %d", code)
	}
	if code := getJSON(t, d, "GET", "/api/founderos/pages/board/deliverables", nil, &list); len(list.NeedsYou.Open) != 1 || code != 200 {
		t.Fatalf("after undo %+v", list.NeedsYou)
	}
	_ = bytes.MinRead
}
