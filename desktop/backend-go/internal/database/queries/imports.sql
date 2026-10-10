-- Import Jobs Queries
-- SQLC queries for data import operations

-- ============================================================================
-- Import Jobs
-- ============================================================================

-- name: CreateImportJob :one
INSERT INTO import_jobs (
    user_id,
    source_type,
    source_provider,
    original_filename,
    file_size_bytes,
    content_type,
    target_module,
    target_entity,
    field_mapping,
    transform_rules,
    import_options
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
RETURNING *;

-- name: GetImportJob :one
SELECT * FROM import_jobs
WHERE id = $1 AND user_id = $2;

-- name: GetImportJobsByUser :many
SELECT * FROM import_jobs
WHERE user_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: GetActiveImportJobs :many
SELECT * FROM import_jobs
WHERE user_id = $1
  AND status IN ('pending', 'validating', 'mapping', 'processing')
ORDER BY created_at DESC;

-- name: UpdateImportJobStatus :exec
UPDATE import_jobs
SET
    status = $3,
    progress_percent = $4,
    started_at = CASE WHEN $3 = 'processing' AND started_at IS NULL THEN NOW() ELSE started_at END,
    completed_at = CASE WHEN $3 IN ('completed', 'failed', 'cancelled') THEN NOW() ELSE completed_at END,
    updated_at = NOW()
WHERE id = $1 AND user_id = $2;

-- name: UpdateImportJobProgress :exec
UPDATE import_jobs
SET
    progress_percent = $3,
    processed_records = $4,
    imported_records = $5,
    skipped_records = $6,
    failed_records = $7
WHERE id = $1 AND user_id = $2;

-- name: UpdateImportJobTotalRecords :exec
UPDATE import_jobs
SET total_records = $3
WHERE id = $1 AND user_id = $2;

-- name: SetImportJobMapping :exec
UPDATE import_jobs
SET
    field_mapping = $3,
    transform_rules = $4,
    status = 'mapping'
WHERE id = $1 AND user_id = $2;

-- name: CompleteImportJob :exec
UPDATE import_jobs
SET
    status = 'completed',
    progress_percent = 100,
    completed_at = NOW(),
    result_summary = $3
WHERE id = $1 AND user_id = $2;

-- name: FailImportJob :exec
UPDATE import_jobs
SET
    status = 'failed',
    completed_at = NOW(),
    error_message = $3,
    error_details = $4
WHERE id = $1 AND user_id = $2;

-- name: AppendImportJobError :exec
UPDATE import_jobs
SET error_log = error_log || $3::jsonb
WHERE id = $1 AND user_id = $2;

-- name: DeleteImportJob :exec
DELETE FROM import_jobs
WHERE id = $1 AND user_id = $2;

-- ============================================================================
-- Imported Records (Deduplication)
-- ============================================================================

-- name: CreateImportedRecord :one
INSERT INTO imported_records (
    user_id,
    import_job_id,
    source_type,
    source_provider,
    external_id,
    target_module,
    target_entity,
    target_record_id,
    external_data_hash
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
ON CONFLICT (user_id, source_type, external_id) DO UPDATE SET
    import_job_id = EXCLUDED.import_job_id,
    target_record_id = EXCLUDED.target_record_id,
    external_data_hash = EXCLUDED.external_data_hash,
    last_synced_at = NOW(),
    updated_at = NOW()
RETURNING *;

-- name: GetImportedRecord :one
SELECT * FROM imported_records
WHERE user_id = $1 AND source_type = $2 AND external_id = $3;

-- name: GetImportedRecordByTarget :one
SELECT * FROM imported_records
WHERE user_id = $1 AND target_module = $2 AND target_record_id = $3;

-- name: CheckExternalRecordExists :one
SELECT EXISTS(
    SELECT 1 FROM imported_records
    WHERE user_id = $1 AND source_type = $2 AND external_id = $3
) as exists;

-- name: GetImportedRecordsForJob :many
SELECT * FROM imported_records
WHERE import_job_id = $1
ORDER BY created_at ASC;

-- name: DeleteImportedRecordsForJob :exec
DELETE FROM imported_records
WHERE import_job_id = $1;

-- ============================================================================
-- Import Mapping Templates
-- ============================================================================

-- name: GetMappingTemplate :one
SELECT * FROM import_mapping_templates
WHERE source_type = $1 AND target_module = $2 AND template_name = $3;

-- name: GetMappingTemplatesForSource :many
SELECT * FROM import_mapping_templates
WHERE source_type = $1
ORDER BY is_system_template DESC, template_name;

-- name: GetUserMappingTemplates :many
SELECT * FROM import_mapping_templates
WHERE created_by = $1 OR is_system_template = true
ORDER BY source_type, is_system_template DESC, template_name;

-- name: CreateMappingTemplate :one
INSERT INTO import_mapping_templates (
    source_type,
    target_module,
    template_name,
    field_mappings,
    transform_rules,
    default_values,
    description,
    is_system_template,
    created_by
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING *;

-- name: UpdateMappingTemplate :exec
UPDATE import_mapping_templates
SET
    field_mappings = $4,
    transform_rules = $5,
    default_values = $6,
    description = $7,
    updated_at = NOW()
WHERE source_type = $1 AND target_module = $2 AND template_name = $3;

-- name: DeleteMappingTemplate :exec
DELETE FROM import_mapping_templates
WHERE source_type = $1 AND target_module = $2 AND template_name = $3
  AND is_system_template = false;

-- ============================================================================
-- Imported Conversations
-- ============================================================================

-- name: CreateImportedConversation :one
INSERT INTO imported_conversations (
    user_id,
    import_job_id,
    source_type,
    external_conversation_id,
    title,
    model,
    messages,
    message_count,
    original_created_at,
    original_updated_at,
    metadata,
    search_content,
    tags
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
RETURNING *;

-- name: GetImportedConversation :one
SELECT * FROM imported_conversations
WHERE id = $1 AND user_id = $2;

-- name: GetImportedConversationsByUser :many
SELECT * FROM imported_conversations
WHERE user_id = $1
ORDER BY original_created_at DESC NULLS LAST, created_at DESC
LIMIT $2 OFFSET $3;

-- name: GetImportedConversationsBySource :many
SELECT * FROM imported_conversations
WHERE user_id = $1 AND source_type = $2
ORDER BY original_created_at DESC NULLS LAST, created_at DESC
LIMIT $3 OFFSET $4;

-- name: GetImportedConversationsForJob :many
SELECT * FROM imported_conversations
WHERE import_job_id = $1
ORDER BY original_created_at ASC NULLS LAST;

-- name: SearchImportedConversations :many
SELECT * FROM imported_conversations
WHERE user_id = $1
  AND to_tsvector('english', search_content) @@ plainto_tsquery('english', $2)
ORDER BY ts_rank(to_tsvector('english', search_content), plainto_tsquery('english', $2)) DESC
LIMIT $3;

-- name: LinkConversationToContext :exec
UPDATE imported_conversations
SET linked_context_id = $3, updated_at = NOW()
WHERE id = $1 AND user_id = $2;

-- name: LinkConversationToProject :exec
UPDATE imported_conversations
SET linked_project_id = $3, updated_at = NOW()
WHERE id = $1 AND user_id = $2;

-- name: UpdateConversationTags :exec
UPDATE imported_conversations
SET tags = $3, updated_at = NOW()
WHERE id = $1 AND user_id = $2;

-- name: DeleteImportedConversation :exec
DELETE FROM imported_conversations
WHERE id = $1 AND user_id = $2;

-- name: DeleteImportedConversationsForJob :exec
DELETE FROM imported_conversations
WHERE import_job_id = $1;

-- name: CountImportedConversations :one
SELECT COUNT(*) FROM imported_conversations
WHERE user_id = $1;

-- name: CountImportedConversationsBySource :one
SELECT COUNT(*) FROM imported_conversations
WHERE user_id = $1 AND source_type = $2;
