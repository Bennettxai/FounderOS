package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/google/uuid"
)

// CompleteOnboarding completes the onboarding and creates the workspace
func (s *OnboardingService) CompleteOnboarding(ctx context.Context, sessionID uuid.UUID, userID string, integrations []string) (*CompleteOnboardingResponse, error) {
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

	// Add integrations
	extractedData.Integrations = integrations

	// Apply defaults for missing required fields (allows "skip" flow)
	if extractedData.WorkspaceName == "" {
		extractedData.WorkspaceName = "My Workspace"
	}
	if extractedData.BusinessType == "" {
		extractedData.BusinessType = "other"
	}
	if extractedData.TeamSize == "" {
		extractedData.TeamSize = "solo"
	}

	// Validate integrations if provided
	if len(integrations) > 0 {
		if err := s.validator.ValidateIntegrations(integrations); err != nil {
			return nil, fmt.Errorf("invalid integrations: %s", err.Message)
		}
	}

	// Start transaction
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Create workspace
	workspaceName := extractedData.WorkspaceName
	if workspaceName == "" {
		workspaceName = "My Workspace"
	}
	slug := generateSlugFromName(workspaceName)

	var workspace struct {
		ID   uuid.UUID
		Name string
		Slug string
	}
	err = tx.QueryRow(ctx, `
		INSERT INTO workspaces (name, slug, owner_id, onboarding_completed_at, onboarding_data)
		VALUES ($1, $2, $3, NOW(), $4)
		RETURNING id, name, slug
	`, workspaceName, slug, userID, structToMap(extractedData)).Scan(&workspace.ID, &workspace.Name, &workspace.Slug)
	if err != nil {
		return nil, fmt.Errorf("create workspace: %w", err)
	}

	// Seed default roles
	_, err = tx.Exec(ctx, "SELECT seed_default_workspace_roles($1)", workspace.ID)
	if err != nil {
		// Try without the function if it doesn't exist
		_, err = tx.Exec(ctx, `
			INSERT INTO workspace_roles (workspace_id, name, display_name, is_system, hierarchy_level, permissions)
			VALUES
				($1, 'owner', 'Owner', true, 100, '{"all": true}'::jsonb),
				($1, 'admin', 'Admin', true, 80, '{"manage_members": true, "manage_settings": true}'::jsonb),
				($1, 'member', 'Member', true, 50, '{"read": true, "write": true}'::jsonb)
			ON CONFLICT DO NOTHING
		`, workspace.ID)
		if err != nil {
			return nil, fmt.Errorf("seed roles: %w", err)
		}
	}

	// Add owner as first member
	_, err = tx.Exec(ctx, `
		INSERT INTO workspace_members (workspace_id, user_id, role_name, status, joined_at)
		VALUES ($1, $2, 'owner', 'active', NOW())
	`, workspace.ID, userID)
	if err != nil {
		return nil, fmt.Errorf("add owner: %w", err)
	}

	// Create onboarding profile
	recommendations := ComputeRecommendations(extractedData.Challenge, extractedData.BusinessType)
	_, err = tx.Exec(ctx, `
		INSERT INTO workspace_onboarding_profiles
			(workspace_id, business_type, team_size, owner_role, main_challenge, recommended_integrations, onboarding_session_id, onboarding_method)
		VALUES ($1, $2, $3, $4, $5, $6, $7, 'conversational')
	`, workspace.ID, extractedData.BusinessType, extractedData.TeamSize, extractedData.Role, extractedData.Challenge, recommendations, session.ID)
	if err != nil {
		return nil, fmt.Errorf("create onboarding profile: %w", err)
	}

	// Commit the transaction first so workspace exists
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}

	// Sync user and workspace to OSA if OSA sync service is available
	if s.osaSyncService != nil {
		// Convert userID string to UUID
		userUUID, err := uuid.Parse(userID)
		if err != nil {
			slog.Warn("Failed to parse user ID for OSA sync",
				"user_id", userID,
				"error", err,
			)
		} else {
			// Sync user to OSA (async)
			go func() {
				bgCtx := context.Background()
				if err := s.osaSyncService.SyncUser(bgCtx, userUUID); err != nil {
					slog.Warn("Failed to sync user to OSA",
						"user_id", userID,
						"error", err,
					)
				} else {
					slog.Info("User synced to OSA successfully",
						"user_id", userID,
					)
				}
			}()

			// Sync workspace to OSA (async)
			go func() {
				bgCtx := context.Background()
				if err := s.osaSyncService.SyncWorkspace(bgCtx, workspace.ID); err != nil {
					slog.Warn("Failed to sync workspace to OSA",
						"workspace_id", workspace.ID,
						"error", err,
					)
				} else {
					slog.Info("Workspace synced to OSA successfully",
						"workspace_id", workspace.ID,
						"name", workspace.Name,
					)
				}
			}()
		}
	} else {
		slog.Debug("OSA sync service not available, skipping OSA sync",
			"workspace_id", workspace.ID,
		)
	}

	// Trigger post-onboarding app generation (fire-and-forget)
	go func() {
		postOnboardingService := NewPostOnboardingService(s.pool, slog.Default())
		if err := postOnboardingService.QueueAppsForWorkspace(context.Background(), workspace.ID); err != nil {
			slog.Warn("Failed to queue post-onboarding apps",
				"workspace_id", workspace.ID,
				"error", err,
			)
		} else {
			slog.Info("Successfully queued post-onboarding apps",
				"workspace_id", workspace.ID,
			)
		}
	}()

	// Transform AI analysis to workspace profile if analysis exists
	if err := s.transformAIAnalysisToWorkspaceProfile(ctx, workspace.ID, userID); err != nil {
		// Log warning but don't fail - workspace was created successfully
		slog.Warn("Failed to transform AI analysis to workspace profile",
			"workspace_id", workspace.ID,
			"error", err,
		)
	}

	// Analyze and save email metadata if Gmail is connected
	if s.emailAnalyzer != nil && s.gmailService != nil {
		slog.Info("Analyzing user emails for onboarding insights",
			"user_id", userID,
			"session_id", session.ID,
		)

		// Analyze and save recent emails (async to not block workspace creation)
		go func() {
			bgCtx := context.Background()
			metadata, err := s.emailAnalyzer.AnalyzeAndSaveRecentEmails(bgCtx, userID, session.ID, 50)
			if err != nil {
				slog.Warn("Failed to analyze emails during onboarding",
					"user_id", userID,
					"session_id", session.ID,
					"error", err,
				)
			} else {
				slog.Info("Email metadata saved successfully",
					"user_id", userID,
					"session_id", session.ID,
					"emails_analyzed", metadata.TotalEmails,
					"tools_detected", len(metadata.DetectedTools),
				)
			}
		}()
	}

	// Update session as completed
	_, err = s.pool.Exec(ctx, `
		UPDATE onboarding_sessions
		SET status = 'completed', workspace_id = $1, completed_at = NOW(),
		    extracted_data = $2, current_step = 'complete', updated_at = NOW()
		WHERE id = $3
	`, workspace.ID, structToMap(extractedData), session.ID)
	if err != nil {
		// Log warning but don't fail - workspace was created successfully
		slog.Warn("Failed to update session status", "error", err)
	}

	return &CompleteOnboardingResponse{
		WorkspaceID:   workspace.ID,
		WorkspaceName: workspace.Name,
		WorkspaceSlug: workspace.Slug,
		RedirectURL:   "/window",
	}, nil
}

// SubmitFallbackForm handles fallback form submission
func (s *OnboardingService) SubmitFallbackForm(ctx context.Context, sessionID uuid.UUID, userID string, data *FallbackFormData) (*CompleteOnboardingResponse, error) {
	// Validate the fallback form data
	if validationErrors := s.validator.ValidateFallbackForm(data); validationErrors.HasErrors() {
		return nil, fmt.Errorf("validation failed: %s", validationErrors.Error())
	}

	session, err := s.GetSession(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	if session.UserID != userID {
		return nil, fmt.Errorf("session does not belong to user")
	}

	// Build extracted_data from fallback form
	extractedData := map[string]interface{}{
		"workspace_name": data.WorkspaceName,
		"business_type":  data.BusinessType,
		"team_size":      data.TeamSize,
		"role":           data.Role,
		"challenge":      data.Challenge,
		"integrations":   data.Integrations,
		// NEW: Fallback form fields
		"tools_used":      data.ToolsUsed,
		"main_focus":      data.MainFocus,
		"work_style":      data.WorkStyle,
		"what_would_help": data.WhatWouldHelp,
	}

	// Update session with fallback flag and extracted data
	_, err = s.pool.Exec(ctx, `
		UPDATE onboarding_sessions
		SET fallback_triggered = true,
		    extracted_data = $1,
		    updated_at = NOW()
		WHERE id = $2
	`, extractedData, sessionID)
	if err != nil {
		return nil, fmt.Errorf("update session with fallback data: %w", err)
	}

	// Complete onboarding with form data
	return s.CompleteOnboarding(ctx, sessionID, userID, data.Integrations)
}

// generateSlugFromName creates a URL-safe slug from a workspace name
func generateSlugFromName(name string) string {
	slug := strings.ToLower(name)
	slug = strings.ReplaceAll(slug, " ", "-")
	// Remove special characters
	var result strings.Builder
	for _, r := range slug {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			result.WriteRune(r)
		}
	}
	slug = result.String()
	// Add random suffix for uniqueness
	slug = fmt.Sprintf("%s-%s", slug, uuid.New().String()[:8])
	return slug
}

// onboardingStrPtr is a small helper to take the address of a string literal
func onboardingStrPtr(s string) *string {
	return &s
}

// structToMap serialises any struct to map[string]interface{} via JSON round-trip
func structToMap(v interface{}) map[string]interface{} {
	data, _ := json.Marshal(v)
	var result map[string]interface{}
	json.Unmarshal(data, &result)
	return result
}
