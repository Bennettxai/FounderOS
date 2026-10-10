package main

import (
	"flag"
	"fmt"
	"strings"
)

// Each command has a single function so the dispatch table can do a clean
// map lookup. Per-command flags are parsed inside the function via a fresh
// FlagSet — that pattern matches mix tasks (--limit, --genre, etc. are
// task-local).

// cmdHealth shows engine diagnostics. Local mode reads the SQLite index;
// remote mode hits /api/optimal/health.
func cmdHealth(args []string, opts globalOpts) error {
	fs := flag.NewFlagSet("health", flag.ExitOnError)
	_ = fs.Parse(args)

	if opts.isRemote() {
		var resp map[string]any
		if err := remoteGet(opts, "/api/optimal/health", &resp); err != nil {
			return fmt.Errorf("health: %w", err)
		}
		printPretty(resp)
		return nil
	}
	return errElixirRequired("health")
}

// cmdStats prints row counts + storage. Remote-only — requires aggregated
// queries the local helpers don't expose.
func cmdStats(args []string, opts globalOpts) error {
	fs := flag.NewFlagSet("stats", flag.ExitOnError)
	_ = fs.Parse(args)

	if opts.isRemote() {
		var resp map[string]any
		if err := remoteGet(opts, "/api/optimal-cloud/stats", &resp); err != nil {
			return fmt.Errorf("stats: %w", err)
		}
		printPretty(resp)
		return nil
	}
	return errElixirRequired("stats")
}

// cmdStatus reports which Phase schemas are present + reachable. Remote
// only — uses /health/detailed which the existing server already serves.
func cmdStatus(args []string, opts globalOpts) error {
	fs := flag.NewFlagSet("status", flag.ExitOnError)
	_ = fs.Parse(args)

	if !opts.isRemote() {
		fmt.Println("status: this command only runs in --remote mode")
		fmt.Println("        try: optimal-cli --remote http://localhost:8801 status")
		return nil
	}
	var resp map[string]any
	if err := remoteGet(opts, "/health/detailed", &resp); err != nil {
		return fmt.Errorf("status: %w", err)
	}
	printPretty(resp)
	return nil
}

// cmdLs lists engine nodes. Local: reads the filesystem under the same
// nodes_root the server uses. Remote: hits /api/optimal/nodes.
func cmdLs(args []string, opts globalOpts) error {
	fs := flag.NewFlagSet("ls", flag.ExitOnError)
	_ = fs.Parse(args)

	if opts.isRemote() {
		var resp struct {
			Nodes []any `json:"nodes"`
			Count int   `json:"count"`
		}
		if err := remoteGet(opts, "/api/optimal/nodes", &resp); err != nil {
			return fmt.Errorf("ls: %w", err)
		}
		fmt.Printf("%d nodes\n", resp.Count)
		printPretty(resp.Nodes)
		return nil
	}
	return errElixirRequired("ls")
}

// cmdSearch runs free-text search.
func cmdSearch(args []string, opts globalOpts) error {
	fs := flag.NewFlagSet("search", flag.ExitOnError)
	limit := fs.Int("limit", 10, "max results")
	_ = fs.Parse(args)
	if fs.NArg() == 0 {
		return fmt.Errorf("search: QUERY argument required")
	}
	query := strings.Join(fs.Args(), " ")

	if opts.isRemote() {
		body := map[string]any{"query": query, "limit": *limit}
		var resp map[string]any
		if err := remotePost(opts, "/api/optimal/search", body, &resp); err != nil {
			return fmt.Errorf("search: %w", err)
		}
		printPretty(resp)
		return nil
	}
	return errElixirRequired("search")
}

// cmdGraph dumps a graph summary.
func cmdGraph(args []string, opts globalOpts) error {
	fs := flag.NewFlagSet("graph", flag.ExitOnError)
	_ = fs.Parse(args)

	if opts.isRemote() {
		var resp map[string]any
		if err := remoteGet(opts, "/api/optimal/graph", &resp); err != nil {
			return fmt.Errorf("graph: %w", err)
		}
		printPretty(resp)
		return nil
	}
	return errElixirRequired("graph")
}

// cmdHubs shows the most-connected entities.
func cmdHubs(args []string, opts globalOpts) error {
	fs := flag.NewFlagSet("hubs", flag.ExitOnError)
	limit := fs.Int("limit", 5, "max hubs to print")
	_ = fs.Parse(args)

	if opts.isRemote() {
		path := fmt.Sprintf("/api/optimal/graph/hubs?limit=%d", *limit)
		var resp map[string]any
		if err := remoteGet(opts, path, &resp); err != nil {
			return fmt.Errorf("hubs: %w", err)
		}
		printPretty(resp)
		return nil
	}
	return errElixirRequired("hubs")
}

// cmdKnowledge prints the markdown context the knowledge bridge produces.
// Remote only — the bridge runs on the server's *sql.DB.
func cmdKnowledge(args []string, opts globalOpts) error {
	fs := flag.NewFlagSet("knowledge", flag.ExitOnError)
	_ = fs.Parse(args)
	if fs.NArg() == 0 {
		return fmt.Errorf("knowledge: ENTITY argument required")
	}
	if !opts.isRemote() {
		return fmt.Errorf("knowledge: only available in --remote mode")
	}
	entity := strings.Join(fs.Args(), " ")
	body := map[string]any{"entity": entity}
	var resp map[string]any
	if err := remotePost(opts, "/api/optimal/knowledge", body, &resp); err != nil {
		return fmt.Errorf("knowledge: %w", err)
	}
	printPretty(resp)
	return nil
}

// cmdIngest writes a text body through the local pipeline. Remote ingest
// goes through /api/optimal-cloud/signals/ingest which has its own auth +
// workspace-scoping; we don't expose that here to keep the CLI simple.
func cmdIngest(args []string, opts globalOpts) error {
	fs := flag.NewFlagSet("ingest", flag.ExitOnError)
	genre := fs.String("genre", "", "override genre (default: auto-classified)")
	_ = fs.Parse(args)
	if fs.NArg() == 0 {
		return fmt.Errorf("ingest: TEXT argument required")
	}
	text := strings.Join(fs.Args(), " ")
	if opts.isRemote() {
		body := map[string]any{"text": text}
		if *genre != "" {
			body["genre"] = *genre
		}
		var resp map[string]any
		if err := remotePost(opts, "/api/optimal/ingest", body, &resp); err != nil {
			return fmt.Errorf("ingest: %w", err)
		}
		printPretty(resp)
		return nil
	}
	return errElixirRequired("ingest")
}

// truncate is a UI helper — keeps output tidy in narrow terminals.
func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

// humanBytes converts bytes to a brief KB/MB/GB string.
func humanBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

func errElixirRequired(command string) error {
	return fmt.Errorf("%s: local Go-port mode has been removed; start the Elixir OptimalEngine and pass --remote http://localhost:8001", command)
}
