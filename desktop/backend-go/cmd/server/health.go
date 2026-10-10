package main

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rhl/businessos-backend/internal/config"
	redisClient "github.com/rhl/businessos-backend/internal/redis"
)

// healthDeps groups the state variables required by the dynamic health handlers.
// All fields are passed explicitly so handlers have no closure over main() locals.
type healthDeps struct {
	instanceID     string
	dbConnected    bool
	dbErr          error
	redisConnected bool
	containerMgr   containerManagerInterface // nil-safe check via != nil
}

// containerManagerInterface is the minimal subset of container.ContainerManager
// needed for health checks (just a nil check). Using interface{} here keeps
// health.go free from importing the container package.
type containerManagerInterface interface{}

// newRootHandler returns the GET / handler.
func newRootHandler(instanceID string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message":     "Business OS API",
			"version":     "1.0.0",
			"instance_id": instanceID,
		})
	}
}

// newHealthHandler returns the GET /health handler.
// In cloud deployment mode it includes tenant, domain, and uptime information.
func newHealthHandler(cfg *config.Config, startTime time.Time) gin.HandlerFunc {
	return func(c *gin.Context) {
		resp := gin.H{
			"status":          "healthy",
			"version":         "1.0.0",
			"deployment_mode": cfg.DeploymentMode,
			"uptime_seconds":  int64(time.Since(startTime).Seconds()),
		}
		if cfg.IsCloudDeployment() {
			resp["tenant_id"] = cfg.MIOSATenantID
			resp["domain"] = cfg.CloudDomain
		}
		c.JSON(http.StatusOK, resp)
	}
}

// newReadinessHandler returns the GET /ready readiness handler.
// cfg is passed as a minimal interface — only IsProduction() and
// DatabaseRequired are required; we accept the full *config struct directly.
func newReadinessHandler(deps healthDeps, cfgDatabaseRequired bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		dbStatus := "disconnected"
		if deps.dbConnected {
			dbStatus = "connected"
		} else if !cfgDatabaseRequired {
			dbStatus = "disabled"
		}

		status := gin.H{
			"status":      "ready",
			"instance_id": deps.instanceID,
			"database":    dbStatus,
			"redis":       "disconnected",
			"containers":  "unavailable",
		}

		if deps.redisConnected && redisClient.IsConnected(c.Request.Context()) {
			status["redis"] = "connected"
		}

		if deps.containerMgr != nil {
			status["containers"] = "available"
		}

		c.JSON(http.StatusOK, status)
	}
}

// newDetailedHealthHandler returns the GET /health/detailed handler.
func newDetailedHealthHandler(deps healthDeps, cfgDatabaseRequired bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		health := gin.H{
			"status":      "healthy",
			"instance_id": deps.instanceID,
			"components":  gin.H{},
		}

		components := health["components"].(gin.H)

		// Database component
		dbComponent := gin.H{}
		if deps.dbConnected {
			dbComponent["status"] = "connected"
		} else if !cfgDatabaseRequired {
			dbComponent["status"] = "disabled"
			if deps.dbErr != nil {
				dbComponent["error"] = deps.dbErr.Error()
			}
		} else {
			dbComponent["status"] = "disconnected"
			if deps.dbErr != nil {
				dbComponent["error"] = deps.dbErr.Error()
			}
		}
		components["database"] = dbComponent

		// Redis component
		if deps.redisConnected {
			redisHealth, err := redisClient.HealthCheck(c.Request.Context())
			if err != nil {
				components["redis"] = gin.H{"status": "error", "error": err.Error()}
			} else {
				components["redis"] = gin.H{
					"status":     "connected",
					"latency_ms": redisHealth.Latency.Milliseconds(),
					"pool_stats": redisHealth.PoolStats,
				}
			}
		} else {
			components["redis"] = gin.H{"status": "not_configured"}
		}

		// Container manager component
		if deps.containerMgr != nil {
			components["containers"] = gin.H{"status": "available"}
		} else {
			components["containers"] = gin.H{"status": "unavailable"}
		}

		c.JSON(http.StatusOK, health)
	}
}
