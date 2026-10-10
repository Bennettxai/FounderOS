package services

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// templateExists checks if a template with the given name exists in the database.
func (s *TemplateSyncService) templateExists(ctx context.Context, name string) (bool, error) {
	var exists bool
	err := s.pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM app_templates WHERE template_name = $1
		)
	`, name).Scan(&exists)

	if err != nil {
		return false, fmt.Errorf("check existence query: %w", err)
	}

	return exists, nil
}

// insertTemplate inserts a new template into the database.
func (s *TemplateSyncService) insertTemplate(ctx context.Context, tmpl *DBTemplate) error {
	configJSON, err := json.Marshal(tmpl.TemplateConfig)
	if err != nil {
		return fmt.Errorf("marshal template_config: %w", err)
	}

	variablesJSON, err := json.Marshal(tmpl.TemplateVariables)
	if err != nil {
		return fmt.Errorf("marshal template_variables: %w", err)
	}

	_, err = s.pool.Exec(ctx, `
		INSERT INTO app_templates (
			template_name, category, display_name, description, icon_type,
			target_business_types, target_challenges, target_team_sizes,
			priority_score, template_config, required_modules, optional_features,
			generation_prompt, scaffold_type,
			yaml_template_name, yaml_version, template_variables
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17
		)
	`,
		tmpl.TemplateName,
		tmpl.Category,
		tmpl.DisplayName,
		tmpl.Description,
		tmpl.IconType,
		tmpl.TargetBusinessTypes,
		tmpl.TargetChallenges,
		tmpl.TargetTeamSizes,
		tmpl.PriorityScore,
		configJSON,
		tmpl.RequiredModules,
		tmpl.OptionalFeatures,
		tmpl.GenerationPrompt,
		tmpl.ScaffoldType,
		tmpl.YAMLTemplateName,
		tmpl.YAMLVersion,
		variablesJSON,
	)

	if err != nil {
		return fmt.Errorf("insert query: %w", err)
	}

	return nil
}

// updateTemplate updates an existing template in the database.
func (s *TemplateSyncService) updateTemplate(ctx context.Context, tmpl *DBTemplate) error {
	configJSON, err := json.Marshal(tmpl.TemplateConfig)
	if err != nil {
		return fmt.Errorf("marshal template_config: %w", err)
	}

	variablesJSON, err := json.Marshal(tmpl.TemplateVariables)
	if err != nil {
		return fmt.Errorf("marshal template_variables: %w", err)
	}

	_, err = s.pool.Exec(ctx, `
		UPDATE app_templates SET
			category = $2,
			display_name = $3,
			description = $4,
			icon_type = $5,
			target_business_types = $6,
			target_challenges = $7,
			target_team_sizes = $8,
			priority_score = $9,
			template_config = $10,
			required_modules = $11,
			optional_features = $12,
			generation_prompt = $13,
			scaffold_type = $14,
			yaml_template_name = $15,
			yaml_version = $16,
			template_variables = $17,
			updated_at = NOW()
		WHERE template_name = $1
	`,
		tmpl.TemplateName,
		tmpl.Category,
		tmpl.DisplayName,
		tmpl.Description,
		tmpl.IconType,
		tmpl.TargetBusinessTypes,
		tmpl.TargetChallenges,
		tmpl.TargetTeamSizes,
		tmpl.PriorityScore,
		configJSON,
		tmpl.RequiredModules,
		tmpl.OptionalFeatures,
		tmpl.GenerationPrompt,
		tmpl.ScaffoldType,
		tmpl.YAMLTemplateName,
		tmpl.YAMLVersion,
		variablesJSON,
	)

	if err != nil {
		return fmt.Errorf("update query: %w", err)
	}

	return nil
}

// GetTemplateByName retrieves a template from the database by name.
func (s *TemplateSyncService) GetTemplateByName(ctx context.Context, name string) (*DBTemplate, error) {
	var tmpl DBTemplate
	var configJSON, variablesJSON []byte

	err := s.pool.QueryRow(ctx, `
		SELECT
			id, template_name, category, display_name, description, icon_type,
			target_business_types, target_challenges, target_team_sizes,
			priority_score, COALESCE(template_config, '{}'::jsonb),
			required_modules, optional_features, generation_prompt,
			COALESCE(scaffold_type, ''), COALESCE(yaml_template_name, ''),
			COALESCE(yaml_version, ''), COALESCE(template_variables, '{}'::jsonb)
		FROM app_templates
		WHERE template_name = $1
	`, name).Scan(
		&tmpl.ID,
		&tmpl.TemplateName,
		&tmpl.Category,
		&tmpl.DisplayName,
		&tmpl.Description,
		&tmpl.IconType,
		&tmpl.TargetBusinessTypes,
		&tmpl.TargetChallenges,
		&tmpl.TargetTeamSizes,
		&tmpl.PriorityScore,
		&configJSON,
		&tmpl.RequiredModules,
		&tmpl.OptionalFeatures,
		&tmpl.GenerationPrompt,
		&tmpl.ScaffoldType,
		&tmpl.YAMLTemplateName,
		&tmpl.YAMLVersion,
		&variablesJSON,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("template not found: %s", name)
		}
		return nil, fmt.Errorf("query template: %w", err)
	}

	// Unmarshal JSON fields
	if len(configJSON) > 0 {
		if err := json.Unmarshal(configJSON, &tmpl.TemplateConfig); err != nil {
			s.logger.Warn("failed to unmarshal template_config", "name", name, "error", err)
		}
	}

	if len(variablesJSON) > 0 {
		if err := json.Unmarshal(variablesJSON, &tmpl.TemplateVariables); err != nil {
			s.logger.Warn("failed to unmarshal template_variables", "name", name, "error", err)
		}
	}

	return &tmpl, nil
}
