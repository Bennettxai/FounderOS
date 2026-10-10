package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rhl/businessos-backend/internal/config"
	"github.com/rhl/businessos-backend/internal/middleware"
	"github.com/rhl/businessos-backend/internal/services"
)

// IntelligenceHandler powers the Intelligence module: signals, risks, and
// recommendations synthesized across the workspace. Deterministic signals are
// computed live from data; the AI synthesis is generated on demand and cached
// per workspace in intelligence_reports. Workspace-scoped via X-Workspace-ID.
type IntelligenceHandler struct {
	pool *pgxpool.Pool
	cfg  *config.Config
}

func NewIntelligenceHandler(pool *pgxpool.Pool, cfg *config.Config) *IntelligenceHandler {
	return &IntelligenceHandler{pool: pool, cfg: cfg}
}

func (h *IntelligenceHandler) workspaceFromHeader(c *gin.Context, userID string) (uuid.UUID, bool) {
	hdr := c.GetHeader("X-Workspace-ID")
	if hdr == "" {
		return uuid.Nil, false
	}
	wsID, err := uuid.Parse(hdr)
	if err != nil {
		return uuid.Nil, false
	}
	var member bool
	err = h.pool.QueryRow(c.Request.Context(),
		`SELECT EXISTS(SELECT 1 FROM workspace_members WHERE workspace_id=$1 AND user_id=$2 AND status='active')`,
		wsID, userID).Scan(&member)
	if err != nil || !member {
		return uuid.Nil, false
	}
	return wsID, true
}

// Signal is one deterministic finding computed straight from workspace data.
type Signal struct {
	Kind       string `json:"kind"`        // "signal" | "risk"
	Title      string `json:"title"`
	Detail     string `json:"detail"`
	Severity   string `json:"severity"`    // "low" | "medium" | "high"
	SourceType string `json:"source_type"` // "task" | "client" | "project"
	Count      int    `json:"count"`
}

// Insight is an AI-synthesized item (may reference the deterministic signals).
type Insight struct {
	Type     string `json:"type"`     // "signal" | "risk" | "recommendation"
	Title    string `json:"title"`
	Detail   string `json:"detail"`
	Priority string `json:"priority"` // "high" | "medium" | "low"
}

// computeSignals runs the deterministic rules over the workspace and returns
// both the structured signals and a compact human-readable snapshot used to
// prime the AI synthesis.
func (h *IntelligenceHandler) computeSignals(ctx context.Context, wsID uuid.UUID) ([]Signal, string, error) {
	signals := []Signal{}
	var snap strings.Builder

	scalar := func(q string) int {
		var n int
		_ = h.pool.QueryRow(ctx, q, wsID).Scan(&n)
		return n
	}
	titles := func(q string, limit int) []string {
		rows, err := h.pool.Query(ctx, q, wsID)
		if err != nil {
			return nil
		}
		defer rows.Close()
		out := []string{}
		for rows.Next() && len(out) < limit {
			var t string
			if rows.Scan(&t) == nil {
				out = append(out, t)
			}
		}
		return out
	}

	// --- Overdue open tasks ---
	overdue := scalar(`SELECT count(*) FROM tasks WHERE workspace_id=$1 AND status<>'done'
		AND completed_at IS NULL AND due_date IS NOT NULL AND due_date < now()`)
	if overdue > 0 {
		ex := titles(`SELECT title FROM tasks WHERE workspace_id=$1 AND status<>'done'
			AND completed_at IS NULL AND due_date IS NOT NULL AND due_date < now()
			ORDER BY due_date ASC`, 5)
		sev := "medium"
		if overdue >= 10 {
			sev = "high"
		}
		signals = append(signals, Signal{
			Kind: "risk", Title: fmt.Sprintf("%d overdue tasks", overdue),
			Detail: "Past due, not completed. Oldest: " + strings.Join(ex, "; "),
			Severity: sev, SourceType: "task", Count: overdue,
		})
		fmt.Fprintf(&snap, "- %d overdue tasks (e.g. %s)\n", overdue, strings.Join(ex, "; "))
	}

	// --- Due today ---
	dueToday := scalar(`SELECT count(*) FROM tasks WHERE workspace_id=$1 AND status<>'done'
		AND completed_at IS NULL AND due_date::date = now()::date`)
	if dueToday > 0 {
		signals = append(signals, Signal{
			Kind: "signal", Title: fmt.Sprintf("%d tasks due today", dueToday),
			Detail: "Due today and not yet completed.", Severity: "medium", SourceType: "task", Count: dueToday,
		})
		fmt.Fprintf(&snap, "- %d tasks due today\n", dueToday)
	}

	// --- Due in the next 7 days ---
	dueWeek := scalar(`SELECT count(*) FROM tasks WHERE workspace_id=$1 AND status<>'done'
		AND completed_at IS NULL AND due_date > now() AND due_date <= now() + interval '7 days'`)
	if dueWeek > 0 {
		fmt.Fprintf(&snap, "- %d tasks due within 7 days\n", dueWeek)
	}

	// --- In-progress load ---
	inProgress := scalar(`SELECT count(*) FROM tasks WHERE workspace_id=$1 AND status='in_progress'`)
	if inProgress > 0 {
		fmt.Fprintf(&snap, "- %d tasks in progress\n", inProgress)
	}

	// --- Clients gone quiet (14+ days, or never contacted) ---
	stale := scalar(`SELECT count(*) FROM clients WHERE workspace_id=$1
		AND (last_contacted_at IS NULL OR last_contacted_at < now() - interval '14 days')`)
	totalClients := scalar(`SELECT count(*) FROM clients WHERE workspace_id=$1`)
	if stale > 0 {
		ex := titles(`SELECT name FROM clients WHERE workspace_id=$1
			AND (last_contacted_at IS NULL OR last_contacted_at < now() - interval '14 days')
			ORDER BY last_contacted_at ASC NULLS FIRST`, 5)
		signals = append(signals, Signal{
			Kind: "risk", Title: fmt.Sprintf("%d client relationships gone quiet", stale),
			Detail: "No contact logged in 14+ days: " + strings.Join(ex, ", "),
			Severity: "medium", SourceType: "client", Count: stale,
		})
		fmt.Fprintf(&snap, "- %d of %d clients not contacted in 14+ days (%s)\n", stale, totalClients, strings.Join(ex, ", "))
	}

	// --- Overdue / stalled projects ---
	overdueProj := scalar(`SELECT count(*) FROM projects WHERE workspace_id=$1
		AND status <> 'completed' AND completed_at IS NULL
		AND due_date IS NOT NULL AND due_date < now()`)
	if overdueProj > 0 {
		ex := titles(`SELECT name FROM projects WHERE workspace_id=$1
			AND status <> 'completed' AND completed_at IS NULL
			AND due_date IS NOT NULL AND due_date < now() ORDER BY due_date ASC`, 5)
		signals = append(signals, Signal{
			Kind: "risk", Title: fmt.Sprintf("%d projects past their due date", overdueProj),
			Detail: "Not completed, past due: " + strings.Join(ex, ", "),
			Severity: "high", SourceType: "project", Count: overdueProj,
		})
		fmt.Fprintf(&snap, "- %d projects past due (%s)\n", overdueProj, strings.Join(ex, ", "))
	}

	// --- Stalled projects (no update in 21+ days, still open) ---
	stalledProj := scalar(`SELECT count(*) FROM projects WHERE workspace_id=$1
		AND status <> 'completed' AND completed_at IS NULL AND updated_at < now() - interval '21 days'`)
	if stalledProj > 0 {
		signals = append(signals, Signal{
			Kind: "signal", Title: fmt.Sprintf("%d projects with no recent activity", stalledProj),
			Detail: "Open but not updated in 21+ days.", Severity: "low", SourceType: "project", Count: stalledProj,
		})
		fmt.Fprintf(&snap, "- %d projects stalled (no update 21+ days)\n", stalledProj)
	}

	if snap.Len() == 0 {
		snap.WriteString("- No notable signals: no overdue tasks, no stale clients, no stalled projects.\n")
	}
	return signals, snap.String(), nil
}

// GetIntelligence returns deterministic signals/risks + the cached AI insights.
// GET /api/intelligence
func (h *IntelligenceHandler) GetIntelligence(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		RespondUnauthorizedErr(c, "authentication required")
		return
	}
	wsID, ok := h.workspaceFromHeader(c, user.ID)
	if !ok {
		c.JSON(http.StatusForbidden, gin.H{"error": "select a workspace first"})
		return
	}

	signals, _, err := h.computeSignals(c.Request.Context(), wsID)
	if err != nil {
		RespondInternalErr(c, "compute signals", err)
		return
	}

	// Load cached AI synthesis (if any).
	var insightsRaw []byte
	var model string
	var generatedAt *time.Time
	_ = h.pool.QueryRow(c.Request.Context(),
		`SELECT insights, model, generated_at FROM intelligence_reports WHERE workspace_id=$1`, wsID).
		Scan(&insightsRaw, &model, &generatedAt)
	insights := []Insight{}
	if len(insightsRaw) > 0 {
		_ = json.Unmarshal(insightsRaw, &insights)
	}

	c.JSON(http.StatusOK, gin.H{
		"signals":      signals,
		"insights":     insights,
		"ai_model":     model,
		"ai_generated": generatedAt,
		"ai_available": h.cfg.GetActiveProvider() != "",
	})
}

// Synthesize (re)generates the AI insights from the current data snapshot.
// POST /api/intelligence/synthesize
func (h *IntelligenceHandler) Synthesize(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		RespondUnauthorizedErr(c, "authentication required")
		return
	}
	wsID, ok := h.workspaceFromHeader(c, user.ID)
	if !ok {
		c.JSON(http.StatusForbidden, gin.H{"error": "select a workspace first"})
		return
	}

	_, snapshot, err := h.computeSignals(c.Request.Context(), wsID)
	if err != nil {
		RespondInternalErr(c, "compute signals", err)
		return
	}

	llm := services.NewLLMService(h.cfg, "")
	system := "You are the Intelligence layer of a business operating system. " +
		"Given a snapshot of deterministic signals from the operator's real data, produce 3 to 6 " +
		"prioritized, specific insights. Each item must be one of type: \"signal\" (a notable pattern), " +
		"\"risk\" (something trending wrong), or \"recommendation\" (a concrete next action). " +
		"Be concrete and reference the actual numbers/names. Return ONLY a JSON array, no prose, " +
		"each element: {\"type\":\"signal|risk|recommendation\",\"title\":\"short\",\"detail\":\"1-2 sentences\",\"priority\":\"high|medium|low\"}."
	userMsg := "Workspace snapshot:\n" + snapshot + "\nProduce the JSON array now."

	resp, err := llm.ChatComplete(c.Request.Context(),
		[]services.ChatMessage{{Role: "user", Content: userMsg}}, system)
	if err != nil {
		RespondInternalErr(c, "ai synthesis", err)
		return
	}

	insights := parseInsights(resp)
	if len(insights) == 0 {
		// Model returned nothing parseable — surface it rather than caching junk.
		c.JSON(http.StatusOK, gin.H{"insights": []Insight{}, "raw": resp, "note": "no parseable insights"})
		return
	}

	payload, _ := json.Marshal(insights)
	model := llm.GetModel()
	_, err = h.pool.Exec(c.Request.Context(), `
		INSERT INTO intelligence_reports (workspace_id, insights, model, generated_at)
		VALUES ($1, $2, $3, now())
		ON CONFLICT (workspace_id) DO UPDATE SET insights=$2, model=$3, generated_at=now()
	`, wsID, payload, model)
	if err != nil {
		RespondInternalErr(c, "cache insights", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"insights": insights, "ai_model": model})
}

// parseInsights extracts the JSON array from a model response, tolerating
// ```json fences and leading/trailing prose.
func parseInsights(resp string) []Insight {
	s := strings.TrimSpace(resp)
	s = strings.TrimPrefix(s, "```json")
	s = strings.TrimPrefix(s, "```")
	s = strings.TrimSuffix(s, "```")
	// Slice to the outermost [ ... ].
	start := strings.Index(s, "[")
	end := strings.LastIndex(s, "]")
	if start >= 0 && end > start {
		s = s[start : end+1]
	}
	var out []Insight
	if err := json.Unmarshal([]byte(s), &out); err != nil {
		return nil
	}
	// Normalize.
	clean := out[:0]
	for _, i := range out {
		if strings.TrimSpace(i.Title) == "" {
			continue
		}
		if i.Type == "" {
			i.Type = "signal"
		}
		if i.Priority == "" {
			i.Priority = "medium"
		}
		clean = append(clean, i)
	}
	return clean
}
