package manychat

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/rhl/businessos-backend/internal/founderos/connectors"
)

func fixedNow() string { return "2026-07-18T00:00:00.000Z" }

func TestParseWebhookCanonicalPayload(t *testing.T) {
	m := ParseWebhook([]byte(`{"subscriber_id":"123","name":"Alex Rivera","handle":"alex.rivera","text":"do you work with agencies?","direction":"in","ts":"2026-07-18T14:02:00.000Z"}`), fixedNow)
	if m == nil {
		t.Fatal("nil message")
	}
	if m.Platform != "instagram" || m.SubscriberID != "123" || m.Name != "Alex Rivera" ||
		m.Handle == nil || *m.Handle != "alex.rivera" || m.Text != "do you work with agencies?" ||
		m.Direction != "in" || m.Source != "manychat" || m.TS != "2026-07-18T14:02:00.000Z" || m.Tag != nil {
		t.Fatalf("got %+v", m)
	}
	if m.ID != "mc-123-2026-07-18T14:02:00.000Z" {
		t.Errorf("id = %q", m.ID)
	}
}

func TestParseWebhookAliases(t *testing.T) {
	m := ParseWebhook([]byte(`{"contact_id":"999","full_name":"Jordan Blake","ig_username":"jordanbuilds","last_input_text":"SCALE","message_tag":"LEAD","message_id":"mid-7","timestamp":"2026-07-19T01:00:00Z"}`), fixedNow)
	if m == nil || m.SubscriberID != "999" || m.Name != "Jordan Blake" || *m.Handle != "jordanbuilds" ||
		m.Text != "SCALE" || m.Direction != "in" || *m.Tag != "LEAD" || m.ID != "mid-7" || m.TS != "2026-07-19T01:00:00Z" {
		t.Fatalf("got %+v", m)
	}
	// {{contact.id}} is a number in ManyChat's JSON, and first_name is the last name fallback.
	n := ParseWebhook([]byte(`{"id":4455667788,"first_name":"  Sam  ","direction":"out","message":"hey"}`), fixedNow)
	if n == nil || n.SubscriberID != "4455667788" || n.Name != "Sam" || n.Direction != "out" || n.Text != "hey" {
		t.Fatalf("got %+v", n)
	}
}

func TestParseWebhookDefaults(t *testing.T) {
	m := ParseWebhook([]byte(`{"subscriber_id":"77","direction":"sideways"}`), fixedNow)
	if m == nil || m.TS != "2026-07-18T00:00:00.000Z" || m.Name != "77" || m.Handle != nil || m.Text != "" || m.Direction != "in" {
		t.Fatalf("got %+v", m)
	}
	// Blank strings count as missing, exactly like the TS str().
	b := ParseWebhook([]byte(`{"subscriber_id":"   ","contact_id":"8","name":"  "}`), fixedNow)
	if b == nil || b.SubscriberID != "8" || b.Name != "8" {
		t.Fatalf("got %+v", b)
	}
}

func TestParseWebhookRejectsMissingSubscriber(t *testing.T) {
	for _, raw := range []string{`{"text":"orphan"}`, `null`, `"nope"`, `[1,2]`, `not json`, ``} {
		if m := ParseWebhook([]byte(raw), fixedNow); m != nil {
			t.Errorf("%q: want nil, got %+v", raw, m)
		}
	}
}

type recorder struct {
	got []DMMessage
	err error
}

func (r *recorder) sink(_ context.Context, m DMMessage) error {
	if r.err != nil {
		return r.err
	}
	r.got = append(r.got, m)
	return nil
}

func newWebhook(t *testing.T, rec *recorder) (*WebhookHandler, *harness) {
	h := newHarness(t)
	wh := NewWebhookHandler(connectors.Resolver{EnvLocal: h.envLocal}, rec.sink)
	wh.Now = fixedNow
	return wh, h
}

func post(t *testing.T, handler http.Handler, body string, header map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/webhooks/manychat", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	for k, v := range header {
		req.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	return w
}

func decode(t *testing.T, w *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var out map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatalf("body %q: %v", w.Body.String(), err)
	}
	return out
}

func TestWebhookIngestsAndReturnsParsedEvent(t *testing.T) {
	rec := &recorder{}
	wh, _ := newWebhook(t, rec)
	w := post(t, wh, `{"subscriber_id":"501","name":"Casey","text":"came from manychat","ts":"2026-07-18T10:00:00Z"}`, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("code %d body %s", w.Code, w.Body)
	}
	out := decode(t, w)
	if out["ok"] != true || out["id"] != "mc-501-2026-07-18T10:00:00Z" || out["subscriberId"] != "501" {
		t.Errorf("body = %v", out)
	}
	if len(rec.got) != 1 || rec.got[0].Text != "came from manychat" || rec.got[0].Source != "manychat" {
		t.Errorf("sink got %+v", rec.got)
	}
}

func TestWebhookRejectsUnparseablePayload(t *testing.T) {
	rec := &recorder{}
	wh, _ := newWebhook(t, rec)
	for _, body := range []string{`{"text":"no id"}`, `{not json`} {
		w := post(t, wh, body, nil)
		if w.Code != http.StatusBadRequest || decode(t, w)["error"] != "payload missing a subscriber id" {
			t.Errorf("%s: code %d body %s", body, w.Code, w.Body)
		}
	}
	if len(rec.got) != 0 {
		t.Errorf("nothing may be stored, got %+v", rec.got)
	}
}

func TestWebhookEnforcesSecretWhenSet(t *testing.T) {
	rec := &recorder{}
	wh, h := newWebhook(t, rec)
	h.setEnvLocal(t, "MANYCHAT_WEBHOOK_SECRET=s3cret\n")
	body := `{"subscriber_id":"1","text":"hi"}`

	for _, hdr := range []map[string]string{nil, {"x-manychat-secret": "wrong"}, {"x-manychat-secret": "s3cre"}} {
		w := post(t, wh, body, hdr)
		if w.Code != http.StatusUnauthorized || decode(t, w)["error"] != "unauthorized" {
			t.Errorf("%v: code %d body %s", hdr, w.Code, w.Body)
		}
	}
	if len(rec.got) != 0 {
		t.Fatalf("an unauthorized request must not be stored")
	}
	if w := post(t, wh, body, map[string]string{"X-ManyChat-Secret": "s3cret"}); w.Code != http.StatusOK {
		t.Errorf("good secret: code %d body %s", w.Code, w.Body)
	}
}

func TestWebhookSecretFromProcessEnv(t *testing.T) {
	rec := &recorder{}
	wh, _ := newWebhook(t, rec)
	t.Setenv("MANYCHAT_WEBHOOK_SECRET", "from-env")
	if w := post(t, wh, `{"subscriber_id":"1"}`, nil); w.Code != http.StatusUnauthorized {
		t.Errorf("code %d", w.Code)
	}
	if w := post(t, wh, `{"subscriber_id":"1"}`, map[string]string{"x-manychat-secret": "from-env"}); w.Code != http.StatusOK {
		t.Errorf("code %d", w.Code)
	}
}

func TestWebhookSinkFailureIs500(t *testing.T) {
	rec := &recorder{err: errors.New("db locked")}
	wh, _ := newWebhook(t, rec)
	if w := post(t, wh, `{"subscriber_id":"1"}`, nil); w.Code != http.StatusInternalServerError {
		t.Errorf("code %d body %s", w.Code, w.Body)
	}
}

func TestWebhookHealthCheck(t *testing.T) {
	rec := &recorder{}
	wh, h := newWebhook(t, rec)
	get := func() map[string]any {
		w := httptest.NewRecorder()
		wh.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/webhooks/manychat", nil))
		if w.Code != http.StatusOK {
			t.Fatalf("code %d", w.Code)
		}
		return decode(t, w)
	}
	out := get()
	if out["ok"] != true || out["endpoint"] != "manychat-webhook" || out["secured"] != false {
		t.Errorf("body = %v", out)
	}
	if v, present := out["stored"]; !present || v != nil {
		t.Errorf("stored must read unknown (null) without a counter, got %v", out["stored"])
	}

	h.setEnvLocal(t, "MANYCHAT_WEBHOOK_SECRET=s\n")
	wh.Stored = func(context.Context) (int, error) { return 4, nil }
	out = get()
	if out["secured"] != true || out["stored"] != float64(4) {
		t.Errorf("body = %v", out)
	}
	wh.Stored = func(context.Context) (int, error) { return 0, errors.New("db down") }
	if out = get(); out["stored"] != nil {
		t.Errorf("a failed count must read unknown, never 0: %v", out)
	}
}

func TestWebhookOtherMethods(t *testing.T) {
	wh, _ := newWebhook(t, &recorder{})
	w := httptest.NewRecorder()
	wh.ServeHTTP(w, httptest.NewRequest(http.MethodDelete, "/api/webhooks/manychat", nil))
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("code %d", w.Code)
	}
}
