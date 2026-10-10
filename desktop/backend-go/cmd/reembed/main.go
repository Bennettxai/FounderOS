package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pgvector/pgvector-go"
	"github.com/rhl/businessos-backend/internal/config"
	"github.com/rhl/businessos-backend/internal/database"
	"github.com/rhl/businessos-backend/internal/services"
)

func main() {
	var (
		tablesFlag = flag.String("tables", "all", "Comma-separated list of tables to re-embed (or 'all')")
		batchSize  = flag.Int("batch", 100, "Batch size per table")
		maxRows    = flag.Int("max", 0, "Max rows per table (0 = no limit)")
	)
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		slog.Error("config load failed", "error", err)
		os.Exit(1)
	}

	pool, err := database.Connect(cfg)
	if err != nil {
		slog.Error("db connect failed", "error", err)
		os.Exit(1)
	}
	defer database.Close()

	embed := services.NewEmbeddingService(pool, cfg.OllamaLocalURL)

	ctx := context.Background()

	selected := parseTables(*tablesFlag)

	jobs := []struct {
		name    string
		process func(context.Context, *services.EmbeddingService, int, int) (int, error)
	}{
		{"memories", reembedMemories},
		{"uploaded_documents", reembedUploadedDocuments},
		{"document_chunks", reembedDocumentChunks},
		{"conversations", reembedConversations},
		{"conversation_summaries", reembedConversationSummaries},
		{"voice_notes", reembedVoiceNotes},
		{"context_profiles", reembedContextProfiles},
		{"application_profiles", reembedApplicationProfiles},
		{"application_components", reembedApplicationComponents},
		{"application_api_endpoints", reembedApplicationAPIEndpoints},
	}

	for _, job := range jobs {
		if !selected.all && !selected.has(job.name) {
			continue
		}
		exists, err := tableExists(ctx, job.name)
		if err != nil {
			slog.Error("table exists check failed", "table", job.name, "error", err)
		}
		if !exists {
			slog.Info("table not found, skipping", "table", job.name)
			continue
		}

		slog.Info("reembed start", "table", job.name)
		count, err := job.process(ctx, embed, *batchSize, *maxRows)
		if err != nil {
			slog.Error("reembed failed", "table", job.name, "error", err)
		}
		slog.Info("reembed done", "table", job.name, "updated", count)
	}
}

type selection struct {
	all    bool
	tables map[string]struct{}
}

func parseTables(raw string) selection {
	raw = strings.TrimSpace(strings.ToLower(raw))
	if raw == "" || raw == "all" {
		return selection{all: true}
	}
	parts := strings.Split(raw, ",")
	m := make(map[string]struct{}, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		m[p] = struct{}{}
	}
	return selection{tables: m}
}

func (s selection) has(name string) bool {
	_, ok := s.tables[name]
	return ok
}

func tableExists(ctx context.Context, table string) (bool, error) {
	if database.Pool == nil {
		return false, fmt.Errorf("database pool not initialized")
	}
	var regclass *string
	err := database.Pool.QueryRow(ctx, `SELECT to_regclass($1)`, "public."+table).Scan(&regclass)
	if err != nil {
		return false, err
	}
	return regclass != nil, nil
}

func reembedVoiceNotes(ctx context.Context, embed *services.EmbeddingService, batchSize int, maxRows int) (int, error) {
	if database.Pool == nil {
		return 0, fmt.Errorf("database pool not initialized")
	}
	updated := 0
	processed := 0
	for {
		if maxRows > 0 && processed >= maxRows {
			break
		}
		limit := batchSize
		if maxRows > 0 && processed+limit > maxRows {
			limit = maxRows - processed
		}

		rows, err := database.Pool.Query(ctx, `
			SELECT id, transcript
			FROM voice_notes
			WHERE embedding IS NULL AND transcript IS NOT NULL AND transcript <> ''
			ORDER BY id
			LIMIT $1
		`, limit)
		if err != nil {
			return updated, err
		}

		batch := make([]struct {
			id   uuid.UUID
			text string
		}, 0, limit)
		for rows.Next() {
			var id uuid.UUID
			var transcript string
			if err := rows.Scan(&id, &transcript); err != nil {
				rows.Close()
				return updated, err
			}
			transcript = strings.TrimSpace(transcript)
			if transcript == "" {
				continue
			}
			batch = append(batch, struct {
				id   uuid.UUID
				text string
			}{id: id, text: transcript})
		}
		rows.Close()

		if len(batch) == 0 {
			break
		}

		for _, item := range batch {
			vec, err := embedText(ctx, embed, item.text)
			if err != nil {
				slog.Warn("voice_notes reembed item failed", "id", item.id, "error", err)
				continue
			}
			_, err = database.Pool.Exec(ctx, `UPDATE voice_notes SET embedding = $2 WHERE id = $1`, item.id, vec)
			if err != nil {
				return updated, err
			}
			updated++
			processed++
		}
	}
	return updated, nil
}

func reembedMemories(ctx context.Context, embed *services.EmbeddingService, batchSize int, maxRows int) (int, error) {
	if database.Pool == nil {
		return 0, fmt.Errorf("database pool not initialized")
	}
	updated := 0
	processed := 0
	for {
		if maxRows > 0 && processed >= maxRows {
			break
		}
		limit := batchSize
		if maxRows > 0 && processed+limit > maxRows {
			limit = maxRows - processed
		}

		rows, err := database.Pool.Query(ctx, `
			SELECT id, COALESCE(title, ''), COALESCE(summary, ''), COALESCE(content, '')
			FROM memories
			WHERE embedding IS NULL
			ORDER BY created_at DESC
			LIMIT $1
		`, limit)
		if err != nil {
			return updated, err
		}

		batch := make([]struct {
			id   uuid.UUID
			text string
		}, 0, limit)
		for rows.Next() {
			var id uuid.UUID
			var title, summary, content string
			if err := rows.Scan(&id, &title, &summary, &content); err != nil {
				rows.Close()
				return updated, err
			}
			text := strings.TrimSpace(strings.Join([]string{title, summary, content}, "\n\n"))
			if text == "" {
				continue
			}
			batch = append(batch, struct {
				id   uuid.UUID
				text string
			}{id: id, text: text})
		}
		rows.Close()

		if len(batch) == 0 {
			break
		}

		for _, item := range batch {
			vec, err := embedText(ctx, embed, item.text)
			if err != nil {
				slog.Warn("memories reembed item failed", "id", item.id, "error", err)
				continue
			}
			_, err = database.Pool.Exec(ctx, `UPDATE memories SET embedding = $2, embedding_model = $3 WHERE id = $1`, item.id, vec, "nomic-embed-text")
			if err != nil {
				return updated, err
			}
			updated++
			processed++
		}
	}
	return updated, nil
}

func reembedDocumentChunks(ctx context.Context, embed *services.EmbeddingService, batchSize int, maxRows int) (int, error) {
	if database.Pool == nil {
		return 0, fmt.Errorf("database pool not initialized")
	}
	updated := 0
	processed := 0
	for {
		if maxRows > 0 && processed >= maxRows {
			break
		}
		limit := batchSize
		if maxRows > 0 && processed+limit > maxRows {
			limit = maxRows - processed
		}

		rows, err := database.Pool.Query(ctx, `
			SELECT id, content
			FROM document_chunks
			WHERE embedding IS NULL AND content IS NOT NULL AND content <> ''
			ORDER BY created_at DESC
			LIMIT $1
		`, limit)
		if err != nil {
			return updated, err
		}

		batch := make([]struct {
			id   uuid.UUID
			text string
		}, 0, limit)
		for rows.Next() {
			var id uuid.UUID
			var content string
			if err := rows.Scan(&id, &content); err != nil {
				rows.Close()
				return updated, err
			}
			content = strings.TrimSpace(content)
			if content == "" {
				continue
			}
			batch = append(batch, struct {
				id   uuid.UUID
				text string
			}{id: id, text: content})
		}
		rows.Close()

		if len(batch) == 0 {
			break
		}

		for _, item := range batch {
			vec, err := embedText(ctx, embed, item.text)
			if err != nil {
				slog.Warn("document_chunks reembed item failed", "id", item.id, "error", err)
				continue
			}
			_, err = database.Pool.Exec(ctx, `UPDATE document_chunks SET embedding = $2 WHERE id = $1`, item.id, vec)
			if err != nil {
				return updated, err
			}
			updated++
			processed++
		}
	}
	return updated, nil
}

func reembedUploadedDocuments(ctx context.Context, embed *services.EmbeddingService, batchSize int, maxRows int) (int, error) {
	if database.Pool == nil {
		return 0, fmt.Errorf("database pool not initialized")
	}
	updated := 0
	processed := 0
	for {
		if maxRows > 0 && processed >= maxRows {
			break
		}
		limit := batchSize
		if maxRows > 0 && processed+limit > maxRows {
			limit = maxRows - processed
		}

		rows, err := database.Pool.Query(ctx, `
			SELECT id, COALESCE(display_name, ''), COALESCE(description, ''), COALESCE(extracted_text, '')
			FROM uploaded_documents
			WHERE embedding IS NULL
			ORDER BY created_at DESC
			LIMIT $1
		`, limit)
		if err != nil {
			return updated, err
		}

		batch := make([]struct {
			id   uuid.UUID
			text string
		}, 0, limit)
		for rows.Next() {
			var id uuid.UUID
			var displayName, description, extracted string
			if err := rows.Scan(&id, &displayName, &description, &extracted); err != nil {
				rows.Close()
				return updated, err
			}
			text := strings.TrimSpace(strings.Join([]string{displayName, description, extracted}, "\n\n"))
			if text == "" {
				continue
			}
			batch = append(batch, struct {
				id   uuid.UUID
				text string
			}{id: id, text: text})
		}
		rows.Close()

		if len(batch) == 0 {
			break
		}

		for _, item := range batch {
			vec, err := embedText(ctx, embed, item.text)
			if err != nil {
				slog.Warn("uploaded_documents reembed item failed", "id", item.id, "error", err)
				continue
			}
			_, err = database.Pool.Exec(ctx, `UPDATE uploaded_documents SET embedding = $2 WHERE id = $1`, item.id, vec)
			if err != nil {
				return updated, err
			}
			updated++
			processed++
		}
	}
	return updated, nil
}

func reembedConversations(ctx context.Context, embed *services.EmbeddingService, batchSize int, maxRows int) (int, error) {
	if database.Pool == nil {
		return 0, fmt.Errorf("database pool not initialized")
	}
	updated := 0
	processed := 0
	for {
		if maxRows > 0 && processed >= maxRows {
			break
		}
		limit := batchSize
		if maxRows > 0 && processed+limit > maxRows {
			limit = maxRows - processed
		}

		rows, err := database.Pool.Query(ctx, `
			SELECT id, COALESCE(title, ''), COALESCE(summary, '')
			FROM conversations
			WHERE embedding IS NULL
			ORDER BY id
			LIMIT $1
		`, limit)
		if err != nil {
			return updated, err
		}

		batch := make([]struct {
			id   uuid.UUID
			text string
		}, 0, limit)
		for rows.Next() {
			var id uuid.UUID
			var title, summary string
			if err := rows.Scan(&id, &title, &summary); err != nil {
				rows.Close()
				return updated, err
			}
			text := strings.TrimSpace(strings.Join([]string{title, summary}, "\n\n"))
			if text == "" {
				continue
			}
			batch = append(batch, struct {
				id   uuid.UUID
				text string
			}{id: id, text: text})
		}
		rows.Close()

		if len(batch) == 0 {
			break
		}

		for _, item := range batch {
			vec, err := embedText(ctx, embed, item.text)
			if err != nil {
				slog.Warn("conversations reembed item failed", "id", item.id, "error", err)
				continue
			}
			_, err = database.Pool.Exec(ctx, `UPDATE conversations SET embedding = $2 WHERE id = $1`, item.id, vec)
			if err != nil {
				return updated, err
			}
			updated++
			processed++
		}
	}
	return updated, nil
}

func reembedConversationSummaries(ctx context.Context, embed *services.EmbeddingService, batchSize int, maxRows int) (int, error) {
	if database.Pool == nil {
		return 0, fmt.Errorf("database pool not initialized")
	}
	updated := 0
	processed := 0
	for {
		if maxRows > 0 && processed >= maxRows {
			break
		}
		limit := batchSize
		if maxRows > 0 && processed+limit > maxRows {
			limit = maxRows - processed
		}

		rows, err := database.Pool.Query(ctx, `
			SELECT id, summary
			FROM conversation_summaries
			WHERE embedding IS NULL AND summary IS NOT NULL AND summary <> ''
			ORDER BY id
			LIMIT $1
		`, limit)
		if err != nil {
			return updated, err
		}

		batch := make([]struct {
			id   uuid.UUID
			text string
		}, 0, limit)
		for rows.Next() {
			var id uuid.UUID
			var summary string
			if err := rows.Scan(&id, &summary); err != nil {
				rows.Close()
				return updated, err
			}
			summary = strings.TrimSpace(summary)
			if summary == "" {
				continue
			}
			batch = append(batch, struct {
				id   uuid.UUID
				text string
			}{id: id, text: summary})
		}
		rows.Close()

		if len(batch) == 0 {
			break
		}

		for _, item := range batch {
			vec, err := embedText(ctx, embed, item.text)
			if err != nil {
				slog.Warn("conversation_summaries reembed item failed", "id", item.id, "error", err)
				continue
			}
			_, err = database.Pool.Exec(ctx, `UPDATE conversation_summaries SET embedding = $2 WHERE id = $1`, item.id, vec)
			if err != nil {
				return updated, err
			}
			updated++
			processed++
		}
	}
	return updated, nil
}

func reembedContextProfiles(ctx context.Context, embed *services.EmbeddingService, batchSize int, maxRows int) (int, error) {
	if database.Pool == nil {
		return 0, fmt.Errorf("database pool not initialized")
	}
	updated := 0
	processed := 0
	for {
		if maxRows > 0 && processed >= maxRows {
			break
		}
		limit := batchSize
		if maxRows > 0 && processed+limit > maxRows {
			limit = maxRows - processed
		}

		rows, err := database.Pool.Query(ctx, `
			SELECT id, COALESCE(name, ''), COALESCE(description, ''), COALESCE(summary, '')
			FROM context_profiles
			WHERE embedding IS NULL
			ORDER BY id
			LIMIT $1
		`, limit)
		if err != nil {
			return updated, err
		}

		batch := make([]struct {
			id   uuid.UUID
			text string
		}, 0, limit)
		for rows.Next() {
			var id uuid.UUID
			var name, desc, summary string
			if err := rows.Scan(&id, &name, &desc, &summary); err != nil {
				rows.Close()
				return updated, err
			}
			text := strings.TrimSpace(strings.Join([]string{name, desc, summary}, "\n\n"))
			if text == "" {
				continue
			}
			batch = append(batch, struct {
				id   uuid.UUID
				text string
			}{id: id, text: text})
		}
		rows.Close()

		if len(batch) == 0 {
			break
		}

		for _, item := range batch {
			vec, err := embedText(ctx, embed, item.text)
			if err != nil {
				slog.Warn("context_profiles reembed item failed", "id", item.id, "error", err)
				continue
			}
			_, err = database.Pool.Exec(ctx, `UPDATE context_profiles SET embedding = $2 WHERE id = $1`, item.id, vec)
			if err != nil {
				return updated, err
			}
			updated++
			processed++
		}
	}
	return updated, nil
}

func reembedApplicationProfiles(ctx context.Context, embed *services.EmbeddingService, batchSize int, maxRows int) (int, error) {
	if database.Pool == nil {
		return 0, fmt.Errorf("database pool not initialized")
	}
	updated := 0
	processed := 0
	for {
		if maxRows > 0 && processed >= maxRows {
			break
		}
		limit := batchSize
		if maxRows > 0 && processed+limit > maxRows {
			limit = maxRows - processed
		}

		rows, err := database.Pool.Query(ctx, `
			SELECT id, COALESCE(name, ''), COALESCE(description, ''), COALESCE(coding_standards, ''), COALESCE(readme_summary, '')
			FROM application_profiles
			WHERE embedding IS NULL
			ORDER BY id
			LIMIT $1
		`, limit)
		if err != nil {
			return updated, err
		}

		batch := make([]struct {
			id   uuid.UUID
			text string
		}, 0, limit)
		for rows.Next() {
			var id uuid.UUID
			var name, desc, standards, readme string
			if err := rows.Scan(&id, &name, &desc, &standards, &readme); err != nil {
				rows.Close()
				return updated, err
			}
			text := strings.TrimSpace(strings.Join([]string{name, desc, standards, readme}, "\n\n"))
			if text == "" {
				continue
			}
			batch = append(batch, struct {
				id   uuid.UUID
				text string
			}{id: id, text: text})
		}
		rows.Close()

		if len(batch) == 0 {
			break
		}

		for _, item := range batch {
			vec, err := embedText(ctx, embed, item.text)
			if err != nil {
				slog.Warn("application_profiles reembed item failed", "id", item.id, "error", err)
				continue
			}
			_, err = database.Pool.Exec(ctx, `UPDATE application_profiles SET embedding = $2 WHERE id = $1`, item.id, vec)
			if err != nil {
				return updated, err
			}
			updated++
			processed++
		}
	}
	return updated, nil
}

func reembedApplicationComponents(ctx context.Context, embed *services.EmbeddingService, batchSize int, maxRows int) (int, error) {
	if database.Pool == nil {
		return 0, fmt.Errorf("database pool not initialized")
	}
	updated := 0
	processed := 0
	for {
		if maxRows > 0 && processed >= maxRows {
			break
		}
		limit := batchSize
		if maxRows > 0 && processed+limit > maxRows {
			limit = maxRows - processed
		}

		rows, err := database.Pool.Query(ctx, `
			SELECT id, COALESCE(name, ''), COALESCE(file_path, ''), COALESCE(component_type, ''), COALESCE(description, '')
			FROM application_components
			WHERE embedding IS NULL
			ORDER BY id
			LIMIT $1
		`, limit)
		if err != nil {
			return updated, err
		}

		batch := make([]struct {
			id   uuid.UUID
			text string
		}, 0, limit)
		for rows.Next() {
			var id uuid.UUID
			var name, path, ctype, desc string
			if err := rows.Scan(&id, &name, &path, &ctype, &desc); err != nil {
				rows.Close()
				return updated, err
			}
			text := strings.TrimSpace(strings.Join([]string{name, ctype, path, desc}, "\n\n"))
			if text == "" {
				continue
			}
			batch = append(batch, struct {
				id   uuid.UUID
				text string
			}{id: id, text: text})
		}
		rows.Close()

		if len(batch) == 0 {
			break
		}

		for _, item := range batch {
			vec, err := embedText(ctx, embed, item.text)
			if err != nil {
				slog.Warn("application_components reembed item failed", "id", item.id, "error", err)
				continue
			}
			_, err = database.Pool.Exec(ctx, `UPDATE application_components SET embedding = $2 WHERE id = $1`, item.id, vec)
			if err != nil {
				return updated, err
			}
			updated++
			processed++
		}
	}
	return updated, nil
}

func reembedApplicationAPIEndpoints(ctx context.Context, embed *services.EmbeddingService, batchSize int, maxRows int) (int, error) {
	if database.Pool == nil {
		return 0, fmt.Errorf("database pool not initialized")
	}
	updated := 0
	processed := 0
	for {
		if maxRows > 0 && processed >= maxRows {
			break
		}
		limit := batchSize
		if maxRows > 0 && processed+limit > maxRows {
			limit = maxRows - processed
		}

		rows, err := database.Pool.Query(ctx, `
			SELECT id, method, path, COALESCE(summary, ''), COALESCE(description, '')
			FROM application_api_endpoints
			WHERE embedding IS NULL
			ORDER BY id
			LIMIT $1
		`, limit)
		if err != nil {
			return updated, err
		}

		batch := make([]struct {
			id   uuid.UUID
			text string
		}, 0, limit)
		for rows.Next() {
			var id uuid.UUID
			var method, path, summary, desc string
			if err := rows.Scan(&id, &method, &path, &summary, &desc); err != nil {
				rows.Close()
				return updated, err
			}
			text := strings.TrimSpace(strings.Join([]string{method + " " + path, summary, desc}, "\n\n"))
			if text == "" {
				continue
			}
			batch = append(batch, struct {
				id   uuid.UUID
				text string
			}{id: id, text: text})
		}
		rows.Close()

		if len(batch) == 0 {
			break
		}

		for _, item := range batch {
			vec, err := embedText(ctx, embed, item.text)
			if err != nil {
				slog.Warn("application_api_endpoints reembed item failed", "id", item.id, "error", err)
				continue
			}
			_, err = database.Pool.Exec(ctx, `UPDATE application_api_endpoints SET embedding = $2 WHERE id = $1`, item.id, vec)
			if err != nil {
				return updated, err
			}
			updated++
			processed++
		}
	}
	return updated, nil
}

func embedText(ctx context.Context, embed *services.EmbeddingService, text string) (pgvector.Vector, error) {
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	v, err := embed.GenerateEmbedding(ctx, text)
	if err != nil {
		return pgvector.Vector{}, err
	}
	return pgvector.NewVector(v), nil
}
