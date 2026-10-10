# Ubiquitous Language

## System Boundary

| Term | Definition | Aliases to avoid |
| --- | --- | --- |
| **Optimal Engine** | The engine that turns source material into governed memory, retrieval packages, workflows, and skill packages. | OptimalOS engine, context engine, file engine |
| **Memory Core** | The target architecture where memory objects, provenance, retrieval, model calls, tools, and audit are governed as database objects. | RAG app, vector database, knowledge base |
| **Enterprise Memory Core** | The durable source of truth for source packages, claims, facts, memory objects, routes, workflows, skill packages, policies, and audit events. | Main database, brain, store |
| **Workspace Export** | A filesystem projection generated from governed database state for human browsing and local work. | Source of truth, node folder truth |
| **Local Memory Node** | A scoped local subset of governed memory used near the point of work and synchronized back to the Enterprise Memory Core. | Local cache, personal database |

## Memory Lifecycle

| Term | Definition | Aliases to avoid |
| --- | --- | --- |
| **Source Package** | A preserved raw input with source metadata, trust label, retention state, and source links. | Raw file, artifact, document |
| **Signal** | The first classified processing unit produced from a source or user input using S=(Mode, Genre, Type, Format, Structure). | Memory, fact, context |
| **Claim** | A source-backed assertion that has not yet been accepted as true. | Fact, extraction, note |
| **Fact** | An accepted assertion with scope, evidence, confidence, precision, validity windows, and lifecycle state. | Claim, truth, memory |
| **Memory Object** | The institutional meaning around claims, facts, sources, relationships, and time. | Summary, note, context |
| **Episode Object** | A governed memory object representing one concrete event or remembered episode. | Event, log entry, episodic memory |
| **Memory Detail Object** | An addressable reusable detail such as a step, command, parameter, validation check, exception, or decision point. | Bullet, paragraph, substep |
| **Workflow Trace** | The observed evidence-linked sequence of work inside one episode or case. | Process, runbook, workflow |
| **Generalized Workflow** | A repeatable candidate process pattern derived from multiple workflow traces or reviewed examples. | Workflow, automation |
| **Procedural Memory Object** | Validated how-to knowledge with preconditions, steps, checks, exceptions, applicability, and evidence. | Runbook, procedure, skill |
| **Skill Package** | A governed package of procedural memory, workflows, evidence, execution policy, validation checks, and audit rules. | Prompt, plugin, tool, skill file |

## Context and Retrieval

| Term | Definition | Aliases to avoid |
| --- | --- | --- |
| **Active Memory Pool** | A task-scoped working-memory space for humans, agents, tools, observations, pending claims, and loaded context. | Chat context, session memory |
| **Context Package** | A compact permission-filtered projection of memory, facts, workflow guidance, source links, and pool observations for a specific actor and task. | Prompt, answer, RAG result |
| **Retrieval Package** | A structured answer package containing returned objects, evidence links, confidence, precision, validity, and policy state. | Search result, chunk list |
| **Security-Aware Retrieval Coordinator** | The subsystem that plans retrieval with authorization, partitions, time mode, evidence requirements, confidence, precision, graph traversal, and audit. | Search, retriever |
| **Semantic Frame** | The structured meaning of a memory around subject, action class, target, purpose, outcome, constraints, and other frame fields. | Predicate, metadata, tags |
| **Relationship Edge** | A typed link between sources, claims, facts, memories, details, workflows, skill packages, entities, and time periods. | Link, relation, preposition |

## Governance

| Term | Definition | Aliases to avoid |
| --- | --- | --- |
| **Derivation Ledger** | The auditable lineage from source packages through derived claims, facts, memories, workflows, summaries, embeddings, and skill packages. | Audit log, provenance note |
| **Temporal Supersession Engine** | The subsystem that manages valid time, transaction time, staleness, contradictions, supersession, and current-valid state. | Versioning, history |
| **Confidence** | A policy-scored measure of reliability based on evidence, source quality, extraction quality, truth status, and verification. | Accuracy, certainty |
| **Precision** | A policy-scored measure of specificity across time, entity, relationship, location, semantic meaning, and scope. | Confidence, correctness |
| **Lifecycle State** | The operational state of a governed object, such as pending, current, historical, stale, superseded, contradicted, retired, or archived. | Status |
| **Partition** | A security and performance scope for memory by workspace, node, project, subject, source domain, time range, workflow, or pool. | Namespace, folder |
| **Access Policy** | The rule set controlling who may retrieve, inspect, summarize, promote, execute, or export governed objects. | Permission, ACL |

## Model and Tool Access

| Term | Definition | Aliases to avoid |
| --- | --- | --- |
| **Native Model Execution Engine** | The subsystem that invokes approved model calls through governed database operations and writes outputs as governed objects. | LLM wrapper, model API |
| **Model-Call Operation** | A registered model invocation with model identity, prompt or embedding config, input schema, output contract, policy, destination, and audit. | LLM call, prompt |
| **LLM Tool-Calling Access Surface** | The database-managed registry, dispatcher, validation, permission, protocol adapter, and audit surface for agent tool calls. | MCP server, tools |
| **Tool-Calling Protocol Adapter** | A protocol-specific adapter, initially MCP, that translates tool calls into the governed tool model. | MCP, API wrapper |
| **MCP Tool Definition** | A catalog object describing an MCP-accessible tool, its schema, implementation, privileges, routing, lifecycle, and audit policy. | Tool config, function |

## Relationships

- A **Source Package** produces zero or more **Signals**.
- A **Signal** may produce zero or more **Claims**, **Relationship Edges**, **Memory Objects**, and **Workflow Traces**.
- A **Claim** must link to at least one **Source Package** or source span.
- A **Fact** is promoted from one or more **Claims** or authoritative sources.
- A **Memory Object** may link many **Claims**, **Facts**, **Source Packages**, and **Relationship Edges**.
- An **Episode Object** is a specific kind of **Memory Object**.
- A **Memory Detail Object** may belong to a **Memory Object**, **Episode Object**, **Workflow Trace**, **Procedural Memory Object**, or **Skill Package**.
- A **Workflow Trace** is derived from one or more **Episode Objects** and **Memory Detail Objects**.
- A **Generalized Workflow** is derived from multiple **Workflow Traces**.
- A **Procedural Memory Object** is promoted from reviewed **Generalized Workflows** and supporting evidence.
- A **Skill Package** packages one or more **Procedural Memory Objects** for governed human or AI use.
- An **Active Memory Pool** loads **Context Packages** and may produce observations that become pending **Claims**.
- A **Context Package** is a projection, not durable truth.
- A **Workspace Export** is derived from the **Enterprise Memory Core**, not the source of truth.

## Example Dialogue

> **Dev:** "When a document enters Optimal Engine, do we store it as a Memory Object immediately?"
>
> **Domain expert:** "No. It first becomes a **Source Package**. The classifier can create a **Signal**, and extraction may create **Claims**."
>
> **Dev:** "When does a Claim become a Fact?"
>
> **Domain expert:** "Only after policy or review accepts it with evidence, confidence, precision, validity windows, and lifecycle state."
>
> **Dev:** "Where do node folder files fit?"
>
> **Domain expert:** "Those are **Workspace Exports**. They help humans browse, but the **Enterprise Memory Core** is the truth."
>
> **Dev:** "How does an agent get context?"
>
> **Domain expert:** "It asks the **Security-Aware Retrieval Coordinator** for a **Context Package** inside an **Active Memory Pool**."

## Flagged Ambiguities

- "Signal" and "Memory" were used interchangeably. A **Signal** is the classified processing unit; a **Memory Object** is institutional meaning with provenance, time, confidence, precision, and lifecycle state.
- "Context" was used to mean database row, prompt material, search result, and agent working memory. Use **Context Package** for AI-ready task context and **Memory Object** for durable memory.
- "File" was treated like truth. Files in node folders should be **Workspace Exports** derived from the **Enterprise Memory Core**.
- "Skill" was used for prompts, tools, procedures, and agent abilities. Use **Skill Package** only for governed procedural knowledge with policy, validation, provenance, and audit.
- "Postgres" was used as shorthand for the architecture. The canonical term is **Enterprise Memory Core**; Postgres is the likely first implementation substrate.
