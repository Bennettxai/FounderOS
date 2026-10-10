package api

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rhl/businessos-backend/internal/founderos/pages/reference"
	"github.com/rhl/businessos-backend/internal/founderos/pages/roadmap"
)

// /os/roadmap and /os/reference (FounderOS v1 app/roadmap, app/reference):
// the build plan board and the reference model's domains, from the FounderOS
// workspace (migration 165). PATCH /pages/roadmap is v1 PATCH /api/roadmap: a
// local status change on the board, not an outbound write, so not guarded.
func init() { RegisterPage(registerRoadmap) }

func roadmapErr(c *gin.Context, err error) {
	switch {
	case errors.Is(err, roadmap.ErrNoWorkspace), errors.Is(err, reference.ErrNoWorkspace):
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
	case errors.Is(err, roadmap.ErrBadStatus):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	case errors.Is(err, roadmap.ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}
}

func registerRoadmap(s *gin.RouterGroup, d *Deps) {
	noPool := func(c *gin.Context, what string) bool {
		if d.Pool == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": what + " unavailable: no Postgres"})
			return true
		}
		return false
	}
	s.GET("/pages/roadmap", func(c *gin.Context) {
		if noPool(c, "roadmap") {
			return
		}
		page, err := roadmap.Load(c.Request.Context(), d.Pool, "founderos")
		if err != nil {
			roadmapErr(c, err)
			return
		}
		c.JSON(http.StatusOK, page)
	})
	s.PATCH("/pages/roadmap", func(c *gin.Context) {
		if noPool(c, "roadmap") {
			return
		}
		var in struct {
			ID     string `json:"id"`
			Status string `json:"status"`
		}
		if err := c.ShouldBindJSON(&in); err != nil {
			roadmapErr(c, roadmap.ErrBadStatus)
			return
		}
		ctx := c.Request.Context()
		item, err := roadmap.SetStatus(ctx, d.Pool, "founderos", in.ID, in.Status)
		if err != nil {
			roadmapErr(c, err)
			return
		}
		board, err := roadmap.Load(ctx, d.Pool, "founderos")
		if err != nil {
			roadmapErr(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"item": item, "board": board})
	})
	s.GET("/pages/reference", func(c *gin.Context) {
		if noPool(c, "reference model") {
			return
		}
		domains, err := reference.Load(c.Request.Context(), d.Pool, "founderos")
		if err != nil {
			roadmapErr(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"domains": domains})
	})
}
