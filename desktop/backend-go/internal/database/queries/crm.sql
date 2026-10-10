-- ============================================================================
-- COMPANIES QUERIES
-- ============================================================================

-- name: ListCompanies :many
SELECT * FROM companies
WHERE user_id = $1
  AND (sqlc.narg(industry)::varchar IS NULL OR industry = sqlc.narg(industry))
  AND (sqlc.narg(lifecycle_stage)::varchar IS NULL OR lifecycle_stage = sqlc.narg(lifecycle_stage))
ORDER BY updated_at DESC
LIMIT sqlc.arg(limit_val)::int OFFSET sqlc.arg(offset_val)::int;

-- name: GetCompany :one
SELECT * FROM companies
WHERE id = $1 AND user_id = $2;

-- name: GetCompanyByName :one
SELECT * FROM companies
WHERE user_id = $1 AND name ILIKE $2
LIMIT 1;

-- name: CreateCompany :one
INSERT INTO companies (
    user_id, name, legal_name, industry, company_size,
    website, email, phone,
    address_line1, address_line2, city, state, postal_code, country,
    annual_revenue, currency, tax_id,
    linkedin_url, twitter_handle,
    owner_id, lifecycle_stage, lead_source,
    logo_url, custom_fields, metadata
) VALUES (
    $1, $2, $3, $4, $5,
    $6, $7, $8,
    $9, $10, $11, $12, $13, $14,
    $15, $16, $17,
    $18, $19,
    $20, $21, $22,
    $23, $24, $25
)
RETURNING *;

-- name: UpdateCompany :one
UPDATE companies
SET name = $3, legal_name = $4, industry = $5, company_size = $6,
    website = $7, email = $8, phone = $9,
    address_line1 = $10, address_line2 = $11, city = $12, state = $13, postal_code = $14, country = $15,
    annual_revenue = $16, lifecycle_stage = $17,
    linkedin_url = $18, twitter_handle = $19,
    logo_url = $20, custom_fields = $21, metadata = $22,
    updated_at = NOW()
WHERE id = $1 AND user_id = $2
RETURNING *;

-- name: DeleteCompany :exec
DELETE FROM companies
WHERE id = $1 AND user_id = $2;

-- name: SearchCompanies :many
SELECT * FROM companies
WHERE user_id = $1
  AND (name ILIKE '%' || $2 || '%' OR website ILIKE '%' || $2 || '%')
ORDER BY name ASC
LIMIT sqlc.arg(limit_val)::int;

-- name: UpdateCompanyScores :exec
UPDATE companies
SET health_score = $3, engagement_score = $4, updated_at = NOW()
WHERE id = $1 AND user_id = $2;

-- ============================================================================
-- CONTACT-COMPANY RELATIONS QUERIES
-- ============================================================================

-- name: ListCompanyContacts :many
SELECT ccr.*, c.name as contact_name, c.email as contact_email
FROM contact_company_relations ccr
JOIN clients c ON ccr.contact_id = c.id
WHERE ccr.company_id = $1
ORDER BY ccr.is_primary DESC, c.name ASC;

-- name: ListContactCompanies :many
SELECT ccr.*, co.name as company_name
FROM contact_company_relations ccr
JOIN companies co ON ccr.company_id = co.id
WHERE ccr.contact_id = $1
ORDER BY ccr.is_primary DESC, co.name ASC;

-- name: CreateContactCompanyRelation :one
INSERT INTO contact_company_relations (contact_id, company_id, job_title, department, role_type, is_primary)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: UpdateContactCompanyRelation :one
UPDATE contact_company_relations
SET job_title = $2, department = $3, role_type = $4, is_primary = $5, is_active = $6, updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteContactCompanyRelation :exec
DELETE FROM contact_company_relations
WHERE id = $1;

-- name: SetPrimaryContact :exec
UPDATE contact_company_relations
SET is_primary = (contact_id = $2)
WHERE company_id = $1;

-- ============================================================================
-- PIPELINES QUERIES
-- ============================================================================

-- name: ListPipelines :many
SELECT * FROM pipelines
WHERE user_id = $1 AND is_active = TRUE
ORDER BY is_default DESC, name ASC;

-- name: GetPipeline :one
SELECT * FROM pipelines
WHERE id = $1 AND user_id = $2;

-- name: GetDefaultPipeline :one
SELECT * FROM pipelines
WHERE user_id = $1 AND is_default = TRUE
LIMIT 1;

-- name: CreatePipeline :one
INSERT INTO pipelines (user_id, name, description, pipeline_type, currency, is_default, color, icon)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: UpdatePipeline :one
UPDATE pipelines
SET name = $3, description = $4, currency = $5, color = $6, icon = $7, is_active = $8, updated_at = NOW()
WHERE id = $1 AND user_id = $2
RETURNING *;

-- name: DeletePipeline :exec
DELETE FROM pipelines
WHERE id = $1 AND user_id = $2;

-- name: SetDefaultPipeline :exec
UPDATE pipelines
SET is_default = (id = $2)
WHERE user_id = $1;

-- ============================================================================
-- PIPELINE STAGES QUERIES
-- ============================================================================

-- name: ListPipelineStages :many
SELECT * FROM pipeline_stages
WHERE pipeline_id = $1
ORDER BY position ASC;

-- name: GetPipelineStage :one
SELECT * FROM pipeline_stages
WHERE id = $1;

-- name: CreatePipelineStage :one
INSERT INTO pipeline_stages (pipeline_id, name, description, position, probability, stage_type, rotting_days, color)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: UpdatePipelineStage :one
UPDATE pipeline_stages
SET name = $2, description = $3, probability = $4, rotting_days = $5, color = $6, updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: UpdateStagePosition :exec
UPDATE pipeline_stages
SET position = $2, updated_at = NOW()
WHERE id = $1;

-- name: DeletePipelineStage :exec
DELETE FROM pipeline_stages
WHERE id = $1;

-- ============================================================================
-- DEALS QUERIES (CRM Pipeline Deals)
-- ============================================================================

-- name: ListCRMDeals :many
SELECT
    d.id,
    d.user_id,
    d.pipeline_id,
    d.stage_id,
    d.name,
    d.description,
    d.amount,
    d.currency,
    d.probability,
    d.expected_close_date,
    d.actual_close_date,
    d.owner_id,
    d.company_id,
    d.primary_contact_id,
    d.status,
    d.lost_reason,
    d.priority,
    d.lead_source,
    d.deal_score,
    d.custom_fields,
    d.client_id,
    d.stage_entered_at,
    d.created_at,
    d.updated_at,
    ps.name as stage_name,
    p.name as pipeline_name,
    c.name as company_name
FROM deals d
JOIN pipeline_stages ps ON d.stage_id = ps.id
JOIN pipelines p ON d.pipeline_id = p.id
LEFT JOIN companies c ON d.company_id = c.id
WHERE d.user_id = $1
  AND (sqlc.narg(pipeline_id)::uuid IS NULL OR d.pipeline_id = sqlc.narg(pipeline_id))
  AND (sqlc.narg(stage_id)::uuid IS NULL OR d.stage_id = sqlc.narg(stage_id))
  AND (sqlc.narg(status)::varchar IS NULL OR d.status = sqlc.narg(status))
  AND (sqlc.narg(owner_id)::varchar IS NULL OR d.owner_id = sqlc.narg(owner_id))
ORDER BY d.updated_at DESC
LIMIT sqlc.arg(limit_val)::int OFFSET sqlc.arg(offset_val)::int;

-- name: ListCRMDealsByStage :many
SELECT d.*, c.name as company_name
FROM deals d
LEFT JOIN companies c ON d.company_id = c.id
WHERE d.stage_id = $1
ORDER BY d.created_at DESC;

-- name: GetCRMDeal :one
SELECT d.*, ps.name as stage_name, p.name as pipeline_name, c.name as company_name
FROM deals d
JOIN pipeline_stages ps ON d.stage_id = ps.id
JOIN pipelines p ON d.pipeline_id = p.id
LEFT JOIN companies c ON d.company_id = c.id
WHERE d.id = $1 AND d.user_id = $2;

-- name: CreateCRMDeal :one
INSERT INTO deals (
    user_id, pipeline_id, stage_id,
    name, description, amount, currency, probability,
    expected_close_date, owner_id, company_id, primary_contact_id,
    status, priority, lead_source, custom_fields
) VALUES (
    $1, $2, $3,
    $4, $5, $6, $7, $8,
    $9, $10, $11, $12,
    $13, $14, $15, $16
)
RETURNING *;

-- name: UpdateCRMDeal :one
UPDATE deals
SET name = $3, description = $4, amount = $5, probability = $6,
    expected_close_date = $7, owner_id = $8, company_id = $9, primary_contact_id = $10,
    priority = $11, custom_fields = $12, updated_at = NOW()
WHERE id = $1 AND user_id = $2
RETURNING *;

-- name: UpdateCRMDealStage :one
UPDATE deals
SET stage_id = $3, updated_at = NOW()
WHERE id = $1 AND user_id = $2
RETURNING *;

-- name: UpdateCRMDealStatus :one
UPDATE deals
SET status = $3,
    lost_reason = $4,
    actual_close_date = CASE WHEN $3 IN ('won', 'lost') THEN NOW() ELSE NULL END,
    updated_at = NOW()
WHERE id = $1 AND user_id = $2
RETURNING *;

-- name: DeleteCRMDeal :exec
DELETE FROM deals
WHERE id = $1 AND user_id = $2;

-- name: GetCRMDealStats :one
SELECT
    COUNT(*) as total_deals,
    COUNT(*) FILTER (WHERE status = 'open') as open_deals,
    COUNT(*) FILTER (WHERE status = 'won') as won_deals,
    COUNT(*) FILTER (WHERE status = 'lost') as lost_deals,
    COALESCE(SUM(amount) FILTER (WHERE status = 'open'), 0) as open_value,
    COALESCE(SUM(amount) FILTER (WHERE status = 'won'), 0) as won_value,
    COALESCE(SUM(amount) FILTER (WHERE status = 'lost'), 0) as lost_value
FROM deals
WHERE user_id = $1
  AND (sqlc.narg(pipeline_id)::uuid IS NULL OR pipeline_id = sqlc.narg(pipeline_id));

-- name: GetCRMDealsByCompany :many
SELECT * FROM deals
WHERE company_id = $1
ORDER BY updated_at DESC;

-- name: SearchCRMDeals :many
SELECT d.*, ps.name as stage_name, c.name as company_name
FROM deals d
JOIN pipeline_stages ps ON d.stage_id = ps.id
LEFT JOIN companies c ON d.company_id = c.id
WHERE d.user_id = $1
  AND d.name ILIKE '%' || $2 || '%'
ORDER BY d.updated_at DESC
LIMIT sqlc.arg(limit_val)::int;

-- ============================================================================
-- CRM ACTIVITIES QUERIES
-- ============================================================================

-- name: ListCRMActivities :many
SELECT * FROM crm_activities
WHERE user_id = $1
  AND (sqlc.narg(activity_type)::crm_activity_type IS NULL OR activity_type = sqlc.narg(activity_type))
  AND (sqlc.narg(is_completed)::boolean IS NULL OR is_completed = sqlc.narg(is_completed))
ORDER BY activity_date DESC
LIMIT sqlc.arg(limit_val)::int OFFSET sqlc.arg(offset_val)::int;

-- name: ListDealActivities :many
SELECT * FROM crm_activities
WHERE deal_id = $1
ORDER BY activity_date DESC;

-- name: ListCompanyActivities :many
SELECT * FROM crm_activities
WHERE company_id = $1
ORDER BY activity_date DESC;

-- name: ListContactActivities :many
SELECT * FROM crm_activities
WHERE contact_id = $1
ORDER BY activity_date DESC;

-- name: GetCRMActivity :one
SELECT * FROM crm_activities
WHERE id = $1 AND user_id = $2;

-- name: CreateCRMActivity :one
INSERT INTO crm_activities (
    user_id, activity_type, subject, description, outcome,
    deal_id, company_id, contact_id, participants,
    activity_date, duration_minutes,
    call_direction, call_disposition, call_recording_url,
    email_direction, email_message_id,
    meeting_location, meeting_url,
    owner_id, is_completed, completed_by, completed_at
) VALUES (
    $1, $2, $3, $4, $5,
    $6, $7, $8, $9,
    $10, $11,
    $12, $13, $14,
    $15, $16,
    $17, $18,
    $19, $20, $21, $22
)
RETURNING *;

-- name: UpdateCRMActivity :one
UPDATE crm_activities
SET subject = $3, description = $4, outcome = $5,
    activity_date = $6, duration_minutes = $7,
    is_completed = $8, completed_by = $9, completed_at = $10,
    updated_at = NOW()
WHERE id = $1 AND user_id = $2
RETURNING *;

-- name: CompleteCRMActivity :one
UPDATE crm_activities
SET is_completed = TRUE, completed_by = $3, completed_at = NOW(), outcome = $4, updated_at = NOW()
WHERE id = $1 AND user_id = $2
RETURNING *;

-- name: DeleteCRMActivity :exec
DELETE FROM crm_activities
WHERE id = $1 AND user_id = $2;

-- name: ListPendingActivities :many
SELECT * FROM crm_activities
WHERE user_id = $1
  AND is_completed = FALSE
  AND activity_date <= NOW() + INTERVAL '7 days'
ORDER BY activity_date ASC
LIMIT sqlc.arg(limit_val)::int;

-- name: ListOverdueActivities :many
SELECT * FROM crm_activities
WHERE user_id = $1
  AND is_completed = FALSE
  AND activity_date < NOW()
ORDER BY activity_date ASC;

-- ============================================================================
-- DEAL STAGE HISTORY QUERIES
-- ============================================================================

-- name: ListDealStageHistory :many
SELECT dsh.*, ps_from.name as from_stage_name, ps_to.name as to_stage_name
FROM deal_stage_history dsh
LEFT JOIN pipeline_stages ps_from ON dsh.from_stage_id = ps_from.id
JOIN pipeline_stages ps_to ON dsh.to_stage_id = ps_to.id
WHERE dsh.deal_id = $1
ORDER BY dsh.changed_at DESC;

-- name: GetStageConversionRate :one
SELECT
    from_stage_id,
    to_stage_id,
    COUNT(*) as total_transitions,
    AVG(duration_seconds) as avg_duration_seconds
FROM deal_stage_history
WHERE from_stage_id = $1 AND to_stage_id = $2
GROUP BY from_stage_id, to_stage_id;

-- name: GetAverageTimeInStage :one
SELECT
    to_stage_id,
    AVG(duration_seconds) as avg_seconds,
    COUNT(*) as sample_size
FROM deal_stage_history
WHERE to_stage_id = $1
GROUP BY to_stage_id;
