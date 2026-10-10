// Package email ports FounderOS v1 lib/connectors/email.ts: up to four IMAP
// inboxes (INBOX_{1-4}_*) read for the Comms feed and unread counts, and a
// guarded SMTP reply.
//
// IMAP client: github.com/emersion/go-imap/v2. v1 is unmaintained and its
// author directs new code to v2, which is actively developed, speaks
// IMAP4rev1 and rev2, and ships imapmemserver, an in-process server the tests
// run the real protocol against. SMTP uses the same author's go-smtp (client
// here, server in the tests) with go-sasl PLAIN.
package email

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/tls"
	"encoding/hex"
	"errors"
	"fmt"
	"mime"
	"mime/quotedprintable"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
	"github.com/emersion/go-sasl"
	"github.com/emersion/go-smtp"

	"github.com/rhl/businessos-backend/internal/founderos/connectors"
	"github.com/rhl/businessos-backend/internal/founderos/guard"
)

var Meta = connectors.Meta{ID: "email", Name: "Email Inboxes", Kind: connectors.KindEmail}

// MaxInboxes is the number of INBOX_n_* slots.
const MaxInboxes = 4

// CacheTTL deliberately exceeds FounderOS v1's 15-minute refresh sweep, so the
// sweep repopulates the cache before a page view has to pay the cold IMAP
// cost (measured at 18s). Only successes are cached.
const CacheTTL = 20 * time.Minute

const (
	connectTimeout = 5 * time.Second  // TS connectionTimeout / greetingTimeout
	sessionBudget  = 15 * time.Second // greeting + socket timeouts, as one budget
	smtpBudget     = 30 * time.Second
)

// ErrNotConfigured means no INBOX_n slot is complete: there is no source, which
// is not the same as an empty inbox.
var ErrNotConfigured = errors.New("email: no inboxes configured")

// InboxConfig is one configured slot. Pass never serializes.
type InboxConfig struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	User     string `json:"user"`
	Pass     string `json:"-"`
	SMTPHost string `json:"smtpHost"`
	SMTPPort int    `json:"smtpPort"`
}

// CommsItem mirrors FounderOS v1's CommsItem for source "email".
type CommsItem struct {
	Source  string `json:"source"`
	Title   string `json:"title"`
	Sender  string `json:"sender,omitempty"`
	ReplyTo string `json:"replyTo,omitempty"`
	Account string `json:"account,omitempty"`
	Preview string `json:"preview"`
	TS      string `json:"ts"`
	Unread  int    `json:"unread"`
}

// InboxUnread is one inbox's unseen count. Unread is nil when the inbox did not
// answer: unknown reads unknown, never 0.
type InboxUnread struct {
	Inbox  string `json:"inbox"`
	Unread *int   `json:"unread"`
	Error  string `json:"error,omitempty"`
}

// Reply is an outbound reply from one of the inboxes.
type Reply struct {
	AccountID string // inbox-n; defaults to the first configured inbox
	To        string
	Subject   string
	Text      string
}

type Connector struct {
	res connectors.Resolver

	// seams: tests swap the socket and the clock, never the protocol.
	dialIMAP func(ctx context.Context, cfg InboxConfig) (net.Conn, error)
	dialSMTP func(ctx context.Context, cfg InboxConfig) (*smtp.Client, error)
	now      func() time.Time

	mu          sync.Mutex
	feedCache   *feedCache
	unreadCache *unreadCache
}

type feedCache struct {
	at    time.Time
	limit int
	items []CommsItem
}

type unreadCache struct {
	at     time.Time
	counts []InboxUnread
}

func New(res connectors.Resolver) *Connector {
	return &Connector{res: res, dialIMAP: dialIMAPTLS, dialSMTP: dialSMTP, now: time.Now}
}

func (c *Connector) env(name string) string { return c.res.Resolve(name) }

// Inboxes returns every complete INBOX_n slot, resolved fresh on each call.
func (c *Connector) Inboxes() []InboxConfig {
	var out []InboxConfig
	for slot := 1; slot <= MaxInboxes; slot++ {
		key := func(k string) string { return c.env(fmt.Sprintf("INBOX_%d_%s", slot, k)) }
		host, user, pass := key("HOST"), key("USER"), key("PASS")
		if host == "" || user == "" || pass == "" {
			continue
		}
		cfg := InboxConfig{
			ID:       fmt.Sprintf("inbox-%d", slot),
			Name:     user,
			Host:     host,
			Port:     intOr(key("PORT"), 993),
			User:     user,
			Pass:     pass,
			SMTPHost: key("SMTP_HOST"),
			SMTPPort: intOr(key("SMTP_PORT"), 465),
		}
		if n := key("NAME"); n != "" {
			cfg.Name = n
		}
		if cfg.SMTPHost == "" {
			cfg.SMTPHost = host
			if strings.HasPrefix(host, "imap.") {
				cfg.SMTPHost = "smtp." + strings.TrimPrefix(host, "imap.")
			}
		}
		out = append(out, cfg)
	}
	return out
}

func intOr(s string, def int) int {
	if n, err := strconv.Atoi(strings.TrimSpace(s)); err == nil {
		return n
	}
	return def
}

// ---- IMAP -------------------------------------------------------------------

func dialIMAPTLS(ctx context.Context, cfg InboxConfig) (net.Conn, error) {
	d := &tls.Dialer{NetDialer: &net.Dialer{Timeout: connectTimeout}, Config: &tls.Config{ServerName: cfg.Host}}
	return d.DialContext(ctx, "tcp", net.JoinHostPort(cfg.Host, strconv.Itoa(cfg.Port)))
}

// session opens, logs in, runs fn and logs out, all inside one time budget:
// a throttled Gmail connect degrades the board, never stalls it.
func (c *Connector) session(ctx context.Context, cfg InboxConfig, fn func(*imapclient.Client) error) error {
	ctx, cancel := context.WithTimeout(ctx, sessionBudget)
	defer cancel()
	conn, err := c.dialIMAP(ctx, cfg)
	if err != nil {
		return err
	}
	if dl, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(dl)
	}
	stop := context.AfterFunc(ctx, func() { conn.Close() })
	defer stop()

	client := imapclient.New(conn, nil)
	defer client.Close()
	if err := client.WaitGreeting(); err != nil {
		return fmt.Errorf("greeting: %w", err)
	}
	if err := client.Login(cfg.User, cfg.Pass).Wait(); err != nil {
		return fmt.Errorf("login: %w", err)
	}
	if err := fn(client); err != nil {
		return err
	}
	_ = client.Logout().Wait()
	return nil
}

func (c *Connector) unreadCount(ctx context.Context, cfg InboxConfig) InboxUnread {
	out := InboxUnread{Inbox: cfg.Name}
	err := c.session(ctx, cfg, func(cl *imapclient.Client) error {
		st, err := cl.Status("INBOX", &imap.StatusOptions{NumUnseen: true}).Wait()
		if err != nil {
			return err
		}
		if st.NumUnseen == nil {
			return errors.New("server returned no UNSEEN count")
		}
		n := int(*st.NumUnseen)
		out.Unread = &n
		return nil
	})
	if err != nil {
		out.Unread, out.Error = nil, err.Error()
	}
	return out
}

// UnreadCounts reads every inbox's unseen count concurrently, in slot order.
// A result where at least one inbox answered is cached for CacheTTL, so a
// total outage retries instead of being pinned.
func (c *Connector) UnreadCounts(ctx context.Context) ([]InboxUnread, error) {
	c.mu.Lock()
	if uc := c.unreadCache; uc != nil && c.now().Sub(uc.at) < CacheTTL {
		c.mu.Unlock()
		return uc.counts, nil
	}
	c.mu.Unlock()

	inboxes := c.Inboxes()
	if len(inboxes) == 0 {
		return nil, ErrNotConfigured
	}
	counts := make([]InboxUnread, len(inboxes))
	var wg sync.WaitGroup
	for i, cfg := range inboxes {
		wg.Add(1)
		go func() {
			defer wg.Done()
			counts[i] = c.unreadCount(ctx, cfg)
		}()
	}
	wg.Wait()
	for _, ct := range counts {
		if ct.Error == "" {
			c.mu.Lock()
			c.unreadCache = &unreadCache{at: c.now(), counts: counts}
			c.mu.Unlock()
			break
		}
	}
	return counts, nil
}

const isoMillis = "2006-01-02T15:04:05.000Z"

func (c *Connector) latestFrom(ctx context.Context, cfg InboxConfig, limit int) ([]CommsItem, error) {
	var items []CommsItem
	err := c.session(ctx, cfg, func(cl *imapclient.Client) error {
		// EXAMINE, not SELECT: reading the feed can never change a flag.
		sel, err := cl.Select("INBOX", &imap.SelectOptions{ReadOnly: true}).Wait()
		if err != nil {
			return err
		}
		exists := sel.NumMessages
		if exists == 0 {
			return nil
		}
		start := uint32(1)
		if int(exists) > limit {
			start = exists - uint32(limit) + 1
		}
		var seq imap.SeqSet
		seq.AddRange(start, exists)
		msgs, err := cl.Fetch(seq, &imap.FetchOptions{Envelope: true, Flags: true}).Collect()
		if err != nil {
			return err
		}
		for _, m := range msgs {
			items = append(items, toItem(cfg, m))
		}
		return nil
	})
	return items, err
}

func toItem(cfg InboxConfig, m *imapclient.FetchMessageBuffer) CommsItem {
	sender, replyTo := "unknown sender", ""
	subject := "(no subject)"
	date := time.Unix(0, 0)
	if env := m.Envelope; env != nil {
		if len(env.From) > 0 {
			from := env.From[0]
			replyTo = from.Addr()
			switch {
			case from.Name != "":
				sender = from.Name
			case replyTo != "":
				sender = replyTo
			}
		}
		if env.Subject != "" {
			subject = env.Subject
		}
		if !env.Date.IsZero() {
			date = env.Date
		}
	}
	unread := 1
	for _, f := range m.Flags {
		if f == imap.FlagSeen {
			unread = 0
		}
	}
	return CommsItem{
		Source:  "email",
		Title:   cfg.Name + " — " + sender,
		Sender:  sender,
		ReplyTo: replyTo,
		Account: cfg.ID,
		Preview: subject,
		TS:      date.UTC().Format(isoMillis),
		Unread:  unread,
	}
}

// LatestEmails returns the newest limitPerInbox envelopes of every inbox, in
// slot order. One inbox failing costs only that inbox (Status reports it);
// every inbox failing is an error, never an empty feed. A cache built at a
// deeper limit serves shallower requests; only non-empty results are cached.
func (c *Connector) LatestEmails(ctx context.Context, limitPerInbox int) ([]CommsItem, error) {
	if limitPerInbox <= 0 {
		limitPerInbox = 40
	}
	c.mu.Lock()
	if fc := c.feedCache; fc != nil && fc.limit >= limitPerInbox && c.now().Sub(fc.at) < CacheTTL {
		c.mu.Unlock()
		return fc.items, nil
	}
	c.mu.Unlock()

	inboxes := c.Inboxes()
	if len(inboxes) == 0 {
		return nil, ErrNotConfigured
	}
	per := make([][]CommsItem, len(inboxes))
	errs := make([]error, len(inboxes))
	var wg sync.WaitGroup
	for i, cfg := range inboxes {
		wg.Add(1)
		go func() {
			defer wg.Done()
			per[i], errs[i] = c.latestFrom(ctx, cfg, limitPerInbox)
		}()
	}
	wg.Wait()

	var items []CommsItem
	failed := 0
	for i := range inboxes {
		if errs[i] != nil {
			failed++
			continue
		}
		items = append(items, per[i]...)
	}
	if failed == len(inboxes) {
		return nil, fmt.Errorf("all %d inbox connections failed: %w", failed, errs[0])
	}
	if items == nil {
		items = []CommsItem{}
	}
	if len(items) > 0 {
		c.mu.Lock()
		c.feedCache = &feedCache{at: c.now(), limit: limitPerInbox, items: items}
		c.mu.Unlock()
	}
	return items, nil
}

// InvalidateCache drops both caches so a reply or a test sees fresh state.
func (c *Connector) InvalidateCache() {
	c.mu.Lock()
	c.feedCache, c.unreadCache = nil, nil
	c.mu.Unlock()
}

// ---- status -----------------------------------------------------------------

// Status reads unseen counts only (IMAP STATUS); it never writes.
func (c *Connector) Status(ctx context.Context) connectors.Status {
	s := connectors.Status{ID: Meta.ID, Name: Meta.Name, Kind: Meta.Kind}
	inboxes := c.Inboxes()
	if len(inboxes) == 0 {
		s.State = connectors.StateNotConfigured
		s.Detail = "No inboxes configured. Set INBOX_1_HOST / _USER / _PASS (up to 4 slots) in ~/.founderos/.env or under API keys."
		s.Meta = map[string]any{"configured": 0, "slots": MaxInboxes}
		return s
	}
	counts, err := c.UnreadCounts(ctx)
	if err != nil {
		s.State, s.Detail = connectors.StateError, err.Error()
		return s
	}
	var failing []InboxUnread
	total := 0
	for _, ct := range counts {
		if ct.Error != "" {
			failing = append(failing, ct)
			continue
		}
		total += *ct.Unread
	}
	if len(failing) == len(counts) {
		s.State = connectors.StateError
		s.Detail = fmt.Sprintf("All %d inbox connections failed: %s", len(counts), failing[0].Error)
		s.Meta = map[string]any{"configured": len(inboxes)}
		return s
	}
	plural := ""
	if len(inboxes) > 1 {
		plural = "es"
	}
	s.State = connectors.StateConnected
	s.Detail = fmt.Sprintf("%d inbox%s · %d unread", len(inboxes), plural, total)
	if len(failing) > 0 {
		s.Detail += fmt.Sprintf(" · %d failing", len(failing))
	}
	s.Meta = map[string]any{"configured": len(inboxes), "unread": total}
	return s
}

// ---- SMTP (guarded) ---------------------------------------------------------

func dialSMTP(ctx context.Context, cfg InboxConfig) (*smtp.Client, error) {
	addr := net.JoinHostPort(cfg.SMTPHost, strconv.Itoa(cfg.SMTPPort))
	tlsCfg := &tls.Config{ServerName: cfg.SMTPHost}
	if cfg.SMTPPort == 465 { // implicit TLS, as nodemailer's secure:true
		d := &tls.Dialer{NetDialer: &net.Dialer{Timeout: connectTimeout}, Config: tlsCfg}
		conn, err := d.DialContext(ctx, "tcp", addr)
		if err != nil {
			return nil, err
		}
		setDeadline(ctx, conn)
		return smtp.NewClient(conn), nil
	}
	// Otherwise STARTTLS is required: the app password never crosses in clear.
	d := &net.Dialer{Timeout: connectTimeout}
	conn, err := d.DialContext(ctx, "tcp", addr)
	if err != nil {
		return nil, err
	}
	setDeadline(ctx, conn)
	return smtp.NewClientStartTLS(conn, tlsCfg)
}

func setDeadline(ctx context.Context, conn net.Conn) {
	if dl, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(dl)
	}
}

// SendReply sends a plain-text reply from the originating inbox. The mail
// guard (an earlier incident) is checked first and refuses unapproved recipients; then the
// send runs only through guard.Outbound("email.send"), so with FOUNDEROS_WRITES
// off nothing is dialled and guard.ErrWritesDisabled comes back. Callers map a
// non-nil error to FounderOS v1's { ok: false, error } (the UI then offers a
// mailto: draft).
func (c *Connector) SendReply(ctx context.Context, r Reply) error {
	inboxes := c.Inboxes()
	if len(inboxes) == 0 {
		return errors.New("no inbox configured (set INBOX_n_* in ~/.founderos/.env or under API keys)")
	}
	if strings.TrimSpace(r.To) == "" {
		return errors.New("no recipient address")
	}
	cfg := inboxes[0]
	for _, in := range inboxes {
		if in.ID == r.AccountID {
			cfg = in
		}
	}
	rcpts, err := CheckOutboundMail(OutboundMail{From: cfg.User, To: r.To}, c.env)
	if err != nil {
		return err
	}
	err = guard.Outbound("email.send", func() error {
		ctx, cancel := context.WithTimeout(ctx, smtpBudget)
		defer cancel()
		client, err := c.dialSMTP(ctx, cfg)
		if err != nil {
			return err
		}
		defer client.Close()
		if err := client.Auth(sasl.NewPlainClient("", cfg.User, cfg.Pass)); err != nil {
			return err
		}
		if err := client.SendMail(cfg.User, rcpts, bytes.NewReader(buildMessage(cfg.User, rcpts, r.Subject, r.Text, c.now()))); err != nil {
			return err
		}
		return client.Quit()
	})
	if err != nil {
		return err
	}
	c.InvalidateCache()
	return nil
}

var headerSafe = strings.NewReplacer("\r", " ", "\n", " ")

func buildMessage(from string, to []string, subject, text string, now time.Time) []byte {
	if subject == "" {
		subject = "(no subject)"
	}
	domain := from[strings.LastIndex(from, "@")+1:]
	id := make([]byte, 12)
	_, _ = rand.Read(id)

	var b bytes.Buffer
	fmt.Fprintf(&b, "From: %s\r\n", headerSafe.Replace(from))
	fmt.Fprintf(&b, "To: %s\r\n", headerSafe.Replace(strings.Join(to, ", ")))
	fmt.Fprintf(&b, "Subject: %s\r\n", mime.QEncoding.Encode("utf-8", headerSafe.Replace(subject)))
	fmt.Fprintf(&b, "Date: %s\r\n", now.Format(time.RFC1123Z))
	fmt.Fprintf(&b, "Message-ID: <%s@%s>\r\n", hex.EncodeToString(id), domain)
	b.WriteString("MIME-Version: 1.0\r\n")
	b.WriteString("Content-Type: text/plain; charset=utf-8\r\n")
	b.WriteString("Content-Transfer-Encoding: quoted-printable\r\n\r\n")
	qp := quotedprintable.NewWriter(&b)
	_, _ = qp.Write([]byte(strings.ReplaceAll(strings.ReplaceAll(text, "\r\n", "\n"), "\n", "\r\n")))
	_ = qp.Close()
	b.WriteString("\r\n")
	return b.Bytes()
}
