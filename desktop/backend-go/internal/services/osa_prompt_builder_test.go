package services

import (
	"context"
	"log/slog"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewOSAPromptBuilder(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	pb, err := NewOSAPromptBuilder(nil, logger)

	assert.NoError(t, err)
	assert.NotNil(t, pb)
	assert.Greater(t, len(pb.systemTemplates), 0, "should load at least one system template")
}

func TestLoadSystemTemplates(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	pb, err := NewOSAPromptBuilder(nil, logger)
	require.NoError(t, err)

	// Check that known templates are loaded
	expectedTemplates := []string{
		"crm-app-generation",
		"data-pipeline-creation",
		"dashboard-creation",
		"feature-addition",
		"bug-fix",
	}

	for _, templateName := range expectedTemplates {
		tpl, ok := pb.systemTemplates[templateName]
		assert.True(t, ok, "template %s should be loaded", templateName)
		if ok {
			assert.NotEmpty(t, tpl.DisplayName)
			assert.NotEmpty(t, tpl.Description)
			assert.NotEmpty(t, tpl.Version)
			assert.NotNil(t, tpl.compiledTpl, "template should be compiled")
			t.Logf("✅ Template loaded: %s (v%s) - %s", tpl.Name, tpl.Version, tpl.DisplayName)
		}
	}
}

func TestBuildAppGenerationPrompt_CRM(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	pb, err := NewOSAPromptBuilder(nil, logger)
	require.NoError(t, err)

	ctx := context.Background()
	userID := uuid.New()
	workspaceID := uuid.New()

	variables := map[string]interface{}{
		"AppType":            "CRM",
		"UserBusiness":       "Real Estate",
		"UserRequirements":   "Track leads, properties, and appointments with automated follow-ups",
		"DatabasePreference": "PostgreSQL",
		"AvailableIntegrations": []map[string]string{
			{"Name": "Stripe", "Description": "Payment processing", "Status": "connected"},
			{"Name": "Twilio", "Description": "SMS notifications", "Status": "disconnected"},
		},
	}

	req := AppGenerationRequest{
		TemplateName: "crm-app-generation",
		Variables:    variables,
		UserID:       &userID,
		WorkspaceID:  &workspaceID,
	}

	result, err := pb.BuildAppGenerationPrompt(ctx, req)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotEmpty(t, result.Prompt)
	assert.Equal(t, "crm-app-generation", result.TemplateName)
	assert.GreaterOrEqual(t, result.RenderTimeMs, int64(0), "render time should be non-negative")

	// Verify prompt contains expected content
	assert.Contains(t, result.Prompt, "CRM")
	assert.Contains(t, result.Prompt, "Real Estate")
	assert.Contains(t, result.Prompt, "PostgreSQL")
	assert.Contains(t, result.Prompt, "Stripe")
	assert.Contains(t, result.Prompt, "automated follow-ups")

	t.Logf("✅ Rendered prompt length: %d characters", len(result.Prompt))
	t.Logf("✅ Render time: %d ms", result.RenderTimeMs)
}

func TestBuildAppGenerationPrompt_DataPipeline(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	pb, err := NewOSAPromptBuilder(nil, logger)
	require.NoError(t, err)

	ctx := context.Background()

	variables := map[string]interface{}{
		"SourceType":          "REST API",
		"DestinationType":     "PostgreSQL",
		"TransformationRules": "Normalize customer names, convert timestamps to UTC, deduplicate by email",
		"Schedule":            "hourly",
		"DataVolume":          "medium",
	}

	req := AppGenerationRequest{
		TemplateName: "data-pipeline-creation",
		Variables:    variables,
	}

	result, err := pb.BuildAppGenerationPrompt(ctx, req)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Contains(t, result.Prompt, "REST API")
	assert.Contains(t, result.Prompt, "PostgreSQL")
	assert.Contains(t, result.Prompt, "hourly")
	assert.Contains(t, result.Prompt, "Normalize customer names")
}

func TestBuildAppGenerationPrompt_FeatureAddition(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	pb, err := NewOSAPromptBuilder(nil, logger)
	require.NoError(t, err)

	ctx := context.Background()

	variables := map[string]interface{}{
		"AppName":            "TaskManager Pro",
		"FeatureDescription": "Add real-time collaboration with WebSocket support",
		"ExistingContext": map[string]interface{}{
			"TechStack":           "Go + Gin + SvelteKit",
			"ArchitecturePattern": "Handler → Service → Repository",
			"CurrentVersion":      "2.1.0",
			"Modules": []map[string]interface{}{
				{"Name": "Tasks", "Description": "Task management", "Status": "stable"},
				{"Name": "Users", "Description": "User authentication", "Status": "stable"},
			},
		},
		"DatabaseChanges": true,
		"BreakingChanges": false,
		"AffectedModules": []string{"Tasks", "WebSocket"},
	}

	req := AppGenerationRequest{
		TemplateName: "feature-addition",
		Variables:    variables,
	}

	result, err := pb.BuildAppGenerationPrompt(ctx, req)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Contains(t, result.Prompt, "TaskManager Pro")
	assert.Contains(t, result.Prompt, "real-time collaboration")
	assert.Contains(t, result.Prompt, "WebSocket")
	assert.Contains(t, result.Prompt, "REQUIRED", "should indicate database changes required")
	assert.Contains(t, result.Prompt, "NO BREAKING CHANGES")
}

func TestBuildAppGenerationPrompt_BugFix(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	pb, err := NewOSAPromptBuilder(nil, logger)
	require.NoError(t, err)

	ctx := context.Background()

	variables := map[string]interface{}{
		"AppName":           "PaymentProcessor",
		"BugDescription":    "Payment webhooks are not processing refunds correctly",
		"ReproductionSteps": "1. Create a payment\n2. Issue a refund via Stripe dashboard\n3. Check webhook logs - refund event is received but not processed",
		"ErrorLogs":         "ERROR: null pointer dereference at webhook_handler.go:142",
		"ExpectedBehavior":  "Refund should be recorded in database and customer notified",
		"ActualBehavior":    "Server returns 500 error and refund is not recorded",
		"Severity":          "high",
		"Environment":       "production",
		"AffectedVersion":   "3.2.1",
	}

	req := AppGenerationRequest{
		TemplateName: "bug-fix",
		Variables:    variables,
	}

	result, err := pb.BuildAppGenerationPrompt(ctx, req)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Contains(t, result.Prompt, "PaymentProcessor")
	assert.Contains(t, result.Prompt, "null pointer dereference")
	assert.Contains(t, result.Prompt, "HIGH PRIORITY")
	assert.Contains(t, result.Prompt, "production")
	assert.Contains(t, result.Prompt, "Root Cause Analysis")
	assert.Contains(t, result.Prompt, "Regression test")
}

func TestBuildAppGenerationPrompt_BugFix_Critical(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	pb, err := NewOSAPromptBuilder(nil, logger)
	require.NoError(t, err)

	ctx := context.Background()

	variables := map[string]interface{}{
		"AppName":           "AuthService",
		"BugDescription":    "Users can access other users' data by manipulating user_id parameter",
		"ReproductionSteps": "1. Login as user A\n2. Change user_id in API request to user B's ID\n3. Receive user B's private data",
		"Severity":          "critical",
		"Environment":       "production",
	}

	req := AppGenerationRequest{
		TemplateName: "bug-fix",
		Variables:    variables,
	}

	result, err := pb.BuildAppGenerationPrompt(ctx, req)

	assert.NoError(t, err)
	assert.Contains(t, result.Prompt, "CRITICAL BUG")
	assert.Contains(t, result.Prompt, "IMMEDIATE FIX")
	assert.Contains(t, result.Prompt, "Hotfix deployment")
}

func TestValidateVariables_MissingRequired(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	pb, err := NewOSAPromptBuilder(nil, logger)
	require.NoError(t, err)

	ctx := context.Background()

	// Missing required "UserBusiness" and "UserRequirements"
	variables := map[string]interface{}{
		"AppType": "CRM",
	}

	req := AppGenerationRequest{
		TemplateName: "crm-app-generation",
		Variables:    variables,
	}

	_, err = pb.BuildAppGenerationPrompt(ctx, req)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "required variables missing")
	assert.Contains(t, err.Error(), "UserBusiness")
	assert.Contains(t, err.Error(), "UserRequirements")
}

func TestValidateVariables_AllRequired(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	pb, err := NewOSAPromptBuilder(nil, logger)
	require.NoError(t, err)

	ctx := context.Background()

	// All required variables provided
	variables := map[string]interface{}{
		"AppType":          "CRM",
		"UserBusiness":     "Healthcare",
		"UserRequirements": "HIPAA-compliant patient management system",
	}

	req := AppGenerationRequest{
		TemplateName: "crm-app-generation",
		Variables:    variables,
	}

	result, err := pb.BuildAppGenerationPrompt(ctx, req)

	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestApplyDefaults(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	pb, err := NewOSAPromptBuilder(nil, logger)
	require.NoError(t, err)

	ctx := context.Background()

	// Not providing optional variables - should use defaults
	variables := map[string]interface{}{
		"AppType":          "CRM",
		"UserBusiness":     "Finance",
		"UserRequirements": "Investment portfolio tracking",
		// DatabasePreference not provided - should default to "PostgreSQL"
		// AvailableIntegrations not provided - should default to []
	}

	req := AppGenerationRequest{
		TemplateName: "crm-app-generation",
		Variables:    variables,
	}

	result, err := pb.BuildAppGenerationPrompt(ctx, req)

	assert.NoError(t, err)
	assert.Contains(t, result.Prompt, "PostgreSQL", "should use default database preference")
	assert.Contains(t, result.Prompt, "No third-party integrations", "should handle empty integrations array")
}

func TestGetTemplateInfo(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	pb, err := NewOSAPromptBuilder(nil, logger)
	require.NoError(t, err)

	ctx := context.Background()

	tpl, err := pb.GetTemplateInfo(ctx, "crm-app-generation")

	assert.NoError(t, err)
	assert.NotNil(t, tpl)
	assert.Equal(t, "crm-app-generation", tpl.Name)
	assert.NotEmpty(t, tpl.DisplayName)
	assert.NotEmpty(t, tpl.Variables)
}

func TestGetTemplateInfo_NotFound(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	pb, err := NewOSAPromptBuilder(nil, logger)
	require.NoError(t, err)

	ctx := context.Background()

	_, err = pb.GetTemplateInfo(ctx, "nonexistent-template")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "template not found")
}

func TestListAvailableTemplates(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	pb, err := NewOSAPromptBuilder(nil, logger)
	require.NoError(t, err)

	ctx := context.Background()

	// List all templates
	templates, err := pb.ListAvailableTemplates(ctx, "")

	assert.NoError(t, err)
	assert.Greater(t, len(templates), 0)

	for _, tpl := range templates {
		t.Logf("Template: %s (category: %s)", tpl.Name, tpl.Category)
	}
}

func TestListAvailableTemplates_FilterByCategory(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	pb, err := NewOSAPromptBuilder(nil, logger)
	require.NoError(t, err)

	ctx := context.Background()

	// List only app-generation templates
	templates, err := pb.ListAvailableTemplates(ctx, "app-generation")

	assert.NoError(t, err)
	assert.Greater(t, len(templates), 0)

	for _, tpl := range templates {
		assert.Equal(t, "app-generation", tpl.Category)
	}
}

func TestGetTemplateVariableSchema(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	pb, err := NewOSAPromptBuilder(nil, logger)
	require.NoError(t, err)

	ctx := context.Background()

	schema, err := pb.GetTemplateVariableSchema(ctx, "crm-app-generation")

	assert.NoError(t, err)
	assert.NotEmpty(t, schema)

	// Check for known required variables
	var foundAppType, foundUserBusiness bool
	for _, varDef := range schema {
		if varDef.Name == "AppType" {
			foundAppType = true
			assert.True(t, varDef.Required)
			assert.Equal(t, "string", varDef.Type)
		}
		if varDef.Name == "UserBusiness" {
			foundUserBusiness = true
			assert.True(t, varDef.Required)
		}
	}

	assert.True(t, foundAppType, "should have AppType variable")
	assert.True(t, foundUserBusiness, "should have UserBusiness variable")
}

func TestValidateTemplateVariables(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	pb, err := NewOSAPromptBuilder(nil, logger)
	require.NoError(t, err)

	ctx := context.Background()

	// Valid variables
	validVars := map[string]interface{}{
		"AppType":          "CRM",
		"UserBusiness":     "Retail",
		"UserRequirements": "E-commerce integration",
	}

	err = pb.ValidateTemplateVariables(ctx, "crm-app-generation", validVars)
	assert.NoError(t, err)

	// Invalid variables (missing required)
	invalidVars := map[string]interface{}{
		"AppType": "CRM",
		// Missing UserBusiness and UserRequirements
	}

	err = pb.ValidateTemplateVariables(ctx, "crm-app-generation", invalidVars)
	assert.Error(t, err)
}

func TestTemplateResolution_SystemOnly(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	pb, err := NewOSAPromptBuilder(nil, logger)
	require.NoError(t, err)

	ctx := context.Background()

	// No user or workspace ID - should resolve to system template
	tpl, templateID, err := pb.resolveTemplate(ctx, "crm-app-generation", nil, nil)

	assert.NoError(t, err)
	assert.NotNil(t, tpl)
	assert.Nil(t, templateID, "system templates don't have database IDs")
	assert.Equal(t, "crm-app-generation", tpl.Name)
}

func TestPromptRendering_SpecialCharacters(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	pb, err := NewOSAPromptBuilder(nil, logger)
	require.NoError(t, err)

	ctx := context.Background()

	// Variables with special characters
	variables := map[string]interface{}{
		"AppType":          "CRM",
		"UserBusiness":     "Finance & Investment's \"Portfolio\" Management",
		"UserRequirements": "Track <user> data with $special characters",
	}

	req := AppGenerationRequest{
		TemplateName: "crm-app-generation",
		Variables:    variables,
	}

	result, err := pb.BuildAppGenerationPrompt(ctx, req)

	assert.NoError(t, err)
	assert.Contains(t, result.Prompt, "Finance & Investment")
	assert.Contains(t, result.Prompt, "$special characters")
}

func TestPromptRendering_EmptyArrays(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	pb, err := NewOSAPromptBuilder(nil, logger)
	require.NoError(t, err)

	ctx := context.Background()

	variables := map[string]interface{}{
		"AppType":               "CRM",
		"UserBusiness":          "Education",
		"UserRequirements":      "Student management system",
		"AvailableIntegrations": []interface{}{}, // Empty array
	}

	req := AppGenerationRequest{
		TemplateName: "crm-app-generation",
		Variables:    variables,
	}

	result, err := pb.BuildAppGenerationPrompt(ctx, req)

	assert.NoError(t, err)
	assert.Contains(t, result.Prompt, "No third-party integrations")
}

func TestLogTemplateUsage(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	pb, err := NewOSAPromptBuilder(nil, logger)
	require.NoError(t, err)

	ctx := context.Background()
	userID := uuid.New()
	workspaceID := uuid.New()

	result := &PromptBuildResult{
		TemplateName: "crm-app-generation",
		Variables: map[string]interface{}{
			"AppType":      "CRM",
			"UserBusiness": "Test",
		},
		RenderTimeMs: 42,
	}

	// Should not panic or error
	pb.LogTemplateUsage(ctx, result, userID, &workspaceID, "success", "")
}

// Benchmark tests
func BenchmarkBuildAppGenerationPrompt(b *testing.B) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	pb, err := NewOSAPromptBuilder(nil, logger)
	require.NoError(b, err)

	ctx := context.Background()

	variables := map[string]interface{}{
		"AppType":          "CRM",
		"UserBusiness":     "Real Estate",
		"UserRequirements": "Lead tracking",
	}

	req := AppGenerationRequest{
		TemplateName: "crm-app-generation",
		Variables:    variables,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = pb.BuildAppGenerationPrompt(ctx, req)
	}
}

func BenchmarkLoadSystemTemplates(b *testing.B) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = NewOSAPromptBuilder(nil, logger)
	}
}

// ---- E-03: Additional tests added by Agent E (QA) ----

// TestGetDBTemplate_ReturnsNilNotError verifies that getDBTemplate behaves as a
// no-op stub: it returns (nil, nil, non-nil error), and that resolveTemplate
// falls through gracefully to the system-template layer when no DB pool is
// provided.  The DB layer is tested indirectly through resolveTemplate because
// getDBTemplate is unexported.
func TestGetDBTemplate_ReturnsNilNotError(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	pb, err := NewOSAPromptBuilder(nil, logger)
	require.NoError(t, err)

	ctx := context.Background()

	// resolveTemplate with both userID and workspaceID provided must still
	// succeed by falling back to the system template layer, proving that
	// getDBTemplate returning a non-nil error is handled as "not found" (not
	// propagated as a hard failure).
	userID := uuid.New()
	workspaceID := uuid.New()

	tpl, templateID, err := pb.resolveTemplate(ctx, "crm-app-generation", &userID, &workspaceID)

	require.NoError(t, err, "resolveTemplate must succeed via system-template fallback")
	require.NotNil(t, tpl, "should have resolved a system template")
	assert.Nil(t, templateID, "system templates carry no DB UUID")
	assert.Equal(t, "crm-app-generation", tpl.Name)
}

// TestBuildStandardVariables tests BuildStandardVariables as delivered by Agent F.
//
// Actual interface (Agent F implementation):
//
//	func (pb *OSAPromptBuilder) BuildStandardVariables(
//	    appName string,
//	    description string,
//	    features []string,
//	    complexity string,
//	) map[string]interface{}
//
// Table-driven test cases:
//  1. All fields populated: keys exist, features_list joined with ", "
//  2. Empty features slice: features_list == "none specified"
//  3. Empty complexity: "complexity" key is absent from the map
//  4. Features slice is joined with ", "
//  5. Timestamp key is always present and non-empty
func TestBuildStandardVariables(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	pb, err := NewOSAPromptBuilder(nil, logger)
	require.NoError(t, err)

	type tc struct {
		name        string
		appName     string
		description string
		features    []string
		complexity  string
		check       func(t *testing.T, vars map[string]interface{})
	}

	cases := []tc{
		{
			name:        "all fields populated",
			appName:     "CRM App",
			description: "Lead tracking for real estate",
			complexity:  "medium",
			features:    []string{"Search", "Export"},
			check: func(t *testing.T, vars map[string]interface{}) {
				assert.Equal(t, "CRM App", vars["app_name"])
				assert.Equal(t, "Lead tracking for real estate", vars["description"])
				assert.Equal(t, "Search, Export", vars["features_list"])
				assert.Equal(t, "medium", vars["complexity"])
				assert.NotEmpty(t, vars["timestamp"], "timestamp must be set")
			},
		},
		{
			name:        "empty features produces none specified",
			appName:     "SaaS App",
			description: "Finance portfolio tracker",
			complexity:  "low",
			features:    []string{},
			check: func(t *testing.T, vars map[string]interface{}) {
				assert.Equal(t, "none specified", vars["features_list"])
			},
		},
		{
			name:        "empty complexity omits key",
			appName:     "API",
			description: "REST backend",
			complexity:  "",
			features:    []string{"Auth"},
			check: func(t *testing.T, vars map[string]interface{}) {
				_, present := vars["complexity"]
				assert.False(t, present, "empty complexity must not appear in map")
			},
		},
		{
			name:        "features joined with comma-space",
			appName:     "Dashboard",
			description: "Analytics charts",
			complexity:  "high",
			features:    []string{"Pie Chart", "Bar Chart", "Line Chart"},
			check: func(t *testing.T, vars map[string]interface{}) {
				assert.Equal(t, "Pie Chart, Bar Chart, Line Chart", vars["features_list"])
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			vars := pb.BuildStandardVariables(tc.appName, tc.description, tc.features, tc.complexity)
			require.NotNil(t, vars)
			tc.check(t, vars)
		})
	}
}

func TestTemplateOutputQuality(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	pb, err := NewOSAPromptBuilder(nil, logger)
	require.NoError(t, err)

	ctx := context.Background()

	testCases := []struct {
		name         string
		templateName string
		variables    map[string]interface{}
		mustContain  []string
	}{
		{
			name:         "CRM with comprehensive requirements",
			templateName: "crm-app-generation",
			variables: map[string]interface{}{
				"AppType":          "CRM",
				"UserBusiness":     "Real Estate",
				"UserRequirements": "Lead tracking with automated follow-ups",
			},
			mustContain: []string{
				"CRM",
				"Real Estate",
				"Handler → Service → Repository",
				"slog",
				"NO `fmt.Printf`",
				"Context as first parameter",
				"Minimum 80% Coverage",
			},
		},
		{
			name:         "Data Pipeline realtime",
			templateName: "data-pipeline-creation",
			variables: map[string]interface{}{
				"SourceType":          "Kafka",
				"DestinationType":     "PostgreSQL",
				"TransformationRules": "Aggregate events by user",
				"Schedule":            "realtime",
			},
			mustContain: []string{
				"Kafka",
				"Real-time streaming",
				"Exactly-once delivery",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := AppGenerationRequest{
				TemplateName: tc.templateName,
				Variables:    tc.variables,
			}

			result, err := pb.BuildAppGenerationPrompt(ctx, req)
			assert.NoError(t, err)

			for _, keyword := range tc.mustContain {
				assert.True(t,
					strings.Contains(result.Prompt, keyword),
					"Prompt should contain '%s'", keyword,
				)
			}

			t.Logf("✅ Template %s rendered correctly with %d characters",
				tc.templateName, len(result.Prompt))
		})
	}
}
