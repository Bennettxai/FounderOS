-- Migration 044: Fix notification_batches columns
-- Adds missing created_at and updated_at columns

-- Add created_at if missing
ALTER TABLE notification_batches 
ADD COLUMN IF NOT EXISTS created_at TIMESTAMPTZ DEFAULT NOW();

-- Add updated_at if missing
ALTER TABLE notification_batches 
ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ DEFAULT NOW();

-- Backfill created_at from first_at if first_at exists and created_at is null
UPDATE notification_batches 
SET created_at = COALESCE(first_at, NOW()) 
WHERE created_at IS NULL;

-- Comment
COMMENT ON COLUMN notification_batches.created_at IS 'When the batch was created';
COMMENT ON COLUMN notification_batches.updated_at IS 'When the batch was last updated';
