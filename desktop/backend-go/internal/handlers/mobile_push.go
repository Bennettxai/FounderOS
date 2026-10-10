package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rhl/businessos-backend/internal/database/sqlc"
	"github.com/rhl/businessos-backend/internal/middleware"
)

// RegisterPushDevice registers or updates a device push token for the current user.
func (h *MobileHandler) RegisterPushDevice(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		MobileRespondUnauthorized(c)
		return
	}

	var req MobilePushRegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		MobileRespondError(c, http.StatusBadRequest, ErrCodeValidation, "Invalid request")
		return
	}

	// Prefer header-supplied device ID; fall back to body field.
	deviceID := middleware.GetDeviceID(c)
	if deviceID == "" {
		deviceID = req.DeviceID
	}

	_, err := h.queries.RegisterPushDevice(c.Request.Context(), sqlc.RegisterPushDeviceParams{
		UserID:      user.ID,
		DeviceID:    deviceID,
		Platform:    req.Platform,
		PushToken:   req.PushToken,
		AppVersion:  &req.AppVersion,
		OsVersion:   &req.OsVersion,
		DeviceModel: &req.DeviceModel,
	})
	if err != nil {
		MobileRespondError(c, http.StatusInternalServerError, ErrCodeInternal, "Failed to register device")
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "registered"})
}

// UnregisterPushDevice removes a device push token so it stops receiving notifications.
func (h *MobileHandler) UnregisterPushDevice(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		MobileRespondUnauthorized(c)
		return
	}

	// Prefer header-supplied device ID; fall back to query param.
	deviceID := middleware.GetDeviceID(c)
	if deviceID == "" {
		deviceID = c.Query("device_id")
	}

	if deviceID == "" {
		MobileRespondError(c, http.StatusBadRequest, ErrCodeValidation, "Device ID required")
		return
	}

	err := h.queries.UnregisterPushDevice(c.Request.Context(), sqlc.UnregisterPushDeviceParams{
		UserID:   user.ID,
		DeviceID: deviceID,
	})
	if err != nil {
		MobileRespondError(c, http.StatusInternalServerError, ErrCodeInternal, "Failed to unregister")
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "unregistered"})
}
