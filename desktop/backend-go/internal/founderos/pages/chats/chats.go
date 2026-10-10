// Package chats is the /chats hub (FounderOS v1 app/chats/page.tsx,
// lib/chats.ts, lib/agents/chat.ts, lib/agents/conductor.ts): every agent
// conversation in one rail, a direct read-only line to any agent, and the
// Conductor's routing. The model is a seam (LLM); with no key it is honestly
// not configured, never faked.
package chats

import (
	"context"
	"regexp"
	"sort"
	"strings"

	"github.com/rhl/businessos-backend/internal/founderos/pages/osdata"
)

// Agent is one roster row the hub can talk to.
type Agent struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// Summary is one conversation row in the rail (v1 ConversationSummary).
type Summary struct {
	AgentID      string `json:"agentId"`
	AgentName    string `json:"agentName"`
	LastMessage  string `json:"lastMessage"`
	LastAt       string `json:"lastAt"`
	MessageCount int    `json:"messageCount"`
}

const previewMax = 140

var spaces = regexp.MustCompile(`\s+`)

// Preview is one line, capped: a list-row preview, never a wall of text.
func Preview(content string) string {
	line := strings.TrimSpace(spaces.ReplaceAllString(content, " "))
	if r := []rune(line); len(r) > previewMax {
		return string(r[:previewMax-1]) + "…"
	}
	return line
}

// Summaries groups messages into one conversation per agent, newest activity first.
func Summaries(messages []osdata.AgentMessage, names map[string]string) []Summary {
	by := map[string][]osdata.AgentMessage{}
	var order []string
	for _, m := range messages {
		if _, ok := by[m.AgentID]; !ok {
			order = append(order, m.AgentID)
		}
		by[m.AgentID] = append(by[m.AgentID], m)
	}
	out := make([]Summary, 0, len(by))
	for _, id := range order {
		rows := by[id]
		last := rows[0]
		for _, r := range rows[1:] {
			if r.CreatedAt >= last.CreatedAt {
				last = r
			}
		}
		name := names[id]
		if name == "" {
			name = id
		}
		out = append(out, Summary{AgentID: id, AgentName: name, LastMessage: Preview(last.Content), LastAt: last.CreatedAt, MessageCount: len(rows)})
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].LastAt > out[j].LastAt })
	return out
}

// SystemPrompt is the agent's system prompt (v1 systemPromptFor, no ambient pack).
func SystemPrompt(a Agent) string {
	return strings.Join([]string{
		"You are " + a.Name + ", an operator agent inside Founder OS.",
		a.Description,
		"Answer concisely.",
		"You are READ-ONLY: never claim to have sent, created, scheduled, or published anything — you can only look things up and report.",
	}, "\n")
}

var atPrefix = regexp.MustCompile(`^@(\S+)\s*`)
var nonSlug = regexp.MustCompile(`[^a-z0-9]+`)

func slug(s string) string {
	return strings.Trim(nonSlug.ReplaceAllString(strings.ToLower(s), "-"), "-")
}

var notIDChar = regexp.MustCompile(`[^a-zA-Z0-9_-]`)

// Route picks the agent a Conductor message goes to (v1 routeConductorMessage):
// a leading @id or @name wins and is stripped; otherwise the model picks the
// single best fit; an unknown pick falls back to the first routable agent.
func Route(ctx context.Context, llm LLM, roster []Agent, message string) (string, string, error) {
	var routable []Agent
	for _, a := range roster {
		if a.ID != "conductor" {
			routable = append(routable, a)
		}
	}
	if len(routable) == 0 {
		return "", message, ErrNoAgents
	}
	if m := atPrefix.FindStringSubmatch(message); m != nil {
		t := slug(m[1])
		for _, a := range routable {
			if a.ID == m[1] || a.ID == t || slug(a.Name) == t {
				body := strings.TrimSpace(atPrefix.ReplaceAllString(message, ""))
				if body == "" {
					body = message
				}
				return a.ID, body, nil
			}
		}
	}
	lines := make([]string, 0, len(routable))
	for _, a := range routable {
		lines = append(lines, "- "+a.ID+": "+a.Name+" — "+a.Description)
	}
	system := strings.Join([]string{
		"You are the Conductor, the router for Founder OS operator agents.",
		"Pick the single best-fit agent for the user message.",
		"Reply with ONLY that agent id and nothing else. Options:",
		strings.Join(lines, "\n"),
	}, "\n")
	text, err := llm.Chat(ctx, system, []Turn{{Role: "user", Content: message}})
	if err != nil {
		return "", message, err
	}
	fields := strings.Fields(text)
	picked := ""
	if len(fields) > 0 {
		picked = notIDChar.ReplaceAllString(fields[0], "")
	}
	for _, a := range routable {
		if a.ID == picked {
			return a.ID, message, nil
		}
	}
	return routable[0].ID, message, nil
}
