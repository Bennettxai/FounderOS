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
// SERPER PROVIDER (Google Results)
// ============================================================================

// SerperProvider implements search using Serper.dev API (Google results)
// Free tier: 2500 queries/month
// Docs: https://serper.dev/
type SerperProvider struct {
	apiKey  string
	client  *http.Client
	baseURL string
}

// NewSerperProvider creates a new Serper provider
func NewSerperProvider(apiKey string) *SerperProvider {
	return &SerperProvider{
		apiKey:  apiKey,
		client:  &http.Client{Timeout: 15 * time.Second},
		baseURL: "https://google.serper.dev/search",
	}
}

func (s *SerperProvider) Name() string { return "serper" }

func (s *SerperProvider) Available() bool { return s.apiKey != "" }

func (s *SerperProvider) Search(ctx context.Context, query string, maxResults int) (*WebSearchResponse, error) {
	if !s.Available() {
		return nil, fmt.Errorf("serper API key not configured")
	}

	startTime := time.Now()

	// Build request body
	reqBody := map[string]interface{}{
		"q":   query,
		"num": min(maxResults, 20),
		"gl":  "us",
		"hl":  "en",
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", s.baseURL, strings.NewReader(string(bodyBytes)))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-KEY", s.apiKey)

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("serper API returned status %d: %s", resp.StatusCode, string(body))
	}

	var serperResp struct {
		Organic []struct {
			Title    string `json:"title"`
			Link     string `json:"link"`
			Snippet  string `json:"snippet"`
			Position int    `json:"position"`
			Date     string `json:"date,omitempty"`
		} `json:"organic"`
		SearchParameters struct {
			Q string `json:"q"`
		} `json:"searchParameters"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&serperResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	results := make([]WebSearchResult, 0, len(serperResp.Organic))
	for _, r := range serperResp.Organic {
		results = append(results, WebSearchResult{
			Title:       r.Title,
			URL:         r.Link,
			Snippet:     r.Snippet,
			Source:      extractDomain(r.Link),
			PublishedAt: r.Date,
		})
	}

	return &WebSearchResponse{
		Query:        query,
		Results:      results,
		TotalResults: len(results),
		SearchTime:   float64(time.Since(startTime).Milliseconds()),
		Provider:     "serper",
	}, nil
}
