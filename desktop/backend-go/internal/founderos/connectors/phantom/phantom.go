// Package phantom ports FounderOS v1's lib/connectors/phantom.ts: the read-only
// SOL balance of the operator's Phantom wallet for the /trading top row.
//
// No credential of any kind is involved. A Solana address is public and the
// balance comes from a public RPC, so this connector holds a wallet ADDRESS,
// never a key. It can never move funds; there is no signing path here and
// there should not be one.
package phantom

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/rhl/businessos-backend/internal/founderos/connectors"
)

var Meta = connectors.Meta{ID: "phantom", Name: "Phantom", Kind: connectors.KindPayments}

const (
	DefaultRPC      = "https://api.mainnet-beta.solana.com"
	DefaultPriceURL = "https://api.coingecko.com/api/v3/simple/price?ids=solana&vs_currencies=usd"
	lamportsPerSOL  = 1_000_000_000
	timeout         = 8 * time.Second
)

// Balance is the SOL balance plus its USD value. A nil price means the price
// feed was down: the balance still stands, the dollar figure is unknown.
type Balance struct {
	Address   string   `json:"address"`
	SOL       float64  `json:"sol"`
	USDPerSOL *float64 `json:"usdPerSol"`
	USDValue  *float64 `json:"usdValue"`
	FetchedAt string   `json:"fetchedAt"`
}

type Connector struct {
	res connectors.Resolver
	// PriceURL is CoinGecko's simple-price endpoint; tests point it elsewhere.
	PriceURL string
	now      func() time.Time
}

func New(res connectors.Resolver) *Connector {
	return &Connector{res: res, PriceURL: DefaultPriceURL, now: time.Now}
}

// jsRound matches JavaScript's Math.round (half toward +Inf).
func jsRound(x float64) float64 { return math.Floor(x + 0.5) }

// SolFromLamports converts at nine decimals, rounded to six like the TS.
func SolFromLamports(lamports int64) float64 {
	return jsRound(float64(lamports)/lamportsPerSOL*1e6) / 1e6
}

// ShortAddress keeps both ends so the address stays identifiable at a glance.
func ShortAddress(address string) string {
	r := []rune(address)
	if len(r) <= 12 {
		return address
	}
	return string(r[:4]) + "…" + string(r[len(r)-4:])
}

// Address is PHANTOM_WALLET_ADDRESS, trimmed; empty when unset.
func (c *Connector) Address() string {
	return strings.TrimSpace(c.res.Resolve("PHANTOM_WALLET_ADDRESS"))
}

func (c *Connector) rpcURL() string {
	if v := strings.TrimSpace(c.res.Resolve("SOLANA_RPC_URL")); v != "" {
		return v
	}
	return DefaultRPC
}

// rpcReadPOST is the guard allowlist entry for the RPC: getBalance is a read
// that JSON-RPC happens to carry over POST.
func rpcReadPOST(rpc string) string {
	u, err := url.Parse(rpc)
	if err != nil {
		return ""
	}
	return u.Host + u.Path
}

func (c *Connector) client(rpc string) *http.Client {
	return connectors.HTTPClient(timeout, rpcReadPOST(rpc))
}

// Balance reads the wallet's balance. The price is a separate, best-effort
// call: a price outage must not hide the balance, so the USD fields go nil
// rather than the whole reading failing.
func (c *Connector) Balance(ctx context.Context) (*Balance, error) {
	address := c.Address()
	if address == "" {
		return nil, errors.New("no PHANTOM_WALLET_ADDRESS set")
	}
	rpc := c.rpcURL()
	client := c.client(rpc)

	body, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": 1, "method": "getBalance", "params": []string{address}})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, rpc, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode > 299 {
		return nil, fmt.Errorf("HTTP %d", res.StatusCode)
	}
	var out struct {
		Result *struct {
			Value *int64 `json:"value"`
		} `json:"result"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("unparseable RPC response: %w", err)
	}
	if out.Error != nil {
		return nil, fmt.Errorf("RPC error: %s", out.Error.Message)
	}
	if out.Result == nil || out.Result.Value == nil {
		return nil, errors.New("RPC response carried no balance")
	}

	sol := SolFromLamports(*out.Result.Value)
	b := &Balance{Address: address, SOL: sol, FetchedAt: c.now().UTC().Format("2006-01-02T15:04:05.000Z")}
	if usd, ok := c.price(ctx, client); ok {
		v := jsRound(sol*usd*100) / 100
		b.USDPerSOL, b.USDValue = &usd, &v
	}
	return b, nil
}

func (c *Connector) price(ctx context.Context, client *http.Client) (float64, bool) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.PriceURL, nil)
	if err != nil {
		return 0, false
	}
	res, err := client.Do(req)
	if err != nil {
		return 0, false
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode > 299 {
		return 0, false
	}
	var out struct {
		Solana *struct {
			USD *float64 `json:"usd"`
		} `json:"solana"`
	}
	if json.NewDecoder(res.Body).Decode(&out) != nil || out.Solana == nil || out.Solana.USD == nil {
		return 0, false
	}
	return *out.Solana.USD, true
}

// Status is honest: no address configured is a different thing from a dead RPC.
func (c *Connector) Status(ctx context.Context) connectors.Status {
	s := connectors.Status{ID: Meta.ID, Name: Meta.Name, Kind: Meta.Kind}
	if c.Address() == "" {
		s.State = connectors.StateNotConfigured
		s.Detail = "No PHANTOM_WALLET_ADDRESS set. It is a public address, not a key — no signing capability is involved."
		return s
	}
	b, err := c.Balance(ctx)
	if err != nil {
		s.State = connectors.StateError
		s.Detail = "Solana RPC did not return a balance: " + err.Error()
		return s
	}
	s.State = connectors.StateConnected
	s.Detail = fmt.Sprintf("%s · %s SOL", ShortAddress(b.Address), formatSOL(b.SOL))
	s.Meta = map[string]any{"sol": b.SOL}
	if b.USDValue == nil {
		s.Detail += " (price unavailable)"
	} else {
		s.Meta["usdValue"] = *b.USDValue
	}
	return s
}

// formatSOL prints like a JavaScript number: no trailing zeros.
func formatSOL(v float64) string {
	return strconv.FormatFloat(v, 'f', -1, 64)
}
