package terminal

import (
	"log/slog"
	"regexp"
	"strings"
	"sync"
	"unicode"
)

// ValidationResult contains the result of input validation
type ValidationResult struct {
	Valid     bool   // Whether the input is allowed
	Sanitized string // The sanitized input (if applicable)
	Blocked   bool   // Whether the input was completely blocked
	Reason    string // Why the input was blocked/modified
	Severity  string // "info", "warning", "critical"
	RiskScore int    // 0-100 risk assessment
}

// SanitizerConfig holds configuration for the input sanitizer
type SanitizerConfig struct {
	// Enable/disable specific checks
	BlockDangerousCommands bool
	FilterEscapeSequences  bool
	LimitInputLength       bool
	MaxInputLength         int

	// Logging
	LogBlockedCommands bool
	LogSanitizedInput  bool

	// Mode: "block" (reject dangerous), "warn" (allow with warning), "passthrough" (no checks)
	Mode string
}

// DefaultSanitizerConfig returns production-safe defaults
func DefaultSanitizerConfig() *SanitizerConfig {
	return &SanitizerConfig{
		BlockDangerousCommands: true,
		FilterEscapeSequences:  true,
		LimitInputLength:       true,
		MaxInputLength:         4096, // 4KB max input per message
		LogBlockedCommands:     true,
		LogSanitizedInput:      false, // Don't log sanitized input in production (PII)
		Mode:                   "block",
	}
}

// InputSanitizer validates and sanitizes terminal input
type InputSanitizer struct {
	config            *SanitizerConfig
	dangerousPatterns []*dangerousPattern
	escapePatterns    []*regexp.Regexp
	mu                sync.RWMutex
}

// dangerousPattern represents a command pattern to detect
type dangerousPattern struct {
	pattern     *regexp.Regexp
	description string
	severity    string // "critical", "high", "medium", "low"
	riskScore   int    // 0-100
}

// Compiled patterns for performance (initialized once)
var (
	sanitizerOnce   sync.Once
	globalSanitizer *InputSanitizer
)

// GetSanitizer returns the singleton sanitizer instance
func GetSanitizer() *InputSanitizer {
	sanitizerOnce.Do(func() {
		globalSanitizer = NewInputSanitizer(DefaultSanitizerConfig())
	})
	return globalSanitizer
}

// NewInputSanitizer creates a new sanitizer with compiled patterns
func NewInputSanitizer(config *SanitizerConfig) *InputSanitizer {
	s := &InputSanitizer{
		config: config,
	}
	s.compilePatterns()
	return s
}

// compilePatterns pre-compiles all regex patterns for O(1) lookup
func (s *InputSanitizer) compilePatterns() {
	// Dangerous command patterns - ordered by severity
	dangerousCommands := []struct {
		pattern     string
		description string
		severity    string
		riskScore   int
	}{
		// CRITICAL - System destruction
		{`(?i)rm\s+(-[rfv]+\s+)*[/~]`, "Recursive delete from root/home", "critical", 100},
		{`(?i)rm\s+(-[rfv]+\s+)*\*`, "Wildcard delete", "critical", 95},
		{`(?i):\s*\(\)\s*\{\s*:\s*\|\s*:\s*&\s*\}\s*;\s*:`, "Fork bomb", "critical", 100},
		{`(?i)dd\s+.*of\s*=\s*/dev/(sd[a-z]|nvme|hd[a-z])`, "Direct disk write", "critical", 100},
		{`(?i)mkfs`, "Filesystem format", "critical", 100},

		// CRITICAL - Container escape attempts
		{`(?i)/proc/[0-9]+/root`, "Proc filesystem escape attempt", "critical", 100},
		{`(?i)nsenter\s+`, "Namespace escape attempt", "critical", 100},
		{`(?i)docker\s+run.*--privileged`, "Privileged container spawn", "critical", 100},
		{`(?i)docker\s+run.*-v\s+/[^:]*:`, "Docker volume mount escape", "critical", 100},
		{`(?i)/var/run/docker\.sock`, "Docker socket access", "critical", 100},
		{`(?i)mount\s+(-o\s+\w+\s+)?/`, "Mount escape attempt", "critical", 100},

		// HIGH - Dangerous operations
		{`(?i)chmod\s+(-R\s+)?(777|666|000)`, "Insecure permissions", "high", 80},
		{`(?i)chown\s+.*root`, "Ownership change to root", "high", 75},
		{`(?i)curl.*\|\s*(ba)?sh`, "Curl pipe to shell", "high", 85},
		{`(?i)wget.*\|\s*(ba)?sh`, "Wget pipe to shell", "high", 85},
		{`(?i)eval\s+.*\$`, "Eval with variable expansion", "high", 70},
		{`(?i)bash\s+-c\s+.*\$\(`, "Command substitution in bash -c", "high", 70},

		// HIGH - Credential access
		{`(?i)cat\s+.*(\.ssh|passwd|shadow|\.env)`, "Sensitive file read", "high", 80},
		{`(?i)export\s+.*(_KEY|_SECRET|PASSWORD|TOKEN)`, "Secret export", "high", 75},

		// MEDIUM - Suspicious patterns
		{`(?i)nc\s+-[el]`, "Netcat listener", "medium", 60},
		{`(?i)python.*-c.*socket`, "Python reverse shell pattern", "medium", 65},
		{`(?i)base64\s+-d`, "Base64 decode (obfuscation)", "medium", 50},
		{`(?i)history\s*(-c|--clear)`, "History clear attempt", "medium", 55},

		// LOW - Monitoring
		{`(?i)sudo\s+`, "Sudo usage (will fail in container)", "low", 30},
		{`(?i)su\s+-`, "Su usage (will fail in container)", "low", 30},
	}

	s.dangerousPatterns = make([]*dangerousPattern, 0, len(dangerousCommands))
	for _, cmd := range dangerousCommands {
		compiled, err := regexp.Compile(cmd.pattern)
		if err != nil {
			slog.Error("[Sanitizer] Failed to compile pattern", "pattern", cmd.pattern, "error", err)
			continue
		}
		s.dangerousPatterns = append(s.dangerousPatterns, &dangerousPattern{
			pattern:     compiled,
			description: cmd.description,
			severity:    cmd.severity,
			riskScore:   cmd.riskScore,
		})
	}

	// Dangerous ANSI escape sequences that can hijack terminals
	// Reference: https://xtermjs.org/docs/guides/security/
	// NOTE: Patterns updated to NOT block safe terminal sequences (arrow keys, function keys)
	escapePatterns := []string{
		`\x1b\].*\x07`,        // OSC (Operating System Command) - can change title, clipboard
		`\x1b\[[0-9;]+[Hf]`,   // Cursor positioning WITH coordinates (dangerous) - safe Home/End allowed
		`\x1b\[2J`,            // Clear screen (potential UI attack)
		`\x1b\[.*[su]`,        // Save/restore cursor (UI manipulation)
		`\x1b\[[0-9]+[ABCDK]`, // Cursor movement WITH parameters (excessive) - bare arrow keys allowed
		`\x1b\[.*[mp]`,        // SGR (can make text invisible)
		`\x1b\[\?1049[hl]`,    // Alternate screen buffer (can hide content)
		`\x1b\[\?25[hl]`,      // Show/hide cursor (UI attack)
		`\x1b\]52;`,           // Clipboard access (critical!)
		`\x1b\]8;`,            // OSC 8 hyperlink injection (2024 attack vector)
		`\x1b\[[0-9]+J`,       // Erase display with params (bare clear allowed)
		`\x9b`,                // CSI (8-bit) - alternative escape
		`\x9d`,                // OSC (8-bit) - alternative escape
	}

	s.escapePatterns = make([]*regexp.Regexp, 0, len(escapePatterns))
	for _, pattern := range escapePatterns {
		compiled, err := regexp.Compile(pattern)
		if err != nil {
			slog.Error("[Sanitizer] Failed to compile escape pattern", "pattern", pattern, "error", err)
			continue
		}
		s.escapePatterns = append(s.escapePatterns, compiled)
	}

	slog.Info("[Sanitizer] Initialized", "command_patterns", len(s.dangerousPatterns), "escape_patterns", len(s.escapePatterns))
}

// ValidateInput checks input and returns a validation result
func (s *InputSanitizer) ValidateInput(input string, userID string) *ValidationResult {
	result := &ValidationResult{
		Valid:     true,
		Sanitized: input,
		Severity:  "info",
		RiskScore: 0,
	}

	// Passthrough mode - no validation
	if s.config.Mode == "passthrough" {
		return result
	}

	// Check input length first (fast path)
	if s.config.LimitInputLength && len(input) > s.config.MaxInputLength {
		result.Valid = false
		result.Blocked = true
		result.Reason = "Input exceeds maximum length"
		result.Severity = "warning"
		result.RiskScore = 40
		s.logBlocked(userID, "length_exceeded", input[:100], result)
		return result
	}

	// Check for null bytes (potential injection)
	if strings.ContainsRune(input, '\x00') {
		result.Valid = false
		result.Blocked = true
		result.Reason = "Null byte injection detected"
		result.Severity = "critical"
		result.RiskScore = 90
		s.logBlocked(userID, "null_byte_injection", "", result)
		return result
	}

	// Check for dangerous commands
	if s.config.BlockDangerousCommands {
		for _, dp := range s.dangerousPatterns {
			if dp.pattern.MatchString(input) {
				result.RiskScore = dp.riskScore
				result.Severity = dp.severity
				result.Reason = dp.description

				if s.config.Mode == "block" && (dp.severity == "critical" || dp.severity == "high") {
					result.Valid = false
					result.Blocked = true
					s.logBlocked(userID, "dangerous_command", dp.description, result)
					return result
				}
				// In warn mode, continue but log
				s.logBlocked(userID, "dangerous_command_warn", dp.description, result)
			}
		}
	}

	// Filter dangerous escape sequences
	if s.config.FilterEscapeSequences {
		sanitized := s.filterEscapeSequences(input)
		if sanitized != input {
			result.Sanitized = sanitized
			if result.RiskScore < 50 {
				result.RiskScore = 50
			}
			if result.Severity == "info" {
				result.Severity = "warning"
			}
			result.Reason = "Escape sequences filtered"
		}
	}

	// Check for non-printable characters (except common terminal codes)
	result.Sanitized = s.filterNonPrintable(result.Sanitized)

	return result
}

// isSafeEscapeSequence checks if a sequence is a safe terminal control code
func isSafeEscapeSequence(input string) bool {
	// Safe cursor movement (arrow keys - no parameters)
	safeSequences := []string{
		"\x1b[A", // Arrow Up
		"\x1b[B", // Arrow Down
		"\x1b[C", // Arrow Right
		"\x1b[D", // Arrow Left

		// Application cursor keys (alternate mode)
		"\x1bOA", // Up in application mode
		"\x1bOB", // Down in application mode
		"\x1bOC", // Right in application mode
		"\x1bOD", // Left in application mode

		// Safe editing keys (no parameters)
		"\x1b[H",  // Home (no params)
		"\x1b[F",  // End (no params)
		"\x1b[3~", // Delete
		"\x1b[2~", // Insert
		"\x1b[5~", // Page Up
		"\x1b[6~", // Page Down

		// Function keys
		"\x1b[OP",  // F1
		"\x1b[OQ",  // F2
		"\x1b[OR",  // F3
		"\x1b[OS",  // F4
		"\x1b[[A",  // F1 (alternate)
		"\x1b[[B",  // F2 (alternate)
		"\x1b[[C",  // F3 (alternate)
		"\x1b[[D",  // F4 (alternate)
		"\x1b[[E",  // F5 (alternate)
		"\x1b[15~", // F5
		"\x1b[17~", // F6
		"\x1b[18~", // F7
		"\x1b[19~", // F8
		"\x1b[20~", // F9
		"\x1b[21~", // F10
		"\x1b[23~", // F11
		"\x1b[24~", // F12

		// Tab and shift-tab
		"\t",
		"\x1b[Z", // Shift-tab
	}

	for _, safe := range safeSequences {
		if input == safe {
			return true
		}
	}

	return false
}

// filterEscapeSequences removes dangerous ANSI escape sequences
func (s *InputSanitizer) filterEscapeSequences(input string) string {
	// First, check if input is a safe terminal sequence
	if isSafeEscapeSequence(input) {
		return input // Pass through unchanged
	}

	// Otherwise, filter dangerous patterns
	result := input
	for _, pattern := range s.escapePatterns {
		result = pattern.ReplaceAllString(result, "")
	}
	return result
}

// filterNonPrintable removes non-printable characters except valid terminal controls
func (s *InputSanitizer) filterNonPrintable(input string) string {
	var result strings.Builder
	result.Grow(len(input))

	for _, r := range input {
		// Allow printable characters
		if unicode.IsPrint(r) {
			result.WriteRune(r)
			continue
		}
		// Allow specific control characters used in terminals
		switch r {
		case '\n', '\r', '\t', '\b', '\x7f': // newline, carriage return, tab, backspace, delete
			result.WriteRune(r)
		case '\x03': // Ctrl+C (interrupt)
			result.WriteRune(r)
		case '\x04': // Ctrl+D (EOF)
			result.WriteRune(r)
		case '\x1a': // Ctrl+Z (suspend)
			result.WriteRune(r)
		case '\x1b': // ESC (only if part of valid sequence - already filtered)
			result.WriteRune(r)
			// Skip other non-printable characters
		}
	}

	return result.String()
}

// logBlocked logs blocked or suspicious input (sanitized for logs)
func (s *InputSanitizer) logBlocked(userID, reason, detail string, result *ValidationResult) {
	if !s.config.LogBlockedCommands {
		return
	}

	// Truncate and sanitize for logging (don't log full input - PII/security)
	safeDetail := detail
	if len(safeDetail) > 50 {
		safeDetail = safeDetail[:50] + "..."
	}

	// Mask user ID for log privacy (only show first 8 chars)
	maskedUser := userID
	if len(maskedUser) > 8 {
		maskedUser = maskedUser[:8] + "***"
	}

	slog.Warn("[Security] Input blocked",
		"user", maskedUser, "reason", reason,
		"severity", result.Severity, "risk", result.RiskScore, "detail", safeDetail)
}

// UpdateConfig updates the sanitizer configuration thread-safely
func (s *InputSanitizer) UpdateConfig(config *SanitizerConfig) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.config = config
}

// GetConfig returns the current configuration
func (s *InputSanitizer) GetConfig() *SanitizerConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.config
}

// QuickValidate is a fast-path validation for simple inputs
// Returns true if input is safe, false if it needs full validation
func QuickValidate(input string) bool {
	// Fast checks without regex
	if len(input) > 100 {
		return false // Needs full validation
	}

	// Check if it's a safe terminal control sequence (arrow keys, function keys)
	if strings.HasPrefix(input, "\x1b") || strings.HasPrefix(input, "\x9b") || strings.HasPrefix(input, "\x9d") {
		if isSafeEscapeSequence(input) {
			return true // Safe sequence - no full validation needed
		}
		return false // Unknown escape - needs full validation
	}

	// Check for null bytes (still dangerous)
	if strings.ContainsRune(input, '\x00') {
		return false
	}

	// Check for common dangerous command prefixes
	lower := strings.ToLower(strings.TrimSpace(input))
	dangerousPrefixes := []string{
		"rm ", "rm\t", "dd ", "mkfs", "chmod", "chown",
		"curl", "wget", "eval", "sudo", "su -",
		"docker", "nsenter", "/proc/", "nc -",
	}
	for _, prefix := range dangerousPrefixes {
		if strings.HasPrefix(lower, prefix) || strings.Contains(lower, "|"+prefix) {
			return false
		}
	}
	return true
}

// SanitizeInput is a convenience function using the global sanitizer
func SanitizeInput(input string, userID string) *ValidationResult {
	return GetSanitizer().ValidateInput(input, userID)
}
