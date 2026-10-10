-- Migration 009: Custom Agents System
-- Allows users to create and configure custom AI agents

-- ===== CUSTOM AGENTS =====

-- User-defined custom agents with custom system prompts and configurations
CREATE TABLE IF NOT EXISTS custom_agents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,

    -- Agent Identity
    name VARCHAR(50) NOT NULL,              -- e.g., "code-reviewer" (internal name, lowercase)
    display_name VARCHAR(100) NOT NULL,     -- e.g., "Code Reviewer" (shown in UI)
    description TEXT,                       -- What the agent does
    avatar VARCHAR(50),                     -- emoji or icon identifier

    -- Agent Configuration
    system_prompt TEXT NOT NULL,            -- Base system prompt for the agent
    model_preference VARCHAR(100),          -- Preferred model (e.g., "claude-3-opus")
    temperature DECIMAL(3,2) DEFAULT 0.7,   -- Default temperature
    max_tokens INTEGER DEFAULT 4096,        -- Default max tokens

    -- Capabilities
    capabilities TEXT[] DEFAULT '{}',       -- e.g., ["code_review", "analysis", "writing"]
    tools_enabled TEXT[] DEFAULT '{}',      -- Which tools the agent can use
    context_sources TEXT[] DEFAULT '{}',    -- What context to auto-load: documents, projects, etc.

    -- Behavior Settings
    thinking_enabled BOOLEAN DEFAULT FALSE,  -- Enable COT for this agent
    streaming_enabled BOOLEAN DEFAULT TRUE,  -- Enable streaming responses

    -- Agent Type/Category
    category VARCHAR(50) DEFAULT 'general', -- general, coding, writing, analysis, business, custom
    is_public BOOLEAN DEFAULT FALSE,        -- Whether to share with team (future)

    -- Usage & Status
    is_active BOOLEAN DEFAULT TRUE,
    times_used INTEGER DEFAULT 0,
    last_used_at TIMESTAMP WITH TIME ZONE,

    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    UNIQUE(user_id, name)
);

CREATE INDEX idx_custom_agents_user_id ON custom_agents(user_id);
CREATE INDEX idx_custom_agents_name ON custom_agents(user_id, name);
CREATE INDEX idx_custom_agents_category ON custom_agents(category);
CREATE INDEX idx_custom_agents_active ON custom_agents(user_id, is_active);

-- ===== AGENT PRESETS (Optional Built-in Templates) =====

-- Store commonly used agent templates that users can copy
CREATE TABLE IF NOT EXISTS agent_presets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- Preset Identity
    name VARCHAR(50) NOT NULL UNIQUE,
    display_name VARCHAR(100) NOT NULL,
    description TEXT,
    avatar VARCHAR(50),

    -- Preset Configuration (same as custom_agents)
    system_prompt TEXT NOT NULL,
    model_preference VARCHAR(100),
    temperature DECIMAL(3,2) DEFAULT 0.7,
    max_tokens INTEGER DEFAULT 4096,
    capabilities TEXT[] DEFAULT '{}',
    tools_enabled TEXT[] DEFAULT '{}',
    context_sources TEXT[] DEFAULT '{}',
    thinking_enabled BOOLEAN DEFAULT FALSE,
    category VARCHAR(50) DEFAULT 'general',

    -- Usage tracking
    times_copied INTEGER DEFAULT 0,

    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Insert default agent presets
INSERT INTO agent_presets (name, display_name, description, avatar, system_prompt, category, capabilities, tools_enabled, thinking_enabled)
VALUES
    ('code-reviewer', 'Code Reviewer', 'Reviews code for bugs, best practices, and improvements', 'magnifying-glass',
     'You are an expert code reviewer. Analyze code for:
1. **Bugs & Errors**: Identify potential bugs, edge cases, and runtime errors
2. **Best Practices**: Check adherence to coding standards and conventions
3. **Performance**: Spot inefficiencies and suggest optimizations
4. **Security**: Flag potential security vulnerabilities
5. **Maintainability**: Assess code readability and suggest improvements

Provide specific, actionable feedback with code examples when suggesting changes.',
     'coding', ARRAY['code_review', 'analysis'], ARRAY['read_file', 'search_code'], TRUE),

    ('technical-writer', 'Technical Writer', 'Creates clear documentation and technical content', 'pencil',
     'You are an expert technical writer. Create clear, well-structured documentation that:
1. Uses simple, precise language
2. Includes relevant code examples
3. Follows standard documentation patterns
4. Anticipates reader questions
5. Provides both quick-start guides and detailed references

Adapt your writing style to the audience - from beginner-friendly tutorials to expert reference docs.',
     'writing', ARRAY['documentation', 'writing'], ARRAY[]::text[], FALSE),

    ('data-analyst', 'Data Analyst', 'Analyzes data and creates insights', 'chart',
     'You are an expert data analyst. When analyzing data:
1. Start with exploratory analysis to understand the data
2. Identify key patterns, trends, and anomalies
3. Use appropriate statistical methods
4. Create clear visualizations (describe them in detail)
5. Provide actionable insights and recommendations

Be precise with numbers and transparent about limitations or assumptions.',
     'analysis', ARRAY['data_analysis', 'visualization'], ARRAY[]::text[], TRUE),

    ('business-strategist', 'Business Strategist', 'Provides strategic business advice and analysis', 'briefcase',
     'You are a senior business strategist. Provide strategic advice by:
1. Understanding the business context and objectives
2. Analyzing market conditions and competition
3. Identifying opportunities and risks
4. Developing actionable recommendations
5. Considering implementation feasibility

Use frameworks like SWOT, Porter''s Five Forces, and business model canvas when appropriate.',
     'business', ARRAY['strategy', 'analysis', 'planning'], ARRAY[]::text[], TRUE),

    ('creative-writer', 'Creative Writer', 'Helps with creative writing and content creation', 'sparkles',
     'You are a talented creative writer. Help with:
1. Generating creative ideas and concepts
2. Writing engaging narratives and copy
3. Developing compelling characters and stories
4. Crafting persuasive marketing content
5. Editing and improving existing content

Match the desired tone, style, and voice. Be creative while staying on-brand.',
     'writing', ARRAY['creative_writing', 'content_creation'], ARRAY[]::text[], FALSE)
ON CONFLICT (name) DO NOTHING;
