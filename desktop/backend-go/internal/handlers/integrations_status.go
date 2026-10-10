package handlers

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rhl/businessos-backend/internal/middleware"
	"github.com/rhl/businessos-backend/internal/utils"
)

// ============================================================================
// Aggregated Status Endpoint
// ============================================================================

// IntegrationStatusInfo represents the status of a single integration.
type IntegrationStatusInfo struct {
	ProviderID    string      `json:"provider_id"`
	ProviderName  string      `json:"provider_name"`
	Category      string      `json:"category"`
	Connected     bool        `json:"connected"`
	Status        string      `json:"status"`
	AccountName   *string     `json:"account_name,omitempty"`
	WorkspaceName *string     `json:"workspace_name,omitempty"`
	ConnectedAt   interface{} `json:"connected_at,omitempty"`
	IconURL       *string     `json:"icon_url,omitempty"`
}

// GetAllIntegrationsStatus handles GET /api/integrations/status
// Returns aggregated status of all integrations from both legacy OAuth tables
// and the new user_integrations table.
func (h *IntegrationsHandler) GetAllIntegrationsStatus(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}
	userID := user.ID
	if userID == "" {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	ctx := c.Request.Context()
	statusMap := make(map[string]*IntegrationStatusInfo)

	// 1. Check Google OAuth (legacy table)
	var googleStatus struct {
		Email       *string     `json:"email"`
		ConnectedAt interface{} `json:"connected_at"`
	}
	err := h.pool.QueryRow(ctx, `
		SELECT google_email, created_at FROM google_oauth_tokens WHERE user_id = $1
	`, userID).Scan(&googleStatus.Email, &googleStatus.ConnectedAt)
	if err == nil {
		statusMap["google_calendar"] = &IntegrationStatusInfo{
			ProviderID:   "google_calendar",
			ProviderName: "Google Calendar",
			Category:     "calendar",
			Connected:    true,
			Status:       "connected",
			AccountName:  googleStatus.Email,
			ConnectedAt:  googleStatus.ConnectedAt,
		}
	}

	// 2. Check Slack OAuth (legacy table)
	var slackStatus struct {
		TeamName    *string     `json:"team_name"`
		ConnectedAt interface{} `json:"connected_at"`
	}
	err = h.pool.QueryRow(ctx, `
		SELECT team_name, created_at FROM slack_oauth_tokens WHERE user_id = $1
	`, userID).Scan(&slackStatus.TeamName, &slackStatus.ConnectedAt)
	if err == nil {
		statusMap["slack"] = &IntegrationStatusInfo{
			ProviderID:    "slack",
			ProviderName:  "Slack",
			Category:      "communication",
			Connected:     true,
			Status:        "connected",
			WorkspaceName: slackStatus.TeamName,
			ConnectedAt:   slackStatus.ConnectedAt,
		}
	}

	// 3. Check Notion OAuth (legacy table)
	var notionStatus struct {
		WorkspaceName *string     `json:"workspace_name"`
		ConnectedAt   interface{} `json:"connected_at"`
	}
	err = h.pool.QueryRow(ctx, `
		SELECT workspace_name, created_at FROM notion_oauth_tokens WHERE user_id = $1
	`, userID).Scan(&notionStatus.WorkspaceName, &notionStatus.ConnectedAt)
	if err == nil {
		statusMap["notion"] = &IntegrationStatusInfo{
			ProviderID:    "notion",
			ProviderName:  "Notion",
			Category:      "storage",
			Connected:     true,
			Status:        "connected",
			WorkspaceName: notionStatus.WorkspaceName,
			ConnectedAt:   notionStatus.ConnectedAt,
		}
	}

	// 4. Get all from user_integrations table (new system)
	rows, err := h.pool.Query(ctx, `
		SELECT ui.provider_id, ui.status, ui.connected_at,
		       ui.external_account_name, ui.external_workspace_name,
		       ip.name, ip.category, ip.icon_url
		FROM user_integrations ui
		JOIN integration_providers ip ON ui.provider_id = ip.id
		WHERE ui.user_id = $1
	`, userID)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var info struct {
				ProviderID    string
				Status        string
				ConnectedAt   interface{}
				AccountName   *string
				WorkspaceName *string
				Name          string
				Category      string
				IconURL       *string
			}
			if err := rows.Scan(&info.ProviderID, &info.Status, &info.ConnectedAt,
				&info.AccountName, &info.WorkspaceName,
				&info.Name, &info.Category, &info.IconURL); err != nil {
				continue
			}
			// Override legacy status if exists in new table
			statusMap[info.ProviderID] = &IntegrationStatusInfo{
				ProviderID:    info.ProviderID,
				ProviderName:  info.Name,
				Category:      info.Category,
				Connected:     info.Status == "connected",
				Status:        info.Status,
				AccountName:   info.AccountName,
				WorkspaceName: info.WorkspaceName,
				ConnectedAt:   info.ConnectedAt,
				IconURL:       info.IconURL,
			}
		}
	}

	// 5. Get all available providers and mark unconnected ones
	providerRows, err := h.pool.Query(ctx, `
		SELECT id, name, category, icon_url, status
		FROM integration_providers
		WHERE status != 'deprecated'
		ORDER BY category, name
	`)
	if err == nil {
		defer providerRows.Close()
		for providerRows.Next() {
			var p struct {
				ID       string
				Name     string
				Category string
				IconURL  *string
				Status   string
			}
			if err := providerRows.Scan(&p.ID, &p.Name, &p.Category, &p.IconURL, &p.Status); err != nil {
				continue
			}
			// Only add if not already in statusMap
			if _, exists := statusMap[p.ID]; !exists {
				statusMap[p.ID] = &IntegrationStatusInfo{
					ProviderID:   p.ID,
					ProviderName: p.Name,
					Category:     p.Category,
					Connected:    false,
					Status:       p.Status, // available, coming_soon, etc.
					IconURL:      p.IconURL,
				}
			} else {
				// Update icon_url if available
				if statusMap[p.ID].IconURL == nil {
					statusMap[p.ID].IconURL = p.IconURL
				}
			}
		}
	}

	// Convert map to slice grouped by category
	categorized := make(map[string][]IntegrationStatusInfo)
	var allIntegrations []IntegrationStatusInfo
	for _, info := range statusMap {
		categorized[info.Category] = append(categorized[info.Category], *info)
		allIntegrations = append(allIntegrations, *info)
	}

	// Count connected
	connectedCount := 0
	for _, info := range allIntegrations {
		if info.Connected {
			connectedCount++
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success":         true,
		"integrations":    allIntegrations,
		"by_category":     categorized,
		"connected_count": connectedCount,
		"total_count":     len(allIntegrations),
	})
}
