-- Fix missing columns in tasks table
-- This migration adds start_date if it doesn't exist

-- Check and add start_date column to tasks
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'tasks' AND column_name = 'start_date'
    ) THEN
        ALTER TABLE tasks ADD COLUMN start_date TIMESTAMP;
    END IF;
END $$;

-- Verify the column exists
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'tasks' AND column_name = 'start_date'
    ) THEN
        RAISE EXCEPTION 'Failed to add start_date column to tasks table';
    END IF;
END $$;
