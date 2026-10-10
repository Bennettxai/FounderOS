package services

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// ============================================================================
// TAVILY PROVIDER (AI-Focused Search)
// ============================================================================

// TavilyProvider implements search using Tavily API (AI-optimized)
// Free tier: 1000 queries/month
// Docs: https://tavily.com/
type TavilyProvider struct {
	apiKey  string
	client  *http.Client
	baseURL string
}

// NewTavilyProvider creates a new Tavily provider
func NewTavilyProvider(apiKey string) *TavilyProvider {
	return &TavilyProvider{
		apiKey:  apiKey,
		client:  &http.Client{Timeout: 30 * time.Second}, // Tavily can be slow
		baseURL: "https://api.tavily.com/search",
	}
}

func (t *TavilyProvider) Name() string { return "tavily" }

func (t *TavilyProvider) Available() bool { return t.apiKey != "" }

func (t *TavilyProvider) Search(ctx context.Context, query string, maxResults int) (*WebSearchResponse, error) {
	if !t.Available() {
		return nil, fmt.Errorf("tavily API key not configured")
	}

	startTime := time.Now()

	// Build request body
	reqBody := map[string]interface{}{
		"api_key":             t.apiKey,
		"query":               query,
		"max_results":         min(maxResults, 10), // Tavily max is 10
		"search_depth":        "basic",             // "basic" or "advanced"
		"include_answer":      false,
		"include_raw_content": false,
		"include_images":      false,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", t.baseURL, strings.NewReader(string(bodyBytes)))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := t.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("tavily API returned status %d: %s", resp.StatusCode, string(body))
	}

	var tavilyResp struct {
		Results []struct {
			Title   string  `json:"title"`
			URL     string  `json:"url"`
			Content string  `json:"content"`
			Score   float64 `json:"score"`
		} `json:"results"`
		Query string `json:"query"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&tavilyResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	results := make([]WebSearchResult, 0, len(tavilyResp.Results))
	for _, r := range tavilyResp.Results {
		results = append(results, WebSearchResult{
			Title:   r.Title,
			URL:     r.URL,
			Snippet: r.Content,
			Source:  extractDomain(r.URL),
			Score:   r.Score,
		})
	}

	return &WebSearchResponse{
		Query:        query,
		Results:      results,
		TotalResults: len(results),
		SearchTime:   float64(time.Since(startTime).Milliseconds()),
		Provider:     "tavily",
	}, nil
}
