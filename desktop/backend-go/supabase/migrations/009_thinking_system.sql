-- Thinking/reasoning tracking
CREATE TABLE thinking_traces (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL, 
    conversation_id UUID NOT NULL, 
    message_id UUID NOT NULL,

    -- Thinking content
    thinking_content TEXT NOT NULL,        -- The actual thinking/reasoning text
    thinking_type VARCHAR(50),             -- 'analysis', 'planning', 'reflection', 'tool_use'
    step_number INT,                       -- Order in the thinking chain

    -- Timing
    started_at TIMESTAMPTZ DEFAULT NOW(),
    completed_at TIMESTAMPTZ,
    duration_ms INT,

    -- Token tracking
    thinking_tokens INT DEFAULT 0,

    -- Metadata
    model_used VARCHAR(100),
    reasoning_template_id UUID,            -- If using a custom template
    metadata JSONB DEFAULT '{}',

    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Custom reasoning templates/systems
CREATE TABLE reasoning_templates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,

    name VARCHAR(255) NOT NULL,
    description TEXT,

    -- Template configuration
    system_prompt TEXT,                    -- Base system prompt for reasoning
    thinking_instruction TEXT,             -- How to structure thinking
    output_format VARCHAR(50),             -- 'streaming', 'collapsed', 'step_by_step'

    -- Options
    show_thinking BOOLEAN DEFAULT true,    -- Show thinking in UI
    save_thinking BOOLEAN DEFAULT true,    -- Save to database
    max_thinking_tokens INT DEFAULT 4096,

    -- Usage tracking
    times_used INT DEFAULT 0,

    is_default BOOLEAN DEFAULT false,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);
