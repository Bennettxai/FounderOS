// Package tech is the TECH department of the FounderOS agent roster on the
// bridge: the Go port of data-agent, markdown-auditor, vector-auditor and
// stack-monitor from FounderOS v1 lib/agents/real.ts.
//
// In FounderOS v1 the first three were bound to G-Brain (its CLI overview,
// stats, doctor and search, the brain-store folder on disk, lib/brain-audit.ts).
// G-Brain is retired on the bridge: every workspace's memory lives in its home
// Optimal Engine (config/founderos/engine-topology.yaml). Each agent keeps its
// purpose and is re-expressed against the engines:
//
//   - data-agent. FounderOS v1: G-Brain overview (store folders, doctor checks)
//     turned into "ideas", and broadcasts answered with a G-Brain search.
//     Bridge: for every staged engine it reads GET /api/health, /api/stores
//     (table counts: contexts, vectors, claims, facts), /api/stores/audit and
//     /api/workspaces, and turns them into ideas: an unreachable engine, failing
//     audit checks, a topology workspace never created on its home engine,
//     contexts without an embedding, and claims never promoted to facts. It
//     implements agents.Responder, so a broadcast becomes Deps.Memory.Search
//     over every workspace whose home engine is staged, answered with the top
//     three hits as "title: abstract". A failed search reads as failed, never
//     as "nothing matches". OK means every staged engine answered and is up.
//
//   - markdown-auditor. FounderOS v1: walked the brain-store markdown for broken
//     wikilinks, orphans, duplicate titles and untitled pages, and compared the
//     folder with the index search reads. Bridge: exports each topology
//     workspace's sources from its home engine (GET
//     /api/batch/export/signals) and runs the same checks over them: broken
//     [[wikilinks]] (code is not prose), orphans, duplicate titles, sources
//     titled only by a file name. It adds checks the engines need: stale
//     workspaces (nothing new in 30 days), empty workspaces, and workspaces the
//     topology routes to an engine that does not have them (an error). The
//     store-vs-index drift check becomes the engine audit's fts_parity check
//     (an error when the full-text index no longer matches the contexts);
//     workspace_scope flags rows left in the retired default workspace. Links
//     resolve by name, date-stripped name, title and slugified title, because
//     the engine stores imported pages as <date>-<name>.md and several sources
//     can share one URI. OK means no error finding and at least one source.
//     FounderOS v1's "links not ingested" check (index Links: 0) is not ported:
//     the engine exposes no per-workspace link count to compare against.
//
//   - vector-auditor. FounderOS v1: the G-Brain doctor (Supabase pgvector
//     connection, embedding checks, health score). Bridge: per staged engine,
//     the storage audit (GET /api/stores/audit, read even when it answers 503)
//     scored as passing checks out of all checks, embedding coverage (vectors
//     vs contexts, chunk vectors), and the claims pipeline (claims vs facts
//     from /api/stores, pending claims per workspace from
//     /api/memory-core/claims). A failing sqlite_integrity, migrations,
//     logical_stores, fts_parity or vector_integrity check fails the run;
//     other failing checks (backups, fixtures) are listed as warnings.
//
//   - stack-monitor. FounderOS v1 probed the local stack (Paperclip, Hermes, the
//     CLIs the OS shells out to) and read Wispr Flow's sqlite on the machine it
//     ran on. The bridge never reads a device port or path: founderos-collector
//     on each Mac runs those same checks and pushes them, and this agent reads
//     the pushed local-stack and wispr rows through Deps.Devices (freshest
//     healthy reading across Macs; a stale push reads as an error), plus each
//     expected Mac's push freshness.
//
// Engines are resolved like api.TopologyEngines (which this package may not
// import): the checked-in topology, each engine's url_env/key_env, else the
// local staging engine when ~/.founderos-bridge/keys/oe-<name>.key exists.
// Engines that are not staged here are named in every summary together with
// the workspaces nobody could see, never counted as empty. Every engine call
// is a GET; no agent in this package writes or calls an LLM.
package tech
