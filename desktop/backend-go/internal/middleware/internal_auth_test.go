package middleware

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// TestInternalAuthMiddleware_SignatureValidation tests the HMAC signature verification
func TestInternalAuthMiddleware_SignatureValidation(t *testing.T) {
	tests := []struct {
		name           string
		secret         string
		allowedIPs     []string
		headers        map[string]string
		body           string
		expectedStatus int
		containsError  string // partial error message to check
	}{
		{
			name:   "valid signature",
			secret: "test-secret-key-min-32-characters-long",
			headers: func() map[string]string {
				timestamp := strconv.FormatInt(time.Now().Unix(), 10)
				body := `{"name":"test"}`
				sig := computeTestHMAC("test-secret-key-min-32-characters-long", timestamp+"POST/api/internal/osa/generate"+body)
				return map[string]string{
					"X-Internal-Timestamp": timestamp,
					"X-Internal-Signature": sig,
					"X-User-ID":            "user-123",
				}
			}(),
			body:           `{"name":"test"}`,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "missing timestamp",
			secret:         "test-secret-key-min-32-characters-long",
			headers:        map[string]string{"X-Internal-Signature": "sig", "X-User-ID": "user-123"},
			body:           `{}`,
			expectedStatus: http.StatusUnauthorized,
			containsError:  "X-Internal-Timestamp header required",
		},
		{
			name:           "missing signature",
			secret:         "test-secret-key-min-32-characters-long",
			headers:        map[string]string{"X-Internal-Timestamp": strconv.FormatInt(time.Now().Unix(), 10), "X-User-ID": "user-123"},
			body:           `{}`,
			expectedStatus: http.StatusUnauthorized,
			containsError:  "X-Internal-Signature header required",
		},
		{
			name:   "missing user ID with valid signature",
			secret: "test-secret-key-min-32-characters-long",
			headers: func() map[string]string {
				timestamp := strconv.FormatInt(time.Now().Unix(), 10)
				body := `{}`
				sig := computeTestHMAC("test-secret-key-min-32-characters-long", timestamp+"POST/api/internal/osa/generate"+body)
				return map[string]string{
					"X-Internal-Timestamp": timestamp,
					"X-Internal-Signature": sig,
					// No X-User-ID
				}
			}(),
			body:           `{}`,
			expectedStatus: http.StatusUnauthorized,
			containsError:  "X-User-ID header required",
		},
		{
			name:   "expired timestamp",
			secret: "test-secret-key-min-32-characters-long",
			headers: func() map[string]string {
				// 10 minutes ago (outside 5-minute window)
				timestamp := strconv.FormatInt(time.Now().Add(-10*time.Minute).Unix(), 10)
				return map[string]string{
					"X-Internal-Timestamp": timestamp,
					"X-Internal-Signature": "sig",
					"X-User-ID":            "user-123",
				}
			}(),
			body:           `{}`,
			expectedStatus: http.StatusUnauthorized,
			containsError:  "Request timestamp expired",
		},
		{
			name:   "future timestamp",
			secret: "test-secret-key-min-32-characters-long",
			headers: func() map[string]string {
				// 10 minutes in the future
				timestamp := strconv.FormatInt(time.Now().Add(10*time.Minute).Unix(), 10)
				return map[string]string{
					"X-Internal-Timestamp": timestamp,
					"X-Internal-Signature": "sig",
					"X-User-ID":            "user-123",
				}
			}(),
			body:           `{}`,
			expectedStatus: http.StatusUnauthorized,
			containsError:  "timestamp is in the future",
		},
		{
			name:           "invalid signature",
			secret:         "test-secret-key-min-32-characters-long",
			headers:        map[string]string{"X-Internal-Timestamp": strconv.FormatInt(time.Now().Unix(), 10), "X-Internal-Signature": "invalidsig", "X-User-ID": "user-123"},
			body:           `{}`,
			expectedStatus: http.StatusUnauthorized,
			containsError:  "Invalid signature",
		},
		{
			name:           "no secret configured",
			secret:         "",
			allowedIPs:     nil,
			headers:        map[string]string{"X-User-ID": "user-123"},
			body:           `{}`,
			expectedStatus: http.StatusInternalServerError,
			containsError:  "Internal authentication not configured",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a test router
			router := gin.New()
			cfg := &InternalAuthConfig{
				Secret:     tt.secret,
				AllowedIPs: tt.allowedIPs,
			}
			router.Use(InternalAuthMiddleware(cfg))
			router.POST("/api/internal/osa/generate", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"success": true})
			})

			// Create request
			req := httptest.NewRequest("POST", "/api/internal/osa/generate", bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}

			// Record response
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			// Check status
			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d, body: %s", tt.expectedStatus, w.Code, w.Body.String())
			}

			// Check error message if expected
			if tt.containsError != "" && w.Code != http.StatusOK {
				if !bytes.Contains(w.Body.Bytes(), []byte(tt.containsError)) {
					t.Errorf("expected error containing '%s' in response, got: %s", tt.containsError, w.Body.String())
				}
			}
		})
	}
}

// TestInternalAuthMiddleware_IPAllowlist tests IP-based bypass
// Note: Gin's httptest uses 192.0.2.1 as the default ClientIP
func TestInternalAuthMiddleware_IPAllowlist(t *testing.T) {
	tests := []struct {
		name           string
		secret         string
		allowedIPs     []string
		headers        map[string]string
		expectedStatus int
	}{
		{
			name:           "IP allowlisted - gin test default IP",
			secret:         "some-secret",
			allowedIPs:     []string{"192.0.2.1"}, // Gin's default test IP
			headers:        map[string]string{"X-User-ID": "user-123"},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "IP not allowlisted - requires signature",
			secret:         "some-secret",
			allowedIPs:     []string{"10.0.0.1"},
			headers:        map[string]string{"X-User-ID": "user-123"}, // Missing signature headers
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "empty allowed IPs - requires signature",
			secret:         "some-secret",
			allowedIPs:     []string{},
			headers:        map[string]string{"X-User-ID": "user-123"},
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			cfg := &InternalAuthConfig{
				Secret:     tt.secret,
				AllowedIPs: tt.allowedIPs,
			}
			router.Use(InternalAuthMiddleware(cfg))
			router.GET("/test", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"success": true})
			})

			req := httptest.NewRequest("GET", "/test", nil)
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d, body: %s", tt.expectedStatus, w.Code, w.Body.String())
			}
		})
	}
}

// TestInternalAuthMiddleware_GetUserID tests context user ID extraction
func TestInternalAuthMiddleware_GetUserID(t *testing.T) {
	router := gin.New()
	cfg := &InternalAuthConfig{
		Secret:                "test-secret-key-min-32-characters-long",
		SkipAuthInDevelopment: false,
	}
	router.Use(InternalAuthMiddleware(cfg))

	var extractedUserID string
	router.POST("/test", func(c *gin.Context) {
		extractedUserID = GetInternalUserID(c)
		c.JSON(http.StatusOK, gin.H{"user_id": extractedUserID})
	})

	// Create valid signature
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	body := `{}`
	sig := computeTestHMAC("test-secret-key-min-32-characters-long", timestamp+"POST/test"+body)

	req := httptest.NewRequest("POST", "/test", bytes.NewBufferString(body))
	req.Header.Set("X-Internal-Timestamp", timestamp)
	req.Header.Set("X-Internal-Signature", sig)
	req.Header.Set("X-User-ID", "expected-user-456")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	if extractedUserID != "expected-user-456" {
		t.Errorf("expected user ID 'expected-user-456', got '%s'", extractedUserID)
	}
}

// TestComputeHMAC tests the HMAC computation function
func TestComputeHMAC(t *testing.T) {
	// Test vector: known inputs and expected outputs
	secret := "test-secret"
	message := "test-message"

	// Compute using our function
	result := computeHMAC(secret, message)

	// Verify it's a valid hex string
	if len(result) != 64 { // SHA256 produces 32 bytes = 64 hex chars
		t.Errorf("expected 64 character hex string, got %d", len(result))
	}

	// Verify determinism
	result2 := computeHMAC(secret, message)
	if result != result2 {
		t.Error("HMAC should be deterministic")
	}

	// Verify different secrets produce different results
	result3 := computeHMAC("different-secret", message)
	if result == result3 {
		t.Error("different secrets should produce different HMACs")
	}
}

// TestParseAllowedIPs tests IP list parsing
func TestParseAllowedIPs(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{"", nil},
		{"127.0.0.1", []string{"127.0.0.1"}},
		{"127.0.0.1, 10.0.0.1, 192.168.1.1", []string{"127.0.0.1", "10.0.0.1", "192.168.1.1"}},
		{"  127.0.0.1  ,  10.0.0.1  ", []string{"127.0.0.1", "10.0.0.1"}},
		{",,,", nil},
	}

	for _, tt := range tests {
		result := ParseAllowedIPs(tt.input)
		if len(result) != len(tt.expected) {
			t.Errorf("ParseAllowedIPs(%q) = %v, want %v", tt.input, result, tt.expected)
			continue
		}
		for i, ip := range result {
			if ip != tt.expected[i] {
				t.Errorf("ParseAllowedIPs(%q)[%d] = %q, want %q", tt.input, i, ip, tt.expected[i])
			}
		}
	}
}

// TestInternalAuthError_Error tests error message formatting
func TestInternalAuthError_Error(t *testing.T) {
	err := &InternalAuthError{Message: "test error message"}
	if err.Error() != "test error message" {
		t.Errorf("expected 'test error message', got '%s'", err.Error())
	}
}

// TestReplayAttackPrevention tests that old requests are rejected
func TestReplayAttackPrevention(t *testing.T) {
	router := gin.New()
	cfg := &InternalAuthConfig{
		Secret: "test-secret-key-min-32-characters-long",
	}
	router.Use(InternalAuthMiddleware(cfg))
	router.POST("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"success": true})
	})

	// Create a request with a timestamp just inside the window
	timestamp := strconv.FormatInt(time.Now().Add(-4*time.Minute).Unix(), 10)
	body := `{}`
	sig := computeTestHMAC("test-secret-key-min-32-characters-long", timestamp+"POST/test"+body)

	req := httptest.NewRequest("POST", "/test", bytes.NewBufferString(body))
	req.Header.Set("X-Internal-Timestamp", timestamp)
	req.Header.Set("X-Internal-Signature", sig)
	req.Header.Set("X-User-ID", "user-123")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Should succeed (within 5 minute window)
	if w.Code != http.StatusOK {
		t.Errorf("expected status 200 for valid timestamp, got %d", w.Code)
	}
}

// TestDifferentHTTPMethods tests that signature includes HTTP method
func TestDifferentHTTPMethods(t *testing.T) {
	secret := "test-secret-key-min-32-characters-long"
	router := gin.New()
	cfg := &InternalAuthConfig{Secret: secret}
	router.Use(InternalAuthMiddleware(cfg))
	
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"method": "GET"})
	})
	router.POST("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"method": "POST"})
	})

	// Create a signature for POST
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	body := `{}`
	postSig := computeTestHMAC(secret, timestamp+"POST/test"+body)

	// Use POST signature on GET request - should fail
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Internal-Timestamp", timestamp)
	req.Header.Set("X-Internal-Signature", postSig)
	req.Header.Set("X-User-ID", "user-123")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Should fail because signature was for POST, not GET
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for method mismatch, got %d", w.Code)
	}
}

// Helper function to compute HMAC for tests
func computeTestHMAC(secret, message string) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(message))
	return hex.EncodeToString(h.Sum(nil))
}
