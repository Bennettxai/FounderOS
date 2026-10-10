package devicepush

import (
	"context"
	"errors"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/rhl/businessos-backend/internal/founderos/connectors"
)

var t0 = time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC)

func st(id string, state connectors.State, detail string) connectors.Status {
	m := Metas[id]
	return connectors.Status{ID: m.ID, Name: m.Name, Kind: m.Kind, State: state, Detail: detail}
}

func receiver(expected ...string) (*Receiver, *time.Time) {
	now := t0
	r := NewReceiver(NewMemStore(), expected...)
	r.now = func() time.Time { return now }
	return r, &now
}

func TestMetasMatchFounderosOS(t *testing.T) {
	want := map[string]connectors.Meta{
		"whatsapp":    {ID: "whatsapp", Name: "WhatsApp", Kind: connectors.KindSocial},
		"wispr":       {ID: "wispr", Name: "Wispr Flow", Kind: connectors.KindLocal},
		"obsidian":    {ID: "obsidian", Name: "Obsidian Vault", Kind: connectors.KindKnowledge},
		"local-stack": {ID: "local-stack", Name: "Local Stack", Kind: connectors.KindLocal},
	}
	if len(Metas) != len(want) {
		t.Fatalf("Metas = %v", Metas)
	}
	for id, m := range want {
		if Metas[id] != m {
			t.Errorf("%s: %+v, want %+v", id, Metas[id], m)
		}
	}
}

func TestAcceptValidates(t *testing.T) {
	r, _ := receiver()
	ctx := context.Background()
	bad := []Payload{
		{},
		{Device: "Bad Device!"},
		{Device: "mini", Statuses: []connectors.Status{{ID: "stripe", State: connectors.StateConnected}}},
		{Device: "mini", Statuses: []connectors.Status{{ID: "wispr", State: "great"}}},
		{Device: "mini", Usage: []SeatUsage{{ID: "", Kind: "claude", Label: "x", CapturedAt: "2026-09-29T00:00:00Z"}}},
		{Device: "mini", Usage: []SeatUsage{{ID: "s", Kind: "gemini", Label: "x", CapturedAt: "2026-09-29T00:00:00Z"}}},
		{Device: "mini", Usage: []SeatUsage{{ID: "s", Kind: "claude", Label: "x", CapturedAt: "2026-09-29T00:00:00Z", Days: []DayBucket{{Day: "29/09"}}}}},
		{Device: "mini", Ollama: &OllamaSnapshot{Kind: "ollama", ID: "o", Label: "x", CapturedAt: "t", Lane: OllamaLane{State: "sideways"}}},
	}
	for i, p := range bad {
		if err := r.Accept(ctx, p); err == nil {
			t.Errorf("payload %d accepted: %+v", i, p)
		}
	}
	if devs, _ := r.Devices(ctx); len(devs) != 0 {
		t.Errorf("a refused payload must store nothing: %+v", devs)
	}
}

func TestAcceptStampsReceiverTimeAndPushProvenance(t *testing.T) {
	r, _ := receiver()
	ctx := context.Background()
	err := r.Accept(ctx, Payload{
		Device: "macbook", Label: "MacBook", CapturedAt: "2020-01-01T00:00:00Z",
		Usage: []SeatUsage{{ID: "claude-macbook", Kind: "claude", Label: "Claude · MacBook", Source: "local", CapturedAt: "2026-09-29T11:59:00Z"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	seats, err := r.UsageSeats(ctx)
	if err != nil || len(seats) != 1 {
		t.Fatalf("seats = %+v, %v", seats, err)
	}
	if seats[0].Data.Source != "push" || !seats[0].PushedAt.Equal(t0) || seats[0].Device != "macbook" {
		t.Errorf("reading = %+v", seats[0])
	}
}

func TestDevicesConnectedStaleNotConfigured(t *testing.T) {
	r, now := receiver("macbook", "mini")
	ctx := context.Background()
	must(t, r.Accept(ctx, Payload{Device: "mini", Label: "Mac mini", Statuses: []connectors.Status{st("whatsapp", connectors.StateConnected, "4 chats")}}))
	*now = now.Add(2 * time.Hour)
	must(t, r.Accept(ctx, Payload{Device: "macbook", Label: "MacBook"}))
	*now = now.Add(23 * time.Hour) // mini last pushed 25h ago, macbook 23h ago

	devs, err := r.Devices(ctx)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]DeviceStatus{}
	for _, d := range devs {
		got[d.Device] = d
	}
	if got["macbook"].State != DeviceConnected || got["mini"].State != DeviceStale {
		t.Fatalf("devices = %+v", devs)
	}
	if got["mini"].LastPushAt == nil || !strings.Contains(got["mini"].Detail, "25h ago") || !strings.Contains(got["mini"].Detail, "stale") {
		t.Errorf("mini = %+v", got["mini"])
	}
	if len(got["mini"].Sources) != 1 || got["mini"].Sources[0] != "whatsapp" {
		t.Errorf("sources = %v", got["mini"].Sources)
	}

	r2, _ := receiver("studio")
	devs, _ = r2.Devices(ctx)
	if len(devs) != 1 || devs[0].State != DeviceNotConfigured || devs[0].LastPushAt != nil {
		t.Fatalf("an expected device that never pushed is not_configured: %+v", devs)
	}
}

func TestSourceStatusNeverPushedIsNotConfigured(t *testing.T) {
	r, _ := receiver()
	s := r.Connector("wispr").Status(context.Background())
	if s.State != connectors.StateNotConfigured || s.ID != "wispr" || s.Name != "Wispr Flow" || !strings.Contains(s.Detail, "founderos-collector") {
		t.Fatalf("status = %+v", s)
	}
}

func TestSourceStatusPrefersAFreshConnectedDevice(t *testing.T) {
	r, now := receiver()
	ctx := context.Background()
	must(t, r.Accept(ctx, Payload{Device: "macbook", Label: "MacBook", Statuses: []connectors.Status{
		st("whatsapp", connectors.StateNotConfigured, "ChatStorage.sqlite not found."),
	}}))
	must(t, r.Accept(ctx, Payload{Device: "mini", Label: "Mac mini", Statuses: []connectors.Status{
		st("whatsapp", connectors.StateConnected, "12 chats · 3 unread"),
	}}))
	*now = now.Add(10 * time.Minute)
	s := r.Connector("whatsapp").Status(ctx)
	if s.State != connectors.StateConnected || s.Detail != "12 chats · 3 unread · pushed by Mac mini 10m ago" {
		t.Fatalf("status = %+v", s)
	}
	if s.Meta["device"] != "mini" || s.Kind != connectors.KindSocial {
		t.Errorf("status = %+v", s)
	}
}

func TestSourceStatusStalePushIsAnErrorNotConnected(t *testing.T) {
	r, now := receiver()
	ctx := context.Background()
	must(t, r.Accept(ctx, Payload{Device: "macbook", Label: "MacBook", Statuses: []connectors.Status{
		st("wispr", connectors.StateConnected, "9,000 dictations"),
	}}))
	*now = now.Add(50 * time.Hour)
	s := r.Connector("wispr").Status(ctx)
	if s.State != connectors.StateError || !strings.Contains(s.Detail, "stale") || !strings.Contains(s.Detail, "MacBook") || !strings.Contains(s.Detail, "2d ago") {
		t.Fatalf("status = %+v", s)
	}
}

func TestReadingsCarryDeviceAndStaleness(t *testing.T) {
	r, now := receiver()
	ctx := context.Background()
	if _, err := r.WhatsAppChats(ctx); !errors.Is(err, ErrNoPush) {
		t.Fatalf("err = %v", err)
	}
	must(t, r.Accept(ctx, Payload{Device: "mini", Label: "Mac mini",
		WhatsApp: &WhatsAppData{Chats: []Chat{{Source: "whatsapp", Title: "Ana", Preview: "hi", TS: "2026-09-29T11:00:00Z"}}},
		Wispr:    &WisprData{Notes: []WisprNote{{ID: "h1", Kind: "dictation", Text: "x", TS: "2026-09-29T11:00:00Z"}}},
		Obsidian: &ObsidianData{Notes: []VaultNote{{Path: "a.md", Content: "# a"}}},
		Ollama:   &OllamaSnapshot{Kind: "ollama", ID: "ollama-mini", Label: "Ollama · mini", CapturedAt: "2026-09-29T12:00:00Z", Lane: OllamaLane{State: "up", Models: []OllamaModel{}}},
	}))
	chats, err := r.WhatsAppChats(ctx)
	if err != nil || chats.Stale || chats.Device != "mini" || len(chats.Data) != 1 {
		t.Fatalf("chats = %+v, %v", chats, err)
	}
	*now = now.Add(25 * time.Hour)
	notes, _ := r.WisprNotes(ctx)
	vault, _ := r.VaultNotes(ctx)
	ollama, _ := r.OllamaSnapshots(ctx)
	if !notes.Stale || !vault.Stale || len(ollama) != 1 || !ollama[0].Stale {
		t.Fatalf("stale readings must say so: %+v %+v %+v", notes, vault, ollama)
	}
}

type brokenStore struct{}

func (brokenStore) Save(context.Context, Received) error { return errors.New("disk full") }
func (brokenStore) Latest(context.Context) ([]Received, error) {
	return nil, errors.New("connection refused")
}

func TestUnreadableStoreIsAnError(t *testing.T) {
	r := NewReceiver(brokenStore{})
	if s := r.Connector("obsidian").Status(context.Background()); s.State != connectors.StateError || !strings.Contains(s.Detail, "connection refused") {
		t.Fatalf("status = %+v", s)
	}
	if err := r.Accept(context.Background(), Payload{Device: "mini"}); err == nil {
		t.Fatal("a failed save must be reported")
	}
}

// The backend never reads device paths: this package must not touch the
// filesystem or the home directory.
func TestBackendPackageHasNoFilesystemReads(t *testing.T) {
	files, _ := filepath.Glob("*.go")
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		parsed, err := parser.ParseFile(token.NewFileSet(), f, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatal(err)
		}
		for _, imp := range parsed.Imports {
			switch strings.Trim(imp.Path.Value, `"`) {
			case "os", "path/filepath", "io/fs", "database/sql":
				t.Errorf("%s imports %s", f, imp.Path.Value)
			}
		}
		raw, _ := os.ReadFile(f)
		if strings.Contains(string(raw), "UserHomeDir") {
			t.Errorf("%s resolves a home directory", f)
		}
	}
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

// validSeat matches lib/usage.ts UsageSnapshotSchema: official gauges need a
// non-negative usedPercent and a positive integer window, and every top
// burner a known lane and a non-negative burn.
func TestSeatValidationMatchesTheZodSchema(t *testing.T) {
	base := func() SeatUsage {
		return SeatUsage{ID: "claude-mac", Kind: "claude", Label: "Claude", CapturedAt: "2026-09-30T09:00:00Z"}
	}
	if err := validSeat(base()); err != nil {
		t.Fatalf("a minimal seat must pass: %v", err)
	}
	bad := map[string]func(*SeatUsage){
		"negative usedPercent": func(s *SeatUsage) {
			s.Official = &Official{Session: &OfficialWindow{UsedPercent: -1, WindowMinutes: 300}}
		},
		"zero windowMinutes": func(s *SeatUsage) { s.Official = &Official{Weekly: &OfficialWindow{UsedPercent: 10, WindowMinutes: 0}} },
		"negative burn": func(s *SeatUsage) {
			s.Breakdown = &Breakdown{Top: []TopBurner{{Source: "board", Label: "x", Burn: -5}}}
		},
		"unknown lane": func(s *SeatUsage) { s.Breakdown = &Breakdown{Top: []TopBurner{{Source: "moon", Label: "x", Burn: 5}}} },
	}
	for name, mut := range bad {
		s := base()
		mut(&s)
		if err := validSeat(s); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
}

// The local stack is the HOST's (FounderOS v1 checked the machine it ran on):
// a healthy MacBook must not stand in for a mini whose services are down.
func TestHostConnectorReadsOnlyTheHostDevice(t *testing.T) {
	r, _ := receiver()
	r.Host = "mini"
	ctx := context.Background()
	must(t, r.Accept(ctx, Payload{Device: "macbook", Label: "MacBook", Statuses: []connectors.Status{
		st(SourceLocalStack, connectors.StateConnected, "9/9 up"),
	}}))
	if s := r.HostConnector(SourceLocalStack).Status(ctx); s.State != connectors.StateError || !strings.Contains(s.Detail, "mini") {
		t.Fatalf("a host that never pushed must be an error naming it: %+v", s)
	}
	must(t, r.Accept(ctx, Payload{Device: "mini", Label: "Mac mini", Statuses: []connectors.Status{
		st(SourceLocalStack, connectors.StateError, "ollama down"),
	}}))
	s := r.HostConnector(SourceLocalStack).Status(ctx)
	if s.State != connectors.StateError || !strings.Contains(s.Detail, "ollama down") || s.Meta["device"] != "mini" {
		t.Fatalf("the host's own reading must win over a healthier Mac: %+v", s)
	}
}

func TestHostDeviceSlugMatchesTheCollector(t *testing.T) {
	if got := HostDeviceSlug("Alexs-Mac-mini.local"); got != "alexs-mac-mini" {
		t.Fatalf("slug = %q", got)
	}
	r, _ := receiver()
	if s := r.HostConnector(SourceLocalStack).Status(context.Background()); s.State != connectors.StateError || !strings.Contains(s.Detail, "FOUNDEROS_HOST_DEVICE") {
		t.Fatalf("an unset host must be an error, never best-of: %+v", s)
	}
}

// FounderOS v1 read the box it ran on. A fresh push from the host device wins
// for every device-local source, even when another Mac's reading is
// healthier or newer; other Macs stand in only when the host has none fresh.
func TestConnectorPrefersAFreshHostReading(t *testing.T) {
	r, now := receiver()
	r.Host = "mini"
	ctx := context.Background()
	must(t, r.Accept(ctx, Payload{Device: "mini", Label: "Mac mini", Statuses: []connectors.Status{
		st(SourceWispr, connectors.StateConnected, "50 dictations"),
	}}))
	*now = now.Add(5 * time.Minute)
	must(t, r.Accept(ctx, Payload{Device: "macbook", Label: "MacBook", Statuses: []connectors.Status{
		st(SourceWispr, connectors.StateConnected, "100 dictations"),
		st(SourceObsidian, connectors.StateConnected, "9 notes"),
	}}))
	if s := r.Connector(SourceWispr).Status(ctx); s.Meta["device"] != "mini" || !strings.Contains(s.Detail, "50 dictations") {
		t.Fatalf("the host's fresh reading must win over a newer one from another Mac: %+v", s)
	}
	if s := r.Connector(SourceObsidian).Status(ctx); s.Meta["device"] != "macbook" || s.State != connectors.StateConnected {
		t.Fatalf("a source the host never pushed falls back to the best other Mac: %+v", s)
	}

	// the host's own error is still the host's reading
	must(t, r.Accept(ctx, Payload{Device: "mini", Label: "Mac mini", Statuses: []connectors.Status{
		st(SourceWispr, connectors.StateError, "read timed out"),
	}}))
	if s := r.Connector(SourceWispr).Status(ctx); s.Meta["device"] != "mini" || s.State != connectors.StateError {
		t.Fatalf("a healthier Mac must not stand in for the host's fresh reading: %+v", s)
	}

	// once the host's push is stale, the freshest other reading answers
	*now = now.Add(StaleAfter + time.Minute)
	must(t, r.Accept(ctx, Payload{Device: "macbook", Label: "MacBook", Statuses: []connectors.Status{
		st(SourceWispr, connectors.StateConnected, "101 dictations"),
	}}))
	if s := r.Connector(SourceWispr).Status(ctx); s.Meta["device"] != "macbook" || s.State != connectors.StateConnected {
		t.Fatalf("a stale host falls back to a fresh Mac: %+v", s)
	}
}
