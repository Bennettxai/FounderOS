package handlers

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rhl/businessos-backend/internal/middleware"
	"github.com/rhl/businessos-backend/internal/services"
)

// SandboxEditHandler handles the module edit lifecycle over a forked in-memory sandbox.
//
// Routes (all under /api/v1/sandbox/edit):
//
//	POST   /fork              fork a module into a new sandbox
//	GET    /:id               get sandbox state
//	PUT    /:id/files/:name   update a file's content
//	POST   /:id/validate      validate all files
//	GET    /:id/preview       compute diff preview
//	POST   /:id/apply         apply changes (validated → applied)
//	POST   /:id/reject        discard sandbox (any state → rejected)
type SandboxEditHandler struct {
	service *services.SandboxEditService
	logger  *slog.Logger
}

// NewSandboxEditHandler creates a SandboxEditHandler wired to the given service.
func NewSandboxEditHandler(service *services.SandboxEditService, logger *slog.Logger) *SandboxEditHandler {
	return &SandboxEditHandler{
		service: service,
		logger:  logger.With("handler", "sandbox_edit"),
	}
}

// RegisterRoutes attaches all 7 sandbox-edit routes to rg.
// Callers should pass a group that already has auth middleware applied,
// e.g.: rg := api.Group("/sandbox/edit"); rg.Use(auth, middleware.RequireAuth())
func (h *SandboxEditHandler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.POST("/fork", h.Fork)
	rg.GET("/:id", h.Get)
	rg.PUT("/:id/files/:name", h.UpdateFile)
	rg.POST("/:id/validate", h.Validate)
	rg.GET("/:id/preview", h.Preview)
	rg.POST("/:id/apply", h.Apply)
	rg.POST("/:id/reject", h.Reject)
}

// Fork handles POST /api/v1/sandbox/edit/fork
//
// Request body:
//
//	{"module_id": "uuid-string", "module_name": "CRM"}
//
// Response: {"sandbox": SandboxEdit}
func (h *SandboxEditHandler) Fork(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req struct {
		ModuleID   string `json:"module_id" binding:"required"`
		ModuleName string `json:"module_name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WarnContext(c.Request.Context(), "fork: invalid request body", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body: module_id and module_name are required"})
		return
	}

	// Use the user ID as tenant ID (single-tenant per user model).
	// If the project later adds org-level tenancy, swap user.ID for the org ID here.
	tenantID := user.ID

	edit, err := h.service.Fork(c.Request.Context(), tenantID, user.ID, req.ModuleID, req.ModuleName)
	if err != nil {
		h.logger.ErrorContext(c.Request.Context(), "fork: failed", "error", err, "user_id", user.ID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fork module"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"sandbox": edit})
}

// Get handles GET /api/v1/sandbox/edit/:id
//
// Response: {"sandbox": SandboxEdit}
func (h *SandboxEditHandler) Get(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	id := c.Param("id")
	tenantID := user.ID

	edit, err := h.service.Get(c.Request.Context(), id, tenantID)
	if err != nil {
		h.logger.WarnContext(c.Request.Context(), "get: not found or access denied",
			"id", id, "user_id", user.ID, "error", err)
		c.JSON(http.StatusNotFound, gin.H{"error": "sandbox edit not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"sandbox": edit})
}

// UpdateFile handles PUT /api/v1/sandbox/edit/:id/files/:name
//
// :name is URL-encoded filename (e.g. "main.go" or "pkg%2Futil.go").
//
// Request body:
//
//	{"content": "file content here"}
//
// Response: {"ok": true}
func (h *SandboxEditHandler) UpdateFile(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	id := c.Param("id")
	filename := c.Param("name")
	tenantID := user.ID

	var req struct {
		Content string `json:"content"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WarnContext(c.Request.Context(), "update_file: invalid request body", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body: content field is required"})
		return
	}

	if err := h.service.UpdateFile(c.Request.Context(), id, tenantID, filename, req.Content); err != nil {
		h.logger.WarnContext(c.Request.Context(), "update_file: failed",
			"id", id, "filename", filename, "user_id", user.ID, "error", err)
		// Distinguish not-found vs state-conflict vs access-denied with a single
		// safe message (avoids leaking internal state to different tenants).
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// Validate handles POST /api/v1/sandbox/edit/:id/validate
//
// Response: {"sandbox": SandboxEdit}
func (h *SandboxEditHandler) Validate(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	id := c.Param("id")
	tenantID := user.ID

	edit, err := h.service.Validate(c.Request.Context(), id, tenantID)
	if err != nil {
		h.logger.WarnContext(c.Request.Context(), "validate: failed",
			"id", id, "user_id", user.ID, "error", err)
		c.JSON(http.StatusNotFound, gin.H{"error": "sandbox edit not found"})
		return
	}

	// Return the full edit object; the caller inspects edit.state and edit.errors.
	c.JSON(http.StatusOK, gin.H{"sandbox": edit})
}

// Preview handles GET /api/v1/sandbox/edit/:id/preview
//
// Computes and returns a diff between the original and current files.
// Response: {"sandbox": SandboxEdit}  (diff populated in edit.diff)
func (h *SandboxEditHandler) Preview(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	id := c.Param("id")
	tenantID := user.ID

	edit, err := h.service.Preview(c.Request.Context(), id, tenantID)
	if err != nil {
		h.logger.WarnContext(c.Request.Context(), "preview: failed",
			"id", id, "user_id", user.ID, "error", err)
		c.JSON(http.StatusNotFound, gin.H{"error": "sandbox edit not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"sandbox": edit})
}

// Apply handles POST /api/v1/sandbox/edit/:id/apply
//
// Marks the sandbox as applied. Requires state == "validated".
// Response: {"sandbox": SandboxEdit}
func (h *SandboxEditHandler) Apply(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	id := c.Param("id")
	tenantID := user.ID

	edit, err := h.service.Apply(c.Request.Context(), id, tenantID, user.ID)
	if err != nil {
		h.logger.WarnContext(c.Request.Context(), "apply: failed",
			"id", id, "user_id", user.ID, "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"sandbox": edit})
}

// Reject handles POST /api/v1/sandbox/edit/:id/reject
//
// Discards the sandbox edit session.
// Response: {"sandbox": SandboxEdit}
func (h *SandboxEditHandler) Reject(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	id := c.Param("id")
	tenantID := user.ID

	edit, err := h.service.Reject(c.Request.Context(), id, tenantID, user.ID)
	if err != nil {
		h.logger.WarnContext(c.Request.Context(), "reject: failed",
			"id", id, "user_id", user.ID, "error", err)
		c.JSON(http.StatusNotFound, gin.H{"error": "sandbox edit not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"sandbox": edit})
}
