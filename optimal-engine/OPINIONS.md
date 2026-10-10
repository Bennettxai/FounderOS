# Optimal Engine Opinions

Read this when changing engine setup, APIs, stores, memory behavior, CLI behavior, or documentation.
These opinions apply in addition to `~/OPINIONS.md`.

## Memory And Stores

Optimal Engine owns knowledge and memory.
BusinessOS and other apps own product state.
Do not collapse app state into memory.
Do not rebuild memory inside app code.
SQLite is the durable local runtime store today.
RocksDB is the default persistent local graph backend when available.
FTS, vectors, chunks, caches, graph projections, and RAG outputs are projections or retrieval surfaces.
They are not the same thing as raw evidence or final truth.

## Agent Interface

For engine development, direct `mix optimal.*` commands are acceptable.
For normal local agent work, use `bin/optimal`.
The wrapper is the agent-facing interface because it handles live-server reads, root paths, workspace scope, and memory commands better than raw task calls.
The engine should keep making that wrapper easier to use, not force agents into low-level internals.

## Isolation

Every API path and write path must preserve tenant, organization, workspace, Node, and policy scope.
Downloaded users must use their own local engine data.
Roberto's private engine data must never leak into public repos, packaged apps, seed data, fixtures, or docs.

## Setup

Fresh clone setup matters.
Docs should explain required stores and startup commands without buzzwords.
A user should know what SQLite, RocksDB, ETS, Mnesia, FTS, graph, and RAG actually do in the stack.
If setup requires a local dependency, document it and provide a fallback.
