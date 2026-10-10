# Agentic Loops

An agentic loop is a repeatable work cycle where an agent keeps acting,
checking, and correcting until a goal is done or a stop condition is reached.

It is not magic autonomy. It is a governed loop:

```text
goal
  -> context
  -> plan/checklist
  -> action
  -> validation
  -> correction
  -> evidence/observation
  -> memory/update
  -> done or continue
```

Optimal Engine should make these loops explicit so people can reuse them safely
instead of relying on one-off prompts.

## Why Loops Matter

Older agent use looked like this:

```text
human prompt
  -> agent output
  -> human manually checks
  -> human prompts again
```

The useful loop is:

```text
human goal
  -> agent gets governed context
  -> agent works through a checklist
  -> agent validates each step
  -> agent fixes failures
  -> agent records what changed
  -> human reviews the final result
```

The key is validation. If the agent cannot check whether it is succeeding, the
loop can burn time, tokens, and trust.

## Loop Object Shape

A loop should be described as a governed object:

```yaml
loop_name: weekly_code_maintenance
owner_node_id: node_product_platform
trigger: manual_or_scheduled
goal: Find maintainability improvements and open a reviewed change.
context_sources:
  - node_product_platform
  - recent_pull_requests
allowed_tools:
  - engine_cli
  - git_cli
  - test_runner
validation_gates:
  - compile
  - tests
  - security_review
  - performance_review
  - maintainability_review
outputs:
  - observations
  - pending_claims
  - pull_request_or_patch
  - workflow_trace
stop_conditions:
  - all_checklist_items_verified
  - validation_failed_three_times
  - human_approval_required
```

This can later become a Workflow Trace, Procedural Memory Object, or Skill
Package after it has been proven useful.

## Create A Loop With Your Own Agent

Users should be able to create loops with whichever agent surface they already
use: a coding agent CLI, a desktop assistant, an MCP-capable chat app, a custom
script, a scheduler, or an internal agent service.

The engine should not force one agent product. It should give the agent a clean
loop contract:

```text
1. Select organization and workspace.
2. Select the owning Node, or ask the user to create/choose one.
3. Define the loop goal.
4. Define the allowed tool surfaces.
5. Retrieve a governed Context Package.
6. Open or attach an Active Memory Pool.
7. Work through phases and validation gates.
8. Record outputs as observations, Source Packages, pending Claims, packages,
   exports, workflow traces, or audit records.
9. Stop when validation passes, policy blocks, retries are exhausted, or human
   review is required.
```

The practical command pattern is:

```bash
mix optimal.initiate my-workspace --name "My Workspace" --dump setup.md
mix optimal.rag "what context should this loop use?" --workspace default:my-workspace --trace
```

Then give the agent the loop prompt:

```text
Use templates/starter-prompts/agentic-loop-design.md.
Create a governed loop for [goal].
Owning workspace: [workspace id].
Owning Node: [node id or unknown].
Allowed surfaces: [CLI/MCP/API/script/scheduler/A2A].
Do not mark complete until validation gates pass.
Record observations and pending Claims instead of writing truth directly.
```

If the agent is running outside Optimal Engine, the same rule still holds: it
uses Optimal Engine for scope, context, memory, and audit. The agent can use
outside tools, but the work products should come back through the governed
lifecycle.

```text
Agent CLI / app / service
  -> asks Optimal Engine for workspace + Node scope
  -> receives Context Package
  -> uses allowed CLI/MCP/API/script/A2A surfaces
  -> returns artifacts and observations
  -> engine records Source Packages, pending Claims, workflow traces, audit
```

### Loop Placement In The Workspace

Loop definitions are projections of governed workflow/skill state. Put local
loop files near the Node they operate on:

```text
workspaces/company-os/
  nodes/operation-weekly-review/
    loops/
      weekly-review.loop.yaml
  nodes/product-customer-portal/
    loops/
      release-maintenance.loop.yaml
  nodes/learning-ai-research/
    loops/
      youtube-learning.loop.yaml
```

A workspace-wide loop belongs under the workspace only when it intentionally
spans multiple Nodes and names those Nodes in its manifest.

```text
workspaces/company-os/
  loops/
    company-weekly-review.loop.yaml
```

Do not hide loop definitions in an agent prompt history. The prompt can launch
the loop, but the loop contract should be stored as a file projection or a
governed workflow/skill record so other agents and humans can inspect it.

### Minimal Loop Manifest

```yaml
loop_id: loop_weekly_review
display_name: Weekly Review
owner_workspace_id: default:company-os
owner_node_id: node_operation_weekly_review
trigger:
  type: manual_or_schedule
  schedule: weekly
goal: Review open decisions, blockers, stale context, and follow-ups.
allowed_surfaces:
  - engine_cli
  - calendar_connector
  - task_api
context:
  required_nodes:
    - node_operation_weekly_review
  context_package_query: "weekly review open decisions blockers follow ups"
phases:
  - name: collect_context
    validation: context_package_created
  - name: summarize_state
    validation: source_links_present
  - name: propose_updates
    validation: pending_claims_created
  - name: publish_review
    validation: export_manifest_complete
stop_conditions:
  - validation_passed
  - human_review_required
  - failed_three_times
outputs:
  - observation
  - pending_claim
  - workflow_trace
  - markdown_export
review_policy:
  external_sends_require_confirmation: true
  facts_require_review: true
```

## Common Loop Types

| Loop | Purpose | Validation |
| --- | --- | --- |
| Feature implementation | Build a scoped feature from a plan. | tests, compile, API checks, screenshots where relevant. |
| Security audit | Check for auth, injection, secrets, unsafe writes, privilege creep. | security checklist, tests, static checks, human review. |
| Performance audit | Improve latency, bundle size, page load, query speed. | benchmarks, instrumentation, browser checks, before/after numbers. |
| Maintainability audit | Reduce duplication, clarify ownership, improve interfaces. | compile/tests, small diff review, architecture rules. |
| Wiki refresh | Keep projected wiki pages current. | citation checks, stale page checks, source links. |
| Context refresh | Refresh stale Context Packages. | retrieval package rebuilt and old package marked refreshed. |
| Connector sync | Pull new data from an outside system. | sync run recorded, payloads preserved, failed items isolated. |
| Agent delegation | Ask another agent to complete or review part of the work. | A2A delegation recorded, returned artifacts preserved, review gate satisfied. |
| Learning loop | Turn videos/books/articles into notes and review items. | sources preserved, transcript/notes linked, pending Claims reviewed. |
| Package loop | Build a proposal, contract, SOP, handoff, or report. | manifest complete, source links present, review status satisfied. |

## Validation Gates

Every loop should have explicit gates. Examples:

```text
compile
unit tests
integration tests
end-to-end tests
browser/screenshot checks
API contract checks
security checklist
performance benchmark
access-control check
source-link check
package manifest check
human review
```

For code loops, tests and compile checks are usually the minimum. For UI loops,
browser screenshots or Playwright-style checks are often needed. For package or
wiki loops, source links and review state matter more than code tests.

## How Loops Map To Optimal Engine

```text
goal prompt or plan file
  -> Source Package

checklist
  -> Workflow Trace candidate

loaded task context
  -> Context Package

working session
  -> Active Memory Pool

tool/model/API/CLI/MCP/A2A calls
  -> governed call runs

new findings
  -> observations and pending Claims

successful repeated pattern
  -> Procedural Memory Object / Skill Package candidate
```

The loop should not skip the truth lifecycle. Output from a loop is still not a
Fact until review or policy accepts it.

A2A is only for agent-to-agent work. Use it when a remote agent is the right
collaborator. Use CLI, MCP, APIs, connectors, scripts, or schedulers when the
outside surface is a file, repo, database, calendar, SaaS app, media tool, or
local command.

## Personal, Learning, And Research Loops

Loops are not only for code.

Examples:

```text
YouTube learning loop
  -> receive video URL
  -> fetch transcript through an allowed surface
  -> preserve URL/transcript as evidence
  -> summarize into Learning Node
  -> create pending Claims for durable lessons
  -> set review reminder

book research loop
  -> search/catalog candidate books
  -> store source links and metadata
  -> label by topic Node
  -> create reading queue
  -> schedule review

weekly review loop
  -> retrieve current Context Package
  -> summarize open decisions, blockers, and follow-ups
  -> update rhythm projection
  -> create pending Claims or tasks
```

The tool surface can vary:

```text
YouTube transcript API
browser extension
MCP browser/fetch tool
REST transcript/search/channel/playlist provider
agent skill connected to a transcript provider
CLI downloader/transcriber where allowed
manual transcript upload
hosted transcription provider
local transcription adapter
```

The storage path stays the same:

```text
URL/file/transcript
  -> Source Package / Asset
  -> adapter or import run
  -> transcript/extraction rows
  -> pending Claims
  -> reviewed memory
```

## Risks

Agentic loops fail when they optimize for "done" without proving quality.

Common risks:

```text
weak validation
unbounded token/tool spend
loop keeps patching symptoms
security/performance/maintainability skipped
agent writes truth directly
agent uses wrong tool surface
agent calls tools outside permission
human review is needed but not requested
```

The fix is not more terminology. The fix is a better loop contract:

```text
clear goal
bounded scope
right context
allowed tools
explicit validation
stop conditions
review gate
memory/audit path
```
