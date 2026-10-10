// Package paypal ports the PayPal slot of FounderOS v1's payments registry.
//
// FounderOS v1 registers PayPal as configured-only (PAYPAL_CLIENT_ID and
// PAYPAL_CLIENT_SECRET both set) and makes no PayPal API call; the finances
// page cut its card on 2026-08-17. There are therefore no reads to port. The
// bridge adds one thing: an honest Status that verifies the credentials by
// minting an OAuth2 client-credentials token, so "configured" is never shown
// as connected without proof. Nothing here moves money.
package paypal

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/rhl/businessos-backend/internal/founderos/connectors"
)

var Meta = connectors.Meta{ID: "paypal", Name: "PayPal", Kind: connectors.KindPayments}

const (
	EnvClientID     = "PAYPAL_CLIENT_ID"
	EnvClientSecret = "PAYPAL_CLIENT_SECRET"
	defaultBaseURL  = "https://api-m.paypal.com"
)

// ReadPOSTs is the one POST the connector sends: the token mint, which reads
// nothing and moves nothing, so it is allowed while FOUNDEROS_WRITES is off.
var ReadPOSTs = []string{"api-m.paypal.com/v1/oauth2/token"}

type Connector struct {
	res     connectors.Resolver
	client  *http.Client
	BaseURL string
}

func New(res connectors.Resolver) *Connector {
	return &Connector{res: res, client: connectors.HTTPClient(8*time.Second, ReadPOSTs...), BaseURL: defaultBaseURL}
}

func (c *Connector) creds() (string, string) {
	return c.res.Resolve(EnvClientID), c.res.Resolve(EnvClientSecret)
}

// Configured matches configuredProcessors: both the id and the secret.
func (c *Connector) Configured() bool {
	id, secret := c.creds()
	return id != "" && secret != ""
}

// Status verifies the client credentials. The token itself is discarded and
// never surfaces in the status.
func (c *Connector) Status(ctx context.Context) connectors.Status {
	st := connectors.Status{ID: Meta.ID, Name: Meta.Name, Kind: Meta.Kind}
	id, secret := c.creds()
	if id == "" || secret == "" {
		st.State = connectors.StateNotConfigured
		st.Detail = connectors.SetKeys(EnvClientID, EnvClientSecret)
		return st
	}
	appID, err := c.verify(ctx, id, secret)
	if err != nil {
		st.State = connectors.StateError
		st.Detail = "PayPal credentials set but verification failed: " + err.Error()
		return st
	}
	st.State = connectors.StateConnected
	st.Detail = "PayPal · credentials verified"
	if appID != "" {
		st.Detail = "PayPal · app " + appID + " verified"
	}
	return st
}

func (c *Connector) verify(ctx context.Context, id, secret string) (string, error) {
	form := url.Values{"grant_type": {"client_credentials"}}.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(c.BaseURL, "/")+"/v1/oauth2/token", strings.NewReader(form))
	if err != nil {
		return "", err
	}
	req.SetBasicAuth(id, secret)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	res, err := c.client.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	body, err := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if err != nil {
		return "", err
	}
	var doc struct {
		AppID       string `json:"app_id"`
		AccessToken string `json:"access_token"`
		Error       string `json:"error"`
		Description string `json:"error_description"`
	}
	_ = json.Unmarshal(body, &doc)
	if res.StatusCode < 200 || res.StatusCode > 299 {
		msg := doc.Description
		if msg == "" {
			msg = doc.Error
		}
		if msg == "" {
			return "", fmt.Errorf("HTTP %d", res.StatusCode)
		}
		return "", fmt.Errorf("HTTP %d: %s", res.StatusCode, msg)
	}
	if doc.AccessToken == "" {
		return "", fmt.Errorf("no access token in response")
	}
	return doc.AppID, nil
}
