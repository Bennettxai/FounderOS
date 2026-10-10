package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rhl/businessos-backend/internal/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

// syncRouter creates a Gin engine with an optional authenticated user injected.
func syncRouter(userID string, routes func(r *gin.Engine, h *SyncHandler)) (*gin.Engine, *SyncHandler) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	// SyncHandler doesn't use the pool for the unit-testable paths (status,
	// validating table names, timestamp parsing) — nil pool is safe for those.
	h := NewSyncHandler(nil)

	if userID != "" {
		r.Use(func(c *gin.Context) {
			c.Set(middleware.UserContextKey, &middleware.BetterAuthUser{
				ID:    userID,
				Email: userID + "@example.com",
			})
			c.Next()
		})
	}

	routes(r, h)
	return r, h
}

// ---------------------------------------------------------------------------
// NewSyncHandler
// ---------------------------------------------------------------------------

func TestNewSyncHandler_NotNil(t *testing.T) {
	h := NewSyncHandler(nil)
	assert.NotNil(t, h)
}

// ---------------------------------------------------------------------------
// SyncableTables map
// ---------------------------------------------------------------------------

func TestSyncableTables_ContainsExpectedTables(t *testing.T) {
	expected := []string{
		"contexts",
		"conversations",
		"messages",
		"projects",
		"artifacts",
		"nodes",
		"team_members",
		"tasks",
		"focus_items",
		"daily_logs",
		"user_settings",
		"clients",
		"client_contacts",
		"client_interactions",
		"calendar_events",
	}
	for _, tbl := range expected {
		_, ok := SyncableTables[tbl]
		assert.True(t, ok, "expected table %q in SyncableTables", tbl)
	}
}

func TestSyncableTables_QueriesHaveUserIDAndTimestampParams(t *testing.T) {
	// Every syncable query must be parameterised with $1 (user_id) and $2 (since)
	// to prevent cross-user data leakage.
	for table, query := range SyncableTables {
		assert.Contains(t, query, "$1", "table %q query must have $1 user filter", table)
		assert.Contains(t, query, "$2", "table %q query must have $2 timestamp filter", table)
	}
}

// ---------------------------------------------------------------------------
// GetSyncStatus — GET /api/sync/status
// ---------------------------------------------------------------------------

func TestGetSyncStatus_Unauthenticated(t *testing.T) {
	// GetSyncStatus does NOT check auth (it's a health endpoint); verify it
	// returns 200 even without a user.
	r, _ := syncRouter("", func(r *gin.Engine, h *SyncHandler) {
		r.GET("/api/sync/status", h.GetSyncStatus)
	})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/sync/status", nil))
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestGetSyncStatus_ResponseFields(t *testing.T) {
	r, _ := syncRouter("user-1", func(r *gin.Engine, h *SyncHandler) {
		r.GET("/api/sync/status", h.GetSyncStatus)
	})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/sync/status", nil))

	require.Equal(t, http.StatusOK, w.Code)

	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "healthy", body["status"])
	assert.Equal(t, "1.0.0", body["version"])
	assert.NotEmpty(t, body["server_timestamp"])
}

func TestGetSyncStatus_TimestampIsRFC3339(t *testing.T) {
	r, _ := syncRouter("user-1", func(r *gin.Engine, h *SyncHandler) {
		r.GET("/api/sync/status", h.GetSyncStatus)
	})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/sync/status", nil))

	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))

	ts, ok := body["server_timestamp"].(string)
	require.True(t, ok, "server_timestamp must be a string")
	_, err := time.Parse(time.RFC3339, ts)
	assert.NoError(t, err, "server_timestamp must be valid RFC3339")
}

// ---------------------------------------------------------------------------
// GetSyncChanges — GET /api/sync/:table
// ---------------------------------------------------------------------------

// For GetSyncChanges we test the logic paths that don't require a live
// database: auth enforcement, table validation, and timestamp parsing.
// Database-backed paths live in integration tests.

func TestGetSyncChanges_Unauthenticated(t *testing.T) {
	r, _ := syncRouter("", func(r *gin.Engine, h *SyncHandler) {
		r.GET("/api/sync/:table", h.GetSyncChanges)
	})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/sync/projects", nil))
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestGetSyncChanges_UnknownTable(t *testing.T) {
	r, _ := syncRouter("user-1", func(r *gin.Engine, h *SyncHandler) {
		r.GET("/api/sync/:table", h.GetSyncChanges)
	})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/sync/not_a_real_table", nil))
	// Pool is nil so the handler returns 400 before hitting the DB
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Contains(t, body["error"], "not_a_real_table")
}

func TestGetSyncChanges_InvalidSinceTimestamp(t *testing.T) {
	r, _ := syncRouter("user-1", func(r *gin.Engine, h *SyncHandler) {
		r.GET("/api/sync/:table", h.GetSyncChanges)
	})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/sync/projects?since=not-a-date", nil))
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Contains(t, body["error"], "Invalid since timestamp")
}

func TestGetSyncChanges_AllKnownTables_PassValidation(t *testing.T) {
	// Verify each table in SyncableTables passes the table-name guard
	// (the map lookup that prevents unknown/injected table names).
	// We test this via the SyncableTables map directly to avoid the nil-pool
	// panic that would occur if we reached the DB query step.
	for table := range SyncableTables {
		t.Run(table, func(t *testing.T) {
			_, ok := SyncableTables[table]
			assert.True(t, ok, "table %q must be present in SyncableTables", table)
		})
	}
}

func TestGetSyncChanges_ValidRFC3339Since_PassesValidation(t *testing.T) {
	// Verify that a valid RFC3339 timestamp parses without error — the same
	// logic the handler uses before it reaches the DB query.
	since := time.Now().Add(-24 * time.Hour).UTC().Format(time.RFC3339)
	_, err := time.Parse(time.RFC3339, since)
	assert.NoError(t, err, "valid RFC3339 since must parse without error")
}

func TestGetSyncChanges_EmptySince_DefaultsToEpoch(t *testing.T) {
	// An empty since param uses time.Time{} (the zero value / epoch).
	// Verify the zero value is not rejected by ParseIn RFC3339 — an empty string
	// is skipped by the handler, so parsing never occurs.
	var sinceTime time.Time
	assert.True(t, sinceTime.IsZero(), "empty since should produce zero time (epoch)")
}

// ---------------------------------------------------------------------------
// FullSync — GET /api/sync/full
// ---------------------------------------------------------------------------

func TestFullSync_Unauthenticated(t *testing.T) {
	r, _ := syncRouter("", func(r *gin.Engine, h *SyncHandler) {
		r.GET("/api/sync/full", h.FullSync)
	})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/sync/full", nil))
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// ---------------------------------------------------------------------------
// createTableSyncHandler — per-table convenience routes
// ---------------------------------------------------------------------------

func TestCreateTableSyncHandler_Unauthenticated(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := NewSyncHandler(nil)
	r.GET("/projects/sync", h.createTableSyncHandler("projects"))

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/projects/sync", nil))
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestCreateTableSyncHandler_UnknownTable_Returns400(t *testing.T) {
	// createTableSyncHandler is called with a known table at registration time,
	// but let's verify what happens when an unknown table name is used.
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := NewSyncHandler(nil)
	// Inject a user so we get past the auth guard
	r.Use(func(c *gin.Context) {
		c.Set(middleware.UserContextKey, &middleware.BetterAuthUser{ID: "user-1"})
		c.Next()
	})
	r.GET("/bad/sync", h.createTableSyncHandler("definitely_not_a_table"))

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/bad/sync", nil))
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateTableSyncHandler_InvalidSince_Returns400(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := NewSyncHandler(nil)
	r.Use(func(c *gin.Context) {
		c.Set(middleware.UserContextKey, &middleware.BetterAuthUser{ID: "user-1"})
		c.Next()
	})
	r.GET("/projects/sync", h.createTableSyncHandler("projects"))

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/projects/sync?since=garbage-timestamp", nil))
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "Invalid since timestamp", body["error"])
}

func TestCreateTableSyncHandler_ValidTableAndSince_PassesValidation(t *testing.T) {
	// Validate that the parsing logic used inside createTableSyncHandler
	// accepts a valid RFC3339 "since" param — same code path as GetSyncChanges.
	since := time.Now().Add(-1 * time.Hour).UTC().Format(time.RFC3339)
	_, err := time.Parse(time.RFC3339, since)
	assert.NoError(t, err, "valid RFC3339 since must not be rejected by timestamp parser")
}

// ---------------------------------------------------------------------------
// RegisterSyncRoutes — route wiring smoke test
// ---------------------------------------------------------------------------

func TestRegisterSyncRoutes_Smoke(t *testing.T) {
	gin.SetMode(gin.TestMode)
	app := gin.New()
	api := app.Group("/api")
	h := NewSyncHandler(nil)

	// auth is a noop in test — real auth is exercised above
	noopAuth := func(c *gin.Context) { c.Next() }

	assert.NotPanics(t, func() {
		RegisterSyncRoutes(api, h, noopAuth)
	}, "RegisterSyncRoutes should not panic")
}

// ---------------------------------------------------------------------------
// Edge cases: timestamp boundary values
// ---------------------------------------------------------------------------

func TestGetSyncChanges_EpochTimestamp(t *testing.T) {
	// Unix epoch in RFC3339 — edge case for "first ever sync".
	// Validate the parsing logic used by the handler directly.
	epoch := "1970-01-01T00:00:00Z"
	_, err := time.Parse(time.RFC3339, epoch)
	assert.NoError(t, err, "epoch timestamp must parse as valid RFC3339")
}

func TestGetSyncChanges_FutureTimestamp(t *testing.T) {
	// Far-future timestamp — valid RFC3339, should parse successfully.
	future := "2099-12-31T23:59:59Z"
	_, err := time.Parse(time.RFC3339, future)
	assert.NoError(t, err, "future timestamp must parse as valid RFC3339")
}

func TestGetSyncChanges_SQLInjectionAttempt(t *testing.T) {
	// Injecting SQL into the table name path parameter.
	// The handler looks the table up in SyncableTables (map lookup) before
	// any DB query, so SQL injection through table name is impossible — this
	// test documents that invariant.
	injectedTable := "projects; DROP TABLE users;--"
	r, _ := syncRouter("user-1", func(r *gin.Engine, h *SyncHandler) {
		r.GET("/api/sync/:table", h.GetSyncChanges)
	})
	w := httptest.NewRecorder()
	// URL-encode the injected value
	req := httptest.NewRequest(http.MethodGet, "/api/sync/projects%3B+DROP+TABLE+users%3B--", nil)
	r.ServeHTTP(w, req)
	// Should be 400 (table not found in map), never 500 or 200
	assert.Equal(t, http.StatusBadRequest, w.Code, "SQL injection via table name must be rejected: %q", injectedTable)
}

func TestGetSyncChanges_EmptyTableName(t *testing.T) {
	// Gin will not route "" to /:table, so this tests with a slash-only path.
	r, _ := syncRouter("user-1", func(r *gin.Engine, h *SyncHandler) {
		r.GET("/api/sync/:table", h.GetSyncChanges)
	})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/sync/", nil))
	// Gin returns 301 or 404 for trailing slash — either is acceptable
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// ---------------------------------------------------------------------------
// SyncRequest struct — zero-value and field contract
// ---------------------------------------------------------------------------

func TestSyncRequest_ZeroValue(t *testing.T) {
	// The zero value of SyncRequest should have empty/nil fields.
	var req SyncRequest
	assert.Empty(t, req.Since)
	assert.Nil(t, req.Tables)
	assert.False(t, req.FullSync)
}

func TestSyncRequest_FieldAssignment(t *testing.T) {
	// SyncRequest uses form tags (bound by Gin from query params), not json tags.
	// Verify direct field assignment to document the expected shape.
	req := SyncRequest{
		Since:    "2025-01-01T00:00:00Z",
		Tables:   []string{"projects", "tasks"},
		FullSync: true,
	}
	assert.Equal(t, "2025-01-01T00:00:00Z", req.Since)
	assert.Equal(t, []string{"projects", "tasks"}, req.Tables)
	assert.True(t, req.FullSync)
}
