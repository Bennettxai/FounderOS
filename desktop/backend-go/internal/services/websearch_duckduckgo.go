package services

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

// ============================================================================
// DUCKDUCKGO PROVIDER (Fallback - No API Key Required)
// ============================================================================

// userAgents is a list of common browser user agents for rotation
var userAgents = []string{
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/121.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/121.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:122.0) Gecko/20100101 Firefox/122.0",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.2 Safari/605.1.15",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/121.0.0.0 Safari/537.36 Edg/121.0.0.0",
}

// DuckDuckGoProvider implements search using DuckDuckGo (no API key required)
type DuckDuckGoProvider struct {
	client *http.Client
}

// NewDuckDuckGoProvider creates a new DuckDuckGo provider
func NewDuckDuckGoProvider() *DuckDuckGoProvider {
	return &DuckDuckGoProvider{
		client: &http.Client{
			Timeout: 15 * time.Second,
			// Don't follow redirects automatically
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 3 {
					return fmt.Errorf("too many redirects")
				}
				return nil
			},
		},
	}
}

func (d *DuckDuckGoProvider) Name() string { return "duckduckgo" }

func (d *DuckDuckGoProvider) Available() bool { return true } // Always available

func (d *DuckDuckGoProvider) Search(ctx context.Context, query string, maxResults int) (*WebSearchResponse, error) {
	startTime := time.Now()

	if maxResults <= 0 {
		maxResults = 10
	}
	if maxResults > 20 {
		maxResults = 20
	}

	slog.Debug("DuckDuckGo search starting", "query", query, "maxResults", maxResults)

	// Try the Lite HTML version first (more reliable)
	results, err := d.searchLite(ctx, query, maxResults)
	if err != nil || len(results) == 0 {
		if err != nil {
			slog.Debug("DuckDuckGo Lite failed with error, trying HTML", "err", err)
		} else {
			slog.Debug("DuckDuckGo Lite returned 0 results, trying HTML")
		}
		// Fallback to standard HTML
		results, err = d.searchHTML(ctx, query, maxResults)
		if err != nil || len(results) == 0 {
			if err != nil {
				slog.Debug("DuckDuckGo HTML failed with error, trying instant", "err", err)
			} else {
				slog.Debug("DuckDuckGo HTML returned 0 results, trying instant")
			}
			// Final fallback to instant answers
			results, err = d.searchInstant(ctx, query)
			if err != nil {
				slog.Error("All DuckDuckGo search methods failed", "err", err, "query", query)
				return nil, fmt.Errorf("all DuckDuckGo search methods failed: %w", err)
			}
		}
	}

	slog.Debug("DuckDuckGo search completed", "query", query, "results", len(results), "duration", time.Since(startTime))

	return &WebSearchResponse{
		Query:        query,
		Results:      results,
		TotalResults: len(results),
		SearchTime:   float64(time.Since(startTime).Milliseconds()),
		Provider:     "duckduckgo",
	}, nil
}

// searchLite uses the Lite (no JS) version of DuckDuckGo
func (d *DuckDuckGoProvider) searchLite(ctx context.Context, query string, maxResults int) ([]WebSearchResult, error) {
	searchURL := "https://lite.duckduckgo.com/lite/"

	// POST request with form data
	formData := url.Values{}
	formData.Set("q", query)
	formData.Set("kl", "us-en") // US English

	req, err := http.NewRequestWithContext(ctx, "POST", searchURL, strings.NewReader(formData.Encode()))
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", getRandomUserAgent())
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")

	resp, err := d.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("search returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return d.parseLiteHTML(string(body), maxResults), nil
}

// parseLiteHTML parses the Lite DuckDuckGo results page
func (d *DuckDuckGoProvider) parseLiteHTML(html string, maxResults int) []WebSearchResult {
	var results []WebSearchResult

	// DuckDuckGo Lite has a table-based structure
	// Results are in table rows with links followed by snippets

	// More flexible pattern - look for links that have uddg= (DuckDuckGo redirect URLs)
	// or result-link class, attributes can be in any order
	linkPatterns := []*regexp.Regexp{
		// Pattern 1: Links with uddg parameter (redirect URLs)
		regexp.MustCompile(`<a[^>]*href="([^"]*uddg=[^"]+)"[^>]*>([^<]+)</a>`),
		// Pattern 2: Links with result-link class
		regexp.MustCompile(`<a[^>]*class="[^"]*result-link[^"]*"[^>]*href="([^"]+)"[^>]*>([^<]+)</a>`),
		// Pattern 3: class before href
		regexp.MustCompile(`<a[^>]*href="([^"]+)"[^>]*class="[^"]*result-link[^"]*"[^>]*>([^<]+)</a>`),
		// Pattern 4: rel="nofollow" links (common for external links)
		regexp.MustCompile(`<a[^>]*rel="nofollow"[^>]*href="([^"]+)"[^>]*>([^<]+)</a>`),
	}

	// Try to find snippets - they appear in various formats
	snippetPatterns := []*regexp.Regexp{
		regexp.MustCompile(`<td[^>]*class="[^"]*result-snippet[^"]*"[^>]*>([^<]+)</td>`),
		regexp.MustCompile(`<span[^>]*class="[^"]*result-snippet[^"]*"[^>]*>([^<]+)</span>`),
		regexp.MustCompile(`class="[^"]*snippet[^"]*"[^>]*>([^<]+)<`),
	}

	// Collect all link matches from all patterns
	var allLinks [][]string
	seenURLs := make(map[string]bool)

	for _, pattern := range linkPatterns {
		matches := pattern.FindAllStringSubmatch(html, maxResults*3)
		for _, match := range matches {
			if len(match) >= 3 {
				rawURL := match[1]
				// Deduplicate
				if !seenURLs[rawURL] {
					seenURLs[rawURL] = true
					allLinks = append(allLinks, match)
				}
			}
		}
	}

	// Collect snippets
	var snippets []string
	for _, pattern := range snippetPatterns {
		matches := pattern.FindAllStringSubmatch(html, maxResults*3)
		for _, match := range matches {
			if len(match) >= 2 {
				snippets = append(snippets, match[1])
			}
		}
	}

	slog.Debug("DuckDuckGo Lite parsing", "linksFound", len(allLinks), "snippetsFound", len(snippets))

	snippetIdx := 0
	for _, match := range allLinks {
		if len(results) >= maxResults {
			break
		}

		rawURL := match[1]
		title := strings.TrimSpace(match[2])

		// Skip DuckDuckGo's internal navigation links
		if strings.Contains(rawURL, "duckduckgo.com") && !strings.Contains(rawURL, "uddg=") {
			continue
		}

		// Skip empty titles
		if title == "" || title == "..." {
			continue
		}

		// Clean the URL (handle DuckDuckGo redirect URLs)
		actualURL := cleanDDGURL(rawURL)
		if actualURL == "" || !strings.HasPrefix(actualURL, "http") {
			continue
		}

		// Skip DuckDuckGo pages
		if strings.Contains(actualURL, "duckduckgo.com") {
			continue
		}

		// Get snippet if available
		snippet := ""
		if snippetIdx < len(snippets) {
			snippet = cleanHTMLText(snippets[snippetIdx])
			snippetIdx++
		}

		results = append(results, WebSearchResult{
			Title:   cleanHTMLText(title),
			URL:     actualURL,
			Snippet: snippet,
			Source:  extractDomain(actualURL),
		})
	}

	slog.Debug("DuckDuckGo Lite results", "count", len(results))

	return results
}

// searchHTML uses the standard HTML version
func (d *DuckDuckGoProvider) searchHTML(ctx context.Context, query string, maxResults int) ([]WebSearchResult, error) {
	searchURL := fmt.Sprintf("https://html.duckduckgo.com/html/?q=%s", url.QueryEscape(query))

	req, err := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", getRandomUserAgent())
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")
	req.Header.Set("DNT", "1")

	resp, err := d.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("search returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return d.parseHTML(string(body), maxResults), nil
}

// parseHTML parses the HTML response from DuckDuckGo
func (d *DuckDuckGoProvider) parseHTML(html string, maxResults int) []WebSearchResult {
	var results []WebSearchResult

	// Multiple patterns to handle different HTML structures
	linkPatterns := []*regexp.Regexp{
		// Pattern for result__a class (standard html.duckduckgo.com)
		regexp.MustCompile(`<a[^>]*class="[^"]*result__a[^"]*"[^>]*href="([^"]+)"[^>]*>([^<]+)</a>`),
		regexp.MustCompile(`<a[^>]*href="([^"]+)"[^>]*class="[^"]*result__a[^"]*"[^>]*>([^<]+)</a>`),
		// Pattern for uddg redirect URLs
		regexp.MustCompile(`<a[^>]*href="([^"]*uddg=[^"]+)"[^>]*>([^<]+)</a>`),
		// Pattern for result-link class
		regexp.MustCompile(`<a[^>]*class="[^"]*result-link[^"]*"[^>]*href="([^"]+)"[^>]*>([^<]+)</a>`),
	}

	snippetPatterns := []*regexp.Regexp{
		regexp.MustCompile(`<a[^>]*class="[^"]*result__snippet[^"]*"[^>]*>([^<]+)</a>`),
		regexp.MustCompile(`class="[^"]*result__snippet[^"]*"[^>]*>([^<]+)<`),
		regexp.MustCompile(`<span[^>]*class="[^"]*snippet[^"]*"[^>]*>([^<]+)</span>`),
	}

	// Collect links
	var allLinks [][]string
	seenURLs := make(map[string]bool)

	for _, pattern := range linkPatterns {
		matches := pattern.FindAllStringSubmatch(html, maxResults*3)
		for _, match := range matches {
			if len(match) >= 3 {
				rawURL := match[1]
				if !seenURLs[rawURL] {
					seenURLs[rawURL] = true
					allLinks = append(allLinks, match)
				}
			}
		}
	}

	// Collect snippets
	var snippets []string
	for _, pattern := range snippetPatterns {
		matches := pattern.FindAllStringSubmatch(html, maxResults*3)
		for _, match := range matches {
			if len(match) >= 2 {
				snippets = append(snippets, match[1])
			}
		}
	}

	slog.Debug("DuckDuckGo HTML parsing", "linksFound", len(allLinks), "snippetsFound", len(snippets))

	snippetIdx := 0
	for _, match := range allLinks {
		if len(results) >= maxResults {
			break
		}

		rawURL := match[1]
		title := strings.TrimSpace(match[2])

		// Skip navigation and empty
		if title == "" || title == "..." || strings.Contains(title, "DuckDuckGo") {
			continue
		}

		// Decode the DuckDuckGo redirect URL
		actualURL := cleanDDGURL(rawURL)
		if actualURL == "" || !strings.HasPrefix(actualURL, "http") {
			continue
		}

		// Skip DuckDuckGo pages
		if strings.Contains(actualURL, "duckduckgo.com") {
			continue
		}

		// Get snippet if available
		snippet := ""
		if snippetIdx < len(snippets) {
			snippet = cleanHTMLText(snippets[snippetIdx])
			snippetIdx++
		}

		results = append(results, WebSearchResult{
			Title:   cleanHTMLText(title),
			URL:     actualURL,
			Snippet: snippet,
			Source:  extractDomain(actualURL),
		})
	}

	slog.Debug("DuckDuckGo HTML results", "count", len(results))

	return results
}

// searchInstant uses the DuckDuckGo Instant Answer API
func (d *DuckDuckGoProvider) searchInstant(ctx context.Context, query string) ([]WebSearchResult, error) {
	apiURL := fmt.Sprintf("https://api.duckduckgo.com/?q=%s&format=json&no_html=1&skip_disambig=1", url.QueryEscape(query))

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "BusinessOS/1.0")

	resp, err := d.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("instant answer API returned status %d", resp.StatusCode)
	}

	var ddgResp struct {
		Abstract       string `json:"Abstract"`
		AbstractText   string `json:"AbstractText"`
		AbstractSource string `json:"AbstractSource"`
		AbstractURL    string `json:"AbstractURL"`
		Heading        string `json:"Heading"`
		RelatedTopics  []struct {
			Text     string `json:"Text"`
			FirstURL string `json:"FirstURL"`
		} `json:"RelatedTopics"`
		Results []struct {
			Text     string `json:"Text"`
			FirstURL string `json:"FirstURL"`
		} `json:"Results"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&ddgResp); err != nil {
		return nil, err
	}

	var results []WebSearchResult

	// Add abstract if available
	if ddgResp.AbstractText != "" && ddgResp.AbstractURL != "" {
		results = append(results, WebSearchResult{
			Title:   ddgResp.Heading,
			URL:     ddgResp.AbstractURL,
			Snippet: ddgResp.AbstractText,
			Source:  ddgResp.AbstractSource,
		})
	}

	// Add related topics
	for _, topic := range ddgResp.RelatedTopics {
		if topic.FirstURL != "" && topic.Text != "" {
			results = append(results, WebSearchResult{
				Title:   extractTitleFromText(topic.Text),
				URL:     topic.FirstURL,
				Snippet: topic.Text,
				Source:  extractDomain(topic.FirstURL),
			})
		}
	}

	// Add direct results
	for _, result := range ddgResp.Results {
		if result.FirstURL != "" && result.Text != "" {
			results = append(results, WebSearchResult{
				Title:   extractTitleFromText(result.Text),
				URL:     result.FirstURL,
				Snippet: result.Text,
				Source:  extractDomain(result.FirstURL),
			})
		}
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("no results from instant answer API")
	}

	return results, nil
}
