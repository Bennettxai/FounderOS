package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostOnboardingService handles app generation after onboarding completes
type PostOnboardingService struct {
	pool   *pgxpool.Pool
	logger *slog.Logger
}

// OnboardingProfile represents user's onboarding data
type OnboardingProfile struct {
	WorkspaceID            uuid.UUID
	BusinessType           string
	TeamSize               string
	OwnerRole              string
	MainChallenge          string
	RecommendedIntegrations []string
}

// TemplateMatch represents a matched template with score
type TemplateMatch struct {
	Template   AppTemplate
	MatchScore int
	Reasons    []string
}

func NewPostOnboardingService(pool *pgxpool.Pool, logger *slog.Logger) *PostOnboardingService {
	return &PostOnboardingService{
		pool:   pool,
		logger: logger,
	}
}

// QueueAppsForWorkspace matches templates and queues apps for generation
func (s *PostOnboardingService) QueueAppsForWorkspace(ctx context.Context, workspaceID uuid.UUID) error {
	s.logger.Info("starting post-onboarding app generation", "workspace_id", workspaceID)

	// 1. Fetch onboarding profile
	profile, err := s.getOnboardingProfile(ctx, workspaceID)
	if err != nil {
		return fmt.Errorf("fetch onboarding profile: %w", err)
	}

	s.logger.Info("fetched onboarding profile",
		"workspace_id", workspaceID,
		"business_type", profile.BusinessType,
		"challenge", profile.MainChallenge,
		"team_size", profile.TeamSize,
	)

	// 2. Match templates
	matches, err := s.matchTemplates(ctx, profile)
	if err != nil {
		return fmt.Errorf("match templates: %w", err)
	}

	s.logger.Info("matched templates", "workspace_id", workspaceID, "count", len(matches))

	// 3. Queue top N templates (max 5)
	maxApps := 5
	if len(matches) < maxApps {
		maxApps = len(matches)
	}

	for i := 0; i < maxApps; i++ {
		match := matches[i]
		err := s.queueTemplateGeneration(ctx, workspaceID, match, profile)
		if err != nil {
			s.logger.Error("failed to queue template",
				"workspace_id", workspaceID,
				"template", match.Template.TemplateName,
				"error", err,
			)
			continue // Don't fail entire process for one template
		}

		s.logger.Info("queued template for generation",
			"workspace_id", workspaceID,
			"template", match.Template.TemplateName,
			"match_score", match.MatchScore,
		)
	}

	s.logger.Info("post-onboarding queue complete",
		"workspace_id", workspaceID,
		"templates_queued", maxApps,
	)

	return nil
}

// getOnboardingProfile fetches the onboarding profile for a workspace
func (s *PostOnboardingService) getOnboardingProfile(ctx context.Context, workspaceID uuid.UUID) (*OnboardingProfile, error) {
	var profile OnboardingProfile
	var recommendedIntegrationsJSON []byte

	err := s.pool.QueryRow(ctx, `
		SELECT
			workspace_id,
			business_type,
			team_size,
			owner_role,
			main_challenge,
			recommended_integrations
		FROM workspace_onboarding_profiles
		WHERE workspace_id = $1
		LIMIT 1
	`, workspaceID).Scan(
		&profile.WorkspaceID,
		&profile.BusinessType,
		&profile.TeamSize,
		&profile.OwnerRole,
		&profile.MainChallenge,
		&recommendedIntegrationsJSON,
	)

	if err != nil {
		return nil, fmt.Errorf("query onboarding profile: %w", err)
	}

	// Parse recommended integrations
	if recommendedIntegrationsJSON != nil {
		json.Unmarshal(recommendedIntegrationsJSON, &profile.RecommendedIntegrations)
	}

	return &profile, nil
}

// matchTemplates finds best matching templates for user profile
func (s *PostOnboardingService) matchTemplates(ctx context.Context, profile *OnboardingProfile) ([]TemplateMatch, error) {
	// Fetch all templates (they're pre-seeded, small dataset)
	templates, err := s.getAllTemplates(ctx)
	if err != nil {
		return nil, err
	}

	// Score each template
	var matches []TemplateMatch
	for _, template := range templates {
		score, reasons := s.scoreTemplate(template, profile)
		if score > 0 {
			matches = append(matches, TemplateMatch{
				Template:   template,
				MatchScore: score,
				Reasons:    reasons,
			})
		}
	}

	// Sort by score descending
	// Simple bubble sort since list is small (<20 items)
	for i := 0; i < len(matches); i++ {
		for j := i + 1; j < len(matches); j++ {
			if matches[j].MatchScore > matches[i].MatchScore {
				matches[i], matches[j] = matches[j], matches[i]
			}
		}
	}

	return matches, nil
}

// scoreTemplate calculates match score for a template
func (s *PostOnboardingService) scoreTemplate(template AppTemplate, profile *OnboardingProfile) (int, []string) {
	score := template.PriorityScore // Start with base priority
	var reasons []string

	// Business type match (strongest signal - +40 points)
	if containsString(template.TargetBusinessTypes, profile.BusinessType) {
		score += 40
		reasons = append(reasons, fmt.Sprintf("Designed for %s", profile.BusinessType))
	}

	// Challenge match (strong signal - +30 points)
	if containsPartialString(template.TargetChallenges, profile.MainChallenge) {
		score += 30
		reasons = append(reasons, fmt.Sprintf("Solves %s", profile.MainChallenge))
	}

	// Team size match (moderate signal - +20 points)
	if containsString(template.TargetTeamSizes, profile.TeamSize) {
		score += 20
		reasons = append(reasons, fmt.Sprintf("Suited for %s team", profile.TeamSize))
	}

	// Apply special business rules
	score = s.applyBusinessRules(template, profile, score)

	return score, reasons
}

// applyBusinessRules applies domain-specific scoring adjustments
func (s *PostOnboardingService) applyBusinessRules(template AppTemplate, profile *OnboardingProfile, score int) int {
	// Solo teams don't need collaboration tools
	if profile.TeamSize == "solo" && containsString(template.RequiredModules, "team") {
		score -= 50
	}

	// Agencies always benefit from client portals
	if profile.BusinessType == "agency" && template.TemplateName == "client_portal" {
		score += 35
	}

	// Consulting needs time tracking for billing
	if profile.BusinessType == "consulting" && template.TemplateName == "time_tracker" {
		score += 30
	}

	// Ecommerce needs analytics
	if profile.BusinessType == "ecommerce" && template.Category == "analytics" {
		score += 25
	}

	// Startups need project management
	if profile.BusinessType == "startup" && template.TemplateName == "project_kanban" {
		score += 25
	}

	return score
}

// getAllTemplates fetches all app templates
func (s *PostOnboardingService) getAllTemplates(ctx context.Context) ([]AppTemplate, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT
			id,
			template_name,
			category,
			display_name,
			COALESCE(description, ''),
			COALESCE(icon_type, ''),
			target_business_types,
			target_challenges,
			target_team_sizes,
			priority_score,
			COALESCE(template_config, '{}'::jsonb),
			required_modules,
			optional_features,
			COALESCE(generation_prompt, ''),
			scaffold_type
		FROM app_templates
		ORDER BY priority_score DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("query templates: %w", err)
	}
	defer rows.Close()

	var templates []AppTemplate
	for rows.Next() {
		var t AppTemplate
		var configJSON []byte

		err := rows.Scan(
			&t.ID,
			&t.TemplateName,
			&t.Category,
			&t.DisplayName,
			&t.Description,
			&t.IconType,
			&t.TargetBusinessTypes,
			&t.TargetChallenges,
			&t.TargetTeamSizes,
			&t.PriorityScore,
			&configJSON,
			&t.RequiredModules,
			&t.OptionalFeatures,
			&t.GenerationPrompt,
			&t.ScaffoldType,
		)
		if err != nil {
			return nil, fmt.Errorf("scan template: %w", err)
		}

		// Parse config JSON
		if len(configJSON) > 0 {
			json.Unmarshal(configJSON, &t.TemplateConfig)
		}

		templates = append(templates, t)
	}

	return templates, nil
}

// queueTemplateGeneration adds a template to the generation queue
func (s *PostOnboardingService) queueTemplateGeneration(
	ctx context.Context,
	workspaceID uuid.UUID,
	match TemplateMatch,
	profile *OnboardingProfile,
) error {
	// Create generation context (snapshot of onboarding data)
	context := map[string]interface{}{
		"business_type":           profile.BusinessType,
		"team_size":                profile.TeamSize,
		"owner_role":               profile.OwnerRole,
		"main_challenge":           profile.MainChallenge,
		"recommended_integrations": profile.RecommendedIntegrations,
		"match_score":              match.MatchScore,
		"match_reasons":            match.Reasons,
	}

	contextJSON, _ := json.Marshal(context)

	_, err := s.pool.Exec(ctx, `
		INSERT INTO app_generation_queue (
			workspace_id,
			template_id,
			status,
			priority,
			generation_context
		) VALUES ($1, $2, 'pending', $3, $4)
	`, workspaceID, match.Template.ID, match.MatchScore, contextJSON)

	if err != nil {
		return fmt.Errorf("insert queue item: %w", err)
	}

	return nil
}

// Helper functions

func containsString(slice []string, item string) bool {
	for _, s := range slice {
		if strings.EqualFold(s, item) {
			return true
		}
	}
	return false
}

func containsPartialString(slice []string, item string) bool {
	itemLower := strings.ToLower(item)
	for _, s := range slice {
		sLower := strings.ToLower(s)
		if strings.Contains(itemLower, sLower) || strings.Contains(sLower, itemLower) {
			return true
		}
	}
	return false
}
