package handlers

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	founderosapi "github.com/rhl/businessos-backend/internal/founderos/api"
	"github.com/rhl/businessos-backend/internal/middleware"
)

// RegisterRoutes runs once for /api and again for /api/v1. The operator's Deps
// (and with it the agent runtime and the cron scheduler) must exist once per
// process.
var (
	founderosOnce sync.Once
	founderosDeps *founderosapi.Deps
)

// registerFounderosRoutes mounts the operator's surface (/api/founderos/*): the
// Connections board, bridge guard state, and device collector pushes.
func (h *Handlers) registerFounderosRoutes(api *gin.RouterGroup, auth gin.HandlerFunc) {
	founderosOnce.Do(func() {
		founderosDeps = founderosapi.NewDeps(h.pool)
		// Prime the comms caches a few seconds after boot (FounderOS v1
		// instrumentation.ts). Never under go test: this Mac's resolver finds
		// real credentials, and a test must not open real inboxes.
		if !testing.Testing() {
			founderosapi.WarmOnBoot(context.Background(), founderosDeps, 4*time.Second)
		}
	})
	// Owner-only: any other signed-up BusinessOS user must not see the operator's data.
	founderosapi.Register(api, founderosDeps, auth, middleware.RequireAuth(), founderosapi.OwnerOnly(founderosDeps))
}
