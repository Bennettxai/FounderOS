package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// syncableTables is the allowlist of tables that are permitted to participate
// in bidirectional sync. Any table name not in this map is rejected before
// any SQL is executed.
var syncableTables = map[string]bool{
	"projects":                    true,
	"tasks":                       true,
	"clients":                     true,
	"custom_modules":              true,
	"custom_module_versions":      true,
	"custom_module_installations": true,
}

// allowedSyncColumns defines the permitted column names per table for cloud sync.
// Only columns present in this map may appear in push payloads. Any key that is
// not listed here is rejected before any SQL is executed, preventing SQL injection
// via crafted JSON keys (e.g. `"role = 'admin', name"`).
var allowedSyncColumns = map[string]map[string]bool{
	"projects": {
		"name": true, "description": true, "status": true, "priority": true,
		"client_id": true, "start_date": true, "end_date": true, "due_date": true,
		"budget": true, "tags": true, "metadata": true, "updated_at": true,
	},
	"tasks": {
		"title": true, "description": true, "status": true, "priority": true,
		"due_date": true, "project_id": true, "assignee_id": true, "tags": true,
		"position": true, "updated_at": true,
	},
	"clients": {
		"name": true, "email": true, "phone": true, "company": true,
		"status": true, "notes": true, "tags": true, "updated_at": true,
	},
	"custom_modules": {
		"name": true, "description": true, "manifest": true, "config": true,
		"is_active": true, "updated_at": true,
	},
	"custom_module_versions": {
		"module_id": true, "version": true, "changelog": true, "manifest": true,
		"is_published": true, "updated_at": true,
	},
	"custom_module_installations": {
		"module_id": true, "version_id": true, "workspace_id": true,
		"config": true, "is_active": true, "updated_at": true,
	},
}

// safeColumnNameRE matches only identifiers that consist of a leading letter or
// underscore followed by letters, digits, or underscores. This is defense-in-depth:
// the allowlist is the primary control; the regex catches anything that slips
// through if the allowlist is extended incorrectly.
var safeColumnNameRE = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

// validateColumnName returns true only when col passes both the character-level
// regex check and the per-table allowlist check.
func validateColumnName(table, col string) bool {
	if !safeColumnNameRE.MatchString(col) {
		return false
	}
	allowed, ok := allowedSyncColumns[table]
	if !ok {
		return false
	}
	return allowed[col]
}

// CloudSyncHandler handles cloud-mode synchronisation endpoints between a local
// BusinessOS desktop client and a MIOSA Firecracker VM instance.
type CloudSyncHandler struct {
	pool *pgxpool.Pool
}

// NewCloudSyncHandler constructs a CloudSyncHandler.
func NewCloudSyncHandler(pool *pgxpool.Pool) *CloudSyncHandler {
	return &CloudSyncHandler{pool: pool}
}

// SyncPushRequest is the payload sent by a local BusinessOS client.
type SyncPushRequest struct {
	DeviceID  string       `json:"device_id" binding:"required"`
	Timestamp time.Time    `json:"timestamp" binding:"required"`
	Changes   []SyncChange `json:"changes"   binding:"required"`
}

// SyncChange describes a single record-level change to be applied.
type SyncChange struct {
	Table     string          `json:"table"` // e.g. "projects", "tasks"
	RecordID  string          `json:"record_id"`
	Action    string          `json:"action"` // "create", "update", "delete"
	Data      json.RawMessage `json:"data"`
	UpdatedAt time.Time       `json:"updated_at"`
}

// SyncPullResponse is the payload returned to a client requesting incremental changes.
type SyncPullResponse struct {
	Changes         []SyncChange `json:"changes"`
	ServerTimestamp time.Time    `json:"server_timestamp"`
	HasMore         bool         `json:"has_more"`
}

// SyncStatusResponse is returned by GET /api/sync/status.
type SyncStatusResponse struct {
	LastPush       *time.Time `json:"last_push"`
	LastPull       *time.Time `json:"last_pull"`
	PendingChanges int        `json:"pending_changes"`
	Connected      bool       `json:"connected"`
}

// Push handles POST /api/sync/push.
//
// Receives a batch of SyncChange records from a local BusinessOS client,
// applies each change to the local PostgreSQL database, and records every
// applied change in the sync_log table.
//
// Individual record failures are logged and skipped — they do not abort the
// entire batch. The response includes the count of successfully applied changes.
func (h *CloudSyncHandler) Push(c *gin.Context) {
	var req SyncPushRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":  "invalid request body",
			"detail": err.Error(),
		})
		return
	}

	ctx := c.Request.Context()
	logger := slog.Default()

	logger.InfoContext(ctx, "cloud_sync: push received",
		"device_id", req.DeviceID,
		"timestamp", req.Timestamp,
		"change_count", len(req.Changes),
	)

	applied := 0
	for _, ch := range req.Changes {
		if err := h.applyChange(ctx, req.DeviceID, ch); err != nil {
			logger.WarnContext(ctx, "cloud_sync: push skipping failed change",
				"device_id", req.DeviceID,
				"table", ch.Table,
				"record_id", ch.RecordID,
				"action", ch.Action,
				"error", err,
			)
			continue
		}
		applied++
	}

	logger.InfoContext(ctx, "cloud_sync: push complete",
		"device_id", req.DeviceID,
		"applied", applied,
		"total", len(req.Changes),
	)

	c.JSON(http.StatusOK, gin.H{
		"applied":          applied,
		"server_timestamp": time.Now().UTC(),
	})
}

// Pull handles GET /api/sync/pull.
//
// Returns all records in each syncable table that have been updated after the
// given ?since timestamp. Each changed row is returned as a SyncChange with
// action="update" (deletes are not tracked via this mechanism — a soft-delete
// pattern would be needed for full tombstone support, which is left as a
// follow-up).
//
// Query params:
//   - since      RFC3339 timestamp (required)
//   - device_id  caller device identifier (required)
func (h *CloudSyncHandler) Pull(c *gin.Context) {
	since := c.Query("since")
	deviceID := c.Query("device_id")

	if since == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing required query parameter: since"})
		return
	}
	if deviceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing required query parameter: device_id"})
		return
	}

	sinceTime, err := time.Parse(time.RFC3339, since)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":  "invalid since timestamp; expected RFC3339 format",
			"detail": err.Error(),
		})
		return
	}

	ctx := c.Request.Context()
	logger := slog.Default()

	logger.InfoContext(ctx, "cloud_sync: pull requested",
		"device_id", deviceID,
		"since", sinceTime,
	)

	changes, err := h.collectChanges(ctx, sinceTime)
	if err != nil {
		logger.ErrorContext(ctx, "cloud_sync: pull failed to collect changes",
			"device_id", deviceID,
			"since", sinceTime,
			"error", err,
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to collect changes"})
		return
	}

	// Log the pull operation in sync_log (one entry per table that returned rows).
	now := time.Now().UTC()
	if err := h.logPull(ctx, deviceID, changes, now); err != nil {
		// Non-fatal — log and continue.
		logger.WarnContext(ctx, "cloud_sync: pull failed to write sync_log",
			"device_id", deviceID,
			"error", err,
		)
	}

	logger.InfoContext(ctx, "cloud_sync: pull complete",
		"device_id", deviceID,
		"change_count", len(changes),
	)

	c.JSON(http.StatusOK, SyncPullResponse{
		Changes:         changes,
		ServerTimestamp: now,
		HasMore:         false,
	})
}

// Status handles GET /api/sync/status.
//
// Returns the last push and pull timestamps for the requesting device (or any
// device when device_id is omitted), along with a simple connectivity signal.
func (h *CloudSyncHandler) Status(c *gin.Context) {
	ctx := c.Request.Context()
	deviceID := c.Query("device_id")

	status, err := h.querySyncStatus(ctx, deviceID)
	if err != nil {
		slog.ErrorContext(ctx, "cloud_sync: status query failed",
			"device_id", deviceID,
			"error", err,
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query sync status"})
		return
	}

	c.JSON(http.StatusOK, status)
}

// ─── internal helpers ─────────────────────────────────────────────────────────

// applyChange applies a single SyncChange to the database using parameterized
// queries. The table name is validated against syncableTables before execution.
func (h *CloudSyncHandler) applyChange(ctx context.Context, deviceID string, ch SyncChange) error {
	if !syncableTables[ch.Table] {
		return fmt.Errorf("table %q is not in the sync allowlist", ch.Table)
	}

	switch ch.Action {
	case "create":
		return h.applyCreate(ctx, deviceID, ch)
	case "update":
		return h.applyUpdate(ctx, deviceID, ch)
	case "delete":
		return h.applyDelete(ctx, deviceID, ch)
	default:
		return fmt.Errorf("unknown action %q", ch.Action)
	}
}

// applyCreate inserts the record and writes a sync_log entry within a
// transaction so both succeed or both roll back atomically.
func (h *CloudSyncHandler) applyCreate(ctx context.Context, deviceID string, ch SyncChange) error {
	if ch.RecordID == "" {
		return fmt.Errorf("create action requires record_id")
	}
	if len(ch.Data) == 0 {
		return fmt.Errorf("create action requires data payload")
	}

	// Decode the client-supplied data map.
	var cols map[string]any
	if err := json.Unmarshal(ch.Data, &cols); err != nil {
		return fmt.Errorf("decode data: %w", err)
	}

	// Ensure the authoritative id column matches the record_id in the envelope.
	// "id" is not in the per-table allowlist (it is handled specially here), so
	// we inject it after column validation to avoid an allowlist bypass.
	colNames, placeholders, args, err := buildInsertParts(ch.Table, cols)
	if err != nil {
		return fmt.Errorf("invalid column in create payload: %w", err)
	}

	// Prepend id to the validated column list.
	if colNames == "" {
		colNames = "id"
		placeholders = "$1"
		args = []any{ch.RecordID}
	} else {
		colNames = "id, " + colNames
		// Shift existing placeholders up by one.
		shifted := make([]string, 0, len(args))
		for i := range args {
			shifted = append(shifted, fmt.Sprintf("$%d", i+2))
		}
		placeholders = "$1, " + strings.Join(shifted, ", ")
		args = append([]any{ch.RecordID}, args...)
	}

	// Table name is allowlisted — safe to embed in the query string.
	query := fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES (%s) ON CONFLICT (id) DO NOTHING",
		ch.Table, colNames, placeholders,
	)

	tx, err := h.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	if _, err := tx.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("insert into %s: %w", ch.Table, err)
	}

	if err := writeSyncLogTx(ctx, tx, deviceID, "push", ch); err != nil {
		return fmt.Errorf("write sync_log: %w", err)
	}

	return tx.Commit(ctx)
}

// applyUpdate runs an UPDATE … WHERE id = $N query built from the data map.
func (h *CloudSyncHandler) applyUpdate(ctx context.Context, deviceID string, ch SyncChange) error {
	if ch.RecordID == "" {
		return fmt.Errorf("update action requires record_id")
	}
	if len(ch.Data) == 0 {
		return fmt.Errorf("update action requires data payload")
	}

	var cols map[string]any
	if err := json.Unmarshal(ch.Data, &cols); err != nil {
		return fmt.Errorf("decode data: %w", err)
	}

	// Remove id from the SET clause — it goes into the WHERE clause.
	delete(cols, "id")
	if len(cols) == 0 {
		return fmt.Errorf("update data contains no columns to update")
	}

	setClauses, args, err := buildSetParts(ch.Table, cols)
	if err != nil {
		return fmt.Errorf("invalid column in update payload: %w", err)
	}
	// Append record_id as the final bind arg for the WHERE clause.
	args = append(args, ch.RecordID)
	idPlaceholder := fmt.Sprintf("$%d", len(args))

	query := fmt.Sprintf(
		"UPDATE %s SET %s WHERE id = %s",
		ch.Table, setClauses, idPlaceholder,
	)

	tx, err := h.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	if _, err := tx.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("update %s id=%s: %w", ch.Table, ch.RecordID, err)
	}

	if err := writeSyncLogTx(ctx, tx, deviceID, "push", ch); err != nil {
		return fmt.Errorf("write sync_log: %w", err)
	}

	return tx.Commit(ctx)
}

// applyDelete removes the record and writes a sync_log entry.
func (h *CloudSyncHandler) applyDelete(ctx context.Context, deviceID string, ch SyncChange) error {
	if ch.RecordID == "" {
		return fmt.Errorf("delete action requires record_id")
	}

	query := fmt.Sprintf("DELETE FROM %s WHERE id = $1", ch.Table)

	tx, err := h.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	if _, err := tx.Exec(ctx, query, ch.RecordID); err != nil {
		return fmt.Errorf("delete from %s id=%s: %w", ch.Table, ch.RecordID, err)
	}

	if err := writeSyncLogTx(ctx, tx, deviceID, "push", ch); err != nil {
		return fmt.Errorf("write sync_log: %w", err)
	}

	return tx.Commit(ctx)
}

// collectChanges queries every syncable table for rows updated after sinceTime
// and returns them as a flat slice of SyncChange.
func (h *CloudSyncHandler) collectChanges(ctx context.Context, since time.Time) ([]SyncChange, error) {
	var all []SyncChange

	for table := range syncableTables {
		rows, err := h.pool.Query(ctx,
			// Table name is allowlisted — safe to embed.
			fmt.Sprintf(
				`SELECT id, row_to_json(%s)::jsonb, updated_at FROM %s WHERE updated_at > $1 ORDER BY updated_at`,
				table, table,
			),
			since,
		)
		if err != nil {
			// A table may not have an updated_at column or may not exist yet —
			// log and skip rather than aborting the whole pull.
			slog.WarnContext(ctx, "cloud_sync: pull query failed for table",
				"table", table,
				"error", err,
			)
			continue
		}

		for rows.Next() {
			var id string
			var rawJSON []byte
			var updatedAt time.Time

			if err := rows.Scan(&id, &rawJSON, &updatedAt); err != nil {
				slog.WarnContext(ctx, "cloud_sync: scan row failed",
					"table", table,
					"error", err,
				)
				continue
			}

			all = append(all, SyncChange{
				Table:     table,
				RecordID:  id,
				Action:    "update",
				Data:      json.RawMessage(rawJSON),
				UpdatedAt: updatedAt,
			})
		}
		rows.Close()

		if err := rows.Err(); err != nil {
			slog.WarnContext(ctx, "cloud_sync: rows iteration error",
				"table", table,
				"error", err,
			)
		}
	}

	return all, nil
}

// logPull writes one sync_log row per unique table present in the returned changes.
func (h *CloudSyncHandler) logPull(ctx context.Context, deviceID string, changes []SyncChange, at time.Time) error {
	seen := make(map[string]struct{})
	for _, ch := range changes {
		if _, ok := seen[ch.Table]; ok {
			continue
		}
		seen[ch.Table] = struct{}{}

		_, err := h.pool.Exec(ctx,
			`INSERT INTO sync_log (device_id, direction, table_name, record_id, action, synced_at)
			 VALUES ($1, 'pull', $2, NULL, 'update', $3)`,
			deviceID, ch.Table, at,
		)
		if err != nil {
			return err
		}
	}
	return nil
}

// querySyncStatus reads the most recent push and pull timestamps from sync_log.
// When deviceID is empty, it queries across all devices.
func (h *CloudSyncHandler) querySyncStatus(ctx context.Context, deviceID string) (*SyncStatusResponse, error) {
	var lastPush, lastPull *time.Time

	fetchLatest := func(direction string) (*time.Time, error) {
		var query string
		var args []any

		if deviceID != "" {
			query = `SELECT MAX(synced_at) FROM sync_log WHERE direction = $1 AND device_id = $2`
			args = []any{direction, deviceID}
		} else {
			query = `SELECT MAX(synced_at) FROM sync_log WHERE direction = $1`
			args = []any{direction}
		}

		var t *time.Time
		err := h.pool.QueryRow(ctx, query, args...).Scan(&t)
		if err != nil {
			return nil, err
		}
		return t, nil
	}

	var err error
	lastPush, err = fetchLatest("push")
	if err != nil {
		return nil, fmt.Errorf("query last push: %w", err)
	}

	lastPull, err = fetchLatest("pull")
	if err != nil {
		return nil, fmt.Errorf("query last pull: %w", err)
	}

	return &SyncStatusResponse{
		LastPush:       lastPush,
		LastPull:       lastPull,
		PendingChanges: 0,    // Future: count from a pending_changes queue.
		Connected:      true, // Connected as long as the handler is reachable.
	}, nil
}

// ─── SQL builder helpers ──────────────────────────────────────────────────────

// buildInsertParts converts a column map into the components needed for an
// INSERT statement: "col1, col2, col3", "$1, $2, $3", and the value slice.
// Column order is deterministic (sorted by name).
//
// Every key in cols is validated against the per-table allowlist via
// validateColumnName. An error is returned if any key fails validation so that
// the caller can reject the request before any SQL is executed.
func buildInsertParts(table string, cols map[string]any) (colNames, placeholders string, args []any, err error) {
	keys := sortedKeys(cols)
	names := make([]string, 0, len(keys))
	pholds := make([]string, 0, len(keys))

	for i, k := range keys {
		if !validateColumnName(table, k) {
			return "", "", nil, fmt.Errorf("column %q is not permitted for table %q", k, table)
		}
		names = append(names, k)
		pholds = append(pholds, fmt.Sprintf("$%d", i+1))
		args = append(args, cols[k])
	}

	return strings.Join(names, ", "), strings.Join(pholds, ", "), args, nil
}

// buildSetParts converts a column map into the SET clause fragment and bound
// args for an UPDATE statement. Placeholder numbering starts at $1.
//
// Every key in cols is validated against the per-table allowlist via
// validateColumnName. An error is returned if any key fails validation so that
// the caller can reject the request before any SQL is executed.
func buildSetParts(table string, cols map[string]any) (setClauses string, args []any, err error) {
	keys := sortedKeys(cols)
	parts := make([]string, 0, len(keys))

	for i, k := range keys {
		if !validateColumnName(table, k) {
			return "", nil, fmt.Errorf("column %q is not permitted for table %q", k, table)
		}
		parts = append(parts, fmt.Sprintf("%s = $%d", k, i+1))
		args = append(args, cols[k])
	}

	return strings.Join(parts, ", "), args, nil
}

// sortedKeys returns map keys in alphabetical order for stable SQL generation.
func sortedKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	// Simple insertion sort is fine for the small column counts expected here.
	for i := 1; i < len(keys); i++ {
		for j := i; j > 0 && keys[j] < keys[j-1]; j-- {
			keys[j], keys[j-1] = keys[j-1], keys[j]
		}
	}
	return keys
}

// writeSyncLogTx inserts one row into sync_log within an existing pgx transaction.
func writeSyncLogTx(ctx context.Context, tx pgx.Tx, deviceID, direction string, ch SyncChange) error {
	_, err := tx.Exec(ctx,
		`INSERT INTO sync_log (device_id, direction, table_name, record_id, action)
		 VALUES ($1, $2, $3, $4, $5)`,
		deviceID, direction, ch.Table, ch.RecordID, ch.Action,
	)
	return err
}
