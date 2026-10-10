package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// AppTemplateService handles app template management
type AppTemplateService struct {
	pool   *pgxpool.Pool
	logger *slog.Logger
}

// AppTemplate represents an app template
type AppTemplate struct {
	ID                  uuid.UUID              `json:"id"`
	TemplateName        string                 `json:"template_name"`
	Category            string                 `json:"category"`
	DisplayName         string                 `json:"display_name"`
	Description         string                 `json:"description"`
	IconType            string                 `json:"icon_type"`
	TargetBusinessTypes []string               `json:"target_business_types"`
	TargetChallenges    []string               `json:"target_challenges"`
	TargetTeamSizes     []string               `json:"target_team_sizes"`
	PriorityScore       int                    `json:"priority_score"`
	TemplateConfig      map[string]interface{} `json:"template_config"`
	RequiredModules     []string               `json:"required_modules"`
	OptionalFeatures    []string               `json:"optional_features"`
	GenerationPrompt    string                 `json:"generation_prompt"`
	ScaffoldType        *string                `json:"scaffold_type"`
	CreatedAt           string                 `json:"created_at"`
	UpdatedAt           string                 `json:"updated_at"`
}

// AppTemplateFilters represents filters for listing templates
type AppTemplateFilters struct {
	Category     *string
	BusinessType *string
	Challenge    *string
	TeamSize     *string
}

// MatchedTemplate represents a template with a match score
type MatchedTemplate struct {
	Template   AppTemplate `json:"template"`
	MatchScore int         `json:"match_score"`
}

func NewAppTemplateService(pool *pgxpool.Pool, logger *slog.Logger) *AppTemplateService {
	return &AppTemplateService{
		pool:   pool,
		logger: logger,
	}
}

// ListTemplates returns all templates with optional filtering
func (s *AppTemplateService) ListTemplates(ctx context.Context, filters AppTemplateFilters) ([]AppTemplate, error) {
	query := `
		SELECT
			id,
			template_name,
			category,
			display_name,
			description,
			icon_type,
			target_business_types,
			target_challenges,
			target_team_sizes,
			priority_score,
			COALESCE(template_config, '{}'::jsonb),
			required_modules,
			optional_features,
			generation_prompt,
			scaffold_type,
			created_at,
			updated_at
		FROM app_templates
		WHERE 1=1`

	args := []interface{}{}
	argPos := 1

	if filters.Category != nil {
		query += fmt.Sprintf(" AND category = $%d", argPos)
		args = append(args, *filters.Category)
		argPos++
	}

	if filters.BusinessType != nil {
		query += fmt.Sprintf(" AND $%d = ANY(target_business_types)", argPos)
		args = append(args, *filters.BusinessType)
		argPos++
	}

	if filters.Challenge != nil {
		query += fmt.Sprintf(" AND $%d = ANY(target_challenges)", argPos)
		args = append(args, *filters.Challenge)
		argPos++
	}

	if filters.TeamSize != nil {
		query += fmt.Sprintf(" AND $%d = ANY(target_team_sizes)", argPos)
		args = append(args, *filters.TeamSize)
		argPos++
	}

	query += " ORDER BY priority_score DESC, created_at DESC"

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		s.logger.Error("failed to query templates", "error", err)
		return nil, fmt.Errorf("query templates: %w", err)
	}
	defer rows.Close()

	var templates []AppTemplate
	for rows.Next() {
		var template AppTemplate
		var configJSON []byte

		err := rows.Scan(
			&template.ID,
			&template.TemplateName,
			&template.Category,
			&template.DisplayName,
			&template.Description,
			&template.IconType,
			&template.TargetBusinessTypes,
			&template.TargetChallenges,
			&template.TargetTeamSizes,
			&template.PriorityScore,
			&configJSON,
			&template.RequiredModules,
			&template.OptionalFeatures,
			&template.GenerationPrompt,
			&template.ScaffoldType,
			&template.CreatedAt,
			&template.UpdatedAt,
		)
		if err != nil {
			s.logger.Error("failed to scan template", "error", err)
			return nil, fmt.Errorf("scan template: %w", err)
		}

		// Parse template config
		if len(configJSON) > 0 {
			if err := json.Unmarshal(configJSON, &template.TemplateConfig); err != nil {
				s.logger.Warn("failed to parse template config", "template_id", template.ID, "error", err)
			}
		}

		templates = append(templates, template)
	}

	if err := rows.Err(); err != nil {
		s.logger.Error("rows iteration error", "error", err)
		return nil, fmt.Errorf("rows iteration: %w", err)
	}

	return templates, nil
}

// GetTemplate returns a single template by ID
func (s *AppTemplateService) GetTemplate(ctx context.Context, templateID uuid.UUID) (*AppTemplate, error) {
	var template AppTemplate
	var configJSON []byte

	err := s.pool.QueryRow(ctx, `
		SELECT
			id,
			template_name,
			category,
			display_name,
			description,
			icon_type,
			target_business_types,
			target_challenges,
			target_team_sizes,
			priority_score,
			COALESCE(template_config, '{}'::jsonb),
			required_modules,
			optional_features,
			generation_prompt,
			scaffold_type,
			created_at,
			updated_at
		FROM app_templates
		WHERE id = $1
	`, templateID).Scan(
		&template.ID,
		&template.TemplateName,
		&template.Category,
		&template.DisplayName,
		&template.Description,
		&template.IconType,
		&template.TargetBusinessTypes,
		&template.TargetChallenges,
		&template.TargetTeamSizes,
		&template.PriorityScore,
		&configJSON,
		&template.RequiredModules,
		&template.OptionalFeatures,
		&template.GenerationPrompt,
		&template.ScaffoldType,
		&template.CreatedAt,
		&template.UpdatedAt,
	)

	if err != nil {
		s.logger.Error("failed to query template", "template_id", templateID, "error", err)
		return nil, fmt.Errorf("query template: %w", err)
	}

	// Parse template config
	if len(configJSON) > 0 {
		if err := json.Unmarshal(configJSON, &template.TemplateConfig); err != nil {
			s.logger.Warn("failed to parse template config", "template_id", template.ID, "error", err)
		}
	}

	return &template, nil
}

// GetRecommendedTemplates returns personalized template recommendations
func (s *AppTemplateService) GetRecommendedTemplates(ctx context.Context, userID uuid.UUID, workspaceID uuid.UUID, limit int) ([]MatchedTemplate, error) {
	if limit <= 0 {
		limit = 5
	}

	// Get user's onboarding profile to personalize recommendations
	var businessType, challenge, teamSize string
	err := s.pool.QueryRow(ctx, `
		SELECT
			COALESCE(business_type, ''),
			COALESCE(challenge, ''),
			COALESCE(team_size, '')
		FROM onboarding_profiles
		WHERE workspace_id = $1
		LIMIT 1
	`, workspaceID).Scan(&businessType, &challenge, &teamSize)

	if err != nil {
		s.logger.Info("no onboarding profile found, returning general recommendations", "workspace_id", workspaceID)
		// If no profile, return top templates by priority
		templates, err := s.ListTemplates(ctx, AppTemplateFilters{})
		if err != nil {
			return nil, err
		}

		var matched []MatchedTemplate
		for i, t := range templates {
			if i >= limit {
				break
			}
			matched = append(matched, MatchedTemplate{
				Template:   t,
				MatchScore: t.PriorityScore,
			})
		}
		return matched, nil
	}

	// Use the matching query with profile data
	rows, err := s.pool.Query(ctx, `
		SELECT
			id,
			template_name,
			category,
			display_name,
			description,
			icon_type,
			target_business_types,
			target_challenges,
			target_team_sizes,
			priority_score,
			COALESCE(template_config, '{}'::jsonb),
			required_modules,
			optional_features,
			generation_prompt,
			scaffold_type,
			created_at,
			updated_at,
			-- Calculate match score
			CASE
				WHEN $1::text = ANY(target_business_types) THEN 40
				ELSE 0
			END +
			CASE
				WHEN $2::text = ANY(target_challenges) THEN 30
				ELSE 0
			END +
			CASE
				WHEN $3::text = ANY(target_team_sizes) THEN 20
				ELSE 0
			END +
			priority_score as match_score
		FROM app_templates
		WHERE
			$1::text = ANY(target_business_types)
			OR $2::text = ANY(target_challenges)
			OR $3::text = ANY(target_team_sizes)
		ORDER BY match_score DESC
		LIMIT $4
	`, businessType, challenge, teamSize, limit)

	if err != nil {
		s.logger.Error("failed to query recommended templates", "error", err)
		return nil, fmt.Errorf("query recommended templates: %w", err)
	}
	defer rows.Close()

	var matched []MatchedTemplate
	for rows.Next() {
		var template AppTemplate
		var configJSON []byte
		var matchScore int

		err := rows.Scan(
			&template.ID,
			&template.TemplateName,
			&template.Category,
			&template.DisplayName,
			&template.Description,
			&template.IconType,
			&template.TargetBusinessTypes,
			&template.TargetChallenges,
			&template.TargetTeamSizes,
			&template.PriorityScore,
			&configJSON,
			&template.RequiredModules,
			&template.OptionalFeatures,
			&template.GenerationPrompt,
			&template.ScaffoldType,
			&template.CreatedAt,
			&template.UpdatedAt,
			&matchScore,
		)
		if err != nil {
			s.logger.Error("failed to scan recommended template", "error", err)
			return nil, fmt.Errorf("scan recommended template: %w", err)
		}

		// Parse template config
		if len(configJSON) > 0 {
			if err := json.Unmarshal(configJSON, &template.TemplateConfig); err != nil {
				s.logger.Warn("failed to parse template config", "template_id", template.ID, "error", err)
			}
		}

		matched = append(matched, MatchedTemplate{
			Template:   template,
			MatchScore: matchScore,
		})
	}

	if err := rows.Err(); err != nil {
		s.logger.Error("rows iteration error", "error", err)
		return nil, fmt.Errorf("rows iteration: %w", err)
	}

	return matched, nil
}
