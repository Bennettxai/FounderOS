package api

import (
	"context"
	"encoding/json"
	"errors"
	"sort"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/rhl/businessos-backend/internal/founderos/connectors"
	"github.com/rhl/businessos-backend/internal/founderos/connectors/beehiiv"
	"github.com/rhl/businessos-backend/internal/founderos/connectors/zernio"
	"github.com/rhl/businessos-backend/internal/founderos/pages/analytics"
	"github.com/rhl/businessos-backend/internal/founderos/pages/pgtest"
	"github.com/rhl/businessos-backend/internal/founderos/pages/social"
)

// fakeRefreshStore is the heartbeat's Postgres surface, in memory.
type fakeRefreshStore struct {
	socialErr error
	socialIn  []social.Snapshot
	platforms []analytics.Platform
	metrics   map[string]float64
	metricAt  time.Time
	emailSnap *analytics.EmailListSnapshot
	misses    map[string]string
}

func newFakeRefreshStore() *fakeRefreshStore {
	return &fakeRefreshStore{metrics: map[string]float64{}, misses: map[string]string{}}
}

func (f *fakeRefreshStore) InsertSocialSnapshots(_ context.Context, rows []social.Snapshot) error {
	f.socialIn = rows
	return f.socialErr
}
func (f *fakeRefreshStore) Social(context.Context) ([]analytics.Platform, error) {
	return f.platforms, nil
}
func (f *fakeRefreshStore) EmailPoints(context.Context) ([]analytics.Point, error) {
	return []analytics.Point{}, nil
}
func (f *fakeRefreshStore) RunCount(context.Context, time.Time) (int, int, error) { return 5, 2, nil }
func (f *fakeRefreshStore) RecordMetric(_ context.Context, id string, v float64, at time.Time) error {
	f.metrics[id], f.metricAt = v, at
	return nil
}
func (f *fakeRefreshStore) RecordEmailListSnapshot(_ context.Context, e analytics.EmailListSnapshot) error {
	f.emailSnap = &e
	return nil
}
func (f *fakeRefreshStore) RecordEmailListMiss(_ context.Context, day, reason string) error {
	f.misses[day] = reason
	return nil
}

func f64(v float64) *float64 { return &v }

func fakeRefreshSources(warmed *atomic.Int32, reading *beehiiv.Reading) refreshSources {
	warm := func(context.Context) { warmed.Add(1) }
	return refreshSources{
		ZernioConfig: func() zernio.Accounts {
			return zernio.Accounts{
				{Platform: "instagram", Handle: "@b", Followers: f64(1000)},
				{Platform: "myspace", Handle: "@b", Followers: f64(9)}, // untracked: skipped
				{Platform: "tiktok", Handle: "@b"},                     // no count: skipped
			}
		},
		Warm:    []func(context.Context){warm, warm, warm, warm},
		Beehiiv: func(context.Context) *beehiiv.Reading { return reading },
		Reads: func(context.Context, time.Time) analyticsReads {
			return analyticsReads{Leads30d: f64(12)}
		},
		Brain: func(context.Context) (*float64, string) { return f64(1423), "Optimal Engine · 2 engines" },
	}
}

var refreshNow = func() time.Time { return time.Date(2026, 9, 30, 9, 15, 0, 0, time.UTC) }

// One sweep, as app/api/analytics/refresh/route.ts: the Zernio config sync
// failing does not stop it, every warmer runs, a fresh Beehiiv reading is
// today's snapshot, and only the metrics that have a value are recorded.
func TestRefreshSweepMatchesTheTS(t *testing.T) {
	st := newFakeRefreshStore()
	st.socialErr = errors.New("social write failed")
	st.platforms = []analytics.Platform{{Platform: "instagram", Points: []analytics.Point{{CapturedAt: "2026-09-30", Value: 1000}}}}
	var warmed atomic.Int32
	src := fakeRefreshSources(&warmed, &beehiiv.Reading{Subscribers: 20315, PublicationID: "pub_1", Metric: "active_subscriptions", Fresh: true})

	res := runRefresh(context.Background(), src, st, refreshNow)

	if len(st.socialIn) != 1 || st.socialIn[0].Platform != "instagram" || st.socialIn[0].Source != "zernio-config" || st.socialIn[0].CapturedAt != "2026-09-30" {
		t.Fatalf("social rows = %+v", st.socialIn)
	}
	if warmed.Load() != 4 {
		t.Fatalf("warmed %d of 4", warmed.Load())
	}
	if !res.EmailListRecorded || st.emailSnap == nil || st.emailSnap.Subscribers != 20315 || st.emailSnap.CapturedOn != "2026-09-30" {
		t.Fatalf("email list: %v %+v", res.EmailListRecorded, st.emailSnap)
	}
	// Of the eight tiles, audience, leads, agent-runs and brain have values;
	// subscribers, stripe, unread and dictations are honest-pending nulls.
	got := []string{}
	for id := range st.metrics {
		got = append(got, id)
	}
	sort.Strings(got)
	if res.Of != 8 || res.Recorded != 4 || strings.Join(got, ",") != "agent-runs,audience,brain,leads" {
		t.Fatalf("recorded %d of %d: %v", res.Recorded, res.Of, st.metrics)
	}
	if st.metrics["audience"] != 1000 || st.metrics["agent-runs"] != 5 || st.metrics["brain"] != 1423 {
		t.Fatalf("values = %v", st.metrics)
	}
	if !res.OK || res.At != "2026-09-30T09:15:00.000Z" || !st.metricAt.Equal(refreshNow()) {
		t.Fatalf("result = %+v, metricAt %v", res, st.metricAt)
	}
}

// A day with no fresh reading is recorded as a miss, not an observation.
func TestRefreshRecordsABeehiivMiss(t *testing.T) {
	for _, r := range []*beehiiv.Reading{nil, {Subscribers: 18266, Fresh: false}} {
		st := newFakeRefreshStore()
		var warmed atomic.Int32
		res := runRefresh(context.Background(), fakeRefreshSources(&warmed, r), st, refreshNow)
		if res.EmailListRecorded || st.emailSnap != nil || st.misses["2026-09-30"] == "" {
			t.Fatalf("reading %+v: recorded=%v snap=%+v misses=%v", r, res.EmailListRecorded, st.emailSnap, st.misses)
		}
	}
}

// The launchd caller's contract: exactly the TS response keys.
func TestRefreshResultHasTheTSShape(t *testing.T) {
	b, _ := json.Marshal(RefreshResult{OK: true, Recorded: 4, Of: 8, At: "2026-09-30T09:15:00.000Z", EmailListRecorded: true})
	var m map[string]any
	_ = json.Unmarshal(b, &m)
	keys := []string{}
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	if strings.Join(keys, ",") != "at,emailListRecorded,of,ok,recorded" {
		t.Fatalf("keys = %v", keys)
	}
}

// The warm-up warms the shared per-Deps connectors, so /comms, the console
// and /brand-deals read what the heartbeat already fetched.
func TestRefreshWarmsTheSharedConnectors(t *testing.T) {
	emails, slacks := countConnectorBuilds(t)
	t.Setenv("NOTION_API_KEY", "")
	d := &Deps{Resolver: connectors.Resolver{EnvLocal: t.TempDir() + "/env.local"}}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	src := refreshSourcesFor(d)
	if len(src.Warm) != 4 {
		t.Fatalf("warmers = %d, want email latest, email unread, Slack recent, brand deals", len(src.Warm))
	}
	for _, w := range src.Warm {
		w(ctx)
	}
	commsLiveSources(d)
	consoleFeed(ctx, d)
	if *emails != 1 || *slacks != 1 {
		t.Fatalf("connectors built emails=%d slacks=%d, want the one shared pair", *emails, *slacks)
	}
	if brandDealsSource(d) != depsBrandDeals(d) {
		t.Fatal("the brand-deals page must read the connector the heartbeat warms")
	}
}

func TestAnalyticsRefreshWithoutPostgresIsAnError(t *testing.T) {
	if _, err := AnalyticsRefresh(context.Background(), &Deps{}, refreshNow); !errors.Is(err, ErrRefreshNoPostgres) {
		t.Fatalf("err = %v", err)
	}
}

// End to end over a throwaway bridge Postgres: the sweep writes the social,
// email-list and metric rows into the right workspaces.
func TestAnalyticsRefreshWritesPostgres(t *testing.T) {
	pool := pgtest.ThrowawayDB(t)
	ctx := context.Background()
	for _, slug := range []string{"founderos", "personal"} {
		if _, err := pool.Exec(ctx, `INSERT INTO workspaces (name, slug, owner_id) VALUES ($1,$1,'u1')`, slug); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := pool.Exec(ctx, `INSERT INTO founderos_social_accounts (platform, workspace_id, handle, ord)
		SELECT 'instagram', id, '@b', 1 FROM workspaces WHERE slug = 'personal'`); err != nil {
		t.Fatal(err)
	}
	var warmed atomic.Int32
	prev := refreshSourcesFor
	refreshSourcesFor = func(*Deps) refreshSources {
		return fakeRefreshSources(&warmed, &beehiiv.Reading{Subscribers: 20315, Fresh: true})
	}
	t.Cleanup(func() { refreshSourcesFor = prev })

	for i := 0; i < 2; i++ { // a second tick the same day is idempotent for the daily rows
		res, err := AnalyticsRefresh(ctx, &Deps{Pool: pool}, refreshNow)
		if err != nil {
			t.Fatal(err)
		}
		// agent-runs is null here: the throwaway DB has no runs.
		if !res.OK || !res.EmailListRecorded || res.Recorded != 3 || res.Of != 8 {
			t.Fatalf("tick %d: %+v", i, res)
		}
	}
	var socialRows, emailRows, metricRows int
	var followers int
	if err := pool.QueryRow(ctx, `SELECT (SELECT count(*) FROM founderos_social_snapshots), (SELECT max(followers) FROM founderos_social_snapshots),
		(SELECT count(*) FROM founderos_email_list_snapshots), (SELECT count(*) FROM founderos_metric_snapshots)`).Scan(&socialRows, &followers, &emailRows, &metricRows); err != nil {
		t.Fatal(err)
	}
	// Same instant both ticks (fixed clock), so metrics overwrite too.
	if socialRows != 1 || followers != 1000 || emailRows != 1 || metricRows != 3 {
		t.Fatalf("rows: social=%d (%d) email=%d metric=%d", socialRows, followers, emailRows, metricRows)
	}
}
