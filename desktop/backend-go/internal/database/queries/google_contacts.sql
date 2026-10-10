-- Google Contacts Queries
-- SQLC queries for Google Contacts integration

-- ============================================================================
-- Google Contacts
-- ============================================================================

-- name: UpsertGoogleContact :one
INSERT INTO google_contacts (
    user_id, resource_name, display_name, given_name, family_name, middle_name,
    emails, phone_numbers, addresses, organization, job_title, department,
    photo_url, contact_groups, metadata, created_time, modified_time, synced_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, NOW())
ON CONFLICT (user_id, resource_name) DO UPDATE SET
    display_name = EXCLUDED.display_name,
    given_name = EXCLUDED.given_name,
    family_name = EXCLUDED.family_name,
    middle_name = EXCLUDED.middle_name,
    emails = EXCLUDED.emails,
    phone_numbers = EXCLUDED.phone_numbers,
    addresses = EXCLUDED.addresses,
    organization = EXCLUDED.organization,
    job_title = EXCLUDED.job_title,
    department = EXCLUDED.department,
    photo_url = EXCLUDED.photo_url,
    contact_groups = EXCLUDED.contact_groups,
    metadata = EXCLUDED.metadata,
    modified_time = EXCLUDED.modified_time,
    synced_at = NOW(),
    updated_at = NOW()
RETURNING *;

-- name: GetGoogleContact :one
SELECT * FROM google_contacts
WHERE user_id = $1 AND resource_name = $2;

-- name: GetGoogleContactsByUser :many
SELECT * FROM google_contacts
WHERE user_id = $1
ORDER BY display_name NULLS LAST
LIMIT $2 OFFSET $3;

-- name: SearchGoogleContacts :many
SELECT * FROM google_contacts
WHERE user_id = $1
  AND (display_name ILIKE $2 OR organization ILIKE $2)
ORDER BY display_name NULLS LAST
LIMIT $3;

-- name: GetGoogleContactsByOrganization :many
SELECT * FROM google_contacts
WHERE user_id = $1 AND organization = $2
ORDER BY display_name NULLS LAST;

-- name: GetGoogleContactsCount :one
SELECT COUNT(*) FROM google_contacts
WHERE user_id = $1;

-- name: DeleteGoogleContact :exec
DELETE FROM google_contacts
WHERE user_id = $1 AND resource_name = $2;

-- name: DeleteGoogleContactsByUser :exec
DELETE FROM google_contacts WHERE user_id = $1;
