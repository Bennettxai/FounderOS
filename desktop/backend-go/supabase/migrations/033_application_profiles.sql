-- ================================================
-- Migration 022: Application Profiles
-- Description: Store context profiles for applications/codebases (future IDE integration)
-- Author: Pedro
-- Date: 2025-12-31
-- ================================================

-- ================================================
-- APPLICATION PROFILES TABLE
-- Store context profiles for applications and codebases
-- Enables future integration with coding assistants
-- ================================================
CREATE TABLE IF NOT EXISTS application_profiles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,

    -- Application Identity
    name VARCHAR(255) NOT NULL,
    description TEXT,
    app_type VARCHAR(100),                    -- 'web_app', 'mobile_app', 'api', 'library', 'platform', 'cli'
    version VARCHAR(50),

    -- Tech Stack
    tech_stack JSONB DEFAULT '{}',            -- {"frontend": "svelte", "backend": "go", "database": "postgres", "hosting": "vercel"}
    languages TEXT[] DEFAULT '{}',            -- ['typescript', 'go', 'sql']
    frameworks TEXT[] DEFAULT '{}',           -- ['sveltekit', 'tailwind', 'drizzle']

    -- Structure (tree representation of the app)
    structure_tree JSONB NOT NULL DEFAULT '{}',
    root_path VARCHAR(1000),                  -- Absolute path to project root

    -- Components Registry
    components JSONB DEFAULT '[]',            -- [{name, path, description, props, events, slots}]
    total_components INTEGER DEFAULT 0,

    -- Modules Registry
    modules JSONB DEFAULT '[]',               -- [{name, path, description, exports, dependencies}]
    total_modules INTEGER DEFAULT 0,

    -- Icons/Assets Registry
    icons JSONB DEFAULT '[]',                 -- [{name, path, usage, type}]
    assets JSONB DEFAULT '[]',                -- [{name, path, type, size}]

    -- API Endpoints
    api_endpoints JSONB DEFAULT '[]',         -- [{method, path, description, params, response, auth}]
    total_endpoints INTEGER DEFAULT 0,

    -- Database Schema Summary
    database_schema JSONB DEFAULT '{}',       -- {tables: [{name, columns, relations}]}
    total_tables INTEGER DEFAULT 0,

    -- Conventions & Patterns
    conventions JSONB DEFAULT '{}',           -- {"naming": {}, "structure": {}, "patterns": [], "style_guide": ""}
    coding_standards TEXT,                    -- Description of coding standards

    -- Integration Points
    integration_points JSONB DEFAULT '[]',    -- External services, APIs [{name, type, url, auth_type}]

    -- Documentation
    readme_summary TEXT,                      -- Extracted from README
    documentation_urls TEXT[] DEFAULT '{}',

    -- Embeddings for semantic search
    embedding vector(1536),

    -- Sync info
    last_synced_at TIMESTAMPTZ,
    sync_source VARCHAR(255),                 -- Git repo URL, file path, etc.
    sync_branch VARCHAR(100),                 -- Git branch
    sync_commit VARCHAR(100),                 -- Last synced commit hash
    auto_sync_enabled BOOLEAN DEFAULT FALSE,

    -- Analysis stats
    last_analyzed_at TIMESTAMPTZ,
    analysis_version INTEGER DEFAULT 1,
    lines_of_code INTEGER,
    file_count INTEGER,

    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Indexes for application_profiles
CREATE INDEX IF NOT EXISTS idx_app_profiles_user ON application_profiles(user_id);
CREATE INDEX IF NOT EXISTS idx_app_profiles_type ON application_profiles(app_type);
CREATE INDEX IF NOT EXISTS idx_app_profiles_name ON application_profiles(user_id, name);
-- Note: embedding index created in migration 037
-- CREATE INDEX IF NOT EXISTS idx_app_profiles_embedding ON application_profiles USING ivfflat (embedding vector_cosine_ops) WITH (lists = 100);

-- ================================================
-- APPLICATION COMPONENTS TABLE
-- Detailed component registry for larger apps
-- ================================================
CREATE TABLE IF NOT EXISTS application_components (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    app_profile_id UUID NOT NULL REFERENCES application_profiles(id) ON DELETE CASCADE,

    -- Component Identity
    name VARCHAR(255) NOT NULL,
    file_path VARCHAR(1000) NOT NULL,
    component_type VARCHAR(100),              -- 'page', 'layout', 'ui', 'form', 'modal', 'widget'

    -- Component Details
    description TEXT,
    props JSONB DEFAULT '[]',                 -- [{name, type, required, default, description}]
    events JSONB DEFAULT '[]',                -- [{name, payload, description}]
    slots JSONB DEFAULT '[]',                 -- [{name, description}]

    -- Dependencies
    imports TEXT[] DEFAULT '{}',              -- Other components/modules imported
    exported_as VARCHAR(255),                 -- Export name if different from file

    -- Usage
    usage_examples JSONB DEFAULT '[]',        -- [{code, description}]
    used_in TEXT[] DEFAULT '{}',              -- Paths where this component is used

    -- Metadata
    lines_of_code INTEGER,
    last_modified_at TIMESTAMPTZ,

    -- Embedding
    embedding vector(1536),

    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),

    UNIQUE(app_profile_id, file_path)
);

-- Indexes for application_components
CREATE INDEX IF NOT EXISTS idx_app_components_profile ON application_components(app_profile_id);
CREATE INDEX IF NOT EXISTS idx_app_components_type ON application_components(component_type);
CREATE INDEX IF NOT EXISTS idx_app_components_name ON application_components(app_profile_id, name);
-- Note: embedding index created in migration 037
-- CREATE INDEX IF NOT EXISTS idx_app_components_embedding ON application_components USING ivfflat (embedding vector_cosine_ops) WITH (lists = 100);

-- ================================================
-- APPLICATION API ENDPOINTS TABLE
-- Detailed API endpoint registry
-- ================================================
CREATE TABLE IF NOT EXISTS application_api_endpoints (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    app_profile_id UUID NOT NULL REFERENCES application_profiles(id) ON DELETE CASCADE,

    -- Endpoint Identity
    method VARCHAR(10) NOT NULL,              -- 'GET', 'POST', 'PUT', 'DELETE', 'PATCH'
    path VARCHAR(500) NOT NULL,
    handler_path VARCHAR(1000),               -- Path to handler file

    -- Endpoint Details
    description TEXT,
    summary VARCHAR(255),

    -- Parameters
    path_params JSONB DEFAULT '[]',           -- [{name, type, description}]
    query_params JSONB DEFAULT '[]',          -- [{name, type, required, description}]
    body_schema JSONB DEFAULT '{}',           -- Request body JSON schema
    response_schema JSONB DEFAULT '{}',       -- Response JSON schema

    -- Auth & Permissions
    auth_required BOOLEAN DEFAULT FALSE,
    required_permissions TEXT[] DEFAULT '{}',

    -- Metadata
    tags TEXT[] DEFAULT '{}',
    deprecated BOOLEAN DEFAULT FALSE,

    -- Embedding
    embedding vector(1536),

    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),

    UNIQUE(app_profile_id, method, path)
);

-- Indexes for application_api_endpoints
CREATE INDEX IF NOT EXISTS idx_app_endpoints_profile ON application_api_endpoints(app_profile_id);
CREATE INDEX IF NOT EXISTS idx_app_endpoints_method ON application_api_endpoints(method);
CREATE INDEX IF NOT EXISTS idx_app_endpoints_path ON application_api_endpoints(app_profile_id, path);
-- Note: embedding index created in migration 037
-- CREATE INDEX IF NOT EXISTS idx_app_endpoints_embedding ON application_api_endpoints USING ivfflat (embedding vector_cosine_ops) WITH (lists = 100);

-- ================================================
-- TRIGGERS
-- ================================================

-- Update updated_at on application_profiles
CREATE OR REPLACE FUNCTION update_app_profiles_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trigger_app_profiles_updated_at ON application_profiles;
CREATE TRIGGER trigger_app_profiles_updated_at
    BEFORE UPDATE ON application_profiles
    FOR EACH ROW
    EXECUTE FUNCTION update_app_profiles_updated_at();

-- Update updated_at on application_components
DROP TRIGGER IF EXISTS trigger_app_components_updated_at ON application_components;
CREATE TRIGGER trigger_app_components_updated_at
    BEFORE UPDATE ON application_components
    FOR EACH ROW
    EXECUTE FUNCTION update_app_profiles_updated_at();

-- Update updated_at on application_api_endpoints
DROP TRIGGER IF EXISTS trigger_app_endpoints_updated_at ON application_api_endpoints;
CREATE TRIGGER trigger_app_endpoints_updated_at
    BEFORE UPDATE ON application_api_endpoints
    FOR EACH ROW
    EXECUTE FUNCTION update_app_profiles_updated_at();

-- Update component/endpoint counts on application_profiles
CREATE OR REPLACE FUNCTION update_app_profile_counts()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_TABLE_NAME = 'application_components' THEN
        UPDATE application_profiles
        SET total_components = (SELECT COUNT(*) FROM application_components WHERE app_profile_id = COALESCE(NEW.app_profile_id, OLD.app_profile_id))
        WHERE id = COALESCE(NEW.app_profile_id, OLD.app_profile_id);
    ELSIF TG_TABLE_NAME = 'application_api_endpoints' THEN
        UPDATE application_profiles
        SET total_endpoints = (SELECT COUNT(*) FROM application_api_endpoints WHERE app_profile_id = COALESCE(NEW.app_profile_id, OLD.app_profile_id))
        WHERE id = COALESCE(NEW.app_profile_id, OLD.app_profile_id);
    END IF;
    RETURN COALESCE(NEW, OLD);
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trigger_update_component_count ON application_components;
CREATE TRIGGER trigger_update_component_count
    AFTER INSERT OR DELETE ON application_components
    FOR EACH ROW
    EXECUTE FUNCTION update_app_profile_counts();

DROP TRIGGER IF EXISTS trigger_update_endpoint_count ON application_api_endpoints;
CREATE TRIGGER trigger_update_endpoint_count
    AFTER INSERT OR DELETE ON application_api_endpoints
    FOR EACH ROW
    EXECUTE FUNCTION update_app_profile_counts();

-- ================================================
-- COMMENTS
-- ================================================
COMMENT ON TABLE application_profiles IS 'Context profiles for applications and codebases';
COMMENT ON TABLE application_components IS 'Detailed component registry for applications';
COMMENT ON TABLE application_api_endpoints IS 'API endpoint registry for applications';

COMMENT ON COLUMN application_profiles.app_type IS 'Type: web_app, mobile_app, api, library, platform, cli';
COMMENT ON COLUMN application_profiles.tech_stack IS 'JSON: {"frontend": "", "backend": "", "database": "", "hosting": ""}';
COMMENT ON COLUMN application_profiles.structure_tree IS 'JSON tree representation of file/folder structure';
COMMENT ON COLUMN application_components.component_type IS 'Type: page, layout, ui, form, modal, widget';
