-- Migration 092: Fix app_generation_queue missing updated_at column
-- This fixes a schema mismatch where updated_at might be missing from the table

-- Add updated_at column if it doesn't exist
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'app_generation_queue'
        AND column_name = 'updated_at'
    ) THEN
        ALTER TABLE app_generation_queue
        ADD COLUMN updated_at TIMESTAMPTZ DEFAULT NOW();
    END IF;
END $$;

-- Also ensure priority constraint allows 1-10 range (fix any existing values)
UPDATE app_generation_queue
SET priority = CASE
    WHEN priority < 1 THEN 1
    WHEN priority > 10 THEN 10
    ELSE priority
END
WHERE priority < 1 OR priority > 10;
