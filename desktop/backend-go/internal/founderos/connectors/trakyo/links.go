package trakyo

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/rhl/businessos-backend/internal/founderos/guard"
)

// OperationError is a Trakyo link/content operation failure worded for the
// person pressing the button (lib/trakyo-links.ts TrakyoOperationError).
// Uncertain marks a write whose outcome is unknown: the item may exist, so
// the caller must not retry blindly.
type OperationError struct {
	Message   string
	Status    int
	Uncertain bool
}

func (e *OperationError) Error() string { return e.Message }

// ValidationError is input rejected before any request is spent.
type ValidationError struct{ Field, Reason string }

func (e *ValidationError) Error() string { return fmt.Sprintf("trakyo: %s: %s", e.Field, e.Reason) }

var statusMessages = map[int]string{
	401: "Trakyo rejected the API key. Reconnect in Integrations.",
	409: "That slug or item already exists. Choose another slug or use the existing item.",
	429: "Trakyo rate limit reached. Wait before retrying.",
	400: "Trakyo rejected these fields. Check the URL, content and slug.",
	404: "The selected Trakyo item no longer exists. Refresh the list.",
}

// request mirrors trakyo-links.ts request(): GET without a body, POST with
// one. Every failure becomes an OperationError.
func (c *Connector) request(ctx context.Context, key, path, scope string, body any) ([]byte, error) {
	if key == "" {
		return nil, &OperationError{Message: "Connect Trakyo in Integrations first.", Status: 503}
	}
	write := body != nil
	ctx, cancel := context.WithTimeout(ctx, linksTimeout)
	defer cancel()
	method := http.MethodGet
	var reader io.Reader
	if write {
		raw, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		method, reader = http.MethodPost, bytes.NewReader(raw)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.url(path), reader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+key)
	req.Header.Set("Accept", "application/json")
	if write {
		req.Header.Set("Content-Type", "application/json")
	}
	res, err := c.client.Do(req)
	if err != nil {
		if errors.Is(err, guard.ErrWritesDisabled) {
			return nil, err
		}
		if write {
			return nil, &OperationError{Message: "Result unconfirmed. Check Trakyo before retrying; the item may have been created.", Status: 502, Uncertain: true}
		}
		return nil, &OperationError{Message: "Trakyo is unreachable. Try refreshing.", Status: 502}
	}
	defer res.Body.Close()
	raw, readErr := io.ReadAll(io.LimitReader(res.Body, 32<<20))
	if res.StatusCode < 200 || res.StatusCode > 299 {
		msg, ok := statusMessages[res.StatusCode]
		if res.StatusCode == 403 {
			msg, ok = fmt.Sprintf("The Trakyo key needs %s. Enable that scope in Trakyo.", scope), true
		}
		if !ok {
			msg = "Trakyo returned an error."
			if write {
				msg = "Result unconfirmed. Check Trakyo before retrying."
			}
		}
		status := res.StatusCode
		if status >= 500 {
			status = 502
		}
		return nil, &OperationError{Message: msg, Status: status, Uncertain: write && res.StatusCode >= 500}
	}
	if readErr != nil || !json.Valid(raw) {
		if write {
			return nil, &OperationError{Message: "Result unconfirmed. Check Trakyo before retrying.", Status: 502, Uncertain: true}
		}
		return nil, &OperationError{Message: "Trakyo returned unreadable data.", Status: 502}
	}
	return raw, nil
}

// ---- reads: the link workspace ----------------------------------------------

// Domain is a Trakyo short-link domain.
type Domain struct {
	ID       string `json:"id"`
	Hostname string `json:"hostname"`
}

// Link is one saved tracked link.
type Link struct {
	Key                 string  `json:"key"`
	URL                 string  `json:"url"`
	ShortLink           string  `json:"short_link"`
	DomainID            string  `json:"domain_id"`
	ContentItemID       *string `json:"content_item_id"`
	Clicks              int64   `json:"clicks"`
	AppendTrackingParam bool    `json:"append_tracking_param"`
}

// LinkWorkspace is what the link builder needs: the key's scopes, domains,
// and the first 100 saved links. State is ready, partial, not_configured or
// error; Messages say what is missing.
type LinkWorkspace struct {
	State    string   `json:"state"`
	Scopes   []string `json:"scopes"`
	Domains  []Domain `json:"domains"`
	Links    []Link   `json:"links"`
	HasMore  bool     `json:"hasMore"`
	Messages []string `json:"messages"`
}

var uuidRe = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// publicHTTPURL is TrakyoHttpUrlSchema: http(s), ≤4096 chars, no credentials.
func publicHTTPURL(s string) bool {
	if len(s) > 4096 {
		return false
	}
	u, err := url.Parse(s)
	if err != nil || u.Host == "" || u.User != nil {
		return false
	}
	return u.Scheme == "http" || u.Scheme == "https"
}

func parseDomain(raw json.RawMessage) (Domain, error) {
	f, err := objectOf(raw)
	if err != nil {
		return Domain{}, err
	}
	var d Domain
	var e1, e2 error
	d.ID, e1 = f.str("id")
	d.Hostname, e2 = f.str("hostname")
	if e1 == nil && !uuidRe.MatchString(d.ID) {
		e1 = errors.New("id is not a uuid")
	}
	return d, firstErr(e1, e2)
}

func parseLink(raw json.RawMessage) (Link, error) {
	f, err := objectOf(raw)
	if err != nil {
		return Link{}, err
	}
	var l Link
	var e [7]error
	l.Key, e[0] = f.str("key")
	l.URL, e[1] = f.str("url")
	if e[1] == nil && !publicHTTPURL(l.URL) {
		e[1] = errors.New("url is not a public http(s) url")
	}
	l.ShortLink, e[2] = f.str("short_link")
	if e[2] == nil && !publicHTTPURL(l.ShortLink) {
		e[2] = errors.New("short_link is not a public http(s) url")
	}
	l.DomainID, e[3] = f.str("domain_id")
	if e[3] == nil && !uuidRe.MatchString(l.DomainID) {
		e[3] = errors.New("domain_id is not a uuid")
	}
	l.ContentItemID, e[4] = f.nullableStr("content_item_id")
	l.Clicks, e[5] = f.count("clicks")
	l.AppendTrackingParam, e[6] = f.boolean("append_tracking_param")
	return l, firstErr(e[:]...)
}

func dataArray(body []byte) ([]json.RawMessage, fields, error) {
	f, err := objectOf(body)
	if err != nil {
		return nil, nil, err
	}
	var rows []json.RawMessage
	if err := json.Unmarshal(f["data"], &rows); err != nil || rows == nil {
		return nil, nil, errors.New("data is not an array")
	}
	return rows, f, nil
}

func messageOr(err error, fallback string) string {
	var op *OperationError
	if errors.As(err, &op) {
		return op.Message
	}
	return fallback
}

// LinkWorkspace reads the key's scopes, then (with links:read) the domains
// and saved links independently, so a link-list failure does not hide the
// domain choices. It never includes the key.
func (c *Connector) LinkWorkspace(ctx context.Context) LinkWorkspace {
	out := LinkWorkspace{State: StateNotConfigured, Scopes: []string{}, Domains: []Domain{}, Links: []Link{}, Messages: []string{}}
	key := c.key()
	if key == "" {
		return out
	}
	raw, err := c.request(ctx, key, "keys/self", "", nil)
	var info struct {
		Scopes []string `json:"scopes"`
	}
	if err == nil {
		if json.Unmarshal(raw, &info) != nil || info.Scopes == nil {
			err = errors.New("unreadable scopes")
		}
	}
	if err != nil {
		out.State = StateError
		out.Messages = append(out.Messages, messageOr(err, "Could not read Trakyo permissions."))
		return out
	}
	out.Scopes, out.State = info.Scopes, StateReady
	hasLinksRead := false
	for _, s := range info.Scopes {
		if s == "links:read" {
			hasLinksRead = true
		}
	}
	if !hasLinksRead {
		out.State = StatePartial
		out.Messages = append(out.Messages, "Enable links:read in Trakyo to list domains and saved links.")
		return out
	}

	var mu sync.Mutex
	fail := func(msg string) {
		mu.Lock()
		out.State = StatePartial
		out.Messages = append(out.Messages, msg)
		mu.Unlock()
	}
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		raw, err := c.request(ctx, key, "domains", "links:read", nil)
		var domains []Domain
		if err == nil {
			var rows []json.RawMessage
			if rows, _, err = dataArray(raw); err == nil {
				for _, r := range rows {
					d, perr := parseDomain(r)
					if perr != nil {
						err = perr
						break
					}
					domains = append(domains, d)
				}
			}
		}
		if err != nil {
			fail(messageOr(err, "Could not read Trakyo domains."))
			return
		}
		mu.Lock()
		out.Domains = append(out.Domains, domains...)
		mu.Unlock()
	}()
	go func() {
		defer wg.Done()
		raw, err := c.request(ctx, key, "links?limit=100", "links:read", nil)
		var links []Link
		var more bool
		if err == nil {
			var rows []json.RawMessage
			var f fields
			if rows, f, err = dataArray(raw); err == nil {
				if more, err = f.boolean("has_more"); err == nil {
					for _, r := range rows {
						l, perr := parseLink(r)
						if perr != nil {
							err = perr
							break
						}
						links = append(links, l)
					}
				}
			}
		}
		if err != nil {
			fail(messageOr(err, "Could not read Trakyo links."))
			return
		}
		mu.Lock()
		out.Links, out.HasMore = append(out.Links, links...), more
		mu.Unlock()
	}()
	wg.Wait()
	return out
}

// ---- writes (guarded) --------------------------------------------------------

// ContentInput is TrakyoContentCreateSchema. Empty optional fields are omitted.
type ContentInput struct {
	Source      string `json:"source"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	URL         string `json:"url,omitempty"`
	PublishedAt string `json:"published_at,omitempty"`
}

// CreatedContent is TrakyoCreatedContentSchema.
type CreatedContent struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Source string `json:"source"`
	Type   string `json:"type"`
}

// LinkInput is TrakyoLinkCreateSchema. AppendTrackingParam defaults to true.
type LinkInput struct {
	URL                 string `json:"url"`
	ContentItemID       string `json:"content_item_id"`
	Domain              string `json:"domain,omitempty"`
	Slug                string `json:"slug,omitempty"`
	AppendTrackingParam *bool  `json:"append_tracking_param"`
}

var (
	digitsRe    = regexp.MustCompile(`^\d+$`)
	contentIDRe = regexp.MustCompile(`^ci_[A-Za-z0-9_-]+$`)
	slugRe      = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
)

func (in ContentInput) validate() (ContentInput, error) {
	in.Source, in.Name = strings.TrimSpace(in.Source), strings.TrimSpace(in.Name)
	switch l := strings.ToLower(in.Source); {
	case in.Source == "" || len([]rune(in.Source)) > 200:
		return in, &ValidationError{"source", "must be 1–200 characters"}
	case l == "youtube" || l == "other" || digitsRe.MatchString(in.Source):
		return in, &ValidationError{"source", "Choose a custom source such as Instagram"}
	case in.Name == "" || len([]rune(in.Name)) > 500:
		return in, &ValidationError{"name", "must be 1–500 characters"}
	case len([]rune(in.Description)) > 5000:
		return in, &ValidationError{"description", "must be at most 5000 characters"}
	case in.URL != "" && !publicHTTPURL(in.URL):
		return in, &ValidationError{"url", "Use a public HTTP or HTTPS URL without credentials"}
	}
	if in.PublishedAt != "" {
		if _, err := time.Parse(time.RFC3339Nano, in.PublishedAt); err != nil || !strings.HasSuffix(in.PublishedAt, "Z") {
			return in, &ValidationError{"published_at", "must be an ISO datetime in UTC"}
		}
	}
	return in, nil
}

func (in LinkInput) validate() (LinkInput, error) {
	in.Slug = strings.TrimSpace(in.Slug)
	switch {
	case !publicHTTPURL(in.URL):
		return in, &ValidationError{"url", "Use a public HTTP or HTTPS URL without credentials"}
	case !contentIDRe.MatchString(in.ContentItemID):
		return in, &ValidationError{"content_item_id", "must be a Trakyo content id (ci_…)"}
	case in.Domain != "" && !uuidRe.MatchString(in.Domain):
		return in, &ValidationError{"domain", "must be a domain id"}
	case in.Slug != "" && (len(in.Slug) > 200 || !slugRe.MatchString(in.Slug)):
		return in, &ValidationError{"slug", "letters, digits, _ and - only"}
	}
	if in.AppendTrackingParam == nil {
		yes := true
		in.AppendTrackingParam = &yes
	}
	return in, nil
}

func unconfirmed() error {
	return &OperationError{Message: "Result unconfirmed. Check Trakyo before retrying.", Status: 502, Uncertain: true}
}

// CreateContent registers a tracked content item (POST /content, scope
// content:write). Input is validated before anything is spent; the request is
// refused with guard.ErrWritesDisabled while FOUNDEROS_WRITES is off.
func (c *Connector) CreateContent(ctx context.Context, in ContentInput) (CreatedContent, error) {
	in, err := in.validate()
	if err != nil {
		return CreatedContent{}, err
	}
	var out CreatedContent
	err = guard.Outbound("trakyo.create_content", func() error {
		raw, err := c.request(ctx, c.key(), "content", "content:write", in)
		if err != nil {
			return err
		}
		f, err := objectOf(raw)
		if err != nil {
			return unconfirmed()
		}
		var e [4]error
		out.ID, e[0] = f.str("id")
		out.Name, e[1] = f.str("name")
		out.Source, e[2] = f.str("source")
		out.Type, e[3] = f.str("type")
		if firstErr(e[:]...) != nil || !strings.HasPrefix(out.ID, "ci_") || (out.Type != "custom" && out.Type != "youtube") {
			return unconfirmed()
		}
		return nil
	})
	return out, err
}

// CreateLink creates a tracked short link (POST /links, scope links:write).
// Content and link are separate calls so a failed link never recreates the
// content. Guarded like CreateContent.
func (c *Connector) CreateLink(ctx context.Context, in LinkInput) (Link, error) {
	in, err := in.validate()
	if err != nil {
		return Link{}, err
	}
	var out Link
	err = guard.Outbound("trakyo.create_link", func() error {
		raw, err := c.request(ctx, c.key(), "links", "links:write", in)
		if err != nil {
			return err
		}
		if out, err = parseLink(raw); err != nil {
			return unconfirmed()
		}
		return nil
	})
	return out, err
}
