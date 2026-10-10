package collect

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/rhl/businessos-backend/internal/founderos/connectors"
	"github.com/rhl/businessos-backend/internal/founderos/connectors/devicepush"
)

// Config is where this Mac keeps each source. DefaultConfig fills it from
// the home directory plus the same env overrides FounderOS v1 honours.
type Config struct {
	Device string
	Label  string

	WhatsAppDBs   []string
	WisprDB       string
	ObsidianVault string
	Home          string
	// FullDiskAccess says whether macOS-protected files may be opened; when
	// it says no, WhatsApp and a protected Obsidian vault are skipped (no
	// privacy prompt) and report NeedsFullDiskAccess. nil means yes.
	FullDiskAccess func() bool

	ClaudeProjectsDir string
	Claude            SeatInfo
	// ClaudeToken yields the OAuth token for the official plan gauge
	// (CLAUDE_OAUTH_TOKEN, else this Mac's Claude Code Keychain login); nil
	// or "" means no gauge. ClaudeUsageURL is where the gauge is read.
	ClaudeToken    TokenSource
	ClaudeUsageURL string
	CodexDir       string
	// CodexBoardRoot is ~/.paperclip; empty when CODEX_SESSIONS_DIR points
	// the lane at one directory (the TS scans board homes only by default).
	CodexBoardRoot string
	CodexLabel     string

	OllamaBaseURL string
	OllamaLogDir  string

	Stack LocalStackConfig

	ChatLimit  int
	WisprLimit int
	// VaultNotes pushes the vault's note contents for the /brain view.
	VaultNotes bool
	// SourceTimeout is each source's own budget (0: DefaultSourceTimeout).
	// A read past it is abandoned and the tick goes on without it.
	SourceTimeout time.Duration
}

// DefaultSourceTimeout bounds one source's read. A whole tick takes seconds;
// a read still going after a minute is stuck (a macOS privacy prompt holding
// open(2), a locked database, an iCloud file that never downloads).
const DefaultSourceTimeout = 60 * time.Second

var (
	localSuffix = regexp.MustCompile(`(?i)\.local$`)
)

func shortHost(hostname string) string { return localSuffix.ReplaceAllString(hostname, "") }

// DeviceSlug is the hostname as a device id: lowercase, no .local, dashes.
// It is devicepush.HostDeviceSlug, so the bridge's host device and the
// collector's push id can never disagree.
func DeviceSlug(hostname string) string { return devicepush.HostDeviceSlug(hostname) }

// DefaultConfig resolves every location for this Mac. Values come from the
// resolver (env.local, then the process env), else the macOS defaults.
func DefaultConfig(res connectors.Resolver, home, hostname string) Config {
	get := func(k, def string) string {
		if v := res.Resolve(k); v != "" {
			return v
		}
		return def
	}
	host := shortHost(hostname)
	slug := DeviceSlug(hostname)
	c := Config{
		Device:            get("FOUNDEROS_COLLECTOR_DEVICE", slug),
		Label:             get("FOUNDEROS_COLLECTOR_LABEL", host),
		WhatsAppDBs:       WhatsAppContainerPaths(home),
		WisprDB:           filepath.Join(home, "Library", "Application Support", "Wispr Flow", "flow.sqlite"),
		ObsidianVault:     get("OBSIDIAN_VAULT", filepath.Join(home, "Documents", "Obsidian Vault")),
		Home:              home,
		FullDiskAccess:    func() bool { return HasFullDiskAccess(home) },
		ClaudeProjectsDir: get("CLAUDE_PROJECTS_DIR", filepath.Join(home, ".claude", "projects")),
		Claude: SeatInfo{
			ID:         get("FOUNDEROS_OS_SEAT_ID", "claude-"+slug),
			Label:      get("FOUNDEROS_OS_SEAT_LABEL", "Claude · "+host),
			ConfigJSON: get("CLAUDE_CONFIG_JSON", filepath.Join(home, ".claude.json")),
		},
		ClaudeToken:    ClaudeTokenSource(res, KeychainClaudeToken),
		ClaudeUsageURL: ClaudeUsageURL,
		CodexDir:       get("CODEX_SESSIONS_DIR", filepath.Join(home, ".codex", "sessions")),
		CodexLabel:     "Codex · " + host,
		OllamaBaseURL:  get("OLLAMA_BASE_URL", "http://localhost:11434"),
		OllamaLogDir:   get("OLLAMA_LOG_DIR", filepath.Join(home, ".ollama", "logs")),
		Stack: LocalStackConfig{
			PaperclipURL: res.Resolve("PAPERCLIP_API_URL"),
			HermesURL:    res.Resolve("HERMES_GATEWAY_URL"),
			Home:         home,
			BrewDir:      "/opt/homebrew/bin",
			UsrLocalBin:  "/usr/local/bin",
		},
		ChatLimit:  40,
		WisprLimit: 40,
		VaultNotes: true,
	}
	if res.Resolve("CODEX_SESSIONS_DIR") == "" {
		c.CodexBoardRoot = get("PAPERCLIP_HOME", filepath.Join(home, ".paperclip"))
	}
	return c
}

// Collector keeps scan caches between ticks when run as a loop, and knows
// which reads are running and which timed out.
type Collector struct {
	cfg    Config
	claude *ScanCache
	codex  *CodexCache
	gauge  *ClaudeGauge

	mu       sync.Mutex
	inFlight map[string]int
	timedOut []string
}

func NewCollector(cfg Config) *Collector {
	gauge := NewClaudeGauge(cfg.ClaudeToken)
	if cfg.ClaudeUsageURL != "" {
		gauge.URL = cfg.ClaudeUsageURL
	}
	return &Collector{cfg: cfg, claude: NewScanCache(), codex: NewCodexCache(), gauge: gauge, inFlight: map[string]int{}}
}

func (c *Collector) sourceTimeout() time.Duration {
	if c.cfg.SourceTimeout > 0 {
		return c.cfg.SourceTimeout
	}
	return DefaultSourceTimeout
}

// TimedOut names the reads the last Collect gave up on, sorted.
func (c *Collector) TimedOut() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := append([]string(nil), c.timedOut...)
	sort.Strings(out)
	return out
}

// InFlight names the reads still running, sorted: during a tick, what it is
// waiting on; after one, the abandoned reads that are still stuck.
func (c *Collector) InFlight() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := []string{}
	for name, n := range c.inFlight {
		if n > 0 {
			out = append(out, name)
		}
	}
	sort.Strings(out)
	return out
}

// within runs one named read on its own budget (bounded). A read past it is
// abandoned: Go cannot interrupt a blocked syscall, so its goroutine stays
// counted in InFlight until it returns, and the caller gets errTimedOut.
func within[T any](ctx context.Context, c *Collector, name string, fn func(ctx context.Context) (T, error)) (T, error) {
	c.mu.Lock()
	c.inFlight[name]++
	c.mu.Unlock()
	v, err := bounded(ctx, c.sourceTimeout(), func(ctx context.Context) (T, error) {
		defer func() {
			c.mu.Lock()
			c.inFlight[name]--
			c.mu.Unlock()
		}()
		return fn(ctx)
	})
	if errors.Is(err, errTimedOut) {
		c.mu.Lock()
		c.timedOut = append(c.timedOut, name)
		c.mu.Unlock()
	}
	return v, err
}

// status reads one Connections-board row; a timed-out read is an honest
// error row, never a missing one.
func (c *Collector) status(ctx context.Context, id string, fn func(ctx context.Context) connectors.Status) connectors.Status {
	s, err := within(ctx, c, id, func(ctx context.Context) (connectors.Status, error) { return fn(ctx), nil })
	if err != nil {
		m := devicepush.Metas[id]
		return connectors.Status{ID: m.ID, Name: m.Name, Kind: m.Kind, State: connectors.StateError,
			Detail: fmt.Sprintf("read timed out after %s (a macOS privacy prompt, a locked file or a stalled disk); founderos-collector moved on", c.sourceTimeout())}
	}
	return s
}

// Collect reads every source once, concurrently, each on its own budget, so
// one stuck source cannot hold the others or the push. A source whose data
// could not be read (or timed out) is left out of the payload rather than
// pushed as an empty list, so the backend keeps reading it as unknown; its
// status row says why.
func (c *Collector) Collect(ctx context.Context, now time.Time) devicepush.Payload {
	cfg := c.cfg
	c.mu.Lock()
	c.timedOut = nil
	c.mu.Unlock()
	p := devicepush.Payload{Device: cfg.Device, Label: cfg.Label, CapturedAt: isoMillis(now)}

	var (
		wa, wi, ob, ls connectors.Status
		waData         *devicepush.WhatsAppData
		wiData         *devicepush.WisprData
		obData         *devicepush.ObsidianData
		claudeSeat     *devicepush.SeatUsage
		claudeOfficial *devicepush.Official
		codexSeat      *devicepush.SeatUsage
		ollama         *devicepush.OllamaSnapshot
		wg             sync.WaitGroup
	)
	goSource := func(f func()) {
		wg.Add(1)
		go func() { defer wg.Done(); f() }()
	}

	fda := cfg.FullDiskAccess == nil || cfg.FullDiskAccess()
	goSource(func() {
		if !fda { // WhatsApp lives in another app's group container
			wa = needsAccess(devicepush.SourceWhatsApp)
			return
		}
		wa = c.status(ctx, devicepush.SourceWhatsApp, func(ctx context.Context) connectors.Status { return WhatsAppStatus(ctx, cfg.WhatsAppDBs, now) })
		if wa.State != connectors.StateConnected {
			return
		}
		if chats, err := within(ctx, c, "whatsapp-chats", func(ctx context.Context) ([]devicepush.Chat, error) {
			return RecentChats(ctx, ResolveChatDB(cfg.WhatsAppDBs), cfg.ChatLimit, now)
		}); err == nil {
			waData = &devicepush.WhatsAppData{Chats: chats}
		}
	})
	goSource(func() {
		wi = c.status(ctx, devicepush.SourceWispr, func(ctx context.Context) connectors.Status { return WisprStatus(ctx, cfg.WisprDB, now) })
		if wi.State != connectors.StateConnected {
			return
		}
		if notes, err := within(ctx, c, "wispr-notes", func(ctx context.Context) ([]devicepush.WisprNote, error) {
			return RecentWisprNotes(ctx, cfg.WisprDB, cfg.WisprLimit), nil
		}); err == nil {
			wiData = &devicepush.WisprData{Notes: notes}
		}
	})
	goSource(func() {
		if !fda && protectedPath(cfg.ObsidianVault, cfg.Home) {
			ob = needsAccess(devicepush.SourceObsidian)
			return
		}
		ob = c.status(ctx, devicepush.SourceObsidian, func(context.Context) connectors.Status { return ObsidianStatus(cfg.ObsidianVault, cfg.Home) })
		if ob.State != connectors.StateConnected || !cfg.VaultNotes {
			return
		}
		if notes, err := within(ctx, c, "obsidian-notes", func(context.Context) ([]devicepush.VaultNote, error) {
			return ReadVaultNotes(cfg.ObsidianVault), nil
		}); err == nil {
			obData = &devicepush.ObsidianData{Notes: notes}
		}
	})
	goSource(func() {
		ls = c.status(ctx, devicepush.SourceLocalStack, func(ctx context.Context) connectors.Status { return LocalStackStatus(ctx, cfg.Stack) })
	})
	goSource(func() {
		cache := c.claude
		seat, err := within(ctx, c, devicepush.SourceClaudeUsage, func(context.Context) (devicepush.SeatUsage, error) {
			return ScanClaudeProjects(cfg.ClaudeProjectsDir, now, cache, cfg.Claude), nil
		})
		if err != nil {
			c.claude = NewScanCache() // the abandoned scan still holds the old one
			return
		}
		claudeSeat = &seat
	})
	goSource(func() {
		// The official plan gauge, beside the transcript scan rather than after
		// it: its own 4s HTTP budget inside this source's, and nil on any failure.
		gauge := c.gauge
		if o, err := within(ctx, c, "claude-official", func(ctx context.Context) (*devicepush.Official, error) {
			return gauge.Read(ctx), nil
		}); err == nil {
			claudeOfficial = o
		}
	})
	goSource(func() {
		cache := c.codex
		seat, err := within(ctx, c, devicepush.SourceCodexUsage, func(context.Context) (*devicepush.SeatUsage, error) {
			var boards []string
			if cfg.CodexBoardRoot != "" {
				boards = PaperclipCodexHomes(cfg.CodexBoardRoot)
			}
			return CodexSeat([]string{cfg.CodexDir}, boards, now, cache, SeatInfo{ID: "codex-" + DeviceSlug(cfg.Device), Label: cfg.CodexLabel}), nil
		})
		if err != nil {
			c.codex = NewCodexCache()
			return
		}
		codexSeat = seat
	})
	goSource(func() {
		lane, err := within(ctx, c, devicepush.SourceOllamaUsage, func(ctx context.Context) (devicepush.OllamaLane, error) {
			return ReadOllamaLane(ctx, nil, cfg.OllamaBaseURL, cfg.OllamaLogDir, now), nil
		})
		if err == nil && (lane.State == "up" || lane.Requests != nil) {
			ollama = &devicepush.OllamaSnapshot{
				Kind: "ollama", ID: "ollama-" + DeviceSlug(cfg.Device), Label: "Ollama · " + cfg.Label,
				CapturedAt: isoMillis(now), Lane: lane,
			}
		}
	})
	wg.Wait()

	p.Statuses = []connectors.Status{wa, wi, ob, ls}
	p.WhatsApp, p.Wispr, p.Obsidian, p.Ollama = waData, wiData, obData, ollama
	if claudeSeat != nil {
		p.Usage = append(p.Usage, WithClaudeOfficial(*claudeSeat, claudeOfficial))
	}
	if codexSeat != nil {
		p.Usage = append(p.Usage, *codexSeat)
	}
	return p
}

// Push POSTs the payload to the backend's ingestion URL (on this Mac or the
// tailnet). This is the bridge's own ingestion, not an outbound side effect,
// so it uses a plain client. token, when set, rides as a Bearer header.
func Push(ctx context.Context, client *http.Client, url, token string, p devicepush.Payload) error {
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	body, err := json.Marshal(p)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode > 299 {
		raw, _ := io.ReadAll(io.LimitReader(res.Body, 300))
		return fmt.Errorf("push to %s refused: HTTP %d: %s", url, res.StatusCode, strings.TrimSpace(string(raw)))
	}
	return nil
}
