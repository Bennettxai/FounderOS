package handlers

// team_test.go — unit tests for TeamHandler (CRUD + member operations).
//
// Tests cover request-layer logic only (auth guard, binding, ID validation,
// response shape) without a real database. DB-required paths that can only be
// verified end-to-end are annotated with t.Skip(...) for short mode.

import (
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rhl/businessos-backend/internal/database/sqlc"
	"github.com/stretchr/testify/assert"
)

// ─── constructor ────────────────────────────────────────────────────────────

func newTeamHandler() *TeamHandler {
	return &TeamHandler{pool: nil, queryCache: nil}
}

// ─── ListTeamMembers ─────────────────────────────────────────────────────────

func TestListTeamMembers_Unauthorized(t *testing.T) {
	h := newTeamHandler()
	w, c := setupGin(http.MethodGet, "/api/team", nil)

	h.ListTeamMembers(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// ─── CreateTeamMember ────────────────────────────────────────────────────────

func TestCreateTeamMember_Unauthorized(t *testing.T) {
	h := newTeamHandler()
	body := mustMarshal(t, map[string]string{"name": "Bob", "email": "bob@test.com", "role": "dev"})
	w, c := setupGin(http.MethodPost, "/api/team", body)

	h.CreateTeamMember(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestCreateTeamMember_MissingName(t *testing.T) {
	h := newTeamHandler()
	body := mustMarshal(t, map[string]string{"email": "bob@test.com", "role": "dev"})
	w, c := setupGin(http.MethodPost, "/api/team", body)
	injectUser(c, uuid.New().String(), "Alice", "alice@test.com")

	h.CreateTeamMember(c)

	// name is binding:"required".
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateTeamMember_MissingEmail(t *testing.T) {
	h := newTeamHandler()
	body := mustMarshal(t, map[string]string{"name": "Bob", "role": "dev"})
	w, c := setupGin(http.MethodPost, "/api/team", body)
	injectUser(c, uuid.New().String(), "Alice", "alice@test.com")

	h.CreateTeamMember(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateTeamMember_MissingRole(t *testing.T) {
	h := newTeamHandler()
	body := mustMarshal(t, map[string]string{"name": "Bob", "email": "bob@test.com"})
	w, c := setupGin(http.MethodPost, "/api/team", body)
	injectUser(c, uuid.New().String(), "Alice", "alice@test.com")

	h.CreateTeamMember(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateTeamMember_InvalidJSON(t *testing.T) {
	h := newTeamHandler()
	w, c := setupGin(http.MethodPost, "/api/team", []byte("{bad"))
	injectUser(c, uuid.New().String(), "Alice", "alice@test.com")

	h.CreateTeamMember(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateTeamMember_AllFieldsPresent_ReachesDB(t *testing.T) {
	// Verifies binding passes for a full payload. Nil pool panics when DB is hit.
	body := mustMarshal(t, map[string]interface{}{
		"name":        "Carol",
		"email":       "carol@example.com",
		"role":        "designer",
		"avatar_url":  "https://example.com/avatar.png",
		"status":      "available",
		"capacity":    int(80),
		"hourly_rate": 95.5,
	})
	panicked := false
	func() {
		defer func() {
			if r := recover(); r != nil {
				panicked = true
			}
		}()
		h := newTeamHandler()
		w, c := setupGin(http.MethodPost, "/api/team", body)
		injectUser(c, uuid.New().String(), "Alice", "alice@test.com")
		h.CreateTeamMember(c)
		// No panic → handler returned early (should not be 400).
		assert.NotEqual(t, http.StatusBadRequest, w.Code)
	}()
	// Nil pool panics at DB → confirms binding passed.
	assert.True(t, panicked, "full payload should reach DB (panics on nil pool)")
}

// ─── GetTeamMember ───────────────────────────────────────────────────────────

func TestGetTeamMember_Unauthorized(t *testing.T) {
	h := newTeamHandler()
	w, c := setupGin(http.MethodGet, "/api/team/"+uuid.New().String(), nil)
	c.Params = gin.Params{{Key: "id", Value: uuid.New().String()}}

	h.GetTeamMember(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestGetTeamMember_InvalidUUID(t *testing.T) {
	h := newTeamHandler()
	w, c := setupGin(http.MethodGet, "/api/team/bad-uuid", nil)
	injectUser(c, uuid.New().String(), "Alice", "alice@test.com")
	c.Params = gin.Params{{Key: "id", Value: "bad-uuid"}}

	h.GetTeamMember(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ─── UpdateTeamMember ────────────────────────────────────────────────────────

func TestUpdateTeamMember_Unauthorized(t *testing.T) {
	h := newTeamHandler()
	body := mustMarshal(t, map[string]string{"name": "NewName"})
	w, c := setupGin(http.MethodPut, "/api/team/"+uuid.New().String(), body)
	c.Params = gin.Params{{Key: "id", Value: uuid.New().String()}}

	h.UpdateTeamMember(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestUpdateTeamMember_InvalidUUID(t *testing.T) {
	h := newTeamHandler()
	body := mustMarshal(t, map[string]string{"name": "NewName"})
	w, c := setupGin(http.MethodPut, "/api/team/bad", body)
	injectUser(c, uuid.New().String(), "Alice", "alice@test.com")
	c.Params = gin.Params{{Key: "id", Value: "bad"}}

	h.UpdateTeamMember(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdateTeamMember_InvalidJSON(t *testing.T) {
	h := newTeamHandler()
	w, c := setupGin(http.MethodPut, "/api/team/"+uuid.New().String(), []byte("{broken"))
	injectUser(c, uuid.New().String(), "Alice", "alice@test.com")
	c.Params = gin.Params{{Key: "id", Value: uuid.New().String()}}

	h.UpdateTeamMember(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ─── DeleteTeamMember ────────────────────────────────────────────────────────

func TestDeleteTeamMember_Unauthorized(t *testing.T) {
	h := newTeamHandler()
	w, c := setupGin(http.MethodDelete, "/api/team/"+uuid.New().String(), nil)
	c.Params = gin.Params{{Key: "id", Value: uuid.New().String()}}

	h.DeleteTeamMember(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestDeleteTeamMember_InvalidUUID(t *testing.T) {
	h := newTeamHandler()
	w, c := setupGin(http.MethodDelete, "/api/team/not-a-uuid", nil)
	injectUser(c, uuid.New().String(), "Alice", "alice@test.com")
	c.Params = gin.Params{{Key: "id", Value: "not-a-uuid"}}

	h.DeleteTeamMember(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ─── UpdateTeamMemberStatus ──────────────────────────────────────────────────

func TestUpdateTeamMemberStatus_Unauthorized(t *testing.T) {
	h := newTeamHandler()
	body := mustMarshal(t, map[string]string{"status": "busy"})
	w, c := setupGin(http.MethodPatch, "/api/team/"+uuid.New().String()+"/status", body)
	c.Params = gin.Params{{Key: "id", Value: uuid.New().String()}}

	h.UpdateTeamMemberStatus(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestUpdateTeamMemberStatus_InvalidUUID(t *testing.T) {
	h := newTeamHandler()
	body := mustMarshal(t, map[string]string{"status": "busy"})
	w, c := setupGin(http.MethodPatch, "/api/team/bad/status", body)
	injectUser(c, uuid.New().String(), "Alice", "alice@test.com")
	c.Params = gin.Params{{Key: "id", Value: "bad"}}

	h.UpdateTeamMemberStatus(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdateTeamMemberStatus_MissingStatus(t *testing.T) {
	h := newTeamHandler()
	body := mustMarshal(t, map[string]string{})
	w, c := setupGin(http.MethodPatch, "/api/team/"+uuid.New().String()+"/status", body)
	injectUser(c, uuid.New().String(), "Alice", "alice@test.com")
	c.Params = gin.Params{{Key: "id", Value: uuid.New().String()}}

	h.UpdateTeamMemberStatus(c)

	// status is binding:"required".
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ─── UpdateTeamMemberCapacity ─────────────────────────────────────────────────

func TestUpdateTeamMemberCapacity_Unauthorized(t *testing.T) {
	h := newTeamHandler()
	body := mustMarshal(t, map[string]int32{"capacity": 80})
	w, c := setupGin(http.MethodPatch, "/api/team/"+uuid.New().String()+"/capacity", body)
	c.Params = gin.Params{{Key: "id", Value: uuid.New().String()}}

	h.UpdateTeamMemberCapacity(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestUpdateTeamMemberCapacity_InvalidUUID(t *testing.T) {
	h := newTeamHandler()
	body := mustMarshal(t, map[string]int32{"capacity": 80})
	w, c := setupGin(http.MethodPatch, "/api/team/x/capacity", body)
	injectUser(c, uuid.New().String(), "Alice", "alice@test.com")
	c.Params = gin.Params{{Key: "id", Value: "x"}}

	h.UpdateTeamMemberCapacity(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdateTeamMemberCapacity_MissingCapacity(t *testing.T) {
	h := newTeamHandler()
	body := mustMarshal(t, map[string]string{})
	w, c := setupGin(http.MethodPatch, "/api/team/"+uuid.New().String()+"/capacity", body)
	injectUser(c, uuid.New().String(), "Alice", "alice@test.com")
	c.Params = gin.Params{{Key: "id", Value: uuid.New().String()}}

	h.UpdateTeamMemberCapacity(c)

	// capacity is binding:"required".
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ─── AddTeamMemberActivity ───────────────────────────────────────────────────

func TestAddTeamMemberActivity_Unauthorized(t *testing.T) {
	h := newTeamHandler()
	body := mustMarshal(t, map[string]string{"activity_type": "meeting", "description": "Team sync"})
	w, c := setupGin(http.MethodPost, "/api/team/"+uuid.New().String()+"/activity", body)
	c.Params = gin.Params{{Key: "id", Value: uuid.New().String()}}

	h.AddTeamMemberActivity(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAddTeamMemberActivity_InvalidUUID(t *testing.T) {
	h := newTeamHandler()
	body := mustMarshal(t, map[string]string{"activity_type": "meeting", "description": "sync"})
	w, c := setupGin(http.MethodPost, "/api/team/bad/activity", body)
	injectUser(c, uuid.New().String(), "Alice", "alice@test.com")
	c.Params = gin.Params{{Key: "id", Value: "bad"}}

	h.AddTeamMemberActivity(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAddTeamMemberActivity_MissingFields(t *testing.T) {
	h := newTeamHandler()
	// Missing both required fields.
	body := mustMarshal(t, map[string]string{})
	w, c := setupGin(http.MethodPost, "/api/team/"+uuid.New().String()+"/activity", body)
	injectUser(c, uuid.New().String(), "Alice", "alice@test.com")
	c.Params = gin.Params{{Key: "id", Value: uuid.New().String()}}

	h.AddTeamMemberActivity(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAddTeamMemberActivity_MissingDescription(t *testing.T) {
	h := newTeamHandler()
	body := mustMarshal(t, map[string]string{"activity_type": "meeting"})
	w, c := setupGin(http.MethodPost, "/api/team/"+uuid.New().String()+"/activity", body)
	injectUser(c, uuid.New().String(), "Alice", "alice@test.com")
	c.Params = gin.Params{{Key: "id", Value: uuid.New().String()}}

	h.AddTeamMemberActivity(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ─── GetTeamMemberActivities ─────────────────────────────────────────────────

func TestGetTeamMemberActivities_Unauthorized(t *testing.T) {
	h := newTeamHandler()
	w, c := setupGin(http.MethodGet, "/api/team/"+uuid.New().String()+"/activities", nil)
	c.Params = gin.Params{{Key: "id", Value: uuid.New().String()}}

	h.GetTeamMemberActivities(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestGetTeamMemberActivities_InvalidUUID(t *testing.T) {
	h := newTeamHandler()
	w, c := setupGin(http.MethodGet, "/api/team/bad/activities", nil)
	injectUser(c, uuid.New().String(), "Alice", "alice@test.com")
	c.Params = gin.Params{{Key: "id", Value: "bad"}}

	h.GetTeamMemberActivities(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ─── Helper: stringToMemberStatus ────────────────────────────────────────────

func TestStringToMemberStatus(t *testing.T) {
	tests := []struct {
		input    string
		expected sqlc.Memberstatus
	}{
		{"available", sqlc.MemberstatusAVAILABLE},
		{"AVAILABLE", sqlc.MemberstatusAVAILABLE},
		{"busy", sqlc.MemberstatusBUSY},
		{"overloaded", sqlc.MemberstatusOVERLOADED},
		{"ooo", sqlc.MemberstatusOOO},
		{"OOO", sqlc.MemberstatusOOO},
		// Unknown values fall back to AVAILABLE.
		{"vacation", sqlc.MemberstatusAVAILABLE},
		{"", sqlc.MemberstatusAVAILABLE},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := stringToMemberStatus(tt.input)
			assert.Equal(t, tt.expected, got)
		})
	}
}

// ─── Route registration smoke ─────────────────────────────────────────────────

func TestTeamRoutes_RegistrationSmoke(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	api := r.Group("/api")
	h := newTeamHandler()
	noopAuth := func(c *gin.Context) { c.Next() }
	assert.NotPanics(t, func() {
		RegisterTeamRoutes(api, h, noopAuth)
	})
}

// ─── Table-driven binding validation ─────────────────────────────────────────

// TestCreateTeamMember_BindingValidation covers requests rejected before DB.
func TestCreateTeamMember_BindingValidation(t *testing.T) {
	cases := []struct {
		name       string
		body       map[string]interface{}
		wantStatus int
	}{
		{
			name:       "missing name → 400",
			body:       map[string]interface{}{"email": "x@x.com", "role": "dev"},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "missing email → 400",
			body:       map[string]interface{}{"name": "Dave", "role": "dev"},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "missing role → 400",
			body:       map[string]interface{}{"name": "Dave", "email": "d@d.com"},
			wantStatus: http.StatusBadRequest,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := newTeamHandler()
			body := mustMarshal(t, tc.body)
			w, c := setupGin(http.MethodPost, "/api/team", body)
			injectUser(c, uuid.New().String(), "Alice", "alice@test.com")

			h.CreateTeamMember(c)

			assert.Equal(t, tc.wantStatus, w.Code)
		})
	}
}

// TestCreateTeamMember_MinimalBody_ReachesDB verifies binding passes for
// minimum required fields. Nil pool causes a panic which confirms we got past
// the binding layer.
func TestCreateTeamMember_MinimalBody_ReachesDB(t *testing.T) {
	body := mustMarshal(t, map[string]interface{}{
		"name":  "Dave",
		"email": "dave@example.com",
		"role":  "engineer",
	})
	panicked := false
	func() {
		defer func() {
			if r := recover(); r != nil {
				panicked = true
			}
		}()
		h := newTeamHandler()
		w, c := setupGin(http.MethodPost, "/api/team", body)
		injectUser(c, uuid.New().String(), "Alice", "alice@test.com")
		h.CreateTeamMember(c)
		assert.NotEqual(t, http.StatusBadRequest, w.Code)
	}()
	assert.True(t, panicked, "minimal body should reach DB (panics on nil pool)")
}

// ─── GetTeamMember include_activities query parameter ────────────────────────

func TestGetTeamMember_IncludeActivities_InvalidUUID(t *testing.T) {
	// Even with include_activities=true, a bad UUID must be rejected before
	// reaching any DB call.
	h := newTeamHandler()
	w, c := setupGin(http.MethodGet, "/api/team/bad?include_activities=true", nil)
	injectUser(c, uuid.New().String(), "Alice", "alice@test.com")
	c.Params = gin.Params{{Key: "id", Value: "bad"}}
	c.Request.URL.RawQuery = "include_activities=true"

	h.GetTeamMember(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}
