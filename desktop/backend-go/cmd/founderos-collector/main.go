// Command founderos-collector runs on each of the operator's Macs. It reads the
// device-local sources (WhatsApp, Wispr Flow, the Obsidian vault, the local
// stack, and Claude/Codex/Ollama usage) read-only, exactly as FounderOS v1's
// lib/connectors modules do, and POSTs one devicepush.Payload to the bridge
// backend, the same model as scripts/push-usage.mjs. The backend never reads
// a device path itself.
//
//	founderos-collector -url http://mini:8801/<ingest path>          # one tick
//	founderos-collector -url ... -every 10m                                    # loop
//	founderos-collector -dry-run                                               # print, push nothing
//
// Every tick has a deadline (-deadline, 4m) under launchd's 600s
// StartInterval, which never starts a run while the last one is alive: a tick
// past it ends the process, naming the reads it was still waiting on, so one
// stuck source costs a tick, never every push after it. Each source also has
// its own budget (-source-timeout); a source past it is pushed as timed out.
//
// The URL can also come from FOUNDEROS_COLLECTOR_URL and a bearer token from
// FOUNDEROS_COLLECTOR_TOKEN (env.local or the process env). It invokes no LLM
// and bills nothing. Its one Anthropic call is the official Claude plan gauge
// (a free GET to api.anthropic.com/api/oauth/usage, at most once a minute)
// with CLAUDE_OAUTH_TOKEN or this Mac's Claude Code Keychain login, read-only;
// only the percentages and reset times are pushed, never the token.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/rhl/businessos-backend/internal/founderos/connectors"
	"github.com/rhl/businessos-backend/internal/founderos/connectors/devicepush/collect"
)

const (
	// defaultDeadline is one tick's whole budget, under launchd's 600s interval.
	defaultDeadline = 4 * time.Minute
	// defaultSourceTimeout is each source's budget within a tick.
	defaultSourceTimeout = collect.DefaultSourceTimeout
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("founderos-collector", flag.ContinueOnError)
	fs.SetOutput(stderr)
	url := fs.String("url", "", "backend ingestion URL (default $FOUNDEROS_COLLECTOR_URL)")
	every := fs.Duration("every", 0, "repeat on this interval; 0 runs one tick")
	dryRun := fs.Bool("dry-run", false, "print the payload as JSON instead of pushing it")
	noVault := fs.Bool("no-vault-notes", false, "push the vault's status only, not its note contents")
	deadline := fs.Duration("deadline", defaultDeadline, "end the process when a tick runs past this")
	sourceTimeout := fs.Duration("source-timeout", defaultSourceTimeout, "give up on one source's read after this")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	res := connectors.DefaultResolver()
	if *url == "" {
		*url = res.Resolve("FOUNDEROS_COLLECTOR_URL")
	}
	if *url == "" && !*dryRun {
		fmt.Fprintln(stderr, "founderos-collector: no backend URL: pass -url or set FOUNDEROS_COLLECTOR_URL (or use -dry-run)")
		return 2
	}
	token := res.Resolve("FOUNDEROS_COLLECTOR_TOKEN")

	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintln(stderr, "founderos-collector: no home directory:", err)
		return 1
	}
	host, _ := os.Hostname()
	cfg := collect.DefaultConfig(res, home, host)
	cfg.VaultNotes = !*noVault
	cfg.SourceTimeout = *sourceTimeout
	c := collect.NewCollector(cfg)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	// work is one tick's reads and push. It writes nothing itself: a tick
	// abandoned at its deadline must not print after run has returned.
	type result struct {
		code     int
		out, err string
	}
	work := func(ctx context.Context) result {
		var r result
		p := c.Collect(ctx, time.Now())
		if late := c.TimedOut(); len(late) > 0 {
			r.err = fmt.Sprintf("founderos-collector: %s timed out after %s; pushed without it\n", strings.Join(late, ", "), *sourceTimeout)
		}
		if skipped := collect.SkippedForAccess(p); len(skipped) > 0 {
			r.err += fmt.Sprintf("founderos-collector: skipped %s (no Full Disk Access, so no privacy prompt); grant it once in System Settings → Privacy & Security → Full Disk Access\n", strings.Join(skipped, ", "))
		}
		if *dryRun {
			raw, err := json.MarshalIndent(p, "", "  ")
			if err != nil {
				r.code, r.err = 1, r.err+fmt.Sprintln("founderos-collector:", err)
				return r
			}
			r.out = string(raw) + "\n"
			return r
		}
		if err := collect.Push(ctx, nil, *url, token, p); err != nil {
			r.code, r.err = 1, r.err+fmt.Sprintln("founderos-collector:", err)
			return r
		}
		r.out = fmt.Sprintf("pushed %s to %s\n", p.Device, *url)
		return r
	}
	emit := func(r result) int {
		fmt.Fprint(stdout, r.out)
		fmt.Fprint(stderr, r.err)
		return r.code
	}

	// tick runs work under the deadline. expired means the process must end:
	// a read blocked in a syscall cannot be interrupted, only exited.
	tick := func() (code int, expired bool) {
		tctx, cancel := context.WithTimeout(ctx, *deadline)
		defer cancel()
		done := make(chan result, 1)
		go func() { done <- work(tctx) }()
		select {
		case r := <-done:
			if tctx.Err() == nil {
				return emit(r), false
			}
			// finished only because the deadline cut its reads short
		case <-tctx.Done():
		}
		if ctx.Err() != nil {
			return 1, true // interrupted
		}
		waiting := strings.Join(c.InFlight(), ", ")
		if waiting == "" {
			waiting = "the push"
		}
		fmt.Fprintf(stderr, "founderos-collector: tick passed its %s deadline still waiting on %s; exiting so the next run can start\n", *deadline, waiting)
		return 1, true
	}

	if *every <= 0 {
		code, _ := tick()
		return code
	}
	t := time.NewTicker(*every)
	defer t.Stop()
	for {
		if code, expired := tick(); expired { // a failed tick is logged; the loop keeps going
			return code
		}
		select {
		case <-ctx.Done():
			return 0
		case <-t.C:
		}
	}
}
