// Package handlers - OSA workflow and file operation handlers.
//
// File layout:
//   - osa_workflows.go         — OSAWorkflowsHandler struct and constructor
//   - osa_workflows_helpers.go — Shared constants and utility functions
//   - osa_workflows_list.go    — ListWorkflows, GetWorkflow
//   - osa_workflows_files.go   — GetWorkflowFiles, GetFileContent, GetFileContentByID
//   - osa_workflows_module.go  — InstallModule, TriggerSync
package handlers

import (
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rhl/businessos-backend/internal/services"
)

// OSAWorkflowsHandler handles OSA workflow and file operations.
type OSAWorkflowsHandler struct {
	pool        *pgxpool.Pool
	syncService *services.OSAFileSyncService
}

// NewOSAWorkflowsHandler creates a new workflows handler.
func NewOSAWorkflowsHandler(pool *pgxpool.Pool, syncService *services.OSAFileSyncService) *OSAWorkflowsHandler {
	return &OSAWorkflowsHandler{
		pool:        pool,
		syncService: syncService,
	}
}
