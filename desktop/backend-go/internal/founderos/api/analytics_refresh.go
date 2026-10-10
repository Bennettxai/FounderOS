package api

// The analytics heartbeat: FounderOS v1 app/api/analytics/refresh/route.ts.
// The compat listener serves it as POST /api/analytics/refresh, which the
// 15-minute launchd job on the mini curls so metric, social and email-list
// history keeps accruing when nobody opens the app.
//
// Writes go only to the bridge's own Postgres (inbound, allowed while
// FOUNDEROS_WRITES=0). Every outbound call is a read through the guarded
// connectors: IMAP, Slack history, the Notion brand-deals query (an
// allowlisted read POST), Beehiiv stats, Typeform, Stripe balance and the
// engines' package counts. Nothing metered (no Vidalytics stats), no ManyChat.

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/rhl/businessos-backend/internal/founderos/connectors/beehiiv"
	"github.com/rhl/businessos-backend/internal/founderos/connectors/zernio"
	"github.com/rhl/businessos-backend/internal/founderos/pages/analytics"
	"github.com/rhl/businessos-backend/internal/founderos/pages/social"
)

// RefreshResult is the TS route's exact response body.
type RefreshResult struct {
	OK                bool   `json:"ok"`
	Recorded          int    `json:"recorded"`
	Of                int    `json:"of"`
	At                string `json:"at"`
	EmailListRecorded bool   `json:"emailListRecorded"`
}

// ErrRefreshNoPostgres: without the bridge database there is nowhere to
// record, and a sweep that records nothing must not read as a good tick.
var ErrRefreshNoPostgres = errors.New("analytics refresh: founderos Postgres is not wired")

// refreshSources are the sweep's reads. Tests swap refreshSourcesFor.
type refreshSources struct {
	// ZernioConfig is syncFromZernioConfig's source: the on-disk Zernio
	// config (~/.social-media/config.json), a file read, no network.
	ZernioConfig func() zernio.Accounts
	// Warm fills the comms and brand-deals caches the pages read.
	Warm []func(ctx context.Context)
	// Beehiiv is the live subscriber reading for the email-list snapshot.
	Beehiiv func(ctx context.Context) *beehiiv.Reading
	// Reads and Brain are gatherOperatingMetrics' connector tiles.
	Reads func(ctx context.Context, now time.Time) analyticsReads
	Brain func(ctx context.Context) (*float64, string)
}

// refreshStore is the sweep's Postgres surface.
type refreshStore interface {
	InsertSocialSnapshots(ctx context.Context, rows []social.Snapshot) error
	Social(ctx context.Context) ([]analytics.Platform, error)
	EmailPoints(ctx context.Context) ([]analytics.Point, error)
	RunCount(ctx context.Context, now time.Time) (all, last7 int, err error)
	analytics.MetricWriter
	RecordEmailListSnapshot(ctx context.Context, e analytics.EmailListSnapshot) error
	RecordEmailListMiss(ctx context.Context, day, reason string) error
}

var refreshSourcesFor = func(d *Deps) refreshSources {
	mail, sl, deals := depsEmail(d), depsSlack(d), depsBrandDeals(d)
	bh := depsBeehiiv(d)
	zc := zernio.New(d.Resolver)
	return refreshSources{
		ZernioConfig: zc.ConfigAccounts,
		// The same limits the pages ask for (comms: 40 emails, 30 Slack
		// messages), so their reads are cache hits. WhatsApp is not here:
		// on the bridge it arrives by device push, there is nothing to fetch.
		Warm: []func(context.Context){
			func(ctx context.Context) { _, _ = mail.LatestEmails(ctx, 40) },
			func(ctx context.Context) { _, _ = mail.UnreadCounts(ctx) },
			func(ctx context.Context) { _, _ = sl.RecentMessages(ctx, 30) },
			func(ctx context.Context) { _ = deals.FetchDeals(ctx) },
		},
		Beehiiv: bh.Reading,
		Reads:   func(ctx context.Context, now time.Time) analyticsReads { return analyticsConnectorReads(ctx, d, now) },
		Brain: func(ctx context.Context) (*float64, string) {
			bctx, cancel := context.WithTimeout(ctx, 6*time.Second)
			defer cancel()
			return brainTile(bctx, d)
		},
	}
}

// pgRefreshStore joins the analytics store with the social snapshot writer.
type pgRefreshStore struct {
	*analytics.PgStore
	social *social.Store
}

func (s pgRefreshStore) InsertSocialSnapshots(ctx context.Context, rows []social.Snapshot) error {
	return s.social.InsertSnapshots(ctx, rows)
}

// AnalyticsRefresh runs one heartbeat sweep against the bridge Postgres.
func AnalyticsRefresh(ctx context.Context, d *Deps, now func() time.Time) (RefreshResult, error) {
	if d == nil || d.Pool == nil {
		return RefreshResult{}, ErrRefreshNoPostgres
	}
	if now == nil {
		now = time.Now
	}
	ws := funnelWorkspaceIDs(ctx, d.Pool)
	st := pgRefreshStore{
		PgStore: &analytics.PgStore{Pool: d.Pool, Workspaces: ws},
		social:  &social.Store{Pool: d.Pool, WorkspaceID: ws[socialWorkspace]},
	}
	return runRefresh(ctx, refreshSourcesFor(d), st, now), nil
}

// runRefresh is the route body, step for step. No step failing stops the
// sweep: each degrades on its own, as the TS try/catch and allSettled do.
func runRefresh(ctx context.Context, src refreshSources, st refreshStore, now func() time.Time) RefreshResult {
	start := now().UTC()
	today := start.Format("2006-01-02")

	// syncFromZernioConfig: a social sync degrading must not block the sweep.
	var rows []social.LiveAccount
	for _, a := range src.ZernioConfig() {
		rows = append(rows, social.LiveAccount{Platform: a.Platform, Handle: a.Handle, Followers: a.Followers})
	}
	if snaps := social.SyncRows(rows, today, "zernio-config"); len(snaps) > 0 {
		if err := st.InsertSocialSnapshots(ctx, snaps); err != nil {
			slog.Warn("analytics refresh: social snapshots not recorded", "err", err)
		}
	}

	// Warm the comms and brand-deals caches here rather than making a
	// person wait for them (Promise.allSettled).
	var wg sync.WaitGroup
	for _, w := range src.Warm {
		wg.Add(1)
		go func(w func(context.Context)) {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					slog.Error("analytics refresh: warm-up panicked", "panic", r)
				}
			}()
			w(ctx)
		}(w)
	}
	wg.Wait()

	// syncBeehiivEmail: a same-day reading overwrites; a miss is recorded.
	emailListRecorded := false
	snap, miss := analytics.EmailListOutcome(src.Beehiiv(ctx), today)
	if snap != nil {
		if err := st.RecordEmailListSnapshot(ctx, *snap); err != nil {
			slog.Warn("analytics refresh: email-list snapshot not recorded", "err", err)
		} else {
			emailListRecorded = true
		}
	} else if err := st.RecordEmailListMiss(ctx, today, miss); err != nil {
		slog.Warn("analytics refresh: email-list miss not recorded", "err", err)
	}

	// gatherOperatingMetrics, then recordOperatingSnapshots.
	inputs := gatherRefreshInputs(ctx, src, st, start)
	at := now().UTC()
	recorded := analytics.RecordOperatingSnapshots(ctx, st, inputs, at)
	return RefreshResult{OK: true, Recorded: recorded, Of: len(inputs), At: at.Format("2006-01-02T15:04:05.000Z"), EmailListRecorded: emailListRecorded}
}

// gatherRefreshInputs reads the same eight tiles the /analytics page shows.
// A store read that fails leaves its tile an honest nil.
func gatherRefreshInputs(ctx context.Context, src refreshSources, st refreshStore, now time.Time) []analytics.MetricInput {
	var reads analyticsReads
	var brainVal *float64
	var brainSource string
	var wg sync.WaitGroup
	wg.Add(2)
	go func() { defer wg.Done(); reads = src.Reads(ctx, now) }()
	go func() { defer wg.Done(); brainVal, brainSource = src.Brain(ctx) }()

	plats, err := st.Social(ctx)
	if err != nil {
		plats = nil
	}
	emailPts, err := st.EmailPoints(ctx)
	if err != nil {
		emailPts = nil
	}
	allRuns, runs7d, err := st.RunCount(ctx, now)
	if err != nil {
		allRuns, runs7d = 0, 0
	}
	wg.Wait()
	total, _, audience7d := audienceOf(plats, emailPts)
	return operatingInputs(total, audience7d, allRuns, runs7d, reads, brainVal, brainSource)
}
