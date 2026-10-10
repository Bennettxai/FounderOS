-- HubSpot CRM Queries
-- SQLC queries for HubSpot CRM integration

-- ============================================================================
-- HubSpot Contacts
-- ============================================================================

-- name: UpsertHubSpotContact :one
INSERT INTO hubspot_contacts (
    user_id, hubspot_id, email, first_name, last_name, phone,
    company, job_title, lifecycle_stage, lead_status, owner_id,
    properties, created_at_hubspot, updated_at_hubspot, synced_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, NOW())
ON CONFLICT (user_id, hubspot_id) DO UPDATE SET
    email = EXCLUDED.email,
    first_name = EXCLUDED.first_name,
    last_name = EXCLUDED.last_name,
    phone = EXCLUDED.phone,
    company = EXCLUDED.company,
    job_title = EXCLUDED.job_title,
    lifecycle_stage = EXCLUDED.lifecycle_stage,
    lead_status = EXCLUDED.lead_status,
    owner_id = EXCLUDED.owner_id,
    properties = EXCLUDED.properties,
    updated_at_hubspot = EXCLUDED.updated_at_hubspot,
    synced_at = NOW(),
    updated_at = NOW()
RETURNING *;

-- name: GetHubSpotContact :one
SELECT * FROM hubspot_contacts
WHERE user_id = $1 AND hubspot_id = $2;

-- name: GetHubSpotContactByEmail :one
SELECT * FROM hubspot_contacts
WHERE user_id = $1 AND email = $2;

-- name: GetHubSpotContactsByUser :many
SELECT * FROM hubspot_contacts
WHERE user_id = $1
ORDER BY updated_at_hubspot DESC NULLS LAST
LIMIT $2 OFFSET $3;

-- name: SearchHubSpotContacts :many
SELECT * FROM hubspot_contacts
WHERE user_id = $1
  AND (
    email ILIKE $2
    OR first_name ILIKE $2
    OR last_name ILIKE $2
    OR company ILIKE $2
  )
ORDER BY updated_at_hubspot DESC NULLS LAST
LIMIT $3;

-- name: GetHubSpotContactsByLifecycleStage :many
SELECT * FROM hubspot_contacts
WHERE user_id = $1 AND lifecycle_stage = $2
ORDER BY updated_at_hubspot DESC NULLS LAST
LIMIT $3 OFFSET $4;

-- name: GetHubSpotContactsByOwner :many
SELECT * FROM hubspot_contacts
WHERE user_id = $1 AND owner_id = $2
ORDER BY updated_at_hubspot DESC NULLS LAST
LIMIT $3 OFFSET $4;

-- name: CountHubSpotContacts :one
SELECT COUNT(*) FROM hubspot_contacts
WHERE user_id = $1;

-- name: CountHubSpotContactsByStage :many
SELECT lifecycle_stage, COUNT(*) as count
FROM hubspot_contacts
WHERE user_id = $1
GROUP BY lifecycle_stage
ORDER BY count DESC;

-- name: DeleteHubSpotContact :exec
DELETE FROM hubspot_contacts
WHERE user_id = $1 AND hubspot_id = $2;

-- name: DeleteHubSpotContactsByUser :exec
DELETE FROM hubspot_contacts WHERE user_id = $1;

-- ============================================================================
-- HubSpot Companies
-- ============================================================================

-- name: UpsertHubSpotCompany :one
INSERT INTO hubspot_companies (
    user_id, hubspot_id, name, domain, industry, number_of_employees,
    annual_revenue, city, state, country, owner_id, properties,
    created_at_hubspot, updated_at_hubspot, synced_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, NOW())
ON CONFLICT (user_id, hubspot_id) DO UPDATE SET
    name = EXCLUDED.name,
    domain = EXCLUDED.domain,
    industry = EXCLUDED.industry,
    number_of_employees = EXCLUDED.number_of_employees,
    annual_revenue = EXCLUDED.annual_revenue,
    city = EXCLUDED.city,
    state = EXCLUDED.state,
    country = EXCLUDED.country,
    owner_id = EXCLUDED.owner_id,
    properties = EXCLUDED.properties,
    updated_at_hubspot = EXCLUDED.updated_at_hubspot,
    synced_at = NOW(),
    updated_at = NOW()
RETURNING *;

-- name: GetHubSpotCompany :one
SELECT * FROM hubspot_companies
WHERE user_id = $1 AND hubspot_id = $2;

-- name: GetHubSpotCompanyByDomain :one
SELECT * FROM hubspot_companies
WHERE user_id = $1 AND domain = $2;

-- name: GetHubSpotCompaniesByUser :many
SELECT * FROM hubspot_companies
WHERE user_id = $1
ORDER BY updated_at_hubspot DESC NULLS LAST
LIMIT $2 OFFSET $3;

-- name: SearchHubSpotCompanies :many
SELECT * FROM hubspot_companies
WHERE user_id = $1
  AND (name ILIKE $2 OR domain ILIKE $2)
ORDER BY updated_at_hubspot DESC NULLS LAST
LIMIT $3;

-- name: GetHubSpotCompaniesByIndustry :many
SELECT * FROM hubspot_companies
WHERE user_id = $1 AND industry = $2
ORDER BY annual_revenue DESC NULLS LAST
LIMIT $3 OFFSET $4;

-- name: CountHubSpotCompanies :one
SELECT COUNT(*) FROM hubspot_companies
WHERE user_id = $1;

-- name: DeleteHubSpotCompany :exec
DELETE FROM hubspot_companies
WHERE user_id = $1 AND hubspot_id = $2;

-- name: DeleteHubSpotCompaniesByUser :exec
DELETE FROM hubspot_companies WHERE user_id = $1;

-- ============================================================================
-- HubSpot Deals
-- ============================================================================

-- name: UpsertHubSpotDeal :one
INSERT INTO hubspot_deals (
    user_id, hubspot_id, deal_name, amount, pipeline, deal_stage,
    close_date, owner_id, associated_company_ids, associated_contact_ids,
    properties, created_at_hubspot, updated_at_hubspot, synced_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, NOW())
ON CONFLICT (user_id, hubspot_id) DO UPDATE SET
    deal_name = EXCLUDED.deal_name,
    amount = EXCLUDED.amount,
    pipeline = EXCLUDED.pipeline,
    deal_stage = EXCLUDED.deal_stage,
    close_date = EXCLUDED.close_date,
    owner_id = EXCLUDED.owner_id,
    associated_company_ids = EXCLUDED.associated_company_ids,
    associated_contact_ids = EXCLUDED.associated_contact_ids,
    properties = EXCLUDED.properties,
    updated_at_hubspot = EXCLUDED.updated_at_hubspot,
    synced_at = NOW(),
    updated_at = NOW()
RETURNING *;

-- name: GetHubSpotDeal :one
SELECT * FROM hubspot_deals
WHERE user_id = $1 AND hubspot_id = $2;

-- name: GetHubSpotDealsByUser :many
SELECT * FROM hubspot_deals
WHERE user_id = $1
ORDER BY updated_at_hubspot DESC NULLS LAST
LIMIT $2 OFFSET $3;

-- name: GetHubSpotDealsByStage :many
SELECT * FROM hubspot_deals
WHERE user_id = $1 AND deal_stage = $2
ORDER BY amount DESC NULLS LAST
LIMIT $3 OFFSET $4;

-- name: GetHubSpotDealsByPipeline :many
SELECT * FROM hubspot_deals
WHERE user_id = $1 AND pipeline = $2
ORDER BY close_date NULLS LAST
LIMIT $3 OFFSET $4;

-- name: GetHubSpotDealsClosingSoon :many
SELECT * FROM hubspot_deals
WHERE user_id = $1
  AND close_date IS NOT NULL
  AND close_date <= $2
  AND deal_stage NOT IN ('closedwon', 'closedlost')
ORDER BY close_date;

-- name: SearchHubSpotDeals :many
SELECT * FROM hubspot_deals
WHERE user_id = $1 AND deal_name ILIKE $2
ORDER BY amount DESC NULLS LAST
LIMIT $3;

-- name: GetHubSpotDealsPipelineValue :one
SELECT
    COUNT(*) as deal_count,
    COALESCE(SUM(amount), 0) as total_value
FROM hubspot_deals
WHERE user_id = $1
  AND deal_stage NOT IN ('closedwon', 'closedlost');

-- name: GetHubSpotDealsWonValue :one
SELECT
    COUNT(*) as deal_count,
    COALESCE(SUM(amount), 0) as total_value
FROM hubspot_deals
WHERE user_id = $1 AND deal_stage = 'closedwon';

-- name: CountHubSpotDealsByStage :many
SELECT deal_stage, COUNT(*) as count, COALESCE(SUM(amount), 0) as total_value
FROM hubspot_deals
WHERE user_id = $1
GROUP BY deal_stage
ORDER BY count DESC;

-- name: DeleteHubSpotDeal :exec
DELETE FROM hubspot_deals
WHERE user_id = $1 AND hubspot_id = $2;

-- name: DeleteHubSpotDealsByUser :exec
DELETE FROM hubspot_deals WHERE user_id = $1;
