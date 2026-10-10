-- Migration 010: Custom Commands System
-- Maps /slash commands to agents or prompt templates

-- ===== CUSTOM COMMANDS =====

-- Command registry mapping /triggers to actions
CREATE TABLE IF NOT EXISTS custom_commands (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,

    -- Command Identity
    trigger VARCHAR(50) NOT NULL,           -- e.g., "/review" (must start with /)
    display_name VARCHAR(100) NOT NULL,     -- e.g., "Code Review"
    description TEXT,                       -- What the command does
    
    -- Command Action
    action_type VARCHAR(50) NOT NULL,       -- 'agent', 'template', 'tool'
    target_agent_id UUID,                   -- FK to custom_agents (if action_type = 'agent')
    prompt_template TEXT,                   -- Template with {{placeholders}} (if action_type = 'template')
    tool_name VARCHAR(100),                 -- Tool to execute (if action_type = 'tool')
    
    -- Command Behavior
    requires_input BOOLEAN DEFAULT FALSE,   -- Whether command needs user input after trigger
    input_placeholder TEXT,                 -- Placeholder text for input (e.g., "Enter code to review...")
    
    -- Command Configuration
    parameters JSONB DEFAULT '{}',          -- Additional configuration
    streaming_enabled BOOLEAN DEFAULT TRUE, -- Enable streaming for this command
    thinking_enabled BOOLEAN DEFAULT FALSE, -- Enable COT for this command
    
    -- Metadata
    category VARCHAR(50) DEFAULT 'general', -- general, coding, writing, analysis, productivity
    is_active BOOLEAN DEFAULT TRUE,
    is_system BOOLEAN DEFAULT FALSE,        -- System commands cannot be deleted by users
    times_used INTEGER DEFAULT 0,
    last_used_at TIMESTAMP WITH TIME ZONE,
    
    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    UNIQUE(user_id, trigger)
);

CREATE INDEX idx_custom_commands_user_id ON custom_commands(user_id);
CREATE INDEX idx_custom_commands_trigger ON custom_commands(user_id, trigger);
CREATE INDEX idx_custom_commands_active ON custom_commands(user_id, is_active);
CREATE INDEX idx_custom_commands_category ON custom_commands(category);

-- Add FK constraint
ALTER TABLE custom_commands 
ADD CONSTRAINT fk_custom_commands_agent 
FOREIGN KEY (target_agent_id) REFERENCES custom_agents(id) ON DELETE SET NULL;

-- ===== AGENT MENTIONS TRACKING =====

-- Track @agent mentions in messages for context
CREATE TABLE IF NOT EXISTS agent_mentions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    conversation_id UUID NOT NULL,
    message_id UUID NOT NULL,
    
    -- Mention details
    mentioned_agent_id UUID,                -- FK to custom_agents
    mention_text VARCHAR(100) NOT NULL,     -- The actual @mention (e.g., "@code-reviewer")
    position_in_message INT,                -- Character position of mention
    
    -- Resolution
    resolved BOOLEAN DEFAULT TRUE,          -- Whether agent was invoked
    resolution_note TEXT,                   -- Why it failed (if any)
    
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_agent_mentions_user_id ON agent_mentions(user_id);
CREATE INDEX idx_agent_mentions_conversation ON agent_mentions(conversation_id);
CREATE INDEX idx_agent_mentions_agent ON agent_mentions(mentioned_agent_id);

-- Add FK constraint
ALTER TABLE agent_mentions 
ADD CONSTRAINT fk_agent_mentions_agent 
FOREIGN KEY (mentioned_agent_id) REFERENCES custom_agents(id) ON DELETE CASCADE;

-- ===== SEED SYSTEM COMMANDS =====

-- Insert default system commands (placeholder - will reference actual agents after they're created)
INSERT INTO custom_commands (user_id, trigger, display_name, description, action_type, prompt_template, category, is_system, streaming_enabled, thinking_enabled)
VALUES
    ('SYSTEM', '/help', 'Show Help', 'Display available commands and agents', 'template', 
     'Here are the available commands and agents:\n\n**Commands:**\n{{commands_list}}\n\n**Agents:**\n{{agents_list}}', 
     'productivity', TRUE, TRUE, FALSE),
    
    ('SYSTEM', '/clear', 'Clear Context', 'Clear conversation context', 'tool', 
     NULL, 
     'productivity', TRUE, FALSE, FALSE),
    
    ('SYSTEM', '/summarize', 'Summarize Conversation', 'Create a summary of the current conversation', 'template',
     'Please provide a concise summary of this conversation, highlighting:\n1. Key topics discussed\n2. Decisions made\n3. Action items identified\n4. Open questions remaining',
     'productivity', TRUE, TRUE, TRUE)
ON CONFLICT (user_id, trigger) DO NOTHING;
