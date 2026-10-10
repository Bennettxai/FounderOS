-- name: ListClients :many
SELECT * FROM clients
WHERE user_id = $1
  AND (sqlc.narg(status)::clientstatus IS NULL OR status = sqlc.narg(status))
  AND (sqlc.narg(client_type)::clienttype IS NULL OR type = sqlc.narg(client_type))
  AND (sqlc.narg(search)::text IS NULL OR name ILIKE '%' || sqlc.narg(search) || '%')
ORDER BY updated_at DESC;

-- name: GetClient :one
SELECT * FROM clients
WHERE id = $1 AND user_id = $2;

-- name: CreateClient :one
INSERT INTO clients (user_id, name, type, email, phone, website, industry, company_size, address, city, state, zip_code, country, status, source, assigned_to, tags, custom_fields, notes)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19)
RETURNING *;

-- name: UpdateClient :one
UPDATE clients
SET name = $2, type = $3, email = $4, phone = $5, website = $6, industry = $7, company_size = $8,
    address = $9, city = $10, state = $11, zip_code = $12, country = $13, status = $14, source = $15,
    assigned_to = $16, tags = $17, custom_fields = $18, notes = $19, updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: UpdateClientStatus :one
UPDATE clients
SET status = $2, updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteClient :exec
DELETE FROM clients
WHERE id = $1 AND user_id = $2;

-- name: ListClientContacts :many
SELECT * FROM client_contacts
WHERE client_id = $1
ORDER BY is_primary DESC, name ASC;

-- name: CreateClientContact :one
INSERT INTO client_contacts (client_id, name, email, phone, role, is_primary, notes)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: UpdateClientContact :one
UPDATE client_contacts
SET name = $2, email = $3, phone = $4, role = $5, is_primary = $6, notes = $7, updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteClientContact :exec
DELETE FROM client_contacts
WHERE id = $1 AND client_id = $2;

-- name: ListClientInteractions :many
SELECT * FROM client_interactions
WHERE client_id = $1
ORDER BY occurred_at DESC;

-- name: CreateClientInteraction :one
INSERT INTO client_interactions (client_id, contact_id, type, subject, description, outcome, occurred_at)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: ListClientDeals :many
SELECT d.*, ps.name as stage_name
FROM deals d
JOIN pipeline_stages ps ON d.stage_id = ps.id
WHERE d.client_id = $1
ORDER BY d.created_at DESC;

-- name: CreateClientDeal :one
INSERT INTO deals (
    client_id, user_id, pipeline_id, stage_id,
    name, amount, probability, expected_close_date,
    description, status
)
SELECT
    $1,                                           -- client_id
    c.user_id,                                    -- user_id (from client's owner)
    p.id,                                         -- pipeline_id (default pipeline)
    ps.id,                                        -- stage_id (matched by stage name)
    $2,                                           -- name
    $3,                                           -- amount (was value)
    $5,                                           -- probability
    $6,                                           -- expected_close_date
    $7,                                           -- description (was notes)
    'open'                                        -- status
FROM clients c
JOIN pipelines p ON p.user_id = c.user_id AND p.is_default = TRUE
JOIN pipeline_stages ps ON ps.pipeline_id = p.id AND ps.name = $4  -- stage name lookup
WHERE c.id = $1
RETURNING *;

-- name: UpdateClientDeal :one
UPDATE deals
SET name = $2, amount = $3, probability = $5,
    expected_close_date = $6, description = $7,
    status = $4, updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: UpdateDealStage :one
UPDATE deals
SET stage_id = (
        SELECT ps.id FROM pipeline_stages ps
        WHERE ps.pipeline_id = deals.pipeline_id AND ps.name = $2
        LIMIT 1
    ),
    status = CASE
        WHEN $2 = 'Won' THEN 'won'
        WHEN $2 = 'Lost' THEN 'lost'
        ELSE status
    END,
    actual_close_date = CASE
        WHEN $2 IN ('Won', 'Lost') THEN NOW()
        ELSE NULL
    END,
    updated_at = NOW()
WHERE deals.id = $1
RETURNING *;

-- name: ListDeals :many
SELECT d.*, c.name as client_name, ps.name as stage_name
FROM deals d
JOIN clients c ON c.id = d.client_id
JOIN pipeline_stages ps ON d.stage_id = ps.id
WHERE c.user_id = $1
  AND (sqlc.narg(status)::varchar IS NULL OR d.status = sqlc.narg(status))
ORDER BY d.updated_at DESC;
