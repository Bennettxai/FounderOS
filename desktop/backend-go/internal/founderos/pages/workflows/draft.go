package workflows

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/rhl/businessos-backend/internal/founderos/pages/shared"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

// The drafting assistant (app/api/workflows/draft): a free-text description
// becomes a builder draft by shelling out to the `claude` CLI. The runner is
// injectable so tests never spawn the binary; nothing here writes anywhere.

const SchemaInstructions = `Reply with STRICT JSON only: no prose, no markdown code fences: matching exactly this shape:
{
  "name": string,
  "subtitle": string,
  "steps": [
    {
      "title": string,
      "detail": string,           // one or two sentences, what actually happens in this step
      "ownerKind": "human" | "agent",
      "owner": string,            // a name: a real person or a plausible agent name
      "hoursPerWeek": number,     // >= 0
      "tools": string[],          // short lowercase tool ids, e.g. "gmail", "typeform", "notion"
      "automation": null | { "title": string, "state": "live" | "suggested", "recoveredUsd": number },
      "branchFromIndex": null | number,   // index (0-based) of an EARLIER step in this same array that this step forks from, or null for the normal sequence
      "branchCondition": null | string    // required together with branchFromIndex, e.g. "approved", "went quiet"
    }
  ]
}
Steps run in array order by default (step i follows step i-1). Only set branchFromIndex when the workflow genuinely forks (e.g. "if approved, do X; if rejected, do Y"): branchFromIndex must point at an earlier index in the SAME array. Most workflows do not need any branching at all.`

func BuildPrompt(userPrompt, priorError string) string {
	base := "You are drafting a business workflow for an operator dashboard. The operator describes a process; you map it into discrete steps, each owned by a human or an agent.\n\nDescribe: " + userPrompt + "\n\n" + SchemaInstructions
	if priorError == "" {
		return base
	}
	return base + "\n\nYour previous reply failed validation with this error:\n" + priorError + "\nReply again with ONLY corrected strict JSON: nothing else."
}

// ExtractReplyText pulls the reply out of a `claude -p --output-format json`
// envelope ({result} or {message}); anything else is the reply itself.
func ExtractReplyText(stdout string) string {
	trimmed := strings.TrimSpace(stdout)
	var env map[string]any
	if json.Unmarshal([]byte(trimmed), &env) == nil && env != nil {
		if s, ok := env["result"].(string); ok {
			return s
		}
		if s, ok := env["message"].(string); ok {
			return s
		}
	}
	return trimmed
}

var fenceRE = regexp.MustCompile("(?is)```(?:json)?\\s*(.*?)```")

// ParseDraft parses and validates one CLI reply.
func ParseDraft(stdout string) (Input, error) {
	text := ExtractReplyText(stdout)
	if m := fenceRE.FindStringSubmatch(text); m != nil {
		text = m[1]
	}
	text = strings.TrimSpace(text)
	var probe any
	if err := json.Unmarshal([]byte(text), &probe); err != nil {
		return Input{}, errors.New("reply was not valid JSON: " + err.Error())
	}
	return ParseInput([]byte(text))
}

// CLIResult is one CLI call: OK with stdout, or an error, Unavailable when
// the binary is missing or timed out.
type CLIResult struct {
	OK          bool
	Stdout      string
	Unavailable bool
	Err         error
}

type Runner interface {
	Run(ctx context.Context, prompt string) CLIResult
}

// ClaudeCLI is the default runner: `claude -p <prompt> --output-format json`
// from PATH, 60s timeout.
type ClaudeCLI struct{ Timeout time.Duration }

func (c ClaudeCLI) Run(ctx context.Context, prompt string) CLIResult {
	bin, err := exec.LookPath("claude")
	if err != nil {
		return CLIResult{Unavailable: true, Err: err}
	}
	timeout := c.Timeout
	if timeout == 0 {
		timeout = 60 * time.Second
	}
	cctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	cmd, cleanup, err := shared.ClaudeCommand(cctx, bin, prompt)
	if err != nil {
		return CLIResult{Unavailable: true, Err: err}
	}
	defer cleanup()
	var stdout, stderr strings.Builder
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		unavailable := errors.Is(cctx.Err(), context.DeadlineExceeded) || strings.Contains(strings.ToLower(stderr.String()), "command not found")
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return CLIResult{Unavailable: unavailable, Err: errors.New(msg)}
	}
	return CLIResult{OK: true, Stdout: stdout.String()}
}

type DraftResult struct {
	OK          bool   `json:"ok"`
	Draft       *Input `json:"draft,omitempty"`
	Unavailable bool   `json:"unavailable"`
	Error       string `json:"error,omitempty"`
}

func cliFailure(r CLIResult) DraftResult {
	if r.Unavailable {
		return DraftResult{Unavailable: true, Error: "drafting assistant unavailable: build manually"}
	}
	msg := "unknown error"
	if r.Err != nil {
		msg = r.Err.Error()
	}
	return DraftResult{Error: "drafting assistant failed: " + msg}
}

// Draft runs the assistant, retrying once with the validation error fed back.
func Draft(ctx context.Context, run Runner, prompt string) DraftResult {
	first := run.Run(ctx, BuildPrompt(prompt, ""))
	if !first.OK {
		return cliFailure(first)
	}
	d, err := ParseDraft(first.Stdout)
	if err == nil {
		return DraftResult{OK: true, Draft: &d}
	}
	second := run.Run(ctx, BuildPrompt(prompt, err.Error()))
	if !second.OK {
		return cliFailure(second)
	}
	d, err = ParseDraft(second.Stdout)
	if err == nil {
		return DraftResult{OK: true, Draft: &d}
	}
	return DraftResult{Error: "drafting assistant returned an invalid draft twice: build manually (" + err.Error() + ")"}
}
