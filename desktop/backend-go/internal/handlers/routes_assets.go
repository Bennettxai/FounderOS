package handlers

import (
	"log/slog"

	"github.com/gin-gonic/gin"

	"github.com/rhl/businessos-backend/internal/middleware"
)

// registerAssetsRoutes wires the per-workspace brand/media asset library.
func (h *Handlers) registerAssetsRoutes(api *gin.RouterGroup, auth gin.HandlerFunc) {
	ah := NewAssetsHandler(h.pool)
	g := api.Group("/assets")
	g.Use(auth, middleware.RequireAuth())
	{
		g.GET("", ah.ListAssets)
		g.POST("", ah.CreateAssetLink)
		g.POST("/upload", ah.UploadAsset)
		g.GET("/:id/raw", ah.ServeAssetRaw)
		g.PUT("/:id", ah.UpdateAsset)
		g.DELETE("/:id", ah.DeleteAsset)
	}
	slog.Info("Assets routes registered at /api/assets/*")
}
