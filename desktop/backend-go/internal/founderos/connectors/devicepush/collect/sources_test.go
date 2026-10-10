package collect

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/rhl/businessos-backend/internal/founderos/connectors"
)

var now = time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC)

func fixtureDB(t *testing.T, path string, stmts ...string) string {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	for _, s := range stmts {
		if _, err := db.Exec(s); err != nil {
			t.Fatalf("%s: %v", s, err)
		}
	}
	return path
}

func touch(t *testing.T, path string, at time.Time) {
	t.Helper()
	if err := os.Chtimes(path, at, at); err != nil {
		t.Fatal(err)
	}
}

// ---- WhatsApp -------------------------------------------------------------

// Core Data seconds since 2001-01-01 for a UTC time.
func cd(at time.Time) float64 { return at.Sub(coreDataEpoch).Seconds() }

func chatDB(t *testing.T, dir string) string {
	t.Helper()
	recent := cd(now.Add(-10 * time.Minute))
	older := cd(now.Add(-3 * time.Hour))
	return fixtureDB(t, filepath.Join(dir, "group.net.whatsapp.WhatsAppSMB.shared", "ChatStorage.sqlite"),
		`CREATE TABLE ZWACHATSESSION (Z_PK INTEGER PRIMARY KEY, ZPARTNERNAME TEXT, ZCONTACTJID TEXT, ZLASTMESSAGEDATE REAL, ZUNREADCOUNT INTEGER, ZARCHIVED INTEGER)`,
		`CREATE TABLE ZWAMESSAGE (Z_PK INTEGER PRIMARY KEY, ZCHATSESSION INTEGER, ZTEXT TEXT, ZMESSAGEDATE REAL)`,
		`INSERT INTO ZWACHATSESSION VALUES (1, 'Ana', '447700900000@s.whatsapp.net', `+ftoa(recent)+`, 2, 0)`,
		`INSERT INTO ZWACHATSESSION VALUES (2, 'Family group', '120363@g.us', `+ftoa(older)+`, 1, NULL)`,
		`INSERT INTO ZWACHATSESSION VALUES (3, 'Archived', '1@s.whatsapp.net', `+ftoa(recent)+`, 5, 1)`,
		`INSERT INTO ZWACHATSESSION VALUES (4, NULL, 'status@broadcast', `+ftoa(recent)+`, 0, 0)`,
		`INSERT INTO ZWAMESSAGE VALUES (1, 1, 'first', `+ftoa(recent-60)+`)`,
		`INSERT INTO ZWAMESSAGE VALUES (2, 1, 'latest from Ana', `+ftoa(recent)+`)`,
		`INSERT INTO ZWAMESSAGE VALUES (3, 1, NULL, `+ftoa(recent+1)+`)`,
	)
}

func TestResolveChatDBPrefersTheNewestExisting(t *testing.T) {
	dir := t.TempDir()
	if got := ResolveChatDB([]string{filepath.Join(dir, "nope")}); got != "" {
		t.Fatalf("got %q", got)
	}
	a, b := filepath.Join(dir, "a.sqlite"), filepath.Join(dir, "b.sqlite")
	for _, p := range []string{a, b} {
		_ = os.WriteFile(p, []byte("x"), 0o600)
	}
	touch(t, a, now.Add(-time.Hour))
	touch(t, b, now)
	if got := ResolveChatDB([]string{a, b}); got != b {
		t.Errorf("got %q", got)
	}
	if got := ResolveChatDB([]string{b, a}); got != b {
		t.Errorf("got %q", got)
	}
	paths := WhatsAppContainerPaths("/home/x")
	if len(paths) != 2 || !strings.Contains(paths[0], "group.net.whatsapp.WhatsApp.shared") || !strings.Contains(paths[1], "WhatsAppSMB") {
		t.Errorf("paths = %v", paths)
	}
}

func TestWhatsAppStatusStates(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	s := WhatsAppStatus(ctx, []string{filepath.Join(dir, "missing.sqlite")}, now)
	if s.State != connectors.StateNotConfigured || !strings.Contains(s.Detail, "ChatStorage.sqlite not found") {
		t.Fatalf("status = %+v", s)
	}

	db := chatDB(t, dir)
	touch(t, db, now.Add(-7*time.Minute))
	s = WhatsAppStatus(ctx, []string{db}, now)
	if s.State != connectors.StateConnected {
		t.Fatalf("status = %+v", s)
	}
	if s.Detail != "4 chats · 8 unread · last activity 7m ago (local desktop DB, read-only)" {
		t.Errorf("detail = %q", s.Detail)
	}
	if s.Meta["source"] != "business" || s.Meta["chats"] != 4 || s.Meta["unread"] != 8 {
		t.Errorf("meta = %v", s.Meta)
	}

	junk := filepath.Join(dir, "junk", "ChatStorage.sqlite")
	_ = os.MkdirAll(filepath.Dir(junk), 0o755)
	_ = os.WriteFile(junk, []byte("this is not a database at all, not even close......"), 0o600)
	s = WhatsAppStatus(ctx, []string{junk}, now)
	if s.State != connectors.StateError || !strings.Contains(s.Detail, "exists but read failed") {
		t.Fatalf("status = %+v", s)
	}
}

func TestRecentChatsMapsRowsNewestFirst(t *testing.T) {
	db := chatDB(t, t.TempDir())
	chats, err := RecentChats(context.Background(), db, 40, now)
	if err != nil {
		t.Fatal(err)
	}
	if len(chats) != 2 {
		t.Fatalf("archived and nameless chats must be skipped: %+v", chats)
	}
	ana := chats[0]
	if ana.Title != "Ana" || ana.Sender != "Ana" || ana.ReplyTo != "447700900000" || ana.Preview != "latest from Ana" || ana.Unread != 2 || ana.Source != "whatsapp" {
		t.Errorf("ana = %+v", ana)
	}
	// lastDate is the newest message (recent+1s), in plain Core Data seconds
	if ana.TS != "2026-09-29T11:50:01.000Z" {
		t.Errorf("ts = %s", ana.TS)
	}
	if chats[1].ReplyTo != "" {
		t.Errorf("a group JID has no dialable number: %+v", chats[1])
	}
	if one, _ := RecentChats(context.Background(), db, 1, now); len(one) != 1 {
		t.Errorf("limit ignored: %d", len(one))
	}
}

func TestCalibratedDate(t *testing.T) {
	// ticks at 1000/s: the max raw value maps to "now"
	elapsed := cd(now)
	got := CalibratedDate(elapsed*1000, elapsed*1000, now)
	if !got.Equal(now) {
		t.Errorf("got %v", got)
	}
	if got := CalibratedDate(0, 5, now); !got.Equal(coreDataEpoch) {
		t.Errorf("zero raw = %v", got)
	}
}

// ---- Wispr ---------------------------------------------------------------

func flowDB(t *testing.T, dir string) string {
	t.Helper()
	return fixtureDB(t, filepath.Join(dir, "flow.sqlite"),
		`CREATE TABLE History (transcriptEntityId TEXT, asrText TEXT, formattedText TEXT, editedText TEXT, timestamp TEXT, app TEXT, numWords INTEGER, isArchived INTEGER)`,
		`CREATE TABLE Notes (id TEXT, title TEXT, contentPreview TEXT, createdAt TEXT, modifiedAt TEXT, isDeleted INTEGER)`,
		`CREATE TABLE Todos (id TEXT, title TEXT, status TEXT, createdAt TEXT, isDeleted INTEGER, isArchived INTEGER)`,
		`INSERT INTO History VALUES ('h1','raw one','fmt one','edited one','2026-07-27T12:00:00.000Z','Mail',2,0)`,
		`INSERT INTO History VALUES ('h2','raw two','fmt two',NULL,'2026-07-27 09:00:00.500 +00:00','Slack',2,0)`,
		`INSERT INTO History VALUES ('h3','raw three',NULL,NULL,'2026-07-26T09:00:00.000Z','',2,0)`,
		`INSERT INTO History VALUES ('h4','archived','archived','archived','2026-07-27T13:00:00.000Z','Mail',1,1)`,
		`INSERT INTO History VALUES ('h5','','','','2026-07-27T13:30:00.000Z','Mail',0,0)`,
		`INSERT INTO Notes VALUES ('n1','A real note','preview','2026-07-25T08:00:00.000Z','2026-07-27T11:00:00.000Z',0)`,
		`INSERT INTO Notes VALUES ('n2','Deleted note','x','2026-07-25T08:00:00.000Z','2026-07-27T11:30:00.000Z',1)`,
		`INSERT INTO Todos VALUES ('t1','Ship the board','open','2026-07-27T10:00:00.000Z',0,0)`,
		`INSERT INTO Todos VALUES ('t2','Archived todo','done','2026-07-27T10:00:00.000Z',0,1)`,
		`INSERT INTO Todos VALUES ('t3','Call Ana','snoozed','2026-07-26T10:00:00.000Z',0,0)`,
	)
}

func TestWisprDetail(t *testing.T) {
	cases := map[float64]string{
		7:    "12,345 dictations · 1 notes · 2 todos · 0 meetings · last activity 7m ago",
		180:  "12,345 dictations · 1 notes · 2 todos · 0 meetings · last activity 3h ago",
		1440: "12,345 dictations · 1 notes · 2 todos · 0 meetings · last activity 1d ago · stale on this box",
	}
	for mins, want := range cases {
		if got := WisprDetail(12345, 1, 2, 0, mins); got != want {
			t.Errorf("%v: %q", mins, got)
		}
	}
}

func TestWisprStatusStates(t *testing.T) {
	dir := t.TempDir()
	if s := WisprStatus(context.Background(), filepath.Join(dir, "flow.sqlite"), now); s.State != connectors.StateNotConfigured || !strings.Contains(s.Detail, "flow.sqlite not found") {
		t.Fatalf("status = %+v", s)
	}
	db := flowDB(t, dir)
	touch(t, db, now.Add(-5*time.Minute))
	s := WisprStatus(context.Background(), db, now)
	if s.State != connectors.StateConnected || s.Detail != "5 dictations · 2 notes · 3 todos · 0 meetings · last activity 5m ago" {
		t.Fatalf("status = %+v", s)
	}
	if s.Meta["dictations"] != 5 || s.Meta["meetings"] != 0 {
		t.Errorf("meta = %v", s.Meta)
	}
	junk := filepath.Join(dir, "junk.sqlite")
	_ = os.WriteFile(junk, []byte("definitely not sqlite, just some text in a file......"), 0o600)
	if s := WisprStatus(context.Background(), junk, now); s.State != connectors.StateError || !strings.Contains(s.Detail, "flow.sqlite exists but read failed") {
		t.Fatalf("junk status = %+v", s)
	}
}

func TestRecentWisprNotes(t *testing.T) {
	notes := RecentWisprNotes(context.Background(), flowDB(t, t.TempDir()), 40)
	got := []string{}
	for _, n := range notes {
		got = append(got, n.ID+":"+n.Kind+":"+n.Text)
	}
	want := "h1:dictation:edited one n1:note:A real note t1:todo:Ship the board h2:dictation:fmt two t3:todo:Call Ana · snoozed h3:dictation:raw three"
	if strings.Join(got, " ") != want {
		t.Fatalf("notes = %v", got)
	}
	if notes[0].App == nil || *notes[0].App != "Mail" || notes[0].WordCount == nil || *notes[0].WordCount != 2 || notes[0].TS != "2026-07-27T12:00:00.000Z" {
		t.Errorf("h1 = %+v", notes[0])
	}
	if notes[3].TS != "2026-07-27T09:00:00.500Z" {
		t.Errorf("Wispr's own DATETIME format must parse: %s", notes[3].TS)
	}
	if notes[5].App != nil {
		t.Errorf("empty app must be nil: %+v", notes[5])
	}
	if len(RecentWisprNotes(context.Background(), filepath.Join(t.TempDir(), "absent.sqlite"), 40)) != 0 {
		t.Error("absent db must be empty")
	}
	if n := RecentWisprNotes(context.Background(), flowDB(t, t.TempDir()), 2); len(n) != 2 {
		t.Errorf("limit = %d", len(n))
	}
}

func TestRecentWisprNotesSurvivesAnOlderSchema(t *testing.T) {
	db := fixtureDB(t, filepath.Join(t.TempDir(), "flow.sqlite"),
		`CREATE TABLE History (transcriptEntityId TEXT, asrText TEXT, timestamp TEXT)`,
		`CREATE TABLE Todos (id TEXT, title TEXT, status TEXT, createdAt TEXT, isDeleted INTEGER, isArchived INTEGER)`,
		`INSERT INTO Todos VALUES ('t1','Only todos','open','2026-07-27T10:00:00.000Z',0,0)`,
	)
	notes := RecentWisprNotes(context.Background(), db, 40)
	if len(notes) != 1 || notes[0].ID != "t1" {
		t.Fatalf("notes = %+v", notes)
	}
}

// ---- Obsidian -------------------------------------------------------------

func vault(t *testing.T) (home, dir string) {
	t.Helper()
	home = t.TempDir()
	dir = filepath.Join(home, "Documents", "Obsidian Vault")
	files := map[string]string{
		"b.md":                       "# b",
		"a.md":                       "# a",
		"Claude Archive/chat-1.md":   strings.Repeat("x", 25_000),
		"Claude Archive/image.png":   "png",
		".obsidian/workspace.md":     "hidden",
		"Projects/.trash/deleted.md": "hidden",
	}
	for rel, body := range files {
		p := filepath.Join(dir, rel)
		_ = os.MkdirAll(filepath.Dir(p), 0o755)
		if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return home, dir
}

func TestObsidianStatus(t *testing.T) {
	home, dir := vault(t)
	s := ObsidianStatus(dir, home)
	if s.State != connectors.StateConnected || s.Detail != "3 markdown notes (incl. Claude Archive) at ~/Documents/Obsidian Vault" || s.Meta["notes"] != 3 {
		t.Fatalf("status = %+v", s)
	}
	if s := ObsidianStatus(filepath.Join(home, "nope"), home); s.State != connectors.StateNotConfigured || !strings.Contains(s.Detail, "OBSIDIAN_VAULT") {
		t.Fatalf("status = %+v", s)
	}
	if os.Getuid() != 0 {
		locked := filepath.Join(home, "locked")
		_ = os.Mkdir(locked, 0o000)
		defer os.Chmod(locked, 0o755)
		if s := ObsidianStatus(locked, home); s.State != connectors.StateError || !strings.Contains(s.Detail, "denied access") {
			t.Fatalf("status = %+v", s)
		}
	}
}

func TestReadVaultNotes(t *testing.T) {
	_, dir := vault(t)
	notes := ReadVaultNotes(dir)
	paths := []string{}
	for _, n := range notes {
		paths = append(paths, n.Path)
	}
	if strings.Join(paths, ",") != "a.md,b.md,Claude Archive/chat-1.md" {
		t.Fatalf("paths = %v", paths)
	}
	if len(notes[2].Content) != noteContentCap {
		t.Errorf("content cap = %d", len(notes[2].Content))
	}
	if ReadVaultNotes(filepath.Join(dir, "absent")) == nil {
		t.Error("a missing vault is an empty list, never nil")
	}
}

// ---- local stack ------------------------------------------------------------

func TestLocalStackStatus(t *testing.T) {
	board := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(401) }))
	defer board.Close()
	home, brew, usrLocal := t.TempDir(), t.TempDir(), t.TempDir()
	for _, bin := range []string{filepath.Join(brew, "ffmpeg"), filepath.Join(usrLocal, "gh"), filepath.Join(brew, "pdftotext")} {
		_ = os.WriteFile(bin, []byte("#!/bin/sh\n"), 0o755)
	}
	_ = os.WriteFile(filepath.Join(brew, "whisper-cli"), []byte("not executable"), 0o644)

	cfg := LocalStackConfig{
		PaperclipURL: board.URL, HermesURL: "http://127.0.0.1:1",
		Home: home, BrewDir: brew, UsrLocalBin: usrLocal,
	}
	s := LocalStackStatus(context.Background(), cfg)
	host := strings.TrimPrefix(board.URL, "http://")
	if s.State != connectors.StateConnected || s.Detail != "4/6 up — paperclip, ffmpeg, pdftotext, gh · down: hermes, whisper" {
		t.Fatalf("status = %+v", s)
	}
	if s.Meta["paperclip"] != "up · agent board · "+host || s.Meta["hermes"] != "down" {
		t.Errorf("meta = %v", s.Meta)
	}

	cfg.PaperclipURL = "http://127.0.0.1:1"
	cfg.BrewDir, cfg.UsrLocalBin = t.TempDir(), t.TempDir()
	if s := LocalStackStatus(context.Background(), cfg); s.State != connectors.StateError || !strings.HasPrefix(s.Detail, "0/6 up") {
		t.Fatalf("status = %+v", s)
	}
}

func ftoa(f float64) string { return strconv.FormatFloat(f, 'f', -1, 64) }
