package api

import (
	"encoding/json"
	"io"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/rhl/businessos-backend/internal/founderos/pages/osdata"
	"github.com/rhl/businessos-backend/internal/founderos/pages/shared"
	"github.com/rhl/businessos-backend/internal/founderos/pages/workflows"
)

// /workflows (spec 6.17): scheduled tasks (crons with their real run
// history) and the process map (workflows with a builder and a drafting
// assistant). Cron add/pause/delete go through /pages/agents/work (the
// /tasks page's route); "run now" is /pages/cron/run here.
//
// Route map: /api/workflows, /api/workflows/[id], /api/workflows/draft,
// /api/cron/run.

func init() { RegisterPage(registerWorkflowsPage) }

const (
	wfWindowDays   = 14
	wfRunsPerOwner = 4
	wfCronHistory  = 12
)

var (
	draftRunnerMu sync.Mutex
	draftRunner   workflows.Runner = workflows.ClaudeCLI{}
)

// SetWorkflowDraftRunner swaps the drafting CLI (tests inject a fake so the
// real `claude` binary is never spawned) and returns a restore func.
func SetWorkflowDraftRunner(r workflows.Runner) func() {
	draftRunnerMu.Lock()
	prev := draftRunner
	draftRunner = r
	draftRunnerMu.Unlock()
	return func() { SetWorkflowDraftRunner(prev) }
}

func currentDraftRunner() workflows.Runner {
	draftRunnerMu.Lock()
	defer draftRunnerMu.Unlock()
	return draftRunner
}

func registerWorkflowsPage(s *gin.RouterGroup, d *Deps) {
	s.GET("/pages/workflows", func(c *gin.Context) {
		list, err := storeFor(d).Workflows(c.Request.Context())
		if err != nil {
			dataErr(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"workflows": list})
	})
	s.GET("/pages/workflows/view", func(c *gin.Context) { workflowsView(c, d) })
	s.POST("/pages/workflows", func(c *gin.Context) { createWorkflow(c, d) })
	s.PATCH("/pages/workflows/:id", func(c *gin.Context) { updateWorkflow(c, d) })
	s.DELETE("/pages/workflows/:id", func(c *gin.Context) {
		ctx, st := c.Request.Context(), storeFor(d)
		w, err := st.Workflow(ctx, c.Param("id"))
		if err != nil {
			dataErr(c, err)
			return
		}
		if w == nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "unknown workflow: " + c.Param("id")})
			return
		}
		if err := st.RemoveWorkflow(ctx, w.ID); err != nil {
			dataErr(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})
	s.POST("/pages/workflows/draft", func(c *gin.Context) {
		var body struct {
			Prompt string `json:"prompt"`
		}
		if err := c.ShouldBindJSON(&body); err != nil || strings.TrimSpace(body.Prompt) == "" || len(body.Prompt) > 4000 {
			c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": "a prompt is required"})
			return
		}
		c.JSON(http.StatusOK, workflows.Draft(c.Request.Context(), currentDraftRunner(), body.Prompt))
	})
	s.POST("/pages/cron/run", func(c *gin.Context) { runCronNow(c, d) })
}

func readWorkflowInput(c *gin.Context) (workflows.Input, bool) {
	raw, err := io.ReadAll(http.MaxBytesReader(c.Writer, c.Request.Body, 1<<20))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unreadable body"})
		return workflows.Input{}, false
	}
	in, err := workflows.ParseInput(raw)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return workflows.Input{}, false
	}
	return in, true
}

func createWorkflow(c *gin.Context, d *Deps) {
	in, ok := readWorkflowInput(c)
	if !ok {
		return
	}
	ctx, st := c.Request.Context(), storeFor(d)
	existing, err := st.Workflows(ctx)
	if err != nil {
		dataErr(c, err)
		return
	}
	order := -1
	for _, w := range existing {
		order = max(order, w.Order)
	}
	id := workflows.NewID(in.Name)
	w := osdata.Workflow{ID: id, Name: in.Name, Subtitle: in.Subtitle, Order: order + 1, Steps: workflows.BuildSteps(id, in.Steps)}
	if err := st.UpsertWorkflow(ctx, w); err != nil {
		dataErr(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"workflow": w})
}

func updateWorkflow(c *gin.Context, d *Deps) {
	ctx, st := c.Request.Context(), storeFor(d)
	existing, err := st.Workflow(ctx, c.Param("id"))
	if err != nil {
		dataErr(c, err)
		return
	}
	if existing == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "unknown workflow: " + c.Param("id")})
		return
	}
	in, ok := readWorkflowInput(c)
	if !ok {
		return
	}
	w := osdata.Workflow{ID: existing.ID, Name: in.Name, Subtitle: in.Subtitle, RevenueUSD: existing.RevenueUSD, Order: existing.Order,
		Steps: workflows.BuildSteps(existing.ID, in.Steps)}
	if err := st.UpsertWorkflow(ctx, w); err != nil {
		dataErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"workflow": w})
}

type simpleAgent struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// workflowsView is the page model: both halves plus the slab numbers, from
// the same rows the panels render.
func workflowsView(c *gin.Context, d *Deps) {
	ctx, st := c.Request.Context(), storeFor(d)
	wfs, err := st.Workflows(ctx)
	if err != nil {
		dataErr(c, err)
		return
	}
	roster, err := st.Agents(ctx)
	if err != nil {
		dataErr(c, err)
		return
	}
	crons, err := st.Crons(ctx, "")
	if err != nil {
		dataErr(c, err)
		return
	}
	stats, err := st.CronStats(ctx)
	if err != nil {
		dataErr(c, err)
		return
	}
	recent := map[string][]osdata.CronRun{}
	for _, cr := range crons {
		runs, err := st.CronRunsByCron(ctx, cr.ID, wfCronHistory)
		if err != nil {
			dataErr(c, err)
			return
		}
		recent[cr.ID] = runs
	}
	now := time.Now()
	window, err := st.CronRunsSince(ctx, now.Add(-time.Duration(wfWindowDays+1)*24*time.Hour))
	if err != nil {
		dataErr(c, err)
		return
	}
	names := map[string]string{}
	simple := make([]simpleAgent, len(roster))
	presence := map[string]string{}
	byName := map[string]osdata.Agent{}
	for i, a := range roster {
		names[a.ID] = a.Name
		simple[i] = simpleAgent{a.ID, a.Name}
		byName[a.Name] = a
		if a.Status == "active" {
			presence[a.Name] = "active"
		} else {
			presence[a.Name] = "inactive"
		}
	}
	jobs := shared.ScheduledJobRows(crons, stats, names, recent, now)

	// An owner's runs only when the owner resolves to a real agent; a human
	// or an unknown name has none, never a borrowed history.
	runsByOwner := map[string][]osdata.AgentRun{}
	tools := map[string]bool{}
	for _, w := range wfs {
		for _, s := range w.Steps {
			for _, t := range s.Tools {
				tools[t] = true
			}
			if _, done := runsByOwner[s.Owner]; done {
				continue
			}
			runsByOwner[s.Owner] = []osdata.AgentRun{}
			if a, ok := byName[s.Owner]; ok {
				runs, err := st.AgentRuns(ctx, a.ID, wfRunsPerOwner)
				if err != nil {
					dataErr(c, err)
					return
				}
				if runs != nil {
					runsByOwner[s.Owner] = runs
				}
			}
		}
	}
	toolIDs := make([]string, 0, len(tools))
	for t := range tools {
		toolIDs = append(toolIDs, t)
	}
	sort.Strings(toolIDs)
	if jobs == nil {
		jobs = []shared.JobRow{}
	}
	c.JSON(http.StatusOK, gin.H{
		"workflows": wfs, "jobs": jobs, "agents": simple, "agentPresence": presence, "runsByOwner": runsByOwner, "toolIds": toolIDs,
		"volume": workflows.Volume(jobs, wfs, window, now, wfWindowDays),
	})
}

// runCronNow fires one scheduled task immediately and records it exactly
// like the tick does (the agent run plus a cron run row). An agent the
// runtime does not have is recorded as a failed run, as in FounderOS v1.
func runCronNow(c *gin.Context, d *Deps) {
	var body struct {
		CronID string `json:"cronId"`
	}
	raw, _ := io.ReadAll(http.MaxBytesReader(c.Writer, c.Request.Body, 1<<16))
	if json.Unmarshal(raw, &body) != nil || body.CronID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cronId required"})
		return
	}
	if d.Agents == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "agent runtime unavailable: run founderos-bootstrap so the FounderOS workspace exists"})
		return
	}
	ctx, st := c.Request.Context(), storeFor(d)
	crons, err := st.Crons(ctx, "")
	if err != nil {
		dataErr(c, err)
		return
	}
	var cron *osdata.Cron
	for i := range crons {
		if crons[i].ID == body.CronID {
			cron = &crons[i]
		}
	}
	if cron == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "unknown scheduled task: " + body.CronID})
		return
	}
	started := time.Now().UTC()
	ok, summary := false, ""
	run, err := d.Agents.Run(ctx, cron.AgentID)
	if err != nil {
		summary = err.Error()
	} else {
		ok, summary = run.OK, run.Summary
	}
	if r := []rune(summary); len(r) > 2000 {
		summary = string(r[:2000])
	}
	fin := time.Now().UTC().Format(time.RFC3339Nano)
	if err := st.InsertCronRun(ctx, osdata.CronRun{ID: uuid.NewString(), CronID: cron.ID, AgentID: cron.AgentID,
		StartedAt: started.Format(time.RFC3339Nano), FinishedAt: &fin, OK: ok, Summary: summary}); err != nil {
		dataErr(c, err)
		return
	}
	code := http.StatusOK
	if !ok {
		code = http.StatusBadGateway
	}
	c.JSON(code, gin.H{"ok": ok, "summary": summary})
}
