package services

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Mock service since we don't have the actual implementation yet
type MockModuleExportService struct {
	pool   interface{}
	logger *slog.Logger
}

func NewMockModuleExportService(pool interface{}, logger *slog.Logger) *MockModuleExportService {
	return &MockModuleExportService{
		pool:   pool,
		logger: logger,
	}
}

func (s *MockModuleExportService) ExportModule(ctx context.Context, moduleID uuid.UUID) (*bytes.Buffer, error) {
	// Mock implementation
	return nil, nil
}

// TestModuleExportService_ExportModule tests ZIP generation for modules
func TestModuleExportService_ExportModule(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	t.Run("successfully exports module as ZIP", func(t *testing.T) {
		t.Skip("Requires test database setup and ModuleExportService implementation")

		// Setup
		// pool := testutil.RequireTestDatabase(t)
		// defer pool.Close()
		// service := NewModuleExportService(pool, slog.Default())
		// ctx := context.Background()

		// Create a test module
		// moduleID := createTestModule(t, pool)

		// Execute export
		// zipBuffer, err := service.ExportModule(ctx, moduleID)

		// Assert
		// require.NoError(t, err)
		// assert.NotNil(t, zipBuffer)
		// assert.Greater(t, zipBuffer.Len(), 0)

		// Verify ZIP structure
		// reader := bytes.NewReader(zipBuffer.Bytes())
		// zipReader, err := zip.NewReader(reader, int64(zipBuffer.Len()))
		// require.NoError(t, err)

		// // Check for required files
		// files := make(map[string]bool)
		// for _, file := range zipReader.File {
		// 	files[file.Name] = true
		// }

		// assert.True(t, files["manifest.json"], "ZIP should contain manifest.json")
		// assert.True(t, files["module.json"], "ZIP should contain module.json")
	})

	t.Run("ZIP contains valid manifest.json", func(t *testing.T) {
		t.Skip("Requires test database setup")

		// Export module
		// Extract manifest.json from ZIP
		// Parse JSON
		// Verify structure and required fields
	})

	t.Run("ZIP contains module metadata", func(t *testing.T) {
		t.Skip("Requires test database setup")

		// Export module
		// Extract module.json from ZIP
		// Verify it contains: id, name, version, description, etc.
	})

	t.Run("handles module with actions", func(t *testing.T) {
		t.Skip("Requires test database setup")

		// Create module with multiple actions
		// Export
		// Verify all actions included in manifest
	})

	t.Run("handles module with custom config", func(t *testing.T) {
		t.Skip("Requires test database setup")

		// Create module with complex config
		// Export
		// Verify config preserved in ZIP
	})

	t.Run("handles module with assets", func(t *testing.T) {
		t.Skip("Requires test database setup")

		// Create module with icon, images, etc.
		// Export
		// Verify assets included in ZIP
	})

	t.Run("fails for non-existent module", func(t *testing.T) {
		t.Skip("Requires test database setup")

		// Execute export with random UUID
		// Assert error
	})

	t.Run("fails for module user doesn't own", func(t *testing.T) {
		t.Skip("Requires test database setup")

		// Create module with userID1
		// Try to export as userID2
		// Assert authorization error
	})
}

// TestZIPStructure tests ZIP file structure and contents
func TestZIPStructure(t *testing.T) {
	t.Run("creates valid ZIP with correct file paths", func(t *testing.T) {
		// Create a mock ZIP buffer
		var buf bytes.Buffer
		zipWriter := zip.NewWriter(&buf)

		// Add files
		files := []struct {
			name    string
			content string
		}{
			{"manifest.json", `{"version":"1.0.0","actions":[]}`},
			{"module.json", `{"name":"Test Module"}`},
			{"README.md", "# Test Module"},
		}

		for _, f := range files {
			writer, err := zipWriter.Create(f.name)
			require.NoError(t, err)
			_, err = writer.Write([]byte(f.content))
			require.NoError(t, err)
		}

		err := zipWriter.Close()
		require.NoError(t, err)

		// Verify ZIP contents
		reader := bytes.NewReader(buf.Bytes())
		zipReader, err := zip.NewReader(reader, int64(buf.Len()))
		require.NoError(t, err)

		assert.Len(t, zipReader.File, 3)

		// Verify file names
		fileNames := make([]string, len(zipReader.File))
		for i, f := range zipReader.File {
			fileNames[i] = f.Name
		}
		assert.Contains(t, fileNames, "manifest.json")
		assert.Contains(t, fileNames, "module.json")
		assert.Contains(t, fileNames, "README.md")
	})

	t.Run("preserves JSON structure in manifest", func(t *testing.T) {
		// Create ZIP with complex manifest
		var buf bytes.Buffer
		zipWriter := zip.NewWriter(&buf)

		manifest := map[string]interface{}{
			"version": "1.0.0",
			"actions": []map[string]interface{}{
				{
					"name": "test_action",
					"type": "api_call",
					"parameters": map[string]string{
						"url": "string",
					},
				},
			},
		}

		manifestJSON, err := json.Marshal(manifest)
		require.NoError(t, err)

		writer, err := zipWriter.Create("manifest.json")
		require.NoError(t, err)
		_, err = writer.Write(manifestJSON)
		require.NoError(t, err)

		err = zipWriter.Close()
		require.NoError(t, err)

		// Extract and parse manifest
		reader := bytes.NewReader(buf.Bytes())
		zipReader, err := zip.NewReader(reader, int64(buf.Len()))
		require.NoError(t, err)

		file := zipReader.File[0]
		rc, err := file.Open()
		require.NoError(t, err)
		defer rc.Close()

		var extracted map[string]interface{}
		err = json.NewDecoder(rc).Decode(&extracted)
		require.NoError(t, err)

		assert.Equal(t, "1.0.0", extracted["version"])
		actions := extracted["actions"].([]interface{})
		assert.Len(t, actions, 1)
	})

	t.Run("handles binary assets in ZIP", func(t *testing.T) {
		// Test adding binary files (icons, images) to ZIP
		var buf bytes.Buffer
		zipWriter := zip.NewWriter(&buf)

		// Simulate binary data (e.g., PNG image)
		binaryData := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A} // PNG header

		writer, err := zipWriter.Create("assets/icon.png")
		require.NoError(t, err)
		_, err = writer.Write(binaryData)
		require.NoError(t, err)

		err = zipWriter.Close()
		require.NoError(t, err)

		// Verify binary data preserved
		reader := bytes.NewReader(buf.Bytes())
		zipReader, err := zip.NewReader(reader, int64(buf.Len()))
		require.NoError(t, err)

		file := zipReader.File[0]
		rc, err := file.Open()
		require.NoError(t, err)
		defer rc.Close()

		extractedData, err := io.ReadAll(rc)
		require.NoError(t, err)
		assert.Equal(t, binaryData, extractedData)
	})
}

// TestManifestSerialization tests manifest serialization for export
func TestManifestSerialization(t *testing.T) {
	t.Run("serializes manifest with all fields", func(t *testing.T) {
		manifest := map[string]interface{}{
			"version":     "1.0.0",
			"name":        "Test Module",
			"description": "A test module",
			"author":      "Test Author",
			"actions": []map[string]interface{}{
				{
					"name":        "send_email",
					"type":        "api_call",
					"description": "Send an email",
					"parameters": map[string]interface{}{
						"to":      map[string]string{"type": "string", "required": "true"},
						"subject": map[string]string{"type": "string"},
						"body":    map[string]string{"type": "string"},
					},
				},
			},
		}

		data, err := json.Marshal(manifest)
		require.NoError(t, err)
		assert.NotEmpty(t, data)

		// Verify deserialization
		var result map[string]interface{}
		err = json.Unmarshal(data, &result)
		require.NoError(t, err)
		assert.Equal(t, "1.0.0", result["version"])
		assert.Equal(t, "Test Module", result["name"])
	})

	t.Run("handles empty arrays", func(t *testing.T) {
		manifest := map[string]interface{}{
			"version": "1.0.0",
			"actions": []interface{}{},
		}

		data, err := json.Marshal(manifest)
		require.NoError(t, err)

		var result map[string]interface{}
		err = json.Unmarshal(data, &result)
		require.NoError(t, err)
		assert.Empty(t, result["actions"])
	})

	t.Run("preserves nested structures", func(t *testing.T) {
		manifest := map[string]interface{}{
			"config": map[string]interface{}{
				"api": map[string]interface{}{
					"endpoint": "https://api.example.com",
					"timeout":  30,
					"retry": map[string]interface{}{
						"enabled":    true,
						"maxRetries": 3,
					},
				},
			},
		}

		data, err := json.Marshal(manifest)
		require.NoError(t, err)

		var result map[string]interface{}
		err = json.Unmarshal(data, &result)
		require.NoError(t, err)

		config := result["config"].(map[string]interface{})
		api := config["api"].(map[string]interface{})
		assert.Equal(t, "https://api.example.com", api["endpoint"])
		assert.Equal(t, float64(30), api["timeout"])
	})
}

// TestZIPCompression tests ZIP compression levels
func TestZIPCompression(t *testing.T) {
	t.Run("compresses large text files effectively", func(t *testing.T) {
		// Create large text content
		largeText := make([]byte, 10000)
		for i := range largeText {
			largeText[i] = 'A'
		}

		var buf bytes.Buffer
		zipWriter := zip.NewWriter(&buf)

		writer, err := zipWriter.Create("large.txt")
		require.NoError(t, err)
		_, err = writer.Write(largeText)
		require.NoError(t, err)

		err = zipWriter.Close()
		require.NoError(t, err)

		// Compressed size should be much smaller than original
		assert.Less(t, buf.Len(), len(largeText)/2)
	})
}
