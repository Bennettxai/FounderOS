package main

import (
	"errors"
	"os"
)

// globalOpts gathers the cross-command flags. Per-command flags are parsed
// inside each command's own FlagSet.
type globalOpts struct {
	remote string // base URL when set; "" = local mode
	dbPath string // SQLite index path
	cookie string // cookie jar for --remote auth
}

// parseGlobalFlags pulls --remote / --db / --cookie out of argv before
// the subcommand sees it. Subcommands then get a clean argv to parse.
//
// We don't use flag.Parse here because the standard library would consume
// every --x=… on the line, including subcommand-specific ones. Instead we
// walk argv manually until we hit the first non-flag token (the command
// name), then hand the rest off untouched.
func parseGlobalFlags(args []string) (globalOpts, []string) {
	opts := globalOpts{
		remote: os.Getenv("OPTIMAL_REMOTE_URL"),
		dbPath: envOr("OPTIMAL_ENGINE_DB", "./optimal.db"),
		cookie: os.Getenv("OPTIMAL_COOKIE_JAR"),
	}
	rest := []string{}
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--remote" && i+1 < len(args):
			opts.remote = args[i+1]
			i++
		case startsWith(a, "--remote="):
			opts.remote = a[len("--remote="):]
		case a == "--db" && i+1 < len(args):
			opts.dbPath = args[i+1]
			i++
		case startsWith(a, "--db="):
			opts.dbPath = a[len("--db="):]
		case a == "--cookie" && i+1 < len(args):
			opts.cookie = args[i+1]
			i++
		case startsWith(a, "--cookie="):
			opts.cookie = a[len("--cookie="):]
		default:
			// First non-flag token plus everything after it belongs to the
			// subcommand. Stop consuming.
			rest = args[i:]
			return opts, rest
		}
	}
	return opts, rest
}

// isRemote reports whether the user asked for remote mode.
func (g globalOpts) isRemote() bool { return g.remote != "" }

// requireDB returns an error if local mode was selected but the SQLite
// file doesn't exist on disk. Callers fail fast rather than walking into
// a sqlite "unable to open" mid-command.
func (g globalOpts) requireDB() error {
	if _, err := os.Stat(g.dbPath); err != nil {
		return errors.New("local db not found at " + g.dbPath +
			" — set --db PATH or OPTIMAL_ENGINE_DB, or use --remote URL")
	}
	return nil
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func startsWith(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}
