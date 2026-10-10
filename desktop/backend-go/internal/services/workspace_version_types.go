package services

import (
	"time"

	"github.com/google/uuid"
)

// WorkspaceSnapshot represents a complete workspace state
type WorkspaceSnapshot struct {
	Timestamp time.Time              `json:"timestamp"`
	Apps      []AppSnapshot          `json:"apps"`
	Members   []MemberSnapshot       `json:"members"`
	Roles     []RoleSnapshot         `json:"roles"`
	Settings  map[string]interface{} `json:"settings"`
	Memories  []MemorySnapshot       `json:"memories"`
	Metadata  SnapshotMetadata       `json:"metadata"`
}

type AppSnapshot struct {
	ID               uuid.UUID              `json:"id"`
	AppName          string                 `json:"app_name"`
	TemplateID       *uuid.UUID             `json:"template_id"`
	ModuleInstanceID *uuid.UUID             `json:"module_instance_id"`
	IsVisible        bool                   `json:"is_visible"`
	IsPinned         bool                   `json:"is_pinned"`
	IsFavorite       bool                   `json:"is_favorite"`
	PositionIndex    *int                   `json:"position_index"`
	CustomConfig     map[string]interface{} `json:"custom_config"`
	CustomIcon       *string                `json:"custom_icon"`
}

type MemberSnapshot struct {
	ID        uuid.UUID  `json:"id"`
	UserID    string     `json:"user_id"`
	RoleID    *uuid.UUID `json:"role_id"`
	RoleName  string     `json:"role_name"`
	Status    string     `json:"status"`
	InvitedAt *time.Time `json:"invited_at"`
	JoinedAt  time.Time  `json:"joined_at"`
	InvitedBy *string    `json:"invited_by"`
}

type RoleSnapshot struct {
	ID             uuid.UUID              `json:"id"`
	Name           string                 `json:"name"`
	DisplayName    *string                `json:"display_name"`
	Description    *string                `json:"description"`
	Color          *string                `json:"color"`
	Icon           *string                `json:"icon"`
	HierarchyLevel int                    `json:"hierarchy_level"`
	IsSystem       bool                   `json:"is_system"`
	IsDefault      bool                   `json:"is_default"`
	Permissions    map[string]interface{} `json:"permissions"`
}

type MemorySnapshot struct {
	ID              uuid.UUID              `json:"id"`
	UserID          *string                `json:"user_id"`
	Title           *string                `json:"title"`
	Summary         *string                `json:"summary"`
	Content         string                 `json:"content"`
	MemoryType      string                 `json:"memory_type"`
	Category        string                 `json:"category"`
	ScopeType       *string                `json:"scope_type"`
	ScopeID         *uuid.UUID             `json:"scope_id"`
	Visibility      string                 `json:"visibility"`
	CreatedBy       *string                `json:"created_by"`
	ImportanceScore float64                `json:"importance_score"`
	Tags            []string               `json:"tags"`
	Source          *string                `json:"source"`
	Metadata        map[string]interface{} `json:"metadata"`
	IsPinned        bool                   `json:"is_pinned"`
	IsActive        bool                   `json:"is_active"`
	IsArchived      bool                   `json:"is_archived"`
}

type SnapshotMetadata struct {
	AppCount    int `json:"app_count"`
	MemberCount int `json:"member_count"`
	RoleCount   int `json:"role_count"`
	MemoryCount int `json:"memory_count"`
}

// VersionDiffResult represents the diff between two workspace versions
type VersionDiffResult struct {
	FromVersion string             `json:"from_version"`
	ToVersion   string             `json:"to_version"`
	Summary     VersionDiffSummary `json:"summary"`
	Files       []FileDiff         `json:"files"`
}

// VersionDiffSummary provides a summary of changes between versions
type VersionDiffSummary struct {
	FilesAdded        int `json:"files_added"`
	FilesRemoved      int `json:"files_removed"`
	FilesModified     int `json:"files_modified"`
	FilesUnchanged    int `json:"files_unchanged"`
	TotalLinesAdded   int `json:"total_lines_added"`
	TotalLinesRemoved int `json:"total_lines_removed"`
	AppsAdded         int `json:"apps_added"`
	AppsRemoved       int `json:"apps_removed"`
}

// FileDiff represents the diff for a single file
type FileDiff struct {
	FilePath     string `json:"file_path"`
	ChangeType   string `json:"change_type"` // "added", "removed", "modified", "unchanged"
	Language     string `json:"language,omitempty"`
	FileType     string `json:"file_type,omitempty"`
	OldContent   string `json:"old_content,omitempty"`
	NewContent   string `json:"new_content,omitempty"`
	UnifiedDiff  string `json:"unified_diff,omitempty"`
	LinesAdded   int    `json:"lines_added"`
	LinesRemoved int    `json:"lines_removed"`
}

// generatedFileInfo holds file info for diff comparison
type generatedFileInfo struct {
	FilePath    string
	Content     string
	ContentHash string
	Language    string
	FileType    string
}

// diffResult holds the computed diff text and line counts
type diffResult struct {
	Text    string
	Added   int
	Removed int
}
