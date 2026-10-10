package api

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"

	"github.com/rhl/businessos-backend/internal/founderos/connectors/paperclip"
	"github.com/rhl/businessos-backend/internal/founderos/guard"
	"github.com/rhl/businessos-backend/internal/founderos/pages/osdata"
)

// Shared glue for the agent-side pages (/agents, /org, /tasks, /workflows):
// the Paperclip board they read, the osdata store they read and write, and
// the error mapping they answer with. Deps stays untouched (it is shared
// foundation), so the board is built per Deps once and tests swap it in.

// BoardSource is the part of the Paperclip connector these pages use.
type BoardSource interface {
	Agents(ctx context.Context) ([]paperclip.Agent, error)
	Issues(ctx context.Context, limit int) ([]paperclip.Issue, error)
	Runs(ctx context.Context, limit int) ([]paperclip.Run, error)
	FailoverRuns(ctx context.Context, limit int) ([]paperclip.FailoverRun, error)
	CreateIssue(ctx context.Context, title, description string) (*paperclip.Issue, error)
	InvokeHeartbeat(ctx context.Context, agentID string) (bool, error)
	SetAgentModel(ctx context.Context, agentID, model string) (bool, error)
	ClearAgentError(ctx context.Context, agentID string) (bool, error)
	ReassignCockpitIssue(ctx context.Context, agentID string) (bool, error)
	RepairCockpitIssue(ctx context.Context, activeCeoID string) string
	PostOnce(ctx context.Context, marker, message string) string
}

var boards sync.Map // *Deps → BoardSource

// boardFor is the Paperclip board for these Deps, built once so its breaker
// state is shared by every page that reads it.
func boardFor(d *Deps) BoardSource {
	if b, ok := boards.Load(d); ok {
		return b.(BoardSource)
	}
	b, _ := boards.LoadOrStore(d, BoardSource(paperclip.New(d.Resolver)))
	return b.(BoardSource)
}

// SetBoardSource swaps the board for tests (and for a future Deps field).
func SetBoardSource(d *Deps, b BoardSource) { boards.Store(d, b) }

func storeFor(d *Deps) *osdata.Store { return osdata.New(d.Pool) }

// resolve reads a setting through the connector resolver (env, then creds).
func resolve(d *Deps, key string) string {
	if d == nil {
		return ""
	}
	return strings.TrimSpace(d.Resolver.Resolve(key))
}

// dataErr answers a store error: a missing workspace or database is 503
// (the bridge is not bootstrapped), anything else 500.
func dataErr(c *gin.Context, err error) {
	code := http.StatusInternalServerError
	if errors.Is(err, osdata.ErrNoWorkspace) || strings.Contains(err.Error(), "no database") {
		code = http.StatusServiceUnavailable
	}
	c.JSON(code, gin.H{"error": err.Error()})
}

// writeErr answers a guarded write: FOUNDEROS_WRITES=0 is 403 with the reason,
// anything else is the board's failure (502).
func writeErr(c *gin.Context, err error) {
	if errors.Is(err, guard.ErrWritesDisabled) {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error(), "guarded": true})
		return
	}
	c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
}
