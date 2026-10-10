package services

import (
	"testing"
	"time"

	"github.com/rhl/businessos-backend/internal/integrations/google"
	"github.com/stretchr/testify/assert"
)

func TestCalculateSentiment(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		expected string
	}{
		{
			name:     "Positive sentiment",
			text:     "Great job! This is amazing work. Thank you so much!",
			expected: "positive",
		},
		{
			name:     "Negative sentiment",
			text:     "Unfortunately we encountered a critical problem. This is a major issue.",
			expected: "negative",
		},
		{
			name:     "Neutral sentiment",
			text:     "Please review the attached document and let me know.",
			expected: "neutral",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculateSentiment(tt.text)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCalculateImportanceScore(t *testing.T) {
	domainFreq := map[string]int{
		"company.com":     15,
		"client.com":      5,
		"newsletter.com":  1,
	}

	tests := []struct {
		name         string
		senderDomain string
		toolCount    int
		topicCount   int
		minScore     float64
		maxScore     float64
	}{
		{
			name:         "High frequency sender with tools",
			senderDomain: "company.com",
			toolCount:    3,
			topicCount:   3,
			minScore:     0.9,
			maxScore:     1.0,
		},
		{
			name:         "Medium frequency sender",
			senderDomain: "client.com",
			toolCount:    1,
			topicCount:   1,
			minScore:     0.3,
			maxScore:     0.5,
		},
		{
			name:         "Low frequency sender",
			senderDomain: "newsletter.com",
			toolCount:    0,
			topicCount:   0,
			minScore:     0.0,
			maxScore:     0.1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := calculateImportanceScore(tt.senderDomain, domainFreq, tt.toolCount, tt.topicCount)
			assert.GreaterOrEqual(t, score, tt.minScore)
			assert.LessOrEqual(t, score, tt.maxScore)
		})
	}
}

func TestCategorizeEmail(t *testing.T) {
	tests := []struct {
		name         string
		topics       map[string]int
		tools        map[string]int
		senderDomain string
		expected     string
	}{
		{
			name:         "Work email with development topics",
			topics:       map[string]int{"development": 5, "collaboration": 3},
			tools:        map[string]int{"GitHub": 2, "Slack": 1},
			senderDomain: "company.com",
			expected:     "work",
		},
		{
			name:         "Marketing email from newsletter",
			topics:       map[string]int{"marketing": 3},
			tools:        map[string]int{},
			senderDomain: "newsletter.mailchimp.com",
			expected:     "marketing",
		},
		{
			name:         "Personal email",
			topics:       map[string]int{},
			tools:        map[string]int{},
			senderDomain: "gmail.com",
			expected:     "personal",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			category := categorizeEmail(tt.topics, tt.tools, tt.senderDomain)
			assert.Equal(t, tt.expected, category)
		})
	}
}

func TestExtractMetadata(t *testing.T) {
	// Create test email analyzer service (without DB connection for unit test)
	service := &EmailAnalyzerService{}

	emails := []*google.Email{
		{
			ID:        "email1",
			FromEmail: "john@company.com",
			Subject:   "Project Update - Design Review",
			Snippet:   "Please review the latest Figma designs",
			BodyText:  "We need to collaborate on the new design using Figma and Slack",
			Date:      time.Now(),
		},
		{
			ID:        "email2",
			FromEmail: "jane@client.com",
			Subject:   "Code Review Needed",
			Snippet:   "Let's sync on the backend deploy",
			BodyText:  "GitHub PR is ready for review. We need to deploy the code on Slack.",
			Date:      time.Now(),
		},
	}

	metadata := service.extractMetadata(emails)

	// Verify aggregated metadata
	assert.Equal(t, 0, metadata.TotalEmails) // Not set by extractMetadata, set by caller
	assert.Len(t, metadata.SenderDomains, 2)
	assert.Equal(t, 1, metadata.SenderDomains["company.com"])
	assert.Equal(t, 1, metadata.SenderDomains["client.com"])

	// Verify tool detection
	assert.Contains(t, metadata.DetectedTools, "Figma")
	assert.Contains(t, metadata.DetectedTools, "Slack")
	assert.Contains(t, metadata.DetectedTools, "GitHub")

	// Verify topic detection
	assert.Contains(t, metadata.TopicFrequency, "design")
	assert.Contains(t, metadata.TopicFrequency, "development")
	assert.Contains(t, metadata.TopicFrequency, "collaboration")
}

func TestExtractKeywordsFromText(t *testing.T) {
	text := "This is a test message about design and development collaboration"
	keywords := extractKeywordsFromText(text)

	// Verify stop words are filtered out
	assert.NotContains(t, keywords, "this")
	assert.NotContains(t, keywords, "is")
	assert.NotContains(t, keywords, "a")

	// Verify meaningful words are extracted
	assert.Contains(t, keywords, "test")
	assert.Contains(t, keywords, "message")
	assert.Contains(t, keywords, "design")
	assert.Contains(t, keywords, "development")
	assert.Contains(t, keywords, "collaboration")
}

func TestExtractDomainFromEmail(t *testing.T) {
	tests := []struct {
		email    string
		expected string
	}{
		{"john@company.com", "company.com"},
		{"jane.doe@client.org", "client.org"},
		{"invalid-email", ""},
		{"multiple@at@signs.com", ""},
	}

	for _, tt := range tests {
		t.Run(tt.email, func(t *testing.T) {
			domain := extractDomainFromEmail(tt.email)
			assert.Equal(t, tt.expected, domain)
		})
	}
}
