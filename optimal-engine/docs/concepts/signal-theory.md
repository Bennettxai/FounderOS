# Signal Theory

Signal Theory is the engine's classification model for turning noisy input into
structured operating context.

The engine receives messy human language, files, events, API payloads, tool
outputs, media, and agent observations. It must not treat all of that as generic
text. It first asks what kind of signal arrived.

## Core Shape

```text
S = (M, G, T, F, W)

M = Mode
G = Genre
T = Type
F = Format
W = Structure
```

| Dimension | Question | Examples |
| --- | --- | --- |
| Mode | What medium carries the signal? | text, image, audio, video, code, table, event, multimodal |
| Genre | What kind of work communication is it? | note, decision, transcript, spec, runbook, brief, report, incident, proposal |
| Type | What action does it perform? | inform, direct, decide, ask, commit, warn, delegate, evidence |
| Format | What container is it encoded in? | markdown, JSON, PDF, image, audio, video, email, API payload, calendar event |
| Structure | What internal skeleton does it follow? | freeform note, checklist, meeting log, ADR, table, contract, issue, postmortem |

These dimensions stay separate. A markdown file can be a decision, meeting note,
spec, SOP, report, or package manifest. A PDF can contain a contract, deck,
invoice, research paper, or scanned image. A calendar event can be an event
source, a meeting signal, or evidence for a later Fact.

## Why It Exists

Signal classification answers:

```text
What is this?
Where should it route?
Which Node owns the context?
Which parser or adapter should process it?
What Claims can be extracted from it?
How much review is needed?
Which package, report, wiki page, or app view should display it?
```

Without Signal classification, the engine creates noise:

```text
wrong Node
wrong package
wrong workflow
wrong receiver
wrong level of detail
unsupported "facts"
```

## Signal Is Not Truth

Signals are interpretations, not accepted truth.

```text
Source Package preserves what arrived.
Signal classifies what kind of thing it is.
Claim records what the source appears to say.
Fact records what review/policy accepts as true.
Memory Object records why accepted truth matters.
```

This separation is the main safety rule. The engine can classify messy input
without believing it.

## Signal Breakdown Example

User writes:

```text
Send the updated launch packet to the partner team after today's pricing call.
```

Possible classification:

```text
Mode: text
Genre: operational instruction
Type: direct
Format: chat message
Structure: task request
Primary Node: Project Node / Partner Launch
Receiver: partner team
Package implication: update or create a Node-local package
Truth implication: create pending Claims only if new factual assertions appear
```

The engine should not randomly create a workspace-level package. The request is
about a launch Node and a receiver. The package belongs under that Node unless
the package intentionally spans multiple Nodes.

## Signal-To-Node Routing

Routing should use this order:

```text
explicit workspace/node ID
  -> current session scope
  -> scoped aliases
  -> signal genre/type hints
  -> entity relationships
  -> fuzzy candidates with confirmation
  -> inbox quarantine
```

If the engine is not sure, it should ask or route to the workspace inbox. It
should not make durable topology or package placement guesses.

## Signal-To-Package Routing

A package is a deliverable bundle for a receiver or channel. It may contain
markdown, PDFs, HTML, images, JSON, attachments, or zipped artifacts.

```text
Node-specific package
  -> nodes/<node-slug>/packages/<package-slug>/

Cross-node package
  -> workspace-level package only when the manifest explicitly lists source Nodes
```

The package manifest should say:

```text
receiver
channel
purpose
source_node_ids
source_object_links
included_files
generated_by
review_status
```

If the package exists because of one Node, keep it inside that Node.

## Anti-Noise Rules

| Rule | Reason |
| --- | --- |
| Keep Mode, Genre, Type, Format, and Structure separate. | One label cannot route or process every input correctly. |
| Do not treat Signal as Fact. | Classification is not verification. |
| Do not create workspaces from project names. | Projects are normally Nodes. |
| Do not put Node packages at workspace root. | Packages need ownership, receiver, source links, and review path. |
| Ask when aliases are ambiguous. | Clarification prevents corrupt context. |
| Use inbox quarantine for uncertain routing. | It is better to review than to pollute a Node. |

## What Good Looks Like

Good Signal handling produces:

```text
preserved source
clear classification
correct workspace and Node scope
candidate Claims
review path
receiver-aware package/export placement
source-linked retrieval later
```

Bad Signal handling produces unsupported summaries, misplaced files, wrong
packages, duplicate Nodes, and agent work that cannot be audited.
