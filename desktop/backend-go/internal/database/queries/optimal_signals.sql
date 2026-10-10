-- ===== SIGNALS CRUD =====

-- name: CreateOptimalSignal :one
INSERT INTO optimal_signals (
    workspace_id, node_id, created_by, title, slug, uri, path,
    content, mode, genre, signal_type, format, structure,
    sn_ratio, l0_abstract, l1_overview, primary_node, cross_ref_nodes,
    entities, valid_from, metadata
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13,
    $14, $15, $16, $17, $18, $19, $20, $21
) RETURNING *;

-- name: GetOptimalSignal :one
SELECT * FROM optimal_signals
WHERE id = $1 AND workspace_id = $2;

-- name: ListOptimalSignals :many
SELECT * FROM optimal_signals
WHERE workspace_id = $1
  AND (sqlc.narg(node_id)::uuid IS NULL OR node_id = sqlc.narg(node_id))
  AND (sqlc.narg(genre)::text IS NULL OR genre = sqlc.narg(genre))
  AND (sqlc.narg(mode)::text IS NULL OR mode = sqlc.narg(mode))
  AND valid_until IS NULL
ORDER BY created_at DESC
LIMIT sqlc.arg(limit_val)::int;

-- name: ListOptimalSignalsByNode :many
SELECT * FROM optimal_signals
WHERE workspace_id = $1 AND node_id = $2
  AND valid_until IS NULL
ORDER BY created_at DESC;

-- name: UpdateOptimalSignal :one
UPDATE optimal_signals
SET title = $3, content = $4, l0_abstract = $5, l1_overview = $6,
    mode = $7, genre = $8, signal_type = $9, sn_ratio = $10,
    entities = $11, metadata = $12, updated_at = NOW()
WHERE id = $1 AND workspace_id = $2
RETURNING *;

-- name: SupersedeOptimalSignal :exec
-- Marks a signal as superseded; a newer version should be created separately
UPDATE optimal_signals
SET valid_until = NOW(), updated_at = NOW()
WHERE id = $1 AND workspace_id = $2;

-- name: DeleteOptimalSignal :exec
DELETE FROM optimal_signals
WHERE id = $1 AND workspace_id = $2;

-- ===== FULL-TEXT SEARCH =====

-- name: SearchOptimalSignals :many
SELECT id, workspace_id, node_id, title, slug, path,
       l0_abstract, l1_overview, genre, mode, signal_type, sn_ratio,
       primary_node, entities, created_at,
       ts_rank_cd(search_vector, websearch_to_tsquery('english', $2)) AS rank
FROM optimal_signals
WHERE workspace_id = $1
  AND search_vector @@ websearch_to_tsquery('english', $2)
  AND valid_until IS NULL
ORDER BY rank DESC
LIMIT $3;

-- name: SearchOptimalSignalsByNode :many
SELECT id, workspace_id, node_id, title, slug, path,
       l0_abstract, l1_overview, genre, mode, signal_type, sn_ratio,
       primary_node, entities, created_at,
       ts_rank_cd(search_vector, websearch_to_tsquery('english', $3)) AS rank
FROM optimal_signals
WHERE workspace_id = $1
  AND node_id = $2
  AND search_vector @@ websearch_to_tsquery('english', $3)
  AND valid_until IS NULL
ORDER BY rank DESC
LIMIT $4;

-- ===== ENTITIES =====

-- name: UpsertOptimalEntity :one
INSERT INTO optimal_entities (workspace_id, name, canonical_name, type, mention_count)
VALUES ($1, $2, LOWER($2), $3, 1)
ON CONFLICT (workspace_id, canonical_name)
DO UPDATE SET mention_count = optimal_entities.mention_count + 1,
              last_seen = NOW()
RETURNING *;

-- name: ListOptimalEntities :many
SELECT * FROM optimal_entities
WHERE workspace_id = $1
  AND (sqlc.narg(type)::text IS NULL OR type = sqlc.narg(type))
ORDER BY mention_count DESC
LIMIT $2;

-- name: GetOptimalEntity :one
SELECT * FROM optimal_entities
WHERE id = $1 AND workspace_id = $2;

-- name: DeleteOptimalEntity :exec
DELETE FROM optimal_entities
WHERE id = $1 AND workspace_id = $2;

-- ===== EDGES (KNOWLEDGE GRAPH) =====

-- name: CreateOptimalEdge :one
INSERT INTO optimal_edges (workspace_id, source_id, source_type, target_id, target_type, relation, weight, reason)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
ON CONFLICT (workspace_id, source_id, target_id, relation) DO UPDATE
SET weight = EXCLUDED.weight, valid_from = NOW()
RETURNING *;

-- name: ListOptimalEdges :many
SELECT * FROM optimal_edges
WHERE workspace_id = $1
  AND valid_until IS NULL
ORDER BY created_at DESC;

-- name: GetOptimalEdgesByEntity :many
SELECT * FROM optimal_edges
WHERE workspace_id = $1
  AND (source_id = $2 OR target_id = $2)
  AND valid_until IS NULL;

-- name: GetOptimalGraphHubs :many
-- Returns the most connected entities by total edge count
SELECT e.id, e.name, e.type, e.mention_count,
       COUNT(DISTINCT ed.id) AS connection_count
FROM optimal_entities e
LEFT JOIN optimal_edges ed ON (ed.source_id = e.id OR ed.target_id = e.id)
    AND ed.workspace_id = e.workspace_id
    AND ed.valid_until IS NULL
WHERE e.workspace_id = $1
GROUP BY e.id, e.name, e.type, e.mention_count
ORDER BY connection_count DESC
LIMIT $2;

-- name: DeleteOptimalEdge :exec
DELETE FROM optimal_edges
WHERE id = $1 AND workspace_id = $2;

-- ===== OBSERVATIONS (LEARNING LOOP) =====

-- name: CreateOptimalObservation :one
INSERT INTO optimal_observations (workspace_id, created_by, category, content, confidence, source)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: ListOptimalObservations :many
SELECT * FROM optimal_observations
WHERE workspace_id = $1
  AND (sqlc.narg(category)::text IS NULL OR category = sqlc.narg(category))
ORDER BY created_at DESC;

-- name: GetOptimalEscalations :many
-- Returns categories with 3 or more observations; signals patterns requiring escalation
SELECT category, COUNT(*) AS observation_count
FROM optimal_observations
WHERE workspace_id = $1
GROUP BY category
HAVING COUNT(*) >= 3
ORDER BY observation_count DESC;

-- ===== TOPOLOGY =====

-- name: GetOptimalTopology :one
SELECT * FROM optimal_topology
WHERE workspace_id = $1;

-- name: UpsertOptimalTopology :one
INSERT INTO optimal_topology (workspace_id, config, genre_skeletons, genre_half_lives)
VALUES ($1, $2, $3, $4)
ON CONFLICT (workspace_id)
DO UPDATE SET config = $2, genre_skeletons = $3, genre_half_lives = $4, updated_at = NOW()
RETURNING *;

-- ===== DECISION LOG =====

-- name: CreateOptimalDecisionLog :one
INSERT INTO optimal_decision_log (workspace_id, actor, action, target_type, target_id, what_changed, why, alternatives_considered, context)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING *;

-- name: ListOptimalDecisionLog :many
SELECT * FROM optimal_decision_log
WHERE workspace_id = $1
  AND (sqlc.narg(target_type)::text IS NULL OR target_type = sqlc.narg(target_type))
ORDER BY created_at DESC
LIMIT $2;

-- ===== DASHBOARD AGGREGATES =====

-- name: GetOptimalDashboardStats :one
SELECT
    (SELECT COUNT(*) FROM optimal_signals WHERE workspace_id = v.wid AND valid_until IS NULL) AS signal_count,
    (SELECT COUNT(*) FROM optimal_entities WHERE workspace_id = v.wid) AS entity_count,
    (SELECT COUNT(*) FROM optimal_edges WHERE workspace_id = v.wid AND valid_until IS NULL) AS edge_count,
    (SELECT COUNT(*) FROM optimal_observations WHERE workspace_id = v.wid) AS observation_count,
    (SELECT COUNT(*) FROM optimal_sessions WHERE workspace_id = v.wid) AS session_count
FROM (VALUES ($1::uuid)) AS v(wid);

-- name: GetOptimalSignalsByGenre :many
SELECT genre, COUNT(*) AS count
FROM optimal_signals
WHERE workspace_id = $1 AND valid_until IS NULL
GROUP BY genre
ORDER BY count DESC;

-- name: GetOptimalRecentSignals :many
SELECT id, title, genre, mode, signal_type, sn_ratio, l0_abstract, primary_node, created_at
FROM optimal_signals
WHERE workspace_id = $1 AND valid_until IS NULL
ORDER BY created_at DESC
LIMIT $2;
