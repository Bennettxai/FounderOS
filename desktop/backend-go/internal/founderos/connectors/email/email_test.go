package email

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapserver"
	"github.com/emersion/go-imap/v2/imapserver/imapmemserver"
	"github.com/emersion/go-sasl"
	"github.com/emersion/go-smtp"

	"github.com/rhl/businessos-backend/internal/founderos/connectors"
	"github.com/rhl/businessos-backend/internal/founderos/guard"
)

// ---- fixtures ---------------------------------------------------------------

// fakeIMAP is an in-process IMAP server (go-imap's imapmemserver) speaking the
// real protocol on a loopback port, so the whole client path is exercised with
// no network.
type fakeIMAP struct {
	mem  *imapmemserver.Server
	ln   net.Listener
	port int
}

func startIMAP(t *testing.T) *fakeIMAP {
	t.Helper()
	mem := imapmemserver.New()
	srv := imapserver.New(&imapserver.Options{
		NewSession: func(*imapserver.Conn) (imapserver.Session, *imapserver.GreetingData, error) {
			return mem.NewSession(), nil, nil
		},
		Caps:         imap.CapSet{imap.CapIMAP4rev1: {}, imap.CapIMAP4rev2: {}},
		InsecureAuth: true,
		Logger:       quietLogger{},
	})
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	go srv.Serve(ln)
	t.Cleanup(func() { srv.Close() })
	return &fakeIMAP{mem: mem, ln: ln, port: ln.Addr().(*net.TCPAddr).Port}
}

type quietLogger struct{}

func (quietLogger) Printf(string, ...interface{}) {}

type literal struct{ *bytes.Reader }

func (l literal) Size() int64 { return l.Reader.Size() }

type msg struct {
	fromName, fromAddr, subject string
	date                        time.Time
	seen                        bool
}

func (f *fakeIMAP) addUser(t *testing.T, user, pass string, msgs ...msg) *imapmemserver.User {
	t.Helper()
	u := imapmemserver.NewUser(user, pass)
	if err := u.Create("INBOX", nil); err != nil {
		t.Fatal(err)
	}
	for _, m := range msgs {
		from := m.fromAddr
		if m.fromName != "" {
			from = fmt.Sprintf("%q <%s>", m.fromName, m.fromAddr)
		}
		raw := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nDate: %s\r\nMessage-ID: <%d@test>\r\n\r\nbody\r\n",
			from, user, m.subject, m.date.Format(time.RFC1123Z), m.date.UnixNano())
		var flags []imap.Flag
		if m.seen {
			flags = []imap.Flag{imap.FlagSeen}
		}
		if _, err := u.Append("INBOX", literal{bytes.NewReader([]byte(raw))}, &imap.AppendOptions{Flags: flags, Time: m.date}); err != nil {
			t.Fatal(err)
		}
	}
	f.mem.AddUser(u)
	return u
}

func unseen(t *testing.T, u *imapmemserver.User) uint32 {
	t.Helper()
	st, err := u.Status("INBOX", &imap.StatusOptions{NumUnseen: true})
	if err != nil {
		t.Fatal(err)
	}
	return *st.NumUnseen
}

// writeEnv plants an env.local for the resolver and clears the process env
// for every key the connector reads, so the developer's shell cannot leak in.
func writeEnv(t *testing.T, kv map[string]string) connectors.Resolver {
	t.Helper()
	for slot := 1; slot <= MaxInboxes; slot++ {
		for _, k := range []string{"HOST", "USER", "PASS", "NAME", "PORT", "SMTP_HOST", "SMTP_PORT"} {
			t.Setenv(fmt.Sprintf("INBOX_%d_%s", slot, k), "")
		}
	}
	t.Setenv("MAIL_ALLOW_EXTERNAL", "")
	t.Setenv("MAIL_ALLOW_SYSTEM_FROM", "")
	var b strings.Builder
	for k, v := range kv {
		fmt.Fprintf(&b, "%s=%s\n", k, v)
	}
	p := filepath.Join(t.TempDir(), "env.local")
	if err := os.WriteFile(p, []byte(b.String()), 0o600); err != nil {
		t.Fatal(err)
	}
	return connectors.Resolver{EnvLocal: p}
}

func slotEnv(kv map[string]string, slot int, user, pass string, port int) {
	kv[fmt.Sprintf("INBOX_%d_HOST", slot)] = "127.0.0.1"
	kv[fmt.Sprintf("INBOX_%d_USER", slot)] = user
	kv[fmt.Sprintf("INBOX_%d_PASS", slot)] = pass
	kv[fmt.Sprintf("INBOX_%d_PORT", slot)] = strconv.Itoa(port)
}

// testConnector dials plain TCP instead of implicit TLS; everything above the
// socket is the production path.
func testConnector(res connectors.Resolver) *Connector {
	c := New(res)
	c.dialIMAP = func(ctx context.Context, cfg InboxConfig) (net.Conn, error) {
		var d net.Dialer
		return d.DialContext(ctx, "tcp", net.JoinHostPort(cfg.Host, strconv.Itoa(cfg.Port)))
	}
	return c
}

var t0 = time.Date(2026, 9, 20, 12, 0, 0, 0, time.UTC)

// ---- config -----------------------------------------------------------------

func TestInboxesParsesSlotsWithDefaults(t *testing.T) {
	res := writeEnv(t, map[string]string{
		"INBOX_1_HOST": "imap.gmail.com", "INBOX_1_USER": "one@x.com", "INBOX_1_PASS": "p1", "INBOX_1_NAME": "One",
		"INBOX_2_HOST": "imap.gmail.com", "INBOX_2_USER": "two@x.com", // incomplete: no pass
		"INBOX_3_HOST": "mail.example.com", "INBOX_3_USER": "three@x.com", "INBOX_3_PASS": "p3",
		"INBOX_3_PORT": "143", "INBOX_3_SMTP_HOST": "out.example.com", "INBOX_3_SMTP_PORT": "587",
	})
	got := New(res).Inboxes()
	if len(got) != 2 {
		t.Fatalf("want 2 inboxes, got %d: %+v", len(got), got)
	}
	a, b := got[0], got[1]
	if a.ID != "inbox-1" || a.Name != "One" || a.Port != 993 || a.SMTPHost != "smtp.gmail.com" || a.SMTPPort != 465 {
		t.Errorf("slot 1 defaults wrong: %+v", a)
	}
	if b.ID != "inbox-3" || b.Name != "three@x.com" || b.Port != 143 || b.SMTPHost != "out.example.com" || b.SMTPPort != 587 {
		t.Errorf("slot 3 overrides wrong: %+v", b)
	}
}

// ---- status -----------------------------------------------------------------

func TestStatusNotConfigured(t *testing.T) {
	s := New(writeEnv(t, nil)).Status(context.Background())
	if s.State != connectors.StateNotConfigured {
		t.Fatalf("state = %s", s.State)
	}
	if s.ID != "email" || s.Name != "Email Inboxes" || s.Kind != connectors.KindEmail {
		t.Errorf("identity wrong: %+v", s)
	}
	if s.Detail != "No inboxes configured. Set INBOX_1_HOST / _USER / _PASS (up to 4 slots) in ~/.founderos/.env or under API keys." {
		t.Errorf("detail = %q", s.Detail)
	}
	if s.Meta["configured"] != 0 || s.Meta["slots"] != MaxInboxes {
		t.Errorf("meta = %v", s.Meta)
	}
}

func TestStatusConnectedCountsUnread(t *testing.T) {
	srv := startIMAP(t)
	srv.addUser(t, "a@x.com", "pa",
		msg{fromAddr: "p@q.com", subject: "one", date: t0},
		msg{fromAddr: "p@q.com", subject: "two", date: t0, seen: true},
		msg{fromAddr: "p@q.com", subject: "three", date: t0})
	srv.addUser(t, "b@x.com", "pb", msg{fromAddr: "p@q.com", subject: "x", date: t0})
	kv := map[string]string{}
	slotEnv(kv, 1, "a@x.com", "pa", srv.port)
	slotEnv(kv, 2, "b@x.com", "pb", srv.port)
	s := testConnector(writeEnv(t, kv)).Status(context.Background())
	if s.State != connectors.StateConnected {
		t.Fatalf("state = %s (%s)", s.State, s.Detail)
	}
	if s.Detail != "2 inboxes · 3 unread" {
		t.Errorf("detail = %q", s.Detail)
	}
	if s.Meta["configured"] != 2 || s.Meta["unread"] != 3 {
		t.Errorf("meta = %v", s.Meta)
	}
}

func TestStatusPartialFailureStaysConnectedAndSaysSo(t *testing.T) {
	srv := startIMAP(t)
	srv.addUser(t, "a@x.com", "pa", msg{fromAddr: "p@q.com", subject: "one", date: t0})
	srv.addUser(t, "b@x.com", "pb")
	kv := map[string]string{}
	slotEnv(kv, 1, "a@x.com", "pa", srv.port)
	slotEnv(kv, 2, "b@x.com", "WRONG", srv.port)
	c := testConnector(writeEnv(t, kv))
	s := c.Status(context.Background())
	if s.State != connectors.StateConnected || s.Detail != "2 inboxes · 1 unread · 1 failing" {
		t.Fatalf("got %s %q", s.State, s.Detail)
	}
	counts, err := c.UnreadCounts(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if counts[1].Error == "" || counts[1].Unread != nil {
		t.Errorf("a failing inbox must read unknown, not 0: %+v", counts[1])
	}
	if counts[0].Unread == nil || *counts[0].Unread != 1 {
		t.Errorf("answered inbox: %+v", counts[0])
	}
}

func TestStatusErrorWhenEveryInboxFails(t *testing.T) {
	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	port := ln.Addr().(*net.TCPAddr).Port
	ln.Close() // nothing listens: connection refused
	kv := map[string]string{}
	slotEnv(kv, 1, "a@x.com", "pa", port)
	s := testConnector(writeEnv(t, kv)).Status(context.Background())
	if s.State != connectors.StateError {
		t.Fatalf("state = %s", s.State)
	}
	if !strings.HasPrefix(s.Detail, "All 1 inbox connections failed: ") {
		t.Errorf("detail = %q", s.Detail)
	}
}

func TestUnreadCountsCachedOnlyWhenSomethingAnswered(t *testing.T) {
	srv := startIMAP(t)
	u := srv.addUser(t, "a@x.com", "pa", msg{fromAddr: "p@q.com", subject: "one", date: t0})
	kv := map[string]string{}
	slotEnv(kv, 1, "a@x.com", "pa", srv.port)
	c := testConnector(writeEnv(t, kv))
	now := t0
	c.now = func() time.Time { return now }
	first, _ := c.UnreadCounts(context.Background())
	u.Append("INBOX", literal{bytes.NewReader([]byte("Subject: new\r\n\r\nx\r\n"))}, &imap.AppendOptions{})
	second, _ := c.UnreadCounts(context.Background())
	if *second[0].Unread != *first[0].Unread {
		t.Errorf("inside the window the cached count must be served")
	}
	now = now.Add(CacheTTL + time.Second)
	third, _ := c.UnreadCounts(context.Background())
	if *third[0].Unread != 2 {
		t.Errorf("after the window a fresh count is read, got %d", *third[0].Unread)
	}
}

// ---- feed -------------------------------------------------------------------

func TestLatestEmailsReadsTheNewestEnvelopesPerInbox(t *testing.T) {
	srv := startIMAP(t)
	var msgs []msg
	for i := 1; i <= 5; i++ {
		msgs = append(msgs, msg{fromName: "Dana Reyes", fromAddr: "dana@reyes.co", subject: fmt.Sprintf("m%d", i), date: t0.Add(time.Duration(i) * time.Hour), seen: i == 5})
	}
	u := srv.addUser(t, "a@x.com", "pa", msgs...)
	srv.addUser(t, "b@x.com", "pb", msg{fromAddr: "bare@z.com", subject: "", date: t0})
	kv := map[string]string{"INBOX_1_NAME": "Agency"}
	slotEnv(kv, 1, "a@x.com", "pa", srv.port)
	slotEnv(kv, 2, "b@x.com", "pb", srv.port)
	c := testConnector(writeEnv(t, kv))

	before := unseen(t, u)
	items, err := c.LatestEmails(context.Background(), 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 4 {
		t.Fatalf("want 3 from inbox-1 + 1 from inbox-2, got %d: %+v", len(items), items)
	}
	var subjects []string
	for _, it := range items[:3] {
		subjects = append(subjects, it.Preview)
	}
	if strings.Join(subjects, ",") != "m3,m4,m5" {
		t.Errorf("paged window = %v, want the newest 3", subjects)
	}
	m5 := items[2]
	want := CommsItem{
		Source: "email", Title: "Agency — Dana Reyes", Sender: "Dana Reyes", ReplyTo: "dana@reyes.co",
		Account: "inbox-1", Preview: "m5", TS: "2026-09-20T17:00:00.000Z", Unread: 0,
	}
	if m5 != want {
		t.Errorf("item = %+v\nwant   %+v", m5, want)
	}
	if items[0].Unread != 1 {
		t.Errorf("unseen message must read unread=1")
	}
	bare := items[3]
	if bare.Sender != "bare@z.com" || bare.Title != "b@x.com — bare@z.com" || bare.Preview != "(no subject)" || bare.Account != "inbox-2" {
		t.Errorf("fallbacks wrong: %+v", bare)
	}
	if after := unseen(t, u); after != before {
		t.Errorf("reading the feed changed \\Seen flags: %d -> %d", before, after)
	}
}

func TestLatestEmailsCacheServesDeeperWindowAndExpires(t *testing.T) {
	srv := startIMAP(t)
	srv.addUser(t, "a@x.com", "pa",
		msg{fromAddr: "p@q.com", subject: "1", date: t0},
		msg{fromAddr: "p@q.com", subject: "2", date: t0},
		msg{fromAddr: "p@q.com", subject: "3", date: t0})
	kv := map[string]string{}
	slotEnv(kv, 1, "a@x.com", "pa", srv.port)
	c := testConnector(writeEnv(t, kv))
	now := t0
	c.now = func() time.Time { return now }
	var dials int
	var mu sync.Mutex
	inner := c.dialIMAP
	c.dialIMAP = func(ctx context.Context, cfg InboxConfig) (net.Conn, error) {
		mu.Lock()
		dials++
		mu.Unlock()
		return inner(ctx, cfg)
	}
	ctx := context.Background()
	if items, _ := c.LatestEmails(ctx, 3); len(items) != 3 {
		t.Fatalf("got %d", len(items))
	}
	if items, _ := c.LatestEmails(ctx, 1); len(items) != 3 || dials != 1 {
		t.Errorf("a shallower request must be served from the deeper cache (dials=%d)", dials)
	}
	c.LatestEmails(ctx, 10)
	if dials != 2 {
		t.Errorf("a deeper request than cached must refetch (dials=%d)", dials)
	}
	now = now.Add(CacheTTL + time.Second)
	c.LatestEmails(ctx, 10)
	if dials != 3 {
		t.Errorf("an expired cache must refetch (dials=%d)", dials)
	}
	c.InvalidateCache()
	c.LatestEmails(ctx, 10)
	if dials != 4 {
		t.Errorf("InvalidateCache must drop the cache (dials=%d)", dials)
	}
}

func TestLatestEmailsAllInboxesFailingIsAnErrorNotAnEmptyFeed(t *testing.T) {
	srv := startIMAP(t)
	srv.addUser(t, "a@x.com", "pa")
	kv := map[string]string{}
	slotEnv(kv, 1, "a@x.com", "WRONG", srv.port)
	items, err := testConnector(writeEnv(t, kv)).LatestEmails(context.Background(), 5)
	if err == nil || items != nil {
		t.Fatalf("want error and nil items, got %v %v", items, err)
	}
}

func TestLatestEmailsNotConfigured(t *testing.T) {
	_, err := New(writeEnv(t, nil)).LatestEmails(context.Background(), 5)
	if !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("err = %v", err)
	}
	_, err = New(writeEnv(t, nil)).UnreadCounts(context.Background())
	if !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("err = %v", err)
	}
}

// ---- send -------------------------------------------------------------------

type fakeSMTP struct {
	mu   sync.Mutex
	user string
	pass string
	got  []sent
}

type sent struct {
	authUser, from string
	to             []string
	data           string
}

func (f *fakeSMTP) NewSession(*smtp.Conn) (smtp.Session, error) { return &smtpSession{srv: f}, nil }

type smtpSession struct {
	srv  *fakeSMTP
	cur  sent
	auth bool
}

func (s *smtpSession) AuthMechanisms() []string { return []string{sasl.Plain} }
func (s *smtpSession) Auth(string) (sasl.Server, error) {
	return sasl.NewPlainServer(func(_, user, pass string) error {
		if user != s.srv.user || pass != s.srv.pass {
			return errors.New("bad credentials")
		}
		s.auth, s.cur.authUser = true, user
		return nil
	}), nil
}
func (s *smtpSession) Mail(from string, _ *smtp.MailOptions) error {
	if !s.auth {
		return errors.New("auth required")
	}
	s.cur.from = from
	return nil
}
func (s *smtpSession) Rcpt(to string, _ *smtp.RcptOptions) error {
	s.cur.to = append(s.cur.to, to)
	return nil
}
func (s *smtpSession) Data(r io.Reader) error {
	b, _ := io.ReadAll(r)
	s.cur.data = string(b)
	s.srv.mu.Lock()
	s.srv.got = append(s.srv.got, s.cur)
	s.srv.mu.Unlock()
	return nil
}
func (s *smtpSession) Reset()        { s.cur = sent{authUser: s.cur.authUser} }
func (s *smtpSession) Logout() error { return nil }

func startSMTP(t *testing.T, user, pass string) (*fakeSMTP, string) {
	t.Helper()
	f := &fakeSMTP{user: user, pass: pass}
	srv := smtp.NewServer(f)
	srv.AllowInsecureAuth = true
	srv.Domain = "localhost"
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	go srv.Serve(ln)
	t.Cleanup(func() { srv.Close() })
	return f, ln.Addr().String()
}

func sendConnector(t *testing.T, from, pass, smtpAddr string, extra map[string]string) *Connector {
	t.Helper()
	kv := map[string]string{"INBOX_1_HOST": "imap.gmail.com", "INBOX_1_USER": from, "INBOX_1_PASS": pass}
	for k, v := range extra {
		kv[k] = v
	}
	c := New(writeEnv(t, kv))
	c.dialSMTP = func(ctx context.Context, cfg InboxConfig) (*smtp.Client, error) {
		if smtpAddr == "" {
			t.Fatal("SMTP must not be dialled in this test")
		}
		return smtp.Dial(smtpAddr)
	}
	return c
}

func TestSendReplyRefusedWhileWritesAreOff(t *testing.T) {
	t.Setenv("FOUNDEROS_WRITES", "0")
	guard.Reset()
	srv, addr := startSMTP(t, "alex@vantage.example", "app-pass")
	c := sendConnector(t, "alex@vantage.example", "app-pass", addr, nil)
	err := c.SendReply(context.Background(), Reply{To: "alex@personal.example", Subject: "hi", Text: "body"})
	if !errors.Is(err, guard.ErrWritesDisabled) {
		t.Fatalf("err = %v, want ErrWritesDisabled", err)
	}
	if len(srv.got) != 0 {
		t.Fatalf("a message left the bridge while writes were off")
	}
	refused := guard.Refused()
	if len(refused) == 0 || refused[len(refused)-1].Action != "email.send" {
		t.Errorf("refusal not recorded as email.send: %+v", refused)
	}
}

func TestSendReplyDeliversOverSMTPWhenWritesAreOn(t *testing.T) {
	t.Setenv("FOUNDEROS_WRITES", "1")
	srv, addr := startSMTP(t, "alex@vantage.example", "app-pass")
	c := sendConnector(t, "alex@vantage.example", "app-pass", addr, nil)
	err := c.SendReply(context.Background(), Reply{To: "Alex <Alex@Personal.example>", Subject: "Re: numbers\r\nBcc: evil@x.com", Text: "héllo\nthere"})
	if err != nil {
		t.Fatal(err)
	}
	if len(srv.got) != 1 {
		t.Fatalf("want 1 message, got %d", len(srv.got))
	}
	m := srv.got[0]
	if m.authUser != "alex@vantage.example" || m.from != "alex@vantage.example" {
		t.Errorf("auth/from = %q/%q", m.authUser, m.from)
	}
	if len(m.to) != 1 || m.to[0] != "alex@personal.example" {
		t.Errorf("rcpt = %v", m.to)
	}
	for _, want := range []string{"From: alex@vantage.example\r\n", "To: alex@personal.example\r\n", "Content-Type: text/plain; charset=utf-8\r\n"} {
		if !strings.Contains(m.data, want) {
			t.Errorf("message lacks %q:\n%s", want, m.data)
		}
	}
	if strings.Contains(m.data, "\r\nBcc:") {
		t.Errorf("a CRLF in the subject injected a header:\n%s", m.data)
	}
	if !strings.Contains(m.data, "h=C3=A9llo") {
		t.Errorf("body not quoted-printable UTF-8:\n%s", m.data)
	}
}

func TestSendReplyPicksTheOriginatingInbox(t *testing.T) {
	t.Setenv("FOUNDEROS_WRITES", "1")
	srv, addr := startSMTP(t, "alex@launchpadcohort.example", "p2")
	c := sendConnector(t, "alex@vantage.example", "p1", addr, map[string]string{
		"INBOX_2_HOST": "imap.gmail.com", "INBOX_2_USER": "alex@launchpadcohort.example", "INBOX_2_PASS": "p2",
	})
	if err := c.SendReply(context.Background(), Reply{AccountID: "inbox-2", To: "alex@vantage.example", Subject: "s", Text: "t"}); err != nil {
		t.Fatal(err)
	}
	if srv.got[0].from != "alex@launchpadcohort.example" {
		t.Errorf("sent from %q", srv.got[0].from)
	}
}

func TestSendReplyMailGuardRefusesExternalBeforeSMTP(t *testing.T) {
	t.Setenv("FOUNDEROS_WRITES", "1")
	c := sendConnector(t, "alex@vantage.example", "p", "", nil)
	err := c.SendReply(context.Background(), Reply{To: "customer@client.example", Subject: "s", Text: "t"})
	if err == nil || !strings.Contains(err.Error(), "blocked by mail-guard") {
		t.Fatalf("err = %v", err)
	}
}

func TestSendReplyHonestErrors(t *testing.T) {
	t.Setenv("FOUNDEROS_WRITES", "1")
	err := New(writeEnv(t, nil)).SendReply(context.Background(), Reply{To: "alex@vantage.example", Text: "t"})
	if err == nil || err.Error() != "no inbox configured (set INBOX_n_* in ~/.founderos/.env or under API keys)" {
		t.Errorf("err = %v", err)
	}
	c := sendConnector(t, "alex@vantage.example", "p", "", nil)
	if err := c.SendReply(context.Background(), Reply{To: "  ", Text: "t"}); err == nil || err.Error() != "no recipient address" {
		t.Errorf("err = %v", err)
	}
}

func TestSendReplySurfacesSMTPAuthFailure(t *testing.T) {
	t.Setenv("FOUNDEROS_WRITES", "1")
	srv, addr := startSMTP(t, "alex@vantage.example", "right")
	c := sendConnector(t, "alex@vantage.example", "wrong", addr, nil)
	if err := c.SendReply(context.Background(), Reply{To: "alex@vantage.example", Subject: "s", Text: "t"}); err == nil {
		t.Fatal("a refused login must not read as sent")
	}
	if len(srv.got) != 0 {
		t.Fatal("nothing should have been delivered")
	}
}
