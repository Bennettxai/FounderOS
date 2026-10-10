package api

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/rhl/businessos-backend/internal/founderos/pages/chats"
	"github.com/rhl/businessos-backend/internal/founderos/pages/osdata"
)

// /chats (FounderOS v1 app/chats/page.tsx + /api/agents/[id]/chat): the chat
// hub's view (conversations, roster, the board Conductor's model), one
// agent's stored thread, and a direct send. "conductor" as the id routes the
// message to the best-fit agent (v1 routeConductorMessage). With no
// AI_GATEWAY_API_KEY a send is a 200 with configured:false and nothing is
// stored, never a 5xx.

func init() { RegisterPage(registerChats) }

var chatLLMs sync.Map // *Deps → chats.LLM

// chatLLMFor is the hub's model: the AI Gateway keyed through the resolver.
func chatLLMFor(d *Deps) chats.LLM {
	if l, ok := chatLLMs.Load(d); ok {
		return l.(chats.LLM)
	}
	g := chats.Gateway{Key: func() string { return resolve(d, chats.GatewayKey) }, Model: resolve(d, "LLM_MODEL")}
	l, _ := chatLLMs.LoadOrStore(d, chats.LLM(g))
	return l.(chats.LLM)
}

// SetChatLLM swaps the hub's model (tests).
func SetChatLLM(d *Deps, l chats.LLM) { chatLLMs.Store(d, l) }

// chatLLMStatus mirrors v1 llmStatus for the view: configured or the hint.
func chatLLMStatus(l chats.LLM) (bool, string) {
	if g, ok := l.(chats.Gateway); ok && !g.Configured() {
		return false, "Set AI_GATEWAY_API_KEY in ~/.founderos/.env to enable agent chat via the Vercel AI Gateway."
	}
	return true, "Vercel AI Gateway"
}

func chatRoster(ctx context.Context, d *Deps) ([]chats.Agent, map[string]string, error) {
	list, err := storeFor(d).Agents(ctx)
	if err != nil {
		return nil, nil, err
	}
	roster := make([]chats.Agent, 0, len(list))
	names := map[string]string{}
	for _, a := range list {
		names[a.ID] = a.Name
		roster = append(roster, chats.Agent{ID: a.ID, Name: a.Name, Description: a.Description})
	}
	return roster, names, nil
}

func registerChats(s *gin.RouterGroup, d *Deps) {
	s.GET("/pages/chats", func(c *gin.Context) {
		ctx := c.Request.Context()
		roster, names, err := chatRoster(ctx, d)
		if err != nil {
			dataErr(c, err)
			return
		}
		msgs, err := storeFor(d).AgentMessages(ctx, 500)
		if err != nil {
			dataErr(c, err)
			return
		}
		direct := make([]chats.Agent, 0, len(roster))
		for _, a := range roster {
			if a.ID != "conductor" {
				direct = append(direct, a)
			}
		}
		var model *string
		bctx, cancel := context.WithTimeout(ctx, contextBudget)
		defer cancel()
		if agents, err := boardFor(d).Agents(bctx); err == nil {
			for _, a := range agents {
				if a.Name == "Conductor" && a.Model != nil {
					m := *a.Model
					model = &m
				}
			}
		}
		ok, detail := chatLLMStatus(chatLLMFor(d))
		c.JSON(http.StatusOK, gin.H{
			"summaries":      chats.Summaries(msgs, names),
			"roster":         direct,
			"conductorModel": model,
			"boardUrl":       optional(resolve(d, "PAPERCLIP_API_URL")),
			"llm":            gin.H{"configured": ok, "detail": detail},
		})
	})

	known := func(c *gin.Context, id string) ([]chats.Agent, bool) {
		roster, _, err := chatRoster(c.Request.Context(), d)
		if err != nil {
			dataErr(c, err)
			return nil, false
		}
		if id == "conductor" {
			return roster, true
		}
		for _, a := range roster {
			if a.ID == id {
				return roster, true
			}
		}
		c.JSON(http.StatusNotFound, gin.H{"error": "unknown agent: " + id})
		return nil, false
	}

	s.GET("/pages/chats/:id", func(c *gin.Context) {
		id := c.Param("id")
		if _, ok := known(c, id); !ok {
			return
		}
		msgs, err := chats.Store{Pool: d.Pool}.Thread(c.Request.Context(), id)
		if err != nil {
			dataErr(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"messages": msgs})
	})

	s.POST("/pages/chats/:id", func(c *gin.Context) {
		var body struct {
			Message string `json:"message"`
		}
		_ = c.ShouldBindJSON(&body)
		message := strings.TrimSpace(body.Message)
		if message == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "message is required"})
			return
		}
		id := c.Param("id")
		roster, ok := known(c, id)
		if !ok {
			return
		}
		ctx := c.Request.Context()
		llm := chatLLMFor(d)
		st := chats.Store{Pool: d.Pool}
		notConfigured := func(target string) {
			msgs, _ := st.Thread(ctx, target)
			if msgs == nil {
				msgs = []osdata.AgentMessage{}
			}
			c.JSON(http.StatusOK, gin.H{"configured": false, "error": chats.ErrNotConfigured.Error(), "messages": msgs})
		}
		target, delivered := id, message
		if id == "conductor" {
			var err error
			target, delivered, err = chats.Route(ctx, llm, roster, message)
			if errors.Is(err, chats.ErrNotConfigured) {
				notConfigured("conductor")
				return
			}
			if err != nil {
				c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
				return
			}
		}
		var agent chats.Agent
		for _, a := range roster {
			if a.ID == target {
				agent = a
			}
		}
		history, err := st.Thread(ctx, target)
		if err != nil {
			dataErr(c, err)
			return
		}
		turns := make([]chats.Turn, 0, len(history)+1)
		for _, m := range history {
			turns = append(turns, chats.Turn{Role: m.Role, Content: m.Content})
		}
		turns = append(turns, chats.Turn{Role: "user", Content: delivered})
		asked := time.Now()
		reply, err := llm.Chat(ctx, chats.SystemPrompt(agent), turns)
		if errors.Is(err, chats.ErrNotConfigured) {
			notConfigured(target)
			return
		}
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
			return
		}
		if err := st.Append(ctx, target, "user", delivered, asked); err != nil {
			dataErr(c, err)
			return
		}
		if err := st.Append(ctx, target, "assistant", reply, time.Now()); err != nil {
			dataErr(c, err)
			return
		}
		msgs, err := st.Thread(ctx, target)
		if err != nil {
			dataErr(c, err)
			return
		}
		out := gin.H{"configured": true, "reply": reply, "messages": msgs}
		if id == "conductor" {
			out["routedTo"] = target
		}
		c.JSON(http.StatusOK, out)
	})
}
