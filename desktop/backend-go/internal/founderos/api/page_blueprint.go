package api

import (
	"context"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/rhl/businessos-backend/internal/founderos/pages/blueprint"
	"github.com/rhl/businessos-backend/internal/founderos/topology"
)

// /blueprint (spec 6.19): the system map, compiled from the bridge's live
// registries on every request (FounderOS v1 GET /api/blueprint).
func init() { RegisterPage(registerBlueprint) }

// blueprintHost says which machine is compiling; tests swap it out so the
// endpoint probes no real ports.
var blueprintHost = func() string {
	return blueprint.CurrentHostID(os.Getenv)
}

func registerBlueprint(s *gin.RouterGroup, d *Deps) {
	s.GET("/pages/blueprint", func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 15*time.Second)
		defer cancel()
		if d.Pool == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"ok": false, "error": "the roster is unreachable: no database"})
			return
		}
		var ws string
		if err := d.Pool.QueryRow(ctx, `SELECT id::text FROM workspaces WHERE slug = 'founderos'`).Scan(&ws); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"ok": false, "error": "the roster is unreachable: no founderos workspace (run founderos-bootstrap)"})
			return
		}
		reg, err := blueprint.LoadRegistries(ctx, d.Pool, ws)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": err.Error()})
			return
		}
		o := blueprint.Options{
			Operator:  os.Getenv("FOUNDEROS_OPERATOR_NAME"),
			Engines:   blueprintEngines(),
			HostID:    blueprintHost(),
			Probe:     blueprint.ProbeLoopback,
			AppExists: blueprint.AppExists,
			Env:       os.Getenv,
		}
		if d.Agents != nil {
			o.RuntimeKnown = true
			for _, m := range d.Agents.List() {
				o.Runtime = append(o.Runtime, m.ID)
			}
		}
		if d.Board != nil {
			cctx, ccancel := context.WithTimeout(ctx, 12*time.Second)
			o.Connectors = d.Board.Statuses(cctx)
			ccancel()
		}
		g, err := blueprint.Compile(reg, o)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true, "graph": g})
	})
}

// blueprintEngines lists the engines workspaces are routed to on this
// machine, each with the workspaces whose home it is.
func blueprintEngines() []blueprint.EngineInfo {
	topo, err := topology.LoadRepo()
	if err != nil {
		return nil
	}
	var out []blueprint.EngineInfo
	for _, e := range TopologyEngines() {
		info := blueprint.EngineInfo{Name: e.Name, Tier: topo.Engines[e.Name].Tier, URL: strings.TrimRight(e.URL, "/")}
		for _, w := range topo.Workspaces {
			if w.Home == e.Name {
				info.Workspaces = append(info.Workspaces, w.Slug)
			}
		}
		out = append(out, info)
	}
	return out
}
