package tech

import (
	"github.com/rhl/businessos-backend/internal/founderos/agents"
	"github.com/rhl/businessos-backend/internal/founderos/agents/roster"
)

// Agents returns the TECH department's agents in FounderOS v1 roster order.
func Agents(d roster.Deps) []agents.Agent {
	data := &DataAgent{Engines: DefaultEngineSource}
	if d.Memory != nil { // never a typed-nil Searcher
		data.Search = d.Memory
	}
	return []agents.Agent{
		data,
		&MarkdownAuditor{Engines: DefaultEngineSource},
		&VectorAuditor{Engines: DefaultEngineSource},
		&StackMonitor{Devices: d.Devices},
	}
}
