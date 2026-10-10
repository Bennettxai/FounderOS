package api

import (
	"github.com/rhl/businessos-backend/internal/founderos/connectors/beehiiv"
	"sync"

	"github.com/rhl/businessos-backend/internal/founderos/connectors/branddeals"
	"github.com/rhl/businessos-backend/internal/founderos/connectors/email"
	"github.com/rhl/businessos-backend/internal/founderos/connectors/slack"
)

// Constructors, swappable so a test can count builds.
var (
	newEmailConnector = email.New
	newSlackConnector = slack.New
)

var (
	depsConnMu sync.Mutex
	depsEmails = map[*Deps]*email.Connector{}
	depsSlacks = map[*Deps]*slack.Connector{}
	depsDeals  = map[*Deps]*branddeals.Connector{}
)

// depsEmail keeps one email connector per Deps so its IMAP feed and unread
// caches (email.CacheTTL) hold across requests and across pages (FounderOS v1
// caches at module level). Every page that reads the inboxes goes through it.
func depsEmail(d *Deps) *email.Connector {
	depsConnMu.Lock()
	defer depsConnMu.Unlock()
	if c, ok := depsEmails[d]; ok {
		return c
	}
	c := newEmailConnector(d.Resolver)
	depsEmails[d] = c
	return c
}

// depsSlack is depsEmail for Slack's recent-message cache.
func depsSlack(d *Deps) *slack.Connector {
	depsConnMu.Lock()
	defer depsConnMu.Unlock()
	if c, ok := depsSlacks[d]; ok {
		return c
	}
	c := newSlackConnector(d.Resolver)
	depsSlacks[d] = c
	return c
}

// depsBrandDeals is depsEmail for the Notion brand-deals cache, so the
// analytics heartbeat's warm-up is the fetch the /brand-deals page reads.
func depsBrandDeals(d *Deps) *branddeals.Connector {
	depsConnMu.Lock()
	defer depsConnMu.Unlock()
	if c, ok := depsDeals[d]; ok {
		return c
	}
	c := branddeals.New(d.Resolver)
	depsDeals[d] = c
	return c
}

// newBeehiivConnector is swappable so a test can count builds.
var newBeehiivConnector = beehiiv.New

var depsBeehiivs = map[*Deps]*beehiiv.Connector{}

// depsBeehiiv keeps one Beehiiv connector per Deps: its 60s reading cache is
// shared by the heartbeat, the analytics tile and /social (FounderOS v1 shares
// a module-level cache), so one sweep reads Beehiiv once.
func depsBeehiiv(d *Deps) *beehiiv.Connector {
	depsConnMu.Lock()
	defer depsConnMu.Unlock()
	if c, ok := depsBeehiivs[d]; ok {
		return c
	}
	c := newBeehiivConnector(d.Resolver)
	depsBeehiivs[d] = c
	return c
}
