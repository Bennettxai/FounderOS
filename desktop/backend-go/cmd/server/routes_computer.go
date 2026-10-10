package main

import (
	"github.com/gin-gonic/gin"
	"github.com/rhl/businessos-backend/internal/handlers"
	"github.com/rhl/businessos-backend/internal/middleware"
)

// registerComputerRoutes wires up all Computer and Billing API endpoints.
func registerComputerRoutes(api *gin.RouterGroup, auth gin.HandlerFunc, ch *handlers.ComputerHandler, bh *handlers.BillingHandler) {
	comp := api.Group("/computer")
	comp.Use(auth, middleware.RequireAuth())
	{
		comp.GET("", ch.GetComputer)
		comp.POST("", ch.CreateComputer)
		comp.PUT("/upgrade", ch.UpgradeComputer)
		comp.DELETE("", ch.DeleteComputer)
		comp.GET("/metrics", ch.GetMetrics)
		comp.GET("/runtimes", ch.GetRuntimes)
		comp.POST("/runtimes/:name/start", ch.StartRuntime)
		comp.POST("/runtimes/:name/stop", ch.StopRuntime)
		comp.GET("/terminal-session", ch.GetTerminalSession)
		comp.GET("/desktop-stream", ch.GetDesktopStream)
	}

	billing := api.Group("/billing")
	billing.Use(auth, middleware.RequireAuth())
	{
		billing.GET("/plans", bh.GetPlans)
		billing.GET("/subscription", bh.GetSubscription)
		billing.POST("/subscribe", bh.Subscribe)
		billing.POST("/credits/purchase", bh.PurchaseCredits)
	}
}
