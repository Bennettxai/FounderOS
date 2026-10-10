package services

import (
	"log/slog"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestProtectionService creates a ModuleProtectionService with a nil pool
// suitable for calling check* methods that do not hit the database.
func newTestProtectionService() *ModuleProtectionService {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	return &ModuleProtectionService{
		pool:   nil,
		logger: logger.With("component", "module_protection"),
	}
}

// newEmptyResult returns a fresh ValidationResult with Allowed=true and no violations.
func newEmptyResult() *ValidationResult {
	return &ValidationResult{Allowed: true}
}

// hasViolationWithSeverity reports whether result contains at least one violation with the given severity.
func hasViolationWithSeverity(result *ValidationResult, severity string) bool {
	for _, v := range result.Violations {
		if v.Severity == severity {
			return true
		}
	}
	return false
}

// ─────────────────────────────────────────────────────────────────
// matchesPattern
// ─────────────────────────────────────────────────────────────────

func TestMatchesPattern(t *testing.T) {
	t.Run("exact match returns true", func(t *testing.T) {
		assert.True(t, matchesPattern("users", "users"))
	})

	t.Run("exact match is case-sensitive", func(t *testing.T) {
		assert.False(t, matchesPattern("Users", "users"))
	})

	t.Run("no match returns false", func(t *testing.T) {
		assert.False(t, matchesPattern("orders", "users"))
	})

	t.Run("empty target and empty pattern match", func(t *testing.T) {
		assert.True(t, matchesPattern("", ""))
	})

	t.Run("empty target does not match non-empty pattern", func(t *testing.T) {
		assert.False(t, matchesPattern("", "users"))
	})

	t.Run("non-empty target does not match empty pattern", func(t *testing.T) {
		assert.False(t, matchesPattern("users", ""))
	})

	// Prefix wildcard: "api/*"
	t.Run("prefix wildcard matches deeper path", func(t *testing.T) {
		assert.True(t, matchesPattern("api/v1/users", "api/*"))
	})

	t.Run("prefix wildcard matches direct child", func(t *testing.T) {
		assert.True(t, matchesPattern("api/users", "api/*"))
	})

	t.Run("prefix wildcard matches prefix itself (no trailing slash)", func(t *testing.T) {
		assert.True(t, matchesPattern("api", "api/*"))
	})

	t.Run("prefix wildcard does not match unrelated path", func(t *testing.T) {
		assert.False(t, matchesPattern("internal/users", "api/*"))
	})

	// Suffix wildcard: "*.go"
	t.Run("suffix wildcard matches file with correct extension", func(t *testing.T) {
		assert.True(t, matchesPattern("handler.go", "*.go"))
	})

	t.Run("suffix wildcard matches nested file with correct extension", func(t *testing.T) {
		assert.True(t, matchesPattern("internal/handler.go", "*.go"))
	})

	t.Run("suffix wildcard does not match wrong extension", func(t *testing.T) {
		assert.False(t, matchesPattern("handler.ts", "*.go"))
	})

	t.Run("suffix wildcard does not match no extension", func(t *testing.T) {
		assert.False(t, matchesPattern("handler", "*.go"))
	})

	// General glob (contains "*" in middle)
	t.Run("glob pattern with middle wildcard matches", func(t *testing.T) {
		assert.True(t, matchesPattern("api/v1/health", "api/*/health"))
	})

	t.Run("glob pattern with multiple wildcards matches", func(t *testing.T) {
		assert.True(t, matchesPattern("api/v1/users/list", "api/*/users/*"))
	})

	t.Run("glob pattern does not match when part is absent", func(t *testing.T) {
		assert.False(t, matchesPattern("api/v1/products", "api/*/users"))
	})
}

// ─────────────────────────────────────────────────────────────────
// checkSchemaProtection
// ─────────────────────────────────────────────────────────────────

func TestCheckSchemaProtection(t *testing.T) {
	svc := newTestProtectionService()

	t.Run("delete on protected schema creates error violation", func(t *testing.T) {
		manifest := &ModuleProtectionManifest{
			ProtectedSchemas: []string{"public"},
		}
		change := ChangeRequest{
			ChangeType: "schema",
			Target:     "public",
			Operation:  "delete",
		}
		result := newEmptyResult()

		svc.checkSchemaProtection(manifest, change, result)

		require.Len(t, result.Violations, 1)
		v := result.Violations[0]
		assert.Equal(t, "protected_schema", v.Rule)
		assert.Equal(t, "public", v.Target)
		assert.Equal(t, "error", v.Severity)
		assert.Contains(t, v.Message, "public")
		assert.Empty(t, result.Warnings)
	})

	t.Run("update on protected schema creates warning (not error)", func(t *testing.T) {
		manifest := &ModuleProtectionManifest{
			ProtectedSchemas: []string{"public"},
		}
		change := ChangeRequest{
			ChangeType: "schema",
			Target:     "public",
			Operation:  "update",
		}
		result := newEmptyResult()

		svc.checkSchemaProtection(manifest, change, result)

		assert.Empty(t, result.Violations)
		require.Len(t, result.Warnings, 1)
		assert.Contains(t, result.Warnings[0], "public")
	})

	t.Run("create on protected schema produces no violation and no warning", func(t *testing.T) {
		manifest := &ModuleProtectionManifest{
			ProtectedSchemas: []string{"public"},
		}
		change := ChangeRequest{
			ChangeType: "schema",
			Target:     "public",
			Operation:  "create",
		}
		result := newEmptyResult()

		svc.checkSchemaProtection(manifest, change, result)

		assert.Empty(t, result.Violations)
		assert.Empty(t, result.Warnings)
	})

	t.Run("delete on non-protected schema produces no violation", func(t *testing.T) {
		manifest := &ModuleProtectionManifest{
			ProtectedSchemas: []string{"public"},
		}
		change := ChangeRequest{
			ChangeType: "schema",
			Target:     "audit",
			Operation:  "delete",
		}
		result := newEmptyResult()

		svc.checkSchemaProtection(manifest, change, result)

		assert.Empty(t, result.Violations)
		assert.Empty(t, result.Warnings)
	})

	t.Run("delete on schema matched by wildcard pattern creates error violation", func(t *testing.T) {
		manifest := &ModuleProtectionManifest{
			ProtectedSchemas: []string{"app/*"},
		}
		change := ChangeRequest{
			ChangeType: "schema",
			Target:     "app/users",
			Operation:  "delete",
		}
		result := newEmptyResult()

		svc.checkSchemaProtection(manifest, change, result)

		require.Len(t, result.Violations, 1)
		assert.Equal(t, "error", result.Violations[0].Severity)
	})

	t.Run("no protected schemas means no violations", func(t *testing.T) {
		manifest := &ModuleProtectionManifest{
			ProtectedSchemas: []string{},
		}
		change := ChangeRequest{
			ChangeType: "schema",
			Target:     "public",
			Operation:  "delete",
		}
		result := newEmptyResult()

		svc.checkSchemaProtection(manifest, change, result)

		assert.Empty(t, result.Violations)
		assert.Empty(t, result.Warnings)
	})

	t.Run("multiple protected schemas — only matching one fires", func(t *testing.T) {
		manifest := &ModuleProtectionManifest{
			ProtectedSchemas: []string{"private", "public"},
		}
		change := ChangeRequest{
			ChangeType: "schema",
			Target:     "public",
			Operation:  "delete",
		}
		result := newEmptyResult()

		svc.checkSchemaProtection(manifest, change, result)

		require.Len(t, result.Violations, 1)
		assert.Equal(t, "public", result.Violations[0].Target)
	})
}

// ─────────────────────────────────────────────────────────────────
// checkRouteProtection
// ─────────────────────────────────────────────────────────────────

func TestCheckRouteProtection(t *testing.T) {
	svc := newTestProtectionService()

	t.Run("any modification to a protected route creates error violation", func(t *testing.T) {
		manifest := &ModuleProtectionManifest{
			ProtectedRoutes: []string{"/api/v1/health"},
		}
		change := ChangeRequest{
			ChangeType: "route",
			Target:     "/api/v1/health",
			Operation:  "update",
		}
		result := newEmptyResult()

		svc.checkRouteProtection(manifest, change, result)

		require.Len(t, result.Violations, 1)
		v := result.Violations[0]
		assert.Equal(t, "protected_route", v.Rule)
		assert.Equal(t, "/api/v1/health", v.Target)
		assert.Equal(t, "error", v.Severity)
		assert.Contains(t, v.Message, "/api/v1/health")
	})

	t.Run("delete on protected route creates error violation", func(t *testing.T) {
		manifest := &ModuleProtectionManifest{
			ProtectedRoutes: []string{"/api/v1/health"},
		}
		change := ChangeRequest{
			ChangeType: "route",
			Target:     "/api/v1/health",
			Operation:  "delete",
		}
		result := newEmptyResult()

		svc.checkRouteProtection(manifest, change, result)

		require.Len(t, result.Violations, 1)
		assert.Equal(t, "error", result.Violations[0].Severity)
	})

	t.Run("create on protected route creates error violation", func(t *testing.T) {
		manifest := &ModuleProtectionManifest{
			ProtectedRoutes: []string{"/api/v1/health"},
		}
		change := ChangeRequest{
			ChangeType: "route",
			Target:     "/api/v1/health",
			Operation:  "create",
		}
		result := newEmptyResult()

		svc.checkRouteProtection(manifest, change, result)

		require.Len(t, result.Violations, 1)
		assert.Equal(t, "error", result.Violations[0].Severity)
	})

	t.Run("route not in protected list produces no violation", func(t *testing.T) {
		manifest := &ModuleProtectionManifest{
			ProtectedRoutes: []string{"/api/v1/health"},
		}
		change := ChangeRequest{
			ChangeType: "route",
			Target:     "/api/v1/metrics",
			Operation:  "update",
		}
		result := newEmptyResult()

		svc.checkRouteProtection(manifest, change, result)

		assert.Empty(t, result.Violations)
		assert.Empty(t, result.Warnings)
	})

	t.Run("route matched via prefix wildcard creates error violation", func(t *testing.T) {
		manifest := &ModuleProtectionManifest{
			ProtectedRoutes: []string{"/api/*"},
		}
		change := ChangeRequest{
			ChangeType: "route",
			Target:     "/api/v1/users",
			Operation:  "update",
		}
		result := newEmptyResult()

		svc.checkRouteProtection(manifest, change, result)

		require.Len(t, result.Violations, 1)
		assert.Equal(t, "error", result.Violations[0].Severity)
	})

	t.Run("no protected routes means no violations", func(t *testing.T) {
		manifest := &ModuleProtectionManifest{
			ProtectedRoutes: []string{},
		}
		change := ChangeRequest{
			ChangeType: "route",
			Target:     "/api/v1/users",
			Operation:  "delete",
		}
		result := newEmptyResult()

		svc.checkRouteProtection(manifest, change, result)

		assert.Empty(t, result.Violations)
	})
}

// ─────────────────────────────────────────────────────────────────
// checkOperationProtection
// ─────────────────────────────────────────────────────────────────

func TestCheckOperationProtection(t *testing.T) {
	svc := newTestProtectionService()

	t.Run("operating on a protected operation creates error violation", func(t *testing.T) {
		manifest := &ModuleProtectionManifest{
			ProtectedOps: []string{"send_email"},
		}
		change := ChangeRequest{
			ChangeType: "operation",
			Target:     "send_email",
			Operation:  "update",
		}
		result := newEmptyResult()

		svc.checkOperationProtection(manifest, change, result)

		require.Len(t, result.Violations, 1)
		v := result.Violations[0]
		assert.Equal(t, "protected_operation", v.Rule)
		assert.Equal(t, "send_email", v.Target)
		assert.Equal(t, "error", v.Severity)
		assert.Contains(t, v.Message, "send_email")
	})

	t.Run("delete on protected operation creates error violation", func(t *testing.T) {
		manifest := &ModuleProtectionManifest{
			ProtectedOps: []string{"send_email"},
		}
		change := ChangeRequest{
			ChangeType: "operation",
			Target:     "send_email",
			Operation:  "delete",
		}
		result := newEmptyResult()

		svc.checkOperationProtection(manifest, change, result)

		require.Len(t, result.Violations, 1)
		assert.Equal(t, "error", result.Violations[0].Severity)
	})

	t.Run("operation not in protected list produces no violation", func(t *testing.T) {
		manifest := &ModuleProtectionManifest{
			ProtectedOps: []string{"send_email"},
		}
		change := ChangeRequest{
			ChangeType: "operation",
			Target:     "send_sms",
			Operation:  "update",
		}
		result := newEmptyResult()

		svc.checkOperationProtection(manifest, change, result)

		assert.Empty(t, result.Violations)
		assert.Empty(t, result.Warnings)
	})

	t.Run("protected operation matched by suffix wildcard creates error", func(t *testing.T) {
		manifest := &ModuleProtectionManifest{
			ProtectedOps: []string{"*.send"},
		}
		change := ChangeRequest{
			ChangeType: "operation",
			Target:     "email.send",
			Operation:  "update",
		}
		result := newEmptyResult()

		svc.checkOperationProtection(manifest, change, result)

		require.Len(t, result.Violations, 1)
		assert.Equal(t, "error", result.Violations[0].Severity)
	})

	t.Run("no protected operations means no violations", func(t *testing.T) {
		manifest := &ModuleProtectionManifest{
			ProtectedOps: []string{},
		}
		change := ChangeRequest{
			ChangeType: "operation",
			Target:     "send_email",
			Operation:  "delete",
		}
		result := newEmptyResult()

		svc.checkOperationProtection(manifest, change, result)

		assert.Empty(t, result.Violations)
	})
}

// ─────────────────────────────────────────────────────────────────
// checkFileProtection
// ─────────────────────────────────────────────────────────────────

func TestCheckFileProtection(t *testing.T) {
	svc := newTestProtectionService()

	t.Run("delete on protected file creates error violation", func(t *testing.T) {
		manifest := &ModuleProtectionManifest{
			ProtectedFiles: []string{"config/settings.json"},
		}
		change := ChangeRequest{
			ChangeType: "file",
			Target:     "config/settings.json",
			Operation:  "delete",
		}
		result := newEmptyResult()

		svc.checkFileProtection(manifest, change, result)

		require.Len(t, result.Violations, 1)
		v := result.Violations[0]
		assert.Equal(t, "protected_file", v.Rule)
		assert.Equal(t, "config/settings.json", v.Target)
		assert.Equal(t, "error", v.Severity)
		assert.Contains(t, v.Message, "config/settings.json")
		assert.Empty(t, result.Warnings)
	})

	t.Run("update on protected file creates warning (not error)", func(t *testing.T) {
		manifest := &ModuleProtectionManifest{
			ProtectedFiles: []string{"config/settings.json"},
		}
		change := ChangeRequest{
			ChangeType: "file",
			Target:     "config/settings.json",
			Operation:  "update",
		}
		result := newEmptyResult()

		svc.checkFileProtection(manifest, change, result)

		assert.Empty(t, result.Violations)
		require.Len(t, result.Warnings, 1)
		assert.Contains(t, result.Warnings[0], "config/settings.json")
	})

	t.Run("create on protected file creates warning (not error)", func(t *testing.T) {
		manifest := &ModuleProtectionManifest{
			ProtectedFiles: []string{"config/settings.json"},
		}
		change := ChangeRequest{
			ChangeType: "file",
			Target:     "config/settings.json",
			Operation:  "create",
		}
		result := newEmptyResult()

		svc.checkFileProtection(manifest, change, result)

		assert.Empty(t, result.Violations)
		require.Len(t, result.Warnings, 1)
	})

	t.Run("delete on non-protected file produces no violation", func(t *testing.T) {
		manifest := &ModuleProtectionManifest{
			ProtectedFiles: []string{"config/settings.json"},
		}
		change := ChangeRequest{
			ChangeType: "file",
			Target:     "config/other.json",
			Operation:  "delete",
		}
		result := newEmptyResult()

		svc.checkFileProtection(manifest, change, result)

		assert.Empty(t, result.Violations)
		assert.Empty(t, result.Warnings)
	})

	t.Run("delete on file matched by glob pattern creates error violation", func(t *testing.T) {
		manifest := &ModuleProtectionManifest{
			ProtectedFiles: []string{"*.go"},
		}
		change := ChangeRequest{
			ChangeType: "file",
			Target:     "main.go",
			Operation:  "delete",
		}
		result := newEmptyResult()

		svc.checkFileProtection(manifest, change, result)

		require.Len(t, result.Violations, 1)
		assert.Equal(t, "error", result.Violations[0].Severity)
	})

	t.Run("update on file matched by glob pattern creates warning", func(t *testing.T) {
		manifest := &ModuleProtectionManifest{
			ProtectedFiles: []string{"*.go"},
		}
		change := ChangeRequest{
			ChangeType: "file",
			Target:     "handler.go",
			Operation:  "update",
		}
		result := newEmptyResult()

		svc.checkFileProtection(manifest, change, result)

		assert.Empty(t, result.Violations)
		require.Len(t, result.Warnings, 1)
	})

	t.Run("no protected files means no violations", func(t *testing.T) {
		manifest := &ModuleProtectionManifest{
			ProtectedFiles: []string{},
		}
		change := ChangeRequest{
			ChangeType: "file",
			Target:     "main.go",
			Operation:  "delete",
		}
		result := newEmptyResult()

		svc.checkFileProtection(manifest, change, result)

		assert.Empty(t, result.Violations)
		assert.Empty(t, result.Warnings)
	})
}

// ─────────────────────────────────────────────────────────────────
// ValidationResult.Allowed logic
// ─────────────────────────────────────────────────────────────────

// TestValidationResultAllowed verifies that Allowed is false when any violation
// has severity "error", and true when only warnings (or no violations) exist.
// We exercise this through the check* helpers and then apply the same logic
// that ValidateChange applies internally.
func TestValidationResultAllowed(t *testing.T) {
	svc := newTestProtectionService()

	applyAllowedLogic := func(result *ValidationResult) {
		for _, v := range result.Violations {
			if v.Severity == "error" {
				result.Allowed = false
				break
			}
		}
	}

	t.Run("Allowed is false when there is an error violation", func(t *testing.T) {
		manifest := &ModuleProtectionManifest{
			ProtectedSchemas: []string{"public"},
		}
		change := ChangeRequest{
			ChangeType: "schema",
			Target:     "public",
			Operation:  "delete",
		}
		result := newEmptyResult()

		svc.checkSchemaProtection(manifest, change, result)
		applyAllowedLogic(result)

		assert.False(t, result.Allowed)
		assert.True(t, hasViolationWithSeverity(result, "error"))
	})

	t.Run("Allowed remains true when only a warning exists (schema update)", func(t *testing.T) {
		manifest := &ModuleProtectionManifest{
			ProtectedSchemas: []string{"public"},
		}
		change := ChangeRequest{
			ChangeType: "schema",
			Target:     "public",
			Operation:  "update",
		}
		result := newEmptyResult()

		svc.checkSchemaProtection(manifest, change, result)
		applyAllowedLogic(result)

		assert.True(t, result.Allowed)
		assert.Empty(t, result.Violations)
		assert.NotEmpty(t, result.Warnings)
	})

	t.Run("Allowed remains true when only a warning exists (file modify)", func(t *testing.T) {
		manifest := &ModuleProtectionManifest{
			ProtectedFiles: []string{"config.json"},
		}
		change := ChangeRequest{
			ChangeType: "file",
			Target:     "config.json",
			Operation:  "update",
		}
		result := newEmptyResult()

		svc.checkFileProtection(manifest, change, result)
		applyAllowedLogic(result)

		assert.True(t, result.Allowed)
		assert.Empty(t, result.Violations)
	})

	t.Run("Allowed is false when route error violation is present", func(t *testing.T) {
		manifest := &ModuleProtectionManifest{
			ProtectedRoutes: []string{"/api/v1/users"},
		}
		change := ChangeRequest{
			ChangeType: "route",
			Target:     "/api/v1/users",
			Operation:  "update",
		}
		result := newEmptyResult()

		svc.checkRouteProtection(manifest, change, result)
		applyAllowedLogic(result)

		assert.False(t, result.Allowed)
	})

	t.Run("Allowed is false when operation error violation is present", func(t *testing.T) {
		manifest := &ModuleProtectionManifest{
			ProtectedOps: []string{"delete_user"},
		}
		change := ChangeRequest{
			ChangeType: "operation",
			Target:     "delete_user",
			Operation:  "update",
		}
		result := newEmptyResult()

		svc.checkOperationProtection(manifest, change, result)
		applyAllowedLogic(result)

		assert.False(t, result.Allowed)
	})

	t.Run("Allowed remains true when no violations and no warnings", func(t *testing.T) {
		result := newEmptyResult()
		applyAllowedLogic(result)

		assert.True(t, result.Allowed)
		assert.Empty(t, result.Violations)
		assert.Empty(t, result.Warnings)
	})

	t.Run("Allowed is false only when at least one violation is severity error", func(t *testing.T) {
		// Inject one warning violation and one error violation
		result := &ValidationResult{
			Allowed: true,
			Violations: []ProtectionViolation{
				{Rule: "protected_schema", Target: "staging", Severity: "warning", Message: "warning msg"},
				{Rule: "protected_schema", Target: "public", Severity: "error", Message: "error msg"},
			},
		}
		applyAllowedLogic(result)

		assert.False(t, result.Allowed)
	})
}

// ─────────────────────────────────────────────────────────────────
// ModuleProtectionService constructor
// ─────────────────────────────────────────────────────────────────

func TestNewModuleProtectionService(t *testing.T) {
	t.Run("uses provided logger", func(t *testing.T) {
		logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
		svc := NewModuleProtectionService(nil, logger)
		require.NotNil(t, svc)
	})

	t.Run("falls back to default logger when nil is passed", func(t *testing.T) {
		svc := NewModuleProtectionService(nil, nil)
		require.NotNil(t, svc)
		assert.NotNil(t, svc.logger)
	})
}

// ─────────────────────────────────────────────────────────────────
// ProtectionViolation struct
// ─────────────────────────────────────────────────────────────────

func TestProtectionViolation_Fields(t *testing.T) {
	v := ProtectionViolation{
		Rule:     "protected_schema",
		Target:   "public",
		Message:  "cannot delete protected schema 'public'",
		Severity: "error",
	}

	assert.Equal(t, "protected_schema", v.Rule)
	assert.Equal(t, "public", v.Target)
	assert.Contains(t, v.Message, "public")
	assert.Equal(t, "error", v.Severity)
}

// ─────────────────────────────────────────────────────────────────
// ModuleProtectionManifest — zero value and nil-slice safety
// ─────────────────────────────────────────────────────────────────

func TestCheckProtection_NilSlicesInManifest(t *testing.T) {
	svc := newTestProtectionService()

	// A manifest with nil slices (zero value) must not panic.
	manifest := &ModuleProtectionManifest{}

	changes := []ChangeRequest{
		{ChangeType: "schema", Target: "public", Operation: "delete"},
		{ChangeType: "route", Target: "/api/users", Operation: "update"},
		{ChangeType: "operation", Target: "send_email", Operation: "delete"},
		{ChangeType: "file", Target: "main.go", Operation: "delete"},
	}

	for _, change := range changes {
		result := newEmptyResult()

		switch change.ChangeType {
		case "schema":
			assert.NotPanics(t, func() { svc.checkSchemaProtection(manifest, change, result) })
		case "route":
			assert.NotPanics(t, func() { svc.checkRouteProtection(manifest, change, result) })
		case "operation":
			assert.NotPanics(t, func() { svc.checkOperationProtection(manifest, change, result) })
		case "file":
			assert.NotPanics(t, func() { svc.checkFileProtection(manifest, change, result) })
		}

		assert.Empty(t, result.Violations, "zero-value manifest should produce no violations for %s", change.ChangeType)
		assert.Empty(t, result.Warnings, "zero-value manifest should produce no warnings for %s", change.ChangeType)
	}
}
