package handlers

import "github.com/gin-gonic/gin"

// registerCRMRoutes wires up CRM and business-data routes:
// /api/clients, /api/crm, /api/tables.
func (h *Handlers) registerCRMRoutes(api *gin.RouterGroup, auth gin.HandlerFunc) {
	// Client routes - /api/clients
	clientHandler := NewClientHandler(h.pool)
	clientHandler.SetEngineSync(h.engineSync)
	RegisterClientRoutes(api, clientHandler, auth)

	// CRM routes - /api/crm
	crmHandler := NewCRMHandler(h.pool)
	crmHandler.SetEngineSync(h.engineSync)
	RegisterCRMRoutes(api, crmHandler, auth)

	// Table routes - /api/tables
	RegisterTableRoutes(api, NewTableHandler(h.pool), auth)
}
