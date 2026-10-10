package adpilot

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/rhl/businessos-backend/internal/founderos/pages/shared"
	"os/exec"
	"regexp"
	"sort"
	"strings"
	"time"
)

// LLM is the one model seam Adscout's ask and mine use. It is injected, so
// tests never call a model, and nothing on page load does either. The
// production implementation is ClaudeCLI, the same `claude -p` call the operator
// OS makes (app/api/workflows/draft/logic.ts runClaude).
type LLM interface {
	Complete(ctx context.Context, prompt string) (string, error)
}

// ErrLLMUnavailable means the model could not be reached at all (no CLI on
// PATH, or a timeout), which the routes answer with 503, not 502.
var ErrLLMUnavailable = errors.New("analyst unavailable")

// ClaudeCLI shells out to the `claude` CLI resolved from PATH.
type ClaudeCLI struct {
	Bin     string
	Timeout time.Duration
}

func (c ClaudeCLI) Complete(ctx context.Context, prompt string) (string, error) {
	bin := c.Bin
	if bin == "" {
		bin = "claude"
	}
	path, err := exec.LookPath(bin)
	if err != nil {
		return "", ErrLLMUnavailable
	}
	timeout := c.Timeout
	if timeout == 0 {
		timeout = 60 * time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	var out, stderr bytes.Buffer
	cmd, cleanup, err := shared.ClaudeCommand(ctx, path, prompt)
	if err != nil {
		return "", ErrLLMUnavailable
	}
	defer cleanup()
	cmd.Stdout, cmd.Stderr = &out, &stderr
	if err := cmd.Run(); err != nil {
		if ctx.Err() != nil {
			return "", ErrLLMUnavailable
		}
		if msg := strings.TrimSpace(stderr.String()); msg != "" {
			return "", errors.New(msg)
		}
		return "", err
	}
	return ExtractReplyText(out.String()), nil
}

// ExtractReplyText unwraps the CLI's JSON envelope ({result} or {message});
// anything else is the reply text itself.
func ExtractReplyText(stdout string) string {
	trimmed := strings.TrimSpace(stdout)
	var env map[string]any
	if json.Unmarshal([]byte(trimmed), &env) == nil {
		if s, ok := env["result"].(string); ok {
			return s
		}
		if s, ok := env["message"].(string); ok {
			return s
		}
	}
	return trimmed
}

func marshal(v any) string {
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(v)
	return strings.TrimRight(b.String(), "\n")
}

type askAd struct {
	Brand       string   `json:"brand"`
	Hook        *string  `json:"hook"`
	DaysRunning int      `json:"daysRunning"`
	Live        bool     `json:"live"`
	Format      *string  `json:"format"`
	TopDrivers  []string `json:"topDrivers"`
}

// AskPrompt grounds a question in the synced store: the longevity-ranked
// wall and the recent signal messages. Zero Foreplay credits.
func AskPrompt(question string, wall []WallAd, signals []string) string {
	ads := make([]askAd, 0, len(wall))
	for _, w := range wall {
		type kv struct {
			k string
			v float64
		}
		var d []kv
		for k, v := range w.Drivers {
			d = append(d, kv{k, v})
		}
		sort.Slice(d, func(i, j int) bool {
			if d[i].v != d[j].v {
				return d[i].v > d[j].v
			}
			return d[i].k < d[j].k
		})
		top := []string{}
		for i := 0; i < len(d) && i < 3; i++ {
			top = append(top, d[i].k+":"+jsNum(d[i].v))
		}
		ads = append(ads, askAd{Brand: w.Brand, Hook: w.Hook, DaysRunning: w.DaysRunning, Live: w.Live, Format: w.Format, TopDrivers: top})
	}
	if signals == nil {
		signals = []string{}
	}
	return strings.Join([]string{
		"You are Adscout, the user's ad-intelligence analyst. Answer from the DATA below: competitor ads tracked via Foreplay.",
		"Rules: days-running-while-live is the only spend proxy (no engagement/spend numbers exist: never invent any).",
		"Be direct and specific; cite brands and hooks from the data. 120 words max. Plain text, no markdown.",
		"",
		"DATA: tracked ads (longevity-ranked): " + marshal(ads),
		"DATA: recent signals: " + marshal(signals),
		"",
		"QUESTION: " + question,
	}, "\n")
}

var fence = regexp.MustCompile("```(?:json)?|```")

// ExpandProbes asks the model for 4-6 ad-copy phrasings of a concept and
// falls back to NaiveProbes on any failure or bad reply.
func ExpandProbes(ctx context.Context, llm LLM, concept string) ([]string, string) {
	if llm != nil {
		prompt := strings.Join([]string{
			"You expand an ad-concept into search queries for an ad-library keyword search (searches ad text, not transcripts).",
			`Concept: "` + concept + `"`,
			"Reply with STRICT JSON only: an array of 4-6 short probe queries (2-5 words each): phrasings this concept leaves in real ad copy. Mix exact phrases and keyword pairs. No markdown.",
		}, "\n")
		if reply, err := llm.Complete(ctx, prompt); err == nil {
			var probes []string
			if json.Unmarshal([]byte(strings.TrimSpace(fence.ReplaceAllString(reply, ""))), &probes) == nil && validProbes(probes) {
				return probes, "claude"
			}
		}
	}
	return NaiveProbes(concept), "fallback"
}

func validProbes(p []string) bool {
	if len(p) < 2 || len(p) > 8 {
		return false
	}
	for _, s := range p {
		if n := len([]rune(s)); n < 2 || n > 80 {
			return false
		}
	}
	return true
}
