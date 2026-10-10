package tech

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/rhl/businessos-backend/internal/founderos/agents"
	"github.com/rhl/businessos-backend/internal/founderos/memory"
)

// Searcher is the slice of memory.Router the data agent uses.
type Searcher interface {
	Search(ctx context.Context, query string, workspaces []string, limit int) ([]memory.Hit, error)
}

// DataAgent is the knowledge analyst: it reads every staged engine's storage
// and surfaces ideas, and answers broadcasts by searching memory.
type DataAgent struct {
	Engines EngineSource
	Client  *http.Client
	Search  Searcher
}

func (a *DataAgent) Meta() agents.Meta {
	return agents.Meta{
		ID:           "data-agent",
		Name:         "Data Agent",
		Description:  "Analyzes Optimal Engine storage health and surfaces ideas; answers broadcasts by querying memory.",
		DepartmentID: "dept-tech",
	}
}

func (a *DataAgent) Run(ctx context.Context) (agents.Result, error) {
	v, err := resolveView(a.Engines)
	if err != nil {
		return agents.Result{OK: false, Summary: err.Error()}, nil
	}
	if len(v.engines) == 0 {
		return agents.Result{OK: false, Summary: noEnginesSummary}, nil
	}
	ovs := overview(ctx, v, newClient(a.Client), false)

	ok := true
	var parts, ideas []string
	for _, o := range ovs {
		if !o.Reachable {
			ok = false
			parts = append(parts, fmt.Sprintf("%s unreachable (%s)", o.Engine, o.Error))
			ideas = append(ideas, o.Engine+" unreachable: check the engine before trusting memory search")
			continue
		}
		if o.Health != "up" {
			ok = false
			parts = append(parts, fmt.Sprintf("%s health %q", o.Engine, o.Health))
		} else {
			parts = append(parts, fmt.Sprintf("%s up · %s contexts · %s embedded · %s claims / %s facts",
				o.Engine, itoa(o.count("contexts")), itoa(o.count("vectors")), itoa(o.count("claims")), itoa(o.count("facts"))))
		}
		if o.Error != "" {
			ok = false
			ideas = append(ideas, fmt.Sprintf("%s answered health but a read failed (%s)", o.Engine, o.Error))
		}
		if o.AuditError != "" {
			ideas = append(ideas, fmt.Sprintf("%s storage audit unreadable (%s)", o.Engine, o.AuditError))
		} else if o.Audit != nil {
			if f := o.Audit.failing(); len(f) > 0 {
				names := make([]string, len(f))
				for i, c := range f {
					names[i] = c.Name
				}
				ideas = append(ideas, fmt.Sprintf("%d audit check(s) on %s need attention (%s)", len(f), o.Engine, strings.Join(names, ", ")))
			}
		}
		if len(o.Missing) > 0 {
			ideas = append(ideas, fmt.Sprintf("%s routed to %s but not created there", strings.Join(o.Missing, ", "), o.Engine))
		}
		if c, vec := o.count("contexts"), o.count("vectors"); c != nil && vec != nil && *c > *vec {
			ideas = append(ideas, fmt.Sprintf("%d context(s) on %s have no embedding", *c-*vec, o.Engine))
		}
		if c, f := o.count("claims"), o.count("facts"); c != nil && f != nil && *c > 0 && *f == 0 {
			ideas = append(ideas, fmt.Sprintf("%d claim(s) on %s, 0 facts: review and promote the claim queue", *c, o.Engine))
		}
	}
	if len(ideas) == 0 {
		ideas = append(ideas, "storage healthy — no action needed")
	}
	if note := v.unstagedNote(); note != "" {
		parts = append(parts, note)
	}
	return agents.Result{
		OK:      ok,
		Summary: strings.Join(parts, " · ") + " · ideas: " + strings.Join(ideas, " | "),
		Data:    map[string]any{"engines": ovs, "ideas": ideas, "unstaged": v.unstaged},
	}, nil
}

// Respond answers a broadcast by searching every staged workspace.
func (a *DataAgent) Respond(ctx context.Context, message string) (agents.Result, error) {
	if a.Search == nil {
		return agents.Result{OK: false, Summary: "memory router is not configured on the bridge; cannot search memory"}, nil
	}
	v, err := resolveView(a.Engines)
	if err != nil {
		return agents.Result{OK: false, Summary: err.Error()}, nil
	}
	ws := v.stagedWorkspaces()
	if len(ws) == 0 {
		return agents.Result{OK: false, Summary: noEnginesSummary}, nil
	}
	hits, err := a.Search.Search(ctx, message, ws, 10)
	// One engine down is a partial answer: keep the hits, name the gap.
	missing := ""
	var partial *memory.PartialError
	if errors.As(err, &partial) {
		missing = " · partial: " + strings.Join(partial.Failed, ", ") + " not searched"
		err = nil
	}
	if err != nil {
		return agents.Result{OK: false, Summary: "memory search failed: " + err.Error()}, nil
	}
	if len(hits) == 0 {
		return agents.Result{OK: false, Summary: fmt.Sprintf("Nothing in memory matches %q", clip(message, 80)) + missing}, nil
	}
	top := hits
	if len(top) > 3 {
		top = top[:3]
	}
	lines := make([]string, len(top))
	for i, h := range top {
		lines[i] = h.Title + ": " + clip(h.Abstract, 100)
	}
	return agents.Result{OK: true, Summary: strings.Join(lines, " · ") + missing, Data: hits}, nil
}

// clip cuts s to n runes.
func clip(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n])
}
