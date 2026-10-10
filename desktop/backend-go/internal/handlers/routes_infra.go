package handlers

import (
	"log/slog"

	"github.com/gin-gonic/gin"
	"github.com/rhl/businessos-backend/internal/middleware"
)

// registerInfraRoutes wires up infrastructure and operational routes:
// /api/terminal, /api/filesystem, /api/sync, /api/mobile/v1,
// /api/calendar (stats), /api/analytics, /api/comments, /api/usage.
func (h *Handlers) registerInfraRoutes(api *gin.RouterGroup, auth gin.HandlerFunc) {
	// Terminal routes - /api/terminal
	terminalHandler := NewTerminalHandler(h.containerMgr, h.terminalPubSub)
	terminalRoutes := api.Group("/terminal")
	terminalRoutes.Use(auth, middleware.RequireAuth())
	{
		terminalRoutes.GET("/ws", terminalHandler.HandleWebSocket)
		terminalRoutes.GET("/sessions", terminalHandler.ListSessions)
		terminalRoutes.DELETE("/sessions/:id", terminalHandler.CloseSession)
		terminalRoutes.GET("/sessions/:id/changes", terminalHandler.GetSandboxChanges)
	}

	// Filesystem routes - /api/filesystem (require auth — filesystem access is sensitive)
	fsH := NewFilesystemHandler(h.containerMgr)
	filesystem := api.Group("/filesystem")
	filesystem.Use(auth, middleware.RequireAuth())
	{
		filesystem.GET("/list", fsH.ListDirectory)
		filesystem.GET("/read", fsH.ReadFile)
		filesystem.GET("/download", fsH.DownloadFile)
		filesystem.GET("/info", fsH.GetFileInfo)
		filesystem.GET("/quick-access", fsH.GetQuickAccessPaths)
		filesystem.POST("/mkdir", fsH.CreateDirectory)
		filesystem.POST("/upload", fsH.UploadFile)
		filesystem.DELETE("/delete", fsH.DeleteFileOrDir)
	}

	// Sync routes — replaced by cloud_sync.go (registered in routes_sync.go)
	// The old SyncHandler is superseded by CloudSyncHandler which handles
	// bidirectional local ↔ cloud sync with MIOSA integration.
	// RegisterSyncRoutes(api, NewSyncHandler(h.pool), auth)

	// =============================================================================
	// MOBILE API - /api/mobile/v1
	// Optimized endpoints for mobile clients (PWA, native apps):
	// lean payloads, cursor-based pagination, field selection, unix timestamps.
	// =============================================================================
	mobileHandler := NewMobileHandler(h.pool, h.notificationService)
	mobile := api.Group("/mobile/v1")
	mobile.Use(auth, middleware.RequireAuth())
	mobile.Use(middleware.DeviceIDMiddleware()) // Extract X-Device-ID header
	{
		// User profile
		mobile.GET("/me", mobileHandler.GetMe)

		// Tasks (Phase 1)
		mobile.GET("/tasks", mobileHandler.ListTasks)
		mobile.GET("/tasks/:id", mobileHandler.GetTask)
		mobile.POST("/tasks/quick", mobileHandler.QuickCreateTask)
		mobile.PUT("/tasks/:id/status", mobileHandler.UpdateTaskStatus)
		mobile.PUT("/tasks/:id/toggle", mobileHandler.ToggleTask)

		// Notifications (Phase 2)
		mobile.GET("/notifications", mobileHandler.ListNotifications)
		mobile.GET("/notifications/count", mobileHandler.GetNotificationCount)
		mobile.POST("/notifications/mark-read", mobileHandler.MarkNotificationsRead)

		// Daily Log (Phase 3)
		mobile.GET("/dailylog/today", mobileHandler.GetTodayLog)
		mobile.GET("/dailylog/history", mobileHandler.GetLogHistory)

		// Sync (Phase 3)
		mobile.GET("/sync", mobileHandler.DeltaSync)

		// Chat (Phase 4)
		mobile.GET("/chat/threads", mobileHandler.ListChatThreads)
		mobile.GET("/chat/history/:id", mobileHandler.GetChatHistory)
		mobile.POST("/chat/message", mobileHandler.SendChatMessage)

		// Push Registration (Phase 5)
		mobile.POST("/push/register", mobileHandler.RegisterPushDevice)
		mobile.DELETE("/push/unregister", mobileHandler.UnregisterPushDevice)
	}

	// Calendar routes - /api/calendar (aggregate stats from calendar_events table)
	RegisterCalendarStatsRoutes(api, NewCalendarStatsHandler(h.pool), auth)

	// Analytics routes - /api/analytics
	RegisterAnalyticsRoutes(api, NewAnalyticsHandler(h.pool), auth)

	// Knowledge routes - /api/knowledge (workspace markdown tree + file bodies)
	RegisterKnowledgeRoutes(api, NewKnowledgeHandler(h.pool), auth)

	// Comments routes - /api/comments
	if h.commentService != nil {
		RegisterCommentRoutes(api, NewCommentHandler(h.commentService), auth)
	} else {
		slog.Info("Comment routes skipped: commentService not initialized")
	}

	// Usage routes - /api/usage
	RegisterUsageRoutes(api, NewUsageHandler(h.pool), auth)
}
