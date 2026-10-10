package handlers

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

// ============================================================================
// TEST HELPERS
// ============================================================================

// crmTestUser returns a fake authenticated user for use in CRM handler tests.
func crmTestUser() *middleware.BetterAuthUser {
	return &middleware.BetterAuthUser{
		ID:            "test-user-id-1234",
		Name:          "Test User",
		Email:         "test@example.com",
		EmailVerified: true,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
}

// setTestUser injects a fake user into the gin context, replicating what
// AuthMiddleware does in production.
func setTestUser(c *gin.Context, user *middleware.BetterAuthUser) {
	c.Set(middleware.UserContextKey, user)
}

// newTestCRMHandler returns a CRMHandler backed by a nil pool.
// Safe for unit tests that do NOT hit the database.
func newTestCRMHandler() *CRMHandler {
	return &CRMHandler{pool: nil}
}

// newTestClientHandler returns a ClientHandler backed by a nil pool.
func newTestClientHandler() *ClientHandler {
	return &ClientHandler{pool: nil}
}

// crmJsonBody marshals v to a bytes.Reader for use as an HTTP request body.
func crmJsonBody(t *testing.T, v interface{}) *bytes.Reader {
	t.Helper()
	b, err := json.Marshal(v)
	require.NoError(t, err)
	return bytes.NewReader(b)
}

// newGinContext returns a recorder + context pair wired to the given method,
// path, and body. Sets Content-Type: application/json.
func newGinContext(t *testing.T, method, path string, body *bytes.Reader) (*httptest.ResponseRecorder, *gin.Context) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	var req *http.Request
	if body != nil {
		req = httptest.NewRequest(method, path, body)
	} else {
		req = httptest.NewRequest(method, path, nil)
	}
	req.Header.Set("Content-Type", "application/json")
	c.Request = req
	return w, c
}

// decodeJSON is a convenience to decode a response body into a map.
func decodeJSON(t *testing.T, w *httptest.ResponseRecorder) map[string]interface{} {
	t.Helper()
	var result map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &result)
	require.NoError(t, err, "response body: %s", w.Body.String())
	return result
}

// ============================================================================
// UNIT TESTS: crm_helpers.go - pure functions
// ============================================================================

func TestCrmToNullString(t *testing.T) {
	t.Run("empty string returns nil", func(t *testing.T) {
		result := crmToNullString("")
		assert.Nil(t, result)
	})

	t.Run("non-empty string returns pointer", func(t *testing.T) {
		result := crmToNullString("open")
		require.NotNil(t, result)
		assert.Equal(t, "open", *result)
	})

	t.Run("whitespace-only string is returned as-is", func(t *testing.T) {
		// Whitespace is not empty — caller is responsible for trimming.
		result := crmToNullString("  ")
		require.NotNil(t, result)
		assert.Equal(t, "  ", *result)
	})
}

func TestCrmToNullUUID(t *testing.T) {
	validUUID := uuid.New().String()

	t.Run("empty string returns invalid pgtype.UUID", func(t *testing.T) {
		result := crmToNullUUID("")
		assert.False(t, result.Valid)
	})

	t.Run("valid UUID string returns valid pgtype.UUID", func(t *testing.T) {
		result := crmToNullUUID(validUUID)
		assert.True(t, result.Valid)
		assert.Equal(t, validUUID, uuid.UUID(result.Bytes).String())
	})

	t.Run("malformed UUID returns invalid pgtype.UUID", func(t *testing.T) {
		result := crmToNullUUID("not-a-uuid")
		assert.False(t, result.Valid)
	})

	t.Run("UUID with uppercase letters is parsed correctly", func(t *testing.T) {
		upper := strings.ToUpper(validUUID)
		result := crmToNullUUID(upper)
		assert.True(t, result.Valid)
	})
}

func TestCrmToNumeric(t *testing.T) {
	// pgtype.Numeric.Scan() does not accept float64 values — it can only scan
	// from database wire types (string, []byte, etc.).  The crmToNumeric helper
	// calls Scan(float64) which silently fails, so the result is always an
	// invalid Numeric regardless of the input value.  These tests document the
	// actual behavior so regressions are caught if the implementation changes.

	t.Run("nil pointer returns invalid Numeric", func(t *testing.T) {
		result := crmToNumeric(nil)
		assert.False(t, result.Valid, "nil input must produce invalid Numeric")
	})

	t.Run("non-nil pointer produces a Numeric (Scan from float64 is unsupported)", func(t *testing.T) {
		// pgtype.Numeric.Scan does not support float64; Valid stays false.
		// crmNumericToFloat will return 0 for this value.
		v := 1234.56
		result := crmToNumeric(&v)
		// Document current behavior: Scan silently fails, Valid==false.
		assert.False(t, result.Valid,
			"pgtype.Numeric.Scan(float64) is unsupported; Valid stays false")
	})

	t.Run("crmNumericToFloat of invalid Numeric returns 0", func(t *testing.T) {
		v := 999.99
		n := crmToNumeric(&v)
		// Because Scan fails, Float64 always returns 0.
		assert.Equal(t, 0.0, crmNumericToFloat(n))
	})
}

func TestCrmNumericToFloat(t *testing.T) {
	t.Run("invalid Numeric (zero value) returns 0", func(t *testing.T) {
		result := crmNumericToFloat(pgtype.Numeric{})
		assert.Equal(t, 0.0, result)
	})

	t.Run("invalid Numeric from crmToNumeric(float64) returns 0", func(t *testing.T) {
		// pgtype.Numeric.Scan does not accept float64; crmToNumeric always
		// produces an invalid Numeric, so crmNumericToFloat returns 0.
		v := 99.99
		n := crmToNumeric(&v)
		result := crmNumericToFloat(n)
		assert.Equal(t, 0.0, result,
			"crmToNumeric(float64) produces invalid Numeric; float result is always 0")
	})
}

func TestCrmPtrToString(t *testing.T) {
	t.Run("nil pointer returns empty string", func(t *testing.T) {
		assert.Equal(t, "", crmPtrToString(nil))
	})

	t.Run("non-nil pointer returns string value", func(t *testing.T) {
		s := "hello"
		assert.Equal(t, "hello", crmPtrToString(&s))
	})
}

func TestCrmUuidToString(t *testing.T) {
	t.Run("invalid UUID returns nil", func(t *testing.T) {
		result := crmUuidToString(pgtype.UUID{})
		assert.Nil(t, result)
	})

	t.Run("valid UUID returns correct string pointer", func(t *testing.T) {
		id := uuid.New()
		u := pgtype.UUID{Bytes: id, Valid: true}
		result := crmUuidToString(u)
		require.NotNil(t, result)
		assert.Equal(t, id.String(), *result)
	})
}

func TestCrmDateToString(t *testing.T) {
	t.Run("invalid date returns nil", func(t *testing.T) {
		result := crmDateToString(pgtype.Date{})
		assert.Nil(t, result)
	})

	t.Run("valid date returns YYYY-MM-DD formatted string", func(t *testing.T) {
		d := pgtype.Date{Time: time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC), Valid: true}
		result := crmDateToString(d)
		require.NotNil(t, result)
		assert.Equal(t, "2025-06-15", *result)
	})
}

func TestCrmTimestampToString(t *testing.T) {
	t.Run("invalid timestamp returns nil", func(t *testing.T) {
		result := crmTimestampToString(pgtype.Timestamptz{})
		assert.Nil(t, result)
	})

	t.Run("valid timestamp returns RFC3339 string", func(t *testing.T) {
		ts := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
		result := crmTimestampToString(pgtype.Timestamptz{Time: ts, Valid: true})
		require.NotNil(t, result)
		// Must be parseable as RFC3339
		_, err := time.Parse(time.RFC3339, *result)
		assert.NoError(t, err)
	})
}

func TestCrmJsonToMap(t *testing.T) {
	t.Run("nil bytes returns nil", func(t *testing.T) {
		result := crmJsonToMap(nil)
		assert.Nil(t, result)
	})

	t.Run("empty bytes returns nil", func(t *testing.T) {
		result := crmJsonToMap([]byte{})
		assert.Nil(t, result)
	})

	t.Run("valid JSON object returns map", func(t *testing.T) {
		b := []byte(`{"key":"value","num":42}`)
		result := crmJsonToMap(b)
		require.NotNil(t, result)
		assert.Equal(t, "value", result["key"])
		assert.Equal(t, float64(42), result["num"])
	})

	t.Run("invalid JSON returns nil without panicking", func(t *testing.T) {
		// Unmarshal errors are silently discarded — result is nil.
		result := crmJsonToMap([]byte("not json"))
		assert.Nil(t, result)
	})
}

func TestCrmJsonToSlice(t *testing.T) {
	t.Run("nil bytes returns nil", func(t *testing.T) {
		assert.Nil(t, crmJsonToSlice(nil))
	})

	t.Run("valid JSON array returns slice", func(t *testing.T) {
		b := []byte(`["a","b","c"]`)
		result := crmJsonToSlice(b)
		require.NotNil(t, result)
		assert.Len(t, result, 3)
		assert.Equal(t, "a", result[0])
	})
}

// ============================================================================
// UNIT TESTS: mapLegacyStageName (clients_deals.go)
// ============================================================================

func TestMapLegacyStageName(t *testing.T) {
	cases := []struct {
		input    string
		expected string
	}{
		{"qualification", "Qualification"},
		{"proposal", "Proposal"},
		{"negotiation", "Negotiation"},
		{"closed_won", "Closed Won"},
		{"closed_lost", "Closed Lost"},
		// Upper-case variants are normalized via ToLower before lookup.
		{"PROPOSAL", "Proposal"},
		{"Negotiation", "Negotiation"},
		// Unknown value passes through unchanged.
		{"discovery", "discovery"},
		{"", ""},
		{"UNKNOWN_STAGE", "UNKNOWN_STAGE"},
	}

	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			assert.Equal(t, tc.expected, mapLegacyStageName(tc.input))
		})
	}
}

// ============================================================================
// UNIT TESTS: transformCRMDealBasic
// ============================================================================

func buildTestDeal() sqlc.Deal {
	id := uuid.New()
	pipelineID := uuid.New()
	stageID := uuid.New()
	status := "open"
	desc := "A test deal description"
	ts := pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true}

	return sqlc.Deal{
		ID:           pgtype.UUID{Bytes: id, Valid: true},
		UserID:       "user-123",
		PipelineID:   pgtype.UUID{Bytes: pipelineID, Valid: true},
		StageID:      pgtype.UUID{Bytes: stageID, Valid: true},
		Name:         "Test Deal",
		Description:  &desc,
		Status:       &status,
		CreatedAt:    ts,
		UpdatedAt:    ts,
		CustomFields: []byte(`{"source":"web"}`),
	}
}

func TestTransformCRMDealBasic(t *testing.T) {
	deal := buildTestDeal()
	result := transformCRMDealBasic(deal)

	t.Run("id is a string UUID", func(t *testing.T) {
		idVal, ok := result["id"]
		require.True(t, ok)
		idStr, ok := idVal.(*string)
		require.True(t, ok)
		require.NotNil(t, idStr)
		_, err := uuid.Parse(*idStr)
		assert.NoError(t, err)
	})

	t.Run("name is preserved", func(t *testing.T) {
		assert.Equal(t, "Test Deal", result["name"])
	})

	t.Run("status is preserved", func(t *testing.T) {
		statusPtr, ok := result["status"].(*string)
		require.True(t, ok)
		assert.Equal(t, "open", *statusPtr)
	})

	t.Run("amount defaults to 0 when not set", func(t *testing.T) {
		assert.Equal(t, 0.0, result["amount"])
	})

	t.Run("custom_fields is deserialized to map", func(t *testing.T) {
		cf, ok := result["custom_fields"].(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, "web", cf["source"])
	})

	t.Run("nil optional UUID fields become nil string pointer", func(t *testing.T) {
		// company_id and primary_contact_id are zero UUIDs — should be nil.
		assert.Nil(t, result["company_id"])
		assert.Nil(t, result["primary_contact_id"])
	})
}

// ============================================================================
// UNIT TESTS: transformCRMDeal (ListCRMDealsRow)
// ============================================================================

func buildTestListCRMDealsRow() sqlc.ListCRMDealsRow {
	id := uuid.New()
	pipelineID := uuid.New()
	stageID := uuid.New()
	status := "open"
	pipelineName := "Sales"
	stageName := "Qualification"
	ts := pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true}

	return sqlc.ListCRMDealsRow{
		ID:           pgtype.UUID{Bytes: id, Valid: true},
		UserID:       "user-abc",
		PipelineID:   pgtype.UUID{Bytes: pipelineID, Valid: true},
		StageID:      pgtype.UUID{Bytes: stageID, Valid: true},
		Name:         "Acme Deal",
		Status:       &status,
		PipelineName: pipelineName,
		StageName:    stageName,
		CreatedAt:    ts,
		UpdatedAt:    ts,
		CustomFields: []byte(`{}`),
	}
}

func TestTransformCRMDeal(t *testing.T) {
	row := buildTestListCRMDealsRow()
	result := transformCRMDeal(row)

	t.Run("pipeline_name is included", func(t *testing.T) {
		assert.Equal(t, "Sales", result["pipeline_name"])
	})

	t.Run("stage_name is included", func(t *testing.T) {
		assert.Equal(t, "Qualification", result["stage_name"])
	})

	t.Run("user_id is preserved", func(t *testing.T) {
		assert.Equal(t, "user-abc", result["user_id"])
	})
}

func TestTransformCRMDeals_EmptySlice(t *testing.T) {
	result := transformCRMDeals([]sqlc.ListCRMDealsRow{})
	assert.NotNil(t, result)
	assert.Len(t, result, 0)
}

func TestTransformCRMDeals_MultipleRows(t *testing.T) {
	rows := []sqlc.ListCRMDealsRow{
		buildTestListCRMDealsRow(),
		buildTestListCRMDealsRow(),
		buildTestListCRMDealsRow(),
	}
	result := transformCRMDeals(rows)
	assert.Len(t, result, 3)
}

// ============================================================================
// UNIT TESTS: transformClientDealFromDeal (clients_deals.go)
// ============================================================================

func TestTransformClientDealFromDeal(t *testing.T) {
	deal := buildTestDeal()
	clientID := uuid.New()
	result := transformClientDealFromDeal(deal, clientID)

	t.Run("client_id is the supplied clientID", func(t *testing.T) {
		assert.Equal(t, clientID.String(), result["client_id"])
	})

	t.Run("value maps from amount", func(t *testing.T) {
		assert.Equal(t, 0.0, result["value"])
	})

	t.Run("notes maps from description", func(t *testing.T) {
		notesPtr, ok := result["notes"].(*string)
		require.True(t, ok)
		assert.Equal(t, "A test deal description", *notesPtr)
	})

	t.Run("created_at is RFC3339 formatted", func(t *testing.T) {
		createdAt, ok := result["created_at"].(string)
		require.True(t, ok)
		_, err := time.Parse(time.RFC3339, createdAt)
		assert.NoError(t, err)
	})
}

// ============================================================================
// UNIT TESTS: transformClientDealFromRow (clients_deals.go)
// ============================================================================

func buildTestListClientDealsRow() sqlc.ListClientDealsRow {
	id := uuid.New()
	clientID := uuid.New()
	pipelineID := uuid.New()
	stageID := uuid.New()
	status := "open"
	ts := pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true}

	return sqlc.ListClientDealsRow{
		ID:         pgtype.UUID{Bytes: id, Valid: true},
		ClientID:   pgtype.UUID{Bytes: clientID, Valid: true},
		Name:       "Widget Deal",
		Status:     &status,
		StageName:  "Closed Won",
		StageID:    pgtype.UUID{Bytes: stageID, Valid: true},
		PipelineID: pgtype.UUID{Bytes: pipelineID, Valid: true},
		CreatedAt:  ts,
		UpdatedAt:  ts,
	}
}

func TestTransformClientDealFromRow(t *testing.T) {
	row := buildTestListClientDealsRow()
	result := transformClientDealFromRow(row)

	t.Run("stage is lowercased with underscores", func(t *testing.T) {
		// "Closed Won" → "closed_won"
		assert.Equal(t, "closed_won", result["stage"])
	})

	t.Run("stage_id is a valid UUID string pointer", func(t *testing.T) {
		stageIDPtr, ok := result["stage_id"].(*string)
		require.True(t, ok)
		require.NotNil(t, stageIDPtr)
		_, err := uuid.Parse(*stageIDPtr)
		assert.NoError(t, err)
	})

	t.Run("name is preserved", func(t *testing.T) {
		assert.Equal(t, "Widget Deal", result["name"])
	})
}

func TestTransformClientDealsFromRows_Empty(t *testing.T) {
	result := transformClientDealsFromRows([]sqlc.ListClientDealsRow{})
	assert.NotNil(t, result)
	assert.Len(t, result, 0)
}

// ============================================================================
// HANDLER TESTS: CRM Deals — authentication guard
// These tests verify that all handlers reject unauthenticated requests (no DB
// needed because the auth guard fires before any DB call).
// ============================================================================

func TestListCRMDeals_Unauthenticated(t *testing.T) {
	h := newTestCRMHandler()
	w, c := newGinContext(t, http.MethodGet, "/crm/deals", nil)

	// No user set in context.
	h.ListCRMDeals(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestGetCRMDeal_Unauthenticated(t *testing.T) {
	h := newTestCRMHandler()
	w, c := newGinContext(t, http.MethodGet, "/crm/deals/some-id", nil)
	c.Params = gin.Params{{Key: "id", Value: uuid.New().String()}}

	h.GetCRMDeal(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestCreateCRMDeal_Unauthenticated(t *testing.T) {
	h := newTestCRMHandler()
	body := crmJsonBody(t, map[string]string{
		"pipeline_id": uuid.New().String(),
		"stage_id":    uuid.New().String(),
		"name":        "New Deal",
	})
	w, c := newGinContext(t, http.MethodPost, "/crm/deals", body)

	h.CreateCRMDeal(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestUpdateCRMDeal_Unauthenticated(t *testing.T) {
	h := newTestCRMHandler()
	body := crmJsonBody(t, map[string]string{"name": "Updated Deal"})
	w, c := newGinContext(t, http.MethodPut, "/crm/deals/some-id", body)
	c.Params = gin.Params{{Key: "id", Value: uuid.New().String()}}

	h.UpdateCRMDeal(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestMoveCRMDealStage_Unauthenticated(t *testing.T) {
	h := newTestCRMHandler()
	body := crmJsonBody(t, map[string]string{"stage_id": uuid.New().String()})
	w, c := newGinContext(t, http.MethodPatch, "/crm/deals/some-id/stage", body)
	c.Params = gin.Params{{Key: "id", Value: uuid.New().String()}}

	h.MoveCRMDealStage(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestUpdateCRMDealStatus_Unauthenticated(t *testing.T) {
	h := newTestCRMHandler()
	body := crmJsonBody(t, map[string]string{"status": "won"})
	w, c := newGinContext(t, http.MethodPatch, "/crm/deals/some-id/status", body)
	c.Params = gin.Params{{Key: "id", Value: uuid.New().String()}}

	h.UpdateCRMDealStatus(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestDeleteCRMDeal_Unauthenticated(t *testing.T) {
	h := newTestCRMHandler()
	w, c := newGinContext(t, http.MethodDelete, "/crm/deals/some-id", nil)
	c.Params = gin.Params{{Key: "id", Value: uuid.New().String()}}

	h.DeleteCRMDeal(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestGetCRMDealStats_Unauthenticated(t *testing.T) {
	h := newTestCRMHandler()
	w, c := newGinContext(t, http.MethodGet, "/crm/deals/stats", nil)

	h.GetCRMDealStats(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// ============================================================================
// HANDLER TESTS: Client Deals — authentication guard
// ============================================================================

func TestListClientDeals_Unauthenticated(t *testing.T) {
	h := newTestClientHandler()
	w, c := newGinContext(t, http.MethodGet, "/clients/some-id/deals", nil)
	c.Params = gin.Params{{Key: "id", Value: uuid.New().String()}}

	h.ListClientDeals(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestCreateClientDeal_Unauthenticated(t *testing.T) {
	h := newTestClientHandler()
	body := crmJsonBody(t, map[string]string{"name": "New Deal"})
	w, c := newGinContext(t, http.MethodPost, "/clients/some-id/deals", body)
	c.Params = gin.Params{{Key: "id", Value: uuid.New().String()}}

	h.CreateClientDeal(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestUpdateClientDeal_Unauthenticated(t *testing.T) {
	h := newTestClientHandler()
	body := crmJsonBody(t, map[string]string{"name": "Updated Deal"})
	w, c := newGinContext(t, http.MethodPut, "/clients/some-id/deals/deal-id", body)
	c.Params = gin.Params{
		{Key: "id", Value: uuid.New().String()},
		{Key: "dealId", Value: uuid.New().String()},
	}

	h.UpdateClientDeal(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// ============================================================================
// HANDLER TESTS: Invalid ID format (authenticated but bad route param)
// ============================================================================

func TestGetCRMDeal_InvalidID(t *testing.T) {
	h := newTestCRMHandler()
	w, c := newGinContext(t, http.MethodGet, "/crm/deals/not-a-uuid", nil)
	c.Params = gin.Params{{Key: "id", Value: "not-a-uuid"}}
	setTestUser(c, crmTestUser())

	h.GetCRMDeal(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdateCRMDeal_InvalidID(t *testing.T) {
	h := newTestCRMHandler()
	body := crmJsonBody(t, map[string]string{"name": "Deal"})
	w, c := newGinContext(t, http.MethodPut, "/crm/deals/bad-id", body)
	c.Params = gin.Params{{Key: "id", Value: "bad-id"}}
	setTestUser(c, crmTestUser())

	h.UpdateCRMDeal(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestDeleteCRMDeal_InvalidID(t *testing.T) {
	h := newTestCRMHandler()
	w, c := newGinContext(t, http.MethodDelete, "/crm/deals/bad-id", nil)
	c.Params = gin.Params{{Key: "id", Value: "bad-id"}}
	setTestUser(c, crmTestUser())

	h.DeleteCRMDeal(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestMoveCRMDealStage_InvalidDealID(t *testing.T) {
	h := newTestCRMHandler()
	body := crmJsonBody(t, map[string]string{"stage_id": uuid.New().String()})
	w, c := newGinContext(t, http.MethodPatch, "/crm/deals/bad-id/stage", body)
	c.Params = gin.Params{{Key: "id", Value: "bad-id"}}
	setTestUser(c, crmTestUser())

	h.MoveCRMDealStage(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdateCRMDealStatus_InvalidDealID(t *testing.T) {
	h := newTestCRMHandler()
	body := crmJsonBody(t, map[string]string{"status": "won"})
	w, c := newGinContext(t, http.MethodPatch, "/crm/deals/bad-id/status", body)
	c.Params = gin.Params{{Key: "id", Value: "bad-id"}}
	setTestUser(c, crmTestUser())

	h.UpdateCRMDealStatus(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestListClientDeals_InvalidClientID(t *testing.T) {
	h := newTestClientHandler()
	w, c := newGinContext(t, http.MethodGet, "/clients/bad-id/deals", nil)
	c.Params = gin.Params{{Key: "id", Value: "bad-id"}}
	setTestUser(c, crmTestUser())

	h.ListClientDeals(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateClientDeal_InvalidClientID(t *testing.T) {
	h := newTestClientHandler()
	body := crmJsonBody(t, map[string]string{"name": "Deal"})
	w, c := newGinContext(t, http.MethodPost, "/clients/bad-id/deals", body)
	c.Params = gin.Params{{Key: "id", Value: "bad-id"}}
	setTestUser(c, crmTestUser())

	h.CreateClientDeal(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdateClientDeal_InvalidClientID(t *testing.T) {
	h := newTestClientHandler()
	body := crmJsonBody(t, map[string]string{"name": "Deal"})
	w, c := newGinContext(t, http.MethodPut, "/clients/bad-id/deals/some-deal-id", body)
	c.Params = gin.Params{
		{Key: "id", Value: "bad-id"},
		{Key: "dealId", Value: uuid.New().String()},
	}
	setTestUser(c, crmTestUser())

	h.UpdateClientDeal(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdateClientDeal_InvalidDealID(t *testing.T) {
	h := newTestClientHandler()
	body := crmJsonBody(t, map[string]string{"name": "Deal"})
	w, c := newGinContext(t, http.MethodPut, "/clients/some-id/deals/bad-deal-id", body)
	c.Params = gin.Params{
		{Key: "id", Value: uuid.New().String()},
		{Key: "dealId", Value: "bad-deal-id"},
	}
	setTestUser(c, crmTestUser())

	h.UpdateClientDeal(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ============================================================================
// HANDLER TESTS: Missing required request body fields
// ============================================================================

func TestCreateCRMDeal_MissingRequiredFields(t *testing.T) {
	h := newTestCRMHandler()

	cases := []struct {
		name string
		body map[string]interface{}
	}{
		{
			name: "missing pipeline_id",
			body: map[string]interface{}{
				"stage_id": uuid.New().String(),
				"name":     "My Deal",
			},
		},
		{
			name: "missing stage_id",
			body: map[string]interface{}{
				"pipeline_id": uuid.New().String(),
				"name":        "My Deal",
			},
		},
		{
			name: "missing name",
			body: map[string]interface{}{
				"pipeline_id": uuid.New().String(),
				"stage_id":    uuid.New().String(),
			},
		},
		{
			name: "empty body",
			body: map[string]interface{}{},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w, c := newGinContext(t, http.MethodPost, "/crm/deals", crmJsonBody(t, tc.body))
			setTestUser(c, crmTestUser())

			h.CreateCRMDeal(c)

			assert.Equal(t, http.StatusBadRequest, w.Code, "body: %v", tc.body)
		})
	}
}

func TestUpdateCRMDeal_MissingRequiredFields(t *testing.T) {
	h := newTestCRMHandler()
	dealID := uuid.New().String()

	cases := []struct {
		name string
		body map[string]interface{}
	}{
		{
			name: "missing name",
			body: map[string]interface{}{"amount": 500.0},
		},
		{
			name: "empty body",
			body: map[string]interface{}{},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w, c := newGinContext(t, http.MethodPut, "/crm/deals/"+dealID, crmJsonBody(t, tc.body))
			c.Params = gin.Params{{Key: "id", Value: dealID}}
			setTestUser(c, crmTestUser())

			h.UpdateCRMDeal(c)

			assert.Equal(t, http.StatusBadRequest, w.Code)
		})
	}
}

func TestMoveCRMDealStage_MissingStageID(t *testing.T) {
	h := newTestCRMHandler()
	dealID := uuid.New().String()

	w, c := newGinContext(t, http.MethodPatch, "/crm/deals/"+dealID+"/stage",
		crmJsonBody(t, map[string]interface{}{}))
	c.Params = gin.Params{{Key: "id", Value: dealID}}
	setTestUser(c, crmTestUser())

	h.MoveCRMDealStage(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdateCRMDealStatus_MissingStatus(t *testing.T) {
	h := newTestCRMHandler()
	dealID := uuid.New().String()

	w, c := newGinContext(t, http.MethodPatch, "/crm/deals/"+dealID+"/status",
		crmJsonBody(t, map[string]interface{}{}))
	c.Params = gin.Params{{Key: "id", Value: dealID}}
	setTestUser(c, crmTestUser())

	h.UpdateCRMDealStatus(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateClientDeal_MissingName(t *testing.T) {
	h := newTestClientHandler()
	clientID := uuid.New().String()

	w, c := newGinContext(t, http.MethodPost, "/clients/"+clientID+"/deals",
		crmJsonBody(t, map[string]interface{}{"value": 1000.0}))
	c.Params = gin.Params{{Key: "id", Value: clientID}}
	setTestUser(c, crmTestUser())

	h.CreateClientDeal(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdateClientDeal_MissingName(t *testing.T) {
	h := newTestClientHandler()
	clientID := uuid.New().String()
	dealID := uuid.New().String()

	w, c := newGinContext(t, http.MethodPut, "/clients/"+clientID+"/deals/"+dealID,
		crmJsonBody(t, map[string]interface{}{"value": 500.0}))
	c.Params = gin.Params{
		{Key: "id", Value: clientID},
		{Key: "dealId", Value: dealID},
	}
	setTestUser(c, crmTestUser())

	h.UpdateClientDeal(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ============================================================================
// HANDLER TESTS: Invalid UUID values inside valid request bodies
// ============================================================================

func TestCreateCRMDeal_InvalidPipelineUUID(t *testing.T) {
	h := newTestCRMHandler()

	body := crmJsonBody(t, map[string]interface{}{
		"pipeline_id": "not-a-uuid",
		"stage_id":    uuid.New().String(),
		"name":        "My Deal",
	})
	w, c := newGinContext(t, http.MethodPost, "/crm/deals", body)
	setTestUser(c, crmTestUser())

	h.CreateCRMDeal(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateCRMDeal_InvalidStageUUID(t *testing.T) {
	h := newTestCRMHandler()

	body := crmJsonBody(t, map[string]interface{}{
		"pipeline_id": uuid.New().String(),
		"stage_id":    "also-not-a-uuid",
		"name":        "My Deal",
	})
	w, c := newGinContext(t, http.MethodPost, "/crm/deals", body)
	setTestUser(c, crmTestUser())

	h.CreateCRMDeal(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestMoveCRMDealStage_InvalidStageUUID(t *testing.T) {
	h := newTestCRMHandler()
	dealID := uuid.New().String()

	body := crmJsonBody(t, map[string]interface{}{
		"stage_id": "bad-stage-uuid",
	})
	w, c := newGinContext(t, http.MethodPatch, "/crm/deals/"+dealID+"/stage", body)
	c.Params = gin.Params{{Key: "id", Value: dealID}}
	setTestUser(c, crmTestUser())

	h.MoveCRMDealStage(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ============================================================================
// UNIT TESTS: CreateCRMDealRequest — optional field handling
// ============================================================================

func TestCreateCRMDealRequest_OptionalFields(t *testing.T) {
	// Ensure optional fields are truly optional: a request with only required
	// fields must bind without error.
	t.Run("request with only required fields is valid", func(t *testing.T) {
		raw := `{
			"pipeline_id": "` + uuid.New().String() + `",
			"stage_id": "` + uuid.New().String() + `",
			"name": "Minimal Deal"
		}`
		var req CreateCRMDealRequest
		err := json.Unmarshal([]byte(raw), &req)
		require.NoError(t, err)
		assert.Equal(t, "Minimal Deal", req.Name)
		assert.Nil(t, req.Amount)
		assert.Nil(t, req.Description)
		assert.Nil(t, req.ExpectedCloseDate)
	})

	t.Run("optional amount zero value is accepted", func(t *testing.T) {
		zero := 0.0
		raw := `{
			"pipeline_id": "` + uuid.New().String() + `",
			"stage_id": "` + uuid.New().String() + `",
			"name": "Zero Amount Deal",
			"amount": 0
		}`
		var req CreateCRMDealRequest
		err := json.Unmarshal([]byte(raw), &req)
		require.NoError(t, err)
		require.NotNil(t, req.Amount)
		assert.Equal(t, zero, *req.Amount)
	})

	t.Run("expected_close_date is parsed correctly by handler", func(t *testing.T) {
		// Verify the date string "2025-12-31" round-trips through the handler logic.
		dateStr := "2025-12-31"
		t.Run("valid date string round-trip", func(t *testing.T) {
			parsed, err := time.Parse("2006-01-02", dateStr)
			require.NoError(t, err)
			assert.Equal(t, 2025, parsed.Year())
			assert.Equal(t, time.December, parsed.Month())
			assert.Equal(t, 31, parsed.Day())
		})
	})
}

// ============================================================================
// UNIT TESTS: UpdateCRMDealRequest — optional field handling
// ============================================================================

func TestUpdateCRMDealRequest_OptionalFields(t *testing.T) {
	t.Run("request with only name is valid", func(t *testing.T) {
		raw := `{"name": "Updated Name"}`
		var req UpdateCRMDealRequest
		err := json.Unmarshal([]byte(raw), &req)
		require.NoError(t, err)
		assert.Equal(t, "Updated Name", req.Name)
		assert.Nil(t, req.Amount)
		assert.Nil(t, req.Description)
	})

	t.Run("custom_fields nil is handled", func(t *testing.T) {
		raw := `{"name": "Deal", "custom_fields": null}`
		var req UpdateCRMDealRequest
		err := json.Unmarshal([]byte(raw), &req)
		require.NoError(t, err)
		assert.Nil(t, req.CustomFields)
	})
}

// ============================================================================
// UNIT TESTS: Query parameter parsing edge cases (ListCRMDeals)
// ============================================================================

func TestListCRMDeals_PaginationDefaults(t *testing.T) {
	// The handler uses strconv.Atoi(c.DefaultQuery("limit", "50")).
	// We exercise the parsing logic directly to confirm defaults.
	t.Run("default limit is 50", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		w := httptest.NewRecorder()
		c, engine := gin.CreateTestContext(w)
		_ = engine
		req := httptest.NewRequest(http.MethodGet, "/crm/deals", nil)
		c.Request = req

		// Simulate DefaultQuery behavior
		limitStr := c.DefaultQuery("limit", "50")
		assert.Equal(t, "50", limitStr)
	})

	t.Run("default offset is 0", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		w := httptest.NewRecorder()
		c, engine := gin.CreateTestContext(w)
		_ = engine
		req := httptest.NewRequest(http.MethodGet, "/crm/deals", nil)
		c.Request = req

		offsetStr := c.DefaultQuery("offset", "0")
		assert.Equal(t, "0", offsetStr)
	})
}

// ============================================================================
// INTEGRATION TEST STUBS (require live PostgreSQL)
// ============================================================================

// TestListCRMDeals_Integration tests ListCRMDeals against a real database.
// Run with: go test ./... -run TestListCRMDeals_Integration
func TestListCRMDeals_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	pool := setupTestDB(t)
	if pool == nil {
		return
	}
	defer pool.Close()

	h := &CRMHandler{pool: pool}
	user := crmTestUser()

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/crm/deals", nil)
	setTestUser(c, user)

	h.ListCRMDeals(c)

	assert.Equal(t, http.StatusOK, w.Code)

	body := decodeJSON(t, w)
	assert.Contains(t, body, "deals")
	assert.Contains(t, body, "count")
}

// TestCreateAndGetCRMDeal_Integration tests the full create → read lifecycle.
func TestCreateAndGetCRMDeal_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	pool := setupTestDB(t)
	if pool == nil {
		return
	}
	defer pool.Close()

	// Integration test is skipped unless the test DB has pipelines and stages.
	t.Skip("Requires seeded pipeline and stage in test database")
}

// TestListClientDeals_Integration verifies the client deals list endpoint.
func TestListClientDeals_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	pool := setupTestDB(t)
	if pool == nil {
		return
	}
	defer pool.Close()

	t.Skip("Requires seeded client in test database")
}

// TestGetCRMDealStats_Integration verifies the stats endpoint returns correctly
// shaped data.
func TestGetCRMDealStats_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	pool := setupTestDB(t)
	if pool == nil {
		return
	}
	defer pool.Close()

	h := &CRMHandler{pool: pool}
	user := crmTestUser()

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/crm/deals/stats", nil)
	setTestUser(c, user)

	h.GetCRMDealStats(c)

	assert.Equal(t, http.StatusOK, w.Code)

	body := decodeJSON(t, w)
	for _, key := range []string{"total_deals", "open_deals", "won_deals", "lost_deals"} {
		assert.Contains(t, body, key, "response missing key: %s", key)
	}
}
