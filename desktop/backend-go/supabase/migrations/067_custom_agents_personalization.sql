-- 042_custom_agents_personalization.sql
-- Add personalization support to custom agents

-- Add apply_personalization column to custom_agents table
ALTER TABLE custom_agents
ADD COLUMN IF NOT EXISTS apply_personalization BOOLEAN DEFAULT FALSE;

-- Add comment explaining the column
COMMENT ON COLUMN custom_agents.apply_personalization IS
'If true, the agent will use prompt personalizations from the learning system';

-- Create index for faster lookups on personalization-enabled agents
CREATE INDEX IF NOT EXISTS idx_custom_agents_personalization
ON custom_agents(user_id, apply_personalization)
WHERE apply_personalization = TRUE;

-- Update existing agents to have apply_personalization = FALSE by default
UPDATE custom_agents
SET apply_personalization = FALSE
WHERE apply_personalization IS NULL;
