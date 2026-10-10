package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/rhl/businessos-backend/internal/founderos/agents"
	"github.com/rhl/businessos-backend/internal/founderos/pages/osdata"
	"github.com/rhl/businessos-backend/internal/founderos/pages/shared"
	"github.com/rhl/businessos-backend/internal/founderos/pages/workflows"
)

const wfInput = `{"name":"Test onboarding flow","subtitle":"s","steps":[
 {"title":"Kickoff","detail":"d","ownerKind":"agent","owner":"CRM Pulse","hoursPerWeek":1,"tools":["notion"],"automation":null,"branchFromIndex":null,"branchCondition":null},
 {"title":"Approved","detail":"d","ownerKind":"human","owner":"the operator","hoursPerWeek":2,"tools":[],"automation":null,"branchFromIndex":0,"branchCondition":"approved"}]}`

func TestWorkflowsCRUD(t *testing.T) {
	d := pageDeps(t, nil)
	var list struct{ Workflows []osdata.Workflow }
	if code := getJSON(t, d, "GET", "/api/founderos/pages/workflows", nil, &list); code != 200 || len(list.Workflows) != 1 {
		t.Fatalf("list %d %+v", code, list)
	}
	var created struct{ Workflow osdata.Workflow }
	if code := getJSON(t, d, "POST", "/api/founderos/pages/workflows", []byte(wfInput), &created); code != 201 || !strings.HasPrefix(created.Workflow.ID, "wf-test-onboarding-flow-") || created.Workflow.Order != 1 {
		t.Fatalf("create %d %+v", code, created)
	}
	w := created.Workflow
	if w.Steps[1].Branch == nil || w.Steps[1].Branch.From != w.Steps[0].ID {
		t.Fatalf("branch %+v", w.Steps[1])
	}
	var bad map[string]any
	for _, body := range []string{`{"name":"","subtitle":"","steps":[]}`, `nope`, strings.Replace(wfInput, `"branchFromIndex":null`, `"branchFromIndex":1`, 1)} {
		if code := getJSON(t, d, "POST", "/api/founderos/pages/workflows", []byte(body), &bad); code != 400 {
			t.Fatalf("bad create %d for %s", code, body)
		}
	}
	patched := strings.Replace(wfInput, "Test onboarding flow", "Renamed", 1)
	var up struct{ Workflow osdata.Workflow }
	if code := getJSON(t, d, "PATCH", "/api/founderos/pages/workflows/"+w.ID, []byte(patched), &up); code != 200 || up.Workflow.ID != w.ID || up.Workflow.Name != "Renamed" || up.Workflow.Order != w.Order {
		t.Fatalf("patch %d %+v", code, up)
	}
	if code := getJSON(t, d, "PATCH", "/api/founderos/pages/workflows/wf-nope", []byte(wfInput), &bad); code != 404 {
		t.Fatalf("patch unknown %d", code)
	}
	if code := getJSON(t, d, "DELETE", "/api/founderos/pages/workflows/"+w.ID, nil, &bad); code != 200 {
		t.Fatalf("delete %d", code)
	}
	if code := getJSON(t, d, "DELETE", "/api/founderos/pages/workflows/"+w.ID, nil, &bad); code != 404 {
		t.Fatalf("delete again %d", code)
	}
}

func TestWorkflowsView(t *testing.T) {
	d := pageDeps(t, nil)
	var v struct {
		Workflows     []osdata.Workflow            `json:"workflows"`
		Jobs          []shared.JobRow              `json:"jobs"`
		Volume        workflows.WorkflowsVolume    `json:"volume"`
		Agents        []struct{ ID, Name string }  `json:"agents"`
		AgentPresence map[string]string            `json:"agentPresence"`
		RunsByOwner   map[string][]osdata.AgentRun `json:"runsByOwner"`
		ToolIDs       []string                     `json:"toolIds"`
	}
	if code := getJSON(t, d, "GET", "/api/founderos/pages/workflows/view", nil, &v); code != 200 {
		t.Fatalf("view %d", code)
	}
	if len(v.Jobs) != 2 || v.Volume.Headline != 2 || len(v.Agents) != 4 || v.AgentPresence["Sales Agent"] != "active" || v.AgentPresence["CRM Pulse"] != "inactive" {
		t.Fatalf("view %+v", v)
	}
	if runs := v.RunsByOwner["Sales Agent"]; len(runs) != 1 || runs[0].Summary != "failed" {
		t.Fatalf("runs by owner %+v", v.RunsByOwner)
	}
	if _, ok := v.RunsByOwner["the operator"]; !ok || strings.Join(v.ToolIDs, ",") != "typeform,zoom" {
		t.Fatalf("human owner / tools %+v %v", v.RunsByOwner, v.ToolIDs)
	}
}

type wfFakeRunner struct {
	calls int
	out   []workflows.CLIResult
}

func (f *wfFakeRunner) Run(_ context.Context, _ string) workflows.CLIResult {
	f.calls++
	return f.out[min(f.calls-1, len(f.out)-1)]
}

func TestWorkflowDraftUsesTheInjectedRunner(t *testing.T) {
	d := pageDeps(t, nil)
	valid := `{"name":"Weekly review","subtitle":"","steps":[{"title":"Pull","detail":"d","ownerKind":"agent","owner":"M","hoursPerWeek":1,"tools":[],"automation":null,"branchFromIndex":null,"branchCondition":null}]}`
	env, _ := json.Marshal(map[string]string{"result": valid})
	fake := &wfFakeRunner{out: []workflows.CLIResult{{OK: true, Stdout: "not json"}, {OK: true, Stdout: string(env)}}}
	restore := SetWorkflowDraftRunner(fake)
	defer restore()

	var r workflows.DraftResult
	if code := getJSON(t, d, "POST", "/api/founderos/pages/workflows/draft", []byte(`{"prompt":""}`), &r); code != 400 || fake.calls != 0 {
		t.Fatalf("empty prompt %d calls=%d", code, fake.calls)
	}
	if code := getJSON(t, d, "POST", "/api/founderos/pages/workflows/draft", []byte(`{"prompt":"weekly review"}`), &r); code != 200 || !r.OK || r.Draft.Name != "Weekly review" || fake.calls != 2 {
		t.Fatalf("retry draft %d %+v calls=%d", code, r, fake.calls)
	}
	var n int
	_ = d.Pool.QueryRow(context.Background(), `SELECT count(*) FROM founderos_workflows`).Scan(&n)
	if n != 1 {
		t.Fatalf("draft wrote to the db: %d rows", n)
	}
	SetWorkflowDraftRunner(&wfFakeRunner{out: []workflows.CLIResult{{Unavailable: true}}})
	if code := getJSON(t, d, "POST", "/api/founderos/pages/workflows/draft", []byte(`{"prompt":"x"}`), &r); code != 200 || r.OK || !r.Unavailable {
		t.Fatalf("unavailable %d %+v", code, r)
	}
}

func TestCronRunNow(t *testing.T) {
	d := pageDeps(t, nil)
	var out map[string]any
	if code := getJSON(t, d, "POST", "/api/founderos/pages/cron/run", []byte(`{"cronId":"cron-a"}`), &out); code != http.StatusServiceUnavailable {
		t.Fatalf("no runtime %d", code)
	}
	d.Agents = agents.New(agents.NewMemStore(), stubAgent{"crm-pulse"})
	if code := getJSON(t, d, "POST", "/api/founderos/pages/cron/run", []byte(`{}`), &out); code != 400 {
		t.Fatalf("no id %d", code)
	}
	if code := getJSON(t, d, "POST", "/api/founderos/pages/cron/run", []byte(`{"cronId":"nope"}`), &out); code != 404 {
		t.Fatalf("unknown %d", code)
	}
	if code := getJSON(t, d, "POST", "/api/founderos/pages/cron/run", []byte(`{"cronId":"cron-a"}`), &out); code != 200 || out["ok"] != true || out["summary"] != "ran crm-pulse" {
		t.Fatalf("run %d %v", code, out)
	}
	if code := getJSON(t, d, "POST", "/api/founderos/pages/cron/run", []byte(`{"cronId":"cron-b"}`), &out); code != http.StatusBadGateway || out["ok"] != false {
		t.Fatalf("ghost agent %d %v", code, out)
	}
	runs, _ := osdata.New(d.Pool).CronRunsByCron(context.Background(), "cron-b", 5)
	if len(runs) != 2 || runs[0].OK || !strings.Contains(runs[0].Summary, "unknown agent") {
		t.Fatalf("failed run not recorded: %+v", runs)
	}
}
