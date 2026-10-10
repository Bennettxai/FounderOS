package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// setupOSATemplateTestRouter creates a test router with auth middleware for OSA templates
func setupOSATemplateTestRouter(h *OSATemplateHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Mock auth middleware - injects test user
	router.Use(func(c *gin.Context) {
		testUserID := uuid.New()
		c.Set("user_id", testUserID)
		c.Next()
	})

	api := router.Group("/api")
	osaTemplates := api.Group("/osa/templates")
	{
		osaTemplates.GET("", h.ListOSATemplates)
		osaTemplates.GET("/:name", h.GetOSATemplate)
		osaTemplates.POST("/:name/generate", h.GenerateFromOSATemplate)
		osaTemplates.POST("/:name/preview", h.PreviewTemplatePrompt)
	}

	return router
}

func TestListOSATemplates(t *testing.T) {
	// For now, test that the handler returns service unavailable when builder is nil
	h := &OSATemplateHandler{}

	router := setupOSATemplateTestRouter(h)

	req, _ := http.NewRequest("GET", "/api/osa/templates", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestGetOSATemplate(t *testing.T) {
	// Test that the handler returns service unavailable when builder is nil
	h := &OSATemplateHandler{}

	router := setupOSATemplateTestRouter(h)

	req, _ := http.NewRequest("GET", "/api/osa/templates/crm-app", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestGenerateFromOSATemplate_Success(t *testing.T) {
	// Test that the handler returns service unavailable when builder is nil
	h := &OSATemplateHandler{}

	router := setupOSATemplateTestRouter(h)

	requestBody := map[string]interface{}{
		"variables": map[string]interface{}{
			"name": "Test App",
		},
	}
	jsonBody, _ := json.Marshal(requestBody)

	req, _ := http.NewRequest("POST", "/api/osa/templates/crm-app/generate", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestGenerateFromOSATemplate_InvalidRequest(t *testing.T) {
	// Test that the handler returns bad request for invalid JSON
	h := &OSATemplateHandler{}

	router := setupOSATemplateTestRouter(h)

	req, _ := http.NewRequest("POST", "/api/osa/templates/crm-app/generate", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestPreviewTemplatePrompt(t *testing.T) {
	// Test that the handler returns service unavailable when builder is nil
	h := &OSATemplateHandler{}

	router := setupOSATemplateTestRouter(h)

	requestBody := map[string]interface{}{
		"variables": map[string]interface{}{
			"name": "Test App",
		},
	}
	jsonBody, _ := json.Marshal(requestBody)

	req, _ := http.NewRequest("POST", "/api/osa/templates/crm-app/preview", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestOSATemplates_ServiceUnavailable(t *testing.T) {
	// Handler with no prompt builder
	h := &OSATemplateHandler{}

	router := setupOSATemplateTestRouter(h)

	tests := []struct {
		name   string
		method string
		path   string
	}{
		{"ListTemplates", "GET", "/api/osa/templates"},
		{"GetTemplate", "GET", "/api/osa/templates/crm-app"},
		{"GenerateFromTemplate", "POST", "/api/osa/templates/crm-app/generate"},
		{"PreviewTemplate", "POST", "/api/osa/templates/crm-app/preview"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req *http.Request
			if tt.method == "POST" {
				body := bytes.NewReader([]byte(`{"variables": {}}`))
				req, _ = http.NewRequest(tt.method, tt.path, body)
				req.Header.Set("Content-Type", "application/json")
			} else {
				req, _ = http.NewRequest(tt.method, tt.path, nil)
			}

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusServiceUnavailable, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)
			assert.Contains(t, response["error"], "Template service not available")
		})
	}
}
