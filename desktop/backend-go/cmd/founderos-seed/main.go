// Command founderos-seed loads the FounderOS demo data (internal/founderos/seed:
// v1's seeded databases, relative dates moved to today) into the founderos_*
// tables. Run founderos-bootstrap first: the ETL never creates workspaces.
// It then imports the demo's knowledge pages (internal/founderos/seed/brain)
// into the Optimal Engine at OPTIMAL_ENGINE_URL, routed by
// config/founderos/engine-topology.yaml.
// Idempotent: a re-run reports zero changes (apart from the day's date shift)
// and ingests zero pages.
//
//	founderos-seed [--database-url postgres://...] [--report-dir .] [--skip-brain]
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rhl/businessos-backend/internal/founderos/brainimport"
	"github.com/rhl/businessos-backend/internal/founderos/connectors"
	"github.com/rhl/businessos-backend/internal/founderos/etl"
	"github.com/rhl/businessos-backend/internal/founderos/seed"
	"github.com/rhl/businessos-backend/internal/founderos/topology"
)

func main() {
	dbURL := flag.String("database-url", os.Getenv("DATABASE_URL"), "target Postgres URL (default $DATABASE_URL, else desktop/backend-go/.env)")
	reportDir := flag.String("report-dir", os.TempDir(), "where to write the JSON + markdown load report")
	skipBrain := flag.Bool("skip-brain", false, "load Postgres only; do not import the knowledge pages into the Optimal Engine")
	flag.Parse()
	if *dbURL == "" {
		*dbURL = backendEnv("DATABASE_URL")
	}
	if *dbURL == "" {
		fail(fmt.Errorf("no --database-url, $DATABASE_URL or DATABASE_URL in desktop/backend-go/.env"))
	}

	src, err := os.MkdirTemp("", "founderos-seed-")
	must(err)
	defer os.RemoveAll(src)
	must(seed.Materialize(src, time.Now()))

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	pool, err := pgxpool.New(ctx, *dbURL)
	must(err)
	defer pool.Close()

	rep, err := etl.Run(ctx, pool, etl.Options{
		From:          src,
		WorkspaceMap:  map[string]string{"personal-brand": "personal"},
		EncryptionKey: os.Getenv("TOKEN_ENCRYPTION_KEY"),
		DatabaseLabel: "founderos demo seed",
	})
	if rep != nil {
		if jp, _, werr := rep.Write(*reportDir, "founderos-seed-report"); werr == nil {
			fmt.Println("report:", jp)
		}
		fmt.Printf("tables: %d, changes: %d\n", len(rep.Tables), rep.Changes())
	}
	must(err)

	if *skipBrain {
		return
	}
	topo, err := topology.LoadRepo()
	must(err)
	must(topo.Validate())
	engines := engineEndpoints(topo)
	if len(engines) == 0 {
		fmt.Println("brain: skipped, no Optimal Engine URL (set OPTIMAL_ENGINE_URL)")
		return
	}
	brep, err := seed.SeedBrain(topo, engines)
	if brep != nil {
		fmt.Printf("brain: %d pages, ingested %d, already in the engine %d, too short %d, errors %d\n",
			brep.Planned, brep.Ingested, brep.Landed, brep.Skipped, len(brep.Errors))
		for _, f := range brep.Errors {
			fmt.Fprintf(os.Stderr, "brain: %s: %s\n", f.Rel, f.Reason)
		}
	}
	must(err)
	if len(brep.Errors) > 0 {
		os.Exit(1)
	}
}

// engineEndpoints resolves each topology engine from its url_env/key_env:
// the process env first, then desktop/backend-go/.env.
func engineEndpoints(topo *topology.Topology) map[string]brainimport.Endpoint {
	eps := map[string]brainimport.Endpoint{}
	for name, e := range topo.Engines {
		if e.Deferred {
			continue
		}
		get := func(key string) string {
			if key == "" {
				return ""
			}
			if v := os.Getenv(key); v != "" {
				return v
			}
			return backendEnv(key)
		}
		if url := get(e.URLEnv); url != "" {
			eps[name] = brainimport.Endpoint{URL: url, Key: get(e.KeyEnv)}
		}
	}
	return eps
}

func backendEnv(key string) string {
	dir, _ := os.Getwd()
	for {
		p := filepath.Join(dir, "desktop", "backend-go", ".env")
		if raw, err := os.ReadFile(p); err == nil {
			return connectors.ParseEnvFile(string(raw))[key]
		}
		if raw, err := os.ReadFile(filepath.Join(dir, ".env")); err == nil {
			if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
				return connectors.ParseEnvFile(string(raw))[key]
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

func must(err error) {
	if err != nil {
		fail(err)
	}
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "founderos-seed:", err)
	os.Exit(1)
}
