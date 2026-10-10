-- Migration 027: Add all thinking-related columns to user_settings
-- Adds all missing thinking columns required by UpdateThinkingSettings query

ALTER TABLE user_settings
ADD COLUMN IF NOT EXISTS thinking_enabled BOOLEAN DEFAULT FALSE,
ADD COLUMN IF NOT EXISTS thinking_show_in_ui BOOLEAN DEFAULT TRUE,
ADD COLUMN IF NOT EXISTS thinking_save_traces BOOLEAN DEFAULT FALSE,
ADD COLUMN IF NOT EXISTS thinking_default_template_id UUID,
ADD COLUMN IF NOT EXISTS thinking_max_tokens INTEGER DEFAULT 2048;

-- Update existing rows to have default values
UPDATE user_settings
SET
  thinking_enabled = COALESCE(thinking_enabled, FALSE),
  thinking_show_in_ui = COALESCE(thinking_show_in_ui, TRUE),
  thinking_save_traces = COALESCE(thinking_save_traces, FALSE),
  thinking_max_tokens = COALESCE(thinking_max_tokens, 2048)
WHERE thinking_enabled IS NULL OR thinking_show_in_ui IS NULL;
