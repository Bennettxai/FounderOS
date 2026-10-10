package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pgvector/pgvector-go"
	"github.com/redis/go-redis/v9"
)

const (
	skillCacheTTL    = 5 * time.Minute
	skillCachePrefix = "sorx:skills:"
	skillCacheAll    = "sorx:skills:all"
)

// SORXSkill is a skill definition loaded from the sorx_skills table.
type SORXSkill struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Tier        string    `json:"tier"`
	Description string    `json:"description"`
	Embedding   []float32 `json:"embedding,omitempty"`
	Config      map[string]interface{} `json:"config"`
	Enabled     bool      `json:"enabled"`
}

// SkillLoaderService loads SORX skill definitions from the DB and caches them in Redis.
// On load, it generates pgvector embeddings for any skill that is missing one.
type SkillLoaderService struct {
	pool      *pgxpool.Pool
	redis     *redis.Client   // nil = cache disabled
	embedding *EmbeddingService
	logger    *slog.Logger
}

// NewSkillLoaderService creates a new SkillLoaderService.
// redisClient and embeddingService are optional — pass nil to disable caching/embedding.
func NewSkillLoaderService(
	pool *pgxpool.Pool,
	redisClient *redis.Client,
	embeddingService *EmbeddingService,
	logger *slog.Logger,
) *SkillLoaderService {
	if logger == nil {
		logger = slog.Default()
	}
	return &SkillLoaderService{
		pool:      pool,
		redis:     redisClient,
		embedding: embeddingService,
		logger:    logger.With("component", "skill_loader"),
	}
}

// LoadAll returns all enabled skills, using Redis cache when available.
// On cache miss it reads from DB and repopulates the cache.
func (s *SkillLoaderService) LoadAll(ctx context.Context) ([]*SORXSkill, error) {
	// Try cache first
	if s.redis != nil {
		if skills, err := s.loadFromCache(ctx); err == nil && len(skills) > 0 {
			return skills, nil
		}
	}

	skills, err := s.loadFromDB(ctx)
	if err != nil {
		return nil, fmt.Errorf("skill_loader: load from db: %w", err)
	}

	// Backfill embeddings for any skill that doesn't have one
	if s.embedding != nil {
		if err := s.backfillEmbeddings(ctx, skills); err != nil {
			// Non-fatal — log and continue
			s.logger.Warn("failed to backfill skill embeddings", "error", err)
		}
	}

	// Populate cache
	if s.redis != nil {
		if err := s.saveToCache(ctx, skills); err != nil {
			s.logger.Warn("failed to cache skills", "error", err)
		}
	}

	return skills, nil
}

// InvalidateCache removes the skill cache so the next LoadAll re-reads from DB.
func (s *SkillLoaderService) InvalidateCache(ctx context.Context) error {
	if s.redis == nil {
		return nil
	}
	return s.redis.Del(ctx, skillCacheAll).Err()
}

// loadFromDB reads all enabled skills from sorx_skills.
func (s *SkillLoaderService) loadFromDB(ctx context.Context) ([]*SORXSkill, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, name, tier, description, embedding, config, enabled
		FROM sorx_skills
		WHERE enabled = true
		ORDER BY name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var skills []*SORXSkill
	for rows.Next() {
		var skill SORXSkill
		var vec *pgvector.Vector
		var configRaw []byte

		if err := rows.Scan(
			&skill.ID,
			&skill.Name,
			&skill.Tier,
			&skill.Description,
			&vec,
			&configRaw,
			&skill.Enabled,
		); err != nil {
			return nil, fmt.Errorf("scan skill row: %w", err)
		}

		if vec != nil {
			skill.Embedding = vec.Slice()
		}

		if configRaw != nil {
			if err := json.Unmarshal(configRaw, &skill.Config); err != nil {
				skill.Config = map[string]interface{}{}
			}
		}

		skills = append(skills, &skill)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	s.logger.Info("loaded skills from db", "count", len(skills))
	return skills, nil
}

// backfillEmbeddings generates and persists embeddings for skills missing one.
func (s *SkillLoaderService) backfillEmbeddings(ctx context.Context, skills []*SORXSkill) error {
	for _, skill := range skills {
		if len(skill.Embedding) > 0 {
			continue
		}

		emb, err := s.embedding.GenerateEmbedding(ctx, skill.Description)
		if err != nil {
			s.logger.Warn("failed to generate embedding for skill",
				"skill", skill.Name, "error", err)
			continue
		}

		vec := pgvector.NewVector(emb)
		_, err = s.pool.Exec(ctx, `
			UPDATE sorx_skills SET embedding = $1 WHERE id = $2
		`, vec, skill.ID)
		if err != nil {
			s.logger.Warn("failed to persist embedding for skill",
				"skill", skill.Name, "error", err)
			continue
		}

		skill.Embedding = emb
		s.logger.Info("backfilled embedding for skill", "skill", skill.Name)
	}
	return nil
}

// loadFromCache reads skills from Redis.
func (s *SkillLoaderService) loadFromCache(ctx context.Context) ([]*SORXSkill, error) {
	data, err := s.redis.Get(ctx, skillCacheAll).Bytes()
	if err != nil {
		return nil, err
	}

	var skills []*SORXSkill
	if err := json.Unmarshal(data, &skills); err != nil {
		return nil, err
	}
	return skills, nil
}

// saveToCache writes skills to Redis with TTL.
func (s *SkillLoaderService) saveToCache(ctx context.Context, skills []*SORXSkill) error {
	data, err := json.Marshal(skills)
	if err != nil {
		return err
	}
	return s.redis.Set(ctx, skillCacheAll, data, skillCacheTTL).Err()
}
