package handlers

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rhl/businessos-backend/internal/services"
)

// OSAWebhooksHandler handles incoming webhooks from OSA-5
type OSAWebhooksHandler struct {
	pool          *pgxpool.Pool
	webhookSecret string
	isProduction  bool                    // Enforces strict signature verification in production
	eventBus      *services.BuildEventBus // Event bus for real-time streaming
}

// NewOSAWebhooksHandler creates a new webhooks handler.
// isProduction must be true when ENVIRONMENT=production; unsigned webhooks are
// then rejected unconditionally, even when OSA_ALLOW_UNSIGNED_WEBHOOKS is set.
func NewOSAWebhooksHandler(pool *pgxpool.Pool, webhookSecret string, eventBus *services.BuildEventBus, isProduction bool) (*OSAWebhooksHandler, error) {
	if webhookSecret == "" {
		return nil, fmt.Errorf("OSA_WEBHOOK_SECRET environment variable is required. Generate with: openssl rand -hex 32")
	}
	return &OSAWebhooksHandler{
		pool:          pool,
		webhookSecret: webhookSecret,
		isProduction:  isProduction,
		eventBus:      eventBus,
	}, nil
}

// WebhookEvent represents an incoming webhook event from OSA-5
type WebhookEvent struct {
	EventType  string                 `json:"event_type"`
	WorkflowID string                 `json:"workflow_id"`
	Timestamp  time.Time              `json:"timestamp"`
	Status     string                 `json:"status"`
	Data       map[string]interface{} `json:"data"`
	Signature  string                 `json:"signature,omitempty"`
}

// HandleWorkflowComplete handles workflow completion webhooks
// POST /api/osa/webhooks/workflow-complete
func (h *OSAWebhooksHandler) HandleWorkflowComplete(c *gin.Context) {
	// Read request body
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read request body"})
		return
	}

	// Verify signature
	signature := c.GetHeader("X-OSA-Signature")
	if !h.verifySignature(body, signature) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid signature"})
		return
	}

	// Parse webhook event
	var event WebhookEvent
	if err := json.Unmarshal(body, &event); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON payload"})
		return
	}

	// Verify timestamp to prevent replay attacks
	if err := h.verifyTimestamp(event.Timestamp); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid timestamp"})
		return
	}

	// Process the event
	ctx := c.Request.Context()
	if err := h.processWorkflowComplete(ctx, &event); err != nil {
		RespondInternalErr(c, "process workflow complete webhook", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":     "Webhook processed successfully",
		"event_type":  event.EventType,
		"workflow_id": event.WorkflowID,
	})
}

// HandleBuildEvent handles build progress webhooks
// POST /api/osa/webhooks/build-event
func (h *OSAWebhooksHandler) HandleBuildEvent(c *gin.Context) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read request body"})
		return
	}

	signature := c.GetHeader("X-OSA-Signature")
	if !h.verifySignature(body, signature) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid signature"})
		return
	}

	var event WebhookEvent
	if err := json.Unmarshal(body, &event); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON payload"})
		return
	}

	// Verify timestamp to prevent replay attacks
	if err := h.verifyTimestamp(event.Timestamp); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid timestamp"})
		return
	}

	ctx := c.Request.Context()
	if err := h.processBuildEvent(ctx, &event); err != nil {
		RespondInternalErr(c, "process build event webhook", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":     "Build event processed",
		"workflow_id": event.WorkflowID,
	})
}

// verifySignature verifies the HMAC signature of the webhook.
//
// Production (isProduction == true): a missing or invalid signature ALWAYS
// results in rejection. The OSA_ALLOW_UNSIGNED_WEBHOOKS flag is ignored.
//
// Development: a missing signature is accepted only when
// OSA_ALLOW_UNSIGNED_WEBHOOKS=true, and a warning is logged.
func (h *OSAWebhooksHandler) verifySignature(body []byte, signature string) bool {
	if signature == "" {
		if h.isProduction {
			slog.Error("OSA webhook rejected: signature missing in production",
				"action", "rejected")
			return false
		}

		// Development only: allow unsigned webhooks when explicitly opted in.
		// OSA_ALLOW_UNSIGNED_WEBHOOKS must never be set in a production environment.
		allowUnsigned := os.Getenv("OSA_ALLOW_UNSIGNED_WEBHOOKS") == "true"
		if allowUnsigned {
			slog.Warn("OSA webhook signature missing - accepting in development mode only",
				"env_var", "OSA_ALLOW_UNSIGNED_WEBHOOKS=true",
				"warning", "NEVER set this variable in production")
			return true
		}

		slog.Error("OSA webhook rejected: signature missing",
			"hint", "Set OSA_ALLOW_UNSIGNED_WEBHOOKS=true for local dev only (NOT PRODUCTION)")
		return false
	}

	mac := hmac.New(sha256.New, []byte(h.webhookSecret))
	mac.Write(body)
	expectedMAC := hex.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(signature), []byte(expectedMAC))
}

// verifyTimestamp verifies the webhook timestamp to prevent replay attacks
func (h *OSAWebhooksHandler) verifyTimestamp(timestamp time.Time) error {
	now := time.Now()
	age := now.Sub(timestamp)

	// Webhook is too old (replay attack?)
	if age > 5*time.Minute {
		return fmt.Errorf("webhook expired: %v old (max 5 minutes)", age)
	}

	// Webhook timestamp is in the future (clock skew?)
	if timestamp.After(now.Add(1 * time.Minute)) {
		return fmt.Errorf("webhook timestamp %v is in the future", timestamp.Sub(now))
	}

	return nil
}

// processWorkflowComplete handles workflow completion events
func (h *OSAWebhooksHandler) processWorkflowComplete(ctx context.Context, event *WebhookEvent) error {
	// Validate workflow ID format (prevent SQL injection)
	workflowUUID, err := uuid.Parse(event.WorkflowID)
	if err != nil {
		return fmt.Errorf("invalid workflow_id format: %w", err)
	}

	// Update workflow status in database
	query := `
		UPDATE osa_module_instances
		SET
			status = $1,
			build_status = $2,
			generated_at = COALESCE(generated_at, NOW()),
			updated_at = NOW()
		WHERE osa_workflow_id = $3
		RETURNING id, workspace_id
	`

	var appID uuid.UUID
	var workspaceID *uuid.UUID
	err = h.pool.QueryRow(ctx, query,
		event.Status,
		event.Data["build_status"],
		workflowUUID,
	).Scan(&appID, &workspaceID)

	if err != nil {
		// Workflow might not exist yet - it will be picked up by polling
		return nil
	}

	// Create build event
	eventData, _ := json.Marshal(event.Data)
	buildEventQuery := `
		INSERT INTO osa_build_events (
			module_instance_id,
			event_type,
			event_data,
			phase,
			progress_percent,
			status_message
		) VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`

	phase := "completion"
	progressPercent := 100
	statusMessage := "Workflow completed successfully"

	if event.Status == "failed" {
		statusMessage = "Workflow failed"
		if msg, ok := event.Data["error_message"].(string); ok {
			statusMessage = msg
		}
	}

	var buildEventID uuid.UUID
	err = h.pool.QueryRow(ctx, buildEventQuery,
		appID,
		event.EventType,
		eventData,
		phase,
		progressPercent,
		statusMessage,
	).Scan(&buildEventID)

	if err != nil {
		return err
	}

	// Broadcast completion event to SSE subscribers
	if h.eventBus != nil {
		buildEvent := services.BuildEvent{
			ID:              buildEventID,
			AppID:           appID,
			WorkspaceID:     workspaceID,
			EventType:       event.EventType,
			Phase:           phase,
			ProgressPercent: progressPercent,
			StatusMessage:   statusMessage,
			Data:            event.Data,
			Timestamp:       time.Now(),
		}
		h.eventBus.Publish(buildEvent)
	}

	return nil
}

// processBuildEvent handles build progress events
func (h *OSAWebhooksHandler) processBuildEvent(ctx context.Context, event *WebhookEvent) error {
	// Validate workflow ID format (prevent SQL injection)
	workflowUUID, err := uuid.Parse(event.WorkflowID)
	if err != nil {
		return fmt.Errorf("invalid workflow_id format: %w", err)
	}

	// Find the app by workflow ID
	var appID uuid.UUID
	var workspaceID *uuid.UUID
	err = h.pool.QueryRow(ctx,
		`SELECT id, workspace_id FROM osa_module_instances WHERE osa_workflow_id = $1`,
		workflowUUID,
	).Scan(&appID, &workspaceID)

	if err != nil {
		// App not found - will be picked up later
		return nil
	}

	// Insert build event
	eventData, _ := json.Marshal(event.Data)
	phase := "building"
	progressPercent := 0

	if p, ok := event.Data["progress"].(float64); ok {
		progressPercent = int(p)
	}

	statusMessage := ""
	if msg, ok := event.Data["message"].(string); ok {
		statusMessage = msg
	}

	query := `
		INSERT INTO osa_build_events (
			module_instance_id,
			event_type,
			event_data,
			phase,
			progress_percent,
			status_message
		) VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`

	var buildEventID uuid.UUID
	err = h.pool.QueryRow(ctx, query,
		appID,
		event.EventType,
		eventData,
		phase,
		progressPercent,
		statusMessage,
	).Scan(&buildEventID)

	if err != nil {
		return err
	}

	// Broadcast event to SSE subscribers
	if h.eventBus != nil {
		buildEvent := services.BuildEvent{
			ID:              buildEventID,
			AppID:           appID,
			WorkspaceID:     workspaceID,
			EventType:       event.EventType,
			Phase:           phase,
			ProgressPercent: progressPercent,
			StatusMessage:   statusMessage,
			Data:            event.Data,
			Timestamp:       time.Now(),
		}
		h.eventBus.Publish(buildEvent)
	}

	return nil
}

// RegisterWebhook registers a new webhook configuration
// POST /api/osa/webhooks/register
func (h *OSAWebhooksHandler) RegisterWebhook(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var req struct {
		WorkspaceID *uuid.UUID `json:"workspace_id"`
		AppID       *uuid.UUID `json:"app_id"`
		EventType   string     `json:"event_type"`
		WebhookURL  string     `json:"webhook_url"`
	}

	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Generate secret key for this webhook
	secretKey := uuid.New().String()

	query := `
		INSERT INTO osa_webhooks (
			workspace_id,
			module_instance_id,
			event_type,
			webhook_url,
			secret_key,
			enabled
		) VALUES ($1, $2, $3, $4, $5, true)
		RETURNING id
	`

	var webhookID uuid.UUID
	err := h.pool.QueryRow(c.Request.Context(), query,
		req.WorkspaceID,
		req.AppID,
		req.EventType,
		req.WebhookURL,
		secretKey,
	).Scan(&webhookID)

	if err != nil {
		RespondInternalErr(c, "register webhook", err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":         webhookID,
		"secret_key": secretKey,
		"message":    "Webhook registered successfully",
		"user_id":    userID,
	})
}

// ListWebhooks lists all webhooks for the user
// GET /api/osa/webhooks
func (h *OSAWebhooksHandler) ListWebhooks(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	query := `
		SELECT
			wh.id,
			wh.event_type,
			wh.webhook_url,
			wh.enabled,
			wh.last_triggered_at,
			wh.success_count,
			wh.failure_count,
			wh.created_at
		FROM osa_webhooks wh
		LEFT JOIN osa_workspaces w ON wh.workspace_id = w.id
		WHERE w.user_id = $1 OR wh.workspace_id IS NULL
		ORDER BY wh.created_at DESC
	`

	rows, err := h.pool.Query(c.Request.Context(), query, userID)
	if err != nil {
		RespondInternalErr(c, "fetch webhooks", err)
		return
	}
	defer rows.Close()

	type WebhookInfo struct {
		ID              uuid.UUID  `json:"id"`
		EventType       string     `json:"event_type"`
		WebhookURL      string     `json:"webhook_url"`
		Enabled         bool       `json:"enabled"`
		LastTriggeredAt *time.Time `json:"last_triggered_at"`
		SuccessCount    int        `json:"success_count"`
		FailureCount    int        `json:"failure_count"`
		CreatedAt       time.Time  `json:"created_at"`
	}

	webhooks := []WebhookInfo{}
	for rows.Next() {
		var wh WebhookInfo
		err := rows.Scan(
			&wh.ID,
			&wh.EventType,
			&wh.WebhookURL,
			&wh.Enabled,
			&wh.LastTriggeredAt,
			&wh.SuccessCount,
			&wh.FailureCount,
			&wh.CreatedAt,
		)
		if err != nil {
			RespondInternalErr(c, "scan webhook row", err)
			return
		}
		webhooks = append(webhooks, wh)
	}

	c.JSON(http.StatusOK, gin.H{
		"webhooks": webhooks,
		"count":    len(webhooks),
	})
}
