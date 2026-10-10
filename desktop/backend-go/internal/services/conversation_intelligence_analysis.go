package services

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/google/uuid"
)

// extractTopics extracts topics from messages using keyword frequency analysis.
func (s *ConversationIntelligenceService) extractTopics(messages []Message) []ConversationTopic {
	wordFreq := make(map[string]int)
	wordFirstMention := make(map[string]int)

	// Common stop words to filter
	stopWords := map[string]bool{
		"the": true, "a": true, "an": true, "is": true, "are": true,
		"was": true, "were": true, "be": true, "been": true, "being": true,
		"have": true, "has": true, "had": true, "do": true, "does": true,
		"did": true, "will": true, "would": true, "could": true, "should": true,
		"may": true, "might": true, "must": true, "shall": true, "can": true,
		"this": true, "that": true, "these": true, "those": true, "i": true,
		"you": true, "he": true, "she": true, "it": true, "we": true, "they": true,
		"what": true, "which": true, "who": true, "whom": true, "whose": true,
		"where": true, "when": true, "why": true, "how": true, "all": true,
		"each": true, "every": true, "both": true, "few": true, "more": true,
		"most": true, "other": true, "some": true, "such": true, "no": true,
		"nor": true, "not": true, "only": true, "own": true, "same": true,
		"so": true, "than": true, "too": true, "very": true, "just": true,
		"and": true, "but": true, "if": true, "or": true, "because": true,
		"as": true, "until": true, "while": true, "of": true, "at": true,
		"by": true, "for": true, "with": true, "about": true, "against": true,
		"between": true, "into": true, "through": true, "during": true,
		"before": true, "after": true, "above": true, "below": true, "to": true,
		"from": true, "up": true, "down": true, "in": true, "out": true,
		"on": true, "off": true, "over": true, "under": true, "again": true,
		"further": true, "then": true, "once": true, "here": true, "there": true,
	}

	wordPattern := regexp.MustCompile(`\b[a-zA-Z]{3,}\b`)
	for idx, msg := range messages {
		words := wordPattern.FindAllString(strings.ToLower(msg.Content), -1)
		for _, word := range words {
			if stopWords[word] {
				continue
			}
			wordFreq[word]++
			if _, exists := wordFirstMention[word]; !exists {
				wordFirstMention[word] = idx
			}
		}
	}

	// Sort by frequency, apply minimum threshold
	var counts []wordCount
	for word, count := range wordFreq {
		if count >= 2 {
			counts = append(counts, wordCount{word, count})
		}
	}
	sort.Slice(counts, func(i, j int) bool {
		return counts[i].count > counts[j].count
	})

	topics := make([]ConversationTopic, 0)
	maxTopics := 5
	for i, wc := range counts {
		if i >= maxTopics {
			break
		}
		topics = append(topics, ConversationTopic{
			Name:         wc.word,
			Confidence:   float64(wc.count) / float64(len(messages)),
			Keywords:     s.findRelatedKeywords(wc.word, counts),
			FirstMention: wordFirstMention[wc.word],
			Frequency:    wc.count,
		})
	}

	return topics
}

// findRelatedKeywords returns up to 3 other high-frequency keywords as related terms.
func (s *ConversationIntelligenceService) findRelatedKeywords(word string, counts []wordCount) []string {
	related := make([]string, 0, 3)
	for _, wc := range counts {
		if wc.word != word && len(related) < 3 {
			related = append(related, wc.word)
		}
	}
	return related
}

// extractEntities extracts named entities (files, paths, technologies, URLs) from messages.
func (s *ConversationIntelligenceService) extractEntities(messages []Message) []ConversationEntity {
	entities := make(map[string]*ConversationEntity)

	patterns := map[string]*regexp.Regexp{
		"file":       regexp.MustCompile(`(?i)[\w-]+\.(go|ts|js|svelte|py|sql|json|yaml|yml|md|txt|css|html|tsx|jsx)`),
		"path":       regexp.MustCompile(`(?:^|[^a-zA-Z0-9])(/[\w/.-]+|[\w]+/[\w/.-]+)`),
		"function":   regexp.MustCompile(`\b[a-z][a-zA-Z0-9]*\([^)]*\)`),
		"technology": regexp.MustCompile(`(?i)\b(react|svelte|vue|angular|node|go|golang|python|typescript|javascript|postgresql|redis|docker|kubernetes|aws|gcp|azure)\b`),
		"url":        regexp.MustCompile(`https?://[^\s]+`),
	}

	for idx, msg := range messages {
		for entityType, pattern := range patterns {
			matches := pattern.FindAllString(msg.Content, -1)
			for _, match := range matches {
				key := strings.ToLower(match)
				if existing, ok := entities[key]; ok {
					existing.Mentions++
					if len(existing.Context) < 3 {
						existing.Context = append(existing.Context, s.extractContext(msg.Content, match, 50))
					}
				} else {
					entities[key] = &ConversationEntity{
						Name:     match,
						Type:     entityType,
						Mentions: 1,
						Context:  []string{s.extractContext(msg.Content, match, 50)},
						Related:  make([]string, 0),
					}
					_ = idx
				}
			}
		}
	}

	result := make([]ConversationEntity, 0, len(entities))
	for _, e := range entities {
		result = append(result, *e)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Mentions > result[j].Mentions
	})
	if len(result) > 20 {
		result = result[:20]
	}

	return result
}

// extractContext returns up to chars characters surrounding the entity mention in text.
func (s *ConversationIntelligenceService) extractContext(text, entity string, chars int) string {
	idx := strings.Index(strings.ToLower(text), strings.ToLower(entity))
	if idx == -1 {
		return ""
	}
	start := idx - chars
	if start < 0 {
		start = 0
	}
	end := idx + len(entity) + chars
	if end > len(text) {
		end = len(text)
	}
	return strings.TrimSpace(text[start:end])
}

// extractQuestions extracts interrogative sentences from messages and checks whether
// the next message (by the other role) provides an answer.
func (s *ConversationIntelligenceService) extractQuestions(messages []Message) []Question {
	questions := make([]Question, 0)
	questionPattern := regexp.MustCompile(`[^.!?]*\?`)

	for idx, msg := range messages {
		matches := questionPattern.FindAllString(msg.Content, -1)
		for _, match := range matches {
			match = strings.TrimSpace(match)
			if len(match) < 10 {
				continue
			}

			q := Question{
				Text:         match,
				AskedBy:      msg.Role,
				MessageIndex: idx,
				Answered:     false,
			}

			// Check the next 1-3 messages for an answer from a different role.
			if idx < len(messages)-1 {
				for i := idx + 1; i < len(messages) && i <= idx+3; i++ {
					if messages[i].Role != msg.Role && len(messages[i].Content) > 20 {
						q.Answered = true
						q.Answer = s.truncateText(messages[i].Content, 200)
						break
					}
				}
			}

			questions = append(questions, q)
		}
	}

	return questions
}

// extractActionItems extracts action items (tasks, todos, requests) from messages.
func (s *ConversationIntelligenceService) extractActionItems(messages []Message) []ActionItem {
	actionItems := make([]ActionItem, 0)

	actionPatterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)(?:need to|should|must|have to|going to|will|todo|task:|action:)\s+([^.!?\n]+)`),
		regexp.MustCompile(`(?i)(?:please|could you|can you)\s+([^.!?\n]+)`),
		regexp.MustCompile(`(?i)- \[ \]\s+([^\n]+)`), // Markdown task
		regexp.MustCompile(`(?i)(?:implement|create|add|fix|update|remove|delete|refactor)\s+([^.!?\n]+)`),
	}

	for idx, msg := range messages {
		for _, pattern := range actionPatterns {
			matches := pattern.FindAllStringSubmatch(msg.Content, -1)
			for _, match := range matches {
				if len(match) > 1 {
					description := strings.TrimSpace(match[1])
					if len(description) < 5 || len(description) > 200 {
						continue
					}
					actionItems = append(actionItems, ActionItem{
						ID:           uuid.New().String(),
						Description:  description,
						Priority:     s.inferPriority(description),
						Status:       "pending",
						MessageIndex: idx,
						Tags:         s.extractTags(description),
					})
				}
			}
		}
	}

	// Deduplicate by lowercased description.
	seen := make(map[string]bool)
	unique := make([]ActionItem, 0, len(actionItems))
	for _, item := range actionItems {
		key := strings.ToLower(item.Description)
		if !seen[key] {
			seen[key] = true
			unique = append(unique, item)
		}
	}

	return unique
}

// inferPriority infers priority from action item description keywords.
func (s *ConversationIntelligenceService) inferPriority(description string) string {
	lower := strings.ToLower(description)

	for _, word := range []string{"urgent", "critical", "asap", "immediately", "important", "must", "blocker"} {
		if strings.Contains(lower, word) {
			return "high"
		}
	}
	for _, word := range []string{"maybe", "consider", "could", "nice to have", "eventually", "later"} {
		if strings.Contains(lower, word) {
			return "low"
		}
	}

	return "medium"
}

// extractTags maps description keywords to category tags.
func (s *ConversationIntelligenceService) extractTags(description string) []string {
	tags := make([]string, 0)

	categories := map[string][]string{
		"bug":         {"fix", "bug", "error", "issue", "broken"},
		"feature":     {"add", "implement", "create", "new", "feature"},
		"refactor":    {"refactor", "clean", "improve", "optimize"},
		"docs":        {"document", "readme", "comment", "docs"},
		"test":        {"test", "spec", "coverage"},
		"security":    {"security", "auth", "permission", "vulnerability"},
		"performance": {"performance", "speed", "optimize", "cache"},
	}

	lower := strings.ToLower(description)
	for tag, keywords := range categories {
		for _, keyword := range keywords {
			if strings.Contains(lower, keyword) {
				tags = append(tags, tag)
				break
			}
		}
	}

	return tags
}

// extractDecisions extracts decision statements from messages.
func (s *ConversationIntelligenceService) extractDecisions(messages []Message) []ConversationDecision {
	decisions := make([]ConversationDecision, 0)

	decisionPatterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)(?:decided to|will use|going with|chose|selected|opted for)\s+([^.!?\n]+)`),
		regexp.MustCompile(`(?i)(?:let's|we should|we'll)\s+(?:go with|use|implement)\s+([^.!?\n]+)`),
		regexp.MustCompile(`(?i)(?:the (?:best|better) (?:approach|solution|option) is)\s+([^.!?\n]+)`),
	}

	for idx, msg := range messages {
		for _, pattern := range decisionPatterns {
			matches := pattern.FindAllStringSubmatch(msg.Content, -1)
			for _, match := range matches {
				if len(match) > 1 {
					decisions = append(decisions, ConversationDecision{
						Description:  strings.TrimSpace(match[1]),
						Context:      s.truncateText(msg.Content, 200),
						MessageIndex: idx,
					})
				}
			}
		}
	}

	return decisions
}

// extractCodeMentions extracts fenced code blocks and file path references from messages.
func (s *ConversationIntelligenceService) extractCodeMentions(messages []Message) []CodeMention {
	mentions := make([]CodeMention, 0)

	codeBlockPattern := regexp.MustCompile("```(\\w*)\\n([\\s\\S]*?)```")
	filePathPattern := regexp.MustCompile(`(?:^|[^a-zA-Z0-9])([\w/-]+\.(go|ts|js|svelte|py|sql|json|yaml|yml|md|css|html|tsx|jsx))`)

	for idx, msg := range messages {
		for _, match := range codeBlockPattern.FindAllStringSubmatch(msg.Content, -1) {
			if len(match) > 2 {
				mentions = append(mentions, CodeMention{
					Language:     match[1],
					Snippet:      s.truncateText(match[2], 500),
					Context:      "code block",
					MessageIndex: idx,
				})
			}
		}
		for _, match := range filePathPattern.FindAllStringSubmatch(msg.Content, -1) {
			if len(match) > 1 {
				mentions = append(mentions, CodeMention{
					FilePath:     match[1],
					Context:      s.extractContext(msg.Content, match[1], 100),
					MessageIndex: idx,
				})
			}
		}
	}

	return mentions
}

// analyzeSentiment performs lexicon-based sentiment analysis on messages.
func (s *ConversationIntelligenceService) analyzeSentiment(messages []Message) SentimentAnalysis {
	result := SentimentAnalysis{
		Progression: make([]SentimentPoint, 0),
		Highlights:  make([]SentimentHighlight, 0),
	}

	positiveWords := map[string]float64{
		"great": 0.8, "good": 0.6, "excellent": 0.9, "perfect": 1.0,
		"thanks": 0.5, "thank": 0.5, "helpful": 0.7, "awesome": 0.9,
		"love": 0.8, "nice": 0.5, "wonderful": 0.8, "amazing": 0.9,
		"works": 0.6, "working": 0.5, "solved": 0.7, "fixed": 0.7,
		"yes": 0.3, "correct": 0.5, "right": 0.4, "exactly": 0.6,
	}

	negativeWords := map[string]float64{
		"bad": -0.6, "wrong": -0.5, "error": -0.4, "fail": -0.6,
		"failed": -0.6, "broken": -0.7, "issue": -0.3, "problem": -0.4,
		"bug": -0.3, "not working": -0.6, "doesn't work": -0.7,
		"confused": -0.4, "frustrating": -0.7, "annoying": -0.6,
		"hate": -0.8, "terrible": -0.9, "awful": -0.8, "worst": -0.9,
		"unfortunately": -0.3, "sadly": -0.4, "sorry": -0.2,
	}

	totalScore := 0.0

	for idx, msg := range messages {
		if msg.Role == "system" {
			continue
		}

		words := strings.Fields(strings.ToLower(msg.Content))
		score := 0.0
		scored := 0

		for _, word := range words {
			word = strings.Trim(word, ".,!?;:\"'")
			if val, ok := positiveWords[word]; ok {
				score += val
				scored++
			}
			if val, ok := negativeWords[word]; ok {
				score += val
				scored++
			}
		}

		if scored > 0 {
			score = score / float64(scored)
		}
		totalScore += score

		sentiment := "neutral"
		if score > 0.2 {
			sentiment = "positive"
		} else if score < -0.2 {
			sentiment = "negative"
		}

		result.Progression = append(result.Progression, SentimentPoint{
			MessageIndex: idx,
			Sentiment:    sentiment,
			Score:        score,
		})

		if score > 0.5 || score < -0.5 {
			result.Highlights = append(result.Highlights, SentimentHighlight{
				MessageIndex: idx,
				Text:         s.truncateText(msg.Content, 100),
				Sentiment:    sentiment,
				Reason:       fmt.Sprintf("Strong %s sentiment detected", sentiment),
			})
		}
	}

	avgScore := 0.0
	if len(messages) > 0 {
		avgScore = totalScore / float64(len(messages))
	}
	result.Score = avgScore
	switch {
	case avgScore > 0.2:
		result.Overall = "positive"
	case avgScore < -0.2:
		result.Overall = "negative"
	case len(result.Highlights) > 2:
		result.Overall = "mixed"
	default:
		result.Overall = "neutral"
	}

	return result
}

// generateTitle derives a short conversation title from the first user message or topics.
func (s *ConversationIntelligenceService) generateTitle(messages []Message, topics []ConversationTopic) string {
	for _, msg := range messages {
		if msg.Role == "user" && len(msg.Content) > 10 {
			content := msg.Content
			if idx := strings.IndexAny(content, ".!?\n"); idx > 0 && idx < 100 {
				content = content[:idx]
			}
			if len(content) > 60 {
				content = content[:60] + "..."
			}
			return strings.TrimSpace(content)
		}
	}
	if len(topics) > 0 {
		return fmt.Sprintf("Discussion about %s", topics[0].Name)
	}
	return "Untitled Conversation"
}

// generateSummary produces a human-readable summary sentence from the analysis.
func (s *ConversationIntelligenceService) generateSummary(messages []Message, analysis *ConversationAnalysis) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("A conversation with %d messages", len(messages)))
	if len(analysis.Topics) > 0 {
		topicNames := make([]string, 0, len(analysis.Topics))
		for _, t := range analysis.Topics {
			topicNames = append(topicNames, t.Name)
		}
		sb.WriteString(fmt.Sprintf(" discussing %s", strings.Join(topicNames, ", ")))
	}
	sb.WriteString(". ")

	if len(analysis.CodeMentions) > 0 {
		sb.WriteString(fmt.Sprintf("Code was discussed in %d instances. ", len(analysis.CodeMentions)))
	}
	if len(analysis.ActionItems) > 0 {
		sb.WriteString(fmt.Sprintf("%d action items were identified. ", len(analysis.ActionItems)))
	}
	if len(analysis.Decisions) > 0 {
		sb.WriteString(fmt.Sprintf("%d decisions were made. ", len(analysis.Decisions)))
	}
	if len(analysis.Questions) > 0 {
		answered := 0
		for _, q := range analysis.Questions {
			if q.Answered {
				answered++
			}
		}
		sb.WriteString(fmt.Sprintf("%d of %d questions were answered. ", answered, len(analysis.Questions)))
	}

	sb.WriteString(fmt.Sprintf("Overall sentiment was %s.", analysis.Sentiment.Overall))
	return sb.String()
}

// extractKeyPoints collects decisions and high-priority action items as bullet points.
func (s *ConversationIntelligenceService) extractKeyPoints(messages []Message, analysis *ConversationAnalysis) []string {
	_ = messages // reserved for future use
	points := make([]string, 0)

	for _, d := range analysis.Decisions {
		if len(d.Description) > 10 {
			points = append(points, "Decision: "+d.Description)
		}
	}
	for _, a := range analysis.ActionItems {
		if a.Priority == "high" {
			points = append(points, "Action: "+a.Description)
		}
	}

	if len(points) > 5 {
		points = points[:5]
	}
	return points
}

// truncateText truncates text to maxLen characters, appending "..." when cut.
func (s *ConversationIntelligenceService) truncateText(text string, maxLen int) string {
	if len(text) <= maxLen {
		return text
	}
	return text[:maxLen] + "..."
}
