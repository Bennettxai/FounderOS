-- name: ListDailyLogs :many
SELECT * FROM daily_logs
WHERE user_id = $1
ORDER BY date DESC
LIMIT $2 OFFSET $3;

-- name: GetTodayLog :one
SELECT * FROM daily_logs
WHERE user_id = $1 AND date = CURRENT_DATE
LIMIT 1;

-- name: GetDailyLogByDate :one
SELECT * FROM daily_logs
WHERE user_id = $1 AND date = $2
LIMIT 1;

-- name: CreateDailyLog :one
INSERT INTO daily_logs (user_id, date, content, transcription_source, extracted_actions, extracted_patterns, energy_level)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: UpdateDailyLog :one
UPDATE daily_logs
SET content = $2, transcription_source = $3, extracted_actions = $4, extracted_patterns = $5, energy_level = $6, updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: UpsertDailyLog :one
INSERT INTO daily_logs (user_id, date, content, transcription_source, extracted_actions, extracted_patterns, energy_level)
VALUES ($1, $2, $3, $4, $5, $6, $7)
ON CONFLICT (user_id, date) DO UPDATE
SET content = EXCLUDED.content, transcription_source = EXCLUDED.transcription_source,
    extracted_actions = EXCLUDED.extracted_actions, extracted_patterns = EXCLUDED.extracted_patterns,
    energy_level = EXCLUDED.energy_level, updated_at = NOW()
RETURNING *;

-- name: DeleteDailyLog :exec
DELETE FROM daily_logs
WHERE id = $1 AND user_id = $2;
