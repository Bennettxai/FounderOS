package linear

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	integrations "github.com/rhl/businessos-backend/internal/integrations"
)

// Handler provides HTTP handlers for Linear integration routes.
type Handler struct {
	provider *Provider
}

// NewHandler creates a new Linear integration handler.
func NewHandler(provider *Provider) *Handler {
	return &Handler{
		provider: provider,
	}
}

// RegisterRoutes registers all Linear integration routes.
func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	// OAuth routes
	r.GET("/auth", h.GetAuthURL)
	r.GET("/callback", h.HandleCallback)
	r.POST("/disconnect", h.Disconnect)
	r.GET("/status", h.GetStatus)

	// Issue routes
	issues := r.Group("/issues")
	{
		issues.GET("", h.GetIssues)
		issues.GET("/:id", h.GetIssue)
		issues.POST("", h.CreateIssue)
		issues.PUT("/:id", h.UpdateIssue)
		issues.POST("/sync", h.SyncIssues)
	}

	// Project routes
	projects := r.Group("/projects")
	{
		projects.GET("", h.GetProjects)
		projects.GET("/:id", h.GetProject)
		projects.POST("/sync", h.SyncProjects)
	}

	// Team routes
	teams := r.Group("/teams")
	{
		teams.GET("", h.GetTeams)
		teams.POST("/sync", h.SyncTeams)
	}
}

// ============================================================================
// OAuth Handlers
// ============================================================================

// GetAuthURL returns the OAuth authorization URL.
func (h *Handler) GetAuthURL(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	state := integrations.GenerateUserState(userID)
	authURL := h.provider.GetAuthURL(state)

	c.JSON(http.StatusOK, gin.H{
		"auth_url": authURL,
	})
}

// HandleCallback handles the OAuth callback.
func (h *Handler) HandleCallback(c *gin.Context) {
	code := c.Query("code")
	state := c.Query("state")

	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing authorization code"})
		return
	}

	userID := integrations.ExtractUserIDFromState(state)
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid state"})
		return
	}

	token, err := h.provider.ExchangeCode(c.Request.Context(), code)
	if err != nil {
		slog.Info("Failed to exchange code", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to exchange code"})
		return
	}

	if err := h.provider.SaveToken(c.Request.Context(), userID, token); err != nil {
		slog.Info("Failed to save token", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":          true,
		"organization":     token.AccountName,
		"scopes":           token.Scopes,
	})
}

// Disconnect disconnects the Linear integration.
func (h *Handler) Disconnect(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	if err := h.provider.Disconnect(c.Request.Context(), userID); err != nil {
		slog.Info("Failed to disconnect", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to disconnect"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// GetStatus returns the connection status.
func (h *Handler) GetStatus(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	status, err := h.provider.GetConnectionStatus(c.Request.Context(), userID)
	if err != nil {
		slog.Info("Failed to get status", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get status"})
		return
	}

	c.JSON(http.StatusOK, status)
}

// ============================================================================
// Issue Handlers
// ============================================================================

// GetIssues returns issues from the local database.
func (h *Handler) GetIssues(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))

	issues, err := h.provider.GetIssues(c.Request.Context(), userID, limit)
	if err != nil {
		slog.Info("Failed to get issues", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get issues"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"issues": issues,
		"count":  len(issues),
	})
}

// GetIssue returns a single issue by ID.
func (h *Handler) GetIssue(c *gin.Context) {
	userID := c.GetString("user_id")
	issueID := c.Param("id")

	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var issue Issue
	err := h.provider.Pool().QueryRow(c.Request.Context(), `
		SELECT external_id, identifier, title, description, state, priority,
			assignee, project, team, due_date, external_created_at, external_updated_at
		FROM linear_issues
		WHERE user_id = $1 AND external_id = $2
	`, userID, issueID).Scan(
		&issue.ID, &issue.Identifier, &issue.Title, &issue.Description,
		&issue.State, &issue.Priority, &issue.Assignee, &issue.Project,
		&issue.Team, &issue.DueDate, &issue.CreatedAt, &issue.UpdatedAt,
	)

	if err != nil {
		slog.Info("Failed to get issue", "error", err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Issue not found"})
		return
	}

	c.JSON(http.StatusOK, issue)
}

// CreateIssue creates a new issue in Linear.
func (h *Handler) CreateIssue(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var input CreateIssueInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	issue, err := h.provider.CreateIssue(c.Request.Context(), userID, input)
	if err != nil {
		slog.Info("Failed to create issue", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create issue"})
		return
	}

	c.JSON(http.StatusCreated, issue)
}

// UpdateIssue updates an existing issue in Linear.
func (h *Handler) UpdateIssue(c *gin.Context) {
	userID := c.GetString("user_id")
	issueID := c.Param("id")

	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	if issueID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Issue ID is required"})
		return
	}

	var input map[string]interface{}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	issue, err := h.provider.UpdateIssue(c.Request.Context(), userID, issueID, input)
	if err != nil {
		slog.Info("Failed to update issue", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update issue"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"issue": issue,
	})
}

// SyncIssues syncs issues from Linear.
func (h *Handler) SyncIssues(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	token, err := h.provider.GetToken(c.Request.Context(), userID)
	if err != nil {
		slog.Info("Failed to get token", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get token"})
		return
	}

	result, err := h.provider.syncIssues(c.Request.Context(), userID, token.AccessToken)
	if err != nil {
		slog.Info("Failed to sync issues", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to sync issues"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"created": result.Created,
		"updated": result.Updated,
	})
}

// ============================================================================
// Project Handlers
// ============================================================================

// GetProjects returns projects from the local database.
func (h *Handler) GetProjects(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	projects, err := h.provider.GetProjects(c.Request.Context(), userID)
	if err != nil {
		slog.Info("Failed to get projects", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get projects"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"projects": projects,
		"count":    len(projects),
	})
}

// GetProject returns a single project by ID.
func (h *Handler) GetProject(c *gin.Context) {
	userID := c.GetString("user_id")
	projectID := c.Param("id")

	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var project Project
	err := h.provider.Pool().QueryRow(c.Request.Context(), `
		SELECT external_id, name, description, state, progress,
			start_date, target_date, team
		FROM linear_projects
		WHERE user_id = $1 AND external_id = $2
	`, userID, projectID).Scan(
		&project.ID, &project.Name, &project.Description,
		&project.State, &project.Progress, &project.StartDate,
		&project.TargetDate, &project.Team,
	)

	if err != nil {
		slog.Info("Failed to get project", "error", err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Project not found"})
		return
	}

	c.JSON(http.StatusOK, project)
}

// SyncProjects syncs projects from Linear.
func (h *Handler) SyncProjects(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	token, err := h.provider.GetToken(c.Request.Context(), userID)
	if err != nil {
		slog.Info("Failed to get token", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get token"})
		return
	}

	result, err := h.provider.syncProjects(c.Request.Context(), userID, token.AccessToken)
	if err != nil {
		slog.Info("Failed to sync projects", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to sync projects"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"created": result.Created,
		"updated": result.Updated,
	})
}

// ============================================================================
// Team Handlers
// ============================================================================

// GetTeams returns teams from the local database.
func (h *Handler) GetTeams(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	teams, err := h.provider.GetTeams(c.Request.Context(), userID)
	if err != nil {
		slog.Info("Failed to get teams", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get teams"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"teams": teams,
		"count": len(teams),
	})
}

// SyncTeams syncs teams from Linear.
func (h *Handler) SyncTeams(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	token, err := h.provider.GetToken(c.Request.Context(), userID)
	if err != nil {
		slog.Info("Failed to get token", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get token"})
		return
	}

	result, err := h.provider.syncTeams(c.Request.Context(), userID, token.AccessToken)
	if err != nil {
		slog.Info("Failed to sync teams", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to sync teams"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"created": result.Created,
		"updated": result.Updated,
	})
}

// ============================================================================
// Helper Functions
// ============================================================================
