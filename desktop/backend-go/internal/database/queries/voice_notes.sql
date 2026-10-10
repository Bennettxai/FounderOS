-- name: CreateVoiceNote :one
INSERT INTO voice_notes (
    user_id,
    transcript,
    duration_seconds,
    word_count,
    words_per_minute,
    language,
    audio_file_path,
    context_id,
    project_id,
    conversation_id
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10
)
RETURNING *;

-- name: GetVoiceNote :one
SELECT * FROM voice_notes
WHERE id = $1 AND user_id = $2;

-- name: ListVoiceNotes :many
SELECT * FROM voice_notes
WHERE user_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: ListVoiceNotesByDate :many
SELECT * FROM voice_notes
WHERE user_id = $1
  AND created_at >= $2
  AND created_at < $3
ORDER BY created_at DESC;

-- name: ListVoiceNotesByProject :many
SELECT * FROM voice_notes
WHERE user_id = $1 AND project_id = $2
ORDER BY created_at DESC;

-- name: ListVoiceNotesByContext :many
SELECT * FROM voice_notes
WHERE user_id = $1 AND context_id = $2
ORDER BY created_at DESC;

-- name: GetVoiceNoteStats :one
SELECT
    COUNT(*)::INTEGER as total_notes,
    COALESCE(SUM(duration_seconds), 0)::INTEGER as total_duration_seconds,
    COALESCE(SUM(word_count), 0)::INTEGER as total_words,
    COALESCE(AVG(words_per_minute), 0)::NUMERIC(10,2) as avg_words_per_minute
FROM voice_notes
WHERE user_id = $1;

-- name: GetVoiceNoteStatsByDateRange :one
SELECT
    COUNT(*)::INTEGER as total_notes,
    COALESCE(SUM(duration_seconds), 0)::INTEGER as total_duration_seconds,
    COALESCE(SUM(word_count), 0)::INTEGER as total_words,
    COALESCE(AVG(words_per_minute), 0)::NUMERIC(10,2) as avg_words_per_minute
FROM voice_notes
WHERE user_id = $1
  AND created_at >= $2
  AND created_at < $3;

-- name: DeleteVoiceNote :exec
DELETE FROM voice_notes
WHERE id = $1 AND user_id = $2;

-- name: UpdateVoiceNoteContext :exec
UPDATE voice_notes
SET context_id = $3, updated_at = NOW()
WHERE id = $1 AND user_id = $2;

-- name: UpdateVoiceNoteProject :exec
UPDATE voice_notes
SET project_id = $3, updated_at = NOW()
WHERE id = $1 AND user_id = $2;

-- name: CountVoiceNotesByDate :many
SELECT
    DATE(created_at) as date,
    COUNT(*)::INTEGER as count,
    SUM(duration_seconds)::INTEGER as total_duration
FROM voice_notes
WHERE user_id = $1
GROUP BY DATE(created_at)
ORDER BY date DESC
LIMIT $2;
