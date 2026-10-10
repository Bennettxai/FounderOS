package api

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/rhl/businessos-backend/internal/founderos/pages/org"
	"github.com/rhl/businessos-backend/internal/founderos/pages/osdata"
)

// /org (spec 6.5): the hierarchy board. The roster is the bridge's
// founderos_agents; the Paperclip board overlays live statuses by name and an
// unreachable board is reported, never faked. Runs and broadcasts go through
// the existing /agents/:id/run and /agents/broadcast routes (agents.go).
func init() { RegisterPage(registerOrg) }

func registerOrg(s *gin.RouterGroup, d *Deps) {
	s.GET("/pages/org", func(c *gin.Context) {
		ctx := c.Request.Context()
		st := storeFor(d)
		departments, err := st.Departments(ctx)
		if err != nil {
			dataErr(c, err)
			return
		}
		agents, err := st.Agents(ctx)
		if err != nil {
			dataErr(c, err)
			return
		}
		var last *osdata.Broadcast
		if list, err := st.Broadcasts(ctx, 1); err != nil {
			dataErr(c, err)
			return
		} else if len(list) > 0 {
			last = &list[0]
		}
		bctx, cancel := context.WithTimeout(ctx, 4*time.Second)
		defer cancel()
		live, boardErr := boardFor(d).Agents(bctx)
		c.JSON(http.StatusOK, org.BuildView(departments, agents, live, boardErr, last))
	})
	s.GET("/pages/departments", func(c *gin.Context) {
		departments, err := storeFor(d).Departments(c.Request.Context())
		if err != nil {
			dataErr(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"departments": departments})
	})
}
