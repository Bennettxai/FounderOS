package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rhl/businessos-backend/internal/middleware"
	"github.com/rhl/businessos-backend/internal/services"
)

// GenerateOSAAppRequest represents the request body for generating an OSA app via queue.
// template_id is OPTIONAL to support pure AI generative mode.
type GenerateOSAAppRequest struct {
	TemplateID  *string                `json:"template_id"` // Optional - omit for pure AI generation
	AppName     string                 `json:"app_name" binding:"required"`
	Description string                 `json:"description"`
	Features    []string               `json:"features,omitempty"` // Feature flags for generation (e.g. "auth", "payments")
	Config      map[string]interface{} `json:"config"`
	Complexity  string                 `json:"complexity"` // For generative mode: simple, standard, complex
}

// GenerateOSAAppResponse represents the response when queuing an app generation
type GenerateOSAAppResponse struct {
	QueueItemID uuid.UUID `json:"queue_item_id"`
	Status      string    `json:"status"`
	Message     string    `json:"message"`
}

// GenerateOSAApp - POST /api/workspaces/:id/apps/generate-osa
// Queues an OSA app generation request
func (h *OSAAppsHandler) GenerateOSAApp(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	workspaceIDStr := c.Param("id")
	workspaceID, err := uuid.Parse(workspaceIDStr)
	if err != nil {
		h.logger.Error("invalid workspace ID", "error", err, "workspace_id", workspaceIDStr)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid workspace ID"})
		return
	}

	// Verify workspace ownership
	pgWsID := pgtype.UUID{Bytes: workspaceID, Valid: true}
	workspace, err := h.queries.GetWorkspaceByID(c.Request.Context(), pgWsID)
	if err != nil {
		h.logger.Error("failed to verify workspace", "error", err, "workspace_id", workspaceID)
		c.JSON(http.StatusNotFound, gin.H{"error": "Workspace not found"})
		return
	}
	if workspace.OwnerID != user.ID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	// Parse request body
	var req GenerateOSAAppRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Build generation context
	generationContext := map[string]interface{}{
		"app_name":    req.AppName,
		"description": req.Description,
	}

	// Add complexity if provided (for generative mode)
	if req.Complexity != "" {
		generationContext["complexity"] = req.Complexity
	}

	// Add features if provided (queue worker extracts these into MultiAgentAppRequest.Features)
	if len(req.Features) > 0 {
		generationContext["features"] = req.Features
	}

	// Merge user-provided config
	if req.Config != nil {
		for k, v := range req.Config {
			generationContext[k] = v
		}
	}

	// GENERATIVE MODE: No template_id provided - pure AI generation
	if req.TemplateID == nil || *req.TemplateID == "" {
		h.logger.Info("queueing pure AI app generation (no template)",
			"workspace_id", workspaceID,
			"app_name", req.AppName,
			"user_id", user.ID,
			"complexity", req.Complexity,
		)

		// Build generation prompt from name and description
		prompt := req.AppName
		if req.Description != "" {
			prompt = fmt.Sprintf("%s: %s", req.AppName, req.Description)
		}
		generationContext["prompt"] = prompt
		generationContext["mode"] = "generative"

		// Use OSAPromptBuilder for rich structured prompt when available
		if h.promptBuilder != nil {
			promptReq := services.AppGenerationRequest{
				TemplateName: "app_generation",
				Variables: map[string]interface{}{
					"app_name":    req.AppName,
					"description": req.Description,
					"complexity":  req.Complexity,
					"features":    req.Features,
				},
			}
			// Pass user and workspace context for template resolution
			if userUUID, parseErr := uuid.Parse(user.ID); parseErr == nil {
				promptReq.UserID = &userUUID
			}
			promptReq.WorkspaceID = &workspaceID

			if result, pbErr := h.promptBuilder.BuildAppGenerationPrompt(c.Request.Context(), promptReq); pbErr != nil {
				h.logger.Warn("prompt builder failed, using fallback prompt", "error", pbErr)
			} else {
				generationContext["prompt"] = result.Prompt
				generationContext["prompt_template"] = result.TemplateName
				h.logger.Info("using OSA prompt builder for rich prompt",
					"template", result.TemplateName,
					"render_ms", result.RenderTimeMs,
				)
			}
		}

		// Marshal generation context to JSON
		contextJSON, err := json.Marshal(generationContext)
		if err != nil {
			h.logger.Error("failed to marshal generation context", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to prepare generation context"})
			return
		}

		// Insert into app_generation_queue WITHOUT template_id
		var queueItemID uuid.UUID
		err = h.pool.QueryRow(c.Request.Context(), `
			INSERT INTO app_generation_queue (
				workspace_id,
				status,
				priority,
				generation_context,
				max_retries
			) VALUES ($1, 'pending', 8, $2, 3)
			RETURNING id
		`, workspaceID, contextJSON).Scan(&queueItemID)

		if err != nil {
			h.logger.Error("failed to insert into app_generation_queue", "error", err, "workspace_id", workspaceID)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to queue app generation"})
			return
		}

		h.logger.Info("app generation queued (generative mode)",
			"queue_item_id", queueItemID,
			"workspace_id", workspaceID,
			"app_name", req.AppName,
		)

		// Immediately notify the local queue worker so it grabs the item
		// before the cloud worker can race for it
		if h.queueWorker != nil {
			h.queueWorker.NotifyNewItem(queueItemID.String())
		}

		c.JSON(http.StatusCreated, GenerateOSAAppResponse{
			QueueItemID: queueItemID,
			Status:      "pending",
			Message:     "App generation queued. AI will build your app from your description.",
		})
		return
	}

	// TEMPLATE MODE: template_id provided - use existing template-based flow
	templateID, err := uuid.Parse(*req.TemplateID)
	if err != nil {
		h.logger.Error("invalid template ID", "error", err, "template_id", *req.TemplateID)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid template ID"})
		return
	}

	h.logger.Info("queueing OSA app generation (template mode)",
		"workspace_id", workspaceID,
		"template_id", templateID,
		"app_name", req.AppName,
		"user_id", user.ID,
	)

	// Marshal generation context to JSON
	contextJSON, err := json.Marshal(generationContext)
	if err != nil {
		h.logger.Error("failed to marshal generation context", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to prepare generation context"})
		return
	}

	// Insert into app_generation_queue with template_id
	var queueItemID uuid.UUID
	err = h.pool.QueryRow(c.Request.Context(), `
		INSERT INTO app_generation_queue (
			workspace_id,
			template_id,
			status,
			priority,
			generation_context,
			max_retries
		) VALUES ($1, $2, 'pending', 8, $3, 3)
		RETURNING id
	`, workspaceID, templateID, contextJSON).Scan(&queueItemID)

	if err != nil {
		h.logger.Error("failed to insert into app_generation_queue", "error", err, "workspace_id", workspaceID, "template_id", templateID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to queue app generation"})
		return
	}

	h.logger.Info("app generation queued successfully",
		"queue_item_id", queueItemID,
		"workspace_id", workspaceID,
		"template_id", templateID,
		"app_name", req.AppName,
	)

	c.JSON(http.StatusCreated, GenerateOSAAppResponse{
		QueueItemID: queueItemID,
		Status:      "pending",
		Message:     "App generation request queued successfully. Use the queue_item_id to track progress via SSE.",
	})
}

// StreamAppGeneration - GET /api/osa/apps/generate/:queue_item_id/stream
// Streams real-time app generation progress via Server-Sent Events (SSE)
func (h *OSAAppsHandler) StreamAppGeneration(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Note: user.ID may be CUID (from Better Auth) or UUID format
	// We use the string ID for logging and uuid.Nil for the event bus subscription
	// since the subscription only uses UserID for logging purposes
	userIDStr := user.ID

	queueItemIDStr := c.Param("queue_item_id")
	queueItemID, err := uuid.Parse(queueItemIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid queue_item_id"})
		return
	}

	h.logger.Info("SSE stream requested",
		"queue_item_id", queueItemID,
		"user_id", userIDStr,
		"remote_addr", c.ClientIP(),
	)

	// Set SSE headers
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	// Create flusher
	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Streaming not supported"})
		return
	}

	// Check if eventBus is available
	if h.eventBus == nil {
		h.logger.Error("eventBus not configured")
		fmt.Fprintf(c.Writer, "data: {\"type\":\"error\",\"message\":\"Event bus not configured\"}\n\n")
		flusher.Flush()
		return
	}

	// Send initial connection event
	fmt.Fprintf(c.Writer, "data: {\"type\":\"connected\",\"message\":\"Stream connected\",\"queue_item_id\":\"%s\"}\n\n", queueItemID)
	flusher.Flush()

	// Subscribe to BuildEventBus for this app
	// Note: We use uuid.Nil for userID since the event bus only uses it for logging
	ctx := c.Request.Context()
	subscriber := h.eventBus.Subscribe(ctx, uuid.Nil, queueItemID)

	// Stream events until context is cancelled or generation completes
	h.logger.Info("subscribed to build events", "queue_item_id", queueItemID, "user_id", userIDStr, "subscriber_id", subscriber.ID)

	// Heartbeat ticker to keep SSE connection alive through proxies
	heartbeat := time.NewTicker(15 * time.Second)
	defer heartbeat.Stop()

	for {
		select {
		case <-ctx.Done():
			h.logger.Info("SSE stream cancelled by client", "queue_item_id", queueItemID)
			return

		case <-heartbeat.C:
			// Send SSE comment as keep-alive (not a data event, won't trigger onmessage)
			fmt.Fprintf(c.Writer, ": heartbeat\n\n")
			flusher.Flush()

		case event, ok := <-subscriber.Events:
			if !ok {
				// Channel closed - send final disconnect event
				h.logger.Info("event channel closed", "queue_item_id", queueItemID)
				fmt.Fprintf(c.Writer, "data: {\"type\":\"disconnected\",\"message\":\"Stream ended\"}\n\n")
				flusher.Flush()
				return
			}

			// Convert BuildEvent to SSE JSON
			// Frontend expects task_id and status at top level for AgentProgressPanel
			eventPayload := map[string]interface{}{
				"type":      event.EventType,
				"message":   event.StatusMessage,
				"progress":  event.ProgressPercent,
				"phase":     event.Phase,
				"timestamp": event.Timestamp.Format("2006-01-02T15:04:05Z"),
				"data":      event.Data,
			}
			// Copy task_id and status from data to top level (frontend expects these)
			// Map agent types to frontend-expected task IDs
			if event.Data != nil {
				if agentType, ok := event.Data["agent_type"].(string); ok {
					// Frontend expects "task-frontend", "task-backend", etc.
					eventPayload["task_id"] = "task-" + agentType
				} else if taskID, ok := event.Data["task_id"]; ok {
					eventPayload["task_id"] = taskID
				}
				if status, ok := event.Data["status"].(string); ok {
					// Map backend statuses to frontend-expected values
					// Frontend expects: pending, starting, in_progress, completed, failed
					eventPayload["status"] = mapStatusForFrontend(status)
				}
			}
			eventJSON, err := json.Marshal(eventPayload)
			if err != nil {
				h.logger.Error("failed to marshal event", "error", err, "event_type", event.EventType)
				continue
			}

			// Send SSE event
			fmt.Fprintf(c.Writer, "data: %s\n\n", eventJSON)
			flusher.Flush()

			h.logger.Info("sent SSE event",
				"queue_item_id", queueItemID,
				"event_type", event.EventType,
				"progress", event.ProgressPercent,
			)

			// If generation completed or failed, close stream after sending event
			if event.EventType == "generation_complete" || event.EventType == "error" {
				h.logger.Info("generation finished, closing SSE stream",
					"queue_item_id", queueItemID,
					"event_type", event.EventType,
				)
				return
			}
		}
	}
}
