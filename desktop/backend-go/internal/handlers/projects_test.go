package handlers

// projects_test.go — unit tests for ProjectHandler (CRUD + helpers).
//
// Strategy: no real database required. Tests use gin.CreateTestContext with
// a nil pool (query execution is NOT called because the tests exercise
// request-layer logic — auth guard, binding, parameter parsing, response
// shape — by injecting a mock user into the Gin context.
//
// Tests that would require a live DB are skipped with testing.Short().

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rhl/businessos-backend/internal/database/sqlc"
	"github.com/rhl/businessos-backend/internal/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ─── shared test helpers ────────────────────────────────────────────────────

// newProjectHandler returns a handler with nil dependencies — safe for request-
// layer tests that never reach the database.
func newProjectHandler() *ProjectHandler {
	return &ProjectHandler{
		pool:                 nil,
		queryCache:           nil,
		notificationTriggers: nil,
		projectAccessService: nil,
	}
}

// injectUser plants a BetterAuthUser into a Gin context so handlers that call
// middleware.GetCurrentUser() succeed without real auth middleware.
func injectUser(c *gin.Context, id, name, email string) {
	c.Set(middleware.UserContextKey, &middleware.BetterAuthUser{
		ID:    id,
		Name:  name,
		Email: email,
	})
}

// setupGin switches Gin to test mode and returns a recorder + context.
func setupGin(method, path string, body []byte) (*httptest.ResponseRecorder, *gin.Context) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	var req *http.Request
	if body != nil {
		req = httptest.NewRequest(method, path, bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req = httptest.NewRequest(method, path, nil)
	}
	c.Request = req
	return w, c
}

// ─── ListProjects ────────────────────────────────────────────────────────────

func TestListProjects_Unauthorized(t *testing.T) {
	h := newProjectHandler()
	w, c := setupGin(http.MethodGet, "/api/projects", nil)

	// No user in context → should return 401.
	h.ListProjects(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// ─── CreateProject ───────────────────────────────────────────────────────────

func TestCreateProject_Unauthorized(t *testing.T) {
	h := newProjectHandler()
	body := mustMarshal(t, map[string]string{"name": "My Project"})
	w, c := setupGin(http.MethodPost, "/api/projects", body)

	h.CreateProject(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestCreateProject_MissingName(t *testing.T) {
	// The binding:"required" tag on Name must reject requests without it.
	// We inject a valid user so the auth guard passes.
	h := newProjectHandler()
	body := mustMarshal(t, map[string]string{"description": "no name"})
	w, c := setupGin(http.MethodPost, "/api/projects", body)
	injectUser(c, uuid.New().String(), "Alice", "alice@example.com")

	h.CreateProject(c)

	// 400 because ShouldBindJSON fails — name is required.
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateProject_InvalidJSON(t *testing.T) {
	h := newProjectHandler()
	w, c := setupGin(http.MethodPost, "/api/projects", []byte("{bad json"))
	injectUser(c, uuid.New().String(), "Alice", "alice@example.com")

	h.CreateProject(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ─── GetProject ──────────────────────────────────────────────────────────────

func TestGetProject_Unauthorized(t *testing.T) {
	h := newProjectHandler()
	w, c := setupGin(http.MethodGet, "/api/projects/"+uuid.New().String(), nil)
	c.Params = gin.Params{{Key: "id", Value: uuid.New().String()}}

	h.GetProject(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestGetProject_InvalidUUID(t *testing.T) {
	h := newProjectHandler()
	w, c := setupGin(http.MethodGet, "/api/projects/not-a-uuid", nil)
	injectUser(c, uuid.New().String(), "Alice", "alice@example.com")
	c.Params = gin.Params{{Key: "id", Value: "not-a-uuid"}}

	h.GetProject(c)

	// 400 from RespondInvalidID.
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ─── UpdateProject ───────────────────────────────────────────────────────────

func TestUpdateProject_Unauthorized(t *testing.T) {
	h := newProjectHandler()
	body := mustMarshal(t, map[string]string{"name": "New Name"})
	w, c := setupGin(http.MethodPut, "/api/projects/"+uuid.New().String(), body)
	c.Params = gin.Params{{Key: "id", Value: uuid.New().String()}}

	h.UpdateProject(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestUpdateProject_InvalidUUID(t *testing.T) {
	h := newProjectHandler()
	body := mustMarshal(t, map[string]string{"name": "New Name"})
	w, c := setupGin(http.MethodPut, "/api/projects/bad-uuid", body)
	injectUser(c, uuid.New().String(), "Alice", "alice@example.com")
	c.Params = gin.Params{{Key: "id", Value: "bad-uuid"}}

	h.UpdateProject(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdateProject_InvalidJSON(t *testing.T) {
	h := newProjectHandler()
	w, c := setupGin(http.MethodPut, "/api/projects/"+uuid.New().String(), []byte("{corrupt"))
	injectUser(c, uuid.New().String(), "Alice", "alice@example.com")
	c.Params = gin.Params{{Key: "id", Value: uuid.New().String()}}

	h.UpdateProject(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ─── DeleteProject ───────────────────────────────────────────────────────────

func TestDeleteProject_Unauthorized(t *testing.T) {
	h := newProjectHandler()
	w, c := setupGin(http.MethodDelete, "/api/projects/"+uuid.New().String(), nil)
	c.Params = gin.Params{{Key: "id", Value: uuid.New().String()}}

	h.DeleteProject(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestDeleteProject_InvalidUUID(t *testing.T) {
	h := newProjectHandler()
	w, c := setupGin(http.MethodDelete, "/api/projects/not-uuid", nil)
	injectUser(c, uuid.New().String(), "Alice", "alice@example.com")
	c.Params = gin.Params{{Key: "id", Value: "not-uuid"}}

	h.DeleteProject(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ─── AddProjectNote ──────────────────────────────────────────────────────────

func TestAddProjectNote_Unauthorized(t *testing.T) {
	h := newProjectHandler()
	body := mustMarshal(t, map[string]string{"content": "A note"})
	w, c := setupGin(http.MethodPost, "/api/projects/"+uuid.New().String()+"/notes", body)
	c.Params = gin.Params{{Key: "id", Value: uuid.New().String()}}

	h.AddProjectNote(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAddProjectNote_InvalidUUID(t *testing.T) {
	h := newProjectHandler()
	body := mustMarshal(t, map[string]string{"content": "A note"})
	w, c := setupGin(http.MethodPost, "/api/projects/bad-uuid/notes", body)
	injectUser(c, uuid.New().String(), "Alice", "alice@example.com")
	c.Params = gin.Params{{Key: "id", Value: "bad-uuid"}}

	h.AddProjectNote(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAddProjectNote_MissingContent(t *testing.T) {
	// content is binding:"required" — empty body must be rejected.
	h := newProjectHandler()
	body := mustMarshal(t, map[string]string{})
	w, c := setupGin(http.MethodPost, "/api/projects/"+uuid.New().String()+"/notes", body)
	injectUser(c, uuid.New().String(), "Alice", "alice@example.com")
	c.Params = gin.Params{{Key: "id", Value: uuid.New().String()}}

	h.AddProjectNote(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ─── GetProjectStats / GetOverdueProjects / GetUpcomingProjects ──────────────

func TestGetProjectStats_Unauthorized(t *testing.T) {
	h := newProjectHandler()
	w, c := setupGin(http.MethodGet, "/api/projects/stats", nil)
	h.GetProjectStats(c)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestGetOverdueProjects_Unauthorized(t *testing.T) {
	h := newProjectHandler()
	w, c := setupGin(http.MethodGet, "/api/projects/overdue", nil)
	h.GetOverdueProjects(c)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestGetUpcomingProjects_Unauthorized(t *testing.T) {
	h := newProjectHandler()
	w, c := setupGin(http.MethodGet, "/api/projects/upcoming", nil)
	h.GetUpcomingProjects(c)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// ─── Helper: stringToProjectStatus ───────────────────────────────────────────

func TestStringToProjectStatus(t *testing.T) {
	tests := []struct {
		input    string
		expected sqlc.Projectstatus
	}{
		{"active", sqlc.ProjectstatusACTIVE},
		{"ACTIVE", sqlc.ProjectstatusACTIVE},
		{"paused", sqlc.ProjectstatusPAUSED},
		{"Paused", sqlc.ProjectstatusPAUSED},
		{"completed", sqlc.ProjectstatusCOMPLETED},
		{"archived", sqlc.ProjectstatusARCHIVED},
		// Unknown inputs fall back to ACTIVE.
		{"unknown", sqlc.ProjectstatusACTIVE},
		{"", sqlc.ProjectstatusACTIVE},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := stringToProjectStatus(tt.input)
			assert.Equal(t, tt.expected, got)
		})
	}
}

// ─── Helper: stringToProjectPriority ─────────────────────────────────────────

func TestStringToProjectPriority(t *testing.T) {
	tests := []struct {
		input    string
		expected sqlc.Projectpriority
	}{
		{"critical", sqlc.ProjectpriorityCRITICAL},
		{"CRITICAL", sqlc.ProjectpriorityCRITICAL},
		{"high", sqlc.ProjectpriorityHIGH},
		{"medium", sqlc.ProjectpriorityMEDIUM},
		{"low", sqlc.ProjectpriorityLOW},
		// Unknown falls back to MEDIUM.
		{"", sqlc.ProjectpriorityMEDIUM},
		{"urgent", sqlc.ProjectpriorityMEDIUM},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := stringToProjectPriority(tt.input)
			assert.Equal(t, tt.expected, got)
		})
	}
}

// ─── Helper: TransformProjectRows ────────────────────────────────────────────

func TestTransformProjectRows_EmptySlice(t *testing.T) {
	result := TransformProjectRows([]sqlc.ListProjectsRow{})
	require.NotNil(t, result)
	assert.Len(t, result, 0)
}

func TestTransformProjectRows_SingleRow(t *testing.T) {
	id := uuid.New()
	row := sqlc.ListProjectsRow{
		ID:     pgtype.UUID{Bytes: id, Valid: true},
		UserID: "user-123",
		Name:   "Alpha Project",
		Status: sqlc.NullProjectstatus{
			Projectstatus: sqlc.ProjectstatusACTIVE,
			Valid:         true,
		},
		Priority: sqlc.NullProjectpriority{
			Projectpriority: sqlc.ProjectpriorityHIGH,
			Valid:           true,
		},
	}

	result := TransformProjectRows([]sqlc.ListProjectsRow{row})
	require.Len(t, result, 1)

	m := result[0]
	assert.Equal(t, id.String(), *m["id"].(*string))
	assert.Equal(t, "user-123", m["user_id"])
	assert.Equal(t, "Alpha Project", m["name"])
	assert.Equal(t, "active", m["status"])
	assert.Equal(t, "high", m["priority"])
}

func TestTransformProjectRows_NilUUID(t *testing.T) {
	row := sqlc.ListProjectsRow{
		ID:     pgtype.UUID{Valid: false}, // nil UUID
		UserID: "user-x",
		Name:   "No UUID",
	}
	result := TransformProjectRows([]sqlc.ListProjectsRow{row})
	require.Len(t, result, 1)
	// id should be nil when UUID is not valid.
	assert.Nil(t, result[0]["id"])
}

func TestTransformProjectRows_MultipleRows(t *testing.T) {
	rows := make([]sqlc.ListProjectsRow, 5)
	for i := range rows {
		rows[i] = sqlc.ListProjectsRow{
			ID:     pgtype.UUID{Bytes: uuid.New(), Valid: true},
			UserID: "user-multi",
			Name:   "Project",
		}
	}
	result := TransformProjectRows(rows)
	assert.Len(t, result, 5)
}

// ─── Helper: dateToString ─────────────────────────────────────────────────────

func TestDateToString_NilDate(t *testing.T) {
	result := dateToString(pgtype.Date{Valid: false})
	assert.Nil(t, result)
}

func TestDateToString_ValidDate(t *testing.T) {
	d := pgtype.Date{Time: time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC), Valid: true}
	result := dateToString(d)
	require.NotNil(t, result)
	assert.Equal(t, "2025-06-15", *result)
}

// ─── Helper: projectUUIDToString ─────────────────────────────────────────────

func TestProjectUUIDToString_Invalid(t *testing.T) {
	result := projectUUIDToString(pgtype.UUID{Valid: false})
	assert.Nil(t, result)
}

func TestProjectUUIDToString_Valid(t *testing.T) {
	id := uuid.New()
	result := projectUUIDToString(pgtype.UUID{Bytes: id, Valid: true})
	require.NotNil(t, result)
	assert.Equal(t, id.String(), *result)
}

// ─── Helper: projectTimestampToString ─────────────────────────────────────────

func TestProjectTimestampToString_Invalid(t *testing.T) {
	result := projectTimestampToString(pgtype.Timestamp{Valid: false})
	assert.Nil(t, result)
}

func TestProjectTimestampToString_Valid(t *testing.T) {
	ts := time.Date(2025, 1, 2, 15, 4, 5, 0, time.UTC)
	result := projectTimestampToString(pgtype.Timestamp{Time: ts, Valid: true})
	require.NotNil(t, result)
	assert.True(t, strings.HasPrefix(*result, "2025-01-02"))
}

// ─── Helper: projectTimestamptzToString ──────────────────────────────────────

func TestProjectTimestamptzToString_Invalid(t *testing.T) {
	result := projectTimestamptzToString(pgtype.Timestamptz{Valid: false})
	assert.Nil(t, result)
}

func TestProjectTimestamptzToString_Valid(t *testing.T) {
	ts := time.Date(2025, 3, 7, 12, 0, 0, 0, time.UTC)
	result := projectTimestamptzToString(pgtype.Timestamptz{Time: ts, Valid: true})
	require.NotNil(t, result)
	assert.Contains(t, *result, "2025-03-07")
}

// ─── Route shape (no DB) ──────────────────────────────────────────────────────

// TestProjectRoutes_RegistrationSmoke verifies the router can be built without
// panicking when RegisterProjectRoutes is called with a no-op auth handler.
func TestProjectRoutes_RegistrationSmoke(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	api := r.Group("/api")
	h := newProjectHandler()
	noopAuth := func(c *gin.Context) { c.Next() }
	// Must not panic.
	assert.NotPanics(t, func() {
		RegisterProjectRoutes(api, h, noopAuth)
	})
}

// ─── Binding validation table ─────────────────────────────────────────────────

// TestCreateProject_BindingValidation covers cases rejected before reaching the DB.
func TestCreateProject_BindingValidation(t *testing.T) {
	cases := []struct {
		name       string
		body       map[string]interface{}
		wantStatus int
	}{
		{
			name:       "missing name — rejected at binding",
			body:       map[string]interface{}{"status": "active"},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "empty name string — rejected at binding",
			body:       map[string]interface{}{"name": ""},
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := newProjectHandler()
			body := mustMarshal(t, tc.body)
			w, c := setupGin(http.MethodPost, "/api/projects", body)
			injectUser(c, uuid.New().String(), "Alice", "alice@example.com")

			h.CreateProject(c)

			assert.Equal(t, tc.wantStatus, w.Code)
		})
	}
}

// TestCreateProject_ValidBody_ReachesDB verifies a minimal valid body passes
// binding. The nil pool panics when the DB is reached — we intercept.
func TestCreateProject_ValidBody_ReachesDB(t *testing.T) {
	body := mustMarshal(t, map[string]interface{}{"name": "Proj A"})
	panicked := false
	func() {
		defer func() {
			if r := recover(); r != nil {
				panicked = true
			}
		}()
		h := newProjectHandler()
		w, c := setupGin(http.MethodPost, "/api/projects", body)
		injectUser(c, uuid.New().String(), "Alice", "alice@example.com")
		h.CreateProject(c)
		assert.NotEqual(t, http.StatusBadRequest, w.Code)
	}()
	assert.True(t, panicked, "valid body should reach DB call (panics on nil pool)")
}

// ─── JSON encode helpers ──────────────────────────────────────────────────────

func mustMarshal(t *testing.T, v interface{}) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	require.NoError(t, err)
	return b
}
