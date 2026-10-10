-- 043_custom_agents_behavior_fields.sql
-- Add behavior and visibility fields to custom agents

-- Add welcome_message column
ALTER TABLE custom_agents
ADD COLUMN IF NOT EXISTS welcome_message TEXT;

-- Add suggested_prompts column (JSON array)
ALTER TABLE custom_agents
ADD COLUMN IF NOT EXISTS suggested_prompts TEXT[] DEFAULT '{}';

-- Add is_featured column
ALTER TABLE custom_agents
ADD COLUMN IF NOT EXISTS is_featured BOOLEAN DEFAULT FALSE;

-- Add comments explaining the columns
COMMENT ON COLUMN custom_agents.welcome_message IS
'Welcome message shown when starting a conversation with this agent';

COMMENT ON COLUMN custom_agents.suggested_prompts IS
'Array of suggested prompts to help users interact with the agent';

COMMENT ON COLUMN custom_agents.is_featured IS
'If true, agent will be prominently displayed in featured list (requires is_public=true)';

-- Create index for featured agents lookup
CREATE INDEX IF NOT EXISTS idx_custom_agents_featured
ON custom_agents(user_id, is_featured, is_public)
WHERE is_featured = TRUE AND is_public = TRUE;
