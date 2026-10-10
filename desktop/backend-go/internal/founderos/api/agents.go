package api

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/rhl/businessos-backend/internal/founderos/agents"
)

func registerAgents(s *gin.RouterGroup, d *Deps) {
	need := func(c *gin.Context) bool {
		if d.Agents == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "agent runtime unavailable: run founderos-bootstrap so the FounderOS workspace exists"})
			return false
		}
		return true
	}
	s.GET("/agents", func(c *gin.Context) {
		if !need(c) {
			return
		}
		type row struct {
			agents.Meta
			LastRun *agents.Run `json:"lastRun"`
		}
		var out []row
		for _, m := range d.Agents.List() {
			runs, err := d.Agents.Recent(c.Request.Context(), m.ID, 1)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			r := row{Meta: m}
			if len(runs) > 0 {
				r.LastRun = &runs[0]
			}
			out = append(out, r)
		}
		c.JSON(http.StatusOK, gin.H{"agents": out})
	})
	s.GET("/agents/:id/runs", func(c *gin.Context) {
		if !need(c) {
			return
		}
		runs, err := d.Agents.Recent(c.Request.Context(), c.Param("id"), 50)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"runs": runs})
	})
	s.POST("/agents/broadcast", func(c *gin.Context) {
		if !need(c) {
			return
		}
		var body struct {
			Message string `json:"message"`
		}
		if err := c.ShouldBindJSON(&body); err != nil || strings.TrimSpace(body.Message) == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "message is required"})
			return
		}
		b, err := d.Agents.Broadcast(c.Request.Context(), strings.TrimSpace(body.Message))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, b)
	})
	s.POST("/agents/:id/run", func(c *gin.Context) {
		if !need(c) {
			return
		}
		run, err := d.Agents.Run(c.Request.Context(), c.Param("id"))
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
}
