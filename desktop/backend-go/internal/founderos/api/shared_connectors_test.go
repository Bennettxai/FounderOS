package api

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/rhl/businessos-backend/internal/founderos/connectors"
	"github.com/rhl/businessos-backend/internal/founderos/connectors/beehiiv"
	"github.com/rhl/businessos-backend/internal/founderos/connectors/email"
	"github.com/rhl/businessos-backend/internal/founderos/connectors/slack"
)

// countConnectorBuilds swaps the email/Slack constructors for counting ones.
func countConnectorBuilds(t *testing.T) (emails, slacks *int) {
	t.Helper()
	// No real inbox or Slack token may leak in from the test process env.
	t.Setenv("SLACK_BOT_TOKEN", "")
	for n := 1; n <= email.MaxInboxes; n++ {
		t.Setenv(fmt.Sprintf("INBOX_%d_HOST", n), "")
	}
	var e, s int
	prevE, prevS := newEmailConnector, newSlackConnector
	newEmailConnector = func(res connectors.Resolver) *email.Connector { e++; return prevE(res) }
	newSlackConnector = func(res connectors.Resolver) *slack.Connector { s++; return prevS(res) }
	t.Cleanup(func() { newEmailConnector, newSlackConnector = prevE, prevS })
	return &e, &s
}

// Every route that reads email or Slack shares one connector per Deps, so the
// 20-minute IMAP and Slack caches survive across requests and across pages.
func TestConnectorsBuiltOncePerDeps(t *testing.T) {
	emails, slacks := countConnectorBuilds(t)
	d := &Deps{Resolver: connectors.Resolver{EnvLocal: t.TempDir() + "/env.local"}}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	for i := 0; i < 2; i++ {
		funnelCommsFeed(ctx, d, 5)                  // funnel lead-message lookup
		consoleFeed(ctx, d)                         // console feed
		analyticsConnectorReads(ctx, d, time.Now()) // analytics unread tile
		_, _ = metricReaders(d).Unread(ctx)         // metrics unread tile
		commsLiveSources(d)                         // comms page sources
	}
	if *emails != 1 || *slacks != 1 {
		t.Fatalf("connectors built emails=%d slacks=%d, want 1 each for one Deps", *emails, *slacks)
	}
	if depsEmail(d) != depsEmail(d) || depsSlack(d) != depsSlack(d) {
		t.Fatal("same Deps must return the same connector pointer")
	}

	other := &Deps{Resolver: d.Resolver}
	if depsEmail(other) == depsEmail(d) || depsSlack(other) == depsSlack(d) {
		t.Fatal("a different Deps must get its own connectors")
	}
}

// One Beehiiv connector per Deps: its 60s reading cache is shared by the
// heartbeat's email-list snapshot, the analytics subscribers tile and /social,
// so a sweep reads Beehiiv once (FounderOS v1 shares a module-level cache).
func TestBeehiivBuiltOncePerDeps(t *testing.T) {
	t.Setenv("BEEHIIV_API_KEY", "")
	t.Setenv("BEEHIIV_PUBLICATION_ID", "")
	n := 0
	prev := newBeehiivConnector
	newBeehiivConnector = func(res connectors.Resolver) *beehiiv.Connector { n++; return prev(res) }
	t.Cleanup(func() { newBeehiivConnector = prev })
	d := &Deps{Resolver: connectors.Resolver{EnvLocal: t.TempDir() + "/env.local"}}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	for i := 0; i < 2; i++ {
		analyticsConnectorReads(ctx, d, time.Now())
		_ = refreshSourcesFor(d)
		_ = socialNewSources(d)
	}
	if n != 1 {
		t.Fatalf("beehiiv connectors built: %d, want 1", n)
	}
}
