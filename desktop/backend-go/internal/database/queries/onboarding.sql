-- name: CreateOnboardingEmailMetadata :one
INSERT INTO onboarding_email_metadata (
    session_id,
    email_id,
    sender_domain,
    subject_keywords,
    body_keywords,
    detected_tools,
    topics,
    sentiment,
    importance_score,
    category
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
ON CONFLICT (session_id, email_id) DO UPDATE SET
    sender_domain = EXCLUDED.sender_domain,
    subject_keywords = EXCLUDED.subject_keywords,
    body_keywords = EXCLUDED.body_keywords,
    detected_tools = EXCLUDED.detected_tools,
    topics = EXCLUDED.topics,
    sentiment = EXCLUDED.sentiment,
    importance_score = EXCLUDED.importance_score,
    category = EXCLUDED.category,
    updated_at = NOW()
RETURNING *;

-- name: GetOnboardingEmailMetadataBySession :many
SELECT * FROM onboarding_email_metadata
WHERE session_id = $1
ORDER BY created_at DESC;

-- name: GetOnboardingEmailMetadataByEmail :one
SELECT * FROM onboarding_email_metadata
WHERE session_id = $1 AND email_id = $2;

-- name: GetOnboardingSession :one
SELECT * FROM onboarding_sessions
WHERE id = $1;

-- name: UpdateOnboardingSessionAnalysisCompleted :exec
UPDATE onboarding_sessions
SET analysis_completed = TRUE, updated_at = NOW()
WHERE id = $1;

-- name: GetOnboardingSessionByWorkspace :one
SELECT id, user_id, status, extracted_data, workspace_id, analysis_completed
FROM onboarding_sessions
WHERE workspace_id = $1
AND analysis_completed = TRUE
ORDER BY created_at DESC
LIMIT 1;
