package services

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// UsageMeteringService tracks AI calls, storage, and compute usage per workspace
// to enable plan enforcement and billing.
type UsageMeteringService struct {
	pool   *pgxpool.Pool
	logger *slog.Logger
}

// NewUsageMeteringService constructs a UsageMeteringService.
func NewUsageMeteringService(pool *pgxpool.Pool) *UsageMeteringService {
	return &UsageMeteringService{
		pool:   pool,
		logger: slog.Default().With("component", "usage_metering"),
	}
}

// UsageSummary holds the current-period usage for a workspace.
type UsageSummary struct {
	AICallsToday     int    `json:"ai_calls_today"`
	AICallsLimit     int    `json:"ai_calls_limit"`
	StorageUsed      int64  `json:"storage_used_bytes"`
	StorageLimit     int64  `json:"storage_limit_bytes"`
	ModulesCount     int    `json:"modules_count"`
	ModulesLimit     int    `json:"modules_limit"`
	TeamMembers      int    `json:"team_members"`
	TeamMembersLimit int    `json:"team_members_limit"`
	Plan             string `json:"plan"`
}

// PlanLimits holds the configured limits for a given plan tier.
type PlanLimits struct {
	PlanName                string   `json:"plan_name"`
	AICallsPerDay           int      `json:"ai_calls_per_day"`
	AIModelTier             string   `json:"ai_model_tier"`
	MaxModules              int      `json:"max_modules"`
	StorageBytesLimit       int64    `json:"storage_bytes_limit"`
	MaxTeamMembers          int      `json:"max_team_members"`
	OSAModes                []string `json:"osa_modes"`
	ComputeCPUHoursPerMonth float64  `json:"compute_cpu_hours_per_month"`
}

// UsageHistoryEntry is a single row from usage_meters used for history responses.
type UsageHistoryEntry struct {
	MeterType   string    `json:"meter_type"`
	Quantity    int64     `json:"quantity"`
	PeriodStart time.Time `json:"period_start"`
	PeriodEnd   time.Time `json:"period_end"`
}

// RecordAICall increments the AI call counter for the workspace in the current daily period.
// It upserts a row in usage_meters so concurrent increments are race-safe.
func (s *UsageMeteringService) RecordAICall(ctx context.Context, workspaceID uuid.UUID) error {
	periodStart, periodEnd := dailyPeriod(time.Now().UTC())

	const q = `
		INSERT INTO usage_meters (workspace_id, meter_type, quantity, period_start, period_end)
		VALUES ($1, 'ai_calls', 1, $2, $3)
		ON CONFLICT (workspace_id, meter_type, period_start)
		DO UPDATE SET
			quantity   = usage_meters.quantity + 1,
			updated_at = NOW()
	`
	_, err := s.pool.Exec(ctx, q, workspaceID, periodStart, periodEnd)
	if err != nil {
		return fmt.Errorf("usage_metering: record ai call: %w", err)
	}
	return nil
}

// CheckAILimit reports whether a workspace may make another AI call today.
// allowed is false when the workspace has reached its daily limit.
// remaining is -1 for unlimited (enterprise) plans.
func (s *UsageMeteringService) CheckAILimit(ctx context.Context, workspaceID uuid.UUID) (allowed bool, remaining int, err error) {
	limits, err := s.GetPlanLimits(ctx, workspaceID)
	if err != nil {
		return false, 0, err
	}

	// -1 means unlimited (enterprise tier).
	if limits.AICallsPerDay == -1 {
		return true, -1, nil
	}

	periodStart, _ := dailyPeriod(time.Now().UTC())

	const q = `
		SELECT COALESCE(SUM(quantity), 0)
		FROM usage_meters
		WHERE workspace_id = $1
		  AND meter_type   = 'ai_calls'
		  AND period_start = $2
	`
	var used int
	if err := s.pool.QueryRow(ctx, q, workspaceID, periodStart).Scan(&used); err != nil {
		return false, 0, fmt.Errorf("usage_metering: check ai limit: %w", err)
	}

	rem := limits.AICallsPerDay - used
	if rem < 0 {
		rem = 0
	}
	return used < limits.AICallsPerDay, rem, nil
}

// GetUsageSummary returns the current-period usage across all dimensions for the workspace.
func (s *UsageMeteringService) GetUsageSummary(ctx context.Context, workspaceID uuid.UUID) (*UsageSummary, error) {
	limits, err := s.GetPlanLimits(ctx, workspaceID)
	if err != nil {
		return nil, err
	}

	periodStart, _ := dailyPeriod(time.Now().UTC())

	// AI calls today
	var aiToday int
	const aiQ = `
		SELECT COALESCE(SUM(quantity), 0)
		FROM usage_meters
		WHERE workspace_id = $1
		  AND meter_type   = 'ai_calls'
		  AND period_start = $2
	`
	if err := s.pool.QueryRow(ctx, aiQ, workspaceID, periodStart).Scan(&aiToday); err != nil {
		return nil, fmt.Errorf("usage_metering: ai calls today: %w", err)
	}

	// Storage used (latest row; we treat storage as a gauge, not a counter)
	var storageUsed int64
	const storageQ = `
		SELECT COALESCE(MAX(quantity), 0)
		FROM usage_meters
		WHERE workspace_id = $1
		  AND meter_type   = 'storage_bytes'
	`
	if err := s.pool.QueryRow(ctx, storageQ, workspaceID).Scan(&storageUsed); err != nil {
		return nil, fmt.Errorf("usage_metering: storage used: %w", err)
	}

	// Module count — count rows in osa_module_instances (may not exist yet; handle gracefully)
	var modulesCount int
	const modQ = `SELECT COUNT(*) FROM osa_module_instances WHERE workspace_id = $1`
	if err := s.pool.QueryRow(ctx, modQ, workspaceID).Scan(&modulesCount); err != nil {
		s.logger.WarnContext(ctx, "usage_metering: could not count modules (table may not exist)",
			"workspace_id", workspaceID, "error", err)
	}

	// Team member count
	var teamCount int
	const teamQ = `SELECT COUNT(*) FROM workspace_members WHERE workspace_id = $1`
	if err := s.pool.QueryRow(ctx, teamQ, workspaceID).Scan(&teamCount); err != nil {
		s.logger.WarnContext(ctx, "usage_metering: could not count team members",
			"workspace_id", workspaceID, "error", err)
	}

	return &UsageSummary{
		AICallsToday:     aiToday,
		AICallsLimit:     limits.AICallsPerDay,
		StorageUsed:      storageUsed,
		StorageLimit:     limits.StorageBytesLimit,
		ModulesCount:     modulesCount,
		ModulesLimit:     limits.MaxModules,
		TeamMembers:      teamCount,
		TeamMembersLimit: limits.MaxTeamMembers,
		Plan:             limits.PlanName,
	}, nil
}

// GetPlanLimits returns the limits configured for the workspace's current plan.
func (s *UsageMeteringService) GetPlanLimits(ctx context.Context, workspaceID uuid.UUID) (*PlanLimits, error) {
	const q = `
		SELECT pl.plan_name,
		       pl.ai_calls_per_day,
		       pl.ai_model_tier,
		       pl.max_modules,
		       pl.storage_bytes_limit,
		       pl.max_team_members,
		       pl.osa_modes,
		       pl.compute_cpu_hours_per_month
		FROM   workspaces w
		JOIN   plan_limits pl ON pl.plan_name = w.plan_name
		WHERE  w.id = $1
	`
	var pl PlanLimits
	err := s.pool.QueryRow(ctx, q, workspaceID).Scan(
		&pl.PlanName,
		&pl.AICallsPerDay,
		&pl.AIModelTier,
		&pl.MaxModules,
		&pl.StorageBytesLimit,
		&pl.MaxTeamMembers,
		&pl.OSAModes,
		&pl.ComputeCPUHoursPerMonth,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("usage_metering: workspace %s not found or plan missing", workspaceID)
		}
		return nil, fmt.Errorf("usage_metering: get plan limits: %w", err)
	}
	return &pl, nil
}

// CheckFeatureAccess reports whether the workspace's plan includes the requested feature.
// Recognised feature values: "build", "focus", "research", "plan" (OSA modes beyond "chat"),
// as well as "modules", "team".
func (s *UsageMeteringService) CheckFeatureAccess(ctx context.Context, workspaceID uuid.UUID, feature string) (bool, error) {
	limits, err := s.GetPlanLimits(ctx, workspaceID)
	if err != nil {
		return false, err
	}

	switch feature {
	case "build", "focus", "research", "plan":
		for _, m := range limits.OSAModes {
			if m == feature {
				return true, nil
			}
		}
		return false, nil
	case "modules":
		return limits.MaxModules != 0, nil
	case "team":
		return limits.MaxTeamMembers != 1, nil
	default:
		// Unknown feature — default to allowed so we never hard-block unknown checks.
		s.logger.WarnContext(ctx, "usage_metering: unknown feature check", "feature", feature)
		return true, nil
	}
}

// GetUsageHistory returns all usage_meters rows for the workspace, newest first.
func (s *UsageMeteringService) GetUsageHistory(ctx context.Context, workspaceID uuid.UUID, limit int) ([]UsageHistoryEntry, error) {
	if limit <= 0 || limit > 500 {
		limit = 90
	}

	const q = `
		SELECT meter_type, quantity, period_start, period_end
		FROM   usage_meters
		WHERE  workspace_id = $1
		ORDER  BY period_start DESC
		LIMIT  $2
	`
	rows, err := s.pool.Query(ctx, q, workspaceID, limit)
	if err != nil {
		return nil, fmt.Errorf("usage_metering: get history: %w", err)
	}
	defer rows.Close()

	var entries []UsageHistoryEntry
	for rows.Next() {
		var e UsageHistoryEntry
		if err := rows.Scan(&e.MeterType, &e.Quantity, &e.PeriodStart, &e.PeriodEnd); err != nil {
			return nil, fmt.Errorf("usage_metering: scan history row: %w", err)
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

// dailyPeriod returns the UTC [start, end) for the calendar day containing t.
func dailyPeriod(t time.Time) (start, end time.Time) {
	y, m, d := t.Date()
	start = time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
	end = start.Add(24 * time.Hour)
	return
}
