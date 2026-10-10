package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rhl/businessos-backend/internal/database/sqlc"
	"github.com/rhl/businessos-backend/internal/integrations/osa"
)

// OSAQueueWorker processes app generation queue items in background
type OSAQueueWorker struct {
	pool           *pgxpool.Pool
	queries        *sqlc.Queries
	osaClient      *osa.ResilientClient // Legacy OSA client (fallback)
	eventBus       *BuildEventBus
	promptBuilder  *OSAPromptBuilder
	versionService *WorkspaceVersionService // Auto-snapshot after generation
	logger         *slog.Logger
	stopCh         chan struct{}
	notifyCh       chan string // Immediate notification when new item is queued
	pollInterval   time.Duration
	useMultiAgent  bool // Feature flag: use multi-agent vs OSA client
}

// NewOSAQueueWorker creates a new queue worker
func NewOSAQueueWorker(
	pool *pgxpool.Pool,
	queries *sqlc.Queries,
	osaClient *osa.ResilientClient,
	eventBus *BuildEventBus,
	promptBuilder *OSAPromptBuilder,
) *OSAQueueWorker {
	return &OSAQueueWorker{
		pool:          pool,
		queries:       queries,
		osaClient:     osaClient,
		eventBus:      eventBus,
		promptBuilder: promptBuilder,
		logger:        slog.Default().With("component", "osa_queue_worker"),
		stopCh:        make(chan struct{}),
		notifyCh:      make(chan string, 10), // Buffered channel for immediate processing
		pollInterval:  5 * time.Second,
		useMultiAgent: true,
	}
}

// Start begins processing queue items in background
func (w *OSAQueueWorker) Start(ctx context.Context) {
	w.logger.Info("OSA queue worker started", "poll_interval", w.pollInterval)

	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()

	// Process immediately on startup
	w.processQueue(ctx)

	for {
		select {
		case <-ticker.C:
			w.processQueue(ctx)
		case <-w.notifyCh:
			// Immediate processing when notified by HTTP handler
			w.logger.Info("OSA queue worker notified of new item, processing immediately")
			w.processQueue(ctx)
		case <-w.stopCh:
			w.logger.Info("OSA queue worker stopped")
			return
		case <-ctx.Done():
			w.logger.Info("OSA queue worker context cancelled")
			return
		}
	}
}

// NotifyNewItem signals the worker to immediately check for new items
func (w *OSAQueueWorker) NotifyNewItem(queueItemID string) {
	select {
	case w.notifyCh <- queueItemID:
		w.logger.Info("notified worker of new queue item", "queue_item_id", queueItemID)
	default:
		w.logger.Warn("worker notification channel full, relying on poll", "queue_item_id", queueItemID)
	}
}

// Stop gracefully stops the worker
func (w *OSAQueueWorker) Stop() {
	w.logger.Info("stopping OSA queue worker...")
	close(w.stopCh)
	w.logger.Info("OSA queue worker stopped")
}

// processQueue fetches and processes pending queue items
func (w *OSAQueueWorker) processQueue(ctx context.Context) {
	w.logger.Info("OSA queue worker polling for pending items")
	// Get next pending item with lock (FOR UPDATE SKIP LOCKED)
	item, err := w.queries.GetNextPendingItem(ctx)
	if err != nil {
		// Log only if it's a real error (not "no rows")
		if err.Error() != "no rows in result set" {
			w.logger.Error("failed to get pending item", "error", err, "error_type", fmt.Sprintf("%T", err))
		} else {
			w.logger.Debug("no pending items in queue")
		}
		return
	}

	w.logger.Info("processing queue item",
		"id", item.ID,
		"workspace_id", item.WorkspaceID,
		"template_id", item.TemplateID,
	)

	// Process the item
	if err := w.processQueueItem(ctx, item); err != nil {
		w.logger.Error("failed to process queue item",
			"id", item.ID,
			"error", err,
		)

		// Update error and retry count
		errorMsg := err.Error()
		_, updateErr := w.queries.UpdateQueueItemError(ctx, sqlc.UpdateQueueItemErrorParams{
			ID:           item.ID,
			ErrorMessage: &errorMsg,
		})
		if updateErr != nil {
			w.logger.Error("failed to update queue item error", "error", updateErr)
		}
		return
	}

	w.logger.Info("queue item completed successfully", "id", item.ID)
}

// processQueueItem processes a single queue item
func (w *OSAQueueWorker) processQueueItem(ctx context.Context, item sqlc.AppGenerationQueue) error {
	// Create a context with 10 minute timeout for complex generations
	// This allows enough time for multi-agent orchestration with multiple Claude API calls
	genCtx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()
	ctx = genCtx // Use the timeout context for all operations

	// 1. Update status to processing
	_, err := w.queries.UpdateQueueItemStatus(ctx, sqlc.UpdateQueueItemStatusParams{
		ID:     item.ID,
		Status: "processing",
	})
	if err != nil {
		return fmt.Errorf("failed to update status to processing: %w", err)
	}

	// 2. Parse generation context
	var config map[string]interface{}
	if len(item.GenerationContext) > 0 {
		if err := json.Unmarshal(item.GenerationContext, &config); err != nil {
			return fmt.Errorf("failed to parse generation context: %w", err)
		}
	}

	// 3. Build prompt
	prompt, err := w.buildPrompt(ctx, item, config)
	if err != nil {
		return fmt.Errorf("failed to build prompt: %w", err)
	}

	// 4. Publish start event
	if w.eventBus != nil {
		appID := pgUUIDToUUID(item.ID)
		wsID := pgUUIDToUUID(item.WorkspaceID)
		w.eventBus.Publish(BuildEvent{
			ID:              uuid.New(),
			AppID:           appID,
			WorkspaceID:     &wsID,
			EventType:       "started",
			ProgressPercent: 0,
			StatusMessage:   "App generation started",
			Data: map[string]interface{}{
				"template_id": item.TemplateID,
			},
			Timestamp: time.Now(),
		})
	}

	// 5. Extract app details
	// IMPORTANT: 'description' is used as the generation prompt for the orchestrator.
	// We use the rich prompt from buildPrompt() which includes user context, templates,
	// industry info, etc. — NOT the raw config["description"] which is just the short
	// user-entered text. The raw description is kept in 'rawDescription' for DB records.
	appName := "Generated App"
	description := prompt // Rich prompt from buildPrompt() — do NOT overwrite
	rawDescription := ""
	var features []string

	if name, ok := config["app_name"].(string); ok {
		appName = name
	}
	if desc, ok := config["description"].(string); ok {
		rawDescription = desc
		// If the rich prompt is empty or just the app name, use the raw description
		if description == "" || description == appName {
			description = desc
		}
	}
	_ = rawDescription // Used for DB record metadata, not for generation prompt
	if feat, ok := config["features"].([]interface{}); ok {
		for _, f := range feat {
			if fStr, ok := f.(string); ok {
				features = append(features, fStr)
			}
		}
	}

	w.logger.Info("processing app generation",
		"app_name", appName,
		"use_multi_agent", w.useMultiAgent,
	)

	// 6. Generate app (multi-agent vs OSA client)
	var generationErr error

	if w.useMultiAgent {
		// Multi-agent orchestration path
		// Create a FRESH orchestrator per queue item to avoid shared state bugs
		// (workers, callbacks, etc. must not carry over between runs)
		orchestrator := NewAppGenerationOrchestrator(w.pool, w.queries, w.eventBus, "")
		w.logger.Info("using multi-agent orchestrator", "app_name", appName)

		multiAgentReq := MultiAgentAppRequest{
			AppName:     appName,
			Description: description,
			Features:    features,
			QueueItemID: pgUUIDToUUID(item.ID).String(),
			WorkspaceID: pgUUIDToUUID(item.WorkspaceID),
		}

		// 8-minute timeout for entire generation (plan + 4 parallel agents + file save)
		// Per-agent timeout is 5min, so this must exceed it to avoid premature cancellation
		// Outer queue timeout is 10min, leaving 2min margin for status updates and cleanup
		genCtx, genCancel := context.WithTimeout(ctx, 8*time.Minute)
		defer genCancel()

		_, generationErr = orchestrator.Generate(genCtx, multiAgentReq)

	} else {
		// LEGACY: External OSA client path (fallback)
		w.logger.Info("using external OSA client", "app_name", appName)

		client := w.osaClient.GetClient()
		osaRequest := &osa.AppGenerationRequest{
			UserID:      pgUUIDToUUID(item.WorkspaceID),
			WorkspaceID: pgUUIDToUUID(item.WorkspaceID),
			Name:        appName,
			Description: description,
			Type:        "full-stack",
			Parameters:  config,
		}

		// Stream progress event for legacy path
		if w.eventBus != nil {
			appID := pgUUIDToUUID(item.ID)
			wsID := pgUUIDToUUID(item.WorkspaceID)
			w.eventBus.Publish(BuildEvent{
				ID:              uuid.New(),
				AppID:           appID,
				WorkspaceID:     &wsID,
				EventType:       "progress",
				ProgressPercent: 25,
				StatusMessage:   "Generating application code...",
				Timestamp:       time.Now(),
			})
		}

		response, err := w.generateApp(ctx, client, osaRequest)
		if err != nil {
			generationErr = fmt.Errorf("OSA client generation failed: %w", err)
		} else if response.Status != "completed" {
			generationErr = fmt.Errorf("OSA generation incomplete: %s", response.Status)
		}
	}

	// Handle generation errors
	if generationErr != nil {
		if w.eventBus != nil {
			appID := pgUUIDToUUID(item.ID)
			wsID := pgUUIDToUUID(item.WorkspaceID)
			w.eventBus.Publish(BuildEvent{
				ID:              uuid.New(),
				AppID:           appID,
				WorkspaceID:     &wsID,
				EventType:       "failed",
				ProgressPercent: 0,
				StatusMessage:   fmt.Sprintf("Generation failed: %s", generationErr.Error()),
				Timestamp:       time.Now(),
			})
		}
		return generationErr
	}

	// 7. Generation completed successfully
	w.logger.Info("app generation completed", "app_name", appName)

	// 7b. Auto-create initial snapshot (v0.0.1)
	if w.versionService != nil && item.WorkspaceID.Valid {
		wsID := pgUUIDToUUID(item.WorkspaceID)
		version, snapErr := w.versionService.CreateSnapshot(ctx, wsID, "system")
		if snapErr != nil {
			w.logger.Warn("failed to create auto-snapshot after generation",
				"workspace_id", wsID,
				"error", snapErr,
			)
		} else {
			w.logger.Info("auto-snapshot created after generation",
				"workspace_id", wsID,
				"version", version,
			)
		}
	}

	// 8. Mark as completed
	_, err = w.queries.MarkQueueItemCompleted(ctx, item.ID)
	if err != nil {
		return fmt.Errorf("failed to mark item as completed: %w", err)
	}

	// 9. Publish completion event (only needed for legacy path, multi-agent sends its own)
	if !w.useMultiAgent && w.eventBus != nil {
		appID := pgUUIDToUUID(item.ID)
		wsID := pgUUIDToUUID(item.WorkspaceID)
		w.eventBus.Publish(BuildEvent{
			ID:              uuid.New(),
			AppID:           appID,
			WorkspaceID:     &wsID,
			EventType:       "completed",
			ProgressPercent: 100,
			StatusMessage:   "App generated successfully",
			Timestamp:       time.Now(),
		})
	}

	return nil
}

// buildPrompt constructs the generation prompt from template or config.
// Priority order for the base prompt:
//  1. Template generation_prompt (when template_id is present and has a stored prompt)
//  2. config["prompt"] — the verbatim user message set by the HTTP handler
//  3. Constructed from config["app_name"] + config["description"]
//  4. Generic fallback (only when no user context exists at all)
func (w *OSAQueueWorker) buildPrompt(ctx context.Context, item sqlc.AppGenerationQueue, config map[string]interface{}) (string, error) {
	var basePrompt string

	// 1. If template_id is provided, try to get prompt from template
	if item.TemplateID.Valid && w.promptBuilder != nil {
		// Use item.TemplateID directly since it's already pgtype.UUID
		appTemplate, err := w.queries.GetAppTemplate(ctx, item.TemplateID)
		if err == nil && appTemplate.GenerationPrompt != nil {
			basePrompt = *appTemplate.GenerationPrompt
			w.logger.Info("buildPrompt: using template generation prompt",
				"template_id", item.TemplateID,
				"prompt_length", len(basePrompt),
			)
		}
	}

	// 2. Use the verbatim user prompt stored by the HTTP handler under "prompt".
	// This is set by GenerateOSAApp (osa_apps.go) and contains the user's actual
	// message, e.g. "Build me an invoicing app: tracks invoices, payments, clients".
	// Must be checked before falling back to the constructed prompt so the user's
	// original intent is never silently discarded.
	if basePrompt == "" {
		if userPrompt, ok := config["prompt"].(string); ok && userPrompt != "" {
			basePrompt = userPrompt
			w.logger.Info("buildPrompt: using user prompt from generation context",
				"prompt_length", len(basePrompt),
				"prompt_preview", truncatePromptForLog(basePrompt, 120),
			)
		}
	}

	// 3. Construct from app_name + description when explicit prompt is absent
	if basePrompt == "" {
		appName, hasName := config["app_name"].(string)
		if hasName && appName != "" {
			description := ""
			if desc, ok := config["description"].(string); ok {
				description = desc
			}
			basePrompt = fmt.Sprintf("Generate an application named '%s'.\n\nDescription: %s\n\nRequirements:\n- Modern architecture\n- Clean code\n- Proper error handling\n- Comprehensive tests", appName, description)
			w.logger.Info("buildPrompt: constructed prompt from app_name+description",
				"app_name", appName,
				"description_length", len(description),
			)
		}
	}

	// 4. Last-resort generic fallback — should only occur for onboarding-triggered
	// queue items that carry no user-supplied app intent at all.
	if basePrompt == "" {
		basePrompt = "Generate a basic application with best practices."
		w.logger.Warn("buildPrompt: falling back to generic prompt — no user prompt, app_name, or template found",
			"item_id", item.ID,
			"config_keys", configKeys(config),
		)
	}

	// Enrich with user context from onboarding
	userContext := w.fetchUserContext(ctx, item.WorkspaceID)
	if userContext != "" {
		basePrompt = basePrompt + "\n\n## User Context (from onboarding analysis)\n" + userContext
	}

	return basePrompt, nil
}

// truncatePromptForLog returns a preview of a prompt string safe for log output.
func truncatePromptForLog(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// configKeys returns the keys of a generation-context config map for logging.
func configKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// fetchUserContext fetches onboarding analysis data for the workspace
func (w *OSAQueueWorker) fetchUserContext(ctx context.Context, workspaceID pgtype.UUID) string {
	if !workspaceID.Valid {
		return ""
	}

	// Guard against nil queries (e.g. in tests without a DB connection)
	if w.queries == nil {
		return ""
	}

	session, err := w.queries.GetOnboardingSessionByWorkspace(ctx, workspaceID)
	if err != nil {
		w.logger.Debug("no onboarding context found for workspace", "error", err)
		return ""
	}

	// Parse extracted_data JSONB
	if len(session.ExtractedData) == 0 {
		return ""
	}

	var extractedData map[string]interface{}
	if err := json.Unmarshal(session.ExtractedData, &extractedData); err != nil {
		w.logger.Warn("failed to parse onboarding extracted_data", "error", err)
		return ""
	}

	// Build context string from extracted data
	var parts []string

	if role, ok := extractedData["role"].(string); ok && role != "" {
		parts = append(parts, fmt.Sprintf("- User role: %s", role))
	}
	if industry, ok := extractedData["industry"].(string); ok && industry != "" {
		parts = append(parts, fmt.Sprintf("- Industry: %s", industry))
	}
	if businessType, ok := extractedData["business_type"].(string); ok && businessType != "" {
		parts = append(parts, fmt.Sprintf("- Business type: %s", businessType))
	}
	if teamSize, ok := extractedData["team_size"].(string); ok && teamSize != "" {
		parts = append(parts, fmt.Sprintf("- Team size: %s", teamSize))
	}
	if tools, ok := extractedData["tools_used"].([]interface{}); ok && len(tools) > 0 {
		toolNames := make([]string, 0, len(tools))
		for _, t := range tools {
			if name, ok := t.(string); ok {
				toolNames = append(toolNames, name)
			}
		}
		if len(toolNames) > 0 {
			parts = append(parts, fmt.Sprintf("- Tools they currently use: %s", strings.Join(toolNames, ", ")))
		}
	}
	if needs, ok := extractedData["detected_needs"].([]interface{}); ok && len(needs) > 0 {
		needNames := make([]string, 0, len(needs))
		for _, n := range needs {
			if name, ok := n.(string); ok {
				needNames = append(needNames, name)
			}
		}
		if len(needNames) > 0 {
			parts = append(parts, fmt.Sprintf("- Detected business needs: %s", strings.Join(needNames, ", ")))
		}
	}
	if mainFocus, ok := extractedData["main_focus"].(string); ok && mainFocus != "" {
		parts = append(parts, fmt.Sprintf("- Main focus area: %s", mainFocus))
	}
	if challenge, ok := extractedData["challenge"].(string); ok && challenge != "" {
		parts = append(parts, fmt.Sprintf("- Key challenge: %s", challenge))
	}

	if len(parts) == 0 {
		return ""
	}

	w.logger.Info("enriching prompt with user context",
		"workspace_id", workspaceID,
		"context_fields", len(parts),
	)

	return strings.Join(parts, "\n")
}

// generateApp calls OSA Client to generate application
func (w *OSAQueueWorker) generateApp(ctx context.Context, client *osa.Client, request *osa.AppGenerationRequest) (*osa.AppGenerationResponse, error) {
	w.logger.Info("generating app via OSA client",
		"workspace_id", request.WorkspaceID,
		"app_name", request.Name,
	)

	// Use the injected OSA client to generate the app
	// The OSA Client expects *AppGenerationRequest directly
	response, err := client.GenerateApp(ctx, request)
	if err != nil {
		w.logger.Error("OSA client generation failed",
			"error", err,
			"workspace_id", request.WorkspaceID,
			"app_name", request.Name,
		)
		return nil, fmt.Errorf("OSA client generate app: %w", err)
	}

	w.logger.Info("app generation completed",
		"app_id", response.AppID,
		"status", response.Status,
	)

	return response, nil
}

// SetPollInterval changes the polling interval (useful for testing)
func (w *OSAQueueWorker) SetPollInterval(interval time.Duration) {
	w.pollInterval = interval
}

// SetVersionService sets the workspace version service for auto-snapshots
func (w *OSAQueueWorker) SetVersionService(vs *WorkspaceVersionService) {
	w.versionService = vs
}

// pgUUIDToUUID converts pgtype.UUID to uuid.UUID
func pgUUIDToUUID(pgUUID pgtype.UUID) uuid.UUID {
	if !pgUUID.Valid {
		return uuid.Nil
	}
	id, _ := uuid.FromBytes(pgUUID.Bytes[:])
	return id
}
