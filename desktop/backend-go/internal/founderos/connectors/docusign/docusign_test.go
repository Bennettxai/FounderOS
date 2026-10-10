package docusign

import (
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
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/rhl/businessos-backend/internal/founderos/connectors"
	"github.com/rhl/businessos-backend/internal/founderos/guard"
)

const (
	integrationKey = "ik-123"
	userID         = "user-456"
	accountID      = "acct-789"
)

var (
	keyOnce sync.Once
	testKey *rsa.PrivateKey
)

func rsaKey(t *testing.T) *rsa.PrivateKey {
	t.Helper()
	keyOnce.Do(func() {
		k, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			panic(err)
		}
		testKey = k
	})
	return testKey
}

func pkcs8B64(t *testing.T) string {
	der, err := x509.MarshalPKCS8PrivateKey(rsaKey(t))
	if err != nil {
		t.Fatal(err)
	}
	return base64.StdEncoding.EncodeToString(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der}))
}

func pkcs1B64(t *testing.T) string {
	der := x509.MarshalPKCS1PrivateKey(rsaKey(t))
	return base64.StdEncoding.EncodeToString(pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: der}))
}

var credKeys = []string{"DOCUSIGN_INTEGRATION_KEY", "DOCUSIGN_USER_ID", "DOCUSIGN_ACCOUNT_ID", "DOCUSIGN_PRIVATE_KEY_B64", "DOCUSIGN_ENV"}

// newConnector isolates creds: an env.local the test writes, an empty
// process env, and fallback files that do not exist.
func newConnector(t *testing.T, envLocal string) *Connector {
	t.Helper()
	for _, k := range credKeys {
		t.Setenv(k, "")
	}
	dir := t.TempDir()
	p := filepath.Join(dir, "env.local")
	if err := os.WriteFile(p, []byte(envLocal), 0o600); err != nil {
		t.Fatal(err)
	}
	c := New(connectors.Resolver{EnvLocal: p})
	c.Files = []string{filepath.Join(dir, "absent.env")}
	return c
}

func fullCreds(t *testing.T) string {
	return "DOCUSIGN_INTEGRATION_KEY=" + integrationKey + "\nDOCUSIGN_USER_ID=" + userID +
		"\nDOCUSIGN_ACCOUNT_ID=" + accountID + "\nDOCUSIGN_PRIVATE_KEY_B64=" + pkcs8B64(t) + "\n"
}

// Fixtures shaped like DocuSign's real answers.
const envelopesFixture = `{"resultSetSize":"2","totalSetSize":"2","startPosition":"0","endPosition":"1","envelopes":[
 {"envelopeId":"env-1","status":"completed","emailSubject":"Vantage - AI intake build agreement","statusChangedDateTime":"2026-08-05T12:00:00Z","createdDateTime":"2026-08-01T09:00:00Z"},
 {"envelopeId":"env-2","status":"sent","createdDateTime":"2026-08-06T09:00:00Z"}]}`

type fakeDocuSign struct {
	srv           *httptest.Server
	tokenStatus   int
	envStatus     int
	userinfoEmpty bool
	assertion     string
	envQuery      string
	sendBody      string
	hits          atomic.Int32
}

func newFake(t *testing.T) *fakeDocuSign {
	f := &fakeDocuSign{tokenStatus: 200, envStatus: 200}
	f.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.hits.Add(1)
		switch {
		case r.URL.Path == "/oauth/token":
			if r.Method != http.MethodPost || r.Header.Get("Content-Type") != "application/x-www-form-urlencoded" {
				t.Errorf("token: %s %s", r.Method, r.Header.Get("Content-Type"))
			}
			r.ParseForm()
			if r.Form.Get("grant_type") != "urn:ietf:params:oauth:grant-type:jwt-bearer" {
				t.Errorf("grant_type = %q", r.Form.Get("grant_type"))
			}
			f.assertion = r.Form.Get("assertion")
			if f.tokenStatus != 200 {
				w.WriteHeader(f.tokenStatus)
				io.WriteString(w, `{"error":"consent_required"}`)
				return
			}
			io.WriteString(w, `{"access_token":"tok-abc","token_type":"Bearer","expires_in":3600}`)
		case r.URL.Path == "/oauth/userinfo":
			if r.Header.Get("Authorization") != "Bearer tok-abc" {
				t.Errorf("userinfo auth = %q", r.Header.Get("Authorization"))
			}
			if f.userinfoEmpty {
				io.WriteString(w, `{"sub":"user-456","accounts":[]}`)
				return
			}
			io.WriteString(w, `{"sub":"user-456","accounts":[
			 {"account_id":"other","is_default":true,"base_uri":"http://127.0.0.1:1"},
			 {"account_id":"`+accountID+`","is_default":false,"base_uri":"`+f.srv.URL+`"}]}`)
		case r.URL.Path == "/restapi/v2.1/accounts/"+accountID+"/envelopes" && r.Method == http.MethodGet:
			f.envQuery = r.URL.RawQuery
			if f.envStatus != 200 {
				w.WriteHeader(f.envStatus)
				return
			}
			io.WriteString(w, envelopesFixture)
		case strings.HasPrefix(r.URL.Path, "/restapi/v2.1/accounts/"+accountID+"/envelopes/") && r.Method == http.MethodPut:
			b, _ := io.ReadAll(r.Body)
			f.sendBody = string(b)
			io.WriteString(w, `{"envelopeId":"env-9","status":"sent"}`)
		default:
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
			w.WriteHeader(404)
		}
	}))
	t.Cleanup(f.srv.Close)
	return f
}

func (f *fakeDocuSign) wire(c *Connector) *Connector {
	c.AuthBaseURL = f.srv.URL
	return c
}

func TestMetaMatchesFounderosOS(t *testing.T) {
	if Meta.ID != "docusign" || Meta.Name != "DocuSign" || Meta.Kind != connectors.KindCRM {
		t.Fatalf("Meta = %+v", Meta)
	}
}

func TestBuildJWTAssertionIsValidRS256(t *testing.T) {
	cfg := Config{IntegrationKey: integrationKey, UserID: userID, AccountID: accountID, AuthHost: "account-d.docusign.com", PrivateKey: rsaKey(t)}
	a, err := BuildJWTAssertion(cfg, 1_700_000_000)
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.Split(a, ".")
	if len(parts) != 3 || parts[2] == "" {
		t.Fatalf("assertion = %q", a)
	}
	var header, payload map[string]any
	hb, _ := base64.RawURLEncoding.DecodeString(parts[0])
	pb, _ := base64.RawURLEncoding.DecodeString(parts[1])
	json.Unmarshal(hb, &header)
	json.Unmarshal(pb, &payload)
	if header["alg"] != "RS256" || header["typ"] != "JWT" {
		t.Errorf("header = %v", header)
	}
	if payload["iss"] != integrationKey || payload["sub"] != userID || payload["aud"] != "account-d.docusign.com" ||
		payload["scope"] != "signature impersonation" || payload["iat"] != float64(1_700_000_000) || payload["exp"] != float64(1_700_000_000+3600) {
		t.Errorf("payload = %v", payload)
	}
	sig, _ := base64.RawURLEncoding.DecodeString(parts[2])
	sum := sha256.Sum256([]byte(parts[0] + "." + parts[1]))
	if err := rsa.VerifyPKCS1v15(&rsaKey(t).PublicKey, crypto.SHA256, sum[:], sig); err != nil {
		t.Errorf("signature does not verify: %v", err)
	}
}

func TestParseEnvelopes(t *testing.T) {
	rows := ParseEnvelopes([]byte(envelopesFixture))
	if len(rows) != 2 {
		t.Fatalf("rows = %+v", rows)
	}
	if rows[0] != (Envelope{EnvelopeID: "env-1", Subject: "Vantage - AI intake build agreement", Status: "completed", At: "2026-08-05T12:00:00Z"}) {
		t.Errorf("row0 = %+v", rows[0])
	}
	if rows[1].Subject != "Untitled envelope" || rows[1].At != "2026-08-06T09:00:00Z" {
		t.Errorf("row1 = %+v", rows[1])
	}
	for _, raw := range []string{`{}`, `null`, `{"envelopes":"nope"}`, `garbage`} {
		if got := ParseEnvelopes([]byte(raw)); len(got) != 0 {
			t.Errorf("%s: want empty, got %+v", raw, got)
		}
	}
}

func TestStatusNotConfiguredNamesTheKeys(t *testing.T) {
	c := newConnector(t, "DOCUSIGN_INTEGRATION_KEY=x\n") // partial creds are still not configured
	s := c.Status(context.Background())
	if s.State != connectors.StateNotConfigured || s.ID != "docusign" || s.Kind != connectors.KindCRM {
		t.Fatalf("status = %+v", s)
	}
	for _, k := range []string{"DOCUSIGN_INTEGRATION_KEY", "DOCUSIGN_USER_ID", "DOCUSIGN_ACCOUNT_ID", "DOCUSIGN_PRIVATE_KEY_B64", "DOCUSIGN_ENV=demo"} {
		if !strings.Contains(s.Detail, k) {
			t.Errorf("detail missing %s: %q", k, s.Detail)
		}
	}
}

func TestStatusConnectedCountsRecentEnvelopes(t *testing.T) {
	f := newFake(t)
	c := f.wire(newConnector(t, fullCreds(t)))
	now := time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC)
	c.Now = func() time.Time { return now }

	s := c.Status(context.Background())
	if s.State != connectors.StateConnected {
		t.Fatalf("status = %+v", s)
	}
	if s.Detail != "DocuSign reachable · 2 envelopes in the last 90 days" || s.Meta["envelopes"] != 2 {
		t.Errorf("status = %+v", s)
	}
	if !strings.Contains(f.envQuery, "from_date=2026-07-01T12%3A00%3A00.000Z") ||
		!strings.Contains(f.envQuery, "order_by=last_modified") || !strings.Contains(f.envQuery, "order=desc") {
		t.Errorf("envelopes query = %q", f.envQuery)
	}
	// The assertion is signed for the demo host by default.
	pb, _ := base64.RawURLEncoding.DecodeString(strings.Split(f.assertion, ".")[1])
	var payload map[string]any
	json.Unmarshal(pb, &payload)
	if payload["aud"] != "account-d.docusign.com" || payload["iat"] != float64(now.Unix()) {
		t.Errorf("payload = %v", payload)
	}
}

func TestProductionHostWhenEnvIsNotDemo(t *testing.T) {
	c := newConnector(t, fullCreds(t)+"DOCUSIGN_ENV=production\n")
	cfg, ok := c.config()
	if !ok || cfg.AuthHost != "account.docusign.com" {
		t.Fatalf("cfg = %+v ok=%v", cfg, ok)
	}
	c = newConnector(t, strings.Replace(fullCreds(t), pkcs8B64(t), pkcs1B64(t), 1))
	if _, ok := c.config(); !ok {
		t.Fatal("a PKCS#1 key must load too")
	}
}

func TestStatusErrorOnRejectedTokenExchange(t *testing.T) {
	f := newFake(t)
	f.tokenStatus = 400
	s := f.wire(newConnector(t, fullCreds(t))).Status(context.Background())
	if s.State != connectors.StateError || s.Detail != "DocuSign creds are set but the call failed: token exchange failed: HTTP 400" {
		t.Fatalf("status = %+v", s)
	}
}

func TestStatusErrorWhenEnvelopesFail(t *testing.T) {
	f := newFake(t)
	f.envStatus = 500
	s := f.wire(newConnector(t, fullCreds(t))).Status(context.Background())
	if s.State != connectors.StateError || !strings.Contains(s.Detail, "envelopes failed: HTTP 500") {
		t.Fatalf("status = %+v", s)
	}
}

func TestStatusErrorOnUndecodableKey(t *testing.T) {
	f := newFake(t)
	creds := strings.Replace(fullCreds(t), pkcs8B64(t), base64.StdEncoding.EncodeToString([]byte("not a pem")), 1)
	s := f.wire(newConnector(t, creds)).Status(context.Background())
	if s.State != connectors.StateError || !strings.Contains(s.Detail, "private key") {
		t.Fatalf("status = %+v", s)
	}
	if f.hits.Load() != 0 {
		t.Errorf("a bad key must fail before any call, got %d hits", f.hits.Load())
	}
}

func TestStatusErrorWhenUserinfoHasNoAccount(t *testing.T) {
	f := newFake(t)
	f.userinfoEmpty = true
	s := f.wire(newConnector(t, fullCreds(t))).Status(context.Background())
	if s.State != connectors.StateError || !strings.Contains(s.Detail, "userinfo returned no account base_uri") {
		t.Fatalf("status = %+v", s)
	}
}

func TestRecentEnvelopes(t *testing.T) {
	f := newFake(t)
	c := f.wire(newConnector(t, fullCreds(t)))
	rows, err := c.RecentEnvelopes(context.Background(), 1)
	if err != nil || len(rows) != 1 || rows[0].Subject != "Vantage - AI intake build agreement" {
		t.Fatalf("rows=%+v err=%v", rows, err)
	}

	f.envStatus = 500
	if rows, err := c.RecentEnvelopes(context.Background(), 5); err == nil || rows != nil {
		t.Errorf("a failure must be an error, never an empty list: rows=%v err=%v", rows, err)
	}

	unconfigured := newConnector(t, "")
	if rows, err := unconfigured.RecentEnvelopes(context.Background(), 5); !errors.Is(err, ErrNotConfigured) || rows != nil {
		t.Errorf("rows=%v err=%v", rows, err)
	}
}

func TestSendEnvelopeRefusedWhileBridgeWritesOff(t *testing.T) {
	t.Setenv("FOUNDEROS_WRITES", "0")
	guard.Reset()
	f := newFake(t)
	c := f.wire(newConnector(t, fullCreds(t)))
	err := c.SendEnvelope(context.Background(), "env-9")
	if !errors.Is(err, guard.ErrWritesDisabled) {
		t.Fatalf("err = %v", err)
	}
	if f.hits.Load() != 0 {
		t.Errorf("a refused send must not even fetch a token, got %d hits", f.hits.Load())
	}
	refused := guard.Refused()
	if len(refused) == 0 || refused[len(refused)-1].Action != "docusign.send_envelope" {
		t.Errorf("refusals = %v", refused)
	}
}

func TestSendEnvelopeWhenWritesOn(t *testing.T) {
	t.Setenv("FOUNDEROS_WRITES", "1")
	f := newFake(t)
	c := f.wire(newConnector(t, fullCreds(t)))
	if err := c.SendEnvelope(context.Background(), "env-9"); err != nil {
		t.Fatal(err)
	}
	if f.sendBody != `{"status":"sent"}` {
		t.Errorf("body = %q", f.sendBody)
	}
}

// stubRT answers for DocuSign's real hosts so the guard can be exercised
// against production hostnames without any network.
type stubRT struct {
	base string
	seen []string
}

func (s *stubRT) RoundTrip(r *http.Request) (*http.Response, error) {
	s.seen = append(s.seen, r.Method+" "+r.URL.Host+r.URL.Path)
	body := `{}`
	switch r.URL.Path {
	case "/oauth/token":
		body = `{"access_token":"tok-abc"}`
	case "/oauth/userinfo":
		body = `{"accounts":[{"account_id":"` + accountID + `","base_uri":"https://na4.docusign.net"}]}`
	case "/restapi/v2.1/accounts/" + accountID + "/envelopes":
		body = envelopesFixture
	}
	return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{}, Request: r}, nil
}

// The JWT token exchange is a POST, so it must be on the guard's read-POST
// list, or the status check could never run while writes are off.
func TestTokenExchangePassesTheGuardOnRealHosts(t *testing.T) {
	t.Setenv("FOUNDEROS_WRITES", "0")
	stub := &stubRT{}
	orig := http.DefaultTransport
	http.DefaultTransport = stub
	defer func() { http.DefaultTransport = orig }()

	for _, env := range []string{"demo", "production"} {
		c := newConnector(t, fullCreds(t)+"DOCUSIGN_ENV="+env+"\n")
		if s := c.Status(context.Background()); s.State != connectors.StateConnected {
			t.Fatalf("%s: status = %+v (seen %v)", env, s, stub.seen)
		}
	}
	want := []string{"POST account-d.docusign.com/oauth/token", "POST account.docusign.com/oauth/token"}
	for _, w := range want {
		found := false
		for _, s := range stub.seen {
			found = found || s == w
		}
		if !found {
			t.Errorf("%q never reached the transport: %v", w, stub.seen)
		}
	}
}
