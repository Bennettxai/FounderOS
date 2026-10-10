// Package paperclip ports FounderOS v1's lib/connectors/paperclip.ts: the agent
// harness board (Conductor → department leads → Hermes worker pool), served
// on the mini at :3100 over the tailnet.
//
// Creds: PAPERCLIP_API_URL + PAPERCLIP_BOARD_KEY (a USER-class board key, sent
// as a Bearer token) + PAPERCLIP_COMPANY_ID; PAPERCLIP_COCKPIT_ISSUE_ID pins
// the cockpit thread. The board key is refused by the agent-gated
// /api/agents/me BY DESIGN, so this connector never calls it.
//
// Differences from the TS, on purpose: every read returns an error instead
// of swallowing it into an empty list (unknown reads unknown), and every
// write is wrapped in guard.Outbound.
package paperclip

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/rhl/businessos-backend/internal/founderos/connectors"
	"github.com/rhl/businessos-backend/internal/founderos/guard"
)

var Meta = connectors.Meta{ID: "paperclip", Name: "Paperclip (agent harness)", Kind: connectors.KindOrchestration}

const (
	breakerWindow   = 30 * time.Second
	listTimeout     = 2500 * time.Millisecond
	issueTimeout    = 4 * time.Second
	writeTimeout    = 8 * time.Second
	cockpitPageSize = 100
)

var errNoCreds = errors.New("paperclip creds missing")

type Connector struct {
	res    connectors.Resolver
	now    func() time.Time
	client *http.Client

	mu         sync.Mutex
	downUntil  time.Time
	downReason string
	cockpitID  string
}

func New(res connectors.Resolver) *Connector {
	return &Connector{res: res, now: time.Now, client: connectors.HTTPClient(writeTimeout + time.Second)}
}

type creds struct{ url, key, company string }

func (c *Connector) creds() (creds, bool) {
	u := c.res.Resolve("PAPERCLIP_API_URL")
	k := c.res.Resolve("PAPERCLIP_BOARD_KEY")
	co := c.res.Resolve("PAPERCLIP_COMPANY_ID")
	if u == "" || k == "" || co == "" {
		return creds{}, false
	}
	return creds{url: strings.TrimSuffix(u, "/"), key: k, company: co}, true
}

func (c *Connector) pinnedCockpit() string {
	return strings.TrimSpace(c.res.Resolve("PAPERCLIP_COCKPIT_ISSUE_ID"))
}

// ---- transport + breaker ----------------------------------------------------

// fetch is boardFetch: the first network-level failure opens a 30s breaker
// and every board call inside the window fails at once. An HTTP error never
// opens it (the board answered), and any success closes it.
func (c *Connector) fetch(ctx context.Context, method, url string, key string, body any, timeout time.Duration) (*http.Response, error) {
	now := c.now()
	c.mu.Lock()
	if now.Before(c.downUntil) {
		left := int(math.Ceil(c.downUntil.Sub(now).Seconds()))
		reason := c.downReason
		c.mu.Unlock()
		return nil, fmt.Errorf("board unreachable (%s); not re-probed for %ds", reason, left)
	}
	c.mu.Unlock()

	var rd io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		rd = bytes.NewReader(raw)
	}
	rctx, cancel := context.WithTimeout(ctx, timeout)
	req, err := http.NewRequestWithContext(rctx, method, url, rd)
	if err != nil {
		cancel()
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+key)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	res, err := c.client.Do(req)
	if err != nil {
		cancel()
		if errors.Is(err, guard.ErrWritesDisabled) || ctx.Err() != nil {
			// a guard refusal or the caller giving up says nothing about the board
			return nil, err
		}
		c.mu.Lock()
		c.downReason = err.Error()
		c.downUntil = c.now().Add(breakerWindow)
		c.mu.Unlock()
		return nil, fmt.Errorf("board unreachable (%s)", err.Error())
	}
	c.mu.Lock()
	c.downUntil = time.Time{}
	c.mu.Unlock()
	res.Body = cancelOnClose{res.Body, cancel}
	return res, nil
}

type cancelOnClose struct {
	io.ReadCloser
	cancel context.CancelFunc
}

func (b cancelOnClose) Close() error {
	err := b.ReadCloser.Close()
	b.cancel()
	return err
}

func ok(res *http.Response) bool { return res.StatusCode >= 200 && res.StatusCode <= 299 }

// warnBoardHTTP makes a refused read loud: without it a 401 and an empty
// board look identical downstream (FOS-838).
func warnBoardHTTP(status int, path string) {
	why := "board error"
	if status == 401 || status == 403 {
		why = "credential rejected (check PAPERCLIP_BOARD_KEY in ~/.founderos/.env or under API keys)"
	}
	slog.Warn(fmt.Sprintf("[paperclip] HTTP %d on %s — %s", status, path, why))
}

func decode(res *http.Response) (any, error) {
	defer res.Body.Close()
	var body any
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		return nil, fmt.Errorf("unparseable board response: %w", err)
	}
	return body, nil
}

// get reads an absolute board path (e.g. /api/issues/x) as JSON.
func (c *Connector) get(ctx context.Context, path string, timeout time.Duration) (any, error) {
	cr, has := c.creds()
	if !has {
		return nil, errNoCreds
	}
	res, err := c.fetch(ctx, http.MethodGet, cr.url+path, cr.key, nil, timeout)
	if err != nil {
		return nil, err
	}
	if !ok(res) {
		res.Body.Close()
		warnBoardHTTP(res.StatusCode, path)
		return nil, fmt.Errorf("HTTP %d", res.StatusCode)
	}
	return decode(res)
}

// companyGet is boardGet: a path under /api/companies/{id}.
func (c *Connector) companyGet(ctx context.Context, path string) (any, error) {
	cr, has := c.creds()
	if !has {
		return nil, errNoCreds
	}
	return c.get(ctx, "/api/companies/"+cr.company+path, listTimeout)
}

// send is a guarded write. The response body is returned decoded when the
// board answered 2xx; a non-2xx is an error carrying the board's words.
func (c *Connector) send(ctx context.Context, action, method, path string, body any) (any, error) {
	cr, has := c.creds()
	if !has {
		return nil, errNoCreds
	}
	var out any
	err := guard.Outbound(action, func() error {
		res, err := c.fetch(ctx, method, cr.url+path, cr.key, body, writeTimeout)
		if err != nil {
			return err
		}
		if !ok(res) {
			raw, _ := io.ReadAll(io.LimitReader(res.Body, 200))
			res.Body.Close()
			return fmt.Errorf("HTTP %d: %s", res.StatusCode, strings.TrimSpace(string(raw)))
		}
		out, _ = decode(res)
		return nil
	})
	return out, err
}

// ---- reads ------------------------------------------------------------------

// Agents is the live agents list; an unreachable or refusing board is an error.
func (c *Connector) Agents(ctx context.Context) ([]Agent, error) {
	body, err := c.companyGet(ctx, "/agents")
	if err != nil {
		return nil, err
	}
	return MapAgents(listFrom(body, "agents")), nil
}

// Org is the live org tree flattened to preorder rows.
func (c *Connector) Org(ctx context.Context) ([]OrgNode, error) {
	body, err := c.companyGet(ctx, "/org")
	if err != nil {
		return nil, err
	}
	arr, _ := body.([]any)
	return FlattenOrg(arr), nil
}

// Issues are board tasks, newest first; limit <= 0 means 30.
func (c *Connector) Issues(ctx context.Context, limit int) ([]Issue, error) {
	if limit <= 0 {
		limit = 30
	}
	body, err := c.companyGet(ctx, fmt.Sprintf("/issues?limit=%d", limit))
	if err != nil {
		return nil, err
	}
	return MapIssues(listFrom(body, "issues")), nil
}

// Runs are recent heartbeat runs across the company; limit <= 0 means 30.
func (c *Connector) Runs(ctx context.Context, limit int) ([]Run, error) {
	if limit <= 0 {
		limit = 30
	}
	body, err := c.companyGet(ctx, fmt.Sprintf("/heartbeat-runs?limit=%d", limit))
	if err != nil {
		return nil, err
	}
	return MapRuns(listFrom(body, "runs")), nil
}

// FailoverRuns are runs shaped for the failover loop; limit <= 0 means 40.
func (c *Connector) FailoverRuns(ctx context.Context, limit int) ([]FailoverRun, error) {
	if limit <= 0 {
		limit = 40
	}
	body, err := c.companyGet(ctx, fmt.Sprintf("/heartbeat-runs?limit=%d", limit))
	if err != nil {
		return nil, err
	}
	return mapFailoverRuns(listFrom(body, "runs")), nil
}

type issueMatch struct {
	id  string
	num *float64
}

// findCockpitByScan pages the whole issue list (100 at a time) and returns
// the open cockpit issue with the lowest issue number.
func (c *Connector) findCockpitByScan(ctx context.Context) (string, error) {
	var matches []issueMatch
	for offset := 0; ; {
		body, err := c.companyGet(ctx, fmt.Sprintf("/issues?limit=%d&offset=%d", cockpitPageSize, offset))
		if err != nil {
			return "", err
		}
		list := listFrom(body, "issues")
		if len(list) == 0 {
			break
		}
		for _, rec := range list {
			r := obj(rec)
			id, isStr := r["id"].(string)
			if r["title"] != CockpitTitle || r["status"] == "done" || r["status"] == "cancelled" || !isStr {
				continue
			}
			m := issueMatch{id: id}
			if n, isNum := r["issueNumber"].(float64); isNum {
				m.num = &n
			}
			matches = append(matches, m)
		}
		if len(list) < cockpitPageSize {
			break
		}
		offset += len(list)
	}
	sort.SliceStable(matches, func(i, j int) bool {
		a, b := matches[i], matches[j]
		switch {
		case a.num == nil && b.num == nil:
			return a.id < b.id
		case a.num == nil:
			return false
		case b.num == nil:
			return true
		default:
			return *a.num < *b.num
		}
	})
	if len(matches) == 0 {
		return "", nil
	}
	return matches[0].id, nil
}

// EnsureCockpitIssue returns the standing cockpit issue id: the pinned one,
// else the open one found by scan, else a new one assigned to the Conductor.
// Creating it is a write and is guarded.
func (c *Connector) EnsureCockpitIssue(ctx context.Context) (string, error) {
	c.mu.Lock()
	cached := c.cockpitID
	c.mu.Unlock()
	if cached != "" {
		return cached, nil
	}
	remember := func(id string) string {
		c.mu.Lock()
		c.cockpitID = id
		c.mu.Unlock()
		return id
	}
	if pin := c.pinnedCockpit(); pin != "" {
		return remember(pin), nil
	}
	cr, has := c.creds()
	if !has {
		return "", errNoCreds
	}
	found, err := c.findCockpitByScan(ctx)
	if err != nil {
		return "", err
	}
	if found != "" {
		return remember(found), nil
	}
	conductor := ""
	if agents, err := c.Agents(ctx); err == nil {
		for _, a := range agents {
			if strings.EqualFold(a.Name, "conductor") {
				conductor = a.ID
				break
			}
		}
	}
	body, err := c.send(ctx, "paperclip.issue.create", http.MethodPost, "/api/companies/"+cr.company+"/issues", cockpitCreateBody(conductor))
	if err != nil {
		return "", fmt.Errorf("cockpit issue create failed: %w", err)
	}
	issue := obj(body)
	if inner := obj(issue["issue"]); inner != nil {
		issue = inner
	}
	id, isStr := issue["id"].(string)
	if !isStr {
		return "", errors.New("cockpit issue create returned no id")
	}
	slog.Warn("[paperclip] created fallback FounderOS v1 Cockpit issue", "id", id)
	return remember(id), nil
}

// CockpitThread is the cockpit thread, oldest to newest; limit <= 0 means 50.
func (c *Connector) CockpitThread(ctx context.Context, limit int) ([]Comment, error) {
	if limit <= 0 {
		limit = 50
	}
	id, err := c.EnsureCockpitIssue(ctx)
	if err != nil {
		return nil, err
	}
	body, err := c.get(ctx, fmt.Sprintf("/api/issues/%s/comments?limit=%d", id, limit), issueTimeout)
	if err != nil {
		return nil, err
	}
	out := MapComments(listFrom(body, "comments"))
	sort.SliceStable(out, func(i, j int) bool { return out[i].CreatedAt < out[j].CreatedAt })
	return out, nil
}

// ---- status -----------------------------------------------------------------

func (c *Connector) Status(ctx context.Context) connectors.Status {
	s := connectors.Status{ID: Meta.ID, Name: Meta.Name, Kind: Meta.Kind}
	if _, has := c.creds(); !has {
		s.State = connectors.StateNotConfigured
		s.Detail = connectors.SetKeys("PAPERCLIP_API_URL", "PAPERCLIP_BOARD_KEY", "PAPERCLIP_COMPANY_ID")
		return s
	}
	agents, err := c.Agents(ctx)
	if err == nil && len(agents) == 0 {
		err = errors.New("board reachable but returned no agents")
	}
	if err != nil {
		s.State = connectors.StateError
		s.Detail = "Creds set but board call failed: " + err.Error()
		return s
	}
	running := 0
	for _, a := range agents {
		if a.Status == "running" {
			running++
		}
	}
	s.State = connectors.StateConnected
	s.Detail = fmt.Sprintf("Board live · %d agents · %d running", len(agents), running)
	s.Meta = map[string]any{"agents": len(agents), "running": running}
	return s
}

// ---- guarded writes ---------------------------------------------------------

// CreateIssue creates a task on the board (the Conductor triages it).
func (c *Connector) CreateIssue(ctx context.Context, title, description string) (*Issue, error) {
	cr, has := c.creds()
	if !has {
		return nil, errNoCreds
	}
	body, err := c.send(ctx, "paperclip.issue.create", http.MethodPost, "/api/companies/"+cr.company+"/issues",
		map[string]any{"title": title, "description": description})
	if err != nil {
		return nil, err
	}
	rec := obj(body)
	if inner, has := rec["issue"]; has {
		return first(MapIssues([]any{inner})), nil
	}
	return first(MapIssues([]any{body})), nil
}

func first[T any](xs []T) *T {
	if len(xs) == 0 {
		return nil
	}
	return &xs[0]
}

// InvokeHeartbeat triggers a heartbeat run for a board agent.
func (c *Connector) InvokeHeartbeat(ctx context.Context, agentID string) (bool, error) {
	_, err := c.send(ctx, "paperclip.heartbeat.invoke", http.MethodPost, "/api/agents/"+agentID+"/heartbeat/invoke", map[string]any{})
	return err == nil, err
}

// PostCockpitMessage sends the operator's message into the cockpit thread (wakes
// the Conductor).
func (c *Connector) PostCockpitMessage(ctx context.Context, message string) (*Comment, error) {
	id, err := c.EnsureCockpitIssue(ctx)
	if err != nil {
		return nil, err
	}
	body, err := c.send(ctx, "paperclip.comment.create", http.MethodPost, "/api/issues/"+id+"/comments", map[string]any{"body": message})
	if err != nil {
		return nil, fmt.Errorf("comment failed: %w", err)
	}
	rec := obj(body)
	if inner, has := rec["comment"]; has {
		return first(MapComments([]any{inner})), nil
	}
	return first(MapComments([]any{body})), nil
}

// PostOnce puts a note on the cockpit thread unless a comment already carries
// marker, so a five-minute tick cannot nag. It returns posted, already or
// failed (the TS postFailoverAlert, with the rendering left to the caller).
func (c *Connector) PostOnce(ctx context.Context, marker, message string) string {
	thread, err := c.CockpitThread(ctx, 50)
	if err != nil {
		return "failed"
	}
	for _, cm := range thread {
		if strings.Contains(cm.Body, marker) {
			return "already"
		}
	}
	if posted, err := c.PostCockpitMessage(ctx, message); err != nil || posted == nil {
		return "failed"
	}
	return "posted"
}

// SetAgentModel repoints one seat at another model. The board merges
// adapterConfig, so this cannot clobber the seat's other settings.
func (c *Connector) SetAgentModel(ctx context.Context, agentID, model string) (bool, error) {
	_, err := c.send(ctx, "paperclip.agent.update", http.MethodPatch, "/api/agents/"+agentID,
		map[string]any{"adapterConfig": map[string]any{"model": model}})
	return err == nil, err
}

// ReassignCockpitIssue hands the cockpit thread to another CEO seat: the
// board allows one assignee, and a comment wakes that assignee.
func (c *Connector) ReassignCockpitIssue(ctx context.Context, agentID string) (bool, error) {
	if _, has := c.creds(); !has {
		return false, errNoCreds
	}
	id, err := c.EnsureCockpitIssue(ctx)
	if err != nil {
		return false, err
	}
	_, err = c.send(ctx, "paperclip.issue.update", http.MethodPatch, "/api/issues/"+id,
		map[string]any{"assigneeAgentId": agentID, "assigneeUserId": nil})
	return err == nil, err
}

// ClearAgentError takes a seat out of `error` so its timer schedules it again.
func (c *Connector) ClearAgentError(ctx context.Context, agentID string) (bool, error) {
	_, err := c.send(ctx, "paperclip.agent.clear_error", http.MethodPost, "/api/agents/"+agentID+"/clear-error", map[string]any{})
	return err == nil, err
}

// RepairCockpitIssue puts the open cockpit issue back into the loop-proof
// shape, repairing toward activeCeoID when given (so a standby handoff is not
// undone), else toward the Conductor. It returns ok, patched, failed or skipped.
func (c *Connector) RepairCockpitIssue(ctx context.Context, activeCeoID string) string {
	if _, has := c.creds(); !has {
		return "skipped"
	}
	id := c.pinnedCockpit()
	if id == "" {
		found, err := c.findCockpitByScan(ctx)
		if err != nil {
			return "failed"
		}
		id = found
	}
	if id == "" {
		return "ok"
	}
	body, err := c.get(ctx, "/api/issues/"+id, issueTimeout)
	if err != nil {
		return "failed"
	}
	issue := obj(body)
	if inner := obj(issue["issue"]); inner != nil {
		issue = inner
	}
	if _, isStr := issue["id"].(string); !isStr {
		return "ok"
	}
	conductor := activeCeoID
	if conductor == "" {
		if agents, err := c.Agents(ctx); err == nil {
			for _, a := range agents {
				if strings.EqualFold(a.Name, "conductor") {
					conductor = a.ID
					break
				}
			}
		}
	}
	s := func(k string) string { v, _ := issue[k].(string); return v }
	patch := CockpitRepairPatch(s("status"), s("assigneeAgentId"), s("assigneeUserId"), conductor)
	if patch == nil {
		return "ok"
	}
	if _, err := c.send(ctx, "paperclip.issue.update", http.MethodPatch, "/api/issues/"+s("id"), patch); err != nil {
		return "failed"
	}
	return "patched"
}
