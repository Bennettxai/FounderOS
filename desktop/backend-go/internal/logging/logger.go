package logging

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// MaskEmail masks an email address for safe logging
// Shows first character + ***@domain.com
func MaskEmail(email string) string {
	if email == "" {
		return ""
	}

	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		// Invalid email format - mask entirely
		return "***@***"
	}

	localPart := parts[0]
	domain := parts[1]

	if len(localPart) == 0 {
		return "***@" + domain
	}

	// Show first character only
	return string(localPart[0]) + "***@" + domain
}

// MaskToken completely masks a token/secret, showing only type prefix
func MaskToken(token string) string {
	if token == "" {
		return ""
	}

	// For JWT tokens (eyJ prefix), show prefix
	if strings.HasPrefix(token, "eyJ") {
		return "eyJ***[JWT_REDACTED]"
	}

	// For Bearer tokens
	if strings.HasPrefix(token, "Bearer ") {
		return "Bearer ***[TOKEN_REDACTED]"
	}

	// For base64-looking tokens (long alphanumeric)
	if len(token) > 16 {
		return token[:4] + "***[TOKEN_REDACTED]"
	}

	// Short tokens - completely mask
	return "***[REDACTED]"
}

// SecretPatterns contains regex patterns for detecting secrets
var SecretPatterns = []*regexp.Regexp{
	// AWS Access Keys
	regexp.MustCompile(`(?i)(A3T[A-Z0-9]|AKIA|AGPA|AIDA|AROA|AIPA|ANPA|ANVA|ASIA)[A-Z0-9]{16}`),

	// GitHub Personal Access Token
	regexp.MustCompile(`ghp_[0-9a-zA-Z]{36}`),
	regexp.MustCompile(`gho_[0-9a-zA-Z]{36}`),
	regexp.MustCompile(`github_pat_[0-9a-zA-Z_]{82}`),

	// Generic API keys (long alphanumeric strings in key=value format)
	regexp.MustCompile(`(?i)(api[_-]?key|apikey|api[_-]?token|access[_-]?token|auth[_-]?token|secret[_-]?key|password)\s*[:=]\s*['"]?([A-Za-z0-9+/=_-]{20,})['"]?`),

	// Base64 encoded secrets (very long base64 strings)
	regexp.MustCompile(`(?i)(secret|password|token|key)\s*[:=]\s*['"]?([A-Za-z0-9+/=]{40,})['"]?`),

	// Private keys
	regexp.MustCompile(`-----BEGIN\s+(?:RSA\s+)?PRIVATE\s+KEY-----`),

	// OAuth tokens
	regexp.MustCompile(`ya29\.[0-9A-Za-z\-_]+`), // Google OAuth

	// Slack tokens
	regexp.MustCompile(`xox[baprs]-[0-9a-zA-Z]{10,48}`),

	// Generic long alphanumeric tokens (32+ chars)
	regexp.MustCompile(`\b[A-Za-z0-9_-]{32,}\b`),
}

// DetectAndRedactSecrets scans text for common secret patterns and redacts them
func DetectAndRedactSecrets(text string) (sanitized string, detected bool) {
	sanitized = text
	detected = false

	for _, pattern := range SecretPatterns {
		if pattern.MatchString(sanitized) {
			detected = true
			sanitized = pattern.ReplaceAllString(sanitized, "[SECRET_REDACTED]")
		}
	}

	return sanitized, detected
}

// StructuredLog creates a structured log entry with automatic sanitization
type StructuredLog struct {
	Timestamp string                 `json:"timestamp"`
	Level     string                 `json:"level"`
	Message   string                 `json:"message"`
	Fields    map[string]interface{} `json:"fields,omitempty"`
}

// NewStructuredLog creates a new structured log entry
func NewStructuredLog(level LogLevel, message string, fields map[string]interface{}) *StructuredLog {
	return &StructuredLog{
		Timestamp: time.Now().Format(time.RFC3339),
		Level:     levelNames[level],
		Message:   message,
		Fields:    SafeLogFields(fields),
	}
}

// JSON returns the JSON representation of the log entry
func (sl *StructuredLog) JSON() string {
	bytes, err := json.Marshal(sl)
	if err != nil {
		return fmt.Sprintf(`{"error":"failed to marshal log entry: %v"}`, err)
	}
	return string(bytes)
}

// LogWithFields logs a message with structured fields
func LogWithFields(level LogLevel, message string, fields map[string]interface{}) {
	logger := GetLogger()

	logger.mu.RLock()
	format := logger.config.Format
	logger.mu.RUnlock()

	if format == "json" {
		structuredLog := NewStructuredLog(level, message, fields)
		logger.Log(level, "%s", structuredLog.JSON())
	} else {
		// Text format - append key=value pairs
		var fieldStrs []string
		safeFields := SafeLogFields(fields)
		for k, v := range safeFields {
			fieldStrs = append(fieldStrs, fmt.Sprintf("%s=%v", k, v))
		}
		if len(fieldStrs) > 0 {
			logger.Log(level, "%s | %s", message, strings.Join(fieldStrs, " "))
		} else {
			logger.Log(level, "%s", message)
		}
	}
}

// InfoWithFields logs at info level with structured fields
func InfoWithFields(message string, fields map[string]interface{}) {
	LogWithFields(LevelInfo, message, fields)
}

// ErrorWithFields logs at error level with structured fields
func ErrorWithFields(message string, fields map[string]interface{}) {
	LogWithFields(LevelError, message, fields)
}

// SecurityWithFields logs security events with structured fields
func SecurityWithFields(message string, fields map[string]interface{}) {
	LogWithFields(LevelSecurity, message, fields)
}

// DebugWithFields logs at debug level with structured fields
func DebugWithFields(message string, fields map[string]interface{}) {
	LogWithFields(LevelDebug, message, fields)
}

// SanitizeSQL redacts SQL query parameters for logging
func SanitizeSQL(query string) string {
	// Replace common SQL patterns with placeholders
	patterns := map[string]string{
		// String literals
		`'[^']*'`:           "'[REDACTED]'",
		// Numeric parameters
		`\$\d+\s*=\s*\d+`:   "$N = [REDACTED]",
		// Email patterns in WHERE clauses
		`email\s*=\s*'[^']*'`: "email = '[REDACTED]'",
		// Token patterns
		`token\s*=\s*'[^']*'`: "token = '[REDACTED]'",
	}

	result := query
	for pattern, replacement := range patterns {
		re := regexp.MustCompile(pattern)
		result = re.ReplaceAllString(result, replacement)
	}

	return result
}

// RedactURLForLogging redacts sensitive information from URLs for safe logging
func RedactURLForLogging(rawURL string) string {
	// Remove query parameters that might contain tokens
	if idx := strings.Index(rawURL, "?"); idx != -1 {
		return rawURL[:idx] + "?[PARAMS_REDACTED]"
	}

	// Redact paths that contain tokens or IDs
	if strings.Contains(rawURL, "/token/") || strings.Contains(rawURL, "/session/") {
		parts := strings.Split(rawURL, "/")
		for i := range parts {
			if i > 0 && (parts[i-1] == "token" || parts[i-1] == "session") {
				parts[i] = "[REDACTED]"
			}
		}
		return strings.Join(parts, "/")
	}

	return rawURL
}

// SanitizeCookies redacts cookie values for logging
func SanitizeCookies(cookies string) string {
	if cookies == "" {
		return ""
	}

	// Split by semicolon (cookie separator)
	cookieParts := strings.Split(cookies, ";")
	var sanitized []string

	for _, cookie := range cookieParts {
		cookie = strings.TrimSpace(cookie)
		if idx := strings.Index(cookie, "="); idx != -1 {
			name := cookie[:idx]
			// Always redact cookie values
			sanitized = append(sanitized, name+"=[REDACTED]")
		}
	}

	return strings.Join(sanitized, "; ")
}

// MaskUserID masks a user ID for logging (similar to session ID)
func MaskUserID(userID string) string {
	return MaskSessionID(userID)
}

// SecurityEvent logs a security event with standard fields
type SecurityEvent struct {
	EventType   string
	UserID      string
	IP          string
	Description string
	Severity    string // "low", "medium", "high", "critical"
	Metadata    map[string]interface{}
}

// LogSecurityEvent logs a structured security event
func LogSecurityEvent(event SecurityEvent) {
	fields := map[string]interface{}{
		"event_type":  event.EventType,
		"user_id":     MaskUserID(event.UserID),
		"ip":          MaskIP(event.IP),
		"severity":    event.Severity,
		"description": event.Description,
	}

	// Add any additional metadata
	for k, v := range event.Metadata {
		fields[k] = v
	}

	SecurityWithFields(fmt.Sprintf("[SECURITY] %s: %s", event.EventType, event.Description), fields)
}

// HTTPRequestLog logs HTTP request details with sanitization
type HTTPRequestLog struct {
	Method     string
	Path       string
	UserAgent  string
	IP         string
	UserID     string
	StatusCode int
	Duration   time.Duration
}

// LogHTTPRequest logs HTTP request with automatic sanitization
func LogHTTPRequest(req HTTPRequestLog) {
	fields := map[string]interface{}{
		"method":      req.Method,
		"path":        RedactURLForLogging(req.Path),
		"user_agent":  req.UserAgent,
		"ip":          MaskIP(req.IP),
		"status_code": req.StatusCode,
		"duration_ms": req.Duration.Milliseconds(),
	}

	if req.UserID != "" {
		fields["user_id"] = MaskUserID(req.UserID)
	}

	level := LevelInfo
	if req.StatusCode >= 500 {
		level = LevelError
	} else if req.StatusCode >= 400 {
		level = LevelWarn
	}

	LogWithFields(level, fmt.Sprintf("%s %s - %d", req.Method, RedactURLForLogging(req.Path), req.StatusCode), fields)
}
