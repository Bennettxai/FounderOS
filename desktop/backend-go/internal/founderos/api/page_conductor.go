package api

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/rhl/businessos-backend/internal/founderos/connectors/paperclip"
	"github.com/rhl/businessos-backend/internal/founderos/connectors/superset"
	"github.com/rhl/businessos-backend/internal/founderos/guard"
	"github.com/rhl/businessos-backend/internal/founderos/memory"
	"github.com/rhl/businessos-backend/internal/founderos/pages/conductor"
)

// ConductorDeps are the Conductor panel's seams (Paperclip cockpit, Superset
// dispatch, memory capture), swappable in tests.
type ConductorDeps struct {
	Board interface {
		CockpitThread(ctx context.Context, limit int) ([]paperclip.Comment, error)
		PostCockpitMessage(ctx context.Context, message string) (*paperclip.Comment, error)
		CreateIssue(ctx context.Context, title, description string) (*paperclip.Issue, error)
		Agents(ctx context.Context) ([]paperclip.Agent, error)
	}
	Dispatcher interface {
		DispatchCodingTask(ctx context.Context, request string) (superset.DispatchResult, error)
	}
	Memory interface {
		Capture(ctx context.Context, c memory.Capture) (string, error)
	}
}

// FounderOS v1 /api/conductor/{chat,context,dispatch} and /api/interject.
func init() { RegisterPage(registerConductor) }

const contextBudget = 2500 * time.Millisecond

func conductorDeps(d *Deps) *ConductorDeps {
	if d.Conductor != nil {
		return d.Conductor
	}
	cd := &ConductorDeps{Board: paperclip.New(d.Resolver), Dispatcher: superset.New(d.Resolver)}
	if m := RosterDeps(d).Memory; m != nil {
		cd.Memory = m
	}
	return cd
}

// boardNotConfigured: the Paperclip connector has no credentials (its
// unexported errNoCreds), as opposed to a board that is set up but failing.
func boardNotConfigured(err error) bool {
	return err != nil && strings.Contains(err.Error(), "creds missing")
}

// writeError maps a failed write: refused by the bridge guard → 409, anything
// else → 502 (the TS contract).
func writeError(c *gin.Context, err error) {
	if errors.Is(err, guard.ErrWritesDisabled) {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error(), "refused": true, "guarded": true})
		return
	}
	c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
}

func registerConductor(s *gin.RouterGroup, d *Deps) {
	s.GET("/pages/conductor/chat", func(c *gin.Context) {
		// v1 paperclipCockpitThread: an unset or unreachable board is an empty
		// thread (200), never a 5xx; `configured` says which one it is.
		msgs, err := conductorDeps(d).Board.CockpitThread(c.Request.Context(), 50)
		if err != nil {
			out := gin.H{"messages": []paperclip.Comment{}, "configured": !boardNotConfigured(err)}
			if !boardNotConfigured(err) {
				out["error"] = err.Error()
			}
			c.JSON(http.StatusOK, out)
			return
		}
		if msgs == nil {
			msgs = []paperclip.Comment{}
		}
		c.JSON(http.StatusOK, gin.H{"messages": msgs, "configured": true})
	})
	s.POST("/pages/conductor/chat", func(c *gin.Context) {
		var body struct {
			Message string `json:"message"`
		}
		if err := c.ShouldBindJSON(&body); err != nil || strings.TrimSpace(body.Message) == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "message required"})
			return
		}
		comment, err := conductorDeps(d).Board.PostCockpitMessage(c.Request.Context(), strings.TrimSpace(body.Message))
		if err != nil && boardNotConfigured(err) {
			// nothing was sent, and nothing is broken: the board is simply not set up
			c.JSON(http.StatusConflict, gin.H{"error": "Paperclip is not configured, nothing was sent.", "notConfigured": true})
			return
		}
		if err != nil {
			writeError(c, err)
			return
		}
		c.JSON(http.StatusCreated, gin.H{"comment": comment})
	})
	s.GET("/pages/conductor/context", func(c *gin.Context) {
		path := c.DefaultQuery("path", "/")
		title := conductor.ScreenTitle(path)
		out := gin.H{
			"title":        title,
			"context":      screenContext(c.Request.Context(), d, path, title),
			"quickActions": conductor.QuickActions(path),
			"model":        nil,
		}
		ctx, cancel := context.WithTimeout(c.Request.Context(), contextBudget)
		defer cancel()
		if agents, err := conductorDeps(d).Board.Agents(ctx); err == nil {
			for _, a := range agents {
				if a.Name == "Conductor" && a.Model != nil {
					out["model"] = *a.Model
				}
			}
		}
		c.JSON(http.StatusOK, out)
	})
	s.POST("/pages/conductor/dispatch", func(c *gin.Context) {
		var body struct {
			Request string `json:"request"`
		}
		if err := c.ShouldBindJSON(&body); err != nil || strings.TrimSpace(body.Request) == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "request required"})
			return
		}
		cd := conductorDeps(d)
		req := strings.TrimSpace(body.Request)
		res, err := cd.Dispatcher.DispatchCodingTask(c.Request.Context(), req)
		if err != nil {
			writeError(c, err)
			return
		}
		// Best effort, as in the TS: a board note about the dispatch.
		_, _ = cd.Board.CreateIssue(c.Request.Context(), "UI change: "+clip(req, 80),
			"Dispatched to a Superset coding agent.\n\nRequest: "+req+"\nBranch: "+res.Branch+"\nWorkspace: "+res.WorkspaceID)
		c.JSON(http.StatusCreated, res)
	})
}

// screenContext is a short, honest description of the screen for the
// Conductor. Per-page detail (e.g. the funnel summary) lands with each page.
func screenContext(ctx context.Context, d *Deps, path, title string) string {
	ctx, cancel := context.WithTimeout(ctx, contextBudget)
	defer cancel()
	base := title + " view of Founder OS."
	switch p := strings.TrimPrefix(strings.SplitN(path, "?", 2)[0], "/os"); {
	case p == "" || p == "/" || strings.HasPrefix(p, "/agents") || strings.HasPrefix(p, "/org"):
		if d.Agents == nil {
			return base
		}
		list := d.Agents.List()
		ran, failed := 0, 0
		for _, m := range list {
			runs, err := d.Agents.Recent(ctx, m.ID, 1)
			if err != nil || len(runs) == 0 {
				continue
			}
			ran++
			if !runs[0].OK {
				failed++
			}
		}
		return base + " Agents: " + strconv.Itoa(len(list)) + " registered, " + strconv.Itoa(ran) + " have run, " + strconv.Itoa(failed) + " last run failed."
	case strings.HasPrefix(p, "/integrations"):
		if d.Board == nil {
			return base
		}
		st := d.Board.Statuses(ctx)
		up := 0
		for _, s := range st {
			if s.State == "connected" {
				up++
			}
		}
		return base + " Connections: " + strconv.Itoa(up) + "/" + strconv.Itoa(len(st)) + " connected."
	}
	return base
}

// clip keeps the first n characters; it never splits a UTF-8 sequence.
func clip(s string, n int) string {
	if n < 0 {
		n = 0
	}
	if r := []rune(s); len(r) > n {
		return string(r[:n])
	}
	return s
}
