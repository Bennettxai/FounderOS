package phantom

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/rhl/businessos-backend/internal/founderos/connectors"
	"github.com/rhl/businessos-backend/internal/founderos/guard"
)

const address = "6BsHf5rDHjpzeKFNLBD4GnsBpCUKGUYPT4yjTRk2bSt2"

func resolver(t *testing.T, lines ...string) connectors.Resolver {
	t.Helper()
	p := filepath.Join(t.TempDir(), "env.local")
	if err := os.WriteFile(p, []byte(strings.Join(lines, "\n")+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PHANTOM_WALLET_ADDRESS", "")
	t.Setenv("SOLANA_RPC_URL", "")
	return connectors.Resolver{EnvLocal: p}
}

// rpcServer answers getBalance like api.mainnet-beta.solana.com does.
func rpcServer(t *testing.T, lamports string, calls *int32) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if calls != nil {
			atomic.AddInt32(calls, 1)
		}
		if r.Method != http.MethodPost {
			t.Errorf("rpc method = %s, want POST", r.Method)
		}
		var req struct {
			Method string   `json:"method"`
			Params []string `json:"params"`
		}
		body, _ := io.ReadAll(r.Body)
		if err := json.Unmarshal(body, &req); err != nil {
			t.Errorf("bad rpc body %q", body)
		}
		if req.Method != "getBalance" || len(req.Params) != 1 || req.Params[0] != address {
			t.Errorf("rpc request = %+v", req)
		}
		if lamports == "" {
			_, _ = io.WriteString(w, `{"jsonrpc":"2.0","error":{"code":-32602,"message":"Invalid param: WrongSize"},"id":1}`)
			return
		}
		_, _ = io.WriteString(w, `{"jsonrpc":"2.0","result":{"context":{"apiVersion":"2.2.1","slot":361111111},"value":`+lamports+`},"id":1}`)
	}))
}

func priceServer(t *testing.T, body string, status int) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("ids") != "solana" {
			t.Errorf("price query = %s", r.URL.RawQuery)
		}
		w.WriteHeader(status)
		_, _ = io.WriteString(w, body)
	}))
}

func TestMeta(t *testing.T) {
	if Meta != (connectors.Meta{ID: "phantom", Name: "Phantom", Kind: connectors.KindPayments}) {
		t.Fatalf("Meta = %+v", Meta)
	}
}

func TestSolFromLamportsAtNineDecimals(t *testing.T) {
	cases := map[int64]float64{10_200_600_779: 10.200601, 1_000_000_000: 1, 0: 0}
	for in, want := range cases {
		if got := SolFromLamports(in); got != want {
			t.Errorf("SolFromLamports(%d) = %v, want %v", in, got, want)
		}
	}
}

func TestShortAddressKeepsBothEnds(t *testing.T) {
	if got := ShortAddress(address); got != "6BsH…bSt2" {
		t.Errorf("got %q", got)
	}
	if got := ShortAddress("short"); got != "short" {
		t.Errorf("got %q", got)
	}
}

func TestStatusNotConfiguredSaysItIsNotAKey(t *testing.T) {
	c := New(resolver(t))
	s := c.Status(context.Background())
	if s.State != connectors.StateNotConfigured {
		t.Fatalf("state = %s", s.State)
	}
	if !strings.Contains(strings.ToLower(s.Detail), "not a key") {
		t.Errorf("detail = %q", s.Detail)
	}
	if s.ID != "phantom" || s.Kind != connectors.KindPayments {
		t.Errorf("identity = %+v", s)
	}
}

func TestStatusConnectedWithBalanceAndPrice(t *testing.T) {
	rpc := rpcServer(t, "10200600779", nil)
	defer rpc.Close()
	price := priceServer(t, `{"solana":{"usd":76.15}}`, 200)
	defer price.Close()
	c := New(resolver(t, "PHANTOM_WALLET_ADDRESS="+address, "SOLANA_RPC_URL="+rpc.URL))
	c.PriceURL = price.URL + "/api/v3/simple/price?ids=solana&vs_currencies=usd"

	b, err := c.Balance(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if b.SOL != 10.200601 || b.USDPerSOL == nil || *b.USDPerSOL != 76.15 || b.USDValue == nil || *b.USDValue != 776.78 {
		t.Fatalf("balance = %+v", b)
	}
	s := c.Status(context.Background())
	if s.State != connectors.StateConnected {
		t.Fatalf("state = %s (%s)", s.State, s.Detail)
	}
	if s.Detail != "6BsH…bSt2 · 10.200601 SOL" {
		t.Errorf("detail = %q", s.Detail)
	}
	if s.Meta["sol"] != 10.200601 || s.Meta["usdValue"] != 776.78 {
		t.Errorf("meta = %v", s.Meta)
	}
}

func TestPriceOutageKeepsTheBalanceAndNeverReadsZero(t *testing.T) {
	rpc := rpcServer(t, "1000000000", nil)
	defer rpc.Close()
	price := priceServer(t, `{"status":{"error_code":429}}`, 429)
	defer price.Close()
	c := New(resolver(t, "PHANTOM_WALLET_ADDRESS="+address, "SOLANA_RPC_URL="+rpc.URL))
	c.PriceURL = price.URL + "/?ids=solana"

	s := c.Status(context.Background())
	if s.State != connectors.StateConnected {
		t.Fatalf("state = %s (%s)", s.State, s.Detail)
	}
	if !strings.Contains(s.Detail, "(price unavailable)") {
		t.Errorf("detail = %q", s.Detail)
	}
	if _, ok := s.Meta["usdValue"]; ok {
		t.Errorf("unknown usd value must be absent, not 0: %v", s.Meta)
	}
	b, _ := c.Balance(context.Background())
	if b.USDValue != nil || b.USDPerSOL != nil {
		t.Errorf("unknown price must stay nil: %+v", b)
	}
}

func TestStatusErrorWhenRPCGivesNoBalance(t *testing.T) {
	rpc := rpcServer(t, "", nil)
	defer rpc.Close()
	c := New(resolver(t, "PHANTOM_WALLET_ADDRESS="+address, "SOLANA_RPC_URL="+rpc.URL))
	s := c.Status(context.Background())
	if s.State != connectors.StateError {
		t.Fatalf("state = %s", s.State)
	}
	if !strings.Contains(s.Detail, "Solana RPC did not return a balance") || !strings.Contains(s.Detail, "Invalid param") {
		t.Errorf("detail = %q", s.Detail)
	}
}

func TestStatusErrorWhenRPCUnreachable(t *testing.T) {
	rpc := rpcServer(t, "1", nil)
	url := rpc.URL
	rpc.Close()
	c := New(resolver(t, "PHANTOM_WALLET_ADDRESS="+address, "SOLANA_RPC_URL="+url))
	if s := c.Status(context.Background()); s.State != connectors.StateError {
		t.Fatalf("state = %s", s.State)
	}
}

// The balance read is a JSON-RPC POST. With writes off the guard must still
// let it through for the configured RPC host, and only for that host.
type okTransport struct{ hits int32 }

func (o *okTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	atomic.AddInt32(&o.hits, 1)
	return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader("{}")), Header: http.Header{}, Request: r}, nil
}

func TestRPCPostIsAllowlistedAsARead(t *testing.T) {
	t.Setenv("FOUNDEROS_WRITES", "0")
	if got := rpcReadPOST(DefaultRPC); got != "api.mainnet-beta.solana.com" {
		t.Fatalf("rpcReadPOST = %q", got)
	}
	if got := rpcReadPOST("https://mainnet.helius-rpc.com/?api-key=x"); got != "mainnet.helius-rpc.com/" {
		t.Fatalf("rpcReadPOST(helius) = %q", got)
	}
	base := &okTransport{}
	client := &http.Client{Transport: guard.Transport(base, rpcReadPOST(DefaultRPC))}
	res, err := client.Post(DefaultRPC, "application/json", strings.NewReader(`{}`))
	if err != nil {
		t.Fatalf("rpc read refused: %v", err)
	}
	res.Body.Close()
	if _, err := client.Post("https://evil.example/send", "application/json", strings.NewReader(`{}`)); err == nil {
		t.Fatal("a POST to another host must still be refused")
	}
	if base.hits != 1 {
		t.Errorf("hits = %d", base.hits)
	}
}
