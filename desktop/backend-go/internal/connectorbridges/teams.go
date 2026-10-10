// teams.go — Microsoft Teams fetcher for the OptimalEngine connector layer.
//
// Implements adapters.TeamsFetcher. Phase-1 strategy: surface the user's
// joined-teams chat messages via Microsoft Graph. We list joined teams
// first, then walk channels in the first-found team. Multi-team fan-out
// is a follow-up — keeps the wire narrow and predictable.
package connectorbridges

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/rhl/businessos-backend/internal/integrations/microsoft"
)

// TeamsFetcher pulls chatMessages from joined Teams. Cursor format:
// "teamID|channelID|@odata.nextLink" — encodes the team + channel + page
// so subsequent calls page through the same channel. Empty cursor restarts
// from the first joined team's first channel.
type TeamsFetcher struct {
	prov   *microsoft.Provider
	client *http.Client
}

// NewTeamsFetcher returns a fetcher wired to the Microsoft provider.
func NewTeamsFetcher(prov *microsoft.Provider) *TeamsFetcher {
	return &TeamsFetcher{
		prov:   prov,
		client: &http.Client{Timeout: 15 * time.Second},
	}
}

// ListMessages pulls one batch of channel messages.
func (f *TeamsFetcher) ListMessages(ctx context.Context, userID, cursor string, max int) ([]map[string]any, string, error) {
	tok, err := f.prov.GetToken(ctx, userID)
	if err != nil {
		return nil, "", fmt.Errorf("teams: token: %w", err)
	}

	teamID, channelID, pageURL := splitTeamsCursor(cursor)
	if teamID == "" {
		teamID, err = f.firstJoinedTeam(ctx, tok.AccessToken)
		if err != nil {
			return nil, "", err
		}
		if teamID == "" {
			return nil, "", nil // no teams joined → caught up
		}
	}
	if channelID == "" {
		channelID, err = f.firstChannel(ctx, tok.AccessToken, teamID)
		if err != nil {
			return nil, "", err
		}
		if channelID == "" {
			return nil, "", nil // no channels → caught up
		}
	}

	target := pageURL
	if target == "" {
		target = fmt.Sprintf(
			"https://graph.microsoft.com/v1.0/teams/%s/channels/%s/messages?$top=%d",
			teamID, channelID, max,
		)
	}

	body, err := f.do(ctx, tok.AccessToken, target)
	if err != nil {
		return nil, "", err
	}
	var resp struct {
		Value    []map[string]any `json:"value"`
		NextLink string           `json:"@odata.nextLink"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, "", fmt.Errorf("teams: decode: %w", err)
	}
	next := ""
	if resp.NextLink != "" {
		next = teamID + "|" + channelID + "|" + resp.NextLink
	}
	return resp.Value, next, nil
}

func (f *TeamsFetcher) firstJoinedTeam(ctx context.Context, token string) (string, error) {
	body, err := f.do(ctx, token, "https://graph.microsoft.com/v1.0/me/joinedTeams?$top=1")
	if err != nil {
		return "", err
	}
	var resp struct {
		Value []struct {
			ID string `json:"id"`
		} `json:"value"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return "", fmt.Errorf("teams: decode joinedTeams: %w", err)
	}
	if len(resp.Value) == 0 {
		return "", nil
	}
	return resp.Value[0].ID, nil
}

func (f *TeamsFetcher) firstChannel(ctx context.Context, token, teamID string) (string, error) {
	body, err := f.do(ctx, token,
		fmt.Sprintf("https://graph.microsoft.com/v1.0/teams/%s/channels?$top=1", teamID))
	if err != nil {
		return "", err
	}
	var resp struct {
		Value []struct {
			ID string `json:"id"`
		} `json:"value"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return "", fmt.Errorf("teams: decode channels: %w", err)
	}
	if len(resp.Value) == 0 {
		return "", nil
	}
	return resp.Value[0].ID, nil
}

func (f *TeamsFetcher) do(ctx context.Context, token, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := f.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("teams: http %d: %s", resp.StatusCode, string(body))
	}
	return body, nil
}

// splitTeamsCursor decodes "teamID|channelID|nextLink" — empty input gives
// (empty, empty, empty) so the caller bootstraps from joined-teams lookup.
func splitTeamsCursor(s string) (team, channel, page string) {
	first := -1
	for i := 0; i < len(s); i++ {
		if s[i] == '|' {
			first = i
			break
		}
	}
	if first < 0 {
		return s, "", ""
	}
	team = s[:first]
	rest := s[first+1:]
	for i := 0; i < len(rest); i++ {
		if rest[i] == '|' {
			return team, rest[:i], rest[i+1:]
		}
	}
	return team, rest, ""
}
