package api

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/rhl/businessos-backend/internal/founderos/connectors/paperclip"
	"github.com/rhl/businessos-backend/internal/founderos/guard"
	agentspage "github.com/rhl/businessos-backend/internal/founderos/pages/agents"
	"github.com/rhl/businessos-backend/internal/founderos/pages/osdata"
)

// /agents (spec 6.4): the live Paperclip board (seats, run feed, task
// lanes, the slab hero), the heartbeat Run buttons, model failover, the
// agent activity feed and the deliverables queue with the operator's decisions.
// The OS roster with its own Run buttons reads the existing GET /agents and
// POST /agents/:id/run (agents.go).
//
// Route map: /api/agents, /api/agents/activity, /api/agents/failover,
// /api/board/live, /api/board/agents/[id]/run, /api/board/deliverables,
// /api/board/deliverables/decision.

func init() { RegisterPage(registerAgentsPage) }

const defaultHermesURL = "https://os.example.internal:9000"

func registerAgentsPage(s *gin.RouterGroup, d *Deps) {
	s.GET("/pages/agents", func(c *gin.Context) {
		list, err := storeFor(d).Agents(c.Request.Context())
		if err != nil {
			dataErr(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"agents": list})
	})
	s.GET("/pages/agents/view", func(c *gin.Context) {
		live := boardLive(c.Request.Context(), d)
		c.JSON(http.StatusOK, gin.H{
			"boardUrl":  optional(resolve(d, "PAPERCLIP_API_URL")),
			"hermesUrl": orDefault(resolve(d, "HERMES_DASH_URL"), defaultHermesURL),
			"board":     live.Board, "volume": live.Volume, "stats": live.Stats,
		})
	})
	s.GET("/pages/agents/activity", agentsActivity(d))
	s.GET("/pages/agents/failover", func(c *gin.Context) {
		agents, plan := failoverPlan(c.Request.Context(), d)
		c.JSON(http.StatusOK, gin.H{"ok": true, "applied": false, "inspected": len(agents),
			"actions": plan.Actions, "resumes": plan.Resumes, "alerts": plan.Alerts, "handoffs": plan.Handoffs,
			"exhausted": plan.Exhausted, "notes": plan.Notes})
	})
	s.POST("/pages/agents/failover", func(c *gin.Context) { applyFailover(c, d) })
	s.GET("/pages/board/live", func(c *gin.Context) { c.JSON(http.StatusOK, boardLive(c.Request.Context(), d)) })
	s.POST("/pages/board/agents/:id/run", func(c *gin.Context) {
		ok, err := boardFor(d).InvokeHeartbeat(c.Request.Context(), c.Param("id"))
		if err != nil {
			writeErr(c, err)
			return
		}
		if !ok {
			c.JSON(http.StatusBadGateway, gin.H{"error": "board rejected the heartbeat"})
			return
		}
		c.JSON(http.StatusAccepted, gin.H{"ok": true})
	})
	s.GET("/pages/board/deliverables", func(c *gin.Context) { deliverables(c, d) })
	s.POST("/pages/board/deliverables/decision", func(c *gin.Context) { decide(c, d) })
}

func optional(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func orDefault(s, def string) string {
	if s == "" {
		return def
	}
	return s
}

// ---- board live ---------------------------------------------------------------

type liveBody struct {
	agentspage.Board
	DecisionsError string                `json:"decisionsError,omitempty"`
	Volume         agentspage.Volume     `json:"volume"`
	Stats          agentspage.BoardStats `json:"stats"`
}

// boardLive is one aggregated snapshot. An unreachable board is an honest
// connected:false with the reason and empty lists, still a 200.
func boardLive(ctx context.Context, d *Deps) liveBody {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	b := boardFor(d)
	var (
		wg               sync.WaitGroup
		agents           []paperclip.Agent
		issues           []paperclip.Issue
		runs             []paperclip.Run
		aErr, iErr, rErr error
		decisions        []osdata.Decision
		dErr             error
	)
	wg.Add(4)
	go func() { defer wg.Done(); agents, aErr = b.Agents(ctx) }()
	go func() { defer wg.Done(); issues, iErr = b.Issues(ctx, agentspage.IssuesFetched) }()
	go func() { defer wg.Done(); runs, rErr = b.Runs(ctx, agentspage.RunsFetched) }()
	go func() { defer wg.Done(); decisions, dErr = storeFor(d).Decisions(ctx) }()
	wg.Wait()

	now := time.Now()
	out := liveBody{Board: agentspage.Board{
		Agents: []paperclip.Agent{}, Issues: []paperclip.Issue{}, Runs: []paperclip.Run{},
		CheckedAt: now.UTC().Format(time.RFC3339Nano), Decisions: decisions,
	}}
	if out.Decisions == nil {
		out.Decisions = []osdata.Decision{}
	}
	if dErr != nil {
		out.DecisionsError = dErr.Error()
	}
	switch {
	case aErr != nil:
		out.Error = aErr.Error()
	case len(agents) == 0:
		out.Error = "board answered with no seats"
	default:
		out.Connected = true
		out.Agents = agents
		var partial []string
		if iErr == nil {
			out.Issues = issues
		} else {
			partial = append(partial, "issues: "+iErr.Error())
		}
		if rErr == nil {
			out.Runs = runs
		} else {
			partial = append(partial, "runs: "+rErr.Error())
		}
		out.Error = strings.Join(partial, "; ")
	}
	out.Volume = agentspage.AgentsVolume(out.Board, now, 14)
	out.Stats = agentspage.Stats(out.Board, now)
	return out
}

// ---- activity -------------------------------------------------------------------

func agentsActivity(d *Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		limit := 50
		if n, err := strconv.Atoi(c.Query("limit")); err == nil && n > 0 {
			limit = min(n, 200)
		}
		ctx, st := c.Request.Context(), storeFor(d)
		runs, err := st.AgentRuns(ctx, "", limit)
		if err != nil {
			dataErr(c, err)
			return
		}
		msgs, err := st.AgentMessages(ctx, limit)
		if err != nil {
			dataErr(c, err)
			return
		}
		bcs, err := st.Broadcasts(ctx, limit)
		if err != nil {
			dataErr(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"events": agentspage.RecentActivity(runs, msgs, bcs, limit)})
	}
}

// ---- failover ---------------------------------------------------------------------

func failoverPlan(ctx context.Context, d *Deps) ([]paperclip.Agent, agentspage.FailoverPlan) {
	b := boardFor(d)
	agents, aErr := b.Agents(ctx)
	runs, rErr := b.FailoverRuns(ctx, 40)
	in := make([]agentspage.FailoverAgent, len(agents))
	for i, a := range agents {
		in[i] = agentspage.FailoverAgent{ID: a.ID, Name: a.Name, Model: a.Model, Status: a.Status}
	}
	plan := agentspage.PlanFailover(in, runs, time.Now())
	if aErr != nil || len(agents) == 0 {
		agents = nil
		plan.Notes = append(plan.Notes, "board unreachable or unconfigured, nothing inspected")
	} else if rErr != nil {
		plan.Notes = append(plan.Notes, "runs unreadable ("+rErr.Error()+"), no failure evidence inspected")
	}
	return agents, plan
}

// applyFailover is the POST: every change is a guarded board write, so with
// FOUNDEROS_WRITES=0 the report says each one did not happen.
func applyFailover(c *gin.Context, d *Deps) {
	ctx := c.Request.Context()
	agents, plan := failoverPlan(ctx, d)
	b := boardFor(d)
	status := map[string]string{}
	for _, a := range agents {
		status[a.ID] = a.Status
	}
	type applied struct {
		AgentName    string `json:"agentName"`
		From         string `json:"from"`
		To           string `json:"to"`
		Moved        bool   `json:"moved"`
		ErrorCleared bool   `json:"errorCleared"`
		ReWoken      bool   `json:"reWoken"`
		Error        string `json:"error,omitempty"`
	}
	errText := func(err error) string {
		if err == nil {
			return ""
		}
		return err.Error()
	}
	actions := []applied{}
	for _, a := range plan.Actions {
		moved, err := b.SetAgentModel(ctx, a.AgentID, a.To)
		cleared, woken := false, false
		if moved && status[a.AgentID] == "error" {
			cleared, _ = b.ClearAgentError(ctx, a.AgentID)
		}
		if cleared {
			woken, _ = b.InvokeHeartbeat(ctx, a.AgentID)
		}
		actions = append(actions, applied{a.AgentName, a.From, a.To, moved, cleared, woken, errText(err)})
	}
	type resumed struct {
		AgentName    string `json:"agentName"`
		Reason       string `json:"reason"`
		ErrorCleared bool   `json:"errorCleared"`
		ReWoken      bool   `json:"reWoken"`
	}
	resumes := []resumed{}
	for _, r := range plan.Resumes {
		cleared, _ := b.ClearAgentError(ctx, r.AgentID)
		woken := false
		if cleared {
			woken, _ = b.InvokeHeartbeat(ctx, r.AgentID)
		}
		resumes = append(resumes, resumed{r.AgentName, r.Reason, cleared, woken})
	}
	type alerted struct {
		AgentName     string `json:"agentName"`
		Kind          string `json:"kind"`
		RunFinishedAt string `json:"runFinishedAt"`
		Result        string `json:"result"`
	}
	alerts := []alerted{}
	for _, a := range plan.Alerts {
		res := b.PostOnce(ctx, agentspage.AlertMarker(a), agentspage.RenderAlert(a))
		alerts = append(alerts, alerted{a.AgentName, a.Kind, a.RunFinishedAt, res})
	}
	type handed struct {
		From   string `json:"from"`
		To     string `json:"to"`
		Reason string `json:"reason"`
		Moved  bool   `json:"moved"`
	}
	handoffs := []handed{}
	activeCeo := ""
	for _, h := range plan.Handoffs {
		moved, _ := b.ReassignCockpitIssue(ctx, h.ToAgentID)
		if moved {
			activeCeo = h.ToAgentID
		}
		handoffs = append(handoffs, handed{h.FromAgentName, h.ToAgentName, h.Reason, moved})
	}
	cockpit := "skipped"
	if len(agents) > 0 {
		cockpit = b.RepairCockpitIssue(ctx, activeCeo)
	}
	c.JSON(http.StatusOK, gin.H{"ok": true, "applied": true, "inspected": len(agents), "exhausted": plan.Exhausted, "notes": plan.Notes,
		"actions": actions, "resumes": resumes, "alerts": alerts, "handoffs": handoffs, "cockpit": cockpit})
}

// ---- deliverables -------------------------------------------------------------------

// deliverablesDir is PAPERCLIP_WORKSPACES_DIR, defaulting where prod looks
// (lib/board-deliverables WORKSPACES_DIR): ~/.paperclip/instances/default/
// workspaces. The workspaces live on the board host, so on a box without that
// directory the list is an honest empty.
func deliverablesDir(d *Deps) string {
	if dir := resolve(d, "PAPERCLIP_WORKSPACES_DIR"); dir != "" {
		return dir
	}
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return ""
	}
	return filepath.Join(home, ".paperclip", "instances", "default", "workspaces")
}

func deliverables(c *gin.Context, d *Deps) {
	dir := deliverablesDir(d)
	file := c.Query("file")
	if file == "" {
		ctx, st := c.Request.Context(), storeFor(d)
		decisions, err := st.Decisions(ctx)
		if err != nil {
			dataErr(c, err)
			return
		}
		proposals, err := st.Proposals(ctx)
		if err != nil {
			dataErr(c, err)
			return
		}
		available, reason := true, ""
		if dir == "" {
			available, reason = false, "PAPERCLIP_WORKSPACES_DIR is not set: the board's agent workspaces live on the board host"
		} else if fi, err := os.Stat(dir); err != nil || !fi.IsDir() {
			available, reason = false, "the board workspaces directory is not on this machine"
		}
		files := agentspage.ListDeliverables(dir)
		groups := agentspage.GroupDeliverables(files, proposals)
		var fileItems []agentspage.DeliverableItem
		for _, g := range groups {
			for _, it := range g.Items {
				if it.Kind == "file" {
					fileItems = append(fileItems, it)
				}
			}
		}
		now := time.Now()
		queue := agentspage.NeedsYou(fileItems, now)
		asItems := make([]agentspage.DeliverableItem, len(queue))
		byID := map[string]agentspage.Classified{}
		for i, q := range queue {
			asItems[i] = q.DeliverableItem
			byID[q.ID] = q
		}
		open, decided := agentspage.PartitionByDecision(asItems, decisions)
		openC := make([]agentspage.Classified, len(open))
		for i, it := range open {
			openC[i] = byID[it.ID]
		}
		c.JSON(http.StatusOK, gin.H{"groups": groups, "deliverables": files, "decisions": decisions, "dir": dir,
			"available": available, "reason": reason, "needsYou": gin.H{"open": openC, "decided": decided},
			"boardUrl": optional(resolve(d, "PAPERCLIP_API_URL"))})
		return
	}

	full := agentspage.ResolveDeliverable(file, dir)
	if full == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad file key"})
		return
	}
	name := filepath.Base(full)
	kind := agentspage.PreviewKind(name)
	if c.Query("mode") == "view" {
		if kind != "text" {
			c.JSON(http.StatusOK, gin.H{"id": file, "name": name, "kind": kind, "text": nil, "truncated": false})
			return
		}
		raw, err := os.ReadFile(full)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "file not found"})
			return
		}
		text, cut := agentspage.ClampPreview(string(raw), agentspage.PreviewMaxChars)
		c.JSON(http.StatusOK, gin.H{"id": file, "name": name, "kind": kind, "text": text, "truncated": cut})
		return
	}
	f, err := os.Open(full)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "file not found"})
		return
	}
	defer f.Close()
	mime := agentspage.DeliverableMIME[strings.ToLower(filepath.Ext(full))]
	if mime == "" {
		mime = "application/octet-stream"
	}
	disp := "attachment"
	if c.Query("inline") == "1" {
		disp = "inline"
	}
	c.Header("Content-Type", mime)
	c.Header("Content-Disposition", disp+`; filename="`+strings.ReplaceAll(name, `"`, "")+`"`)
	c.Header("Cache-Control", "no-store")
	c.Status(http.StatusOK)
	_, _ = io.Copy(c.Writer, f)
}

const decisionBatchMax = 500

// decide records the operator's call on one piece of agent work. Bulk is
// dismiss-only: approve means send, one at a time.
func decide(c *gin.Context, d *Deps) {
	var raw map[string]json.RawMessage
	if err := json.NewDecoder(http.MaxBytesReader(c.Writer, c.Request.Body, 1<<20)).Decode(&raw); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
		return
	}
	ctx, st := c.Request.Context(), storeFor(d)
	decision := strings.TrimSpace(string(raw["decision"]))
	isNull := decision == "null"
	var kind string
	if !isNull {
		_ = json.Unmarshal(raw["decision"], &kind)
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)

	if items, bulk := raw["items"]; bulk {
		var list []struct {
			ID              string `json:"id"`
			DecidedRevision string `json:"decidedRevision"`
		}
		if json.Unmarshal(items, &list) != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "items must be a list"})
			return
		}
		if !isNull && kind != "dismissed" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "bulk supports dismiss only; approve is one at a time because approve means send"})
			return
		}
		if len(list) > decisionBatchMax {
			c.JSON(http.StatusBadRequest, gin.H{"error": "batch too large (max 500)"})
			return
		}
		count := 0
		for _, it := range list {
			if it.ID == "" {
				continue
			}
			var err error
			if isNull {
				err = st.ClearDecision(ctx, it.ID)
			} else {
				err = st.SetDecision(ctx, osdata.Decision{ID: it.ID, Decision: "dismissed", DecidedAt: now, DecidedRevision: it.DecidedRevision})
			}
			if err != nil {
				dataErr(c, err)
				return
			}
			count++
		}
		c.JSON(http.StatusOK, gin.H{"ok": true, "count": count})
		return
	}

	var id, rev, note string
	_ = json.Unmarshal(raw["id"], &id)
	_ = json.Unmarshal(raw["decidedRevision"], &rev)
	_ = json.Unmarshal(raw["note"], &note)
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id is required"})
		return
	}
	if isNull {
		if err := st.ClearDecision(ctx, id); err != nil {
			dataErr(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true, "decision": nil})
		return
	}
	if !osdata.ValidDecision(kind) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid decision"})
		return
	}
	dec := osdata.Decision{ID: id, Decision: kind, DecidedAt: now, DecidedRevision: rev, Note: note}
	// Approve is the outward call (send it, publish, go ahead): agents act on
	// it. With FOUNDEROS_WRITES=0 the guard refuses it, records the refusal, and
	// the item stays open. Dismiss only changes his own queue.
	if kind == "approved" {
		if err := guard.Outbound("deliverable:approve:"+id, func() error { return nil }); err != nil {
			writeErr(c, err)
			return
		}
	}
	if err := st.SetDecision(ctx, dec); err != nil {
		if errors.Is(err, osdata.ErrNoWorkspace) {
			dataErr(c, err)
			return
		}
		dataErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true, "decision": dec})
}
