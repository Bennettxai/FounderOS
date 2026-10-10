package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rhl/businessos-backend/internal/integrations/google"
)

// EmailAnalyzerService extracts metadata from user emails for onboarding analysis
type EmailAnalyzerService struct {
	pool         *pgxpool.Pool
	gmailService *google.GmailService
}

// EmailAnalysisMetadata represents extracted metadata from emails
type EmailAnalysisMetadata struct {
	TotalEmails     int                    `json:"total_emails"`
	SenderDomains   map[string]int         `json:"sender_domains"`
	SubjectKeywords []string               `json:"subject_keywords"`
	BodyKeywords    []string               `json:"body_keywords"`
	DetectedTools   map[string]int         `json:"detected_tools"`
	TopicFrequency  map[string]int         `json:"topic_frequency"`
	EmailDates      []time.Time            `json:"email_dates"`
	RawMetadata     map[string]interface{} `json:"raw_metadata,omitempty"`
}

// Known tools/platforms to detect in emails
var knownTools = []string{
	// Design & Creative
	"Figma", "Sketch", "Adobe XD", "Canva", "Photoshop", "Illustrator",
	// Development
	"GitHub", "GitLab", "VS Code", "Visual Studio", "IntelliJ", "Docker", "Kubernetes",
	// Project Management
	"Notion", "Asana", "Trello", "Jira", "Monday.com", "ClickUp", "Linear",
	// Communication
	"Slack", "Discord", "Teams", "Zoom", "Meet", "Loom",
	// Productivity
	"Airtable", "Coda", "Roam", "Obsidian", "Evernote", "Todoist",
	// Marketing
	"HubSpot", "Mailchimp", "SendGrid", "Intercom", "Segment",
	// Analytics
	"Google Analytics", "Mixpanel", "Amplitude", "Heap",
	// No-code
	"Webflow", "Bubble", "Zapier", "Make", "n8n", "Retool",
	// CRM
	"Salesforce", "HubSpot CRM", "Pipedrive", "Copper",
	// Other
	"Stripe", "PayPal", "Shopify", "WordPress", "Framer",
}

// Common topic patterns to detect
var topicPatterns = map[string]*regexp.Regexp{
	"design":        regexp.MustCompile(`(?i)\b(design|ui|ux|prototype|mockup|wireframe)\b`),
	"development":   regexp.MustCompile(`(?i)\b(code|develop|build|api|backend|frontend|deploy)\b`),
	"marketing":     regexp.MustCompile(`(?i)\b(market|campaign|seo|content|social|ads)\b`),
	"sales":         regexp.MustCompile(`(?i)\b(sale|deal|prospect|lead|client|revenue)\b`),
	"product":       regexp.MustCompile(`(?i)\b(product|feature|roadmap|launch|release)\b`),
	"analytics":     regexp.MustCompile(`(?i)\b(analytic|metric|data|report|insight)\b`),
	"automation":    regexp.MustCompile(`(?i)\b(automat|workflow|integration|api)\b`),
	"collaboration": regexp.MustCompile(`(?i)\b(collaborat|team|meeting|sync|standup)\b`),
}

func NewEmailAnalyzerService(pool *pgxpool.Pool, gmailService *google.GmailService) *EmailAnalyzerService {
	return &EmailAnalyzerService{
		pool:         pool,
		gmailService: gmailService,
	}
}

// AnalyzeRecentEmails fetches and analyzes recent emails for a user
func (s *EmailAnalyzerService) AnalyzeRecentEmails(ctx context.Context, userID string, maxEmails int) (*EmailAnalysisMetadata, error) {
	slog.Info("EmailAnalyzerService starting analysis",
		"user_id", userID,
		"max_emails", maxEmails,
	)

	// Fetch recent emails from Gmail
	slog.Info("Syncing emails from Gmail")
	syncResult, err := s.gmailService.SyncEmails(ctx, userID, int64(maxEmails))
	if err != nil {
		slog.Error("Failed to sync emails", "error", err)
		return nil, fmt.Errorf("failed to sync emails: %w", err)
	}

	slog.Info("Emails synced",
		"total", syncResult.TotalEmails,
		"synced", syncResult.SyncedEmails,
		"failed", syncResult.FailedEmails,
	)

	// Get synced emails from database
	emails, err := s.gmailService.GetEmails(ctx, userID, google.FolderInbox, maxEmails, 0)
	if err != nil {
		slog.Error("Failed to retrieve emails", "error", err)
		return nil, fmt.Errorf("failed to retrieve emails: %w", err)
	}

	if len(emails) == 0 {
		slog.Warn("No emails found for user", "user_id", userID)
		return &EmailAnalysisMetadata{
			TotalEmails:     0,
			SenderDomains:   make(map[string]int),
			SubjectKeywords: []string{},
			BodyKeywords:    []string{},
			DetectedTools:   make(map[string]int),
			TopicFrequency:  make(map[string]int),
			EmailDates:      []time.Time{},
		}, nil
	}

	// Extract metadata from emails
	metadata := s.extractMetadata(emails)
	metadata.TotalEmails = len(emails)

	slog.Info("Email analysis complete",
		"emails_analyzed", metadata.TotalEmails,
		"unique_domains", len(metadata.SenderDomains),
		"tools_detected", len(metadata.DetectedTools),
		"topics_detected", len(metadata.TopicFrequency),
	)

	return metadata, nil
}

// AnalyzeAndSaveRecentEmails fetches, analyzes, and persists email metadata to database
func (s *EmailAnalyzerService) AnalyzeAndSaveRecentEmails(ctx context.Context, userID string, sessionID uuid.UUID, maxEmails int) (*EmailAnalysisMetadata, error) {
	slog.Info("EmailAnalyzerService starting analysis with persistence",
		"user_id", userID,
		"session_id", sessionID,
		"max_emails", maxEmails,
	)

	// Fetch recent emails from Gmail
	slog.Info("Syncing emails from Gmail")
	syncResult, err := s.gmailService.SyncEmails(ctx, userID, int64(maxEmails))
	if err != nil {
		slog.Error("Failed to sync emails", "error", err)
		return nil, fmt.Errorf("failed to sync emails: %w", err)
	}

	slog.Info("Emails synced",
		"total", syncResult.TotalEmails,
		"synced", syncResult.SyncedEmails,
		"failed", syncResult.FailedEmails,
	)

	// Get synced emails from database
	emails, err := s.gmailService.GetEmails(ctx, userID, google.FolderInbox, maxEmails, 0)
	if err != nil {
		slog.Error("Failed to retrieve emails", "error", err)
		return nil, fmt.Errorf("failed to retrieve emails: %w", err)
	}

	if len(emails) == 0 {
		slog.Warn("No emails found for user", "user_id", userID)
		return &EmailAnalysisMetadata{
			TotalEmails:     0,
			SenderDomains:   make(map[string]int),
			SubjectKeywords: []string{},
			BodyKeywords:    []string{},
			DetectedTools:   make(map[string]int),
			TopicFrequency:  make(map[string]int),
			EmailDates:      []time.Time{},
		}, nil
	}

	// Extract metadata from emails
	metadata := s.extractMetadata(emails)
	metadata.TotalEmails = len(emails)

	// Save per-email metadata to database
	if err := s.saveEmailMetadata(ctx, userID, sessionID, emails, metadata); err != nil {
		slog.Error("Failed to save email metadata", "error", err)
		return nil, fmt.Errorf("failed to save email metadata: %w", err)
	}

	slog.Info("Email analysis and persistence complete",
		"emails_analyzed", metadata.TotalEmails,
		"unique_domains", len(metadata.SenderDomains),
		"tools_detected", len(metadata.DetectedTools),
		"topics_detected", len(metadata.TopicFrequency),
	)

	return metadata, nil
}

// extractMetadata processes emails and extracts structured metadata
func (s *EmailAnalyzerService) extractMetadata(emails []*google.Email) *EmailAnalysisMetadata {
	metadata := &EmailAnalysisMetadata{
		SenderDomains:   make(map[string]int),
		SubjectKeywords: []string{},
		BodyKeywords:    []string{},
		DetectedTools:   make(map[string]int),
		TopicFrequency:  make(map[string]int),
		EmailDates:      []time.Time{},
	}

	subjectWords := make(map[string]int)
	bodyWords := make(map[string]int)

	for _, email := range emails {
		// Extract sender domain
		domain := extractDomainFromEmail(email.FromEmail)
		if domain != "" {
			metadata.SenderDomains[domain]++
		}

		// Extract date
		if !email.Date.IsZero() {
			metadata.EmailDates = append(metadata.EmailDates, email.Date)
		}

		// Process subject
		subjectText := strings.ToLower(email.Subject)
		for _, word := range extractKeywordsFromText(subjectText) {
			subjectWords[word]++
		}

		// Process body (snippet + full text if available)
		bodyText := strings.ToLower(email.Snippet)
		if email.BodyText != "" {
			bodyText += " " + strings.ToLower(email.BodyText)
		}

		for _, word := range extractKeywordsFromText(bodyText) {
			bodyWords[word]++
		}

		// Detect tools
		fullText := email.Subject + " " + email.Snippet + " " + email.BodyText
		for _, tool := range knownTools {
			if strings.Contains(strings.ToLower(fullText), strings.ToLower(tool)) {
				metadata.DetectedTools[tool]++
			}
		}

		// Detect topics
		for topic, pattern := range topicPatterns {
			if pattern.MatchString(fullText) {
				metadata.TopicFrequency[topic]++
			}
		}
	}

	// Convert word maps to top keywords (top 20)
	metadata.SubjectKeywords = topNWords(subjectWords, 20)
	metadata.BodyKeywords = topNWords(bodyWords, 20)

	return metadata
}

// extractDomainFromEmail extracts domain from email address
func extractDomainFromEmail(email string) string {
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return ""
	}
	return strings.ToLower(parts[1])
}

// extractKeywordsFromText extracts meaningful words from text (filters stop words)
func extractKeywordsFromText(text string) []string {
	// Simple word extraction (split on non-alphanumeric)
	words := regexp.MustCompile(`[^\w]+`).Split(text, -1)

	// Filter stop words and short words
	stopWords := map[string]bool{
		"the": true, "a": true, "an": true, "and": true, "or": true, "but": true,
		"is": true, "are": true, "was": true, "were": true, "be": true, "been": true,
		"to": true, "of": true, "in": true, "for": true, "on": true, "with": true,
		"as": true, "at": true, "by": true, "from": true, "it": true, "that": true,
		"this": true, "you": true, "your": true, "we": true, "our": true, "have": true,
		"has": true, "had": true, "can": true, "will": true, "would": true,
	}

	var keywords []string
	for _, word := range words {
		word = strings.ToLower(strings.TrimSpace(word))
		if len(word) > 3 && !stopWords[word] {
			keywords = append(keywords, word)
		}
	}

	return keywords
}

// topNWords returns the top N words by frequency
func topNWords(wordMap map[string]int, n int) []string {
	type wordCount struct {
		word  string
		count int
	}

	var counts []wordCount
	for word, count := range wordMap {
		counts = append(counts, wordCount{word, count})
	}

	// Sort by count (descending)
	for i := 0; i < len(counts); i++ {
		for j := i + 1; j < len(counts); j++ {
			if counts[j].count > counts[i].count {
				counts[i], counts[j] = counts[j], counts[i]
			}
		}
	}

	// Take top N
	if len(counts) > n {
		counts = counts[:n]
	}

	var result []string
	for _, wc := range counts {
		result = append(result, wc.word)
	}

	return result
}

// saveEmailMetadata persists per-email metadata to the database
func (s *EmailAnalyzerService) saveEmailMetadata(ctx context.Context, userID string, sessionID uuid.UUID, emails []*google.Email, metadata *EmailAnalysisMetadata) error {
	slog.Info("Saving email metadata to database",
		"user_id", userID,
		"session_id", sessionID,
		"email_count", len(emails),
	)

	// Build per-email metadata
	for _, email := range emails {
		// Extract sender domain
		senderDomain := extractDomainFromEmail(email.FromEmail)

		// Extract keywords for this specific email
		subjectText := strings.ToLower(email.Subject)
		subjectKeywords := extractKeywordsFromText(subjectText)
		if len(subjectKeywords) > 10 {
			subjectKeywords = subjectKeywords[:10] // Limit to top 10
		}

		bodyText := strings.ToLower(email.Snippet)
		if email.BodyText != "" {
			bodyText += " " + strings.ToLower(email.BodyText)
		}
		bodyKeywords := extractKeywordsFromText(bodyText)
		if len(bodyKeywords) > 20 {
			bodyKeywords = bodyKeywords[:20] // Limit to top 20
		}

		// Detect tools for this email
		detectedTools := make(map[string]int)
		fullText := email.Subject + " " + email.Snippet + " " + email.BodyText
		for _, tool := range knownTools {
			if strings.Contains(strings.ToLower(fullText), strings.ToLower(tool)) {
				detectedTools[tool]++
			}
		}

		// Detect topics for this email
		topics := make(map[string]int)
		for topic, pattern := range topicPatterns {
			if pattern.MatchString(fullText) {
				topics[topic]++
			}
		}

		// Calculate sentiment (simple keyword-based)
		sentiment := calculateSentiment(fullText)

		// Calculate importance score based on:
		// - Sender frequency (how often this domain appears)
		// - Number of detected tools
		// - Number of topics
		importanceScore := calculateImportanceScore(senderDomain, metadata.SenderDomains, len(detectedTools), len(topics))

		// Categorize email
		category := categorizeEmail(topics, detectedTools, senderDomain)

		// Convert maps to JSON
		detectedToolsJSON, err := json.Marshal(detectedTools)
		if err != nil {
			slog.Warn("Failed to marshal detected_tools", "email_id", email.ID, "error", err)
			detectedToolsJSON = nil
		}

		topicsJSON, err := json.Marshal(topics)
		if err != nil {
			slog.Warn("Failed to marshal topics", "email_id", email.ID, "error", err)
			topicsJSON = nil
		}

		// Insert or update email metadata
		_, err = s.pool.Exec(ctx, `
			INSERT INTO onboarding_email_metadata (
				session_id, email_id, sender_domain, subject_keywords, body_keywords,
				detected_tools, topics, sentiment, importance_score, category
			) VALUES ($1, $2, $3, $4, $5, $6::jsonb, $7::jsonb, $8, $9, $10)
			ON CONFLICT (session_id, email_id) DO UPDATE SET
				sender_domain = EXCLUDED.sender_domain,
				subject_keywords = EXCLUDED.subject_keywords,
				body_keywords = EXCLUDED.body_keywords,
				detected_tools = EXCLUDED.detected_tools,
				topics = EXCLUDED.topics,
				sentiment = EXCLUDED.sentiment,
				importance_score = EXCLUDED.importance_score,
				category = EXCLUDED.category,
				updated_at = NOW()
		`, sessionID, email.ID, senderDomain, subjectKeywords, bodyKeywords,
			detectedToolsJSON, topicsJSON, sentiment, importanceScore, category)

		if err != nil {
			slog.Error("Failed to insert email metadata",
				"email_id", email.ID,
				"error", err,
			)
			return fmt.Errorf("failed to insert email metadata for %s: %w", email.ID, err)
		}
	}

	slog.Info("Successfully saved email metadata",
		"session_id", sessionID,
		"emails_saved", len(emails),
	)

	return nil
}

// calculateSentiment performs simple keyword-based sentiment analysis
func calculateSentiment(text string) string {
	textLower := strings.ToLower(text)

	positiveKeywords := []string{
		"great", "excellent", "amazing", "awesome", "fantastic", "wonderful",
		"success", "congrat", "appreciate", "thanks", "thank you", "love",
		"perfect", "brilliant", "outstanding", "excited", "happy",
	}

	negativeKeywords := []string{
		"problem", "issue", "error", "fail", "bug", "wrong", "unfortunately",
		"sorry", "concern", "difficult", "urgent", "critical", "frustrated",
		"disappointed", "delay", "cancel",
	}

	positiveCount := 0
	negativeCount := 0

	for _, keyword := range positiveKeywords {
		if strings.Contains(textLower, keyword) {
			positiveCount++
		}
	}

	for _, keyword := range negativeKeywords {
		if strings.Contains(textLower, keyword) {
			negativeCount++
		}
	}

	if positiveCount > negativeCount {
		return "positive"
	} else if negativeCount > positiveCount {
		return "negative"
	}
	return "neutral"
}

// calculateImportanceScore calculates importance based on various factors
func calculateImportanceScore(senderDomain string, domainFrequency map[string]int, toolCount, topicCount int) float64 {
	score := 0.0

	// Factor 1: Sender frequency (0-0.4)
	// Higher frequency = more important relationship
	frequency := domainFrequency[senderDomain]
	if frequency >= 10 {
		score += 0.4
	} else {
		score += float64(frequency) * 0.04
	}

	// Factor 2: Tool mentions (0-0.3)
	// More tools = more relevant to work
	if toolCount >= 3 {
		score += 0.3
	} else {
		score += float64(toolCount) * 0.1
	}

	// Factor 3: Topic relevance (0-0.3)
	// More topics = more comprehensive
	if topicCount >= 3 {
		score += 0.3
	} else {
		score += float64(topicCount) * 0.1
	}

	// Cap at 1.0
	if score > 1.0 {
		score = 1.0
	}

	return score
}

// categorizeEmail assigns a category based on topics and tools
func categorizeEmail(topics map[string]int, tools map[string]int, senderDomain string) string {
	// Check for marketing/newsletter patterns
	marketingDomains := []string{"sendgrid", "mailchimp", "newsletter", "noreply", "notifications"}
	for _, pattern := range marketingDomains {
		if strings.Contains(senderDomain, pattern) {
			return "marketing"
		}
	}

	// Check topics
	if topics["development"] > 0 || topics["product"] > 0 {
		return "work"
	}

	if topics["collaboration"] > 0 || topics["analytics"] > 0 {
		return "work"
	}

	// Check for specific tool types
	if len(tools) >= 2 {
		return "work"
	}

	// Default categorization
	if topics["sales"] > 0 || topics["marketing"] > 0 {
		return "marketing"
	}

	return "personal"
}
