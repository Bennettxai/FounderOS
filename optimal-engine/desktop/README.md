# Optimal Engine — Desktop

Tauri + SvelteKit shell that talks to the engine's HTTP API.

## Prerequisites

- Node 20+
- Rust stable (for the Tauri bundle)
- The engine running on `127.0.0.1:4200`

## Boot the engine

In `config/dev.exs` or `config/runtime.exs` for the parent repo:

```elixir
config :optimal_engine, :api, enabled: true, port: 4200
```

Then `mix phx.server` (or `iex -S mix`) from the repo root.

## Run the desktop in dev

```bash
cd desktop
npm install
npm run tauri:dev   # opens the native window + hot-reloads the Svelte UI
# — or —
npm run dev          # browser-only preview at http://localhost:1420
```

## Build a signed bundle

```bash
npm run tauri:build
```

Artifacts land in `src-tauri/target/release/bundle/`.

## What's in the shell

- `/`             — Ask: query box that hits `POST /api/rag`
- `/wiki`         — Wiki: page list + render (`/api/wiki/:slug`)
- `/status`       — Engine readiness (`/api/status`)

Adding a page: drop a `+page.svelte` under `src/routes/`. The API
client lives in `src/lib/api.ts`.
