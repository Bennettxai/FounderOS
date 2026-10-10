package services

import "strings"

// MapYAMLToDB maps a YAML template to a database template structure.
func (s *TemplateSyncService) MapYAMLToDB(yaml *TemplateDefinition) *DBTemplate {
	dbTemplate := &DBTemplate{
		TemplateName:     yaml.Name,
		DisplayName:      yaml.DisplayName,
		Description:      yaml.Description,
		Category:         yaml.Category,
		YAMLTemplateName: yaml.Name,
		YAMLVersion:      yaml.Version,
		GenerationPrompt: yaml.Template,
	}

	// Map category to icon type
	dbTemplate.IconType = s.categoryToIcon(yaml.Category)

	// Extract target business types, challenges, and team sizes from tags
	dbTemplate.TargetBusinessTypes = s.extractBusinessTypes(yaml.Tags, yaml.Category)
	dbTemplate.TargetChallenges = s.extractChallenges(yaml.Tags, yaml.Category)
	dbTemplate.TargetTeamSizes = s.extractTeamSizes(yaml.Tags)

	// Calculate priority score based on category and tags
	dbTemplate.PriorityScore = s.calculatePriorityScore(yaml.Category, yaml.Tags)

	// Determine scaffold type
	dbTemplate.ScaffoldType = s.determineScaffoldType(yaml.Tags, yaml.Category)

	// Extract required modules and optional features from variables and tags
	dbTemplate.RequiredModules = s.extractRequiredModules(yaml.Category)
	dbTemplate.OptionalFeatures = s.extractOptionalFeatures(yaml.Variables)

	// Build template config
	dbTemplate.TemplateConfig = s.buildTemplateConfig(yaml)

	// Store template variables as JSONB
	dbTemplate.TemplateVariables = s.mapVariables(yaml.Variables)

	return dbTemplate
}

// categoryToIcon maps template categories to icon types.
func (s *TemplateSyncService) categoryToIcon(category string) string {
	iconMap := map[string]string{
		"app-generation":     "users",
		"data-visualization": "chart",
		"maintenance":        "wrench",
		"feature":            "plus",
		"operations":         "server",
		"marketing":          "globe",
		"crm":                "users",
		"project_management": "kanban",
	}

	if icon, ok := iconMap[category]; ok {
		return icon
	}

	return "file" // default icon
}

// extractBusinessTypes extracts target business types from tags and category.
func (s *TemplateSyncService) extractBusinessTypes(tags []string, category string) []string {
	businessTypes := []string{}

	// Category-based defaults
	if category == "app-generation" || category == "crm" {
		businessTypes = append(businessTypes, "saas", "startup", "enterprise", "small_business")
	} else if category == "data-visualization" {
		businessTypes = append(businessTypes, "saas", "enterprise", "agency")
	} else if category == "maintenance" {
		businessTypes = append(businessTypes, "saas", "startup", "enterprise", "agency", "small_business")
	}

	// Tag-based additions
	for _, tag := range tags {
		switch tag {
		case "crm", "business":
			if !stringSliceContains(businessTypes, "small_business") {
				businessTypes = append(businessTypes, "small_business")
			}
		case "full-stack":
			if !stringSliceContains(businessTypes, "startup") {
				businessTypes = append(businessTypes, "startup")
			}
		}
	}

	if len(businessTypes) == 0 {
		// Default fallback
		businessTypes = []string{"saas", "startup"}
	}

	return businessTypes
}

// extractChallenges extracts target challenges from tags and category.
func (s *TemplateSyncService) extractChallenges(tags []string, category string) []string {
	challenges := []string{}

	// Category-based challenges
	switch category {
	case "app-generation":
		challenges = []string{"rapid_prototyping", "scalability", "time_to_market"}
	case "data-visualization":
		challenges = []string{"analytics", "reporting", "data_insights"}
	case "maintenance":
		challenges = []string{"bug_fixing", "code_quality", "stability"}
	case "feature":
		challenges = []string{"feature_development", "user_experience", "innovation"}
	}

	// Tag-based additions
	for _, tag := range tags {
		switch tag {
		case "crm":
			if !stringSliceContains(challenges, "client_relationships") {
				challenges = append(challenges, "client_relationships")
			}
		case "analytics", "dashboard", "charts":
			if !stringSliceContains(challenges, "analytics") {
				challenges = append(challenges, "analytics")
			}
		case "bug", "debugging":
			if !stringSliceContains(challenges, "bug_fixing") {
				challenges = append(challenges, "bug_fixing")
			}
		}
	}

	if len(challenges) == 0 {
		challenges = []string{"development"}
	}

	return challenges
}

// extractTeamSizes extracts target team sizes.
func (s *TemplateSyncService) extractTeamSizes(tags []string) []string {
	// Default: suitable for all team sizes
	return []string{"solo", "small", "medium", "large"}
}

// calculatePriorityScore calculates priority score based on category and tags.
func (s *TemplateSyncService) calculatePriorityScore(category string, tags []string) int {
	baseScore := 70

	// Category-based scoring
	switch category {
	case "app-generation":
		baseScore = 85
	case "data-visualization":
		baseScore = 90
	case "maintenance":
		baseScore = 75
	case "feature":
		baseScore = 80
	}

	// Tag-based adjustments
	for _, tag := range tags {
		switch tag {
		case "full-stack":
			baseScore += 5
		case "crm", "dashboard":
			baseScore += 5
		case "bug":
			baseScore -= 5 // Maintenance has lower priority
		}
	}

	// Ensure score is within valid range
	if baseScore > 100 {
		baseScore = 100
	}
	if baseScore < 1 {
		baseScore = 1
	}

	return baseScore
}

// determineScaffoldType determines the scaffold type based on tags and category.
func (s *TemplateSyncService) determineScaffoldType(tags []string, category string) string {
	// Check tags for explicit scaffold types
	for _, tag := range tags {
		if tag == "full-stack" {
			return "full-stack"
		}
	}

	// Category-based defaults
	switch category {
	case "app-generation":
		return "full-stack"
	case "data-visualization":
		return "svelte"
	case "maintenance":
		return "go"
	default:
		return "svelte"
	}
}

// extractRequiredModules extracts required modules based on category.
func (s *TemplateSyncService) extractRequiredModules(category string) []string {
	moduleMap := map[string][]string{
		"app-generation":     {"database", "api", "auth"},
		"data-visualization": {"dashboard", "analytics"},
		"maintenance":        {"logging", "testing"},
		"feature":            {"api"},
	}

	if modules, ok := moduleMap[category]; ok {
		return modules
	}

	return []string{}
}

// extractOptionalFeatures extracts optional features from template variables.
func (s *TemplateSyncService) extractOptionalFeatures(variables []TemplateVariable) []string {
	features := []string{}

	for _, v := range variables {
		if !v.Required && v.Default != nil {
			// Optional variables suggest optional features
			switch v.Name {
			case "AvailableIntegrations":
				features = append(features, "third_party_integrations")
			case "RefreshInterval":
				features = append(features, "real_time_updates")
			case "UserRoles":
				features = append(features, "role_based_access")
			case "ChartTypes":
				features = append(features, "custom_charts")
			}
		}
	}

	return features
}

// buildTemplateConfig builds the template configuration map.
func (s *TemplateSyncService) buildTemplateConfig(yaml *TemplateDefinition) map[string]interface{} {
	config := map[string]interface{}{
		"category": yaml.Category,
		"version":  yaml.Version,
		"tags":     yaml.Tags,
	}

	// Add scaffold type hint
	if strings.Contains(yaml.Template, "Go + Gin") || strings.Contains(yaml.Template, "Backend API") {
		config["has_backend"] = true
	}
	if strings.Contains(yaml.Template, "SvelteKit") || strings.Contains(yaml.Template, "Frontend UI") {
		config["has_frontend"] = true
	}

	return config
}

// mapVariables maps YAML variables to JSONB structure.
func (s *TemplateSyncService) mapVariables(variables []TemplateVariable) map[string]interface{} {
	varMap := make(map[string]interface{})

	for _, v := range variables {
		varMap[v.Name] = map[string]interface{}{
			"type":        v.Type,
			"required":    v.Required,
			"default":     v.Default,
			"description": v.Description,
		}
	}

	return varMap
}
