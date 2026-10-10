package services

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rhl/businessos-backend/internal/appgen"
	"github.com/rhl/businessos-backend/internal/database/sqlc"
	"golang.org/x/sync/semaphore"
)

// AppGenerationOrchestrator wraps appgen.Orchestrator with production patterns
type AppGenerationOrchestrator struct {
	orchestrator appgen.Orchestrator

	// Database access for persisting generated files
	queries *sqlc.Queries
	pool    *pgxpool.Pool

	// Resource management
	apiSem     *semaphore.Weighted // Rate limiting
	maxRetries int

	// Progress tracking
	eventBus *BuildEventBus

	// Workspace management
	workspaceManager *WorkspaceManager

	// Metrics
	mu          sync.RWMutex
	totalRuns   int64
	successRuns int64
	failedRuns  int64

	logger *slog.Logger
}

// MultiAgentAppRequest represents a multi-agent app generation request
type MultiAgentAppRequest struct {
	AppName     string    `json:"app_name"`
	Description string    `json:"description"`
	Features    []string  `json:"features,omitempty"`
	QueueItemID string    `json:"queue_item_id"` // For event routing
	WorkspaceID uuid.UUID `json:"workspace_id"`  // For FK-safe app record creation
}

// NewAppGenerationOrchestrator creates a production-ready orchestrator
func NewAppGenerationOrchestrator(pool *pgxpool.Pool, queries *sqlc.Queries, eventBus *BuildEventBus, workspaceBaseDir string) *AppGenerationOrchestrator {
	if workspaceBaseDir == "" {
		workspaceBaseDir = "/tmp/businessos-agent-workspaces"
	}

	return &AppGenerationOrchestrator{
		orchestrator:     appgen.NewOrchestrator(pool),
		queries:          queries,
		pool:             pool,
		apiSem:           semaphore.NewWeighted(50), // Max 50 concurrent API calls
		maxRetries:       5,
		eventBus:         eventBus,
		workspaceManager: NewWorkspaceManager(workspaceBaseDir, slog.Default()),
		logger:           slog.Default(),
	}
}

// Generate processes an app generation request with full production patterns
func (o *AppGenerationOrchestrator) Generate(ctx context.Context, req MultiAgentAppRequest) (*appgen.GeneratedApp, error) {
	startTime := time.Now()

	o.logger.InfoContext(ctx, "starting app generation",
		slog.String("app_name", req.AppName),
		slog.String("queue_item_id", req.QueueItemID),
	)

	// Emit initial event
	o.emitEvent(req.QueueItemID, "generation_started", map[string]interface{}{
		"status":   "started",
		"message":  "Starting app generation...",
		"progress": 0.0,
	})

	// Set up progress callback to stream events
	o.orchestrator.SetProgressCallback(func(event appgen.ProgressEvent) {
		o.handleProgressEvent(req.QueueItemID, event)
	})

	// Convert request
	agentReq := appgen.AppRequest{
		AppName:     req.AppName,
		Description: req.Description,
		Features:    req.Features,
	}

	// Step 1: Create plan with retry
	o.emitEvent(req.QueueItemID, "planning", map[string]interface{}{
		"status":   "planning",
		"message":  "Creating architectural plan...",
		"progress": 0.1,
	})

	var plan *appgen.Plan
	err := o.withRetry(ctx, func() error {
		p, err := o.orchestrator.CreatePlan(ctx, agentReq)
		if err != nil {
			return err
		}
		plan = p
		return nil
	})

	if err != nil {
		o.recordFailure()
		o.emitEvent(req.QueueItemID, "error", map[string]interface{}{
			"error":   fmt.Sprintf("Planning failed: %v", err),
			"message": "Planning failed",
		})
		return nil, fmt.Errorf("create plan failed: %w", err)
	}

	o.logger.InfoContext(ctx, "plan created",
		slog.Int("tasks", len(plan.Tasks)),
	)

	// A-01: Enrich hardcoded task descriptions with user request context
	enrichTaskDescriptions(plan, req)

	// Step 2: Execute with parallel workers
	o.emitEvent(req.QueueItemID, "executing", map[string]interface{}{
		"status":   "executing",
		"message":  fmt.Sprintf("Executing %d agents in parallel...", len(plan.Tasks)),
		"progress": 0.3,
	})

	var result *appgen.GeneratedApp
	err = o.withRetry(ctx, func() error {
		r, err := o.orchestrator.Execute(ctx, plan)
		if err != nil {
			return err
		}
		result = r
		return nil
	})

	if err != nil {
		o.recordFailure()
		o.emitEvent(req.QueueItemID, "error", map[string]interface{}{
			"error":   fmt.Sprintf("Execution failed: %v", err),
			"message": "Execution failed",
		})
		return nil, fmt.Errorf("execute failed: %w", err)
	}

	// Check if all agents succeeded
	if !result.Success {
		o.recordFailure()
		o.emitEvent(req.QueueItemID, "error", map[string]interface{}{
			"error":   result.ErrorMessage,
			"message": "Generation failed",
		})
		return result, fmt.Errorf("generation failed: %s", result.ErrorMessage)
	}

	// Step 3: Save generated files to workspace AND database
	appID, err := parseQueueItemID(req.QueueItemID)
	if err != nil {
		o.logger.WarnContext(ctx, "invalid queue item ID, skipping file save", "error", err)
	} else {
		// Step 3a: Create the module instance row BEFORE saving files.
		// Without this row, saveFileToDatabase() hits an FK violation because
		// osa_module_files.module_instance_id references osa_module_instances.id.
		//
		// We use raw SQL with explicit ID so the app_id matches the queue_item_id
		// that the file-saving loop uses. The SQLC-generated CreateOSAModuleInstance
		// auto-generates a UUID which would not match.
		if o.queries != nil {
			status := "generating"
			buildStatus := "pending"

			// Determine workspace_id: use request value, fall back to appID
			wsID := req.WorkspaceID
			if wsID == uuid.Nil {
				wsID = appID
			}

			_, createErr := o.pool.Exec(ctx, `
				INSERT INTO osa_module_instances (
					id, workspace_id, name, display_name, description,
					status, build_status, metadata
				) VALUES ($1, $2, $3, $4, $5, $6, $7, $8::jsonb)
				ON CONFLICT (id) DO NOTHING
			`, appID, wsID, req.AppName, req.AppName, req.Description, status, buildStatus, nil)

			if createErr != nil {
				o.logger.ErrorContext(ctx, "failed to create module instance row",
					"app_id", appID.String(),
					"workspace_id", wsID.String(),
					"error", createErr,
				)
			} else {
				o.logger.InfoContext(ctx, "created module instance row for file persistence",
					"app_id", appID.String(),
					"workspace_id", wsID.String(),
					"app_name", req.AppName,
				)
			}
		}

		o.emitEvent(req.QueueItemID, "saving_files", map[string]interface{}{
			"status":   "saving",
			"message":  "Saving generated files to workspace...",
			"progress": 0.9,
		})

		workspacePath, err := o.workspaceManager.CreateWorkspace(appID)
		if err != nil {
			o.logger.ErrorContext(ctx, "failed to create workspace",
				"app_id", appID.String(),
				"error", err,
			)
		} else {
			totalFiles := 0
			totalSize := int64(0)
			var savedFilePaths []string

			// Debug: Log agent results count and CodeBlocks status
			o.logger.InfoContext(ctx, "processing agent results",
				"app_id", appID.String(),
				"results_count", len(result.Results),
			)

			for i, agentResult := range result.Results {
				codeBlocksCount := 0
				if agentResult.CodeBlocks != nil {
					codeBlocksCount = len(agentResult.CodeBlocks)
				}
				o.logger.InfoContext(ctx, "agent result",
					"index", i,
					"agent_type", agentResult.AgentType,
					"code_blocks_count", codeBlocksCount,
					"output_length", len(agentResult.Output),
				)

				if agentResult.CodeBlocks != nil && len(agentResult.CodeBlocks) > 0 {
					for filePath, content := range agentResult.CodeBlocks {
						// Infer category and create proper path
						category := inferFileCategory(filePath)
						relativePath := fmt.Sprintf("%s/%s", category, filePath)

						// Save to filesystem
						if err := o.workspaceManager.SaveFile(workspacePath, relativePath, content); err != nil {
							o.logger.ErrorContext(ctx, "failed to save file to filesystem",
								"file", filePath,
								"error", err,
							)
							continue
						}

						// Save to database
						if o.queries != nil {
							if err := o.saveFileToDatabase(ctx, appID, relativePath, content); err != nil {
								o.logger.ErrorContext(ctx, "failed to save file to database",
									"file", filePath,
									"error", err,
								)
							} else {
								savedFilePaths = append(savedFilePaths, relativePath)
							}
						}

						totalFiles++
						totalSize += int64(len(content))
					}
				}
			}

			o.logger.InfoContext(ctx, "files saved to workspace and database",
				"app_id", appID.String(),
				"workspace", workspacePath,
				"total_files", totalFiles,
				"total_size_bytes", totalSize,
				"db_files", len(savedFilePaths),
			)

			result.WorkspacePath = workspacePath

			// Update app status to 'generated' after files are persisted
			if o.queries != nil && totalFiles > 0 {
				status := "generated"
				_, updateErr := o.queries.UpdateOSAModuleInstanceStatus(ctx, sqlc.UpdateOSAModuleInstanceStatusParams{
					ID:            pgtype.UUID{Bytes: appID, Valid: true},
					Status:        &status,
					ErrorMessage:  nil, // No error
					DeploymentUrl: nil, // Not deployed yet
				})
				if updateErr != nil {
					o.logger.ErrorContext(ctx, "failed to update app status to generated",
						"app_id", appID.String(),
						"error", updateErr,
					)
					o.emitEvent(req.QueueItemID, "status_update_failed", map[string]interface{}{
						"status":  "error",
						"error":   updateErr.Error(),
						"app_id":  appID.String(),
						"message": "Failed to update app status after file persistence",
					})
				} else {
					o.logger.InfoContext(ctx, "app status updated to generated",
						"app_id", appID.String(),
						"files_count", totalFiles,
					)
				}
			}
		}
	}

	// Success
	o.recordSuccess()

	duration := time.Since(startTime)
	o.logger.InfoContext(ctx, "app generation completed",
		slog.String("app_name", req.AppName),
		slog.Duration("duration", duration),
		slog.Bool("success", result.Success),
	)

	// Extract totalFiles and totalSize from result context (they're in scope from above)
	// Note: These variables are only set if workspace save succeeded
	fileCount := 0
	var fileSize int64 = 0
	if result.WorkspacePath != "" {
		// Variables totalFiles and totalSize are set in the workspace save block above
		// We need to pass them to the event. Since they're in the same function scope,
		// we can reference them here if they were set.
		// For safety, we'll count from result.Results as fallback
		for _, agentResult := range result.Results {
			if agentResult.CodeBlocks != nil {
				fileCount += len(agentResult.CodeBlocks)
				for _, content := range agentResult.CodeBlocks {
					fileSize += int64(len(content))
				}
			}
		}
	}

	o.emitEvent(req.QueueItemID, "generation_complete", map[string]interface{}{
		"status":           "completed",
		"message":          "App generated successfully",
		"progress":         1.0,
		"duration_ms":      duration.Milliseconds(),
		"files_created":    fileCount, // FIXED: Real file count, not agent count
		"total_size_bytes": fileSize,  // NEW: Total size of all files
		"workspace":        result.WorkspacePath,
	})

	// Clear replay buffer now that generation is complete — prevents indefinite
	// memory growth and avoids replaying stale events to future subscribers.
	if appID, err := parseQueueItemID(req.QueueItemID); err == nil && o.eventBus != nil {
		o.eventBus.ClearReplayBuffer(appID)
	}

	return result, nil
}
