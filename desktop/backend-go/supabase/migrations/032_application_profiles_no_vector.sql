-- ================================================
-- Migration 022: Application Profiles (NO VECTOR)
-- ================================================

CREATE TABLE IF NOT EXISTS application_profiles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    app_type VARCHAR(100),
    version VARCHAR(50),
    tech_stack JSONB DEFAULT '{}',
    languages TEXT[] DEFAULT '{}',
    frameworks TEXT[] DEFAULT '{}',
    structure_tree JSONB NOT NULL DEFAULT '{}',
    root_path VARCHAR(1000),
    components JSONB DEFAULT '[]',
    total_components INTEGER DEFAULT 0,
    modules JSONB DEFAULT '[]',
    total_modules INTEGER DEFAULT 0,
    icons JSONB DEFAULT '[]',
    assets JSONB DEFAULT '[]',
    api_endpoints JSONB DEFAULT '[]',
    total_endpoints INTEGER DEFAULT 0,
    database_schema JSONB DEFAULT '{}',
    total_tables INTEGER DEFAULT 0,
    conventions JSONB DEFAULT '{}',
    coding_standards TEXT,
    integration_points JSONB DEFAULT '[]',
    readme_summary TEXT,
    documentation_urls TEXT[] DEFAULT '{}',
    -- embedding TEXT,  -- Placeholder for vector(1536)
    last_synced_at TIMESTAMPTZ,
    sync_source VARCHAR(255),
    sync_branch VARCHAR(100),
    sync_commit VARCHAR(100),
    auto_sync_enabled BOOLEAN DEFAULT FALSE,
    last_analyzed_at TIMESTAMPTZ,
    analysis_version INTEGER DEFAULT 1,
    lines_of_code INTEGER,
    file_count INTEGER,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_app_profiles_user ON application_profiles(user_id);
CREATE INDEX IF NOT EXISTS idx_app_profiles_type ON application_profiles(app_type);
CREATE INDEX IF NOT EXISTS idx_app_profiles_name ON application_profiles(user_id, name);

CREATE TABLE IF NOT EXISTS application_components (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    app_profile_id UUID NOT NULL REFERENCES application_profiles(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    file_path VARCHAR(1000) NOT NULL,
    component_type VARCHAR(100),
    description TEXT,
    props JSONB DEFAULT '[]',
    events JSONB DEFAULT '[]',
    slots JSONB DEFAULT '[]',
    imports TEXT[] DEFAULT '{}',
    exported_as VARCHAR(255),
    usage_examples JSONB DEFAULT '[]',
    used_in TEXT[] DEFAULT '{}',
    lines_of_code INTEGER,
    last_modified_at TIMESTAMPTZ,
    -- embedding TEXT,  -- Placeholder for vector(1536)
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(app_profile_id, file_path)
);

CREATE INDEX IF NOT EXISTS idx_app_components_profile ON application_components(app_profile_id);
CREATE INDEX IF NOT EXISTS idx_app_components_type ON application_components(component_type);
CREATE INDEX IF NOT EXISTS idx_app_components_name ON application_components(app_profile_id, name);

CREATE TABLE IF NOT EXISTS application_api_endpoints (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    app_profile_id UUID NOT NULL REFERENCES application_profiles(id) ON DELETE CASCADE,
    method VARCHAR(10) NOT NULL,
    path VARCHAR(500) NOT NULL,
    handler_path VARCHAR(1000),
    description TEXT,
    summary VARCHAR(255),
    path_params JSONB DEFAULT '[]',
    query_params JSONB DEFAULT '[]',
    body_schema JSONB DEFAULT '{}',
    response_schema JSONB DEFAULT '{}',
    auth_required BOOLEAN DEFAULT FALSE,
    required_permissions TEXT[] DEFAULT '{}',
    tags TEXT[] DEFAULT '{}',
    deprecated BOOLEAN DEFAULT FALSE,
    -- embedding TEXT,  -- Placeholder for vector(1536)
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(app_profile_id, method, path)
);

CREATE INDEX IF NOT EXISTS idx_app_endpoints_profile ON application_api_endpoints(app_profile_id);
CREATE INDEX IF NOT EXISTS idx_app_endpoints_method ON application_api_endpoints(method);
CREATE INDEX IF NOT EXISTS idx_app_endpoints_path ON application_api_endpoints(app_profile_id, path);

-- Triggers
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

DROP TRIGGER IF EXISTS trigger_app_components_updated_at ON application_components;
CREATE TRIGGER trigger_app_components_updated_at
    BEFORE UPDATE ON application_components
    FOR EACH ROW
    EXECUTE FUNCTION update_app_profiles_updated_at();

DROP TRIGGER IF EXISTS trigger_app_endpoints_updated_at ON application_api_endpoints;
CREATE TRIGGER trigger_app_endpoints_updated_at
    BEFORE UPDATE ON application_api_endpoints
    FOR EACH ROW
    EXECUTE FUNCTION update_app_profiles_updated_at();

-- Update counts trigger
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

COMMENT ON TABLE application_profiles IS 'Context profiles for applications and codebases';
COMMENT ON TABLE application_components IS 'Detailed component registry for applications';
COMMENT ON TABLE application_api_endpoints IS 'API endpoint registry for applications';
