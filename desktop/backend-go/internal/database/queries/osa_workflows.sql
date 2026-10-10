-- name: CreateWorkflow :one
INSERT INTO osa_workflows (
    user_id,
    workspace_id,
    module_instance_id,
    osa_workflow_id,
    workflow_type,
    title,
    description,
    user_prompt,
    status,
    tags
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10
) RETURNING *;

-- name: GetWorkflow :one
SELECT * FROM osa_workflows
WHERE id = $1;

-- name: GetWorkflowByOSAId :one
SELECT * FROM osa_workflows
WHERE osa_workflow_id = $1
LIMIT 1;

-- name: ListWorkflowsByUser :many
SELECT * FROM osa_workflows
WHERE user_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: ListWorkflowsByWorkspace :many
SELECT * FROM osa_workflows
WHERE workspace_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: ListWorkflowsByApp :many
SELECT * FROM osa_workflows
WHERE module_instance_id = $1
ORDER BY created_at DESC;

-- name: ListWorkflowsByStatus :many
SELECT * FROM osa_workflows
WHERE status = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: ListActiveWorkflows :many
SELECT * FROM osa_workflows
WHERE status IN ('pending', 'planning', 'generating', 'testing')
ORDER BY created_at ASC;

-- name: UpdateWorkflowStatus :one
UPDATE osa_workflows
SET
    status = $2,
    progress_percent = COALESCE($3, progress_percent),
    current_phase = COALESCE($4, current_phase)
WHERE id = $1
RETURNING *;

-- name: UpdateWorkflowProgress :one
UPDATE osa_workflows
SET
    progress_percent = $2,
    current_phase = $3,
    files_generated = COALESCE($4, files_generated)
WHERE id = $1
RETURNING *;

-- name: UpdateWorkflowResults :one
UPDATE osa_workflows
SET
    files_generated = $2,
    tests_passed = $3,
    tests_failed = $4,
    build_successful = $5,
    status = $6
WHERE id = $1
RETURNING *;

-- name: UpdateWorkflowError :one
UPDATE osa_workflows
SET
    status = 'failed',
    error_message = $2,
    error_details = $3,
    retry_count = retry_count + 1
WHERE id = $1
RETURNING *;

-- name: UpdateWorkflowMetrics :one
UPDATE osa_workflows
SET
    duration_seconds = $2,
    tokens_used = $3,
    estimated_cost_usd = $4
WHERE id = $1
RETURNING *;

-- name: CancelWorkflow :one
UPDATE osa_workflows
SET
    status = 'cancelled',
    cancelled_at = NOW()
WHERE id = $1 AND status NOT IN ('completed', 'failed', 'cancelled')
RETURNING *;

-- name: DeleteWorkflow :exec
DELETE FROM osa_workflows
WHERE id = $1;

-- name: CountWorkflowsByUser :one
SELECT COUNT(*) FROM osa_workflows
WHERE user_id = $1;

-- name: CountWorkflowsByStatus :one
SELECT COUNT(*) FROM osa_workflows
WHERE status = $1;

-- name: GetWorkflowStats :one
SELECT
    COUNT(*) as total_workflows,
    COUNT(*) FILTER (WHERE status = 'completed') as completed_count,
    COUNT(*) FILTER (WHERE status = 'failed') as failed_count,
    COUNT(*) FILTER (WHERE status IN ('pending', 'planning', 'generating', 'testing')) as active_count,
    AVG(duration_seconds) FILTER (WHERE status = 'completed') as avg_duration_seconds,
    SUM(files_generated) as total_files_generated,
    SUM(tokens_used) as total_tokens_used,
    SUM(estimated_cost_usd) as total_estimated_cost
FROM osa_workflows
WHERE user_id = $1;

-- name: SearchWorkflows :many
SELECT * FROM osa_workflows
WHERE
    (user_id = $1) AND
    (
        title ILIKE '%' || $2 || '%' OR
        description ILIKE '%' || $2 || '%' OR
        user_prompt ILIKE '%' || $2 || '%' OR
        $2 = ANY(tags)
    )
ORDER BY created_at DESC
LIMIT $3 OFFSET $4;
