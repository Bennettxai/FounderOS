package chats

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/rhl/businessos-backend/internal/founderos/pages/osdata"
)

func msg(id, agent, role, content, at string) osdata.AgentMessage {
	return osdata.AgentMessage{ID: id, AgentID: agent, Role: role, Content: content, CreatedAt: at, ToolCalls: []osdata.ToolCallBrief{}}
}

// FounderOS v1 lib/chats.ts conversationSummaries.
func TestSummariesOnePerAgentNewestFirst(t *testing.T) {
	long := strings.Repeat("word ", 60)
	got := Summaries([]osdata.AgentMessage{
		msg("1", "social-agent", "user", "hi", "2026-10-01T10:00:00Z"),
		msg("2", "social-agent", "assistant", "  hello\n\nthere  ", "2026-10-01T10:01:00Z"),
		msg("3", "gmail-worker", "assistant", long, "2026-10-02T09:00:00Z"),
		msg("4", "ghost", "user", "who", "2026-09-30T09:00:00Z"),
	}, map[string]string{"social-agent": "Social Agent", "gmail-worker": "Gmail Worker"})
	if len(got) != 3 {
		t.Fatalf("want 3 conversations, got %+v", got)
	}
	if got[0].AgentID != "gmail-worker" || got[1].AgentID != "social-agent" || got[2].AgentID != "ghost" {
		t.Fatalf("order %+v", got)
	}
	if got[1].AgentName != "Social Agent" || got[1].LastMessage != "hello there" || got[1].MessageCount != 2 || got[1].LastAt != "2026-10-01T10:01:00Z" {
		t.Fatalf("social %+v", got[1])
	}
	if got[2].AgentName != "ghost" {
		t.Fatalf("unknown agent keeps its id as name: %+v", got[2])
	}
	if r := []rune(got[0].LastMessage); len(r) != 140 || !strings.HasSuffix(got[0].LastMessage, "…") {
		t.Fatalf("preview capped at 140 with an ellipsis: %d %q", len(r), got[0].LastMessage)
	}
	if out := Summaries(nil, nil); out == nil || len(out) != 0 {
		t.Fatalf("empty input is an empty list, not nil")
	}
}

var roster = []Agent{
	{ID: "conductor", Name: "Conductor", Description: "routes"},
	{ID: "social-agent", Name: "Social Agent", Description: "posts"},
	{ID: "gmail-worker", Name: "Gmail Worker", Description: "inbox"},
}

type fakeLLM struct {
	replies []string
	err     error
	calls   []string
}

func (f *fakeLLM) Chat(_ context.Context, system string, msgs []Turn) (string, error) {
	f.calls = append(f.calls, system)
	if f.err != nil {
		return "", f.err
	}
	r := f.replies[0]
	if len(f.replies) > 1 {
		f.replies = f.replies[1:]
	}
	return r, nil
}

// FounderOS v1 lib/agents/conductor.ts: an explicit @name wins and is stripped;
// otherwise the model picks; a bad pick or @unknown falls back, never throws.
func TestRouteConductor(t *testing.T) {
	llm := &fakeLLM{replies: []string{"never-called"}}
	id, body, err := Route(context.Background(), llm, roster, "@gmail-worker triage my inbox")
	if err != nil || id != "gmail-worker" || body != "triage my inbox" || len(llm.calls) != 0 {
		t.Fatalf("explicit id: %q %q %v %d", id, body, err, len(llm.calls))
	}
	id, body, _ = Route(context.Background(), llm, roster, "@Social-Agent post it")
	if id != "social-agent" || body != "post it" {
		t.Fatalf("explicit name slug: %q %q", id, body)
	}
	llm = &fakeLLM{replies: []string{" social-agent\n because"}}
	id, body, _ = Route(context.Background(), llm, roster, "@nobody write a hook")
	if id != "social-agent" || body != "@nobody write a hook" || len(llm.calls) != 1 || !strings.Contains(llm.calls[0], "- gmail-worker: Gmail Worker") || strings.Contains(llm.calls[0], "- conductor:") {
		t.Fatalf("model pick: %q %q %v", id, body, llm.calls)
	}
	llm = &fakeLLM{replies: []string{"conductor"}}
	if id, _, _ = Route(context.Background(), llm, roster, "anything"); id != "social-agent" {
		t.Fatalf("a pick outside the routable roster falls back to the first agent: %q", id)
	}
	llm = &fakeLLM{err: ErrNotConfigured}
	if _, _, err = Route(context.Background(), llm, roster, "anything"); !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("an unconfigured model surfaces as ErrNotConfigured: %v", err)
	}
}

func TestSystemPrompt(t *testing.T) {
	p := SystemPrompt(Agent{Name: "Gmail Worker", Description: "inbox"})
	for _, want := range []string{"You are Gmail Worker, an operator agent inside Founder OS.", "inbox", "READ-ONLY"} {
		if !strings.Contains(p, want) {
			t.Fatalf("prompt lacks %q: %s", want, p)
		}
	}
}

func TestGatewayNotConfiguredWithoutKey(t *testing.T) {
	g := Gateway{Key: func() string { return "" }}
	if g.Configured() {
		t.Fatal("no key must read as not configured")
	}
	if _, err := g.Chat(context.Background(), "s", []Turn{{Role: "user", Content: "hi"}}); !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("want ErrNotConfigured, got %v", err)
	}
}

func TestGatewayFallsDownTheModelChain(t *testing.T) {
	var models []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer k" || r.URL.Path != "/v1/chat/completions" {
			w.WriteHeader(400)
			return
		}
		body := make([]byte, r.ContentLength)
		_, _ = r.Body.Read(body)
		switch {
		case strings.Contains(string(body), `"model":"anthropic/x"`):
			models = append(models, "anthropic/x")
			w.WriteHeader(403)
			_, _ = w.Write([]byte(`{"error":{"message":"RestrictedModelsError: upgrade to paid credits"}}`))
		default:
			models = append(models, "free")
			_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"hey there"}}]}`))
		}
	}))
	defer srv.Close()
	g := Gateway{Key: func() string { return "k" }, BaseURL: srv.URL + "/v1", Model: "anthropic/x", Client: srv.Client()}
	got, err := g.Chat(context.Background(), "sys", []Turn{{Role: "user", Content: "hi"}, {Role: "tool", Content: "x"}})
	if err != nil || got != "hey there" || len(models) != 2 {
		t.Fatalf("chain: %q %v %v", got, err, models)
	}
}
