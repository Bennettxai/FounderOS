// Package miro ports FounderOS v1's lib/connectors/miro.ts: a status check
// against the Miro boards API. FounderOS v1 exports no Miro reads beyond it.
package miro

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/rhl/businessos-backend/internal/founderos/connectors"
)

var Meta = connectors.Meta{ID: "miro", Name: "Miro", Kind: connectors.KindCreative}

const (
	DefaultBaseURL = "https://api.miro.com"
	Timeout        = 4 * time.Second
)

type Connector struct {
	res connectors.Resolver
	// Files are fallback credential files (none: connectors.CredFiles).
	Files   []string
	BaseURL string
	client  *http.Client
}

func New(res connectors.Resolver) *Connector {
	_, clue, _, _ := connectors.CredFiles()
	return &Connector{res: res, Files: []string{clue}, BaseURL: DefaultBaseURL, client: connectors.HTTPClient(Timeout)}
}

func (c *Connector) Status(ctx context.Context) connectors.Status {
	st := connectors.Status{ID: Meta.ID, Name: Meta.Name, Kind: Meta.Kind}
	token := c.res.Resolve("MIRO_ACCESS_TOKEN", c.Files...)
	if token == "" {
		st.State = connectors.StateNotConfigured
		st.Detail = connectors.NotFound("MIRO_ACCESS_TOKEN")
		return st
	}
	boards, err := c.boardCount(ctx, token)
	if err != nil {
		st.State = connectors.StateError
		st.Detail = "Token found but API check failed: " + reason(err)
		return st
	}
	st.State = connectors.StateConnected
	if boards < 0 {
		st.Detail = "Boards API reachable · board count unknown"
		return st
	}
	st.Detail = fmt.Sprintf("Boards API reachable · %d boards", boards)
	st.Meta = map[string]any{"boards": boards}
	return st
}

// boardCount is total, else the page size, else -1 (unknown).
func (c *Connector) boardCount(ctx context.Context, token string) (int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+"/v2/boards?limit=10", nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	res, err := c.client.Do(req)
	if err != nil {
		return 0, err
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode > 299 {
		return 0, fmt.Errorf("HTTP %d", res.StatusCode)
	}
	var body struct {
		Data  []json.RawMessage `json:"data"`
		Total *int              `json:"total"`
	}
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		return 0, errors.New("unreadable response")
	}
	switch {
	case body.Total != nil:
		return *body.Total, nil
	case body.Data != nil:
		return len(body.Data), nil
	}
	return -1, nil
}

func reason(err error) string {
	var ue *url.Error
	if errors.As(err, &ue) {
		if ue.Timeout() {
			return "request timed out"
		}
		return ue.Err.Error()
	}
	return err.Error()
}
