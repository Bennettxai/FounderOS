-- ═══════════════════════════════════════════════════════════════════════════════
-- Custom Modules SQLC Queries
-- ═══════════════════════════════════════════════════════════════════════════════

-- ───────────────────────────────────────────────────────────────────────────────
-- CREATE OPERATIONS
-- ───────────────────────────────────────────────────────────────────────────────

-- name: CreateCustomModule :one
INSERT INTO custom_modules (
    created_by,
    workspace_id,
    name,
    slug,
    description,
    category,
    version,
    manifest,
    config,
    icon,
    tags,
    keywords
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12
) RETURNING *;

-- name: CreateModuleVersion :one
INSERT INTO custom_module_versions (
    module_id,
    version,
    changelog,
    manifest_snapshot,
    config_snapshot,
    created_by,
    is_stable,
    is_breaking
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8
) RETURNING *;

-- name: InstallModule :one
INSERT INTO module_installations (
    module_id,
    workspace_id,
    installed_by,
    installed_version,
    config_override,
    is_enabled,
    is_auto_update
) VALUES (
    $1, $2, $3, $4, $5, $6, $7
) RETURNING *;

-- name: ShareModule :one
INSERT INTO module_shares (
    module_id,
    shared_with_user_id,
    shared_with_workspace_id,
    shared_with_email,
    can_view,
    can_install,
    can_modify,
    can_reshare,
    shared_by,
    expires_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10
) RETURNING *;

-- ───────────────────────────────────────────────────────────────────────────────
-- READ OPERATIONS
-- ───────────────────────────────────────────────────────────────────────────────

-- name: GetCustomModule :one
SELECT * FROM custom_modules
WHERE id = $1;

-- name: GetCustomModuleBySlug :one
SELECT * FROM custom_modules
WHERE workspace_id = $1 AND slug = $2;

-- name: ListCustomModules :many
SELECT * FROM custom_modules
WHERE workspace_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: ListPublicModules :many
SELECT * FROM custom_modules
WHERE is_public = TRUE AND is_published = TRUE
ORDER BY install_count DESC, created_at DESC
LIMIT $1 OFFSET $2;

-- name: ListModulesByCategory :many
SELECT * FROM custom_modules
WHERE category = $1 AND is_public = TRUE
ORDER BY install_count DESC
LIMIT $2 OFFSET $3;

-- name: SearchModules :many
SELECT * FROM custom_modules
WHERE
    is_public = TRUE
    AND (
        name ILIKE '%' || $1 || '%'
        OR description ILIKE '%' || $1 || '%'
        OR $1 = ANY(tags)
        OR $1 = ANY(keywords)
    )
ORDER BY install_count DESC
LIMIT $2 OFFSET $3;

-- name: GetModuleVersion :one
SELECT * FROM custom_module_versions
WHERE module_id = $1 AND version = $2;

-- name: ListModuleVersions :many
SELECT * FROM custom_module_versions
WHERE module_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: GetLatestModuleVersion :one
SELECT * FROM custom_module_versions
WHERE module_id = $1
ORDER BY created_at DESC
LIMIT 1;

-- name: GetModuleInstallation :one
SELECT * FROM module_installations
WHERE module_id = $1 AND workspace_id = $2;

-- name: ListInstalledModules :many
SELECT
    mi.*,
    cm.name,
    cm.slug,
    cm.description,
    cm.category,
    cm.icon
FROM module_installations mi
JOIN custom_modules cm ON mi.module_id = cm.id
WHERE mi.workspace_id = $1 AND mi.is_enabled = TRUE
ORDER BY mi.installed_at DESC;

-- name: ListModuleShares :many
SELECT * FROM module_shares
WHERE module_id = $1
ORDER BY shared_at DESC;

-- name: ListSharedWithUser :many
SELECT
    ms.*,
    cm.name,
    cm.slug,
    cm.description,
    cm.category,
    cm.icon
FROM module_shares ms
JOIN custom_modules cm ON ms.module_id = cm.id
WHERE ms.shared_with_user_id = $1
ORDER BY ms.shared_at DESC;

-- name: CheckModulePermission :one
SELECT
    can_view,
    can_install,
    can_modify,
    can_reshare
FROM module_shares
WHERE module_id = $1 AND (
    shared_with_user_id = $2
    OR shared_with_workspace_id = $3
);

-- ───────────────────────────────────────────────────────────────────────────────
-- UPDATE OPERATIONS
-- ───────────────────────────────────────────────────────────────────────────────

-- name: UpdateCustomModule :one
UPDATE custom_modules SET
    name = COALESCE(sqlc.narg('name'), name),
    description = COALESCE(sqlc.narg('description'), description),
    category = COALESCE(sqlc.narg('category'), category),
    version = COALESCE(sqlc.narg('version'), version),
    manifest = COALESCE(sqlc.narg('manifest'), manifest),
    config = COALESCE(sqlc.narg('config'), config),
    icon = COALESCE(sqlc.narg('icon'), icon),
    tags = COALESCE(sqlc.narg('tags'), tags),
    keywords = COALESCE(sqlc.narg('keywords'), keywords),
    is_public = COALESCE(sqlc.narg('is_public'), is_public),
    is_published = COALESCE(sqlc.narg('is_published'), is_published),
    is_template = COALESCE(sqlc.narg('is_template'), is_template),
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: PublishModule :one
UPDATE custom_modules SET
    is_published = TRUE,
    published_at = NOW()
WHERE id = $1
RETURNING *;

-- name: UnpublishModule :one
UPDATE custom_modules SET
    is_published = FALSE
WHERE id = $1
RETURNING *;

-- name: UpdateModuleInstallation :one
UPDATE module_installations SET
    installed_version = COALESCE(sqlc.narg('installed_version'), installed_version),
    config_override = COALESCE(sqlc.narg('config_override'), config_override),
    is_enabled = COALESCE(sqlc.narg('is_enabled'), is_enabled),
    is_auto_update = COALESCE(sqlc.narg('is_auto_update'), is_auto_update),
    last_used_at = COALESCE(sqlc.narg('last_used_at'), last_used_at),
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: UpdateModuleShare :one
UPDATE module_shares SET
    can_view = COALESCE(sqlc.narg('can_view'), can_view),
    can_install = COALESCE(sqlc.narg('can_install'), can_install),
    can_modify = COALESCE(sqlc.narg('can_modify'), can_modify),
    can_reshare = COALESCE(sqlc.narg('can_reshare'), can_reshare),
    expires_at = COALESCE(sqlc.narg('expires_at'), expires_at)
WHERE id = $1
RETURNING *;

-- name: IncrementModuleStarCount :exec
UPDATE custom_modules SET
    star_count = star_count + 1
WHERE id = $1;

-- name: DecrementModuleStarCount :exec
UPDATE custom_modules SET
    star_count = GREATEST(star_count - 1, 0)
WHERE id = $1;

-- ───────────────────────────────────────────────────────────────────────────────
-- DELETE OPERATIONS
-- ───────────────────────────────────────────────────────────────────────────────

-- name: DeleteCustomModule :exec
DELETE FROM custom_modules
WHERE id = $1 AND created_by = $2;

-- name: UninstallModule :exec
DELETE FROM module_installations
WHERE module_id = $1 AND workspace_id = $2;

-- name: DeleteModuleShare :exec
DELETE FROM module_shares
WHERE id = $1 AND shared_by = $2;

-- name: RevokeModuleShare :exec
DELETE FROM module_shares
WHERE module_id = $1 AND (
    shared_with_user_id = $2
    OR shared_with_workspace_id = $3
);

-- ───────────────────────────────────────────────────────────────────────────────
-- STATS & ANALYTICS
-- ───────────────────────────────────────────────────────────────────────────────

-- name: GetModuleStats :one
SELECT
    COUNT(*) as total_modules,
    COUNT(*) FILTER (WHERE is_published = TRUE) as published_count,
    COUNT(*) FILTER (WHERE is_public = TRUE) as public_count,
    SUM(install_count) as total_installs
FROM custom_modules
WHERE workspace_id = $1;

-- name: GetPopularModules :many
SELECT * FROM custom_modules
WHERE is_public = TRUE AND is_published = TRUE
ORDER BY install_count DESC, star_count DESC
LIMIT $1;

-- name: GetTrendingModules :many
SELECT
    cm.*,
    COUNT(mi.id) as recent_installs
FROM custom_modules cm
LEFT JOIN module_installations mi ON cm.id = mi.module_id
    AND mi.installed_at > NOW() - INTERVAL '7 days'
WHERE cm.is_public = TRUE AND cm.is_published = TRUE
GROUP BY cm.id
ORDER BY recent_installs DESC, cm.star_count DESC
LIMIT $1;

-- name: GetModulesByUser :many
SELECT * FROM custom_modules
WHERE created_by = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: CountUserModules :one
SELECT COUNT(*) FROM custom_modules
WHERE created_by = $1;

-- name: CountWorkspaceModules :one
SELECT COUNT(*) FROM custom_modules
WHERE workspace_id = $1;

-- name: CountInstalledModules :one
SELECT COUNT(*) FROM module_installations
WHERE workspace_id = $1 AND is_enabled = TRUE;

-- ═══════════════════════════════════════════════════════════════════════════════
-- END OF QUERIES
-- ═══════════════════════════════════════════════════════════════════════════════
