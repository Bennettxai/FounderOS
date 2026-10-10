// Command founderos-bootstrap creates the operator's workspaces in the bridge's
// BusinessOS database, each routed to its home staging engine (spec 2.2).
//
//	founderos-bootstrap -owner-email you@example.com
//
// DATABASE_URL defaults to the one in desktop/backend-go/.env. Engine URLs
// and keys come from each engine's url_env/key_env, else the local staging
// engines and ~/.founderos-bridge/keys. Safe to re-run.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rhl/businessos-backend/internal/founderos/bootstrap"
	"github.com/rhl/businessos-backend/internal/founderos/connectors"
	"github.com/rhl/businessos-backend/internal/founderos/topology"
)

func main() {
	owner := flag.String("owner-email", os.Getenv("FOUNDEROS_OWNER_EMAIL"), "email of the BusinessOS user who owns the workspaces")
	dsn := flag.String("db", "", "Postgres URL (default: DATABASE_URL from desktop/backend-go/.env)")
	flag.Parse()
	if *owner == "" {
		fail(fmt.Errorf("-owner-email is required"))
	}
	topo, err := topology.LoadRepo()
	must(err)
	must(topo.Validate())
	if *dsn == "" {
		*dsn = backendEnv("DATABASE_URL")
	}
	if *dsn == "" {
		fail(fmt.Errorf("no -db and no DATABASE_URL in desktop/backend-go/.env"))
	}

	// Each engine's URL and key come from the environment, else from the
	// backend's .env (where dev-local.sh writes OPTIMAL_ENGINE_URL).
	engines := map[string]bootstrap.Engine{}
	for name, e := range topo.Engines {
		url, key := os.Getenv(e.URLEnv), os.Getenv(e.KeyEnv)
		if url == "" {
			url = backendEnv(e.URLEnv)
		}
		if key == "" {
			key = backendEnv(e.KeyEnv)
		}
		if url != "" && !e.Deferred {
			engines[name] = bootstrap.Engine{URL: url, Key: key}
		}
	}
	specs, err := bootstrap.Plan(topo, engines)
	must(err)

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, *dsn)
	must(err)
	defer pool.Close()
	res, err := bootstrap.Apply(ctx, pool, *owner, specs)
	must(err)
	var slugs []string
	for _, s := range specs {
		slugs = append(slugs, s.Slug+"→"+s.Engine.BaseURL)
	}
	out, _ := json.MarshalIndent(map[string]any{"result": res, "workspaces": slugs}, "", "  ")
	fmt.Println(string(out))
}

func backendEnv(key string) string {
	dir, _ := os.Getwd()
	for {
		p := filepath.Join(dir, "desktop", "backend-go", ".env")
		if _, err := os.Stat(p); err == nil {
			raw, _ := os.ReadFile(p)
			return connectors.ParseEnvFile(string(raw))[key]
		}
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			raw, _ := os.ReadFile(filepath.Join(dir, ".env"))
			if v := connectors.ParseEnvFile(string(raw))[key]; v != "" {
				return v
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
	fmt.Fprintln(os.Stderr, "founderos-bootstrap:", err)
	os.Exit(1)
}
