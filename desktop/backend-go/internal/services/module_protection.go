package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ModuleProtectionService validates changes against module manifests
// to prevent unauthorized modifications to protected module resources.
type ModuleProtectionService struct {
	pool   *pgxpool.Pool
	logger *slog.Logger
}

// NewModuleProtectionService creates a new module protection service
func NewModuleProtectionService(pool *pgxpool.Pool, logger *slog.Logger) *ModuleProtectionService {
	if logger == nil {
		logger = slog.Default()
	}
	return &ModuleProtectionService{
		pool:   pool,
		logger: logger.With("component", "module_protection"),
	}
}

// ChangeRequest represents a proposed change to a module
type ChangeRequest struct {
	// Type of change: "schema", "route", "operation", "file", "config"
	ChangeType string `json:"change_type"`
	// Target path or identifier being changed
	Target string `json:"target"`
	// Operation being performed: "create", "update", "delete"
	Operation string `json:"operation"`
	// Description of what is changing
	Description string `json:"description,omitempty"`
}

// ValidationResult contains the result of a protection validation
type ValidationResult struct {
	Allowed    bool                  `json:"allowed"`
	Violations []ProtectionViolation `json:"violations,omitempty"`
	Warnings   []string              `json:"warnings,omitempty"`
}

// ProtectionViolation describes a specific protection rule violation
type ProtectionViolation struct {
	Rule     string `json:"rule"`
	Target   string `json:"target"`
	Message  string `json:"message"`
	Severity string `json:"severity"` // "error" or "warning"
}

// ModuleProtectionManifest represents the protection rules stored in a module's manifest JSONB column.
// It is distinct from ModuleManifest (the export/ZIP manifest) defined in module_export_service.go.
type ModuleProtectionManifest struct {
	Name             string            `json:"name"`
	Version          string            `json:"version"`
	ProtectedSchemas []string          `json:"protected_schemas,omitempty"`
	ProtectedRoutes  []string          `json:"protected_routes,omitempty"`
	ProtectedOps     []string          `json:"protected_operations,omitempty"`
	ProtectedFiles   []string          `json:"protected_files,omitempty"`
	Permissions      map[string]string `json:"permissions,omitempty"`
}

// ValidateChange checks if a proposed change is allowed by the module's protection rules
func (s *ModuleProtectionService) ValidateChange(
	ctx context.Context,
	moduleID uuid.UUID,
	change ChangeRequest,
) (*ValidationResult, error) {
	// Load module manifest from database
	manifest, err := s.loadManifest(ctx, moduleID)
	if err != nil {
		return nil, fmt.Errorf("failed to load module manifest: %w", err)
	}

	// If no manifest found, allow the change (no protection rules)
	if manifest == nil {
		s.logger.Debug("no manifest found, allowing change",
			"module_id", moduleID,
			"change_type", change.ChangeType,
		)
		return &ValidationResult{Allowed: true}, nil
	}

	result := &ValidationResult{Allowed: true}

	// Check protection rules based on change type
	switch change.ChangeType {
	case "schema":
		s.checkSchemaProtection(manifest, change, result)
	case "route":
		s.checkRouteProtection(manifest, change, result)
	case "operation":
		s.checkOperationProtection(manifest, change, result)
	case "file":
		s.checkFileProtection(manifest, change, result)
	default:
		// Unknown change type - add warning but allow
		result.Warnings = append(result.Warnings,
			fmt.Sprintf("unknown change type '%s', no protection rules applied", change.ChangeType))
	}

	// If any violations have severity "error", disallow the change
	for _, v := range result.Violations {
		if v.Severity == "error" {
			result.Allowed = false
			break
		}
	}

	s.logger.Info("change validation completed",
		"module_id", moduleID,
		"change_type", change.ChangeType,
		"target", change.Target,
		"allowed", result.Allowed,
		"violations", len(result.Violations),
		"warnings", len(result.Warnings),
	)

	return result, nil
}

// ValidateChanges validates multiple changes at once (batch validation)
func (s *ModuleProtectionService) ValidateChanges(
	ctx context.Context,
	moduleID uuid.UUID,
	changes []ChangeRequest,
) (*ValidationResult, error) {
	combined := &ValidationResult{Allowed: true}

	for _, change := range changes {
		result, err := s.ValidateChange(ctx, moduleID, change)
		if err != nil {
			return nil, err
		}
		if !result.Allowed {
			combined.Allowed = false
		}
		combined.Violations = append(combined.Violations, result.Violations...)
		combined.Warnings = append(combined.Warnings, result.Warnings...)
	}

	return combined, nil
}

// loadManifest reads the module manifest from the database
func (s *ModuleProtectionService) loadManifest(ctx context.Context, moduleID uuid.UUID) (*ModuleProtectionManifest, error) {
	pgID := pgtype.UUID{Bytes: moduleID, Valid: true}

	var manifestJSON []byte
	err := s.pool.QueryRow(ctx,
		`SELECT manifest FROM custom_modules WHERE id = $1 AND manifest IS NOT NULL`,
		pgID,
	).Scan(&manifestJSON)

	if err != nil {
		// No manifest found (could be pgx.ErrNoRows or null)
		s.logger.Debug("no manifest in database", "module_id", moduleID, "error", err)
		return nil, nil
	}

	var manifest ModuleProtectionManifest
	if err := json.Unmarshal(manifestJSON, &manifest); err != nil {
		return nil, fmt.Errorf("failed to parse manifest JSON: %w", err)
	}

	return &manifest, nil
}

// checkSchemaProtection validates changes against protected schemas
func (s *ModuleProtectionService) checkSchemaProtection(
	manifest *ModuleProtectionManifest,
	change ChangeRequest,
	result *ValidationResult,
) {
	for _, protected := range manifest.ProtectedSchemas {
		if matchesPattern(change.Target, protected) {
			if change.Operation == "delete" {
				result.Violations = append(result.Violations, ProtectionViolation{
					Rule:     "protected_schema",
					Target:   change.Target,
					Message:  fmt.Sprintf("cannot delete protected schema '%s' (matches rule '%s')", change.Target, protected),
					Severity: "error",
				})
			} else if change.Operation == "update" {
				result.Warnings = append(result.Warnings,
					fmt.Sprintf("modifying protected schema '%s' — changes may break module functionality", change.Target))
			}
		}
	}
}

// checkRouteProtection validates changes against protected routes
func (s *ModuleProtectionService) checkRouteProtection(
	manifest *ModuleProtectionManifest,
	change ChangeRequest,
	result *ValidationResult,
) {
	for _, protected := range manifest.ProtectedRoutes {
		if matchesPattern(change.Target, protected) {
			result.Violations = append(result.Violations, ProtectionViolation{
				Rule:     "protected_route",
				Target:   change.Target,
				Message:  fmt.Sprintf("cannot modify protected route '%s' (matches rule '%s')", change.Target, protected),
				Severity: "error",
			})
		}
	}
}

// checkOperationProtection validates changes against protected operations
func (s *ModuleProtectionService) checkOperationProtection(
	manifest *ModuleProtectionManifest,
	change ChangeRequest,
	result *ValidationResult,
) {
	for _, protected := range manifest.ProtectedOps {
		if matchesPattern(change.Target, protected) {
			result.Violations = append(result.Violations, ProtectionViolation{
				Rule:     "protected_operation",
				Target:   change.Target,
				Message:  fmt.Sprintf("operation '%s' is protected (matches rule '%s')", change.Target, protected),
				Severity: "error",
			})
		}
	}
}

// checkFileProtection validates changes against protected files
func (s *ModuleProtectionService) checkFileProtection(
	manifest *ModuleProtectionManifest,
	change ChangeRequest,
	result *ValidationResult,
) {
	for _, protected := range manifest.ProtectedFiles {
		if matchesPattern(change.Target, protected) {
			if change.Operation == "delete" {
				result.Violations = append(result.Violations, ProtectionViolation{
					Rule:     "protected_file",
					Target:   change.Target,
					Message:  fmt.Sprintf("cannot delete protected file '%s' (matches rule '%s')", change.Target, protected),
					Severity: "error",
				})
			} else {
				result.Warnings = append(result.Warnings,
					fmt.Sprintf("modifying protected file '%s' — review changes carefully", change.Target))
			}
		}
	}
}

// matchesPattern checks if a target matches a protection pattern.
// Supports:
// - Exact match: "users" matches "users"
// - Prefix match with wildcard: "api/*" matches "api/v1/users"
// - Suffix match: "*.go" matches "handler.go"
func matchesPattern(target, pattern string) bool {
	// Exact match
	if target == pattern {
		return true
	}

	// Wildcard suffix: "api/*" matches anything starting with "api/"
	if strings.HasSuffix(pattern, "/*") {
		prefix := strings.TrimSuffix(pattern, "/*")
		if strings.HasPrefix(target, prefix+"/") || target == prefix {
			return true
		}
	}

	// Wildcard prefix: "*.go" matches anything ending with ".go"
	if strings.HasPrefix(pattern, "*.") {
		suffix := strings.TrimPrefix(pattern, "*")
		if strings.HasSuffix(target, suffix) {
			return true
		}
	}

	// Contains match for broader patterns
	if strings.Contains(pattern, "*") {
		// Split on * and check all parts exist in order
		parts := strings.Split(pattern, "*")
		remaining := target
		for _, part := range parts {
			if part == "" {
				continue
			}
			idx := strings.Index(remaining, part)
			if idx < 0 {
				return false
			}
			remaining = remaining[idx+len(part):]
		}
		return true
	}

	return false
}
