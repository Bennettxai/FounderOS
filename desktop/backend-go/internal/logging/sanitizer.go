package logging

import (
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"
)

// LogLevel represents logging severity levels
type LogLevel int

const (
	LevelDebug LogLevel = iota
	LevelInfo
	LevelWarn
	LevelError
	LevelSecurity // For security-relevant events
)

var levelNames = map[LogLevel]string{
	LevelDebug:    "DEBUG",
	LevelInfo:     "INFO",
	LevelWarn:     "WARN",
	LevelError:    "ERROR",
	LevelSecurity: "SECURITY",
}

// LogConfig holds logging configuration
type LogConfig struct {
	// Output format: "text" or "json"
	Format string

	// Minimum log level to output
	MinLevel LogLevel

	// Enable sensitive data masking
	MaskSensitiveData bool

	// Session ID mask length (0 = full mask, 8 = show first 8 chars)
	SessionIDMaskLength int

	// Don't log terminal I/O content
	FilterTerminalIO bool

	// Redact these field names in structured logs
	RedactFields []string
}

// DefaultLogConfig returns production-safe defaults
func DefaultLogConfig() *LogConfig {
	return &LogConfig{
		Format:              "text",
		MinLevel:            LevelInfo,
		MaskSensitiveData:   true,
		SessionIDMaskLength: 8,
		FilterTerminalIO:    true,
		RedactFields:        defaultRedactFields,
	}
}

// Default fields to redact in logs
var defaultRedactFields = []string{
	"password", "passwd", "pwd",
	"token", "api_key", "apikey", "api-key",
	"secret", "credential", "auth",
	"session_id", "sessionid", "session-id",
	"bearer", "authorization",
	"private_key", "privatekey", "private-key",
	"cookie", "csrf",
}

// SanitizedLogger is a thread-safe logger that masks sensitive data
type SanitizedLogger struct {
	config         *LogConfig
	mu             sync.RWMutex
	sensitiveRegex []*regexp.Regexp
	output         *log.Logger
}

// Global logger instance
var (
	globalLogger     *SanitizedLogger
	globalLoggerOnce sync.Once
)

// GetLogger returns the global sanitized logger
func GetLogger() *SanitizedLogger {
	globalLoggerOnce.Do(func() {
		globalLogger = NewSanitizedLogger(DefaultLogConfig())
	})
	return globalLogger
}

// SetGlobalConfig updates the global logger configuration
func SetGlobalConfig(config *LogConfig) {
	GetLogger().UpdateConfig(config)
}

// NewSanitizedLogger creates a new sanitized logger
func NewSanitizedLogger(config *LogConfig) *SanitizedLogger {
	l := &SanitizedLogger{
		config: config,
		output: log.New(os.Stderr, "", 0), // No prefix, we add our own
	}
	l.compileSensitivePatterns()
	return l
}

// compileSensitivePatterns creates regex patterns for sensitive data
func (l *SanitizedLogger) compileSensitivePatterns() {
	patterns := []string{
		// Session IDs (UUID format)
		`[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`,
		// Bearer tokens
		`Bearer\s+[A-Za-z0-9\-_\.]+`,
		// JWT tokens
		`eyJ[A-Za-z0-9\-_\.]+`,
		// Base64-ish secrets (long alphanumeric strings)
		`(?:password|secret|token|key|api_key|apikey)\s*[=:]\s*['"]?[A-Za-z0-9+/=]{16,}['"]?`,
		// Email addresses (PII)
		`[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`,
		// IP addresses (partial masking handled separately)
		`\b\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}\b`,
	}

	l.sensitiveRegex = make([]*regexp.Regexp, 0, len(patterns))
	for _, p := range patterns {
		compiled, err := regexp.Compile(`(?i)` + p)
		if err != nil {
			slog.Error("[Logger] Failed to compile pattern", "pattern", p)
			continue
		}
		l.sensitiveRegex = append(l.sensitiveRegex, compiled)
	}
}

// UpdateConfig updates the logger configuration thread-safely
func (l *SanitizedLogger) UpdateConfig(config *LogConfig) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.config = config
	l.compileSensitivePatterns()
}

// Log writes a log entry at the specified level
func (l *SanitizedLogger) Log(level LogLevel, format string, args ...interface{}) {
	l.mu.RLock()
	config := l.config
	l.mu.RUnlock()

	if level < config.MinLevel {
		return
	}

	message := fmt.Sprintf(format, args...)

	// Sanitize message
	if config.MaskSensitiveData {
		message = l.sanitize(message)
	}

	if config.Format == "json" {
		l.logJSON(level, message)
	} else {
		l.logText(level, message)
	}
}

// logText outputs a text-format log line
func (l *SanitizedLogger) logText(level LogLevel, message string) {
	timestamp := time.Now().Format("2006/01/02 15:04:05")
	levelStr := levelNames[level]
	l.output.Printf("%s [%s] %s", timestamp, levelStr, message)
}

// logJSON outputs a JSON-format log line
func (l *SanitizedLogger) logJSON(level LogLevel, message string) {
	entry := map[string]interface{}{
		"timestamp": time.Now().Format(time.RFC3339),
		"level":     levelNames[level],
		"message":   message,
	}

	jsonBytes, err := json.Marshal(entry)
	if err != nil {
		l.output.Printf(`{"error":"failed to marshal log entry"}`)
		return
	}
	l.output.Println(string(jsonBytes))
}

// sanitize removes or masks sensitive data from a string
func (l *SanitizedLogger) sanitize(input string) string {
	result := input

	l.mu.RLock()
	config := l.config
	l.mu.RUnlock()

	// Filter terminal I/O content
	if config.FilterTerminalIO {
		// Remove content that looks like terminal output
		result = filterTerminalContent(result)
	}

	// Mask sensitive patterns
	for _, pattern := range l.sensitiveRegex {
		result = pattern.ReplaceAllStringFunc(result, func(match string) string {
			return maskValue(match, config.SessionIDMaskLength)
		})
	}

	return result
}

// maskValue masks a sensitive value, optionally keeping first N chars
func maskValue(value string, keepFirst int) string {
	if len(value) <= keepFirst {
		return strings.Repeat("*", len(value))
	}
	return value[:keepFirst] + strings.Repeat("*", min(8, len(value)-keepFirst))
}

// filterTerminalContent removes terminal I/O content from log messages
func filterTerminalContent(input string) string {
	// Don't log raw terminal content - replace with placeholder
	if len(input) > 200 && containsTerminalPatterns(input) {
		return "[terminal output filtered]"
	}
	return input
}

// containsTerminalPatterns checks if content looks like terminal output
func containsTerminalPatterns(input string) bool {
	terminalPatterns := []string{
		"\x1b[", // ANSI escape
		"[0m",   // Reset
		"[1m",   // Bold
		"[32m",  // Color codes
		"\r\n",  // CRLF
		"bash-", // Shell prompts
		"$>",    // Prompt
		"#>",    // Root prompt
	}
	for _, pattern := range terminalPatterns {
		if strings.Contains(input, pattern) {
			return true
		}
	}
	return false
}

// Helper functions for convenience

// Debug logs at debug level
func (l *SanitizedLogger) Debug(format string, args ...interface{}) {
	l.Log(LevelDebug, format, args...)
}

// Info logs at info level
func (l *SanitizedLogger) Info(format string, args ...interface{}) {
	l.Log(LevelInfo, format, args...)
}

// Warn logs at warning level
func (l *SanitizedLogger) Warn(format string, args ...interface{}) {
	l.Log(LevelWarn, format, args...)
}

// Error logs at error level
func (l *SanitizedLogger) Error(format string, args ...interface{}) {
	l.Log(LevelError, format, args...)
}

// Security logs security-relevant events
func (l *SanitizedLogger) Security(format string, args ...interface{}) {
	l.Log(LevelSecurity, format, args...)
}

// Package-level convenience functions using global logger

// Debug logs at debug level using global logger
func Debug(format string, args ...interface{}) {
	GetLogger().Debug(format, args...)
}

// Info logs at info level using global logger
func Info(format string, args ...interface{}) {
	GetLogger().Info(format, args...)
}

// Warn logs at warning level using global logger
func Warn(format string, args ...interface{}) {
	GetLogger().Warn(format, args...)
}

// Error logs at error level using global logger
func Error(format string, args ...interface{}) {
	GetLogger().Error(format, args...)
}

// Security logs security events using global logger
func Security(format string, args ...interface{}) {
	GetLogger().Security(format, args...)
}

// MaskSessionID masks a session ID for safe logging
func MaskSessionID(sessionID string) string {
	config := GetLogger().config
	return maskValue(sessionID, config.SessionIDMaskLength)
}

// MaskIP partially masks an IP address for privacy
func MaskIP(ip string) string {
	parts := strings.Split(ip, ".")
	if len(parts) == 4 {
		// Show first two octets, mask last two
		return fmt.Sprintf("%s.%s.xxx.xxx", parts[0], parts[1])
	}
	// IPv6 or other - just show first 8 chars
	if len(ip) > 8 {
		return ip[:8] + "..."
	}
	return ip
}

// SafeLogFields sanitizes a map of fields for logging
func SafeLogFields(fields map[string]interface{}) map[string]interface{} {
	config := GetLogger().config
	result := make(map[string]interface{}, len(fields))

	for key, value := range fields {
		lowerKey := strings.ToLower(key)

		// Check if this field should be redacted
		shouldRedact := false
		for _, redactField := range config.RedactFields {
			if strings.Contains(lowerKey, redactField) {
				shouldRedact = true
				break
			}
		}

		if shouldRedact {
			result[key] = "[REDACTED]"
		} else {
			result[key] = value
		}
	}

	return result
}

// SanitizeURL redacts sensitive parts of a URL (tokens, keys, session IDs in path/query)
func SanitizeURL(rawURL string) string {
	return RedactURLForLogging(rawURL)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
