// Package arcads ports FounderOS v1's lib/connectors/arcads.ts: a status check
// against the Arcads external API (UGC ad generation). FounderOS v1 exports no
// Arcads reads beyond it; the Arcads agent only reads this status.
package arcads

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/rhl/businessos-backend/internal/founderos/connectors"
)

var Meta = connectors.Meta{ID: "arcads", Name: "Arcads (UGC Ads)", Kind: connectors.KindCreative}

const (
	DefaultBaseURL = "https://external-api.arcads.ai"
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
	_, _, arcadsEnv, _ := connectors.CredFiles()
	return &Connector{res: res, Files: []string{arcadsEnv}, BaseURL: DefaultBaseURL, client: connectors.HTTPClient(Timeout)}
}

func (c *Connector) Status(ctx context.Context) connectors.Status {
	st := connectors.Status{ID: Meta.ID, Name: Meta.Name, Kind: Meta.Kind}
	auth := c.res.Resolve("ARCADS_BASIC_AUTH", c.Files...)
	if auth == "" {
		st.State = connectors.StateNotConfigured
		st.Detail = connectors.NotFound("ARCADS_BASIC_AUTH")
		return st
	}
	products, err := c.productCount(ctx, auth)
	if err != nil {
		st.State = connectors.StateError
		st.Detail = "Creds found but API check failed: " + reason(err)
		return st
	}
	st.State = connectors.StateConnected
	if products < 0 {
		st.Detail = "Vantage workspace reachable · product count unknown"
		return st
	}
	plural := "s"
	if products == 1 {
		plural = ""
	}
	st.Detail = fmt.Sprintf("Vantage workspace reachable · %d product%s", products, plural)
	st.Meta = map[string]any{"products": products}
	return st
}

// productCount reads GET /v1/products: a bare array or {results: [...]};
// -1 when neither shape is present.
func (c *Connector) productCount(ctx context.Context, auth string) (int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+"/v1/products", nil)
	if err != nil {
		return 0, err
	}
	if !strings.HasPrefix(auth, "Basic ") {
		auth = "Basic " + auth
	}
	req.Header.Set("Authorization", auth)
	res, err := c.client.Do(req)
	if err != nil {
		return 0, err
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode > 299 {
		msg := fmt.Sprintf("HTTP %d", res.StatusCode)
		if res.StatusCode == http.StatusForbidden {
			msg += " — rotate creds in Arcads dashboard"
		}
		return 0, errors.New(msg)
	}
	var raw json.RawMessage
	if err := json.NewDecoder(res.Body).Decode(&raw); err != nil {
		return 0, errors.New("unreadable response")
	}
	var list []json.RawMessage
	if json.Unmarshal(raw, &list) == nil && list != nil {
		return len(list), nil
	}
	var wrapped struct {
		Results []json.RawMessage `json:"results"`
	}
	if json.Unmarshal(raw, &wrapped) == nil && wrapped.Results != nil {
		return len(wrapped.Results), nil
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
