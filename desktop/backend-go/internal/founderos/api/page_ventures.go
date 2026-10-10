package api

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/rhl/businessos-backend/internal/founderos/pages/ventures"
)

// FounderOS v1 GET /api/ventures.
func init() {
	RegisterPage(func(s *gin.RouterGroup, d *Deps) {
		s.GET("/pages/ventures", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"ventures": ventures.All})
		})
	})
}
