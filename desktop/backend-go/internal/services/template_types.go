package services

// TemplateDefinition represents a YAML template with metadata, variables, and template content
type TemplateDefinition struct {
	Name        string              `yaml:"name"`
	DisplayName string              `yaml:"display_name"`
	Description string              `yaml:"description"`
	Category    string              `yaml:"category"`
	Version     string              `yaml:"version"`
	Tags        []string            `yaml:"tags"`
	Variables   []TemplateVariable  `yaml:"variables"`
	Template    string              `yaml:"template"`
}

// TemplateVariable represents a variable definition in a template
type TemplateVariable struct {
	Name        string      `yaml:"name"`
	Type        string      `yaml:"type"` // "string", "array", "object"
	Required    bool        `yaml:"required"`
	Default     interface{} `yaml:"default"`
	Description string      `yaml:"description"`
}

// TemplateMetadata contains summary information about a template
type TemplateMetadata struct {
	Name        string   `json:"name"`
	DisplayName string   `json:"display_name"`
	Description string   `json:"description"`
	Category    string   `json:"category"`
	Version     string   `json:"version"`
	Tags        []string `json:"tags"`
}
