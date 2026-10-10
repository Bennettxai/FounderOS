package collect

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/rhl/businessos-backend/internal/founderos/connectors"
	"github.com/rhl/businessos-backend/internal/founderos/connectors/devicepush"
)

// LocalStackConfig is what lib/connectors/local-stack.ts checks: the two
// agent services FounderOS v1 talks to and the CLIs it shells out to.
type LocalStackConfig struct {
	PaperclipURL string // PAPERCLIP_API_URL, else http://localhost:3100
	HermesURL    string // HERMES_GATEWAY_URL, else http://localhost:8642
	Home         string
	BrewDir      string // /opt/homebrew/bin
	UsrLocalBin  string // /usr/local/bin
	// Client pings the services; nil means a plain client. These are
	// liveness GETs to the Mac itself or the tailnet, never outbound writes.
	Client *http.Client
}

type stackCheck struct {
	name   string
	up     bool
	detail string
}

func ping(ctx context.Context, client *http.Client, url string) bool {
	ctx, cancel := context.WithTimeout(ctx, 1500*time.Millisecond)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return false
	}
	res, err := client.Do(req)
	if err != nil {
		return false
	}
	res.Body.Close()
	return res.StatusCode > 0 // any answer means the service is up
}

// binExists reports whether any candidate is an executable file (X_OK).
func binExists(candidates ...string) bool {
	for _, c := range candidates {
		if c == "" {
			continue
		}
		if fi, err := os.Stat(c); err == nil && !fi.IsDir() && syscall.Access(c, 0x1) == nil {
			return true
		}
	}
	return false
}

var scheme = regexp.MustCompile(`^https?://`)

// LocalStackStatus lists only what FounderOS v1 itself runs or shells out to,
// all checked live. No self-check: a check that can only be true-or-wrong is
// worse than none.
func LocalStackStatus(ctx context.Context, cfg LocalStackConfig) connectors.Status {
	client := cfg.Client
	if client == nil {
		client = &http.Client{}
	}
	paperclip := cfg.PaperclipURL
	if paperclip == "" {
		paperclip = "http://localhost:3100"
	}
	hermes := cfg.HermesURL
	if hermes == "" {
		hermes = "http://localhost:8642"
	}
	var board, gateway bool
	var wg sync.WaitGroup
	wg.Add(2)
	go func() { defer wg.Done(); board = ping(ctx, client, paperclip) }()
	go func() { defer wg.Done(); gateway = ping(ctx, client, hermes) }()
	wg.Wait()

	brew := func(n string) string { return filepath.Join(cfg.BrewDir, n) }
	usr := func(n string) string { return filepath.Join(cfg.UsrLocalBin, n) }
	checks := []stackCheck{
		{"paperclip", board, "agent board · " + scheme.ReplaceAllString(paperclip, "")},
		{"hermes", gateway, "worker pool · " + scheme.ReplaceAllString(hermes, "")},
		{"ffmpeg", binExists(brew("ffmpeg"), usr("ffmpeg")), "content-gen media processing"},
		{"pdftotext", binExists(brew("pdftotext"), usr("pdftotext")), "statement ingestion (/finances)"},
		{"whisper", binExists(brew("whisper-cli"), usr("whisper-cli")), "local transcription"},
		{"gh", binExists(brew("gh"), usr("gh")), "GitHub CLI · deploys"},
	}

	var up, down []string
	meta := map[string]any{}
	for _, c := range checks {
		if c.up {
			up = append(up, c.name)
			meta[c.name] = "up · " + c.detail
		} else {
			down = append(down, c.name)
			meta[c.name] = "down"
		}
	}
	m := devicepush.Metas[devicepush.SourceLocalStack]
	s := connectors.Status{ID: m.ID, Name: m.Name, Kind: m.Kind, State: connectors.StateError, Meta: meta}
	if len(up) > 0 {
		s.State = connectors.StateConnected
	}
	s.Detail = fmt.Sprintf("%d/%d up — %s", len(up), len(checks), strings.Join(up, ", "))
	if len(down) > 0 {
		s.Detail += " · down: " + strings.Join(down, ", ")
	}
	return s
}
