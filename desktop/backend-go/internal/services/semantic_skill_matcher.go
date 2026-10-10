package services

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pgvector/pgvector-go"
)

const defaultConfidenceThreshold = 0.72

// SkillMatch is the result of a semantic skill lookup.
type SkillMatch struct {
	SkillID    string // e.g. "gmail.sync"
	SkillName  string
	Confidence float64 // cosine similarity (0-1); higher = better match
}

// SemanticSkillMatcher replaces the 11 hardcoded string patterns in
// osa_orchestrator.go with pgvector cosine similarity over sorx_skills.embedding.
//
// Workflow:
//  1. Embed the incoming user message via EmbeddingService
//  2. Run a nearest-neighbor query on sorx_skills.embedding
//  3. Return the best match if confidence >= threshold
type SemanticSkillMatcher struct {
	pool      *pgxpool.Pool
	embedding *EmbeddingService
	loader    *SkillLoaderService
	threshold float64
	logger    *slog.Logger
}

// NewSemanticSkillMatcher creates a matcher with the default confidence threshold (0.72).
func NewSemanticSkillMatcher(
	pool *pgxpool.Pool,
	embedding *EmbeddingService,
	loader *SkillLoaderService,
	logger *slog.Logger,
) *SemanticSkillMatcher {
	if logger == nil {
		logger = slog.Default()
	}
	return &SemanticSkillMatcher{
		pool:      pool,
		embedding: embedding,
		loader:    loader,
		threshold: defaultConfidenceThreshold,
		logger:    logger.With("component", "semantic_skill_matcher"),
	}
}

// WithThreshold returns a copy of the matcher with a custom confidence threshold.
func (m *SemanticSkillMatcher) WithThreshold(t float64) *SemanticSkillMatcher {
	cp := *m
	cp.threshold = t
	return &cp
}

// Match finds the best matching SORX skill for the given user message.
// Returns nil if no skill meets the confidence threshold.
func (m *SemanticSkillMatcher) Match(ctx context.Context, message string) (*SkillMatch, error) {
	if m.embedding == nil {
		return nil, nil
	}

	// Ensure skills have embeddings (backfill happens inside LoadAll)
	if _, err := m.loader.LoadAll(ctx); err != nil {
		m.logger.Warn("skill loader failed before matching, falling back", "error", err)
		return nil, nil
	}

	// Embed the user message
	emb, err := m.embedding.GenerateEmbedding(ctx, message)
	if err != nil {
		return nil, fmt.Errorf("semantic_skill_matcher: embed message: %w", err)
	}

	vec := pgvector.NewVector(emb)

	// Cosine similarity: 1 - cosine_distance = similarity
	// pgvector's <=> operator is cosine distance, so similarity = 1 - distance
	var skillID, skillName string
	var distance float64

	queryCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	err = m.pool.QueryRow(queryCtx, `
		SELECT id, name, embedding <=> $1 AS distance
		FROM sorx_skills
		WHERE enabled = true AND embedding IS NOT NULL
		ORDER BY distance ASC
		LIMIT 1
	`, vec).Scan(&skillID, &skillName, &distance)

	if err != nil {
		// No rows or DB error — degrade gracefully
		m.logger.Debug("no skill match found", "error", err)
		return nil, nil
	}

	confidence := 1.0 - distance
	m.logger.Info("semantic skill match",
		"skill", skillName,
		"confidence", confidence,
		"threshold", m.threshold,
	)

	if confidence < m.threshold {
		m.logger.Debug("skill match below threshold, discarding",
			"skill", skillName,
			"confidence", confidence,
		)
		return nil, nil
	}

	return &SkillMatch{
		SkillID:    skillName,
		SkillName:  skillName,
		Confidence: confidence,
	}, nil
}
