package services

import (
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

// extractFacts extracts factual information
func (s *MemoryExtractorService) extractFacts(userID, content string, messages []Message) []ExtractedMemory {
	memories := make([]ExtractedMemory, 0)

	// Patterns for facts
	factPatterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)(?:i am|i'm|my name is|i work at|i work as|i live in|my job is)\s+([^.!?\n]+)`),
		regexp.MustCompile(`(?i)(?:the (?:answer|solution|result) is)\s+([^.!?\n]+)`),
		regexp.MustCompile(`(?i)(?:we use|our team uses|the project uses)\s+([^.!?\n]+)`),
		regexp.MustCompile(`(?i)(?:the (?:api|endpoint|url|port|host) is)\s+([^.!?\n]+)`),
		regexp.MustCompile(`(?i)(?:the password is|the key is|the secret is)\s+([^.!?\n]+)`),
	}

	for _, pattern := range factPatterns {
		matches := pattern.FindAllStringSubmatch(content, -1)
		for _, match := range matches {
			if len(match) > 1 {
				fact := strings.TrimSpace(match[1])
				if len(fact) < 5 || len(fact) > 500 {
					continue
				}

				memories = append(memories, ExtractedMemory{
					ID:          uuid.New().String(),
					UserID:      userID,
					Type:        MemoryTypeFact,
					Content:     match[0],
					Summary:     fact,
					Source:      MemorySourceConversation,
					Confidence:  0.7,
					Tags:        s.extractTags(fact),
					Entities:    s.extractEntitiesFromText(fact),
					Importance:  5,
					Metadata:    make(map[string]interface{}),
					ExtractedAt: time.Now(),
				})
			}
		}
	}

	return s.deduplicateMemories(memories)
}

// extractPreferences extracts user preferences
func (s *MemoryExtractorService) extractPreferences(userID, content string, messages []Message) []ExtractedMemory {
	memories := make([]ExtractedMemory, 0)

	prefPatterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)(?:i (?:prefer|like|love|enjoy|hate|dislike|want|need))\s+([^.!?\n]+)`),
		regexp.MustCompile(`(?i)(?:i (?:always|usually|typically|never))\s+([^.!?\n]+)`),
		regexp.MustCompile(`(?i)(?:my favorite|my preferred)\s+([^.!?\n]+)`),
		regexp.MustCompile(`(?i)(?:i'd rather|i would prefer)\s+([^.!?\n]+)`),
	}

	for _, pattern := range prefPatterns {
		matches := pattern.FindAllStringSubmatch(content, -1)
		for _, match := range matches {
			if len(match) > 1 {
				pref := strings.TrimSpace(match[1])
				if len(pref) < 5 || len(pref) > 300 {
					continue
				}

				memories = append(memories, ExtractedMemory{
					ID:          uuid.New().String(),
					UserID:      userID,
					Type:        MemoryTypePreference,
					Content:     match[0],
					Summary:     pref,
					Source:      MemorySourceConversation,
					Confidence:  0.8,
					Tags:        s.extractTags(pref),
					Entities:    s.extractEntitiesFromText(pref),
					Importance:  6,
					Metadata:    make(map[string]interface{}),
					ExtractedAt: time.Now(),
				})
			}
		}
	}

	return s.deduplicateMemories(memories)
}

// extractDecisionsFromContent extracts decisions from content
func (s *MemoryExtractorService) extractDecisionsFromContent(userID, content string, messages []Message) []ExtractedMemory {
	memories := make([]ExtractedMemory, 0)

	decisionPatterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)(?:decided to|will use|going with|chose|selected)\s+([^.!?\n]+)`),
		regexp.MustCompile(`(?i)(?:let's go with|we'll use)\s+([^.!?\n]+)`),
		regexp.MustCompile(`(?i)(?:the decision is|agreed to)\s+([^.!?\n]+)`),
	}

	for _, pattern := range decisionPatterns {
		matches := pattern.FindAllStringSubmatch(content, -1)
		for _, match := range matches {
			if len(match) > 1 {
				decision := strings.TrimSpace(match[1])
				if len(decision) < 5 || len(decision) > 500 {
					continue
				}

				memories = append(memories, ExtractedMemory{
					ID:          uuid.New().String(),
					UserID:      userID,
					Type:        MemoryTypeDecision,
					Content:     match[0],
					Summary:     decision,
					Source:      MemorySourceConversation,
					Confidence:  0.75,
					Tags:        append(s.extractTags(decision), "decision"),
					Entities:    s.extractEntitiesFromText(decision),
					Importance:  7,
					Metadata:    make(map[string]interface{}),
					ExtractedAt: time.Now(),
				})
			}
		}
	}

	return s.deduplicateMemories(memories)
}

// extractTasks extracts tasks from content
func (s *MemoryExtractorService) extractTasks(userID, content string, messages []Message) []ExtractedMemory {
	memories := make([]ExtractedMemory, 0)

	taskPatterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)(?:need to|should|must|have to|todo:|task:)\s+([^.!?\n]+)`),
		regexp.MustCompile(`(?i)- \[ \]\s+([^\n]+)`),
		regexp.MustCompile(`(?i)(?:implement|create|add|fix|update|remove)\s+([^.!?\n]+)`),
		regexp.MustCompile(`(?i)(?:don't forget to|remember to|make sure to)\s+([^.!?\n]+)`),
	}

	for _, pattern := range taskPatterns {
		matches := pattern.FindAllStringSubmatch(content, -1)
		for _, match := range matches {
			if len(match) > 1 {
				task := strings.TrimSpace(match[1])
				if len(task) < 5 || len(task) > 300 {
					continue
				}

				memories = append(memories, ExtractedMemory{
					ID:          uuid.New().String(),
					UserID:      userID,
					Type:        MemoryTypeTask,
					Content:     match[0],
					Summary:     task,
					Source:      MemorySourceConversation,
					Confidence:  0.7,
					Tags:        append(s.extractTags(task), "task"),
					Entities:    s.extractEntitiesFromText(task),
					Importance:  s.inferTaskImportance(task),
					Metadata:    map[string]interface{}{"status": "pending"},
					ExtractedAt: time.Now(),
				})
			}
		}
	}

	return s.deduplicateMemories(memories)
}

// extractTasksFromVoice extracts tasks from voice notes
func (s *MemoryExtractorService) extractTasksFromVoice(userID, transcript string) []ExtractedMemory {
	memories := make([]ExtractedMemory, 0)

	// Voice-specific patterns (more conversational)
	voiceTaskPatterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)(?:i need to|i have to|i should|i must)\s+([^.!?\n]+)`),
		regexp.MustCompile(`(?i)(?:gotta|gonna|need to)\s+([^.!?\n]+)`),
		regexp.MustCompile(`(?i)(?:note to self|reminder|don't forget)\s*[:-]?\s*([^.!?\n]+)`),
	}

	for _, pattern := range voiceTaskPatterns {
		matches := pattern.FindAllStringSubmatch(transcript, -1)
		for _, match := range matches {
			if len(match) > 1 {
				task := strings.TrimSpace(match[1])
				if len(task) < 5 || len(task) > 300 {
					continue
				}

				memories = append(memories, ExtractedMemory{
					ID:          uuid.New().String(),
					UserID:      userID,
					Type:        MemoryTypeTask,
					Content:     match[0],
					Summary:     task,
					Source:      MemorySourceVoiceNote,
					Confidence:  0.65,
					Tags:        append(s.extractTags(task), "task", "voice"),
					Entities:    s.extractEntitiesFromText(task),
					Importance:  s.inferTaskImportance(task),
					Metadata:    map[string]interface{}{"status": "pending"},
					ExtractedAt: time.Now(),
				})
			}
		}
	}

	return s.deduplicateMemories(memories)
}

// extractReminders extracts reminders from voice notes
func (s *MemoryExtractorService) extractReminders(userID, transcript string) []ExtractedMemory {
	memories := make([]ExtractedMemory, 0)

	reminderPatterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)(?:remind me to|reminder|don't let me forget)\s+([^.!?\n]+)`),
		regexp.MustCompile(`(?i)(?:at|on|by|before)\s+(\d{1,2}(?::\d{2})?\s*(?:am|pm)?|\w+day|\d{1,2}/\d{1,2})\s*(?:,|:)?\s*([^.!?\n]+)`),
		regexp.MustCompile(`(?i)(?:tomorrow|next week|later today)\s*(?:,|:)?\s*([^.!?\n]+)`),
	}

	for _, pattern := range reminderPatterns {
		matches := pattern.FindAllStringSubmatch(transcript, -1)
		for _, match := range matches {
			if len(match) > 1 {
				reminder := strings.TrimSpace(match[len(match)-1])
				if len(reminder) < 5 || len(reminder) > 200 {
					continue
				}

				memories = append(memories, ExtractedMemory{
					ID:          uuid.New().String(),
					UserID:      userID,
					Type:        MemoryTypeReminder,
					Content:     match[0],
					Summary:     reminder,
					Source:      MemorySourceVoiceNote,
					Confidence:  0.7,
					Tags:        []string{"reminder", "voice"},
					Entities:    s.extractEntitiesFromText(reminder),
					Importance:  6,
					Metadata:    make(map[string]interface{}),
					ExtractedAt: time.Now(),
				})
			}
		}
	}

	return memories
}

// extractInsights extracts insights from content
func (s *MemoryExtractorService) extractInsights(userID, content string, messages []Message) []ExtractedMemory {
	memories := make([]ExtractedMemory, 0)

	insightPatterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)(?:i realized|i learned|i discovered|i noticed|i found out)\s+([^.!?\n]+)`),
		regexp.MustCompile(`(?i)(?:the key (?:insight|takeaway|learning) is)\s+([^.!?\n]+)`),
		regexp.MustCompile(`(?i)(?:interesting(?:ly)?|importantly)\s*,?\s+([^.!?\n]+)`),
		regexp.MustCompile(`(?i)(?:turns out|it seems|apparently)\s+([^.!?\n]+)`),
	}

	for _, pattern := range insightPatterns {
		matches := pattern.FindAllStringSubmatch(content, -1)
		for _, match := range matches {
			if len(match) > 1 {
				insight := strings.TrimSpace(match[1])
				if len(insight) < 10 || len(insight) > 500 {
					continue
				}

				memories = append(memories, ExtractedMemory{
					ID:          uuid.New().String(),
					UserID:      userID,
					Type:        MemoryTypeInsight,
					Content:     match[0],
					Summary:     insight,
					Source:      MemorySourceConversation,
					Confidence:  0.65,
					Tags:        append(s.extractTags(insight), "insight"),
					Entities:    s.extractEntitiesFromText(insight),
					Importance:  6,
					Metadata:    make(map[string]interface{}),
					ExtractedAt: time.Now(),
				})
			}
		}
	}

	return s.deduplicateMemories(memories)
}
