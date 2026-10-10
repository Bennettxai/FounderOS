package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/rhl/businessos-backend/internal/founderos/connectors"
	"github.com/rhl/businessos-backend/internal/founderos/connectors/email"
	"github.com/rhl/businessos-backend/internal/founderos/connectors/fathomcalls"
	"github.com/rhl/businessos-backend/internal/founderos/connectors/gcal"
	"github.com/rhl/businessos-backend/internal/founderos/connectors/stripe"
	"github.com/rhl/businessos-backend/internal/founderos/connectors/trakyo"
	"github.com/rhl/businessos-backend/internal/founderos/connectors/typeform"
	"github.com/rhl/businessos-backend/internal/founderos/pages/funnel"
)

// fakeFunnel swaps the live lanes, source checks and comms feed for the test.
func fakeFunnel(t *testing.T, lanes funnel.Lanes, feed []email.CommsItem, feedOK bool) {
	t.Helper()
	oldLanes, oldStatuses, oldFeed, oldKeys := funnelLanesFor, funnelStatusesFor, funnelCommsFeed, funnelPaymentKeys
	t.Cleanup(func() {
		funnelLanesFor, funnelStatusesFor, funnelCommsFeed, funnelPaymentKeys = oldLanes, oldStatuses, oldFeed, oldKeys
	})
	funnelLanesFor = func(*Deps, map[string]string) funnel.Lanes { return lanes }
	funnelPaymentKeys = func(*Deps) (bool, bool) { return true, false }
	funnelStatusesFor = func(*Deps) []connectors.Connector {
		mk := func(id string, st connectors.State) connectors.Connector {
			return fakeConn{connectors.Status{ID: id, Name: id, State: st, Detail: id + " detail"}}
		}
		return []connectors.Connector{mk("typeform", connectors.StateConnected), mk("calendar", connectors.StateNotConfigured),
			mk("fathom", connectors.StateError), mk("trakyo", connectors.StateConnected), mk("meta-ads", connectors.StateNotConfigured)}
	}
	funnelCommsFeed = func(context.Context, *Deps, int) ([]email.CommsItem, bool) { return feed, feedOK }
}

func down() funnel.Lanes {
	no := errors.New("not configured")
	return funnel.Lanes{
		Typeform: func(context.Context, time.Time) ([]typeform.Lead, error) { return nil, no },
		Calendar: func(context.Context, time.Time) ([]gcal.CalEvent, error) { return nil, no },
		Fathom:   func(context.Context, time.Time) ([]fathomcalls.Call, error) { return nil, no },
		Stripe:   func(context.Context, time.Time) ([]stripe.Win, error) { return nil, no },
		Trakyo:   func(context.Context) ([]trakyo.Event, error) { return nil, no },
		Seed: func(context.Context, string) ([]funnel.Journey, error) {
			return nil, errors.New("founderos Postgres is not wired")
		},
	}
}

func funnelGetJSON(t *testing.T, path string, into any) int {
	t.Helper()
	w := httptest.NewRecorder()
	router(t, &Deps{Board: connectors.NewRegistry()}).ServeHTTP(w, authed(http.MethodGet, path, nil))
	if into != nil && w.Code == 200 {
		if err := json.Unmarshal(w.Body.Bytes(), into); err != nil {
			t.Fatalf("%s: %v %s", path, err, w.Body)
		}
	}
	return w.Code
}

func TestFunnelRejectsUnknownVentureAndStage(t *testing.T) {
	fakeFunnel(t, down(), nil, false)
	if code := funnelGetJSON(t, "/api/founderos/pages/funnel?venture=acme", nil); code != 400 {
		t.Fatalf("venture: %d", code)
	}
	if code := funnelGetJSON(t, "/api/founderos/pages/funnel?stage=won", nil); code != 400 {
		t.Fatalf("stage: %d", code)
	}
}

func TestFunnelWithNothingAnsweringSaysSoInsteadOfEmpty(t *testing.T) {
	fakeFunnel(t, down(), nil, false)
	var body struct {
		IsLive     bool              `json:"isLive"`
		Source     string            `json:"source"`
		SeedError  *string           `json:"seedError"`
		LaneErrors map[string]string `json:"laneErrors"`
		Journeys   []funnel.Journey  `json:"journeys"`
		Sources    []funnelSource    `json:"sources"`
		VSL        json.RawMessage   `json:"vsl"`
	}
	if code := funnelGetJSON(t, "/api/founderos/pages/funnel", &body); code != 200 {
		t.Fatalf("code %d", code)
	}
	if body.IsLive || body.Source != "seed" || body.SeedError == nil || body.LaneErrors["typeform"] == "" || len(body.Journeys) != 0 {
		t.Fatalf("body = %+v", body)
	}
	// v1's page draws no VSL card on /funnel (that lives on /analytics)
	if body.VSL != nil {
		t.Fatalf("vsl = %s", body.VSL)
	}
	// v1's control line: Attio (CRM) · GoHighLevel · Trakyo · Meta Ads, always.
	// A v2 lane joins only once it is set up (connected or erroring): an
	// unconfigured lane (calendar, unkeyed PayKit) stays off the line.
	ids := []string{}
	for _, s := range body.Sources {
		ids = append(ids, s.ID)
	}
	if strings.Join(ids, ",") != "attio,ghl,trakyo,meta-ads,typeform,fathom,stripe" {
		t.Fatalf("source ids = %v", ids)
	}
	attio, ghl := body.Sources[0], body.Sources[1]
	if attio.Name != "Attio (CRM)" || attio.State != connectors.StateNotConfigured || attio.Detail == "" || attio.Count != nil ||
		ghl.Name != "GoHighLevel" || ghl.State != connectors.StateNotConfigured || ghl.Detail == "" {
		t.Fatalf("crm checks = %+v / %+v", attio, ghl)
	}
	// Every lane errored: its count is unknown (null), never 0 leads; keyed
	// but silent Stripe reads error.
	src := map[string]funnelSource{}
	for _, s := range body.Sources {
		src[s.ID] = s
	}
	if src["stripe"].State != connectors.StateError || src["stripe"].Count != nil || src["typeform"].Count != nil || src["fathom"].Count != nil || src["trakyo"].Count != nil {
		t.Fatalf("sources = %+v", body.Sources)
	}
}

func TestFunnelComposesLiveLanesAndShapesThePage(t *testing.T) {
	l := down()
	now := time.Now().UTC()
	l.Typeform = func(context.Context, time.Time) ([]typeform.Lead, error) {
		return []typeform.Lead{{ResponseID: "r1", FormTitle: "Vantage discovery", Name: "Dana Reyes", Email: "dana@reyes.co", SubmittedAt: now.Add(-48 * time.Hour).Format(time.RFC3339), Hidden: map[string]string{}}}, nil
	}
	l.Stripe = func(context.Context, time.Time) ([]stripe.Win, error) {
		e := "buyer@x.com"
		return []stripe.Win{{ID: "ch_1", Venture: "launchpad-cohort", Email: &e, AmountUSD: 997, At: now.Format("2006-01-02")}}, nil
	}
	l.Paykit = func(context.Context, time.Time) ([]stripe.Win, error) {
		n := "Fan Buyer"
		return []stripe.Win{{ID: "fb-9", Source: "paykit", Venture: "launchpad-cohort", Name: &n, AmountUSD: 1500, Payments: 1, At: now.Format("2006-01-02")}}, nil
	}
	// The restored Attio / GoHighLevel history rides into the archive tab.
	l.Archive = func(context.Context, string) ([]funnel.Journey, error) {
		return []funnel.Journey{{ID: "attio-old", Name: "Historic", Venture: "vantage", Status: "engaged", Relationship: "warm", Likelihood: 50, CreatedAt: "2026-09-20",
			Touches: []funnel.Touch{{ID: "attio-old-t1", ContactID: "attio-old", Seq: 1, Stage: "engaged", Channel: "crm", Label: "Historic call", Source: "attio", At: now.Format("2006-01-02")}}}}, nil
	}
	fakeFunnel(t, l, []email.CommsItem{{Source: "email", Title: "hi", ReplyTo: "dana@reyes.co", Preview: "see you Friday", TS: now.Format(time.RFC3339)}}, true)
	funnelPaymentKeys = func(*Deps) (bool, bool) { return true, true }

	var body struct {
		IsLive    bool             `json:"isLive"`
		Source    string           `json:"source"`
		LiveLabel string           `json:"liveLabel"`
		Leads     *int             `json:"leads"`
		Journeys  []funnel.Journey `json:"journeys"`
		Summary   funnel.Summary   `json:"summary"`
		Radial    funnel.Radial    `json:"radial"`
		Volume    funnel.Volume    `json:"volume"`
		Archived  []funnel.Journey `json:"archived"`
		Sources   []funnelSource   `json:"sources"`
		Paykit    *int             `json:"paykit"`
		Attention funnel.Queue     `json:"attention"`
		Table     []funnel.Journey `json:"table"`
		LastMsgs  map[string]struct {
			Message *email.CommsItem `json:"message"`
		} `json:"lastMessages"`
	}
	if code := funnelGetJSON(t, "/api/founderos/pages/funnel", &body); code != 200 {
		t.Fatalf("code %d", code)
	}
	if !body.IsLive || body.Source != "typeform+stripe+paykit" || body.LiveLabel != "Typeform 1 + Stripe 1 + PayKit 1" || body.Leads == nil || *body.Leads != 1 || body.Paykit == nil || *body.Paykit != 1 {
		t.Fatalf("body = %+v", body)
	}
	if len(body.Journeys) != 3 || body.Summary.Clients != 3 || body.Summary.Converted != 2 || len(body.Radial.Nodes) != 3 || len(body.Radial.Segments) != 6 {
		t.Fatalf("shaped = %+v", body)
	}
	// v1 funnelVolume: every active lead, touches over the last 30 days, no entry window
	if body.Volume.RevenueUSD != 2497 || body.Volume.Window != nil || len(body.Volume.Series) != 30 || len(body.Volume.Meters) != 4 {
		t.Fatalf("volume = %+v", body.Volume)
	}
	if len(body.Archived) != 1 || body.Archived[0].ID != "attio-old" {
		t.Fatalf("archived = %+v", body.Archived)
	}
	src := map[string]funnelSource{}
	for _, s := range body.Sources {
		src[s.ID] = s
	}
	st, fb := src["stripe"], src["paykit"]
	if st.State != connectors.StateConnected || !st.Live || st.Count == nil || *st.Count != 1 || st.Detail != "1 successful payment in the last 90d" {
		t.Fatalf("stripe source = %+v", st)
	}
	if fb.State != connectors.StateConnected || !fb.Live || fb.Count == nil || *fb.Count != 1 || fb.Detail != "1 buyer paid in the last 90d" {
		t.Fatalf("paykit source = %+v", fb)
	}
	if body.LastMsgs != nil {
		t.Fatal("no stage selected: the comms lookup does not run")
	}

	var vantage struct {
		Journeys []funnel.Journey `json:"journeys"`
		Table    []funnel.Journey `json:"table"`
		LastMsgs map[string]struct {
			Message *email.CommsItem `json:"message"`
		} `json:"lastMessages"`
	}
	funnelGetJSON(t, "/api/founderos/pages/funnel?venture=vantage&stage=first_touch", &vantage)
	if len(vantage.Journeys) != 1 || vantage.Journeys[0].Venture != "vantage" || len(vantage.Table) != 1 {
		t.Fatalf("vantage = %+v", vantage)
	}
	if m := vantage.LastMsgs["typeform-r1"].Message; m == nil || m.Preview != "see you Friday" {
		t.Fatalf("last message = %+v", vantage.LastMsgs)
	}
}

func TestFunnelLeadMessageIsAHonestRead(t *testing.T) {
	fakeFunnel(t, down(), nil, false)
	if code := funnelGetJSON(t, "/api/founderos/pages/funnel/lead-message", nil); code != 400 {
		t.Fatalf("no name: %d", code)
	}
	var body struct {
		Message     *email.CommsItem `json:"message"`
		Unavailable bool             `json:"unavailable"`
	}
	funnelGetJSON(t, "/api/founderos/pages/funnel/lead-message?name=Dana+Reyes", &body)
	if !body.Unavailable || body.Message != nil {
		t.Fatalf("feed down = %+v", body)
	}
	fakeFunnel(t, down(), []email.CommsItem{{Source: "whatsapp", Title: "Dana Reyes", Sender: "Dana Reyes", Preview: "yo", TS: "2026-09-01T00:00:00Z"}}, true)
	body.Unavailable = true
	funnelGetJSON(t, "/api/founderos/pages/funnel/lead-message?name=Dana+Reyes&email=d@x.com", &body)
	if body.Unavailable || body.Message == nil || body.Message.Preview != "yo" {
		t.Fatalf("found = %+v", body)
	}
	funnelGetJSON(t, "/api/founderos/pages/funnel/lead-message?name=Nobody+Known", &body)
	if body.Unavailable || body.Message != nil {
		t.Fatalf("no thread = %+v", body)
	}
}

func TestFunnelAnalyticsValidatesAndSaysNotConfigured(t *testing.T) {
	old := funnelTrakyoFor
	t.Cleanup(func() { funnelTrakyoFor = old })
	t.Setenv("TRAKYO_API_KEY", "")
	p := filepath.Join(t.TempDir(), "env.local")
	_ = os.WriteFile(p, nil, 0o600)
	funnelTrakyoFor = func(*Deps) *trakyo.Connector {
		c := trakyo.New(connectors.Resolver{EnvLocal: p})
		c.CredFiles = []string{} // never the operator's real key files in a test
		c.BaseURL = "http://127.0.0.1:1"
		return c
	}
	if code := funnelGetJSON(t, "/api/founderos/pages/funnel/analytics?period=1y", nil); code != 400 {
		t.Fatalf("bad period: %d", code)
	}
	var body struct {
		Period  string `json:"period"`
		Content struct {
			State string `json:"state"`
		} `json:"content"`
		Channels []trakyo.ChannelCard `json:"channels"`
	}
	if code := funnelGetJSON(t, "/api/founderos/pages/funnel/analytics", &body); code != 200 {
		t.Fatalf("code %d", code)
	}
	if body.Period != "30d" || len(body.Channels) != 12 {
		t.Fatalf("body = %+v", body)
	}
	// Without a key it reads not_configured, never an empty "ready".
	if body.Content.State != trakyo.StateNotConfigured {
		t.Fatalf("content state %q", body.Content.State)
	}
}
