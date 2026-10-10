-- Custom Agents table
CREATE TABLE IF NOT EXISTS custom_agents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,

    -- Agent Identity
    name VARCHAR(50) NOT NULL,
    display_name VARCHAR(100) NOT NULL,
    description TEXT,
    avatar VARCHAR(50),

    -- Agent Configuration
    system_prompt TEXT NOT NULL,
    model_preference VARCHAR(100),
    temperature DECIMAL(3,2) DEFAULT 0.7,
    max_tokens INTEGER DEFAULT 4096,

    -- Capabilities
    capabilities TEXT[] DEFAULT '{}',
    tools_enabled TEXT[] DEFAULT '{}',
    context_sources TEXT[] DEFAULT '{}',

    -- Behavior Settings
    thinking_enabled BOOLEAN DEFAULT FALSE,
    streaming_enabled BOOLEAN DEFAULT TRUE,

    -- Agent Type/Category
    category VARCHAR(50) DEFAULT 'general',
    is_public BOOLEAN DEFAULT FALSE,

    -- Usage & Status
    is_active BOOLEAN DEFAULT TRUE,
    times_used INTEGER DEFAULT 0,
    last_used_at TIMESTAMP WITH TIME ZONE,

    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    UNIQUE(user_id, name)
);

CREATE INDEX IF NOT EXISTS idx_custom_agents_user_id ON custom_agents(user_id);
CREATE INDEX IF NOT EXISTS idx_custom_agents_name ON custom_agents(user_id, name);
CREATE INDEX IF NOT EXISTS idx_custom_agents_category ON custom_agents(category);
CREATE INDEX IF NOT EXISTS idx_custom_agents_active ON custom_agents(user_id, is_active);

-- Agent Presets table
CREATE TABLE IF NOT EXISTS agent_presets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- Preset Identity
    name VARCHAR(50) NOT NULL UNIQUE,
    display_name VARCHAR(100) NOT NULL,
    description TEXT,
    avatar VARCHAR(50),

    -- Preset Configuration
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

-- Insert default agent presets with explicit type casting
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
     'coding', ARRAY['code_review', 'analysis']::TEXT[], ARRAY['read_file', 'search_code']::TEXT[], TRUE),

    ('technical-writer', 'Technical Writer', 'Creates clear documentation and technical content', 'pencil',
     'You are an expert technical writer. Create clear, well-structured documentation that:
1. Uses simple, precise language
2. Includes relevant code examples
3. Follows standard documentation patterns
4. Anticipates reader questions
5. Provides both quick-start guides and detailed references

Adapt your writing style to the audience - from beginner-friendly tutorials to expert reference docs.',
     'writing', ARRAY['documentation', 'writing']::TEXT[], ARRAY[]::TEXT[], FALSE),

    ('data-analyst', 'Data Analyst', 'Analyzes data and creates insights', 'chart',
     'You are an expert data analyst. When analyzing data:
1. Start with exploratory analysis to understand the data
2. Identify key patterns, trends, and anomalies
3. Use appropriate statistical methods
4. Create clear visualizations (describe them in detail)
5. Provide actionable insights and recommendations

Be precise with numbers and transparent about limitations or assumptions.',
     'analysis', ARRAY['data_analysis', 'visualization']::TEXT[], ARRAY[]::TEXT[], TRUE),

    ('business-strategist', 'Business Strategist', 'Provides strategic business advice and analysis', 'briefcase',
     'You are a senior business strategist. Provide strategic advice by:
1. Understanding the business context and objectives
2. Analyzing market conditions and competition
3. Identifying opportunities and risks
4. Developing actionable recommendations
5. Considering implementation feasibility

Use frameworks like SWOT, Porter''s Five Forces, and business model canvas when appropriate.',
     'business', ARRAY['strategy', 'analysis', 'planning']::TEXT[], ARRAY[]::TEXT[], TRUE),

    ('creative-writer', 'Creative Writer', 'Helps with creative writing and content creation', 'sparkles',
     'You are a talented creative writer. Help with:
1. Generating creative ideas and concepts
2. Writing engaging narratives and copy
3. Developing compelling characters and stories
4. Crafting persuasive marketing content
5. Editing and improving existing content

Match the desired tone, style, and voice. Be creative while staying on-brand.',
     'writing', ARRAY['creative_writing', 'content_creation']::TEXT[], ARRAY[]::TEXT[], FALSE)
ON CONFLICT (name) DO NOTHING;
