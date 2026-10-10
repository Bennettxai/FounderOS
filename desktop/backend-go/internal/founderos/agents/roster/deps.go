// Package roster holds the FounderOS agent implementations (spec §5), one
// subpackage per department. Each subpackage exposes
//
//	func Agents(d roster.Deps) []agents.Agent
//
// and the API layer registers them all (api.AllAgents).
package roster

import (
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rhl/businessos-backend/internal/founderos/connectors"
	"github.com/rhl/businessos-backend/internal/founderos/connectors/devicepush"
	"github.com/rhl/businessos-backend/internal/founderos/connectors/robinhood"
	"github.com/rhl/businessos-backend/internal/founderos/memory"
)

// Deps is what agents may use. Connectors are constructed from Res on
// demand (connectors.New(d.Res)); memory goes through Memory only (never
// GBrain); the operator's relational data lives in the founderos_* tables, scoped by
// Workspaces (slug → BusinessOS workspace UUID).
type Deps struct {
	Pool       *pgxpool.Pool
	Res        connectors.Resolver
	Workspaces map[string]string
	Memory     *memory.Router
	Devices    *devicepush.Receiver
	Robinhood  robinhood.Store
}
