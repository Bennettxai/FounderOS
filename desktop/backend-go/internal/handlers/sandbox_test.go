package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rhl/businessos-backend/internal/middleware"
	"github.com/rhl/businessos-backend/internal/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockSandboxDeploymentService is a mock implementation of SandboxDeploymentService
type MockSandboxDeploymentService struct {
	mock.Mock
}

func (m *MockSandboxDeploymentService) Deploy(ctx context.Context, req services.SandboxDeploymentRequest) (*services.SandboxInfo, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*services.SandboxInfo), args.Error(1)
}

func (m *MockSandboxDeploymentService) Stop(ctx context.Context, appID uuid.UUID) error {
	args := m.Called(ctx, appID)
	return args.Error(0)
}

func (m *MockSandboxDeploymentService) Restart(ctx context.Context, appID uuid.UUID) (*services.SandboxInfo, error) {
	args := m.Called(ctx, appID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*services.SandboxInfo), args.Error(1)
}

func (m *MockSandboxDeploymentService) Remove(ctx context.Context, appID uuid.UUID) error {
	args := m.Called(ctx, appID)
	return args.Error(0)
}

func (m *MockSandboxDeploymentService) GetSandboxInfo(ctx context.Context, appID uuid.UUID) (*services.SandboxInfo, error) {
	args := m.Called(ctx, appID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*services.SandboxInfo), args.Error(1)
}

func (m *MockSandboxDeploymentService) GetSandboxLogs(ctx context.Context, appID uuid.UUID, tail string, since string) (string, error) {
	args := m.Called(ctx, appID, tail, since)
	return args.String(0), args.Error(1)
}

func (m *MockSandboxDeploymentService) ListUserSandboxes(ctx context.Context, userID uuid.UUID) ([]*services.SandboxInfo, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*services.SandboxInfo), args.Error(1)
}

func (m *MockSandboxDeploymentService) GetStats() map[string]interface{} {
	args := m.Called()
	return args.Get(0).(map[string]interface{})
}

// setupSandboxTestRouter creates a test router with sandbox handler
func setupSandboxTestRouter(mockService *MockSandboxDeploymentService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	handler := &SandboxHandler{deploymentService: mockService, logger: slog.Default()}

	// Add test auth middleware
	router.Use(func(c *gin.Context) {
		c.Set("user", &middleware.BetterAuthUser{
			ID:   "550e8400-e29b-41d4-a716-446655440000",
			Name: "Test User",
		})
		c.Next()
	})

	api := router.Group("/api/v1")
	{
		api.POST("/sandbox/deploy", handler.DeploySandbox)
		api.POST("/sandbox/:app_id/stop", handler.StopSandbox)
		api.POST("/sandbox/:app_id/restart", handler.RestartSandbox)
		api.DELETE("/sandbox/:app_id", handler.RemoveSandbox)
		api.GET("/sandbox/:app_id", handler.GetSandboxInfo)
		api.GET("/sandbox/:app_id/logs", handler.GetSandboxLogs)
		api.GET("/sandboxes", handler.ListUserSandboxes)
		api.GET("/sandbox/stats", handler.GetSandboxStats)
	}

	return router
}

func TestDeploySandbox_Success(t *testing.T) {
	mockService := new(MockSandboxDeploymentService)
	router := setupSandboxTestRouter(mockService)

	appID := uuid.New()
	userID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	now := time.Now()

	expectedInfo := &services.SandboxInfo{
		AppID:        appID,
		AppName:      "test-app",
		UserID:       userID,
		ContainerID:  "container-123",
		Status:       services.SandboxStatusRunning,
		Port:         8080,
		URL:          "http://localhost:8080",
		Image:        "node:18-alpine",
		CreatedAt:    now,
		StartedAt:    &now,
		HealthStatus: "healthy",
	}

	mockService.On("Deploy", mock.Anything, mock.MatchedBy(func(req services.SandboxDeploymentRequest) bool {
		return req.AppID == appID && req.AppName == "test-app"
	})).Return(expectedInfo, nil)

	reqBody := map[string]interface{}{
		"app_id":   appID.String(),
		"app_name": "test-app",
		"image":    "node:18-alpine",
	}
	body, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest(http.MethodPost, "/api/v1/sandbox/deploy", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var response services.SandboxInfo
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, appID, response.AppID)
	assert.Equal(t, "test-app", response.AppName)

	mockService.AssertExpectations(t)
}

func TestDeploySandbox_MissingAppID(t *testing.T) {
	mockService := new(MockSandboxDeploymentService)
	router := setupSandboxTestRouter(mockService)

	reqBody := map[string]interface{}{
		"app_name": "test-app",
		"image":    "node:18-alpine",
	}
	body, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest(http.MethodPost, "/api/v1/sandbox/deploy", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestDeploySandbox_AlreadyRunning(t *testing.T) {
	mockService := new(MockSandboxDeploymentService)
	router := setupSandboxTestRouter(mockService)

	appID := uuid.New()

	mockService.On("Deploy", mock.Anything, mock.Anything).Return(nil, services.ErrSandboxAlreadyRunning)

	reqBody := map[string]interface{}{
		"app_id":   appID.String(),
		"app_name": "test-app",
		"image":    "node:18-alpine",
	}
	body, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest(http.MethodPost, "/api/v1/sandbox/deploy", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)
	mockService.AssertExpectations(t)
}

func TestStopSandbox_Success(t *testing.T) {
	mockService := new(MockSandboxDeploymentService)
	router := setupSandboxTestRouter(mockService)

	appID := uuid.New()

	mockService.On("Stop", mock.Anything, appID).Return(nil)

	req, _ := http.NewRequest(http.MethodPost, "/api/v1/sandbox/"+appID.String()+"/stop", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "sandbox stopped successfully", response["message"])

	mockService.AssertExpectations(t)
}

func TestStopSandbox_NotFound(t *testing.T) {
	mockService := new(MockSandboxDeploymentService)
	router := setupSandboxTestRouter(mockService)

	appID := uuid.New()

	mockService.On("Stop", mock.Anything, appID).Return(services.ErrSandboxNotFound)

	req, _ := http.NewRequest(http.MethodPost, "/api/v1/sandbox/"+appID.String()+"/stop", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	mockService.AssertExpectations(t)
}

func TestRestartSandbox_Success(t *testing.T) {
	mockService := new(MockSandboxDeploymentService)
	router := setupSandboxTestRouter(mockService)

	appID := uuid.New()
	now := time.Now()

	expectedInfo := &services.SandboxInfo{
		AppID:       appID,
		AppName:     "test-app",
		Status:      services.SandboxStatusRunning,
		ContainerID: "container-123",
		StartedAt:   &now,
	}

	mockService.On("Restart", mock.Anything, appID).Return(expectedInfo, nil)

	req, _ := http.NewRequest(http.MethodPost, "/api/v1/sandbox/"+appID.String()+"/restart", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response services.SandboxInfo
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, appID, response.AppID)
	assert.Equal(t, services.SandboxStatusRunning, response.Status)

	mockService.AssertExpectations(t)
}

func TestRemoveSandbox_Success(t *testing.T) {
	mockService := new(MockSandboxDeploymentService)
	router := setupSandboxTestRouter(mockService)

	appID := uuid.New()

	mockService.On("Remove", mock.Anything, appID).Return(nil)

	req, _ := http.NewRequest(http.MethodDelete, "/api/v1/sandbox/"+appID.String(), nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "sandbox removed successfully", response["message"])

	mockService.AssertExpectations(t)
}

func TestGetSandboxInfo_Success(t *testing.T) {
	mockService := new(MockSandboxDeploymentService)
	router := setupSandboxTestRouter(mockService)

	appID := uuid.New()

	expectedInfo := &services.SandboxInfo{
		AppID:        appID,
		AppName:      "test-app",
		Status:       services.SandboxStatusRunning,
		ContainerID:  "container-123",
		Port:         8080,
		URL:          "http://localhost:8080",
		HealthStatus: "healthy",
	}

	mockService.On("GetSandboxInfo", mock.Anything, appID).Return(expectedInfo, nil)

	req, _ := http.NewRequest(http.MethodGet, "/api/v1/sandbox/"+appID.String(), nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response services.SandboxInfo
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, appID, response.AppID)
	assert.Equal(t, "test-app", response.AppName)
	assert.Equal(t, services.SandboxStatusRunning, response.Status)

	mockService.AssertExpectations(t)
}

func TestGetSandboxLogs_Success(t *testing.T) {
	mockService := new(MockSandboxDeploymentService)
	router := setupSandboxTestRouter(mockService)

	appID := uuid.New()
	expectedLogs := "line 1\nline 2\nline 3"

	mockService.On("GetSandboxLogs", mock.Anything, appID, "100", "").Return(expectedLogs, nil)

	req, _ := http.NewRequest(http.MethodGet, "/api/v1/sandbox/"+appID.String()+"/logs?tail=100", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, expectedLogs, response["logs"])

	mockService.AssertExpectations(t)
}

func TestListUserSandboxes_Success(t *testing.T) {
	mockService := new(MockSandboxDeploymentService)
	router := setupSandboxTestRouter(mockService)

	userID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	appID1 := uuid.New()
	appID2 := uuid.New()

	expectedSandboxes := []*services.SandboxInfo{
		{
			AppID:   appID1,
			AppName: "app-1",
			Status:  services.SandboxStatusRunning,
			Port:    8080,
		},
		{
			AppID:   appID2,
			AppName: "app-2",
			Status:  services.SandboxStatusStopped,
			Port:    8081,
		},
	}

	mockService.On("ListUserSandboxes", mock.Anything, userID).Return(expectedSandboxes, nil)

	req, _ := http.NewRequest(http.MethodGet, "/api/v1/sandboxes", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, float64(2), response["count"])

	mockService.AssertExpectations(t)
}

func TestGetSandboxStats_Success(t *testing.T) {
	mockService := new(MockSandboxDeploymentService)
	router := setupSandboxTestRouter(mockService)

	expectedStats := map[string]interface{}{
		"in_progress_deployments": 2,
		"port_allocated_count":    5,
		"port_available_count":    95,
	}

	mockService.On("GetStats").Return(expectedStats)

	req, _ := http.NewRequest(http.MethodGet, "/api/v1/sandbox/stats", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, float64(2), response["in_progress_deployments"])

	mockService.AssertExpectations(t)
}

func TestDeploySandbox_InvalidJSON(t *testing.T) {
	mockService := new(MockSandboxDeploymentService)
	router := setupSandboxTestRouter(mockService)

	req, _ := http.NewRequest(http.MethodPost, "/api/v1/sandbox/deploy", bytes.NewBufferString("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGetSandboxInfo_InvalidAppID(t *testing.T) {
	mockService := new(MockSandboxDeploymentService)
	router := setupSandboxTestRouter(mockService)

	req, _ := http.NewRequest(http.MethodGet, "/api/v1/sandbox/invalid-uuid", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGetSandboxLogs_NotRunning(t *testing.T) {
	mockService := new(MockSandboxDeploymentService)
	router := setupSandboxTestRouter(mockService)

	appID := uuid.New()

	mockService.On("GetSandboxLogs", mock.Anything, appID, "all", "").Return("", services.ErrSandboxNotRunning)

	req, _ := http.NewRequest(http.MethodGet, "/api/v1/sandbox/"+appID.String()+"/logs", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	mockService.AssertExpectations(t)
}

func TestListUserSandboxes_EmptyList(t *testing.T) {
	mockService := new(MockSandboxDeploymentService)
	router := setupSandboxTestRouter(mockService)

	userID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")

	mockService.On("ListUserSandboxes", mock.Anything, userID).Return([]*services.SandboxInfo{}, nil)

	req, _ := http.NewRequest(http.MethodGet, "/api/v1/sandboxes", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, float64(0), response["count"])

	mockService.AssertExpectations(t)
}

func TestDeploySandbox_MaxSandboxesReached(t *testing.T) {
	mockService := new(MockSandboxDeploymentService)
	router := setupSandboxTestRouter(mockService)

	appID := uuid.New()

	mockService.On("Deploy", mock.Anything, mock.Anything).Return(nil, services.ErrMaxSandboxesReached)

	reqBody := map[string]interface{}{
		"app_id":   appID.String(),
		"app_name": "test-app",
		"image":    "node:18-alpine",
	}
	body, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest(http.MethodPost, "/api/v1/sandbox/deploy", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
	mockService.AssertExpectations(t)
}

func TestRemoveSandbox_InternalError(t *testing.T) {
	mockService := new(MockSandboxDeploymentService)
	router := setupSandboxTestRouter(mockService)

	appID := uuid.New()

	mockService.On("Remove", mock.Anything, appID).Return(errors.New("internal error"))

	req, _ := http.NewRequest(http.MethodDelete, "/api/v1/sandbox/"+appID.String(), nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockService.AssertExpectations(t)
}
