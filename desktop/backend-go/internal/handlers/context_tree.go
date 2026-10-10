package handlers

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rhl/businessos-backend/internal/middleware"
	"github.com/rhl/businessos-backend/internal/services"
)

// ContextTreeHandler handles context tree-related HTTP endpoints
type ContextTreeHandler struct {
	pool           *pgxpool.Pool
	contextService *services.ContextService
	logger         *slog.Logger
}

// NewContextTreeHandler creates a new ContextTreeHandler
func NewContextTreeHandler(pool *pgxpool.Pool, embeddingService *services.EmbeddingService) *ContextTreeHandler {
	return &ContextTreeHandler{
		pool:           pool,
		contextService: services.NewContextService(pool, embeddingService),
		logger:         slog.Default().With("handler", "context_tree"),
	}
}

// ================================================
// REQUEST/RESPONSE TYPES
// ================================================

// TreeSearchRequest represents a context tree search request
type TreeSearchRequest struct {
	Query       string   `json:"query" binding:"required"`
	SearchType  string   `json:"search_type"`  // semantic, title, content
	EntityTypes []string `json:"entity_types"` // memories, documents, artifacts, contexts
	MaxResults  int      `json:"max_results"`
}

// LoadContextItemRequest represents a request to load a context item
type LoadContextItemRequest struct {
	ItemID   string `json:"item_id" binding:"required"`
	ItemType string `json:"item_type" binding:"required"`
}

// CreateSessionRequest represents a request to create a context session
type CreateSessionRequest struct {
	ConversationID string `json:"conversation_id" binding:"required"`
	AgentType      string `json:"agent_type" binding:"required"`
	MaxTokens      int    `json:"max_tokens"`
}

// ContextTreeResponse represents the full context tree response
type ContextTreeResponse struct {
	RootNode    *services.ContextTreeNode `json:"root_node"`
	TotalItems  int                       `json:"total_items"`
	LastUpdated string                    `json:"last_updated"`
}

// TreeSearchResultResponse represents a search result
type TreeSearchResultResponse struct {
	ID             string   `json:"id"`
	Title          string   `json:"title"`
	Type           string   `json:"type"`
	Summary        string   `json:"summary"`
	RelevanceScore float64  `json:"relevance_score"`
	TreePath       []string `json:"tree_path"`
	TokenEstimate  int      `json:"token_estimate"`
}

// ContextItemResponse represents a loaded context item
type ContextItemResponse struct {
	ID         string                 `json:"id"`
	Type       string                 `json:"type"`
	Title      string                 `json:"title"`
	Content    string                 `json:"content"`
	TokenCount int                    `json:"token_count"`
	Metadata   map[string]interface{} `json:"metadata"`
}

// ContextSessionResponse represents a context session
type ContextSessionResponse struct {
	ID                string `json:"id"`
	UserID            string `json:"user_id"`
	ConversationID    string `json:"conversation_id"`
	AgentType         string `json:"agent_type"`
	MaxContextTokens  int    `json:"max_context_tokens"`
	UsedContextTokens int    `json:"used_context_tokens"`
	AvailableTokens   int    `json:"available_tokens"`
	StartedAt         string `json:"started_at"`
	LastActivityAt    string `json:"last_activity_at"`
}

// ================================================
// CONTEXT TREE HANDLERS
// ================================================

// GetContextTree returns the context tree for an entity (project, node, or application)
// GET /api/context-tree/:entityType/:entityId
func (h *ContextTreeHandler) GetContextTree(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}

	entityType := c.Param("entityType")
	entityIDStr := c.Param("entityId")

	// Validate entity type
	validTypes := map[string]bool{
		"project": true, "node": true, "application": true,
	}
	if !validTypes[entityType] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid entity type. Must be: project, node, or application"})
		return
	}

	entityID, err := uuid.Parse(entityIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid entity ID"})
		return
	}

	// Get the context tree from service
	var projectID, nodeID *uuid.UUID
	switch entityType {
	case "project":
		projectID = &entityID
	case "node":
		nodeID = &entityID
	}

	tree, err := h.contextService.GetContextTree(c.Request.Context(), user.ID, projectID, nodeID)
	if err != nil {
		h.logger.Error("Failed to get context tree", "error", err, "entityType", entityType, "entityID", entityIDStr)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get context tree"})
		return
	}

	// Transform to response
	response := ContextTreeResponse{
		RootNode:    tree.RootNode,
		TotalItems:  tree.TotalItems,
		LastUpdated: tree.LastUpdated.Format("2006-01-02T15:04:05Z07:00"),
	}

	c.JSON(http.StatusOK, response)
}

// SearchContextTree performs a search within the context tree
// POST /api/context-tree/search
func (h *ContextTreeHandler) SearchContextTree(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}

	var req TreeSearchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Set defaults
	if req.SearchType == "" {
		req.SearchType = "title"
	}
	if req.MaxResults <= 0 {
		req.MaxResults = 10
	}

	// Build search params
	params := services.TreeSearchParams{
		Query:       req.Query,
		SearchType:  req.SearchType,
		EntityTypes: req.EntityTypes,
		MaxResults:  req.MaxResults,
	}

	// Perform search
	results, err := h.contextService.SearchTree(c.Request.Context(), user.ID, params)
	if err != nil {
		h.logger.Error("Failed to search context tree", "error", err, "query", req.Query)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Search failed"})
		return
	}

	// Transform results
	response := make([]TreeSearchResultResponse, len(results))
	for i, r := range results {
		response[i] = TreeSearchResultResponse{
			ID:             r.ID.String(),
			Title:          r.Title,
			Type:           r.Type,
			Summary:        r.Summary,
			RelevanceScore: r.RelevanceScore,
			TreePath:       r.TreePath,
			TokenEstimate:  r.TokenEstimate,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"results":     response,
		"count":       len(response),
		"query":       req.Query,
		"search_type": req.SearchType,
	})
}

// LoadContextItem loads a specific context item by ID and type
// POST /api/context-tree/load
func (h *ContextTreeHandler) LoadContextItem(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}

	var req LoadContextItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	itemID, err := uuid.Parse(req.ItemID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid item ID"})
		return
	}

	// Validate item type
	validTypes := map[string]bool{
		"memory": true, "document": true, "artifact": true, "context": true,
	}
	if !validTypes[req.ItemType] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid item type"})
		return
	}

	// Load the item
	item, err := h.contextService.LoadContextItem(c.Request.Context(), user.ID, itemID, req.ItemType)
	if err != nil {
		h.logger.Error("Failed to load context item", "error", err, "itemID", req.ItemID, "itemType", req.ItemType)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load context item"})
		return
	}

	if item == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Item not found"})
		return
	}

	// Transform response
	response := ContextItemResponse{
		ID:         item.ID.String(),
		Type:       item.Type,
		Title:      item.Title,
		Content:    item.Content,
		TokenCount: item.TokenCount,
		Metadata:   item.Metadata,
	}

	c.JSON(http.StatusOK, response)
}

// CreateContextSession creates a new context session for an agent
// POST /api/context-tree/session
func (h *ContextTreeHandler) CreateContextSession(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}

	var req CreateSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	conversationID, err := uuid.Parse(req.ConversationID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid conversation ID"})
		return
	}

	// Set default max tokens
	if req.MaxTokens <= 0 {
		req.MaxTokens = 8000
	}

	// Create session
	session, err := h.contextService.CreateContextSession(c.Request.Context(), user.ID, conversationID, req.AgentType, req.MaxTokens)
	if err != nil {
		h.logger.Error("Failed to create context session", "error", err, "agentType", req.AgentType)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create context session"})
		return
	}

	// Transform response
	response := ContextSessionResponse{
		ID:                session.ID.String(),
		UserID:            session.UserID,
		ConversationID:    session.ConversationID.String(),
		AgentType:         session.AgentType,
		MaxContextTokens:  session.MaxContextTokens,
		UsedContextTokens: session.UsedContextTokens,
		AvailableTokens:   session.AvailableTokens,
		StartedAt:         session.StartedAt.Format("2006-01-02T15:04:05Z07:00"),
		LastActivityAt:    session.LastActivityAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	c.JSON(http.StatusCreated, response)
}

// GetContextSession retrieves an existing context session
// GET /api/context-tree/session/:sessionId
func (h *ContextTreeHandler) GetContextSession(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}

	sessionID, err := uuid.Parse(c.Param("sessionId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid session ID"})
		return
	}

	session, err := h.contextService.GetContextSession(c.Request.Context(), sessionID)
	if err != nil {
		h.logger.Error("Failed to get context session", "error", err, "sessionID", sessionID.String())
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get context session"})
		return
	}

	if session == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Session not found"})
		return
	}

	// Verify ownership
	if session.UserID != user.ID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	response := ContextSessionResponse{
		ID:                session.ID.String(),
		UserID:            session.UserID,
		ConversationID:    session.ConversationID.String(),
		AgentType:         session.AgentType,
		MaxContextTokens:  session.MaxContextTokens,
		UsedContextTokens: session.UsedContextTokens,
		AvailableTokens:   session.AvailableTokens,
		StartedAt:         session.StartedAt.Format("2006-01-02T15:04:05Z07:00"),
		LastActivityAt:    session.LastActivityAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	c.JSON(http.StatusOK, response)
}

// UpdateContextSession updates token usage in a context session
// PUT /api/context-tree/session/:sessionId
func (h *ContextTreeHandler) UpdateContextSession(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}

	sessionID, err := uuid.Parse(c.Param("sessionId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid session ID"})
		return
	}

	var req struct {
		UsedTokens int `json:"used_tokens"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Verify ownership
	session, err := h.contextService.GetContextSession(c.Request.Context(), sessionID)
	if err != nil || session == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Session not found"})
		return
	}
	if session.UserID != user.ID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	err = h.contextService.UpdateSessionTokenUsage(c.Request.Context(), sessionID, req.UsedTokens)
	if err != nil {
		h.logger.Error("Failed to update context session", "error", err, "sessionID", sessionID.String())
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update session"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":     "Session updated",
		"session_id":  sessionID.String(),
		"used_tokens": req.UsedTokens,
	})
}

// EndContextSession ends a context session
// DELETE /api/context-tree/session/:sessionId
func (h *ContextTreeHandler) EndContextSession(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}

	sessionID, err := uuid.Parse(c.Param("sessionId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid session ID"})
		return
	}

	// Verify ownership
	session, err := h.contextService.GetContextSession(c.Request.Context(), sessionID)
	if err != nil || session == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Session not found"})
		return
	}
	if session.UserID != user.ID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	// Update session to mark as ended
	_, err = h.pool.Exec(c.Request.Context(), `
		UPDATE agent_context_sessions SET ended_at = NOW() WHERE id = $1
	`, sessionID)
	if err != nil {
		h.logger.Error("Failed to end context session", "error", err, "sessionID", sessionID.String())
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to end session"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":    "Session ended",
		"session_id": sessionID.String(),
	})
}

// GetLoadingRules returns context loading rules for a trigger
// GET /api/context-tree/rules/:entityType/:entityId
func (h *ContextTreeHandler) GetLoadingRules(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}

	triggerType := c.Param("entityType")
	triggerValue := c.Param("entityId")

	rules, err := h.contextService.GetLoadingRules(c.Request.Context(), user.ID, triggerType, triggerValue)
	if err != nil {
		h.logger.Error("Failed to get loading rules", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get loading rules"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"rules":         rules,
		"trigger_type":  triggerType,
		"trigger_value": triggerValue,
	})
}

// GetContextStats returns statistics about context usage
// GET /api/context-tree/stats
func (h *ContextTreeHandler) GetContextStats(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}

	stats, err := h.contextService.GetTreeStatistics(c.Request.Context(), user.ID)
	if err != nil {
		h.logger.Error("Failed to get context stats", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get statistics"})
		return
	}

	c.JSON(http.StatusOK, stats)
}
