-- DIKW Classification for workspace memories
-- Maps memories to Data-Information-Knowledge-Wisdom hierarchy

ALTER TABLE workspace_memories ADD COLUMN IF NOT EXISTS dikw_level TEXT DEFAULT 'DATA';

-- Backfill existing rows based on memory_type
UPDATE workspace_memories SET dikw_level = CASE
    WHEN memory_type IN ('fact', 'observation', 'log', 'event') THEN 'DATA'
    WHEN memory_type IN ('process', 'pattern', 'summary', 'context') THEN 'INFORMATION'
    WHEN memory_type IN ('knowledge', 'decision', 'lesson', 'insight') THEN 'KNOWLEDGE'
    WHEN memory_type IN ('policy', 'principle', 'strategy', 'value') THEN 'WISDOM'
    ELSE 'DATA'
END
WHERE dikw_level IS NULL OR dikw_level = 'DATA';

CREATE INDEX IF NOT EXISTS idx_workspace_memories_dikw ON workspace_memories (workspace_id, dikw_level);

COMMENT ON COLUMN workspace_memories.dikw_level IS 'DIKW hierarchy level: DATA, INFORMATION, KNOWLEDGE, WISDOM';
