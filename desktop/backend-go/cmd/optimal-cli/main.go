// optimal-cli — Go counterpart of the Elixir `mix optimal.*` task suite.
//
// Two modes:
//
//	LOCAL  — calls internal/optimal/* directly against a SQLite index db.
//	         Mirrors the desktop / single-machine deployment path.
//	REMOTE — hits the running BusinessOS API (default http://localhost:8801).
//	         Mirrors `mix optimal.api …` against a deployed cluster.
//
// The mode is per-invocation:
//
//	optimal-cli health                          # local, OPTIMAL_ENGINE_DB or ./optimal.db
//	optimal-cli search "shipping" --limit 5     # local
//	optimal-cli --remote http://prod stats      # remote
//	OPTIMAL_REMOTE_URL=https://… optimal-cli ls # remote, env-driven
//
// Subcommands implemented (covers ~80% of the original mix-task usage):
//
//	health      — engine health diagnostics
//	stats       — row counts, byte usage
//	status      — schema readiness across phases
//	ls          — list nodes
//	search      — text search over signals
//	graph       — top entities + edges summary
//	hubs        — most-connected entities
//	knowledge   — graph context for an entity
//	ingest      — ingest a text body (local only)
package main

import (
	"fmt"
	"os"
)

const usage = `optimal-cli — Go CLI for the OptimalOS engine

Usage:
  optimal-cli [global flags] <command> [command flags] [args]

Global flags:
  --remote URL     Hit the API instead of a local SQLite db. May also be set via OPTIMAL_REMOTE_URL.
  --db PATH        Local SQLite index path. May also be set via OPTIMAL_ENGINE_DB. Default ./optimal.db
  --cookie FILE    Path to a netscape-format cookie jar for authenticated --remote requests.

Commands:
  health         Engine health diagnostics
  stats          Row counts and storage info
  status         Schema readiness for every Phase
  ls             List engine nodes
  search QUERY   Free-text search over signals
  graph          Top entities + edges summary
  hubs           Most-connected entities (use --limit N)
  knowledge ENT  Graph context for an entity (--remote only)
  ingest TEXT    Ingest a text body (local only; pair with --genre)

Run "optimal-cli <command> -h" for command-specific flags.
`

func main() {
	if len(os.Args) < 2 {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(2)
	}

	gopts, args := parseGlobalFlags(os.Args[1:])
	if len(args) == 0 {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(2)
	}

	cmd, rest := args[0], args[1:]
	if cmd == "-h" || cmd == "--help" || cmd == "help" {
		fmt.Print(usage)
		return
	}

	if err := dispatch(cmd, rest, gopts); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
