package handlers

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rhl/businessos-backend/internal/config"
	"github.com/rhl/businessos-backend/internal/database/sqlc"
	"github.com/rhl/businessos-backend/internal/feedback"
	osa "github.com/rhl/businessos-backend/internal/integrations/osa"
	"github.com/rhl/businessos-backend/internal/middleware"
	"github.com/rhl/businessos-backend/internal/services"
	"github.com/rhl/businessos-backend/internal/subconscious"
	"github.com/rhl/businessos-backend/internal/utils"
)

// ChatHandler handles all chat-domain HTTP requests.
type ChatHandler struct {
	EngineSyncHook
	pool                 *pgxpool.Pool
	cfg                  *config.Config
	tieredContextService *services.TieredContextService
	contextTracker       *services.ContextTrackerService
	autoLearningTriggers *services.AutoLearningTriggers
	blockMapper          *services.BlockMapperService
	roleContextService   *services.RoleContextService
	// Additional fields required by SendMessage / streaming logic
	embeddingService       *services.EmbeddingService
	promptPersonalizer     *services.PromptPersonalizer
	memoryHierarchyService *services.MemoryHierarchyService
	skillsLoader           *services.SkillsLoader
	sessionHealthSvc       *services.SessionHealthService
	osaClient              *osa.ResilientClient
	signalHints            feedback.SignalHintProvider // Homeostatic feedback → prompt corrections
	subconsciousObserver   *subconscious.Observer      // Subconscious observer for async pattern detection
}

// NewChatHandler constructs a ChatHandler with the required dependencies.
func NewChatHandler(
	pool *pgxpool.Pool,
	cfg *config.Config,
	tieredCtx *services.TieredContextService,
	ctxTracker *services.ContextTrackerService,
	autoLearn *services.AutoLearningTriggers,
	blockMapper *services.BlockMapperService,
	roleCtx *services.RoleContextService,
) *ChatHandler {
	return &ChatHandler{
		pool:                 pool,
		cfg:                  cfg,
		tieredContextService: tieredCtx,
		contextTracker:       ctxTracker,
		autoLearningTriggers: autoLearn,
		blockMapper:          blockMapper,
		roleContextService:   roleCtx,
	}
}

// withOptionalDeps populates the optional service dependencies and returns the handler
// for fluent wiring in RegisterRoutes.
func (h *ChatHandler) withOptionalDeps(
	embedding *services.EmbeddingService,
	personalizer *services.PromptPersonalizer,
	memHierarchy *services.MemoryHierarchyService,
	skills *services.SkillsLoader,
	sessionHealth *services.SessionHealthService,
	osaClient *osa.ResilientClient,
	signalHints feedback.SignalHintProvider,
	observer *subconscious.Observer,
) *ChatHandler {
	h.embeddingService = embedding
	h.promptPersonalizer = personalizer
	h.memoryHierarchyService = memHierarchy
	h.skillsLoader = skills
	h.sessionHealthSvc = sessionHealth
	h.osaClient = osaClient
	h.signalHints = signalHints
	h.subconsciousObserver = observer
	return h
}

// RegisterChatRoutes mounts all chat-domain routes onto the provided router group.
func RegisterChatRoutes(api *gin.RouterGroup, h *ChatHandler, auth gin.HandlerFunc) {
	chat := api.Group("/chat")
	chat.Use(auth, middleware.RequireAuth())
	chat.Use(middleware.AuditSensitiveAccess("conversation"))
	{
		chat.GET("/conversations", h.ListConversations)
		chat.POST("/conversations", h.CreateConversation)
		chat.GET("/conversations/:id", h.GetConversation)
		chat.PUT("/conversations/:id", h.UpdateConversation)
		chat.DELETE("/conversations/:id", h.DeleteConversation)
		chat.POST("/message", h.SendMessage)    // Primary endpoint (streaming SSE + artifacts)
		chat.POST("/v2/message", h.SendMessage) // Deprecated: kept for backwards compatibility
		chat.GET("/search", h.SearchConversations)
		// AI-powered endpoints
		chat.POST("/ai/document", h.DocumentAI)
		chat.POST("/ai/analyze", h.AnalyzeContent)
		chat.POST("/ai/extract-tasks", h.ExtractTasks)
		chat.POST("/ai/plan", h.CreatePlan)
	}
}

// SendMessageRequest represents the request body for sending a message
type SendMessageRequest struct {
	Message        string            `json:"message" binding:"required"`
	ConversationID *string           `json:"conversation_id"`
	ContextID      *string           `json:"context_id"`  // Legacy: single context ID
	ContextIDs     []string          `json:"context_ids"` // NEW: Multiple context IDs for tiered context
	ProjectID      *string           `json:"project_id"`
	NodeID         *string           `json:"node_id"`      // NEW: Business node context
	WorkspaceID    *string           `json:"workspace_id"` // NEW (Feature 1): Workspace context for role-based permissions
	DocumentIDs    []string          `json:"document_ids"` // NEW: Attached document IDs for RAG
	Model          *string           `json:"model"`
	AgentType      *string           `json:"agent_type"`    // orchestrator, document, analysis, planning
	FocusMode      *string           `json:"focus_mode"`    // research, analyze, write, build, general
	FocusOptions   map[string]string `json:"focus_options"` // depth, output, searchScope, etc.
	Command        *string           `json:"command"`       // slash command: analyze, summarize, explain, etc.
	Temperature    *float64          `json:"temperature"`
	MaxTokens      *int              `json:"max_tokens"`
	TopP           *float64          `json:"top_p"`
	UseCOT         *bool             `json:"use_cot"` // Enable Chain of Thought with multi-agent coordination
	// Thinking/COT settings
	ThinkingEnabled     *bool   `json:"thinking_enabled"`      // Enable thinking/reasoning display
	ReasoningTemplateID *string `json:"reasoning_template_id"` // Custom reasoning template to use
	SaveThinking        *bool   `json:"save_thinking"`         // Save thinking traces to database
	MaxThinkingTokens   *int    `json:"max_thinking_tokens"`   // Max tokens for thinking
	// Output Style settings
	OutputStyle      *string `json:"output_style"`      // technical, creative, executive, concise
	StructuredOutput *bool   `json:"structured_output"` // If true, backend will return structured Blocks
}

// ListConversations returns all conversations for the current user
// Optional query parameter: context_id to filter by context
func (h *ChatHandler) ListConversations(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	queries := sqlc.New(h.pool)

	// Check for optional context_id filter
	contextIDStr := c.Query("context_id")
	if contextIDStr != "" {
		contextID, err := uuid.Parse(contextIDStr)
		if err != nil {
			utils.RespondInvalidID(c, slog.Default(), "context_id")
			return
		}

		// Filter conversations by context
		conversations, err := queries.ListConversationsByContext(c.Request.Context(), sqlc.ListConversationsByContextParams{
			UserID:    user.ID,
			ContextID: pgtype.UUID{Bytes: contextID, Valid: true},
		})
		if err != nil {
			utils.RespondInternalError(c, slog.Default(), "list conversations", err)
			return
		}
		c.JSON(http.StatusOK, TransformConversationsByContextRows(conversations))
		return
	}

	// No filter, return all conversations with pagination
	pg := ParsePagination(c)

	conversations, err := queries.ListConversations(c.Request.Context(), user.ID)
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "list conversations", err)
		return
	}

	all := TransformConversationListRows(conversations)
	total := int64(len(all))
	start := int(pg.Offset)
	end := start + int(pg.Limit)
	if start > len(all) {
		start = len(all)
	}
	if end > len(all) {
		end = len(all)
	}

	c.JSON(http.StatusOK, NewPaginatedResponse(all[start:end], total, pg))
}

// CreateConversation creates a new conversation
func (h *ChatHandler) CreateConversation(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	var req struct {
		Title     string  `json:"title"`
		ContextID *string `json:"context_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondInvalidRequest(c, slog.Default(), err)
		return
	}

	if req.Title == "" {
		req.Title = "New Conversation"
	}

	queries := sqlc.New(h.pool)

	var contextID pgtype.UUID
	if req.ContextID != nil {
		parsed, err := uuid.Parse(*req.ContextID)
		if err == nil {
			contextID = pgtype.UUID{Bytes: parsed, Valid: true}
		}
	}

	conversation, err := queries.CreateConversation(c.Request.Context(), sqlc.CreateConversationParams{
		UserID:    user.ID,
		Title:     &req.Title,
		ContextID: contextID,
	})
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "create conversation", err)
		return
	}

	// Mirror conversation start into the OptimalEngine knowledge graph
	// so chat threads surface in the Pages graph view alongside Pages,
	// Tasks, etc. Individual messages stream through the model and aren't
	// re-mirrored — only the conversation envelope.
	h.enqueue(c.Request.Context(), services.Signal{
		Module:     services.ModuleChat,
		ID:         "conversation-" + uuidString(conversation.ID.Bytes),
		AuthorID:   user.ID,
		Title:      req.Title,
		Body:       req.Title,
		Genre:      "conversation",
		ModifiedAt: pgPlainTimestampToTime(conversation.UpdatedAt),
	})

	c.JSON(http.StatusCreated, TransformConversation(conversation))
}

// GetConversation returns a single conversation with messages
func (h *ChatHandler) GetConversation(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "conversation ID")
		return
	}

	queries := sqlc.New(h.pool)

	conversation, err := queries.GetConversation(c.Request.Context(), sqlc.GetConversationParams{
		ID:     pgtype.UUID{Bytes: id, Valid: true},
		UserID: user.ID,
	})
	if err != nil {
		utils.RespondNotFound(c, slog.Default(), "Conversation")
		return
	}

	messages, err := queries.ListMessages(c.Request.Context(), pgtype.UUID{Bytes: id, Valid: true})
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "get messages", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"conversation": TransformConversation(conversation),
		"messages":     TransformMessages(messages),
	})
}

// DeleteConversation deletes a conversation
func (h *ChatHandler) DeleteConversation(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "conversation ID")
		return
	}

	queries := sqlc.New(h.pool)
	err = queries.DeleteConversation(c.Request.Context(), sqlc.DeleteConversationParams{
		ID:     pgtype.UUID{Bytes: id, Valid: true},
		UserID: user.ID,
	})
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "delete conversation", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Conversation deleted"})
}

// UpdateConversation updates a conversation's title or context
func (h *ChatHandler) UpdateConversation(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "conversation ID")
		return
	}

	var req struct {
		Title     *string `json:"title"`
		ContextID *string `json:"context_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondInvalidRequest(c, slog.Default(), err)
		return
	}

	queries := sqlc.New(h.pool)

	// Build update params
	var contextID pgtype.UUID
	if req.ContextID != nil {
		if *req.ContextID == "" {
			// Explicitly unlinking - set to NULL
			contextID = pgtype.UUID{Valid: false}
		} else {
			parsed, err := uuid.Parse(*req.ContextID)
			if err != nil {
				utils.RespondInvalidID(c, slog.Default(), "context_id")
				return
			}
			contextID = pgtype.UUID{Bytes: parsed, Valid: true}
		}
	}

	// Get current conversation to preserve existing values
	current, err := queries.GetConversation(c.Request.Context(), sqlc.GetConversationParams{
		ID:     pgtype.UUID{Bytes: id, Valid: true},
		UserID: user.ID,
	})
	if err != nil {
		utils.RespondNotFound(c, slog.Default(), "Conversation")
		return
	}

	// Use provided title or keep existing
	title := current.Title
	if req.Title != nil {
		title = req.Title
	}

	// Use provided contextID or keep existing
	if req.ContextID == nil {
		contextID = current.ContextID
	}

	conversation, err := queries.UpdateConversation(c.Request.Context(), sqlc.UpdateConversationParams{
		ID:        pgtype.UUID{Bytes: id, Valid: true},
		Title:     title,
		ContextID: contextID,
	})
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "update conversation", err)
		return
	}

	c.JSON(http.StatusOK, TransformConversation(conversation))
}

// SearchConversations searches across conversations
func (h *ChatHandler) SearchConversations(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	query := c.Query("q")
	if query == "" {
		utils.RespondBadRequest(c, slog.Default(), "Search query required")
		return
	}

	queries := sqlc.New(h.pool)
	conversations, err := queries.SearchConversations(c.Request.Context(), sqlc.SearchConversationsParams{
		UserID:  user.ID,
		Column2: &query,
	})
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "search conversations", err)
		return
	}

	c.JSON(http.StatusOK, conversations)
}

// Helper function to convert pgtype.UUID to string
func uuidToString(u pgtype.UUID) string {
	if !u.Valid {
		return ""
	}
	return uuid.UUID(u.Bytes).String()
}
