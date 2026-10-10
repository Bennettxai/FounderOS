package services

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// ============================================================================
// BRAVE SEARCH PROVIDER
// ============================================================================

// BraveSearchProvider implements search using Brave Search API
// Free tier: 2000 queries/month
// Docs: https://brave.com/search/api/
type BraveSearchProvider struct {
	apiKey  string
	client  *http.Client
	baseURL string
}

// NewBraveSearchProvider creates a new Brave Search provider
func NewBraveSearchProvider(apiKey string) *BraveSearchProvider {
	return &BraveSearchProvider{
		apiKey:  apiKey,
		client:  &http.Client{Timeout: 15 * time.Second},
		baseURL: "https://api.search.brave.com/res/v1/web/search",
	}
}

func (b *BraveSearchProvider) Name() string { return "brave" }

func (b *BraveSearchProvider) Available() bool { return b.apiKey != "" }

func (b *BraveSearchProvider) Search(ctx context.Context, query string, maxResults int) (*WebSearchResponse, error) {
	if !b.Available() {
		return nil, fmt.Errorf("brave search API key not configured")
	}

	startTime := time.Now()

	// Build request URL
	params := url.Values{}
	params.Set("q", query)
	params.Set("count", fmt.Sprintf("%d", min(maxResults, 20)))
	params.Set("text_decorations", "false")
	params.Set("search_lang", "en")
	params.Set("safesearch", "moderate")

	reqURL := b.baseURL + "?" + params.Encode()

	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Accept-Encoding", "gzip")
	req.Header.Set("X-Subscription-Token", b.apiKey)

	resp, err := b.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("brave API returned status %d: %s", resp.StatusCode, string(body))
	}

	var braveResp struct {
		Web struct {
			Results []struct {
				Title       string `json:"title"`
				URL         string `json:"url"`
				Description string `json:"description"`
				PageAge     string `json:"page_age,omitempty"`
			} `json:"results"`
		} `json:"web"`
		Query struct {
			Original string `json:"original"`
		} `json:"query"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&braveResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	results := make([]WebSearchResult, 0, len(braveResp.Web.Results))
	for _, r := range braveResp.Web.Results {
		results = append(results, WebSearchResult{
			Title:       r.Title,
			URL:         r.URL,
			Snippet:     r.Description,
			Source:      extractDomain(r.URL),
			PublishedAt: r.PageAge,
		})
	}

	return &WebSearchResponse{
		Query:        query,
		Results:      results,
		TotalResults: len(results),
		SearchTime:   float64(time.Since(startTime).Milliseconds()),
		Provider:     "brave",
	}, nil
}
