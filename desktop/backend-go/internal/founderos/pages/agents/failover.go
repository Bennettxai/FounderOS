package agentspage

import (
	"regexp"
	"strings"
	"time"

	"github.com/rhl/businessos-backend/internal/founderos/connectors/paperclip"
)

// Model failover for the Paperclip board (lib/agent-failover.ts). Pure: the
// plan is computed from the seats and their latest runs; the route applies
// it through guarded board writes.

// FailoverChain is each model's fallback ladder, preferred first.
var FailoverChain = map[string][]string{
	"claude-fable-5":    {"claude-opus-5", "claude-sonnet-5"},
	"claude-mythos-5":   {"claude-opus-5", "claude-sonnet-5"},
	"claude-opus-5":     {"claude-sonnet-5", "claude-haiku-4-5"},
	"claude-opus-4-8":   {"claude-opus-5", "claude-sonnet-5"},
	"claude-opus-4-7":   {"claude-opus-5", "claude-sonnet-5"},
	"claude-opus-4-6":   {"claude-opus-5", "claude-sonnet-5"},
	"claude-sonnet-5":   {"claude-haiku-4-5"},
	"claude-sonnet-4-6": {"claude-sonnet-5", "claude-haiku-4-5"},
	"claude-sonnet-4-5": {"claude-sonnet-5", "claude-haiku-4-5"},
	"claude-haiku-4-6":  {"claude-haiku-4-5"},
	"claude-haiku-4-5":  {},
}

var (
	exhaustedRE = regexp.MustCompile(`(?i)` + strings.Join([]string{
		`out\s+of\s+(?:extra\s+)?usage(?:\s+credits?)?`,
		`claude\s+usage\s+limit\s+reached`,
		`usage\s+limit\s+reached`,
		`usage\s+cap\s+reached`,
		`5[-\s]?hour\s+limit\s+reached`,
		`weekly\s+limit\s+reached`,
		`you(?:'|’)ve\s+hit\s+your\s+session\s+limit`,
		`session\s+limit\s+(?:reached|exceeded)`,
	}, "|"))
	orgPolicyRE = regexp.MustCompile(`(?i)organization\s+has\s+disabled|subscription\s+access\s+for\s+claude\s+code|ask\s+your\s+admin\s+to\s+enable`)
	quotaRE     = regexp.MustCompile(`(?i)(?:you(?:'|’)?ve\s+)?hit\s+your\s+usage\s+limit`)
	authRE      = regexp.MustCompile(`(?i)` + strings.Join([]string{
		`oauth\s+session\s+expired`, `failed\s+to\s+authenticate`, `could\s+not\s+be\s+refreshed`,
		`not\s+logged\s+in`, `please\s+log\s*in`, `run\s+/login`, `claude\s+login`,
	}, "|"))
)

func IsCreditExhaustion(text string) bool { return text != "" && exhaustedRE.MatchString(text) }

// ClassifyFailure is credits | quota_window | auth | org_policy, or "".
func ClassifyFailure(text string) string {
	switch {
	case text == "":
		return ""
	case orgPolicyRE.MatchString(text):
		return "org_policy"
	case authRE.MatchString(text):
		return "auth"
	case quotaRE.MatchString(text):
		return "quota_window"
	case exhaustedRE.MatchString(text):
		return "credits"
	}
	return ""
}

func laneFatal(kind string) bool { return kind == "org_policy" || kind == "auth" }

// StandbyPairs: a dead primary lane hands its thread to the standby seat.
var StandbyPairs = map[string]string{"Conductor": "Conductor (Astra)"}

const QuotaWindow = 5 * time.Hour

type FailoverAgent struct {
	ID     string  `json:"id"`
	Name   string  `json:"name"`
	Model  *string `json:"model"`
	Status string  `json:"status"`
}

type FailoverRunIn = paperclip.FailoverRun

type FailoverAction struct {
	AgentID   string `json:"agentId"`
	AgentName string `json:"agentName"`
	From      string `json:"from"`
	To        string `json:"to"`
	Reason    string `json:"reason"`
}

type FailoverResume struct {
	AgentID   string `json:"agentId"`
	AgentName string `json:"agentName"`
	Reason    string `json:"reason"`
}

type FailoverAlert struct {
	AgentID       string `json:"agentId"`
	AgentName     string `json:"agentName"`
	Kind          string `json:"kind"`
	RunFinishedAt string `json:"runFinishedAt"`
	Message       string `json:"message"`
}

type FailoverHandoff struct {
	FromAgentID   string `json:"fromAgentId"`
	FromAgentName string `json:"fromAgentName"`
	ToAgentID     string `json:"toAgentId"`
	ToAgentName   string `json:"toAgentName"`
	Reason        string `json:"reason"`
}

type FailoverPlan struct {
	Actions   []FailoverAction  `json:"actions"`
	Resumes   []FailoverResume  `json:"resumes"`
	Alerts    []FailoverAlert   `json:"alerts"`
	Handoffs  []FailoverHandoff `json:"handoffs"`
	Exhausted []string          `json:"exhausted"`
	Notes     []string          `json:"notes"`
}

var notFailure = map[string]bool{"succeeded": true, "success": true, "completed": true, "running": true, "queued": true, "pending": true, "cancelled": true, "canceled": true}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func failureText(r FailoverRunIn) string {
	var parts []string
	for _, p := range []*string{r.Error, r.Summary} {
		if p != nil && *p != "" {
			parts = append(parts, *p)
		}
	}
	return strings.Join(parts, "\n")
}

func recency(r FailoverRunIn) string {
	if r.FinishedAt == nil {
		return "9999-12-31"
	}
	return *r.FinishedAt
}

func isoMillis(t time.Time) string { return t.UTC().Format("2006-01-02T15:04:05.000Z") }

// PlanFailover decides, from each seat's latest run, which seats move down
// their ladder, which parked seats resume, which failures need a person,
// and which dead lanes hand off to a standby.
func PlanFailover(agentsIn []FailoverAgent, runs []FailoverRunIn, now time.Time) FailoverPlan {
	plan := FailoverPlan{Actions: []FailoverAction{}, Resumes: []FailoverResume{}, Alerts: []FailoverAlert{}, Handoffs: []FailoverHandoff{}, Exhausted: []string{}, Notes: []string{}}
	byID := map[string]FailoverAgent{}
	for _, a := range agentsIn {
		byID[a.ID] = a
	}
	latest := map[string]FailoverRunIn{}
	var order []string
	for _, r := range runs {
		prev, seen := latest[r.AgentID]
		if !seen {
			order = append(order, r.AgentID)
		}
		if !seen || recency(r) > recency(prev) {
			latest[r.AgentID] = r
		}
	}
	exhausted := map[string]bool{}
	laneDead := map[string]string{}
	for _, id := range order {
		r := latest[id]
		if notFailure[strings.ToLower(r.Status)] {
			continue
		}
		kind := ClassifyFailure(failureText(r))
		if kind == "" {
			continue
		}
		agent, known := byID[r.AgentID]
		if kind == "credits" {
			model := deref(r.Model)
			if model == "" && known {
				model = deref(agent.Model)
			}
			if model != "" && !exhausted[model] {
				exhausted[model] = true
				plan.Exhausted = append(plan.Exhausted, model)
			}
			continue
		}
		if !known {
			continue
		}
		if laneFatal(kind) {
			if kind == "org_policy" {
				laneDead[agent.ID] = "the organization on this login has disabled Claude Code subscription access"
			} else {
				laneDead[agent.ID] = "the login on the board host has expired and no model on the ladder fixes that"
			}
		}
		at := deref(r.FinishedAt)
		if at == "" {
			at = isoMillis(now)
		}
		if kind == "quota_window" {
			finished, err := time.Parse(time.RFC3339Nano, at)
			if err != nil {
				finished = now
			}
			reset := finished.Add(QuotaWindow)
			if !now.Before(reset) {
				if agent.Status == "error" {
					plan.Resumes = append(plan.Resumes, FailoverResume{AgentID: agent.ID, AgentName: agent.Name, Reason: "usage window that closed at " + at + " has rolled over"})
				}
				continue
			}
			model := deref(r.Model)
			if model == "" {
				model = deref(agent.Model)
			}
			if model == "" {
				model = "model"
			}
			plan.Alerts = append(plan.Alerts, FailoverAlert{AgentID: agent.ID, AgentName: agent.Name, Kind: kind, RunFinishedAt: at,
				Message: agent.Name + " hit the " + model + " usage window at " + at + ". Not a credit problem: the seat is parked and will be resumed automatically after " + isoMillis(reset) + "."})
			continue
		}
		msg := agent.Name + " cannot authenticate (run finished " + at + "). The Claude login on the board host has expired; run `claude` and log in as the board user. Not a model problem, so no model change fixes it."
		if kind == "org_policy" {
			msg = agent.Name + " cannot run on this Claude login (run finished " + at + "). The organization on that login has disabled Claude Code subscription access, so logging in again and stepping down the model ladder both do nothing: point the seat at an Anthropic API key or the Codex lane, or have the org admin re-enable access."
		}
		plan.Alerts = append(plan.Alerts, FailoverAlert{AgentID: agent.ID, AgentName: agent.Name, Kind: kind, RunFinishedAt: at, Message: msg})
	}

	moved := map[string]bool{}
	for _, a := range agentsIn {
		model := deref(a.Model)
		if model == "" || !exhausted[model] {
			continue
		}
		to := ""
		for _, c := range FailoverChain[model] {
			if !exhausted[c] {
				to = c
				break
			}
		}
		if to == "" {
			plan.Notes = append(plan.Notes, a.Name+": no healthy fallback left on the "+model+" ladder")
			continue
		}
		moved[a.ID] = true
		plan.Actions = append(plan.Actions, FailoverAction{AgentID: a.ID, AgentName: a.Name, From: model, To: to, Reason: model + " is out of usage credits"})
	}

	byName := map[string]FailoverAgent{}
	for _, a := range agentsIn {
		byName[a.Name] = a
	}
	for _, primary := range agentsIn {
		standbyName, paired := StandbyPairs[primary.Name]
		if !paired || primary.Status == "paused" {
			continue
		}
		reason := laneDead[primary.ID]
		stranded := primary.Status == "error" && primary.Model != nil && exhausted[*primary.Model] && !moved[primary.ID]
		if reason == "" && !stranded {
			continue
		}
		standby, ok := byName[standbyName]
		if !ok {
			plan.Notes = append(plan.Notes, primary.Name+": lane is down and its standby "+standbyName+" does not exist on this board")
			continue
		}
		if standby.Status == "error" || standby.Status == "paused" {
			plan.Notes = append(plan.Notes, primary.Name+": lane is down but the standby "+standbyName+" is "+standby.Status+", so there is nobody to hand to")
			continue
		}
		if reason == "" {
			reason = deref(primary.Model) + " is out of usage credits and the ladder has no healthy fallback left"
		}
		plan.Handoffs = append(plan.Handoffs, FailoverHandoff{FromAgentID: primary.ID, FromAgentName: primary.Name, ToAgentID: standby.ID, ToAgentName: standby.Name, Reason: reason})
	}
	return plan
}

const FailoverNotePrefix = "[FounderOS v1 failover]"

// AlertMarker dedupes an alert on the cockpit thread (once per failing run).
func AlertMarker(a FailoverAlert) string {
	return FailoverNotePrefix + " " + a.AgentName + " " + a.Kind + " " + a.RunFinishedAt
}

func RenderAlert(a FailoverAlert) string {
	return AlertMarker(a) + "\n" + a.Message + "\nAutomated note from the OS failover tick, no reply needed."
}
