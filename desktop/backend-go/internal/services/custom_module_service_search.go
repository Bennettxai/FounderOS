package services

import (
	"context"
	"fmt"
)

// SearchModules searches public modules
func (s *CustomModuleService) SearchModules(
	ctx context.Context,
	query string,
	limit int,
	offset int,
) ([]CustomModule, error) {
	if limit <= 0 {
		limit = 20
	}

	sqlQuery := `
		SELECT
			id, created_by, workspace_id, name, slug, description, category,
			version, manifest, config, icon, tags, keywords, is_public, is_published,
			is_template, install_count, star_count, share_scope, created_at, updated_at, published_at
		FROM custom_modules
		WHERE is_public = TRUE AND is_published = TRUE
			AND (
				name ILIKE '%' || $1 || '%'
				OR description ILIKE '%' || $1 || '%'
				OR $1 = ANY(tags)
				OR $1 = ANY(keywords)
			)
		ORDER BY install_count DESC, created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := s.pool.Query(ctx, sqlQuery, query, limit, offset)
	if err != nil {
		s.logger.Error("Failed to search modules", "error", err, "query", query)
		return nil, fmt.Errorf("failed to search modules: %w", err)
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
