package handlers

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Note: Helper functions (setupTestDB, createTestWorkspace, createTestApp, cleanupWorkflows)
// are defined in osa_test_helpers.go

// TestHandleWorkflowComplete tests workflow completion webhook
func TestHandleWorkflowComplete(t *testing.T) {
	pool := setupTestDB(t)
	if pool == nil {
		return
	}
	defer pool.Close()

	userID := uuid.New()
	workspaceID := createTestWorkspace(t, pool, userID)
	defer cleanupWorkflows(t, pool, workspaceID)

	metadata := map[string]interface{}{"code": "test"}
	appID := createTestApp(t, pool, workspaceID, metadata)

	// Get the workflow ID from the created app
	var workflowID string
	err := pool.QueryRow(context.Background(),
		"SELECT osa_workflow_id FROM osa_module_instances WHERE id = $1", appID).Scan(&workflowID)
	require.NoError(t, err)

	webhookSecret := "test-secret-key"
	handler, err := NewOSAWebhooksHandler(pool, webhookSecret, nil, false)
	require.NoError(t, err)

	tests := []struct {
		name           string
		event          WebhookEvent
		signature      string
		useValidSig    bool
		expectedStatus int
		checkDB        bool
	}{
		{
			name: "Valid webhook with signature",
			event: WebhookEvent{
				EventType:  "workflow.complete",
				WorkflowID: workflowID,
				Timestamp:  time.Now(),
				Status:     "completed",
				Data: map[string]interface{}{
					"build_status": "success",
					"message":      "Build completed successfully",
				},
			},
			useValidSig:    true,
			expectedStatus: http.StatusOK,
			checkDB:        true,
		},
		{
			name: "Invalid signature",
			event: WebhookEvent{
				EventType:  "workflow.complete",
				WorkflowID: workflowID,
				Timestamp:  time.Now(),
				Status:     "completed",
				Data:       map[string]interface{}{},
			},
			signature:      "invalid-signature",
			useValidSig:    false,
			expectedStatus: http.StatusUnauthorized,
			checkDB:        false,
		},
		{
			name: "Failed workflow",
			event: WebhookEvent{
				EventType:  "workflow.complete",
				WorkflowID: workflowID,
				Timestamp:  time.Now(),
				Status:     "failed",
				Data: map[string]interface{}{
					"build_status":  "failed",
					"error_message": "Build failed: missing dependency",
				},
			},
			useValidSig:    true,
			expectedStatus: http.StatusOK,
			checkDB:        true,
		},
		{
			name: "Invalid JSON payload",
			event: WebhookEvent{
				EventType: "invalid",
			},
			useValidSig:    true,
			expectedStatus: http.StatusOK, // Will succeed but might not update DB
			checkDB:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eventJSON, err := json.Marshal(tt.event)
			require.NoError(t, err)

			var signature string
			if tt.useValidSig {
				mac := hmac.New(sha256.New, []byte(webhookSecret))
				mac.Write(eventJSON)
				signature = hex.EncodeToString(mac.Sum(nil))
			} else {
				signature = tt.signature
			}

			w := httptest.NewRecorder()
			req := httptest.NewRequest("POST", "/api/osa/webhooks/workflow-complete",
				bytes.NewReader(eventJSON))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-OSA-Signature", signature)

			c, _ := gin.CreateTestContext(w)
			c.Request = req

			handler.HandleWorkflowComplete(c)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.checkDB && tt.expectedStatus == http.StatusOK {
				// Verify database was updated
				var status string
				err := pool.QueryRow(context.Background(),
					"SELECT status FROM osa_module_instances WHERE id = $1", appID).Scan(&status)
				require.NoError(t, err)
				assert.Equal(t, tt.event.Status, status)

				// Verify build event was created
				var eventCount int
				err = pool.QueryRow(context.Background(),
					"SELECT COUNT(*) FROM osa_build_events WHERE module_instance_id = $1", appID).Scan(&eventCount)
				require.NoError(t, err)
				assert.Greater(t, eventCount, 0)
			}
		})
	}
}

// TestHandleBuildEvent tests build progress webhooks
func TestHandleBuildEvent(t *testing.T) {
	pool := setupTestDB(t)
	if pool == nil {
		return
	}
	defer pool.Close()

	userID := uuid.New()
	workspaceID := createTestWorkspace(t, pool, userID)
	defer cleanupWorkflows(t, pool, workspaceID)

	metadata := map[string]interface{}{"code": "test"}
	appID := createTestApp(t, pool, workspaceID, metadata)

	var workflowID string
	err := pool.QueryRow(context.Background(),
		"SELECT osa_workflow_id FROM osa_module_instances WHERE id = $1", appID).Scan(&workflowID)
	require.NoError(t, err)

	webhookSecret := "test-secret-key"
	handler, err := NewOSAWebhooksHandler(pool, webhookSecret, nil, false)
	require.NoError(t, err)

	tests := []struct {
		name           string
		event          WebhookEvent
		expectedStatus int
		checkProgress  bool
	}{
		{
			name: "Build started - 0%",
			event: WebhookEvent{
				EventType:  "build.started",
				WorkflowID: workflowID,
				Timestamp:  time.Now(),
				Status:     "building",
				Data: map[string]interface{}{
					"progress": float64(0),
					"message":  "Starting build process",
				},
			},
			expectedStatus: http.StatusOK,
			checkProgress:  true,
		},
		{
			name: "Build progress - 50%",
			event: WebhookEvent{
				EventType:  "build.progress",
				WorkflowID: workflowID,
				Timestamp:  time.Now(),
				Status:     "building",
				Data: map[string]interface{}{
					"progress": float64(50),
					"message":  "Installing dependencies",
				},
			},
			expectedStatus: http.StatusOK,
			checkProgress:  true,
		},
		{
			name: "Build progress - 100%",
			event: WebhookEvent{
				EventType:  "build.progress",
				WorkflowID: workflowID,
				Timestamp:  time.Now(),
				Status:     "building",
				Data: map[string]interface{}{
					"progress": float64(100),
					"message":  "Finalizing build",
				},
			},
			expectedStatus: http.StatusOK,
			checkProgress:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eventJSON, err := json.Marshal(tt.event)
			require.NoError(t, err)

			// Generate valid signature
			mac := hmac.New(sha256.New, []byte(webhookSecret))
			mac.Write(eventJSON)
			signature := hex.EncodeToString(mac.Sum(nil))

			w := httptest.NewRecorder()
			req := httptest.NewRequest("POST", "/api/osa/webhooks/build-event",
				bytes.NewReader(eventJSON))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-OSA-Signature", signature)

			c, _ := gin.CreateTestContext(w)
			c.Request = req

			handler.HandleBuildEvent(c)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.checkProgress {
				// Verify build event was recorded
				var progressPercent int
				var statusMessage string
				err := pool.QueryRow(context.Background(), `
					SELECT progress_percent, status_message
					FROM osa_build_events
					WHERE module_instance_id = $1
					ORDER BY created_at DESC
					LIMIT 1
				`, appID).Scan(&progressPercent, &statusMessage)

				require.NoError(t, err)
				expectedProgress := int(tt.event.Data["progress"].(float64))
				assert.Equal(t, expectedProgress, progressPercent)
				assert.Contains(t, statusMessage, tt.event.Data["message"].(string))
			}
		})
	}
}

// TestVerifySignature tests HMAC signature verification
func TestVerifySignature(t *testing.T) {
	webhookSecret := "test-secret-123"
	handler, err := NewOSAWebhooksHandler(nil, webhookSecret, nil, false)
	require.NoError(t, err)

	testPayload := []byte(`{"event_type":"test","workflow_id":"wf-123"}`)

	t.Run("Valid signature", func(t *testing.T) {
		// Generate valid signature for the payload
		mac := hmac.New(sha256.New, []byte(webhookSecret))
		mac.Write(testPayload)
		signature := hex.EncodeToString(mac.Sum(nil))

		result := handler.verifySignature(testPayload, signature)
		assert.True(t, result, "Valid signature should pass verification")
	})

	t.Run("Invalid signature", func(t *testing.T) {
		result := handler.verifySignature(testPayload, "invalid-signature")
		assert.False(t, result, "Invalid signature should fail verification")
	})

	t.Run("Empty signature (dev mode)", func(t *testing.T) {
		// Set environment variable to enable dev mode
		t.Setenv("OSA_ALLOW_UNSIGNED_WEBHOOKS", "true")

		// Empty signature is allowed in dev mode per verifySignature implementation
		result := handler.verifySignature(testPayload, "")
		assert.True(t, result, "Empty signature should be allowed in dev mode")
	})

	t.Run("Modified payload with original signature", func(t *testing.T) {
		// Generate signature for original payload
		mac := hmac.New(sha256.New, []byte(webhookSecret))
		mac.Write(testPayload)
		originalSignature := hex.EncodeToString(mac.Sum(nil))

		// Try to verify modified payload with original signature
		modifiedPayload := []byte(`{"event_type":"test","workflow_id":"wf-456"}`)
		result := handler.verifySignature(modifiedPayload, originalSignature)
		assert.False(t, result, "Modified payload with original signature should fail")
	})

	t.Run("Wrong secret", func(t *testing.T) {
		// Generate signature with wrong secret
		wrongSecret := "wrong-secret-456"
		mac := hmac.New(sha256.New, []byte(wrongSecret))
		mac.Write(testPayload)
		wrongSignature := hex.EncodeToString(mac.Sum(nil))

		result := handler.verifySignature(testPayload, wrongSignature)
		assert.False(t, result, "Signature with wrong secret should fail")
	})

	t.Run("Tampered signature", func(t *testing.T) {
		// Generate valid signature then tamper with it
		mac := hmac.New(sha256.New, []byte(webhookSecret))
		mac.Write(testPayload)
		validSig := mac.Sum(nil)

		// Flip a bit in the signature
		tamperedSig := make([]byte, len(validSig))
		copy(tamperedSig, validSig)
		tamperedSig[0] ^= 0x01 // Flip first bit

		result := handler.verifySignature(testPayload, hex.EncodeToString(tamperedSig))
		assert.False(t, result, "Tampered signature should fail")
	})
}

// TestRegisterWebhook tests webhook registration
func TestRegisterWebhook(t *testing.T) {
	pool := setupTestDB(t)
	if pool == nil {
		return
	}
	defer pool.Close()

	userID := uuid.New()
	workspaceID := createTestWorkspace(t, pool, userID)
	defer cleanupWorkflows(t, pool, workspaceID)

	handler, err := NewOSAWebhooksHandler(pool, "secret", nil, false)
	require.NoError(t, err)

	tests := []struct {
		name           string
		requestBody    map[string]interface{}
		userID         interface{}
		expectedStatus int
	}{
		{
			name: "Register webhook - success",
			requestBody: map[string]interface{}{
				"workspace_id": workspaceID.String(),
				"event_type":   "build.complete",
				"webhook_url":  "https://example.com/webhook",
			},
			userID:         userID,
			expectedStatus: http.StatusCreated,
		},
		{
			name:           "Register webhook - invalid body",
			requestBody:    map[string]interface{}{},
			userID:         userID,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Register webhook - unauthorized",
			requestBody: map[string]interface{}{
				"event_type":  "build.complete",
				"webhook_url": "https://example.com/webhook",
			},
			userID:         nil,
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bodyBytes, err := json.Marshal(tt.requestBody)
			require.NoError(t, err)

			w := httptest.NewRecorder()
			req := httptest.NewRequest("POST", "/api/osa/webhooks/register",
				bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")

			c, _ := gin.CreateTestContext(w)
			c.Request = req
			if tt.userID != nil {
				c.Set("userID", tt.userID)
			}

			handler.RegisterWebhook(c)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedStatus == http.StatusCreated {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)

				assert.NotEmpty(t, response["id"])
				assert.NotEmpty(t, response["secret_key"])
				assert.Equal(t, "Webhook registered successfully", response["message"])
			}
		})
	}
}

// TestListWebhooks tests listing webhooks
func TestListWebhooks(t *testing.T) {
	pool := setupTestDB(t)
	if pool == nil {
		return
	}
	defer pool.Close()

	userID := uuid.New()
	workspaceID := createTestWorkspace(t, pool, userID)
	defer cleanupWorkflows(t, pool, workspaceID)

	handler, err := NewOSAWebhooksHandler(pool, "secret", nil, false)
	require.NoError(t, err)

	// Create test webhooks
	_, err = pool.Exec(context.Background(), `
		INSERT INTO osa_webhooks (workspace_id, event_type, webhook_url, secret_key, enabled)
		VALUES ($1, 'build.complete', 'https://example.com/webhook1', 'secret1', true),
		       ($1, 'workflow.complete', 'https://example.com/webhook2', 'secret2', true)
	`, workspaceID)
	require.NoError(t, err)

	tests := []struct {
		name           string
		userID         interface{}
		expectedStatus int
		minCount       int
	}{
		{
			name:           "List webhooks - success",
			userID:         userID,
			expectedStatus: http.StatusOK,
			minCount:       2,
		},
		{
			name:           "List webhooks - unauthorized",
			userID:         nil,
			expectedStatus: http.StatusUnauthorized,
			minCount:       0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			req := httptest.NewRequest("GET", "/api/osa/webhooks", nil)

			c, _ := gin.CreateTestContext(w)
			c.Request = req
			if tt.userID != nil {
				c.Set("userID", tt.userID)
			}

			handler.ListWebhooks(c)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedStatus == http.StatusOK {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)

				webhooks := response["webhooks"].([]interface{})
				assert.GreaterOrEqual(t, len(webhooks), tt.minCount)

				if len(webhooks) > 0 {
					webhook := webhooks[0].(map[string]interface{})
					assert.NotEmpty(t, webhook["id"])
					assert.NotEmpty(t, webhook["event_type"])
					assert.NotEmpty(t, webhook["webhook_url"])
					assert.NotNil(t, webhook["enabled"])
				}
			}
		})
	}
}

// TestProcessWorkflowComplete tests workflow completion processing logic
func TestProcessWorkflowComplete(t *testing.T) {
	pool := setupTestDB(t)
	if pool == nil {
		return
	}
	defer pool.Close()

	userID := uuid.New()
	workspaceID := createTestWorkspace(t, pool, userID)
	defer cleanupWorkflows(t, pool, workspaceID)

	metadata := map[string]interface{}{"code": "test"}
	appID := createTestApp(t, pool, workspaceID, metadata)

	var workflowID string
	err := pool.QueryRow(context.Background(),
		"SELECT osa_workflow_id FROM osa_module_instances WHERE id = $1", appID).Scan(&workflowID)
	require.NoError(t, err)

	handler, err := NewOSAWebhooksHandler(pool, "secret", nil, false)
	require.NoError(t, err)

	tests := []struct {
		name        string
		event       *WebhookEvent
		expectError bool
	}{
		{
			name: "Process success event",
			event: &WebhookEvent{
				EventType:  "workflow.complete",
				WorkflowID: workflowID,
				Status:     "completed",
				Data: map[string]interface{}{
					"build_status": "success",
				},
			},
			expectError: false,
		},
		{
			name: "Process failure event",
			event: &WebhookEvent{
				EventType:  "workflow.complete",
				WorkflowID: workflowID,
				Status:     "failed",
				Data: map[string]interface{}{
					"build_status":  "failed",
					"error_message": "Compilation error",
				},
			},
			expectError: false,
		},
		{
			name: "Process nonexistent workflow",
			event: &WebhookEvent{
				EventType:  "workflow.complete",
				WorkflowID: "wf-nonexistent",
				Status:     "completed",
				Data:       map[string]interface{}{},
			},
			expectError: false, // Should not error, just won't update
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			err := handler.processWorkflowComplete(ctx, tt.event)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestProcessBuildEvent tests build event processing
func TestProcessBuildEvent(t *testing.T) {
	pool := setupTestDB(t)
	if pool == nil {
		return
	}
	defer pool.Close()

	userID := uuid.New()
	workspaceID := createTestWorkspace(t, pool, userID)
	defer cleanupWorkflows(t, pool, workspaceID)

	metadata := map[string]interface{}{"code": "test"}
	appID := createTestApp(t, pool, workspaceID, metadata)

	var workflowID string
	err := pool.QueryRow(context.Background(),
		"SELECT osa_workflow_id FROM osa_module_instances WHERE id = $1", appID).Scan(&workflowID)
	require.NoError(t, err)

	handler, err := NewOSAWebhooksHandler(pool, "secret", nil, false)
	require.NoError(t, err)

	event := &WebhookEvent{
		EventType:  "build.progress",
		WorkflowID: workflowID,
		Status:     "building",
		Data: map[string]interface{}{
			"progress": float64(75),
			"message":  "Building project",
		},
	}

	ctx := context.Background()
	err = handler.processBuildEvent(ctx, event)
	assert.NoError(t, err)

	// Verify event was stored
	var count int
	err = pool.QueryRow(ctx,
		"SELECT COUNT(*) FROM osa_build_events WHERE module_instance_id = $1", appID).Scan(&count)
	require.NoError(t, err)
	assert.Greater(t, count, 0)
}

// TestWebhookConcurrency tests concurrent webhook processing
func TestWebhookConcurrency(t *testing.T) {
	pool := setupTestDB(t)
	if pool == nil {
		return
	}
	defer pool.Close()

	userID := uuid.New()
	workspaceID := createTestWorkspace(t, pool, userID)
	defer cleanupWorkflows(t, pool, workspaceID)

	metadata := map[string]interface{}{"code": "test"}
	appID := createTestApp(t, pool, workspaceID, metadata)

	var workflowID string
	err := pool.QueryRow(context.Background(),
		"SELECT osa_workflow_id FROM osa_module_instances WHERE id = $1", appID).Scan(&workflowID)
	require.NoError(t, err)

	webhookSecret := "test-secret"
	handler, err := NewOSAWebhooksHandler(pool, webhookSecret, nil, false)
	require.NoError(t, err)

	// Send 20 concurrent build events
	concurrency := 20
	results := make(chan int, concurrency)

	for i := 0; i < concurrency; i++ {
		go func(progress int) {
			event := WebhookEvent{
				EventType:  "build.progress",
				WorkflowID: workflowID,
				Timestamp:  time.Now(),
				Status:     "building",
				Data: map[string]interface{}{
					"progress": float64(progress),
					"message":  "Building...",
				},
			}

			eventJSON, _ := json.Marshal(event)
			mac := hmac.New(sha256.New, []byte(webhookSecret))
			mac.Write(eventJSON)
			signature := hex.EncodeToString(mac.Sum(nil))

			w := httptest.NewRecorder()
			req := httptest.NewRequest("POST", "/api/osa/webhooks/build-event",
				bytes.NewReader(eventJSON))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-OSA-Signature", signature)

			c, _ := gin.CreateTestContext(w)
			c.Request = req

			handler.HandleBuildEvent(c)
			results <- w.Code
		}(i * 5)
	}

	// Collect results
	successCount := 0
	for i := 0; i < concurrency; i++ {
		if <-results == http.StatusOK {
			successCount++
		}
	}

	// All webhooks should be processed successfully
	assert.Equal(t, concurrency, successCount)

	// Verify all events were stored
	var eventCount int
	err = pool.QueryRow(context.Background(),
		"SELECT COUNT(*) FROM osa_build_events WHERE module_instance_id = $1", appID).Scan(&eventCount)
	require.NoError(t, err)
	assert.Equal(t, concurrency, eventCount)
}
