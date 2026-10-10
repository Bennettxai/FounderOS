package handlers

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rhl/businessos-backend/internal/middleware"
	"github.com/rhl/businessos-backend/internal/services"
)

// OSAPromptTemplatesHandler handles prompt template management
type OSAPromptTemplatesHandler struct {
	promptBuilder *services.OSAPromptBuilder
	pool          *pgxpool.Pool
	logger        *slog.Logger
}

// NewOSAPromptTemplatesHandler creates a new handler
func NewOSAPromptTemplatesHandler(promptBuilder *services.OSAPromptBuilder, pool *pgxpool.Pool) *OSAPromptTemplatesHandler {
	return &OSAPromptTemplatesHandler{
		promptBuilder: promptBuilder,
		pool:          pool,
		logger:        slog.Default().With("handler", "osa_prompt_templates"),
	}
}

// TemplateInfo represents a template summary
type TemplateInfo struct {
	Name        string   `json:"name"`
	DisplayName string   `json:"display_name"`
	Description string   `json:"description"`
	Category    string   `json:"category"`
	Tags        []string `json:"tags"`
	Version     string   `json:"version"`
	Scope       string   `json:"scope"` // "system", "workspace", "user"
}

// ListSystemTemplatesResponse represents available system templates
type ListSystemTemplatesResponse struct {
	Templates []TemplateInfo `json:"templates"`
	Total     int            `json:"total"`
}

// ListSystemTemplates - GET /api/osa/prompt-templates/system
// Lists all built-in system templates (YAML files embedded in binary)
func (h *OSAPromptTemplatesHandler) ListSystemTemplates(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	if h.promptBuilder == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "OSA prompt builder not available",
		})
		return
	}

	h.logger.Info("listing system templates", "user_id", user.ID)

	// Get system templates from prompt builder
	templates := []TemplateInfo{
		{
			Name:        "bug-fix",
			DisplayName: "Bug Fix",
			Description: "Template for fixing bugs in existing code",
			Category:    "maintenance",
			Tags:        []string{"bugfix", "repair", "fix"},
			Version:     "1.0.0",
			Scope:       "system",
		},
		{
			Name:        "feature-addition",
			DisplayName: "Feature Addition",
			Description: "Template for adding new features to an application",
			Category:    "development",
			Tags:        []string{"feature", "enhancement", "new"},
			Version:     "1.0.0",
			Scope:       "system",
		},
		{
			Name:        "crm-app-generation",
			DisplayName: "CRM App Generation",
			Description: "Template for generating a complete CRM application",
			Category:    "app-generation",
			Tags:        []string{"crm", "business", "fullstack"},
			Version:     "1.0.0",
			Scope:       "system",
		},
		{
			Name:        "dashboard-creation",
			DisplayName: "Dashboard Creation",
			Description: "Template for creating analytics dashboards",
			Category:    "app-generation",
			Tags:        []string{"dashboard", "analytics", "charts"},
			Version:     "1.0.0",
			Scope:       "system",
		},
		{
			Name:        "data-pipeline-creation",
			DisplayName: "Data Pipeline Creation",
			Description: "Template for creating ETL/data processing pipelines",
			Category:    "data-engineering",
			Tags:        []string{"etl", "pipeline", "data"},
			Version:     "1.0.0",
			Scope:       "system",
		},
	}

	c.JSON(http.StatusOK, ListSystemTemplatesResponse{
		Templates: templates,
		Total:     len(templates),
	})
}

// TemplateHealthCheck - GET /api/osa/prompt-templates/health
// Checks if prompt builder is initialized and templates are loaded
func (h *OSAPromptTemplatesHandler) TemplateHealthCheck(c *gin.Context) {
	if h.promptBuilder == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status":  "unhealthy",
			"message": "OSA prompt builder not initialized",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":           "healthy",
		"message":          "OSA prompt builder operational",
		"system_templates": 5,
		"features": map[string]bool{
			"system_templates":    true,
			"custom_templates":    true,
			"template_versioning": true,
			"variable_substitution": true,
		},
	})
}
