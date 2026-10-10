package handlers

import (
	"net/http"

	"github.com/rhl/businessos-backend/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// OSADeploymentHandler handles app deployment operations
type OSADeploymentHandler struct {
	deploymentService *services.AppDeploymentService
}

// NewOSADeploymentHandler creates a new deployment handler
func NewOSADeploymentHandler(deploymentService *services.AppDeploymentService) *OSADeploymentHandler {
	return &OSADeploymentHandler{
		deploymentService: deploymentService,
	}
}

// NOTE: Route registration happens in handlers.go (lines 1360-1367)
// Routes use /deployment namespace to avoid conflicts with /apps/:id wildcard
// Deployment routes:
//   POST /api/osa/deployment/:app_id/deploy
//   POST /api/osa/deployment/:app_id/stop
//   GET  /api/osa/deployment/:app_id/status
//   GET  /api/osa/deployments

// DeployApp deploys a generated application
// POST /api/osa/deployment/:app_id/deploy
func (h *OSADeploymentHandler) DeployApp(c *gin.Context) {
	appIDStr := c.Param("app_id")
	appID, err := uuid.Parse(appIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid app ID"})
		return
	}

	deployedApp, err := h.deploymentService.DeployApp(c.Request.Context(), appID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to deploy app",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":             deployedApp.ID,
		"name":           deployedApp.Name,
		"workflow_id":    deployedApp.WorkflowID,
		"url":            deployedApp.URL,
		"port":           deployedApp.Port,
		"status":         deployedApp.Status,
		"app_type":       deployedApp.AppType,
		"deployed_at":    deployedApp.DeployedAt,
		"build_output":   deployedApp.BuildOutput,
		"startup_output": deployedApp.StartupOutput,
	})
}

// StopApp stops a running application
// POST /api/osa/deployment/:app_id/stop
func (h *OSADeploymentHandler) StopApp(c *gin.Context) {
	appIDStr := c.Param("app_id")
	appID, err := uuid.Parse(appIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid app ID"})
		return
	}

	if err := h.deploymentService.StopApp(appID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to stop app",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "App stopped successfully"})
}

// GetAppStatus retrieves status of a deployed app
// GET /api/osa/deployment/:app_id/status
func (h *OSADeploymentHandler) GetAppStatus(c *gin.Context) {
	appIDStr := c.Param("app_id")
	appID, err := uuid.Parse(appIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid app ID"})
		return
	}

	deployedApp, exists := h.deploymentService.GetDeployedApp(appID)
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "App not deployed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":           deployedApp.ID,
		"name":         deployedApp.Name,
		"url":          deployedApp.URL,
		"port":         deployedApp.Port,
		"status":       deployedApp.Status,
		"app_type":     deployedApp.AppType,
		"deployed_at":  deployedApp.DeployedAt,
		"last_healthy": deployedApp.LastHealthy,
	})
}

// ListDeployedApps lists all currently deployed apps
// GET /api/osa/deployments
func (h *OSADeploymentHandler) ListDeployedApps(c *gin.Context) {
	apps := h.deploymentService.ListDeployedApps()

	appsJSON := make([]gin.H, len(apps))
	for i, app := range apps {
		metadata := gin.H{}
		if app.Metadata != nil {
			metadata = gin.H{
				"name":        app.Metadata.Name,
				"description": app.Metadata.Description,
				"category":    app.Metadata.Category,
				"icon":        app.Metadata.Icon,
				"keywords":    app.Metadata.Keywords,
			}
		}

		appsJSON[i] = gin.H{
			"id":           app.ID,
			"name":         app.Name,
			"url":          app.URL,
			"port":         app.Port,
			"status":       app.Status,
			"app_type":     app.AppType,
			"deployed_at":  app.DeployedAt,
			"last_healthy": app.LastHealthy,
			"metadata":     metadata,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"apps":  appsJSON,
		"count": len(apps),
	})
}
