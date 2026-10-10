// Package metaads ports FounderOS v1's lib/connectors/meta-ads.ts: paid-funnel
// attribution (which ads produced the touches that became opt-ins and
// purchases). It is deliberately status-only: live wiring arrives with the
// Meta Ads MCP, so until then Status reports whether META_ADS_ACCESS_TOKEN is
// present and makes no network call. There is no error state to reach.
package metaads

import (
	"context"

	"github.com/rhl/businessos-backend/internal/founderos/connectors"
)

var Meta = connectors.Meta{ID: "meta-ads", Name: "Meta Ads", Kind: connectors.KindAds}

const keyName = "META_ADS_ACCESS_TOKEN"

type Connector struct {
	res connectors.Resolver
	// Files are the fallback credential files (clue-agent, then social-media).
	Files []string
}

func New(res connectors.Resolver) *Connector {
	social, clue, _, _ := connectors.CredFiles()
	return &Connector{res: res, Files: []string{clue, social}}
}

func (c *Connector) Status(context.Context) connectors.Status {
	st := connectors.Status{ID: Meta.ID, Name: Meta.Name, Kind: Meta.Kind}
	if c.res.Resolve(keyName, c.Files...) == "" {
		st.State = connectors.StateNotConfigured
		st.Detail = "Paid-funnel attribution (ad → opt-in → purchase) for Vantage + Launchpad Cohort. Set META_ADS_ACCESS_TOKEN to wire the Meta Ads MCP."
		return st
	}
	st.State = connectors.StateConnected
	st.Detail = "META_ADS_ACCESS_TOKEN present · ad-touch attribution ready (live pull lands with the Meta Ads MCP wiring)."
	st.Meta = map[string]any{"keyed": "yes"}
	return st
}
