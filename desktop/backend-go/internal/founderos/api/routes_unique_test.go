package api

import (
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/rhl/businessos-backend/internal/founderos/connectors"
)

// Parallel page ports register routes from init(); two claiming the same
// path would panic the server at start. Mounting everything once catches it.
func TestAllRegisteredRoutesMountWithoutConflicts(t *testing.T) {
	gin.SetMode(gin.TestMode)
	defer func() {
		if p := recover(); p != nil {
			t.Fatalf("route conflict: %v", p)
		}
	}()
	r := gin.New()
	Register(r.Group("/api"), &Deps{Board: connectors.NewRegistry()}, func(c *gin.Context) { c.Next() })
	if len(r.Routes()) < 20 {
		t.Fatalf("only %d routes mounted", len(r.Routes()))
	}
}
