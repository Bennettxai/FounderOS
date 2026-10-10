package handlers

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rhl/businessos-backend/internal/middleware"
	"github.com/rhl/businessos-backend/internal/utils"
)

// ProfileHandler handles profile and account management endpoints.
type ProfileHandler struct {
	pool         *pgxpool.Pool
	sessionCache *middleware.SessionCache
}

// NewProfileHandler constructs a ProfileHandler.
func NewProfileHandler(pool *pgxpool.Pool, sessionCache *middleware.SessionCache) *ProfileHandler {
	return &ProfileHandler{pool: pool, sessionCache: sessionCache}
}

// RegisterProfileRoutes wires /api/profile and /api/account routes.
func RegisterProfileRoutes(api *gin.RouterGroup, h *ProfileHandler, auth gin.HandlerFunc) {
	// Profile routes - /api/profile
	profile := api.Group("/profile")
	profile.Use(auth, middleware.RequireAuth())
	{
		profile.PUT("", h.UpdateProfile)
		profile.POST("/photo", h.UploadProfilePhoto)
		profile.DELETE("/photo", h.DeleteProfilePhoto)
		profile.POST("/background", h.UploadBackground)
		profile.GET("/background/:filename", h.GetBackground)
		profile.DELETE("/background", h.DeleteBackground)
	}

	// Account management routes - /api/account (GDPR compliance)
	account := api.Group("/account")
	account.Use(auth, middleware.RequireAuth())
	{
		// DELETE /api/account — soft-delete account (GDPR Article 17, Right to Erasure)
		account.DELETE("", h.DeleteAccount)
		// GET /api/account/export — export all user data (GDPR Article 20, Right to Data Portability)
		account.GET("/export", middleware.StrictRateLimitMiddleware(), middleware.AuditSensitiveAccess("account_export"), h.ExportAccountData)
	}
}

// UploadProfilePhoto handles profile photo upload
func (h *ProfileHandler) UploadProfilePhoto(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	// Parse multipart form with 5MB max for profile photos
	if err := c.Request.ParseMultipartForm(5 << 20); err != nil {
		utils.RespondBadRequest(c, slog.Default(), "File too large (max 5MB)")
		return
	}

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		utils.RespondBadRequest(c, slog.Default(), "No file provided")
		return
	}
	defer file.Close()

	// Validate file size (double-check even after ParseMultipartForm)
	const maxFileSize = 5 << 20 // 5MB
	if header.Size > maxFileSize {
		utils.RespondBadRequest(c, slog.Default(), "File too large (max 5MB)")
		return
	}

	// Validate file extension first (quick check)
	ext := strings.ToLower(filepath.Ext(header.Filename))
	allowedExts := map[string]bool{
		".jpg":  true,
		".jpeg": true,
		".png":  true,
		".gif":  true,
		".webp": true,
	}
	if !allowedExts[ext] {
		utils.RespondBadRequest(c, slog.Default(), "Invalid file type. Allowed: jpg, jpeg, png, gif, webp")
		return
	}

	// Block dangerous extensions (defense in depth)
	dangerousExts := []string{
		".php", ".exe", ".sh", ".bat", ".cmd", ".com", ".pif", ".scr",
		".vbs", ".js", ".jar", ".app", ".deb", ".rpm", ".dmg", ".pkg",
		".html", ".htm", ".svg", // SVG can contain scripts
	}
	for _, dangerous := range dangerousExts {
		if ext == dangerous {
			utils.RespondBadRequest(c, slog.Default(), "Dangerous file type blocked")
			return
		}
	}

	// Read first 512 bytes to detect MIME type via magic bytes
	buffer := make([]byte, 512)
	n, err := file.Read(buffer)
	if err != nil && err != io.EOF {
		utils.RespondBadRequest(c, slog.Default(), "Failed to read file")
		return
	}

	// Detect MIME type from content
	contentType := http.DetectContentType(buffer[:n])

	// Validate MIME type matches allowed image types
	allowedMimeTypes := map[string]bool{
		"image/jpeg": true,
		"image/png":  true,
		"image/gif":  true,
		"image/webp": true,
	}
	if !allowedMimeTypes[contentType] {
		slog.Warn("file upload rejected due to invalid MIME type",
			"user_id", user.ID,
			"filename", header.Filename,
			"detected_mime", contentType,
		)
		utils.RespondBadRequest(c, slog.Default(), fmt.Sprintf("Invalid file content type: %s. Expected image format.", contentType))
		return
	}

	// Reset file pointer to beginning after reading magic bytes
	if _, err := file.Seek(0, 0); err != nil {
		utils.RespondInternalError(c, slog.Default(), "reset file pointer", err)
		return
	}

	// Create uploads directory if it doesn't exist
	uploadsDir := filepath.Join("uploads", "profiles", user.ID)
	if err := os.MkdirAll(uploadsDir, 0755); err != nil {
		utils.RespondInternalError(c, slog.Default(), "create upload directory", err)
		return
	}

	// Generate unique filename
	filename := fmt.Sprintf("avatar_%d%s", time.Now().Unix(), ext)
	filePath := filepath.Join(uploadsDir, filename)

	// Create destination file
	dst, err := os.Create(filePath)
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "save file", err)
		return
	}
	defer dst.Close()

	// Copy file content
	if _, err := io.Copy(dst, file); err != nil {
		utils.RespondInternalError(c, slog.Default(), "save file", err)
		return
	}

	// Build the file URL
	fileURL := fmt.Sprintf("/uploads/profiles/%s/%s", user.ID, filename)

	// Update the user's image in Better Auth user table
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	_, err = h.pool.Exec(ctx, `UPDATE "user" SET image = $1, "updatedAt" = NOW() WHERE id = $2`, fileURL, user.ID)
	if err != nil {
		// Still return success since file was uploaded, just log the error
		slog.Warn("failed to update user image in database", "error", err)
	}

	c.JSON(http.StatusOK, gin.H{
		"url":      fileURL,
		"filename": filename,
		"message":  "Profile photo updated",
	})
}

// GetProfilePhoto serves a profile photo file
func (h *ProfileHandler) GetProfilePhoto(c *gin.Context) {
	userID := c.Param("user_id")
	filename := c.Param("filename")

	if userID == "" || filename == "" {
		utils.RespondBadRequest(c, slog.Default(), "User ID and filename required")
		return
	}

	// Sanitize to prevent directory traversal
	userID = filepath.Base(userID)
	filename = filepath.Base(filename)
	filePath := filepath.Join("uploads", "profiles", userID, filename)

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		utils.RespondNotFound(c, slog.Default(), "File")
		return
	}

	c.File(filePath)
}

// DeleteProfilePhoto removes the user's profile photo
func (h *ProfileHandler) DeleteProfilePhoto(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	// Remove profile photo directory
	profileDir := filepath.Join("uploads", "profiles", user.ID)
	if err := os.RemoveAll(profileDir); err != nil && !os.IsNotExist(err) {
		utils.RespondInternalError(c, slog.Default(), "delete profile photo", err)
		return
	}

	// Clear the image in Better Auth user table
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	_, err := h.pool.Exec(ctx, `UPDATE "user" SET image = NULL, "updatedAt" = NOW() WHERE id = $1`, user.ID)
	if err != nil {
		slog.Warn("failed to clear user image in database", "error", err)
	}

	c.JSON(http.StatusOK, gin.H{"message": "Profile photo deleted"})
}

// UpdateProfile updates the user's profile information
func (h *ProfileHandler) UpdateProfile(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	var req struct {
		Name string `json:"name"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondInvalidRequest(c, slog.Default(), err)
		return
	}

	// Update user name in Better Auth user table
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	_, err := h.pool.Exec(ctx, `UPDATE "user" SET name = $1, "updatedAt" = NOW() WHERE id = $2`, req.Name, user.ID)
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "update profile", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Profile updated",
		"name":    req.Name,
	})
}

// UploadBackground handles background image upload
func (h *ProfileHandler) UploadBackground(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	// Parse multipart form with 10MB max
	if err := c.Request.ParseMultipartForm(10 << 20); err != nil {
		utils.RespondBadRequest(c, slog.Default(), "File too large (max 10MB)")
		return
	}

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		utils.RespondBadRequest(c, slog.Default(), "No file provided")
		return
	}
	defer file.Close()

	// Validate file size (double-check even after ParseMultipartForm)
	const maxFileSize = 10 << 20 // 10MB
	if header.Size > maxFileSize {
		utils.RespondBadRequest(c, slog.Default(), "File too large (max 10MB)")
		return
	}

	// Validate file extension first (quick check)
	ext := strings.ToLower(filepath.Ext(header.Filename))
	allowedExts := map[string]bool{
		".jpg":  true,
		".jpeg": true,
		".png":  true,
		".gif":  true,
		".webp": true,
	}
	if !allowedExts[ext] {
		utils.RespondBadRequest(c, slog.Default(), "Invalid file type. Allowed: jpg, jpeg, png, gif, webp")
		return
	}

	// Block dangerous extensions (defense in depth)
	dangerousExts := []string{
		".php", ".exe", ".sh", ".bat", ".cmd", ".com", ".pif", ".scr",
		".vbs", ".js", ".jar", ".app", ".deb", ".rpm", ".dmg", ".pkg",
		".html", ".htm", ".svg", // SVG can contain scripts
	}
	for _, dangerous := range dangerousExts {
		if ext == dangerous {
			utils.RespondBadRequest(c, slog.Default(), "Dangerous file type blocked")
			return
		}
	}

	// Read first 512 bytes to detect MIME type via magic bytes
	buffer := make([]byte, 512)
	n, err := file.Read(buffer)
	if err != nil && err != io.EOF {
		utils.RespondBadRequest(c, slog.Default(), "Failed to read file")
		return
	}

	// Detect MIME type from content
	contentType := http.DetectContentType(buffer[:n])

	// Validate MIME type matches allowed image types
	allowedMimeTypes := map[string]bool{
		"image/jpeg": true,
		"image/png":  true,
		"image/gif":  true,
		"image/webp": true,
	}
	if !allowedMimeTypes[contentType] {
		slog.Warn("background upload rejected due to invalid MIME type",
			"user_id", user.ID,
			"filename", header.Filename,
			"detected_mime", contentType,
		)
		utils.RespondBadRequest(c, slog.Default(), fmt.Sprintf("Invalid file content type: %s. Expected image format.", contentType))
		return
	}

	// Reset file pointer to beginning after reading magic bytes
	if _, err := file.Seek(0, 0); err != nil {
		utils.RespondInternalError(c, slog.Default(), "reset file pointer", err)
		return
	}

	// Create uploads directory if it doesn't exist
	uploadsDir := filepath.Join("uploads", "backgrounds", user.ID)
	if err := os.MkdirAll(uploadsDir, 0755); err != nil {
		utils.RespondInternalError(c, slog.Default(), "create upload directory", err)
		return
	}

	// Generate unique filename
	filename := fmt.Sprintf("%s_%d%s", uuid.New().String(), time.Now().Unix(), ext)
	filePath := filepath.Join(uploadsDir, filename)

	// Create destination file
	dst, err := os.Create(filePath)
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "save file", err)
		return
	}
	defer dst.Close()

	// Copy file content
	if _, err := io.Copy(dst, file); err != nil {
		utils.RespondInternalError(c, slog.Default(), "save file", err)
		return
	}

	// Return the file path/URL
	// In production, this would be a CDN URL or similar
	fileURL := fmt.Sprintf("/uploads/backgrounds/%s/%s", user.ID, filename)

	c.JSON(http.StatusOK, gin.H{
		"url":      fileURL,
		"filename": filename,
	})
}

// GetBackground serves a background image file
func (h *ProfileHandler) GetBackground(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	filename := c.Param("filename")
	if filename == "" {
		utils.RespondBadRequest(c, slog.Default(), "Filename required")
		return
	}

	// Sanitize filename to prevent directory traversal
	filename = filepath.Base(filename)
	filePath := filepath.Join("uploads", "backgrounds", user.ID, filename)

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		utils.RespondNotFound(c, slog.Default(), "File")
		return
	}

	c.File(filePath)
}

// DeleteBackground deletes the user's background image
func (h *ProfileHandler) DeleteBackground(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	// Get background directory
	backgroundDir := filepath.Join("uploads", "backgrounds", user.ID)

	// Remove entire directory and contents
	if err := os.RemoveAll(backgroundDir); err != nil && !os.IsNotExist(err) {
		utils.RespondInternalError(c, slog.Default(), "delete background", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Background deleted"})
}
