package osdatatest

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Seed loads a small the operator roster into a throwaway DB: two departments,
// four agents (a Conductor, a lead with a worker child, a planned
// specialist), two crons with runs, two tasks and a vantage workflow.
// Fixed times hang off now so window maths is deterministic.
func Seed(t testing.TB, pool *pgxpool.Pool, ws map[string]string, now time.Time) {
	t.Helper()
	ctx := context.Background()
	f := ws["founderos"]
	exec := func(sql string, args ...any) {
		t.Helper()
		if _, err := pool.Exec(ctx, sql, args...); err != nil {
			t.Fatalf("seed: %v\n%s", err, sql)
		}
	}
	exec(`INSERT INTO founderos_departments (id, workspace_id, name, slug, tagline, color, ord) VALUES
		('dept-tech',$1,'TECH','tech','builds','#8b7cf6',3),
		('dept-sales',$1,'Sales','sales','sells','#ffd166',1)`, f)
	exec(`INSERT INTO founderos_agents (id, workspace_id, department_id, name, role, status, tier, description, model, tools, parent_id, instance) VALUES
		('conductor',$1,'dept-tech','Conductor','super agent','active','lead','routes work','opus','["slack","gbrain"]',NULL,'claude-code'),
		('sales-agent',$1,'dept-sales','Sales Agent','closer','active','lead','runs the pipeline','sonnet','["typeform"]',NULL,'openclaw'),
		('crm-pulse',$1,'dept-sales','CRM Pulse','watches the CRM','idle','worker','pulse','haiku','["attio"]','sales-agent','builtin'),
		('stack-monitor',$1,'dept-tech','Stack Monitor','uptime','planned','specialist','ports','', '[]',NULL,'builtin')`, f)
	exec(`INSERT INTO founderos_agent_crons (id, workspace_id, agent_id, schedule, description, enabled, created_at) VALUES
		('cron-a',$1,'crm-pulse','0 9 * * *','Morning CRM pulse',true,$2),
		('cron-b',$1,'ghost-agent','*/30 * * * *','Ghost poll',false,$3)`, f, now.Add(-72*time.Hour), now.Add(-48*time.Hour))
	exec(`INSERT INTO founderos_cron_runs (id, workspace_id, cron_id, agent_id, started_at, finished_at, ok, summary) VALUES
		('cr1',$1,'cron-a','crm-pulse',$2,$2,true,'ok one'),
		('cr2',$1,'cron-a','crm-pulse',$3,$3,false,'boom'),
		('cr3',$1,'cron-b','ghost-agent',$4,NULL,false,'unknown agent')`,
		f, now.Add(-50*time.Hour), now.Add(-2*time.Hour), now.Add(-30*24*time.Hour))
	exec(`INSERT INTO founderos_agent_tasks (id, workspace_id, agent_id, title, status, created_at, updated_at) VALUES
		('t1',$1,'crm-pulse','Clean the CRM','open',$2,$2),
		('t2',$1,'sales-agent','Follow up leads','done',$3,$4)`, f, now.Add(-3*time.Hour), now.Add(-5*24*time.Hour), now.Add(-24*time.Hour))
	exec(`INSERT INTO founderos_agent_runs (id, workspace_id, agent_id, started_at, finished_at, ok, summary) VALUES
		('r1',$1,'crm-pulse',$2,$2,true,'pulsed'),
		('r2',$1,'sales-agent',$3,$3,false,'failed')`, f, now.Add(-time.Hour), now.Add(-30*time.Minute))
	exec(`INSERT INTO founderos_workflows (id, workspace_id, name, subtitle, revenue_usd, ord, steps) VALUES
		('wf-vantage-sales',$1,'Vantage sales machine','lead to close',5000,0,
		 '[{"id":"s1","title":"Lead in","detail":"Typeform","ownerKind":"agent","owner":"Sales Agent","hoursPerWeek":2,"tools":["typeform"],"edgeLabel":null,"leakUsd":null,"automation":{"title":"auto-tag","state":"live","recoveredUsd":300},"branch":null},
		   {"id":"s2","title":"Call","detail":"the operator calls","ownerKind":"human","owner":"the operator","hoursPerWeek":5,"tools":["zoom","typeform"],"edgeLabel":null,"leakUsd":1200,"automation":null,"branch":null}]')`, ws["vantage"])
}
