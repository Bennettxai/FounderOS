# Integrations And Imports

Most users do not start from an empty system. They already have context spread
across chat, email, calendar, calls, docs, project tools, CRM, code repos,
spreadsheets, local files, and older knowledge systems.

Optimal Engine should not force those tools into one flat import. It should help
the user identify where information lives, preserve it as source evidence, route
it into the right Workspace and Nodes, and keep the original channel meaning.

## Integration Inventory

During setup, ask the user where their work already happens.

| Channel or system | Common examples | What to import first | Engine meaning |
| --- | --- | --- | --- |
| Email | Gmail, Outlook | important threads, attachments, sent proposals, approvals | Source Packages, Claims, people/company links, package evidence |
| Calendar | Google Calendar, Outlook Calendar | meetings, attendees, descriptions, recurrence, links | event Signals, Person/Project/Operational Node links, rhythm state |
| Documents | Google Drive, OneDrive, Dropbox, local folders | docs, PDFs, decks, spreadsheets, forms | assets, Source Packages, document Signals, package inputs |
| Chat | Slack, Teams, Discord | channels, threads, decisions, files, handoffs | conversation Sources, Claims, decisions, open questions |
| Calls and meetings | Zoom, Meet, Teams, Fathom, recordings/transcripts | recordings, transcripts, summaries, attendees | audio/video assets, transcripts, Episode Objects, pending Claims |
| Project tools | Linear, Jira, Asana, Trello, ClickUp | issues, tasks, statuses, milestones, comments | Project/Operational Node state, blockers, workflow traces |
| CRM and sales | HubSpot, Salesforce, Pipedrive, Airtable | accounts, contacts, deals, notes, emails, stages | Entity/Person/Client/Deal Nodes, package lifecycle, relationship history |
| Code and product | GitHub, GitLab, local repos | issues, PRs, commits, releases, docs | Product/Project Nodes, implementation history, decisions |
| Finance and ops | accounting exports, invoices, banking CSVs, metrics sheets | CSVs, reports, invoices, dashboards | money/metrics Nodes, Facts after review, reports |
| Knowledge systems | Notion, Confluence, Obsidian, wikis, old folders | pages, databases, backlinks, exports | workspace import Sources, candidate Nodes, wiki projections |
| Custom systems | MCP servers, APIs, scripts, cron jobs, internal tools, A2A agents | schemas, endpoints, payloads, logs, Agent Cards | governed connector/tool/agent definitions and Source Packages |

These examples are not requirements. They are prompts to help a person find the
places where their context already lives.

## Connection Types

Different users connect systems in different ways. Treat each connection as a
governed integration surface.

| Type | Use when | Engine record |
| --- | --- | --- |
| Manual import | User uploads/export files, CSVs, zip files, markdown folders, call transcripts. | Source Packages, assets, import run, pending Claims. |
| Connector sync | A known provider has a sync adapter. | connector definition, sync run, Source Packages, assets, audit. |
| MCP tool | Agent needs interactive access through an MCP server. | tool definition, grants, tool call runs, observations. |
| CLI tool | A command-line tool maps directly to the job. | registered command/script surface, run record, captured output, Source Package or observation when useful. |
| API integration | Organization has a REST/GraphQL/internal API. | connector/tool definition, schema, policy, call runs. |
| Script or cron | Organization has local jobs or one-off scripts. | script tool definition, schedule, run record, Source Packages. |
| A2A agent | Agent needs to delegate or coordinate with another agent. | remote Agent Card, grants, delegated task run, returned artifacts, observations. |
| Workspace edit | Human or agent edits markdown/files directly. | Source Package or topology change request, then projection refresh. |

The same outside system can have more than one connection type. For example, a
team may import old calendar exports manually, then later configure a calendar
MCP or connector for live use.

See [Tool surfaces and loops](tool-surfaces-and-loops.md) for the CLI vs MCP vs
API vs A2A decision rule.

## Setup Questions

Use these questions during workspace initiation:

```text
Which organizations or workspaces do you want to manage?
Where do your conversations happen?
Where do meetings and calls live?
Where are documents, proposals, contracts, and SOPs stored?
Where are projects, tickets, and tasks tracked?
Where are customers, partners, or relationships tracked?
Where are code, product, or release records stored?
Which tools should agents be allowed to call?
Which command-line tools do you already use?
Which MCP servers do you already use?
Which other agents or A2A Agent Cards should this workspace know about?
Which scheduled loops or cron jobs should run?
Which systems are read-only, and which can be written to?
Which data is private, sensitive, or restricted?
What packages do you commonly send to other people?
```

If the user does not know, the setup agent should create open questions instead
of inventing integrations.

## Import Flow

```text
Existing system or file export
  -> import run
  -> Source Packages and assets
  -> Signal classification
  -> candidate Nodes / aliases / relationships
  -> pending Claims
  -> human or policy review
  -> Facts, Memory Objects, workflow traces, or package templates
  -> markdown/wiki/API projections
```

The first import should favor preservation over cleverness. It is better to
preserve raw source evidence and create review queues than to over-route bad
guesses into durable memory.

## Mapping Old Systems

Old systems often have names that are too flat for Optimal Engine:

| Old shape | Problem | Better mapping |
| --- | --- | --- |
| One giant folder | No ownership, no routing, weak retrieval. | Workspace plus Nodes by purpose. |
| One knowledge-base page per topic | Pages mix facts, notes, tasks, and decisions. | Sources, Claims/Facts, Memory Objects, wiki projections. |
| One CRM account with everything inside it | Deals, people, calls, contracts, and requirements collapse together. | Client/Entity Node plus Person, Deal/Project, Package, and Decision records. |
| Project tool as the only system of record | Tickets track work but not why decisions were made. | Project Node plus tickets as Sources/Signals and decisions as Claims/Facts. |
| Call summaries as truth | Summaries omit evidence and may be wrong. | Recording/transcript as Source Package; summary as derived output; Claims reviewed before Facts. |
| Random "packages" at workspace root | Receiver and ownership are unclear. | Node-owned package templates with manifest and source links. |

## CLI, MCP, API, Script, Connector, Or A2A

Do not force every integration into one shape.

```text
CLI
  -> best when a local command directly solves the job

MCP/API/connector
  -> best when auth, schemas, per-user access, audit, or SaaS state matter

script/cron
  -> best when the organization already has a repeatable local job

A2A
  -> best when the other side is another agent that can receive tasks,
     negotiate work, stream progress, or return artifacts
```

Examples:

| Job | Likely surface |
| --- | --- |
| Search local markdown files. | CLI: `rg`, `grep`, `find`. |
| Inspect a repo. | CLI: `git`, `ls`, `cat`, `rg`. |
| Download/process a video or audio file. | CLI/script: media tools, transcription adapters, FFmpeg. |
| Fetch a JavaScript-rendered page. | MCP/browser/fetch service. |
| Send email or update calendar. | MCP/API/connector with user grants and audit. |
| Sync chat, CRM, docs, or project systems. | Connector sync or provider API. |
| Run daily/weekly refresh. | Cron/scheduler calling engine CLI/API. |
| Delegate review or work to another agent. | A2A with Agent Card discovery and task/audit records. |

Whatever surface is used, useful output should return to Optimal Engine as a
Source Package, observation, pending Claim, package delivery record, or audit
event.

## Package Inventory During Setup

Every organization sends recurring bundles to people. Capture those early.

Examples:

| Package name | Receiver | Stage or situation | Typical contents | Owning Node |
| --- | --- | --- | --- | --- |
| Proposal | prospect, buyer, partner | sales or partnership evaluation | offer, scope, pricing, proof, next step | Deal, Client, Partnership, or Project Node |
| Contract | client, vendor, partner | agreement stage | terms, signatures, exhibits, attachments | Client, Vendor, Partnership, or Legal/Operational Node |
| SOP document | team, client, operator | onboarding or operations | steps, roles, checks, exceptions, links | Operational or Project Node |
| Client requirements document | client, internal delivery team | discovery or implementation | goals, constraints, acceptance criteria, dependencies | Client or Project Node |
| Handoff packet | team, client, implementation partner | transition | context, assets, decisions, open risks, contacts | Project or Client Node |
| Board or leadership report | executives, board, advisors | review cadence | metrics, decisions, risks, narrative, asks | Organization or Workspace-level package when cross-node |
| Onboarding packet | new hire, client, partner | onboarding | profile, access, docs, SOPs, first tasks | Person, Client, Team, or Operational Node |

The user can call these by any name. The engine should preserve display names
and aliases while keeping the canonical package manifest clear.

## Package Template Shape

```yaml
package_type: proposal
display_name: Launch Proposal
receiver_type: prospect
channel: email
stage: sales_evaluation
owning_node_id: node_deal_launch
source_node_ids:
  - node_deal_launch
required_sections:
  - executive_summary
  - scope
  - pricing
  - proof
  - next_steps
review_required: true
delivery_formats:
  - pdf
  - docx
  - zip
```

This does not mean every package must be generated by the engine. It means the
engine knows what the package is, who receives it, where it belongs, and which
source evidence supports it.

## Agent Rules For Integrations

Agents should follow this order:

```text
resolve organization/workspace scope
  -> identify the outside system or import source
  -> check connector/tool/agent grant
  -> preserve raw payloads and files
  -> classify Signals
  -> route uncertain items to inbox/review
  -> create pending Claims, not direct Facts
  -> update package templates only through owning Node or workspace scope
  -> schedule repeat work only through an explicit scheduler/cron/tool record
```

If an agent cannot determine receiver, channel, owning Node, or source objects
for a package, it should ask before writing package files.
