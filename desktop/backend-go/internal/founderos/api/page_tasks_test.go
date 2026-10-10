package api

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/rhl/businessos-backend/internal/founderos/connectors/paperclip"
	"github.com/rhl/businessos-backend/internal/founderos/pages/osdata"
	"github.com/rhl/businessos-backend/internal/founderos/pages/shared"
	"github.com/rhl/businessos-backend/internal/founderos/pages/tasks"
)

func tasksBoard() *fakeBoard {
	up := testNow.Add(-2 * time.Hour).Format(time.RFC3339)
	return &fakeBoard{
		agents: []paperclip.Agent{{ID: "b1", Name: "Conductor", Status: "idle"}},
		issues: []paperclip.Issue{
			{ID: "i1", Identifier: "FOS-1", Title: "Ship reel", Status: "in_progress", UpdatedAt: &up},
			{ID: "i2", Identifier: "FOS-2", Title: "Stuck", Status: "blocked", UpdatedAt: &up},
		},
	}
}

type tasksViewBody struct {
	Tasks      []osdata.Task     `json:"tasks"`
	AgentNames map[string]string `json:"agentNames"`
	Issues     []paperclip.Issue `json:"issues"`
	Board      struct {
		Connected bool    `json:"connected"`
		Error     string  `json:"error"`
		URL       *string `json:"url"`
	} `json:"board"`
	Crons  []osdata.Cron   `json:"crons"`
	Jobs   []shared.JobRow `json:"jobs"`
	Volume tasks.Result    `json:"volume"`
	Stats  map[string]any  `json:"cronStats"`
}

func TestTasksPageViewBoardUpAndDown(t *testing.T) {
	fb := tasksBoard()
	d := pageDeps(t, fb)
	t.Setenv("PAPERCLIP_API_URL", "http://board.test:3100")
	var v tasksViewBody
	if code := getJSON(t, d, http.MethodGet, "/api/founderos/pages/tasks", nil, &v); code != 200 {
		t.Fatalf("view %d", code)
	}
	if len(v.Tasks) != 2 || v.AgentNames["crm-pulse"] != "CRM Pulse" || len(v.Issues) != 2 || !v.Board.Connected || v.Board.URL == nil || *v.Board.URL != "http://board.test:3100" {
		t.Fatalf("view %+v", v)
	}
	if len(v.Crons) != 2 || len(v.Jobs) != 2 || v.Stats["cron-a"] == nil {
		t.Fatalf("crons %+v", v)
	}
	if v.Volume.Headline != 4 || v.Volume.Board.Blocked != 1 || !v.Volume.BoardOnline || v.Volume.Cron.RunsInWindow != 2 {
		t.Fatalf("volume %+v", v.Volume)
	}

	fb.down = true
	v = tasksViewBody{}
	if code := getJSON(t, d, http.MethodGet, "/api/founderos/pages/tasks", nil, &v); code != 200 {
		t.Fatalf("down view %d", code)
	}
	if v.Board.Connected || v.Board.Error == "" || v.Issues == nil || len(v.Issues) != 0 || v.Volume.BoardOnline || v.Volume.Meters[2].Display != "board empty or offline" {
		t.Fatalf("down %+v", v)
	}
}

func TestBoardTasksReadAndGuardedCreate(t *testing.T) {
	fb := tasksBoard()
	d := pageDeps(t, fb)
	var list struct{ Issues []paperclip.Issue }
	if code := getJSON(t, d, http.MethodGet, "/api/founderos/pages/board/tasks", nil, &list); code != 200 || len(list.Issues) != 2 {
		t.Fatalf("list %d %+v", code, list)
	}
	var out map[string]any
	if code := getJSON(t, d, http.MethodPost, "/api/founderos/pages/board/tasks", []byte(`{"title":"  "}`), &out); code != 400 {
		t.Fatalf("no title %d", code)
	}
	t.Setenv("FOUNDEROS_WRITES", "0")
	if code := getJSON(t, d, http.MethodPost, "/api/founderos/pages/board/tasks", []byte(`{"title":"Do it"}`), &out); code != http.StatusForbidden || out["guarded"] != true {
		t.Fatalf("guarded %d %v", code, out)
	}
	for _, c := range fb.calls {
		if strings.HasPrefix(c, "issue.create") {
			t.Fatal("guarded create reached the board")
		}
	}
	t.Setenv("FOUNDEROS_WRITES", "1")
	var created struct{ Issue paperclip.Issue }
	if code := getJSON(t, d, http.MethodPost, "/api/founderos/pages/board/tasks", []byte(`{"title":"Do it","description":"Route to the TECH pillar."}`), &created); code != http.StatusCreated || created.Issue.Title != "Do it" {
		t.Fatalf("create %d %+v", code, created)
	}
	fb.down = true
	if code := getJSON(t, d, http.MethodGet, "/api/founderos/pages/board/tasks", nil, &out); code != http.StatusBadGateway || out["error"] == nil {
		t.Fatalf("down list %d %v", code, out)
	}
}

func TestAgentsWorkCRUD(t *testing.T) {
	d := pageDeps(t, tasksBoard())
	var got struct {
		Tasks []osdata.Task `json:"tasks"`
		Crons []osdata.Cron `json:"crons"`
	}
	if code := getJSON(t, d, http.MethodGet, "/api/founderos/pages/agents/work?agentId=crm-pulse", nil, &got); code != 200 || len(got.Tasks) != 1 || len(got.Crons) != 1 {
		t.Fatalf("by agent %d %+v", code, got)
	}
	var out map[string]any
	for _, bad := range []string{`{}`, `{"kind":"task","agentId":"x"}`, `{"kind":"cron","agentId":"x","schedule":"0 9 * * *"}`, `nope`} {
		if code := getJSON(t, d, http.MethodPost, "/api/founderos/pages/agents/work", []byte(bad), nil); code != 400 {
			t.Fatalf("%s → %d", bad, code)
		}
	}
	if code := getJSON(t, d, http.MethodPost, "/api/founderos/pages/agents/work", []byte(`{"kind":"cron","agentId":"x","schedule":"every day","description":"d"}`), &out); code != 400 ||
		out["error"] != `invalid cron schedule: every day — use 5 fields like "0 9 * * 1-5"` {
		t.Fatalf("bad cron %d %v", code, out)
	}
	var task struct {
		OK   bool        `json:"ok"`
		Task osdata.Task `json:"task"`
	}
	if code := getJSON(t, d, http.MethodPost, "/api/founderos/pages/agents/work", []byte(`{"kind":"task","agentId":"crm-pulse","title":"New"}`), &task); code != 200 || !task.OK || task.Task.Status != "open" || task.Task.ID == "" {
		t.Fatalf("create task %d %+v", code, task)
	}
	var cron struct {
		Cron osdata.Cron `json:"cron"`
	}
	if code := getJSON(t, d, http.MethodPost, "/api/founderos/pages/agents/work", []byte(`{"kind":"cron","agentId":"crm-pulse","schedule":"0 9 * * 1-5","description":"Weekdays"}`), &cron); code != 200 || !cron.Cron.Enabled {
		t.Fatalf("create cron %d %+v", code, cron)
	}
	if code := getJSON(t, d, http.MethodPatch, "/api/founderos/pages/agents/work", []byte(`{"kind":"task","id":"`+task.Task.ID+`","status":"review"}`), &out); code != 200 {
		t.Fatalf("patch task %d %v", code, out)
	}
	if code := getJSON(t, d, http.MethodPatch, "/api/founderos/pages/agents/work", []byte(`{"kind":"task","id":"x","status":"nope"}`), nil); code != 400 {
		t.Fatalf("bad status %d", code)
	}
	if code := getJSON(t, d, http.MethodPatch, "/api/founderos/pages/agents/work", []byte(`{"kind":"task","id":"ghost","status":"done"}`), nil); code != 404 {
		t.Fatalf("missing task %d", code)
	}
	if code := getJSON(t, d, http.MethodPatch, "/api/founderos/pages/agents/work", []byte(`{"kind":"cron","id":"`+cron.Cron.ID+`","enabled":false}`), nil); code != 200 {
		t.Fatalf("patch cron %d", code)
	}
	if code := getJSON(t, d, http.MethodPatch, "/api/founderos/pages/agents/work", []byte(`{"kind":"cron","id":"x"}`), nil); code != 400 {
		t.Fatalf("cron without enabled %d", code)
	}
	all := struct {
		Tasks []osdata.Task `json:"tasks"`
		Crons []osdata.Cron `json:"crons"`
	}{}
	getJSON(t, d, http.MethodGet, "/api/founderos/pages/agents/work", nil, &all)
	if len(all.Tasks) != 3 || all.Tasks[0].Status != "review" || len(all.Crons) != 3 || all.Crons[0].Enabled {
		t.Fatalf("after patch %+v", all)
	}
	for _, body := range []string{`{"kind":"task","id":"` + task.Task.ID + `"}`, `{"kind":"cron","id":"` + cron.Cron.ID + `"}`} {
		if code := getJSON(t, d, http.MethodDelete, "/api/founderos/pages/agents/work", []byte(body), nil); code != 200 {
			t.Fatalf("delete %s %d", body, code)
		}
	}
	if code := getJSON(t, d, http.MethodDelete, "/api/founderos/pages/agents/work", []byte(`{"kind":"other","id":"x"}`), nil); code != 400 {
		t.Fatalf("bad delete %d", code)
	}
	getJSON(t, d, http.MethodGet, "/api/founderos/pages/agents/work", nil, &all)
	if len(all.Tasks) != 2 || len(all.Crons) != 2 {
		t.Fatalf("after delete %+v", all)
	}
}
