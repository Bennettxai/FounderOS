# FounderOS

## Your AI-powered command center for a one-person company

FounderOS brings the moving parts of a small business into one place: clients,
comms, social growth, finances, knowledge, and a roster of AI-assisted agents.
You set the direction; your departments and team work from the same operating
picture.

## Tech stack

FounderOS is a single Node.js app. There is no separate backend, database
server, or Docker setup to run.

| Layer | What it uses |
| --- | --- |
| Runtime | Node.js 22 LTS (what CI runs), npm |
| Framework | Next.js 14 (App Router), React 18, TypeScript 5 |
| Styling | Tailwind CSS 3, PostCSS, Geist and Geist Mono (bundled locally), lucide-react, simple-icons |
| Storage | SQLite through `better-sqlite3`, kept in local files under `data/` (your workspace) and `demo-data/` (the demo company) |
| Validation | zod |
| Visuals | d3-force for the knowledge graph |
| AI | Vercel AI Gateway (`AI_GATEWAY_API_KEY`) for agent chat; `LLM_PROVIDER=stub` runs offline |
| Integrations | IMAP/SMTP (imapflow, nodemailer), Slack, Notion, Stripe, iCal, plus HTTP connectors for the rest |
| Tests and tooling | Vitest, `tsc --noEmit`, tsx for scripts, GitHub Actions CI (typecheck and tests on every push) |

### What you need to run and build it

**Required**

- **Node.js 22 LTS** and **npm** (bundled with Node). CI runs on Node 22.
  Newer releases can crash `better-sqlite3` inside the test runner (seen on
  Node 24.20 as `Assertion failed: (env) != nullptr`), so use 22 if `npm test`
  fails that way.
- **Git**, to clone the repo.
- **A C/C++ toolchain**, only if `better-sqlite3` has no prebuilt binary for
  your platform and has to compile during `npm ci`:
  - macOS: `xcode-select --install`
  - Debian/Ubuntu: `sudo apt install build-essential python3`
  - Windows: Visual Studio Build Tools with the "Desktop development with C++"
    workload

That is all the demo needs. No API keys, accounts, or database server.

**Optional**

- **Vercel AI Gateway key** (`AI_GATEWAY_API_KEY`), to talk to agents.
- **Credentials for the tools you connect** (email, Slack, Notion, Stripe, and
  others). See [Connect your tools](#connect-your-tools) and
  [`.env.example`](.env.example).
- **Bun and the G-Brain CLI**, for local knowledge search. See
  [Optional knowledge search](#optional-knowledge-search).

## Getting started

### Prerequisites

- Node.js 22 LTS
- npm

You do not need API keys or external accounts to try the demo. See
[What you need to run and build it](#what-you-need-to-run-and-build-it) for the
full list.

### Clone the repo

```bash
git clone https://github.com/Bennettxai/FounderOS.git
cd FounderOS
```

### Install dependencies

```bash
npm ci
```

### Start FounderOS

```bash
npm run dev
```

Open [http://localhost:4100](http://localhost:4100). On first start, FounderOS
opens in demo mode with a fictional company, sample departments, agents, tasks,
brand deals, social accounts, campaigns, and knowledge. Demo content is built
under `demo-data/`; it never writes into your `data/` directory.

## What you can do

- **Run your business:** use the command center, client funnel, inbox, finances,
  and department workspaces to see what needs attention.
- **Work with agents:** review the agent roster, task lanes, run history, and
  conversations from the Agents board.
- **Grow your audience:** manage social accounts, campaigns, brand deals, and
  content workflows.
- **Keep company context close:** search your knowledge base and connect the
  tools your business already uses.

FounderOS is useful without integrations. Unconfigured connectors stay clearly
marked until you add credentials; the app does not pretend they are connected.

## Connect your tools

Open **Integrations** to configure the services you want to use, such as email,
Slack, Notion, Attio, WhatsApp, Stripe, PayPal, Beehiiv, or Zernio. Keep keys in
your local `.env.local` file; see [`.env.example`](.env.example) for supported
environment variables. Never commit credentials.

## Use your own business

In **Settings**, turn **Filming mode** off to leave the fictional demo company
and use your own workspace. Your `data/` directory starts blank. To have a fresh
install start in this mode, set:

```bash
GLADOS_DEFAULT_MODE=real
```

## Keep it local or set a password

On first run, FounderOS has no login and binds to `127.0.0.1`, so it is only
reachable from your computer. Before accessing it from another device, open
**Settings → Security** and set an operator password. FounderOS will then
require that password for app pages and API routes.

## Optional knowledge search

FounderOS can search a local markdown knowledge base with the G-Brain CLI:

```bash
bun install -g github:garrytan/gbrain
gbrain init --pglite --no-embedding
gbrain import /path/to/your/notes --no-embed
```

Set `GBRAIN_BIN` and `GBRAIN_STORE` in `.env.local`. The embedded store is
single-writer: use the CLI or a long-running `gbrain serve`, not both at once.

## Development

```bash
npm test
npm run typecheck
npm run build
```

Use `npm run preview` to build and serve a production-like review version. It
uses a separate `.next-preview` build directory, leaving the development
server's `.next` output alone.

## License

MIT. See [LICENSE](LICENSE).
