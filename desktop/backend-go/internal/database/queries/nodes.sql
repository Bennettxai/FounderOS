-- name: ListNodes :many
SELECT * FROM nodes
WHERE user_id = $1
  AND is_archived = FALSE
ORDER BY sort_order ASC, name ASC;

-- name: GetNodeTree :many
SELECT * FROM nodes
WHERE user_id = $1
  AND is_archived = FALSE
ORDER BY parent_id NULLS FIRST, sort_order ASC, name ASC;

-- name: GetActiveNode :one
SELECT * FROM nodes
WHERE user_id = $1 AND is_active = TRUE
LIMIT 1;

-- name: GetNode :one
SELECT * FROM nodes
WHERE id = $1 AND user_id = $2;

-- name: CreateNode :one
INSERT INTO nodes (user_id, parent_id, context_id, name, type, health, purpose, current_status, this_week_focus, decision_queue, delegation_ready, sort_order)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
RETURNING *;

-- name: UpdateNode :one
UPDATE nodes
SET name = $2, type = $3, health = $4, purpose = $5, current_status = $6, this_week_focus = $7, decision_queue = $8, delegation_ready = $9, context_id = $10, updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: ActivateNode :exec
UPDATE nodes
SET is_active = FALSE
WHERE user_id = $1;

-- name: SetNodeActive :one
UPDATE nodes
SET is_active = TRUE, updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeactivateNode :one
UPDATE nodes
SET is_active = FALSE, updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: ArchiveNode :one
UPDATE nodes
SET is_archived = TRUE, updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: UnarchiveNode :one
UPDATE nodes
SET is_archived = FALSE, updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteNode :exec
DELETE FROM nodes
WHERE id = $1 AND user_id = $2;

-- name: GetNodeChildren :many
SELECT * FROM nodes
WHERE parent_id = $1 AND user_id = $2 AND is_archived = FALSE
ORDER BY sort_order ASC, name ASC;

-- name: UpdateNodeSortOrder :exec
UPDATE nodes
SET sort_order = $2, updated_at = NOW()
WHERE id = $1;

-- ===== NODE LINKING QUERIES =====

-- name: GetNodeLinkedProjects :many
SELECT p.*, np.linked_at
FROM projects p
JOIN node_projects np ON p.id = np.project_id
WHERE np.node_id = $1
ORDER BY np.linked_at DESC;

-- name: GetNodeLinkedContexts :many
SELECT c.*, nc.linked_at
FROM contexts c
JOIN node_contexts nc ON c.id = nc.context_id
WHERE nc.node_id = $1
ORDER BY nc.linked_at DESC;

-- name: GetNodeLinkedConversations :many
SELECT conv.*, nconv.linked_at
FROM conversations conv
JOIN node_conversations nconv ON conv.id = nconv.conversation_id
WHERE nconv.node_id = $1
ORDER BY nconv.linked_at DESC;

-- name: LinkNodeProject :exec
INSERT INTO node_projects (node_id, project_id, linked_by)
VALUES ($1, $2, $3)
ON CONFLICT (node_id, project_id) DO NOTHING;

-- name: UnlinkNodeProject :exec
DELETE FROM node_projects
WHERE node_id = $1 AND project_id = $2;

-- name: LinkNodeContext :exec
INSERT INTO node_contexts (node_id, context_id, linked_by)
VALUES ($1, $2, $3)
ON CONFLICT (node_id, context_id) DO NOTHING;

-- name: UnlinkNodeContext :exec
DELETE FROM node_contexts
WHERE node_id = $1 AND context_id = $2;

-- name: LinkNodeConversation :exec
INSERT INTO node_conversations (node_id, conversation_id, linked_by)
VALUES ($1, $2, $3)
ON CONFLICT (node_id, conversation_id) DO NOTHING;

-- name: UnlinkNodeConversation :exec
DELETE FROM node_conversations
WHERE node_id = $1 AND conversation_id = $2;

-- name: GetNodeLinkCounts :one
SELECT
    (SELECT COUNT(*) FROM node_projects np WHERE np.node_id = $1) as linked_projects_count,
    (SELECT COUNT(*) FROM node_contexts nc WHERE nc.node_id = $1) as linked_contexts_count,
    (SELECT COUNT(*) FROM node_conversations nconv WHERE nconv.node_id = $1) as linked_conversations_count;
