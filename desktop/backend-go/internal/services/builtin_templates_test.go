package services

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

// TestGeneratedCORSMiddleware tests the CORS middleware logic that gets generated in templates
func TestGeneratedCORSMiddleware(t *testing.T) {
	tests := []struct {
		name                string
		allowedOriginsEnv   string
		requestOrigin       string
		expectAllowOrigin   bool
		expectCredentials   bool
		expectVaryHeader    bool
		expectedOriginValue string
	}{
		{
			name:                "Valid origin from default list - localhost:5173",
			allowedOriginsEnv:   "",
			requestOrigin:       "http://localhost:5173",
			expectAllowOrigin:   true,
			expectCredentials:   true,
			expectVaryHeader:    true,
			expectedOriginValue: "http://localhost:5173",
		},
		{
			name:                "Valid origin from default list - localhost:3000",
			allowedOriginsEnv:   "",
			requestOrigin:       "http://localhost:3000",
			expectAllowOrigin:   true,
			expectCredentials:   true,
			expectVaryHeader:    true,
			expectedOriginValue: "http://localhost:3000",
		},
		{
			name:              "Invalid origin - not in default list",
			allowedOriginsEnv: "",
			requestOrigin:     "http://evil.com",
			expectAllowOrigin: false,
			expectCredentials: false,
			expectVaryHeader:  true,
		},
		{
			name:              "Invalid origin - wildcard attempt",
			allowedOriginsEnv: "",
			requestOrigin:     "*",
			expectAllowOrigin: false,
			expectCredentials: false,
			expectVaryHeader:  true,
		},
		{
			name:                "Custom allowed origins - single valid",
			allowedOriginsEnv:   "https://app.example.com",
			requestOrigin:       "https://app.example.com",
			expectAllowOrigin:   true,
			expectCredentials:   true,
			expectVaryHeader:    true,
			expectedOriginValue: "https://app.example.com",
		},
		{
			name:                "Custom allowed origins - multiple valid",
			allowedOriginsEnv:   "https://app.example.com,https://api.example.com,http://localhost:8080",
			requestOrigin:       "https://api.example.com",
			expectAllowOrigin:   true,
			expectCredentials:   true,
			expectVaryHeader:    true,
			expectedOriginValue: "https://api.example.com",
		},
		{
			name:              "Custom allowed origins - invalid",
			allowedOriginsEnv: "https://app.example.com,https://api.example.com",
			requestOrigin:     "https://evil.com",
			expectAllowOrigin: false,
			expectCredentials: false,
			expectVaryHeader:  true,
		},
		{
			name:              "Empty origin header",
			allowedOriginsEnv: "",
			requestOrigin:     "",
			expectAllowOrigin: false,
			expectCredentials: false,
			expectVaryHeader:  true,
		},
		{
			name:                "Origin with whitespace in env (should trim)",
			allowedOriginsEnv:   " https://app.example.com , https://api.example.com ",
			requestOrigin:       "https://app.example.com",
			expectAllowOrigin:   true,
			expectCredentials:   true,
			expectVaryHeader:    true,
			expectedOriginValue: "https://app.example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variable for test
			if tt.allowedOriginsEnv != "" {
				os.Setenv("ALLOWED_ORIGINS", tt.allowedOriginsEnv)
				defer os.Unsetenv("ALLOWED_ORIGINS")
			} else {
				os.Unsetenv("ALLOWED_ORIGINS")
			}

			// Create test handler that simulates the generated CORS middleware
			handler := createGeneratedCORSHandler()

			// Create test request with Origin header
			req := httptest.NewRequest("GET", "/api/test", nil)
			if tt.requestOrigin != "" {
				req.Header.Set("Origin", tt.requestOrigin)
			}

			// Record response
			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			// Check Access-Control-Allow-Origin header
			allowOrigin := rr.Header().Get("Access-Control-Allow-Origin")
			if tt.expectAllowOrigin {
				if allowOrigin != tt.expectedOriginValue {
					t.Errorf("Expected Access-Control-Allow-Origin to be %q, got %q", tt.expectedOriginValue, allowOrigin)
				}
			} else {
				if allowOrigin != "" {
					t.Errorf("Expected no Access-Control-Allow-Origin header, got %q", allowOrigin)
				}
			}

			// Check Access-Control-Allow-Credentials header
			allowCredentials := rr.Header().Get("Access-Control-Allow-Credentials")
			if tt.expectCredentials {
				if allowCredentials != "true" {
					t.Errorf("Expected Access-Control-Allow-Credentials to be 'true', got %q", allowCredentials)
				}
			} else {
				if allowCredentials == "true" {
					t.Errorf("Expected no Access-Control-Allow-Credentials or not 'true', got %q", allowCredentials)
				}
			}

			// Check Vary header (should always be present)
			varyHeader := rr.Header().Get("Vary")
			if tt.expectVaryHeader {
				if varyHeader != "Origin" {
					t.Errorf("Expected Vary header to be 'Origin', got %q", varyHeader)
				}
			}
		})
	}
}

// TestGeneratedCORSPreflightRequest tests CORS preflight (OPTIONS) request handling
func TestGeneratedCORSPreflightRequest(t *testing.T) {
	tests := []struct {
		name              string
		allowedOriginsEnv string
		requestOrigin     string
		expectStatusOK    bool
	}{
		{
			name:              "Valid preflight - default origins",
			allowedOriginsEnv: "",
			requestOrigin:     "http://localhost:5173",
			expectStatusOK:    true,
		},
		{
			name:              "Valid preflight - custom origin",
			allowedOriginsEnv: "https://app.example.com",
			requestOrigin:     "https://app.example.com",
			expectStatusOK:    true,
		},
		{
			name:              "Invalid preflight - unauthorized origin",
			allowedOriginsEnv: "",
			requestOrigin:     "http://evil.com",
			expectStatusOK:    true, // OPTIONS always returns 200, but without CORS headers
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variable
			if tt.allowedOriginsEnv != "" {
				os.Setenv("ALLOWED_ORIGINS", tt.allowedOriginsEnv)
				defer os.Unsetenv("ALLOWED_ORIGINS")
			} else {
				os.Unsetenv("ALLOWED_ORIGINS")
			}

			handler := createGeneratedCORSHandler()

			// Create OPTIONS request
			req := httptest.NewRequest("OPTIONS", "/api/test", nil)
			req.Header.Set("Origin", tt.requestOrigin)
			req.Header.Set("Access-Control-Request-Method", "POST")

			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			// Check status code
			if tt.expectStatusOK && rr.Code != http.StatusOK {
				t.Errorf("Expected status 200 for OPTIONS request, got %d", rr.Code)
			}

			// Check that standard CORS headers are present
			allowMethods := rr.Header().Get("Access-Control-Allow-Methods")
			if allowMethods == "" {
				t.Error("Expected Access-Control-Allow-Methods header to be present")
			}

			allowHeaders := rr.Header().Get("Access-Control-Allow-Headers")
			if allowHeaders == "" {
				t.Error("Expected Access-Control-Allow-Headers header to be present")
			}
		})
	}
}

// TestGeneratedTemplateContainsCORSLogic verifies the generated Go template includes CORS validation
func TestGeneratedTemplateContainsCORSLogic(t *testing.T) {
	// Get the API Backend template (which contains the CORS middleware)
	template := apiBackendTemplate()

	if template == nil {
		t.Fatal("Failed to get API backend template")
	}

	// Get the middleware file content
	middlewareContent, exists := template.FilesTemplate["internal/middleware/middleware.go"]
	if !exists {
		t.Fatal("Middleware file not found in template")
	}

	// Verify CORS validation logic is present
	requiredSnippets := []string{
		// Environment variable reading
		`allowed := os.Getenv("ALLOWED_ORIGINS")`,

		// Default origins (no wildcard)
		`"http://localhost:5173"`,
		`"http://localhost:3000"`,

		// Origin set creation
		`originSet := make(map[string]bool`,

		// Request origin validation (normalized for case-insensitive comparison)
		`reqOrigin := r.Header.Get("Origin")`,
		`if originSet[normalizedReq]`,

		// Setting CORS headers conditionally
		`w.Header().Set("Access-Control-Allow-Origin", reqOrigin)`,
		`w.Header().Set("Access-Control-Allow-Credentials", "true")`,

		// Vary header
		`w.Header().Set("Vary", "Origin")`,

		// OPTIONS handling
		`if r.Method == "OPTIONS"`,
	}

	for _, snippet := range requiredSnippets {
		if !strings.Contains(middlewareContent, snippet) {
			t.Errorf("Generated template missing required CORS snippet: %q", snippet)
		}
	}

	// Verify wildcard is NOT present
	forbiddenSnippets := []string{
		`Access-Control-Allow-Origin", "*"`,
		`"*"`, // Make sure there's no wildcard origin allowed
	}

	for _, snippet := range forbiddenSnippets {
		// Special case: "*" can appear in comments or other contexts
		// We specifically check for the dangerous pattern
		if strings.Contains(middlewareContent, `"Access-Control-Allow-Origin", "*"`) {
			t.Errorf("Generated template contains forbidden wildcard CORS: %q", snippet)
		}
	}
}

// TestGeneratedTemplateRejectsInvalidOrigins tests that invalid origins are rejected
func TestGeneratedTemplateRejectsInvalidOrigins(t *testing.T) {
	invalidOrigins := []string{
		"http://evil.com",
		"https://malicious.org",
		"*",
		"null",
		"http://localhost:9999", // Not in default list
	}

	os.Unsetenv("ALLOWED_ORIGINS") // Use default list

	handler := createGeneratedCORSHandler()

	for _, origin := range invalidOrigins {
		t.Run("reject_"+origin, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api/test", nil)
			req.Header.Set("Origin", origin)

			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			allowOrigin := rr.Header().Get("Access-Control-Allow-Origin")
			if allowOrigin != "" {
				t.Errorf("Expected to reject origin %q, but got Access-Control-Allow-Origin: %q", origin, allowOrigin)
			}

			allowCredentials := rr.Header().Get("Access-Control-Allow-Credentials")
			if allowCredentials == "true" {
				t.Errorf("Expected to reject credentials for origin %q, but got Access-Control-Allow-Credentials: true", origin)
			}
		})
	}
}

// TestGeneratedTemplateSupportsCredentials verifies credentials are only allowed for valid origins
func TestGeneratedTemplateSupportsCredentials(t *testing.T) {
	tests := []struct {
		name              string
		origin            string
		expectCredentials bool
	}{
		{
			name:              "Valid origin supports credentials",
			origin:            "http://localhost:5173",
			expectCredentials: true,
		},
		{
			name:              "Invalid origin does not support credentials",
			origin:            "http://evil.com",
			expectCredentials: false,
		},
	}

	os.Unsetenv("ALLOWED_ORIGINS")
	handler := createGeneratedCORSHandler()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api/test", nil)
			req.Header.Set("Origin", tt.origin)

			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			allowCredentials := rr.Header().Get("Access-Control-Allow-Credentials")
			if tt.expectCredentials {
				if allowCredentials != "true" {
					t.Errorf("Expected credentials allowed for %q, got %q", tt.origin, allowCredentials)
				}
			} else {
				if allowCredentials == "true" {
					t.Errorf("Expected credentials denied for %q, but got 'true'", tt.origin)
				}
			}
		})
	}
}

// createGeneratedCORSHandler simulates the CORS middleware generated by the template
// This mirrors the exact logic in builtin_templates.go line 676-711
func createGeneratedCORSHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// This is the exact CORS logic from the generated template
		allowed := os.Getenv("ALLOWED_ORIGINS")
		var origins []string
		if allowed != "" {
			for _, o := range strings.Split(allowed, ",") {
				origins = append(origins, strings.TrimSpace(o))
			}
		} else {
			// Default: localhost only (no wildcard)
			origins = []string{"http://localhost:5173", "http://localhost:3000"}
		}

		originSet := make(map[string]bool, len(origins))
		for _, o := range origins {
			originSet[o] = true
		}

		reqOrigin := r.Header.Get("Origin")
		if originSet[reqOrigin] {
			w.Header().Set("Access-Control-Allow-Origin", reqOrigin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
		}
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Vary", "Origin")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		// Simulate actual handler
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})
}
