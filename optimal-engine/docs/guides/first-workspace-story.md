# First Workspace Story

Most people do not need another empty productivity app. They already have a
messy operating system spread across files, chats, calls, docs, calendars,
project tools, CRMs, and agent conversations.

Optimal Engine starts by accepting that reality.

The first workflow is:

```text
Bring what already exists
  -> preserve it as evidence
  -> classify the Signals
  -> propose Workspaces and Nodes
  -> ask about ambiguity
  -> review before durable truth
  -> render markdown/wiki/API/app views
  -> keep learning from work
```

## The Story

A user arrives with scattered context:

```text
markdown files
company wiki
calendar events
meeting transcripts
project tasks
CRM notes
proposal folders
contracts
SOPs
client requirements documents
agent chats
other agents
local scripts
MCP tools
command-line tools
custom APIs
```

Traditional systems usually do one of four things:

```text
store pages
search documents
run agent tasks
show dashboards
```

Optimal Engine connects those into one governed loop:

```mermaid
flowchart LR
  Existing[Existing work<br/>files, wiki, tools, calls, docs] --> Preserve[Preserve evidence]
  Preserve --> Signal[Classify Signals]
  Signal --> Topology[Propose Workspace + Nodes]
  Topology --> Review[Human/policy review]
  Review --> Memory[Claims, Facts, Memory Objects]
  Memory --> Context[Context Packages]
  Context --> Agent[Human or agent work]
  Agent --> Observation[Observations]
  Observation --> Review
  Memory --> Projection[Markdown / wiki / API / packages]
```

The system is self-learning because work feeds back into memory. It does not
mean the agent invents truth. It means every useful observation can re-enter the
governed lifecycle:

```text
work happens
  -> observation
  -> pending Claim
  -> review/policy
  -> Fact or Memory Object
  -> context refresh
  -> workflow trace
  -> reusable Skill Package when validated
```

## What The User Does First

The user can start in three ways.

### 1. Data Dump

Use this when the user has messy context and wants the engine to propose
structure.

```bash
mix optimal.initiate my-workspace --name "My Workspace" --dump setup.md
```

By default, conservative Node candidates from explicit headings, labels, and
lists are applied immediately so the user gets a usable workspace. The dump
itself remains source evidence and the setup Claim remains unreviewed. Add
`--review-only` when every topology change must wait for human or policy review.

The dump can describe:

```text
what the company does
who the people are
which projects matter
where the wiki lives
which tools are connected
what documents are sent externally
what rhythm or review cadence exists
what packages are commonly created
```

### 2. Known Structure

Use this when the user already knows the starting Nodes.

```bash
mix optimal.setup my-workspace \
  --node project:launch:"Launch" \
  --node person:client-lead:"Client Lead" \
  --node operational:client-onboarding:"Client Onboarding"
```

### 3. Existing Wiki Or Folder

Use this when the user already has a company wiki, markdown vault, document
folder, or exported knowledge base.

```text
old wiki/folder/export
  -> import run
  -> Source Packages
  -> candidate Nodes
  -> pending Claims
  -> reviewed memory
  -> rebuilt projections
```

Use the starter prompt at:

```text
templates/starter-prompts/company-wiki-import.md
```

## What The Agent Does

The agent should not act like a magic organizer. It should follow the engine
rules:

```text
read or receive user context
  -> identify organization/workspace scope
  -> preserve raw input
  -> classify Signals
  -> propose Nodes, aliases, integrations, package types
  -> ask open questions
  -> run setup/initiation commands
  -> inspect output
  -> update projections
```

If it is not sure where something belongs, it routes to inbox/review instead of
polluting a durable Node.

## Communication Channels

People use different channels for different work. The setup should inventory
them.

| Channel | What it usually contains |
| --- | --- |
| Email | approvals, sent proposals, contracts, client follow-up |
| Calendar | meetings, recurrence, relationships, rhythm |
| Chat | decisions, handoffs, questions, files |
| Calls | transcripts, commitments, objections, requirements |
| Docs/wiki | reference, SOPs, strategy, project memory |
| CRM | relationships, deals, stages, requirements |
| Project tools | tasks, blockers, milestones, status |
| Code repos | implementation history, issues, releases |
| CLI tools | local files, Git, Docker, media processing, scripts, simple command work |
| Scripts/API/MCP | authenticated actions, structured tools, app writes, scheduled jobs |
| A2A agents | specialist agents, reviewer agents, partner agents, supplier agents |

Each channel becomes an integration surface. Each imported item becomes source
evidence before it becomes memory.

## Agent Tool Surfaces

Agents can act through several surfaces:

```text
CLI command
MCP tool
API call
connector sync
local script
scheduled job
A2A remote agent
```

The user should not have to know the implementation detail every time. The
engine should help the agent choose the right surface:

```text
simple local command -> CLI
authenticated app action -> MCP/API/connector
repeating operation -> scheduler/cron
another agent needs to collaborate -> A2A
knowledge-bearing output -> Source Package or observation
```

Optimal Engine provides context, memory, package manifests, permissions, and
audit. The agent/tool surface performs the outside action.

Example:

```text
User asks to send a proposal
  -> Optimal Engine retrieves client/project/package context
  -> agent confirms receiver/channel/review state
  -> email tool sends the package
  -> delivery receipt returns as Source Package evidence
```

Multi-agent example:

```text
Inventory agent detects low stock
  -> Optimal Engine loads product, vendor, and policy context
  -> inventory data is read through MCP/API/connector
  -> internal order agent receives the task through A2A
  -> supplier agent receives the order request through A2A when allowed
  -> confirmation returns as source evidence
  -> pending Claims, package records, and audit are created
```

## Self-Updating Loops

The system learns by closing loops, not by hallucinating durable truth.

```text
scheduled sync or agent action
  -> new source evidence
  -> Signal classification
  -> pending Claims
  -> review/policy
  -> Facts and Memory Objects
  -> refreshed Context Packages and projections
```

Examples:

```text
save watched videos to a Learning Node
turn transcripts into study notes
research books and label them by topic
set reminders from commitments
refresh project context before weekly review
sync new customer notes from CRM
rebuild stale wiki pages
```

The user can use the same architecture for business, learning, research,
personal operations, and agent work.

## Package Examples

Every organization sends things to people. Those things should be named in the
organization's language, then stored with the engine's lifecycle.

```text
Proposal
Contract
SOP document
Client requirements document
Handoff packet
Board report
Onboarding packet
Evidence packet
```

The engine stores:

```text
receiver
channel
stage
owning Node
source objects
required sections
review status
delivery artifact
audit record
```

That is why packages are not random folders. They are structured communication
bundles.

## Starter Prompts

Copy-paste starter prompts live here:

```text
templates/starter-prompts/
```

Use them with Codex, Claude Code, ChatGPT, MCP clients, or another agent that
can read the user's files.

The prompts are intentionally conservative. They tell the agent to preserve
evidence, propose structure, and ask before turning guesses into durable system
state.
