package services

import (
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// CustomModuleService handles custom module management
type CustomModuleService struct {
	pool   *pgxpool.Pool
	logger *slog.Logger
}

// CustomModule represents a user-created module
type CustomModule struct {
	ID           uuid.UUID              `json:"id"`
	CreatedBy    string                 `json:"created_by"`
	WorkspaceID  uuid.UUID              `json:"workspace_id"`
	Name         string                 `json:"name"`
	Slug         string                 `json:"slug"`
	Description  *string                `json:"description"`
	Category     string                 `json:"category"`
	Version      string                 `json:"version"`
	Manifest     map[string]interface{} `json:"manifest"`
	Config       map[string]interface{} `json:"config"`
	Icon         *string                `json:"icon"`
	Tags         []string               `json:"tags"`
	Keywords     []string               `json:"keywords"`
	IsPublic     bool                   `json:"is_public"`
	IsPublished  bool                   `json:"is_published"`
	IsTemplate   bool                   `json:"is_template"`
	InstallCount int                    `json:"install_count"`
	StarCount    int                    `json:"star_count"`
	ShareScope   string                 `json:"share_scope"` // 'workspace' | 'organization'
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
	PublishedAt  *time.Time             `json:"published_at,omitempty"`
}

// ModuleVersion represents a version snapshot
type ModuleVersion struct {
	ID               uuid.UUID              `json:"id"`
	ModuleID         uuid.UUID              `json:"module_id"`
	Version          string                 `json:"version"`
	Changelog        string                 `json:"changelog"`
	ManifestSnapshot map[string]interface{} `json:"manifest_snapshot"`
	ConfigSnapshot   map[string]interface{} `json:"config_snapshot"`
	CreatedBy        uuid.UUID              `json:"created_by"`
	CreatedAt        time.Time              `json:"created_at"`
	IsStable         bool                   `json:"is_stable"`
	IsBreaking       bool                   `json:"is_breaking"`
}

// ModuleInstallation represents an installed module
type ModuleInstallation struct {
	ID               uuid.UUID              `json:"id"`
	ModuleID         uuid.UUID              `json:"module_id"`
	WorkspaceID      uuid.UUID              `json:"workspace_id"`
	InstalledBy      uuid.UUID              `json:"installed_by"`
	InstalledVersion string                 `json:"installed_version"`
	ConfigOverride   map[string]interface{} `json:"config_override"`
	IsEnabled        bool                   `json:"is_enabled"`
	IsAutoUpdate     bool                   `json:"is_auto_update"`
	InstalledAt      time.Time              `json:"installed_at"`
	UpdatedAt        time.Time              `json:"updated_at"`
	LastUsedAt       *time.Time             `json:"last_used_at,omitempty"`
}

// ModuleShare represents sharing permissions
type ModuleShare struct {
	ID                    uuid.UUID  `json:"id"`
	ModuleID              uuid.UUID  `json:"module_id"`
	SharedWithUserID      *uuid.UUID `json:"shared_with_user_id,omitempty"`
	SharedWithWorkspaceID *uuid.UUID `json:"shared_with_workspace_id,omitempty"`
	SharedWithEmail       *string    `json:"shared_with_email,omitempty"`
	CanView               bool       `json:"can_view"`
	CanInstall            bool       `json:"can_install"`
	CanModify             bool       `json:"can_modify"`
	CanReshare            bool       `json:"can_reshare"`
	SharedBy              uuid.UUID  `json:"shared_by"`
	SharedAt              time.Time  `json:"shared_at"`
	ExpiresAt             *time.Time `json:"expires_at,omitempty"`
}

// CreateModuleRequest contains data for creating a module
type CreateModuleRequest struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Category    string                 `json:"category"`
	Manifest    map[string]interface{} `json:"manifest"`
	Config      map[string]interface{} `json:"config,omitempty"`
	Icon        string                 `json:"icon,omitempty"`
	Tags        []string               `json:"tags,omitempty"`
	Keywords    []string               `json:"keywords,omitempty"`
}

// UpdateModuleRequest contains data for updating a module
type UpdateModuleRequest struct {
	Name        *string                 `json:"name,omitempty"`
	Description *string                 `json:"description,omitempty"`
	Category    *string                 `json:"category,omitempty"`
	Version     *string                 `json:"version,omitempty"`
	Manifest    *map[string]interface{} `json:"manifest,omitempty"`
	Config      *map[string]interface{} `json:"config,omitempty"`
	Icon        *string                 `json:"icon,omitempty"`
	Tags        *[]string               `json:"tags,omitempty"`
	Keywords    *[]string               `json:"keywords,omitempty"`
	IsPublic    *bool                   `json:"is_public,omitempty"`
	IsTemplate  *bool                   `json:"is_template,omitempty"`
	ShareScope  *string                 `json:"share_scope,omitempty"` // 'workspace' | 'organization'
}

// CreateShareRequest represents a request to share a module
type CreateShareRequest struct {
	ModuleID              uuid.UUID
	SharedWithUserID      *uuid.UUID
	SharedWithWorkspaceID *uuid.UUID
	SharedWithEmail       *string
	CanView               bool
	CanInstall            bool
	CanModify             bool
	CanReshare            bool
	SharedBy              uuid.UUID
}

func NewCustomModuleService(pool *pgxpool.Pool, logger *slog.Logger) *CustomModuleService {
	return &CustomModuleService{
		pool:   pool,
		logger: logger,
	}
}

// GenerateModuleSlug converts a module name into a URL-friendly slug
func GenerateModuleSlug(name string) string {
	// Convert to lowercase
	slug := strings.ToLower(name)
	// Replace spaces with hyphens
	slug = strings.ReplaceAll(slug, " ", "-")
	// Remove non-alphanumeric characters (except hyphens)
	reg := regexp.MustCompile("[^a-z0-9-]+")
	slug = reg.ReplaceAllString(slug, "")
	// Remove consecutive hyphens
	reg = regexp.MustCompile("-+")
	slug = reg.ReplaceAllString(slug, "-")
	// Trim hyphens from start and end
	slug = strings.Trim(slug, "-")
	return slug
}

func validateManifest(manifest map[string]interface{}) error {
	// Simple modules (records/list/dashboard) carry their schema in `config`
	// and declare no actions - allow them. Only validate action structure when
	// actions are actually declared.
	rawActions, ok := manifest["actions"]
	if !ok || rawActions == nil {
		return nil
	}

	actions, ok := rawActions.([]interface{})
	if !ok {
		return fmt.Errorf("manifest 'actions' must be an array")
	}

	if len(actions) == 0 {
		return nil
	}

	// Validate each action
	for i, action := range actions {
		actionMap, ok := action.(map[string]interface{})
		if !ok {
			return fmt.Errorf("action %d must be an object", i)
		}

		if _, ok := actionMap["name"]; !ok {
			return fmt.Errorf("action %d must have a 'name' field", i)
		}

		if _, ok := actionMap["type"]; !ok {
			return fmt.Errorf("action %d must have a 'type' field", i)
		}
	}

	return nil
}
