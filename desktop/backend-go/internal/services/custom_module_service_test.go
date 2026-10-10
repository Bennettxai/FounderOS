package services

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCustomModuleService_CreateModule tests module creation with various scenarios
func TestCustomModuleService_CreateModule(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// TODO: Initialize test database pool
	// This would require setting up testutil package
	t.Run("successfully creates module with valid manifest", func(t *testing.T) {
		t.Skip("Requires test database setup")

		// Setup
		// pool := testutil.RequireTestDatabase(t)
		// defer pool.Close()
		// service := NewCustomModuleService(pool, slog.Default())

		// ctx := context.Background()
		// workspaceID := uuid.New()
		// userID := uuid.New()

		// req := CreateModuleRequest{
		// 	Name:        "Test Module",
		// 	Description: "A test module for unit testing",
		// 	Category:    "utility",
		// 	Manifest: map[string]interface{}{
		// 		"actions": []interface{}{
		// 			map[string]interface{}{
		// 				"name": "test_action",
		// 				"type": "api_call",
		// 			},
		// 		},
		// 	},
		// 	Config: map[string]interface{}{
		// 		"api_url": "https://api.example.com",
		// 	},
		// 	Icon: "📦",
		// 	Tags: []string{"test", "utility"},
		// 	Keywords: []string{"testing", "example"},
		// }

		// Execute
		// module, err := service.CreateModule(ctx, workspaceID, userID, req)

		// Assert
		// require.NoError(t, err)
		// assert.NotEqual(t, uuid.Nil, module.ID)
		// assert.Equal(t, "Test Module", module.Name)
		// assert.Equal(t, "test-module", module.Slug)
		// assert.Equal(t, "0.0.1", module.Version)
		// assert.Equal(t, workspaceID, module.WorkspaceID)
		// assert.Equal(t, userID, module.CreatedBy)
	})

	t.Run("fails with invalid manifest - missing actions", func(t *testing.T) {
		t.Skip("Requires test database setup")

		// Setup with invalid manifest
		// req := CreateModuleRequest{
		// 	Name:     "Invalid Module",
		// 	Category: "utility",
		// 	Manifest: map[string]interface{}{}, // No actions field
		// }

		// Execute
		// _, err := service.CreateModule(ctx, workspaceID, userID, req)

		// Assert
		// assert.Error(t, err)
		// assert.Contains(t, err.Error(), "manifest must contain 'actions' field")
	})

	t.Run("accepts simple manifest with empty actions array", func(t *testing.T) {
		err := validateManifest(map[string]interface{}{
			"actions": []interface{}{},
		})
		assert.NoError(t, err)
	})

	t.Run("fails with invalid manifest - action missing name", func(t *testing.T) {
		// Test action validation
		err := validateManifest(map[string]interface{}{
			"actions": []interface{}{
				map[string]interface{}{
					"type": "api_call", // Missing name field
				},
			},
		})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "must have a 'name' field")
	})

	t.Run("fails with invalid manifest - action missing type", func(t *testing.T) {
		// Test action validation
		err := validateManifest(map[string]interface{}{
			"actions": []interface{}{
				map[string]interface{}{
					"name": "test_action", // Missing type field
				},
			},
		})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "must have a 'type' field")
	})

	t.Run("creates module with default config when not provided", func(t *testing.T) {
		t.Skip("Requires test database setup")

		// Test that empty config is handled properly
		// req := CreateModuleRequest{
		// 	Name:     "Module No Config",
		// 	Category: "utility",
		// 	Manifest: map[string]interface{}{
		// 		"actions": []interface{}{
		// 			map[string]interface{}{
		// 				"name": "action1",
		// 				"type": "api_call",
		// 			},
		// 		},
		// 	},
		// 	// Config is nil
		// }

		// module, err := service.CreateModule(ctx, workspaceID, userID, req)

		// require.NoError(t, err)
		// assert.NotNil(t, module.Config)
		// assert.Empty(t, module.Config)
	})
}

// TestCustomModuleService_GetModule tests retrieving a single module
func TestCustomModuleService_GetModule(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	t.Run("successfully retrieves existing module", func(t *testing.T) {
		t.Skip("Requires test database setup")
	})

	t.Run("returns error for non-existent module", func(t *testing.T) {
		t.Skip("Requires test database setup")

		// Execute with random UUID
		// _, err := service.GetModule(ctx, uuid.New())

		// Assert
		// assert.Error(t, err)
		// assert.Contains(t, err.Error(), "module not found")
	})
}

// TestCustomModuleService_ListModules tests listing modules in a workspace
func TestCustomModuleService_ListModules(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	t.Run("lists all modules in workspace", func(t *testing.T) {
		t.Skip("Requires test database setup")

		// Create multiple modules first
		// Then list them
	})

	t.Run("returns empty list for workspace with no modules", func(t *testing.T) {
		t.Skip("Requires test database setup")

		// Execute
		// modules, err := service.ListModules(ctx, uuid.New(), 20, 0)

		// Assert
		// require.NoError(t, err)
		// assert.Empty(t, modules)
	})

	t.Run("respects limit and offset for pagination", func(t *testing.T) {
		t.Skip("Requires test database setup")

		// Create 5 modules, list with limit 2, offset 1
		// Verify only 2 results returned and correct offset
	})

	t.Run("defaults to limit 20 when limit is 0", func(t *testing.T) {
		t.Skip("Requires test database setup")

		// Test default limit behavior
	})
}

// TestCustomModuleService_UpdateModule tests module updates
func TestCustomModuleService_UpdateModule(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	t.Run("successfully updates module name", func(t *testing.T) {
		t.Skip("Requires test database setup")

		// Create module, update name, verify slug also updated
	})

	t.Run("successfully updates description", func(t *testing.T) {
		t.Skip("Requires test database setup")
	})

	t.Run("successfully updates manifest", func(t *testing.T) {
		t.Skip("Requires test database setup")
	})

	t.Run("successfully updates multiple fields at once", func(t *testing.T) {
		t.Skip("Requires test database setup")

		// Update name, description, and config simultaneously
	})

	t.Run("returns unchanged module when no updates provided", func(t *testing.T) {
		t.Skip("Requires test database setup")

		// Execute with empty UpdateModuleRequest
		// Verify module unchanged
	})

	t.Run("fails when user is not owner", func(t *testing.T) {
		t.Skip("Requires test database setup")

		// Create module with userID1
		// Try to update with userID2
		// Assert error about unauthorized
	})

	t.Run("updates slug when name is updated", func(t *testing.T) {
		t.Skip("Requires test database setup")

		// Update name from "Test Module" to "New Module Name"
		// Verify slug changed from "test-module" to "new-module-name"
	})

	t.Run("updates updated_at timestamp", func(t *testing.T) {
		t.Skip("Requires test database setup")

		// Create module, note timestamp
		// Update module
		// Verify updated_at changed
	})
}

// TestCustomModuleService_DeleteModule tests module deletion
func TestCustomModuleService_DeleteModule(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	t.Run("successfully deletes module", func(t *testing.T) {
		t.Skip("Requires test database setup")

		// Create module, delete it, verify it's gone
	})

	t.Run("fails to delete when user is not owner", func(t *testing.T) {
		t.Skip("Requires test database setup")

		// Create module with userID1
		// Try to delete with userID2
		// Assert error
	})

	t.Run("returns error for non-existent module", func(t *testing.T) {
		t.Skip("Requires test database setup")

		// Execute delete with random UUID
		// Assert error about module not found
	})
}

// TestCustomModuleService_PublishModule tests publishing modules
func TestCustomModuleService_PublishModule(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	t.Run("successfully publishes module", func(t *testing.T) {
		t.Skip("Requires test database setup")

		// Create module (is_published=false)
		// Publish it
		// Verify is_published=true, is_public=true, published_at set
	})

	t.Run("fails when user is not owner", func(t *testing.T) {
		t.Skip("Requires test database setup")
	})

	t.Run("sets published_at timestamp", func(t *testing.T) {
		t.Skip("Requires test database setup")
	})
}

// TestCustomModuleService_SearchModules tests searching public modules
func TestCustomModuleService_SearchModules(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	t.Run("finds modules by name", func(t *testing.T) {
		t.Skip("Requires test database setup")

		// Create modules: "Email Module", "Calendar Module", "Slack Module"
		// Search "Email"
		// Verify only "Email Module" returned
	})

	t.Run("finds modules by description", func(t *testing.T) {
		t.Skip("Requires test database setup")

		// Create module with description containing "automation"
		// Search "automation"
		// Verify module found
	})

	t.Run("finds modules by tag", func(t *testing.T) {
		t.Skip("Requires test database setup")

		// Create module with tag "productivity"
		// Search "productivity"
		// Verify module found
	})

	t.Run("finds modules by keyword", func(t *testing.T) {
		t.Skip("Requires test database setup")

		// Create module with keyword "integration"
		// Search "integration"
		// Verify module found
	})

	t.Run("only returns public and published modules", func(t *testing.T) {
		t.Skip("Requires test database setup")

		// Create modules:
		// - Module A: public=true, published=true
		// - Module B: public=false, published=true
		// - Module C: public=true, published=false
		// Search for all
		// Verify only Module A returned
	})

	t.Run("orders by install_count DESC", func(t *testing.T) {
		t.Skip("Requires test database setup")

		// Create modules with different install_count
		// Search
		// Verify order is highest install_count first
	})

	t.Run("supports pagination", func(t *testing.T) {
		t.Skip("Requires test database setup")

		// Create 5 modules
		// Search with limit 2, offset 2
		// Verify correct results
	})

	t.Run("case insensitive search", func(t *testing.T) {
		t.Skip("Requires test database setup")

		// Create module "Email Module"
		// Search "EMAIL" or "email"
		// Verify module found
	})
}

// TestGenerateModuleSlug tests slug generation function
func TestGenerateModuleSlug(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple name",
			input:    "Test Module",
			expected: "test-module",
		},
		{
			name:     "multiple spaces",
			input:    "My   Awesome    Module",
			expected: "my-awesome-module",
		},
		{
			name:     "special characters",
			input:    "Test@Module#123!",
			expected: "testmodule123",
		},
		{
			name:     "leading and trailing spaces",
			input:    "  Test Module  ",
			expected: "test-module",
		},
		{
			name:     "consecutive hyphens",
			input:    "Test---Module",
			expected: "test-module",
		},
		{
			name:     "unicode characters",
			input:    "Módulo de Prueba",
			expected: "mdulo-de-prueba",
		},
		{
			name:     "mixed case",
			input:    "MyTestModule",
			expected: "mytestmodule",
		},
		{
			name:     "numbers",
			input:    "Module 2000",
			expected: "module-2000",
		},
		{
			name:     "only special characters",
			input:    "@#$%",
			expected: "",
		},
		{
			name:     "hyphens at start and end",
			input:    "-Test-Module-",
			expected: "test-module",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GenerateModuleSlug(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestValidateManifest tests manifest validation function
func TestValidateManifest(t *testing.T) {
	t.Run("valid manifest with single action", func(t *testing.T) {
		manifest := map[string]interface{}{
			"actions": []interface{}{
				map[string]interface{}{
					"name": "send_email",
					"type": "api_call",
				},
			},
		}
		err := validateManifest(manifest)
		assert.NoError(t, err)
	})

	t.Run("valid manifest with multiple actions", func(t *testing.T) {
		manifest := map[string]interface{}{
			"actions": []interface{}{
				map[string]interface{}{
					"name": "send_email",
					"type": "api_call",
				},
				map[string]interface{}{
					"name": "get_emails",
					"type": "api_call",
				},
			},
		}
		err := validateManifest(manifest)
		assert.NoError(t, err)
	})

	t.Run("valid simple module - missing actions field", func(t *testing.T) {
		manifest := map[string]interface{}{
			"version": "1.0.0",
		}
		err := validateManifest(manifest)
		assert.NoError(t, err)
	})

	t.Run("invalid - actions not an array", func(t *testing.T) {
		manifest := map[string]interface{}{
			"actions": "not an array",
		}
		err := validateManifest(manifest)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "manifest 'actions' must be an array")
	})

	t.Run("valid simple module - empty actions array", func(t *testing.T) {
		manifest := map[string]interface{}{
			"actions": []interface{}{},
		}
		err := validateManifest(manifest)
		assert.NoError(t, err)
	})

	t.Run("invalid - action not an object", func(t *testing.T) {
		manifest := map[string]interface{}{
			"actions": []interface{}{
				"not an object",
			},
		}
		err := validateManifest(manifest)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "action 0 must be an object")
	})

	t.Run("invalid - action missing name", func(t *testing.T) {
		manifest := map[string]interface{}{
			"actions": []interface{}{
				map[string]interface{}{
					"type": "api_call",
				},
			},
		}
		err := validateManifest(manifest)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "action 0 must have a 'name' field")
	})

	t.Run("invalid - action missing type", func(t *testing.T) {
		manifest := map[string]interface{}{
			"actions": []interface{}{
				map[string]interface{}{
					"name": "test_action",
				},
			},
		}
		err := validateManifest(manifest)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "action 0 must have a 'type' field")
	})

	t.Run("invalid - second action has error", func(t *testing.T) {
		manifest := map[string]interface{}{
			"actions": []interface{}{
				map[string]interface{}{
					"name": "action1",
					"type": "api_call",
				},
				map[string]interface{}{
					"name": "action2",
					// Missing type
				},
			},
		}
		err := validateManifest(manifest)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "action 1 must have a 'type' field")
	})
}

// TestManifestJSONMarshaling tests JSON marshaling/unmarshaling
func TestManifestJSONMarshaling(t *testing.T) {
	t.Run("successfully marshals complex manifest", func(t *testing.T) {
		manifest := map[string]interface{}{
			"version": "1.0.0",
			"actions": []interface{}{
				map[string]interface{}{
					"name":        "send_email",
					"type":        "api_call",
					"description": "Send an email",
					"parameters": map[string]interface{}{
						"to":      "string",
						"subject": "string",
						"body":    "string",
					},
				},
			},
		}

		data, err := json.Marshal(manifest)
		require.NoError(t, err)
		assert.NotEmpty(t, data)

		// Unmarshal back
		var result map[string]interface{}
		err = json.Unmarshal(data, &result)
		require.NoError(t, err)
		assert.Equal(t, "1.0.0", result["version"])
	})
}
