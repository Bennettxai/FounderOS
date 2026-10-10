# Founder OS

**A personal operating system for a single person company: a command center
that runs your business as a set of AI-assisted "departments."**

Built for whoever is holding the whole thing together: a freelancer, a
solopreneur, a solo founder, or a small team where one person still runs point.
You set the direction, AI agents cover the departments, and any employees you do
have plug into the same board.

Founder OS turns the tabs, tools and mental overhead of running that business
into one screen: unified comms, a client funnel, social growth, finances, a
knowledge graph, and a roster of named AI agents that each own a real job.

> Want to build your own, live, with guidance? That's what the cohort is for:
> [founderos.sh](https://founderos.sh)

---

## v2: Founder OS on BusinessOS

Founder OS v1 was a Next.js + SQLite app. v2 keeps the same operator UI and runs
it on the [BusinessOS](https://github.com/Miosa-osa/BusinessOS) stack:

| Layer | v1 | v2 |
| --- | --- | --- |
| Frontend | Next.js 14 (React) | SvelteKit 2 (Svelte 5) |
| Backend | Next.js API routes | Go (Gin) |
| Operational data | SQLite (better-sqlite3) | PostgreSQL + Redis |
| Knowledge / memory | G-Brain | [Optimal Engine](https://github.com/Miosa-osa/OptimalEngine) (Elixir/OTP), bundled |
| Desktop | — | Electron shell (optional) |

The operator views live under `/os`. Everything else BusinessOS provides
(workspaces, roles, teams, the module desktop, the terminal) comes with it.
The last v1 release is tagged `v1-nextjs`.

---

## Quick start

Prerequisites (macOS shown):

```bash
brew install node@22 go postgresql@17 pgvector redis elixir python
corepack enable && corepack prepare pnpm@9.15.4 --activate
```

The schema needs the `vector` extension. Homebrew's `pgvector` bottle builds for
its current Postgres releases (17/18), so pair it with `postgresql@17`; on another
Postgres, build pgvector for it, or point the launcher at a matching install
with `BUSINESSOS_PG_BIN=/path/to/pg/bin`.

Then:

```bash
BUSINESSOS_HEADLESS=1 make dev-local   # Postgres, Redis, Optimal Engine, backend, frontend
                                       # (drop BUSINESSOS_HEADLESS=1 to also open the desktop app)
open http://localhost:5273/register    # create your local account
make demo OWNER=you@example.com        # Founder OS workspaces + the demo data
open http://localhost:5273/os          # the operator console
```

`make demo` creates four workspaces (HQ, Vantage, Launchpad Cohort, Personal),
loads the demo's seeded data into Postgres and its knowledge into the Optimal
Engine. It is idempotent; re-running it refreshes the demo and moves its dates
to today. No credentials are needed to browse.

Ports live in `.env.dev`; `make dev-local-status` shows what is running and
`make dev-local-stop` stops it. See [docs/BUSINESSOS.md](docs/BUSINESSOS.md) for
the platform's own setup notes.

---

## What you're looking at

| Route | What it is |
| --- | --- |
| `/os` | Operator console: pulse row, what needs you, operating volume |
| `/os/comms` | Unified inbox: email, Slack, WhatsApp, calendar and recorder lanes |
| `/os/funnel` | Living client-journey flow and acquisition wheel |
| `/os/workflows` | Multi-step tool workflows, as a tree and a builder |
| `/os/social`, `/os/content` | Growth dashboard; content pipeline, calendar and lead magnets |
| `/os/brand-deals`, `/os/finances`, `/os/trading`, `/os/adpilot` | Deals, money, brokerage monitor, paid-media planner |
| `/os/agents`, `/os/chats`, `/os/tasks`, `/os/skills` | The agent roster, chat hub, task board and skills |
| `/os/org`, `/os/blueprint` | Org hierarchy (operator → Conductor → pillars → workers) and the system blueprint |
| `/os/brain`, `/os/doctor` | The knowledge core on the Optimal Engine, and its health |
| `/os/integrations`, `/os/usage`, `/os/analytics` | Connections board, token usage, connector numbers |
| `/os/roadmap`, `/os/reference`, `/os/personas` | Phases and quarters, the reference model, persona templates |

Navigate with the sidebar or the command palette (Cmd/Ctrl + K; digits 1–9 jump
to the first nine views).

---

## Architecture: demo-first, real-ready

The demo looks alive because of seeded data, but every view reads through the
same path a live install uses, and every connector reports its honest state
(`connected`, `not_configured` or `error`), never a fake green light.

```
frontend/src/routes/(founderos)/os/      SvelteKit routes, one per view
frontend/src/lib/founderos/              views, kit (slab, charts, motion), chrome
desktop/backend-go/internal/founderos/
  api/          GET /api/founderos/pages/<view> and actions
  pages/        page builders (read Postgres + connectors + the engine)
  connectors/   24 integrations, honest status, writes behind FOUNDEROS_WRITES
  agents/       the agent registry; every seeded agent has a real run()
  memory/       retrieval over the Optimal Engine
  seed/, etl/   the demo data and the loader behind `make demo`
config/founderos/engine-topology.yaml    which engine each workspace's memory lives on
optimal-engine/                          the bundled Elixir engine
```

Data lives in `founderos_*` Postgres tables, scoped by workspace. Knowledge
lives in the Optimal Engine: agents write sources, signals and pending claims;
facts are promotion-gated, so memory stays trustworthy as more agents write to it.

**Safety switches.** `FOUNDEROS_WRITES=1` allows outbound writes (sends, posts,
payments); `FOUNDEROS_CRONS=1` allows scheduled agent runs. Both are off unless
you turn them on.

---

## Credentials

Connectors read keys from the process environment, `~/.founderos/.env`, or keys
planted through the Connections board. Nothing else on your machine is read.
Never commit keys. `.env*` files are gitignored.

---

## Tests

```bash
make test                                   # Go + frontend suites
cd frontend && npx vitest run src/lib/founderos
cd desktop/backend-go && go test ./internal/founderos/...
```

Go tests create throwaway databases on this checkout's dev Postgres (the
`POSTGRES_PORT` in `.env.dev`); set `FOUNDEROS_PG_ADMIN_URL` to use another.
Write the failing test first.

---

## Credits

Founder OS v2 is built on [BusinessOS](https://github.com/Miosa-osa/BusinessOS)
and [Optimal Engine](https://github.com/Miosa-osa/OptimalEngine) by Roberto H.
Luna and the Miosa team. If you use this stack, star their repos.

## License

MIT, see [LICENSE](LICENSE).
