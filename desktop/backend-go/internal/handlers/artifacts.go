package handlers

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rhl/businessos-backend/internal/database/sqlc"
	"github.com/rhl/businessos-backend/internal/middleware"
	"github.com/rhl/businessos-backend/internal/utils"
)

// ArtifactHandler handles artifact-related HTTP requests.
type ArtifactHandler struct {
	pool *pgxpool.Pool
}

// NewArtifactHandler creates a new ArtifactHandler.
func NewArtifactHandler(pool *pgxpool.Pool) *ArtifactHandler {
	return &ArtifactHandler{pool: pool}
}

// RegisterArtifactRoutes registers artifact routes on the given router group.
func RegisterArtifactRoutes(api *gin.RouterGroup, h *ArtifactHandler, auth gin.HandlerFunc) {
	artifacts := api.Group("/artifacts")
	artifacts.Use(auth, middleware.RequireAuth())
	{
		artifacts.GET("", h.ListArtifacts)
		artifacts.POST("", h.CreateArtifact)
		artifacts.GET("/:id", h.GetArtifact)
		artifacts.PATCH("/:id", h.UpdateArtifact)
		artifacts.PATCH("/:id/link", h.LinkArtifact)
		artifacts.DELETE("/:id", h.DeleteArtifact)
		artifacts.GET("/:id/versions", h.GetArtifactVersions)
		artifacts.POST("/:id/restore", h.RestoreArtifactVersion)
	}
}

// ListArtifacts returns all artifacts for the current user.
func (h *ArtifactHandler) ListArtifacts(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	queries := sqlc.New(h.pool)

	// Parse optional filters
	var conversationID, projectID, contextID pgtype.UUID
	if cid := c.Query("conversation_id"); cid != "" {
		if parsed, err := uuid.Parse(cid); err == nil {
			conversationID = pgtype.UUID{Bytes: parsed, Valid: true}
		}
	}
	if pid := c.Query("project_id"); pid != "" {
		if parsed, err := uuid.Parse(pid); err == nil {
			projectID = pgtype.UUID{Bytes: parsed, Valid: true}
		}
	}
	if ctxid := c.Query("context_id"); ctxid != "" {
		if parsed, err := uuid.Parse(ctxid); err == nil {
			contextID = pgtype.UUID{Bytes: parsed, Valid: true}
		}
	}

	artifacts, err := queries.ListArtifacts(c.Request.Context(), sqlc.ListArtifactsParams{
		UserID:         user.ID,
		ConversationID: conversationID,
		ProjectID:      projectID,
		ContextID:      contextID,
	})
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "list artifacts", err)
		return
	}

	c.JSON(http.StatusOK, TransformArtifacts(artifacts))
}

// CreateArtifact creates a new artifact.
func (h *ArtifactHandler) CreateArtifact(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	var req struct {
		Title          string  `json:"title" binding:"required"`
		Type           string  `json:"type" binding:"required"`
		Content        string  `json:"content" binding:"required"`
		Language       *string `json:"language"`
		Summary        *string `json:"summary"`
		ConversationID *string `json:"conversation_id"`
		MessageID      *string `json:"message_id"`
		ProjectID      *string `json:"project_id"`
		ContextID      *string `json:"context_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondInvalidRequest(c, slog.Default(), err)
		return
	}

	queries := sqlc.New(h.pool)

	// Parse optional IDs
	var conversationID, messageID, projectID, contextID pgtype.UUID
	if req.ConversationID != nil {
		if parsed, err := uuid.Parse(*req.ConversationID); err == nil {
			conversationID = pgtype.UUID{Bytes: parsed, Valid: true}
		}
	}
	if req.MessageID != nil {
		if parsed, err := uuid.Parse(*req.MessageID); err == nil {
			messageID = pgtype.UUID{Bytes: parsed, Valid: true}
		}
	}
	if req.ProjectID != nil {
		if parsed, err := uuid.Parse(*req.ProjectID); err == nil {
			projectID = pgtype.UUID{Bytes: parsed, Valid: true}
		}
	}
	if req.ContextID != nil {
		if parsed, err := uuid.Parse(*req.ContextID); err == nil {
			contextID = pgtype.UUID{Bytes: parsed, Valid: true}
		}
	}

	// Map string type to enum
	artifactType := stringToArtifactType(req.Type)

	artifact, err := queries.CreateArtifact(c.Request.Context(), sqlc.CreateArtifactParams{
		UserID:         user.ID,
		ConversationID: conversationID,
		MessageID:      messageID,
		ProjectID:      projectID,
		ContextID:      contextID,
		Title:          req.Title,
		Type:           artifactType,
		Language:       req.Language,
		Content:        req.Content,
		Summary:        req.Summary,
	})
	if err != nil {
		slog.Error("[Artifacts] Failed to create artifact", "error", err)
		utils.RespondInternalError(c, slog.Default(), "create artifact", err)
		return
	}

	c.JSON(http.StatusCreated, TransformArtifact(artifact))
}

// GetArtifact returns a single artifact.
func (h *ArtifactHandler) GetArtifact(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "artifact_id")
		return
	}

	queries := sqlc.New(h.pool)
	artifact, err := queries.GetArtifact(c.Request.Context(), sqlc.GetArtifactParams{
		ID:     pgtype.UUID{Bytes: id, Valid: true},
		UserID: user.ID,
	})
	if err != nil {
		utils.RespondNotFound(c, slog.Default(), "Artifact")
		return
	}

	// Check if versions are requested
	if c.Query("include_versions") == "true" {
		versions, err := queries.GetArtifactVersions(c.Request.Context(), pgtype.UUID{Bytes: id, Valid: true})
		if err == nil {
			c.JSON(http.StatusOK, gin.H{
				"artifact": TransformArtifact(artifact),
				"versions": versions,
			})
			return
		}
	}

	c.JSON(http.StatusOK, TransformArtifact(artifact))
}

// UpdateArtifact updates an existing artifact.
func (h *ArtifactHandler) UpdateArtifact(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "artifact_id")
		return
	}

	var req struct {
		Title   string  `json:"title"`
		Content string  `json:"content"`
		Summary *string `json:"summary"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondInvalidRequest(c, slog.Default(), err)
		return
	}

	queries := sqlc.New(h.pool)

	// First verify the artifact belongs to the user
	existing, err := queries.GetArtifact(c.Request.Context(), sqlc.GetArtifactParams{
		ID:     pgtype.UUID{Bytes: id, Valid: true},
		UserID: user.ID,
	})
	if err != nil {
		utils.RespondNotFound(c, slog.Default(), "Artifact")
		return
	}

	// Save current version before updating
	version := int32(1)
	if existing.Version != nil {
		version = *existing.Version
	}
	_, _ = queries.CreateArtifactVersion(c.Request.Context(), sqlc.CreateArtifactVersionParams{
		ArtifactID: existing.ID,
		Version:    version,
		Content:    existing.Content,
	})

	// Use existing values if not provided
	title := req.Title
	if title == "" {
		title = existing.Title
	}
	content := req.Content
	if content == "" {
		content = existing.Content
	}

	artifact, err := queries.UpdateArtifact(c.Request.Context(), sqlc.UpdateArtifactParams{
		ID:      pgtype.UUID{Bytes: id, Valid: true},
		Title:   title,
		Content: content,
		Summary: req.Summary,
	})
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "update artifact", err)
		return
	}

	c.JSON(http.StatusOK, TransformArtifact(artifact))
}

// LinkArtifact links an artifact to a project or context.
func (h *ArtifactHandler) LinkArtifact(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "artifact_id")
		return
	}

	var req struct {
		ProjectID *string `json:"project_id"`
		ContextID *string `json:"context_id"`
		SyncToKB  bool    `json:"sync_to_kb"` // If true, sync artifact content to context
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondInvalidRequest(c, slog.Default(), err)
		return
	}

	queries := sqlc.New(h.pool)

	// Get artifact with content for potential sync
	existingArtifact, err := queries.GetArtifact(c.Request.Context(), sqlc.GetArtifactParams{
		ID:     pgtype.UUID{Bytes: id, Valid: true},
		UserID: user.ID,
	})
	if err != nil {
		utils.RespondNotFound(c, slog.Default(), "Artifact")
		return
	}

	var projectID, contextID pgtype.UUID
	if req.ProjectID != nil {
		if parsed, err := uuid.Parse(*req.ProjectID); err == nil {
			projectID = pgtype.UUID{Bytes: parsed, Valid: true}
		}
	}
	if req.ContextID != nil {
		if parsed, err := uuid.Parse(*req.ContextID); err == nil {
			contextID = pgtype.UUID{Bytes: parsed, Valid: true}
		}
	}

	artifact, err := queries.LinkArtifact(c.Request.Context(), sqlc.LinkArtifactParams{
		ID:        pgtype.UUID{Bytes: id, Valid: true},
		ProjectID: projectID,
		ContextID: contextID,
	})
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "link artifact", err)
		return
	}

	// Sync content to context if requested and context is linked
	if req.SyncToKB && contextID.Valid {
		// Calculate word count
		wordCount := int32(len(strings.Fields(existingArtifact.Content)))

		// Format content with title header
		syncContent := "# " + existingArtifact.Title + "\n\n" + existingArtifact.Content

		_, syncErr := queries.SyncArtifactToContext(c.Request.Context(), sqlc.SyncArtifactToContextParams{
			ID:        contextID,
			Content:   &syncContent,
			WordCount: &wordCount,
		})
		if syncErr != nil {
			// Log but don't fail the request - linking succeeded
			// The user can still sync manually later
			slog.Warn("[Artifacts] Artifact sync failed", "error", syncErr)
		}
	}

	c.JSON(http.StatusOK, TransformArtifact(artifact))
}

// DeleteArtifact deletes an artifact.
func (h *ArtifactHandler) DeleteArtifact(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "artifact_id")
		return
	}

	queries := sqlc.New(h.pool)
	err = queries.DeleteArtifact(c.Request.Context(), sqlc.DeleteArtifactParams{
		ID:     pgtype.UUID{Bytes: id, Valid: true},
		UserID: user.ID,
	})
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "delete artifact", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Artifact deleted"})
}

// GetArtifactVersions returns version history for an artifact.
func (h *ArtifactHandler) GetArtifactVersions(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "artifact_id")
		return
	}

	queries := sqlc.New(h.pool)

	// Verify ownership
	_, err = queries.GetArtifact(c.Request.Context(), sqlc.GetArtifactParams{
		ID:     pgtype.UUID{Bytes: id, Valid: true},
		UserID: user.ID,
	})
	if err != nil {
		utils.RespondNotFound(c, slog.Default(), "Artifact")
		return
	}

	versions, err := queries.GetArtifactVersions(c.Request.Context(), pgtype.UUID{Bytes: id, Valid: true})
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "get versions", err)
		return
	}

	c.JSON(http.StatusOK, versions)
}

// RestoreArtifactVersion restores a previous version of an artifact.
func (h *ArtifactHandler) RestoreArtifactVersion(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "artifact_id")
		return
	}

	var req struct {
		Version int32 `json:"version" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondInvalidRequest(c, slog.Default(), err)
		return
	}

	queries := sqlc.New(h.pool)

	// Verify ownership and get current artifact
	existing, err := queries.GetArtifact(c.Request.Context(), sqlc.GetArtifactParams{
		ID:     pgtype.UUID{Bytes: id, Valid: true},
		UserID: user.ID,
	})
	if err != nil {
		utils.RespondNotFound(c, slog.Default(), "Artifact")
		return
	}

	// Get the version to restore
	versionToRestore, err := queries.GetArtifactVersion(c.Request.Context(), sqlc.GetArtifactVersionParams{
		ArtifactID: pgtype.UUID{Bytes: id, Valid: true},
		Version:    req.Version,
	})
	if err != nil {
		utils.RespondNotFound(c, slog.Default(), "Version")
		return
	}

	// Save current version before restoring
	currentVersion := int32(1)
	if existing.Version != nil {
		currentVersion = *existing.Version
	}
	_, _ = queries.CreateArtifactVersion(c.Request.Context(), sqlc.CreateArtifactVersionParams{
		ArtifactID: existing.ID,
		Version:    currentVersion,
		Content:    existing.Content,
	})

	// Update artifact with restored content
	artifact, err := queries.UpdateArtifact(c.Request.Context(), sqlc.UpdateArtifactParams{
		ID:      pgtype.UUID{Bytes: id, Valid: true},
		Title:   existing.Title,
		Content: versionToRestore.Content,
		Summary: existing.Summary,
	})
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "restore version", err)
		return
	}

	c.JSON(http.StatusOK, TransformArtifact(artifact))
}

// stringToArtifactType converts a string to sqlc.Artifacttype.
func stringToArtifactType(t string) sqlc.Artifacttype {
	typeMap := map[string]sqlc.Artifacttype{
		"code":     sqlc.ArtifacttypeCODE,
		"document": sqlc.ArtifacttypeDOCUMENT,
		"markdown": sqlc.ArtifacttypeMARKDOWN,
		"react":    sqlc.ArtifacttypeREACT,
		"html":     sqlc.ArtifacttypeHTML,
		"svg":      sqlc.ArtifacttypeSVG,
		// Map old types to DOCUMENT
		"proposal":  sqlc.ArtifacttypeDOCUMENT,
		"sop":       sqlc.ArtifacttypeDOCUMENT,
		"framework": sqlc.ArtifacttypeDOCUMENT,
		"agenda":    sqlc.ArtifacttypeDOCUMENT,
		"report":    sqlc.ArtifacttypeDOCUMENT,
		"plan":      sqlc.ArtifacttypeDOCUMENT,
		"other":     sqlc.ArtifacttypeDOCUMENT,
	}
	if enum, ok := typeMap[strings.ToLower(t)]; ok {
		return enum
	}
	return sqlc.ArtifacttypeDOCUMENT
}
