package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// OSAFileSyncService polls OSA-5 generated files and syncs to database
type OSAFileSyncService struct {
	pool              *pgxpool.Pool
	logger            *slog.Logger
	osaWorkspacePath  string
	pollInterval      time.Duration
	seenWorkflows     map[string]bool
	mu                sync.RWMutex
	stopChan          chan struct{}
	wg                sync.WaitGroup
	deploymentService *AppDeploymentService
}

// WorkflowFiles represents all files for a single workflow
type WorkflowFiles struct {
	WorkflowID      string
	Analysis        string
	Architecture    string
	Code            string
	Quality         string
	Deployment      string
	Monitoring      string
	Strategy        string
	Recommendations string
	DiscoveredAt    time.Time
}

// NewOSAFileSyncService creates a new file sync service
func NewOSAFileSyncService(pool *pgxpool.Pool, logger *slog.Logger, workspacePath string) *OSAFileSyncService {
	if workspacePath == "" {
		workspacePath = os.Getenv("OSA_WORKSPACE_PATH")
	}
	if workspacePath == "" {
		workspacePath = "./generated"
		logger.Warn("OSA_WORKSPACE_PATH not set, using default: ./generated")
	}

	return &OSAFileSyncService{
		pool:             pool,
		logger:           logger,
		osaWorkspacePath: workspacePath,
		pollInterval:     30 * time.Second,
		seenWorkflows:    make(map[string]bool),
		stopChan:         make(chan struct{}),
	}
}

// SetDeploymentService sets the deployment service for auto-deployment
func (s *OSAFileSyncService) SetDeploymentService(deploymentService *AppDeploymentService) {
	s.deploymentService = deploymentService
	s.logger.Info("Auto-deployment enabled for new workflows")
}

// Start begins polling for new workflow files
func (s *OSAFileSyncService) Start(ctx context.Context) {
	s.wg.Add(1)
	go s.pollLoop(ctx)
	s.logger.Info("OSA file sync service started",
		"workspace_path", s.osaWorkspacePath,
		"poll_interval", s.pollInterval)
}

// Stop gracefully stops the polling service
func (s *OSAFileSyncService) Stop() {
	close(s.stopChan)
	s.wg.Wait()
	s.logger.Info("OSA file sync service stopped")
}

func (s *OSAFileSyncService) pollLoop(ctx context.Context) {
	defer s.wg.Done()

	ticker := time.NewTicker(s.pollInterval)
	defer ticker.Stop()

	// Initial scan
	if err := s.scanWorkspace(ctx); err != nil {
		s.logger.Error("Initial workspace scan failed", "error", err)
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopChan:
			return
		case <-ticker.C:
			if err := s.scanWorkspace(ctx); err != nil {
				s.logger.Error("Workspace scan failed", "error", err)
			}
		}
	}
}

// scanWorkspace walks the OSA workspace directory and discovers new workflows
func (s *OSAFileSyncService) scanWorkspace(ctx context.Context) error {
	if _, err := os.Stat(s.osaWorkspacePath); os.IsNotExist(err) {
		s.logger.Warn("OSA workspace path does not exist", "path", s.osaWorkspacePath)
		return nil
	}

	workflowFiles := make(map[string]*WorkflowFiles)

	// Scan all subdirectories
	subdirs := []string{"analysis", "architecture", "code", "quality", "deployment", "monitoring", "strategy", "recommendations"}

	for _, subdir := range subdirs {
		dirPath := filepath.Join(s.osaWorkspacePath, subdir)
		if _, err := os.Stat(dirPath); os.IsNotExist(err) {
			continue
		}

		err := filepath.WalkDir(dirPath, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}

			// Extract workflow ID from filename
			// Format: {type}_{workflowID}.{ext}
			// Example: analysis_11af0132.md
			filename := d.Name()
			workflowID := s.extractWorkflowID(filename)
			if workflowID == "" {
				return nil
			}

			// Initialize workflow entry if not exists
			if workflowFiles[workflowID] == nil {
				workflowFiles[workflowID] = &WorkflowFiles{
					WorkflowID:   workflowID,
					DiscoveredAt: time.Now(),
				}
			}

			// Read file content
			content, err := os.ReadFile(path)
			if err != nil {
				s.logger.Warn("Failed to read file", "path", path, "error", err)
				return nil
			}

			// Assign content to appropriate field
			switch subdir {
			case "analysis":
				workflowFiles[workflowID].Analysis = string(content)
			case "architecture":
				workflowFiles[workflowID].Architecture = string(content)
			case "code":
				workflowFiles[workflowID].Code = string(content)
			case "quality":
				workflowFiles[workflowID].Quality = string(content)
			case "deployment":
				workflowFiles[workflowID].Deployment = string(content)
			case "monitoring":
				workflowFiles[workflowID].Monitoring = string(content)
			case "strategy":
				workflowFiles[workflowID].Strategy = string(content)
			case "recommendations":
				workflowFiles[workflowID].Recommendations = string(content)
			}

			return nil
		})

		if err != nil {
			s.logger.Error("Failed to walk directory", "dir", dirPath, "error", err)
		}
	}

	// Process discovered workflows
	for workflowID, files := range workflowFiles {
		if s.isWorkflowSeen(workflowID) {
			continue
		}

		s.logger.Info("New workflow discovered", "workflow_id", workflowID)
		if err := s.processWorkflow(ctx, files); err != nil {
			s.logger.Error("Failed to process workflow", "workflow_id", workflowID, "error", err)
		} else {
			s.markWorkflowSeen(workflowID)
		}
	}

	return nil
}

// extractWorkflowID extracts the workflow ID from a filename
// Format: {type}_{workflowID}.{ext} -> returns first 8 chars of workflow UUID
func (s *OSAFileSyncService) extractWorkflowID(filename string) string {
	// Remove extension
	name := strings.TrimSuffix(filename, filepath.Ext(filename))

	// Split by underscore
	parts := strings.Split(name, "_")
	if len(parts) < 2 {
		return ""
	}

	// Last part should be the workflow ID (8 chars from UUID)
	workflowIDPart := parts[len(parts)-1]
	if len(workflowIDPart) != 8 {
		return ""
	}

	return workflowIDPart
}

// processWorkflow stores workflow files in database
func (s *OSAFileSyncService) processWorkflow(ctx context.Context, files *WorkflowFiles) error {
	// Create metadata JSON with all file contents
	metadata := map[string]interface{}{
		"analysis":        files.Analysis,
		"architecture":    files.Architecture,
		"code":            files.Code,
		"quality":         files.Quality,
		"deployment":      files.Deployment,
		"monitoring":      files.Monitoring,
		"strategy":        files.Strategy,
		"recommendations": files.Recommendations,
		"discovered_at":   files.DiscoveredAt,
	}

	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	// Count files created
	filesCreated := 0
	if files.Analysis != "" {
		filesCreated++
	}
	if files.Architecture != "" {
		filesCreated++
	}
	if files.Code != "" {
		filesCreated++
	}
	if files.Quality != "" {
		filesCreated++
	}
	if files.Deployment != "" {
		filesCreated++
	}
	if files.Monitoring != "" {
		filesCreated++
	}
	if files.Strategy != "" {
		filesCreated++
	}
	if files.Recommendations != "" {
		filesCreated++
	}

	// Parse analysis to extract app name and description
	appName, description := s.parseAnalysis(files.Analysis)

	// Insert into osa_module_instances
	// Note: We need a workspace_id. For now, we'll need to look up the default workspace
	// or create one. This is a simplified version - in production, we'd track which user
	// initiated the workflow.

	// Look up workspace_id from osa_workflows table using workflow_id
	// If not found, fall back to first available workspace
	query := `
		INSERT INTO osa_module_instances (
			workspace_id,
			name,
			display_name,
			description,
			osa_workflow_id,
			status,
			files_created,
			metadata,
			generated_at
		) VALUES (
			COALESCE(
				(SELECT workspace_id FROM osa_workflows WHERE osa_workflow_id = $4 LIMIT 1),
				(SELECT id FROM osa_workspaces ORDER BY created_at DESC LIMIT 1)
			),
			$1,
			$2,
			$3,
			$4,
			'generated',
			$5,
			$6,
			NOW()
		)
		ON CONFLICT DO NOTHING
		RETURNING id
	`

	var appID uuid.UUID
	err = s.pool.QueryRow(ctx, query,
		appName,
		appName,
		description,
		files.WorkflowID,
		filesCreated,
		metadataJSON,
	).Scan(&appID)

	if err != nil {
		// Check if it's a "no workspace" error - this is a constraint violation (null value in not null column)
		// Use pgconn.PgError type assertion to check for specific PostgreSQL error codes
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23502" { // 23502 = not_null_violation
			s.logger.Warn("No workspace found - workflow will be processed when workspace is created",
				"workflow_id", files.WorkflowID)
			return nil
		}
		return fmt.Errorf("failed to insert workflow: %w", err)
	}

	s.logger.Info("Workflow synced to database",
		"workflow_id", files.WorkflowID,
		"app_id", appID,
		"files_created", filesCreated)

	// Create sync status entry
	syncQuery := `
		INSERT INTO osa_sync_status (
			entity_type,
			entity_id,
			osa_entity_id,
			osa_entity_type,
			sync_status,
			last_sync_at,
			sync_direction
		) VALUES (
			'app',
			$1,
			$2,
			'workflow',
			'synced',
			NOW(),
			'from_osa'
		)
		ON CONFLICT (entity_type, entity_id) DO UPDATE
		SET last_sync_at = NOW(),
		    sync_status = 'synced'
	`

	_, err = s.pool.Exec(ctx, syncQuery, appID, files.WorkflowID)
	if err != nil {
		s.logger.Warn("Failed to create sync status", "error", err)
		// Non-fatal - app is already created
	}

	// Auto-deploy the app if deployment service is available
	if s.deploymentService != nil && files.Code != "" {
		s.logger.Info("Auto-deploying app", "app_id", appID, "workflow_id", files.WorkflowID)

		// Deploy in background to avoid blocking the sync loop
		s.wg.Add(1)
		go func(id uuid.UUID) {
			defer s.wg.Done()

			// Create deployment context with timeout, derived from parent context
			deployCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
			defer cancel()

			deployedApp, err := s.deploymentService.DeployApp(deployCtx, id)
			if err != nil {
				s.logger.Error("Auto-deployment failed",
					"app_id", id,
					"error", err)
			} else {
				s.logger.Info("Auto-deployment successful",
					"app_id", id,
					"url", deployedApp.URL,
					"port", deployedApp.Port)
			}
		}(appID)
	}

	return nil
}

// parseAnalysis extracts app name and description from analysis file
func (s *OSAFileSyncService) parseAnalysis(analysis string) (name string, description string) {
	if analysis == "" {
		return "Generated App", "Application generated by OSA-5"
	}

	// Try to extract app name from first line or heading
	lines := strings.Split(analysis, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Remove markdown heading markers
		line = strings.TrimPrefix(line, "#")
		line = strings.TrimSpace(line)
		if line != "" {
			name = line
			break
		}
	}

	if name == "" {
		name = "Generated App"
	}

	// Use first paragraph as description
	description = "Application generated by OSA-5"
	inContent := false
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			if inContent {
				break // End of first paragraph
			}
			continue
		}
		if strings.HasPrefix(line, "#") {
			continue // Skip headings
		}
		if !inContent {
			inContent = true
			description = line
		} else {
			description += " " + line
		}
		if len(description) > 500 {
			description = description[:500] + "..."
			break
		}
	}

	return name, description
}

// isWorkflowSeen checks if a workflow has been processed
func (s *OSAFileSyncService) isWorkflowSeen(workflowID string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.seenWorkflows[workflowID]
}

// markWorkflowSeen marks a workflow as processed
func (s *OSAFileSyncService) markWorkflowSeen(workflowID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.seenWorkflows[workflowID] = true
}

// GetWorkflowStatus retrieves the status of a workflow from database
func (s *OSAFileSyncService) GetWorkflowStatus(ctx context.Context, workflowID string) (*WorkflowStatus, error) {
	query := `
		SELECT
			id,
			name,
			display_name,
			description,
			status,
			files_created,
			build_status,
			error_message,
			created_at,
			generated_at,
			deployed_at
		FROM osa_module_instances
		WHERE osa_workflow_id = $1
	`

	var status WorkflowStatus
	var generatedAt, deployedAt *time.Time

	err := s.pool.QueryRow(ctx, query, workflowID).Scan(
		&status.ID,
		&status.Name,
		&status.DisplayName,
		&status.Description,
		&status.Status,
		&status.FilesCreated,
		&status.BuildStatus,
		&status.ErrorMessage,
		&status.CreatedAt,
		&generatedAt,
		&deployedAt,
	)

	if err != nil {
		return nil, err
	}

	if generatedAt != nil {
		status.GeneratedAt = generatedAt
	}
	if deployedAt != nil {
		status.DeployedAt = deployedAt
	}

	return &status, nil
}

// WorkflowStatus represents the current status of a workflow
type WorkflowStatus struct {
	ID           uuid.UUID  `json:"id"`
	Name         string     `json:"name"`
	DisplayName  string     `json:"display_name"`
	Description  string     `json:"description"`
	Status       string     `json:"status"`
	FilesCreated int        `json:"files_created"`
	BuildStatus  *string    `json:"build_status"`
	ErrorMessage *string    `json:"error_message"`
	CreatedAt    time.Time  `json:"created_at"`
	GeneratedAt  *time.Time `json:"generated_at"`
	DeployedAt   *time.Time `json:"deployed_at"`
}
