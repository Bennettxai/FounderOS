package services

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// AppMetadata represents extracted metadata from a deployed application
type AppMetadata struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Category    string   `json:"category"`
	Icon        string   `json:"icon"`
	Keywords    []string `json:"keywords"`
}

// ExtractAppMetadata parses package.json and infers category/icon
// It supports both file system paths and bundle content (bundleContent parameter)
func ExtractAppMetadata(appPath string, bundleContent map[string]string) (*AppMetadata, error) {
	var pkgData []byte
	var err error

	// 1. Try to parse package.json from bundleContent first
	if bundleContent != nil {
		if content, ok := bundleContent["package.json"]; ok {
			pkgData = []byte(content)
		} else if content, ok := bundleContent["frontend/package.json"]; ok {
			pkgData = []byte(content)
		}
	}

	// 2. Fallback to reading from file system if bundleContent is empty
	if pkgData == nil && appPath != "" {
		pkgPath := filepath.Join(appPath, "package.json")
		pkgData, err = os.ReadFile(pkgPath)
		if err != nil {
			// Try frontend/package.json (monorepo)
			pkgPath = filepath.Join(appPath, "frontend", "package.json")
			pkgData, err = os.ReadFile(pkgPath)
			if err != nil {
				return defaultMetadata(appPath), nil // Return defaults instead of error
			}
		}
	}

	// 3. If still no package.json found, return defaults
	if pkgData == nil {
		return defaultMetadata(appPath), nil
	}

	// 4. Parse package.json
	var pkg map[string]interface{}
	if err := json.Unmarshal(pkgData, &pkg); err != nil {
		// Malformed JSON - return defaults
		return defaultMetadata(appPath), nil
	}

	name := getString(pkg, "name")
	description := getString(pkg, "description")
	keywords := getStringSlice(pkg, "keywords")

	// 5. Infer category from package.json + bundle content analysis
	category := inferCategory(keywords, name, description, bundleContent)

	// 6. Map category to icon
	icon := categoryToIcon(category)

	// 7. Generate description if missing
	if description == "" {
		description = generateDescription(name, category, keywords)
	}

	// 8. Truncate description to 200 characters
	description = truncateDescription(description, 200)

	return &AppMetadata{
		Name:        cleanName(name),
		Description: description,
		Category:    category,
		Icon:        icon,
		Keywords:    keywords,
	}, nil
}

// defaultMetadata returns sensible defaults when package.json is missing
func defaultMetadata(appPath string) *AppMetadata {
	name := "Generated App"
	if appPath != "" {
		name = filepath.Base(appPath)
	}

	return &AppMetadata{
		Name:        cleanName(name),
		Description: "A generated application",
		Category:    "general",
		Icon:        "AppWindow",
		Keywords:    []string{},
	}
}

// inferCategory determines the app category from keywords, name, description, and bundle content
func inferCategory(keywords []string, name, description string, bundleContent map[string]string) string {
	// Build search text from package.json fields
	text := strings.ToLower(name + " " + description + " " + strings.Join(keywords, " "))

	// Also analyze bundle content for additional context
	if bundleContent != nil {
		// Look for README or documentation
		for path, content := range bundleContent {
			lowerPath := strings.ToLower(path)
			if strings.Contains(lowerPath, "readme") || strings.Contains(lowerPath, "description") {
				// Add first 500 chars of README to analysis
				if len(content) > 500 {
					text += " " + strings.ToLower(content[:500])
				} else {
					text += " " + strings.ToLower(content)
				}
				break
			}
		}
	}

	// Category patterns with fuzzy keyword matching
	patterns := map[string][]string{
		"finance":       {"invoice", "billing", "payment", "accounting", "expense", "budget", "money", "financial", "stripe", "paypal", "transaction"},
		"communication": {"chat", "messaging", "email", "slack", "discord", "conversation", "message", "notification", "sms", "whatsapp"},
		"productivity":  {"todo", "task", "calendar", "notes", "reminder", "schedule", "planner", "organizer", "notebook", "agenda"},
		"analytics":     {"dashboard", "analytics", "metrics", "reporting", "chart", "graph", "stats", "data", "visualization", "insight"},
		"ecommerce":     {"shop", "store", "cart", "product", "checkout", "ecommerce", "marketplace", "inventory", "order", "catalog"},
		"crm":           {"crm", "customer", "contact", "lead", "sales", "deal", "pipeline", "opportunity", "account", "client"},
		"hr":            {"employee", "hr", "payroll", "recruitment", "hiring", "onboarding", "leave", "attendance", "performance"},
		"inventory":     {"inventory", "stock", "warehouse", "asset", "equipment", "tracking", "supply", "logistics"},
		"marketing":     {"marketing", "campaign", "seo", "content", "social media", "advertising", "newsletter", "promotion"},
		"project":       {"project", "milestone", "sprint", "agile", "scrum", "kanban", "board", "issue", "ticket", "workflow"},
	}

	// Score each category based on keyword matches
	scores := make(map[string]int)
	for category, terms := range patterns {
		for _, term := range terms {
			if strings.Contains(text, term) {
				scores[category]++
			}
		}
	}

	// Return category with highest score
	var bestCategory string
	var bestScore int
	for category, score := range scores {
		if score > bestScore {
			bestScore = score
			bestCategory = category
		}
	}

	if bestCategory != "" {
		return bestCategory
	}

	return "general"
}

// generateDescription creates a description when none is provided
func generateDescription(name, category string, keywords []string) string {
	categoryDescriptions := map[string]string{
		"finance":       "A financial management application",
		"communication": "A communication and messaging tool",
		"productivity":  "A productivity and organization tool",
		"analytics":     "An analytics and reporting dashboard",
		"ecommerce":     "An e-commerce platform",
		"crm":           "A customer relationship management system",
		"hr":            "A human resources management tool",
		"inventory":     "An inventory management system",
		"marketing":     "A marketing automation platform",
		"project":       "A project management tool",
		"general":       "A custom application",
	}

	desc := categoryDescriptions[category]

	// Add keywords context if available
	if len(keywords) > 0 {
		desc += " for " + strings.Join(keywords[:minInt(3, len(keywords))], ", ")
	}

	return desc
}

// truncateDescription ensures description doesn't exceed maxLength
func truncateDescription(desc string, maxLength int) string {
	if len(desc) <= maxLength {
		return desc
	}

	// Truncate at word boundary
	truncated := desc[:maxLength]
	lastSpace := strings.LastIndex(truncated, " ")
	if lastSpace > 0 {
		truncated = truncated[:lastSpace]
	}
	return truncated + "..."
}

// minInt returns the smaller of two integers
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// categoryToIcon maps categories to Lucide icon names
func categoryToIcon(category string) string {
	icons := map[string]string{
		"finance":       "DollarSign",
		"communication": "MessageSquare",
		"productivity":  "Calendar",
		"analytics":     "BarChart",
		"ecommerce":     "ShoppingCart",
		"crm":           "Users",
		"hr":            "UserCheck",
		"inventory":     "Package",
		"marketing":     "Megaphone",
		"project":       "FolderKanban",
		"general":       "AppWindow",
	}

	if icon, ok := icons[category]; ok {
		return icon
	}
	return "AppWindow"
}

// cleanName converts npm package names to human-readable titles
func cleanName(name string) string {
	// Remove npm scope (@company/)
	name = strings.TrimPrefix(name, "@")
	if idx := strings.Index(name, "/"); idx != -1 {
		name = name[idx+1:]
	}

	// Convert kebab-case to Title Case
	parts := strings.Split(name, "-")
	for i, part := range parts {
		if len(part) > 0 {
			parts[i] = strings.ToUpper(part[:1]) + part[1:]
		}
	}
	return strings.Join(parts, " ")
}

// getString safely extracts a string value from a map
func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

// getStringSlice safely extracts a string slice from a map
func getStringSlice(m map[string]interface{}, key string) []string {
	if v, ok := m[key].([]interface{}); ok {
		result := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				result = append(result, s)
			}
		}
		return result
	}
	return []string{}
}
