package services

import (
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rhl/businessos-backend/internal/integrations/google"
)

// OnboardingService orchestrates the conversational onboarding flow.
// Conversation history and session state live in the database.
// AI inference is delegated to OnboardingAIService.
type OnboardingService struct {
	pool           *pgxpool.Pool
	aiService      *OnboardingAIService
	validator      *OnboardingValidator
	emailAnalyzer  *EmailAnalyzerService
	gmailService   *google.GmailService
	osaSyncService *OSASyncService
}

func NewOnboardingService(pool *pgxpool.Pool, aiService *OnboardingAIService, gmailService *google.GmailService, osaSyncService *OSASyncService) *OnboardingService {
	var emailAnalyzer *EmailAnalyzerService
	if gmailService != nil {
		emailAnalyzer = NewEmailAnalyzerService(pool, gmailService)
	}

	return &OnboardingService{
		pool:           pool,
		aiService:      aiService,
		validator:      NewOnboardingValidator(),
		emailAnalyzer:  emailAnalyzer,
		gmailService:   gmailService,
		osaSyncService: osaSyncService,
	}
}
