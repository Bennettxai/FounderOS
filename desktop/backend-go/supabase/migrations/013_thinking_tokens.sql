-- Migration 012: Add thinking_tokens to usage tracking
-- Track thinking/reasoning tokens separately for cost analysis

-- Add thinking_tokens column to ai_usage_logs
ALTER TABLE ai_usage_logs
ADD COLUMN IF NOT EXISTS thinking_tokens INTEGER DEFAULT 0;

-- Add comment explaining the field
COMMENT ON COLUMN ai_usage_logs.thinking_tokens IS 'Number of tokens used in Chain of Thought/reasoning (tracked separately for cost analysis)';

-- Add thinking_tokens to daily summary
ALTER TABLE usage_daily_summary
ADD COLUMN IF NOT EXISTS ai_thinking_tokens BIGINT DEFAULT 0;

COMMENT ON COLUMN usage_daily_summary.ai_thinking_tokens IS 'Total thinking tokens used (for COT cost analysis)';
