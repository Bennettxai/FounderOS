-- ============================================================================
-- Workspace Queries for SQLC
-- ============================================================================

-- =========================
-- WORKSPACES
-- =========================

-- name: CreateWorkspace :one
INSERT INTO workspaces (name, slug, description, logo_url, plan_type, owner_id)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: GetWorkspaceByID :one
SELECT * FROM workspaces WHERE id = $1;

-- name: GetWorkspaceBySlug :one
SELECT * FROM workspaces WHERE slug = $1;

-- name: ListUserWorkspaces :many
SELECT
    w.*,
    wm.role_name,
    wm.status as member_status,
    wm.joined_at,
    (SELECT COUNT(*) FROM workspace_members WHERE workspace_id = w.id AND status = 'active') as member_count
FROM workspaces w
JOIN workspace_members wm ON w.id = wm.workspace_id
WHERE wm.user_id = $1 AND wm.status = 'active'
ORDER BY w.name;

-- name: GetUserDefaultWorkspace :one
SELECT
    w.*,
    wm.role_name,
    wm.status as member_status,
    wm.joined_at
FROM workspaces w
JOIN workspace_members wm ON w.id = wm.workspace_id
WHERE wm.user_id = $1 AND wm.status = 'active'
ORDER BY w.owner_id = $1 DESC, w.created_at ASC
LIMIT 1;

-- name: UpdateWorkspace :one
UPDATE workspaces
SET 
    name = COALESCE(sqlc.narg('name'), name),
    slug = COALESCE(sqlc.narg('slug'), slug),
    description = COALESCE(sqlc.narg('description'), description),
    logo_url = COALESCE(sqlc.narg('logo_url'), logo_url),
    settings = COALESCE(sqlc.narg('settings'), settings),
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteWorkspace :exec
DELETE FROM workspaces WHERE id = $1;

-- name: CheckSlugExists :one
SELECT EXISTS(SELECT 1 FROM workspaces WHERE slug = $1);

-- name: CheckSlugExistsExcluding :one
SELECT EXISTS(SELECT 1 FROM workspaces WHERE slug = $1 AND id != $2);

-- =========================
-- WORKSPACE ROLES
-- =========================

-- name: CreateWorkspaceRole :one
INSERT INTO workspace_roles (workspace_id, name, display_name, description, color, icon, hierarchy_level, is_system, is_default, permissions)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
RETURNING *;

-- name: GetWorkspaceRole :one
SELECT * FROM workspace_roles WHERE id = $1 AND workspace_id = $2;

-- name: GetWorkspaceRoleByName :one
SELECT * FROM workspace_roles WHERE workspace_id = $1 AND name = $2;

-- name: GetDefaultWorkspaceRole :one
SELECT * FROM workspace_roles WHERE workspace_id = $1 AND is_default = TRUE LIMIT 1;

-- name: ListWorkspaceRoles :many
SELECT 
    wr.*,
    (SELECT COUNT(*) FROM workspace_members WHERE role_id = wr.id) as member_count
FROM workspace_roles wr
WHERE wr.workspace_id = $1
ORDER BY wr.hierarchy_level DESC;

-- name: UpdateWorkspaceRole :one
UPDATE workspace_roles
SET 
    display_name = COALESCE(sqlc.narg('display_name'), display_name),
    description = COALESCE(sqlc.narg('description'), description),
    color = COALESCE(sqlc.narg('color'), color),
    icon = COALESCE(sqlc.narg('icon'), icon),
    hierarchy_level = COALESCE(sqlc.narg('hierarchy_level'), hierarchy_level),
    is_default = COALESCE(sqlc.narg('is_default'), is_default),
    permissions = COALESCE(sqlc.narg('permissions'), permissions),
    updated_at = NOW()
WHERE id = $1 AND workspace_id = $2 AND is_system = FALSE
RETURNING *;

-- name: DeleteWorkspaceRole :exec
DELETE FROM workspace_roles WHERE id = $1 AND workspace_id = $2 AND is_system = FALSE;

-- name: ReassignRoleMembers :exec
UPDATE workspace_members
SET role_id = $2, role_name = $3, updated_at = NOW()
WHERE role_id = $1;

-- =========================
-- WORKSPACE MEMBERS
-- =========================

-- name: CreateWorkspaceMember :one
INSERT INTO workspace_members (workspace_id, user_id, role_id, role_name, status, invited_by, invited_at, joined_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: GetWorkspaceMember :one
SELECT 
    wm.*,
    wr.display_name as role_display_name,
    wr.color as role_color,
    wr.hierarchy_level,
    wr.permissions
FROM workspace_members wm
LEFT JOIN workspace_roles wr ON wm.role_id = wr.id
WHERE wm.workspace_id = $1 AND wm.user_id = $2;

-- name: ListWorkspaceMembers :many
SELECT 
    wm.*,
    wr.display_name as role_display_name,
    wr.color as role_color,
    wr.hierarchy_level
FROM workspace_members wm
LEFT JOIN workspace_roles wr ON wm.role_id = wr.id
WHERE wm.workspace_id = $1
ORDER BY wr.hierarchy_level DESC, wm.joined_at;

-- name: ListWorkspaceMembersByStatus :many
SELECT 
    wm.*,
    wr.display_name as role_display_name,
    wr.color as role_color,
    wr.hierarchy_level
FROM workspace_members wm
LEFT JOIN workspace_roles wr ON wm.role_id = wr.id
WHERE wm.workspace_id = $1 AND wm.status = $2
ORDER BY wr.hierarchy_level DESC, wm.joined_at;

-- name: UpdateWorkspaceMemberRole :one
UPDATE workspace_members
SET role_id = $3, role_name = $4, updated_at = NOW()
WHERE workspace_id = $1 AND user_id = $2
RETURNING *;

-- name: UpdateWorkspaceMemberStatus :exec
UPDATE workspace_members
SET status = $3, joined_at = CASE WHEN $3 = 'active' THEN NOW() ELSE joined_at END, updated_at = NOW()
WHERE workspace_id = $1 AND user_id = $2;

-- name: DeleteWorkspaceMember :exec
DELETE FROM workspace_members WHERE workspace_id = $1 AND user_id = $2;

-- name: CountWorkspaceMembers :one
SELECT COUNT(*) FROM workspace_members WHERE workspace_id = $1 AND status = 'active';

-- name: CheckUserIsWorkspaceMember :one
SELECT EXISTS(SELECT 1 FROM workspace_members WHERE workspace_id = $1 AND user_id = $2 AND status = 'active');

-- name: CheckUserIsWorkspaceOwner :one
SELECT EXISTS(SELECT 1 FROM workspaces WHERE id = $1 AND owner_id = $2);

-- =========================
-- WORKSPACE INVITATIONS
-- =========================

-- name: CreateWorkspaceInvitation :one
INSERT INTO workspace_invites (workspace_id, email, token, role, invited_by, expires_at)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: GetWorkspaceInvitationByToken :one
SELECT
    wi.id, wi.workspace_id, wi.email, wi.role, wi.invited_by, wi.token, wi.status, wi.expires_at, wi.accepted_at, wi.created_at, wi.updated_at,
    w.name as workspace_name,
    w.slug as workspace_slug,
    w.logo_url as workspace_logo
FROM workspace_invites wi
JOIN workspaces w ON wi.workspace_id = w.id
WHERE wi.token = $1;

-- name: GetWorkspaceInvitation :one
SELECT * FROM workspace_invites WHERE id = $1 AND workspace_id = $2;

-- name: ListWorkspaceInvitations :many
SELECT * FROM workspace_invites
WHERE workspace_id = $1
ORDER BY created_at DESC;

-- name: ListWorkspaceInvitationsByStatus :many
SELECT * FROM workspace_invites
WHERE workspace_id = $1 AND status = $2
ORDER BY created_at DESC;

-- name: UpdateInvitationStatus :exec
UPDATE workspace_invites
SET status = $2, updated_at = NOW()
WHERE id = $1;

-- name: AcceptInvitation :exec
UPDATE workspace_invites
SET status = 'accepted', accepted_at = NOW(), updated_at = NOW()
WHERE id = $1;

-- name: RevokeInvitation :exec
UPDATE workspace_invites
SET status = 'revoked', updated_at = NOW()
WHERE id = $1 AND workspace_id = $2 AND status = 'pending';

-- name: UpdateInvitationToken :one
UPDATE workspace_invites
SET token = $2, expires_at = $3, updated_at = NOW()
WHERE id = $1 AND status = 'pending'
RETURNING *;

-- name: CheckPendingInvitationExists :one
SELECT EXISTS(SELECT 1 FROM workspace_invites WHERE workspace_id = $1 AND email = $2 AND status = 'pending');

-- =========================
-- WORKSPACE MEMORIES
-- =========================

-- name: CreateWorkspaceMemory :one
INSERT INTO workspace_memories (workspace_id, title, summary, content, memory_type, category, scope_type, scope_id, visibility, created_by, importance_score, tags, metadata, is_pinned)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
RETURNING *;

-- name: GetWorkspaceMemory :one
SELECT * FROM workspace_memories WHERE id = $1 AND workspace_id = $2;

-- name: ListWorkspaceMemories :many
SELECT wm.*
FROM workspace_memories wm
WHERE wm.workspace_id = $1 AND wm.is_active = TRUE
ORDER BY wm.is_pinned DESC, wm.importance_score DESC, wm.created_at DESC
LIMIT $2 OFFSET $3;

-- name: ListWorkspaceMemoriesByType :many
SELECT wm.*
FROM workspace_memories wm
WHERE wm.workspace_id = $1 AND wm.memory_type = $2 AND wm.is_active = TRUE
ORDER BY wm.is_pinned DESC, wm.importance_score DESC, wm.created_at DESC
LIMIT $3 OFFSET $4;

-- name: ListWorkspaceMemoriesByScope :many
SELECT wm.*
FROM workspace_memories wm
WHERE wm.workspace_id = $1 AND wm.scope_type = $2 AND wm.scope_id = $3 AND wm.is_active = TRUE
ORDER BY wm.is_pinned DESC, wm.importance_score DESC, wm.created_at DESC
LIMIT $4 OFFSET $5;

-- name: SearchWorkspaceMemories :many
SELECT wm.*
FROM workspace_memories wm
WHERE wm.workspace_id = $1 
  AND wm.is_active = TRUE
  AND (wm.title ILIKE '%' || $2 || '%' OR wm.summary ILIKE '%' || $2 || '%' OR wm.content ILIKE '%' || $2 || '%')
ORDER BY wm.is_pinned DESC, wm.importance_score DESC
LIMIT $3;

-- name: UpdateWorkspaceMemory :one
UPDATE workspace_memories
SET 
    title = COALESCE(sqlc.narg('title'), title),
    summary = COALESCE(sqlc.narg('summary'), summary),
    content = COALESCE(sqlc.narg('content'), content),
    memory_type = COALESCE(sqlc.narg('memory_type'), memory_type),
    category = COALESCE(sqlc.narg('category'), category),
    visibility = COALESCE(sqlc.narg('visibility'), visibility),
    importance_score = COALESCE(sqlc.narg('importance_score'), importance_score),
    tags = COALESCE(sqlc.narg('tags'), tags),
    metadata = COALESCE(sqlc.narg('metadata'), metadata),
    is_pinned = COALESCE(sqlc.narg('is_pinned'), is_pinned),
    updated_at = NOW()
WHERE id = $1 AND workspace_id = $2
RETURNING *;

-- name: DeleteWorkspaceMemory :exec
UPDATE workspace_memories SET is_active = FALSE, updated_at = NOW() WHERE id = $1 AND workspace_id = $2;

-- name: IncrementMemoryAccessCount :exec
UPDATE workspace_memories SET access_count = access_count + 1 WHERE id = $1;

-- =========================
-- USER WORKSPACE PROFILES
-- =========================

-- name: CreateUserWorkspaceProfile :one
INSERT INTO user_workspace_profiles (workspace_id, user_id, display_name, title, department, avatar_url, work_email, phone, timezone, working_hours, notification_preferences, expertise_areas)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
RETURNING *;

-- name: GetUserWorkspaceProfile :one
SELECT * FROM user_workspace_profiles WHERE workspace_id = $1 AND user_id = $2;

-- name: UpdateUserWorkspaceProfile :one
UPDATE user_workspace_profiles
SET 
    display_name = COALESCE(sqlc.narg('display_name'), display_name),
    title = COALESCE(sqlc.narg('title'), title),
    department = COALESCE(sqlc.narg('department'), department),
    avatar_url = COALESCE(sqlc.narg('avatar_url'), avatar_url),
    work_email = COALESCE(sqlc.narg('work_email'), work_email),
    phone = COALESCE(sqlc.narg('phone'), phone),
    timezone = COALESCE(sqlc.narg('timezone'), timezone),
    working_hours = COALESCE(sqlc.narg('working_hours'), working_hours),
    notification_preferences = COALESCE(sqlc.narg('notification_preferences'), notification_preferences),
    expertise_areas = COALESCE(sqlc.narg('expertise_areas'), expertise_areas),
    updated_at = NOW()
WHERE workspace_id = $1 AND user_id = $2
RETURNING *;

-- name: UpsertUserWorkspaceProfile :one
INSERT INTO user_workspace_profiles (workspace_id, user_id, display_name, title, department, avatar_url, work_email, phone, timezone, working_hours, notification_preferences, expertise_areas)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
ON CONFLICT (workspace_id, user_id) DO UPDATE
SET 
    display_name = COALESCE(EXCLUDED.display_name, user_workspace_profiles.display_name),
    title = COALESCE(EXCLUDED.title, user_workspace_profiles.title),
    department = COALESCE(EXCLUDED.department, user_workspace_profiles.department),
    avatar_url = COALESCE(EXCLUDED.avatar_url, user_workspace_profiles.avatar_url),
    work_email = COALESCE(EXCLUDED.work_email, user_workspace_profiles.work_email),
    phone = COALESCE(EXCLUDED.phone, user_workspace_profiles.phone),
    timezone = COALESCE(EXCLUDED.timezone, user_workspace_profiles.timezone),
    working_hours = COALESCE(EXCLUDED.working_hours, user_workspace_profiles.working_hours),
    notification_preferences = COALESCE(EXCLUDED.notification_preferences, user_workspace_profiles.notification_preferences),
    expertise_areas = COALESCE(EXCLUDED.expertise_areas, user_workspace_profiles.expertise_areas),
    updated_at = NOW()
RETURNING *;

-- =========================
-- WORKSPACE PROJECT MEMBERS (different table from project_members)
-- =========================

-- name: AddWorkspaceProjectMember :one
INSERT INTO workspace_project_members (project_id, user_id, workspace_id, project_role, assigned_by, notification_level)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: GetWorkspaceProjectMember :one
SELECT * FROM workspace_project_members WHERE project_id = $1 AND user_id = $2;

-- name: ListWorkspaceProjectMembers :many
SELECT wpm.*
FROM workspace_project_members wpm
WHERE wpm.project_id = $1
ORDER BY 
    CASE wpm.project_role 
        WHEN 'lead' THEN 1 
        WHEN 'contributor' THEN 2 
        WHEN 'reviewer' THEN 3 
        WHEN 'viewer' THEN 4 
    END,
    wpm.assigned_at;

-- name: UpdateWorkspaceProjectMemberRole :one
UPDATE workspace_project_members
SET project_role = $3, notification_level = COALESCE(sqlc.narg('notification_level'), notification_level), updated_at = NOW()
WHERE project_id = $1 AND user_id = $2
RETURNING *;

-- name: RemoveWorkspaceProjectMember :exec
DELETE FROM workspace_project_members WHERE project_id = $1 AND user_id = $2;

-- name: CheckUserIsWorkspaceProjectMember :one
SELECT EXISTS(SELECT 1 FROM workspace_project_members WHERE project_id = $1 AND user_id = $2);

-- name: ListUserWorkspaceProjectAssignments :many
SELECT wpm.project_id, wpm.project_role, wpm.assigned_at, wpm.notification_level
FROM workspace_project_members wpm
WHERE wpm.workspace_id = $1 AND wpm.user_id = $2
ORDER BY wpm.assigned_at DESC;

-- name: GetWorkspaceRolePermissions :one
SELECT permissions FROM workspace_roles WHERE id = $1 AND workspace_id = $2;

-- name: GetUserWorkspacePermissions :one
SELECT wr.permissions
FROM workspace_members wm
JOIN workspace_roles wr ON wm.role_id = wr.id
WHERE wm.workspace_id = $1 AND wm.user_id = $2;
