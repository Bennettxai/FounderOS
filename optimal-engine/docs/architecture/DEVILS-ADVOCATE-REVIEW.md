# Devil's Advocate Review

This page records the main ways the Optimal Engine architecture can fail if it
is built carelessly.

The goal is not to weaken the design. The goal is to turn the uncomfortable
questions into build rules.

## Highest-Risk Mistakes

| Risk | Why it matters | Guardrail |
| --- | --- | --- |
| Node becomes a junk drawer. | If every concept is a Node with loose metadata, the model becomes hard to query and govern. | Keep universal Node fields small, use explicit Node Types, use typed relationships, and add profile tables only when stable fields earn schema. |
| Markdown and runtime state drift. | If files and database state both pretend to be truth, humans and agents will see different realities. | Treat markdown as projection/editing surface. Edits re-enter as Source Packages or governed topology changes. Exports need manifests, hashes, and invalidation. |
| Agents write truth too easily. | Tool results and generated text can be wrong, stale, or unauthorized. | Agent outputs become observations or pending Claims. Facts require policy, evidence, or review. |
| Retrieval looks smart but leaks. | Semantic search, graph expansion, summaries, and citations can expose restricted context. | Apply authorization before retrieval expansion, during graph traversal, during summarization, and during export/tool calls. |
| Too many interfaces arrive before the core works. | HTML, dashboards, agents, apps, and CLI can encode temporary assumptions. | Build one vertical slice first: Node -> Source -> Claim -> Fact -> Context Package -> Markdown/HTML projection. |
| Skills become prompt files. | A prompt is not a governed reusable procedure. | Skill Packages need sources, preconditions, steps, tools, validation checks, exceptions, permissions, and audit. |
| Store becomes the business layer. | If low-level Store decides meaning, lifecycle logic spreads everywhere. | Store executes persistence. WorkspaceTopology, MemoryCore, Retrieval, Workflow, Skill, Tool, and Model layers own meaning. |
| Relationship edges explode. | Too many weak edges make graph traversal noisy and dangerous. | Use typed edge policies, confidence, source evidence, direction, lifecycle state, and permission labels. |
| Custom metadata hides real schema. | If everything stays JSON forever, query and validation quality suffer. | Promote repeated, operationally important metadata into profile tables. |
| Profile tables arrive too early. | Premature schema creates rigid concepts before real use proves them. | Start with Node + metadata; add profiles only after repeated use and query pressure. |

## Hard Answers

### Should every thing be a Node?

No.

A thing should become a Node when it needs identity, lifecycle, relationships,
review, permissions, durable context, or independent retrieval. A one-off task,
note, source, artifact, or temporary output should attach to an existing Node.

### Should projects be a separate hierarchy?

No, not by default.

The stable hierarchy is:

```text
Workspace -> Nodes
```

A project is normally:

```text
Node(node_type = project)
```

Use `project_profiles` only when project-specific fields need stable schema.

### Should markdown be source of truth?

Markdown is a human-operable projection and editing surface. It can become new
source evidence when edited, but canonical identity, lifecycle, permissions, and
promotion state live in the engine.

### Should the app come before the lifecycle?

No.

The app should display and control objects whose lifecycle already exists. Build
the app around a working vertical slice, not around speculative screens.

### Should an Active Memory Pool be called a workspace?

No.

Workspace is the top-level operating container. Active Memory Pool is a
task-scoped working context inside a Workspace.

### Should one database hold everything?

Yes initially, but ownership must remain explicit.

```text
One physical store.
Many lifecycle owners.
```

The database can hold all layer data, but each layer owns its own write
lifecycle.

## Safer Build Path

The safer sequence is:

```text
Node Core
  -> Source Package
  -> Claim / Fact
  -> Memory Object
  -> Retrieval Package
  -> Context Package
  -> Markdown / HTML projection
  -> Active Memory Pool
  -> Agent observation loop
  -> Workflow / Skill promotion
```

Do not begin with:

```text
many domain tables
many interfaces
loose search answers
agent-written truth
stale generated exports
```

The first proof should be boring and complete:

```text
Human creates a Node.
Human adds a note.
Engine preserves the source.
Engine extracts a Claim.
Human/policy promotes a Fact.
Engine creates a Memory Object.
Retrieval returns a source-linked Context Package.
Engine generates a markdown/HTML projection.
Agent can use the same Context Package and write an observation back.
```

That proves the core without overbuilding the surface.
