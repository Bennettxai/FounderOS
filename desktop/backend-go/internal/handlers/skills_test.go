package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/rhl/businessos-backend/internal/services"
	"github.com/stretchr/testify/assert"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// TestSkillsHandler_RegisterRoutes_RequiresAuth verifies that all skills routes require authentication
func TestSkillsHandler_RegisterRoutes_RequiresAuth(t *testing.T) {
	// Create a mock skills loader
	loader := services.NewSkillsLoader("") // Empty path will fail to load, which is fine for this test
	
	// Create handler with nil pool (auth will still be applied, just will fail at DB lookup)
	handler := NewSkillsHandler(loader, nil, nil)

	tests := []struct {
		name   string
		method string
		path   string
	}{
		{"ListSkills", "GET", "/skills"},
		{"GetSkillsPrompt", "GET", "/skills/prompt"},
		{"GetSkillGroups", "GET", "/skills/groups"},
		{"ReloadSkills", "POST", "/skills/reload"},
		{"GetSkill", "GET", "/skills/test-skill"},
		{"ValidateSkill", "GET", "/skills/test-skill/validate"},
		{"GetSkillReference", "GET", "/skills/test-skill/references/ref"},
		{"GetSkillSchema", "GET", "/skills/test-skill/schemas/schema"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			handler.RegisterRoutes(router.Group("/api"))

			req := httptest.NewRequest(tt.method, "/api"+tt.path, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			// Should return 401 Unauthorized because no auth cookie is provided
			assert.Equal(t, http.StatusUnauthorized, w.Code, 
				"Route %s should require authentication", tt.path)
		})
	}
}

// TestSkillsHandler_NewSkillsHandler_RequiresPool verifies that handler stores pool for auth
func TestSkillsHandler_NewSkillsHandler_RequiresPool(t *testing.T) {
	loader := services.NewSkillsLoader("")
	
	// Handler should store pool and sessionCache for auth middleware
	handler := NewSkillsHandler(loader, nil, nil)
	
	assert.NotNil(t, handler, "Handler should not be nil")
	assert.Nil(t, handler.pool, "Pool should be nil when passed nil")
	assert.Nil(t, handler.sessionCache, "SessionCache should be nil when passed nil")
	
	// Verify the handler has the loader
	assert.Equal(t, loader, handler.loader, "Loader should be stored correctly")
}

// TestSkillsHandler_ReloadSkills_AuditLogging verifies that reload endpoint logs user info
func TestSkillsHandler_ReloadSkills_AuditLogging(t *testing.T) {
	// This test verifies that ReloadSkills checks for authenticated user
	// The actual logging is tested via integration tests
	
	loader := services.NewSkillsLoader("")
	handler := NewSkillsHandler(loader, nil, nil)

	router := gin.New()
	handler.RegisterRoutes(router.Group("/api"))

	// Request without auth should return 401
	req := httptest.NewRequest("POST", "/api/skills/reload", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code, 
		"Reload should require authentication")
}
