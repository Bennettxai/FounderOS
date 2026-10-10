package workflows

import (
	"context"
	"encoding/json"
	"errors"
	"regexp"
	"strings"
	"testing"

	"github.com/rhl/businessos-backend/internal/founderos/pages/osdata"
)

// Ported from FounderOS v1 tests/workflows-api.test.ts (shaping + validation)
// and tests/workflows-draft.test.ts (reply parsing + the drafting loop).

const basicInput = `{
  "name": "Test onboarding flow", "subtitle": "A workflow authored through the API test.",
  "steps": [
    {"title":"Kickoff","detail":"Opens the kickoff packet.","ownerKind":"agent","owner":"Test Agent","hoursPerWeek":1,"tools":["notion"],"automation":null,"branchFromIndex":null,"branchCondition":null},
    {"title":"Approved path","detail":"Runs once approved.","ownerKind":"agent","owner":"Test Agent","hoursPerWeek":0,"tools":[],"automation":{"title":"Auto-approve","state":"live","recoveredUsd":0},"branchFromIndex":0,"branchCondition":"approved"}
  ]}`

func TestParseInputAndBuildSteps(t *testing.T) {
	in, err := ParseInput([]byte(basicInput))
	if err != nil {
		t.Fatal(err)
	}
	steps := BuildSteps("wf-x", in.Steps)
	if steps[0].ID != "wf-x-s1" || steps[0].Branch != nil || steps[1].Branch == nil || steps[1].Branch.From != "wf-x-s1" || steps[1].Branch.Condition != "approved" {
		t.Fatalf("steps %+v", steps)
	}
	if steps[1].Automation.State != "live" || steps[0].Tools[0] != "notion" || steps[1].Tools == nil {
		t.Fatalf("fields %+v", steps)
	}
	back := ToInput(osdataWorkflow("wf-x", in.Name, steps))
	if *back.Steps[1].BranchFromIndex != 0 || *back.Steps[1].BranchCondition != "approved" || back.Steps[0].BranchFromIndex != nil {
		t.Fatalf("round trip %+v", back)
	}
	blank := BuildSteps("wf-y", []StepInput{{Title: "a", Detail: "d", OwnerKind: "human", Owner: "B", Tools: []string{}}, {Title: "b", Detail: "d", OwnerKind: "human", Owner: "B", Tools: []string{}, BranchFromIndex: ip(0), BranchCondition: sp("  ")}})
	if blank[1].Branch.Condition != "branch" {
		t.Fatalf("empty condition %+v", blank[1].Branch)
	}
}

func TestParseInputRejects(t *testing.T) {
	var base map[string]any
	_ = json.Unmarshal([]byte(basicInput), &base)
	mutate := func(f func(m map[string]any)) []byte {
		var m map[string]any
		_ = json.Unmarshal([]byte(basicInput), &m)
		f(m)
		raw, _ := json.Marshal(m)
		return raw
	}
	cases := map[string][]byte{
		"no name":        mutate(func(m map[string]any) { m["name"] = "" }),
		"zero steps":     mutate(func(m map[string]any) { m["steps"] = []any{} }),
		"forward branch": mutate(func(m map[string]any) { m["steps"].([]any)[0].(map[string]any)["branchFromIndex"] = 1 }),
		"bad kind":       mutate(func(m map[string]any) { m["steps"].([]any)[0].(map[string]any)["ownerKind"] = "robot" }),
		"neg hours":      mutate(func(m map[string]any) { m["steps"].([]any)[0].(map[string]any)["hoursPerWeek"] = -1 }),
		"no detail":      mutate(func(m map[string]any) { m["steps"].([]any)[0].(map[string]any)["detail"] = "" }),
		"missing steps":  []byte(`{"name":"Missing steps"}`),
		"not json":       []byte(`nope`),
		"bad state": mutate(func(m map[string]any) {
			m["steps"].([]any)[1].(map[string]any)["automation"].(map[string]any)["state"] = "maybe"
		}),
	}
	for name, raw := range cases {
		if _, err := ParseInput(raw); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
}

func TestSlugify(t *testing.T) {
	if Slugify("Test onboarding flow!") != "test-onboarding-flow" || Slugify("!!!") != "workflow" || len(Slugify(strings.Repeat("ab ", 40))) > 40 {
		t.Fatal("slug")
	}
	if !regexp.MustCompile(`^wf-test-onboarding-flow-[0-9a-f]{6}$`).MatchString(NewID("Test onboarding flow")) {
		t.Fatal(NewID("Test onboarding flow"))
	}
}

const validDraft = `{"name":"Weekly review","subtitle":"Every Friday.","steps":[{"title":"Pull the numbers","detail":"Gathers metrics.","ownerKind":"agent","owner":"Metrics Agent","hoursPerWeek":1,"tools":["stripe"],"automation":null,"branchFromIndex":null,"branchCondition":null}]}`

func envelope(s string) string {
	raw, _ := json.Marshal(map[string]string{"result": s})
	return string(raw)
}

func TestExtractAndParseDraft(t *testing.T) {
	if ExtractReplyText(`{"type":"result","result":"hello world"}`) != "hello world" || ExtractReplyText(`{"message":"fallback text"}`) != "fallback text" || ExtractReplyText("  plain  \n") != "plain" {
		t.Fatal("extract")
	}
	if d, err := ParseDraft(envelope(validDraft)); err != nil || d.Name != "Weekly review" {
		t.Fatalf("valid %v", err)
	}
	if _, err := ParseDraft(envelope("```json\n" + validDraft + "\n```")); err != nil {
		t.Fatalf("fenced %v", err)
	}
	if _, err := ParseDraft(envelope("Sure! Here is a plan")); err == nil || !strings.Contains(err.Error(), "not valid JSON") {
		t.Fatalf("prose %v", err)
	}
	if _, err := ParseDraft(envelope(`{"name":"Missing steps"}`)); err == nil {
		t.Fatal("schema")
	}
	bad := strings.Replace(validDraft, `"branchFromIndex":null`, `"branchFromIndex":0`, 1)
	if _, err := ParseDraft(envelope(bad)); err == nil {
		t.Fatal("forward branch")
	}
}

type fakeRunner struct {
	replies []CLIResult
	prompts []string
}

func (f *fakeRunner) Run(_ context.Context, prompt string) CLIResult {
	f.prompts = append(f.prompts, prompt)
	r := f.replies[0]
	if len(f.replies) > 1 {
		f.replies = f.replies[1:]
	}
	return r
}

func TestDraftLoop(t *testing.T) {
	ok := &fakeRunner{replies: []CLIResult{{OK: true, Stdout: envelope(validDraft)}}}
	if r := Draft(context.Background(), ok, "weekly review"); !r.OK || r.Draft.Name != "Weekly review" || len(ok.prompts) != 1 || !strings.Contains(ok.prompts[0], "Describe: weekly review") {
		t.Fatalf("clean %+v", r)
	}
	retry := &fakeRunner{replies: []CLIResult{{OK: true, Stdout: envelope("not json at all")}, {OK: true, Stdout: envelope(validDraft)}}}
	if r := Draft(context.Background(), retry, "x"); !r.OK || len(retry.prompts) != 2 || !strings.Contains(retry.prompts[1], "failed validation") {
		t.Fatalf("retry %+v %d", r, len(retry.prompts))
	}
	twice := &fakeRunner{replies: []CLIResult{{OK: true, Stdout: "nope"}}}
	if r := Draft(context.Background(), twice, "x"); r.OK || r.Unavailable || !strings.Contains(r.Error, "invalid draft twice") {
		t.Fatalf("twice %+v", r)
	}
	missing := &fakeRunner{replies: []CLIResult{{Unavailable: true, Err: errors.New("exec: claude: not found")}}}
	if r := Draft(context.Background(), missing, "x"); r.OK || !r.Unavailable || !strings.Contains(r.Error, "unavailable: build manually") {
		t.Fatalf("missing %+v", r)
	}
	broken := &fakeRunner{replies: []CLIResult{{Err: errors.New("exit 1")}}}
	if r := Draft(context.Background(), broken, "x"); r.OK || r.Unavailable || !strings.Contains(r.Error, "drafting assistant failed: exit 1") {
		t.Fatalf("failed %+v", r)
	}
}

func ip(n int) *int       { return &n }
func sp(s string) *string { return &s }

func osdataWorkflow(id, name string, steps []osdata.WorkflowStep) osdata.Workflow {
	return osdata.Workflow{ID: id, Name: name, Steps: steps}
}
