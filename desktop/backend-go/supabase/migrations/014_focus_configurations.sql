-- Migration 013: Focus Configurations System
-- Stores user-specific overrides for Quick, Deep, and Creative modes

-- Focus configuration templates (system-level defaults)
CREATE TABLE IF NOT EXISTS focus_mode_templates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(50) NOT NULL UNIQUE,
    display_name VARCHAR(100) NOT NULL,
    description TEXT,
    icon VARCHAR(50),

    -- Behavior settings
    default_model VARCHAR(100),
    temperature DECIMAL(3,2) DEFAULT 0.7,
    max_tokens INTEGER DEFAULT 4096,

    -- Output constraints
    output_style VARCHAR(50) DEFAULT 'balanced', -- concise, balanced, detailed, structured
    response_format VARCHAR(50) DEFAULT 'markdown', -- markdown, plain, json, artifact
    max_response_length INTEGER, -- null = no limit
    require_sources BOOLEAN DEFAULT false,

    -- Context settings
    auto_search BOOLEAN DEFAULT false,
    search_depth VARCHAR(20) DEFAULT 'quick', -- quick, standard, deep
    kb_context_limit INTEGER DEFAULT 5, -- max KB items to inject
    include_history_count INTEGER DEFAULT 10, -- conversation history to include

    -- Thinking/COT settings
    thinking_enabled BOOLEAN DEFAULT false,
    thinking_style VARCHAR(50), -- analytical, creative, step-by-step

    -- System prompt additions
    system_prompt_prefix TEXT,
    system_prompt_suffix TEXT,

    -- Metadata
    sort_order INTEGER DEFAULT 0,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- User-specific focus configuration overrides
CREATE TABLE IF NOT EXISTS focus_configurations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    template_id UUID REFERENCES focus_mode_templates(id) ON DELETE CASCADE,

    -- Override settings (null = use template default)
    custom_name VARCHAR(100),
    temperature DECIMAL(3,2),
    max_tokens INTEGER,
    output_style VARCHAR(50),
    response_format VARCHAR(50),
    max_response_length INTEGER,
    require_sources BOOLEAN,
    auto_search BOOLEAN,
    search_depth VARCHAR(20),
    kb_context_limit INTEGER,
    include_history_count INTEGER,
    thinking_enabled BOOLEAN,
    thinking_style VARCHAR(50),
    custom_system_prompt TEXT,

    -- Preferred model override
    preferred_model VARCHAR(100),

    -- Auto-load KB categories
    auto_load_kb_categories TEXT[], -- array of category slugs to auto-include

    -- Keyboard shortcut
    keyboard_shortcut VARCHAR(20),

    -- Metadata
    is_favorite BOOLEAN DEFAULT false,
    use_count INTEGER DEFAULT 0,
    last_used_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),

    UNIQUE(user_id, template_id)
);

-- Focus mode context presets (for auto-load)
CREATE TABLE IF NOT EXISTS focus_context_presets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    name VARCHAR(100) NOT NULL,
    description TEXT,

    -- What to include
    kb_artifact_ids UUID[], -- specific artifacts to include
    kb_categories TEXT[], -- category slugs
    project_ids UUID[], -- projects to include context from

    -- Search settings
    default_search_queries TEXT[], -- pre-defined searches to run
    search_domains TEXT[], -- domains to search

    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Link presets to focus configurations
CREATE TABLE IF NOT EXISTS focus_configuration_presets (
    focus_config_id UUID REFERENCES focus_configurations(id) ON DELETE CASCADE,
    preset_id UUID REFERENCES focus_context_presets(id) ON DELETE CASCADE,
    sort_order INTEGER DEFAULT 0,
    PRIMARY KEY (focus_config_id, preset_id)
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_focus_configurations_user ON focus_configurations(user_id);
CREATE INDEX IF NOT EXISTS idx_focus_configurations_template ON focus_configurations(template_id);
CREATE INDEX IF NOT EXISTS idx_focus_context_presets_user ON focus_context_presets(user_id);

-- Insert default focus mode templates
INSERT INTO focus_mode_templates (name, display_name, description, icon, default_model, temperature, output_style, auto_search, search_depth, thinking_enabled, thinking_style, system_prompt_prefix, sort_order) VALUES
('quick', 'Quick', 'Fast, concise responses for simple questions', 'zap', NULL, 0.5, 'concise', false, 'quick', false, NULL,
'You are in Quick Mode. Provide brief, direct answers. Be concise and to the point. Avoid unnecessary elaboration.', 1),

('deep', 'Deep Research', 'Thorough research with sources and citations', 'search', 'claude-sonnet-4-20250514', 0.7, 'detailed', true, 'deep', true, 'analytical',
'You are in Deep Research Mode. Conduct thorough research and provide comprehensive, well-sourced answers. Include citations where possible. Analyze multiple perspectives.', 2),

('creative', 'Creative', 'Imaginative and exploratory responses', 'sparkles', NULL, 0.9, 'balanced', false, 'quick', true, 'creative',
'You are in Creative Mode. Think outside the box. Explore unconventional ideas and approaches. Be imaginative and innovative in your responses.', 3),

('analyze', 'Analysis', 'Data-driven analysis and insights', 'chart-bar', 'claude-sonnet-4-20250514', 0.6, 'structured', false, 'standard', true, 'analytical',
'You are in Analysis Mode. Focus on data-driven insights. Structure your response with clear sections. Use quantitative reasoning where applicable.', 4),

('write', 'Writing', 'Document creation and editing', 'file-text', NULL, 0.7, 'detailed', false, 'quick', false, NULL,
'You are in Writing Mode. Create well-structured, polished content. Focus on clarity, flow, and appropriate tone. Generate artifacts for longer documents.', 5),

('plan', 'Planning', 'Strategic planning and project organization', 'clipboard-list', NULL, 0.6, 'structured', false, 'standard', true, 'step-by-step',
'You are in Planning Mode. Create actionable plans with clear steps. Consider dependencies and timelines. Structure output as organized lists or project artifacts.', 6),

('code', 'Coding', 'Software development assistance', 'code', 'claude-sonnet-4-20250514', 0.4, 'structured', false, 'quick', true, 'step-by-step',
'You are in Coding Mode. Write clean, efficient code. Follow best practices. Include comments where helpful. Generate code artifacts for complete implementations.', 7)

ON CONFLICT (name) DO UPDATE SET
    display_name = EXCLUDED.display_name,
    description = EXCLUDED.description,
    icon = EXCLUDED.icon,
    default_model = EXCLUDED.default_model,
    temperature = EXCLUDED.temperature,
    output_style = EXCLUDED.output_style,
    auto_search = EXCLUDED.auto_search,
    search_depth = EXCLUDED.search_depth,
    thinking_enabled = EXCLUDED.thinking_enabled,
    thinking_style = EXCLUDED.thinking_style,
    system_prompt_prefix = EXCLUDED.system_prompt_prefix,
    sort_order = EXCLUDED.sort_order,
    updated_at = NOW();
