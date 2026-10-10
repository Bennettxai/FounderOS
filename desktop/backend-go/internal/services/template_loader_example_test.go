package services

import (
	"fmt"
	"log"
)

// Example demonstrates how to use the TemplateLoaderService
func Example_templateLoaderBasicUsage() {
	// Create service with templates directory
	templatesDir := GetTemplatesDirectory()
	service := NewTemplateLoaderService(templatesDir)

	// Load a specific template
	tmpl, err := service.LoadTemplate("bug-fix")
	if err != nil {
		log.Fatalf("Failed to load template: %v", err)
	}

	fmt.Printf("Template: %s\n", tmpl.Name)
	fmt.Printf("Display Name: %s\n", tmpl.DisplayName)
	fmt.Printf("Category: %s\n", tmpl.Category)
	fmt.Printf("Variables: %d\n", len(tmpl.Variables))

	// Prepare variables for rendering
	variables := map[string]interface{}{
		"AppName":           "MyApp",
		"BugDescription":    "Login button not working",
		"ReproductionSteps": "1. Navigate to login page\n2. Click login button\n3. Nothing happens",
	}

	// Render the template
	rendered, err := service.RenderTemplate("bug-fix", variables)
	if err != nil {
		log.Fatalf("Failed to render template: %v", err)
	}

	fmt.Printf("Rendered output length: %d characters\n", len(rendered))
	// Output would contain the fully rendered template with variables substituted
}

// Example demonstrates listing all available templates
func Example_templateLoaderListTemplates() {
	templatesDir := GetTemplatesDirectory()
	service := NewTemplateLoaderService(templatesDir)

	// List all templates
	templates, err := service.ListTemplates()
	if err != nil {
		log.Fatalf("Failed to list templates: %v", err)
	}

	fmt.Printf("Available templates: %d\n", len(templates))
	for _, tmpl := range templates {
		fmt.Printf("- %s (%s)\n", tmpl.DisplayName, tmpl.Category)
	}
}

// Example demonstrates validation of template variables
func Example_templateLoaderValidation() {
	templatesDir := GetTemplatesDirectory()
	service := NewTemplateLoaderService(templatesDir)

	tmpl, err := service.LoadTemplate("crm-app-generation")
	if err != nil {
		log.Fatalf("Failed to load template: %v", err)
	}

	// Try with missing required variables (will fail)
	invalidVars := map[string]interface{}{
		"UserBusiness": "Real Estate",
		// Missing "AppType" and "UserRequirements"
	}

	err = service.ValidateVariables(tmpl, invalidVars)
	if err != nil {
		fmt.Printf("Validation failed (expected): %v\n", err)
	}

	// Try with all required variables (will succeed)
	validVars := map[string]interface{}{
		"AppType":          "CRM",
		"UserBusiness":     "Real Estate",
		"UserRequirements": "Property management, client tracking",
	}

	err = service.ValidateVariables(tmpl, validVars)
	if err != nil {
		log.Fatalf("Unexpected validation error: %v", err)
	}

	fmt.Println("Validation succeeded")
}

// Example demonstrates caching behavior
func Example_templateLoaderCaching() {
	templatesDir := GetTemplatesDirectory()
	service := NewTemplateLoaderService(templatesDir)

	// First load - reads from file
	tmpl1, _ := service.LoadTemplate("dashboard-creation")
	fmt.Printf("First load: %s\n", tmpl1.Name)

	// Second load - uses cache (same pointer)
	tmpl2, _ := service.LoadTemplate("dashboard-creation")
	fmt.Printf("Second load (cached): %s\n", tmpl2.Name)

	// Clear cache
	service.ClearCache()

	// Third load - reads from file again (new pointer)
	tmpl3, _ := service.LoadTemplate("dashboard-creation")
	fmt.Printf("Third load (after cache clear): %s\n", tmpl3.Name)
}
