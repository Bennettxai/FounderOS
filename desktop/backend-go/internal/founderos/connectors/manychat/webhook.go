package manychat

import (
	"bytes"
	"context"
	"crypto/subtle"
	"encoding/json"
	"io"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/rhl/businessos-backend/internal/founderos/connectors"
)

// DMMessage is FounderOS v1's SocialDmMessage (lib/schemas.ts), JSON-compatible.
type DMMessage struct {
	ID           string  `json:"id"`
	Platform     string  `json:"platform"`
	SubscriberID string  `json:"subscriberId"`
	Name         string  `json:"name"`
	Handle       *string `json:"handle"`
	Text         string  `json:"text"`
	Direction    string  `json:"direction"`
	Tag          *string `json:"tag"`
	TS           string  `json:"ts"`
	Source       string  `json:"source"`
}

// ParseWebhook maps the body of a ManyChat "External Request" to a DM, or nil
// when it lacks a subscriber id (lib/connectors/manychat-webhook.ts). The body
// is user-defined in ManyChat, so the common field-name variants are accepted.
// now supplies the fallback timestamp.
func ParseWebhook(raw []byte, now func() string) *DMMessage {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	var p map[string]any
	if dec.Decode(&p) != nil || p == nil {
		return nil
	}
	first := func(keys ...string) string {
		for _, k := range keys {
			if v := str(p[k]); v != "" {
				return v
			}
		}
		return ""
	}
	optional := func(keys ...string) *string {
		if v := first(keys...); v != "" {
			return &v
		}
		return nil
	}

	subscriberID := first("subscriber_id", "contact_id", "id")
	if subscriberID == "" {
		return nil
	}
	name := first("name", "full_name", "first_name")
	if name == "" {
		name = subscriberID
	}
	direction := "in"
	if first("direction") == "out" {
		direction = "out"
	}
	ts := first("ts", "timestamp")
	if ts == "" {
		ts = now()
	}
	id := first("message_id")
	if id == "" {
		id = "mc-" + subscriberID + "-" + ts
	}
	return &DMMessage{
		ID:           id,
		Platform:     "instagram",
		SubscriberID: subscriberID,
		Name:         name,
		Handle:       optional("handle", "ig_username", "username"),
		Text:         first("text", "message", "last_input_text"),
		Direction:    direction,
		Tag:          optional("tag", "message_tag"),
		TS:           ts,
		Source:       "manychat",
	}
}

// str is the TS str(): a trimmed non-empty string, or a finite number
// rendered the way JavaScript's String(n) renders it.
func str(v any) string {
	switch t := v.(type) {
	case string:
		return strings.TrimSpace(t)
	case json.Number:
		f, err := t.Float64()
		if err != nil || math.IsInf(f, 0) || math.IsNaN(f) {
			return ""
		}
		if math.Abs(f) < 1e21 {
			return strconv.FormatFloat(f, 'f', -1, 64)
		}
		return strconv.FormatFloat(f, 'g', -1, 64)
	}
	return ""
}

const maxWebhookBody = 1 << 20

// WebhookHandler is the ManyChat External Request ingest. POST parses the DM
// and hands it to Sink; GET is a health check. It reads
// MANYCHAT_WEBHOOK_SECRET at request time (env.local, then the process env)
// and, when set, requires a matching x-manychat-secret header.
type WebhookHandler struct {
	res connectors.Resolver
	// Sink stores a parsed DM (upsert by id, so replays don't duplicate).
	Sink func(ctx context.Context, m DMMessage) error
	// Stored counts stored Instagram DMs for the health check. Nil or an
	// error reads as unknown (null), never 0.
	Stored func(ctx context.Context) (int, error)
	Now    func() string
}

func NewWebhookHandler(res connectors.Resolver, sink func(context.Context, DMMessage) error) *WebhookHandler {
	return &WebhookHandler{
		res:  res,
		Sink: sink,
		Now:  func() string { return time.Now().UTC().Format("2006-01-02T15:04:05.000Z") },
	}
}

func (h *WebhookHandler) secret() string { return h.res.Resolve("MANYCHAT_WEBHOOK_SECRET") }

func (h *WebhookHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		h.ingest(w, r)
	case http.MethodGet:
		h.health(w, r)
	default:
		w.Header().Set("Allow", "GET, POST")
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
	}
}

func (h *WebhookHandler) ingest(w http.ResponseWriter, r *http.Request) {
	if secret := h.secret(); secret != "" {
		got := r.Header.Get("x-manychat-secret")
		if subtle.ConstantTimeCompare([]byte(got), []byte(secret)) != 1 {
			writeJSON(w, http.StatusUnauthorized, map[string]any{"error": "unauthorized"})
			return
		}
	}
	raw, _ := io.ReadAll(http.MaxBytesReader(w, r.Body, maxWebhookBody))
	m := ParseWebhook(raw, h.Now)
	if m == nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "payload missing a subscriber id"})
		return
	}
	if h.Sink != nil {
		if err := h.Sink(r.Context(), *m); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "id": m.ID, "subscriberId": m.SubscriberID})
}

func (h *WebhookHandler) health(w http.ResponseWriter, r *http.Request) {
	var stored any
	if h.Stored != nil {
		if n, err := h.Stored(r.Context()); err == nil {
			stored = n
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":       true,
		"endpoint": "manychat-webhook",
		"secured":  h.secret() != "",
		"stored":   stored,
	})
}

func writeJSON(w http.ResponseWriter, code int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(body)
}
