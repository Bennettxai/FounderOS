package handlers

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rhl/businessos-backend/internal/cache"
	"github.com/rhl/businessos-backend/internal/database/sqlc"
	"github.com/rhl/businessos-backend/internal/middleware"
	"github.com/rhl/businessos-backend/internal/services"
	"github.com/rhl/businessos-backend/internal/utils"
)

// ProjectHandler handles project CRUD operations and member management.
type ProjectHandler struct {
	EngineSyncHook
	pool                 *pgxpool.Pool
	queryCache           *cache.QueryCache
	notificationTriggers *services.NotificationTriggers
	projectAccessService *services.ProjectAccessService
}

// NewProjectHandler constructs a ProjectHandler with all required dependencies.
func NewProjectHandler(pool *pgxpool.Pool, queryCache *cache.QueryCache, notifTriggers *services.NotificationTriggers, projectAccessService *services.ProjectAccessService) *ProjectHandler {
	return &ProjectHandler{
		pool:                 pool,
		queryCache:           queryCache,
		notificationTriggers: notifTriggers,
		projectAccessService: projectAccessService,
	}
}

// RegisterProjectRoutes registers all project and project-member routes under the given api group.
func RegisterProjectRoutes(api *gin.RouterGroup, h *ProjectHandler, auth gin.HandlerFunc) {
	projects := api.Group("/projects")
	projects.Use(auth, middleware.RequireAuth())
	{
		projects.GET("", h.ListProjects)
		projects.POST("", h.CreateProject)
		projects.GET("/stats", h.GetProjectStats)
		projects.GET("/overdue", h.GetOverdueProjects)
		projects.GET("/upcoming", h.GetUpcomingProjects)
		projects.GET("/:id", h.GetProject)
		projects.PUT("/:id", h.UpdateProject)
		projects.DELETE("/:id", h.DeleteProject)
		projects.POST("/:id/notes", h.AddProjectNote)
		// File uploads
		projects.POST("/:id/files", h.UploadProjectFile)
		projects.GET("/:id/files/:filename", h.ServeProjectFile)
		// Project members (team assignment with role-based access)
		projects.GET("/:id/members", h.ListProjectMembers)
		projects.POST("/:id/members", h.AddProjectMember)
		projects.PUT("/:id/members/:memberId/role", h.UpdateProjectMemberRole)
		projects.DELETE("/:id/members/:memberId", h.RemoveProjectMember)
		projects.GET("/:id/access/:userId", h.CheckProjectAccess)
	}
}

// ListProjects returns all projects for the current user.
// Results are cached in Redis for 5 minutes with user-specific keys.
func (h *ProjectHandler) ListProjects(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	pg := ParsePagination(c)

	// Build cache filters from query parameters
	filters := make(map[string]string)
	if s := c.Query("status"); s != "" {
		filters["status"] = s
	}
	if p := c.Query("priority"); p != "" {
		filters["priority"] = p
	}
	if cid := c.Query("client_id"); cid != "" {
		filters["client_id"] = cid
	}
	if workspaceID, ok := h.projectWorkspaceScope(c, user.ID); ok {
		filters["workspace_id"] = workspaceID.String()
	}

	// Generate cache key using ProjectListKey helper — cache full (unpaginated) result set
	// then slice in-process so cache entries remain reusable across pages.
	var allProjects []map[string]interface{}

	if h.queryCache != nil {
		cacheKey := h.queryCache.ProjectListKey(user.ID, 0, filters) // page=0 means "all"
		if err := h.queryCache.GetOrCompute(
			c.Request.Context(),
			cacheKey,
			5*time.Minute,
			&allProjects,
			func() (interface{}, error) {
				return h.fetchProjectsFromDB(c, user.ID, filters)
			},
		); err != nil {
			slog.Debug("Cache error for ListProjects, falling back to direct DB query", "key", cacheKey)
			allProjects = nil
		}
	}

	if allProjects == nil {
		raw, err := h.fetchProjectsFromDB(c, user.ID, filters)
		if err != nil {
			slog.Info("ListProjects error for user", "value", user.ID, "error", err)
			utils.RespondInternalError(c, slog.Default(), "list projects", nil)
			return
		}
		allProjects = raw.([]map[string]interface{})
	}

	total := int64(len(allProjects))
	start := int(pg.Offset)
	end := start + int(pg.Limit)
	if start > len(allProjects) {
		start = len(allProjects)
	}
	if end > len(allProjects) {
		end = len(allProjects)
	}

	c.JSON(http.StatusOK, NewPaginatedResponse(allProjects[start:end], total, pg))
}

// fetchProjectsFromDB queries the database for projects.
func (h *ProjectHandler) fetchProjectsFromDB(c *gin.Context, userID string, filters map[string]string) (interface{}, error) {
	queries := sqlc.New(h.pool)

	// Parse optional status filter
	var status sqlc.NullProjectstatus
	if s, ok := filters["status"]; ok && s != "" {
		status = sqlc.NullProjectstatus{Projectstatus: stringToProjectStatus(s), Valid: true}
	}

	// Parse optional priority filter
	var priority sqlc.NullProjectpriority
	if p, ok := filters["priority"]; ok && p != "" {
		priority = sqlc.NullProjectpriority{Projectpriority: stringToProjectPriority(p), Valid: true}
	}

	// Parse optional client_id filter
	var clientID pgtype.UUID
	if cid, ok := filters["client_id"]; ok && cid != "" {
		if id, err := uuid.Parse(cid); err == nil {
			clientID = pgtype.UUID{Bytes: id, Valid: true}
		}
	}

	var workspaceID pgtype.UUID
	if wid, ok := filters["workspace_id"]; ok && wid != "" {
		if id, err := uuid.Parse(wid); err == nil {
			workspaceID = pgtype.UUID{Bytes: id, Valid: true}
		}
	}

	projects, err := queries.ListProjects(c.Request.Context(), sqlc.ListProjectsParams{
		UserID:      userID,
		Status:      status,
		Priority:    priority,
		ClientID:    clientID,
		WorkspaceID: workspaceID,
	})
	if err != nil {
		return nil, err
	}

	return TransformProjectRows(projects), nil
}

// projectWorkspaceScope resolves the active workspace and verifies membership.
// Project lists must never mix records from other workspaces the user belongs to.
func (h *ProjectHandler) projectWorkspaceScope(c *gin.Context, userID string) (uuid.UUID, bool) {
	hdr := c.GetHeader("X-Workspace-ID")
	if hdr == "" {
		return uuid.Nil, false
	}
	workspaceID, err := uuid.Parse(hdr)
	if err != nil {
		return uuid.Nil, false
	}
	var member bool
	err = h.pool.QueryRow(c.Request.Context(),
		`SELECT EXISTS(SELECT 1 FROM workspace_members WHERE workspace_id=$1 AND user_id=$2 AND status='active')`,
		workspaceID, userID).Scan(&member)
	if err != nil || !member {
		return uuid.Nil, false
	}
	return workspaceID, true
}

// CreateProject creates a new project.
func (h *ProjectHandler) CreateProject(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	var req struct {
		Name            string          `json:"name" binding:"required"`
		Description     *string         `json:"description"`
		Status          *string         `json:"status"`
		Priority        *string         `json:"priority"`
		ClientName      *string         `json:"client_name"`
		ClientID        *string         `json:"client_id"`
		ProjectType     *string         `json:"project_type"`
		ProjectMetadata json.RawMessage `json:"project_metadata"`
		StartDate       *string         `json:"start_date"`
		DueDate         *string         `json:"due_date"`
		Visibility      *string         `json:"visibility"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondInvalidRequest(c, slog.Default(), err)
		return
	}

	queries := sqlc.New(h.pool)

	// Parse status with default
	status := sqlc.NullProjectstatus{
		Projectstatus: sqlc.ProjectstatusACTIVE, // Default to "active"
		Valid:         true,
	}
	if req.Status != nil {
		status.Projectstatus = stringToProjectStatus(*req.Status)
	}

	// Parse priority with default
	priority := sqlc.NullProjectpriority{
		Projectpriority: sqlc.ProjectpriorityMEDIUM, // Default to "medium"
		Valid:           true,
	}
	if req.Priority != nil {
		priority.Projectpriority = stringToProjectPriority(*req.Priority)
	}

	// Parse client_id
	var clientID pgtype.UUID
	if req.ClientID != nil {
		if id, err := uuid.Parse(*req.ClientID); err == nil {
			clientID = pgtype.UUID{Bytes: id, Valid: true}
		}
	}

	// Handle metadata (pass nil for empty jsonb — SimpleProtocol compatibility)
	var metadata []byte
	if req.ProjectMetadata != nil {
		metadata = req.ProjectMetadata
	}

	// Parse dates
	var startDate pgtype.Date
	if req.StartDate != nil {
		if t, err := time.Parse("2006-01-02", *req.StartDate); err == nil {
			startDate = pgtype.Date{Time: t, Valid: true}
		}
	}

	var dueDate pgtype.Date
	if req.DueDate != nil {
		if t, err := time.Parse("2006-01-02", *req.DueDate); err == nil {
			dueDate = pgtype.Date{Time: t, Valid: true}
		}
	}

	// Set owner_id to current user
	ownerID := user.ID

	project, err := queries.CreateProject(c.Request.Context(), sqlc.CreateProjectParams{
		UserID:          user.ID,
		Name:            req.Name,
		Description:     req.Description,
		Status:          status,
		Priority:        priority,
		ClientName:      req.ClientName,
		ClientID:        clientID,
		ProjectType:     req.ProjectType,
		ProjectMetadata: metadata,
		StartDate:       startDate,
		DueDate:         dueDate,
		Visibility:      req.Visibility,
		OwnerID:         &ownerID,
	})
	if err != nil {
		slog.Info("CreateProject error for user", "value", user.ID, "error", err)
		utils.RespondInternalError(c, slog.Default(), "create project", nil)
		return
	}

	// Invalidate projects cache for this user after successful creation
	if h.queryCache != nil {
		cachePattern := fmt.Sprintf("projects:%s:*", user.ID)
		go h.invalidateProjectsCachePattern(c.Request.Context(), cachePattern)
	}

	// BusinessOSSync: share to the workspace when posted with an authorized
	// X-Workspace-ID (projects module is workspace-mode), so members see it.
	wsID := stampWorkspaceOnCreate(c, h.pool, "projects", project.ID, user.ID, "projects")

	// Mirror into the OptimalEngine knowledge graph so the project
	// surfaces in the Pages graph view alongside other modules' content.
	h.enqueue(c.Request.Context(), services.Signal{
		Module:      services.ModuleProjects,
		ID:          uuidString(project.ID.Bytes),
		AuthorID:    user.ID,
		WorkspaceID: wsID,
		Title:       project.Name,
		Body:        optString(project.Description),
		Genre:       "project",
		ModifiedAt:  pgPlainTimestampToTime(project.UpdatedAt),
	})

	c.JSON(http.StatusCreated, project)
}

// ---------------------------------------------------------------------------
// Package-level helper functions (no receiver — used by dashboard and tools)
// ---------------------------------------------------------------------------
