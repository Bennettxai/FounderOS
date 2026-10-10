package services

import (
	"context"
	"strings"
	"time"

	"github.com/pgvector/pgvector-go"
)

// SearchTree searches the context tree based on parameters
func (s *ContextService) SearchTree(ctx context.Context, userID string, params TreeSearchParams) ([]TreeSearchResult, error) {
	if params.MaxResults <= 0 {
		params.MaxResults = 10
	}

	switch params.SearchType {
	case "semantic":
		return s.semanticSearch(ctx, userID, params)
	case "title":
		return s.titleSearch(ctx, userID, params)
	case "content":
		return s.contentSearch(ctx, userID, params)
	default:
		return s.titleSearch(ctx, userID, params)
	}
}

// semanticSearch performs semantic search using embeddings
func (s *ContextService) semanticSearch(ctx context.Context, userID string, params TreeSearchParams) ([]TreeSearchResult, error) {
	if s.embeddingService == nil {
		return s.titleSearch(ctx, userID, params)
	}

	// Generate query embedding
	queryEmbedding, err := s.embeddingService.GenerateEmbedding(ctx, params.Query)
	if err != nil {
		s.logger.Warn("failed to generate embedding, falling back to title search", "error", err)
		return s.titleSearch(ctx, userID, params)
	}

	vec := pgvector.NewVector(queryEmbedding)
	var results []TreeSearchResult

	// Search memories
	if containsType(params.EntityTypes, "memories") || len(params.EntityTypes) == 0 {
		memResults, _ := s.searchMemoriesSemantic(ctx, userID, vec, params)
		results = append(results, memResults...)
	}

	// Search documents
	if containsType(params.EntityTypes, "documents") || len(params.EntityTypes) == 0 {
		docResults, _ := s.searchDocumentsSemantic(ctx, userID, vec, params)
		results = append(results, docResults...)
	}

	// Search voice notes
	if containsType(params.EntityTypes, "voice_notes") || len(params.EntityTypes) == 0 {
		voiceResults, _ := s.searchVoiceNotesSemantic(ctx, userID, vec, params)
		results = append(results, voiceResults...)
	}

	// Search conversation summaries (past chat history)
	if containsType(params.EntityTypes, "conversations") || containsType(params.EntityTypes, "conversation_summaries") || len(params.EntityTypes) == 0 {
		convResults, _ := s.searchConversationSummariesSemantic(ctx, userID, vec, params)
		results = append(results, convResults...)
	}

	// Sort by relevance and limit
	sortByRelevance(results)
	if len(results) > params.MaxResults {
		results = results[:params.MaxResults]
	}

	return results, nil
}

// searchConversationSummariesSemantic searches conversation summaries using embedding
func (s *ContextService) searchConversationSummariesSemantic(ctx context.Context, userID string, vec pgvector.Vector, params TreeSearchParams) ([]TreeSearchResult, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT cs.conversation_id,
		       COALESCE(cs.title, ''),
		       LEFT(cs.summary, 200) as snippet,
		       COALESCE(cs.summarized_at, cs.created_at) as ts,
		       1 - (cs.embedding <=> $1) as similarity
		FROM conversation_summaries cs
		WHERE cs.user_id = $2 AND cs.embedding IS NOT NULL
		ORDER BY cs.embedding <=> $1
		LIMIT $3
	`, vec, userID, params.MaxResults)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []TreeSearchResult
	for rows.Next() {
		var r TreeSearchResult
		var title string
		var snippet string
		var ts time.Time
		if err := rows.Scan(&r.ID, &title, &snippet, &ts, &r.RelevanceScore); err != nil {
			continue
		}
		r.Type = "conversation"
		if strings.TrimSpace(title) != "" {
			r.Title = title
		} else {
			r.Title = "Conversation (" + ts.UTC().Format("2006-01-02") + ")"
		}
		r.Summary = snippet
		r.TreePath = []string{"Conversations"}
		results = append(results, r)
	}

	return results, nil
}

// searchVoiceNotesSemantic searches voice notes using embedding
func (s *ContextService) searchVoiceNotesSemantic(ctx context.Context, userID string, vec pgvector.Vector, params TreeSearchParams) ([]TreeSearchResult, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, LEFT(transcript, 200) as snippet, created_at, 1 - (embedding <=> $1) as similarity
		FROM voice_notes
		WHERE user_id = $2 AND is_context_source = true AND embedding IS NOT NULL
		ORDER BY embedding <=> $1
		LIMIT $3
	`, vec, userID, params.MaxResults)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []TreeSearchResult
	for rows.Next() {
		var r TreeSearchResult
		var snippet string
		var createdAt time.Time
		if err := rows.Scan(&r.ID, &snippet, &createdAt, &r.RelevanceScore); err != nil {
			continue
		}
		r.Type = "voice_note"
		r.Title = "Voice note (" + createdAt.UTC().Format("2006-01-02") + ")"
		r.Summary = snippet
		r.TreePath = []string{"Voice Notes"}
		results = append(results, r)
	}

	return results, nil
}

// searchMemoriesSemantic searches memories using embedding
func (s *ContextService) searchMemoriesSemantic(ctx context.Context, userID string, vec pgvector.Vector, params TreeSearchParams) ([]TreeSearchResult, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, title, memory_type, summary, 1 - (embedding <=> $1) as similarity
		FROM memories
		WHERE user_id = $2 AND is_active = true AND embedding IS NOT NULL
		ORDER BY embedding <=> $1
		LIMIT $3
	`, vec, userID, params.MaxResults)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []TreeSearchResult
	for rows.Next() {
		var r TreeSearchResult
		var memType string
		if err := rows.Scan(&r.ID, &r.Title, &memType, &r.Summary, &r.RelevanceScore); err != nil {
			continue
		}
		r.Type = "memory"
		r.TreePath = []string{"Memories", memType}
		results = append(results, r)
	}

	return results, nil
}

// searchDocumentsSemantic searches documents using embedding
func (s *ContextService) searchDocumentsSemantic(ctx context.Context, userID string, vec pgvector.Vector, params TreeSearchParams) ([]TreeSearchResult, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, display_name, document_type, description, 1 - (embedding <=> $1) as similarity
		FROM uploaded_documents
		WHERE user_id = $2 AND embedding IS NOT NULL
		ORDER BY embedding <=> $1
		LIMIT $3
	`, vec, userID, params.MaxResults)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []TreeSearchResult
	for rows.Next() {
		var r TreeSearchResult
		var docType string
		if err := rows.Scan(&r.ID, &r.Title, &docType, &r.Summary, &r.RelevanceScore); err != nil {
			continue
		}
		r.Type = "document"
		r.TreePath = []string{"Documents", docType}
		results = append(results, r)
	}

	return results, nil
}

// titleSearch searches by title
func (s *ContextService) titleSearch(ctx context.Context, userID string, params TreeSearchParams) ([]TreeSearchResult, error) {
	var results []TreeSearchResult
	searchPattern := "%" + params.Query + "%"

	// Search memories by title
	if containsType(params.EntityTypes, "memories") || len(params.EntityTypes) == 0 {
		rows, _ := s.pool.Query(ctx, `
			SELECT id, title, memory_type, summary
			FROM memories
			WHERE user_id = $1 AND is_active = true AND (title ILIKE $2 OR summary ILIKE $2)
			ORDER BY importance_score DESC
			LIMIT $3
		`, userID, searchPattern, params.MaxResults)
		if rows != nil {
			for rows.Next() {
				var r TreeSearchResult
				var memType string
				if err := rows.Scan(&r.ID, &r.Title, &memType, &r.Summary); err != nil {
					continue
				}
				r.Type = "memory"
				r.TreePath = []string{"Memories", memType}
				r.RelevanceScore = 0.8
				results = append(results, r)
			}
			rows.Close()
		}
	}

	// Search documents by title
	if containsType(params.EntityTypes, "documents") || len(params.EntityTypes) == 0 {
		rows, _ := s.pool.Query(ctx, `
			SELECT id, COALESCE(display_name, filename), document_type, description
			FROM uploaded_documents
			WHERE user_id = $1 AND (display_name ILIKE $2 OR filename ILIKE $2 OR description ILIKE $2)
			ORDER BY created_at DESC
			LIMIT $3
		`, userID, searchPattern, params.MaxResults)
		if rows != nil {
			for rows.Next() {
				var r TreeSearchResult
				var docType *string
				if err := rows.Scan(&r.ID, &r.Title, &docType, &r.Summary); err != nil {
					continue
				}
				r.Type = "document"
				if docType != nil {
					r.TreePath = []string{"Documents", *docType}
				} else {
					r.TreePath = []string{"Documents"}
				}
				r.RelevanceScore = 0.7
				results = append(results, r)
			}
			rows.Close()
		}
	}

	// Search voice notes by transcript (best-effort "title")
	if containsType(params.EntityTypes, "voice_notes") || len(params.EntityTypes) == 0 {
		rows, _ := s.pool.Query(ctx, `
			SELECT id, created_at, LEFT(transcript, 200)
			FROM voice_notes
			WHERE user_id = $1 AND is_context_source = true AND transcript ILIKE $2
			ORDER BY created_at DESC
			LIMIT $3
		`, userID, searchPattern, params.MaxResults)
		if rows != nil {
			for rows.Next() {
				var r TreeSearchResult
				var createdAt time.Time
				if err := rows.Scan(&r.ID, &createdAt, &r.Summary); err != nil {
					continue
				}
				r.Type = "voice_note"
				r.Title = "Voice note (" + createdAt.UTC().Format("2006-01-02") + ")"
				r.TreePath = []string{"Voice Notes"}
				r.RelevanceScore = 0.65
				results = append(results, r)
			}
			rows.Close()
		}
	}

	if len(results) > params.MaxResults {
		results = results[:params.MaxResults]
	}

	return results, nil
}

// contentSearch searches by content
func (s *ContextService) contentSearch(ctx context.Context, userID string, params TreeSearchParams) ([]TreeSearchResult, error) {
	var results []TreeSearchResult
	searchPattern := "%" + params.Query + "%"

	// Search memories by content
	if containsType(params.EntityTypes, "memories") || len(params.EntityTypes) == 0 {
		rows, _ := s.pool.Query(ctx, `
			SELECT id, title, memory_type, LEFT(content, 200)
			FROM memories
			WHERE user_id = $1 AND is_active = true AND content ILIKE $2
			ORDER BY importance_score DESC
			LIMIT $3
		`, userID, searchPattern, params.MaxResults)
		if rows != nil {
			for rows.Next() {
				var r TreeSearchResult
				var memType string
				if err := rows.Scan(&r.ID, &r.Title, &memType, &r.Summary); err != nil {
					continue
				}
				r.Type = "memory"
				r.TreePath = []string{"Memories", memType}
				r.RelevanceScore = 0.7
				results = append(results, r)
			}
			rows.Close()
		}
	}

	// Search documents by extracted text
	if containsType(params.EntityTypes, "documents") || len(params.EntityTypes) == 0 {
		rows, _ := s.pool.Query(ctx, `
			SELECT id, COALESCE(display_name, filename), document_type, LEFT(extracted_text, 200)
			FROM uploaded_documents
			WHERE user_id = $1 AND extracted_text ILIKE $2
			ORDER BY created_at DESC
			LIMIT $3
		`, userID, searchPattern, params.MaxResults)
		if rows != nil {
			for rows.Next() {
				var r TreeSearchResult
				var docType *string
				if err := rows.Scan(&r.ID, &r.Title, &docType, &r.Summary); err != nil {
					continue
				}
				r.Type = "document"
				if docType != nil {
					r.TreePath = []string{"Documents", *docType}
				} else {
					r.TreePath = []string{"Documents"}
				}
				r.RelevanceScore = 0.6
				results = append(results, r)
			}
			rows.Close()
		}
	}

	// Search voice notes by transcript
	if containsType(params.EntityTypes, "voice_notes") || len(params.EntityTypes) == 0 {
		rows, _ := s.pool.Query(ctx, `
			SELECT id, created_at, LEFT(transcript, 200)
			FROM voice_notes
			WHERE user_id = $1 AND is_context_source = true AND transcript ILIKE $2
			ORDER BY created_at DESC
			LIMIT $3
		`, userID, searchPattern, params.MaxResults)
		if rows != nil {
			for rows.Next() {
				var r TreeSearchResult
				var createdAt time.Time
				if err := rows.Scan(&r.ID, &createdAt, &r.Summary); err != nil {
					continue
				}
				r.Type = "voice_note"
				r.Title = "Voice note (" + createdAt.UTC().Format("2006-01-02") + ")"
				r.TreePath = []string{"Voice Notes"}
				r.RelevanceScore = 0.6
				results = append(results, r)
			}
			rows.Close()
		}
	}

	if len(results) > params.MaxResults {
		results = results[:params.MaxResults]
	}

	return results, nil
}

// containsType reports whether the slice contains the given type string
func containsType(types []string, t string) bool {
	for _, typ := range types {
		if typ == t {
			return true
		}
	}
	return false
}

// sortByRelevance sorts results descending by RelevanceScore (simple bubble sort for small arrays)
func sortByRelevance(results []TreeSearchResult) {
	for i := 0; i < len(results); i++ {
		for j := i + 1; j < len(results); j++ {
			if results[j].RelevanceScore > results[i].RelevanceScore {
				results[i], results[j] = results[j], results[i]
			}
		}
	}
}
