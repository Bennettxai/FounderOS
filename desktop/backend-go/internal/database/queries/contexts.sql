-- name: ListContexts :many
SELECT * FROM contexts
WHERE user_id = $1
  AND (sqlc.narg(is_archived)::boolean IS NULL OR is_archived = sqlc.narg(is_archived))
  AND (sqlc.narg(context_type)::contexttype IS NULL OR type = sqlc.narg(context_type))
  AND (sqlc.narg(is_template)::boolean IS NULL OR is_template = sqlc.narg(is_template))
  AND (sqlc.narg(search)::text IS NULL OR name ILIKE '%' || sqlc.narg(search) || '%')
ORDER BY updated_at DESC;

-- name: ListContextsForWorkspace :many
-- BusinessOSSync: the user's own contexts PLUS any shared into the given
-- workspace (workspace_id stamped). Used when the request carries an authorized
-- X-Workspace-ID so team members see each other's published docs.
SELECT * FROM contexts
WHERE (user_id = $1 OR workspace_id = sqlc.narg(workspace_id)::uuid)
  AND (sqlc.narg(is_archived)::boolean IS NULL OR is_archived = sqlc.narg(is_archived))
  AND (sqlc.narg(context_type)::contexttype IS NULL OR type = sqlc.narg(context_type))
  AND (sqlc.narg(is_template)::boolean IS NULL OR is_template = sqlc.narg(is_template))
  AND (sqlc.narg(search)::text IS NULL OR name ILIKE '%' || sqlc.narg(search) || '%')
ORDER BY updated_at DESC;

-- name: GetContext :one
SELECT * FROM contexts
WHERE id = $1 AND user_id = $2;

-- name: GetPublicContext :one
SELECT * FROM contexts
WHERE share_id = $1 AND is_public = TRUE;

-- name: CreateContext :one
INSERT INTO contexts (id, user_id, name, type, content, structured_data, system_prompt_template, blocks, cover_image, icon, parent_id, is_template, property_schema, properties, client_id, created_at, updated_at)
VALUES (gen_random_uuid(), $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, NOW(), NOW())
RETURNING *;

-- name: UpdateContext :one
UPDATE contexts
SET name = $2, type = $3, content = $4, structured_data = $5, system_prompt_template = $6,
    cover_image = $7, icon = $8, parent_id = $9, is_template = $10, property_schema = $11,
    properties = $12, client_id = $13, updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: UpdateContextBlocks :one
UPDATE contexts
SET blocks = $2, word_count = $3, last_edited_at = NOW(), updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: ShareContext :one
UPDATE contexts
SET is_public = TRUE, share_id = $2
WHERE id = $1
RETURNING *;

-- name: UnshareContext :one
UPDATE contexts
SET is_public = FALSE, share_id = NULL
WHERE id = $1
RETURNING *;

-- name: ArchiveContext :one
UPDATE contexts
SET is_archived = TRUE, updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: UnarchiveContext :one
UPDATE contexts
SET is_archived = FALSE, updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteContext :exec
DELETE FROM contexts
WHERE id = $1 AND user_id = $2;

-- name: GetContextChildren :many
SELECT * FROM contexts
WHERE parent_id = $1 AND user_id = $2
ORDER BY name ASC;

-- name: SyncArtifactToContext :one
-- Syncs artifact content to a context (appends or replaces based on use case)
UPDATE contexts
SET content = CASE
      WHEN content IS NULL OR content = '' THEN $2
      ELSE content || E'\n\n---\n\n' || $2
    END,
    word_count = COALESCE(word_count, 0) + $3,
    last_edited_at = NOW(),
    updated_at = NOW()
WHERE id = $1
RETURNING *;
