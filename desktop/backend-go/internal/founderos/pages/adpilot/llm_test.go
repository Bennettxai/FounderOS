package adpilot

import (
	"context"
	"errors"
	"strings"
	"testing"
)

type fakeLLM struct {
	reply   string
	err     error
	prompts []string
}

func (f *fakeLLM) Complete(_ context.Context, prompt string) (string, error) {
	f.prompts = append(f.prompts, prompt)
	return f.reply, f.err
}

func TestAskPromptGroundsTheQuestionInTheStore(t *testing.T) {
	w := ToWallAd(mk("a", true, 40, "AI agents replace your staff"), "Acme")
	w.Drivers = map[string]float64{"urgency": 8, "trust": 4, "fear": 6, "joy": 1}
	p := AskPrompt("what works?", []WallAd{w}, []string{"Acme launched x"})
	for _, want := range []string{
		"You are Adscout, the user's ad-intelligence analyst.",
		`"brand":"Acme","hook":"AI agents replace your staff","daysRunning":40,"live":true,"format":null,"topDrivers":["urgency:8","fear:6","trust:4"]`,
		`DATA: recent signals: ["Acme launched x"]`,
		"QUESTION: what works?",
	} {
		if !strings.Contains(p, want) {
			t.Fatalf("prompt lacks %q:\n%s", want, p)
		}
	}
}

func TestExpandProbesUsesTheModelThenFallsBack(t *testing.T) {
	ctx := context.Background()
	llm := &fakeLLM{reply: "```json\n[\"ai agents\",\"ai staff\",\"hire ai\"]\n```"}
	probes, by := ExpandProbes(ctx, llm, "ai employees")
	if by != "claude" || len(probes) != 3 || !strings.Contains(llm.prompts[0], `Concept: "ai employees"`) {
		t.Fatalf("%q %s", probes, by)
	}
	for _, bad := range []*fakeLLM{{reply: "not json"}, {reply: `["only one"]`}, {err: errors.New("down")}} {
		probes, by = ExpandProbes(ctx, bad, "founders replacing staff with AI")
		if by != "fallback" || probes[0] != "founders replacing staff with AI" {
			t.Fatalf("fallback: %q %s", probes, by)
		}
	}
	if _, by := ExpandProbes(ctx, nil, "ai employees"); by != "fallback" {
		t.Fatal("no model → fallback")
	}
}
