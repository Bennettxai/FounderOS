// Package sprites provides a client for the sprites.dev REST API, which manages
// ephemeral and persistent compute instances (Sprites) used to host customer BOS
// environments and to power OSA BUILD-mode sandboxes.
package sprites

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"
)

const (
	defaultBaseURL      = "https://api.sprites.dev/v1"
	defaultTimeout      = 30 * time.Second
	provisioningTimeout = 5 * time.Minute
	defaultContentType  = "application/json"
)

// Sentinel errors for the sprites package.
var (
	// ErrNotFound is returned when the requested sprite does not exist.
	ErrNotFound = errors.New("sprite not found")

	// ErrUnauthorized is returned when the API key is invalid or missing.
	ErrUnauthorized = errors.New("sprites: unauthorized")

	// ErrSpriteNotRunning is returned when an operation requires a running sprite
	// but the target is in a different state.
	ErrSpriteNotRunning = errors.New("sprite is not in running state")
)

// SpriteStatus represents the lifecycle state of a Sprite instance.
type SpriteStatus string

const (
	StatusRunning      SpriteStatus = "running"
	StatusStopped      SpriteStatus = "stopped"
	StatusCheckpointed SpriteStatus = "checkpointed"
	StatusProvisioning SpriteStatus = "provisioning"
	StatusError        SpriteStatus = "error"
)

// Sprite represents a customer Sprite instance returned by the sprites.dev API.
type Sprite struct {
	ID         string       `json:"id"`
	Name       string       `json:"name"`
	Status     SpriteStatus `json:"status"`
	URL        string       `json:"url"` // HTTP access URL for the instance
	CreatedAt  time.Time    `json:"created_at"`
	CustomerID string       `json:"customer_id"` // MIOSA customer mapping (internal)
}

// CreateSpriteRequest is the payload for provisioning a new Sprite instance.
type CreateSpriteRequest struct {
	Name       string            `json:"name"`
	Image      string            `json:"image,omitempty"` // base image tag
	Env        map[string]string `json:"env,omitempty"`   // environment variables
	CustomerID string            `json:"-"`               // internal tracking, not sent to API
}

// CheckpointRequest snapshots a running Sprite under a human-readable tag.
type CheckpointRequest struct {
	SpriteID string `json:"sprite_id"`
	Tag      string `json:"tag"` // e.g. "v1.2.0", "pre-upgrade"
}

// ExecResult holds the output of a command executed inside a Sprite.
type ExecResult struct {
	Output   string `json:"output"`
	ExitCode int    `json:"exit_code"`
	Error    string `json:"error,omitempty"`
}

// apiError is the error envelope returned by the sprites.dev API.
type apiError struct {
	StatusCode int
	Message    string
}

func (e *apiError) Error() string {
	return fmt.Sprintf("sprites api error (status %d): %s", e.StatusCode, e.Message)
}

// SpritesClient wraps the sprites.dev REST API.
// Use NewClient to construct one; the zero value is not safe for use.
type SpritesClient struct {
	apiKey  string
	baseURL string
	http    *http.Client
	logger  *slog.Logger
}

// NewClient creates a SpritesClient with the given API key and a 30-second default
// timeout. Pass a nil *slog.Logger to use the default package-level logger.
func NewClient(apiKey string) *SpritesClient {
	return &SpritesClient{
		apiKey:  apiKey,
		baseURL: defaultBaseURL,
		http:    &http.Client{Timeout: defaultTimeout},
		logger:  slog.Default(),
	}
}

// WithBaseURL overrides the API base URL, useful for testing against a local stub.
func (c *SpritesClient) WithBaseURL(u string) *SpritesClient {
	c.baseURL = u
	return c
}

// WithHTTPClient replaces the internal http.Client, useful for injecting transport
// wrappers (e.g. tracing, custom timeouts for provisioning calls).
func (c *SpritesClient) WithHTTPClient(hc *http.Client) *SpritesClient {
	c.http = hc
	return c
}

// WithLogger attaches a structured logger to the client.
func (c *SpritesClient) WithLogger(l *slog.Logger) *SpritesClient {
	c.logger = l
	return c
}

// ---- CRUD operations ---------------------------------------------------------

// CreateSprite provisions a new Sprite instance.
func (c *SpritesClient) CreateSprite(ctx context.Context, req CreateSpriteRequest) (*Sprite, error) {
	// Use a longer timeout for provisioning calls since they involve container
	// image pulls and init script execution.
	httpCtx, cancel := context.WithTimeout(ctx, provisioningTimeout)
	defer cancel()

	// Strip the internal-only CustomerID field before serialising.
	wire := struct {
		Name  string            `json:"name"`
		Image string            `json:"image,omitempty"`
		Env   map[string]string `json:"env,omitempty"`
	}{
		Name:  req.Name,
		Image: req.Image,
		Env:   req.Env,
	}

	var sprite Sprite
	if err := c.do(httpCtx, http.MethodPost, "/sprites", wire, &sprite); err != nil {
		return nil, fmt.Errorf("create sprite %q: %w", req.Name, err)
	}

	// Surface the caller-supplied CustomerID so callers can correlate without a
	// round-trip; it is not persisted by the API.
	sprite.CustomerID = req.CustomerID

	c.logger.InfoContext(ctx, "sprite created",
		"sprite_id", sprite.ID,
		"name", sprite.Name,
		"customer_id", req.CustomerID,
	)
	return &sprite, nil
}

// GetSprite retrieves a single Sprite by its ID.
func (c *SpritesClient) GetSprite(ctx context.Context, id string) (*Sprite, error) {
	var sprite Sprite
	if err := c.do(ctx, http.MethodGet, "/sprites/"+id, nil, &sprite); err != nil {
		return nil, fmt.Errorf("get sprite %q: %w", id, err)
	}
	return &sprite, nil
}

// DeleteSprite permanently destroys a Sprite instance and all its data.
func (c *SpritesClient) DeleteSprite(ctx context.Context, id string) error {
	if err := c.do(ctx, http.MethodDelete, "/sprites/"+id, nil, nil); err != nil {
		return fmt.Errorf("delete sprite %q: %w", id, err)
	}
	c.logger.InfoContext(ctx, "sprite deleted", "sprite_id", id)
	return nil
}

// ListSprites returns all Sprites, optionally filtered by a MIOSA customer ID.
// Pass an empty customerID to list all Sprites for the API key.
func (c *SpritesClient) ListSprites(ctx context.Context, customerID string) ([]Sprite, error) {
	path := "/sprites"
	if customerID != "" {
		path += "?customer_id=" + customerID
	}

	var result struct {
		Sprites []Sprite `json:"sprites"`
	}
	if err := c.do(ctx, http.MethodGet, path, nil, &result); err != nil {
		return nil, fmt.Errorf("list sprites (customer_id=%q): %w", customerID, err)
	}
	return result.Sprites, nil
}

// GetSpriteURL returns the HTTP access URL for a running Sprite.
// Returns ErrSpriteNotRunning if the sprite is not in the running state.
func (c *SpritesClient) GetSpriteURL(ctx context.Context, spriteID string) (string, error) {
	sprite, err := c.GetSprite(ctx, spriteID)
	if err != nil {
		return "", fmt.Errorf("get sprite url %q: %w", spriteID, err)
	}
	if sprite.Status != StatusRunning {
		return "", fmt.Errorf("get sprite url %q: %w (current status: %s)",
			spriteID, ErrSpriteNotRunning, sprite.Status)
	}
	return sprite.URL, nil
}

// ---- Exec / Checkpoint / Restore ---------------------------------------------

// ExecCommand executes a command inside a running Sprite and returns its output.
func (c *SpritesClient) ExecCommand(ctx context.Context, spriteID string, cmd []string) (*ExecResult, error) {
	payload := struct {
		Command []string `json:"command"`
	}{Command: cmd}

	var result ExecResult
	if err := c.do(ctx, http.MethodPost, "/sprites/"+spriteID+"/exec", payload, &result); err != nil {
		return nil, fmt.Errorf("exec in sprite %q: %w", spriteID, err)
	}
	return &result, nil
}

// Checkpoint snapshots the current state of a Sprite under a named tag.
// The tag can be used later with Restore to roll back to this point.
func (c *SpritesClient) Checkpoint(ctx context.Context, req CheckpointRequest) error {
	payload := struct {
		Tag string `json:"tag"`
	}{Tag: req.Tag}

	if err := c.do(ctx, http.MethodPost, "/sprites/"+req.SpriteID+"/checkpoint", payload, nil); err != nil {
		return fmt.Errorf("checkpoint sprite %q (tag=%q): %w", req.SpriteID, req.Tag, err)
	}
	c.logger.InfoContext(ctx, "sprite checkpointed",
		"sprite_id", req.SpriteID,
		"tag", req.Tag,
	)
	return nil
}

// Restore rolls a Sprite back to a previously created checkpoint tag.
func (c *SpritesClient) Restore(ctx context.Context, spriteID string, tag string) error {
	payload := struct {
		Tag string `json:"tag"`
	}{Tag: tag}

	if err := c.do(ctx, http.MethodPost, "/sprites/"+spriteID+"/restore", payload, nil); err != nil {
		return fmt.Errorf("restore sprite %q to tag %q: %w", spriteID, tag, err)
	}
	c.logger.InfoContext(ctx, "sprite restored",
		"sprite_id", spriteID,
		"tag", tag,
	)
	return nil
}

// ---- internal HTTP helper ----------------------------------------------------

// do is the single choke-point for all HTTP calls. It serialises the request
// body (if non-nil), attaches auth headers, executes the request, checks the
// status code, and deserialises the response into out (if non-nil).
func (c *SpritesClient) do(ctx context.Context, method, path string, body, out any) error {
	var reqBody io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal request body: %w", err)
		}
		reqBody = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reqBody)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", defaultContentType)
	req.Header.Set("Accept", defaultContentType)

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	if err := checkStatus(resp); err != nil {
		return err
	}

	if out != nil && resp.StatusCode != http.StatusNoContent {
		if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
			return fmt.Errorf("decode response: %w", err)
		}
	}
	return nil
}

// checkStatus maps non-2xx HTTP responses to typed errors where possible.
func checkStatus(resp *http.Response) error {
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}

	// Attempt to read the API error body for a descriptive message.
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	msg := string(raw)

	switch resp.StatusCode {
	case http.StatusUnauthorized, http.StatusForbidden:
		return fmt.Errorf("%w: %s", ErrUnauthorized, msg)
	case http.StatusNotFound:
		return fmt.Errorf("%w: %s", ErrNotFound, msg)
	default:
		return &apiError{StatusCode: resp.StatusCode, Message: msg}
	}
}
