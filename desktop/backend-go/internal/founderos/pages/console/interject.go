package console

import (
	"context"
	"regexp"
	"strings"
	"time"
)

// Interject is the home composer's no-LLM router (lib/interject.ts). Free
// text becomes a board task, an agent relay (also a board task, keeping the
// "tell/ask <agent>" phrasing), or a note. Bridge change: notes are captured
// into the Optimal Engine (GBrain is retired), not a G-Brain inbox page.

type NoteInput struct {
	Text  string
	Title string
	Slug  string
}

// InterjectDeps are the two places an interject can land.
type InterjectDeps interface {
	CreateTask(ctx context.Context, title, description string) (ref, url string, err error)
	CaptureNote(ctx context.Context, n NoteInput) (id string, err error)
}

type Receipt struct {
	OK    bool   `json:"ok"`
	Route string `json:"route"`
	Ref   string `json:"ref,omitempty"`
	URL   string `json:"url,omitempty"`
	Slug  string `json:"slug,omitempty"`
	Error string `json:"error,omitempty"`
}

var (
	taskRE   = regexp.MustCompile(`(?i)^task[: ]|\b(todo|task)\b`)
	agentRE  = regexp.MustCompile(`(?i)^(tell|ask)\s+\w+`)
	prefixRE = regexp.MustCompile(`(?i)^\s*(task|todo)\s*[:\-]\s*`)
)

// ClassifyInterject picks a route; task patterns win over agent ones.
func ClassifyInterject(text string) string {
	switch {
	case taskRE.MatchString(text):
		return "task"
	case agentRE.MatchString(text):
		return "agent"
	}
	return "note"
}

func firstLine(text string) string { return strings.TrimSpace(strings.SplitN(text, "\n", 2)[0]) }

func clamp(s string, n int) string {
	if r := []rune(s); len(r) > n {
		return string(r[:n])
	}
	return s
}

// PerformInterject lands text on its route. route "" means auto.
func PerformInterject(ctx context.Context, text, route string, deps InterjectDeps, now time.Time) Receipt {
	if route == "" {
		route = ClassifyInterject(text)
	}
	if route == "note" {
		day := now.UTC().Format("2006-01-02")
		slug := "inbox/" + day + "-note"
		if _, err := deps.CaptureNote(ctx, NoteInput{Text: text, Title: "Interject " + day, Slug: slug}); err != nil {
			return Receipt{Route: "note", Error: err.Error()}
		}
		return Receipt{OK: true, Route: "note", Slug: slug}
	}
	title := firstLine(text)
	if route == "task" {
		title = strings.TrimSpace(prefixRE.ReplaceAllString(strings.SplitN(text, "\n", 2)[0], ""))
		if title == "" {
			title = strings.TrimSpace(text)
		}
	}
	title = clamp(title, 120)
	desc := text + "\n\n— interjected from the OS home (route: " + route + ")"
	ref, url, err := deps.CreateTask(ctx, title, desc)
	if err != nil {
		return Receipt{Route: route, Error: err.Error()}
	}
	return Receipt{OK: true, Route: route, Ref: ref, URL: url}
}
