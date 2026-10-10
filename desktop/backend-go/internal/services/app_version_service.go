package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rhl/businessos-backend/internal/database/sqlc"
)

// AppVersionService handles versioning and snapshotting of user-generated apps
type AppVersionService struct {
	pool    *pgxpool.Pool
	queries *sqlc.Queries
	logger  *slog.Logger
}

// AppVersionSnapshot represents the complete state of an app at a point in time
type AppVersionSnapshot struct {
	ID               uuid.UUID              `json:"id"`
	AppID            uuid.UUID              `json:"app_id"`
	VersionNumber    string                 `json:"version_number"`
	SnapshotData     map[string]interface{} `json:"snapshot_data"`
	SnapshotMetadata map[string]interface{} `json:"snapshot_metadata,omitempty"`
	ChangeSummary    *string                `json:"change_summary,omitempty"`
	CreatedBy        *string                `json:"created_by,omitempty"`
	CreatedAt        time.Time              `json:"created_at"`
}

// VersionStats provides statistics about app versions
type VersionStats struct {
	TotalVersions   int64     `json:"total_versions"`
	FirstVersionAt  time.Time `json:"first_version_at"`
	LatestVersionAt time.Time `json:"latest_version_at"`
	UniqueCreators  int64     `json:"unique_creators"`
}

// SemanticVersion represents a parsed semantic version
type SemanticVersion struct {
	Major int
	Minor int
	Patch int
}

func NewAppVersionService(pool *pgxpool.Pool, logger *slog.Logger) *AppVersionService {
	return &AppVersionService{
		pool:    pool,
		queries: sqlc.New(pool),
		logger:  logger,
	}
}

// CreateSnapshot creates a new version snapshot of an app
func (s *AppVersionService) CreateSnapshot(ctx context.Context, appID uuid.UUID, userID *string, changeSummary *string) (*AppVersionSnapshot, error) {
	s.logger.Info("Creating app version snapshot",
		slog.String("app_id", appID.String()),
		slog.Any("user_id", userID))

	// Get current app state
	appState, err := s.getCurrentAppState(ctx, appID)
	if err != nil {
		return nil, fmt.Errorf("get current app state: %w", err)
	}

	// Get next version number
	nextVersion, err := s.getNextVersionNumber(ctx, appID)
	if err != nil {
		return nil, fmt.Errorf("get next version number: %w", err)
	}

	// Prepare snapshot data
	snapshotDataJSON, err := json.Marshal(appState)
	if err != nil {
		return nil, fmt.Errorf("marshal snapshot data: %w", err)
	}

	// Create metadata
	metadata := map[string]interface{}{
		"snapshot_size": len(snapshotDataJSON),
		"timestamp":     time.Now().Unix(),
	}
	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return nil, fmt.Errorf("marshal metadata: %w", err)
	}

	// Convert userID string to UUID if provided
	var createdByUUID pgtype.UUID
	if userID != nil {
		parsedUserID, err := uuid.Parse(*userID)
		if err == nil {
			createdByUUID = pgtype.UUID{Bytes: parsedUserID, Valid: true}
		}
	}

	// Create version record
	var changeSummaryStr *string
	if changeSummary != nil {
		changeSummaryStr = changeSummary
	} else {
		autoSummary := fmt.Sprintf("Auto-snapshot v%s", nextVersion)
		changeSummaryStr = &autoSummary
	}

	version, err := s.queries.CreateAppVersion(ctx, sqlc.CreateAppVersionParams{
		AppID:            pgtype.UUID{Bytes: appID, Valid: true},
		VersionNumber:    nextVersion,
		SnapshotData:     snapshotDataJSON,
		SnapshotMetadata: metadataJSON,
		ChangeSummary:    changeSummaryStr,
		CreatedBy:        createdByUUID,
	})
	if err != nil {
		return nil, fmt.Errorf("create version record: %w", err)
	}

	s.logger.Info("App version snapshot created",
		slog.String("version_number", nextVersion),
		slog.String("app_id", appID.String()))

	return s.convertToSnapshot(version), nil
}

// RestoreVersion restores an app to a specific version
func (s *AppVersionService) RestoreVersion(ctx context.Context, appID uuid.UUID, versionNumber string) error {
	s.logger.Info("Restoring app to version",
		slog.String("app_id", appID.String()),
		slog.String("version_number", versionNumber))

	// Get the version snapshot
	version, err := s.queries.GetRestoreData(ctx, sqlc.GetRestoreDataParams{
		AppID:         pgtype.UUID{Bytes: appID, Valid: true},
		VersionNumber: versionNumber,
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			return fmt.Errorf("version %s not found for app %s", versionNumber, appID)
		}
		return fmt.Errorf("get restore data: %w", err)
	}

	// Parse snapshot data
	var snapshotData map[string]interface{}
	if err := json.Unmarshal(version.SnapshotData, &snapshotData); err != nil {
		return fmt.Errorf("unmarshal snapshot data: %w", err)
	}

	// Restore app state
	if err := s.restoreAppState(ctx, appID, snapshotData); err != nil {
		return fmt.Errorf("restore app state: %w", err)
	}

	s.logger.Info("App restored to version",
		slog.String("version_number", versionNumber),
		slog.String("app_id", appID.String()))

	return nil
}

// ListVersions returns all versions for an app
func (s *AppVersionService) ListVersions(ctx context.Context, appID uuid.UUID) ([]AppVersionSnapshot, error) {
	versions, err := s.queries.ListAppVersions(ctx, pgtype.UUID{Bytes: appID, Valid: true})
	if err != nil {
		return nil, fmt.Errorf("list versions: %w", err)
	}

	snapshots := make([]AppVersionSnapshot, 0, len(versions))
	for _, v := range versions {
		snapshots = append(snapshots, *s.convertToSnapshot(v))
	}

	return snapshots, nil
}

// GetVersion retrieves a specific version by version number
func (s *AppVersionService) GetVersion(ctx context.Context, appID uuid.UUID, versionNumber string) (*AppVersionSnapshot, error) {
	version, err := s.queries.GetAppVersionByNumber(ctx, sqlc.GetAppVersionByNumberParams{
		AppID:         pgtype.UUID{Bytes: appID, Valid: true},
		VersionNumber: versionNumber,
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("version %s not found", versionNumber)
		}
		return nil, fmt.Errorf("get version: %w", err)
	}

	return s.convertToSnapshot(version), nil
}

// GetLatestVersion gets the most recent version of an app
func (s *AppVersionService) GetLatestVersion(ctx context.Context, appID uuid.UUID) (*AppVersionSnapshot, error) {
	version, err := s.queries.GetLatestAppVersion(ctx, pgtype.UUID{Bytes: appID, Valid: true})
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("no versions found for app %s", appID)
		}
		return nil, fmt.Errorf("get latest version: %w", err)
	}

	return s.convertToSnapshot(version), nil
}

// GetVersionStats returns statistics about versions for an app
func (s *AppVersionService) GetVersionStats(ctx context.Context, appID uuid.UUID) (*VersionStats, error) {
	stats, err := s.queries.GetVersionStats(ctx, pgtype.UUID{Bytes: appID, Valid: true})
	if err != nil {
		return nil, fmt.Errorf("get version stats: %w", err)
	}

	return &VersionStats{
		TotalVersions:   stats.TotalVersions,
		FirstVersionAt:  stats.FirstVersionAt.(time.Time),
		LatestVersionAt: stats.LatestVersionAt.(time.Time),
		UniqueCreators:  stats.UniqueCreators,
	}, nil
}

// DeleteOldVersions removes old versions, keeping only the N most recent
func (s *AppVersionService) DeleteOldVersions(ctx context.Context, appID uuid.UUID, keepCount int32) error {
	if err := s.queries.DeleteOldVersions(ctx, sqlc.DeleteOldVersionsParams{
		AppID: pgtype.UUID{Bytes: appID, Valid: true},
		Limit: keepCount,
	}); err != nil {
		return fmt.Errorf("delete old versions: %w", err)
	}

	s.logger.Info("Deleted old versions",
		slog.String("app_id", appID.String()),
		slog.Int("kept_count", int(keepCount)))

	return nil
}

// Helper: Get current app state (from user_generated_apps table)
func (s *AppVersionService) getCurrentAppState(ctx context.Context, appID uuid.UUID) (map[string]interface{}, error) {
	var state map[string]interface{}

	row := s.pool.QueryRow(ctx, `
		SELECT jsonb_build_object(
			'id', id,
			'workspace_id', workspace_id,
			'template_id', template_id,
			'app_name', app_name,
			'module_instance_id', module_instance_id,
			'is_visible', is_visible,
			'is_pinned', is_pinned,
			'is_favorite', is_favorite,
			'position_index', position_index,
			'custom_config', custom_config,
			'custom_icon', custom_icon,
			'generated_at', generated_at,
			'last_accessed_at', last_accessed_at,
			'access_count', access_count
		) as app_state
		FROM user_generated_apps
		WHERE id = $1
	`, appID)

	var stateJSON []byte
	if err := row.Scan(&stateJSON); err != nil {
		return nil, fmt.Errorf("scan app state: %w", err)
	}

	if err := json.Unmarshal(stateJSON, &state); err != nil {
		return nil, fmt.Errorf("unmarshal app state: %w", err)
	}

	return state, nil
}

// Helper: Restore app state to the database
func (s *AppVersionService) restoreAppState(ctx context.Context, appID uuid.UUID, state map[string]interface{}) error {
	// Extract fields from state
	customConfig, _ := json.Marshal(state["custom_config"])

	_, err := s.pool.Exec(ctx, `
		UPDATE user_generated_apps
		SET
			is_visible = COALESCE($2::boolean, is_visible),
			is_pinned = COALESCE($3::boolean, is_pinned),
			is_favorite = COALESCE($4::boolean, is_favorite),
			position_index = COALESCE($5::int, position_index),
			custom_config = COALESCE($6::jsonb, custom_config),
			custom_icon = COALESCE($7::varchar, custom_icon),
			updated_at = NOW()
		WHERE id = $1
	`,
		appID,
		state["is_visible"],
		state["is_pinned"],
		state["is_favorite"],
		state["position_index"],
		customConfig,
		state["custom_icon"],
	)

	return err
}

// Helper: Get next semantic version number
func (s *AppVersionService) getNextVersionNumber(ctx context.Context, appID uuid.UUID) (string, error) {
	// Get the latest version
	latestVersion, err := s.queries.GetLatestAppVersion(ctx, pgtype.UUID{Bytes: appID, Valid: true})
	if err != nil {
		if err == pgx.ErrNoRows {
			// First version
			return "0.0.1", nil
		}
		return "", fmt.Errorf("get latest version: %w", err)
	}

	// Parse current version
	current, err := parseSemanticVersion(latestVersion.VersionNumber)
	if err != nil {
		s.logger.Warn("Failed to parse version, starting from 0.0.1",
			slog.String("version", latestVersion.VersionNumber),
			slog.Any("error", err))
		return "0.0.1", nil
	}

	// Increment patch version
	current.Patch++

	return current.String(), nil
}

// Helper: Convert SQLC model to service model
func (s *AppVersionService) convertToSnapshot(v sqlc.AppVersion) *AppVersionSnapshot {
	snapshot := &AppVersionSnapshot{
		ID:            uuid.UUID(v.ID.Bytes),
		AppID:         uuid.UUID(v.AppID.Bytes),
		VersionNumber: v.VersionNumber,
		CreatedAt:     v.CreatedAt.Time,
	}

	// Parse snapshot data
	if len(v.SnapshotData) > 0 {
		_ = json.Unmarshal(v.SnapshotData, &snapshot.SnapshotData)
	}

	// Parse snapshot metadata
	if len(v.SnapshotMetadata) > 0 {
		_ = json.Unmarshal(v.SnapshotMetadata, &snapshot.SnapshotMetadata)
	}

	// Set optional fields
	if v.ChangeSummary != nil {
		snapshot.ChangeSummary = v.ChangeSummary
	}

	if v.CreatedBy.Valid {
		createdBy := uuid.UUID(v.CreatedBy.Bytes).String()
		snapshot.CreatedBy = &createdBy
	}

	return snapshot
}

// Helper: Parse semantic version string
func parseSemanticVersion(version string) (*SemanticVersion, error) {
	parts := strings.Split(version, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid version format: %s", version)
	}

	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return nil, fmt.Errorf("invalid major version: %w", err)
	}

	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		return nil, fmt.Errorf("invalid minor version: %w", err)
	}

	patch, err := strconv.Atoi(parts[2])
	if err != nil {
		return nil, fmt.Errorf("invalid patch version: %w", err)
	}

	return &SemanticVersion{
		Major: major,
		Minor: minor,
		Patch: patch,
	}, nil
}

// Helper: Convert semantic version to string
func (v *SemanticVersion) String() string {
	return fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
}

// IncrementMinor increments the minor version and resets patch
func (v *SemanticVersion) IncrementMinor() {
	v.Minor++
	v.Patch = 0
}

// IncrementMajor increments the major version and resets minor and patch
func (v *SemanticVersion) IncrementMajor() {
	v.Major++
	v.Minor = 0
	v.Patch = 0
}
