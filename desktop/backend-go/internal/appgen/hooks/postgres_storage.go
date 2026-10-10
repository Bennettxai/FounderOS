package hooks

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresLearningStorage implements LearningStorage using PostgreSQL
type PostgresLearningStorage struct {
	pool   *pgxpool.Pool
	logger *slog.Logger
}

// NewPostgresLearningStorage creates a new PostgreSQL-backed learning storage
func NewPostgresLearningStorage(pool *pgxpool.Pool, logger *slog.Logger) *PostgresLearningStorage {
	return &PostgresLearningStorage{
		pool:   pool,
		logger: logger,
	}
}

// Save stores a learning record in the database
func (s *PostgresLearningStorage) Save(ctx context.Context, record LearningRecord) error {
	query := `
		INSERT INTO learning_records (
			timestamp, agent_type, task_id, success, duration_ms,
			tokens_used, patterns, classification, metadata
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id
	`

	// Convert patterns slice to JSONB
	patternsJSON, err := json.Marshal(record.Patterns)
	if err != nil {
		return fmt.Errorf("failed to marshal patterns: %w", err)
	}

	// Convert classification map to JSONB
	classificationJSON, err := json.Marshal(record.Classification)
	if err != nil {
		return fmt.Errorf("failed to marshal classification: %w", err)
	}

	// Create metadata JSONB (nil for empty — SimpleProtocol compatibility)
	var metadataJSON []byte
	if len(record.Classification) > 0 {
		// Store some useful metadata
		metadata := map[string]interface{}{
			"domain":     record.Classification["domain"],
			"complexity": record.Classification["complexity"],
			"value":      record.Classification["value"],
		}
		metadataJSON, err = json.Marshal(metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata: %w", err)
		}
	}

	var recordID string
	err = s.pool.QueryRow(
		ctx,
		query,
		record.Timestamp,
		record.AgentType,
		record.TaskID,
		record.Success,
		record.Duration.Milliseconds(),
		record.TokensUsed,
		patternsJSON,
		classificationJSON,
		metadataJSON,
	).Scan(&recordID)

	if err != nil {
		s.logger.ErrorContext(ctx, "failed to save learning record",
			"error", err,
			"agent_type", record.AgentType,
			"task_id", record.TaskID)
		return fmt.Errorf("failed to save learning record: %w", err)
	}

	s.logger.InfoContext(ctx, "learning record saved",
		"record_id", recordID,
		"agent_type", record.AgentType,
		"success", record.Success,
		"duration_ms", record.Duration.Milliseconds())

	return nil
}

// GetPatterns retrieves successful patterns for a specific agent type
func (s *PostgresLearningStorage) GetPatterns(ctx context.Context, agentType string, limit int) ([]string, error) {
	query := `
		SELECT patterns
		FROM learning_records
		WHERE agent_type = $1
		  AND success = true
		  AND jsonb_array_length(patterns) > 0
		ORDER BY timestamp DESC
		LIMIT $2
	`

	rows, err := s.pool.Query(ctx, query, agentType, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query patterns: %w", err)
	}
	defer rows.Close()

	var allPatterns []string
	for rows.Next() {
		var patternsJSON []byte
		if err := rows.Scan(&patternsJSON); err != nil {
			s.logger.WarnContext(ctx, "failed to scan pattern row", "error", err)
			continue
		}

		var patterns []string
		if err := json.Unmarshal(patternsJSON, &patterns); err != nil {
			s.logger.WarnContext(ctx, "failed to unmarshal patterns", "error", err)
			continue
		}

		allPatterns = append(allPatterns, patterns...)

		// Stop if we've collected enough patterns
		if len(allPatterns) >= limit {
			break
		}
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating patterns: %w", err)
	}

	// Trim to exact limit
	if len(allPatterns) > limit {
		allPatterns = allPatterns[:limit]
	}

	s.logger.InfoContext(ctx, "retrieved patterns",
		"agent_type", agentType,
		"pattern_count", len(allPatterns))

	return allPatterns, nil
}

// GetSuccessRate calculates success rate for an agent type
func (s *PostgresLearningStorage) GetSuccessRate(ctx context.Context, agentType string) (float64, error) {
	query := `
		SELECT
			COUNT(*) FILTER (WHERE success = true) as success_count,
			COUNT(*) as total_count
		FROM learning_records
		WHERE agent_type = $1
	`

	var successCount, totalCount int64
	err := s.pool.QueryRow(ctx, query, agentType).Scan(&successCount, &totalCount)
	if err != nil {
		return 0, fmt.Errorf("failed to calculate success rate: %w", err)
	}

	if totalCount == 0 {
		return 0, nil
	}

	return float64(successCount) / float64(totalCount), nil
}

// GetAverageDuration calculates average execution duration for an agent type
func (s *PostgresLearningStorage) GetAverageDuration(ctx context.Context, agentType string) (time.Duration, error) {
	query := `
		SELECT COALESCE(AVG(duration_ms), 0)
		FROM learning_records
		WHERE agent_type = $1
		  AND success = true
	`

	var avgMs float64
	err := s.pool.QueryRow(ctx, query, agentType).Scan(&avgMs)
	if err != nil {
		return 0, fmt.Errorf("failed to calculate average duration: %w", err)
	}

	return time.Duration(avgMs) * time.Millisecond, nil
}

// GetRecentErrors retrieves recent errors for analysis
func (s *PostgresLearningStorage) GetRecentErrors(ctx context.Context, agentType string, limit int) ([]LearningRecord, error) {
	query := `
		SELECT
			timestamp, agent_type, task_id, success, duration_ms,
			tokens_used, patterns, classification
		FROM learning_records
		WHERE agent_type = $1
		  AND success = false
		ORDER BY timestamp DESC
		LIMIT $2
	`

	rows, err := s.pool.Query(ctx, query, agentType, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query recent errors: %w", err)
	}
	defer rows.Close()

	var records []LearningRecord
	for rows.Next() {
		var record LearningRecord
		var durationMs int64
		var patternsJSON, classificationJSON []byte

		err := rows.Scan(
			&record.Timestamp,
			&record.AgentType,
			&record.TaskID,
			&record.Success,
			&durationMs,
			&record.TokensUsed,
			&patternsJSON,
			&classificationJSON,
		)
		if err != nil {
			s.logger.WarnContext(ctx, "failed to scan error record", "error", err)
			continue
		}

		record.Duration = time.Duration(durationMs) * time.Millisecond

		if err := json.Unmarshal(patternsJSON, &record.Patterns); err != nil {
			s.logger.WarnContext(ctx, "failed to unmarshal patterns", "error", err)
		}

		if err := json.Unmarshal(classificationJSON, &record.Classification); err != nil {
			s.logger.WarnContext(ctx, "failed to unmarshal classification", "error", err)
		}

		records = append(records, record)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating error records: %w", err)
	}

	return records, nil
}

// CleanOldRecords removes learning records older than the specified duration
func (s *PostgresLearningStorage) CleanOldRecords(ctx context.Context, olderThan time.Duration) (int64, error) {
	query := `
		DELETE FROM learning_records
		WHERE timestamp < $1
	`

	cutoffTime := time.Now().Add(-olderThan)
	result, err := s.pool.Exec(ctx, query, cutoffTime)
	if err != nil {
		return 0, fmt.Errorf("failed to clean old records: %w", err)
	}

	rowsAffected := result.RowsAffected()

	s.logger.InfoContext(ctx, "cleaned old learning records",
		"rows_deleted", rowsAffected,
		"cutoff_time", cutoffTime)

	return rowsAffected, nil
}

// Ensure PostgresLearningStorage implements LearningStorage interface
var _ LearningStorage = (*PostgresLearningStorage)(nil)
