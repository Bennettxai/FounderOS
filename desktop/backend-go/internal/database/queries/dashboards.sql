-- ============================================================================
-- DASHBOARD QUERIES
-- CRUD operations for user dashboards, widgets, and templates
-- ============================================================================

-- ============================================================================
-- USER DASHBOARDS
-- ============================================================================

-- name: CreateDashboard :one
INSERT INTO user_dashboards (
    user_id,
    workspace_id,
    name,
    description,
    layout,
    visibility,
    created_via
) VALUES (
    $1, $2, $3, $4, $5, $6, $7
)
RETURNING *;

-- name: GetDashboard :one
SELECT * FROM user_dashboards
WHERE id = $1 AND user_id = $2;

-- name: GetDashboardByID :one
SELECT * FROM user_dashboards
WHERE id = $1;

-- name: GetDashboardByShareToken :one
SELECT * FROM user_dashboards
WHERE share_token = $1 AND visibility = 'public_link';

-- name: ListUserDashboards :many
SELECT * FROM user_dashboards
WHERE user_id = $1
ORDER BY is_default DESC, updated_at DESC;

-- name: ListWorkspaceDashboards :many
SELECT * FROM user_dashboards
WHERE workspace_id = $1 AND visibility IN ('workspace', 'public_link')
ORDER BY updated_at DESC;

-- name: GetDefaultDashboard :one
SELECT * FROM user_dashboards
WHERE user_id = $1 AND is_default = TRUE
LIMIT 1;

-- name: UpdateDashboard :one
UPDATE user_dashboards
SET
    name = COALESCE(sqlc.narg('name'), name),
    description = COALESCE(sqlc.narg('description'), description),
    layout = COALESCE(sqlc.narg('layout'), layout),
    visibility = COALESCE(sqlc.narg('visibility'), visibility)
WHERE id = $1 AND user_id = $2
RETURNING *;

-- name: UpdateDashboardLayout :one
UPDATE user_dashboards
SET layout = $3
WHERE id = $1 AND user_id = $2
RETURNING *;

-- name: DeleteDashboard :exec
DELETE FROM user_dashboards
WHERE id = $1 AND user_id = $2;

-- name: ClearDefaultDashboard :exec
UPDATE user_dashboards
SET is_default = FALSE
WHERE user_id = $1 AND is_default = TRUE;

-- name: SetDefaultDashboard :exec
UPDATE user_dashboards
SET is_default = TRUE
WHERE id = $1 AND user_id = $2;

-- name: DuplicateDashboard :one
WITH source AS (
    SELECT workspace_id, description, layout
    FROM user_dashboards
    WHERE user_dashboards.id = $1
)
INSERT INTO user_dashboards (
    user_id,
    workspace_id,
    name,
    description,
    layout,
    visibility,
    created_via
)
SELECT
    $2,
    source.workspace_id,
    $3,
    source.description,
    source.layout,
    'private',
    'manual'
FROM source
RETURNING user_dashboards.id, user_dashboards.user_id, user_dashboards.workspace_id, 
    user_dashboards.name, user_dashboards.description, user_dashboards.is_default,
    user_dashboards.layout, user_dashboards.visibility, user_dashboards.share_token,
    user_dashboards.is_enforced, user_dashboards.enforced_for_roles, user_dashboards.created_via,
    user_dashboards.created_at, user_dashboards.updated_at;

-- name: UpdateShareToken :one
UPDATE user_dashboards
SET 
    visibility = $3,
    share_token = $4
WHERE id = $1 AND user_id = $2
RETURNING *;

-- name: CountUserDashboards :one
SELECT COUNT(*) FROM user_dashboards
WHERE user_id = $1;

-- ============================================================================
-- DASHBOARD WIDGETS (Widget Type Registry)
-- ============================================================================

-- name: ListWidgetTypes :many
SELECT * FROM dashboard_widgets
WHERE is_enabled = TRUE
ORDER BY category, name;

-- name: ListAllWidgetTypes :many
SELECT * FROM dashboard_widgets
ORDER BY category, name;

-- name: GetWidgetType :one
SELECT * FROM dashboard_widgets
WHERE id = $1;

-- name: GetWidgetTypeByName :one
SELECT * FROM dashboard_widgets
WHERE widget_type = $1;

-- name: GetWidgetsByCategory :many
SELECT * FROM dashboard_widgets
WHERE category = $1 AND is_enabled = TRUE
ORDER BY name;

-- name: EnableWidget :exec
UPDATE dashboard_widgets
SET is_enabled = TRUE
WHERE widget_type = $1;

-- name: DisableWidget :exec
UPDATE dashboard_widgets
SET is_enabled = FALSE
WHERE widget_type = $1;

-- ============================================================================
-- DASHBOARD TEMPLATES
-- ============================================================================

-- name: ListDashboardTemplates :many
SELECT * FROM dashboard_templates
ORDER BY sort_order, name;

-- name: GetDashboardTemplate :one
SELECT * FROM dashboard_templates
WHERE id = $1;

-- name: GetDefaultTemplate :one
SELECT * FROM dashboard_templates
WHERE is_default = TRUE
LIMIT 1;

-- name: CreateDashboardFromTemplate :one
WITH template_data AS (
    SELECT dashboard_templates.name, dashboard_templates.description, dashboard_templates.layout
    FROM dashboard_templates
    WHERE dashboard_templates.id = $1
)
INSERT INTO user_dashboards (
    user_id,
    workspace_id,
    name,
    description,
    layout,
    visibility,
    created_via
)
SELECT
    $2,
    $3,
    COALESCE($4::VARCHAR, template_data.name),
    template_data.description,
    template_data.layout,
    'private',
    'template'
FROM template_data
RETURNING user_dashboards.id, user_dashboards.user_id, user_dashboards.workspace_id, 
    user_dashboards.name, user_dashboards.description, user_dashboards.is_default,
    user_dashboards.layout, user_dashboards.visibility, user_dashboards.share_token,
    user_dashboards.is_enforced, user_dashboards.enforced_for_roles, user_dashboards.created_via,
    user_dashboards.created_at, user_dashboards.updated_at;

-- ============================================================================
-- ANALYTICS QUERIES (for widgets)
-- ============================================================================

-- name: CountTasksDueToday :one
SELECT COUNT(*) FROM tasks
WHERE user_id = $1 
  AND due_date::date = CURRENT_DATE
  AND status != 'done';

-- name: CountTasksOverdue :one
SELECT COUNT(*) FROM tasks
WHERE user_id = $1 
  AND due_date < NOW()
  AND status != 'done';

-- name: CountTasksCompletedThisWeek :one
SELECT COUNT(*) FROM tasks
WHERE user_id = $1 
  AND status = 'done'
  AND updated_at >= date_trunc('week', CURRENT_DATE);

-- name: CountActiveProjects :one
SELECT COUNT(*) FROM projects
WHERE user_id = $1 
  AND status NOT IN ('completed', 'archived', 'cancelled');

-- name: GetTaskBurndownData :many
SELECT 
    d::date as date,
    COUNT(*) FILTER (WHERE t.created_at::date = d::date) as created,
    COUNT(*) FILTER (WHERE t.status = 'done' AND t.updated_at::date = d::date) as completed
FROM generate_series(
    CURRENT_DATE - ($2::int || ' days')::interval,
    CURRENT_DATE,
    '1 day'::interval
) d
LEFT JOIN tasks t ON t.user_id = $1 
    AND (t.created_at::date = d::date OR (t.status = 'done' AND t.updated_at::date = d::date))
    AND ($3::uuid IS NULL OR t.project_id = $3)
GROUP BY d::date
ORDER BY d::date;

-- name: GetWorkloadHeatmapData :many
SELECT 
    d::date as date,
    COUNT(*) FILTER (WHERE t.due_date::date = d::date) as tasks_due,
    COUNT(*) FILTER (WHERE t.created_at::date = d::date) as tasks_created
FROM generate_series($2::date, $3::date, '1 day'::interval) d
LEFT JOIN tasks t ON t.user_id = $1 
    AND (t.due_date::date = d::date OR t.created_at::date = d::date)
GROUP BY d::date
ORDER BY d::date;

-- name: GetUpcomingTasksDueByDate :many
SELECT 
    due_date::date as due_date,
    COUNT(*) as task_count
FROM tasks
WHERE user_id = $1 
  AND due_date IS NOT NULL
  AND due_date >= CURRENT_DATE
  AND due_date <= CURRENT_DATE + ($2::int || ' days')::interval
  AND status != 'done'
GROUP BY due_date::date
ORDER BY due_date::date;
