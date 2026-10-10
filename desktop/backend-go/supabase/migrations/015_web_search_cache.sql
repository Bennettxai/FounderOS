-- Migration 014: Web Search Results Cache
-- Prevents redundant API calls within the same conversation or time window

-- Cache table for web search results
CREATE TABLE IF NOT EXISTS web_search_results (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- Query identification
    query_hash VARCHAR(64) NOT NULL, -- SHA256 hash of normalized query
    original_query TEXT NOT NULL,
    optimized_query TEXT, -- Query after optimization

    -- Context (optional)
    user_id VARCHAR(255),
    conversation_id UUID,

    -- Results
    results JSONB NOT NULL DEFAULT '[]', -- Array of search results
    result_count INTEGER DEFAULT 0,
    provider VARCHAR(50) DEFAULT 'duckduckgo',

    -- Performance metrics
    search_time_ms FLOAT,

    -- Cache management
    expires_at TIMESTAMPTZ NOT NULL, -- When this cache entry expires
    hit_count INTEGER DEFAULT 0, -- Number of times this cache was used
    last_hit_at TIMESTAMPTZ,

    -- Metadata
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Index for fast query hash lookups
CREATE INDEX IF NOT EXISTS idx_web_search_query_hash ON web_search_results(query_hash);

-- Index for user-specific cache lookups
CREATE INDEX IF NOT EXISTS idx_web_search_user ON web_search_results(user_id) WHERE user_id IS NOT NULL;

-- Index for conversation-specific cache
CREATE INDEX IF NOT EXISTS idx_web_search_conversation ON web_search_results(conversation_id) WHERE conversation_id IS NOT NULL;

-- Index for cache expiration cleanup
CREATE INDEX IF NOT EXISTS idx_web_search_expires ON web_search_results(expires_at);

-- Composite index for most common lookup pattern
CREATE INDEX IF NOT EXISTS idx_web_search_lookup ON web_search_results(query_hash, expires_at);

-- Function to clean expired cache entries (call periodically)
CREATE OR REPLACE FUNCTION cleanup_expired_search_cache()
RETURNS INTEGER AS $$
DECLARE
    deleted_count INTEGER;
BEGIN
    DELETE FROM web_search_results WHERE expires_at < NOW();
    GET DIAGNOSTICS deleted_count = ROW_COUNT;
    RETURN deleted_count;
END;
$$ LANGUAGE plpgsql;

-- Add comment explaining the table
COMMENT ON TABLE web_search_results IS 'Cache for web search results to prevent redundant API calls';
COMMENT ON COLUMN web_search_results.query_hash IS 'SHA256 hash of normalized (lowercased, trimmed) query for fast lookups';
COMMENT ON COLUMN web_search_results.expires_at IS 'Cache entries expire after a configurable duration (default 1 hour for general, 15 min for news)';
