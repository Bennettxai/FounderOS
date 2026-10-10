package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/rhl/businessos-backend/internal/founderos/agents"
	"github.com/rhl/businessos-backend/internal/founderos/connectors/paperclip"
	"github.com/rhl/businessos-backend/internal/founderos/pages/osdata"
	"github.com/rhl/businessos-backend/internal/founderos/pages/shared"
	"github.com/rhl/businessos-backend/internal/founderos/pages/tasks"
)

// /tasks (spec 6.16): the task volume slab over the local kanban
// (founderos_agent_tasks), the live Paperclip board queue and the scheduled
// jobs; the board composer (a guarded board write); and the agent work
// route the kanban and the cron controls write through.
//
// Route map: /api/board/tasks → /pages/board/tasks, /api/agents/work →
// /pages/agents/work. GET /pages/tasks is the page's own view model.

func init() { RegisterPage(registerTasksPage) }

const tasksWindowDays = 14

// tasksBoardIssues is the depth /tasks reads the board queue at.
const tasksBoardIssues = 30

func registerTasksPage(s *gin.RouterGroup, d *Deps) {
	s.GET("/pages/tasks", func(c *gin.Context) { tasksView(c, d) })
	s.GET("/pages/board/tasks", func(c *gin.Context) {
		issues, err := boardFor(d).Issues(c.Request.Context(), tasksBoardIssues)
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
			return
		}
		if issues == nil {
			issues = []paperclip.Issue{}
		}
		c.JSON(http.StatusOK, gin.H{"issues": issues})
	})
	s.POST("/pages/board/tasks", func(c *gin.Context) {
		var body struct {
			Title       string `json:"title"`
			Description string `json:"description"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON"})
			return
		}
		title := strings.TrimSpace(body.Title)
		if title == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "title required"})
			return
		}
		issue, err := boardFor(d).CreateIssue(c.Request.Context(), title, body.Description)
		if err != nil {
			writeErr(c, err)
			return
		}
		c.JSON(http.StatusCreated, gin.H{"issue": issue})
	})
	s.GET("/pages/agents/work", func(c *gin.Context) { workGet(c, d) })
	s.POST("/pages/agents/work", func(c *gin.Context) { workPost(c, d) })
	s.PATCH("/pages/agents/work", func(c *gin.Context) { workPatch(c, d) })
	s.DELETE("/pages/agents/work", func(c *gin.Context) { workDelete(c, d) })
}

func tasksView(c *gin.Context, d *Deps) {
	ctx := c.Request.Context()
	st := storeFor(d)
	now := time.Now()

	// the board read runs beside the store reads; it has its own deadline
	type boardRead struct {
		issues []paperclip.Issue
		err    error
	}
	boardCh := make(chan boardRead, 1)
	go func() {
		bctx, cancel := context.WithTimeout(ctx, 8*time.Second)
		defer cancel()
		is, err := boardFor(d).Issues(bctx, tasksBoardIssues)
		boardCh <- boardRead{is, err}
	}()

	taskRows, err := st.Tasks(ctx, "")
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
	runs, err := st.CronRunsSince(ctx, now.Add(-(tasksWindowDays+1)*24*time.Hour))
	if err != nil {
		dataErr(c, err)
		return
	}
	names := map[string]string{}
	for _, a := range roster {
		names[a.ID] = a.Name
	}
	jobs := shared.ScheduledJobRows(crons, stats, names, nil, now)

	br := <-boardCh
	issues := []paperclip.Issue{}
	board := gin.H{"connected": br.err == nil, "url": optional(resolve(d, "PAPERCLIP_API_URL"))}
	if br.err != nil {
		board["error"] = br.err.Error()
	} else if br.issues != nil {
		issues = br.issues
	}

	in := tasks.Input{BoardConnected: br.err == nil, AgentNames: names, Now: now, Days: tasksWindowDays}
	for _, t := range taskRows {
		in.Tasks = append(in.Tasks, tasks.Task{AgentID: t.AgentID, Status: t.Status, UpdatedAt: t.UpdatedAt})
	}
	for _, i := range issues {
		in.Issues = append(in.Issues, tasks.Issue{Status: i.Status, UpdatedAt: i.UpdatedAt})
	}
	for _, j := range jobs {
		in.Jobs = append(in.Jobs, tasks.Job{Description: j.Description, Enabled: j.Enabled, UnknownAgent: j.UnknownAgent, Overdue: j.Overdue, LastOK: j.LastOK, Runs: j.Runs, OK: j.OK})
	}
	for _, r := range runs {
		in.Runs = append(in.Runs, tasks.Run{StartedAt: r.StartedAt, OK: r.OK})
	}
	if taskRows == nil {
		taskRows = []osdata.Task{}
	}
	if crons == nil {
		crons = []osdata.Cron{}
	}
	c.JSON(http.StatusOK, gin.H{
		"tasks": taskRows, "agentNames": names, "issues": issues, "board": board,
		"crons": crons, "cronStats": stats, "jobs": jobs, "volume": tasks.Volume(in),
	})
}

// ---- /pages/agents/work (app/api/agents/work/route.ts) ------------------------

func workGet(c *gin.Context, d *Deps) {
	ctx, st, agentID := c.Request.Context(), storeFor(d), c.Query("agentId")
	ts, err := st.Tasks(ctx, agentID)
	if err != nil {
		dataErr(c, err)
		return
	}
	cs, err := st.Crons(ctx, agentID)
	if err != nil {
		dataErr(c, err)
		return
	}
	if ts == nil {
		ts = []osdata.Task{}
	}
	if cs == nil {
		cs = []osdata.Cron{}
	}
	c.JSON(http.StatusOK, gin.H{"tasks": ts, "crons": cs})
}

type workBody struct {
	Kind        string `json:"kind"`
	ID          string `json:"id"`
	AgentID     string `json:"agentId"`
	Title       string `json:"title"`
	Schedule    string `json:"schedule"`
	Description string `json:"description"`
	Status      string `json:"status"`
	Enabled     *bool  `json:"enabled"`
}

func readWork(c *gin.Context) (workBody, bool) {
	var b workBody
	if err := json.NewDecoder(http.MaxBytesReader(c.Writer, c.Request.Body, 1<<20)).Decode(&b); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON body"})
		return b, false
	}
	return b, true
}

func badWork(c *gin.Context, msg string) { c.JSON(http.StatusBadRequest, gin.H{"error": msg}) }

func workPost(c *gin.Context, d *Deps) {
	b, ok := readWork(c)
	if !ok {
		return
	}
	ctx, st := c.Request.Context(), storeFor(d)
	now := time.Now().UTC().Format(time.RFC3339Nano)
	switch b.Kind {
	case "task":
		if b.AgentID == "" || b.Title == "" {
			badWork(c, "task needs agentId and title")
			return
		}
		t := osdata.Task{ID: uuid.NewString(), AgentID: b.AgentID, Title: b.Title, Status: "open", CreatedAt: now, UpdatedAt: now}
		if err := st.InsertTask(ctx, t); err != nil {
			dataErr(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true, "task": t})
	case "cron":
		if b.AgentID == "" || b.Schedule == "" || b.Description == "" {
			badWork(c, "cron needs agentId, schedule and description")
			return
		}
		if !agents.ValidCron(b.Schedule) {
			badWork(c, "invalid cron schedule: "+b.Schedule+` — use 5 fields like "0 9 * * 1-5"`)
			return
		}
		cr := osdata.Cron{ID: uuid.NewString(), AgentID: b.AgentID, Schedule: b.Schedule, Description: b.Description, Enabled: true, CreatedAt: now}
		if err := st.InsertCron(ctx, cr); err != nil {
			dataErr(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true, "cron": cr})
	default:
		badWork(c, `kind must be "task" or "cron"`)
	}
}

func workPatch(c *gin.Context, d *Deps) {
	b, ok := readWork(c)
	if !ok {
		return
	}
	ctx, st := c.Request.Context(), storeFor(d)
	if b.ID == "" {
		badWork(c, "id is required")
		return
	}
	var err error
	switch b.Kind {
	case "task":
		if !osdata.ValidTaskStatus(b.Status) {
			badWork(c, "status must be one of open, doing, review, done")
			return
		}
		err = st.SetTaskStatus(ctx, b.ID, b.Status, time.Now().UTC())
	case "cron":
		if b.Enabled == nil {
			badWork(c, "enabled (boolean) is required")
			return
		}
		err = st.SetCronEnabled(ctx, b.ID, *b.Enabled)
	default:
		badWork(c, `kind must be "task" or "cron"`)
		return
	}
	if errors.Is(err, osdata.ErrNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": b.Kind + " not found: " + b.ID})
		return
	}
	if err != nil {
		dataErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func workDelete(c *gin.Context, d *Deps) {
	b, ok := readWork(c)
	if !ok {
		return
	}
	if b.ID == "" || (b.Kind != "task" && b.Kind != "cron") {
		badWork(c, `kind ("task" | "cron") and id are required`)
		return
	}
	ctx, st := c.Request.Context(), storeFor(d)
	var err error
	if b.Kind == "task" {
		err = st.RemoveTask(ctx, b.ID)
	} else {
		err = st.RemoveCron(ctx, b.ID)
	}
	if err != nil {
		dataErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}
