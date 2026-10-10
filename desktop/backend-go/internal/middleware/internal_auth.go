package middleware

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	// InternalAPISecretEnv is the environment variable for the shared secret.
	InternalAPISecretEnv = "INTERNAL_API_SECRET"

	// InternalAllowedIPsEnv is the environment variable for allowed IPs.
	InternalAllowedIPsEnv = "INTERNAL_ALLOWED_IPS"

	// InternalUserIDKey is the context key for the validated user ID.
	InternalUserIDKey = "internal_user_id"

	internalTimestampHeader = "X-Internal-Timestamp"
	internalSignatureHeader = "X-Internal-Signature"
	internalUserIDHeader    = "X-User-ID"
	internalMaxSkew         = 5 * time.Minute
)

// InternalAuthConfig holds configuration for internal auth middleware.
type InternalAuthConfig struct {
	Secret                string
	AllowedIPs            []string
	SkipAuthInDevelopment bool
}

// InternalAuthError represents an internal authentication error.
type InternalAuthError struct {
	Message string
}

func (e *InternalAuthError) Error() string {
	return e.Message
}

// Sentinel errors (kept for API compatibility).
var (
	ErrMissingTimestamp    = &InternalAuthError{Message: "X-Internal-Timestamp header required"}
	ErrMissingSignature    = &InternalAuthError{Message: "X-Internal-Signature header required"}
	ErrInvalidTimestamp    = &InternalAuthError{Message: "Invalid timestamp format"}
	ErrTimestampExpired    = &InternalAuthError{Message: "Request timestamp expired"}
	ErrTimestampInFuture   = &InternalAuthError{Message: "Request timestamp is in the future"}
	ErrSignatureMismatch   = &InternalAuthError{Message: "Invalid signature"}
	ErrBodyRead            = &InternalAuthError{Message: "Failed to read request body"}
	ErrIPNotAllowlisted    = &InternalAuthError{Message: "IP not in allowlist"}
	ErrSecretNotConfigured = &InternalAuthError{Message: "Internal API secret not configured"}
)

// InternalAuthMiddleware verifies internal service-to-service requests with
// either an allowlisted source IP or an HMAC signature over:
//
//	timestamp + method + path + body
func InternalAuthMiddleware(cfg *InternalAuthConfig) gin.HandlerFunc {
	if cfg == nil {
		cfg = &InternalAuthConfig{}
	}
	return func(c *gin.Context) {
		userID := strings.TrimSpace(c.GetHeader(internalUserIDHeader))
		if userID == "" {
			abortInternalAuth(c, http.StatusUnauthorized, "X-User-ID header required")
			return
		}

		if cfg.SkipAuthInDevelopment {
			c.Set(InternalUserIDKey, userID)
			c.Next()
			return
		}

		if ipAllowed(c.ClientIP(), cfg.AllowedIPs) {
			c.Set(InternalUserIDKey, userID)
			c.Next()
			return
		}

		if cfg.Secret == "" {
			abortInternalAuth(c, http.StatusInternalServerError, "Internal authentication not configured")
			return
		}

		timestamp := strings.TrimSpace(c.GetHeader(internalTimestampHeader))
		if timestamp == "" {
			abortInternalAuth(c, http.StatusUnauthorized, ErrMissingTimestamp.Message)
			return
		}
		signature := strings.TrimSpace(c.GetHeader(internalSignatureHeader))
		if signature == "" {
			abortInternalAuth(c, http.StatusUnauthorized, ErrMissingSignature.Message)
			return
		}

		requestTime, err := parseUnixTimestamp(timestamp)
		if err != nil {
			abortInternalAuth(c, http.StatusUnauthorized, ErrInvalidTimestamp.Message)
			return
		}
		now := time.Now()
		if requestTime.Before(now.Add(-internalMaxSkew)) {
			abortInternalAuth(c, http.StatusUnauthorized, ErrTimestampExpired.Message)
			return
		}
		if requestTime.After(now.Add(internalMaxSkew)) {
			abortInternalAuth(c, http.StatusUnauthorized, ErrTimestampInFuture.Message)
			return
		}

		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			abortInternalAuth(c, http.StatusUnauthorized, ErrBodyRead.Message)
			return
		}
		c.Request.Body = io.NopCloser(bytes.NewReader(body))

		message := timestamp + c.Request.Method + c.Request.URL.Path + string(body)
		expected := computeHMAC(cfg.Secret, message)
		if !hmac.Equal([]byte(expected), []byte(signature)) {
			abortInternalAuth(c, http.StatusUnauthorized, ErrSignatureMismatch.Message)
			return
		}

		c.Set(InternalUserIDKey, userID)
		c.Next()
	}
}

// GetInternalUserID retrieves the internal user ID from context.
func GetInternalUserID(c *gin.Context) string {
	userID, exists := c.Get(InternalUserIDKey)
	if !exists {
		return ""
	}
	return userID.(string)
}

// MustGetInternalUserID retrieves the internal user ID or returns empty string.
func MustGetInternalUserID(c *gin.Context) string {
	return GetInternalUserID(c)
}

// ParseAllowedIPs parses a comma-separated list of allowed IPs.
func ParseAllowedIPs(ipList string) []string {
	if ipList == "" {
		return nil
	}
	ips := strings.Split(ipList, ",")
	result := make([]string, 0, len(ips))
	for _, ip := range ips {
		trimmed := strings.TrimSpace(ip)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func parseUnixTimestamp(value string) (time.Time, error) {
	seconds, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return time.Time{}, err
	}
	return time.Unix(seconds, 0), nil
}

func computeHMAC(secret, message string) string {
	h := hmac.New(sha256.New, []byte(secret))
	_, _ = h.Write([]byte(message))
	return hex.EncodeToString(h.Sum(nil))
}

func ipAllowed(clientIP string, allowedIPs []string) bool {
	if len(allowedIPs) == 0 {
		return false
	}
	parsedClient := net.ParseIP(clientIP)
	for _, allowed := range allowedIPs {
		if allowed == clientIP {
			return true
		}
		if _, ipNet, err := net.ParseCIDR(allowed); err == nil && parsedClient != nil && ipNet.Contains(parsedClient) {
			return true
		}
	}
	return false
}

func abortInternalAuth(c *gin.Context, status int, message string) {
	c.AbortWithStatusJSON(status, gin.H{"error": message})
}

// NewInternalAuthConfigFromEnv creates an InternalAuthConfig from environment variables.
func NewInternalAuthConfigFromEnv() *InternalAuthConfig {
	secret := os.Getenv(InternalAPISecretEnv)
	allowedIPs := ParseAllowedIPs(os.Getenv(InternalAllowedIPsEnv))
	env := os.Getenv("ENVIRONMENT")

	return &InternalAuthConfig{
		Secret:                secret,
		AllowedIPs:            allowedIPs,
		SkipAuthInDevelopment: env == "development" && secret == "" && len(allowedIPs) == 0,
	}
}
