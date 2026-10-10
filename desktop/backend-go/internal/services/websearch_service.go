package services

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/rhl/businessos-backend/internal/config"
)

// ============================================================================
// TYPES AND INTERFACES
// ============================================================================

// WebSearchResult represents a single search result
type WebSearchResult struct {
	Title       string  `json:"title"`
	URL         string  `json:"url"`
	Snippet     string  `json:"snippet"`
	Source      string  `json:"source"`
	PublishedAt string  `json:"published_at,omitempty"`
	Score       float64 `json:"score,omitempty"` // Relevance score if available
}

// WebSearchResponse represents the full search response
type WebSearchResponse struct {
	Query        string            `json:"query"`
	Results      []WebSearchResult `json:"results"`
	TotalResults int               `json:"total_results"`
	SearchTime   float64           `json:"search_time_ms"`
	Provider     string            `json:"provider"`
	Cached       bool              `json:"cached,omitempty"`
}

// SearchProvider interface for different search backends
type SearchProvider interface {
	Search(ctx context.Context, query string, maxResults int) (*WebSearchResponse, error)
	Name() string
	Available() bool
}

// ============================================================================
// MULTI-PROVIDER SEARCH SERVICE
// ============================================================================

// WebSearchService is the main search service with multi-provider support
type WebSearchService struct {
	providers []SearchProvider
	primary   SearchProvider
	client    *http.Client
	timeout   time.Duration
	mu        sync.RWMutex
}

// NewWebSearchService creates a new web search service with providers based on config
func NewWebSearchService() *WebSearchService {
	cfg := config.AppConfig
	if cfg == nil {
		// Fallback if config not loaded
		return &WebSearchService{
			providers: []SearchProvider{NewDuckDuckGoProvider()},
			primary:   NewDuckDuckGoProvider(),
			client:    &http.Client{Timeout: 15 * time.Second},
			timeout:   15 * time.Second,
		}
	}

	return NewWebSearchServiceWithConfig(cfg)
}

// NewWebSearchServiceWithConfig creates a search service with explicit config
func NewWebSearchServiceWithConfig(cfg *config.Config) *WebSearchService {
	var providers []SearchProvider
	var primary SearchProvider

	// Add providers based on available API keys
	if cfg.HasBraveSearch() {
		provider := NewBraveSearchProvider(cfg.BraveSearchAPIKey)
		providers = append(providers, provider)
		if primary == nil {
			primary = provider
		}
	}

	if cfg.HasSerper() {
		provider := NewSerperProvider(cfg.SerperAPIKey)
		providers = append(providers, provider)
		if primary == nil {
			primary = provider
		}
	}

	if cfg.HasTavily() {
		provider := NewTavilyProvider(cfg.TavilyAPIKey)
		providers = append(providers, provider)
		if primary == nil {
			primary = provider
		}
	}

	// Always add DuckDuckGo as fallback
	ddg := NewDuckDuckGoProvider()
	providers = append(providers, ddg)
	if primary == nil {
		primary = ddg
	}

	// Override primary if explicitly set in config
	if cfg.SearchProvider != "" && cfg.SearchProvider != "auto" {
		for _, p := range providers {
			if p.Name() == cfg.SearchProvider && p.Available() {
				primary = p
				break
			}
		}
	}

	slog.Info("Web search service initialized",
		"primary", primary.Name(),
		"providers", len(providers),
		"available", getAvailableProviderNames(providers))

	return &WebSearchService{
		providers: providers,
		primary:   primary,
		client:    &http.Client{Timeout: 15 * time.Second},
		timeout:   15 * time.Second,
	}
}

// getAvailableProviderNames returns names of available providers
func getAvailableProviderNames(providers []SearchProvider) []string {
	names := make([]string, 0, len(providers))
	for _, p := range providers {
		if p.Available() {
			names = append(names, p.Name())
		}
	}
	return names
}

// Search performs a web search with automatic fallback
func (s *WebSearchService) Search(ctx context.Context, query string, maxResults int) (*WebSearchResponse, error) {
	if maxResults <= 0 {
		maxResults = 10
	}
	if maxResults > 20 {
		maxResults = 20
	}

	s.mu.RLock()
	primary := s.primary
	providers := s.providers
	s.mu.RUnlock()

	// Try primary provider first
	if primary != nil && primary.Available() {
		response, err := primary.Search(ctx, query, maxResults)
		if err == nil && len(response.Results) > 0 {
			return response, nil
		}
		slog.Warn("Primary search provider failed, trying fallbacks",
			"provider", primary.Name(),
			"error", err)
	}

	// Try fallback providers
	var lastErr error
	for _, provider := range providers {
		if provider == primary || !provider.Available() {
			continue
		}

		response, err := provider.Search(ctx, query, maxResults)
		if err == nil && len(response.Results) > 0 {
			slog.Info("Fallback search provider succeeded", "provider", provider.Name())
			return response, nil
		}
		lastErr = err
		slog.Warn("Fallback search provider failed",
			"provider", provider.Name(),
			"error", err)
	}

	if lastErr != nil {
		return nil, fmt.Errorf("all search providers failed: %w", lastErr)
	}
	return nil, fmt.Errorf("no search providers available")
}

// SearchWithProvider performs a search using a specific provider
func (s *WebSearchService) SearchWithProvider(ctx context.Context, providerName string, query string, maxResults int) (*WebSearchResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, provider := range s.providers {
		if provider.Name() == providerName && provider.Available() {
			return provider.Search(ctx, query, maxResults)
		}
	}

	return nil, fmt.Errorf("provider %q not found or not available", providerName)
}

// GetAvailableProviders returns list of available provider names
func (s *WebSearchService) GetAvailableProviders() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return getAvailableProviderNames(s.providers)
}

// GetPrimaryProvider returns the name of the primary provider
func (s *WebSearchService) GetPrimaryProvider() string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.primary != nil {
		return s.primary.Name()
	}
	return "none"
}

// SearchWithOptimization performs a search with automatic query optimization
func (s *WebSearchService) SearchWithOptimization(ctx context.Context, input string, maxResults int) (*WebSearchResponse, error) {
	optimizer := NewQueryOptimizer()
	optimizedQuery := optimizer.OptimizeQuery(input)

	if optimizedQuery == "" {
		optimizedQuery = input
	}

	return s.Search(ctx, optimizedQuery, maxResults)
}

// FormatResultsAsContext formats search results for injection into AI context
func (s *WebSearchService) FormatResultsAsContext(results *WebSearchResponse) string {
	if results == nil || len(results.Results) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("## Web Search Results for: %q\n\n", results.Query))

	for i, result := range results.Results {
		sb.WriteString(fmt.Sprintf("### [%d] %s\n", i+1, result.Title))
		sb.WriteString(fmt.Sprintf("**Source:** %s\n", result.Source))
		sb.WriteString(fmt.Sprintf("**URL:** %s\n", result.URL))
		if result.Snippet != "" {
			sb.WriteString(fmt.Sprintf("**Summary:** %s\n", result.Snippet))
		}
		sb.WriteString("\n")
	}

	sb.WriteString("---\n")
	sb.WriteString(fmt.Sprintf("*Search completed in %.0fms via %s*\n", results.SearchTime, results.Provider))

	return sb.String()
}
