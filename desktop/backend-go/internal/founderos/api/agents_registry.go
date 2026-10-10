package api

import (
	"context"
	"log/slog"
	"time"

	"github.com/rhl/businessos-backend/internal/founderos/agents"
	"github.com/rhl/businessos-backend/internal/founderos/agents/roster"
	"github.com/rhl/businessos-backend/internal/founderos/agents/roster/clients"
	"github.com/rhl/businessos-backend/internal/founderos/agents/roster/comms"
	"github.com/rhl/businessos-backend/internal/founderos/agents/roster/sales"
	"github.com/rhl/businessos-backend/internal/founderos/agents/roster/tech"
	"github.com/rhl/businessos-backend/internal/founderos/memory"
	"github.com/rhl/businessos-backend/internal/founderos/topology"
)

// rosterGroups are the ported departments; each returns its agents.
var rosterGroups = []func(roster.Deps) []agents.Agent{
	clients.Agents,
	comms.Agents,
	tech.Agents,
	sales.Agents,
}

// AllAgents is the bridge's agent roster: one implementation per the operator
// agent row (spec 5.1).
func AllAgents(d *Deps) []agents.Agent {
	rd := RosterDeps(d)
	var out []agents.Agent
	for _, g := range rosterGroups {
		out = append(out, g(rd)...)
	}
	return out
}

// RosterDeps builds what agents may use from the server's Deps.
func RosterDeps(d *Deps) roster.Deps {
	rd := roster.Deps{Pool: d.Pool, Res: d.Resolver, Devices: d.Devices, Robinhood: d.Robinhood, Workspaces: map[string]string{}}
	if d.Pool != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if rows, err := d.Pool.Query(ctx, `SELECT slug, id::text FROM workspaces`); err == nil {
			for rows.Next() {
				var slug, id string
				if rows.Scan(&slug, &id) == nil {
					rd.Workspaces[slug] = id
				}
			}
			rows.Close()
			if err := rows.Err(); err != nil {
				// agents then report their workspace as unknown, never a stale half
				rd.Workspaces = map[string]string{}
				slog.Warn("founderos roster: workspace read failed", "err", err)
			}
		}
	}
	if topo, err := topology.LoadRepo(); err == nil {
		eps := map[string]memory.Endpoint{}
		for _, e := range TopologyEngines() {
			eps[e.Name] = memory.Endpoint{URL: e.URL, Key: e.Key}
		}
		rd.Memory = memory.New(topo, eps)
	}
	return rd
}
