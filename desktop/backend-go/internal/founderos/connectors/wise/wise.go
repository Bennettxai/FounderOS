// Package wise ports the Wise slot of FounderOS v1's payments registry
// (lib/connectors/payments.ts: parseWiseTransfers, wiseOutgoing): recent
// outgoing transfers, read with the WISE_1_TOKEN personal token. It is read
// only; FounderOS v1 never moves money through Wise and neither does the bridge.
package wise

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/rhl/businessos-backend/internal/founderos/connectors"
)

// Meta is the processor id FounderOS v1's configuredProcessors uses for this
// slot. The Connections board carries it inside the aggregate "payments" row.
var Meta = connectors.Meta{ID: "wise-1", Name: "Wise", Kind: connectors.KindPayments}

// EnvToken is the credential name (lib/keys.ts / configuredProcessors).
const EnvToken = "WISE_1_TOKEN"

const defaultBaseURL = "https://api.wise.com"

// ErrNotConfigured is returned by reads that need a token when none is set.
var ErrNotConfigured = errors.New("wise: WISE_1_TOKEN is not set")

// Transfer is one outgoing transfer (lib/finances.ts OutgoingTransfer).
// Created keeps Wise's JSON type: a string timestamp or a number.
type Transfer struct {
	AmountCents int64   `json:"amountCents"`
	Currency    string  `json:"currency"`
	Status      string  `json:"status"`
	Created     any     `json:"created"`
	Reference   *string `json:"reference,omitempty"`
}

type Connector struct {
	res     connectors.Resolver
	client  *http.Client
	BaseURL string
}

// New builds the connector. The token is resolved on every call, never cached.
func New(res connectors.Resolver) *Connector {
	// FounderOS v1 used a 6s abort signal for the transfers read.
	return &Connector{res: res, client: connectors.HTTPClient(6 * time.Second), BaseURL: defaultBaseURL}
}

func (c *Connector) token() string { return c.res.Resolve(EnvToken) }

// Configured reports whether a token is set (configuredProcessors).
func (c *Connector) Configured() bool { return c.token() != "" }

// Outgoing returns recent outgoing transfers (GET /v1/transfers?limit=10).
//
//   - no token: (nil, nil), so the finances page hides the section;
//   - keyed, nothing sent: an empty, non-nil slice;
//   - any HTTP or transport failure: an error.
//
// FounderOS v1 skipped a non-OK response and returned [], which renders a dead
// token as "nothing sent". The bridge reports it as the error it is.
func (c *Connector) Outgoing(ctx context.Context) ([]Transfer, error) {
	tok := c.token()
	if tok == "" {
		return nil, nil
	}
	body, err := c.get(ctx, tok, "/v1/transfers?limit=10")
	if err != nil {
		return nil, err
	}
	return ParseTransfers(body), nil
}

// Status verifies the token with a read of /v1/profiles. It never writes.
func (c *Connector) Status(ctx context.Context) connectors.Status {
	st := connectors.Status{ID: Meta.ID, Name: Meta.Name, Kind: Meta.Kind}
	tok := c.token()
	if tok == "" {
		st.State = connectors.StateNotConfigured
		st.Detail = connectors.SetKeys(EnvToken)
		return st
	}
	body, err := c.get(ctx, tok, "/v1/profiles")
	if err != nil {
		st.State = connectors.StateError
		st.Detail = "Wise token set but verification failed: " + err.Error()
		return st
	}
	var profiles []json.RawMessage
	if err := json.Unmarshal(body, &profiles); err != nil {
		st.State = connectors.StateError
		st.Detail = "Wise token set but verification failed: unexpected profiles payload"
		return st
	}
	st.State = connectors.StateConnected
	st.Detail = fmt.Sprintf("Wise · %d profiles", len(profiles))
	st.Meta = map[string]any{"profiles": len(profiles)}
	return st
}

func (c *Connector) get(ctx context.Context, tok, path string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(c.BaseURL, "/")+path, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Accept", "application/json")
	res, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	body, err := io.ReadAll(io.LimitReader(res.Body, 8<<20))
	if err != nil {
		return nil, err
	}
	if res.StatusCode < 200 || res.StatusCode > 299 {
		return nil, fmt.Errorf("wise %s: HTTP %d", path, res.StatusCode)
	}
	return body, nil
}

// ParseTransfers maps a Wise /transfers payload (a bare array, or an object
// with a "transfers" array) to outgoing transfers. Malformed rows are skipped,
// never invented: targetValue must be a number, targetCurrency a string, and
// created a string or a number.
func ParseTransfers(raw []byte) []Transfer {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	var doc any
	if dec.Decode(&doc) != nil {
		return []Transfer{}
	}
	var arr []any
	switch v := doc.(type) {
	case []any:
		arr = v
	case map[string]any:
		if a, ok := v["transfers"].([]any); ok {
			arr = a
		}
	}
	out := []Transfer{}
	for _, row := range arr {
		t, _ := row.(map[string]any)
		if t == nil {
			continue
		}
		num, ok := t["targetValue"].(json.Number)
		if !ok {
			continue
		}
		value, err := num.Float64()
		if err != nil {
			continue
		}
		currency, ok := t["targetCurrency"].(string)
		if !ok {
			continue
		}
		created := t["created"]
		switch created.(type) {
		case string, json.Number:
		default:
			continue
		}
		tr := Transfer{
			AmountCents: jsRound(value * 100),
			Currency:    currency,
			Status:      "unknown",
			Created:     created,
		}
		if s, ok := t["status"].(string); ok {
			tr.Status = s
		}
		if r, ok := t["reference"].(string); ok {
			tr.Reference = &r
		}
		out = append(out, tr)
	}
	return out
}

// jsRound matches JavaScript's Math.round (half rounds toward +Inf).
func jsRound(x float64) int64 { return int64(math.Floor(x + 0.5)) }
