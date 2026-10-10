// Package optimalengine is a typed Go client for the OptimalEngine HTTP API
// (the Elixir process that ships at optimal-engine/ and runs on :4200).
//
// The engine exposes a rich surface — memory, RAG, recall, subscriptions,
// auth keys, wiki, etc. This client wraps the endpoints BusinessOS uses
// against typed structs so handlers don't have to hand-roll JSON
// marshalling at every call site.
//
// Construction is nil-safe: NewClient("") returns a client whose every
// method is a silent no-op so callers don't need to nil-check when the
// engine isn't wired (e.g. in unit tests or when OPTIMAL_ENGINE_URL
// isn't set).
package optimalengine

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Client talks to a running OptimalEngine over HTTP. Zero-value Client is
// disabled (every call returns nil/no-op). Use NewClient with a non-empty
// base URL to actually issue requests.
type Client struct {
	baseURL string
	http    *http.Client
	// apiKey, when non-empty, is sent as Authorization: Bearer <apiKey>.
	// The engine's AuthPlug allows /api/health, /api/optimal/*, and
	// /api/reindex without auth, so the BO server can run without an
	// API key for the routes it currently uses. Required for memory
	// writes once the upstream gates them.
	apiKey string
}

// NewClient builds a client. Pass an empty baseURL to get a no-op stub
// (useful in tests or when the engine isn't running).
func NewClient(baseURL string) *Client {
	if baseURL == "" {
		return &Client{}
	}
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		http: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// SetAPIKey configures the bearer token sent on every request.
func (c *Client) SetAPIKey(key string) {
	if c == nil {
		return
	}
	c.apiKey = key
}

// Enabled reports whether the client will actually issue requests.
func (c *Client) Enabled() bool {
	return c != nil && c.baseURL != ""
}

// ----- Memory -----------------------------------------------------------------

// Memory mirrors the engine's memory_to_map response. Optional fields use
// pointers so we can distinguish "not set" from zero values.
type Memory struct {
	ID             string                 `json:"id"`
	TenantID       string                 `json:"tenant_id,omitempty"`
	WorkspaceID    string                 `json:"workspace_id"`
	Content        string                 `json:"content"`
	IsStatic       bool                   `json:"is_static"`
	IsForgotten    bool                   `json:"is_forgotten"`
	ForgetAfter    *string                `json:"forget_after,omitempty"`
	ForgetReason   *string                `json:"forget_reason,omitempty"`
	Version        int                    `json:"version"`
	ParentMemoryID *string                `json:"parent_memory_id,omitempty"`
	RootMemoryID   *string                `json:"root_memory_id,omitempty"`
	IsLatest       bool                   `json:"is_latest"`
	CitationURI    *string                `json:"citation_uri,omitempty"`
	SourceChunkID  *string                `json:"source_chunk_id,omitempty"`
	Audience       *string                `json:"audience,omitempty"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt      string                 `json:"created_at"`
	UpdatedAt      string                 `json:"updated_at"`
	WasExisting    bool                   `json:"was_existing,omitempty"`
}

// CreateMemoryRequest is the body for POST /api/memory.
type CreateMemoryRequest struct {
	Content       string                 `json:"content"`
	Workspace     string                 `json:"workspace,omitempty"`
	IsStatic      *bool                  `json:"is_static,omitempty"`
	Audience      string                 `json:"audience,omitempty"`
	CitationURI   string                 `json:"citation_uri,omitempty"`
	SourceChunkID string                 `json:"source_chunk_id,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// CreateMemory writes a memory to the engine. Returns the canonical
// memory record (which may be a duplicate-detected existing one — check
// WasExisting). No-op when the client isn't enabled.
func (c *Client) CreateMemory(ctx context.Context, req CreateMemoryRequest) (*Memory, error) {
	if !c.Enabled() {
		return nil, nil
	}
	var out Memory
	if err := c.do(ctx, http.MethodPost, "/api/memory", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ListMemoriesOptions narrows GET /api/memory.
type ListMemoriesOptions struct {
	Workspace          string
	Audience           string
	Limit              int
	IncludeForgotten   bool
	IncludeOldVersions bool
}

// ListMemoriesResponse is the engine's GET /api/memory shape.
type ListMemoriesResponse struct {
	WorkspaceID string   `json:"workspace_id"`
	Count       int      `json:"count"`
	Memories    []Memory `json:"memories"`
}

func (c *Client) ListMemories(ctx context.Context, opts ListMemoriesOptions) (*ListMemoriesResponse, error) {
	if !c.Enabled() {
		return nil, nil
	}
	q := url.Values{}
	if opts.Workspace != "" {
		q.Set("workspace", opts.Workspace)
	}
	if opts.Audience != "" {
		q.Set("audience", opts.Audience)
	}
	if opts.Limit > 0 {
		q.Set("limit", strconv.Itoa(opts.Limit))
	}
	if opts.IncludeForgotten {
		q.Set("include_forgotten", "true")
	}
	if opts.IncludeOldVersions {
		q.Set("include_old_versions", "true")
	}
	var out ListMemoriesResponse
	if err := c.do(ctx, http.MethodGet, "/api/memory?"+q.Encode(), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ForgetMemory marks a memory forgotten. The engine keeps it around with
// is_forgotten=true so retrieval can audit. reason is optional context.
func (c *Client) ForgetMemory(ctx context.Context, id, reason string) error {
	if !c.Enabled() {
		return nil
	}
	body := map[string]string{}
	if reason != "" {
		body["reason"] = reason
	}
	return c.do(ctx, http.MethodPost, "/api/memory/"+url.PathEscape(id)+"/forget", body, nil)
}

// ----- RAG --------------------------------------------------------------------

// AskRequest is the body for POST /api/rag.
type AskRequest struct {
	Query     string `json:"query"`
	Workspace string `json:"workspace,omitempty"`
	Audience  string `json:"audience,omitempty"`
	Format    string `json:"format,omitempty"`    // markdown | plain | claude | openai
	Bandwidth string `json:"bandwidth,omitempty"` // small | medium | large
}

// AskResponse is the composed envelope returned by the engine. The shape
// is intentionally generic — the engine renders into whatever Format
// the caller requested. We pass it through to BO callers verbatim.
type AskResponse map[string]interface{}

func (c *Client) Ask(ctx context.Context, req AskRequest) (AskResponse, error) {
	if !c.Enabled() {
		return nil, nil
	}
	out := AskResponse{}
	if err := c.do(ctx, http.MethodPost, "/api/rag", req, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// ----- Recall -----------------------------------------------------------------

// RecallActions returns "what's been done about <topic>, by <actor>, since <when>".
// Empty params skip that filter. Workspace defaults to "default" engine-side.
func (c *Client) RecallActions(ctx context.Context, actor, topic, since, workspace string) (AskResponse, error) {
	return c.recall(ctx, "/api/recall/actions", url.Values{
		"actor":     {actor},
		"topic":     {topic},
		"since":     {since},
		"workspace": {workspace},
	})
}

// RecallWho — owner / contact lookup for a topic.
func (c *Client) RecallWho(ctx context.Context, topic, role, workspace string) (AskResponse, error) {
	return c.recall(ctx, "/api/recall/who", url.Values{
		"topic":     {topic},
		"role":      {role},
		"workspace": {workspace},
	})
}

// RecallWhen — schedule / temporal lookup for an event.
func (c *Client) RecallWhen(ctx context.Context, event, workspace string) (AskResponse, error) {
	return c.recall(ctx, "/api/recall/when", url.Values{
		"event":     {event},
		"workspace": {workspace},
	})
}

// RecallWhere — object location lookup.
func (c *Client) RecallWhere(ctx context.Context, thing, workspace string) (AskResponse, error) {
	return c.recall(ctx, "/api/recall/where", url.Values{
		"thing":     {thing},
		"workspace": {workspace},
	})
}

// RecallOwns — open commitments / tasks owned by an actor.
func (c *Client) RecallOwns(ctx context.Context, actor, workspace string) (AskResponse, error) {
	return c.recall(ctx, "/api/recall/owns", url.Values{
		"actor":     {actor},
		"workspace": {workspace},
	})
}

func (c *Client) recall(ctx context.Context, path string, params url.Values) (AskResponse, error) {
	if !c.Enabled() {
		return nil, nil
	}
	// Drop empty params so the engine sees a clean signature.
	q := url.Values{}
	for k, vs := range params {
		for _, v := range vs {
			if v != "" {
				q.Set(k, v)
			}
		}
	}
	out := AskResponse{}
	if err := c.do(ctx, http.MethodGet, path+"?"+q.Encode(), nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// ----- low-level --------------------------------------------------------------

func (c *Client) do(ctx context.Context, method, path string, body, into interface{}) error {
	var buf io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("optimalengine: marshal %s %s: %w", method, path, err)
		}
		buf = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, buf)
	if err != nil {
		return fmt.Errorf("optimalengine: build %s %s: %w", method, path, err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("optimalengine: %s %s: %w", method, path, err)
	}
	defer resp.Body.Close()

	// 2xx = success. The engine returns 201 from POST /api/memory and
	// 200 from most reads; anything else we treat as a failure and
	// surface the body so callers can log it.
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		if into == nil {
			io.Copy(io.Discard, resp.Body)
			return nil
		}
		return json.NewDecoder(resp.Body).Decode(into)
	}

	body4xx, _ := io.ReadAll(resp.Body)
	return fmt.Errorf("optimalengine: %s %s: %d %s", method, path, resp.StatusCode, strings.TrimSpace(string(body4xx)))
}
