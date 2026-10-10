package services

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/pgvector/pgvector-go"
)

// saveAnalysis persists a ConversationAnalysis to the conversation_summaries table,
// upserting on conversation_id. A best-effort embedding is stored for semantic search.
func (s *ConversationIntelligenceService) saveAnalysis(ctx context.Context, analysis *ConversationAnalysis) error {
	// TEXT[] projections for lightweight columns
	topicNames := make([]string, 0, len(analysis.Topics))
	for _, t := range analysis.Topics {
		if t.Name != "" {
			topicNames = append(topicNames, t.Name)
		}
	}

	actionItemTexts := make([]string, 0, len(analysis.ActionItems))
	for _, a := range analysis.ActionItems {
		if strings.TrimSpace(a.Description) != "" {
			actionItemTexts = append(actionItemTexts, a.Description)
		}
	}

	questionTexts := make([]string, 0, len(analysis.Questions))
	for _, q := range analysis.Questions {
		if strings.TrimSpace(q.Text) != "" {
			questionTexts = append(questionTexts, q.Text)
		}
	}

	decisionTexts := make([]string, 0, len(analysis.Decisions))
	for _, d := range analysis.Decisions {
		if strings.TrimSpace(d.Description) != "" {
			decisionTexts = append(decisionTexts, d.Description)
		}
	}

	// Build mentioned_entities map grouped by entity type.
	mentioned := map[string][]string{}
	for _, e := range analysis.Entities {
		name := strings.TrimSpace(e.Name)
		if name == "" {
			continue
		}
		switch strings.ToLower(strings.TrimSpace(e.Type)) {
		case "person", "people":
			mentioned["people"] = append(mentioned["people"], name)
		case "project", "projects":
			mentioned["projects"] = append(mentioned["projects"], name)
		case "client", "clients", "organization", "org":
			mentioned["clients"] = append(mentioned["clients"], name)
		case "task", "tasks":
			mentioned["tasks"] = append(mentioned["tasks"], name)
		default:
			mentioned["other"] = append(mentioned["other"], name)
		}
	}

	// JSONB payloads
	sentimentJSON, _ := json.Marshal(analysis.Sentiment)
	entitiesJSON, _ := json.Marshal(analysis.Entities)
	codeMentionsJSON, _ := json.Marshal(analysis.CodeMentions)
	mentionedJSON, _ := json.Marshal(mentioned)

	// Store rich structured data in metadata so it survives the round-trip.
	if analysis.Metadata == nil {
		analysis.Metadata = make(map[string]interface{})
	}
	analysis.Metadata["topics_detail"] = analysis.Topics
	analysis.Metadata["action_items_detail"] = analysis.ActionItems
	analysis.Metadata["questions_detail"] = analysis.Questions
	analysis.Metadata["decisions_detail"] = analysis.Decisions
	metadataJSON, _ := json.Marshal(analysis.Metadata)

	// Best-effort embedding of the summary for semantic search.
	var embedding any
	if s.embed != nil {
		embedText := strings.TrimSpace(strings.Join([]string{
			analysis.Title,
			analysis.Summary,
			"Key points: " + strings.Join(analysis.KeyPoints, "; "),
			"Topics: " + strings.Join(topicNames, ", "),
		}, "\n"))
		if embedText != "" {
			if v, err := s.embed.GenerateEmbedding(ctx, embedText); err == nil {
				embedding = pgvector.NewVector(v)
			}
		}
	}

	// Optional time range from metadata.
	var timeStart, timeEnd *time.Time
	if v, ok := analysis.Metadata["time_range_start"].(time.Time); ok {
		timeStart = &v
	}
	if v, ok := analysis.Metadata["time_range_end"].(time.Time); ok {
		timeEnd = &v
	}

	_, err := s.pool.Exec(ctx,
		`INSERT INTO conversation_summaries
		 (id, conversation_id, user_id,
		  title, summary,
		  key_points, topics,
		  sentiment, entities, mentioned_entities,
		  action_items, questions,
		  decisions, decisions_made,
		  code_mentions,
		  embedding,
		  message_count, token_count, duration,
		  time_range_start, time_range_end,
		  metadata,
		  summarized_at,
		  created_at, updated_at)
		 VALUES ($1, $2, $3,
		         $4, $5,
		         $6, $7,
		         $8, $9, $10,
		         $11, $12,
		         $13, $14,
		         $15,
		         $16,
		         $17, $18, $19,
		         $20, $21,
		         $22,
		         NOW(),
		         $23, $24)
		 ON CONFLICT (conversation_id) DO UPDATE SET
		    title = EXCLUDED.title,
		    summary = EXCLUDED.summary,
		    key_points = EXCLUDED.key_points,
		    topics = EXCLUDED.topics,
		    sentiment = EXCLUDED.sentiment,
		    entities = EXCLUDED.entities,
		    mentioned_entities = EXCLUDED.mentioned_entities,
		    action_items = EXCLUDED.action_items,
		    questions = EXCLUDED.questions,
		    decisions = EXCLUDED.decisions,
		    decisions_made = EXCLUDED.decisions_made,
		    code_mentions = EXCLUDED.code_mentions,
		    embedding = EXCLUDED.embedding,
		    message_count = EXCLUDED.message_count,
		    token_count = EXCLUDED.token_count,
		    duration = EXCLUDED.duration,
		    time_range_start = EXCLUDED.time_range_start,
		    time_range_end = EXCLUDED.time_range_end,
		    metadata = EXCLUDED.metadata,
		    summarized_at = NOW(),
		    updated_at = EXCLUDED.updated_at`,
		analysis.ID, analysis.ConversationID, analysis.UserID,
		analysis.Title, analysis.Summary,
		analysis.KeyPoints, topicNames,
		sentimentJSON, entitiesJSON, mentionedJSON,
		actionItemTexts, questionTexts,
		decisionTexts, decisionTexts,
		codeMentionsJSON,
		embedding,
		analysis.MessageCount, analysis.TokenCount, analysis.Duration,
		timeStart, timeEnd,
		metadataJSON,
		analysis.CreatedAt, analysis.UpdatedAt)

	return err
}

// GetAnalysis retrieves a ConversationAnalysis by conversation ID.
// Structured fields are rehydrated from the metadata JSONB column when available,
// falling back to the plain TEXT[] columns.
func (s *ConversationIntelligenceService) GetAnalysis(ctx context.Context, conversationID string) (*ConversationAnalysis, error) {
	var analysis ConversationAnalysis
	var keyPoints, topics, actionItems, questions, decisions []string
	var sentimentJSON, entitiesJSON, codeMentionsJSON, metadataJSON []byte

	err := s.pool.QueryRow(ctx,
		`SELECT id, conversation_id, user_id, title, summary,
		        key_points, topics,
		        sentiment, entities,
		        action_items, questions, decisions,
		        code_mentions,
		        message_count, token_count, duration,
		        metadata,
		        created_at, updated_at
		 FROM conversation_summaries
		 WHERE conversation_id = $1`,
		conversationID).Scan(
		&analysis.ID, &analysis.ConversationID, &analysis.UserID, &analysis.Title, &analysis.Summary,
		&keyPoints, &topics,
		&sentimentJSON, &entitiesJSON,
		&actionItems, &questions, &decisions,
		&codeMentionsJSON,
		&analysis.MessageCount, &analysis.TokenCount, &analysis.Duration,
		&metadataJSON,
		&analysis.CreatedAt, &analysis.UpdatedAt)
	if err != nil {
		return nil, err
	}

	analysis.KeyPoints = keyPoints
	json.Unmarshal(sentimentJSON, &analysis.Sentiment)       //nolint:errcheck
	json.Unmarshal(entitiesJSON, &analysis.Entities)         //nolint:errcheck
	json.Unmarshal(codeMentionsJSON, &analysis.CodeMentions) //nolint:errcheck
	json.Unmarshal(metadataJSON, &analysis.Metadata)         //nolint:errcheck

	// Prefer detailed structured data stored in metadata over TEXT[] columns.
	if analysis.Metadata != nil {
		analysis.Topics = rehydrateSlice[ConversationTopic](analysis.Metadata, "topics_detail")
		analysis.ActionItems = rehydrateSlice[ActionItem](analysis.Metadata, "action_items_detail")
		analysis.Questions = rehydrateSlice[Question](analysis.Metadata, "questions_detail")
		analysis.Decisions = rehydrateSlice[ConversationDecision](analysis.Metadata, "decisions_detail")
	}

	// Fall back to TEXT[] columns when metadata detail is absent.
	if len(analysis.Topics) == 0 {
		for _, name := range topics {
			analysis.Topics = append(analysis.Topics, ConversationTopic{Name: name})
		}
	}
	if len(analysis.ActionItems) == 0 {
		for _, t := range actionItems {
			analysis.ActionItems = append(analysis.ActionItems, ActionItem{Description: t, Status: "pending"})
		}
	}
	if len(analysis.Questions) == 0 {
		for _, t := range questions {
			analysis.Questions = append(analysis.Questions, Question{Text: t})
		}
	}
	if len(analysis.Decisions) == 0 {
		for _, t := range decisions {
			analysis.Decisions = append(analysis.Decisions, ConversationDecision{Description: t})
		}
	}

	return &analysis, nil
}

// SearchConversations searches conversation summaries by title/summary text for a user.
func (s *ConversationIntelligenceService) SearchConversations(ctx context.Context, userID, query string, limit int) ([]ConversationAnalysis, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, conversation_id, user_id, title, summary, key_points, topics, sentiment,
		        entities, action_items, message_count, token_count, duration, created_at
		 FROM conversation_summaries
		 WHERE user_id = $1 AND (title ILIKE $2 OR summary ILIKE $2)
		 ORDER BY created_at DESC
		 LIMIT $3`,
		userID, "%"+query+"%", limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	analyses := make([]ConversationAnalysis, 0)
	for rows.Next() {
		var a ConversationAnalysis
		var keyPoints, topics, actionItems []string
		var sentimentJSON, entitiesJSON []byte

		if err := rows.Scan(
			&a.ID, &a.ConversationID, &a.UserID, &a.Title, &a.Summary,
			&keyPoints, &topics, &sentimentJSON, &entitiesJSON, &actionItems,
			&a.MessageCount, &a.TokenCount, &a.Duration, &a.CreatedAt,
		); err != nil {
			continue
		}

		a.KeyPoints = keyPoints
		for _, name := range topics {
			a.Topics = append(a.Topics, ConversationTopic{Name: name})
		}
		json.Unmarshal(sentimentJSON, &a.Sentiment) //nolint:errcheck
		json.Unmarshal(entitiesJSON, &a.Entities)   //nolint:errcheck
		for _, t := range actionItems {
			a.ActionItems = append(a.ActionItems, ActionItem{Description: t, Status: "pending"})
		}

		analyses = append(analyses, a)
	}

	return analyses, nil
}

// fetchConversationMessages fetches up to maxMessages messages for a conversation
// in chronological order.
func (s *ConversationIntelligenceService) fetchConversationMessages(ctx context.Context, conversationID string, maxMessages int) ([]Message, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT role::text, content, created_at
		FROM messages
		WHERE conversation_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`, conversationID, maxMessages)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tmp []Message
	for rows.Next() {
		var role, content string
		var ts time.Time
		if err := rows.Scan(&role, &content, &ts); err != nil {
			continue
		}
		tmp = append(tmp, Message{Role: role, Content: content, Timestamp: ts})
	}

	// Reverse DESC result to chronological order.
	for i, j := 0, len(tmp)-1; i < j; i, j = i+1, j-1 {
		tmp[i], tmp[j] = tmp[j], tmp[i]
	}
	return tmp, nil
}

// rehydrateSlice unmarshals a metadata key into a typed slice T.
// Returns nil when the key is absent or unmarshalling fails.
func rehydrateSlice[T any](metadata map[string]interface{}, key string) []T {
	raw, ok := metadata[key]
	if !ok {
		return nil
	}
	b, err := json.Marshal(raw)
	if err != nil {
		return nil
	}
	var result []T
	if json.Unmarshal(b, &result) != nil {
		return nil
	}
	return result
}
