// Package integrations provides the integration provider registry.
// This follows the INTEGRATION_INFRASTRUCTURE.md architecture spec.
package integrations

import (
	"sort"
	"sync"
)

var (
	registry = make(map[string]Provider)
	mu       sync.RWMutex
)

// Register adds a provider to the registry.
// Should be called during init() in each provider package.
func Register(p Provider) {
	mu.Lock()
	defer mu.Unlock()
	registry[p.Name()] = p
}

// Get retrieves a provider by name.
// Returns the provider and true if found, nil and false otherwise.
func Get(name string) (Provider, bool) {
	mu.RLock()
	defer mu.RUnlock()
	p, ok := registry[name]
	return p, ok
}

// List returns all registered providers sorted by name.
func List() []Provider {
	mu.RLock()
	defer mu.RUnlock()
	result := make([]Provider, 0, len(registry))
	for _, p := range registry {
		result = append(result, p)
	}
	// Sort by name for consistent ordering
	sort.Slice(result, func(i, j int) bool {
		return result[i].Name() < result[j].Name()
	})
	return result
}

// ListByCategory returns providers in a specific category.
func ListByCategory(category string) []Provider {
	mu.RLock()
	defer mu.RUnlock()
	result := make([]Provider, 0)
	for _, p := range registry {
		if p.Category() == category {
			result = append(result, p)
		}
	}
	// Sort by name for consistent ordering
	sort.Slice(result, func(i, j int) bool {
		return result[i].Name() < result[j].Name()
	})
	return result
}

// ListNames returns the names of all registered providers.
func ListNames() []string {
	mu.RLock()
	defer mu.RUnlock()
	names := make([]string, 0, len(registry))
	for name := range registry {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// Count returns the number of registered providers.
func Count() int {
	mu.RLock()
	defer mu.RUnlock()
	return len(registry)
}

// Exists checks if a provider is registered.
func Exists(name string) bool {
	mu.RLock()
	defer mu.RUnlock()
	_, ok := registry[name]
	return ok
}

// Categories returns a list of unique categories from registered providers.
func Categories() []string {
	mu.RLock()
	defer mu.RUnlock()
	categorySet := make(map[string]bool)
	for _, p := range registry {
		categorySet[p.Category()] = true
	}
	categories := make([]string, 0, len(categorySet))
	for cat := range categorySet {
		categories = append(categories, cat)
	}
	sort.Strings(categories)
	return categories
}

// ProviderInfoList returns metadata about all registered providers.
func ProviderInfoList() []ProviderInfo {
	mu.RLock()
	defer mu.RUnlock()
	result := make([]ProviderInfo, 0, len(registry))
	for _, p := range registry {
		result = append(result, ProviderInfo{
			Name:        p.Name(),
			DisplayName: p.DisplayName(),
			Category:    p.Category(),
			Icon:        p.Icon(),
		})
	}
	// Sort by display name for UI
	sort.Slice(result, func(i, j int) bool {
		return result[i].DisplayName < result[j].DisplayName
	})
	return result
}

// Clear removes all providers from the registry.
// Used primarily for testing.
func Clear() {
	mu.Lock()
	defer mu.Unlock()
	registry = make(map[string]Provider)
}
