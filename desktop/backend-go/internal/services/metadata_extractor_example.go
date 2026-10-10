package services

// Example usage of ExtractAppMetadata
func ExampleExtractAppMetadata() {
	// Example 1: Invoice app from bundle content
	bundleContent := map[string]string{
		"package.json": `{
			"name": "@acme/invoice-generator",
			"description": "Generate professional invoices with ease",
			"keywords": ["invoice", "billing", "payment", "accounting"]
		}`,
		"README.md": `# Invoice Generator

		A comprehensive invoicing solution for small businesses.
		Features include payment tracking, recurring billing, and expense management.`,
	}

	metadata, _ := ExtractAppMetadata("", bundleContent)

	// Output:
	// Name: "Invoice Generator"
	// Category: "finance"
	// Icon: "DollarSign"
	// Description: "Generate professional invoices with ease"
	// Keywords: ["invoice", "billing", "payment", "accounting"]
	_ = metadata
}

// Example usage with file system path (legacy mode)
func ExampleExtractAppMetadata_withPath() {
	// This works when the app is already deployed to disk
	appPath := "/tmp/businessos-apps/some-app-id"

	// Pass nil for bundleContent to use file system
	metadata, _ := ExtractAppMetadata(appPath, nil)

	_ = metadata
}

// Example showing graceful fallback with missing package.json
func ExampleExtractAppMetadata_defaults() {
	// Empty bundle - no package.json
	bundleContent := map[string]string{
		"index.html": "<html>...</html>",
	}

	metadata, _ := ExtractAppMetadata("/tmp/my-app", bundleContent)

	// Returns sensible defaults:
	// Name: "My App"
	// Category: "general"
	// Icon: "AppWindow"
	// Description: "A custom application"
	_ = metadata
}
