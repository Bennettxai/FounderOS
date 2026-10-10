# Tool Surfaces And Loops

Optimal Engine does not assume every outside action should be MCP. Agents can
use several tool surfaces:

```text
CLI commands
MCP tools
HTTP/API calls
connector syncs
local scripts
scheduled jobs
model calls
A2A remote agents
```

The engine's job is to provide context, scope, policy, source preservation,
memory, and audit. The agent or worker performs the outside action through the
right surface.

## Agent-To-Tool Vs Agent-To-Agent

Separate these two problems:

```text
agent -> tool/data
  -> CLI, MCP, API, connector, script, scheduler

agent -> agent
  -> A2A or another agent-to-agent protocol
```

MCP is for giving an agent access to tools, APIs, files, databases, and other
resources. A2A is for agents discovering and coordinating with other agents.
They are complementary.

Optimal Engine can sit underneath both:

```text
Optimal Engine
  -> stores workspace topology, source evidence, memory, context packages,
     package manifests, tool grants, observations, and audit

Local or remote agent
  -> uses Optimal Engine context
  -> calls tools through CLI/MCP/API/connectors/scripts
  -> collaborates with other agents through A2A when needed
  -> returns observations and evidence to Optimal Engine
```

## CLI And MCP Are Both Tool Surfaces

CLI means the agent uses the same terminal commands a developer would use.

Examples:

```bash
ls
cat notes.md
rg -n "agent" .
grep -n "agent" *.md
curl -s https://example.com
git status
git log --oneline -10
docker ps
ffmpeg -i video.mp4 audio.wav
```

MCP means a dedicated server exposes structured tools with names, descriptions,
and JSON schemas.

Examples:

```text
read_file
search_files
fetch_url
calendar.read
email.send
crm.update_contact
```

Both are valid. The wrong move is treating one as universally better.

## A2A Is A Coordination Surface

A2A is useful when one agent needs to delegate or coordinate with another agent
that may be built by a different team, framework, vendor, or runtime.

Typical A2A concepts:

```text
Agent Card
  -> describes an agent's identity, endpoint, capabilities, skills, and auth

task request
  -> asks another agent to do work

task response / artifact
  -> returns result, file, structured data, or status

streaming updates
  -> long-running work can report progress over time
```

Use A2A when the remote thing is itself an agent with its own reasoning,
capabilities, tools, memory, or workflow. Do not model a database, file system,
or calendar as an agent just to use A2A. Those are tool/data surfaces.

## Decision Rule

| Use this | When |
| --- | --- |
| CLI | The command maps directly to the job and the model already knows the tool. |
| MCP/API/connector | Auth, IDs, schemas, SaaS state, browser rendering, per-user access, or safe writes matter. |
| Script/cron/scheduler | The operation is repeatable or timed. |
| A2A | Another agent needs to receive, negotiate, delegate, coordinate, stream progress, or return artifacts. |

Practical examples:

```text
Read/search local markdown -> CLI is usually enough.
Inspect Git state -> CLI is usually enough.
Render a JavaScript-heavy webpage -> MCP/browser/fetch service is usually better.
Send email as a user -> MCP/API/connector is usually better.
Sync Slack/CRM/calendar -> connector or MCP is usually better.
Run ffmpeg/yt-dlp/whisper locally -> CLI or script is usually better.
Ask another specialist agent to review a design or place an order -> A2A is usually better.
```

## Governance Rule

The execution surface can vary. The governance path should not.

```text
request
  -> actor/workspace/node scope
  -> permission/grant check
  -> execute through CLI, MCP, API, connector, script, scheduler, or A2A
  -> capture output
  -> preserve useful output as Source Package or observation
  -> create pending Claim when knowledge-bearing
  -> audit run and derivation
```

If the action writes to the outside world, such as sending email, updating a CRM,
posting a message, changing calendar events, or running a deployment, the agent
should require the appropriate grant and confirmation policy.

## What Optimal Engine Does And Does Not Do

Optimal Engine stores and governs the operating context. It does not need to be
the email client, calendar app, video downloader, CRM, or browser.

Correct model:

```text
Optimal Engine
  -> provides context package, memory, package manifest, policy, and audit
Agent/tool surface
  -> sends email, calls API, runs CLI command, executes MCP tool,
     schedules job, or delegates to another agent through A2A
Tool output
  -> returns to Optimal Engine as evidence, observation, or pending Claim
```

For example:

```text
"Send this client proposal"
  -> retrieve client/project/package context from Optimal Engine
  -> verify receiver, channel, and review state
  -> agent uses email MCP/API/CLI/script to send
  -> delivery receipt becomes Source Package evidence
  -> package record gets delivery/audit metadata
```

## Common Personal And Learning Loops

Optimal Engine is not only for companies. The same loop works for personal,
learning, research, and creator workflows.

| User goal | Likely tool surface | Engine records |
| --- | --- | --- |
| Save a YouTube video watched for learning. | Browser extension, manual URL, CLI downloader, transcript API, or MCP/browser tool. | Source Package, learning Node link, transcript/summary as derived output, pending Claims. |
| Turn a video into a script or study notes. | CLI/media tools, hosted transcript tool, model call. | Asset, adapter run, transcript rows, draft package/export, Claims if knowledge-bearing. |
| Research books and store them. | Browser/search API, library/catalog API, manual import, notes folder. | Source Package, book/reference Node or Context Node, reading status, reminders. |
| Set a reminder to revisit something. | Calendar/task API, MCP, CLI, or scheduler. | Observation, schedule metadata, pending Claim if it records a commitment. |
| Build a weekly learning review. | Scheduled job plus retrieval/context package. | Active Memory Pool, review report export, observations, workflow trace. |
| Keep a company wiki current. | Wiki scheduler, connector sync, agent review. | Stale page detection, refreshed projections, source-linked updates. |
| Coordinate internal and external agents. | A2A plus Optimal Engine Context Package. | Delegated task record, returned artifacts, observations, audit. |

## Scheduled Loops

Scheduled work is how the system becomes self-updating without pretending an
agent is always thinking in the background.

Examples:

```text
every morning
  -> refresh calendar/task context
  -> assemble daily Context Package
  -> produce focus export

every hour
  -> sync selected connectors
  -> preserve new payloads
  -> create pending Claims

every week
  -> review stale Node pages
  -> summarize open decisions
  -> update package templates and workflows

after package delivery
  -> preserve sent artifact and receipt
  -> update package audit
  -> create follow-up reminder
```

Current runtime surfaces include:

```text
mix optimal.context.refresh_stale
OptimalEngine.MemoryCore.ContextRefreshScheduler
OptimalEngine.Wiki.Scheduler
mix optimal.connector
connector governed runs
```

External cron, a process manager, an app job runner, or an agent scheduler can
call these surfaces. The scheduled job should still use the same governance
path as an interactive agent.

## Setup Inventory For Tooling

During workspace setup, ask:

```text
Which CLIs do you already use?
Which MCP servers do you already use?
Which APIs/scripts should agents be allowed to call?
Which other agents should this agent be allowed to contact?
Do any agents expose A2A Agent Cards?
Which tools are read-only?
Which tools can write or send things?
Which actions require confirmation?
Which scheduled loops should run daily, weekly, or on events?
Where should outputs be stored?
```

If a tool produces useful knowledge, the output should come back as evidence or
an observation. If it only helps perform a transient action, it can remain a tool
run record.

If another agent produces useful work through A2A, its returned artifact should
be treated the same way: preserve it, link the task, and review knowledge-bearing
claims before they become Facts.
