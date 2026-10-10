// Package conductor is the Conductor panel's logic on the bridge: the port of
// FounderOS v1 lib/screen-context.ts (titles + quick actions) and
// lib/interject.ts (route an interjection to a board task or a note).
package conductor

import (
	"context"
	"regexp"
	"strings"
	"time"
)

type navItem struct{ href, label string }

// FounderOS v1 lib/nav.ts, in order; "G-Brain" is "Brain" on the bridge.
var nav = []navItem{
	{"/", "Home"}, {"/comms", "Comms"}, {"/funnel", "Funnel"}, {"/workflows", "Workflows"},
	{"/social", "Social"}, {"/content", "Content"}, {"/brand-deals", "Brand Deals"},
	{"/finances", "Finances"}, {"/trading", "Trading"}, {"/clients", "Clients"},
	{"/adpilot", "AdPilot"}, {"/agents", "Agents"}, {"/tasks", "Tasks"}, {"/skills", "Skills"},
	{"/org", "Org Chart"}, {"/brain", "Brain"}, {"/doctor", "Doctor"}, {"/blueprint", "Blueprint"},
	{"/integrations", "Connections"}, {"/usage", "Usage"}, {"/analytics", "Analytics"},
	{"/personas", "Personas"},
}

// clean strips the query and the bridge's /os prefix: "/os/funnel?x" → "/funnel".
func clean(path string) string {
	p, _, _ := strings.Cut(path, "?")
	if p == "/os" || p == "" {
		return "/"
	}
	if strings.HasPrefix(p, "/os/") {
		p = strings.TrimPrefix(p, "/os")
	}
	return p
}

func ScreenTitle(path string) string {
	c := clean(path)
	for _, n := range nav {
		if (n.href == "/" && c == "/") || (n.href != "/" && (c == n.href || strings.HasPrefix(c, n.href+"/"))) {
			return n.label
		}
	}
	return c
}

type QuickAction struct {
	Label  string `json:"label"`
	Prompt string `json:"prompt"`
}

var quickByRoute = []struct {
	href    string
	actions []QuickAction
}{
	{"/funnel", []QuickAction{
		{"Who is going cold?", "Which leads in the funnel are fading toward the 90-day archive, and what is the next touch for each?"},
		{"Stage bottleneck", "Which funnel stage is holding the most clients right now, and why are they stuck there?"},
		{"Revenue this month", "What has the funnel converted this month, and how does that compare to the pipeline still open?"},
		{"Draft a follow-up", "Draft a follow-up message for the quietest lead in the funnel. Do not send it, show it to me first."},
	}},
	{"/brain", []QuickAction{
		{"What do you know?", "Summarize what the brain (Optimal Engine) currently knows about my business: the biggest clusters and where the coverage is thin."},
		{"Query the brain", "Run a brain query and show me the top hits with their sources."},
		{"Gaps in the store", "Which workspaces have the fewest brain pages, and what should I capture to fill them?"},
		{"Capture a note", "Capture a note into the brain for me. Ask me what it should say first."},
	}},
	{"/agents", []QuickAction{
		{"What is running?", "Which agents are running right now, which are blocked, and which have not run today?"},
		{"Last run failures", "Show me every agent whose last run failed, with the error and what it would take to fix."},
		{"Create board task", "Create a board task for the work I am about to describe. Ask me for the title first."},
		{"Delegate to a pillar", "Delegate this screen’s open work to the right pillar and tell me who picked it up."},
	}},
	{"/org", []QuickAction{
		{"Who owns what?", "Walk the org: which pillar owns which workers, and where is a seat doing nothing?"},
		{"Broadcast draft", "Draft a broadcast to every pillar about today’s priority. Show it to me before sending."},
		{"Idle seats", "Which agent seats have been idle longest, and what should they be pointed at?"},
		{"Delegate to a pillar", "Delegate a task to one pillar and tell me which agent took it."},
	}},
	{"/comms", []QuickAction{
		{"What needs a reply?", "Across every inbox and Slack lane, what is waiting on a reply from me?"},
		{"Today’s digest", "Summarize today’s comms: who reached out, what they wanted, what is unanswered."},
		{"Draft the hardest one", "Draft a reply to the message that needs the most thought. Show it to me, do not send it."},
		{"Recording action items", "Pull the action items out of the most recent call recordings."},
	}},
	{"/social", []QuickAction{
		{"What is working?", "Which posts drove the most growth in the last 30 days, and what do they have in common?"},
		{"Follower delta", "How did each account’s follower count move this month?"},
		{"Unanswered DMs", "Which DMs are still waiting on a reply?"},
		{"Draft the next post", "Draft the next post in my format based on what performed best recently."},
	}},
	{"/finances", []QuickAction{
		{"Cash this month", "What came in this month across every processor, and how does it compare to last month?"},
		{"Outstanding invoices", "What is invoiced and unpaid right now, and how old is each one?"},
		{"Where is it going?", "Break down this month’s spend by category and flag anything unusual."},
		{"Create a payment link", "Create a Stripe payment link. Ask me the amount and what it is for first."},
	}},
	{"/trading", []QuickAction{
		{"How is the sleeve?", "How is the agentic sleeve doing today, and what did the Markets Agent reason about it?"},
		{"Open orders", "What orders are open right now, and are any of them stale?"},
		{"Biggest movers", "Which positions moved most today and why?"},
		{"Explain a trade", "Explain the most recent trade in the log: the thesis and whether it played out."},
	}},
	{"/integrations", []QuickAction{
		{"What is down?", "Which connections are not connected right now, and what does each one need?"},
		{"Explain a failure", "Take the most broken connector and explain exactly why it is failing."},
		{"Freshest data", "Which connectors have the stalest data, and how stale?"},
		{"Create board task", "Create a board task to fix the connector that is most worth fixing."},
	}},
	{"/content", []QuickAction{
		{"What should I post?", "Based on what has performed, what should the next piece of content be?"},
		{"Open drafts", "Which content pieces are drafted but never shipped?"},
		{"Lead magnet pull", "Which lead magnets are actually converting, and which are dead weight?"},
		{"Draft a caption", "Draft a caption in my format with three hashtags. Ask me the topic first."},
	}},
	{"/workflows", []QuickAction{
		{"What fires next?", "Which scheduled workflow fires next, and what does it do?"},
		{"Recent failures", "Which workflow runs failed recently, and what was the error?"},
		{"Run one now", "Which workflow is worth running right now, and why?"},
		{"Create board task", "Create a board task for a workflow that needs fixing."},
	}},
	{"/tasks", []QuickAction{
		{"What needs me?", "Which tasks are actually waiting on me rather than on an agent?"},
		{"Blocked lane", "What is blocked, and what is each one blocked on?"},
		{"Create board task", "Create a board task. Ask me the title and the owner first."},
		{"Delegate to a pillar", "Take the oldest open task and delegate it to the right pillar."},
	}},
	{"/doctor", []QuickAction{
		{"What is broken?", "Walk the doctor checks and tell me what is genuinely broken versus merely noisy."},
		{"Explain a check", "Explain the worst failing check: what it measures and what fixes it."},
		{"Since when?", "How long has each failing check been failing?"},
		{"Create board task", "Create a board task for the failure most worth fixing today."},
	}},
}

var quickDefault = []QuickAction{
	{"Summarize this screen", "Summarize what I am looking at on this screen and what stands out."},
	{"What needs me?", "What needs my attention right now across the OS?"},
	{"Create board task", "Create a board task for the work on this screen. Ask me the title first."},
	{"Delegate to a pillar", "Delegate the work on this screen to the right pillar and tell me who picked it up."},
}

func QuickActions(path string) []QuickAction {
	c := clean(path)
	for _, q := range quickByRoute {
		if c == q.href || strings.HasPrefix(c, q.href+"/") {
			return q.actions
		}
	}
	return quickDefault
}

// ---- interject (lib/interject.ts) -------------------------------------------

type Route string

const (
	RouteTask  Route = "task"
	RouteAgent Route = "agent"
	RouteNote  Route = "note"
)

var (
	taskRe      = regexp.MustCompile(`(?i)^task[: ]|\b(todo|task)\b`)
	agentRe     = regexp.MustCompile(`(?i)^(tell|ask)\s+\w+`)
	taskPrefixR = regexp.MustCompile(`(?i)^\s*(task|todo)\s*[:\-]\s*(-\s*)?`)
)

func Classify(text string) Route {
	switch {
	case taskRe.MatchString(text):
		return RouteTask
	case agentRe.MatchString(text):
		return RouteAgent
	default:
		return RouteNote
	}
}

type Note struct {
	Text  string
	Title string
	Slug  string
}

// InterjectDeps are the two ways an interjection lands.
type InterjectDeps interface {
	CreateTask(ctx context.Context, title, description string) (ref, url string, err error)
	CaptureNote(ctx context.Context, n Note) (slug string, err error)
}

type Receipt struct {
	OK    bool   `json:"ok"`
	Route Route  `json:"route"`
	Ref   string `json:"ref,omitempty"`
	URL   string `json:"url,omitempty"`
	Slug  string `json:"slug,omitempty"`
	Error string `json:"error,omitempty"`
}

func firstLine(s string) string {
	line, _, _ := strings.Cut(s, "\n")
	return strings.TrimSpace(line)
}

func clip(s string, n int) string {
	if len(s) > n {
		return s[:n]
	}
	return s
}

// Perform routes text to a board task, an agent task, or a brain note.
func Perform(ctx context.Context, text string, route Route, deps InterjectDeps, now time.Time) Receipt {
	if route == "" {
		route = Classify(text)
	}
	if route == RouteNote {
		day := now.UTC().Format("2006-01-02")
		slug, err := deps.CaptureNote(ctx, Note{Text: text, Title: "Interject " + day, Slug: "inbox/" + day + "-note"})
		if err != nil {
			return Receipt{Route: RouteNote, Error: err.Error()}
		}
		return Receipt{OK: true, Route: RouteNote, Slug: slug}
	}
	title := clip(firstLine(text), 120)
	if route == RouteTask {
		t := strings.TrimSpace(taskPrefixR.ReplaceAllString(firstLine(text), ""))
		if t == "" {
			t = strings.TrimSpace(text)
		}
		title = clip(t, 120)
	}
	desc := text + "\n\n— interjected from the OS home (route: " + string(route) + ")"
	ref, url, err := deps.CreateTask(ctx, title, desc)
	if err != nil {
		return Receipt{Route: route, Error: err.Error()}
	}
	return Receipt{OK: true, Route: route, Ref: ref, URL: url}
}
