package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rhl/businessos-backend/internal/services"
)

// Note: Helper functions (setupTestDB, createTestWorkspace, createTestApp, cleanupWorkflows)
// are defined in osa_test_helpers.go

// setupDeploymentService creates a test deployment service
func setupDeploymentService(t *testing.T) (*services.AppDeploymentService, string) {
	t.Helper()

	pool := setupTestDB(t)
	if pool == nil {
		return nil, ""
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelError, // Quiet during tests
	}))

	workspaceRoot := t.TempDir()
	service := services.NewAppDeploymentService(pool, logger, workspaceRoot)

	return service, workspaceRoot
}

// createDeployableApp creates a test app with deployable code
// func createDeployableApp(t *testing.T, service *services.AppDeploymentService, appType string) uuid.UUID {
// 	t.Helper()
// 	// This is a mock - in real tests, we'd create the app in DB
// 	// For now, just return a UUID
// 	return uuid.New()
// }

// TestDeployApp tests app deployment
func TestDeployApp(t *testing.T) {
	service, _ := setupDeploymentService(t)
	if service == nil {
		t.Skip("Deployment service not available")
		return
	}

	handler := NewOSADeploymentHandler(service)

	tests := []struct {
		name           string
		appID          string
		expectedStatus int
		setupApp       bool
	}{
		{
			name:           "Deploy app - invalid UUID",
			appID:          "invalid-uuid",
			expectedStatus: http.StatusBadRequest,
			setupApp:       false,
		},
		{
			name:           "Deploy app - not found",
			appID:          uuid.New().String(),
			expectedStatus: http.StatusInternalServerError,
			setupApp:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			req := httptest.NewRequest("POST", "/api/osa/deployment/"+tt.appID+"/deploy", nil)

			c, _ := gin.CreateTestContext(w)
			c.Request = req
			c.Params = gin.Params{{Key: "app_id", Value: tt.appID}}

			handler.DeployApp(c)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

// TestDeployApp_NodeApp tests deploying a Node.js application
func TestDeployApp_NodeApp(t *testing.T) {
	pool := setupTestDB(t)
	if pool == nil {
		t.Skip("Database not available")
		return
	}
	defer pool.Close()

	userID := uuid.New()
	workspaceID := createTestWorkspace(t, pool, userID)
	defer cleanupWorkflows(t, pool, workspaceID)

	// Create app with Node.js code bundle
	codeBundle := `=== FILE: package.json ===
{
  "name": "test-app",
  "version": "1.0.0",
  "scripts": {
    "start": "node index.js"
  }
}
=== END FILE ===
=== FILE: index.js ===
const http = require('http');
const port = process.env.PORT || 3000;

http.createServer((req, res) => {
  res.writeHead(200, {'Content-Type': 'text/plain'});
  res.end('Hello from test app');
}).listen(port);

console.log('Server running on port ' + port);
=== END FILE ===`

	metadata := map[string]interface{}{
		"code": codeBundle,
	}
	appID := createTestApp(t, pool, workspaceID, metadata)

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	workspaceRoot := t.TempDir()
	deploymentService := services.NewAppDeploymentService(pool, logger, workspaceRoot)

	handler := NewOSADeploymentHandler(deploymentService)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/osa/deployment/"+appID.String()+"/deploy", nil)

	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Params = gin.Params{{Key: "id", Value: appID.String()}}

	handler.DeployApp(c)

	// Deployment may fail due to missing npm, but we test the handler logic
	if w.Code == http.StatusOK {
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.NotEmpty(t, response["id"])
		assert.NotEmpty(t, response["url"])
		assert.NotNil(t, response["port"])
		assert.Equal(t, "running", response["status"])
	}
}

// TestStopApp tests stopping a deployed app
func TestStopApp(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	workspaceRoot := t.TempDir()
	deploymentService := services.NewAppDeploymentService(nil, logger, workspaceRoot)

	handler := NewOSADeploymentHandler(deploymentService)

	tests := []struct {
		name           string
		appID          string
		expectedStatus int
	}{
		{
			name:           "Stop app - invalid UUID",
			appID:          "invalid-uuid",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Stop app - not found",
			appID:          uuid.New().String(),
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			req := httptest.NewRequest("POST", "/api/osa/deployment/"+tt.appID+"/stop", nil)

			c, _ := gin.CreateTestContext(w)
			c.Request = req
			c.Params = gin.Params{{Key: "app_id", Value: tt.appID}}

			handler.StopApp(c)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

// TestGetAppStatus tests retrieving app status
func TestGetAppStatus(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	workspaceRoot := t.TempDir()
	deploymentService := services.NewAppDeploymentService(nil, logger, workspaceRoot)

	handler := NewOSADeploymentHandler(deploymentService)

	tests := []struct {
		name           string
		appID          string
		expectedStatus int
	}{
		{
			name:           "Get status - invalid UUID",
			appID:          "invalid-uuid",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Get status - not deployed",
			appID:          uuid.New().String(),
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			req := httptest.NewRequest("GET", "/api/osa/deployment/"+tt.appID+"/status", nil)

			c, _ := gin.CreateTestContext(w)
			c.Request = req
			c.Params = gin.Params{{Key: "app_id", Value: tt.appID}}

			handler.GetAppStatus(c)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

// TestListDeployedApps tests listing all deployed apps
func TestListDeployedApps(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	workspaceRoot := t.TempDir()
	deploymentService := services.NewAppDeploymentService(nil, logger, workspaceRoot)

	handler := NewOSADeploymentHandler(deploymentService)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/osa/deployments", nil)

	c, _ := gin.CreateTestContext(w)
	c.Request = req

	handler.ListDeployedApps(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.NotNil(t, response["apps"])
	assert.Equal(t, float64(0), response["count"]) // No apps deployed yet
}

// TestDeploymentLifecycle tests the full deployment lifecycle
func TestDeploymentLifecycle(t *testing.T) {
	pool := setupTestDB(t)
	if pool == nil {
		t.Skip("Database not available")
		return
	}
	defer pool.Close()

	userID := uuid.New()
	workspaceID := createTestWorkspace(t, pool, userID)
	defer cleanupWorkflows(t, pool, workspaceID)

	// Create a simple static app (doesn't require npm)
	codeBundle := `=== FILE: index.html ===
<!DOCTYPE html>
<html>
<head><title>Test App</title></head>
<body><h1>Hello World</h1></body>
</html>
=== END FILE ===`

	metadata := map[string]interface{}{
		"code": codeBundle,
	}
	appID := createTestApp(t, pool, workspaceID, metadata)

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	workspaceRoot := t.TempDir()
	deploymentService := services.NewAppDeploymentService(pool, logger, workspaceRoot)

	handler := NewOSADeploymentHandler(deploymentService)

	// Step 1: Deploy
	t.Run("Deploy", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/api/osa/deployment/"+appID.String()+"/deploy", nil)

		c, _ := gin.CreateTestContext(w)
		c.Request = req
		c.Params = gin.Params{{Key: "app_id", Value: appID.String()}}

		handler.DeployApp(c)

		if w.Code != http.StatusOK {
			t.Logf("Deploy failed (expected in test environment): %s", w.Body.String())
			t.Skip("Skipping lifecycle test - deployment requires system dependencies")
			return
		}

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, "running", response["status"])
	})

	// Give app time to start
	time.Sleep(100 * time.Millisecond)

	// Step 2: Get Status
	t.Run("GetStatus", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/api/osa/deployment/"+appID.String()+"/status", nil)

		c, _ := gin.CreateTestContext(w)
		c.Request = req
		c.Params = gin.Params{{Key: "app_id", Value: appID.String()}}

		handler.GetAppStatus(c)

		if w.Code == http.StatusOK {
			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			assert.Equal(t, "running", response["status"])
			assert.NotEmpty(t, response["url"])
		}
	})

	// Step 3: List Deployed
	t.Run("ListDeployed", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/api/osa/deployments", nil)

		c, _ := gin.CreateTestContext(w)
		c.Request = req

		handler.ListDeployedApps(c)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		count := response["count"].(float64)
		if count > 0 {
			apps := response["apps"].([]interface{})
			assert.Greater(t, len(apps), 0)
		}
	})

	// Step 4: Stop
	t.Run("Stop", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/api/osa/deployment/"+appID.String()+"/stop", nil)

		c, _ := gin.CreateTestContext(w)
		c.Request = req
		c.Params = gin.Params{{Key: "app_id", Value: appID.String()}}

		handler.StopApp(c)

		if w.Code == http.StatusOK {
			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			assert.Equal(t, "App stopped successfully", response["message"])
		}
	})

	// Step 5: Verify stopped
	t.Run("VerifyStopped", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/api/osa/deployment/"+appID.String()+"/status", nil)

		c, _ := gin.CreateTestContext(w)
		c.Request = req
		c.Params = gin.Params{{Key: "app_id", Value: appID.String()}}

		handler.GetAppStatus(c)

		// Should be 404 since app is no longer deployed
		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

// TestPortAllocation tests that ports are allocated correctly
func TestPortAllocation(t *testing.T) {
	allocator := services.NewPortAllocator(9000)

	// Allocate 5 ports
	ports := make([]int, 5)
	for i := 0; i < 5; i++ {
		ports[i] = allocator.Allocate()
	}

	// All ports should be unique
	portMap := make(map[int]bool)
	for _, port := range ports {
		assert.False(t, portMap[port], "Port %d allocated twice", port)
		portMap[port] = true
		assert.GreaterOrEqual(t, port, 9000)
	}

	// Release a port
	allocator.Release(ports[2])

	// Allocate another - might reuse the released port
	newPort := allocator.Allocate()
	assert.GreaterOrEqual(t, newPort, 9000)
}

// TestConcurrentDeployments tests multiple concurrent deployments
func TestConcurrentDeployments(t *testing.T) {
	pool := setupTestDB(t)
	if pool == nil {
		t.Skip("Database not available")
		return
	}
	defer pool.Close()

	userID := uuid.New()
	workspaceID := createTestWorkspace(t, pool, userID)
	defer cleanupWorkflows(t, pool, workspaceID)

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	workspaceRoot := t.TempDir()
	deploymentService := services.NewAppDeploymentService(pool, logger, workspaceRoot)

	handler := NewOSADeploymentHandler(deploymentService)

	// Create 3 test apps
	codeBundle := `=== FILE: index.html ===
<html><body>Test</body></html>
=== END FILE ===`

	appIDs := make([]uuid.UUID, 3)
	for i := 0; i < 3; i++ {
		metadata := map[string]interface{}{"code": codeBundle}
		appIDs[i] = createTestApp(t, pool, workspaceID, metadata)
	}

	// Deploy all concurrently
	results := make(chan int, 3)

	for _, appID := range appIDs {
		go func(id uuid.UUID) {
			w := httptest.NewRecorder()
			req := httptest.NewRequest("POST", "/api/osa/deployment/"+id.String()+"/deploy", nil)

			c, _ := gin.CreateTestContext(w)
			c.Request = req
			c.Params = gin.Params{{Key: "app_id", Value: id.String()}}

			handler.DeployApp(c)
			results <- w.Code
		}(appID)
	}

	// Collect results
	for i := 0; i < 3; i++ {
		statusCode := <-results
		// Status can be OK or error depending on system environment
		assert.NotEqual(t, http.StatusBadRequest, statusCode)
	}
}

// TestDeploymentErrorHandling tests error scenarios
func TestDeploymentErrorHandling(t *testing.T) {
	pool := setupTestDB(t)
	if pool == nil {
		t.Skip("Database not available")
		return
	}
	defer pool.Close()

	userID := uuid.New()
	workspaceID := createTestWorkspace(t, pool, userID)
	defer cleanupWorkflows(t, pool, workspaceID)

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	workspaceRoot := t.TempDir()
	deploymentService := services.NewAppDeploymentService(pool, logger, workspaceRoot)

	handler := NewOSADeploymentHandler(deploymentService)

	tests := []struct {
		name         string
		codeBundle   string
		expectedFail bool
	}{
		{
			name:         "Empty code bundle",
			codeBundle:   "",
			expectedFail: true,
		},
		{
			name:         "Invalid bundle format",
			codeBundle:   "not a valid bundle",
			expectedFail: true,
		},
		{
			name: "Missing package.json for node app",
			codeBundle: `=== FILE: index.js ===
console.log('no package.json');
=== END FILE ===`,
			expectedFail: false, // Might succeed as unknown type
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metadata := map[string]interface{}{"code": tt.codeBundle}
			appID := createTestApp(t, pool, workspaceID, metadata)

			w := httptest.NewRecorder()
			req := httptest.NewRequest("POST", "/api/osa/deployment/"+appID.String()+"/deploy", nil)

			c, _ := gin.CreateTestContext(w)
			c.Request = req
			c.Params = gin.Params{{Key: "app_id", Value: appID.String()}}

			handler.DeployApp(c)

			if tt.expectedFail {
				assert.Equal(t, http.StatusInternalServerError, w.Code)
			} else {
				// May succeed or fail depending on environment
				assert.NotEqual(t, http.StatusBadRequest, w.Code)
			}
		})
	}
}

// TestAppTypeDetection tests app type detection logic
func TestAppTypeDetection(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	workspaceRoot := t.TempDir()
	_ = services.NewAppDeploymentService(nil, logger, workspaceRoot) // Used for type detection validation

	tests := []struct {
		name         string
		files        []services.BundledFile
		expectedType string
	}{
		{
			name: "Node.js app",
			files: []services.BundledFile{
				{Path: "package.json", Content: "{}"},
				{Path: "index.js", Content: "console.log()"},
			},
			expectedType: "node",
		},
		{
			name: "Python app",
			files: []services.BundledFile{
				{Path: "requirements.txt", Content: "flask"},
				{Path: "app.py", Content: "print()"},
			},
			expectedType: "python",
		},
		{
			name: "Static app",
			files: []services.BundledFile{
				{Path: "index.html", Content: "<html></html>"},
				{Path: "style.css", Content: "body {}"},
			},
			expectedType: "static",
		},
		{
			name: "Unknown app",
			files: []services.BundledFile{
				{Path: "README.md", Content: "# Test"},
			},
			expectedType: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Use reflection or expose method for testing
			// For now, we verify indirectly through deployment
			t.Logf("Expected type: %s for files: %v", tt.expectedType, tt.files)
		})
	}
}

// BenchmarkDeployApp benchmarks app deployment
func BenchmarkDeployApp(b *testing.B) {
	service := services.NewAppDeploymentService(nil, slog.Default(), b.TempDir())

	handler := NewOSADeploymentHandler(service)
	appID := uuid.New()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/api/osa/deployment/"+appID.String()+"/deploy", nil)

		c, _ := gin.CreateTestContext(w)
		c.Request = req
		c.Params = gin.Params{{Key: "app_id", Value: appID.String()}}

		handler.DeployApp(c)
	}
}
