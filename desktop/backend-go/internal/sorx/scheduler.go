// Package sorx implements the Sorx skill execution engine.
// This file implements the proactive scheduler — it runs skills on cron schedules
// so the system acts without being asked.
package sorx

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/robfig/cron/v3"
)

// ============================================================================
// Schedule Types
// ============================================================================

// Schedule represents a cron-based schedule for a SORX skill.
type Schedule struct {
	ID          uuid.UUID      `json:"id"`
	SkillID     string         `json:"skill_id"`
	UserID      string         `json:"user_id"`
	WorkspaceID string         `json:"workspace_id,omitempty"`
	CronExpr    string         `json:"cron_expr"`
	Params      map[string]any `json:"params,omitempty"`
	Enabled     bool           `json:"enabled"`
	LastRunAt   *time.Time     `json:"last_run_at,omitempty"`
	LastStatus  string         `json:"last_status,omitempty"` // "complete" or "failed"
	LastError   string         `json:"last_error,omitempty"`
	NextRunAt   *time.Time     `json:"next_run_at,omitempty"`
	RunCount    int            `json:"run_count"`
	FailCount   int            `json:"fail_count"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
}

// scheduledEntry couples a Schedule to its cron entry ID.
type scheduledEntry struct {
	schedule *Schedule
	entryID  cron.EntryID
	running  bool
	mu       sync.Mutex
}

// ============================================================================
// Scheduler
// ============================================================================

// Scheduler runs SORX skills on configurable cron schedules.
// Each user owns their own set of schedules. The scheduler is the proactive
// layer of the system — it executes skills without waiting for user input.
type Scheduler struct {
	engine    *Engine
	cron      *cron.Cron
	pool      *pgxpool.Pool
	logger    *slog.Logger
	schedules sync.Map // scheduleID (string) → *scheduledEntry
	done      chan struct{}
}

// NewScheduler creates a new Scheduler backed by the given Engine and database pool.
func NewScheduler(engine *Engine, pool *pgxpool.Pool, logger *slog.Logger) *Scheduler {
	c := cron.New(cron.WithSeconds()) // second-resolution cron
	return &Scheduler{
		engine: engine,
		cron:   c,
		pool:   pool,
		logger: logger,
		done:   make(chan struct{}),
	}
}

// Start loads all enabled schedules from the database and registers their cron
// jobs. It then starts the cron runner. Call Stop to shut down cleanly.
func (s *Scheduler) Start() error {
	ctx := context.Background()

	rows, err := s.pool.Query(ctx, `
		SELECT id, skill_id, user_id, workspace_id, cron_expr, params,
		       enabled, last_run_at, last_status, last_error, next_run_at,
		       run_count, fail_count, created_at, updated_at
		FROM sorx_schedules
		WHERE enabled = true
	`)
	if err != nil {
		return fmt.Errorf("scheduler: failed to load schedules: %w", err)
	}
	defer rows.Close()

	var loaded int
	for rows.Next() {
		sc, err := scanSchedule(rows)
		if err != nil {
			s.logger.Error("scheduler: failed to scan schedule row", "error", err)
			continue
		}

		if err := s.registerCronJob(sc); err != nil {
			s.logger.Error("scheduler: failed to register cron job",
				"schedule_id", sc.ID,
				"skill_id", sc.SkillID,
				"error", err)
			continue
		}
		loaded++
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("scheduler: row iteration error: %w", err)
	}

	s.cron.Start()
	s.logger.Info("scheduler: started", "loaded_schedules", loaded)
	return nil
}

// Stop halts the cron runner and waits for any in-flight jobs to finish.
func (s *Scheduler) Stop() {
	ctx := s.cron.Stop()
	<-ctx.Done()
	close(s.done)
	s.logger.Info("scheduler: stopped")
}

// ============================================================================
// Schedule Management
// ============================================================================

// AddSchedule persists a new schedule and immediately registers it if enabled.
// The schedule's ID and timestamps are set here; callers do not need to set them.
func (s *Scheduler) AddSchedule(ctx context.Context, sc Schedule) error {
	// Validate the skill exists in the registry.
	if _, ok := s.engine.skills.Load(sc.SkillID); !ok {
		return fmt.Errorf("scheduler: unknown skill %q — register it before scheduling", sc.SkillID)
	}

	// Validate the cron expression before touching the database.
	parser := cron.NewParser(cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	cronSched, err := parser.Parse(sc.CronExpr)
	if err != nil {
		return fmt.Errorf("scheduler: invalid cron expression %q: %w", sc.CronExpr, err)
	}

	sc.ID = uuid.New()
	now := time.Now().UTC()
	sc.CreatedAt = now
	sc.UpdatedAt = now
	next := cronSched.Next(now)
	sc.NextRunAt = &next

	paramsJSON, err := json.Marshal(sc.Params)
	if err != nil {
		return fmt.Errorf("scheduler: failed to marshal params: %w", err)
	}

	_, err = s.pool.Exec(ctx, `
		INSERT INTO sorx_schedules (
			id, skill_id, user_id, workspace_id, cron_expr, params,
			enabled, next_run_at, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`, sc.ID, sc.SkillID, sc.UserID, nullableString(sc.WorkspaceID),
		sc.CronExpr, paramsJSON, sc.Enabled, sc.NextRunAt, sc.CreatedAt, sc.UpdatedAt)
	if err != nil {
		return fmt.Errorf("scheduler: failed to persist schedule: %w", err)
	}

	if sc.Enabled {
		if err := s.registerCronJob(&sc); err != nil {
			return fmt.Errorf("scheduler: persisted but failed to register cron job: %w", err)
		}
	}

	s.logger.Info("scheduler: schedule added",
		"schedule_id", sc.ID,
		"skill_id", sc.SkillID,
		"user_id", sc.UserID,
		"cron_expr", sc.CronExpr,
		"enabled", sc.Enabled)
	return nil
}

// RemoveSchedule cancels and removes a schedule by ID.
func (s *Scheduler) RemoveSchedule(ctx context.Context, id uuid.UUID) error {
	s.deregisterCronJob(id)

	_, err := s.pool.Exec(ctx, `DELETE FROM sorx_schedules WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("scheduler: failed to delete schedule %s: %w", id, err)
	}

	s.logger.Info("scheduler: schedule removed", "schedule_id", id)
	return nil
}

// UpdateSchedule replaces an existing schedule's configuration.
// If the schedule is enabled the new cron job is immediately registered;
// if disabled any existing job is cancelled.
func (s *Scheduler) UpdateSchedule(ctx context.Context, sc Schedule) error {
	if _, ok := s.engine.skills.Load(sc.SkillID); !ok {
		return fmt.Errorf("scheduler: unknown skill %q", sc.SkillID)
	}

	parser := cron.NewParser(cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	cronSched, err := parser.Parse(sc.CronExpr)
	if err != nil {
		return fmt.Errorf("scheduler: invalid cron expression %q: %w", sc.CronExpr, err)
	}

	now := time.Now().UTC()
	sc.UpdatedAt = now
	next := cronSched.Next(now)
	sc.NextRunAt = &next

	paramsJSON, err := json.Marshal(sc.Params)
	if err != nil {
		return fmt.Errorf("scheduler: failed to marshal params: %w", err)
	}

	tag, err := s.pool.Exec(ctx, `
		UPDATE sorx_schedules SET
			skill_id     = $2,
			cron_expr    = $3,
			params       = $4,
			enabled      = $5,
			workspace_id = $6,
			next_run_at  = $7,
			updated_at   = $8
		WHERE id = $1
	`, sc.ID, sc.SkillID, sc.CronExpr, paramsJSON, sc.Enabled,
		nullableString(sc.WorkspaceID), sc.NextRunAt, sc.UpdatedAt)
	if err != nil {
		return fmt.Errorf("scheduler: failed to update schedule: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("scheduler: schedule %s not found", sc.ID)
	}

	// Re-register: cancel old job then add new one if enabled.
	s.deregisterCronJob(sc.ID)
	if sc.Enabled {
		if err := s.registerCronJob(&sc); err != nil {
			return fmt.Errorf("scheduler: updated but failed to register cron job: %w", err)
		}
	}

	s.logger.Info("scheduler: schedule updated",
		"schedule_id", sc.ID,
		"skill_id", sc.SkillID,
		"enabled", sc.Enabled)
	return nil
}

// ListSchedules returns all schedules belonging to a user.
func (s *Scheduler) ListSchedules(ctx context.Context, userID string) ([]Schedule, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, skill_id, user_id, workspace_id, cron_expr, params,
		       enabled, last_run_at, last_status, last_error, next_run_at,
		       run_count, fail_count, created_at, updated_at
		FROM sorx_schedules
		WHERE user_id = $1
		ORDER BY created_at DESC
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("scheduler: failed to list schedules: %w", err)
	}
	defer rows.Close()

	var out []Schedule
	for rows.Next() {
		sc, err := scanSchedule(rows)
		if err != nil {
			return nil, fmt.Errorf("scheduler: failed to scan schedule: %w", err)
		}
		out = append(out, *sc)
	}
	return out, rows.Err()
}

// ProvisionDefaultSchedules inserts the default schedules for a new user.
// All defaults are inserted in disabled state so users explicitly opt in.
// Schedules that already exist (unique constraint) are silently skipped.
func (s *Scheduler) ProvisionDefaultSchedules(ctx context.Context, userID string) error {
	for _, tmpl := range DefaultSchedules {
		sc := tmpl // copy
		sc.UserID = userID

		if err := s.AddSchedule(ctx, sc); err != nil {
			// A duplicate (user already has this skill scheduled) is fine — skip it.
			s.logger.Debug("scheduler: skipped default schedule",
				"skill_id", sc.SkillID,
				"user_id", userID,
				"reason", err)
		}
	}
	return nil
}

// ============================================================================
// Internal — Cron Registration
// ============================================================================

// registerCronJob adds a cron entry for the given schedule and stores the entry
// in the in-memory map. The schedule pointer is captured in the closure.
func (s *Scheduler) registerCronJob(sc *Schedule) error {
	// Cron library uses standard 5-field expressions by default; we opened with
	// WithSeconds so 6-field (second-first) expressions also work.
	id, err := s.cron.AddFunc(sc.CronExpr, func() {
		s.runScheduledSkill(sc)
	})
	if err != nil {
		return fmt.Errorf("invalid cron expression %q: %w", sc.CronExpr, err)
	}

	s.schedules.Store(sc.ID.String(), &scheduledEntry{
		schedule: sc,
		entryID:  id,
	})
	return nil
}

// deregisterCronJob removes the cron entry and in-memory record for a schedule.
func (s *Scheduler) deregisterCronJob(id uuid.UUID) {
	key := id.String()
	val, ok := s.schedules.LoadAndDelete(key)
	if !ok {
		return
	}
	entry := val.(*scheduledEntry)
	s.cron.Remove(entry.entryID)
}

// ============================================================================
// Internal — Execution
// ============================================================================

// runScheduledSkill is the function invoked by cron for each scheduled skill.
// It applies all safety checks, executes via the SORX engine, and persists the
// outcome back to the database.
func (s *Scheduler) runScheduledSkill(schedule *Schedule) {
	// Retrieve the live entry to access the running flag.
	val, ok := s.schedules.Load(schedule.ID.String())
	if !ok {
		// Schedule was removed between registration and firing.
		return
	}
	entry := val.(*scheduledEntry)

	// Overlap prevention: if the previous run is still in progress, skip.
	entry.mu.Lock()
	if entry.running {
		entry.mu.Unlock()
		s.logger.Warn("scheduler: skipping run — previous execution still in progress",
			"schedule_id", schedule.ID,
			"skill_id", schedule.SkillID,
			"user_id", schedule.UserID)
		return
	}
	entry.running = true
	entry.mu.Unlock()

	defer func() {
		entry.mu.Lock()
		entry.running = false
		entry.mu.Unlock()
	}()

	// Use a fresh background context so a cancelled request context never
	// interrupts a scheduled execution mid-flight.
	ctx := context.Background()

	s.logger.Info("scheduler: executing skill",
		"schedule_id", schedule.ID,
		"skill_id", schedule.SkillID,
		"user_id", schedule.UserID)

	// Verify the skill still exists in the registry.
	if _, exists := s.engine.skills.Load(schedule.SkillID); !exists {
		s.logger.Warn("scheduler: skill no longer registered — skipping",
			"schedule_id", schedule.ID,
			"skill_id", schedule.SkillID)
		s.persistScheduleOutcome(ctx, schedule, StatusFailed,
			fmt.Sprintf("skill %q is no longer registered", schedule.SkillID))
		return
	}

	// Check required integrations. If any are missing we skip rather than fail —
	// the user may not have connected the integration yet.
	if missing := s.checkRequiredIntegrations(ctx, schedule); len(missing) > 0 {
		s.logger.Info("scheduler: skipping — required integrations not connected",
			"schedule_id", schedule.ID,
			"skill_id", schedule.SkillID,
			"missing", missing)
		// Not an error from the scheduler's perspective; don't increment fail_count.
		return
	}

	// Build params: merge schedule params with scheduled-run marker.
	params := make(map[string]any, len(schedule.Params)+1)
	for k, v := range schedule.Params {
		params[k] = v
	}
	params["_scheduled"] = true
	params["_schedule_id"] = schedule.ID.String()

	_, execErr := s.engine.ExecuteSkill(ctx, ExecuteRequest{
		SkillID:     schedule.SkillID,
		UserID:      schedule.UserID,
		Params:      params,
		Temperature: TemperatureWarm,
	})

	if execErr != nil {
		s.logger.Error("scheduler: skill execution failed",
			"schedule_id", schedule.ID,
			"skill_id", schedule.SkillID,
			"user_id", schedule.UserID,
			"error", execErr)
		s.persistScheduleOutcome(ctx, schedule, StatusFailed, execErr.Error())
		return
	}

	s.persistScheduleOutcome(ctx, schedule, StatusComplete, "")
}

// checkRequiredIntegrations returns the names of integrations the skill requires
// but the user has not yet connected. An empty slice means all good.
func (s *Scheduler) checkRequiredIntegrations(ctx context.Context, schedule *Schedule) []string {
	val, ok := s.engine.skills.Load(schedule.SkillID)
	if !ok {
		return nil
	}
	skill := val.(*SkillDefinition)

	var missing []string
	for _, provider := range skill.RequiredIntegrations {
		ok, err := s.engine.checkIntegrationAccess(ctx, schedule.UserID, provider)
		if err != nil {
			s.logger.Warn("scheduler: integration check error",
				"provider", provider,
				"user_id", schedule.UserID,
				"error", err)
			missing = append(missing, provider)
			continue
		}
		if !ok {
			missing = append(missing, provider)
		}
	}
	return missing
}

// persistScheduleOutcome writes last_run_at, last_status, last_error, run_count,
// and fail_count back to the database and recalculates next_run_at.
func (s *Scheduler) persistScheduleOutcome(ctx context.Context, schedule *Schedule, status, errMsg string) {
	now := time.Now().UTC()

	// Compute next run time from the cron expression.
	var nextRun *time.Time
	parser := cron.NewParser(cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	if cs, err := parser.Parse(schedule.CronExpr); err == nil {
		t := cs.Next(now)
		nextRun = &t
	}

	runCountDelta := 1
	failCountDelta := 0
	if status == StatusFailed {
		failCountDelta = 1
	}

	_, err := s.pool.Exec(ctx, `
		UPDATE sorx_schedules SET
			last_run_at  = $2,
			last_status  = $3,
			last_error   = $4,
			next_run_at  = $5,
			run_count    = run_count  + $6,
			fail_count   = fail_count + $7,
			updated_at   = $2
		WHERE id = $1
	`, schedule.ID, now, status, nullableString(errMsg), nextRun,
		runCountDelta, failCountDelta)
	if err != nil {
		s.logger.Error("scheduler: failed to persist schedule outcome",
			"schedule_id", schedule.ID,
			"error", err)
	}

	// Keep the in-memory copy current so ListSchedules is accurate.
	schedule.LastRunAt = &now
	schedule.LastStatus = status
	schedule.LastError = errMsg
	schedule.NextRunAt = nextRun
	schedule.RunCount += runCountDelta
	schedule.FailCount += failCountDelta
}

// ============================================================================
// Helpers
// ============================================================================

// scanSchedule reads one row from a sorx_schedules query result.
func scanSchedule(rows pgx.Rows) (*Schedule, error) {
	var sc Schedule
	var workspaceID *string
	var paramsJSON []byte
	var lastStatus, lastError *string

	err := rows.Scan(
		&sc.ID, &sc.SkillID, &sc.UserID, &workspaceID, &sc.CronExpr, &paramsJSON,
		&sc.Enabled, &sc.LastRunAt, &lastStatus, &lastError, &sc.NextRunAt,
		&sc.RunCount, &sc.FailCount, &sc.CreatedAt, &sc.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	if workspaceID != nil {
		sc.WorkspaceID = *workspaceID
	}
	if lastStatus != nil {
		sc.LastStatus = *lastStatus
	}
	if lastError != nil {
		sc.LastError = *lastError
	}
	if len(paramsJSON) > 0 {
		if err := json.Unmarshal(paramsJSON, &sc.Params); err != nil {
			return nil, fmt.Errorf("failed to unmarshal schedule params: %w", err)
		}
	}

	return &sc, nil
}

// nullableString converts an empty string to nil so Postgres gets a proper NULL.
func nullableString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
