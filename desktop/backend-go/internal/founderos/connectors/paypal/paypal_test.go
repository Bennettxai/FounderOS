package paypal

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rhl/businessos-backend/internal/founderos/connectors"
	"github.com/rhl/businessos-backend/internal/founderos/guard"
)

func resolver(t *testing.T, body string) connectors.Resolver {
	t.Helper()
	t.Setenv(EnvClientID, "")
	t.Setenv(EnvClientSecret, "")
	p := filepath.Join(t.TempDir(), "env.local")
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return connectors.Resolver{EnvLocal: p}
}

func TestMetaMatchesFounderosOS(t *testing.T) {
	if Meta.ID != "paypal" || Meta.Name != "PayPal" || Meta.Kind != connectors.KindPayments {
		t.Fatalf("Meta = %+v", Meta)
	}
}

func TestConfiguredNeedsBothIDAndSecret(t *testing.T) {
	if New(resolver(t, "PAYPAL_CLIENT_ID=a\n")).Configured() {
		t.Error("id alone must not count as configured")
	}
	if New(resolver(t, "PAYPAL_CLIENT_SECRET=b\n")).Configured() {
		t.Error("secret alone must not count as configured")
	}
	if !New(resolver(t, "PAYPAL_CLIENT_ID=a\nPAYPAL_CLIENT_SECRET=b\n")).Configured() {
		t.Error("id + secret is configured")
	}
}

func TestStatusNotConfigured(t *testing.T) {
	st := New(resolver(t, "PAYPAL_CLIENT_ID=a\n")).Status(context.Background())
	if st.State != connectors.StateNotConfigured || !strings.Contains(st.Detail, "PAYPAL_CLIENT_SECRET") {
		t.Fatalf("%+v", st)
	}
}

func TestStatusConnectedMintsATokenWithBasicAuth(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, secret, ok := r.BasicAuth()
		if r.Method != http.MethodPost || r.URL.Path != "/v1/oauth2/token" || !ok || id != "cid" || secret != "csecret" {
			t.Errorf("unexpected %s %s basic=%v %q", r.Method, r.URL.Path, ok, id)
		}
		r.ParseForm()
		if r.PostForm.Get("grant_type") != "client_credentials" {
			t.Errorf("grant_type = %q", r.PostForm.Get("grant_type"))
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"scope":"https://uri.paypal.com/services/payments/payment","access_token":"A21AA","token_type":"Bearer","app_id":"APP-80W284485P519543T","expires_in":32400,"nonce":"n"}`))
	}))
	defer s.Close()
	c := New(resolver(t, "PAYPAL_CLIENT_ID=cid\nPAYPAL_CLIENT_SECRET=csecret\n"))
	c.BaseURL = s.URL
	st := c.Status(context.Background())
	if st.State != connectors.StateConnected || st.Detail != "PayPal · app APP-80W284485P519543T verified" {
		t.Fatalf("%+v", st)
	}
	if strings.Contains(st.Detail, "A21AA") {
		t.Fatal("the access token must never surface")
	}
}

func TestStatusErrorOnAuthFailureAndUnreachable(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":"invalid_client","error_description":"Client Authentication failed"}`))
	}))
	c := New(resolver(t, "PAYPAL_CLIENT_ID=cid\nPAYPAL_CLIENT_SECRET=bad\n"))
	c.BaseURL = s.URL
	st := c.Status(context.Background())
	if st.State != connectors.StateError || !strings.Contains(st.Detail, "Client Authentication failed") {
		t.Fatalf("auth failure: %+v", st)
	}
	s.Close()
	if st := c.Status(context.Background()); st.State != connectors.StateError {
		t.Fatalf("unreachable: %+v", st)
	}
}

func TestTokenEndpointIsTheOnlyReadPOST(t *testing.T) {
	if len(ReadPOSTs) != 1 || ReadPOSTs[0] != "api-m.paypal.com/v1/oauth2/token" {
		t.Fatalf("ReadPOSTs = %v", ReadPOSTs)
	}
}

func TestClientRefusesWritesWhenDisabled(t *testing.T) {
	t.Setenv("FOUNDEROS_WRITES", "0")
	c := New(resolver(t, ""))
	req, _ := http.NewRequest(http.MethodPost, "https://api-m.paypal.com/v1/payments/payouts", strings.NewReader("{}"))
	if _, err := c.client.Do(req); !errors.Is(err, guard.ErrWritesDisabled) {
		t.Fatalf("want ErrWritesDisabled, got %v", err)
	}
}
