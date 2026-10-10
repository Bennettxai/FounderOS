-- Make workflow_id nullable to allow files to be saved without a workflow
-- This is needed because the app_generation_queue doesn't have user_id/osa_workspace_id
-- required to create a proper workflow record

-- First drop the constraint
ALTER TABLE osa_generated_files ALTER COLUMN workflow_id DROP NOT NULL;

-- Verify the change
SELECT column_name, is_nullable, data_type 
FROM information_schema.columns 
WHERE table_name = 'osa_generated_files' AND column_name = 'workflow_id';
