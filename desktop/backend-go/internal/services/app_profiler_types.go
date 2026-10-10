package services

import "time"

// ApplicationProfile represents a comprehensive profile of an application
type ApplicationProfile struct {
	ID                string                 `json:"id"`
	UserID            string                 `json:"user_id"`
	Name              string                 `json:"name"`
	Description       string                 `json:"description"`
	RootPath          string                 `json:"root_path"`
	AutoSyncEnabled   bool                   `json:"auto_sync_enabled"`
	LastSyncedAt      *time.Time             `json:"last_synced_at,omitempty"`
	SyncSource        string                 `json:"sync_source,omitempty"`
	SyncBranch        string                 `json:"sync_branch,omitempty"`
	SyncCommit        string                 `json:"sync_commit,omitempty"`
	AppType           AppType                `json:"app_type"`
	Version           string                 `json:"version,omitempty"`
	TechStack         TechStack              `json:"tech_stack"`
	Languages         []LanguageInfo         `json:"languages"`
	Frameworks        []string               `json:"frameworks"`
	StructureTree     *DirectoryTree         `json:"structure_tree"`
	Components        []ComponentInfo        `json:"components"`
	TotalComponents   int                    `json:"total_components"`
	Modules           []ModuleInfo           `json:"modules"`
	TotalModules      int                    `json:"total_modules"`
	APIEndpoints      []APIEndpointInfo      `json:"api_endpoints"`
	TotalEndpoints    int                    `json:"total_endpoints"`
	DatabaseSchema    *DatabaseSchemaInfo    `json:"database_schema,omitempty"`
	Conventions       CodeConventions        `json:"conventions"`
	IntegrationPoints []IntegrationPoint     `json:"integration_points"`
	ReadmeSummary     string                 `json:"readme_summary,omitempty"`
	LinesOfCode       int                    `json:"lines_of_code"`
	FileCount         int                    `json:"file_count"`
	LastAnalyzedAt    time.Time              `json:"last_analyzed_at"`
	Metadata          map[string]interface{} `json:"metadata"`
	CreatedAt         time.Time              `json:"created_at"`
	UpdatedAt         time.Time              `json:"updated_at"`
}

// AppType represents the type of application
type AppType string

const (
	AppTypeWeb          AppType = "web"
	AppTypeAPI          AppType = "api"
	AppTypeDesktop      AppType = "desktop"
	AppTypeMobile       AppType = "mobile"
	AppTypeCLI          AppType = "cli"
	AppTypeLibrary      AppType = "library"
	AppTypeMicroservice AppType = "microservice"
	AppTypeMonolith     AppType = "monolith"
	AppTypeFullStack    AppType = "fullstack"
)

// TechStack represents the technology stack
type TechStack struct {
	Frontend  []string `json:"frontend"`
	Backend   []string `json:"backend"`
	Database  []string `json:"database"`
	DevOps    []string `json:"devops"`
	Testing   []string `json:"testing"`
	BuildTool []string `json:"build_tool"`
}

// LanguageInfo represents programming language information
type LanguageInfo struct {
	Name       string  `json:"name"`
	Files      int     `json:"files"`
	Lines      int     `json:"lines"`
	Percentage float64 `json:"percentage"`
}

// DirectoryTree represents the project structure
type DirectoryTree struct {
	Name     string           `json:"name"`
	Path     string           `json:"path"`
	Type     string           `json:"type"` // file, directory
	Children []*DirectoryTree `json:"children,omitempty"`
	FileType string           `json:"file_type,omitempty"`
	Size     int64            `json:"size,omitempty"`
}

// ComponentInfo represents a UI component
type ComponentInfo struct {
	Name        string   `json:"name"`
	FilePath    string   `json:"file_path"`
	Type        string   `json:"type"` // page, component, layout, widget
	Description string   `json:"description,omitempty"`
	Props       []string `json:"props,omitempty"`
	Events      []string `json:"events,omitempty"`
	UsedIn      []string `json:"used_in,omitempty"`
	Lines       int      `json:"lines"`
}

// ModuleInfo represents a code module
type ModuleInfo struct {
	Name        string   `json:"name"`
	Path        string   `json:"path"`
	Type        string   `json:"type"` // handler, service, repository, utility
	Description string   `json:"description,omitempty"`
	Exports     []string `json:"exports,omitempty"`
	Imports     []string `json:"imports,omitempty"`
	Lines       int      `json:"lines"`
}

// APIEndpointInfo represents an API endpoint
type APIEndpointInfo struct {
	Method       string   `json:"method"`
	Path         string   `json:"path"`
	Handler      string   `json:"handler"`
	HandlerPath  string   `json:"handler_path"`
	Description  string   `json:"description,omitempty"`
	AuthRequired bool     `json:"auth_required"`
	Tags         []string `json:"tags,omitempty"`
}

// DatabaseSchemaInfo represents database schema information
type DatabaseSchemaInfo struct {
	Tables      []TableInfo     `json:"tables"`
	TotalTables int             `json:"total_tables"`
	Migrations  []MigrationInfo `json:"migrations,omitempty"`
}

// TableInfo represents database table information
type TableInfo struct {
	Name    string       `json:"name"`
	Columns []ColumnInfo `json:"columns"`
}

// ColumnInfo represents a database column
type ColumnInfo struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Nullable bool   `json:"nullable"`
	Primary  bool   `json:"primary"`
}

// MigrationInfo represents a database migration
type MigrationInfo struct {
	Name      string `json:"name"`
	Path      string `json:"path"`
	Version   int    `json:"version"`
	AppliedAt string `json:"applied_at,omitempty"`
}

// CodeConventions represents coding conventions detected
type CodeConventions struct {
	NamingStyle     string   `json:"naming_style"` // camelCase, snake_case, PascalCase
	IndentStyle     string   `json:"indent_style"` // tabs, spaces
	IndentSize      int      `json:"indent_size"`
	QuoteStyle      string   `json:"quote_style"` // single, double
	Semicolons      bool     `json:"semicolons"`
	TrailingCommas  bool     `json:"trailing_commas"`
	FileNaming      string   `json:"file_naming"`
	DirectoryNaming string   `json:"directory_naming"`
	CommonPatterns  []string `json:"common_patterns"`
}

// IntegrationPoint represents an external integration
type IntegrationPoint struct {
	Name        string `json:"name"`
	Type        string `json:"type"` // api, database, service, webhook
	URL         string `json:"url,omitempty"`
	Description string `json:"description,omitempty"`
}

// ProfileOptions configures profiling behavior
type ProfileOptions struct {
	MaxDepth          int      `json:"max_depth"`
	IncludeHidden     bool     `json:"include_hidden"`
	ExcludePatterns   []string `json:"exclude_patterns"`
	AnalyzeComponents bool     `json:"analyze_components"`
	AnalyzeEndpoints  bool     `json:"analyze_endpoints"`
	AnalyzeDatabase   bool     `json:"analyze_database"`
	ExtractReadme     bool     `json:"extract_readme"`
}

// DefaultProfileOptions returns default profiling options
func DefaultProfileOptions() *ProfileOptions {
	return &ProfileOptions{
		MaxDepth:      10,
		IncludeHidden: false,
		ExcludePatterns: []string{
			"node_modules", "vendor", ".git", "__pycache__", ".next",
			"build", "dist", ".svelte-kit", "coverage", ".cache",
		},
		AnalyzeComponents: true,
		AnalyzeEndpoints:  true,
		AnalyzeDatabase:   true,
		ExtractReadme:     true,
	}
}
