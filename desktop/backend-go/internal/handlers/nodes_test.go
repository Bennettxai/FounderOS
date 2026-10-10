package handlers

// nodes_test.go — unit tests for NodeHandler (CRUD + state management + helpers).
//
// Tests are organized into:
//   1. Request-layer tests (no DB): auth guards, UUID validation, binding.
//   2. Helper function tests: stringToNodeType, stringToNodeHealth, TransformNode(s).
//   3. Route registration smoke test.
//
// Tests that require a live database use t.Skip("Skipping integration test in short mode").

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rhl/businessos-backend/internal/database/sqlc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ─── constructor ────────────────────────────────────────────────────────────

func newNodeHandler() *NodeHandler {
	return NewNodeHandler(nil)
}

// ─── ListNodes ───────────────────────────────────────────────────────────────

func TestListNodes_Unauthorized(t *testing.T) {
	h := newNodeHandler()
	w, c := setupGin(http.MethodGet, "/api/nodes", nil)

	h.ListNodes(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// ─── CreateNode ──────────────────────────────────────────────────────────────

func TestCreateNode_Unauthorized(t *testing.T) {
	h := newNodeHandler()
	body := mustMarshal(t, map[string]string{"name": "Node A", "type": "business"})
	w, c := setupGin(http.MethodPost, "/api/nodes", body)

	h.CreateNode(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestCreateNode_MissingName(t *testing.T) {
	h := newNodeHandler()
	body := mustMarshal(t, map[string]string{"type": "business"})
	w, c := setupGin(http.MethodPost, "/api/nodes", body)
	injectUser(c, uuid.New().String(), "Alice", "alice@test.com")

	h.CreateNode(c)

	// name is binding:"required".
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateNode_MissingType(t *testing.T) {
	h := newNodeHandler()
	body := mustMarshal(t, map[string]string{"name": "Node A"})
	w, c := setupGin(http.MethodPost, "/api/nodes", body)
	injectUser(c, uuid.New().String(), "Alice", "alice@test.com")

	h.CreateNode(c)

	// type is binding:"required".
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateNode_InvalidJSON(t *testing.T) {
	h := newNodeHandler()
	w, c := setupGin(http.MethodPost, "/api/nodes", []byte("{broken"))
	injectUser(c, uuid.New().String(), "Alice", "alice@test.com")

	h.CreateNode(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateNode_AllOptionalFields_BindingPasses(t *testing.T) {
	// Binding must pass for a complete payload. We verify that the request is
	// NOT rejected with 400 (binding error). The pool is nil, so the handler
	// will panic when it reaches the DB call; we catch that with recover.
	parentID := uuid.New().String()
	body := mustMarshal(t, map[string]interface{}{
		"name":             "My Business Node",
		"type":             "business",
		"parent_id":        parentID,
		"health":           "healthy",
		"purpose":          "Lead generation",
		"current_status":   "On track",
		"this_week_focus":  []string{"Hire engineer", "Close deal"},
		"decision_queue":   []string{"Choose CRM vendor"},
		"delegation_ready": []string{"Onboarding docs"},
		"sort_order":       1,
	})

	// Run handler and intercept any panic from the nil pool.
	status := runHandlerCatchPanic(t, body, func(c *gin.Context) {
		h := newNodeHandler()
		injectUser(c, uuid.New().String(), "Alice", "alice@test.com")
		h.CreateNode(c)
	})

	// Binding passed — the status is not a 400.
	assert.NotEqual(t, http.StatusBadRequest, status, "full payload should not be rejected at binding")
}

// ─── GetNode ─────────────────────────────────────────────────────────────────

func TestGetNode_Unauthorized(t *testing.T) {
	h := newNodeHandler()
	w, c := setupGin(http.MethodGet, "/api/nodes/"+uuid.New().String(), nil)
	c.Params = gin.Params{{Key: "id", Value: uuid.New().String()}}

	h.GetNode(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestGetNode_InvalidUUID(t *testing.T) {
	h := newNodeHandler()
	w, c := setupGin(http.MethodGet, "/api/nodes/bad-uuid", nil)
	injectUser(c, uuid.New().String(), "Alice", "alice@test.com")
	c.Params = gin.Params{{Key: "id", Value: "bad-uuid"}}

	h.GetNode(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGetNode_IncludeChildren_InvalidUUID(t *testing.T) {
	h := newNodeHandler()
	w, c := setupGin(http.MethodGet, "/api/nodes/not-uuid?include_children=true", nil)
	injectUser(c, uuid.New().String(), "Alice", "alice@test.com")
	c.Params = gin.Params{{Key: "id", Value: "not-uuid"}}
	c.Request.URL.RawQuery = "include_children=true"

	h.GetNode(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ─── UpdateNode ──────────────────────────────────────────────────────────────

func TestUpdateNode_Unauthorized(t *testing.T) {
	h := newNodeHandler()
	body := mustMarshal(t, map[string]string{"name": "Updated"})
	w, c := setupGin(http.MethodPatch, "/api/nodes/"+uuid.New().String(), body)
	c.Params = gin.Params{{Key: "id", Value: uuid.New().String()}}

	h.UpdateNode(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestUpdateNode_InvalidUUID(t *testing.T) {
	h := newNodeHandler()
	body := mustMarshal(t, map[string]string{"name": "Updated"})
	w, c := setupGin(http.MethodPatch, "/api/nodes/xyz", body)
	injectUser(c, uuid.New().String(), "Alice", "alice@test.com")
	c.Params = gin.Params{{Key: "id", Value: "xyz"}}

	h.UpdateNode(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdateNode_InvalidJSON(t *testing.T) {
	h := newNodeHandler()
	w, c := setupGin(http.MethodPatch, "/api/nodes/"+uuid.New().String(), []byte("{broken"))
	injectUser(c, uuid.New().String(), "Alice", "alice@test.com")
	c.Params = gin.Params{{Key: "id", Value: uuid.New().String()}}

	h.UpdateNode(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ─── DeleteNode ──────────────────────────────────────────────────────────────

func TestDeleteNode_Unauthorized(t *testing.T) {
	h := newNodeHandler()
	w, c := setupGin(http.MethodDelete, "/api/nodes/"+uuid.New().String(), nil)
	c.Params = gin.Params{{Key: "id", Value: uuid.New().String()}}

	h.DeleteNode(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestDeleteNode_InvalidUUID(t *testing.T) {
	h := newNodeHandler()
	w, c := setupGin(http.MethodDelete, "/api/nodes/bad", nil)
	injectUser(c, uuid.New().String(), "Alice", "alice@test.com")
	c.Params = gin.Params{{Key: "id", Value: "bad"}}

	h.DeleteNode(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ─── GetActiveNode ───────────────────────────────────────────────────────────

func TestGetActiveNode_Unauthorized(t *testing.T) {
	h := newNodeHandler()
	w, c := setupGin(http.MethodGet, "/api/nodes/active", nil)

	h.GetActiveNode(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// ─── ActivateNode ────────────────────────────────────────────────────────────

func TestActivateNode_Unauthorized(t *testing.T) {
	h := newNodeHandler()
	w, c := setupGin(http.MethodPost, "/api/nodes/"+uuid.New().String()+"/activate", nil)
	c.Params = gin.Params{{Key: "id", Value: uuid.New().String()}}

	h.ActivateNode(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestActivateNode_InvalidUUID(t *testing.T) {
	h := newNodeHandler()
	w, c := setupGin(http.MethodPost, "/api/nodes/bad/activate", nil)
	injectUser(c, uuid.New().String(), "Alice", "alice@test.com")
	c.Params = gin.Params{{Key: "id", Value: "bad"}}

	h.ActivateNode(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ─── DeactivateNode ──────────────────────────────────────────────────────────

func TestDeactivateNode_Unauthorized(t *testing.T) {
	h := newNodeHandler()
	w, c := setupGin(http.MethodPost, "/api/nodes/"+uuid.New().String()+"/deactivate", nil)
	c.Params = gin.Params{{Key: "id", Value: uuid.New().String()}}

	h.DeactivateNode(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestDeactivateNode_InvalidUUID(t *testing.T) {
	h := newNodeHandler()
	w, c := setupGin(http.MethodPost, "/api/nodes/bad/deactivate", nil)
	injectUser(c, uuid.New().String(), "Alice", "alice@test.com")
	c.Params = gin.Params{{Key: "id", Value: "bad"}}

	h.DeactivateNode(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ─── ArchiveNode / UnarchiveNode ─────────────────────────────────────────────

func TestArchiveNode_Unauthorized(t *testing.T) {
	h := newNodeHandler()
	w, c := setupGin(http.MethodPost, "/api/nodes/"+uuid.New().String()+"/archive", nil)
	c.Params = gin.Params{{Key: "id", Value: uuid.New().String()}}

	h.ArchiveNode(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestArchiveNode_InvalidUUID(t *testing.T) {
	h := newNodeHandler()
	w, c := setupGin(http.MethodPost, "/api/nodes/bad/archive", nil)
	injectUser(c, uuid.New().String(), "Alice", "alice@test.com")
	c.Params = gin.Params{{Key: "id", Value: "bad"}}

	h.ArchiveNode(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUnarchiveNode_Unauthorized(t *testing.T) {
	h := newNodeHandler()
	w, c := setupGin(http.MethodPost, "/api/nodes/"+uuid.New().String()+"/unarchive", nil)
	c.Params = gin.Params{{Key: "id", Value: uuid.New().String()}}

	h.UnarchiveNode(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestUnarchiveNode_InvalidUUID(t *testing.T) {
	h := newNodeHandler()
	w, c := setupGin(http.MethodPost, "/api/nodes/bad/unarchive", nil)
	injectUser(c, uuid.New().String(), "Alice", "alice@test.com")
	c.Params = gin.Params{{Key: "id", Value: "bad"}}

	h.UnarchiveNode(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ─── ReorderNodes ────────────────────────────────────────────────────────────

func TestReorderNodes_Unauthorized(t *testing.T) {
	h := newNodeHandler()
	body := mustMarshal(t, map[string]interface{}{
		"orders": []map[string]interface{}{
			{"id": uuid.New().String(), "sort_order": 1},
		},
	})
	w, c := setupGin(http.MethodPost, "/api/nodes/"+uuid.New().String()+"/reorder", body)
	c.Params = gin.Params{{Key: "id", Value: uuid.New().String()}}

	h.ReorderNodes(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestReorderNodes_MissingOrders(t *testing.T) {
	h := newNodeHandler()
	body := mustMarshal(t, map[string]interface{}{})
	w, c := setupGin(http.MethodPost, "/api/nodes/"+uuid.New().String()+"/reorder", body)
	injectUser(c, uuid.New().String(), "Alice", "alice@test.com")
	c.Params = gin.Params{{Key: "id", Value: uuid.New().String()}}

	h.ReorderNodes(c)

	// orders is binding:"required".
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestReorderNodes_EmptyOrders_NoDB(t *testing.T) {
	// Empty orders slice: binding passes (non-nil), no DB calls → 200.
	// Note: The current implementation skips nil orders but doesn't validate
	// the `id` field of each order item at the binding level — so binding
	// passes for an empty slice and the loop is a no-op.
	h := newNodeHandler()
	body := mustMarshal(t, map[string]interface{}{
		"orders": []map[string]interface{}{},
	})
	w, c := setupGin(http.MethodPost, "/api/nodes/"+uuid.New().String()+"/reorder", body)
	injectUser(c, uuid.New().String(), "Alice", "alice@test.com")
	c.Params = gin.Params{{Key: "id", Value: uuid.New().String()}}

	h.ReorderNodes(c)

	// Empty loop → no DB calls → 200 OK.
	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "Nodes reordered", resp["message"])
}

func TestReorderNodes_InvalidOrderUUID_IsSkipped(t *testing.T) {
	// Orders with invalid UUIDs are silently skipped — the loop continues.
	h := newNodeHandler()
	body := mustMarshal(t, map[string]interface{}{
		"orders": []map[string]interface{}{
			{"id": "not-a-uuid", "sort_order": 1},
			{"id": "also-bad", "sort_order": 2},
		},
	})
	w, c := setupGin(http.MethodPost, "/api/nodes/"+uuid.New().String()+"/reorder", body)
	injectUser(c, uuid.New().String(), "Alice", "alice@test.com")
	c.Params = gin.Params{{Key: "id", Value: uuid.New().String()}}

	h.ReorderNodes(c)

	// All items skipped → no DB calls → 200 OK.
	assert.Equal(t, http.StatusOK, w.Code)
}

// ─── Helper: stringToNodeType ────────────────────────────────────────────────

func TestStringToNodeType(t *testing.T) {
	tests := []struct {
		input    string
		expected sqlc.Nodetype
	}{
		{"business", sqlc.NodetypeBUSINESS},
		{"BUSINESS", sqlc.NodetypeBUSINESS},
		{"project", sqlc.NodetypePROJECT},
		{"Project", sqlc.NodetypePROJECT},
		{"learning", sqlc.NodetypeLEARNING},
		{"operational", sqlc.NodetypeOPERATIONAL},
		// Unknown values default to BUSINESS.
		{"", sqlc.NodetypeBUSINESS},
		{"unknown", sqlc.NodetypeBUSINESS},
		{"RANDOM", sqlc.NodetypeBUSINESS},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := stringToNodeType(tt.input)
			assert.Equal(t, tt.expected, got)
		})
	}
}

// ─── Helper: stringToNodeHealth ──────────────────────────────────────────────

func TestStringToNodeHealth(t *testing.T) {
	tests := []struct {
		input    string
		expected sqlc.Nodehealth
	}{
		{"healthy", sqlc.NodehealthHEALTHY},
		{"HEALTHY", sqlc.NodehealthHEALTHY},
		{"needs_attention", sqlc.NodehealthNEEDSATTENTION},
		{"NEEDS_ATTENTION", sqlc.NodehealthNEEDSATTENTION},
		{"critical", sqlc.NodehealthCRITICAL},
		{"Critical", sqlc.NodehealthCRITICAL},
		{"not_started", sqlc.NodehealthNOTSTARTED},
		// Unknown defaults to NOT_STARTED.
		{"", sqlc.NodehealthNOTSTARTED},
		{"unknown", sqlc.NodehealthNOTSTARTED},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := stringToNodeHealth(tt.input)
			assert.Equal(t, tt.expected, got)
		})
	}
}

// ─── Helper: TransformNode ───────────────────────────────────────────────────

func buildTestNode(id uuid.UUID, userID, name string) sqlc.Node {
	isActive := true
	isArchived := false
	sortOrder := int32(1)
	purpose := "Drive growth"
	status := "On track"

	return sqlc.Node{
		ID:            pgtype.UUID{Bytes: id, Valid: true},
		UserID:        userID,
		Name:          name,
		Type:          sqlc.NodetypeBUSINESS,
		Health:        sqlc.NullNodehealth{Nodehealth: sqlc.NodehealthHEALTHY, Valid: true},
		Purpose:       &purpose,
		CurrentStatus: &status,
		IsActive:      &isActive,
		IsArchived:    &isArchived,
		SortOrder:     &sortOrder,
		CreatedAt:     pgtype.Timestamp{Time: time.Now(), Valid: true},
		UpdatedAt:     pgtype.Timestamp{Time: time.Now(), Valid: true},
	}
}

func TestTransformNode_BasicFields(t *testing.T) {
	id := uuid.New()
	n := buildTestNode(id, "user-001", "Alpha")

	resp := TransformNode(n)

	assert.Equal(t, id.String(), resp.ID)
	assert.Equal(t, "user-001", resp.UserID)
	assert.Equal(t, "Alpha", resp.Name)
	assert.Equal(t, "business", resp.Type)
	assert.Equal(t, "healthy", resp.Health)
	assert.True(t, resp.IsActive)
	assert.False(t, resp.IsArchived)
	assert.Equal(t, int32(1), resp.SortOrder)
}

func TestTransformNode_NilHealth_DefaultsToNotStarted(t *testing.T) {
	id := uuid.New()
	n := sqlc.Node{
		ID:        pgtype.UUID{Bytes: id, Valid: true},
		UserID:    "u1",
		Name:      "Node",
		Type:      sqlc.NodetypePROJECT,
		Health:    sqlc.NullNodehealth{Valid: false}, // nil health
		CreatedAt: pgtype.Timestamp{Time: time.Now(), Valid: true},
		UpdatedAt: pgtype.Timestamp{Time: time.Now(), Valid: true},
	}

	resp := TransformNode(n)

	assert.Equal(t, "not_started", resp.Health)
}

func TestTransformNode_NilIsActive_DefaultsFalse(t *testing.T) {
	id := uuid.New()
	n := sqlc.Node{
		ID:         pgtype.UUID{Bytes: id, Valid: true},
		UserID:     "u1",
		Name:       "Node",
		Type:       sqlc.NodetypeBUSINESS,
		IsActive:   nil,
		IsArchived: nil,
		SortOrder:  nil,
		CreatedAt:  pgtype.Timestamp{Time: time.Now(), Valid: true},
		UpdatedAt:  pgtype.Timestamp{Time: time.Now(), Valid: true},
	}

	resp := TransformNode(n)

	assert.False(t, resp.IsActive)
	assert.False(t, resp.IsArchived)
	assert.Equal(t, int32(0), resp.SortOrder)
}

func TestTransformNode_NilSlices_ReturnsEmptySlices(t *testing.T) {
	// Nil JSONB fields should produce empty []string, not nil.
	id := uuid.New()
	n := sqlc.Node{
		ID:              pgtype.UUID{Bytes: id, Valid: true},
		UserID:          "u1",
		Name:            "Node",
		Type:            sqlc.NodetypeBUSINESS,
		ThisWeekFocus:   nil,
		DecisionQueue:   nil,
		DelegationReady: nil,
		CreatedAt:       pgtype.Timestamp{Time: time.Now(), Valid: true},
		UpdatedAt:       pgtype.Timestamp{Time: time.Now(), Valid: true},
	}

	resp := TransformNode(n)

	assert.NotNil(t, resp.ThisWeekFocus)
	assert.NotNil(t, resp.DecisionQueue)
	assert.NotNil(t, resp.DelegationReady)
	assert.Len(t, resp.ThisWeekFocus, 0)
	assert.Len(t, resp.DecisionQueue, 0)
	assert.Len(t, resp.DelegationReady, 0)
}

func TestTransformNode_JSONSlices_Decoded(t *testing.T) {
	focus, _ := json.Marshal([]string{"Close deal", "Hire eng"})
	queue, _ := json.Marshal([]string{"Pick vendor"})
	delegation, _ := json.Marshal([]string{"Send docs"})

	id := uuid.New()
	n := sqlc.Node{
		ID:              pgtype.UUID{Bytes: id, Valid: true},
		UserID:          "u1",
		Name:            "Node",
		Type:            sqlc.NodetypeBUSINESS,
		ThisWeekFocus:   focus,
		DecisionQueue:   queue,
		DelegationReady: delegation,
		CreatedAt:       pgtype.Timestamp{Time: time.Now(), Valid: true},
		UpdatedAt:       pgtype.Timestamp{Time: time.Now(), Valid: true},
	}

	resp := TransformNode(n)

	assert.Equal(t, []string{"Close deal", "Hire eng"}, resp.ThisWeekFocus)
	assert.Equal(t, []string{"Pick vendor"}, resp.DecisionQueue)
	assert.Equal(t, []string{"Send docs"}, resp.DelegationReady)
}

func TestTransformNode_MalformedJSON_FallsBackToEmpty(t *testing.T) {
	id := uuid.New()
	n := sqlc.Node{
		ID:              pgtype.UUID{Bytes: id, Valid: true},
		UserID:          "u1",
		Name:            "Node",
		Type:            sqlc.NodetypeBUSINESS,
		ThisWeekFocus:   []byte("{not an array}"),
		DecisionQueue:   []byte("null"),
		DelegationReady: []byte(""),
		CreatedAt:       pgtype.Timestamp{Time: time.Now(), Valid: true},
		UpdatedAt:       pgtype.Timestamp{Time: time.Now(), Valid: true},
	}

	resp := TransformNode(n)

	// Malformed JSON → Unmarshal fails → slice stays nil → set to [].
	assert.NotNil(t, resp.ThisWeekFocus)
	assert.NotNil(t, resp.DecisionQueue)
	assert.NotNil(t, resp.DelegationReady)
}

// ─── Helper: TransformNodes ──────────────────────────────────────────────────

func TestTransformNodes_EmptySlice(t *testing.T) {
	result := TransformNodes([]sqlc.Node{})
	require.NotNil(t, result)
	assert.Len(t, result, 0)
}

func TestTransformNodes_MultipleNodes(t *testing.T) {
	nodes := make([]sqlc.Node, 3)
	for i := range nodes {
		nodes[i] = buildTestNode(uuid.New(), "u1", "Node")
	}
	result := TransformNodes(nodes)
	assert.Len(t, result, 3)
}

// ─── NodeResponse JSON shape ──────────────────────────────────────────────────

func TestNodeResponse_JSONRoundTrip(t *testing.T) {
	id := uuid.New()
	n := buildTestNode(id, "user-42", "Project Node")
	n.Type = sqlc.NodetypePROJECT

	resp := TransformNode(n)

	data, err := json.Marshal(resp)
	require.NoError(t, err)

	var decoded NodeResponse
	require.NoError(t, json.Unmarshal(data, &decoded))

	assert.Equal(t, id.String(), decoded.ID)
	assert.Equal(t, "project", decoded.Type)
	assert.Equal(t, "healthy", decoded.Health)
	// Arrays must survive round-trip as non-nil.
	assert.NotNil(t, decoded.ThisWeekFocus)
	assert.NotNil(t, decoded.DecisionQueue)
	assert.NotNil(t, decoded.DelegationReady)
}

// ─── Table-driven CreateNode binding ─────────────────────────────────────────

func TestCreateNode_BindingValidation(t *testing.T) {
	// Only test cases that are rejected BEFORE reaching the DB (binding errors).
	// Cases that pass binding reach a nil pool which panics — tested separately.
	cases := []struct {
		name       string
		body       map[string]interface{}
		wantStatus int
	}{
		{
			name:       "missing name → 400",
			body:       map[string]interface{}{"type": "business"},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "missing type → 400",
			body:       map[string]interface{}{"name": "Ops Node"},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "empty object → 400",
			body:       map[string]interface{}{},
			wantStatus: http.StatusBadRequest,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := newNodeHandler()
			body := mustMarshal(t, tc.body)
			w, c := setupGin(http.MethodPost, "/api/nodes", body)
			injectUser(c, uuid.New().String(), "Alice", "alice@test.com")

			h.CreateNode(c)

			assert.Equal(t, tc.wantStatus, w.Code)
		})
	}
}

// TestCreateNode_ValidBody_ReachesDB verifies that a valid body passes
// binding. The nil pool will panic; we catch it and confirm no early rejection.
func TestCreateNode_ValidBody_ReachesDB(t *testing.T) {
	body := mustMarshal(t, map[string]interface{}{
		"name": "Sales Node",
		"type": "business",
	})

	panicked := false
	func() {
		defer func() {
			if r := recover(); r != nil {
				panicked = true
			}
		}()
		h := newNodeHandler()
		w, c := setupGin(http.MethodPost, "/api/nodes", body)
		injectUser(c, uuid.New().String(), "Alice", "alice@test.com")
		h.CreateNode(c)
		// If we reach here with a non-400, binding passed.
		assert.NotEqual(t, http.StatusBadRequest, w.Code)
	}()

	// The nil pool panic proves we got past binding and auth.
	assert.True(t, panicked, "valid body should reach DB call (which panics on nil pool)")
}

// ─── Table-driven UpdateNode binding ─────────────────────────────────────────

func TestUpdateNode_InvalidJSON_Rejected(t *testing.T) {
	h := newNodeHandler()
	w, c := setupGin(http.MethodPatch, "/api/nodes/"+uuid.New().String(), []byte("{broken"))
	injectUser(c, uuid.New().String(), "Alice", "alice@test.com")
	c.Params = gin.Params{{Key: "id", Value: uuid.New().String()}}

	h.UpdateNode(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestUpdateNode_ValidBody_ReachesDB verifies valid update payloads pass
// binding. Nil pool causes a panic which we intercept.
func TestUpdateNode_ValidBody_ReachesDB(t *testing.T) {
	bodies := []map[string]interface{}{
		{},                                       // empty update is valid (all fields optional)
		{"name": "Updated Name"},                 // single field
		{"health": "healthy", "type": "project"}, // multiple fields
	}

	for _, b := range bodies {
		body := mustMarshal(t, b)
		panicked := false
		func() {
			defer func() {
				if r := recover(); r != nil {
					panicked = true
				}
			}()
			h := newNodeHandler()
			w, c := setupGin(http.MethodPatch, "/api/nodes/"+uuid.New().String(), body)
			injectUser(c, uuid.New().String(), "Alice", "alice@test.com")
			c.Params = gin.Params{{Key: "id", Value: uuid.New().String()}}
			h.UpdateNode(c)
			// If we reach here without panic, check it was not binding rejection.
			assert.NotEqual(t, http.StatusBadRequest, w.Code)
		}()
		// Nil-pool panic confirms we got past binding.
		assert.True(t, panicked, "valid body for update should reach DB (nil pool panics)")
	}
}

// ─── Route registration smoke ─────────────────────────────────────────────────

func TestNodeRoutes_RegistrationSmoke(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	api := r.Group("/api")
	h := newNodeHandler()
	noopAuth := func(c *gin.Context) { c.Next() }
	assert.NotPanics(t, func() {
		RegisterNodeRoutes(api, h, noopAuth)
	})
}

// ─── State management edge cases ─────────────────────────────────────────────

// handlerReachesDB runs a handler func and returns true if a panic occurred
// (meaning the handler got past auth+binding and tried to hit the nil DB pool).
func handlerReachesDB(fn func()) (panicked bool) {
	defer func() {
		if r := recover(); r != nil {
			panicked = true
		}
	}()
	fn()
	return false
}

// TestActivateNode_ValidUUID_NoDB verifies that a valid UUID + auth user
// passes all pre-DB checks. The nil pool panics when the DB is reached.
func TestActivateNode_ValidUUID_NoDB(t *testing.T) {
	nodeID := uuid.New()
	panicked := handlerReachesDB(func() {
		h := newNodeHandler()
		_, c := setupGin(http.MethodPost, "/api/nodes/"+nodeID.String()+"/activate", nil)
		injectUser(c, uuid.New().String(), "Alice", "alice@test.com")
		c.Params = gin.Params{{Key: "id", Value: nodeID.String()}}
		h.ActivateNode(c)
	})
	assert.True(t, panicked, "valid UUID+auth should reach DB (nil pool panics)")
}

// TestDeactivateNode_ValidUUID_NoDB same pattern.
func TestDeactivateNode_ValidUUID_NoDB(t *testing.T) {
	nodeID := uuid.New()
	panicked := handlerReachesDB(func() {
		h := newNodeHandler()
		_, c := setupGin(http.MethodPost, "/api/nodes/"+nodeID.String()+"/deactivate", nil)
		injectUser(c, uuid.New().String(), "Alice", "alice@test.com")
		c.Params = gin.Params{{Key: "id", Value: nodeID.String()}}
		h.DeactivateNode(c)
	})
	assert.True(t, panicked, "valid UUID+auth should reach DB (nil pool panics)")
}

// TestArchiveNode_ValidUUID_NoDB verifies auth and UUID checks pass.
func TestArchiveNode_ValidUUID_NoDB(t *testing.T) {
	nodeID := uuid.New()
	panicked := handlerReachesDB(func() {
		h := newNodeHandler()
		_, c := setupGin(http.MethodPost, "/api/nodes/"+nodeID.String()+"/archive", nil)
		injectUser(c, uuid.New().String(), "Alice", "alice@test.com")
		c.Params = gin.Params{{Key: "id", Value: nodeID.String()}}
		h.ArchiveNode(c)
	})
	assert.True(t, panicked, "valid UUID+auth should reach DB (nil pool panics)")
}

// TestUnarchiveNode_ValidUUID_NoDB verifies auth and UUID checks pass.
func TestUnarchiveNode_ValidUUID_NoDB(t *testing.T) {
	nodeID := uuid.New()
	panicked := handlerReachesDB(func() {
		h := newNodeHandler()
		_, c := setupGin(http.MethodPost, "/api/nodes/"+nodeID.String()+"/unarchive", nil)
		injectUser(c, uuid.New().String(), "Alice", "alice@test.com")
		c.Params = gin.Params{{Key: "id", Value: nodeID.String()}}
		h.UnarchiveNode(c)
	})
	assert.True(t, panicked, "valid UUID+auth should reach DB (nil pool panics)")
}
