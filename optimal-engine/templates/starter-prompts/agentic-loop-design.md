# Agentic Loop Design Prompt

Use this when I want to create a repeatable agent loop for code, research,
learning, package generation, wiki refresh, connector sync, or scheduled review.

## Goal

Design a governed loop the agent can run until the task is complete, verified,
or blocked.

## What I Will Provide

I may provide:

- a feature plan
- a checklist
- a workspace or Node
- a research goal
- a package type
- a video/book/article URL
- a codebase
- a failing test or GitHub Action
- a recurring daily/weekly/monthly review goal

## Your Job

1. Identify the loop goal.
2. Identify the owning Workspace and Node.
3. If the owning Node is unclear, ask whether to create a new Node or attach
   the loop to an existing Node.
4. Identify required context sources.
5. Identify allowed tool surfaces: CLI, MCP, API, connector, script, scheduler,
   model call, or A2A agent.
6. Create a checklist with phases.
7. Define validation gates for every phase.
8. Define stop conditions.
9. Define what outputs should become Source Packages, observations, pending
   Claims, packages, exports, workflow traces, or audit records.
10. Identify when human review is required.
11. Recommend where the loop manifest should live in the workspace projection.

## Required Output

Return:

1. Loop summary.
2. Loop YAML.
3. Phase checklist.
4. Validation gates.
5. Tool permissions needed.
6. Stop conditions.
7. Memory/audit outputs.
8. Command or scheduler recommendation.
9. Workspace path recommendation, such as
   `nodes/<node-slug>/loops/<loop-name>.loop.yaml`.

## Rules

- Do not call the loop done unless validation passes.
- Do not let generated output become Fact directly.
- Do not use MCP when CLI is simpler.
- Do not use CLI when MCP/API/connector is needed for auth, schemas, or writes.
- Do not use A2A unless the other side is another agent.
- Ask before write actions, external sends, purchases, deployments, or
  cross-organization delegation unless policy explicitly allows it.
- Do not hide the loop only in chat history. Store the loop contract as a
  workspace projection or governed workflow/skill record.
