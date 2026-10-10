package e2b

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// ---- Workspace cleanup -------------------------------------------------------

// CleanupConfig controls which file categories are removed before a workspace
// is uploaded to a sandbox. Security-sensitive files (env files, key material)
// are excluded by default to prevent secret leakage.
type CleanupConfig struct {
	RemoveNodeModules    bool     // removes node_modules/
	RemoveBuildArtifacts bool     // removes dist/, build/, .next/, etc.
	RemoveEnvFiles       bool     // SECURITY: removes .env* and credentials files
	RemoveGitDirectory   bool     // removes .git/ to avoid leaking history
	RemoveDependencyDirs bool     // removes vendor/, target/, __pycache__, etc.
	RemoveCacheFiles     bool     // removes .cache/, .turbo/, .DS_Store, etc.
	CustomPatterns       []string // additional glob patterns (matched against basename)
}

// DefaultCleanupConfig returns a safe default that removes all sensitive
// material and generated artefacts before sandbox upload.
func DefaultCleanupConfig() *CleanupConfig {
	return &CleanupConfig{
		RemoveNodeModules:    true,
		RemoveBuildArtifacts: true,
		RemoveEnvFiles:       true,
		RemoveGitDirectory:   true,
		RemoveDependencyDirs: true,
		RemoveCacheFiles:     true,
	}
}

// WorkspaceCleanupResult summarises what was removed during cleanup.
type WorkspaceCleanupResult struct {
	RemovedPaths []string `json:"removed_paths"`
	SkippedPaths []string `json:"skipped_paths"`
	TotalBytes   int64    `json:"total_bytes"`
	ErrorCount   int      `json:"error_count"`
	Errors       []string `json:"errors,omitempty"`
}

// CleanWorkspace removes sensitive files and large generated artefacts from
// projectPath in-place. If config is nil the DefaultCleanupConfig is used.
// The logger parameter is optional; pass slog.Default() if unsure.
func CleanWorkspace(projectPath string, config *CleanupConfig, logger *slog.Logger) (*WorkspaceCleanupResult, error) {
	if config == nil {
		config = DefaultCleanupConfig()
	}
	if logger == nil {
		logger = slog.Default()
	}

	result := &WorkspaceCleanupResult{
		RemovedPaths: make([]string, 0),
		SkippedPaths: make([]string, 0),
		Errors:       make([]string, 0),
	}

	logger.Info("starting workspace cleanup", "project_path", projectPath)

	patterns := buildCleanupPatterns(config)

	err := filepath.Walk(projectPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			logger.Warn("error accessing path during cleanup", "path", path, "error", err)
			result.SkippedPaths = append(result.SkippedPaths, path)
			return nil
		}

		if path == projectPath {
			return nil
		}

		relPath, err := filepath.Rel(projectPath, path)
		if err != nil {
			logger.Warn("failed to compute relative path", "path", path, "error", err)
			return nil
		}

		if !shouldRemovePath(relPath, info, patterns) {
			return nil
		}

		logger.Debug("removing path", "rel_path", relPath, "is_dir", info.IsDir())

		var size int64
		if info.IsDir() {
			size = dirSize(path)
		} else {
			size = info.Size()
		}
		result.TotalBytes += size

		if removeErr := os.RemoveAll(path); removeErr != nil {
			msg := fmt.Sprintf("failed to remove %s: %v", relPath, removeErr)
			result.Errors = append(result.Errors, msg)
			result.ErrorCount++
			logger.Warn("failed to remove path", "rel_path", relPath, "error", removeErr)
		} else {
			result.RemovedPaths = append(result.RemovedPaths, relPath)
		}

		if info.IsDir() {
			return filepath.SkipDir
		}
		return nil
	})

	if err != nil {
		return result, fmt.Errorf("workspace cleanup: %w", err)
	}

	logger.Info("workspace cleanup completed",
		"removed", len(result.RemovedPaths),
		"errors", result.ErrorCount,
		"total_mb", result.TotalBytes/(1024*1024),
	)
	return result, nil
}

// CreateCleanCopy copies srcPath into a fresh temporary directory, then runs
// CleanWorkspace on the copy. The caller is responsible for removing the
// returned directory when done (defer os.RemoveAll(tmpDir)).
func CreateCleanCopy(srcPath string, config *CleanupConfig, logger *slog.Logger) (tmpDir string, result *WorkspaceCleanupResult, err error) {
	if logger == nil {
		logger = slog.Default()
	}

	tmpDir, err = os.MkdirTemp("", "e2b-workspace-*")
	if err != nil {
		return "", nil, fmt.Errorf("create temp directory: %w", err)
	}

	if copyErr := copyDir(srcPath, tmpDir); copyErr != nil {
		_ = os.RemoveAll(tmpDir)
		return "", nil, fmt.Errorf("copy project to temp directory: %w", copyErr)
	}

	result, err = CleanWorkspace(tmpDir, config, logger)
	if err != nil {
		_ = os.RemoveAll(tmpDir)
		return "", nil, err
	}

	return tmpDir, result, nil
}

// buildCleanupPatterns derives the list of glob patterns to remove from config.
func buildCleanupPatterns(config *CleanupConfig) []string {
	var patterns []string

	if config.RemoveEnvFiles {
		patterns = append(patterns,
			".env", ".env.local", ".env.development", ".env.production", ".env.test", ".env.*",
			"*.env", "credentials.json", "secrets.json",
			"*.key", "*.pem", "*.p12", "*.pfx",
		)
	}
	if config.RemoveNodeModules {
		patterns = append(patterns, "node_modules")
	}
	if config.RemoveBuildArtifacts {
		patterns = append(patterns, "dist", "build", ".next", "out", "*.log", "coverage", ".nyc_output")
	}
	if config.RemoveGitDirectory {
		patterns = append(patterns, ".git")
	}
	if config.RemoveDependencyDirs {
		patterns = append(patterns, "vendor", "target", "__pycache__", "*.pyc", ".pytest_cache", "venv", ".venv", "bin", "obj")
	}
	if config.RemoveCacheFiles {
		patterns = append(patterns, ".cache", ".parcel-cache", ".turbo", ".vercel", ".DS_Store", "Thumbs.db")
	}

	patterns = append(patterns, config.CustomPatterns...)
	return patterns
}

// shouldRemovePath returns true when any component of relPath matches a pattern.
func shouldRemovePath(relPath string, _ os.FileInfo, patterns []string) bool {
	components := strings.Split(filepath.ToSlash(relPath), "/")
	base := filepath.Base(relPath)

	for _, pattern := range patterns {
		for _, component := range components {
			if matched, err := filepath.Match(pattern, component); err == nil && matched {
				return true
			}
		}
		if matched, err := filepath.Match(pattern, base); err == nil && matched {
			return true
		}
	}
	return false
}

// dirSize recursively sums the sizes of all files under path.
func dirSize(path string) int64 {
	var total int64
	_ = filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		total += info.Size()
		return nil
	})
	return total
}

// copyDir recursively copies a directory tree from src to dst, preserving file
// permissions.
func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return fmt.Errorf("rel path computation: %w", err)
		}

		target := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(target, info.Mode())
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read %s: %w", path, err)
		}
		return os.WriteFile(target, data, info.Mode())
	})
}

// ---- Secret redaction --------------------------------------------------------

// SecretPattern matches environment-variable keys that are likely to hold
// sensitive values.
type SecretPattern struct {
	Pattern     *regexp.Regexp
	Description string
}

// DefaultSecretPatterns is the built-in list of patterns used by RedactSecrets.
var DefaultSecretPatterns = []SecretPattern{
	{regexp.MustCompile(`(?i)(api[_-]?key|apikey)`), "API keys"},
	{regexp.MustCompile(`(?i)(secret|secrets)`), "secret values"},
	{regexp.MustCompile(`(?i)(password|passwd|pwd)`), "passwords"},
	{regexp.MustCompile(`(?i)(token|auth[_-]?token|access[_-]?token)`), "auth tokens"},
	{regexp.MustCompile(`(?i)(database[_-]?url|db[_-]?url|connection[_-]?string)`), "database URLs"},
	{regexp.MustCompile(`(?i)(private[_-]?key|priv[_-]?key)`), "private keys"},
	{regexp.MustCompile(`(?i)(jwt[_-]?secret|session[_-]?secret)`), "JWT secrets"},
	{regexp.MustCompile(`(?i)(credentials|credential)`), "credentials"},
	{regexp.MustCompile(`(?i)(encryption[_-]?key|master[_-]?key)`), "encryption keys"},
	{regexp.MustCompile(`(?i)(webhook[_-]?secret|signing[_-]?secret)`), "signing secrets"},
}

// RedactedPlaceholder replaces detected secret values in log output.
const RedactedPlaceholder = "***REDACTED***"

// RedactSecrets returns a copy of envVars with secret values replaced by
// RedactedPlaceholder. The original map is never modified.
func RedactSecrets(envVars map[string]string) map[string]string {
	return RedactSecretsWithPatterns(envVars, DefaultSecretPatterns)
}

// RedactSecretsWithPatterns is like RedactSecrets but uses a caller-supplied
// pattern list instead of DefaultSecretPatterns.
func RedactSecretsWithPatterns(envVars map[string]string, patterns []SecretPattern) map[string]string {
	if envVars == nil {
		return nil
	}

	out := make(map[string]string, len(envVars))
	for k, v := range envVars {
		if isSecretKey(k, patterns) {
			out[k] = RedactedPlaceholder
		} else {
			out[k] = v
		}
	}
	return out
}

// SafeLogEnvVars returns a copy of envVars safe for inclusion in structured
// log attributes. Both key-based and value-based heuristics are applied.
func SafeLogEnvVars(envVars map[string]string) map[string]string {
	redacted := RedactSecrets(envVars)
	for k, v := range redacted {
		if v != RedactedPlaceholder {
			redacted[k] = RedactValue(v)
		}
	}
	return redacted
}

// RedactValue attempts to detect and redact a single secret value regardless
// of the key name. It handles URL credentials, JWT tokens, and long random
// strings.
func RedactValue(value string) string {
	if value == "" {
		return value
	}
	if strings.Contains(value, "://") && strings.Contains(value, "@") {
		return RedactURLCredentials(value)
	}
	if isLikelyJWT(value) {
		return RedactedPlaceholder
	}
	if isLikelySecret(value) {
		return RedactedPlaceholder
	}
	return value
}

// RedactURLCredentials removes the user:password portion from URLs such as
// postgresql://user:pass@host:5432/db, replacing it with ***:***.
func RedactURLCredentials(rawURL string) string {
	re := regexp.MustCompile(`(^[a-zA-Z][a-zA-Z0-9+.-]*://)[^:@]+:[^@]+(@.+)`)
	return re.ReplaceAllString(rawURL, "${1}***:***${2}")
}

// isSecretKey reports whether key matches any of the provided patterns.
func isSecretKey(key string, patterns []SecretPattern) bool {
	for _, p := range patterns {
		if p.Pattern.MatchString(key) {
			return true
		}
	}
	return false
}

// isLikelyJWT returns true when s has the header.payload.signature structure
// of a JWT token.
func isLikelyJWT(s string) bool {
	parts := strings.Split(s, ".")
	if len(parts) != 3 {
		return false
	}
	b64url := regexp.MustCompile(`^[A-Za-z0-9_-]+$`)
	for _, part := range parts {
		if len(part) < 10 || !b64url.MatchString(part) {
			return false
		}
	}
	return true
}

// isLikelySecret returns true when s looks like a long random API key or token
// (20-200 chars, at least 80 % alphanumeric).
func isLikelySecret(s string) bool {
	if len(s) < 20 || len(s) > 200 {
		return false
	}
	var alnum int
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			alnum++
		}
	}
	return float64(alnum)/float64(len(s)) >= 0.8
}

// ---- Redaction stats ---------------------------------------------------------

// RedactionStats summarises how many keys were redacted in a single pass.
type RedactionStats struct {
	TotalKeys     int      `json:"total_keys"`
	RedactedKeys  int      `json:"redacted_keys"`
	RedactedNames []string `json:"redacted_names,omitempty"`
	PreservedKeys int      `json:"preserved_keys"`
}

// GetRedactionStats computes statistics by comparing the original and redacted
// maps.
func GetRedactionStats(original, redacted map[string]string) *RedactionStats {
	stats := &RedactionStats{
		TotalKeys:     len(original),
		RedactedNames: make([]string, 0),
	}
	for k := range original {
		if redacted[k] == RedactedPlaceholder {
			stats.RedactedKeys++
			stats.RedactedNames = append(stats.RedactedNames, k)
		} else {
			stats.PreservedKeys++
		}
	}
	return stats
}
