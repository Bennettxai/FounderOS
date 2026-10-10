// Package typeform ports FounderOS v1's lib/connectors/typeform.ts: lead-form
// submissions that feed the funnel (since GoHighLevel and Attio were
// cancelled, 2026-09-23). Read-only; FounderOS v1 never writes to Typeform.
//
// API: personal access token as a Bearer header (scopes forms:read and
// responses:read); GET /forms, /forms/{id} (question titles live only there)
// and /forms/{id}/responses. The account allows 2 requests/second, so the
// lead pull walks forms one at a time and caps how many it reads.
package typeform

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/rhl/businessos-backend/internal/founderos/connectors"
)

var Meta = connectors.Meta{ID: "typeform", Name: "Typeform", Kind: connectors.KindCRM}

// ErrNotConfigured means no TYPEFORM_API_KEY: no source, which the funnel
// must not read as "no leads".
var ErrNotConfigured = errors.New("TYPEFORM_API_KEY is not set")

const (
	defaultBaseURL = "https://api.typeform.com"
	keyName        = "TYPEFORM_API_KEY"
	timeout        = 6 * time.Second
)

type Form struct {
	ID            string `json:"id"`
	Title         string `json:"title"`
	LastUpdatedAt string `json:"lastUpdatedAt,omitempty"` // "" where the TS has null
}

// Lead is one completed submission reduced to who the lead is. Empty strings
// stand for the TS nulls. Email is trimmed and lowercased so it joins against
// the calendar and Stripe.
type Lead struct {
	ResponseID  string            `json:"responseId"`
	FormID      string            `json:"formId"`
	FormTitle   string            `json:"formTitle"`
	Name        string            `json:"name,omitempty"`
	Email       string            `json:"email,omitempty"`
	Phone       string            `json:"phone,omitempty"`
	SubmittedAt string            `json:"submittedAt"`
	Hidden      map[string]string `json:"hidden"`
}

type Connector struct {
	res       connectors.Resolver
	baseURL   string
	http      *http.Client
	credFiles []string
}

func New(res connectors.Resolver) *Connector {
	social, clue, _, _ := connectors.CredFiles()
	return &Connector{
		res:       res,
		baseURL:   defaultBaseURL,
		http:      connectors.HTTPClient(timeout),
		credFiles: []string{clue, social}, // lib/creds.ts order for this key
	}
}

func (c *Connector) key() string { return c.res.Resolve(keyName, c.credFiles...) }

func str(v any) string {
	s, _ := v.(string)
	return strings.TrimSpace(s)
}

// ParseForms reads a GET /forms page; items without an id are skipped.
func ParseForms(body []byte) []Form {
	var doc struct {
		Items []map[string]any `json:"items"`
	}
	if json.Unmarshal(body, &doc) != nil {
		return []Form{}
	}
	out := []Form{}
	for _, f := range doc.Items {
		id := str(f["id"])
		if id == "" {
			continue
		}
		title := str(f["title"])
		if title == "" {
			title = "Untitled form"
		}
		out = append(out, Form{ID: id, Title: title, LastUpdatedAt: str(f["last_updated_at"])})
	}
	return out
}

// FieldTitles maps field id to question title, reaching into grouped blocks
// (group, inline_group and contact_info nest questions under properties.fields).
func FieldTitles(formDef []byte) map[string]string {
	titles := map[string]string{}
	var doc struct {
		Fields []any `json:"fields"`
	}
	if json.Unmarshal(formDef, &doc) != nil {
		return titles
	}
	var walk func([]any)
	walk = func(fields []any) {
		for _, raw := range fields {
			f, _ := raw.(map[string]any)
			if f == nil {
				continue
			}
			if id, title := str(f["id"]), str(f["title"]); id != "" && title != "" {
				titles[id] = title
			}
			if props, ok := f["properties"].(map[string]any); ok {
				if nested, ok := props["fields"].([]any); ok {
					walk(nested)
				}
			}
		}
	}
	walk(doc.Fields)
	return titles
}

// A question is the person's name when its title (or ref) says so, and not
// when it is the company's or business's name.
var (
	nameRE      = regexp.MustCompile(`(?i)name`)
	notPersonRE = regexp.MustCompile(`(?i)company|business|brand|agency|organi[sz]ation|website`)
	firstRE     = regexp.MustCompile(`(?i)first`)
	lastRE      = regexp.MustCompile(`(?i)last|surname`)
)

// ParseResponses reduces a responses page to leads. A submission nobody can
// be matched to or called (no email, name or phone) is not a lead.
func ParseResponses(body []byte, form Form, titles map[string]string) []Lead {
	var doc struct {
		Items []map[string]any `json:"items"`
	}
	if json.Unmarshal(body, &doc) != nil {
		return []Lead{}
	}
	out := []Lead{}
	for _, r := range doc.Items {
		responseID := str(r["response_id"])
		if responseID == "" {
			responseID = str(r["token"])
		}
		submittedAt := str(r["submitted_at"])
		if responseID == "" || submittedAt == "" {
			continue
		}
		hidden := map[string]string{}
		if h, ok := r["hidden"].(map[string]any); ok {
			for k, v := range h {
				if s := str(v); s != "" {
					hidden[k] = s
				}
			}
		}
		var email, phone, first, last, full string
		answers, _ := r["answers"].([]any)
		for _, raw := range answers {
			a, _ := raw.(map[string]any)
			if a == nil {
				continue
			}
			field, _ := a["field"].(map[string]any)
			switch a["type"] {
			case "email":
				if email == "" {
					email = str(a["email"])
				}
			case "phone_number":
				if phone == "" {
					phone = str(a["phone_number"])
				}
			case "text":
				label := titles[fmt.Sprint(field["id"])] + " " + str(field["ref"])
				if !nameRE.MatchString(label) || notPersonRE.MatchString(label) {
					continue
				}
				text := str(a["text"])
				switch {
				case firstRE.MatchString(label):
					if first == "" {
						first = text
					}
				case lastRE.MatchString(label):
					if last == "" {
						last = text
					}
				default:
					if full == "" {
						full = text
					}
				}
			}
		}
		if email == "" {
			email = hidden["email"]
		}
		email = strings.ToLower(email)
		name := full
		if name == "" {
			var parts []string
			for _, p := range []string{first, last} {
				if p != "" {
					parts = append(parts, p)
				}
			}
			name = strings.Join(parts, " ")
		}
		if name == "" {
			name = hidden["name"]
		}
		if email == "" && name == "" && phone == "" {
			continue
		}
		out = append(out, Lead{ResponseID: responseID, FormID: form.ID, FormTitle: form.Title, Name: name,
			Email: email, Phone: phone, SubmittedAt: submittedAt, Hidden: hidden})
	}
	return out
}

func (c *Connector) getJSON(ctx context.Context, key, path string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+key)
	req.Header.Set("Accept", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	return io.ReadAll(io.LimitReader(resp.Body, 32<<20))
}

type LeadOptions struct {
	Now      time.Time // zero means time.Now()
	Days     int       // how far back a submission still counts; 0 means 90
	MaxForms int       // forms read when TYPEFORM_FORM_IDS is unset; 0 means 5
}

// Leads returns completed submissions across the lead forms: those pinned in
// TYPEFORM_FORM_IDS (comma-separated) when set, otherwise the MaxForms most
// recently updated. ErrNotConfigured when unkeyed, an error when Typeform
// refuses the form listing; one form failing drops that form only. A keyed
// read with no submissions is an empty, non-nil slice.
func (c *Connector) Leads(ctx context.Context, opts LeadOptions) ([]Lead, error) {
	if opts.Now.IsZero() {
		opts.Now = time.Now()
	}
	if opts.Days == 0 {
		opts.Days = 90
	}
	if opts.MaxForms == 0 {
		opts.MaxForms = 5
	}
	key := c.key()
	if key == "" {
		return nil, ErrNotConfigured
	}
	since := opts.Now.Add(-time.Duration(opts.Days) * 24 * time.Hour).UTC().Format("2006-01-02T15:04:05.000Z")
	body, err := c.getJSON(ctx, key, "/forms?page_size=200&sort_by=last_updated_at&order_by=desc")
	if err != nil {
		return nil, err
	}
	all := ParseForms(body)
	var pinned []string
	for _, s := range strings.Split(c.res.Resolve("TYPEFORM_FORM_IDS"), ",") {
		if s = strings.TrimSpace(s); s != "" {
			pinned = append(pinned, s)
		}
	}
	var forms []Form
	if len(pinned) > 0 {
		for _, f := range all {
			for _, p := range pinned {
				if f.ID == p {
					forms = append(forms, f)
					break
				}
			}
		}
	} else {
		forms = all[:min(opts.MaxForms, len(all))]
	}

	leads := []Lead{}
	// Sequential on purpose: Typeform allows 2 requests/second per account.
	for _, f := range forms {
		id := url.PathEscape(f.ID)
		def, err := c.getJSON(ctx, key, "/forms/"+id)
		if err != nil {
			continue // one unreadable form costs its own leads, not everyone else's
		}
		resp, err := c.getJSON(ctx, key, "/forms/"+id+"/responses?page_size=200&response_type=completed&since="+url.QueryEscape(since))
		if err != nil {
			continue
		}
		leads = append(leads, ParseResponses(resp, f, FieldTitles(def))...)
	}
	return leads, nil
}

func (c *Connector) st(state connectors.State, detail string, meta map[string]any) connectors.Status {
	return connectors.Status{ID: Meta.ID, Name: Meta.Name, Kind: Meta.Kind, State: state, Detail: detail, Meta: meta}
}

// Status lists one form to prove the token works. It never writes.
func (c *Connector) Status(ctx context.Context) connectors.Status {
	key := c.key()
	if key == "" {
		return c.st(connectors.StateNotConfigured,
			"Set TYPEFORM_API_KEY to pull lead-form submissions into the funnel. Create a personal token in Typeform → Account → Personal tokens with forms:read and responses:read.", nil)
	}
	body, err := c.getJSON(ctx, key, "/forms?page_size=1")
	if err != nil {
		return c.st(connectors.StateError, "TYPEFORM_API_KEY is set but the call failed: "+err.Error(), nil)
	}
	var doc struct {
		TotalItems *float64 `json:"total_items"`
	}
	if err := json.Unmarshal(body, &doc); err != nil {
		return c.st(connectors.StateError, "TYPEFORM_API_KEY is set but the call failed: "+err.Error(), nil)
	}
	forms := len(ParseForms(body))
	if doc.TotalItems != nil {
		forms = int(*doc.TotalItems)
	}
	plural := "s"
	if forms == 1 {
		plural = ""
	}
	return c.st(connectors.StateConnected,
		fmt.Sprintf("Typeform reachable · %d form%s visible to this token", forms, plural),
		map[string]any{"forms": forms})
}
