package services

import (
	"time"

	"github.com/google/uuid"
)

// OnboardingSession represents an onboarding session
type OnboardingSession struct {
	ID                 uuid.UUID              `json:"id"`
	UserID             string                 `json:"user_id"`
	Status             string                 `json:"status"`
	CurrentStep        string                 `json:"current_step"`
	StepsCompleted     []string               `json:"steps_completed"`
	ExtractedData      map[string]interface{} `json:"extracted_data"`
	LowConfidenceCount int                    `json:"low_confidence_count"`
	FallbackTriggered  bool                   `json:"fallback_triggered"`
	WorkspaceID        *uuid.UUID             `json:"workspace_id,omitempty"`
	StartedAt          time.Time              `json:"started_at"`
	CompletedAt        *time.Time             `json:"completed_at,omitempty"`
	ExpiresAt          time.Time              `json:"expires_at"`
	CreatedAt          time.Time              `json:"created_at"`
	UpdatedAt          time.Time              `json:"updated_at"`
}

// ConversationMessage represents a message in the conversation
type ConversationMessage struct {
	ID              uuid.UUID              `json:"id"`
	SessionID       uuid.UUID              `json:"session_id"`
	Role            string                 `json:"role"` // "user", "agent", "system"
	Content         string                 `json:"content"`
	ConfidenceScore *float64               `json:"confidence_score,omitempty"`
	ExtractedFields map[string]interface{} `json:"extracted_fields,omitempty"`
	QuestionType    *string                `json:"question_type,omitempty"`
	SequenceNumber  int                    `json:"sequence_number"`
	CreatedAt       time.Time              `json:"created_at"`
}

// ExtractedOnboardingData represents the data extracted from conversation
type ExtractedOnboardingData struct {
	WorkspaceName string   `json:"workspace_name,omitempty"`
	BusinessType  string   `json:"business_type,omitempty"` // Raw user input
	TeamSize      string   `json:"team_size,omitempty"`     // Raw user input
	Role          string   `json:"role,omitempty"`          // Raw user input
	Challenge     string   `json:"challenge,omitempty"`
	Integrations  []string `json:"integrations,omitempty"`
	// Normalized values for internal use (not stored in DB)
	NormalizedBusinessType string `json:"-"` // Used for logic only
	NormalizedTeamSize     string `json:"-"` // Used for logic only
	NormalizedRole         string `json:"-"` // Used for logic only
}

// OnboardingStatus check result
type OnboardingStatus struct {
	NeedsOnboarding bool               `json:"needs_onboarding"`
	HasSession      bool               `json:"has_session"`
	Session         *OnboardingSession `json:"session,omitempty"`
	WorkspaceCount  int                `json:"workspace_count"`
}

// SendMessageRequest for sending a message
type SendMessageRequest struct {
	Content string `json:"content" binding:"required"`
}

// SendMessageResponse from AI
type SendMessageResponse struct {
	Message                 ConversationMessage     `json:"message"`
	NextStep                string                  `json:"next_step"`
	IsComplete              bool                    `json:"is_complete"`
	ShouldShowFallback      bool                    `json:"should_show_fallback"`
	ExtractedData           ExtractedOnboardingData `json:"extracted_data"`
	RecommendedIntegrations []string                `json:"recommended_integrations,omitempty"`
}

// FallbackFormData for manual form submission
type FallbackFormData struct {
	// From Quick Info
	WorkspaceName string `json:"workspace_name" binding:"required"`
	BusinessType  string `json:"business_type" binding:"required"`
	TeamSize      string `json:"team_size"`
	Role          string `json:"role"`

	// From Fallback Form (5 questions)
	ToolsUsed     []string `json:"tools_used"`      // Q1: What tools do you use?
	MainFocus     string   `json:"main_focus"`      // Q2: Main work focus
	Challenge     string   `json:"challenge"`       // Q3: Biggest challenge
	WorkStyle     string   `json:"work_style"`      // Q4: How you work
	WhatWouldHelp []string `json:"what_would_help"` // Q5: What would help (max 3)

	// Optional integrations
	Integrations []string `json:"integrations"`
}

// CompleteOnboardingResponse after finishing
type CompleteOnboardingResponse struct {
	WorkspaceID   uuid.UUID `json:"workspace_id"`
	WorkspaceName string    `json:"workspace_name"`
	WorkspaceSlug string    `json:"workspace_slug"`
	RedirectURL   string    `json:"redirect_url"`
}

// OnboardingProfileData represents the user's onboarding profile from workspace_onboarding_profiles
type OnboardingProfileData struct {
	BusinessType            string   `json:"business_type"`
	TeamSize                string   `json:"team_size"`
	OwnerRole               string   `json:"owner_role"`
	MainChallenge           string   `json:"main_challenge"`
	RecommendedIntegrations []string `json:"recommended_integrations"`
	// Optional AI analysis data from onboarding_user_analysis table
	ProfileSummary string   `json:"profile_summary,omitempty"`
	Insights       []string `json:"insights,omitempty"`
	ToolsUsed      []string `json:"tools_used,omitempty"`
}
