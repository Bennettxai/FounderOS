package api

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/rhl/businessos-backend/internal/founderos/agents"
	"github.com/rhl/businessos-backend/internal/founderos/agents/roster/sales"
	"github.com/rhl/businessos-backend/internal/founderos/connectors/fathomcalls"
	"github.com/rhl/businessos-backend/internal/founderos/connectors/plaud"
)

// FounderOS v1 POST /api/plaud/ingest and GET|POST /api/calls/archive (spec
// 5.4 writers, route-map bridge paths).
func init() { RegisterPage(registerCalls) }

func registerCalls(s *gin.RouterGroup, d *Deps) {
	s.POST("/pages/plaud/ingest", func(c *gin.Context) {
		if d.Agents == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "agent runtime unavailable"})
			return
		}
		run, err := d.Agents.Run(c.Request.Context(), "sales-calls-data")
		if errors.Is(err, agents.ErrUnknownAgent) {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error(), "run": run})
			return
		}
		c.JSON(http.StatusOK, run)
	})
	s.GET("/pages/calls/archive", func(c *gin.Context) {
		if d.Pool == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database unavailable"})
			return
		}
		rd := RosterDeps(d)
		got, err := sales.NewPgArchiveLedger(d.Pool, rd.Workspaces[sales.LedgerWorkspace]).Archived(c.Request.Context())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		byWS := map[string]int{}
		for _, e := range got {
			byWS[e.Workspace]++
		}
		c.JSON(http.StatusOK, gin.H{"source": "fathom", "archived": len(got), "byWorkspace": byWS})
	})
	s.POST("/pages/calls/archive", func(c *gin.Context) {
		if d.Pool == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database unavailable"})
			return
		}
		cd := sales.NewCallsData(RosterDeps(d), plaud.New(d.Resolver), fathomcalls.New(d.Resolver))
		res, err := cd.ArchiveCalls(c.Request.Context())
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": err.Error(), "result": res})
			return
		}
		c.JSON(http.StatusOK, res)
	})
}
