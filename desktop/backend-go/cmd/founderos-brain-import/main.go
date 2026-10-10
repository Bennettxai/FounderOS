// Command founderos-brain-import moves memory into the bridge's Optimal Engines,
// routed by config/founderos/engine-topology.yaml:
//
//	founderos-brain-import -store ~/clue-agent/brain-store        # GBrain pages (1.5/1.6)
//	founderos-brain-import -snapshot ~/.founderos-bridge/backups/macbook-….db  # engine replay (1.4)
//
// Engine endpoints come from each engine's url_env/key_env, falling back to
// the local staging engines (:4210 macbook, :4211 hub, :4212 mini) and their
// keys in ~/.founderos-bridge/keys. Re-runs are no-ops thanks to the ledger.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/rhl/businessos-backend/internal/founderos/brainimport"
	"github.com/rhl/businessos-backend/internal/founderos/topology"
)

var stagingPorts = map[string]int{"macbook": 4210, "hub": 4211, "mini": 4212}

func main() {
	home, _ := os.UserHomeDir()
	bridge := filepath.Join(home, ".founderos-bridge")
	store := flag.String("store", "", "brain-store directory to import")
	snapshot := flag.String("snapshot", "", "engine snapshot .db to replay")
	ledgerPath := flag.String("ledger", "", "ledger path (default ~/.founderos-bridge/ledgers/<store|snapshot>.json)")
	dryRun := flag.Bool("dry-run", false, "plan and count only; send nothing")
	supersede := flag.String("supersede-store", "", "with -snapshot: drop snapshot items that are older copies of pages in this brain-store")
	flag.Parse()
	if (*store == "") == (*snapshot == "") {
		fmt.Fprintln(os.Stderr, "give exactly one of -store or -snapshot")
		os.Exit(2)
	}

	topo, err := topology.LoadRepo()
	must(err)
	must(topo.Validate())

	var items []brainimport.Item
	var skipped int
	mode := "store"
	if *store != "" {
		items, skipped, err = brainimport.Plan(*store, topo)
	} else {
		mode = "snapshot"
		items, skipped, err = brainimport.FromSnapshot(*snapshot, topo)
	}
	must(err)
	superseded := 0
	if *supersede != "" {
		keys, err := brainimport.StoreTitleKeys(*supersede)
		must(err)
		items, superseded = brainimport.WithoutSuperseded(items, keys)
	}
	if *ledgerPath == "" {
		*ledgerPath = filepath.Join(bridge, "ledgers", mode+".json")
	}

	// Only the engines these items target; a stopped engine nobody needs
	// (e.g. the mini tier during a MacBook import) must not fail the run.
	all := endpoints(topo, bridge)
	engines := map[string]brainimport.Endpoint{}
	for _, it := range items {
		if ep, ok := all[it.Engine]; ok {
			engines[it.Engine] = ep
		}
	}
	if !*dryRun {
		must(brainimport.EnsureWorkspaces(topo, engines))
	}
	rep, err := brainimport.Run(items, engines, *ledgerPath, *dryRun)
	must(err)

	plan := map[string]map[string]int{}
	for _, it := range items {
		if plan[it.Engine] == nil {
			plan[it.Engine] = map[string]int{}
		}
		plan[it.Engine][it.Workspace]++
	}
	out := map[string]any{
		"plan_by_target": plan,
		"mode":           mode, "source": *store + *snapshot, "dry_run": *dryRun,
		"items": len(items), "skipped_short_or_empty": skipped, "superseded_by_store": superseded, "report": rep,
		"at": time.Now().UTC().Format(time.RFC3339),
	}
	raw, _ := json.MarshalIndent(out, "", "  ")
	fmt.Println(string(raw))
	if !*dryRun {
		reports := filepath.Join(bridge, "reports")
		_ = os.MkdirAll(reports, 0o700)
		_ = os.WriteFile(filepath.Join(reports, fmt.Sprintf("import-%s-%s.json", mode, time.Now().UTC().Format("20060102T150405Z"))), raw, 0o600)
	}
	if len(rep.Errors) > 0 {
		os.Exit(1)
	}
}

func endpoints(topo *topology.Topology, bridge string) map[string]brainimport.Endpoint {
	eps := map[string]brainimport.Endpoint{}
	for name, e := range topo.Engines {
		if e.Deferred {
			continue
		}
		url := os.Getenv(e.URLEnv)
		key := os.Getenv(e.KeyEnv)
		if url == "" {
			port, ok := stagingPorts[name]
			if !ok {
				continue
			}
			url = fmt.Sprintf("http://127.0.0.1:%d", port)
		}
		if key == "" {
			if raw, err := os.ReadFile(filepath.Join(bridge, "keys", "oe-"+name+".key")); err == nil {
				key = strings.TrimSpace(string(raw))
			}
		}
		eps[name] = brainimport.Endpoint{URL: url, Key: key}
	}
	return eps
}

func must(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "founderos-brain-import:", err)
		os.Exit(1)
	}
}
