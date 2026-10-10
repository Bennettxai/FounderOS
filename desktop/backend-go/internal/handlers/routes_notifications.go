package handlers

import (
	"log/slog"

	"github.com/gin-gonic/gin"
	"github.com/rhl/businessos-backend/internal/middleware"
)

// registerNotificationRoutes wires up notification, email, and dashboard-CRUD routes:
// /api/notifications, /api/dev/notifications (dev only), /api/email,
// /api/user-dashboards (Dashboard CRUD).
func (h *Handlers) registerNotificationRoutes(api *gin.RouterGroup, auth gin.HandlerFunc) {
	// Notification routes - /api/notifications
	if h.notificationService != nil {
		notifHandler := NewNotificationHandler(h.notificationService)
		notifications := api.Group("/notifications")
		notifications.Use(auth, middleware.RequireAuth())
		{
			notifications.GET("", notifHandler.ListNotifications)
			notifications.GET("/unread-count", notifHandler.GetUnreadCount)
			notifications.GET("/stream", notifHandler.Stream)
			notifications.GET("/preferences", notifHandler.GetPreferences)
			notifications.PUT("/preferences", notifHandler.UpdatePreferences)
			notifications.POST("/:id/read", notifHandler.MarkAsRead)
			notifications.POST("/read", notifHandler.MarkMultipleAsRead)
			notifications.POST("/read-all", notifHandler.MarkAllAsRead)
			notifications.DELETE("/:id", notifHandler.DeleteNotification)

			// Web Push routes
			if h.webPushService != nil {
				pushHandler := NewWebPushHandler(h.webPushService)
				notifications.GET("/push/vapid-public-key", pushHandler.GetVAPIDPublicKey)
				notifications.POST("/push/subscribe", pushHandler.Subscribe)
				notifications.POST("/push/unsubscribe", pushHandler.Unsubscribe)
				notifications.POST("/push/test", pushHandler.TestPush)
			} else {
				slog.Info("Web push notification routes skipped: webPushService not initialized")
			}
		}

	} else {
		slog.Info("Notification routes skipped: notificationService not initialized")
	}

	// Email routes - /api/email
	emailHandler := NewEmailHandler()
	email := api.Group("/email")
	email.Use(auth, middleware.RequireAuth())
	{
		email.GET("/status", emailHandler.GetEmailStatus)
		email.POST("/test", emailHandler.HandleTestEmail)
		email.POST("/send/verification", emailHandler.HandleSendVerificationEmail)
		email.POST("/send/password-reset", emailHandler.HandleSendPasswordResetEmail)
		email.POST("/send/welcome", emailHandler.HandleSendWelcomeEmail)
	}

	// Dashboard CRUD routes (user-dashboards, widgets, dashboard-templates)
	RegisterDashboardCRUDRoutes(api, NewDashboardCRUDHandler(h.pool, h.notificationService), auth)
}
