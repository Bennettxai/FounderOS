package handlers

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rhl/businessos-backend/internal/database/sqlc"
	"github.com/rhl/businessos-backend/internal/middleware"
	"github.com/rhl/businessos-backend/internal/services"
)

// MobileHandler implements the Mobile API (/api/mobile/v1).
// Uses lean payloads, cursor pagination, and Unix timestamps.
type MobileHandler struct {
	pool                *pgxpool.Pool
	queries             *sqlc.Queries
	notificationService *services.NotificationService
}

// NewMobileHandler creates a new mobile handler instance.
func NewMobileHandler(pool *pgxpool.Pool, notificationService *services.NotificationService) *MobileHandler {
	return &MobileHandler{
		pool:                pool,
		queries:             sqlc.New(pool),
		notificationService: notificationService,
	}
}

// GetMe returns the authenticated user's profile and preferences.
func (h *MobileHandler) GetMe(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		MobileRespondUnauthorized(c)
		return
	}

	userResp := MobileUserResponse{
		ID:    user.ID,
		Email: user.Email,
		Name:  user.Name,
	}

	if user.Image != nil && *user.Image != "" {
		userResp.AvatarURL = user.Image
	}

	timezone := "UTC"
	prefs := &MobilePreferencesResponse{
		NotificationsEnabled: true,
	}

	notifPrefs, err := h.queries.GetNotificationPreferencesByUser(c.Request.Context(), user.ID)
	if err == nil {
		if notifPrefs.QuietHoursTimezone != nil && *notifPrefs.QuietHoursTimezone != "" {
			timezone = *notifPrefs.QuietHoursTimezone
		}

		if notifPrefs.InAppEnabled != nil {
			prefs.NotificationsEnabled = *notifPrefs.InAppEnabled
		}

		if notifPrefs.QuietHoursEnabled != nil && *notifPrefs.QuietHoursEnabled {
			if notifPrefs.QuietHoursStart.Valid {
				startSec := notifPrefs.QuietHoursStart.Microseconds / 1_000_000
				s := fmt.Sprintf("%02d:%02d", startSec/3600, (startSec%3600)/60)
				prefs.QuietHoursStart = &s
			}
			if notifPrefs.QuietHoursEnd.Valid {
				endSec := notifPrefs.QuietHoursEnd.Microseconds / 1_000_000
				s := fmt.Sprintf("%02d:%02d", endSec/3600, (endSec%3600)/60)
				prefs.QuietHoursEnd = &s
			}
		}
	}

	userResp.Timezone = timezone

	c.JSON(http.StatusOK, MobileMeResponse{
		User:        userResp,
		Workspace:   nil, // Single-tenant for now
		Preferences: prefs,
	})
}

// verifyTaskOwnership confirms a task belongs to the given user.
func (h *MobileHandler) verifyTaskOwnership(ctx context.Context, taskID uuid.UUID, userID string) bool {
	_, err := h.queries.GetTask(ctx, sqlc.GetTaskParams{
		ID:     pgtype.UUID{Bytes: taskID, Valid: true},
		UserID: userID,
	})
	return err == nil
}
