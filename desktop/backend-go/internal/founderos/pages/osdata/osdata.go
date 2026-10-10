// Package osdata is the repository layer the agent-side /os pages read and
// write through: FounderOS v1's lib/db.ts repos for departments, agents, agent
// tasks and crons, cron runs, agent runs, broadcasts, workflows and
// deliverable decisions, on the founderos_* Postgres tables (spec 3.2).
//
// Rows are scoped to the operator's workspaces by slug (the table map's workspace
// rules): roster and runtime tables live in founderos; workflows and
// decisions are placed per row. Times cross the boundary as RFC 3339 strings,
// the same ISO shape the TS repos returned.
package osdata

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// The operator's workspaces (cmd/founderos-bootstrap).
var Slugs = []string{"founderos", "vantage", "launchpad-cohort", "personal"}

// ErrNoWorkspace means founderos-bootstrap has not created the workspace a row
// belongs to, so there is nowhere honest to read or write it.
var ErrNoWorkspace = errors.New("founderos workspace missing: run founderos-bootstrap")

// ErrNotFound is a missing row on update/delete.
var ErrNotFound = errors.New("not found")

type Store struct{ pool *pgxpool.Pool }

func New(pool *pgxpool.Pool) *Store { return &Store{pool: pool} }

// Workspaces maps the operator's slugs to workspace ids (only those that exist).
func (s *Store) Workspaces(ctx context.Context) (map[string]string, error) {
	if s == nil || s.pool == nil {
		return nil, errors.New("no database")
	}
	rows, err := s.pool.Query(ctx, `SELECT slug, id::text FROM workspaces WHERE slug = ANY($1)`, Slugs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]string{}
	for rows.Next() {
		var slug, id string
		if err := rows.Scan(&slug, &id); err != nil {
			return nil, err
		}
		out[slug] = id
	}
	return out, rows.Err()
}

func (s *Store) ws(ctx context.Context, slug string) (string, error) {
	all, err := s.Workspaces(ctx)
	if err != nil {
		return "", err
	}
	id, ok := all[slug]
	if !ok {
		return "", fmt.Errorf("%w (%s)", ErrNoWorkspace, slug)
	}
	return id, nil
}

func (s *Store) founder(ctx context.Context) (string, error) { return s.ws(ctx, "founderos") }

func iso(t time.Time) string { return t.UTC().Format("2006-01-02T15:04:05.000Z") }

func isoPtr(t *time.Time) *string {
	if t == nil {
		return nil
	}
	v := iso(*t)
	return &v
}

// ParseTime reads the ISO strings the pages send back.
func ParseTime(s string) (time.Time, error) {
	for _, layout := range []string{time.RFC3339Nano, "2006-01-02T15:04:05.000Z"} {
		if t, err := time.Parse(layout, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("bad time %q", s)
}

// ---- roster -----------------------------------------------------------------

type Department struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Slug    string `json:"slug"`
	Tagline string `json:"tagline"`
	Color   string `json:"color"`
	Order   int    `json:"order"`
}

type Agent struct {
	ID           string   `json:"id"`
	DepartmentID string   `json:"departmentId"`
	Name         string   `json:"name"`
	Role         string   `json:"role"`
	Status       string   `json:"status"`
	Tier         string   `json:"tier"`
	Description  string   `json:"description"`
	Model        string   `json:"model"`
	Tools        []string `json:"tools"`
	ParentID     *string  `json:"parentId"`
	Instance     string   `json:"instance"`
}

// Departments in board order (departments.all).
func (s *Store) Departments(ctx context.Context) ([]Department, error) {
	ws, err := s.founder(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := s.pool.Query(ctx, `SELECT id, name, slug, tagline, color, ord FROM founderos_departments WHERE workspace_id = $1 ORDER BY ord, id`, ws)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(r pgx.CollectableRow) (Department, error) {
		var d Department
		err := r.Scan(&d.ID, &d.Name, &d.Slug, &d.Tagline, &d.Color, &d.Order)
		return d, err
	})
}

// Agents ordered by tier then name (agents.all).
func (s *Store) Agents(ctx context.Context) ([]Agent, error) {
	ws, err := s.founder(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := s.pool.Query(ctx, `
		SELECT id, department_id, name, role, status, tier, description, model, tools, parent_id, instance
		FROM founderos_agents WHERE workspace_id = $1 ORDER BY tier, name`, ws)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(r pgx.CollectableRow) (Agent, error) {
		var a Agent
		var tools []byte
		if err := r.Scan(&a.ID, &a.DepartmentID, &a.Name, &a.Role, &a.Status, &a.Tier, &a.Description, &a.Model, &tools, &a.ParentID, &a.Instance); err != nil {
			return a, err
		}
		a.Tools = []string{}
		if len(tools) > 0 {
			if err := json.Unmarshal(tools, &a.Tools); err != nil {
				return a, fmt.Errorf("agent %s tools: %w", a.ID, err)
			}
		}
		return a, nil
	})
}

// ---- agent work: tasks + crons ----------------------------------------------

type Task struct {
	ID        string `json:"id"`
	AgentID   string `json:"agentId"`
	Title     string `json:"title"`
	Status    string `json:"status"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
}

var TaskStatuses = []string{"open", "doing", "review", "done"}

func ValidTaskStatus(s string) bool {
	for _, v := range TaskStatuses {
		if v == s {
			return true
		}
	}
	return false
}

// Tasks newest first; agentID "" means every agent. Ties break the way prod's
// rowid DESC does for the seeds (task-seed-11 above task-seed-9): by the id's
// length, then the id.
func (s *Store) Tasks(ctx context.Context, agentID string) ([]Task, error) {
	ws, err := s.founder(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := s.pool.Query(ctx, `
		SELECT id, agent_id, title, status, created_at, updated_at FROM founderos_agent_tasks
		WHERE workspace_id = $1 AND ($2 = '' OR agent_id = $2)
		ORDER BY created_at DESC, length(id) DESC, id DESC`, ws, agentID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(r pgx.CollectableRow) (Task, error) {
		var t Task
		var c, u time.Time
		err := r.Scan(&t.ID, &t.AgentID, &t.Title, &t.Status, &c, &u)
		t.CreatedAt, t.UpdatedAt = iso(c), iso(u)
		return t, err
	})
}

func (s *Store) InsertTask(ctx context.Context, t Task) error {
	ws, err := s.founder(ctx)
	if err != nil {
		return err
	}
	c, err := ParseTime(t.CreatedAt)
	if err != nil {
		return err
	}
	u, err := ParseTime(t.UpdatedAt)
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx, `INSERT INTO founderos_agent_tasks (id, workspace_id, agent_id, title, status, created_at, updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7)`,
		t.ID, ws, t.AgentID, t.Title, t.Status, c, u)
	return err
}

func (s *Store) SetTaskStatus(ctx context.Context, id, status string, at time.Time) error {
	ws, err := s.founder(ctx)
	if err != nil {
		return err
	}
	tag, err := s.pool.Exec(ctx, `UPDATE founderos_agent_tasks SET status = $3, updated_at = $4 WHERE workspace_id = $1 AND id = $2`, ws, id, status, at)
	if err == nil && tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return err
}

func (s *Store) RemoveTask(ctx context.Context, id string) error {
	ws, err := s.founder(ctx)
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx, `DELETE FROM founderos_agent_tasks WHERE workspace_id = $1 AND id = $2`, ws, id)
	return err
}

type Cron struct {
	ID          string `json:"id"`
	AgentID     string `json:"agentId"`
	Schedule    string `json:"schedule"`
	Description string `json:"description"`
	Enabled     bool   `json:"enabled"`
	CreatedAt   string `json:"createdAt"`
}

// Crons newest first; agentID "" means every agent.
func (s *Store) Crons(ctx context.Context, agentID string) ([]Cron, error) {
	ws, err := s.founder(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := s.pool.Query(ctx, `
		SELECT id, agent_id, schedule, description, enabled, created_at FROM founderos_agent_crons
		WHERE workspace_id = $1 AND ($2 = '' OR agent_id = $2)
		ORDER BY created_at DESC, length(id) DESC, id DESC`, ws, agentID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(r pgx.CollectableRow) (Cron, error) {
		var c Cron
		var at time.Time
		err := r.Scan(&c.ID, &c.AgentID, &c.Schedule, &c.Description, &c.Enabled, &at)
		c.CreatedAt = iso(at)
		return c, err
	})
}

func (s *Store) InsertCron(ctx context.Context, c Cron) error {
	ws, err := s.founder(ctx)
	if err != nil {
		return err
	}
	at, err := ParseTime(c.CreatedAt)
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx, `INSERT INTO founderos_agent_crons (id, workspace_id, agent_id, schedule, description, enabled, created_at) VALUES ($1,$2,$3,$4,$5,$6,$7)`,
		c.ID, ws, c.AgentID, c.Schedule, c.Description, c.Enabled, at)
	return err
}

func (s *Store) SetCronEnabled(ctx context.Context, id string, enabled bool) error {
	ws, err := s.founder(ctx)
	if err != nil {
		return err
	}
	tag, err := s.pool.Exec(ctx, `UPDATE founderos_agent_crons SET enabled = $3 WHERE workspace_id = $1 AND id = $2`, ws, id, enabled)
	if err == nil && tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return err
}

func (s *Store) RemoveCron(ctx context.Context, id string) error {
	ws, err := s.founder(ctx)
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx, `DELETE FROM founderos_agent_crons WHERE workspace_id = $1 AND id = $2`, ws, id)
	return err
}

// ---- cron runs --------------------------------------------------------------

type CronRun struct {
	ID         string  `json:"id"`
	CronID     string  `json:"cronId"`
	AgentID    string  `json:"agentId"`
	StartedAt  string  `json:"startedAt"`
	FinishedAt *string `json:"finishedAt"`
	OK         bool    `json:"ok"`
	Summary    string  `json:"summary"`
}

// CronStat is statsByCron's row.
type CronStat struct {
	Runs      int     `json:"runs"`
	OK        int     `json:"ok"`
	LastRunAt *string `json:"lastRunAt"`
	LastOK    *bool   `json:"lastOk"`
}

func (s *Store) cronRuns(ctx context.Context, where string, args ...any) ([]CronRun, error) {
	ws, err := s.founder(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := s.pool.Query(ctx, `SELECT id, cron_id, agent_id, started_at, finished_at, ok, summary FROM founderos_cron_runs WHERE workspace_id = $1 `+where, append([]any{ws}, args...)...)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(r pgx.CollectableRow) (CronRun, error) {
		var c CronRun
		var st time.Time
		var fin *time.Time
		err := r.Scan(&c.ID, &c.CronID, &c.AgentID, &st, &fin, &c.OK, &c.Summary)
		c.StartedAt, c.FinishedAt = iso(st), isoPtr(fin)
		return c, err
	})
}

// CronRunsByCron is one cron's latest runs, newest first.
func (s *Store) CronRunsByCron(ctx context.Context, cronID string, limit int) ([]CronRun, error) {
	return s.cronRuns(ctx, `AND cron_id = $2 ORDER BY started_at DESC, id DESC LIMIT $3`, cronID, limit)
}

// CronRunsSince is every run started at or after since, newest first.
func (s *Store) CronRunsSince(ctx context.Context, since time.Time) ([]CronRun, error) {
	return s.cronRuns(ctx, `AND started_at >= $2 ORDER BY started_at DESC, id DESC`, since)
}

// CronStats is run counts + last outcome per cron (cronRuns.statsByCron).
func (s *Store) CronStats(ctx context.Context) (map[string]CronStat, error) {
	ws, err := s.founder(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := s.pool.Query(ctx, `
		SELECT cron_id, count(*), count(*) FILTER (WHERE ok), max(started_at),
		       (array_agg(ok ORDER BY started_at DESC, id DESC))[1]
		FROM founderos_cron_runs WHERE workspace_id = $1 GROUP BY cron_id`, ws)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]CronStat{}
	for rows.Next() {
		var id string
		var st CronStat
		var last time.Time
		var lastOK bool
		if err := rows.Scan(&id, &st.Runs, &st.OK, &last, &lastOK); err != nil {
			return nil, err
		}
		st.LastRunAt, st.LastOK = isoPtr(&last), &lastOK
		out[id] = st
	}
	return out, rows.Err()
}

// ---- agent runs + broadcasts ------------------------------------------------

type AgentRun struct {
	ID         string   `json:"id"`
	AgentID    string   `json:"agentId"`
	StartedAt  string   `json:"startedAt"`
	FinishedAt string   `json:"finishedAt"`
	OK         bool     `json:"ok"`
	Summary    string   `json:"summary"`
	Model      *string  `json:"model"`
	TokensIn   *int     `json:"tokensIn"`
	TokensOut  *int     `json:"tokensOut"`
	CostUSD    *float64 `json:"costUsd"`
}

// AgentRuns newest first; agentID "" means every agent.
func (s *Store) AgentRuns(ctx context.Context, agentID string, limit int) ([]AgentRun, error) {
	ws, err := s.founder(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := s.pool.Query(ctx, `
		SELECT id, agent_id, started_at, finished_at, ok, summary, model, tokens_in, tokens_out, cost_usd
		FROM founderos_agent_runs WHERE workspace_id = $1 AND ($2 = '' OR agent_id = $2)
		ORDER BY started_at DESC, id DESC LIMIT $3`, ws, agentID, limit)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(r pgx.CollectableRow) (AgentRun, error) {
		var a AgentRun
		var st, fin time.Time
		err := r.Scan(&a.ID, &a.AgentID, &st, &fin, &a.OK, &a.Summary, &a.Model, &a.TokensIn, &a.TokensOut, &a.CostUSD)
		a.StartedAt, a.FinishedAt = iso(st), iso(fin)
		return a, err
	})
}

type BroadcastReply struct {
	ID         string `json:"id"`
	AgentID    string `json:"agentId"`
	OK         bool   `json:"ok"`
	Reply      string `json:"reply"`
	FinishedAt string `json:"finishedAt"`
}

type Broadcast struct {
	ID        string           `json:"id"`
	Message   string           `json:"message"`
	CreatedAt string           `json:"createdAt"`
	Replies   []BroadcastReply `json:"replies"`
}

// Broadcasts newest first, each with its replies ordered by agent id.
func (s *Store) Broadcasts(ctx context.Context, limit int) ([]Broadcast, error) {
	ws, err := s.founder(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := s.pool.Query(ctx, `SELECT id, message, created_at FROM founderos_broadcasts WHERE workspace_id = $1 ORDER BY created_at DESC, id DESC LIMIT $2`, ws, limit)
	if err != nil {
		return nil, err
	}
	list, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (Broadcast, error) {
		var b Broadcast
		var at time.Time
		err := r.Scan(&b.ID, &b.Message, &at)
		b.CreatedAt, b.Replies = iso(at), []BroadcastReply{}
		return b, err
	})
	if err != nil {
		return nil, err
	}
	for i := range list {
		rr, err := s.pool.Query(ctx, `SELECT id, agent_id, ok, reply, finished_at FROM founderos_broadcast_replies WHERE workspace_id = $1 AND broadcast_id = $2 ORDER BY agent_id`, ws, list[i].ID)
		if err != nil {
			return nil, err
		}
		replies, err := pgx.CollectRows(rr, func(r pgx.CollectableRow) (BroadcastReply, error) {
			var x BroadcastReply
			var at time.Time
			err := r.Scan(&x.ID, &x.AgentID, &x.OK, &x.Reply, &at)
			x.FinishedAt = iso(at)
			return x, err
		})
		if err != nil {
			return nil, err
		}
		list[i].Replies = replies
	}
	return list, nil
}

// AgentMessage is a chat turn (founderos_agent_messages; dormant in FounderOS v1).
type AgentMessage struct {
	ID        string          `json:"id"`
	AgentID   string          `json:"agentId"`
	Role      string          `json:"role"`
	Content   string          `json:"content"`
	ToolCalls []ToolCallBrief `json:"toolCalls"`
	CreatedAt string          `json:"createdAt"`
}

type ToolCallBrief struct {
	Name string `json:"name"`
}

// AgentMessages newest first.
func (s *Store) AgentMessages(ctx context.Context, limit int) ([]AgentMessage, error) {
	ws, err := s.founder(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := s.pool.Query(ctx, `SELECT id, agent_id, role, content, tool_calls, created_at FROM founderos_agent_messages WHERE workspace_id = $1 ORDER BY created_at DESC, id DESC LIMIT $2`, ws, limit)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(r pgx.CollectableRow) (AgentMessage, error) {
		var m AgentMessage
		var raw []byte
		var at time.Time
		if err := r.Scan(&m.ID, &m.AgentID, &m.Role, &m.Content, &raw, &at); err != nil {
			return m, err
		}
		m.CreatedAt, m.ToolCalls = iso(at), []ToolCallBrief{}
		if len(raw) > 0 {
			_ = json.Unmarshal(raw, &m.ToolCalls)
		}
		return m, nil
	})
}

// ---- workflows --------------------------------------------------------------

type Automation struct {
	Title        string  `json:"title"`
	State        string  `json:"state"`
	RecoveredUSD float64 `json:"recoveredUsd"`
}

type Branch struct {
	From      string `json:"from"`
	Condition string `json:"condition"`
}

type WorkflowStep struct {
	ID           string      `json:"id"`
	Title        string      `json:"title"`
	Detail       string      `json:"detail"`
	OwnerKind    string      `json:"ownerKind"`
	Owner        string      `json:"owner"`
	HoursPerWeek float64     `json:"hoursPerWeek"`
	Tools        []string    `json:"tools"`
	EdgeLabel    *string     `json:"edgeLabel"`
	LeakUSD      *float64    `json:"leakUsd"`
	Automation   *Automation `json:"automation"`
	Branch       *Branch     `json:"branch"`
}

type Workflow struct {
	ID         string         `json:"id"`
	Name       string         `json:"name"`
	Subtitle   string         `json:"subtitle"`
	RevenueUSD int            `json:"revenueUsd"`
	Order      int            `json:"order"`
	Steps      []WorkflowStep `json:"steps"`
}

// WorkflowWorkspace is the per-row rule from the table map.
func WorkflowWorkspace(id string) string {
	switch {
	case strings.HasPrefix(id, "wf-vantage"):
		return "vantage"
	case strings.HasPrefix(id, "wf-lc"):
		return "launchpad-cohort"
	}
	return "founderos"
}

func (s *Store) wsIDs(ctx context.Context) ([]string, error) {
	all, err := s.Workspaces(ctx)
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(all))
	for _, id := range all {
		ids = append(ids, id)
	}
	return ids, nil
}

func scanWorkflow(r pgx.CollectableRow) (Workflow, error) {
	var w Workflow
	var raw []byte
	if err := r.Scan(&w.ID, &w.Name, &w.Subtitle, &w.RevenueUSD, &w.Order, &raw); err != nil {
		return w, err
	}
	w.Steps = []WorkflowStep{}
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &w.Steps); err != nil {
			return w, fmt.Errorf("workflow %s steps: %w", w.ID, err)
		}
	}
	for i := range w.Steps {
		if w.Steps[i].Tools == nil {
			w.Steps[i].Tools = []string{}
		}
	}
	return w, nil
}

// Workflows across the operator's workspaces, by order then name.
func (s *Store) Workflows(ctx context.Context) ([]Workflow, error) {
	ids, err := s.wsIDs(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := s.pool.Query(ctx, `SELECT id, name, subtitle, revenue_usd, ord, steps FROM founderos_workflows WHERE workspace_id::text = ANY($1) ORDER BY ord, name`, ids)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, scanWorkflow)
}

// Workflow by id; nil when it does not exist.
func (s *Store) Workflow(ctx context.Context, id string) (*Workflow, error) {
	ids, err := s.wsIDs(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := s.pool.Query(ctx, `SELECT id, name, subtitle, revenue_usd, ord, steps FROM founderos_workflows WHERE workspace_id::text = ANY($1) AND id = $2`, ids, id)
	if err != nil {
		return nil, err
	}
	list, err := pgx.CollectRows(rows, scanWorkflow)
	if err != nil || len(list) == 0 {
		return nil, err
	}
	return &list[0], nil
}

// UpsertWorkflow inserts or replaces a workflow (workflows.insert is an upsert).
func (s *Store) UpsertWorkflow(ctx context.Context, w Workflow) error {
	ws, err := s.ws(ctx, WorkflowWorkspace(w.ID))
	if err != nil {
		return err
	}
	steps, err := json.Marshal(w.Steps)
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx, `
		INSERT INTO founderos_workflows (id, workspace_id, name, subtitle, revenue_usd, ord, steps) VALUES ($1,$2,$3,$4,$5,$6,$7)
		ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name, subtitle = EXCLUDED.subtitle, revenue_usd = EXCLUDED.revenue_usd, ord = EXCLUDED.ord, steps = EXCLUDED.steps`,
		w.ID, ws, w.Name, w.Subtitle, w.RevenueUSD, w.Order, steps)
	return err
}

func (s *Store) RemoveWorkflow(ctx context.Context, id string) error {
	ids, err := s.wsIDs(ctx)
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx, `DELETE FROM founderos_workflows WHERE workspace_id::text = ANY($1) AND id = $2`, ids, id)
	return err
}

// ---- deliverable decisions --------------------------------------------------

type Decision struct {
	ID              string `json:"id"`
	Decision        string `json:"decision"`
	DecidedAt       string `json:"decidedAt"`
	DecidedRevision string `json:"decidedRevision"`
	Note            string `json:"note"`
}

func ValidDecision(k string) bool { return k == "approved" || k == "dismissed" }

// decisionWorkspace: proposal:<pid> → that proposal's workspace, else founderos.
func (s *Store) decisionWorkspace(ctx context.Context, id string) (string, error) {
	if pid, ok := strings.CutPrefix(id, "proposal:"); ok {
		var ws string
		err := s.pool.QueryRow(ctx, `SELECT workspace_id::text FROM founderos_proposals WHERE id = $1`, pid).Scan(&ws)
		if err == nil {
			return ws, nil
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return "", err
		}
	}
	return s.founder(ctx)
}

// Decisions newest first across the operator's workspaces.
func (s *Store) Decisions(ctx context.Context) ([]Decision, error) {
	ids, err := s.wsIDs(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := s.pool.Query(ctx, `SELECT id, decision, decided_at, decided_revision, note FROM founderos_deliverable_decisions WHERE workspace_id::text = ANY($1) ORDER BY decided_at DESC, id`, ids)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(r pgx.CollectableRow) (Decision, error) {
		var d Decision
		var at time.Time
		err := r.Scan(&d.ID, &d.Decision, &at, &d.DecidedRevision, &d.Note)
		d.DecidedAt = iso(at)
		return d, err
	})
}

// SetDecision records (or replaces) the operator's call on one item.
func (s *Store) SetDecision(ctx context.Context, d Decision) error {
	ws, err := s.decisionWorkspace(ctx, d.ID)
	if err != nil {
		return err
	}
	at, err := ParseTime(d.DecidedAt)
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx, `
		INSERT INTO founderos_deliverable_decisions (id, workspace_id, decision, decided_at, decided_revision, note) VALUES ($1,$2,$3,$4,$5,$6)
		ON CONFLICT (id) DO UPDATE SET workspace_id = EXCLUDED.workspace_id, decision = EXCLUDED.decision, decided_at = EXCLUDED.decided_at,
		  decided_revision = EXCLUDED.decided_revision, note = EXCLUDED.note`,
		d.ID, ws, d.Decision, at, d.DecidedRevision, d.Note)
	return err
}

// ClearDecision puts an item back in the open queue.
func (s *Store) ClearDecision(ctx context.Context, id string) error {
	ids, err := s.wsIDs(ctx)
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx, `DELETE FROM founderos_deliverable_decisions WHERE workspace_id::text = ANY($1) AND id = $2`, ids, id)
	return err
}

// Proposal is the slice of founderos_proposals the deliverables grouping needs.
type Proposal struct {
	ID        string   `json:"id"`
	Client    string   `json:"client"`
	Brand     string   `json:"brand"`
	URL       string   `json:"url"`
	Status    string   `json:"status"`
	AmountUSD *float64 `json:"amountUsd"`
	Notes     string   `json:"notes"`
	CreatedAt string   `json:"createdAt"`
}

func (s *Store) Proposals(ctx context.Context) ([]Proposal, error) {
	ids, err := s.wsIDs(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := s.pool.Query(ctx, `SELECT id, client, brand, url, status, amount_usd, notes, created_at FROM founderos_proposals WHERE workspace_id::text = ANY($1) ORDER BY created_at DESC, id`, ids)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(r pgx.CollectableRow) (Proposal, error) {
		var p Proposal
		var at time.Time
		err := r.Scan(&p.ID, &p.Client, &p.Brand, &p.URL, &p.Status, &p.AmountUSD, &p.Notes, &at)
		p.CreatedAt = iso(at)
		return p, err
	})
}
