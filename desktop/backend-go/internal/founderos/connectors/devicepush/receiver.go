package devicepush

import (
	"context"
	"errors"
	"fmt"
	"math"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/rhl/businessos-backend/internal/founderos/connectors"
)

// StaleAfter matches FounderOS v1: a pushed usage snapshot older than 24h is
// stale (STALE_PUSH_MS), and Wispr says "stale on this box" past 1440 minutes.
const StaleAfter = 24 * time.Hour

// Caps on one push, so a runaway collector cannot balloon the store.
const (
	maxChats      = 200
	maxWisprNotes = 200
	maxVaultNotes = 5000
	maxSeats      = 8
)

// ErrNoPush means no device has pushed this source yet: unknown, not empty.
var ErrNoPush = errors.New("devicepush: no device has pushed this source yet")

// Received is a payload as stored, stamped with the receiver's clock.
type Received struct {
	Payload    Payload   `json:"payload"`
	ReceivedAt time.Time `json:"receivedAt"`
}

// Store keeps the latest payload per device. Latest must return an error
// when the store is unreachable, never an empty list.
type Store interface {
	Save(ctx context.Context, r Received) error
	Latest(ctx context.Context) ([]Received, error)
}

// MemStore is the in-process Store.
type MemStore struct {
	mu     sync.RWMutex
	latest map[string]Received
}

func NewMemStore() *MemStore { return &MemStore{latest: map[string]Received{}} }

func (m *MemStore) Save(_ context.Context, r Received) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.latest[r.Payload.Device] = r
	return nil
}

func (m *MemStore) Latest(context.Context) ([]Received, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]Received, 0, len(m.latest))
	for _, r := range m.latest {
		out = append(out, r)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Payload.Device < out[j].Payload.Device })
	return out, nil
}

// Receiver validates and stores pushes and answers status from them.
type Receiver struct {
	store    Store
	expected []string
	now      func() time.Time
	// StaleAfter is how old a push may be before it reads as stale.
	StaleAfter time.Duration
	// Host is the device this bridge runs on (HostDevice()): the machine
	// whose local stack FounderOS v1 checked, read through HostConnector.
	Host string
}

var hostSlugJunk = regexp.MustCompile(`[^a-z0-9]+`)

// HostDeviceSlug is a hostname as a device id (lowercase, no .local, dashes):
// the id founderos-collector pushes under by default.
func HostDeviceSlug(hostname string) string {
	h := strings.TrimSuffix(strings.ToLower(strings.TrimSpace(hostname)), ".local")
	return strings.Trim(hostSlugJunk.ReplaceAllString(h, "-"), "-")
}

// NewReceiver takes the devices expected to push (e.g. "macbook", "mini"),
// so a Mac whose collector never ran shows as not_configured.
func NewReceiver(store Store, expectedDevices ...string) *Receiver {
	return &Receiver{store: store, expected: expectedDevices, now: time.Now, StaleAfter: StaleAfter}
}

var (
	deviceSlug = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,62}$`)
	dayRe      = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
)

func validState(s connectors.State) bool {
	return s == connectors.StateConnected || s == connectors.StateNotConfigured || s == connectors.StateError
}

// Validate checks a payload the way the FounderOS v1 push route's Zod schemas do.
func Validate(p Payload) error {
	if !deviceSlug.MatchString(p.Device) {
		return fmt.Errorf("device must be a lowercase slug, got %q", p.Device)
	}
	for _, s := range p.Statuses {
		if _, ok := Metas[s.ID]; !ok {
			return fmt.Errorf("status for unknown source %q", s.ID)
		}
		if !validState(s.State) {
			return fmt.Errorf("status %s has invalid state %q", s.ID, s.State)
		}
	}
	if p.WhatsApp != nil && len(p.WhatsApp.Chats) > maxChats {
		return fmt.Errorf("at most %d chats per push", maxChats)
	}
	if p.Wispr != nil {
		if len(p.Wispr.Notes) > maxWisprNotes {
			return fmt.Errorf("at most %d wispr notes per push", maxWisprNotes)
		}
		for _, n := range p.Wispr.Notes {
			if n.ID == "" || n.Text == "" || n.TS == "" || (n.Kind != "dictation" && n.Kind != "note" && n.Kind != "todo") {
				return fmt.Errorf("invalid wispr note %q", n.ID)
			}
		}
	}
	if p.Obsidian != nil && len(p.Obsidian.Notes) > maxVaultNotes {
		return fmt.Errorf("at most %d vault notes per push", maxVaultNotes)
	}
	if len(p.Usage) > maxSeats {
		return fmt.Errorf("at most %d usage seats per push", maxSeats)
	}
	for _, s := range p.Usage {
		if err := validSeat(s); err != nil {
			return err
		}
	}
	if o := p.Ollama; o != nil {
		if o.Kind != "ollama" || o.ID == "" || o.Label == "" || o.CapturedAt == "" {
			return errors.New("ollama snapshot needs kind ollama, id, label and capturedAt")
		}
		if o.Lane.State != "up" && o.Lane.State != "down" {
			return fmt.Errorf("ollama lane state %q", o.Lane.State)
		}
	}
	return nil
}

func validSeat(s SeatUsage) error {
	if s.ID == "" || s.Label == "" || s.CapturedAt == "" {
		return errors.New("usage seat needs id, label and capturedAt")
	}
	if s.Kind != "claude" && s.Kind != "codex" {
		return fmt.Errorf("usage seat kind %q", s.Kind)
	}
	for _, d := range s.Days {
		if !dayRe.MatchString(d.Day) || d.In < 0 || d.Out < 0 || d.CacheWrite < 0 || d.CacheRead < 0 {
			return fmt.Errorf("usage seat %s has an invalid day bucket %q", s.ID, d.Day)
		}
	}
	if o := s.Official; o != nil {
		// OfficialWindowSchema: usedPercent >= 0, windowMinutes a positive int
		for _, w := range []*OfficialWindow{o.Session, o.Weekly} {
			if w != nil && (w.UsedPercent < 0 || math.IsNaN(w.UsedPercent) || w.WindowMinutes <= 0) {
				return fmt.Errorf("usage seat %s has an invalid official gauge", s.ID)
			}
		}
	}
	if b := s.Breakdown; b != nil {
		for _, t := range b.Top {
			switch t.Source { // BURN_SOURCES
			case "board", "sessions", "terminal", "automation":
			default:
				return fmt.Errorf("usage seat %s has an unknown burn lane %q", s.ID, t.Source)
			}
			if t.Burn < 0 || math.IsNaN(t.Burn) {
				return fmt.Errorf("usage seat %s has a negative burn for %q", s.ID, t.Label)
			}
		}
	}
	return nil
}

// Accept validates and stores one push. Provenance is the receiver's fact to
// assert: every seat is stored as source "push", whatever the sender said.
func (r *Receiver) Accept(ctx context.Context, p Payload) error {
	if err := Validate(p); err != nil {
		return err
	}
	if p.Label == "" {
		p.Label = p.Device
	}
	seats := make([]SeatUsage, len(p.Usage))
	for i, s := range p.Usage {
		s.Source = "push"
		seats[i] = s
	}
	p.Usage = seats
	return r.store.Save(ctx, Received{Payload: p, ReceivedAt: r.now().UTC()})
}

// ---- device status -------------------------------------------------------------

type DeviceState string

const (
	DeviceConnected     DeviceState = "connected"
	DeviceStale         DeviceState = "stale"
	DeviceNotConfigured DeviceState = "not_configured"
)

type DeviceStatus struct {
	Device     string      `json:"device"`
	Label      string      `json:"label"`
	State      DeviceState `json:"state"`
	Detail     string      `json:"detail"`
	LastPushAt *time.Time  `json:"lastPushAt"`
	Sources    []string    `json:"sources"`
}

// Age is the compact "7m ago" / "25h ago" / "3d ago" the board shows.
func Age(d time.Duration) string {
	mins := int(math.Round(d.Minutes()))
	if mins < 0 {
		mins = 0
	}
	switch {
	case mins < 60:
		return fmt.Sprintf("%dm ago", mins)
	case mins < 48*60:
		return fmt.Sprintf("%dh ago", int(math.Round(float64(mins)/60)))
	default:
		return fmt.Sprintf("%dd ago", int(math.Round(float64(mins)/1440)))
	}
}

func (r *Receiver) stale(rec Received) bool { return r.now().Sub(rec.ReceivedAt) > r.StaleAfter }

func sourcesOf(p Payload) []string {
	var out []string
	for _, s := range p.Statuses {
		out = append(out, s.ID)
	}
	if len(p.Usage) > 0 {
		for _, s := range p.Usage {
			if s.Kind == "claude" {
				out = append(out, SourceClaudeUsage)
			} else {
				out = append(out, SourceCodexUsage)
			}
		}
	}
	if p.Ollama != nil {
		out = append(out, SourceOllamaUsage)
	}
	sort.Strings(out)
	return out
}

// Devices reports every device that has pushed plus every expected one:
// connected while its last push is fresh, stale past StaleAfter, and
// not_configured for an expected device that never pushed.
func (r *Receiver) Devices(ctx context.Context) ([]DeviceStatus, error) {
	latest, err := r.store.Latest(ctx)
	if err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	out := []DeviceStatus{}
	for _, rec := range latest {
		at := rec.ReceivedAt
		d := DeviceStatus{Device: rec.Payload.Device, Label: rec.Payload.Label, LastPushAt: &at, Sources: sourcesOf(rec.Payload)}
		age := Age(r.now().Sub(rec.ReceivedAt))
		if r.stale(rec) {
			d.State = DeviceStale
			d.Detail = fmt.Sprintf("last push %s · stale, is founderos-collector still running on %s?", age, d.Label)
		} else {
			d.State = DeviceConnected
			d.Detail = "last push " + age
		}
		seen[d.Device] = true
		out = append(out, d)
	}
	for _, e := range r.expected {
		if !seen[e] {
			out = append(out, DeviceStatus{Device: e, Label: e, State: DeviceNotConfigured,
				Detail: "no push yet: install founderos-collector on this Mac", Sources: []string{}})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Device < out[j].Device })
	return out, nil
}

// ---- per-source connectors ----------------------------------------------------

// Connector returns the Connections-board row for a device-local source id
// (whatsapp, wispr, obsidian, local-stack).
func (r *Receiver) Connector(id string) connectors.Connector { return sourceConnector{r: r, id: id} }

// HostConnector reads a source from the host device only (Receiver.Host):
// for machine-local checks such as the stack, where another Mac's healthy
// reading must not stand in for this one.
func (r *Receiver) HostConnector(id string) connectors.Connector {
	return sourceConnector{r: r, id: id, device: r.Host, hostOnly: true}
}

type sourceConnector struct {
	r        *Receiver
	id       string
	device   string // set: only this device's reading counts
	hostOnly bool   // HostConnector: an unset host is an error, never best-of
}

type candidate struct {
	rec    Received
	status connectors.Status
}

func stateRank(s connectors.State) int {
	switch s {
	case connectors.StateConnected:
		return 0
	case connectors.StateError:
		return 1
	default:
		return 2
	}
}

// Status reads the host device's fresh push first (Receiver.Host, as the operator
// OS read the machine it ran on). Without one it picks the best fresh reading
// across devices: a connected one first, then an error, then not_configured,
// newest first within each. When every
// reading is stale the source reads as an error that says so, never as a
// connected source nobody has looked at for a day.
func (c sourceConnector) Status(ctx context.Context) connectors.Status {
	meta := Metas[c.id]
	if meta.ID == "" {
		meta = connectors.Meta{ID: c.id, Name: c.id, Kind: connectors.KindLocal}
	}
	base := connectors.Status{ID: meta.ID, Name: meta.Name, Kind: meta.Kind}
	if c.hostOnly && c.device == "" {
		base.State = connectors.StateError
		base.Detail = "FounderOS does not know which device it runs on (set FOUNDEROS_HOST_DEVICE)"
		return base
	}
	latest, err := c.r.store.Latest(ctx)
	if err != nil {
		base.State = connectors.StateError
		base.Detail = "device push store unreadable: " + err.Error()
		return base
	}
	var fresh, old []candidate
	for _, rec := range latest {
		if c.device != "" && rec.Payload.Device != c.device {
			continue
		}
		for _, s := range rec.Payload.Statuses {
			if s.ID != c.id {
				continue
			}
			if c.r.stale(rec) {
				old = append(old, candidate{rec, s})
			} else {
				fresh = append(fresh, candidate{rec, s})
			}
		}
	}
	newest := func(xs []candidate) {
		sort.SliceStable(xs, func(i, j int) bool {
			if a, b := stateRank(xs[i].status.State), stateRank(xs[j].status.State); a != b {
				return a < b
			}
			return xs[i].rec.ReceivedAt.After(xs[j].rec.ReceivedAt)
		})
	}
	if len(fresh) == 0 && len(old) == 0 && c.device != "" {
		// the host must report its own stack; silence is not "fine"
		base.State = connectors.StateError
		base.Detail = fmt.Sprintf("the host %s has not pushed %s; run founderos-collector there", c.device, meta.Name)
		return base
	}
	if len(fresh) == 0 && len(old) == 0 {
		base.State = connectors.StateNotConfigured
		base.Detail = fmt.Sprintf("No collector has pushed %s yet. Run founderos-collector on the Mac that has it.", meta.Name)
		return base
	}
	if len(fresh) == 0 {
		sort.SliceStable(old, func(i, j int) bool { return old[i].rec.ReceivedAt.After(old[j].rec.ReceivedAt) })
		o := old[0]
		base.State = connectors.StateError
		base.Detail = fmt.Sprintf("stale: last push from %s %s (was: %s)", o.rec.Payload.Label, Age(c.r.now().Sub(o.rec.ReceivedAt)), o.status.Detail)
		base.Meta = map[string]any{"device": o.rec.Payload.Device, "pushedAt": o.rec.ReceivedAt.Format(time.RFC3339)}
		return base
	}
	// FounderOS v1 read the box it ran on: a fresh push from the host wins,
	// whatever its state; another Mac stands in only when the host has none.
	if c.r.Host != "" {
		var host []candidate
		for _, x := range fresh {
			if x.rec.Payload.Device == c.r.Host {
				host = append(host, x)
			}
		}
		if len(host) > 0 {
			fresh = host
		}
	}
	newest(fresh)
	best := fresh[0]
	base.State = best.status.State
	base.Detail = fmt.Sprintf("%s · pushed by %s %s", best.status.Detail, best.rec.Payload.Label, Age(c.r.now().Sub(best.rec.ReceivedAt)))
	base.Meta = map[string]any{}
	for k, v := range best.status.Meta {
		base.Meta[k] = v
	}
	base.Meta["device"] = best.rec.Payload.Device
	base.Meta["pushedAt"] = best.rec.ReceivedAt.Format(time.RFC3339)
	return base
}

// ---- reads ----------------------------------------------------------------

// Reading is pushed data with where and when it came from. Stale readings are
// still returned, flagged, so a page can show old data as old.
type Reading[T any] struct {
	Device   string    `json:"device"`
	Label    string    `json:"label"`
	PushedAt time.Time `json:"pushedAt"`
	Stale    bool      `json:"stale"`
	Data     T         `json:"data"`
}

// freshest returns the newest device reading that carries the source.
func freshest[T any](r *Receiver, ctx context.Context, pick func(Payload) (T, bool)) (Reading[T], error) {
	latest, err := r.store.Latest(ctx)
	if err != nil {
		return Reading[T]{}, err
	}
	var best *Reading[T]
	for _, rec := range latest {
		v, ok := pick(rec.Payload)
		if !ok || (best != nil && !rec.ReceivedAt.After(best.PushedAt)) {
			continue
		}
		best = &Reading[T]{Device: rec.Payload.Device, Label: rec.Payload.Label, PushedAt: rec.ReceivedAt, Stale: r.stale(rec), Data: v}
	}
	if best == nil {
		return Reading[T]{}, ErrNoPush
	}
	return *best, nil
}

// WhatsAppChats is the recent-chats lane from the freshest device that has it.
func (r *Receiver) WhatsAppChats(ctx context.Context) (Reading[[]Chat], error) {
	return freshest(r, ctx, func(p Payload) ([]Chat, bool) {
		if p.WhatsApp == nil {
			return nil, false
		}
		return p.WhatsApp.Chats, true
	})
}

// WisprNotes is the Wispr notes lane from the freshest device that has it.
func (r *Receiver) WisprNotes(ctx context.Context) (Reading[[]WisprNote], error) {
	return freshest(r, ctx, func(p Payload) ([]WisprNote, bool) {
		if p.Wispr == nil {
			return nil, false
		}
		return p.Wispr.Notes, true
	})
}

// VaultNotes are the Obsidian vault notes from the freshest device that has them.
func (r *Receiver) VaultNotes(ctx context.Context) (Reading[[]VaultNote], error) {
	return freshest(r, ctx, func(p Payload) ([]VaultNote, bool) {
		if p.Obsidian == nil {
			return nil, false
		}
		return p.Obsidian.Notes, true
	})
}

// UsageSeats are every device's Claude and Codex seats; combining them into
// one plan is the /usage page's job (combineSeats).
func (r *Receiver) UsageSeats(ctx context.Context) ([]Reading[SeatUsage], error) {
	latest, err := r.store.Latest(ctx)
	if err != nil {
		return nil, err
	}
	out := []Reading[SeatUsage]{}
	for _, rec := range latest {
		for _, s := range rec.Payload.Usage {
			out = append(out, Reading[SeatUsage]{Device: rec.Payload.Device, Label: rec.Payload.Label, PushedAt: rec.ReceivedAt, Stale: r.stale(rec), Data: s})
		}
	}
	return out, nil
}

// OllamaSnapshots are every device's Ollama lane.
func (r *Receiver) OllamaSnapshots(ctx context.Context) ([]Reading[OllamaSnapshot], error) {
	latest, err := r.store.Latest(ctx)
	if err != nil {
		return nil, err
	}
	out := []Reading[OllamaSnapshot]{}
	for _, rec := range latest {
		if rec.Payload.Ollama != nil {
			out = append(out, Reading[OllamaSnapshot]{Device: rec.Payload.Device, Label: rec.Payload.Label, PushedAt: rec.ReceivedAt, Stale: r.stale(rec), Data: *rec.Payload.Ollama})
		}
	}
	return out, nil
}
