package api

// /analytics (spec 6.7): FounderOS v1 app/analytics/page.tsx.
//
//	GET /pages/analytics   run log, reach, operating metrics, audience
//
// /api/analytics/refresh (the snapshot cron) is the compat listener's.

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/rhl/businessos-backend/internal/founderos/connectors"
	"github.com/rhl/businessos-backend/internal/founderos/connectors/devicepush"
	"github.com/rhl/businessos-backend/internal/founderos/connectors/email"
	"github.com/rhl/businessos-backend/internal/founderos/connectors/stripe"
	"github.com/rhl/businessos-backend/internal/founderos/connectors/typeform"
	"github.com/rhl/businessos-backend/internal/founderos/pages/analytics"
)

func init() { RegisterPage(registerAnalytics) }

func registerAnalytics(s *gin.RouterGroup, d *Deps) {
	s.GET("/pages/analytics", analyticsPage(d))
}

var platformLabels = map[string]string{"instagram": "Instagram", "tiktok": "TikTok", "twitter": "X", "youtube": "YouTube", "linkedin": "LinkedIn"}

// analyticsConnectorReads are the live connector tiles (gatherOperatingMetrics):
// each is nil when its source did not answer. Tests swap it.
type analyticsReads struct {
	Subs        *int64
	Leads30d    *float64
	StripeUSD   *float64
	Unread      *float64
	UnreadLabel string
	// Dictations is the Wispr lifetime total from the device push.
	Dictations *float64
}

var analyticsConnectorReads = func(ctx context.Context, d *Deps, now time.Time) analyticsReads {
	res := d.Resolver
	var out analyticsReads
	out.Dictations = wisprDictations(ctx, d.Devices)
	var wg sync.WaitGroup
	wg.Add(4)
	go func() {
		defer wg.Done()
		if n, ok := depsBeehiiv(d).Subscribers(ctx); ok {
			v := int64(n)
			out.Subs = &v
		}
	}()
	go func() {
		defer wg.Done()
		// Attio's open-deal count was this tile until Attio was retired;
		// leads now come from Typeform: 30 days of submissions.
		if leads, err := typeform.New(res).Leads(ctx, typeform.LeadOptions{Now: now, Days: 30}); err == nil {
			v := float64(len(leads))
			out.Leads30d = &v
		}
	}()
	go func() {
		defer wg.Done()
		c := stripe.New(res)
		if !c.Configured() {
			return
		}
		if snap, err := c.Snapshot(ctx); err == nil {
			var cents int64
			if len(snap.Available) > 0 {
				cents = snap.Available[0].Amount
			}
			v := math.Round(float64(cents) / 100)
			out.StripeUSD = &v
		}
	}()
	go func() {
		defer wg.Done()
		counts, err := depsEmail(d).UnreadCounts(ctx)
		if err != nil {
			return
		}
		out.Unread, out.UnreadLabel = unreadTile(counts)
	}()
	wg.Wait()
	return out
}

// wisprDictations is prod's wispr.meta.dictations: the lifetime total from the
// Wispr row the device push answers with (the host's fresh push first, see
// devicepush.Receiver). nil unless that row is connected and carries a count:
// unknown stays unknown, never 0.
func wisprDictations(ctx context.Context, devices *devicepush.Receiver) *float64 {
	if devices == nil {
		return nil
	}
	st := devices.Connector(devicepush.SourceWispr).Status(ctx)
	if st.State != connectors.StateConnected {
		return nil
	}
	switch n := st.Meta["dictations"].(type) {
	case int:
		return fptr(float64(n))
	case int64:
		return fptr(float64(n))
	case float64:
		return fptr(n)
	}
	return nil
}

type analyticsPlatform struct {
	Platform  string   `json:"platform"`
	Label     string   `json:"label"`
	Handle    string   `json:"handle"`
	URL       *string  `json:"url"`
	Followers *int64   `json:"followers"`
	Share     *float64 `json:"share"`
	D7        *float64 `json:"d7"`
	// Bars are the last 12 real follower snapshots; nil under two, so the
	// row never draws an invented trend.
	Bars []float64 `json:"bars"`
}

type analyticsTile struct {
	analytics.MetricTile
	Spark     []float64 `json:"spark"`
	SparkReal bool      `json:"sparkReal"`
}

func fptr(v float64) *float64 { return &v }

func analyticsPage(d *Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 20*time.Second)
		defer cancel()
		now := time.Now().UTC()
		today := now.Format("2006-01-02")
		ws := funnelWorkspaceIDs(ctx, d.Pool)
		store := &analytics.PgStore{Pool: d.Pool, Workspaces: ws}
		errs := map[string]string{}

		var reads analyticsReads
		var side sync.WaitGroup
		side.Add(1)
		go func() {
			defer side.Done()
			reads = analyticsConnectorReads(ctx, d, now)
		}()

		// The 30-day window every run card charts (+1 spare day).
		runs, err := store.Runs(ctx, now.Add(-31*24*time.Hour))
		runsKnown := err == nil
		if err != nil {
			errs["runs"] = err.Error()
			runs = []analytics.Run{}
		}
		allRuns, runs7d, err := store.RunCount(ctx, now)
		if err != nil {
			errs["runCount"] = err.Error()
		}
		names, err := store.AgentNames(ctx)
		if err != nil {
			names = map[string]string{}
		}

		social, err := store.Social(ctx)
		socialKnown := err == nil
		if err != nil {
			errs["social"] = err.Error()
			social = []analytics.Platform{}
		}
		emailPts, err := store.EmailPoints(ctx)
		if err != nil {
			errs["emailList"] = err.Error()
			emailPts = []analytics.Point{}
		}
		total, latestAt, audience7d := audienceOf(social, emailPts)

		platforms := []analyticsPlatform{}
		volChannels := []analytics.Channel{}
		for _, p := range social {
			row := analyticsPlatform{Platform: p.Platform, Label: platformLabels[p.Platform], Handle: p.Handle, URL: p.URL, D7: analytics.GrowthOver(p.Points, 7)}
			if n := len(p.Points); n > 0 {
				f := int64(p.Points[n-1].Value)
				row.Followers = &f
				if total > 0 {
					row.Share = fptr(float64(f) / float64(total) * 100)
				}
			}
			if n := len(p.Points); n >= 2 {
				from := max(0, n-12)
				for _, pt := range p.Points[from:] {
					row.Bars = append(row.Bars, pt.Value)
				}
			}
			platforms = append(platforms, row)
			volChannels = append(volChannels, analytics.Channel{Key: p.Platform, Label: row.Label, Followers: row.Followers})
		}
		side.Wait()

		bctx, bcancel := context.WithTimeout(c.Request.Context(), 6*time.Second)
		brainVal, brainSource := brainTile(bctx, d)
		bcancel()
		inputs := operatingInputs(total, audience7d, allRuns, runs7d, reads, brainVal, brainSource)
		live, pending := analytics.SplitMetrics(inputs)
		tiles := make([]analyticsTile, 0, len(live))
		for _, t := range live {
			hist, err := store.MetricHistory(ctx, t.ID, 7, today)
			if err != nil {
				hist = nil
			}
			tiles = append(tiles, analyticsTile{MetricTile: t, Spark: analytics.SparkSeries(hist, t.ID, t.Value), SparkReal: len(hist) >= 2})
		}
		pendingRows := make([]analytics.Pending, 0, len(pending))
		for _, p := range pending {
			pendingRows = append(pendingRows, analytics.Pending{Label: p.Label, Source: p.Source})
		}
		vol := analytics.VolumeOf(analytics.VolumeInput{
			Channels: volChannels, Subs: reads.Subs, Growth7d: audience7d, Live: len(live), Pending: pendingRows,
			Runs: runs, AgentNames: names, Today: today,
		})

		body := gin.H{
			"today":     today,
			"runVolume": analytics.RunVolume(runs, today, 30),
			"runs30d":   analytics.RunsWithin(runs, today, 30),
			"runsKnown": runsKnown,
			"volume":    vol,
			"live":      tiles,
			"pending":   pending,
			"audience": gin.H{
				"known":          socialKnown,
				"totalFollowers": total,
				"asOf":           nilIfEmpty(latestAt),
				"growth7d":       audience7d,
				"platforms":      platforms,
			},
			"errors": errs,
		}
		c.JSON(http.StatusOK, body)
	}
}

// unreadTile sums the inboxes that answered and says how many that was: a
// sum over 3 of 4 inboxes is labelled so, never passed off as all of them.
// No inbox answering is unknown (nil), not 0.
func unreadTile(counts []email.InboxUnread) (*float64, string) {
	sum, known := 0, 0
	for _, c := range counts {
		if c.Unread != nil {
			sum += *c.Unread
			known++
		}
	}
	if known == 0 {
		return nil, unreadAll
	}
	v := float64(sum)
	if known < len(counts) {
		return &v, fmt.Sprintf("Unread · %d of %d inboxes", known, len(counts))
	}
	return &v, unreadAll
}

const unreadAll = "Unread · all inboxes"

func unreadLabelOr(l string) string {
	if l == "" {
		return unreadAll
	}
	return l
}

// audienceOf is the audience strip's figures: total followers across the
// latest per-platform snapshots, the newest capture date, and the 7-day
// growth over every channel plus the email list.
func audienceOf(social []analytics.Platform, emailPts []analytics.Point) (total int64, latestAt string, growth7d *float64) {
	channelPts := [][]analytics.Point{}
	for _, p := range social {
		if n := len(p.Points); n > 0 {
			total += int64(p.Points[n-1].Value)
			if p.Points[n-1].CapturedAt > latestAt {
				latestAt = p.Points[n-1].CapturedAt
			}
		}
		channelPts = append(channelPts, p.Points)
	}
	channelPts = append(channelPts, emailPts)
	return total, latestAt, analytics.AudienceGrowthPct(channelPts, 7)
}

// brainTile is the memory-pages read with its caption; nil when the engines
// did not answer, never a zero. A live tile names one source under its spark,
// as prod's Brain-store Pages tile names "GBrain"; the failure reason stays on
// the pending chip.
func brainTile(ctx context.Context, d *Deps) (*float64, string) {
	r, err := readBrainPages(ctx, d)
	if err != nil {
		return nil, "Optimal Engine · " + err.Error()
	}
	return r.Value, "Optimal Engine"
}

// operatingInputs is gatherOperatingMetrics' tile list, shared by the page
// render and the /api/analytics/refresh heartbeat so both see the same eight
// metrics. Each value is a real read or an honest pending nil.
func operatingInputs(total int64, audience7d *float64, allRuns, runs7d int, reads analyticsReads, brainVal *float64, brainSource string) []analytics.MetricInput {
	audienceDelta := 0.0
	if audience7d != nil {
		audienceDelta = math.Round(*audience7d*10) / 10
	}
	var audienceVal, runsVal, subsVal *float64
	if total > 0 {
		audienceVal = fptr(float64(total))
	}
	if allRuns > 0 {
		runsVal = fptr(float64(allRuns))
	}
	if reads.Subs != nil {
		subsVal = fptr(float64(*reads.Subs))
	}
	return []analytics.MetricInput{
		{ID: "audience", Label: "Audience", Unit: "followers", Source: "7d · Zernio", Value: audienceVal, Delta: audienceDelta, DeltaPct: audience7d != nil},
		{ID: "subscribers", Label: "Subscribers", Unit: "subs", Source: "Beehiiv", Value: subsVal},
		{ID: "leads", Label: "Leads · 30d", Unit: "leads", Source: "Typeform", Value: reads.Leads30d},
		{ID: "stripe", Label: "Stripe Available", Unit: "usd", Source: "Stripe", Value: reads.StripeUSD},
		{ID: "agent-runs", Label: "Agent Runs", Unit: "runs", Source: "all time", Value: runsVal, Delta: float64(runs7d)},
		{ID: "unread", Label: unreadLabelOr(reads.UnreadLabel), Unit: "emails", Source: "Email", Value: reads.Unread},
		// v1's Brain-store Pages: GBrain is retired, so the count is the
		// engines' source packages.
		{ID: "brain", Label: "Brain Pages", Unit: "pages", Source: brainSource, Value: brainVal},
		{ID: "dictations", Label: "Dictations", Unit: "dictations", Source: "Wispr Flow", Value: reads.Dictations},
	}
}
