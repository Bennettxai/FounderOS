package api

// /funnel (spec 6.8): FounderOS v1 app/funnel/page.tsx + app/api/funnel/*.
//
//	GET /pages/funnel               composed journeys + everything the page draws
//	GET /pages/funnel/analytics     Trakyo channel funnels (?period=7d|30d|90d)
//	GET /pages/funnel/lead-message  last message with one lead (a read)

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rhl/businessos-backend/internal/founderos/connectors"
	"github.com/rhl/businessos-backend/internal/founderos/connectors/email"
	"github.com/rhl/businessos-backend/internal/founderos/connectors/fathomcalls"
	"github.com/rhl/businessos-backend/internal/founderos/connectors/gcal"
	"github.com/rhl/businessos-backend/internal/founderos/connectors/metaads"
	"github.com/rhl/businessos-backend/internal/founderos/connectors/paykit"
	"github.com/rhl/businessos-backend/internal/founderos/connectors/stripe"
	"github.com/rhl/businessos-backend/internal/founderos/connectors/trakyo"
	"github.com/rhl/businessos-backend/internal/founderos/connectors/typeform"
	"github.com/rhl/businessos-backend/internal/founderos/pages/funnel"
	"github.com/rhl/businessos-backend/internal/founderos/pages/vsl"
)

func init() { RegisterPage(registerFunnel) }

func registerFunnel(s *gin.RouterGroup, d *Deps) {
	s.GET("/pages/funnel", funnelPage(d))
	s.GET("/pages/funnel/analytics", funnelAnalytics(d))
	s.GET("/pages/funnel/lead-message", funnelLeadMessage(d))
}

// funnelWorkspaceIDs resolves workspace slugs to ids (founderos_* rows are
// scoped by workspace_id). An empty map when Postgres is not wired.
func funnelWorkspaceIDs(ctx context.Context, pool *pgxpool.Pool) map[string]string {
	out := map[string]string{}
	if pool == nil {
		return out
	}
	rows, err := pool.Query(ctx, `SELECT slug, id::text FROM workspaces WHERE slug = ANY($1)`,
		[]string{"vantage", "launchpad-cohort", "founderos", "personal"})
	if err != nil {
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var slug, id string
		if rows.Scan(&slug, &id) == nil {
			out[slug] = id
		}
	}
	if rows.Err() != nil {
		// a read cut short is no read: callers treat a missing id as unwired
		return map[string]string{}
	}
	return out
}

var errFunnelPinned = errors.New("FUNNEL_PROVIDER=seed pins the seeded funnel")

// funnelLanesFor wires the live lanes to the ported connectors. Tests swap it.
var funnelLanesFor = func(d *Deps, ws map[string]string) funnel.Lanes {
	res := d.Resolver
	pinned := func() bool {
		v := res.Resolve("FUNNEL_PROVIDER")
		return v != "" && v != "live" && v != "attio"
	}
	seed := &funnel.PgSeed{Pool: d.Pool, Workspaces: ws}
	return funnel.Lanes{
		Typeform: func(ctx context.Context, now time.Time) ([]typeform.Lead, error) {
			if pinned() {
				return nil, errFunnelPinned
			}
			return typeform.New(res).Leads(ctx, typeform.LeadOptions{Now: now})
		},
		Calendar: func(ctx context.Context, now time.Time) ([]gcal.CalEvent, error) {
			if pinned() {
				return nil, errFunnelPinned
			}
			return gcal.New(res).CalendarBookings(ctx, gcal.BookingOptions{Now: now})
		},
		Fathom: func(ctx context.Context, now time.Time) ([]fathomcalls.Call, error) {
			if pinned() {
				return nil, errFunnelPinned
			}
			return fathomcalls.New(res).Calls(ctx, fathomcalls.CallsOptions{Now: now})
		},
		// Both accounts (lib/funnel-stripe.ts); each degrades on its own, and
		// the lane errors only when no account answered.
		Stripe: func(ctx context.Context, now time.Time) ([]stripe.Win, error) {
			if pinned() {
				return nil, errFunnelPinned
			}
			var wins []stripe.Win
			var problems []string
			answered := 0
			for _, acct := range []stripe.Account{stripe.LaunchpadCohort, stripe.Vantage} {
				c := stripe.NewAccount(res, acct)
				if !c.Configured() {
					continue
				}
				w, err := c.FunnelWins(ctx, now)
				if err != nil {
					problems = append(problems, acct.Name+": "+err.Error())
					continue
				}
				answered++
				wins = append(wins, w...)
			}
			if answered == 0 {
				if len(problems) == 0 {
					return nil, stripe.ErrNotConfigured
				}
				return nil, errors.New(strings.Join(problems, "; "))
			}
			return wins, nil
		},
		// PayKit buyers (lib/funnel-paykit.ts): every /customers page of
		// each keyed account; an account that fails any page contributes
		// nothing (a partial customer list would be a wrong sum).
		Paykit: func(ctx context.Context, now time.Time) ([]stripe.Win, error) {
			if pinned() {
				return nil, errFunnelPinned
			}
			var wins []stripe.Win
			var problems []string
			answered := 0
			for _, acct := range []struct {
				a       paykit.Account
				venture string
			}{{paykit.LaunchpadCohort, "launchpad-cohort"}, {paykit.Vantage, "vantage"}} {
				c := paykit.NewAccount(res, acct.a)
				if !c.Configured() {
					continue
				}
				pages, err := c.CustomerPages(ctx)
				if err != nil {
					problems = append(problems, acct.a.Name+": "+err.Error())
					continue
				}
				answered++
				for _, p := range pages {
					wins = append(wins, funnel.MapPaykitCustomers(p, acct.venture, now)...)
				}
			}
			if answered == 0 {
				if len(problems) == 0 {
					return nil, paykit.ErrNotConfigured
				}
				return nil, errors.New(strings.Join(problems, "; "))
			}
			return wins, nil
		},
		Trakyo:  func(ctx context.Context) ([]trakyo.Event, error) { return trakyo.New(res).Leads(ctx) },
		Seed:    seed.Journeys,
		Archive: (&funnel.PgArchive{Pool: d.Pool, Workspaces: ws}).Journeys,
	}
}

// funnelPaymentKeys says whether any Stripe / PayKit account is keyed; the
// composed pull is the probe for each (prod has no separate status check on
// /funnel). Tests swap it.
var funnelPaymentKeys = func(d *Deps) (stripeKeyed, paykitKeyed bool) {
	r := d.Resolver
	return r.Resolve("STRIPE_SECRET_KEY") != "" || r.Resolve("STRIPE_VANTAGE_KEY") != "",
		r.Resolve("PAYKIT_LC_KEY") != "" || r.Resolve("PAYKIT_VANTAGE_KEY") != ""
}

// paymentSource is the Stripe / PayKit check on the control line: unkeyed
// reads not configured, a pull that answered reads connected with its count,
// a keyed pull that did not answer reads error (count unknown, never 0).
func paymentSource(id, name, keys string, keyed bool, wins []stripe.Win, answered, silent func(int) string) funnelSource {
	src := funnelSource{Status: connectors.Status{ID: id, Name: name, Kind: connectors.KindPayments}}
	switch {
	case !keyed:
		src.State, src.Detail = connectors.StateNotConfigured, keys+" not set"
	case wins != nil:
		src.State, src.Detail = connectors.StateConnected, answered(len(wins))
		src.Live, src.Count = len(wins) > 0, intp(len(wins))
	default:
		src.State, src.Detail = connectors.StateError, silent(0)
	}
	return src
}

func pluralS(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}

// funnelStatusesFor are the source checks on the control line: leads, calls
// booked, calls held, attribution, paid. Tests swap it.
var funnelStatusesFor = func(d *Deps) []connectors.Connector {
	res := d.Resolver
	return []connectors.Connector{typeform.New(res), gcal.New(res), fathomcalls.New(res), trakyo.New(res), metaads.New(res)}
}

type funnelSource struct {
	connectors.Status
	Live  bool `json:"live"`
	Count *int `json:"count"`
}

func funnelStatuses(ctx context.Context, conns []connectors.Connector) []connectors.Status {
	out := make([]connectors.Status, len(conns))
	var wg sync.WaitGroup
	for i, c := range conns {
		wg.Add(1)
		go func() {
			defer wg.Done()
			out[i] = c.Status(ctx)
		}()
	}
	wg.Wait()
	return out
}

// funnelCRMChecks are FounderOS v1's first two source checks. The CRM lanes
// are retired in this build (their history lives in the archive), so they
// always read not configured; nothing is polled for them.
var funnelCRMChecks = []funnelSource{
	{Status: connectors.Status{ID: "attio", Name: "Attio (CRM)", Kind: connectors.KindCRM, State: connectors.StateNotConfigured,
		Detail: "Attio is not connected · its past journeys sit in the archive"}},
	{Status: connectors.Status{ID: "ghl", Name: "GoHighLevel", Kind: connectors.KindCRM, State: connectors.StateNotConfigured,
		Detail: "GoHighLevel is not connected · its past journeys sit in the archive"}},
}

// funnelControlLine is v1's control line (Attio (CRM) · GoHighLevel · Trakyo
// · Meta Ads), always shown, then any other lane that has been set up: a
// connected or erroring Typeform / Calendar / Fathom / Stripe / PayKit joins
// the line, an unconfigured one stays off it.
func funnelControlLine(lanes []funnelSource) []funnelSource {
	out := append([]funnelSource{}, funnelCRMChecks...)
	for _, id := range []string{"trakyo", "meta-ads"} {
		for _, l := range lanes {
			if l.ID == id {
				out = append(out, l)
			}
		}
	}
	for _, l := range lanes {
		if l.ID == "trakyo" || l.ID == "meta-ads" || l.State == connectors.StateNotConfigured {
			continue
		}
		out = append(out, l)
	}
	return out
}

type funnelLastMsg struct {
	Message *email.CommsItem `json:"message"`
}

// funnelCommsFeed is gatherCommsFeed: WhatsApp (device push), the four
// inboxes and Slack, newest first. ok=false only when every lane failed.
var funnelCommsFeed = func(ctx context.Context, d *Deps, limit int) ([]email.CommsItem, bool) {
	type lane struct {
		items []email.CommsItem
		err   error
	}
	lanes := make([]lane, 3)
	var wg sync.WaitGroup
	wg.Add(3)
	go func() {
		defer wg.Done()
		if d.Devices == nil {
			lanes[0].err = errors.New("no device receiver")
			return
		}
		r, err := d.Devices.WhatsAppChats(ctx)
		if err != nil {
			lanes[0].err = err
			return
		}
		for i, c := range r.Data {
			if i == 15 {
				break
			}
			lanes[0].items = append(lanes[0].items, email.CommsItem{Source: "whatsapp", Title: c.Title, Sender: c.Sender, ReplyTo: c.ReplyTo, Preview: c.Preview, TS: c.TS, Unread: c.Unread})
		}
	}()
	go func() {
		defer wg.Done()
		lanes[1].items, lanes[1].err = depsEmail(d).LatestEmails(ctx, 5)
	}()
	go func() {
		defer wg.Done()
		msgs, err := depsSlack(d).RecentMessages(ctx, 15)
		if err != nil {
			lanes[2].err = err
			return
		}
		for _, m := range msgs {
			ts := m.TS
			if f, err := parseSlackTS(m.TS); err == nil {
				ts = f.UTC().Format("2006-01-02T15:04:05.000Z")
			}
			text := clip(m.Text, 140)
			lanes[2].items = append(lanes[2].items, email.CommsItem{Source: "slack", Title: "#" + m.Channel + " — " + m.User, Sender: m.User, ReplyTo: m.Channel, Preview: text, TS: ts})
		}
	}()
	wg.Wait()
	var items []email.CommsItem
	failed := 0
	for _, l := range lanes {
		if l.err != nil {
			failed++
			continue
		}
		items = append(items, l.items...)
	}
	if failed == len(lanes) {
		return nil, false
	}
	sort.SliceStable(items, func(a, b int) bool { return items[a].TS > items[b].TS })
	if len(items) > limit {
		items = items[:limit]
	}
	return items, true
}

func parseSlackTS(ts string) (time.Time, error) {
	f, err := strconv.ParseFloat(ts, 64)
	if err != nil {
		return time.Time{}, err
	}
	return time.UnixMilli(int64(f * 1000)), nil
}

// The comms feed walks live IMAP: cap it so a pinned dossier never hangs.
const funnelFeedBudget = 4 * time.Second

func funnelFeedWithin(ctx context.Context, d *Deps) ([]email.CommsItem, bool) {
	ctx, cancel := context.WithTimeout(ctx, funnelFeedBudget)
	defer cancel()
	type res struct {
		items []email.CommsItem
		ok    bool
	}
	done := make(chan res, 1)
	go func() {
		items, ok := funnelCommsFeed(ctx, d, 200)
		done <- res{items, ok}
	}()
	select {
	case r := <-done:
		return r.items, r.ok
	case <-ctx.Done():
		return nil, false
	}
}

func funnelPage(d *Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		venture := c.Query("venture")
		if venture != "" && !funnel.ValidVenture(venture) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "unknown venture: " + venture})
			return
		}
		stage := c.Query("stage")
		if stage != "" && !funnel.ValidStage(stage) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "unknown stage: " + stage})
			return
		}
		ctx, cancel := context.WithTimeout(c.Request.Context(), 20*time.Second)
		defer cancel()
		now := time.Now()
		ws := funnelWorkspaceIDs(ctx, d.Pool)

		var statuses []connectors.Status
		var side sync.WaitGroup
		side.Add(1)
		go func() {
			defer side.Done()
			statuses = funnelStatuses(ctx, funnelStatusesFor(d))
		}()

		comp := funnel.Compose(ctx, now, venture, funnelLanesFor(d, ws))
		active, archived := funnel.Split(comp.Journeys, now)
		// the restored CRM history joins the archive regardless of its age
		archived = append(archived, comp.ArchivedJourneys...)
		summary := funnel.Summarize(active)

		touched := func(source string) int {
			n := 0
			for _, j := range comp.Journeys {
				for _, t := range j.Touches {
					if t.Source == source {
						n++
						break
					}
				}
			}
			return n
		}
		booked := touched("calendar")
		// Trakyo is live on the page when it actually attributed a journey's first touch.
		attributed := touched("trakyo")
		var liveParts []string
		if n := len(comp.TypeformLeads); n > 0 {
			liveParts = append(liveParts, "Typeform "+itoa(n))
		}
		if n := len(comp.Calls); n > 0 {
			liveParts = append(liveParts, "Fathom "+itoa(n))
		}
		if n := len(comp.StripeWins); n > 0 {
			liveParts = append(liveParts, "Stripe "+itoa(n))
		}
		if n := len(comp.PaykitWins); n > 0 {
			liveParts = append(liveParts, "PayKit "+itoa(n))
		}

		stageCounts := map[string]int{}
		for _, j := range active {
			stageCounts[j.Status]++
		}
		table := active
		if stage != "" {
			table = []funnel.Journey{}
			for _, j := range active {
				if j.Status == stage {
					table = append(table, j)
				}
			}
		}
		// Segment select: a bounded row set can afford the live comms lookup.
		var lastMsgs map[string]funnelLastMsg
		commsUnavailable := false
		if stage != "" && len(table) > 0 {
			if feed, ok := funnelFeedWithin(ctx, d); ok {
				lastMsgs = map[string]funnelLastMsg{}
				for _, j := range table {
					lastMsgs[j.ID] = funnelLastMsg{Message: funnel.LastMessageFor(j.Name, j.Email, feed)}
				}
			} else {
				commsUnavailable = true
			}
		}
		side.Wait()

		lanes := make([]funnelSource, len(statuses))
		counts := []*int{intp(len(comp.TypeformLeads)), intp(booked), intp(len(comp.Calls)), intp(attributed), nil}
		// A lane that errored has an unknown count (null), never 0.
		for i, lane := range []string{"typeform", "calendar", "fathom", "trakyo"} {
			if comp.Errors[lane] != "" {
				counts[i] = nil
			}
		}
		if !comp.IsLive {
			counts[3] = nil // Trakyo is only asked once a live lane answered
		}
		lives := []bool{len(comp.TypeformLeads) > 0, booked > 0, len(comp.Calls) > 0, attributed > 0, false}
		for i, st := range statuses {
			lanes[i] = funnelSource{Status: st}
			if i < len(counts) {
				lanes[i].Live, lanes[i].Count = lives[i], counts[i]
			}
		}
		stripeKeyed, paykitKeyed := funnelPaymentKeys(d)
		lanes = append(lanes,
			paymentSource("stripe", "Stripe", "STRIPE_SECRET_KEY / STRIPE_VANTAGE_KEY", stripeKeyed, comp.StripeWins,
				func(n int) string {
					return fmt.Sprintf("%d successful %s in the last 90d", n, pluralS(n, "payment", "payments"))
				},
				func(int) string { return "Stripe charges did not answer" }),
			paymentSource("paykit", "PayKit", "PAYKIT_LC_KEY / PAYKIT_VANTAGE_KEY", paykitKeyed, comp.PaykitWins,
				func(n int) string { return fmt.Sprintf("%d %s paid in the last 90d", n, pluralS(n, "buyer", "buyers")) },
				func(int) string { return "PayKit /customers did not answer" }),
		)
		sources := funnelControlLine(lanes)

		body := gin.H{
			"now":              now.UTC().Format(time.RFC3339),
			"venture":          nilIfEmpty(venture),
			"stage":            nilIfEmpty(stage),
			"stages":           funnel.Stages,
			"stageLabels":      funnel.StageLabel,
			"acquisitions":     funnel.Acquisitions,
			"summary":          summary,
			"journeys":         active,
			"archived":         archived,
			"table":            table,
			"stageCounts":      stageCounts,
			"lastMessages":     lastMsgs,
			"commsUnavailable": commsUnavailable,
			"source":           funnel.SourceLabel(comp),
			"isLive":           comp.IsLive,
			"liveLabel":        strings.Join(liveParts, " + "),
			"laneErrors":       comp.Errors,
			"seedError":        nilIfEmpty(comp.SeedError),
			"radial":           funnel.RadialModel(active, now),
			"attention":        funnel.Attention(active, now),
			"volume":           funnel.VolumeOf(active, len(archived), now, 30),
			"sources":          sources,
			"constants": gin.H{
				"stallDays": funnel.StallDays, "decayDays": funnel.DecayDays, "decayFadeStart": funnel.DecayFadeStart,
			},
		}
		if comp.IsLive {
			body["leads"] = len(comp.TypeformLeads)
			body["calls"] = len(comp.Calls)
			body["paykit"] = len(comp.PaykitWins)
			body["total"] = len(comp.Journeys)
		}
		c.JSON(http.StatusOK, body)
	}
}

// funnelVSL is the saved Vidalytics import and the IG association; an
// unreadable store says so instead of reading as "no videos".
func funnelVSL(ctx context.Context, s *vsl.Store) gin.H {
	snaps, err := s.Snapshots(ctx)
	if err != nil {
		return gin.H{"snapshots": []json.RawMessage{}, "igVideo": nil, "error": err.Error()}
	}
	ig, err := s.IGVideo(ctx)
	if err != nil {
		return gin.H{"snapshots": snaps, "igVideo": nil, "error": err.Error()}
	}
	return gin.H{"snapshots": snaps, "igVideo": ig, "error": nil}
}

func itoa(n int) string { return strconv.Itoa(n) }

func intp(n int) *int { return &n }

func nilIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// funnelTrakyoFor builds the Trakyo connector; tests swap it.
var funnelTrakyoFor = func(d *Deps) *trakyo.Connector { return trakyo.New(d.Resolver) }

func funnelAnalytics(d *Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		period := c.DefaultQuery("period", "30d")
		if period != "7d" && period != "30d" && period != "90d" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Choose 7d, 30d, or 90d."})
			return
		}
		ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
		defer cancel()
		a := funnelTrakyoFor(d).Analytics(ctx, period)
		c.Header("Cache-Control", "private, no-store")
		c.JSON(http.StatusOK, gin.H{
			"period": a.Period, "fetchedAt": a.FetchedAt, "content": a.Content, "funnel": a.Funnel,
			"channels": trakyo.GroupChannels(a.Content.Rows),
		})
	}
}

func funnelLeadMessage(d *Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		name := strings.TrimSpace(c.Query("name"))
		if name == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "name is required"})
			return
		}
		var mail *string
		if e := strings.TrimSpace(c.Query("email")); e != "" {
			mail = &e
		}
		feed, ok := funnelFeedWithin(c.Request.Context(), d)
		if !ok {
			c.JSON(http.StatusOK, gin.H{"message": nil, "unavailable": true})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": funnel.LastMessageFor(name, mail, feed), "unavailable": false})
	}
}
