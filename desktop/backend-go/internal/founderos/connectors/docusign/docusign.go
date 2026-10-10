// Package docusign ports FounderOS v1's lib/connectors/docusign.ts: contracts
// and agreements for Vantage deals.
//
// Auth is the OAuth JWT grant for a single user:
//  1. Sign an RS256 JWT with the integration key and the operator's user GUID.
//  2. POST it to {authHost}/oauth/token for an access token.
//  3. GET /oauth/userinfo for the account's REST base_uri.
//  4. Call {base_uri}/restapi/v2.1/accounts/{accountId}/... with the token.
//
// DOCUSIGN_ENV=demo (the default) targets account-d.docusign.com; anything
// else targets account.docusign.com. The RSA key rides base64-encoded in
// DOCUSIGN_PRIVATE_KEY_B64 because env slots are single-line.
package docusign

import (
	"bytes"
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/rhl/businessos-backend/internal/founderos/connectors"
	"github.com/rhl/businessos-backend/internal/founderos/guard"
)

var Meta = connectors.Meta{ID: "docusign", Name: "DocuSign", Kind: connectors.KindCRM}

const (
	Timeout  = 8 * time.Second
	demoHost = "account-d.docusign.com"
	prodHost = "account.docusign.com"
	// window is the envelope listing's required from_date.
	window = 90 * 24 * time.Hour
)

// ReadPOSTs: the JWT token exchange is a POST that changes nothing.
var ReadPOSTs = []string{demoHost + "/oauth/token", prodHost + "/oauth/token"}

var ErrNotConfigured = errors.New("docusign: DOCUSIGN_* credentials are not set")

type Connector struct {
	res connectors.Resolver
	// Files are the fallback credential files (clue-agent, then social-media).
	Files []string
	// AuthBaseURL overrides https://{authHost} (tests). The JWT audience
	// stays the real auth host.
	AuthBaseURL string
	Now         func() time.Time
	client      *http.Client
}

func New(res connectors.Resolver) *Connector {
	social, clue, _, _ := connectors.CredFiles()
	return &Connector{
		res:    res,
		Files:  []string{clue, social},
		Now:    time.Now,
		client: connectors.HTTPClient(Timeout, ReadPOSTs...),
	}
}

type Config struct {
	IntegrationKey string
	UserID         string
	AccountID      string
	AuthHost       string
	PrivateKey     *rsa.PrivateKey
	keyErr         error
}

func (c *Connector) config() (Config, bool) {
	get := func(name string) string { return c.res.Resolve(name, c.Files...) }
	cfg := Config{
		IntegrationKey: get("DOCUSIGN_INTEGRATION_KEY"),
		UserID:         get("DOCUSIGN_USER_ID"),
		AccountID:      get("DOCUSIGN_ACCOUNT_ID"),
	}
	keyB64 := get("DOCUSIGN_PRIVATE_KEY_B64")
	if cfg.IntegrationKey == "" || cfg.UserID == "" || cfg.AccountID == "" || keyB64 == "" {
		return Config{}, false
	}
	env := get("DOCUSIGN_ENV")
	if env == "" {
		env = "demo"
	}
	cfg.AuthHost = prodHost
	if env == "demo" {
		cfg.AuthHost = demoHost
	}
	cfg.PrivateKey, cfg.keyErr = parsePrivateKey(keyB64)
	return cfg, true
}

// decodeLenientBase64 accepts what Node's Buffer.from(s, 'base64') accepts:
// whitespace, missing padding and the URL-safe alphabet.
func decodeLenientBase64(s string) ([]byte, error) {
	s = strings.Map(func(r rune) rune {
		switch r {
		case ' ', '\n', '\r', '\t', '=':
			return -1
		case '-':
			return '+'
		case '_':
			return '/'
		}
		return r
	}, s)
	return base64.RawStdEncoding.DecodeString(s)
}

func parsePrivateKey(b64 string) (*rsa.PrivateKey, error) {
	raw, err := decodeLenientBase64(b64)
	if err != nil {
		return nil, errors.New("private key is not valid base64")
	}
	block, _ := pem.Decode(raw)
	if block == nil {
		return nil, errors.New("private key is not a PEM block")
	}
	if k, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		return k, nil
	}
	k, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("private key: %v", err)
	}
	rk, ok := k.(*rsa.PrivateKey)
	if !ok {
		return nil, errors.New("private key is not RSA")
	}
	return rk, nil
}

// BuildJWTAssertion signs the RS256 JWT for the token exchange.
func BuildJWTAssertion(cfg Config, iat int64) (string, error) {
	if cfg.keyErr != nil {
		return "", cfg.keyErr
	}
	if cfg.PrivateKey == nil {
		return "", errors.New("private key missing")
	}
	header, _ := json.Marshal(struct {
		Alg string `json:"alg"`
		Typ string `json:"typ"`
	}{"RS256", "JWT"})
	payload, _ := json.Marshal(struct {
		Iss   string `json:"iss"`
		Sub   string `json:"sub"`
		Aud   string `json:"aud"`
		Iat   int64  `json:"iat"`
		Exp   int64  `json:"exp"`
		Scope string `json:"scope"`
	}{cfg.IntegrationKey, cfg.UserID, cfg.AuthHost, iat, iat + 3600, "signature impersonation"})
	enc := base64.RawURLEncoding
	signing := enc.EncodeToString(header) + "." + enc.EncodeToString(payload)
	sum := sha256.Sum256([]byte(signing))
	sig, err := rsa.SignPKCS1v15(rand.Reader, cfg.PrivateKey, crypto.SHA256, sum[:])
	if err != nil {
		return "", err
	}
	return signing + "." + enc.EncodeToString(sig), nil
}

func (c *Connector) authURL(cfg Config) string {
	if c.AuthBaseURL != "" {
		return strings.TrimRight(c.AuthBaseURL, "/")
	}
	return "https://" + cfg.AuthHost
}

func (c *Connector) do(req *http.Request, what string, into any) error {
	res, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("%s failed: %s", what, reason(err))
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode > 299 {
		return fmt.Errorf("%s failed: HTTP %d", what, res.StatusCode)
	}
	raw, err := io.ReadAll(io.LimitReader(res.Body, 32<<20))
	if err != nil {
		return fmt.Errorf("%s failed: %s", what, reason(err))
	}
	if into == nil {
		return nil
	}
	if b, ok := into.(*[]byte); ok {
		*b = raw
		return nil
	}
	if err := json.Unmarshal(raw, into); err != nil {
		return fmt.Errorf("%s failed: unreadable response", what)
	}
	return nil
}

func (c *Connector) accessToken(ctx context.Context, cfg Config) (string, error) {
	assertion, err := BuildJWTAssertion(cfg, c.Now().Unix())
	if err != nil {
		return "", err
	}
	form := url.Values{
		"grant_type": {"urn:ietf:params:oauth:grant-type:jwt-bearer"},
		"assertion":  {assertion},
	}.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.authURL(cfg)+"/oauth/token", strings.NewReader(form))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	var body struct {
		AccessToken string `json:"access_token"`
	}
	if err := c.do(req, "token exchange", &body); err != nil {
		return "", err
	}
	if body.AccessToken == "" {
		return "", errors.New("token exchange returned no access_token")
	}
	return body.AccessToken, nil
}

// baseURI finds the configured account's REST base (else the first account).
func (c *Connector) baseURI(ctx context.Context, cfg Config, token string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.authURL(cfg)+"/oauth/userinfo", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	var body struct {
		Accounts []struct {
			AccountID string `json:"account_id"`
			BaseURI   string `json:"base_uri"`
		} `json:"accounts"`
	}
	if err := c.do(req, "userinfo", &body); err != nil {
		return "", err
	}
	if len(body.Accounts) == 0 {
		return "", errors.New("userinfo returned no account base_uri")
	}
	account := body.Accounts[0]
	for _, a := range body.Accounts {
		if a.AccountID == cfg.AccountID {
			account = a
			break
		}
	}
	if account.BaseURI == "" {
		return "", errors.New("userinfo returned no account base_uri")
	}
	return strings.TrimRight(account.BaseURI, "/"), nil
}

// Envelope is one flat row of the envelope listing.
type Envelope struct {
	EnvelopeID string `json:"envelopeId"`
	Subject    string `json:"subject"`
	Status     string `json:"status"` // sent | delivered | completed | declined | voided ...
	At         string `json:"at"`     // last status change
}

// ParseEnvelopes maps the listing payload to rows. A shape it does not
// recognise is an empty list, never a panic.
func ParseEnvelopes(raw []byte) []Envelope {
	var body struct {
		Envelopes []map[string]any `json:"envelopes"`
	}
	if json.Unmarshal(raw, &body) != nil {
		return nil
	}
	out := make([]Envelope, 0, len(body.Envelopes))
	for _, e := range body.Envelopes {
		at := jsString(e["statusChangedDateTime"], "")
		if at == "" {
			at = jsString(e["createdDateTime"], "")
		}
		out = append(out, Envelope{
			EnvelopeID: jsString(e["envelopeId"], ""),
			Subject:    jsString(e["emailSubject"], "Untitled envelope"),
			Status:     jsString(e["status"], "unknown"),
			At:         at,
		})
	}
	return out
}

// jsString is String(v ?? fallback) for the scalar shapes DocuSign sends.
func jsString(v any, fallback string) string {
	switch t := v.(type) {
	case nil:
		return fallback
	case string:
		return t
	default:
		b, _ := json.Marshal(t)
		return string(b)
	}
}

func (c *Connector) envelopes(ctx context.Context, cfg Config, limit int) ([]Envelope, error) {
	token, err := c.accessToken(ctx, cfg)
	if err != nil {
		return nil, err
	}
	base, err := c.baseURI(ctx, cfg, token)
	if err != nil {
		return nil, err
	}
	from := c.Now().Add(-window).UTC().Format("2006-01-02T15:04:05.000Z")
	u := fmt.Sprintf("%s/restapi/v2.1/accounts/%s/envelopes?from_date=%s&order_by=last_modified&order=desc",
		base, cfg.AccountID, url.QueryEscape(from))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	var raw []byte
	if err := c.do(req, "envelopes", &raw); err != nil {
		return nil, err
	}
	rows := ParseEnvelopes(raw)
	if len(rows) > limit {
		rows = rows[:limit]
	}
	return rows, nil
}

// RecentEnvelopes lists up to limit envelopes changed in the last 90 days,
// newest first. Unlike the TS (which returns [] on failure), a failure is an
// error so no caller can mistake an outage for "no contracts".
func (c *Connector) RecentEnvelopes(ctx context.Context, limit int) ([]Envelope, error) {
	cfg, ok := c.config()
	if !ok {
		return nil, ErrNotConfigured
	}
	return c.envelopes(ctx, cfg, limit)
}

func (c *Connector) Status(ctx context.Context) connectors.Status {
	st := connectors.Status{ID: Meta.ID, Name: Meta.Name, Kind: Meta.Kind}
	cfg, ok := c.config()
	if !ok {
		st.State = connectors.StateNotConfigured
		st.Detail = "Set DOCUSIGN_INTEGRATION_KEY, DOCUSIGN_USER_ID, DOCUSIGN_ACCOUNT_ID and DOCUSIGN_PRIVATE_KEY_B64 (base64 of the RSA private key) in ~/.founderos/.env or under API keys. DOCUSIGN_ENV=demo for the developer sandbox."
		return st
	}
	rows, err := c.envelopes(ctx, cfg, 100)
	if err != nil {
		st.State = connectors.StateError
		st.Detail = "DocuSign creds are set but the call failed: " + err.Error()
		return st
	}
	plural := "s"
	if len(rows) == 1 {
		plural = ""
	}
	st.State = connectors.StateConnected
	st.Detail = fmt.Sprintf("DocuSign reachable · %d envelope%s in the last 90 days", len(rows), plural)
	st.Meta = map[string]any{"envelopes": len(rows)}
	return st
}

// SendEnvelope sends an existing draft envelope (PUT status "sent"). It is
// guarded by FOUNDEROS_WRITES: while writes are off nothing leaves the bridge,
// not even the token exchange. FounderOS v1 has no send path yet; this is the
// one DocuSign write the bridge exposes.
func (c *Connector) SendEnvelope(ctx context.Context, envelopeID string) error {
	return guard.Outbound("docusign.send_envelope", func() error {
		cfg, ok := c.config()
		if !ok {
			return ErrNotConfigured
		}
		token, err := c.accessToken(ctx, cfg)
		if err != nil {
			return err
		}
		base, err := c.baseURI(ctx, cfg, token)
		if err != nil {
			return err
		}
		u := fmt.Sprintf("%s/restapi/v2.1/accounts/%s/envelopes/%s", base, cfg.AccountID, url.PathEscape(envelopeID))
		req, err := http.NewRequestWithContext(ctx, http.MethodPut, u, bytes.NewReader([]byte(`{"status":"sent"}`)))
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		return c.do(req, "send envelope", nil)
	})
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
