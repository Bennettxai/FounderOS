package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// setupTestContext creates a Gin test context for middleware testing
func setupTestContext() (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest(http.MethodGet, "/test", nil)
	return c, w
}

// TestRequireAuth_WithAuthenticatedUser verifies RequireAuth allows authenticated requests
func TestRequireAuth_WithAuthenticatedUser(t *testing.T) {
	c, w := setupTestContext()

	// Set authenticated user in context
	user := &BetterAuthUser{
		ID:            "test-user-123",
		Name:          "Test User",
		Email:         "test@example.com",
		EmailVerified: true,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
	c.Set(UserContextKey, user)

	// Execute middleware
	middleware := RequireAuth()
	middleware(c)

	// Assert: request was NOT aborted (c.Next() was called)
	assert.False(t, c.IsAborted(), "RequireAuth should not abort request when user is authenticated")
	assert.Equal(t, http.StatusOK, w.Code, "Response code should be 200 when user is authenticated")
}

// TestRequireAuth_WithoutAuthenticatedUser verifies RequireAuth aborts unauthenticated requests
func TestRequireAuth_WithoutAuthenticatedUser(t *testing.T) {
	c, w := setupTestContext()

	// No user set in context (unauthenticated)

	// Execute middleware
	middleware := RequireAuth()
	middleware(c)

	// Assert: request was aborted with 401
	assert.True(t, c.IsAborted(), "RequireAuth should abort request when user is missing")
	assert.Equal(t, http.StatusUnauthorized, w.Code, "Response code should be 401 when user is missing")

	// Assert: response contains proper error structure
	assert.Contains(t, w.Body.String(), "Authentication required", "Response should contain error message")
	assert.Contains(t, w.Body.String(), "UNAUTHENTICATED", "Response should contain error code")
}

// TestRequireAuth_WithNilUser verifies RequireAuth handles nil user
func TestRequireAuth_WithNilUser(t *testing.T) {
	c, w := setupTestContext()

	// Explicitly set nil user
	c.Set(UserContextKey, nil)

	// Execute middleware
	middleware := RequireAuth()
	middleware(c)

	// Assert: request was aborted with 401
	assert.True(t, c.IsAborted(), "RequireAuth should abort request when user is nil")
	assert.Equal(t, http.StatusUnauthorized, w.Code, "Response code should be 401 when user is nil")
}

// TestMustGetCurrentUser_WithAuthenticatedUser verifies MustGetCurrentUser returns user
func TestMustGetCurrentUser_WithAuthenticatedUser(t *testing.T) {
	c, _ := setupTestContext()

	// Set authenticated user in context
	expectedUser := &BetterAuthUser{
		ID:            "test-user-456",
		Name:          "Another User",
		Email:         "another@example.com",
		EmailVerified: false,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
	c.Set(UserContextKey, expectedUser)

	// Execute helper
	user := MustGetCurrentUser(c)

	// Assert: returned user matches expected
	assert.NotNil(t, user, "MustGetCurrentUser should return non-nil user")
	assert.Equal(t, expectedUser.ID, user.ID, "User ID should match")
	assert.Equal(t, expectedUser.Email, user.Email, "User email should match")
	assert.Equal(t, expectedUser.Name, user.Name, "User name should match")
}

// TestMustGetCurrentUser_WithoutAuthenticatedUser verifies MustGetCurrentUser returns 500
func TestMustGetCurrentUser_WithoutAuthenticatedUser(t *testing.T) {
	c, w := setupTestContext()

	// No user set in context (unauthenticated)

	// Execute helper
	user := MustGetCurrentUser(c)

	// Assert: returns nil, sets HTTP 500, and aborts
	assert.Nil(t, user, "MustGetCurrentUser should return nil when user is missing")
	assert.Equal(t, http.StatusInternalServerError, w.Code, "Should return 500 status")
	assert.True(t, c.IsAborted(), "Should abort request")
	assert.Contains(t, w.Body.String(), "authentication middleware misconfiguration", "Should contain error message")
}

// TestMustGetCurrentUser_WithNilUser verifies MustGetCurrentUser returns 500 on nil
func TestMustGetCurrentUser_WithNilUser(t *testing.T) {
	c, w := setupTestContext()

	// Explicitly set nil user
	c.Set(UserContextKey, nil)

	// Execute helper
	user := MustGetCurrentUser(c)

	// Assert: returns nil, sets HTTP 500, and aborts
	assert.Nil(t, user, "MustGetCurrentUser should return nil when user is nil")
	assert.Equal(t, http.StatusInternalServerError, w.Code, "Should return 500 status")
	assert.True(t, c.IsAborted(), "Should abort request")
	assert.Contains(t, w.Body.String(), "authentication middleware misconfiguration", "Should contain error message")
}

// TestGetCurrentUser_WithAuthenticatedUser verifies GetCurrentUser returns user
func TestGetCurrentUser_WithAuthenticatedUser(t *testing.T) {
	c, _ := setupTestContext()

	// Set authenticated user in context
	expectedUser := &BetterAuthUser{
		ID:    "test-user-789",
		Name:  "Third User",
		Email: "third@example.com",
	}
	c.Set(UserContextKey, expectedUser)

	// Execute helper
	user := GetCurrentUser(c)

	// Assert: returned user matches expected
	assert.NotNil(t, user, "GetCurrentUser should return non-nil user")
	assert.Equal(t, expectedUser.ID, user.ID, "User ID should match")
}

// TestGetCurrentUser_WithoutAuthenticatedUser verifies GetCurrentUser returns nil
func TestGetCurrentUser_WithoutAuthenticatedUser(t *testing.T) {
	c, _ := setupTestContext()

	// No user set in context

	// Execute helper
	user := GetCurrentUser(c)

	// Assert: returns nil
	assert.Nil(t, user, "GetCurrentUser should return nil when user is missing")
}

// TestRequireAuth_MiddlewareChaining verifies RequireAuth works in chain
func TestRequireAuth_MiddlewareChaining(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Setup router with middleware chain
	router := gin.New()

	// Mock auth middleware that sets user
	mockAuth := func(c *gin.Context) {
		user := &BetterAuthUser{
			ID:    "chain-test-user",
			Email: "chain@example.com",
		}
		c.Set(UserContextKey, user)
		c.Next()
	}

	// Protected route with RequireAuth
	router.GET("/protected", mockAuth, RequireAuth(), func(c *gin.Context) {
		user := MustGetCurrentUser(c)
		c.JSON(http.StatusOK, gin.H{"user_id": user.ID})
	})

	// Test authenticated request
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code, "Protected route should return 200 with auth")
	assert.Contains(t, w.Body.String(), "chain-test-user", "Response should contain user ID")
}

// TestRequireAuth_MiddlewareChaining_NoAuth verifies RequireAuth blocks unauthenticated
func TestRequireAuth_MiddlewareChaining_NoAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Setup router WITHOUT auth middleware
	router := gin.New()

	// Protected route with RequireAuth (no auth middleware before it)
	router.GET("/protected", RequireAuth(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	// Test unauthenticated request
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code, "Protected route should return 401 without auth")
	assert.Contains(t, w.Body.String(), "Authentication required", "Response should contain error")
}
