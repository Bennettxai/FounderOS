package services

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtractAppMetadata_WithBundleContent(t *testing.T) {
	tests := []struct {
		name           string
		bundleContent  map[string]string
		expectedName   string
		expectedCat    string
		expectedIcon   string
		wantErr        bool
	}{
		{
			name: "Invoice app from bundle",
			bundleContent: map[string]string{
				"package.json": `{
					"name": "@acme/invoice-generator",
					"description": "Generate professional invoices",
					"keywords": ["invoice", "billing", "payment"]
				}`,
			},
			expectedName: "Invoice Generator",
			expectedCat:  "finance",
			expectedIcon: "DollarSign",
			wantErr:      false,
		},
		{
			name: "Chat app from bundle",
			bundleContent: map[string]string{
				"package.json": `{
					"name": "realtime-chat",
					"description": "Real-time messaging platform",
					"keywords": ["chat", "messaging", "websocket"]
				}`,
			},
			expectedName: "Realtime Chat",
			expectedCat:  "communication",
			expectedIcon: "MessageSquare",
			wantErr:      false,
		},
		{
			name: "Todo app from bundle",
			bundleContent: map[string]string{
				"package.json": `{
					"name": "task-manager",
					"description": "Manage your daily tasks",
					"keywords": ["todo", "tasks", "productivity"]
				}`,
			},
			expectedName: "Task Manager",
			expectedCat:  "productivity",
			expectedIcon: "Calendar",
			wantErr:      false,
		},
		{
			name: "Analytics dashboard from bundle",
			bundleContent: map[string]string{
				"package.json": `{
					"name": "business-dashboard",
					"description": "Analytics and reporting dashboard",
					"keywords": ["analytics", "dashboard", "metrics"]
				}`,
			},
			expectedName: "Business Dashboard",
			expectedCat:  "analytics",
			expectedIcon: "BarChart",
			wantErr:      false,
		},
		{
			name: "E-commerce from bundle",
			bundleContent: map[string]string{
				"package.json": `{
					"name": "online-store",
					"description": "Modern e-commerce platform",
					"keywords": ["ecommerce", "shop", "cart"]
				}`,
			},
			expectedName: "Online Store",
			expectedCat:  "ecommerce",
			expectedIcon: "ShoppingCart",
			wantErr:      false,
		},
		{
			name: "CRM from bundle",
			bundleContent: map[string]string{
				"package.json": `{
					"name": "customer-crm",
					"description": "Customer relationship management",
					"keywords": ["crm", "sales", "pipeline"]
				}`,
			},
			expectedName: "Customer Crm",
			expectedCat:  "crm",
			expectedIcon: "Users",
			wantErr:      false,
		},
		{
			name:           "No package.json - returns defaults",
			bundleContent:  map[string]string{},
			expectedName:   "Generated App",
			expectedCat:    "general",
			expectedIcon:   "AppWindow",
			wantErr:        false,
		},
		{
			name: "Malformed JSON - returns defaults",
			bundleContent: map[string]string{
				"package.json": `{invalid json`,
			},
			expectedName: "Generated App",
			expectedCat:  "general",
			expectedIcon: "AppWindow",
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metadata, err := ExtractAppMetadata("", tt.bundleContent)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.expectedName, metadata.Name)
			assert.Equal(t, tt.expectedCat, metadata.Category)
			assert.Equal(t, tt.expectedIcon, metadata.Icon)
			assert.NotEmpty(t, metadata.Description)
		})
	}
}

func TestExtractAppMetadata_WithREADME(t *testing.T) {
	bundleContent := map[string]string{
		"package.json": `{
			"name": "my-app",
			"keywords": []
		}`,
		"README.md": `# Invoice Generator

		This is a professional invoice generation tool that helps you create and manage invoices,
		track payments, and handle billing for your business.`,
	}

	metadata, err := ExtractAppMetadata("", bundleContent)

	assert.NoError(t, err)
	assert.Equal(t, "finance", metadata.Category, "Should infer category from README content")
	assert.Equal(t, "DollarSign", metadata.Icon)
}

func TestInferCategory_Scoring(t *testing.T) {
	tests := []struct {
		name        string
		keywords    []string
		appName     string
		description string
		expected    string
	}{
		{
			name:        "Multiple finance keywords",
			keywords:    []string{"invoice", "payment", "billing"},
			appName:     "invoice-app",
			description: "Generate invoices",
			expected:    "finance",
		},
		{
			name:        "Multiple communication keywords",
			keywords:    []string{"chat", "messaging", "slack"},
			appName:     "chat-app",
			description: "Real-time messaging",
			expected:    "communication",
		},
		{
			name:        "Conflicting keywords - highest score wins",
			keywords:    []string{"task", "todo", "chat"},
			appName:     "productivity-app",
			description: "Manage tasks and organize your day",
			expected:    "productivity", // 4 matches vs 1 for communication
		},
		{
			name:        "No matches - returns general",
			keywords:    []string{"widget", "thing"},
			appName:     "generic-app",
			description: "A generic application",
			expected:    "general",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			category := inferCategory(tt.keywords, tt.appName, tt.description, nil)
			assert.Equal(t, tt.expected, category)
		})
	}
}

func TestCategoryToIcon(t *testing.T) {
	tests := []struct {
		category string
		expected string
	}{
		{"finance", "DollarSign"},
		{"communication", "MessageSquare"},
		{"productivity", "Calendar"},
		{"analytics", "BarChart"},
		{"ecommerce", "ShoppingCart"},
		{"crm", "Users"},
		{"hr", "UserCheck"},
		{"inventory", "Package"},
		{"marketing", "Megaphone"},
		{"project", "FolderKanban"},
		{"general", "AppWindow"},
		{"unknown", "AppWindow"}, // Fallback
	}

	for _, tt := range tests {
		t.Run(tt.category, func(t *testing.T) {
			icon := categoryToIcon(tt.category)
			assert.Equal(t, tt.expected, icon)
		})
	}
}

func TestCleanName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"@acme/invoice-generator", "Invoice Generator"},
		{"task-manager", "Task Manager"},
		{"my-awesome-app", "My Awesome App"},
		{"simple", "Simple"},
		{"@org/multi-word-package-name", "Multi Word Package Name"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := cleanName(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGenerateDescription(t *testing.T) {
	tests := []struct {
		name     string
		category string
		keywords []string
		contains string
	}{
		{
			name:     "invoice-app",
			category: "finance",
			keywords: []string{"billing", "payment"},
			contains: "financial management",
		},
		{
			name:     "chat-app",
			category: "communication",
			keywords: []string{"messaging", "realtime"},
			contains: "communication",
		},
		{
			name:     "generic-app",
			category: "general",
			keywords: []string{},
			contains: "custom application",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			desc := generateDescription(tt.name, tt.category, tt.keywords)
			assert.Contains(t, desc, tt.contains)
		})
	}
}

func TestTruncateDescription(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		maxLength int
		expected  string
	}{
		{
			name:      "Short description - no truncation",
			input:     "A simple app",
			maxLength: 200,
			expected:  "A simple app",
		},
		{
			name:      "Long description - truncated at word boundary",
			input:     "This is a very long description that exceeds the maximum allowed length and needs to be truncated properly at a word boundary to maintain readability",
			maxLength: 50,
			expected:  "This is a very long description that exceeds...",
		},
		{
			name:      "Exactly at limit",
			input:     "Exactly fifty characters in this description!!",
			maxLength: 50,
			expected:  "Exactly fifty characters in this description!!", // No truncation
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := truncateDescription(tt.input, tt.maxLength)
			assert.LessOrEqual(t, len(result), tt.maxLength+3) // +3 for "..."
			if len(tt.input) > tt.maxLength {
				assert.Contains(t, result, "...")
			}
		})
	}
}

func TestDefaultMetadata(t *testing.T) {
	tests := []struct {
		name         string
		appPath      string
		expectedName string
	}{
		{
			name:         "With app path",
			appPath:      "/path/to/my-app",
			expectedName: "My App",
		},
		{
			name:         "Empty path",
			appPath:      "",
			expectedName: "Generated App",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metadata := defaultMetadata(tt.appPath)
			assert.Equal(t, tt.expectedName, metadata.Name)
			assert.Equal(t, "general", metadata.Category)
			assert.Equal(t, "AppWindow", metadata.Icon)
			assert.NotEmpty(t, metadata.Description)
		})
	}
}

func TestExtractAppMetadata_DescriptionLength(t *testing.T) {
	longDescription := "This is an extremely long description that goes on and on about all the amazing features of this application including but not limited to task management, collaboration, real-time updates, notifications, and much more that should be truncated to exactly 200 characters"

	bundleContent := map[string]string{
		"package.json": `{
			"name": "test-app",
			"description": "` + longDescription + `",
			"keywords": ["productivity"]
		}`,
	}

	metadata, err := ExtractAppMetadata("", bundleContent)

	assert.NoError(t, err)
	assert.LessOrEqual(t, len(metadata.Description), 203, "Description should be truncated to 200 chars + '...'")
}

func TestExtractAppMetadata_MonorepoPackageJson(t *testing.T) {
	bundleContent := map[string]string{
		"frontend/package.json": `{
			"name": "frontend-app",
			"description": "Frontend for invoice app",
			"keywords": ["invoice", "billing"]
		}`,
	}

	metadata, err := ExtractAppMetadata("", bundleContent)

	assert.NoError(t, err)
	assert.Equal(t, "Frontend App", metadata.Name)
	assert.Equal(t, "finance", metadata.Category)
}

func TestGetString(t *testing.T) {
	m := map[string]interface{}{
		"string_field": "value",
		"number_field": 123,
		"bool_field":   true,
	}

	assert.Equal(t, "value", getString(m, "string_field"))
	assert.Equal(t, "", getString(m, "number_field"))
	assert.Equal(t, "", getString(m, "bool_field"))
	assert.Equal(t, "", getString(m, "missing_field"))
}

func TestGetStringSlice(t *testing.T) {
	m := map[string]interface{}{
		"string_array": []interface{}{"a", "b", "c"},
		"mixed_array":  []interface{}{"a", 123, "b"},
		"not_array":    "string",
		"empty_array":  []interface{}{},
	}

	assert.Equal(t, []string{"a", "b", "c"}, getStringSlice(m, "string_array"))
	assert.Equal(t, []string{"a", "b"}, getStringSlice(m, "mixed_array"))
	assert.Equal(t, []string{}, getStringSlice(m, "not_array"))
	assert.Equal(t, []string{}, getStringSlice(m, "empty_array"))
	assert.Equal(t, []string{}, getStringSlice(m, "missing_field"))
}
