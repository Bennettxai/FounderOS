# OSA Prompt Template System Design

## Overview

This document describes the configurable, template-based prompt system for OSA (Open Source Agent) integration in BusinessOS. The design prioritizes **zero hard-coded prompts** and enables user customization without code changes.

## Core Principles

1. **Template-Based**: All prompts loaded from files or database, never hard-coded in Go
2. **Variable Substitution**: Dynamic content injected via Go's `text/template` engine
3. **User Customizable**: Users can create/edit prompt templates via UI or API
4. **Versioned**: Templates are versioned for rollback and A/B testing
5. **Context-Aware**: Templates can access user context, workspace info, and integration data

## Architecture

### Three-Tier Template System

```
┌─────────────────────────────────────────────────────────┐
│  1. SYSTEM TEMPLATES (built-in, read-only)              │
│     Location: internal/prompts/templates/*.yaml          │
│     Purpose: Default prompts for core OSA operations     │
└─────────────────────────────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────────┐
│  2. WORKSPACE TEMPLATES (workspace-level overrides)      │
│     Location: Database (osa_prompt_templates table)      │
│     Purpose: Workspace-specific customizations           │
└─────────────────────────────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────────┐
│  3. USER TEMPLATES (user-level overrides)                │
│     Location: Database (osa_prompt_templates table)      │
│     Purpose: Personal customizations for specific tasks  │
└─────────────────────────────────────────────────────────┘
```

**Resolution Order**: User Template → Workspace Template → System Template

## Database Schema

### osa_prompt_templates Table

```sql
CREATE TABLE IF NOT EXISTS osa_prompt_templates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- Identification
    name VARCHAR(255) NOT NULL, -- e.g., "crm-app-generation", "data-pipeline-creation"
    display_name VARCHAR(255) NOT NULL,
    description TEXT,

    -- Scope
    scope VARCHAR(50) NOT NULL, -- 'system', 'workspace', 'user'
    workspace_id UUID REFERENCES osa_workspaces(id) ON DELETE CASCADE,
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,

    -- Template content
    template_content TEXT NOT NULL, -- Go template syntax

    -- Variables (schema for validation)
    variables JSONB NOT NULL, -- {"variables": ["AppType", "UserBusiness", ...], "required": ["AppType"]}

    -- Categorization
    category VARCHAR(100), -- 'app-generation', 'feature-addition', 'bug-fix', 'orchestration'
    tags TEXT[],

    -- Versioning
    version VARCHAR(50) NOT NULL DEFAULT '1.0.0',
    is_active BOOLEAN DEFAULT true,
    parent_template_id UUID REFERENCES osa_prompt_templates(id) ON DELETE SET NULL,

    -- Usage tracking
    usage_count INTEGER DEFAULT 0,
    success_rate DECIMAL(5,2), -- 0-100%

    -- Metadata
    metadata JSONB DEFAULT '{}',

    -- Timestamps
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),

    -- Constraints
    CONSTRAINT osa_prompt_scope_check CHECK (
        (scope = 'system' AND workspace_id IS NULL AND user_id IS NULL) OR
        (scope = 'workspace' AND workspace_id IS NOT NULL AND user_id IS NULL) OR
        (scope = 'user' AND user_id IS NOT NULL)
    ),
    CONSTRAINT osa_prompt_name_scope_unique UNIQUE(name, scope, COALESCE(workspace_id, '00000000-0000-0000-0000-000000000000'::uuid), COALESCE(user_id, '00000000-0000-0000-0000-000000000000'::uuid))
);

CREATE INDEX idx_osa_prompt_templates_name ON osa_prompt_templates(name);
CREATE INDEX idx_osa_prompt_templates_scope ON osa_prompt_templates(scope);
CREATE INDEX idx_osa_prompt_templates_workspace ON osa_prompt_templates(workspace_id);
CREATE INDEX idx_osa_prompt_templates_user ON osa_prompt_templates(user_id);
CREATE INDEX idx_osa_prompt_templates_category ON osa_prompt_templates(category);
CREATE INDEX idx_osa_prompt_templates_active ON osa_prompt_templates(is_active) WHERE is_active = true;
CREATE INDEX idx_osa_prompt_templates_tags ON osa_prompt_templates USING GIN(tags);
```

## Template File Format (YAML)

System templates are stored as YAML files in `internal/prompts/templates/osa/`.

### Example: CRM App Generation Template

**File**: `internal/prompts/templates/osa/crm-app-generation.yaml`

```yaml
name: "crm-app-generation"
display_name: "CRM Application Generation"
description: "Generate a full-stack CRM application tailored to the user's business"
category: "app-generation"
version: "1.0.0"
tags:
  - "crm"
  - "full-stack"
  - "business"

variables:
  # Required variables
  - name: "AppType"
    type: "string"
    required: true
    description: "Type of application (e.g., 'CRM', 'ERP', 'Dashboard')"

  - name: "UserBusiness"
    type: "string"
    required: true
    description: "User's business domain or industry"

  # Optional variables
  - name: "AvailableIntegrations"
    type: "array"
    required: false
    default: []
    description: "List of available third-party integrations"

  - name: "UserRequirements"
    type: "string"
    required: false
    default: "Standard CRM features"
    description: "Specific user requirements or features"

  - name: "DatabasePreference"
    type: "string"
    required: false
    default: "PostgreSQL"
    description: "Preferred database system"

  - name: "ExistingStack"
    type: "object"
    required: false
    description: "Existing tech stack information"

template: |
  # CRM Application Generation

  You are generating a **{{.AppType}}** application for the user's business.

  ## User Context

  - **Business Domain**: {{.UserBusiness}}
  - **Database Preference**: {{.DatabasePreference}}
  {{- if .ExistingStack}}
  - **Existing Tech Stack**:
    {{- range $key, $value := .ExistingStack}}
    - {{$key}}: {{$value}}
    {{- end}}
  {{- end}}

  ## Available Integrations

  {{- if .AvailableIntegrations}}
  The following third-party integrations are available:
  {{- range .AvailableIntegrations}}
  - **{{.Name}}**: {{.Description}} ({{.Status}})
  {{- end}}
  {{- else}}
  No third-party integrations are currently configured. Focus on building a standalone application.
  {{- end}}

  ## User Requirements

  {{.UserRequirements}}

  ## Your Task

  Generate a production-ready, full-stack {{.AppType}} application with the following:

  1. **Database Schema**
     - Design normalized tables for core CRM entities (contacts, companies, deals, activities)
     - Include audit fields (created_at, updated_at, created_by)
     - Use PostgreSQL-specific features (JSONB, arrays, indexes)
     - Include migration files

  2. **Backend API** (Go + Gin)
     - RESTful endpoints for all CRUD operations
     - Authentication and authorization middleware
     - Input validation using struct tags
     - Repository pattern for data access
     - Comprehensive error handling with `slog` logging
     - No `panic` calls in production code
     - Context propagation throughout

  3. **Frontend UI** (SvelteKit + TypeScript)
     - Dashboard with key metrics
     - List views with filtering, sorting, pagination
     - Detail views with edit capabilities
     - Responsive design (mobile-first)
     - Dark mode support
     - Tailwind CSS styling

  4. **Integration Layer**
     {{- if .AvailableIntegrations}}
     - Connect to: {{range $i, $int := .AvailableIntegrations}}{{if $i}}, {{end}}{{$int.Name}}{{end}}
     - Use OAuth 2.0 for authentication
     - Implement retry logic and rate limiting
     {{- else}}
     - Prepare integration stubs for future connectivity
     {{- end}}

  5. **Testing**
     - Unit tests for business logic (Go: `testify`)
     - Integration tests for API endpoints
     - E2E tests for critical user flows (Playwright)
     - Minimum 80% code coverage

  6. **Documentation**
     - API documentation (OpenAPI/Swagger)
     - Setup instructions (README.md)
     - Architecture decision records (ADRs)
     - Deployment guide

  ## Technical Constraints

  - Follow BusinessOS architecture patterns (Handler → Service → Repository)
  - Use `slog` for all logging (NO `fmt.Printf`)
  - All database queries via sqlc-generated code
  - Error handling: wrap errors with context, no naked returns
  - Context as first parameter in all I/O functions

  ## Success Criteria

  - ✅ Application builds without errors
  - ✅ All tests pass
  - ✅ Database migrations run cleanly
  - ✅ Frontend connects to backend successfully
  - ✅ At least one integration works (if available)
  - ✅ No security vulnerabilities (run `gosec`, `npm audit`)

  ---

  **Begin generation now. Create all necessary files, tests, and documentation.**
```

### Example: Data Pipeline Template

**File**: `internal/prompts/templates/osa/data-pipeline-creation.yaml`

```yaml
name: "data-pipeline-creation"
display_name: "Data Pipeline Creation"
description: "Generate ETL/ELT data pipelines with transformation logic"
category: "data-engineering"
version: "1.0.0"
tags:
  - "etl"
  - "data-pipeline"
  - "analytics"

variables:
  - name: "SourceType"
    type: "string"
    required: true
    description: "Source system type (e.g., 'API', 'Database', 'File', 'Stream')"

  - name: "DestinationType"
    type: "string"
    required: true
    description: "Destination type (e.g., 'PostgreSQL', 'BigQuery', 'S3', 'Kafka')"

  - name: "TransformationRules"
    type: "string"
    required: true
    description: "Business logic for data transformation"

  - name: "Schedule"
    type: "string"
    required: false
    default: "daily"
    description: "Pipeline execution schedule (cron format or 'realtime')"

  - name: "DataVolume"
    type: "string"
    required: false
    default: "medium"
    description: "Expected data volume (small, medium, large)"

template: |
  # Data Pipeline Generation

  You are creating a data pipeline to extract data from **{{.SourceType}}**, transform it, and load it into **{{.DestinationType}}**.

  ## Pipeline Configuration

  - **Source**: {{.SourceType}}
  - **Destination**: {{.DestinationType}}
  - **Schedule**: {{.Schedule}}
  - **Expected Volume**: {{.DataVolume}}

  ## Transformation Logic

  {{.TransformationRules}}

  ## Implementation Requirements

  1. **Extraction Layer**
     - Implement source connectors for {{.SourceType}}
     - Handle pagination, rate limiting, and retries
     - Support incremental extraction (track watermarks)

  2. **Transformation Layer**
     - Apply business rules defined above
     - Data validation and cleansing
     - Type casting and normalization
     - Handle nulls, duplicates, and edge cases

  3. **Loading Layer**
     - Bulk insert optimization for {{.DestinationType}}
     - Upsert logic (handle updates to existing records)
     - Transaction management for atomicity

  4. **Orchestration**
     {{- if eq .Schedule "realtime"}}
     - Real-time streaming pipeline (use message queue)
     - Exactly-once delivery semantics
     {{- else}}
     - Batch processing with schedule: {{.Schedule}}
     - Dependency management (upstream/downstream)
     {{- end}}

  5. **Monitoring**
     - Pipeline execution logs (using `slog`)
     - Data quality metrics (row counts, schema drift)
     - Alerting on failures or anomalies
     - SLA tracking (latency, throughput)

  6. **Error Handling**
     - Dead letter queue for failed records
     - Automatic retry with exponential backoff
     - Manual intervention workflow for non-recoverable errors

  ## Technical Constraints

  {{- if eq .DataVolume "large"}}
  - Optimize for high throughput (parallel processing, batching)
  - Use connection pooling and caching
  {{- else}}
  - Standard implementation, prioritize readability
  {{- end}}

  ---

  **Generate the complete pipeline implementation with tests and monitoring.**
```

## Go Template Engine Integration

### Template Loader Service

**File**: `internal/services/prompt_template_service.go`

```go
package services

import (
    "bytes"
    "context"
    "embed"
    "fmt"
    "log/slog"
    "text/template"

    "github.com/google/uuid"
    "gopkg.in/yaml.v3"
)

//go:embed ../prompts/templates/osa/*.yaml
var systemTemplatesFS embed.FS

// PromptTemplateService handles template loading and rendering
type PromptTemplateService struct {
    db             PromptTemplateRepository
    systemTemplates map[string]*PromptTemplate
}

// PromptTemplate represents a parsed template
type PromptTemplate struct {
    Name           string                 `yaml:"name"`
    DisplayName    string                 `yaml:"display_name"`
    Description    string                 `yaml:"description"`
    Category       string                 `yaml:"category"`
    Version        string                 `yaml:"version"`
    Tags           []string               `yaml:"tags"`
    Variables      []VariableDefinition   `yaml:"variables"`
    TemplateString string                 `yaml:"template"`
    compiledTpl    *template.Template
}

// VariableDefinition defines a template variable
type VariableDefinition struct {
    Name        string      `yaml:"name"`
    Type        string      `yaml:"type"`
    Required    bool        `yaml:"required"`
    Default     interface{} `yaml:"default"`
    Description string      `yaml:"description"`
}

// NewPromptTemplateService creates a new service instance
func NewPromptTemplateService(db PromptTemplateRepository) (*PromptTemplateService, error) {
    s := &PromptTemplateService{
        db:              db,
        systemTemplates: make(map[string]*PromptTemplate),
    }

    // Load system templates from embedded filesystem
    if err := s.loadSystemTemplates(); err != nil {
        return nil, fmt.Errorf("failed to load system templates: %w", err)
    }

    return s, nil
}

// loadSystemTemplates reads YAML files from embedded FS
func (s *PromptTemplateService) loadSystemTemplates() error {
    entries, err := systemTemplatesFS.ReadDir("internal/prompts/templates/osa")
    if err != nil {
        return fmt.Errorf("failed to read templates directory: %w", err)
    }

    for _, entry := range entries {
        if entry.IsDir() {
            continue
        }

        data, err := systemTemplatesFS.ReadFile(fmt.Sprintf("internal/prompts/templates/osa/%s", entry.Name()))
        if err != nil {
            slog.Warn("failed to read template file", "file", entry.Name(), "error", err)
            continue
        }

        var tpl PromptTemplate
        if err := yaml.Unmarshal(data, &tpl); err != nil {
            slog.Warn("failed to parse template YAML", "file", entry.Name(), "error", err)
            continue
        }

        // Compile the template
        compiledTpl, err := template.New(tpl.Name).Parse(tpl.TemplateString)
        if err != nil {
            slog.Warn("failed to compile template", "name", tpl.Name, "error", err)
            continue
        }
        tpl.compiledTpl = compiledTpl

        s.systemTemplates[tpl.Name] = &tpl
        slog.Info("loaded system template", "name", tpl.Name, "version", tpl.Version)
    }

    return nil
}

// RenderTemplate renders a template with provided variables
func (s *PromptTemplateService) RenderTemplate(
    ctx context.Context,
    templateName string,
    variables map[string]interface{},
    userID *uuid.UUID,
    workspaceID *uuid.UUID,
) (string, error) {
    // Resolution order: User → Workspace → System
    tpl, err := s.resolveTemplate(ctx, templateName, userID, workspaceID)
    if err != nil {
        return "", fmt.Errorf("failed to resolve template: %w", err)
    }

    // Validate required variables
    if err := s.validateVariables(tpl, variables); err != nil {
        return "", fmt.Errorf("variable validation failed: %w", err)
    }

    // Apply defaults for missing optional variables
    variables = s.applyDefaults(tpl, variables)

    // Render template
    var buf bytes.Buffer
    if err := tpl.compiledTpl.Execute(&buf, variables); err != nil {
        return "", fmt.Errorf("template execution failed: %w", err)
    }

    return buf.String(), nil
}

// resolveTemplate finds the appropriate template (user > workspace > system)
func (s *PromptTemplateService) resolveTemplate(
    ctx context.Context,
    name string,
    userID *uuid.UUID,
    workspaceID *uuid.UUID,
) (*PromptTemplate, error) {
    // Try user-level template first
    if userID != nil {
        tpl, err := s.db.GetTemplateByScope(ctx, name, "user", userID, nil)
        if err == nil && tpl != nil {
            return s.compileDBTemplate(tpl)
        }
    }

    // Try workspace-level template
    if workspaceID != nil {
        tpl, err := s.db.GetTemplateByScope(ctx, name, "workspace", nil, workspaceID)
        if err == nil && tpl != nil {
            return s.compileDBTemplate(tpl)
        }
    }

    // Fall back to system template
    if tpl, ok := s.systemTemplates[name]; ok {
        return tpl, nil
    }

    return nil, fmt.Errorf("template not found: %s", name)
}

// compileDBTemplate converts a database template to a compiled template
func (s *PromptTemplateService) compileDBTemplate(dbTpl *DBPromptTemplate) (*PromptTemplate, error) {
    compiledTpl, err := template.New(dbTpl.Name).Parse(dbTpl.TemplateContent)
    if err != nil {
        return nil, fmt.Errorf("failed to compile template: %w", err)
    }

    tpl := &PromptTemplate{
        Name:           dbTpl.Name,
        DisplayName:    dbTpl.DisplayName,
        Description:    dbTpl.Description,
        Category:       dbTpl.Category,
        Version:        dbTpl.Version,
        Tags:           dbTpl.Tags,
        TemplateString: dbTpl.TemplateContent,
        compiledTpl:    compiledTpl,
    }

    // Parse variables from JSONB
    if len(dbTpl.Variables) > 0 {
        // Convert JSONB to VariableDefinition structs
        // (implementation depends on your JSONB unmarshaling strategy)
    }

    return tpl, nil
}

// validateVariables ensures required variables are provided
func (s *PromptTemplateService) validateVariables(tpl *PromptTemplate, vars map[string]interface{}) error {
    for _, varDef := range tpl.Variables {
        if varDef.Required {
            if _, ok := vars[varDef.Name]; !ok {
                return fmt.Errorf("required variable missing: %s", varDef.Name)
            }
        }
    }
    return nil
}

// applyDefaults fills in default values for missing optional variables
func (s *PromptTemplateService) applyDefaults(tpl *PromptTemplate, vars map[string]interface{}) map[string]interface{} {
    result := make(map[string]interface{})
    for k, v := range vars {
        result[k] = v
    }

    for _, varDef := range tpl.Variables {
        if !varDef.Required {
            if _, ok := result[varDef.Name]; !ok && varDef.Default != nil {
                result[varDef.Name] = varDef.Default
            }
        }
    }

    return result
}
```

## Usage Example

### 1. Rendering a Template in OSA Handler

**File**: `internal/handlers/osa_api.go`

```go
func (h *Handlers) HandleGenerateApp(c *gin.Context) {
    // ... (parse request, get user context) ...

    // Prepare template variables
    variables := map[string]interface{}{
        "AppType":      req.Type,
        "UserBusiness": user.BusinessDomain, // from user profile
        "UserRequirements": req.Description,
        "DatabasePreference": "PostgreSQL",
        "AvailableIntegrations": h.getAvailableIntegrations(c.Request.Context(), userID),
    }

    // Render prompt from template
    prompt, err := h.promptTemplateService.RenderTemplate(
        c.Request.Context(),
        "crm-app-generation",
        variables,
        &userID,
        &workspaceID,
    )
    if err != nil {
        slog.Error("failed to render prompt template", "error", err)
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to prepare generation prompt"})
        return
    }

    // Pass rendered prompt to OSA client
    osaReq := &osa.AppGenerationRequest{
        WorkspaceID: workspaceID,
        UserID:      userID,
        Name:        req.Name,
        Description: req.Description,
        Type:        req.Type,
        Prompt:      prompt, // ← Dynamic prompt from template
        Parameters:  req.Parameters,
    }

    resp, err := h.osaClient.GenerateApp(c.Request.Context(), osaReq)
    // ... (handle response) ...
}
```

### 2. User Creating Custom Template via API

**Endpoint**: `POST /api/osa/templates`

```json
{
  "name": "my-crm-template",
  "display_name": "My Custom CRM Template",
  "description": "CRM with Stripe integration focus",
  "category": "app-generation",
  "scope": "user",
  "template_content": "# Custom CRM\n\nYou are creating a CRM with Stripe payments...\n\n{{.UserRequirements}}",
  "variables": {
    "variables": [
      {
        "name": "UserRequirements",
        "type": "string",
        "required": true
      }
    ],
    "required": ["UserRequirements"]
  },
  "tags": ["crm", "stripe", "payments"]
}
```

**Handler**: `internal/handlers/prompt_templates.go`

```go
func (h *Handlers) CreatePromptTemplate(c *gin.Context) {
    user := middleware.GetCurrentUser(c)
    var req CreatePromptTemplateRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    // Validate template syntax
    _, err := template.New("validation").Parse(req.TemplateContent)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid template syntax", "details": err.Error()})
        return
    }

    // Create template in database
    tpl, err := h.promptTemplateService.CreateTemplate(c.Request.Context(), &CreateTemplateParams{
        Name:            req.Name,
        DisplayName:     req.DisplayName,
        Description:     req.Description,
        Scope:           req.Scope,
        UserID:          user.ID,
        WorkspaceID:     req.WorkspaceID,
        TemplateContent: req.TemplateContent,
        Variables:       req.Variables,
        Category:        req.Category,
        Tags:            req.Tags,
    })
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create template"})
        return
    }

    c.JSON(http.StatusCreated, tpl)
}
```

## Template Variable Context Enrichment

Templates can access a rich context object with user/workspace/integration data.

### Context Structure

```go
type PromptContext struct {
    // User info
    UserID       uuid.UUID
    UserName     string
    UserEmail    string
    UserBusiness string
    UserTimezone string

    // Workspace info
    WorkspaceID   uuid.UUID
    WorkspaceName string
    WorkspaceMode string // '2d', '3d', 'hybrid'

    // Available integrations
    AvailableIntegrations []IntegrationInfo

    // Custom user-provided variables
    Variables map[string]interface{}
}

type IntegrationInfo struct {
    Name        string
    Provider    string
    Status      string // 'connected', 'disconnected', 'error'
    Description string
    Capabilities []string
}
```

### Enriching Context

```go
func (s *PromptTemplateService) BuildContext(
    ctx context.Context,
    userID uuid.UUID,
    workspaceID uuid.UUID,
    customVars map[string]interface{},
) (*PromptContext, error) {
    // Fetch user profile
    user, err := s.userRepo.GetByID(ctx, userID)
    if err != nil {
        return nil, err
    }

    // Fetch workspace info
    workspace, err := s.workspaceRepo.GetByID(ctx, workspaceID)
    if err != nil {
        return nil, err
    }

    // Fetch available integrations
    integrations, err := s.integrationRepo.ListForUser(ctx, userID)
    if err != nil {
        return nil, err
    }

    var integrationInfos []IntegrationInfo
    for _, integ := range integrations {
        integrationInfos = append(integrationInfos, IntegrationInfo{
            Name:         integ.Name,
            Provider:     integ.Provider,
            Status:       integ.Status,
            Description:  integ.Description,
            Capabilities: integ.Capabilities,
        })
    }

    return &PromptContext{
        UserID:                user.ID,
        UserName:              user.Name,
        UserEmail:             user.Email,
        UserBusiness:          user.BusinessDomain,
        UserTimezone:          user.Timezone,
        WorkspaceID:           workspace.ID,
        WorkspaceName:         workspace.Name,
        WorkspaceMode:         workspace.Mode,
        AvailableIntegrations: integrationInfos,
        Variables:             customVars,
    }, nil
}
```

## Template Management UI

### List Templates (Endpoint)

```
GET /api/osa/templates?scope=user&category=app-generation
```

**Response**:
```json
{
  "templates": [
    {
      "id": "uuid",
      "name": "crm-app-generation",
      "display_name": "CRM Application Generation",
      "description": "Generate full-stack CRM apps",
      "scope": "system",
      "category": "app-generation",
      "version": "1.0.0",
      "tags": ["crm", "full-stack"],
      "usage_count": 42,
      "success_rate": 87.5
    }
  ]
}
```

### Get Template Details

```
GET /api/osa/templates/:id
```

**Response**:
```json
{
  "id": "uuid",
  "name": "crm-app-generation",
  "template_content": "# CRM Application\n\n...",
  "variables": {
    "variables": [
      {
        "name": "AppType",
        "type": "string",
        "required": true,
        "description": "Type of application"
      }
    ],
    "required": ["AppType"]
  }
}
```

### Update Template

```
PUT /api/osa/templates/:id
```

### Delete Template (User/Workspace only)

```
DELETE /api/osa/templates/:id
```

## Template Versioning

Templates support versioning for A/B testing and rollback.

### Version Management

```go
// Create new version
func (s *PromptTemplateService) CreateVersion(
    ctx context.Context,
    templateID uuid.UUID,
    newContent string,
    versionTag string,
) (*PromptTemplate, error) {
    // Load existing template
    existing, err := s.db.GetByID(ctx, templateID)
    if err != nil {
        return nil, err
    }

    // Create new version (child template)
    newTpl := &DBPromptTemplate{
        Name:            existing.Name,
        DisplayName:     existing.DisplayName,
        Description:     existing.Description,
        Scope:           existing.Scope,
        UserID:          existing.UserID,
        WorkspaceID:     existing.WorkspaceID,
        TemplateContent: newContent,
        Variables:       existing.Variables,
        Category:        existing.Category,
        Tags:            existing.Tags,
        Version:         versionTag,
        IsActive:        false, // New version starts inactive
        ParentTemplateID: &templateID,
    }

    return s.db.Create(ctx, newTpl)
}

// Activate a specific version (atomic swap)
func (s *PromptTemplateService) ActivateVersion(ctx context.Context, versionID uuid.UUID) error {
    return s.db.ActivateVersion(ctx, versionID)
}
```

## Testing Templates

### Template Test Suite

```go
// internal/services/prompt_template_service_test.go

func TestCRMTemplateRendering(t *testing.T) {
    service := setupTestService(t)

    variables := map[string]interface{}{
        "AppType":            "CRM",
        "UserBusiness":       "Real Estate",
        "UserRequirements":   "Track leads and property listings",
        "DatabasePreference": "PostgreSQL",
        "AvailableIntegrations": []map[string]string{
            {"Name": "Stripe", "Status": "connected"},
            {"Name": "Twilio", "Status": "disconnected"},
        },
    }

    rendered, err := service.RenderTemplate(
        context.Background(),
        "crm-app-generation",
        variables,
        nil, // no user override
        nil, // no workspace override
    )

    assert.NoError(t, err)
    assert.Contains(t, rendered, "Real Estate")
    assert.Contains(t, rendered, "PostgreSQL")
    assert.Contains(t, rendered, "Stripe")
}

func TestMissingRequiredVariable(t *testing.T) {
    service := setupTestService(t)

    variables := map[string]interface{}{
        // Missing required "AppType"
        "UserBusiness": "Real Estate",
    }

    _, err := service.RenderTemplate(
        context.Background(),
        "crm-app-generation",
        variables,
        nil,
        nil,
    )

    assert.Error(t, err)
    assert.Contains(t, err.Error(), "required variable missing: AppType")
}
```

## Migration Plan

### Phase 1: Infrastructure Setup
1. Create `osa_prompt_templates` table (migration)
2. Implement `PromptTemplateService` with file loading
3. Create system template YAML files
4. Add template rendering to OSA client

### Phase 2: API & UI
1. Create REST endpoints for template CRUD
2. Build template editor UI in frontend
3. Add variable picker with autocomplete
4. Template preview functionality

### Phase 3: User Customization
1. Enable user template creation
2. Template marketplace/gallery
3. Template sharing between users
4. Import/export functionality

### Phase 4: Advanced Features
1. A/B testing framework
2. Template analytics (success rate, usage)
3. Template recommendations
4. AI-assisted template generation

## Best Practices

### Template Design
- Keep templates focused on a single task
- Use descriptive variable names
- Provide sensible defaults for optional variables
- Include context clues in the prompt (e.g., "You are a...")
- Structure output with clear sections

### Security
- Sanitize user-provided variables (prevent prompt injection)
- Validate template syntax before saving
- Restrict system template modifications (read-only)
- Audit template usage and changes

### Performance
- Cache compiled templates in memory
- Use connection pooling for database queries
- Lazy-load templates (only when needed)
- Monitor rendering performance (latency, memory)

## Example System Templates to Create

1. **crm-app-generation.yaml** - Full-stack CRM apps
2. **data-pipeline-creation.yaml** - ETL/ELT pipelines
3. **dashboard-creation.yaml** - Analytics dashboards
4. **api-wrapper-generation.yaml** - Third-party API clients
5. **auth-system-setup.yaml** - Authentication/authorization
6. **notification-system.yaml** - Email/SMS/push notifications
7. **file-processor.yaml** - File upload/processing workflows
8. **report-generator.yaml** - PDF/Excel report generation
9. **webhook-handler.yaml** - Webhook receivers
10. **background-job-worker.yaml** - Async job processors

## Conclusion

This template-based prompt system provides:
- **Zero hard-coded prompts** in Go code
- **User customization** without deployments
- **Versioning** for rollback and A/B testing
- **Context-aware** prompts with dynamic data
- **Scalable** architecture (file + database)

All prompts are externalized, making OSA integration flexible and user-friendly.
