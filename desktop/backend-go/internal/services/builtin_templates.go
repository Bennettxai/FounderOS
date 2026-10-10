package services

// BuiltInTemplate represents a complete template definition with file templates
type BuiltInTemplate struct {
	ID            string                 `json:"id"`
	Name          string                 `json:"name"`
	Description   string                 `json:"description"`
	Category      string                 `json:"category"`
	StackType     string                 `json:"stack_type"` // svelte, go, fullstack
	FilesTemplate map[string]string      `json:"files_template"`
	ConfigSchema  map[string]ConfigField `json:"config_schema"`
}

// ConfigField describes a configuration field for a template
type ConfigField struct {
	Type        string   `json:"type"` // string, number, boolean, select
	Label       string   `json:"label"`
	Description string   `json:"description"`
	Default     string   `json:"default"`
	Required    bool     `json:"required"`
	Options     []string `json:"options,omitempty"`
}

// builtInTemplates holds all built-in template definitions
var builtInTemplates = map[string]*BuiltInTemplate{
	"saas_dashboard": saaDashboardTemplate(),
	"api_backend":    apiBackendTemplate(),
	"landing_page":   landingPageTemplate(),
	"crm_module":     crmModuleTemplate(),
	"task_manager":   taskManagerTemplate(),
}

// GetBuiltInTemplate returns a built-in template by name
func GetBuiltInTemplate(name string) (*BuiltInTemplate, bool) {
	t, ok := builtInTemplates[name]
	return t, ok
}

// GetAllBuiltInTemplates returns all built-in templates
func GetAllBuiltInTemplates() map[string]*BuiltInTemplate {
	return builtInTemplates
}
