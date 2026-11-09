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
| Runtime | [Node.js](https://nodejs.org/en/download) 22 LTS (what CI runs), [npm](https://docs.npmjs.com) |
| Framework | [Next.js 14](https://nextjs.org/docs/14) (App Router), [React 18](https://react.dev), [TypeScript 5](https://www.typescriptlang.org) |
| Styling | [Tailwind CSS 3](https://v3.tailwindcss.com), [PostCSS](https://postcss.org), [Geist and Geist Mono](https://vercel.com/font) (bundled locally), [lucide-react](https://lucide.dev), [simple-icons](https://simpleicons.org) |
| Storage | [SQLite](https://sqlite.org) through [`better-sqlite3`](https://github.com/WiseLibs/better-sqlite3), kept in local files under `data/` (your workspace) and `demo-data/` (the demo company) |
| Validation | [zod](https://zod.dev) |
| Visuals | [d3-force](https://d3js.org/d3-force) for the knowledge graph |
| AI | [Vercel AI Gateway](https://vercel.com/docs/ai-gateway) (`AI_GATEWAY_API_KEY`) for agent chat; `LLM_PROVIDER=stub` runs offline |
| Integrations | IMAP/SMTP ([imapflow](https://github.com/postalsys/imapflow), [nodemailer](https://nodemailer.com)), [Slack](https://docs.slack.dev), [Notion](https://developers.notion.com), [Stripe](https://docs.stripe.com), iCal ([node-ical](https://github.com/jens-maus/node-ical)), plus HTTP connectors for the rest |
| Tests and tooling | [Vitest](https://vitest.dev), [`tsc --noEmit`](https://www.typescriptlang.org/docs/handbook/compiler-options.html), [tsx](https://tsx.is) for scripts, [GitHub Actions](https://docs.github.com/actions) CI (typecheck and tests on every push) |

### Tools it works with

These are the services FounderOS is built around. Every one is optional: connect
it from **Integrations**, and until you do it shows as not configured.

| Area | Tools |
| --- | --- |
| AI and agents | [Vercel AI Gateway](https://vercel.com/docs/ai-gateway) (agent and Wizard chat, [Claude](https://claude.com) models by default), [Superset, now Paradev](https://paradev.com) (opens a coding or writing agent in its own workspace) |
| Memory | [Optimal Engine](https://github.com/Miosa-osa/OptimalEngine) through the [G-Brain CLI](https://github.com/garrytan/gbrain) (searchable knowledge base), [Obsidian](https://obsidian.md) |
| Inbox and calendar | Any IMAP/SMTP mailbox ([Gmail app passwords](https://support.google.com/accounts/answer/185833) work), [Google Calendar over CalDAV](https://developers.google.com/workspace/calendar/caldav/v2/guide), [Slack](https://slack.com), [WhatsApp](https://www.whatsapp.com) |
| Clients | [Notion](https://www.notion.com) |
| Money | [Stripe](https://stripe.com), [PayPal](https://www.paypal.com), [FanBasis, now Commas](https://commas.com), [Wise](https://wise.com) |
| Audience and growth | [Zernio](https://zernio.com) (social accounts and posting), [Beehiiv](https://www.beehiiv.com) (newsletter), [ManyChat](https://manychat.com) (Instagram DMs), [Meta Ads](https://www.facebook.com/business/ads) |
| Calls and voice | [Fathom](https://www.fathom.ai) (call notes), [Plaud](https://www.plaud.ai) (voice recorder), [Wispr Flow](https://wisprflow.ai) (dictation) |

WhatsApp and Wispr Flow read the apps' local data, so they need macOS.

### What you need to run and build it

**Required**

- **[Node.js 22 LTS](https://nodejs.org/en/download)** and **[npm](https://docs.npmjs.com)** (bundled with Node). CI runs on Node 22.
  Newer releases can crash `better-sqlite3` inside the test runner (seen on
  Node 24.20 as `Assertion failed: (env) != nullptr`), so use 22 if `npm test`
  fails that way.
- **[Git](https://git-scm.com)**, to clone the repo.
- **A C/C++ toolchain**, only if `better-sqlite3` has no prebuilt binary for
  your platform and has to compile during `npm ci` (see
  [node-gyp's setup notes](https://github.com/nodejs/node-gyp#installation)):
  - macOS: `xcode-select --install` ([Xcode Command Line Tools](https://developer.apple.com/xcode/resources/))
  - Debian/Ubuntu: `sudo apt install build-essential python3`
  - Windows: [Visual Studio Build Tools](https://visualstudio.microsoft.com/downloads/) with the "Desktop development with C++"
    workload

That is all the demo needs. No API keys, accounts, or database server.

**Optional**

- **[Vercel AI Gateway](https://vercel.com/docs/ai-gateway) key** (`AI_GATEWAY_API_KEY`), to talk to agents.
- **Credentials for the tools you connect** (email, Slack, Notion, Stripe, and
  others). See [Connect your tools](#connect-your-tools) and
  [`.env.example`](.env.example).
- **[Bun](https://bun.sh) and the [G-Brain CLI](https://github.com/garrytan/gbrain)**, for local knowledge search. See
  [Optional knowledge search](#optional-knowledge-search).

## Getting started

### Prerequisites

- [Node.js 22 LTS](https://nodejs.org/en/download)
- [npm](https://docs.npmjs.com)

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
[Slack](https://slack.com), [Notion](https://www.notion.com),
[WhatsApp](https://www.whatsapp.com), [Stripe](https://stripe.com),
[PayPal](https://www.paypal.com), [Beehiiv](https://www.beehiiv.com), or
[Zernio](https://zernio.com). Keep keys in
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

FounderOS can search a local markdown knowledge base with the
[G-Brain CLI](https://github.com/garrytan/gbrain). It installs with
[Bun](https://bun.sh):

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
