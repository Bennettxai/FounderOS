package api

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/rhl/businessos-backend/internal/founderos/connectors/paperclip"
	"github.com/rhl/businessos-backend/internal/founderos/pages/chats"
	"github.com/rhl/businessos-backend/internal/founderos/pages/osdata"
)

type chatsFakeLLM struct {
	err     error
	reply   string
	systems []string
}

func (f *chatsFakeLLM) Chat(_ context.Context, system string, msgs []chats.Turn) (string, error) {
	f.systems = append(f.systems, system)
	if f.err != nil {
		return "", f.err
	}
	if strings.HasPrefix(system, "You are the Conductor") {
		return "crm-pulse", nil
	}
	return f.reply + msgs[len(msgs)-1].Content, nil
}

type chatsView struct {
	Summaries      []chats.Summary `json:"summaries"`
	Roster         []chats.Agent   `json:"roster"`
	ConductorModel *string         `json:"conductorModel"`
	BoardURL       *string         `json:"boardUrl"`
	LLM            struct {
		Configured bool   `json:"configured"`
		Detail     string `json:"detail"`
	} `json:"llm"`
}

type chatPost struct {
	Messages   []osdata.AgentMessage `json:"messages"`
	Reply      string                `json:"reply"`
	RoutedTo   string                `json:"routedTo"`
	Configured *bool                 `json:"configured"`
	Error      string                `json:"error"`
}

func TestChatsHubNotConfigured(t *testing.T) {
	d := pageDeps(t, &fakeBoard{down: true})
	SetChatLLM(d, chats.Gateway{Key: func() string { return "" }})
	var v chatsView
	if code := getJSON(t, d, http.MethodGet, "/api/founderos/pages/chats", nil, &v); code != 200 {
		t.Fatalf("view %d", code)
	}
	if v.Summaries == nil || len(v.Summaries) != 0 || v.ConductorModel != nil || v.LLM.Configured || !strings.Contains(v.LLM.Detail, "AI_GATEWAY_API_KEY") {
		t.Fatalf("view %+v", v)
	}
	// the roster is every agent but the Conductor (it is the pinned board thread)
	if len(v.Roster) != 3 {
		t.Fatalf("roster %+v", v.Roster)
	}
	for _, a := range v.Roster {
		if a.ID == "conductor" || a.Name == "" {
			t.Fatalf("roster row %+v", a)
		}
	}
	var th chatPost
	if code := getJSON(t, d, http.MethodGet, "/api/founderos/pages/chats/crm-pulse", nil, &th); code != 200 || th.Messages == nil || len(th.Messages) != 0 {
		t.Fatalf("empty thread %d %+v", code, th)
	}
	if code := getJSON(t, d, http.MethodGet, "/api/founderos/pages/chats/nope", nil, nil); code != 404 {
		t.Fatalf("unknown agent %d", code)
	}
	// no key: 200, honest, nothing persisted (nothing was sent)
	var p chatPost
	if code := getJSON(t, d, http.MethodPost, "/api/founderos/pages/chats/crm-pulse", []byte(`{"message":"hi"}`), &p); code != 200 {
		t.Fatalf("unconfigured send must be 200, got %d", code)
	}
	if p.Configured == nil || *p.Configured || !strings.Contains(p.Error, "AI_GATEWAY_API_KEY") || len(p.Messages) != 0 {
		t.Fatalf("unconfigured send %+v", p)
	}
	if code := getJSON(t, d, http.MethodPost, "/api/founderos/pages/chats/conductor", []byte(`{"message":"hi"}`), &p); code != 200 || *p.Configured {
		t.Fatalf("unconfigured conductor send %d %+v", code, p)
	}
	if code := getJSON(t, d, http.MethodPost, "/api/founderos/pages/chats/crm-pulse", []byte(`{"message":"  "}`), nil); code != 400 {
		t.Fatalf("empty message %d", code)
	}
	if code := getJSON(t, d, http.MethodPost, "/api/founderos/pages/chats/nope", []byte(`{"message":"hi"}`), nil); code != 404 {
		t.Fatalf("unknown agent send %d", code)
	}
}

func TestChatsHubConfigured(t *testing.T) {
	model := "opus"
	d := pageDeps(t, &fakeBoard{agents: []paperclip.Agent{{Name: "Conductor", Model: &model}}})
	llm := &chatsFakeLLM{reply: "echo: "}
	SetChatLLM(d, llm)
	var p chatPost
	if code := getJSON(t, d, http.MethodPost, "/api/founderos/pages/chats/sales-agent", []byte(`{"message":"pipeline?"}`), &p); code != 200 {
		t.Fatalf("send %d %+v", code, p)
	}
	if p.Reply != "echo: pipeline?" || len(p.Messages) != 2 || p.Messages[0].Role != "user" || p.Messages[1].Role != "assistant" || p.Configured == nil || !*p.Configured {
		t.Fatalf("send %+v", p)
	}
	if !strings.Contains(llm.systems[0], "You are Sales Agent") {
		t.Fatalf("system %q", llm.systems[0])
	}
	// the Conductor routes: the model picks crm-pulse, the thread lands there
	if code := getJSON(t, d, http.MethodPost, "/api/founderos/pages/chats/conductor", []byte(`{"message":"clean the crm"}`), &p); code != 200 || p.RoutedTo != "crm-pulse" || len(p.Messages) != 2 {
		t.Fatalf("conductor %d %+v", code, p)
	}
	var v chatsView
	getJSON(t, d, http.MethodGet, "/api/founderos/pages/chats", nil, &v)
	if len(v.Summaries) != 2 || v.Summaries[0].AgentID != "crm-pulse" || v.Summaries[1].AgentName != "Sales Agent" || v.ConductorModel == nil || *v.ConductorModel != "opus" {
		t.Fatalf("view %+v", v)
	}
	// a configured model that fails is the upstream's failure, not a fake reply
	llm.err = errors.New("gateway 500: boom")
	if code := getJSON(t, d, http.MethodPost, "/api/founderos/pages/chats/sales-agent", []byte(`{"message":"again"}`), &p); code != http.StatusBadGateway || !strings.Contains(p.Error, "boom") {
		t.Fatalf("upstream failure %d %+v", code, p)
	}
}
