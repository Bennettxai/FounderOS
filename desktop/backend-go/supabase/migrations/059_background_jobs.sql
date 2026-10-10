-- Migration 036: Background Jobs System
-- Description: Implements reliable task queue for async operations with retry logic and scheduling

-- ============================================================================
-- BACKGROUND JOBS TABLE
-- ============================================================================
-- Stores individual job execution records with retry logic and worker locking

CREATE TABLE IF NOT EXISTS background_jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- Job Definition
    job_type VARCHAR(100) NOT NULL,           -- 'email_send', 'report_generate', 'sync_calendar', etc.
    payload JSONB NOT NULL,                   -- Job-specific parameters

    -- Scheduling
    scheduled_at TIMESTAMPTZ DEFAULT NOW(),   -- When the job should run
    priority INTEGER DEFAULT 0,               -- Higher number = higher priority

    -- Execution Status
    status VARCHAR(50) DEFAULT 'pending',     -- 'pending', 'running', 'completed', 'failed', 'cancelled'
    started_at TIMESTAMPTZ,                   -- When execution began
    completed_at TIMESTAMPTZ,                 -- When execution finished

    -- Worker Management
    worker_id VARCHAR(100),                   -- ID of worker processing this job
    locked_until TIMESTAMPTZ,                 -- Lock expiry time (prevents duplicate processing)

    -- Retry Logic
    attempt_count INTEGER DEFAULT 0,          -- Number of attempts made
    max_attempts INTEGER DEFAULT 3,           -- Maximum retry attempts
    last_error TEXT,                          -- Error from last failed attempt

    -- Result Storage
    result JSONB,                             -- Job execution result

    -- Metadata
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Index for efficient job polling by workers
CREATE INDEX IF NOT EXISTS idx_background_jobs_status
ON background_jobs(status, scheduled_at, priority DESC);

-- Index for filtering by job type
CREATE INDEX IF NOT EXISTS idx_background_jobs_type
ON background_jobs(job_type);

-- Index for worker queries
CREATE INDEX IF NOT EXISTS idx_background_jobs_worker
ON background_jobs(worker_id) WHERE worker_id IS NOT NULL;

-- Index for cleanup queries
CREATE INDEX IF NOT EXISTS idx_background_jobs_created
ON background_jobs(created_at) WHERE status IN ('completed', 'failed', 'cancelled');

-- ============================================================================
-- SCHEDULED JOBS TABLE
-- ============================================================================
-- Stores recurring job definitions with cron-like scheduling

CREATE TABLE IF NOT EXISTS scheduled_jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- Job Definition
    job_type VARCHAR(100) NOT NULL,           -- Type of job to create
    payload JSONB NOT NULL,                   -- Default payload for created jobs

    -- Schedule Configuration
    cron_expression VARCHAR(100) NOT NULL,    -- "0 9 * * 1-5" (9am weekdays), "*/5 * * * *" (every 5 min)
    timezone VARCHAR(50) DEFAULT 'UTC',       -- Timezone for cron calculation

    -- Status & Execution Tracking
    is_active BOOLEAN DEFAULT TRUE,           -- Whether this scheduled job is enabled
    last_run_at TIMESTAMPTZ,                  -- When it last created a background_job
    next_run_at TIMESTAMPTZ,                  -- When it should next run (calculated from cron)

    -- Metadata
    name VARCHAR(255),                        -- Optional friendly name
    description TEXT,                         -- Optional description
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Index for scheduler polling
CREATE INDEX IF NOT EXISTS idx_scheduled_jobs_next_run
ON scheduled_jobs(next_run_at) WHERE is_active = TRUE;

-- Index for filtering by job type
CREATE INDEX IF NOT EXISTS idx_scheduled_jobs_type
ON scheduled_jobs(job_type);

-- ============================================================================
-- HELPER FUNCTIONS
-- ============================================================================

-- Function to acquire next available job (with atomic locking)
CREATE OR REPLACE FUNCTION acquire_background_job(
    p_worker_id VARCHAR(100),
    p_lock_duration_seconds INTEGER DEFAULT 300  -- 5 minutes default lock
)
RETURNS TABLE (
    job_id UUID,
    job_type VARCHAR(100),
    payload JSONB,
    attempt_count INTEGER,
    max_attempts INTEGER
) AS $$
DECLARE
    v_job_id UUID;
BEGIN
    -- Find and lock the next available job atomically
    SELECT id INTO v_job_id
    FROM background_jobs
    WHERE status = 'pending'
      AND scheduled_at <= NOW()
      AND (locked_until IS NULL OR locked_until < NOW())
      AND attempt_count < max_attempts
    ORDER BY priority DESC, scheduled_at ASC
    LIMIT 1
    FOR UPDATE SKIP LOCKED;

    -- If no job found, return empty
    IF v_job_id IS NULL THEN
        RETURN;
    END IF;

    -- Update job with lock and status
    UPDATE background_jobs
    SET
        status = 'running',
        worker_id = p_worker_id,
        locked_until = NOW() + (p_lock_duration_seconds || ' seconds')::INTERVAL,
        started_at = CASE WHEN started_at IS NULL THEN NOW() ELSE started_at END,
        attempt_count = attempt_count + 1
    WHERE id = v_job_id;

    -- Return job details
    RETURN QUERY
    SELECT
        b.id,
        b.job_type,
        b.payload,
        b.attempt_count,
        b.max_attempts
    FROM background_jobs b
    WHERE b.id = v_job_id;
END;
$$ LANGUAGE plpgsql;

-- Function to calculate next retry time with exponential backoff
CREATE OR REPLACE FUNCTION calculate_retry_time(
    p_attempt_count INTEGER
)
RETURNS TIMESTAMPTZ AS $$
BEGIN
    -- Exponential backoff: 1min, 5min, 15min, 15min, ...
    RETURN CASE
        WHEN p_attempt_count <= 1 THEN NOW() + INTERVAL '1 minute'
        WHEN p_attempt_count = 2 THEN NOW() + INTERVAL '5 minutes'
        ELSE NOW() + INTERVAL '15 minutes'
    END;
END;
$$ LANGUAGE plpgsql;

-- Function to release stuck jobs (for cleanup)
CREATE OR REPLACE FUNCTION release_stuck_jobs()
RETURNS INTEGER AS $$
DECLARE
    v_count INTEGER;
BEGIN
    -- Release jobs that have been locked too long
    UPDATE background_jobs
    SET
        status = 'pending',
        locked_until = NULL,
        worker_id = NULL
    WHERE status = 'running'
      AND locked_until < NOW();

    GET DIAGNOSTICS v_count = ROW_COUNT;
    RETURN v_count;
END;
$$ LANGUAGE plpgsql;

-- ============================================================================
-- COMMENTS
-- ============================================================================

COMMENT ON TABLE background_jobs IS 'Stores background job execution records with retry logic and worker locking';
COMMENT ON TABLE scheduled_jobs IS 'Stores recurring job definitions with cron-like scheduling';
COMMENT ON FUNCTION acquire_background_job IS 'Atomically acquires next available job for a worker';
COMMENT ON FUNCTION calculate_retry_time IS 'Calculates next retry time with exponential backoff';
COMMENT ON FUNCTION release_stuck_jobs IS 'Releases jobs stuck in running state (for cleanup)';
