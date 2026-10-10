package handlers

import (
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rhl/businessos-backend/internal/config"
	"github.com/rhl/businessos-backend/internal/integrations/miosa"
)

// Community edition stub. The platform-admin control plane (superadmin user
// management, role changes, audit, platform-wide settings) is part of the hosted
// edition and is not shipped in the open-source mirror. This stub keeps the
// build intact while registering no platform-admin routes.

type PlatformAdminHandler struct{}

type PlatformAdminControlHandler struct{}

func NewPlatformAdminHandler(_ *pgxpool.Pool, _ *config.Config, _ *miosa.ComputeClient) *PlatformAdminHandler {
	return &PlatformAdminHandler{}
}

func RegisterPlatformAdminRoutes(_ *gin.RouterGroup, _ *PlatformAdminHandler, _ gin.HandlerFunc) {}

func NewPlatformAdminControlHandler(_ *pgxpool.Pool) *PlatformAdminControlHandler {
	return &PlatformAdminControlHandler{}
}

func RegisterPlatformAdminControlRoutes(_ *gin.RouterGroup, _ *PlatformAdminControlHandler, _ gin.HandlerFunc) {
}
