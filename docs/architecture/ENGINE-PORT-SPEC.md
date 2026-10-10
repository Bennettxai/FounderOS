# OptimalOS Engine → Go Port Architecture

## Source: Elixir Engine (47 modules, ~11,900 LOC)
## Target: Go package at internal/optimal/

## Package Structure

```
internal/optimal/
├── store/          ← SQLite persistence + in-memory cache (replaces Store GenServer)
│   ├── store.go        Context CRUD, ETS→sync.Map cache
│   ├── schema.go       Table creation, migrations
│   └── fts.go          FTS5 search queries
│
├── signal/         ← Signal Theory classifier (replaces Classifier + Bridge.Signal)
│   ├── classifier.go   S=(M,G,T,F,W) classification
│   ├── dimensions.go   Mode, Genre, Type, Format, Structure enums
│   └── quality.go      S/N ratio scoring
│
├── ingest/         ← Classify → route → write → index pipeline (replaces Intake)
│   ├── ingest.go       Main pipeline
│   ├── router.go       Topology-based routing (replaces Router GenServer)
│   ├── writer.go       YAML frontmatter + genre skeleton writer
│   └── skeleton.go     Genre templates (transcript, brief, spec, etc.)
│
├── search/         ← Hybrid search (replaces SearchEngine)
│   ├── search.go       BM25 + temporal decay + S/N boost
│   ├── vector.go       Cosine similarity search
│   └── intent.go       Query intent analysis
│
├── graph/          ← Knowledge graph (replaces Graph + GraphAnalyzer)
│   ├── graph.go        Edge management (existing, enhanced)
│   ├── analyzer.go     Triangles, clusters, hubs
│   ├── reflector.go    Co-occurrence gap detection
│   └── entities.go     Entity extraction + deduplication
│
├── tiered/         ← L0/L1/L2 loading (replaces ContextAssembler + L0Cache)
│   ├── tiered.go       LoadTiered(tier) dispatcher
│   ├── l0.go           Structural inventory cache
│   ├── l1.go           Summary extraction
│   └── assembler.go    Multi-tier assembly
│
├── simulate/       ← Scenario planning (replaces Simulator + MonteCarlo + MCTS)
│   ├── simulator.go    "What if" mutation tracing
│   ├── montecarlo.go   Probability sampling (1000 sims)
│   └── mcts.go         Monte Carlo Tree Search (UCB1)
│
├── learn/          ← Knowledge evolution (replaces RememberLoop + RethinkEngine)
│   ├── remember.go     Observation capture (explicit, contextual, mining)
│   ├── rethink.go      Evidence synthesis
│   ├── reweave.go      Stale context detection
│   └── observations.go Observation storage
│
├── health/         ← Diagnostics (replaces HealthDiagnostics + VerifyEngine)
│   ├── health.go       10-check diagnostic suite
│   └── verify.go       L0 fidelity testing
│
├── session/        ← Session management (replaces Session GenServer)
│   ├── session.go      Session lifecycle
│   └── compress.go     Transcript compression
│
├── spec/           ← Spec verification (replaces Spec.* modules)
│   ├── parser.go       Parse .spec.md files
│   ├── verifier.go     Requirement verification
│   └── drift.go        Git-based drift detection
│
├── reader.go       ← EXISTING: filesystem reads
├── graph.go        ← EXISTING: SQLite graph queries (move to graph/)
├── engine.go       ← UPDATED: pure Go, no subprocess
└── ingest.go       ← BASIC: from port agent (enhance with full pipeline)
```

## SQLite Schema (same as Elixir, ported to Go migrations)

Tables: contexts, contexts_fts, entities, edges, decisions, sessions, vectors, observations

## Key Algorithms to Port

1. Signal Theory Classifier — regex-based S=(M,G,T,F,W)
2. Hybrid Search — BM25 * temporal_decay * sn_boost
3. Graph Analysis — triangles (open triads), clusters (BFS), hubs (2σ degree)
4. MCTS — UCB1 tree exploration, 32 iterations
5. Monte Carlo — 1000 simulations, edge probability flipping
6. Simulator — mutation tracing through weighted graph edges
7. RememberLoop — 3-mode observation capture, category escalation
8. RethinkEngine — evidence synthesis at confidence >= 1.5
9. Reweaver — staleness scoring with temporal decay
10. Health Diagnostics — 10 checks (orphans, stale, duplicates, etc.)

## Priority Order

Phase 1 (DONE): reader.go, graph.go, basic engine.go
Phase 2 (NOW): ingest, search, tiered, health
Phase 3: signal classifier, graph analyzer, learn
Phase 4: simulate, mcts, montecarlo
Phase 5: session, spec, vector search
