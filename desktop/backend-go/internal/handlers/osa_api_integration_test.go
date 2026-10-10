package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rhl/businessos-backend/internal/database/sqlc"
	"github.com/rhl/businessos-backend/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestOSAAPI_ListApps tests the GET /api/osa/apps endpoint
func TestOSAAPI_ListApps(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	testDB := testutil.RequireTestDatabase(t)
	defer testDB.Close()
	defer testutil.CleanupTestData(ctx, testDB.Pool)

	logger := slog.Default()
	queries := sqlc.New(testDB.Pool)
	handler := NewOSAAppsHandler(queries, testDB.Pool, logger)

	// Create test user
	userID, sessionID := createOSATestUser(t, ctx, testDB)

	// Create test workspace
	workspaceID := createOSATestWorkspace(t, ctx, testDB, userID)

	// Create test apps
	app1ID := createOSATestApp(t, ctx, testDB, workspaceID, "Test App 1", "pending")
	app2ID := createOSATestApp(t, ctx, testDB, workspaceID, "Test App 2", "building")
	app3ID := createOSATestApp(t, ctx, testDB, workspaceID, "Test App 3", "deployed")

	tests := []struct {
		name           string
		queryParams    string
		expectedStatus int
		validateFunc   func(*testing.T, AppListResponse)
	}{
		{
			name:           "list all apps",
			queryParams:    "",
			expectedStatus: http.StatusOK,
			validateFunc: func(t *testing.T, resp AppListResponse) {
				assert.GreaterOrEqual(t, len(resp.Apps), 3)
				assert.GreaterOrEqual(t, resp.TotalCount, int64(3))
			},
		},
		{
			name:           "filter by workspace_id",
			queryParams:    fmt.Sprintf("?workspace_id=%s", workspaceID),
			expectedStatus: http.StatusOK,
			validateFunc: func(t *testing.T, resp AppListResponse) {
				assert.Equal(t, 3, len(resp.Apps))
				for _, app := range resp.Apps {
					assert.Equal(t, workspaceID, app.WorkspaceID.String())
				}
			},
		},
		{
			name:           "filter by status",
			queryParams:    "?status=pending",
			expectedStatus: http.StatusOK,
			validateFunc: func(t *testing.T, resp AppListResponse) {
				for _, app := range resp.Apps {
					if app.Status != nil {
						assert.Equal(t, "pending", *app.Status)
					}
				}
			},
		},
		{
			name:           "pagination - limit 1",
			queryParams:    "?limit=1",
			expectedStatus: http.StatusOK,
			validateFunc: func(t *testing.T, resp AppListResponse) {
				assert.Equal(t, 1, len(resp.Apps))
				assert.Equal(t, int32(1), resp.Limit)
				assert.GreaterOrEqual(t, resp.TotalCount, int64(3))
			},
		},
		{
			name:           "pagination - offset 2",
			queryParams:    "?limit=10&offset=2",
			expectedStatus: http.StatusOK,
			validateFunc: func(t *testing.T, resp AppListResponse) {
				assert.Equal(t, int32(2), resp.Offset)
			},
		},
		{
			name:           "invalid workspace_id",
			queryParams:    "?workspace_id=not-a-uuid",
			expectedStatus: http.StatusBadRequest,
			validateFunc:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := setupOSARouter(userID, sessionID)
			router.GET("/api/osa/apps", handler.ListApps)

			req, _ := http.NewRequest("GET", "/api/osa/apps"+tt.queryParams, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.validateFunc != nil && w.Code == http.StatusOK {
				var response AppListResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				tt.validateFunc(t, response)
			}
		})
	}

	// Cleanup
	_ = app1ID
	_ = app2ID
	_ = app3ID
}

// TestOSAAPI_GetApp tests the GET /api/osa/apps/:id endpoint
func TestOSAAPI_GetApp(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	testDB := testutil.RequireTestDatabase(t)
	defer testDB.Close()
	defer testutil.CleanupTestData(ctx, testDB.Pool)

	logger := slog.Default()
	queries := sqlc.New(testDB.Pool)
	handler := NewOSAAppsHandler(queries, testDB.Pool, logger)

	userID, sessionID := createOSATestUser(t, ctx, testDB)
	otherUserID, otherSessionID := createOSATestUser(t, ctx, testDB)
	workspaceID := createOSATestWorkspace(t, ctx, testDB, userID)
	appID := createOSATestApp(t, ctx, testDB, workspaceID, "Test App", "deployed")

	tests := []struct {
		name           string
		appID          string
		currentUserID  string
		currentSession string
		expectedStatus int
		validateFunc   func(*testing.T, AppDetail)
	}{
		{
			name:           "get app successfully",
			appID:          appID,
			currentUserID:  userID,
			currentSession: sessionID,
			expectedStatus: http.StatusOK,
			validateFunc: func(t *testing.T, resp AppDetail) {
				assert.Equal(t, appID, resp.ID.String())
				assert.Equal(t, "Test App", resp.Name)
				assert.NotNil(t, resp.Status)
				assert.Equal(t, "deployed", *resp.Status)
			},
		},
		{
			name:           "app not found",
			appID:          uuid.New().String(),
			currentUserID:  userID,
			currentSession: sessionID,
			expectedStatus: http.StatusNotFound,
			validateFunc:   nil,
		},
		{
			name:           "invalid app ID",
			appID:          "not-a-uuid",
			currentUserID:  userID,
			currentSession: sessionID,
			expectedStatus: http.StatusBadRequest,
			validateFunc:   nil,
		},
		{
			name:           "unauthorized - different user",
			appID:          appID,
			currentUserID:  otherUserID,
			currentSession: otherSessionID,
			expectedStatus: http.StatusNotFound,
			validateFunc:   nil,
		},
		{
			name:           "unauthorized - no auth",
			appID:          appID,
			currentUserID:  "",
			currentSession: "",
			expectedStatus: http.StatusUnauthorized,
			validateFunc:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := setupOSARouter(tt.currentUserID, tt.currentSession)
			router.GET("/api/osa/apps/:id", handler.GetApp)

			req, _ := http.NewRequest("GET", "/api/osa/apps/"+tt.appID, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.validateFunc != nil && w.Code == http.StatusOK {
				var response AppDetail
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				tt.validateFunc(t, response)
			}
		})
	}
}

// TestOSAAPI_UpdateApp tests the PATCH /api/osa/apps/:id endpoint
func TestOSAAPI_UpdateApp(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	testDB := testutil.RequireTestDatabase(t)
	defer testDB.Close()
	defer testutil.CleanupTestData(ctx, testDB.Pool)

	logger := slog.Default()
	queries := sqlc.New(testDB.Pool)
	handler := NewOSAAppsHandler(queries, testDB.Pool, logger)

	userID, sessionID := createOSATestUser(t, ctx, testDB)
	otherUserID, otherSessionID := createOSATestUser(t, ctx, testDB)
	workspaceID := createOSATestWorkspace(t, ctx, testDB, userID)

	tests := []struct {
		name           string
		setupApp       func() string
		requestBody    UpdateAppMetadataRequest
		currentUserID  string
		currentSession string
		expectedStatus int
		validateFunc   func(*testing.T, AppDetail)
	}{
		{
			name: "update display_name",
			setupApp: func() string {
				return createOSATestApp(t, ctx, testDB, workspaceID, "Original Name", "deployed")
			},
			requestBody: UpdateAppMetadataRequest{
				DisplayName: strPtr("Updated Name"),
			},
			currentUserID:  userID,
			currentSession: sessionID,
			expectedStatus: http.StatusOK,
			validateFunc: func(t *testing.T, resp AppDetail) {
				assert.Equal(t, "Updated Name", resp.DisplayName)
			},
		},
		{
			name: "update description",
			setupApp: func() string {
				return createOSATestApp(t, ctx, testDB, workspaceID, "App", "deployed")
			},
			requestBody: UpdateAppMetadataRequest{
				Description: strPtr("New description"),
			},
			currentUserID:  userID,
			currentSession: sessionID,
			expectedStatus: http.StatusOK,
			validateFunc: func(t *testing.T, resp AppDetail) {
				assert.NotNil(t, resp.Description)
				assert.Equal(t, "New description", *resp.Description)
			},
		},
		{
			name: "update metadata",
			setupApp: func() string {
				return createOSATestApp(t, ctx, testDB, workspaceID, "App", "deployed")
			},
			requestBody: UpdateAppMetadataRequest{
				Metadata: &map[string]any{
					"custom_field": "value",
					"version":      "1.0.0",
				},
			},
			currentUserID:  userID,
			currentSession: sessionID,
			expectedStatus: http.StatusOK,
			validateFunc: func(t *testing.T, resp AppDetail) {
				assert.NotNil(t, resp.Metadata)
				assert.Equal(t, "value", resp.Metadata["custom_field"])
			},
		},
		{
			name: "update all fields",
			setupApp: func() string {
				return createOSATestApp(t, ctx, testDB, workspaceID, "App", "deployed")
			},
			requestBody: UpdateAppMetadataRequest{
				DisplayName: strPtr("All Updated"),
				Description: strPtr("All fields updated"),
				Metadata:    &map[string]any{"complete": true},
			},
			currentUserID:  userID,
			currentSession: sessionID,
			expectedStatus: http.StatusOK,
			validateFunc: func(t *testing.T, resp AppDetail) {
				assert.Equal(t, "All Updated", resp.DisplayName)
				assert.NotNil(t, resp.Description)
				assert.Equal(t, "All fields updated", *resp.Description)
				assert.NotNil(t, resp.Metadata)
			},
		},
		{
			name: "forbidden - different user",
			setupApp: func() string {
				return createOSATestApp(t, ctx, testDB, workspaceID, "App", "deployed")
			},
			requestBody: UpdateAppMetadataRequest{
				DisplayName: strPtr("Hacked"),
			},
			currentUserID:  otherUserID,
			currentSession: otherSessionID,
			expectedStatus: http.StatusForbidden,
			validateFunc:   nil,
		},
		{
			name: "invalid app ID",
			setupApp: func() string {
				return "not-a-uuid"
			},
			requestBody: UpdateAppMetadataRequest{
				DisplayName: strPtr("Test"),
			},
			currentUserID:  userID,
			currentSession: sessionID,
			expectedStatus: http.StatusBadRequest,
			validateFunc:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			appID := tt.setupApp()

			router := setupOSARouter(tt.currentUserID, tt.currentSession)
			router.PATCH("/api/osa/apps/:id", handler.UpdateApp)

			body, _ := json.Marshal(tt.requestBody)
			req, _ := http.NewRequest("PATCH", "/api/osa/apps/"+appID, bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.validateFunc != nil && w.Code == http.StatusOK {
				var response AppDetail
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				tt.validateFunc(t, response)
			}
		})
	}
}

// TestOSAAPI_DeleteApp tests the DELETE /api/osa/apps/:id endpoint
func TestOSAAPI_DeleteApp(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	testDB := testutil.RequireTestDatabase(t)
	defer testDB.Close()
	defer testutil.CleanupTestData(ctx, testDB.Pool)

	logger := slog.Default()
	queries := sqlc.New(testDB.Pool)
	handler := NewOSAAppsHandler(queries, testDB.Pool, logger)

	userID, sessionID := createOSATestUser(t, ctx, testDB)
	otherUserID, otherSessionID := createOSATestUser(t, ctx, testDB)
	workspaceID := createOSATestWorkspace(t, ctx, testDB, userID)

	tests := []struct {
		name            string
		setupApp        func() string
		currentUserID   string
		currentSession  string
		expectedStatus  int
		validateDeleted bool
	}{
		{
			name: "delete app successfully",
			setupApp: func() string {
				return createOSATestApp(t, ctx, testDB, workspaceID, "To Delete", "deployed")
			},
			currentUserID:   userID,
			currentSession:  sessionID,
			expectedStatus:  http.StatusNoContent,
			validateDeleted: true,
		},
		{
			name: "forbidden - different user",
			setupApp: func() string {
				return createOSATestApp(t, ctx, testDB, workspaceID, "Protected", "deployed")
			},
			currentUserID:   otherUserID,
			currentSession:  otherSessionID,
			expectedStatus:  http.StatusForbidden,
			validateDeleted: false,
		},
		{
			name: "app not found",
			setupApp: func() string {
				return uuid.New().String()
			},
			currentUserID:   userID,
			currentSession:  sessionID,
			expectedStatus:  http.StatusForbidden,
			validateDeleted: false,
		},
		{
			name: "invalid app ID",
			setupApp: func() string {
				return "not-a-uuid"
			},
			currentUserID:   userID,
			currentSession:  sessionID,
			expectedStatus:  http.StatusBadRequest,
			validateDeleted: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			appID := tt.setupApp()

			router := setupOSARouter(tt.currentUserID, tt.currentSession)
			router.DELETE("/api/osa/apps/:id", handler.DeleteApp)

			req, _ := http.NewRequest("DELETE", "/api/osa/apps/"+appID, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.validateDeleted {
				// Verify app was actually deleted
				pgAppID := pgtype.UUID{Bytes: uuid.MustParse(appID), Valid: true}
				pgUserID := pgtype.UUID{Bytes: uuid.MustParse(userID), Valid: true}
				_, err := queries.GetOSAModuleInstanceByIDWithAuth(ctx, sqlc.GetOSAModuleInstanceByIDWithAuthParams{
					ID:     pgAppID,
					UserID: pgUserID,
				})
				assert.Error(t, err) // Should return error because app is deleted
			}
		})
	}
}

// TestOSAAPI_GetAppLogs tests the GET /api/osa/apps/:id/logs endpoint
func TestOSAAPI_GetAppLogs(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	testDB := testutil.RequireTestDatabase(t)
	defer testDB.Close()
	defer testutil.CleanupTestData(ctx, testDB.Pool)

	logger := slog.Default()
	queries := sqlc.New(testDB.Pool)
	handler := NewOSAAppsHandler(queries, testDB.Pool, logger)

	userID, sessionID := createOSATestUser(t, ctx, testDB)
	workspaceID := createOSATestWorkspace(t, ctx, testDB, userID)
	appID := createOSATestApp(t, ctx, testDB, workspaceID, "App With Logs", "building")

	// Create test build events
	createOSABuildEvent(t, ctx, testDB, appID, "build_started", "Build started")
	createOSABuildEvent(t, ctx, testDB, appID, "build_progress", "Installing dependencies")
	createOSABuildEvent(t, ctx, testDB, appID, "build_error", "Failed to compile")

	tests := []struct {
		name           string
		queryParams    string
		expectedStatus int
		validateFunc   func(*testing.T, map[string]interface{})
	}{
		{
			name:           "get all logs",
			queryParams:    "",
			expectedStatus: http.StatusOK,
			validateFunc: func(t *testing.T, resp map[string]interface{}) {
				logs := resp["logs"].([]interface{})
				assert.GreaterOrEqual(t, len(logs), 3)
			},
		},
		{
			name:           "filter by level",
			queryParams:    "?level=error",
			expectedStatus: http.StatusOK,
			validateFunc: func(t *testing.T, resp map[string]interface{}) {
				logs := resp["logs"].([]interface{})
				// May have error logs
				assert.NotNil(t, logs)
			},
		},
		{
			name:           "pagination",
			queryParams:    "?limit=2&offset=1",
			expectedStatus: http.StatusOK,
			validateFunc: func(t *testing.T, resp map[string]interface{}) {
				assert.Equal(t, float64(2), resp["limit"])
				assert.Equal(t, float64(1), resp["offset"])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := setupOSARouter(userID, sessionID)
			router.GET("/api/osa/apps/:id/logs", handler.GetAppLogs)

			req, _ := http.NewRequest("GET", "/api/osa/apps/"+appID+"/logs"+tt.queryParams, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.validateFunc != nil && w.Code == http.StatusOK {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				tt.validateFunc(t, response)
			}
		})
	}
}

// TestOSAAPI_GenerateOSAApp tests the POST /api/workspaces/:id/apps/generate-osa endpoint
func TestOSAAPI_GenerateOSAApp(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	testDB := testutil.RequireTestDatabase(t)
	defer testDB.Close()
	defer testutil.CleanupTestData(ctx, testDB.Pool)

	logger := slog.Default()
	queries := sqlc.New(testDB.Pool)
	handler := NewOSAAppsHandler(queries, testDB.Pool, logger)

	userID, sessionID := createOSATestUser(t, ctx, testDB)
	workspaceID := createOSATestWorkspace(t, ctx, testDB, userID)
	templateID := createOSATestTemplate(t, ctx, testDB, workspaceID, "Test Template")

	// Helper to create *string pointer
	templateIDStr := templateID.String()
	invalidTemplateID := "not-a-uuid"

	tests := []struct {
		name           string
		workspaceID    string
		requestBody    GenerateOSAAppRequest
		expectedStatus int
		validateFunc   func(*testing.T, GenerateOSAAppResponse)
	}{
		{
			name:        "generate app successfully",
			workspaceID: workspaceID.String(),
			requestBody: GenerateOSAAppRequest{
				TemplateID:  &templateIDStr,
				AppName:     "New OSA App",
				Description: "Test generated app",
				Config: map[string]interface{}{
					"feature_x": true,
				},
			},
			expectedStatus: http.StatusCreated,
			validateFunc: func(t *testing.T, resp GenerateOSAAppResponse) {
				assert.NotEqual(t, uuid.Nil, resp.QueueItemID)
				assert.Equal(t, "pending", resp.Status)
				assert.Contains(t, resp.Message, "queued successfully")
			},
		},
		{
			name:        "invalid workspace ID",
			workspaceID: "not-a-uuid",
			requestBody: GenerateOSAAppRequest{
				TemplateID:  &templateIDStr,
				AppName:     "App",
				Description: "Test",
			},
			expectedStatus: http.StatusBadRequest,
			validateFunc:   nil,
		},
		{
			name:        "invalid template ID",
			workspaceID: workspaceID.String(),
			requestBody: GenerateOSAAppRequest{
				TemplateID:  &invalidTemplateID,
				AppName:     "App",
				Description: "Test",
			},
			expectedStatus: http.StatusBadRequest,
			validateFunc:   nil,
		},
		{
			name:        "missing required fields",
			workspaceID: workspaceID.String(),
			requestBody: GenerateOSAAppRequest{
				TemplateID: &templateIDStr,
				// Missing AppName
			},
			expectedStatus: http.StatusBadRequest,
			validateFunc:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := setupOSARouter(userID, sessionID)
			router.POST("/api/workspaces/:id/apps/generate-osa", handler.GenerateOSAApp)

			body, _ := json.Marshal(tt.requestBody)
			req, _ := http.NewRequest("POST", "/api/workspaces/"+tt.workspaceID+"/apps/generate-osa", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.validateFunc != nil && w.Code == http.StatusCreated {
				var response GenerateOSAAppResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				tt.validateFunc(t, response)
			}
		})
	}
}

// TestOSAAPI_ConcurrentRequests tests concurrent access to the same app
func TestOSAAPI_ConcurrentRequests(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	testDB := testutil.RequireTestDatabase(t)
	defer testDB.Close()
	defer testutil.CleanupTestData(ctx, testDB.Pool)

	logger := slog.Default()
	queries := sqlc.New(testDB.Pool)
	handler := NewOSAAppsHandler(queries, testDB.Pool, logger)

	userID, sessionID := createOSATestUser(t, ctx, testDB)
	workspaceID := createOSATestWorkspace(t, ctx, testDB, userID)
	appID := createOSATestApp(t, ctx, testDB, workspaceID, "Concurrent Test", "deployed")

	router := setupOSARouter(userID, sessionID)
	router.GET("/api/osa/apps/:id", handler.GetApp)

	// Simulate 10 concurrent GET requests
	numRequests := 10
	done := make(chan bool, numRequests)
	errors := make(chan error, numRequests)

	for i := 0; i < numRequests; i++ {
		go func() {
			req, _ := http.NewRequest("GET", "/api/osa/apps/"+appID, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				errors <- fmt.Errorf("expected 200, got %d", w.Code)
			} else {
				var response AppDetail
				if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
					errors <- err
				}
			}
			done <- true
		}()
	}

	// Wait for all requests to complete
	for i := 0; i < numRequests; i++ {
		<-done
	}

	close(errors)
	for err := range errors {
		assert.NoError(t, err)
	}
}

// TestOSAAPI_EdgeCases tests various edge cases
func TestOSAAPI_EdgeCases(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	testDB := testutil.RequireTestDatabase(t)
	defer testDB.Close()
	defer testutil.CleanupTestData(ctx, testDB.Pool)

	logger := slog.Default()
	queries := sqlc.New(testDB.Pool)
	handler := NewOSAAppsHandler(queries, testDB.Pool, logger)

	userID, sessionID := createOSATestUser(t, ctx, testDB)
	workspaceID := createOSATestWorkspace(t, ctx, testDB, userID)

	t.Run("empty list when no apps", func(t *testing.T) {
		router := setupOSARouter(userID, sessionID)
		router.GET("/api/osa/apps", handler.ListApps)

		req, _ := http.NewRequest("GET", "/api/osa/apps?workspace_id="+uuid.New().String(), nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response AppListResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, 0, len(response.Apps))
	})

	t.Run("large metadata update", func(t *testing.T) {
		appID := createOSATestApp(t, ctx, testDB, workspaceID, "Large Metadata", "deployed")

		router := setupOSARouter(userID, sessionID)
		router.PATCH("/api/osa/apps/:id", handler.UpdateApp)

		largeMetadata := make(map[string]any)
		for i := 0; i < 100; i++ {
			largeMetadata[fmt.Sprintf("field_%d", i)] = fmt.Sprintf("value_%d", i)
		}

		requestBody := UpdateAppMetadataRequest{
			Metadata: &largeMetadata,
		}
		body, _ := json.Marshal(requestBody)
		req, _ := http.NewRequest("PATCH", "/api/osa/apps/"+appID, bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response AppDetail
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, 100, len(response.Metadata))
	})

	t.Run("pagination edge - negative offset", func(t *testing.T) {
		router := setupOSARouter(userID, sessionID)
		router.GET("/api/osa/apps", handler.ListApps)

		req, _ := http.NewRequest("GET", "/api/osa/apps?offset=-1", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code) // Should default to 0
	})

	t.Run("pagination edge - zero limit", func(t *testing.T) {
		router := setupOSARouter(userID, sessionID)
		router.GET("/api/osa/apps", handler.ListApps)

		req, _ := http.NewRequest("GET", "/api/osa/apps?limit=0", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code) // Should use default limit
	})
}

// Helper functions imported from osa_api_test_helpers.go
// - setupOSARouter
// - createOSATestUser
// - createOSATestWorkspace
// - createOSATestApp
// - createOSABuildEvent
// - createOSATestTemplate
// - pgUUIDFromString

// strPtr returns a pointer to the string value (test helper)
func strPtr(s string) *string {
	return &s
}
