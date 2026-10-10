-- name: ListArtifacts :many
SELECT * FROM artifacts
WHERE user_id = $1
  AND (sqlc.narg(conversation_id)::uuid IS NULL OR conversation_id = sqlc.narg(conversation_id))
  AND (sqlc.narg(project_id)::uuid IS NULL OR project_id = sqlc.narg(project_id))
  AND (sqlc.narg(context_id)::uuid IS NULL OR context_id = sqlc.narg(context_id))
  AND (sqlc.narg(artifact_type)::artifacttype IS NULL OR type = sqlc.narg(artifact_type))
ORDER BY updated_at DESC;

-- name: GetArtifact :one
SELECT * FROM artifacts
WHERE id = $1 AND user_id = $2;

-- name: CreateArtifact :one
INSERT INTO artifacts (id, user_id, conversation_id, message_id, project_id, context_id, title, type, language, content, summary, version, created_at, updated_at)
VALUES (gen_random_uuid(), $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, 1, NOW(), NOW())
RETURNING *;

-- name: UpdateArtifact :one
UPDATE artifacts
SET title = $2, content = $3, summary = $4, version = version + 1, updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: LinkArtifact :one
UPDATE artifacts
SET project_id = $2, context_id = $3, updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteArtifact :exec
DELETE FROM artifacts
WHERE id = $1 AND user_id = $2;

-- name: CreateArtifactVersion :one
INSERT INTO artifact_versions (artifact_id, version, content)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetArtifactVersions :many
SELECT * FROM artifact_versions
WHERE artifact_id = $1
ORDER BY version DESC;

-- name: GetArtifactVersion :one
SELECT * FROM artifact_versions
WHERE artifact_id = $1 AND version = $2;
