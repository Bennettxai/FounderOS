-- ============================================================================
-- TAGS QUERIES
-- ============================================================================

-- name: ListTags :many
SELECT * FROM tags
WHERE user_id = $1
ORDER BY usage_count DESC, name ASC;

-- name: ListTagsByGroup :many
SELECT * FROM tags
WHERE user_id = $1 AND group_name = $2
ORDER BY name ASC;

-- name: GetTag :one
SELECT * FROM tags
WHERE id = $1;

-- name: GetTagBySlug :one
SELECT * FROM tags
WHERE user_id = $1 AND slug = $2;

-- name: CreateTag :one
INSERT INTO tags (user_id, name, slug, description, color, icon, parent_id, group_name, allowed_entity_types)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING *;

-- name: UpdateTag :one
UPDATE tags
SET name = $2, slug = $3, description = $4, color = $5, icon = $6, group_name = $7, updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteTag :exec
DELETE FROM tags
WHERE id = $1 AND user_id = $2;

-- name: SearchTags :many
SELECT * FROM tags
WHERE user_id = $1
  AND name ILIKE '%' || $2 || '%'
ORDER BY usage_count DESC
LIMIT sqlc.arg(limit_val)::int;

-- name: ListMostUsedTags :many
SELECT * FROM tags
WHERE user_id = $1
ORDER BY usage_count DESC
LIMIT sqlc.arg(limit_val)::int;

-- name: ListTagGroups :many
SELECT DISTINCT group_name FROM tags
WHERE user_id = $1 AND group_name IS NOT NULL
ORDER BY group_name;

-- name: ListChildTags :many
SELECT * FROM tags
WHERE parent_id = $1
ORDER BY name ASC;

-- ============================================================================
-- TAG ASSIGNMENTS QUERIES
-- ============================================================================

-- name: ListEntityTags :many
SELECT t.* FROM tags t
JOIN tag_assignments ta ON t.id = ta.tag_id
WHERE ta.entity_type = $1 AND ta.entity_id = $2
ORDER BY t.name ASC;

-- name: ListTagEntities :many
SELECT ta.entity_type, ta.entity_id, ta.created_at
FROM tag_assignments ta
WHERE ta.tag_id = $1
ORDER BY ta.created_at DESC
LIMIT sqlc.arg(limit_val)::int;

-- name: AssignTag :one
INSERT INTO tag_assignments (tag_id, entity_type, entity_id, assigned_by)
VALUES ($1, $2, $3, $4)
ON CONFLICT (tag_id, entity_type, entity_id) DO NOTHING
RETURNING *;

-- name: UnassignTag :exec
DELETE FROM tag_assignments
WHERE tag_id = $1 AND entity_type = $2 AND entity_id = $3;

-- name: UnassignAllEntityTags :exec
DELETE FROM tag_assignments
WHERE entity_type = $1 AND entity_id = $2;

-- name: GetTagAssignment :one
SELECT * FROM tag_assignments
WHERE tag_id = $1 AND entity_type = $2 AND entity_id = $3;

-- name: CountTagUsage :one
SELECT COUNT(*) as count FROM tag_assignments
WHERE tag_id = $1;

-- name: ListEntitiesWithAllTags :many
SELECT entity_type, entity_id
FROM tag_assignments
WHERE tag_id = ANY($1::uuid[])
GROUP BY entity_type, entity_id
HAVING COUNT(DISTINCT tag_id) = array_length($1::uuid[], 1);

-- name: ListEntitiesWithAnyTag :many
SELECT DISTINCT entity_type, entity_id
FROM tag_assignments
WHERE tag_id = ANY($1::uuid[]);
