package api

// POST /pages/analytics/vsl/association (FounderOS v1
// app/api/analytics/vsl/association): which saved Vidalytics video the
// Instagram funnel uses. Bridge data in Postgres (founderos_seed_meta), not an
// outbound side effect, so FOUNDEROS_WRITES does not gate it. The session group
// carries BusinessOS's CSRF check in place of FounderOS v1's same-origin test.

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/rhl/businessos-backend/internal/founderos/pages/vsl"
)

func init() { RegisterPage(registerVSL) }

func registerVSL(s *gin.RouterGroup, d *Deps) {
	s.POST("/pages/analytics/vsl/association", vslAssociation(d))
}

func vslAssociation(d *Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		if d.Pool == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "founderos Postgres is not wired"})
			return
		}
		raw, _ := io.ReadAll(io.LimitReader(c.Request.Body, 4<<10))
		var body map[string]json.RawMessage
		var v json.RawMessage
		ok := json.Unmarshal(raw, &body) == nil
		if ok {
			v, ok = body["videoId"]
		}
		if !ok {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Choose an imported video."})
			return
		}
		var id *string
		if string(v) != "null" {
			var s string
			if json.Unmarshal(v, &s) != nil || len(s) < 1 || len(s) > 100 {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Choose an imported video."})
				return
			}
			id = &s
		}
		ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
		defer cancel()
		store := &vsl.Store{Pool: d.Pool, Workspaces: funnelWorkspaceIDs(ctx, d.Pool)}
		set := ""
		if id != nil {
			set = *id
		}
		if err := store.SetIGVideo(ctx, set); err != nil {
			if errors.Is(err, vsl.ErrUnknownVideo) {
				c.JSON(http.StatusNotFound, gin.H{"error": "Video not found."})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		got, err := store.IGVideo(ctx)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"videoId": got})
	}
}
