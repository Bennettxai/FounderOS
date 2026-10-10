package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// GetRecommendations returns integration recommendations based on session data
func (s *OnboardingService) GetRecommendations(ctx context.Context, sessionID uuid.UUID, userID string) ([]string, error) {
	session, err := s.GetSession(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	if session.UserID != userID {
		return nil, fmt.Errorf("session does not belong to user")
	}

	// Parse extracted data
	var extractedData ExtractedOnboardingData
	if session.ExtractedData != nil {
		dataBytes, _ := json.Marshal(session.ExtractedData)
		json.Unmarshal(dataBytes, &extractedData)
	}

	return ComputeRecommendations(extractedData.Challenge, extractedData.BusinessType), nil
}

// ComputeRecommendations returns integration recommendations based on challenge and business type.
// This is the single source of truth for recommendation logic.
func ComputeRecommendations(challenge, businessType string) []string {
	challengeLower := strings.ToLower(challenge)

	if strings.Contains(challengeLower, "organiz") || strings.Contains(challengeLower, "chaos") || strings.Contains(challengeLower, "mess") {
		return []string{"notion", "google", "linear"}
	}
	if strings.Contains(challengeLower, "scale") || strings.Contains(challengeLower, "grow") || strings.Contains(challengeLower, "automat") {
		return []string{"linear", "slack", "airtable"}
	}
	if strings.Contains(challengeLower, "client") || strings.Contains(challengeLower, "customer") || strings.Contains(challengeLower, "crm") {
		return []string{"hubspot", "slack", "google"}
	}
	if strings.Contains(challengeLower, "team") || strings.Contains(challengeLower, "collaborat") || strings.Contains(challengeLower, "communic") {
		return []string{"slack", "notion", "linear"}
	}
	if strings.Contains(challengeLower, "time") || strings.Contains(challengeLower, "busy") || strings.Contains(challengeLower, "meeting") {
		return []string{"google", "fathom", "slack"}
	}

	// Default by business type
	switch businessType {
	case "agency", "consulting":
		return []string{"hubspot", "slack", "notion"}
	case "startup":
		return []string{"linear", "slack", "notion"}
	case "freelance":
		return []string{"google", "notion", "fathom"}
	default:
		return []string{"google", "slack", "notion"}
	}
}

// transformAIAnalysisToWorkspaceProfile extracts structured data from AI analysis
// and populates/updates the workspace_onboarding_profiles table
func (s *OnboardingService) transformAIAnalysisToWorkspaceProfile(
	ctx context.Context,
	workspaceID uuid.UUID,
	userID string,
) error {
	// Fetch onboarding analysis for this user/workspace
	var analysis struct {
		ID             uuid.UUID
		ProfileSummary string
		Insights       []byte
		ToolsUsed      []byte
	}

	err := s.pool.QueryRow(ctx, `
		SELECT id, profile_summary, insights, tools_used
		FROM onboarding_user_analysis
		WHERE user_id = $1 AND workspace_id = $2 AND status = 'completed'
		ORDER BY created_at DESC
		LIMIT 1
	`, userID, workspaceID).Scan(
		&analysis.ID,
		&analysis.ProfileSummary,
		&analysis.Insights,
		&analysis.ToolsUsed,
	)

	if err == pgx.ErrNoRows {
		// No AI analysis found, skip transformation
		slog.Info("No AI analysis found for workspace, skipping transformation",
			"workspace_id", workspaceID,
			"user_id", userID,
		)
		return nil
	}
	if err != nil {
		return fmt.Errorf("fetch analysis: %w", err)
	}

	// Extract structured fields from AI-generated profile summary
	businessType := extractBusinessTypeFromSummary(analysis.ProfileSummary)
	teamSize := extractTeamSizeFromSummary(analysis.ProfileSummary)
	ownerRole := extractOwnerRoleFromSummary(analysis.ProfileSummary)
	mainChallenge := extractMainChallengeFromInsights(analysis.Insights)

	// Extract recommended integrations from tools used
	var recommendedIntegrations []string
	if err := json.Unmarshal(analysis.ToolsUsed, &recommendedIntegrations); err != nil {
		slog.Warn("failed to unmarshal tools_used", "error", err)
		recommendedIntegrations = []string{}
	}

	// Convert to JSONB for database
	integrationsJSON, err := json.Marshal(recommendedIntegrations)
	if err != nil {
		integrationsJSON = nil
	}

	// Check if profile already exists
	var existingID uuid.UUID
	err = s.pool.QueryRow(ctx, `
		SELECT id FROM workspace_onboarding_profiles WHERE workspace_id = $1
	`, workspaceID).Scan(&existingID)

	if err == nil {
		// Update existing profile
		_, err = s.pool.Exec(ctx, `
			UPDATE workspace_onboarding_profiles
			SET business_type = $1,
			    team_size = $2,
			    owner_role = $3,
			    main_challenge = $4,
			    recommended_integrations = $5,
			    updated_at = NOW()
			WHERE workspace_id = $6
		`, businessType, teamSize, ownerRole, mainChallenge, integrationsJSON, workspaceID)

		if err != nil {
			return fmt.Errorf("update profile: %w", err)
		}

		slog.Info("workspace profile updated from AI analysis",
			"workspace_id", workspaceID,
			"business_type", businessType,
			"team_size", teamSize,
		)
		return nil
	}

	// Profile doesn't exist, but it should have been created in CompleteOnboarding
	// Log warning and skip
	if err == pgx.ErrNoRows {
		slog.Warn("workspace_onboarding_profiles entry not found, but should exist",
			"workspace_id", workspaceID,
		)
		return nil
	}

	return fmt.Errorf("check existing profile: %w", err)
}

// GetUserProfile retrieves the user's most recent onboarding profile from workspace_onboarding_profiles.
// This is used to inject personalised context into agent prompts.
func (s *OnboardingService) GetUserProfile(ctx context.Context, userID string) (*OnboardingProfileData, error) {
	// Query the most recent profile for this user
	// Join with workspaces to get the workspace_id
	var profile struct {
		BusinessType            string
		TeamSize                string
		OwnerRole               string
		MainChallenge           string
		RecommendedIntegrations []byte
	}

	// First, check if there's AI analysis data available
	analysisQuery := `
		SELECT
			profile_summary,
			insights,
			tools_used
		FROM onboarding_user_analysis
		WHERE user_id = $1 AND status = 'completed'
		ORDER BY created_at DESC
		LIMIT 1
	`

	var analysisSummary *string
	var analysisInsights []byte
	var analysisTools []byte

	err := s.pool.QueryRow(ctx, analysisQuery, userID).Scan(
		&analysisSummary,
		&analysisInsights,
		&analysisTools,
	)

	// AI analysis is optional, so we continue even if not found
	if err != nil && err != pgx.ErrNoRows {
		slog.Warn("Failed to fetch AI analysis for user", "user_id", userID, "error", err)
	}

	// Now fetch the profile from workspace_onboarding_profiles
	// Join with workspace_members to find the user's workspace
	query := `
		SELECT
			p.business_type,
			p.team_size,
			p.owner_role,
			p.main_challenge,
			p.recommended_integrations
		FROM workspace_onboarding_profiles p
		INNER JOIN workspace_members wm ON wm.workspace_id = p.workspace_id
		WHERE wm.user_id = $1 AND wm.status = 'active'
		ORDER BY p.created_at DESC
		LIMIT 1
	`

	err = s.pool.QueryRow(ctx, query, userID).Scan(
		&profile.BusinessType,
		&profile.TeamSize,
		&profile.OwnerRole,
		&profile.MainChallenge,
		&profile.RecommendedIntegrations,
	)

	if err == pgx.ErrNoRows {
		// No profile found - user hasn't completed onboarding
		slog.Debug("No onboarding profile found for user", "user_id", userID)
		return nil, fmt.Errorf("no onboarding profile found")
	}
	if err != nil {
		return nil, fmt.Errorf("fetch profile: %w", err)
	}

	// Build result
	result := &OnboardingProfileData{
		BusinessType:  profile.BusinessType,
		TeamSize:      profile.TeamSize,
		OwnerRole:     profile.OwnerRole,
		MainChallenge: profile.MainChallenge,
	}

	// Parse recommended integrations
	if len(profile.RecommendedIntegrations) > 0 {
		if err := json.Unmarshal(profile.RecommendedIntegrations, &result.RecommendedIntegrations); err != nil {
			slog.Warn("Failed to unmarshal recommended integrations", "error", err)
			result.RecommendedIntegrations = []string{}
		}
	}

	// Add AI analysis data if available
	if analysisSummary != nil {
		result.ProfileSummary = *analysisSummary
	}

	if len(analysisInsights) > 0 {
		if err := json.Unmarshal(analysisInsights, &result.Insights); err != nil {
			slog.Warn("Failed to unmarshal insights", "error", err)
		}
	}

	if len(analysisTools) > 0 {
		if err := json.Unmarshal(analysisTools, &result.ToolsUsed); err != nil {
			slog.Warn("Failed to unmarshal tools_used", "error", err)
		}
	}

	return result, nil
}

// BuildProfilePrefix constructs a prompt prefix from the user's profile.
// This is injected at the start of agent system prompts for personalisation.
func BuildProfilePrefix(profile *OnboardingProfileData) string {
	if profile == nil {
		return ""
	}

	var builder strings.Builder
	builder.WriteString("## USER PROFILE CONTEXT\n\n")
	builder.WriteString("You are assisting a user with the following profile:\n\n")

	// Business context
	builder.WriteString(fmt.Sprintf("**Business Type:** %s\n", profile.BusinessType))
	builder.WriteString(fmt.Sprintf("**Team Size:** %s\n", profile.TeamSize))
	builder.WriteString(fmt.Sprintf("**Role:** %s\n", profile.OwnerRole))
	builder.WriteString(fmt.Sprintf("**Main Challenge:** %s\n", profile.MainChallenge))

	// Tools/Integrations context
	if len(profile.RecommendedIntegrations) > 0 {
		builder.WriteString(fmt.Sprintf("\n**Preferred Tools:** %s\n", strings.Join(profile.RecommendedIntegrations, ", ")))
	}

	// AI insights (if available)
	if len(profile.Insights) > 0 {
		builder.WriteString("\n**Key Insights:**\n")
		for _, insight := range profile.Insights {
			builder.WriteString(fmt.Sprintf("- %s\n", insight))
		}
	}

	// Profile summary (if available)
	if profile.ProfileSummary != "" {
		builder.WriteString("\n**Profile Summary:**\n")
		builder.WriteString(profile.ProfileSummary)
		builder.WriteString("\n")
	}

	builder.WriteString("\n**IMPORTANT:** Use this profile information to personalize your responses and recommendations. Tailor your language, examples, and suggestions to match the user's business context, team size, and challenges.\n")
	builder.WriteString("\n---\n\n")

	return builder.String()
}

// Helper functions for extracting structured data from AI-generated text

func extractBusinessTypeFromSummary(summary string) string {
	summaryLower := strings.ToLower(summary)

	if strings.Contains(summaryLower, "agency") || strings.Contains(summaryLower, "consulting") {
		return "agency"
	}
	if strings.Contains(summaryLower, "startup") || strings.Contains(summaryLower, "tech") {
		return "startup"
	}
	if strings.Contains(summaryLower, "enterprise") || strings.Contains(summaryLower, "corporation") {
		return "enterprise"
	}
	if strings.Contains(summaryLower, "freelance") || strings.Contains(summaryLower, "contractor") {
		return "freelancer"
	}
	if strings.Contains(summaryLower, "ecommerce") || strings.Contains(summaryLower, "retail") {
		return "ecommerce"
	}

	return "other" // default
}

func extractTeamSizeFromSummary(summary string) string {
	summaryLower := strings.ToLower(summary)

	if strings.Contains(summaryLower, "solo") || strings.Contains(summaryLower, "individual") {
		return "1"
	}
	if strings.Contains(summaryLower, "small team") || strings.Contains(summaryLower, "2-10") {
		return "2-10"
	}
	if strings.Contains(summaryLower, "medium") || strings.Contains(summaryLower, "11-50") {
		return "11-50"
	}
	if strings.Contains(summaryLower, "large") || strings.Contains(summaryLower, "50+") {
		return "51+"
	}

	return "2-10" // default to small team
}

func extractOwnerRoleFromSummary(summary string) string {
	summaryLower := strings.ToLower(summary)

	if strings.Contains(summaryLower, "founder") || strings.Contains(summaryLower, "ceo") {
		return "founder"
	}
	if strings.Contains(summaryLower, "manager") || strings.Contains(summaryLower, "director") {
		return "manager"
	}
	if strings.Contains(summaryLower, "developer") || strings.Contains(summaryLower, "engineer") {
		return "developer"
	}
	if strings.Contains(summaryLower, "designer") {
		return "designer"
	}

	return "other"
}

func extractMainChallengeFromInsights(insightsJSON []byte) string {
	var insights []string
	if err := json.Unmarshal(insightsJSON, &insights); err != nil || len(insights) == 0 {
		return "productivity"
	}

	// Take first insight as main challenge
	firstInsight := strings.ToLower(insights[0])

	if strings.Contains(firstInsight, "time") || strings.Contains(firstInsight, "productivity") {
		return "productivity"
	}
	if strings.Contains(firstInsight, "team") || strings.Contains(firstInsight, "collaboration") {
		return "collaboration"
	}
	if strings.Contains(firstInsight, "communication") {
		return "communication"
	}
	if strings.Contains(firstInsight, "organization") || strings.Contains(firstInsight, "management") {
		return "organization"
	}

	return "productivity" // default
}
