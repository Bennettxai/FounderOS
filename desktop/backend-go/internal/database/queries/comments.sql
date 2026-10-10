-- name: CreateComment :one
INSERT INTO comments (
    user_id, entity_type, entity_id, content, parent_id
) VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetComment :one
SELECT * FROM comments WHERE id = $1 AND is_deleted = FALSE;

-- name: ListCommentsByEntity :many
SELECT * FROM comments
WHERE entity_type = $1 AND entity_id = $2 AND is_deleted = FALSE
ORDER BY created_at ASC;

-- name: ListCommentsByEntityPaginated :many
SELECT * FROM comments
WHERE entity_type = $1 AND entity_id = $2 AND is_deleted = FALSE
ORDER BY created_at DESC
LIMIT $3 OFFSET $4;

-- name: CountCommentsByEntity :one
SELECT COUNT(*) FROM comments
WHERE entity_type = $1 AND entity_id = $2 AND is_deleted = FALSE;

-- name: UpdateCommentContent :one
UPDATE comments
SET content = $2, is_edited = TRUE, edited_at = NOW(), updated_at = NOW()
WHERE id = $1 AND is_deleted = FALSE
RETURNING *;

-- name: SoftDeleteComment :exec
UPDATE comments
SET is_deleted = TRUE, deleted_at = NOW(), updated_at = NOW()
WHERE id = $1;

-- name: DeleteComment :exec
DELETE FROM comments WHERE id = $1;

-- name: GetCommentReplies :many
SELECT * FROM comments
WHERE parent_id = $1 AND is_deleted = FALSE
ORDER BY created_at ASC;

-- name: GetCommentCountsByEntities :many
SELECT entity_type, entity_id, COUNT(*) as count
FROM comments
WHERE entity_type = $1 AND entity_id = ANY($2::uuid[]) AND is_deleted = FALSE
GROUP BY entity_type, entity_id;

-- ===== MENTIONS =====

-- name: CreateEntityMention :one
INSERT INTO entity_mentions (
    source_type, source_id, mentioned_user_id, mention_text,
    position_in_text, entity_type, entity_id, mentioned_by
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: GetMentionsBySource :many
SELECT * FROM entity_mentions
WHERE source_type = $1 AND source_id = $2
ORDER BY position_in_text ASC;

-- name: GetMentionsForUser :many
SELECT * FROM entity_mentions
WHERE mentioned_user_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: GetUnnotifiedMentionsForUser :many
SELECT * FROM entity_mentions
WHERE mentioned_user_id = $1 AND notified = FALSE
ORDER BY created_at DESC;

-- name: MarkMentionNotified :exec
UPDATE entity_mentions
SET notified = TRUE, notified_at = NOW()
WHERE id = $1;

-- name: MarkMentionsNotifiedBulk :exec
UPDATE entity_mentions
SET notified = TRUE, notified_at = NOW()
WHERE id = ANY($1::uuid[]);

-- name: DeleteMentionsBySource :exec
DELETE FROM entity_mentions
WHERE source_type = $1 AND source_id = $2;

-- ===== REACTIONS =====

-- name: AddCommentReaction :one
INSERT INTO comment_reactions (comment_id, user_id, emoji)
VALUES ($1, $2, $3)
ON CONFLICT (comment_id, user_id, emoji) DO NOTHING
RETURNING *;

-- name: RemoveCommentReaction :exec
DELETE FROM comment_reactions
WHERE comment_id = $1 AND user_id = $2 AND emoji = $3;

-- name: GetCommentReactions :many
SELECT emoji, COUNT(*) as count, ARRAY_AGG(user_id) as user_ids
FROM comment_reactions
WHERE comment_id = $1
GROUP BY emoji;

-- name: GetUserReactionsOnComment :many
SELECT emoji FROM comment_reactions
WHERE comment_id = $1 AND user_id = $2;

-- ===== AGGREGATES FOR TASK LISTS =====

-- name: GetTaskCommentCounts :many
SELECT entity_id as task_id, COUNT(*) as comment_count
FROM comments
WHERE entity_type = 'task' AND entity_id = ANY($1::uuid[]) AND is_deleted = FALSE
GROUP BY entity_id;
-- ===== COMMENTS WITH AUTHOR INFO =====

-- name: GetCommentWithAuthor :one
SELECT 
    c.id, c.user_id, c.entity_type, c.entity_id, c.content, c.parent_id,
    c.is_edited, c.edited_at, c.is_deleted, c.created_at, c.updated_at,
    u.name as author_name, u.email as author_email, u.image as avatar_url
FROM comments c
JOIN "user" u ON c.user_id = u.id
WHERE c.id = $1 AND c.is_deleted = FALSE;

-- name: ListCommentsWithAuthor :many
SELECT 
    c.id, c.user_id, c.entity_type, c.entity_id, c.content, c.parent_id,
    c.is_edited, c.edited_at, c.is_deleted, c.created_at, c.updated_at,
    u.name as author_name, u.email as author_email, u.image as avatar_url
FROM comments c
JOIN "user" u ON c.user_id = u.id
WHERE c.entity_type = $1 AND c.entity_id = $2 AND c.is_deleted = FALSE AND c.parent_id IS NULL
ORDER BY c.created_at ASC;

-- name: ListRepliesWithAuthor :many
SELECT 
    c.id, c.user_id, c.entity_type, c.entity_id, c.content, c.parent_id,
    c.is_edited, c.edited_at, c.is_deleted, c.created_at, c.updated_at,
    u.name as author_name, u.email as author_email, u.image as avatar_url
FROM comments c
JOIN "user" u ON c.user_id = u.id
WHERE c.parent_id = $1 AND c.is_deleted = FALSE
ORDER BY c.created_at ASC;

-- name: GetUserByID :one
SELECT id, name, email, image FROM "user" WHERE id = $1;