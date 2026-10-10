package handlers

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/rhl/businessos-backend/internal/middleware"
	"github.com/rhl/businessos-backend/internal/services"
)

type createMIOSASandboxRequest struct {
	WorkspaceID string `json:"workspace_id"`
	Name        string `json:"name"`
	Cols        int    `json:"cols"`
	Rows        int    `json:"rows"`
	Shell       string `json:"shell"`
}

func (h *MIOSAHandler) CreateSandbox(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req createMIOSASandboxRequest
	_ = c.ShouldBindJSON(&req)

	workspaceID := strings.TrimSpace(req.WorkspaceID)
	if workspaceID == "" {
		workspaceID = strings.TrimSpace(c.GetHeader("X-Workspace-ID"))
	}
	if workspaceID == "" {
		var err error
		workspaceID, err = h.defaultWorkspaceID(c, user.ID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "workspace_id is required"})
			return
		}
	}

	if !h.userCanAccessWorkspace(c, user.ID, workspaceID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "workspace access required"})
		return
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		name = fmt.Sprintf("BusinessOS sandbox for %s", user.Email)
	}

	session, err := services.NewMIOSAPlatformService(h.pool, h.cfg).CreateSandboxTerminalSession(c.Request.Context(), services.CreateMIOSASandboxInput{
		WorkspaceID: workspaceID,
		UserID:      user.ID,
		UserEmail:   user.Email,
		Name:        name,
		Cols:        req.Cols,
		Rows:        req.Rows,
		Shell:       req.Shell,
	})
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "failed to create MIOSA sandbox", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, session)
}

func (h *MIOSAHandler) defaultWorkspaceID(c *gin.Context, userID string) (string, error) {
	var workspaceID string
	err := h.pool.QueryRow(c.Request.Context(), `
		SELECT workspace_id::text
		FROM workspace_members
		WHERE user_id = $1
			AND COALESCE(status, 'active') = 'active'
		ORDER BY joined_at ASC NULLS LAST, created_at ASC NULLS LAST
		LIMIT 1
	`, userID).Scan(&workspaceID)
	return workspaceID, err
}

func (h *MIOSAHandler) userCanAccessWorkspace(c *gin.Context, userID, workspaceID string) bool {
	var ok bool
	err := h.pool.QueryRow(c.Request.Context(), `
		SELECT EXISTS (
			SELECT 1
			FROM workspace_members
			WHERE workspace_id = $1
				AND user_id = $2
				AND COALESCE(status, 'active') = 'active'
		)
	`, workspaceID, userID).Scan(&ok)
	return err == nil && ok
}
