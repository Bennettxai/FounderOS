package chats

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Turn is one chat message sent to the model.
type Turn struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// LLM is the model seam the hub talks through; tests inject a fake.
type LLM interface {
	Chat(ctx context.Context, system string, msgs []Turn) (string, error)
}

// ErrNotConfigured: no gateway key, so agent chat is off (v1 llmStatus
// not_configured). The routes answer it with a 200 not-configured body.
var ErrNotConfigured = errors.New("AI_GATEWAY_API_KEY is not set — add it to ~/.founderos/.env to enable agent chat.")

// ErrNoAgents: the roster is empty (the workspace is not seeded).
var ErrNoAgents = errors.New("no agents on the roster")

const (
	GatewayKey     = "AI_GATEWAY_API_KEY"
	defaultBaseURL = "https://ai-gateway.vercel.sh/v1"
	fallbackModel  = "anthropic/claude-sonnet-5"
)

// FreeTierModels answer on a gateway key without paid credits (v1 FREE_TIER_MODELS).
var FreeTierModels = []string{"openai/gpt-oss-20b", "alibaba/qwen-3-14b"}

// Gateway is the production LLM: the Vercel AI Gateway's OpenAI-compatible
// chat endpoint, the preferred model first, then the free-tier chain when the
// gateway refuses a model (v1 createGatewayProvider + modelChain).
type Gateway struct {
	Key     func() string
	BaseURL string
	Model   string
	Client  *http.Client
}

func (g Gateway) key() string {
	if g.Key == nil {
		return ""
	}
	return strings.TrimSpace(g.Key())
}

// Configured reports whether a key resolves (read at call time).
func (g Gateway) Configured() bool { return g.key() != "" }

func (g Gateway) chain() []string {
	pref := g.Model
	if pref == "" {
		pref = fallbackModel
	}
	out := []string{pref}
	for _, m := range FreeTierModels {
		if m != pref {
			out = append(out, m)
		}
	}
	return out
}

type gatewayErr struct {
	status int
	msg    string
}

func (e *gatewayErr) Error() string { return fmt.Sprintf("gateway %d: %s", e.status, e.msg) }

// modelUnavailable: a refusal a different model would survive (v1 isModelUnavailableError).
func modelUnavailable(err error) bool {
	var ge *gatewayErr
	if !errors.As(err, &ge) {
		return false
	}
	m := strings.ToLower(ge.msg)
	if strings.Contains(m, "api_key") || strings.Contains(m, "api key") || strings.Contains(m, "rate limit") {
		return false
	}
	if strings.Contains(m, "restrictedmodels") || strings.Contains(m, "do not have access to this model") || strings.Contains(m, "upgrade to paid credits") {
		return true
	}
	return ge.status == 403 || ge.status == 404
}

func (g Gateway) Chat(ctx context.Context, system string, msgs []Turn) (string, error) {
	key := g.key()
	if key == "" {
		return "", ErrNotConfigured
	}
	body := []Turn{{Role: "system", Content: system}}
	for _, m := range msgs {
		// prior tool turns stay in the record, not in the prompt (v1)
		if m.Role == "user" || m.Role == "assistant" {
			body = append(body, m)
		}
	}
	var last error
	for _, model := range g.chain() {
		text, err := g.once(ctx, key, model, body)
		if err == nil {
			return text, nil
		}
		last = err
		if !modelUnavailable(err) {
			return "", err
		}
	}
	return "", last
}

func (g Gateway) once(ctx context.Context, key, model string, msgs []Turn) (string, error) {
	base := g.BaseURL
	if base == "" {
		base = defaultBaseURL
	}
	client := g.Client
	if client == nil {
		client = &http.Client{Timeout: 90 * time.Second}
	}
	payload, _ := json.Marshal(map[string]any{"model": model, "messages": msgs})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(base, "/")+"/chat/completions", bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+key)
	req.Header.Set("Content-Type", "application/json")
	res, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	var out struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	_ = json.Unmarshal(raw, &out)
	if res.StatusCode >= 300 {
		msg := strings.TrimSpace(string(raw))
		if out.Error != nil && out.Error.Message != "" {
			msg = out.Error.Message
		}
		return "", &gatewayErr{status: res.StatusCode, msg: msg}
	}
	if len(out.Choices) == 0 {
		return "", errors.New("gateway answered with no choices")
	}
	return out.Choices[0].Message.Content, nil
}
