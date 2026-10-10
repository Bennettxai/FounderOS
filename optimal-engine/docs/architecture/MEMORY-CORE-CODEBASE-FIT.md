# Memory Core Codebase Fit

This note defines how the governed-memory work should fit into the current
Optimal Engine codebase.

The important naming correction is:

```text
Optimal Engine = the product / whole engine
Memory Core = the governed memory subsystem inside Optimal Engine
Store = low-level persistence adapter
Context = compatibility/search row
Workspace = operating area
Node = workspace subdivision
```

The code namespace is `OptimalEngine.MemoryCore`. New public-facing docs and UI
should say **Memory Core** when referring to this subsystem. Use database-system
category language only in research notes or implementation discussions where it
adds precision.

## Current Read

The new governed-memory names are useful, but the surrounding codebase still
has older context-engine names carrying too much meaning:

- `OptimalEngine.Context` is a compatibility storage/search row.
- `OptimalEngine.Store` owns too many database responsibilities.
- `OptimalEngine.Retrieval.*` returns ranked contexts, not governed retrieval
  packages yet.
- `OptimalEngine.Memory.*`, `OptimalEngine.Bridge.*`, and
  `OptimalEngine.MemoryCore.*` still use overlapping memory language until the
  older modules are intentionally folded into the new subsystem.
- The governed object tables exist, but only Source Package and Derivation
  Ledger writes are currently implemented.

This is fixable. The next work needs strict ownership so Memory Core does not
become another parallel bucket of generic logic.

## Canonical Flow

New code should follow this flow:

```text
command or raw input
  -> Scope Envelope
       actor
       workspace
       node hints
       operation class
       permissions
  -> Source Package
  -> Signal
  -> Route / Node assignment
  -> compatibility Context row
  -> Claim
  -> Fact
  -> Memory Object / Episode Object / Memory Detail Object
  -> Relationship Edges
  -> Retrieval Package / Context Package
  -> Active Memory Pool
  -> Workflow Trace
  -> Generalized Workflow
  -> Procedural Memory Object
  -> Skill Package
```

`Context` is not durable truth. It is a compatibility/search projection until
the Retrieval Package and Context Package services are built.

If scope is unknown, the engine still stores the Source Package and routes the
derived Signal to inbox/quarantine or a pending topology decision. Missing Node
scope is a routing problem, not a reason to lose evidence.

## Naming Policy

Use these names for new code and docs:

| New code should say | Meaning |
| --- | --- |
| `SourcePackage` | Preserved source evidence. |
| `Signal` | First classified processing unit using Signal Theory dimensions. |
| `Claim` | Source-backed assertion not yet accepted as true. |
| `Fact` | Accepted assertion with evidence, scope, confidence, precision, and time. |
| `MemoryObject` | Institutional meaning around sources, claims, facts, edges, and time. |
| `EpisodeObject` | Governed event memory. |
| `MemoryDetailObject` | Reusable step, command, parameter, validation, or exception. |
| `RelationshipEdge` | Typed graph link between governed objects. |
| `RetrievalPackage` | Structured retrieval result with evidence and policy state. |
| `ContextPackage` | AI-usable projection for an actor, task, time mode, and policy scope. |
| `ActiveMemoryPool` | Task-scoped working memory. |
| `SkillPackage` | Governed procedural package, not a loose prompt or tool. |

Keep these names contained:

| Compatibility name | Where it may remain | Do not use it for |
| --- | --- | --- |
| `Context` | Existing storage/search rows and retrieval compatibility APIs. | Durable memory truth. |
| `Store` | SQLite connection, compatibility storage, low-level adapter functions. | Business meaning or lifecycle logic. |
| `Memory` | Existing legacy memory/session/cortex modules until replaced. | New governed Memory Core objects. |
| `Bridge` | Temporary adapter modules between old subsystems. | New canonical domain modules. |
| `MemoryCore` | Governed memory subsystem inside Optimal Engine. | Broad product identity. |

## Current Module Ownership

| Module area | Current role | Architecture fit |
| --- | --- | --- |
| `OptimalEngine.Pipeline.Intake` | Raw text intake, classification, routing, file write, indexing, provenance trace. | Correct first integration point, but it should delegate lifecycle logic. |
| `OptimalEngine.MemoryCore.SourcePackage` | Builds source evidence objects. | Correct domain object. |
| `OptimalEngine.MemoryCore.SourcePackageService` | Records the first source-to-signal lifecycle operation. | Correct lifecycle owner for the initial slice. |
| `OptimalEngine.MemoryCore.DerivationLedgerEntry` | Builds derivation records. | Correct domain object. |
| `OptimalEngine.MemoryCore.Store` | Typed persistence adapter through `OptimalEngine.Store`. | Should stay storage-only. |
| `OptimalEngine.Store` | SQLite connection, migrations, compatibility context CRUD, FTS, vectors, cache. | Infrastructure, not business lifecycle. |
| `OptimalEngine.Retrieval.Search` | Hybrid search over compatibility Context rows. | Useful mechanism, not the governed retrieval coordinator. |
| `OptimalEngine.Retrieval.ContextAssembler` | Builds tiered context from search results. | Precursor, not the governed Context Package assembler. |
| `OptimalEngine.Memory.*` | Legacy/session/agent memory behavior. | Keep until intentionally folded into Memory Core concepts. |
| `OptimalEngine.Bridge.*` | Adapters between Signal/Memory/Knowledge subsystems. | Transitional; avoid expanding unless adapting old code. |

## Cleanup Rules

1. Do not add new Memory Core lifecycle behavior to `OptimalEngine.Store`.
2. Do not add new governed memory objects under `OptimalEngine.Memory`.
3. Do not return `%OptimalEngine.Context{}` from new Memory Core APIs unless the
   function is explicitly a compatibility adapter.
4. Avoid generic `Manager`, `Service`, or `Engine` modules when a domain verb is
   available. Prefer `FactPromoter`, `ClaimExtractor`, `RetrievalCoordinator`.
5. Keep `SourcePackage -> Signal -> Context` as the compatibility ingestion path
   until Claim and Fact services exist.
6. Treat `Bridge` modules as temporary adapters.
7. Any new retrieval API must say whether it returns raw search hits,
   `RetrievalPackage`, or `ContextPackage`.

## First Refactor Completed

This operation was split:

```elixir
OptimalEngine.MemoryCore.Store.record_ingested_signal/4
```

into:

```elixir
OptimalEngine.MemoryCore.SourcePackageService.record_ingested_signal/4
```

`MemoryCore.Store` now owns persistence only. `SourcePackageService` owns the
lifecycle operation:

```text
Source Package -> Derivation Ledger -> Signal/Context
```

## Next Safe Refactors

1. Add `Claim` and `Fact` structs plus typed store functions.
2. Add `ClaimExtractor.extract_from_source/2`.
3. Add `FactPromoter.promote/2`.
4. Add `ScoringPolicy` for confidence and precision.
5. Add the first `RetrievalPackage` struct before changing the existing RAG API.
6. Update public/internal docs to use Memory Core as the subsystem name.

## Product Naming Rule

Use this language in user-facing or public-facing material:

```text
Optimal Engine is an operating engine for workspaces, nodes, humans, and AI
agents. Its Memory Core preserves source evidence, derives claims and facts,
builds governed context packages, learns workflows, and packages validated
procedures as skills.
```

Avoid saying in product-facing material:

```text
Optimal Engine is only a storage product.
```

Storage-category language may be useful in private implementation discussion,
but product language should make Optimal Engine feel like the whole system and
Memory Core feel like one major subsystem inside it.
