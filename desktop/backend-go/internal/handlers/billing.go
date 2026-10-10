package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rhl/businessos-backend/internal/config"
	"github.com/rhl/businessos-backend/internal/integrations/miosa"
)

type BillingHandler struct{}

func NewBillingHandler(_ *config.Config, _ *miosa.ComputeClient) *BillingHandler {
	return &BillingHandler{}
}

var availablePlans = []PlanInfo{
	{ID: "community", Name: "Community", PriceMonthly: 0, RAMGB: 0, CPUs: 0, StorageGB: 0, Credits: 0,
		Features: []string{"Self-hosted", "All modules", "Community support"}},
}

func (h *BillingHandler) GetPlans(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"plans": availablePlans})
}

func (h *BillingHandler) GetSubscription(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"plan": "community", "status": "active", "message": "open-source edition — self-hosted, no subscription required"})
}

func (h *BillingHandler) Subscribe(c *gin.Context) {
	c.JSON(http.StatusNotFound, gin.H{"error": "billing not available in open-source edition"})
}

func (h *BillingHandler) PurchaseCredits(c *gin.Context) {
	c.JSON(http.StatusNotFound, gin.H{"error": "billing not available in open-source edition"})
}

func (h *BillingHandler) HandleStripeWebhook(c *gin.Context) {
	c.JSON(http.StatusNotFound, gin.H{"error": "billing not available in open-source edition"})
}
