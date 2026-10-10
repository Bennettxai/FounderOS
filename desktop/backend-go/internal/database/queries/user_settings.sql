-- name: GetUserSettings :one
SELECT * FROM user_settings
WHERE user_id = $1;

-- name: CreateUserSettings :one
INSERT INTO user_settings (user_id, default_model, email_notifications, daily_summary, theme, sidebar_collapsed, share_analytics, custom_settings)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: UpdateUserSettings :one
UPDATE user_settings
SET default_model = $2, email_notifications = $3, daily_summary = $4, theme = $5, sidebar_collapsed = $6, share_analytics = $7, custom_settings = $8, updated_at = NOW()
WHERE user_id = $1
RETURNING *;

-- name: UpsertUserSettings :one
INSERT INTO user_settings (id, user_id, default_model, email_notifications, daily_summary, theme, sidebar_collapsed, share_analytics, custom_settings, created_at, updated_at)
VALUES (gen_random_uuid(), $1, $2, $3, $4, $5, $6, $7, $8, NOW(), NOW())
ON CONFLICT (user_id) DO UPDATE
SET default_model = EXCLUDED.default_model, email_notifications = EXCLUDED.email_notifications,
    daily_summary = EXCLUDED.daily_summary, theme = EXCLUDED.theme, sidebar_collapsed = EXCLUDED.sidebar_collapsed,
    share_analytics = EXCLUDED.share_analytics, custom_settings = EXCLUDED.custom_settings, updated_at = NOW()
RETURNING *;

-- name: UpdateThinkingSettings :one
UPDATE user_settings
SET thinking_enabled = $2,
    thinking_show_in_ui = $3,
    thinking_save_traces = $4,
    thinking_default_template_id = $5,
    thinking_max_tokens = $6,
    updated_at = NOW()
WHERE user_id = $1
RETURNING *;

-- name: GetThinkingSettings :one
SELECT thinking_enabled, thinking_show_in_ui, thinking_save_traces,
       thinking_default_template_id, thinking_max_tokens
FROM user_settings
WHERE user_id = $1;
