package services

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// CreateModule creates a new custom module
func (s *CustomModuleService) CreateModule(
	ctx context.Context,
	workspaceID uuid.UUID,
	userID uuid.UUID,
	req CreateModuleRequest,
) (*CustomModule, error) {
	// Generate slug from name
	slug := GenerateModuleSlug(req.Name)

	// Validate manifest
	if err := validateManifest(req.Manifest); err != nil {
		return nil, fmt.Errorf("invalid manifest: %w", err)
	}

	// Default config if not provided
	if req.Config == nil {
		req.Config = make(map[string]interface{})
	}

	// Marshal JSONB fields
	manifestJSON, err := json.Marshal(req.Manifest)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal manifest: %w", err)
	}

	configJSON, err := json.Marshal(req.Config)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal config: %w", err)
	}

	// Insert module
	query := `
		INSERT INTO custom_modules (
			created_by,
			workspace_id,
			name,
			slug,
			description,
			category,
			version,
			manifest,
			config,
			icon,
			tags,
			keywords
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12
		) RETURNING
			id, created_by, workspace_id, name, slug, description, category,
			version, manifest, config, icon, tags, keywords, is_public, is_published,
			is_template, install_count, star_count, share_scope, created_at, updated_at, published_at
	`

	row := s.pool.QueryRow(ctx, query,
		userID,
		workspaceID,
		req.Name,
		slug,
		req.Description,
		req.Category,
		"0.0.1", // Initial version
		manifestJSON,
		configJSON,
		req.Icon,
		req.Tags,
		req.Keywords,
	)

	module := &CustomModule{}
	err = row.Scan(
		&module.ID,
		&module.CreatedBy,
		&module.WorkspaceID,
		&module.Name,
		&module.Slug,
		&module.Description,
		&module.Category,
		&module.Version,
		&module.Manifest,
		&module.Config,
		&module.Icon,
		&module.Tags,
		&module.Keywords,
		&module.IsPublic,
		&module.IsPublished,
		&module.IsTemplate,
		&module.InstallCount,
		&module.StarCount,
		&module.ShareScope,
		&module.CreatedAt,
		&module.UpdatedAt,
		&module.PublishedAt,
	)

	if err != nil {
		s.logger.Error("Failed to create module", "error", err)
		return nil, fmt.Errorf("failed to create module: %w", err)
	}

	s.logger.Info("Module created", "module_id", module.ID, "name", module.Name)
	return module, nil
}

// GetModule retrieves a module by ID
func (s *CustomModuleService) GetModule(ctx context.Context, moduleID uuid.UUID) (*CustomModule, error) {
	query := `
		SELECT
			id, created_by, workspace_id, name, slug, description, category,
			version, manifest, config, icon, tags, keywords, is_public, is_published,
			is_template, install_count, star_count, share_scope, created_at, updated_at, published_at
		FROM custom_modules
		WHERE id = $1
	`

	module := &CustomModule{}
	err := s.pool.QueryRow(ctx, query, moduleID).Scan(
		&module.ID,
		&module.CreatedBy,
		&module.WorkspaceID,
		&module.Name,
		&module.Slug,
		&module.Description,
		&module.Category,
		&module.Version,
		&module.Manifest,
		&module.Config,
		&module.Icon,
		&module.Tags,
		&module.Keywords,
		&module.IsPublic,
		&module.IsPublished,
		&module.IsTemplate,
		&module.InstallCount,
		&module.StarCount,
		&module.ShareScope,
		&module.CreatedAt,
		&module.UpdatedAt,
		&module.PublishedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("module not found")
	}
	if err != nil {
		s.logger.Error("Failed to get module", "error", err, "module_id", moduleID)
		return nil, fmt.Errorf("failed to get module: %w", err)
	}

	return module, nil
}

// ListModules lists modules visible to a workspace: its own modules plus any
// modules from sibling workspaces in the same organization that have
// share_scope = 'organization'.
func (s *CustomModuleService) ListModules(
	ctx context.Context,
	workspaceID uuid.UUID,
	limit int,
	offset int,
) ([]CustomModule, error) {
	if limit <= 0 {
		limit = 20
	}

	// Include the workspace's own modules AND org-shared modules from siblings.
	// The subquery resolves the organization_id for the requesting workspace;
	// workspaces without an org (NULL) only see their own modules.
	query := `
		SELECT
			id, created_by, workspace_id, name, slug, description, category,
			version, manifest, config, icon, tags, keywords, is_public, is_published,
			is_template, install_count, star_count, share_scope, created_at, updated_at, published_at
		FROM custom_modules
		WHERE
			workspace_id = $1
			OR (
				share_scope = 'organization'
				AND workspace_id IN (
					SELECT w.id FROM workspaces w
					WHERE w.organization_id = (
						SELECT organization_id FROM workspaces WHERE id = $1
					)
					AND w.id != $1
				)
			)
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := s.pool.Query(ctx, query, workspaceID, limit, offset)
	if err != nil {
		s.logger.Error("Failed to list modules", "error", err, "workspace_id", workspaceID)
		return nil, fmt.Errorf("failed to list modules: %w", err)
	}
	defer rows.Close()

	var modules []CustomModule
	for rows.Next() {
		var module CustomModule
		err := rows.Scan(
			&module.ID,
			&module.CreatedBy,
			&module.WorkspaceID,
			&module.Name,
			&module.Slug,
			&module.Description,
			&module.Category,
			&module.Version,
			&module.Manifest,
			&module.Config,
			&module.Icon,
			&module.Tags,
			&module.Keywords,
			&module.IsPublic,
			&module.IsPublished,
			&module.IsTemplate,
			&module.InstallCount,
			&module.StarCount,
			&module.ShareScope,
			&module.CreatedAt,
			&module.UpdatedAt,
			&module.PublishedAt,
		)
		if err != nil {
			s.logger.Error("Failed to scan module", "error", err)
			continue
		}
		modules = append(modules, module)
	}

	return modules, nil
}

// UpdateModule updates a module
func (s *CustomModuleService) UpdateModule(
	ctx context.Context,
	moduleID uuid.UUID,
	userID uuid.UUID,
	req UpdateModuleRequest,
) (*CustomModule, error) {
	// First check ownership
	var createdBy string
	err := s.pool.QueryRow(ctx, "SELECT created_by FROM custom_modules WHERE id = $1", moduleID).Scan(&createdBy)
	if err != nil {
		return nil, fmt.Errorf("module not found")
	}
	if createdBy != userID.String() {
		return nil, fmt.Errorf("unauthorized: you don't own this module")
	}

	// Build dynamic update query
	updates := []string{}
	args := []interface{}{moduleID}
	argIdx := 2

	if req.Name != nil {
		updates = append(updates, fmt.Sprintf("name = $%d", argIdx))
		args = append(args, *req.Name)
		argIdx++

		// Update slug too
		slug := GenerateModuleSlug(*req.Name)
		updates = append(updates, fmt.Sprintf("slug = $%d", argIdx))
		args = append(args, slug)
		argIdx++
	}

	if req.Description != nil {
		updates = append(updates, fmt.Sprintf("description = $%d", argIdx))
		args = append(args, *req.Description)
		argIdx++
	}

	if req.Category != nil {
		updates = append(updates, fmt.Sprintf("category = $%d", argIdx))
		args = append(args, *req.Category)
		argIdx++
	}

	if req.Version != nil {
		updates = append(updates, fmt.Sprintf("version = $%d", argIdx))
		args = append(args, *req.Version)
		argIdx++
	}

	if req.Manifest != nil {
		manifestJSON, err := json.Marshal(*req.Manifest)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal manifest: %w", err)
		}
		updates = append(updates, fmt.Sprintf("manifest = $%d", argIdx))
		args = append(args, manifestJSON)
		argIdx++
	}

	if req.Config != nil {
		configJSON, err := json.Marshal(*req.Config)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal config: %w", err)
		}
		updates = append(updates, fmt.Sprintf("config = $%d", argIdx))
		args = append(args, configJSON)
		argIdx++
	}

	if req.Icon != nil {
		updates = append(updates, fmt.Sprintf("icon = $%d", argIdx))
		args = append(args, *req.Icon)
		argIdx++
	}

	if req.Tags != nil {
		updates = append(updates, fmt.Sprintf("tags = $%d", argIdx))
		args = append(args, *req.Tags)
		argIdx++
	}

	if req.Keywords != nil {
		updates = append(updates, fmt.Sprintf("keywords = $%d", argIdx))
		args = append(args, *req.Keywords)
		argIdx++
	}

	if req.IsPublic != nil {
		updates = append(updates, fmt.Sprintf("is_public = $%d", argIdx))
		args = append(args, *req.IsPublic)
		argIdx++
	}

	if req.IsTemplate != nil {
		updates = append(updates, fmt.Sprintf("is_template = $%d", argIdx))
		args = append(args, *req.IsTemplate)
		argIdx++
	}

	if req.ShareScope != nil {
		if *req.ShareScope != "workspace" && *req.ShareScope != "organization" {
			return nil, fmt.Errorf("share_scope must be 'workspace' or 'organization'")
		}
		updates = append(updates, fmt.Sprintf("share_scope = $%d", argIdx))
		args = append(args, *req.ShareScope)
		argIdx++
	}

	if len(updates) == 0 {
		return s.GetModule(ctx, moduleID)
	}

	updates = append(updates, "updated_at = NOW()")

	query := fmt.Sprintf(`
		UPDATE custom_modules SET %s
		WHERE id = $1
		RETURNING
			id, created_by, workspace_id, name, slug, description, category,
			version, manifest, config, icon, tags, keywords, is_public, is_published,
			is_template, install_count, star_count, share_scope, created_at, updated_at, published_at
	`, strings.Join(updates, ", "))

	module := &CustomModule{}
	err = s.pool.QueryRow(ctx, query, args...).Scan(
		&module.ID,
		&module.CreatedBy,
		&module.WorkspaceID,
		&module.Name,
		&module.Slug,
		&module.Description,
		&module.Category,
		&module.Version,
		&module.Manifest,
		&module.Config,
		&module.Icon,
		&module.Tags,
		&module.Keywords,
		&module.IsPublic,
		&module.IsPublished,
		&module.IsTemplate,
		&module.InstallCount,
		&module.StarCount,
		&module.ShareScope,
		&module.CreatedAt,
		&module.UpdatedAt,
		&module.PublishedAt,
	)

	if err != nil {
		s.logger.Error("Failed to update module", "error", err, "module_id", moduleID)
		return nil, fmt.Errorf("failed to update module: %w", err)
	}

	s.logger.Info("Module updated", "module_id", module.ID)
	return module, nil
}
