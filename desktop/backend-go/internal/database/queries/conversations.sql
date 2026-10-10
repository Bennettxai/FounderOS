-- name: ListConversations :many
SELECT c.*,
       COUNT(m.id) as message_count
FROM conversations c
LEFT JOIN messages m ON m.conversation_id = c.id
WHERE c.user_id = $1
GROUP BY c.id
ORDER BY c.updated_at DESC;

-- name: GetConversation :one
SELECT * FROM conversations
WHERE id = $1 AND user_id = $2;

-- name: GetConversationWithMessages :one
SELECT * FROM conversations
WHERE id = $1 AND user_id = $2;

-- name: CreateConversation :one
INSERT INTO conversations (id, user_id, title, context_id, created_at, updated_at)
VALUES (gen_random_uuid(), $1, $2, $3, NOW(), NOW())
RETURNING *;

-- name: UpdateConversation :one
UPDATE conversations
SET title = $2, context_id = $3, updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteConversation :exec
DELETE FROM conversations
WHERE id = $1 AND user_id = $2;

-- name: ListMessages :many
SELECT * FROM messages
WHERE conversation_id = $1
ORDER BY created_at ASC;

-- name: CreateMessage :one
INSERT INTO messages (id, conversation_id, role, content, created_at, message_metadata)
VALUES (gen_random_uuid(), $1, $2, $3, NOW(), $4)
RETURNING *;

-- name: SearchConversations :many
SELECT c.*, COUNT(m.id) as message_count
FROM conversations c
LEFT JOIN messages m ON m.conversation_id = c.id
WHERE c.user_id = $1
  AND (c.title ILIKE '%' || $2 || '%' OR EXISTS (
    SELECT 1 FROM messages msg
    WHERE msg.conversation_id = c.id AND msg.content ILIKE '%' || $2 || '%'
  ))
GROUP BY c.id
ORDER BY c.updated_at DESC;

-- name: ListConversationsByContext :many
SELECT c.*,
       COUNT(m.id) as message_count
FROM conversations c
LEFT JOIN messages m ON m.conversation_id = c.id
WHERE c.user_id = $1 AND c.context_id = $2
GROUP BY c.id
ORDER BY c.updated_at DESC;

-- name: UpdateConversationContext :one
UPDATE conversations
SET context_id = $2, updated_at = NOW()
WHERE id = $1
RETURNING *;
