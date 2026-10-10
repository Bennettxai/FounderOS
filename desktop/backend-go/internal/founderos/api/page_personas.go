package api

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rhl/businessos-backend/internal/founderos/pages/personas"
)

// /os/personas (spec 6.19): the operator variants, from founderos_personas in
// the FounderOS workspace (table map #7).
func init() { RegisterPage(registerPersonas) }

func registerPersonas(s *gin.RouterGroup, d *Deps) {
	s.GET("/pages/personas", func(c *gin.Context) {
		if d.Pool == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "personas unavailable: no Postgres"})
			return
		}
		list, err := personas.Load(c.Request.Context(), d.Pool, "founderos")
		if errors.Is(err, personas.ErrNoWorkspace) {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
			return
		}
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"personas": list})
	})
}
