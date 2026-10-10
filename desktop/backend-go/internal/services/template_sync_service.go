package services

import (
	"log/slog"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// TemplateSyncService handles synchronization of YAML templates to database.
type TemplateSyncService struct {
	pool         *pgxpool.Pool
	logger       *slog.Logger
	templatesDir string
}

// DBTemplate represents an app template in the database.
type DBTemplate struct {
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
	ScaffoldType        string                 `json:"scaffold_type"`
	YAMLTemplateName    string                 `json:"yaml_template_name"`
	YAMLVersion         string                 `json:"yaml_version"`
	TemplateVariables   map[string]interface{} `json:"template_variables"`
}

// SyncResult contains the results of a sync operation.
type SyncResult struct {
	Inserted int      `json:"inserted"`
	Updated  int      `json:"updated"`
	Skipped  int      `json:"skipped"`
	Errors   []string `json:"errors"`
}

// NewTemplateSyncService creates a new template sync service.
func NewTemplateSyncService(pool *pgxpool.Pool, logger *slog.Logger, templatesDir string) *TemplateSyncService {
	return &TemplateSyncService{
		pool:         pool,
		logger:       logger,
		templatesDir: templatesDir,
	}
}

// stringSliceContains checks if a string slice contains a value.
func stringSliceContains(slice []string, value string) bool {
	for _, item := range slice {
		if item == value {
			return true
		}
	}
	return false
}
