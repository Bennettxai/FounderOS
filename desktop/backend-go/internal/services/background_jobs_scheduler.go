package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/robfig/cron/v3"
)

// ErrScheduledJobNotFound indicates that the requested scheduled job was not found
var ErrScheduledJobNotFound = errors.New("scheduled job not found")

// =====================================================================
// TYPES
// =====================================================================

// JobScheduler manages recurring jobs with cron-like scheduling
type JobScheduler struct {
	pool           *pgxpool.Pool
	service        *BackgroundJobsService
	cronParser     cron.Parser
	checkInterval  time.Duration
	stopChan       chan struct{}
	wg             sync.WaitGroup
	running        bool
	runningMu      sync.Mutex
}

// CreateScheduledJobRequest represents a request to create a scheduled job
type CreateScheduledJobRequest struct {
	JobType        string                 `json:"job_type"`
	Payload        map[string]interface{} `json:"payload"`
	CronExpression string                 `json:"cron_expression"`
	Timezone       string                 `json:"timezone,omitempty"` // Default: UTC
	Name           *string                `json:"name,omitempty"`
	Description    *string                `json:"description,omitempty"`
}

// UpdateScheduledJobRequest represents a request to update a scheduled job
type UpdateScheduledJobRequest struct {
	Payload        *map[string]interface{} `json:"payload,omitempty"`
	CronExpression *string                 `json:"cron_expression,omitempty"`
	Timezone       *string                 `json:"timezone,omitempty"`
	Name           *string                 `json:"name,omitempty"`
	Description    *string                 `json:"description,omitempty"`
}

// =====================================================================
// CONSTRUCTOR
// =====================================================================

// NewJobScheduler creates a new job scheduler
func NewJobScheduler(pool *pgxpool.Pool, service *BackgroundJobsService) *JobScheduler {
	return &JobScheduler{
		pool:          pool,
		service:       service,
		cronParser:    cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor),
		checkInterval: 1 * time.Minute, // Check every minute
		stopChan:      make(chan struct{}),
		running:       false,
	}
}

// =====================================================================
// SCHEDULED JOB MANAGEMENT
// =====================================================================

// CreateScheduledJob creates a new recurring job
func (s *JobScheduler) CreateScheduledJob(ctx context.Context, req CreateScheduledJobRequest) (*ScheduledJob, error) {
	// Validate cron expression
	if _, err := s.ParseCronExpression(req.CronExpression); err != nil {
		return nil, fmt.Errorf("invalid cron expression: %w", err)
	}

	// Set default timezone
	if req.Timezone == "" {
		req.Timezone = "UTC"
	}

	// Calculate next run time
	nextRun, err := s.CalculateNextRun(req.CronExpression, req.Timezone)
	if err != nil {
		return nil, fmt.Errorf("calculate next run: %w", err)
	}

	// Marshal payload
	payloadJSON, err := json.Marshal(req.Payload)
	if err != nil {
		return nil, fmt.Errorf("marshal payload: %w", err)
	}

	// Insert scheduled job
	var job ScheduledJob
	err = s.pool.QueryRow(ctx, `
		INSERT INTO scheduled_jobs (
			job_type, payload, cron_expression, timezone, next_run_at, name, description
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, job_type, payload, cron_expression, timezone, is_active,
		          last_run_at, next_run_at, name, description, created_at, updated_at
	`, req.JobType, payloadJSON, req.CronExpression, req.Timezone, nextRun, req.Name, req.Description).Scan(
		&job.ID, &job.JobType, &payloadJSON, &job.CronExpression, &job.Timezone, &job.IsActive,
		&job.LastRunAt, &job.NextRunAt, &job.Name, &job.Description, &job.CreatedAt, &job.UpdatedAt,
	)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to create scheduled job", "error", err)
		return nil, fmt.Errorf("create scheduled job: %w", err)
	}

	json.Unmarshal(payloadJSON, &job.Payload)

	slog.InfoContext(ctx, "Scheduled job created",
		"job_id", job.ID,
		"job_type", job.JobType,
		"cron", job.CronExpression,
		"next_run", job.NextRunAt,
	)

	return &job, nil
}

// GetScheduledJob retrieves a scheduled job by ID
func (s *JobScheduler) GetScheduledJob(ctx context.Context, jobID uuid.UUID) (*ScheduledJob, error) {
	var job ScheduledJob
	var payloadJSON []byte

	err := s.pool.QueryRow(ctx, `
		SELECT id, job_type, payload, cron_expression, timezone, is_active,
		       last_run_at, next_run_at, name, description, created_at, updated_at
		FROM scheduled_jobs
		WHERE id = $1
	`, jobID).Scan(
		&job.ID, &job.JobType, &payloadJSON, &job.CronExpression, &job.Timezone, &job.IsActive,
		&job.LastRunAt, &job.NextRunAt, &job.Name, &job.Description, &job.CreatedAt, &job.UpdatedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, ErrScheduledJobNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get scheduled job: %w", err)
	}

	json.Unmarshal(payloadJSON, &job.Payload)
	return &job, nil
}

// ListScheduledJobs retrieves all scheduled jobs
func (s *JobScheduler) ListScheduledJobs(ctx context.Context, activeOnly bool) ([]ScheduledJob, error) {
	query := `
		SELECT id, job_type, payload, cron_expression, timezone, is_active,
		       last_run_at, next_run_at, name, description, created_at, updated_at
		FROM scheduled_jobs
	`
	if activeOnly {
		query += " WHERE is_active = TRUE"
	}
	query += " ORDER BY next_run_at ASC"

	rows, err := s.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list scheduled jobs: %w", err)
	}
	defer rows.Close()

	var jobs []ScheduledJob
	for rows.Next() {
		var job ScheduledJob
		var payloadJSON []byte

		err := rows.Scan(
			&job.ID, &job.JobType, &payloadJSON, &job.CronExpression, &job.Timezone, &job.IsActive,
			&job.LastRunAt, &job.NextRunAt, &job.Name, &job.Description, &job.CreatedAt, &job.UpdatedAt,
		)
		if err != nil {
			slog.ErrorContext(ctx, "Failed to scan scheduled job", "error", err)
			continue
		}

		json.Unmarshal(payloadJSON, &job.Payload)
		jobs = append(jobs, job)
	}

	return jobs, nil
}

// UpdateScheduledJob updates a scheduled job
func (s *JobScheduler) UpdateScheduledJob(ctx context.Context, jobID uuid.UUID, req UpdateScheduledJobRequest) (*ScheduledJob, error) {
	// Build dynamic update query
	updates := []string{}
	args := []interface{}{}
	argPos := 1

	if req.Payload != nil {
		payloadJSON, err := json.Marshal(*req.Payload)
		if err != nil {
			return nil, fmt.Errorf("marshal payload: %w", err)
		}
		updates = append(updates, fmt.Sprintf("payload = $%d", argPos))
		args = append(args, payloadJSON)
		argPos++
	}

	if req.CronExpression != nil {
		// Validate cron expression
		if _, err := s.ParseCronExpression(*req.CronExpression); err != nil {
			return nil, fmt.Errorf("invalid cron expression: %w", err)
		}
		updates = append(updates, fmt.Sprintf("cron_expression = $%d", argPos))
		args = append(args, *req.CronExpression)
		argPos++
	}

	if req.Timezone != nil {
		updates = append(updates, fmt.Sprintf("timezone = $%d", argPos))
		args = append(args, *req.Timezone)
		argPos++
	}

	if req.Name != nil {
		updates = append(updates, fmt.Sprintf("name = $%d", argPos))
		args = append(args, *req.Name)
		argPos++
	}

	if req.Description != nil {
		updates = append(updates, fmt.Sprintf("description = $%d", argPos))
		args = append(args, *req.Description)
		argPos++
	}

	if len(updates) == 0 {
		return s.GetScheduledJob(ctx, jobID)
	}

	updates = append(updates, "updated_at = NOW()")
	args = append(args, jobID)

	query := fmt.Sprintf(`
		UPDATE scheduled_jobs
		SET %s
		WHERE id = $%d
		RETURNING id, job_type, payload, cron_expression, timezone, is_active,
		          last_run_at, next_run_at, name, description, created_at, updated_at
	`, join(updates, ", "), argPos)

	var job ScheduledJob
	var payloadJSON []byte

	err := s.pool.QueryRow(ctx, query, args...).Scan(
		&job.ID, &job.JobType, &payloadJSON, &job.CronExpression, &job.Timezone, &job.IsActive,
		&job.LastRunAt, &job.NextRunAt, &job.Name, &job.Description, &job.CreatedAt, &job.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("update scheduled job: %w", err)
	}

	json.Unmarshal(payloadJSON, &job.Payload)

	// Recalculate next run if cron or timezone changed
	if req.CronExpression != nil || req.Timezone != nil {
		nextRun, _ := s.CalculateNextRun(job.CronExpression, job.Timezone)
		job.NextRunAt = &nextRun
		s.pool.Exec(ctx, `UPDATE scheduled_jobs SET next_run_at = $1 WHERE id = $2`, nextRun, jobID)
	}

	slog.InfoContext(ctx, "Scheduled job updated", "job_id", jobID)
	return &job, nil
}

// DeleteScheduledJob deletes a scheduled job
func (s *JobScheduler) DeleteScheduledJob(ctx context.Context, jobID uuid.UUID) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM scheduled_jobs WHERE id = $1`, jobID)
	if err != nil {
		return fmt.Errorf("delete scheduled job: %w", err)
	}

	slog.InfoContext(ctx, "Scheduled job deleted", "job_id", jobID)
	return nil
}

// EnableScheduledJob enables a scheduled job
func (s *JobScheduler) EnableScheduledJob(ctx context.Context, jobID uuid.UUID) error {
	_, err := s.pool.Exec(ctx, `UPDATE scheduled_jobs SET is_active = TRUE WHERE id = $1`, jobID)
	if err != nil {
		return fmt.Errorf("enable scheduled job: %w", err)
	}

	slog.InfoContext(ctx, "Scheduled job enabled", "job_id", jobID)
	return nil
}

// DisableScheduledJob disables a scheduled job
func (s *JobScheduler) DisableScheduledJob(ctx context.Context, jobID uuid.UUID) error {
	_, err := s.pool.Exec(ctx, `UPDATE scheduled_jobs SET is_active = FALSE WHERE id = $1`, jobID)
	if err != nil {
		return fmt.Errorf("disable scheduled job: %w", err)
	}

	slog.InfoContext(ctx, "Scheduled job disabled", "job_id", jobID)
	return nil
}

// =====================================================================
// SCHEDULER LIFECYCLE
// =====================================================================

// Start begins the scheduler loop
func (s *JobScheduler) Start(ctx context.Context) error {
	s.runningMu.Lock()
	if s.running {
		s.runningMu.Unlock()
		return fmt.Errorf("scheduler already running")
	}
	s.running = true
	s.runningMu.Unlock()

	slog.InfoContext(ctx, "Scheduler starting", "check_interval", s.checkInterval)

	s.wg.Add(1)
	go s.schedulerLoop(ctx)

	return nil
}

// Stop gracefully stops the scheduler
func (s *JobScheduler) Stop() error {
	s.runningMu.Lock()
	if !s.running {
		s.runningMu.Unlock()
		return fmt.Errorf("scheduler not running")
	}
	s.runningMu.Unlock()

	slog.Info("Scheduler stopping")

	close(s.stopChan)
	s.wg.Wait()

	s.runningMu.Lock()
	s.running = false
	s.runningMu.Unlock()

	slog.Info("Scheduler stopped")
	return nil
}

// =====================================================================
// SCHEDULER LOOP
// =====================================================================

// schedulerLoop is the main scheduler processing loop
func (s *JobScheduler) schedulerLoop(ctx context.Context) {
	defer s.wg.Done()

	ticker := time.NewTicker(s.checkInterval)
	defer ticker.Stop()

	slog.InfoContext(ctx, "Scheduler loop started")

	for {
		select {
		case <-s.stopChan:
			slog.InfoContext(ctx, "Scheduler loop stopped")
			return

		case <-ticker.C:
			if err := s.processDueJobs(ctx); err != nil {
				slog.ErrorContext(ctx, "Error processing due jobs", "error", err)
			}

		case <-ctx.Done():
			slog.InfoContext(ctx, "Scheduler loop context cancelled")
			return
		}
	}
}

// processDueJobs finds and processes scheduled jobs that are due to run
func (s *JobScheduler) processDueJobs(ctx context.Context) error {
	// Find all active scheduled jobs that are due
	rows, err := s.pool.Query(ctx, `
		SELECT id, job_type, payload, cron_expression, timezone
		FROM scheduled_jobs
		WHERE is_active = TRUE
		  AND (next_run_at IS NULL OR next_run_at <= NOW())
	`)
	if err != nil {
		return fmt.Errorf("query due jobs: %w", err)
	}
	defer rows.Close()

	jobsProcessed := 0

	for rows.Next() {
		var jobID uuid.UUID
		var jobType, cronExpr, timezone string
		var payloadJSON []byte

		if err := rows.Scan(&jobID, &jobType, &payloadJSON, &cronExpr, &timezone); err != nil {
			slog.ErrorContext(ctx, "Failed to scan scheduled job", "error", err)
			continue
		}

		// Process this scheduled job
		if err := s.processScheduledJob(ctx, jobID, jobType, payloadJSON, cronExpr, timezone); err != nil {
			slog.ErrorContext(ctx, "Failed to process scheduled job",
				"job_id", jobID,
				"error", err,
			)
			continue
		}

		jobsProcessed++
	}

	if jobsProcessed > 0 {
		slog.InfoContext(ctx, "Processed scheduled jobs", "count", jobsProcessed)
	}

	return nil
}

// processScheduledJob creates a background job from a scheduled job and updates next run time
func (s *JobScheduler) processScheduledJob(
	ctx context.Context,
	schedJobID uuid.UUID,
	jobType string,
	payloadJSON []byte,
	cronExpr string,
	timezone string,
) error {
	// Parse payload
	var payload map[string]interface{}
	if err := json.Unmarshal(payloadJSON, &payload); err != nil {
		return fmt.Errorf("unmarshal payload: %w", err)
	}

	// Create background job
	_, err := s.service.EnqueueJob(ctx, jobType, payload, 0, 3, nil)
	if err != nil {
		return fmt.Errorf("enqueue job: %w", err)
	}

	// Calculate next run time
	nextRun, err := s.CalculateNextRun(cronExpr, timezone)
	if err != nil {
		return fmt.Errorf("calculate next run: %w", err)
	}

	// Update scheduled job
	_, err = s.pool.Exec(ctx, `
		UPDATE scheduled_jobs
		SET last_run_at = NOW(),
		    next_run_at = $1,
		    updated_at = NOW()
		WHERE id = $2
	`, nextRun, schedJobID)
	if err != nil {
		return fmt.Errorf("update scheduled job: %w", err)
	}

	slog.InfoContext(ctx, "Scheduled job triggered",
		"scheduled_job_id", schedJobID,
		"job_type", jobType,
		"next_run", nextRun,
	)

	return nil
}

// =====================================================================
// CRON UTILITIES
// =====================================================================

// ParseCronExpression validates and parses a cron expression
func (s *JobScheduler) ParseCronExpression(expr string) (cron.Schedule, error) {
	schedule, err := s.cronParser.Parse(expr)
	if err != nil {
		return nil, fmt.Errorf("parse cron expression %q: %w", expr, err)
	}
	return schedule, nil
}

// CalculateNextRun calculates the next run time for a cron expression
func (s *JobScheduler) CalculateNextRun(cronExpr string, timezone string) (time.Time, error) {
	// Parse cron expression
	schedule, err := s.ParseCronExpression(cronExpr)
	if err != nil {
		return time.Time{}, err
	}

	// Load timezone
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		slog.Warn("Invalid timezone, using UTC", "timezone", timezone, "error", err)
		loc = time.UTC
	}

	// Calculate next run time
	now := time.Now().In(loc)
	next := schedule.Next(now)

	return next, nil
}

// =====================================================================
// HELPERS
// =====================================================================

func join(parts []string, sep string) string {
	if len(parts) == 0 {
		return ""
	}
	result := parts[0]
	for i := 1; i < len(parts); i++ {
		result += sep + parts[i]
	}
	return result
}
