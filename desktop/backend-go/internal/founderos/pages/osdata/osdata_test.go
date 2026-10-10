package osdata_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/rhl/businessos-backend/internal/founderos/pages/osdata"
	"github.com/rhl/businessos-backend/internal/founderos/pages/osdata/osdatatest"
)

func TestRosterReadsInFounderosOrder(t *testing.T) {
	pool, ws := osdatatest.ThrowawayDB(t)
	now := time.Now().UTC().Truncate(time.Second)
	osdatatest.Seed(t, pool, ws, now)
	st := osdata.New(pool)
	ctx := context.Background()

	deps, err := st.Departments(ctx)
	if err != nil || len(deps) != 2 || deps[0].ID != "dept-sales" || deps[1].Order != 3 {
		t.Fatalf("departments by ord: %+v %v", deps, err)
	}
	agents, err := st.Agents(ctx)
	if err != nil || len(agents) != 4 {
		t.Fatalf("agents: %+v %v", agents, err)
	}
	// tier then name: lead(Conductor, Sales Agent), specialist, worker
	want := []string{"conductor", "sales-agent", "stack-monitor", "crm-pulse"}
	for i, id := range want {
		if agents[i].ID != id {
			t.Fatalf("order[%d] = %s, want %s", i, agents[i].ID, id)
		}
	}
	if agents[3].ParentID == nil || *agents[3].ParentID != "sales-agent" || agents[0].Tools[1] != "gbrain" || agents[2].Tools == nil {
		t.Fatalf("agent fields: %+v", agents)
	}
}

func TestTasksAndCronsRoundTrip(t *testing.T) {
	pool, ws := osdatatest.ThrowawayDB(t)
	now := time.Now().UTC().Truncate(time.Second)
	osdatatest.Seed(t, pool, ws, now)
	st := osdata.New(pool)
	ctx := context.Background()

	tasks, err := st.Tasks(ctx, "")
	if err != nil || len(tasks) != 2 || tasks[0].ID != "t1" {
		t.Fatalf("tasks newest first: %+v %v", tasks, err)
	}
	stamp := now.Format(time.RFC3339)
	if err := st.InsertTask(ctx, osdata.Task{ID: "t3", AgentID: "crm-pulse", Title: "x", Status: "open", CreatedAt: stamp, UpdatedAt: stamp}); err != nil {
		t.Fatal(err)
	}
	if err := st.SetTaskStatus(ctx, "t3", "review", now); err != nil {
		t.Fatal(err)
	}
	if err := st.SetTaskStatus(ctx, "nope", "review", now); !errors.Is(err, osdata.ErrNotFound) {
		t.Fatalf("missing task: %v", err)
	}
	mine, _ := st.Tasks(ctx, "crm-pulse")
	if len(mine) != 2 {
		t.Fatalf("by agent: %+v", mine)
	}
	_ = st.RemoveTask(ctx, "t3")
	if all, _ := st.Tasks(ctx, ""); len(all) != 2 {
		t.Fatalf("remove: %d", len(all))
	}

	crons, err := st.Crons(ctx, "")
	if err != nil || len(crons) != 2 || crons[0].ID != "cron-b" {
		t.Fatalf("crons newest first: %+v %v", crons, err)
	}
	if err := st.SetCronEnabled(ctx, "cron-b", true); err != nil {
		t.Fatal(err)
	}
	if err := st.InsertCron(ctx, osdata.Cron{ID: "c3", AgentID: "a", Schedule: "0 9 * * 1-5", Description: "d", Enabled: true, CreatedAt: stamp}); err != nil {
		t.Fatal(err)
	}
	_ = st.RemoveCron(ctx, "c3")
	crons, _ = st.Crons(ctx, "")
	if len(crons) != 2 || !crons[0].Enabled {
		t.Fatalf("cron writes: %+v", crons)
	}
}

func TestCronRunStats(t *testing.T) {
	pool, ws := osdatatest.ThrowawayDB(t)
	now := time.Now().UTC().Truncate(time.Second)
	osdatatest.Seed(t, pool, ws, now)
	st := osdata.New(pool)
	ctx := context.Background()

	stats, err := st.CronStats(ctx)
	if err != nil {
		t.Fatal(err)
	}
	a := stats["cron-a"]
	if a.Runs != 2 || a.OK != 1 || a.LastOK == nil || *a.LastOK || a.LastRunAt == nil {
		t.Fatalf("cron-a stats: %+v", a)
	}
	runs, _ := st.CronRunsByCron(ctx, "cron-a", 12)
	if len(runs) != 2 || runs[0].ID != "cr2" {
		t.Fatalf("by cron newest first: %+v", runs)
	}
	recent, _ := st.CronRunsSince(ctx, now.Add(-14*24*time.Hour))
	if len(recent) != 2 {
		t.Fatalf("since: %+v", recent)
	}
	if recent[len(recent)-1].FinishedAt == nil {
		t.Fatalf("finished time lost: %+v", recent)
	}
}

func TestWorkflowsArePlacedPerRow(t *testing.T) {
	pool, ws := osdatatest.ThrowawayDB(t)
	now := time.Now().UTC()
	osdatatest.Seed(t, pool, ws, now)
	st := osdata.New(pool)
	ctx := context.Background()

	list, err := st.Workflows(ctx)
	if err != nil || len(list) != 1 || len(list[0].Steps) != 2 || list[0].Steps[0].Automation.State != "live" {
		t.Fatalf("workflows: %+v %v", list, err)
	}
	w := osdata.Workflow{ID: "wf-lc-onboard", Name: "Onboard", Order: 1, Steps: []osdata.WorkflowStep{{ID: "x", Title: "t", OwnerKind: "human", Owner: "B", Tools: []string{}}}}
	if err := st.UpsertWorkflow(ctx, w); err != nil {
		t.Fatal(err)
	}
	var placed string
	_ = pool.QueryRow(ctx, `SELECT workspace_id::text FROM founderos_workflows WHERE id='wf-lc-onboard'`).Scan(&placed)
	if placed != ws["launchpad-cohort"] {
		t.Fatalf("wf-lc row landed in %s", placed)
	}
	w.Name = "Onboard v2"
	_ = st.UpsertWorkflow(ctx, w)
	got, _ := st.Workflow(ctx, "wf-lc-onboard")
	if got == nil || got.Name != "Onboard v2" {
		t.Fatalf("upsert: %+v", got)
	}
	_ = st.RemoveWorkflow(ctx, "wf-lc-onboard")
	if got, _ := st.Workflow(ctx, "wf-lc-onboard"); got != nil {
		t.Fatal("remove")
	}
}

func TestDecisionsSetReplaceClear(t *testing.T) {
	pool, _ := osdatatest.ThrowawayDB(t)
	st := osdata.New(pool)
	ctx := context.Background()
	at := time.Now().UTC().Format(time.RFC3339)
	if err := st.SetDecision(ctx, osdata.Decision{ID: "board:i1", Decision: "approved", DecidedAt: at, DecidedRevision: "r1"}); err != nil {
		t.Fatal(err)
	}
	_ = st.SetDecision(ctx, osdata.Decision{ID: "board:i1", Decision: "dismissed", DecidedAt: at, DecidedRevision: "r2"})
	list, err := st.Decisions(ctx)
	if err != nil || len(list) != 1 || list[0].Decision != "dismissed" || list[0].DecidedRevision != "r2" {
		t.Fatalf("decisions: %+v %v", list, err)
	}
	_ = st.ClearDecision(ctx, "board:i1")
	if list, _ := st.Decisions(ctx); len(list) != 0 {
		t.Fatalf("clear: %+v", list)
	}
}

func TestMissingWorkspaceIsAnError(t *testing.T) {
	pool, _ := osdatatest.ThrowawayDB(t)
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `DELETE FROM workspaces WHERE slug = 'founderos'`); err != nil {
		t.Fatal(err)
	}
	if _, err := osdata.New(pool).Agents(ctx); !errors.Is(err, osdata.ErrNoWorkspace) {
		t.Fatalf("want ErrNoWorkspace, got %v", err)
	}
}

// Prod reads agent_tasks ORDER BY created_at DESC, rowid DESC: the seeded
// tasks share one created_at, so insertion order decides, and the seeds were
// inserted task-seed-1 … task-seed-11. Postgres has no rowid, and a plain
// id DESC puts task-seed-9 above task-seed-11. Ties break on the id's length
// first, which is the same order for the seeds and a no-op for uuids.
func TestTasksTieBreakLikeProdRowid(t *testing.T) {
	pool, ws := osdatatest.ThrowawayDB(t)
	now := time.Now().UTC().Truncate(time.Second)
	osdatatest.Seed(t, pool, ws, now)
	st := osdata.New(pool)
	ctx := context.Background()
	stamp := now.Add(time.Hour).Format(time.RFC3339)
	for _, id := range []string{"task-seed-8", "task-seed-9", "task-seed-10", "task-seed-11"} {
		if err := st.InsertTask(ctx, osdata.Task{ID: id, AgentID: "crm-pulse", Title: id, Status: "done", CreatedAt: stamp, UpdatedAt: stamp}); err != nil {
			t.Fatal(err)
		}
	}
	tasks, err := st.Tasks(ctx, "")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"task-seed-11", "task-seed-10", "task-seed-9", "task-seed-8"}
	for i, id := range want {
		if tasks[i].ID != id {
			t.Fatalf("order[%d] = %s, want %s (%+v)", i, tasks[i].ID, id, tasks[:4])
		}
	}
}
