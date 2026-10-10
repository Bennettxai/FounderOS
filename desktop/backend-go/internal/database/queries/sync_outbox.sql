-- ============================================================================
-- SYNC OUTBOX QUERIES
-- Transactional outbox pattern for reliable event publishing
-- ============================================================================

-- name: CreateOutboxEvent :one
INSERT INTO sync_outbox (
    aggregate_type,
    aggregate_id,
    event_type,
    payload,
    vector_clock,
    status,
    max_attempts
) VALUES (
    $1, $2, $3, $4, $5, 'pending', $6
)
RETURNING *;

-- name: GetPendingOutboxEvents :many
SELECT *
FROM sync_outbox
WHERE status = 'pending'
  AND (scheduled_for IS NULL OR scheduled_for <= NOW())
ORDER BY created_at ASC
LIMIT $1
FOR UPDATE SKIP LOCKED;

-- name: GetOutboxEventByID :one
SELECT * FROM sync_outbox
WHERE id = $1;

-- name: MarkOutboxEventProcessing :exec
UPDATE sync_outbox
SET status = 'processing',
    updated_at = NOW()
WHERE id = $1;

-- name: MarkOutboxEventCompleted :exec
UPDATE sync_outbox
SET status = 'completed',
    processed_at = NOW(),
    updated_at = NOW()
WHERE id = $1;

-- name: MarkOutboxEventFailed :exec
UPDATE sync_outbox
SET status = 'failed',
    attempts = attempts + 1,
    last_error = COALESCE($2, last_error),
    scheduled_for = COALESCE($3, scheduled_for),
    updated_at = NOW()
WHERE id = $1;

-- name: GetOutboxEventStats :one
SELECT
    COUNT(*) FILTER (WHERE status = 'pending') as pending_count,
    COUNT(*) FILTER (WHERE status = 'processing') as processing_count,
    COUNT(*) FILTER (WHERE status = 'completed') as completed_count,
    COUNT(*) FILTER (WHERE status = 'failed') as failed_count,
    COUNT(*) FILTER (WHERE attempts >= max_attempts) as dlq_ready_count
FROM sync_outbox;

-- name: ListFailedOutboxEvents :many
SELECT *
FROM sync_outbox
WHERE status = 'failed'
  AND attempts < max_attempts
  AND (scheduled_for IS NULL OR scheduled_for <= NOW())
ORDER BY created_at ASC
LIMIT $1;

-- name: ListDLQReadyEvents :many
SELECT *
FROM sync_outbox
WHERE attempts >= max_attempts
  AND status = 'failed'
ORDER BY created_at ASC
LIMIT $1;

-- name: MoveEventToDLQ :one
WITH moved AS (
    SELECT
        sync_outbox.id,
        sync_outbox.aggregate_type,
        sync_outbox.aggregate_id,
        sync_outbox.event_type,
        sync_outbox.payload,
        sync_outbox.vector_clock,
        sync_outbox.attempts,
        COALESCE(sync_outbox.last_error, '') as last_error,
        COALESCE($2, '') as failure_reason,
        sync_outbox.created_at as original_created_at
    FROM sync_outbox
    WHERE sync_outbox.id = $1
)
INSERT INTO sync_dlq (
    id,
    aggregate_type,
    aggregate_id,
    event_type,
    payload,
    vector_clock,
    attempts,
    last_error,
    failure_reason,
    original_created_at
)
SELECT * FROM moved
RETURNING *;

-- name: DeleteOutboxEvent :exec
DELETE FROM sync_outbox
WHERE id = $1;

-- name: CleanupOldCompletedEvents :exec
DELETE FROM sync_outbox
WHERE status = 'completed'
  AND processed_at < NOW() - INTERVAL '7 days';

-- name: ResetStuckProcessingEvents :exec
UPDATE sync_outbox
SET status = 'pending',
    updated_at = NOW()
WHERE status = 'processing'
  AND updated_at < NOW() - INTERVAL '5 minutes';

-- ============================================================================
-- DEAD LETTER QUEUE QUERIES
-- ============================================================================

-- name: ListDLQEvents :many
SELECT *
FROM sync_dlq
WHERE resolved = false
ORDER BY moved_to_dlq_at DESC
LIMIT $1;

-- name: ResolveDLQEvent :exec
UPDATE sync_dlq
SET resolved = true,
    resolved_at = NOW(),
    resolution_notes = $2
WHERE id = $1;

-- name: GetDLQEventByID :one
SELECT * FROM sync_dlq
WHERE id = $1;

-- name: RetryDLQEvent :one
INSERT INTO sync_outbox (
    aggregate_type,
    aggregate_id,
    event_type,
    payload,
    vector_clock,
    status,
    max_attempts
)
SELECT
    sync_dlq.aggregate_type,
    sync_dlq.aggregate_id,
    sync_dlq.event_type,
    sync_dlq.payload,
    sync_dlq.vector_clock,
    'pending',
    5 -- Reset to default max attempts
FROM sync_dlq
WHERE sync_dlq.id = $1
RETURNING *;
