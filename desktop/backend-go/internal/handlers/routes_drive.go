package handlers

import (
	"log/slog"

	"github.com/gin-gonic/gin"

	"github.com/rhl/businessos-backend/internal/middleware"
)

// registerDriveRoutes wires the per-workspace file store (files + folders).
func (h *Handlers) registerDriveRoutes(api *gin.RouterGroup, auth gin.HandlerFunc) {
	dh := NewDriveHandler(h.pool)
	g := api.Group("/drive")
	g.Use(auth, middleware.RequireAuth())
	{
		g.GET("", dh.ListDriveFiles)
		g.GET("/folders", dh.ListDriveFolders)
		g.POST("/upload", dh.UploadDriveFile)
		g.GET("/:id/raw", dh.ServeDriveRaw)
		g.PUT("/:id", dh.UpdateDriveFile)
		g.DELETE("/:id", dh.DeleteDriveFile)
	}
	slog.Info("Drive routes registered at /api/drive/*")
}
