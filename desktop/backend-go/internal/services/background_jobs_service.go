package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// =====================================================================
// TYPES
// =====================================================================

type BackgroundJobsService struct {
	pool *pgxpool.Pool
}

type BackgroundJob struct {
	ID            uuid.UUID              `json:"id"`
	JobType       string                 `json:"job_type"`
	Payload       map[string]interface{} `json:"payload"`
	ScheduledAt   time.Time              `json:"scheduled_at"`
	Priority      int                    `json:"priority"`
	Status        string                 `json:"status"` // pending, running, completed, failed, cancelled
	StartedAt     *time.Time             `json:"started_at,omitempty"`
	CompletedAt   *time.Time             `json:"completed_at,omitempty"`
	WorkerID      *string                `json:"worker_id,omitempty"`
	LockedUntil   *time.Time             `json:"locked_until,omitempty"`
	AttemptCount  int                    `json:"attempt_count"`
	MaxAttempts   int                    `json:"max_attempts"`
	LastError     *string                `json:"last_error,omitempty"`
	Result        map[string]interface{} `json:"result,omitempty"`
	CreatedAt     time.Time              `json:"created_at"`
}

type ScheduledJob struct {
	ID             uuid.UUID              `json:"id"`
	JobType        string                 `json:"job_type"`
	Payload        map[string]interface{} `json:"payload"`
	CronExpression string                 `json:"cron_expression"`
	Timezone       string                 `json:"timezone"`
	IsActive       bool                   `json:"is_active"`
	LastRunAt      *time.Time             `json:"last_run_at,omitempty"`
	NextRunAt      *time.Time             `json:"next_run_at,omitempty"`
	Name           *string                `json:"name,omitempty"`
	Description    *string                `json:"description,omitempty"`
	CreatedAt      time.Time              `json:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at"`
}

type JobListFilters struct {
	Status    *string
	JobType   *string
	Limit     int
	Offset    int
	SortBy    string // created_at, priority, scheduled_at
	SortOrder string // asc, desc
}

// =====================================================================
// CONSTRUCTOR
// =====================================================================

func NewBackgroundJobsService(pool *pgxpool.Pool) *BackgroundJobsService {
	return &BackgroundJobsService{pool: pool}
}

// =====================================================================
// JOB MANAGEMENT
// =====================================================================

// EnqueueJob creates a new background job
func (s *BackgroundJobsService) EnqueueJob(
	ctx context.Context,
	jobType string,
	payload map[string]interface{},
	priority int,
	maxAttempts int,
	scheduledAt *time.Time,
) (*BackgroundJob, error) {
	if maxAttempts <= 0 {
		maxAttempts = 3 // Default
	}

	if scheduledAt == nil {
		now := time.Now()
		scheduledAt = &now
	}

	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to marshal payload", "error", err)
		return nil, fmt.Errorf("marshal payload: %w", err)
	}

	var job BackgroundJob
	err = s.pool.QueryRow(ctx, `
		INSERT INTO background_jobs (
			job_type, payload, scheduled_at, priority, max_attempts
		) VALUES ($1, $2, $3, $4, $5)
		RETURNING id, job_type, payload, scheduled_at, priority, status,
		          started_at, completed_at, worker_id, locked_until,
		          attempt_count, max_attempts, last_error, result, created_at
	`, jobType, payloadJSON, scheduledAt, priority, maxAttempts).Scan(
		&job.ID, &job.JobType, &payloadJSON, &job.ScheduledAt, &job.Priority, &job.Status,
		&job.StartedAt, &job.CompletedAt, &job.WorkerID, &job.LockedUntil,
		&job.AttemptCount, &job.MaxAttempts, &job.LastError, &payloadJSON, &job.CreatedAt,
	)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to enqueue job", "error", err, "job_type", jobType)
		return nil, fmt.Errorf("enqueue job: %w", err)
	}

	if err := json.Unmarshal(payloadJSON, &job.Payload); err != nil {
		slog.ErrorContext(ctx, "Failed to unmarshal payload", "error", err)
	}

	slog.InfoContext(ctx, "Job enqueued", "job_id", job.ID, "job_type", jobType, "priority", priority)
	return &job, nil
}

// AcquireJob atomically acquires the next available job for a worker
// WORKAROUND: Using raw SQL instead of PL/pgSQL function due to pgx compatibility issue
func (s *BackgroundJobsService) AcquireJob(ctx context.Context, workerID string) (*BackgroundJob, error) {
	slog.InfoContext(ctx, "DEBUG: Attempting to acquire job", "worker_id", workerID)

	// Begin transaction for atomic operation
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to begin transaction", "error", err)
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Step 1: Find and lock next available job
	var jobID string
	err = tx.QueryRow(ctx, `
		SELECT id
		FROM background_jobs
		WHERE status = 'pending'
		  AND scheduled_at <= NOW()
		  AND (locked_until IS NULL OR locked_until < NOW())
		  AND attempt_count < max_attempts
		ORDER BY priority DESC, scheduled_at ASC
		LIMIT 1
		FOR UPDATE SKIP LOCKED
	`).Scan(&jobID)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			slog.InfoContext(ctx, "DEBUG: No job available", "worker_id", workerID)
			return nil, nil
		}
		slog.ErrorContext(ctx, "Failed to find job", "error", err, "worker_id", workerID)
		return nil, fmt.Errorf("find job: %w", err)
	}

	// Step 2: Update job with lock and status
	_, err = tx.Exec(ctx, `
		UPDATE background_jobs
		SET
			status = 'running',
			worker_id = $1,
			locked_until = NOW() + INTERVAL '300 seconds',
			started_at = CASE WHEN started_at IS NULL THEN NOW() ELSE started_at END,
			attempt_count = attempt_count + 1
		WHERE id = $2
	`, workerID, jobID)

	if err != nil {
		slog.ErrorContext(ctx, "Failed to update job", "error", err, "job_id", jobID)
		return nil, fmt.Errorf("update job: %w", err)
	}

	// Step 3: Fetch updated job details
	var job BackgroundJob
	var payloadJSON []byte

	err = tx.QueryRow(ctx, `
		SELECT id, job_type, payload, attempt_count, max_attempts
		FROM background_jobs
		WHERE id = $1
	`, jobID).Scan(
		&job.ID,
		&job.JobType,
		&payloadJSON,
		&job.AttemptCount,
		&job.MaxAttempts,
	)

	if err != nil {
		slog.ErrorContext(ctx, "Failed to fetch job details", "error", err, "job_id", jobID)
		return nil, fmt.Errorf("fetch job: %w", err)
	}

	// Step 4: Commit transaction
	if err := tx.Commit(ctx); err != nil {
		slog.ErrorContext(ctx, "Failed to commit transaction", "error", err, "job_id", jobID)
		return nil, fmt.Errorf("commit: %w", err)
	}

	if err := json.Unmarshal(payloadJSON, &job.Payload); err != nil {
		slog.ErrorContext(ctx, "Failed to unmarshal payload", "error", err)
		job.Payload = make(map[string]interface{})
	}

	job.Status = "running"
	job.WorkerID = &workerID

	slog.InfoContext(ctx, "Job acquired",
		"job_id", job.ID,
		"job_type", job.JobType,
		"worker_id", workerID,
		"attempt", job.AttemptCount,
	)

	return &job, nil
}

// CompleteJob marks a job as completed with result
func (s *BackgroundJobsService) CompleteJob(
	ctx context.Context,
	jobID uuid.UUID,
	result map[string]interface{},
) error {
	resultJSON, err := json.Marshal(result)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to marshal result", "error", err)
		return fmt.Errorf("marshal result: %w", err)
	}

	_, err = s.pool.Exec(ctx, `
		UPDATE background_jobs
		SET status = 'completed',
		    completed_at = NOW(),
		    result = $2,
		    locked_until = NULL
		WHERE id = $1
	`, jobID, resultJSON)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to complete job", "error", err, "job_id", jobID)
		return fmt.Errorf("complete job: %w", err)
	}

	slog.InfoContext(ctx, "Job completed", "job_id", jobID)
	return nil
}

// FailJob marks a job as failed and schedules retry if attempts remaining
func (s *BackgroundJobsService) FailJob(
	ctx context.Context,
	jobID uuid.UUID,
	errorMsg string,
) error {
	var attemptCount, maxAttempts int
	err := s.pool.QueryRow(ctx, `
		UPDATE background_jobs
		SET last_error = $2,
		    locked_until = NULL
		WHERE id = $1
		RETURNING attempt_count, max_attempts
	`, jobID, errorMsg).Scan(&attemptCount, &maxAttempts)

	if err != nil {
		slog.ErrorContext(ctx, "Failed to update job error", "error", err, "job_id", jobID)
		return fmt.Errorf("fail job: %w", err)
	}

	// Decide whether to retry or mark as failed
	if attemptCount >= maxAttempts {
		// No more retries, mark as failed
		_, err = s.pool.Exec(ctx, `
			UPDATE background_jobs
			SET status = 'failed',
			    completed_at = NOW()
			WHERE id = $1
		`, jobID)
		if err != nil {
			slog.ErrorContext(ctx, "Failed to mark job as failed", "error", err, "job_id", jobID)
			return fmt.Errorf("mark failed: %w", err)
		}
		slog.WarnContext(ctx, "Job failed permanently",
			"job_id", jobID,
			"attempts", attemptCount,
			"error", errorMsg,
		)
	} else {
		// Retry with exponential backoff
		_, err = s.pool.Exec(ctx, `
			UPDATE background_jobs
			SET status = 'pending',
			    scheduled_at = calculate_retry_time(attempt_count)
			WHERE id = $1
		`, jobID)
		if err != nil {
			slog.ErrorContext(ctx, "Failed to schedule retry", "error", err, "job_id", jobID)
			return fmt.Errorf("schedule retry: %w", err)
		}
		slog.InfoContext(ctx, "Job scheduled for retry",
			"job_id", jobID,
			"attempt", attemptCount,
			"max_attempts", maxAttempts,
			"error", errorMsg,
		)
	}

	return nil
}

// RetryJob manually re-enqueues a failed job
func (s *BackgroundJobsService) RetryJob(ctx context.Context, jobID uuid.UUID) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE background_jobs
		SET status = 'pending',
		    scheduled_at = NOW(),
		    attempt_count = 0,
		    last_error = NULL,
		    locked_until = NULL
		WHERE id = $1
	`, jobID)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to retry job", "error", err, "job_id", jobID)
		return fmt.Errorf("retry job: %w", err)
	}

	slog.InfoContext(ctx, "Job manually retried", "job_id", jobID)
	return nil
}

// CancelJob cancels a pending or running job
func (s *BackgroundJobsService) CancelJob(ctx context.Context, jobID uuid.UUID) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE background_jobs
		SET status = 'cancelled',
		    completed_at = NOW(),
		    locked_until = NULL
		WHERE id = $1 AND status IN ('pending', 'running')
	`, jobID)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to cancel job", "error", err, "job_id", jobID)
		return fmt.Errorf("cancel job: %w", err)
	}

	slog.InfoContext(ctx, "Job cancelled", "job_id", jobID)
	return nil
}

// GetJobStatus retrieves the current status of a job
func (s *BackgroundJobsService) GetJobStatus(ctx context.Context, jobID uuid.UUID) (*BackgroundJob, error) {
	var job BackgroundJob
	var payloadJSON, resultJSON []byte

	err := s.pool.QueryRow(ctx, `
		SELECT id, job_type, payload, scheduled_at, priority, status,
		       started_at, completed_at, worker_id, locked_until,
		       attempt_count, max_attempts, last_error, result, created_at
		FROM background_jobs
		WHERE id = $1
	`, jobID).Scan(
		&job.ID, &job.JobType, &payloadJSON, &job.ScheduledAt, &job.Priority, &job.Status,
		&job.StartedAt, &job.CompletedAt, &job.WorkerID, &job.LockedUntil,
		&job.AttemptCount, &job.MaxAttempts, &job.LastError, &resultJSON, &job.CreatedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("job not found")
	}
	if err != nil {
		slog.ErrorContext(ctx, "Failed to get job status", "error", err, "job_id", jobID)
		return nil, fmt.Errorf("get job: %w", err)
	}

	if len(payloadJSON) > 0 {
		json.Unmarshal(payloadJSON, &job.Payload)
	}
	if len(resultJSON) > 0 {
		json.Unmarshal(resultJSON, &job.Result)
	}

	return &job, nil
}

// ListJobs retrieves jobs with filters
func (s *BackgroundJobsService) ListJobs(ctx context.Context, filters JobListFilters) ([]BackgroundJob, error) {
	query := `
		SELECT id, job_type, payload, scheduled_at, priority, status,
		       started_at, completed_at, worker_id, locked_until,
		       attempt_count, max_attempts, last_error, result, created_at
		FROM background_jobs
		WHERE 1=1
	`
	args := []interface{}{}
	argPos := 1

	if filters.Status != nil {
		query += fmt.Sprintf(" AND status = $%d", argPos)
		args = append(args, *filters.Status)
		argPos++
	}

	if filters.JobType != nil {
		query += fmt.Sprintf(" AND job_type = $%d", argPos)
		args = append(args, *filters.JobType)
		argPos++
	}

	// Sorting
	if filters.SortBy == "" {
		filters.SortBy = "created_at"
	}
	if filters.SortOrder == "" {
		filters.SortOrder = "DESC"
	}
	query += fmt.Sprintf(" ORDER BY %s %s", filters.SortBy, filters.SortOrder)

	// Pagination
	if filters.Limit <= 0 {
		filters.Limit = 50
	}
	query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argPos, argPos+1)
	args = append(args, filters.Limit, filters.Offset)

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to list jobs", "error", err)
		return nil, fmt.Errorf("list jobs: %w", err)
	}
	defer rows.Close()

	var jobs []BackgroundJob
	for rows.Next() {
		var job BackgroundJob
		var payloadJSON, resultJSON []byte

		err := rows.Scan(
			&job.ID, &job.JobType, &payloadJSON, &job.ScheduledAt, &job.Priority, &job.Status,
			&job.StartedAt, &job.CompletedAt, &job.WorkerID, &job.LockedUntil,
			&job.AttemptCount, &job.MaxAttempts, &job.LastError, &resultJSON, &job.CreatedAt,
		)
		if err != nil {
			slog.ErrorContext(ctx, "Failed to scan job", "error", err)
			continue
		}

		if len(payloadJSON) > 0 {
			json.Unmarshal(payloadJSON, &job.Payload)
		}
		if len(resultJSON) > 0 {
			json.Unmarshal(resultJSON, &job.Result)
		}

		jobs = append(jobs, job)
	}

	return jobs, nil
}

// CleanupOldJobs removes completed/failed jobs older than specified duration
func (s *BackgroundJobsService) CleanupOldJobs(ctx context.Context, olderThan time.Duration) (int, error) {
	result, err := s.pool.Exec(ctx, `
		DELETE FROM background_jobs
		WHERE status IN ('completed', 'failed', 'cancelled')
		  AND created_at < NOW() - $1::INTERVAL
	`, olderThan.String())

	if err != nil {
		slog.ErrorContext(ctx, "Failed to cleanup old jobs", "error", err)
		return 0, fmt.Errorf("cleanup jobs: %w", err)
	}

	count := result.RowsAffected()
	slog.InfoContext(ctx, "Old jobs cleaned up", "count", count, "older_than", olderThan)
	return int(count), nil
}

// ReleaseStuckJobs releases jobs that have been locked too long
func (s *BackgroundJobsService) ReleaseStuckJobs(ctx context.Context) (int, error) {
	var count int
	err := s.pool.QueryRow(ctx, `SELECT release_stuck_jobs()`).Scan(&count)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to release stuck jobs", "error", err)
		return 0, fmt.Errorf("release stuck jobs: %w", err)
	}

	if count > 0 {
		slog.WarnContext(ctx, "Released stuck jobs", "count", count)
	}

	return count, nil
}
