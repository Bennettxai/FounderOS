package handlers

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rhl/businessos-backend/internal/middleware"
	"github.com/rhl/businessos-backend/internal/services/templates"
	"github.com/rhl/businessos-backend/internal/utils"
)

// OrganizationHandler owns the /api/organizations domain — the account level of
// the hierarchy (Organization → Workspace → Team). Org membership is account-wide;
// workspace access is granular (handled by the workspace domain).
type OrganizationHandler struct {
	pool *pgxpool.Pool
}

func NewOrganizationHandler(pool *pgxpool.Pool) *OrganizationHandler {
	return &OrganizationHandler{pool: pool}
}

func (h *Handlers) registerOrganizationRoutes(api *gin.RouterGroup, auth gin.HandlerFunc) {
	oh := NewOrganizationHandler(h.pool)
	g := api.Group("/organizations")
	g.Use(auth, middleware.RequireAuth())
	{
		g.POST("", oh.CreateOrganization)
		g.GET("", oh.ListMyOrganizations)

		// Public-to-members invite accept/validate (no org scope yet).
		g.POST("/invites/accept", oh.AcceptInvite)
		g.POST("/invites/validate", oh.ValidateInvite)

		g.GET("/:orgId", oh.GetOrganization)
		g.PUT("/:orgId", oh.UpdateOrganization)
		g.GET("/:orgId/members", oh.ListMembers)
		g.PUT("/:orgId/members/:userId", oh.UpdateMemberRole)
		g.DELETE("/:orgId/members/:userId", oh.RemoveMember)
		g.GET("/:orgId/workspaces", oh.ListOrgWorkspaces)
		g.POST("/:orgId/invites", oh.CreateInvite)
		g.GET("/:orgId/invites", oh.ListInvites)
		g.DELETE("/:orgId/invites/:inviteId", oh.RevokeInvite)
	}
}

var orgSlugRe = regexp.MustCompile(`[^a-z0-9]+`)

func slugify(name string) string {
	s := strings.ToLower(strings.TrimSpace(name))
	s = orgSlugRe.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	if s == "" {
		s = "org"
	}
	if len(s) > 40 {
		s = s[:40]
	}
	return s
}

func randToken() string {
	b := make([]byte, 24)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// orgRole returns the caller's role in the org, or "" if not a member.
func (h *OrganizationHandler) orgRole(ctx context.Context, orgID, userID string) string {
	var role string
	err := h.pool.QueryRow(ctx,
		`SELECT role FROM organization_members WHERE organization_id=$1 AND user_id=$2 AND status='active'`,
		orgID, userID).Scan(&role)
	if err != nil {
		return ""
	}
	return role
}

func isOrgManager(role string) bool { return role == "owner" || role == "admin" }

// CreateOrganization — POST /api/organizations
func (h *OrganizationHandler) CreateOrganization(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}
	var req struct {
		Name string `json:"name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondInvalidRequest(c, slog.Default(), err)
		return
	}

	slug := slugify(req.Name) + "-" + hex.EncodeToString([]byte(randToken()))[:6]
	var orgID string
	err := h.pool.QueryRow(c.Request.Context(),
		`INSERT INTO organizations (name, slug, owner_id) VALUES ($1, $2, $3) RETURNING id`,
		req.Name, slug, user.ID).Scan(&orgID)
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "create organization", err)
		return
	}
	_, err = h.pool.Exec(c.Request.Context(),
		`INSERT INTO organization_members (organization_id, user_id, role, status) VALUES ($1, $2, 'owner', 'active')
		 ON CONFLICT (organization_id, user_id) DO NOTHING`, orgID, user.ID)
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "add owner", err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": orgID, "name": req.Name, "slug": slug, "role": "owner"})
}

// ListMyOrganizations — GET /api/organizations
func (h *OrganizationHandler) ListMyOrganizations(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}
	rows, err := h.pool.Query(c.Request.Context(),
		`SELECT o.id, o.name, o.slug, COALESCE(o.logo_url,''), m.role,
		        (SELECT count(*) FROM organization_members m2 WHERE m2.organization_id=o.id AND m2.status='active'),
		        (SELECT count(*) FROM workspaces w WHERE w.organization_id=o.id)
		 FROM organizations o
		 JOIN organization_members m ON m.organization_id = o.id
		 WHERE m.user_id=$1 AND m.status='active'
		 ORDER BY o.created_at`, user.ID)
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "list orgs", err)
		return
	}
	defer rows.Close()
	orgs := []gin.H{}
	for rows.Next() {
		var id, name, slug, logo, role string
		var members, workspaces int
		if err := rows.Scan(&id, &name, &slug, &logo, &role, &members, &workspaces); err != nil {
			continue
		}
		orgs = append(orgs, gin.H{"id": id, "name": name, "slug": slug, "logo_url": logo, "role": role, "member_count": members, "workspace_count": workspaces})
	}
	c.JSON(http.StatusOK, gin.H{"organizations": orgs})
}

// GetOrganization — GET /api/organizations/:orgId
func (h *OrganizationHandler) GetOrganization(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	orgID := c.Param("orgId")
	role := h.orgRole(c.Request.Context(), orgID, user.ID)
	if role == "" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Not a member of this organization"})
		return
	}
	var name, slug, logo, ownerID string
	err := h.pool.QueryRow(c.Request.Context(),
		`SELECT name, slug, COALESCE(logo_url,''), COALESCE(owner_id,'') FROM organizations WHERE id=$1`, orgID).
		Scan(&name, &slug, &logo, &ownerID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Organization not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": orgID, "name": name, "slug": slug, "logo_url": logo, "owner_id": ownerID, "role": role})
}

// UpdateOrganization — PUT /api/organizations/:orgId
func (h *OrganizationHandler) UpdateOrganization(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	orgID := c.Param("orgId")
	if !isOrgManager(h.orgRole(c.Request.Context(), orgID, user.ID)) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only owners and admins can update the organization"})
		return
	}
	var req struct {
		Name    *string `json:"name"`
		LogoURL *string `json:"logo_url"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondInvalidRequest(c, slog.Default(), err)
		return
	}
	_, err := h.pool.Exec(c.Request.Context(),
		`UPDATE organizations SET name=COALESCE($2,name), logo_url=COALESCE($3,logo_url), updated_at=NOW() WHERE id=$1`,
		orgID, req.Name, req.LogoURL)
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "update org", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "updated"})
}

// ListMembers — GET /api/organizations/:orgId/members
func (h *OrganizationHandler) ListMembers(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	orgID := c.Param("orgId")
	if h.orgRole(c.Request.Context(), orgID, user.ID) == "" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Not a member of this organization"})
		return
	}
	rows, err := h.pool.Query(c.Request.Context(),
		`SELECT m.user_id, m.role, m.status, m.joined_at, u.name, u.email, u.image
		 FROM organization_members m LEFT JOIN "user" u ON u.id = m.user_id
		 WHERE m.organization_id=$1 ORDER BY m.joined_at`, orgID)
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "list members", err)
		return
	}
	defer rows.Close()
	members := []gin.H{}
	for rows.Next() {
		var uid, role, status string
		var joinedAt *time.Time
		var name, email, image *string
		if err := rows.Scan(&uid, &role, &status, &joinedAt, &name, &email, &image); err != nil {
			continue
		}
		members = append(members, gin.H{"user_id": uid, "role": role, "status": status, "joined_at": joinedAt, "name": name, "email": email, "image": image})
	}
	c.JSON(http.StatusOK, gin.H{"members": members})
}

// UpdateMemberRole — PUT /api/organizations/:orgId/members/:userId
func (h *OrganizationHandler) UpdateMemberRole(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	orgID := c.Param("orgId")
	if !isOrgManager(h.orgRole(c.Request.Context(), orgID, user.ID)) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only owners and admins can change roles"})
		return
	}
	var req struct {
		Role string `json:"role" binding:"required,oneof=owner admin member"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondInvalidRequest(c, slog.Default(), err)
		return
	}
	_, err := h.pool.Exec(c.Request.Context(),
		`UPDATE organization_members SET role=$3, updated_at=NOW() WHERE organization_id=$1 AND user_id=$2`,
		orgID, c.Param("userId"), req.Role)
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "update role", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "updated"})
}

// RemoveMember — DELETE /api/organizations/:orgId/members/:userId
func (h *OrganizationHandler) RemoveMember(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	orgID := c.Param("orgId")
	if !isOrgManager(h.orgRole(c.Request.Context(), orgID, user.ID)) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only owners and admins can remove members"})
		return
	}
	target := c.Param("userId")
	// Don't allow removing the org owner.
	var ownerID string
	_ = h.pool.QueryRow(c.Request.Context(), `SELECT COALESCE(owner_id,'') FROM organizations WHERE id=$1`, orgID).Scan(&ownerID)
	if target == ownerID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot remove the organization owner"})
		return
	}
	_, err := h.pool.Exec(c.Request.Context(),
		`DELETE FROM organization_members WHERE organization_id=$1 AND user_id=$2`, orgID, target)
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "remove member", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "removed"})
}

// ListOrgWorkspaces — GET /api/organizations/:orgId/workspaces
// Returns workspaces in the org that the caller can access (granular).
// Org admins/owners see all; members see only ones they belong to.
func (h *OrganizationHandler) ListOrgWorkspaces(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	orgID := c.Param("orgId")
	role := h.orgRole(c.Request.Context(), orgID, user.ID)
	if role == "" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Not a member of this organization"})
		return
	}
	q := `SELECT w.id, w.name, w.slug FROM workspaces w WHERE w.organization_id=$1`
	args := []interface{}{orgID}
	if !isOrgManager(role) {
		q += ` AND w.id IN (SELECT workspace_id FROM workspace_members WHERE user_id=$2 AND status='active')`
		args = append(args, user.ID)
	}
	q += ` ORDER BY w.created_at`
	rows, err := h.pool.Query(c.Request.Context(), q, args...)
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "list org workspaces", err)
		return
	}
	defer rows.Close()
	ws := []gin.H{}
	for rows.Next() {
		var id, name, slug string
		if err := rows.Scan(&id, &name, &slug); err != nil {
			continue
		}
		ws = append(ws, gin.H{"id": id, "name": name, "slug": slug})
	}
	c.JSON(http.StatusOK, gin.H{"workspaces": ws})
}

// CreateInvite — POST /api/organizations/:orgId/invites
func (h *OrganizationHandler) CreateInvite(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	orgID := c.Param("orgId")
	if !isOrgManager(h.orgRole(c.Request.Context(), orgID, user.ID)) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only owners and admins can invite"})
		return
	}
	var req struct {
		Email string `json:"email" binding:"required,email"`
		Role  string `json:"role" binding:"required,oneof=admin member"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondInvalidRequest(c, slog.Default(), err)
		return
	}
	email := strings.ToLower(strings.TrimSpace(req.Email))
	token := randToken()
	var inviteID string
	err := h.pool.QueryRow(c.Request.Context(),
		`INSERT INTO organization_invites (organization_id, email, role, token, invited_by, expires_at)
		 VALUES ($1,$2,$3,$4,$5,$6) RETURNING id`,
		orgID, email, req.Role, token, user.ID, time.Now().Add(7*24*time.Hour)).Scan(&inviteID)
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "create invite", err)
		return
	}

	// Best-effort email.
	var orgName string
	_ = h.pool.QueryRow(c.Request.Context(), `SELECT name FROM organizations WHERE id=$1`, orgID).Scan(&orgName)
	appURL := os.Getenv("APP_URL")
	if appURL == "" {
		appURL = "https://app.businessos.dev"
	}
	link := appURL + "/invite/org/" + token
	inviter := user.Name
	if inviter == "" {
		inviter = user.Email
	}
	go func() {
		emailSvc := templates.NewEmailTemplateService()
		if emailSvc.IsEnabled() {
			_ = emailSvc.SendOrganizationInvitation(context.Background(), email, templates.OrganizationInvitationData{
				InviterName: inviter, InviterEmail: user.Email, OrganizationName: orgName,
				Role: req.Role, InvitationLink: link, ExpiresIn: "7 days",
			})
		}
	}()

	c.JSON(http.StatusCreated, gin.H{"id": inviteID, "email": email, "role": req.Role, "status": "pending"})
}

// ListInvites — GET /api/organizations/:orgId/invites
func (h *OrganizationHandler) ListInvites(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	orgID := c.Param("orgId")
	if !isOrgManager(h.orgRole(c.Request.Context(), orgID, user.ID)) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only owners and admins can view invites"})
		return
	}
	rows, err := h.pool.Query(c.Request.Context(),
		`SELECT id, email, role, status, expires_at, created_at FROM organization_invites
		 WHERE organization_id=$1 AND status='pending' ORDER BY created_at DESC`, orgID)
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "list invites", err)
		return
	}
	defer rows.Close()
	invites := []gin.H{}
	for rows.Next() {
		var id, email, role, status string
		var exp, created time.Time
		if err := rows.Scan(&id, &email, &role, &status, &exp, &created); err != nil {
			continue
		}
		invites = append(invites, gin.H{"id": id, "email": email, "role": role, "status": status, "expires_at": exp, "created_at": created})
	}
	c.JSON(http.StatusOK, gin.H{"invites": invites})
}

// RevokeInvite — DELETE /api/organizations/:orgId/invites/:inviteId
func (h *OrganizationHandler) RevokeInvite(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	orgID := c.Param("orgId")
	if !isOrgManager(h.orgRole(c.Request.Context(), orgID, user.ID)) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only owners and admins can revoke invites"})
		return
	}
	_, err := h.pool.Exec(c.Request.Context(),
		`UPDATE organization_invites SET status='revoked' WHERE id=$1 AND organization_id=$2`,
		c.Param("inviteId"), orgID)
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "revoke invite", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "revoked"})
}

// ValidateInvite — POST /api/organizations/invites/validate {token}
func (h *OrganizationHandler) ValidateInvite(c *gin.Context) {
	var req struct {
		Token string `json:"token" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondInvalidRequest(c, slog.Default(), err)
		return
	}
	var orgID, orgName, email, role string
	var exp time.Time
	err := h.pool.QueryRow(c.Request.Context(),
		`SELECT oi.organization_id, o.name, oi.email, oi.role, oi.expires_at
		 FROM organization_invites oi JOIN organizations o ON o.id=oi.organization_id
		 WHERE oi.token=$1 AND oi.status='pending'`, req.Token).Scan(&orgID, &orgName, &email, &role, &exp)
	if err != nil || time.Now().After(exp) {
		c.JSON(http.StatusOK, gin.H{"valid": false, "error": "This invitation is invalid or has expired"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"valid": true, "organization_id": orgID, "organization_name": orgName, "email": email, "role": role})
}

// AcceptInvite — POST /api/organizations/invites/accept {token}
func (h *OrganizationHandler) AcceptInvite(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}
	var req struct {
		Token string `json:"token" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondInvalidRequest(c, slog.Default(), err)
		return
	}
	var inviteID, orgID, inviteEmail, role string
	var exp time.Time
	err := h.pool.QueryRow(c.Request.Context(),
		`SELECT id, organization_id, email, role, expires_at FROM organization_invites WHERE token=$1 AND status='pending'`,
		req.Token).Scan(&inviteID, &orgID, &inviteEmail, &role, &exp)
	if err != nil || time.Now().After(exp) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "This invitation is invalid or has expired"})
		return
	}
	// SECURITY: only the intended recipient may accept.
	if !strings.EqualFold(user.Email, inviteEmail) {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "This invitation was sent to a different email address. Please sign in with the invited address.",
			"code":  "EMAIL_MISMATCH",
		})
		return
	}
	// ON CONFLICT: restore the correct invited role, not just flip status — a
	// previously-removed member re-invited with a different role must get the new role.
	_, err = h.pool.Exec(c.Request.Context(),
		`INSERT INTO organization_members (organization_id, user_id, role, status, invited_by)
		 VALUES ($1,$2,$3,'active',NULL)
		 ON CONFLICT (organization_id, user_id) DO UPDATE SET role=EXCLUDED.role, status='active'`,
		orgID, user.ID, role)
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "accept invite", err)
		return
	}
	_, _ = h.pool.Exec(c.Request.Context(),
		`UPDATE organization_invites SET status='accepted', accepted_at=NOW() WHERE id=$1`, inviteID)
	c.JSON(http.StatusOK, gin.H{"message": "joined", "organization_id": orgID, "role": role})
}
