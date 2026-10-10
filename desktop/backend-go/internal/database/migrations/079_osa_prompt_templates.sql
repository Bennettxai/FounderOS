-- Migration: 078_osa_prompt_templates.sql
-- Description: OSA Prompt Template System - Configurable, user-customizable prompts
-- Created: 2026-01-25
-- Related: docs/OSA_PROMPT_DESIGN.md

-- =============================================================================
-- OSA PROMPT TEMPLATES
-- Template-based prompt system with user customization
-- =============================================================================

CREATE TABLE IF NOT EXISTS osa_prompt_templates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- Identification
    name VARCHAR(255) NOT NULL, -- e.g., "crm-app-generation", "data-pipeline-creation"
    display_name VARCHAR(255) NOT NULL,
    description TEXT,

    -- Scope: system (built-in), workspace (org-level), user (personal)
    scope VARCHAR(50) NOT NULL CHECK (scope IN ('system', 'workspace', 'user')),
    workspace_id UUID REFERENCES osa_workspaces(id) ON DELETE CASCADE,
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,

    -- Template content (Go text/template syntax)
    template_content TEXT NOT NULL,

    -- Variables schema (for validation and UI generation)
    -- Example: {"variables": [{"name": "AppType", "type": "string", "required": true}], "required": ["AppType"]}
    variables JSONB NOT NULL DEFAULT '{"variables": [], "required": []}',

    -- Categorization
    category VARCHAR(100), -- 'app-generation', 'feature-addition', 'bug-fix', 'orchestration', 'data-engineering'
    tags TEXT[], -- For search and filtering

    -- Versioning
    version VARCHAR(50) NOT NULL DEFAULT '1.0.0',
    is_active BOOLEAN DEFAULT true, -- Only one active version per name/scope/user/workspace
    parent_template_id UUID REFERENCES osa_prompt_templates(id) ON DELETE SET NULL,

    -- Usage tracking
    usage_count INTEGER DEFAULT 0,
    success_count INTEGER DEFAULT 0,
    failure_count INTEGER DEFAULT 0,
    success_rate DECIMAL(5,2) GENERATED ALWAYS AS (
        CASE
            WHEN usage_count = 0 THEN NULL
            ELSE ROUND((success_count::DECIMAL / usage_count::DECIMAL) * 100, 2)
        END
    ) STORED,

    -- Performance metrics
    avg_render_time_ms INTEGER, -- Average template rendering time
    avg_generation_time_sec INTEGER, -- Average OSA generation time

    -- Metadata
    metadata JSONB DEFAULT '{}',

    -- Timestamps
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    last_used_at TIMESTAMPTZ,

    -- Constraints
    CONSTRAINT osa_prompt_scope_check CHECK (
        (scope = 'system' AND workspace_id IS NULL AND user_id IS NULL) OR
        (scope = 'workspace' AND workspace_id IS NOT NULL AND user_id IS NULL) OR
        (scope = 'user' AND user_id IS NOT NULL)
    ),
    -- Unique constraint: same name can exist once per scope/user/workspace combination
    CONSTRAINT osa_prompt_name_scope_unique UNIQUE(
        name,
        scope,
        COALESCE(workspace_id, '00000000-0000-0000-0000-000000000000'::uuid),
        COALESCE(user_id, '00000000-0000-0000-0000-000000000000'::uuid)
    )
);

-- Indexes for efficient queries
CREATE INDEX idx_osa_prompt_templates_name ON osa_prompt_templates(name);
CREATE INDEX idx_osa_prompt_templates_scope ON osa_prompt_templates(scope);
CREATE INDEX idx_osa_prompt_templates_workspace ON osa_prompt_templates(workspace_id) WHERE workspace_id IS NOT NULL;
CREATE INDEX idx_osa_prompt_templates_user ON osa_prompt_templates(user_id) WHERE user_id IS NOT NULL;
CREATE INDEX idx_osa_prompt_templates_category ON osa_prompt_templates(category) WHERE category IS NOT NULL;
CREATE INDEX idx_osa_prompt_templates_active ON osa_prompt_templates(is_active) WHERE is_active = true;
CREATE INDEX idx_osa_prompt_templates_tags ON osa_prompt_templates USING GIN(tags);
CREATE INDEX idx_osa_prompt_templates_parent ON osa_prompt_templates(parent_template_id) WHERE parent_template_id IS NOT NULL;
CREATE INDEX idx_osa_prompt_templates_last_used ON osa_prompt_templates(last_used_at DESC) WHERE last_used_at IS NOT NULL;

-- =============================================================================
-- OSA TEMPLATE USAGE LOG
-- Track template usage for analytics and optimization
-- =============================================================================

CREATE TABLE IF NOT EXISTS osa_template_usage_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- Template reference
    template_id UUID NOT NULL REFERENCES osa_prompt_templates(id) ON DELETE CASCADE,

    -- User context
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    workspace_id UUID REFERENCES osa_workspaces(id) ON DELETE SET NULL,

    -- Workflow reference (if template was used for app generation)
    workflow_id UUID REFERENCES osa_workflows(id) ON DELETE SET NULL,
    app_id UUID REFERENCES osa_generated_apps(id) ON DELETE SET NULL,

    -- Variables used for rendering
    variables_used JSONB NOT NULL,

    -- Performance metrics
    render_time_ms INTEGER, -- Time to render template
    generation_time_sec INTEGER, -- Total OSA generation time
    tokens_used INTEGER, -- LLM tokens consumed

    -- Outcome
    status VARCHAR(50) NOT NULL, -- 'success', 'failed', 'cancelled'
    error_message TEXT,
    error_details JSONB,

    -- Quality metrics (user feedback)
    user_rating INTEGER CHECK (user_rating >= 1 AND user_rating <= 5),
    user_feedback TEXT,

    -- Timestamp
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_osa_template_usage_template ON osa_template_usage_log(template_id);
CREATE INDEX idx_osa_template_usage_user ON osa_template_usage_log(user_id);
CREATE INDEX idx_osa_template_usage_workspace ON osa_template_usage_log(workspace_id);
CREATE INDEX idx_osa_template_usage_workflow ON osa_template_usage_log(workflow_id);
CREATE INDEX idx_osa_template_usage_status ON osa_template_usage_log(status);
CREATE INDEX idx_osa_template_usage_created ON osa_template_usage_log(created_at DESC);
CREATE INDEX idx_osa_template_usage_rating ON osa_template_usage_log(user_rating) WHERE user_rating IS NOT NULL;

-- =============================================================================
-- UPDATE TRIGGERS
-- Auto-update timestamps and usage statistics
-- =============================================================================

-- Auto-update updated_at timestamp
CREATE OR REPLACE FUNCTION update_osa_prompt_templates_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_osa_prompt_templates_updated_at
    BEFORE UPDATE ON osa_prompt_templates
    FOR EACH ROW
    EXECUTE FUNCTION update_osa_prompt_templates_updated_at();

-- Auto-update usage statistics from usage log
CREATE OR REPLACE FUNCTION update_template_usage_stats()
RETURNS TRIGGER AS $$
BEGIN
    -- Update template usage count and success/failure counts
    UPDATE osa_prompt_templates
    SET
        usage_count = usage_count + 1,
        success_count = success_count + CASE WHEN NEW.status = 'success' THEN 1 ELSE 0 END,
        failure_count = failure_count + CASE WHEN NEW.status = 'failed' THEN 1 ELSE 0 END,
        last_used_at = NEW.created_at,
        avg_render_time_ms = CASE
            WHEN avg_render_time_ms IS NULL THEN NEW.render_time_ms
            ELSE (avg_render_time_ms + NEW.render_time_ms) / 2
        END,
        avg_generation_time_sec = CASE
            WHEN avg_generation_time_sec IS NULL THEN NEW.generation_time_sec
            ELSE (avg_generation_time_sec + NEW.generation_time_sec) / 2
        END
    WHERE id = NEW.template_id;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_update_template_usage_stats
    AFTER INSERT ON osa_template_usage_log
    FOR EACH ROW
    EXECUTE FUNCTION update_template_usage_stats();

-- =============================================================================
-- HELPER FUNCTIONS
-- =============================================================================

-- Resolve template with inheritance (user > workspace > system)
CREATE OR REPLACE FUNCTION resolve_osa_template(
    p_template_name VARCHAR(255),
    p_user_id UUID DEFAULT NULL,
    p_workspace_id UUID DEFAULT NULL
)
RETURNS osa_prompt_templates AS $$
DECLARE
    v_template osa_prompt_templates;
BEGIN
    -- Try user-level template first
    IF p_user_id IS NOT NULL THEN
        SELECT * INTO v_template
        FROM osa_prompt_templates
        WHERE name = p_template_name
          AND scope = 'user'
          AND user_id = p_user_id
          AND is_active = true
        LIMIT 1;

        IF FOUND THEN
            RETURN v_template;
        END IF;
    END IF;

    -- Try workspace-level template
    IF p_workspace_id IS NOT NULL THEN
        SELECT * INTO v_template
        FROM osa_prompt_templates
        WHERE name = p_template_name
          AND scope = 'workspace'
          AND workspace_id = p_workspace_id
          AND is_active = true
        LIMIT 1;

        IF FOUND THEN
            RETURN v_template;
        END IF;
    END IF;

    -- Fall back to system template
    SELECT * INTO v_template
    FROM osa_prompt_templates
    WHERE name = p_template_name
      AND scope = 'system'
      AND is_active = true
    LIMIT 1;

    RETURN v_template;
END;
$$ LANGUAGE plpgsql;

COMMENT ON FUNCTION resolve_osa_template IS 'Resolves template with inheritance: user > workspace > system';

-- Get template usage statistics
CREATE OR REPLACE FUNCTION get_template_usage_stats(
    p_template_id UUID,
    p_start_date TIMESTAMPTZ DEFAULT NOW() - INTERVAL '30 days',
    p_end_date TIMESTAMPTZ DEFAULT NOW()
)
RETURNS TABLE (
    total_uses INTEGER,
    success_count INTEGER,
    failure_count INTEGER,
    success_rate DECIMAL(5,2),
    avg_render_time_ms INTEGER,
    avg_generation_time_sec INTEGER,
    avg_tokens_used INTEGER,
    avg_user_rating DECIMAL(3,2),
    unique_users INTEGER,
    unique_workspaces INTEGER
) AS $$
BEGIN
    RETURN QUERY
    SELECT
        COUNT(*)::INTEGER AS total_uses,
        SUM(CASE WHEN status = 'success' THEN 1 ELSE 0 END)::INTEGER AS success_count,
        SUM(CASE WHEN status = 'failed' THEN 1 ELSE 0 END)::INTEGER AS failure_count,
        ROUND((SUM(CASE WHEN status = 'success' THEN 1 ELSE 0 END)::DECIMAL / COUNT(*)::DECIMAL) * 100, 2) AS success_rate,
        AVG(render_time_ms)::INTEGER AS avg_render_time_ms,
        AVG(generation_time_sec)::INTEGER AS avg_generation_time_sec,
        AVG(tokens_used)::INTEGER AS avg_tokens_used,
        ROUND(AVG(user_rating), 2) AS avg_user_rating,
        COUNT(DISTINCT user_id)::INTEGER AS unique_users,
        COUNT(DISTINCT workspace_id)::INTEGER AS unique_workspaces
    FROM osa_template_usage_log
    WHERE template_id = p_template_id
      AND created_at >= p_start_date
      AND created_at <= p_end_date;
END;
$$ LANGUAGE plpgsql;

COMMENT ON FUNCTION get_template_usage_stats IS 'Returns aggregate statistics for template usage within a date range';

-- List popular templates (by usage)
CREATE OR REPLACE FUNCTION get_popular_templates(
    p_category VARCHAR(100) DEFAULT NULL,
    p_scope VARCHAR(50) DEFAULT NULL,
    p_limit INTEGER DEFAULT 10
)
RETURNS TABLE (
    template_id UUID,
    name VARCHAR(255),
    display_name VARCHAR(255),
    category VARCHAR(100),
    usage_count INTEGER,
    success_rate DECIMAL(5,2),
    avg_user_rating DECIMAL(3,2)
) AS $$
BEGIN
    RETURN QUERY
    SELECT
        t.id AS template_id,
        t.name,
        t.display_name,
        t.category,
        t.usage_count,
        t.success_rate,
        (
            SELECT ROUND(AVG(user_rating), 2)
            FROM osa_template_usage_log
            WHERE template_id = t.id
              AND user_rating IS NOT NULL
        ) AS avg_user_rating
    FROM osa_prompt_templates t
    WHERE t.is_active = true
      AND (p_category IS NULL OR t.category = p_category)
      AND (p_scope IS NULL OR t.scope = p_scope)
    ORDER BY t.usage_count DESC, t.success_rate DESC
    LIMIT p_limit;
END;
$$ LANGUAGE plpgsql;

COMMENT ON FUNCTION get_popular_templates IS 'Returns most popular templates ordered by usage and success rate';

-- =============================================================================
-- SEED DATA: System Templates
-- =============================================================================

-- Seed CRM App Generation Template
INSERT INTO osa_prompt_templates (
    name,
    display_name,
    description,
    scope,
    template_content,
    variables,
    category,
    tags,
    version
) VALUES (
    'crm-app-generation',
    'CRM Application Generation',
    'Generate a full-stack CRM application tailored to the user''s business',
    'system',
    E'# CRM Application Generation\n\nYou are generating a **{{.AppType}}** application for the user''s business.\n\n## User Context\n\n- **Business Domain**: {{.UserBusiness}}\n- **Database Preference**: {{.DatabasePreference}}\n{{- if .ExistingStack}}\n- **Existing Tech Stack**:\n  {{- range $key, $value := .ExistingStack}}\n  - {{$key}}: {{$value}}\n  {{- end}}\n{{- end}}\n\n## Available Integrations\n\n{{- if .AvailableIntegrations}}\nThe following third-party integrations are available:\n{{- range .AvailableIntegrations}}\n- **{{.Name}}**: {{.Description}} ({{.Status}})\n{{- end}}\n{{- else}}\nNo third-party integrations are currently configured. Focus on building a standalone application.\n{{- end}}\n\n## User Requirements\n\n{{.UserRequirements}}\n\n## Your Task\n\nGenerate a production-ready, full-stack {{.AppType}} application with the following:\n\n1. **Database Schema** (PostgreSQL)\n   - Normalized tables for CRM entities (contacts, companies, deals, activities)\n   - Audit fields (created_at, updated_at, created_by)\n   - Indexes for performance\n   - Migration files\n\n2. **Backend API** (Go + Gin)\n   - RESTful endpoints for CRUD operations\n   - Authentication/authorization middleware\n   - Input validation\n   - Repository pattern for data access\n   - Error handling with slog logging (NO fmt.Printf)\n   - Context propagation\n\n3. **Frontend UI** (SvelteKit + TypeScript)\n   - Dashboard with metrics\n   - List/detail views with filtering and pagination\n   - Responsive design (mobile-first)\n   - Dark mode support\n   - Tailwind CSS\n\n4. **Testing**\n   - Unit tests (Go: testify)\n   - Integration tests for API\n   - E2E tests (Playwright)\n   - 80% code coverage minimum\n\n5. **Documentation**\n   - API docs (OpenAPI/Swagger)\n   - README with setup instructions\n   - Architecture decision records\n\n## Technical Constraints\n\n- Follow BusinessOS patterns (Handler → Service → Repository)\n- Use slog for logging (NO fmt.Printf)\n- Database queries via sqlc\n- Error handling: wrap errors with context\n- Context as first parameter in I/O functions\n\n**Begin generation now.**',
    jsonb_build_object(
        'variables', jsonb_build_array(
            jsonb_build_object('name', 'AppType', 'type', 'string', 'required', true, 'description', 'Type of application (e.g., CRM, ERP)'),
            jsonb_build_object('name', 'UserBusiness', 'type', 'string', 'required', true, 'description', 'User''s business domain'),
            jsonb_build_object('name', 'UserRequirements', 'type', 'string', 'required', true, 'description', 'Specific user requirements'),
            jsonb_build_object('name', 'DatabasePreference', 'type', 'string', 'required', false, 'default', 'PostgreSQL', 'description', 'Preferred database'),
            jsonb_build_object('name', 'AvailableIntegrations', 'type', 'array', 'required', false, 'default', '[]'::jsonb, 'description', 'List of integrations'),
            jsonb_build_object('name', 'ExistingStack', 'type', 'object', 'required', false, 'description', 'Existing tech stack info')
        ),
        'required', jsonb_build_array('AppType', 'UserBusiness', 'UserRequirements')
    ),
    'app-generation',
    ARRAY['crm', 'full-stack', 'business'],
    '1.0.0'
) ON CONFLICT DO NOTHING;

-- Seed Data Pipeline Template
INSERT INTO osa_prompt_templates (
    name,
    display_name,
    description,
    scope,
    template_content,
    variables,
    category,
    tags,
    version
) VALUES (
    'data-pipeline-creation',
    'Data Pipeline Creation',
    'Generate ETL/ELT data pipelines with transformation logic',
    'system',
    E'# Data Pipeline Generation\n\nYou are creating a data pipeline to extract data from **{{.SourceType}}**, transform it, and load it into **{{.DestinationType}}**.\n\n## Pipeline Configuration\n\n- **Source**: {{.SourceType}}\n- **Destination**: {{.DestinationType}}\n- **Schedule**: {{.Schedule}}\n- **Expected Volume**: {{.DataVolume}}\n\n## Transformation Logic\n\n{{.TransformationRules}}\n\n## Implementation Requirements\n\n1. **Extraction Layer**\n   - Source connectors for {{.SourceType}}\n   - Pagination, rate limiting, retries\n   - Incremental extraction (watermarks)\n\n2. **Transformation Layer**\n   - Apply business rules above\n   - Data validation and cleansing\n   - Type casting and normalization\n   - Handle nulls, duplicates, edge cases\n\n3. **Loading Layer**\n   - Bulk insert optimization for {{.DestinationType}}\n   - Upsert logic\n   - Transaction management\n\n4. **Orchestration**\n   {{- if eq .Schedule "realtime"}}\n   - Real-time streaming pipeline\n   - Exactly-once delivery\n   {{- else}}\n   - Batch processing: {{.Schedule}}\n   - Dependency management\n   {{- end}}\n\n5. **Monitoring**\n   - Execution logs (slog)\n   - Data quality metrics\n   - Alerting on failures\n   - SLA tracking\n\n6. **Error Handling**\n   - Dead letter queue\n   - Exponential backoff retry\n   - Manual intervention workflow\n\n**Generate the complete pipeline implementation with tests and monitoring.**',
    jsonb_build_object(
        'variables', jsonb_build_array(
            jsonb_build_object('name', 'SourceType', 'type', 'string', 'required', true, 'description', 'Source system type (API, Database, File, Stream)'),
            jsonb_build_object('name', 'DestinationType', 'type', 'string', 'required', true, 'description', 'Destination type (PostgreSQL, BigQuery, S3, Kafka)'),
            jsonb_build_object('name', 'TransformationRules', 'type', 'string', 'required', true, 'description', 'Business logic for transformation'),
            jsonb_build_object('name', 'Schedule', 'type', 'string', 'required', false, 'default', 'daily', 'description', 'Pipeline schedule (cron or realtime)'),
            jsonb_build_object('name', 'DataVolume', 'type', 'string', 'required', false, 'default', 'medium', 'description', 'Expected data volume (small, medium, large)')
        ),
        'required', jsonb_build_array('SourceType', 'DestinationType', 'TransformationRules')
    ),
    'data-engineering',
    ARRAY['etl', 'data-pipeline', 'analytics'],
    '1.0.0'
) ON CONFLICT DO NOTHING;

-- =============================================================================
-- COMMENTS FOR DOCUMENTATION
-- =============================================================================

COMMENT ON TABLE osa_prompt_templates IS 'Template-based prompt system with user customization and versioning';
COMMENT ON TABLE osa_template_usage_log IS 'Audit trail and analytics for template usage';

COMMENT ON COLUMN osa_prompt_templates.scope IS 'system: built-in (read-only), workspace: org-level, user: personal';
COMMENT ON COLUMN osa_prompt_templates.template_content IS 'Go text/template syntax with variable substitution';
COMMENT ON COLUMN osa_prompt_templates.variables IS 'JSON schema defining template variables for validation and UI generation';
COMMENT ON COLUMN osa_prompt_templates.is_active IS 'Only one active version per name/scope/user/workspace combination';
COMMENT ON COLUMN osa_prompt_templates.parent_template_id IS 'Reference to parent template (for versioning)';
COMMENT ON COLUMN osa_prompt_templates.success_rate IS 'Calculated field: (success_count / usage_count) * 100';

COMMENT ON COLUMN osa_template_usage_log.variables_used IS 'Snapshot of variables used for this rendering (for debugging and analytics)';
COMMENT ON COLUMN osa_template_usage_log.user_rating IS 'User feedback rating (1-5 stars)';
