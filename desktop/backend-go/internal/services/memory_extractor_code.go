package services

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

// extractContacts extracts contact information
func (s *MemoryExtractorService) extractContacts(userID, content string) []ExtractedMemory {
	memories := make([]ExtractedMemory, 0)

	// Email pattern
	emailPattern := regexp.MustCompile(`[\w.-]+@[\w.-]+\.\w+`)
	emails := emailPattern.FindAllString(content, -1)
	for _, email := range emails {
		memories = append(memories, ExtractedMemory{
			ID:          uuid.New().String(),
			UserID:      userID,
			Type:        MemoryTypeContact,
			Content:     email,
			Summary:     fmt.Sprintf("Email: %s", email),
			Source:      MemorySourceConversation,
			Confidence:  0.9,
			Tags:        []string{"contact", "email"},
			Importance:  5,
			Metadata:    map[string]interface{}{"contact_type": "email"},
			ExtractedAt: time.Now(),
		})
	}

	// Phone pattern
	phonePattern := regexp.MustCompile(`\+?[\d\s()-]{10,}`)
	phones := phonePattern.FindAllString(content, -1)
	for _, phone := range phones {
		phone = strings.TrimSpace(phone)
		if len(phone) < 10 {
			continue
		}
		memories = append(memories, ExtractedMemory{
			ID:          uuid.New().String(),
			UserID:      userID,
			Type:        MemoryTypeContact,
			Content:     phone,
			Summary:     fmt.Sprintf("Phone: %s", phone),
			Source:      MemorySourceConversation,
			Confidence:  0.7,
			Tags:        []string{"contact", "phone"},
			Importance:  5,
			Metadata:    map[string]interface{}{"contact_type": "phone"},
			ExtractedAt: time.Now(),
		})
	}

	return memories
}

// extractCodeMemories extracts code-related memories
func (s *MemoryExtractorService) extractCodeMemories(userID, content string, messages []Message) []ExtractedMemory {
	memories := make([]ExtractedMemory, 0)

	// Error patterns
	errorPatterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)error:?\s*([^\n]+)`),
		regexp.MustCompile(`(?i)(?:got|received|seeing)\s+(?:error|exception)\s*:?\s*([^\n]+)`),
		regexp.MustCompile(`(?i)panic:?\s*([^\n]+)`),
	}

	for _, pattern := range errorPatterns {
		matches := pattern.FindAllStringSubmatch(content, -1)
		for _, match := range matches {
			if len(match) > 1 {
				errorMsg := strings.TrimSpace(match[1])
				if len(errorMsg) < 10 || len(errorMsg) > 500 {
					continue
				}

				memories = append(memories, ExtractedMemory{
					ID:          uuid.New().String(),
					UserID:      userID,
					Type:        MemoryTypeError,
					Content:     match[0],
					Summary:     errorMsg,
					Source:      MemorySourceConversation,
					Confidence:  0.8,
					Tags:        []string{"error", "code"},
					Entities:    s.extractEntitiesFromText(errorMsg),
					Importance:  7,
					Metadata:    make(map[string]interface{}),
					ExtractedAt: time.Now(),
				})
			}
		}
	}

	// Solution patterns
	solutionPatterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)(?:the (?:fix|solution) (?:is|was))\s+([^.!?\n]+)`),
		regexp.MustCompile(`(?i)(?:fixed (?:by|it by|this by))\s+([^.!?\n]+)`),
		regexp.MustCompile(`(?i)(?:solved (?:by|it by))\s+([^.!?\n]+)`),
	}

	for _, pattern := range solutionPatterns {
		matches := pattern.FindAllStringSubmatch(content, -1)
		for _, match := range matches {
			if len(match) > 1 {
				solution := strings.TrimSpace(match[1])
				if len(solution) < 10 || len(solution) > 500 {
					continue
				}

				memories = append(memories, ExtractedMemory{
					ID:          uuid.New().String(),
					UserID:      userID,
					Type:        MemoryTypeSolution,
					Content:     match[0],
					Summary:     solution,
					Source:      MemorySourceConversation,
					Confidence:  0.75,
					Tags:        []string{"solution", "code"},
					Entities:    s.extractEntitiesFromText(solution),
					Importance:  8,
					Metadata:    make(map[string]interface{}),
					ExtractedAt: time.Now(),
				})
			}
		}
	}

	return s.deduplicateMemories(memories)
}
