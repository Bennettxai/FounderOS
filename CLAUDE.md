# FOUNDER OS (v2, on BusinessOS)

Personal OS / AI agent command center for a single person company. The Founder
OS operator views run on the BusinessOS stack: Go backend, SvelteKit 2 frontend,
PostgreSQL + Redis, and the bundled Elixir Optimal Engine as the only memory
layer. Platform-wide rules live in `AGENTS.md` (BusinessOS); this file covers
what is Founder OS specific.

## Commands

```bash
BUSINESSOS_HEADLESS=1 make dev-local   # whole stack; ports in .env.dev
make demo OWNER=<registered email>     # workspaces + demo data (idempotent)
make dev-local-status | dev-local-stop
cd frontend && npx vitest run src/lib/founderos "src/routes/(founderos)"
cd frontend && npx svelte-check --threshold error
cd desktop/backend-go && go test ./internal/founderos/... ./cmd/...
```

## Where Founder OS lives

- Views: `frontend/src/routes/(founderos)/os/<view>/+page.svelte`, components and
  logic in `frontend/src/lib/founderos/pages/<view>/`. Shared kit (Slab, SlabCard,
  BigStat, MeterStack, motion) in `lib/founderos/kit`, chrome (Sidebar, Topbar,
  CommandPalette, Conductor, cohort invite) in `lib/founderos/chrome`.
- Navigation: `lib/founderos/nav.ts` is the single source (sidebar, palette digits,
  module catalog, page-existence test). v1's order; G-Brain is "Brain".
- Every view fetches `founderosFetch('/pages/<view>')` → Go
  `GET /api/founderos/pages/<view>` (`internal/founderos/api/page_<view>.go`,
  builders in `internal/founderos/pages/<view>/`).
- Data: `founderos_*` tables (migrations `internal/database/migrations/16x_*`),
  loaded from `internal/founderos/seed/demo/*.sql` by `cmd/founderos-seed` (ETL in
  `internal/founderos/etl`). Relative dates (funnel, agent runs) move to today on
  every seed.
- Memory: the Optimal Engine only, via `internal/founderos/memory` and
  `config/founderos/engine-topology.yaml` (one `local` engine, four workspaces:
  founderos=HQ, vantage, launchpad-cohort, personal). Do not reintroduce G-Brain.

## Rules

- **Demo-first, real-ready.** New data = migration + ETL spec + seed fixture +
  page builder + test. Pages never query around the builders.
- **Honest states.** A connector is `connected`, `not_configured` or `error`,
  never faked. An unconfigured connector answers 200 with a not-configured state,
  never a 5xx or a console error.
- **Credentials** come from the process env, `~/.founderos/.env`, or planted keys.
  Never read other tools' credential files; never commit keys.
- **Writes and crons** are off unless `FOUNDEROS_WRITES=1` / `FOUNDEROS_CRONS=1`.
- **TDD**: failing test first (vitest next to the component, `go test` next to the
  Go code). Keep `go test`, vitest and `svelte-check` (0 errors) green.
- **The UI is v1's.** Founder OS v1 (tag `v1-nextjs`) is the reference for layout,
  copy and behaviour of every view. The demo operator is Alex; the ventures are
  Vantage (#00ffaa) and Launchpad Cohort (#d9263f). No real personal data, ever.
- **Theme**: Monolith Signal is the default (black, white accent, color means
  status only); JetBrains Mono everywhere; square corners, hairline borders.

## Multi-agent etiquette

- Commit small checkpoints often; run the gates before claiming done.
- Don't stop another session's dev stack; use your own ports via `.env.dev`.
- The dev launcher ignores `OPTIMAL_ENGINE_*` from your shell profile; point it at
  other engine data only with `FOUNDEROS_ENGINE_ROOT`.
