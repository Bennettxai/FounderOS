package workflows

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/rhl/businessos-backend/internal/founderos/pages/osdata"
)

// The builder shape (app/api/workflows/shared.ts): steps name their branch
// parent by INDEX, because neither the builder nor a fresh draft has step
// ids yet. Create, update and the drafting assistant all validate through
// ParseInput so the three paths can never disagree.

type AutomationInput struct {
	Title        string  `json:"title"`
	State        string  `json:"state"`
	RecoveredUSD float64 `json:"recoveredUsd"`
}

type StepInput struct {
	Title           string           `json:"title"`
	Detail          string           `json:"detail"`
	OwnerKind       string           `json:"ownerKind"`
	Owner           string           `json:"owner"`
	HoursPerWeek    float64          `json:"hoursPerWeek"`
	Tools           []string         `json:"tools"`
	Automation      *AutomationInput `json:"automation"`
	BranchFromIndex *int             `json:"branchFromIndex"`
	BranchCondition *string          `json:"branchCondition"`
}

type Input struct {
	Name     string      `json:"name"`
	Subtitle string      `json:"subtitle"`
	Steps    []StepInput `json:"steps"`
}

// ParseInput decodes and validates builder/draft JSON (WorkflowInputSchema).
func ParseInput(raw []byte) (Input, error) {
	var probe map[string]json.RawMessage
	if err := json.Unmarshal(raw, &probe); err != nil || probe == nil {
		return Input{}, errors.New("body must be a JSON object")
	}
	for _, k := range []string{"name", "subtitle", "steps"} {
		if _, ok := probe[k]; !ok {
			return Input{}, fmt.Errorf("%s: required", k)
		}
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	var in Input
	if err := dec.Decode(&in); err != nil {
		return Input{}, fmt.Errorf("invalid workflow: %w", err)
	}
	if strings.TrimSpace(in.Name) == "" {
		return Input{}, errors.New("name: name is required")
	}
	if len(in.Steps) == 0 {
		return Input{}, errors.New("steps: a workflow needs at least one step")
	}
	for i, s := range in.Steps {
		switch {
		case s.Title == "":
			return Input{}, fmt.Errorf("steps.%d.title: required", i)
		case s.Detail == "":
			return Input{}, fmt.Errorf("steps.%d.detail: required", i)
		case s.OwnerKind != "human" && s.OwnerKind != "agent":
			return Input{}, fmt.Errorf("steps.%d.ownerKind: must be human or agent", i)
		case s.Owner == "":
			return Input{}, fmt.Errorf("steps.%d.owner: required", i)
		case s.HoursPerWeek < 0:
			return Input{}, fmt.Errorf("steps.%d.hoursPerWeek: must be >= 0", i)
		case s.Tools == nil:
			return Input{}, fmt.Errorf("steps.%d.tools: required", i)
		}
		if a := s.Automation; a != nil {
			if a.Title == "" || (a.State != "live" && a.State != "suggested") || a.RecoveredUSD < 0 {
				return Input{}, fmt.Errorf("steps.%d.automation: needs a title, state live|suggested and recoveredUsd >= 0", i)
			}
		}
		if b := s.BranchFromIndex; b != nil && (*b < 0 || *b >= i) {
			return Input{}, fmt.Errorf("steps.%d.branchFromIndex: step %d names a branch parent (%d) that is not strictly before it", i, i, *b)
		}
	}
	return in, nil
}

// BuildSteps assigns stable step ids (<workflowId>-s<n>) and resolves each
// branchFromIndex into a real branch.from.
func BuildSteps(workflowID string, steps []StepInput) []osdata.WorkflowStep {
	ids := make([]string, len(steps))
	for i := range steps {
		ids[i] = fmt.Sprintf("%s-s%d", workflowID, i+1)
	}
	out := make([]osdata.WorkflowStep, len(steps))
	for i, s := range steps {
		tools := s.Tools
		if tools == nil {
			tools = []string{}
		}
		st := osdata.WorkflowStep{ID: ids[i], Title: s.Title, Detail: s.Detail, OwnerKind: s.OwnerKind, Owner: s.Owner,
			HoursPerWeek: s.HoursPerWeek, Tools: tools}
		if a := s.Automation; a != nil {
			st.Automation = &osdata.Automation{Title: a.Title, State: a.State, RecoveredUSD: a.RecoveredUSD}
		}
		if b := s.BranchFromIndex; b != nil && *b >= 0 && *b < len(ids) {
			cond := ""
			if s.BranchCondition != nil {
				cond = strings.TrimSpace(*s.BranchCondition)
			}
			if cond == "" {
				cond = "branch"
			}
			st.Branch = &osdata.Branch{From: ids[*b], Condition: cond}
		}
		out[i] = st
	}
	return out
}

// ToInput turns a stored workflow back into the builder shape (edit flow).
func ToInput(w osdata.Workflow) Input {
	idx := map[string]int{}
	for i, s := range w.Steps {
		idx[s.ID] = i
	}
	in := Input{Name: w.Name, Subtitle: w.Subtitle, Steps: make([]StepInput, len(w.Steps))}
	for i, s := range w.Steps {
		si := StepInput{Title: s.Title, Detail: s.Detail, OwnerKind: s.OwnerKind, Owner: s.Owner, HoursPerWeek: s.HoursPerWeek, Tools: s.Tools}
		if a := s.Automation; a != nil {
			si.Automation = &AutomationInput{Title: a.Title, State: a.State, RecoveredUSD: a.RecoveredUSD}
		}
		if b := s.Branch; b != nil {
			if j, ok := idx[b.From]; ok {
				si.BranchFromIndex = &j
			}
			c := b.Condition
			si.BranchCondition = &c
		}
		in.Steps[i] = si
	}
	return in
}

var nonSlug = regexp.MustCompile(`[^a-z0-9]+`)

// Slugify is the id stem for a new workflow: ascii, trimmed, never empty.
func Slugify(name string) string {
	s := strings.Trim(nonSlug.ReplaceAllString(strings.ToLower(name), "-"), "-")
	if len(s) > 40 {
		s = s[:40]
	}
	if s == "" {
		return "workflow"
	}
	return s
}

// NewID is wf-<slug>-<6 hex>: a double submit makes two workflows, never
// corrupts one.
func NewID(name string) string {
	b := make([]byte, 3)
	_, _ = rand.Read(b)
	return "wf-" + Slugify(name) + "-" + hex.EncodeToString(b)
}
