-- Migration 005: Task Dependencies for Gantt View
-- Date: 2025-12-21
-- Description: Adds support for task dependencies (predecessor/successor relationships)
-- Used for Gantt chart visualization and dependency tracking

-- Create task_dependencies table
CREATE TABLE IF NOT EXISTS task_dependencies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    predecessor_id UUID NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    successor_id UUID NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    dependency_type VARCHAR(20) DEFAULT 'finish_to_start',  -- 'finish_to_start', 'start_to_start', 'finish_to_finish', 'start_to_finish'
    lag_days INTEGER DEFAULT 0,  -- Delay between predecessor end and successor start
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(predecessor_id, successor_id),
    -- Prevent self-referential dependencies
    CONSTRAINT no_self_dependency CHECK (predecessor_id != successor_id)
);

-- Create indexes for efficient lookups
CREATE INDEX IF NOT EXISTS idx_task_dependencies_predecessor ON task_dependencies(predecessor_id);
CREATE INDEX IF NOT EXISTS idx_task_dependencies_successor ON task_dependencies(successor_id);
CREATE INDEX IF NOT EXISTS idx_task_dependencies_type ON task_dependencies(dependency_type);

-- Create function to detect circular dependencies
CREATE OR REPLACE FUNCTION check_circular_dependency()
RETURNS TRIGGER AS $$
DECLARE
    has_cycle BOOLEAN;
BEGIN
    -- Use recursive CTE to detect cycles
    WITH RECURSIVE dependency_chain AS (
        -- Start with the successor of the new dependency
        SELECT successor_id AS task_id, 1 AS depth
        FROM task_dependencies
        WHERE predecessor_id = NEW.successor_id

        UNION ALL

        -- Follow the chain
        SELECT td.successor_id, dc.depth + 1
        FROM dependency_chain dc
        JOIN task_dependencies td ON td.predecessor_id = dc.task_id
        WHERE dc.depth < 100  -- Prevent infinite loops with depth limit
    )
    SELECT EXISTS (
        SELECT 1 FROM dependency_chain WHERE task_id = NEW.predecessor_id
    ) INTO has_cycle;

    IF has_cycle THEN
        RAISE EXCEPTION 'Circular dependency detected: this would create a dependency loop';
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create trigger to prevent circular dependencies
DROP TRIGGER IF EXISTS prevent_circular_dependencies ON task_dependencies;
CREATE TRIGGER prevent_circular_dependencies
    BEFORE INSERT OR UPDATE ON task_dependencies
    FOR EACH ROW
    EXECUTE FUNCTION check_circular_dependency();
