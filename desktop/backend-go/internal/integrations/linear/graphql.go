package linear

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// ============================================================================
// Linear GraphQL Types
// ============================================================================

type linearUserInfo struct {
	UserID           string
	UserName         string
	Email            string
	OrganizationID   string
	OrganizationName string
}

type graphQLRequest struct {
	Query     string                 `json:"query"`
	Variables map[string]interface{} `json:"variables,omitempty"`
}

type graphQLResponse struct {
	Data   json.RawMessage `json:"data"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors,omitempty"`
}

// ============================================================================
// GraphQL Methods
// ============================================================================

// executeGraphQL executes a GraphQL query against the Linear API.
func (p *Provider) executeGraphQL(ctx context.Context, accessToken string, query string, variables map[string]interface{}) (*graphQLResponse, error) {
	reqBody := graphQLRequest{
		Query:     query,
		Variables: variables,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", GraphQLURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", accessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var result graphQLResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	if len(result.Errors) > 0 {
		return nil, fmt.Errorf("graphql error: %s", result.Errors[0].Message)
	}

	return &result, nil
}

// getUserInfo retrieves the current user's information from Linear.
func (p *Provider) getUserInfo(ctx context.Context, accessToken string) (*linearUserInfo, error) {
	query := `
		query {
			viewer {
				id
				name
				email
				organization {
					id
					name
				}
			}
		}
	`

	resp, err := p.executeGraphQL(ctx, accessToken, query, nil)
	if err != nil {
		return nil, err
	}

	var data struct {
		Viewer struct {
			ID           string `json:"id"`
			Name         string `json:"name"`
			Email        string `json:"email"`
			Organization struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			} `json:"organization"`
		} `json:"viewer"`
	}

	if err := json.Unmarshal(resp.Data, &data); err != nil {
		return nil, err
	}

	return &linearUserInfo{
		UserID:           data.Viewer.ID,
		UserName:         data.Viewer.Name,
		Email:            data.Viewer.Email,
		OrganizationID:   data.Viewer.Organization.ID,
		OrganizationName: data.Viewer.Organization.Name,
	}, nil
}
