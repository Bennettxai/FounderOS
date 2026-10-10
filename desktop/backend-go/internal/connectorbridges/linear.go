// linear_connector.go — Linear fetcher for the OptimalEngine connector layer.
//
// Linear is GraphQL-only. We issue a single query for the user's recent
// issues, ordered by updatedAt. The cursor is Linear's pageInfo.endCursor.
package connectorbridges

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/rhl/businessos-backend/internal/integrations/linear"
)

const linearGraphQLEndpoint = "https://api.linear.app/graphql"

const linearIssuesQuery = `
query ConnectorIssues($first: Int!, $after: String) {
  issues(first: $first, after: $after, orderBy: updatedAt) {
    nodes {
      id
      title
      description
      updatedAt
      assignee { name }
      creator { name }
    }
    pageInfo { endCursor hasNextPage }
  }
}
`

// LinearFetcher implements adapters.LinearFetcher using the GraphQL
// API. The token comes from the existing linear.Provider.
type LinearFetcher struct {
	prov   *linear.Provider
	client *http.Client
}

func NewLinearFetcher(prov *linear.Provider) *LinearFetcher {
	return &LinearFetcher{
		prov:   prov,
		client: &http.Client{Timeout: 15 * time.Second},
	}
}

// ListIssues runs a single GraphQL query. Returns the `nodes` array as
// generic maps so the adapter's Transform can read the same field names
// it uses for direct API payload tests.
func (f *LinearFetcher) ListIssues(ctx context.Context, userID, cursor string, max int) ([]map[string]any, string, error) {
	tok, err := f.prov.GetToken(ctx, userID)
	if err != nil {
		return nil, "", fmt.Errorf("linear: token: %w", err)
	}

	vars := map[string]any{"first": max}
	if cursor != "" {
		vars["after"] = cursor
	}
	body := map[string]any{"query": linearIssuesQuery, "variables": vars}
	encoded, _ := json.Marshal(body)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		linearGraphQLEndpoint, bytes.NewReader(encoded))
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("Authorization", "Bearer "+tok.AccessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := f.client.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("linear: request: %w", err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", err
	}
	if resp.StatusCode >= 400 {
		return nil, "", fmt.Errorf("linear: http %d: %s", resp.StatusCode, string(raw))
	}

	var out struct {
		Data struct {
			Issues struct {
				Nodes    []map[string]any `json:"nodes"`
				PageInfo struct {
					EndCursor   string `json:"endCursor"`
					HasNextPage bool   `json:"hasNextPage"`
				} `json:"pageInfo"`
			} `json:"issues"`
		} `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, "", fmt.Errorf("linear: decode: %w", err)
	}
	if len(out.Errors) > 0 {
		return nil, "", fmt.Errorf("linear: %s", out.Errors[0].Message)
	}
	next := ""
	if out.Data.Issues.PageInfo.HasNextPage {
		next = out.Data.Issues.PageInfo.EndCursor
	}
	return out.Data.Issues.Nodes, next, nil
}
