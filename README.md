# FounderOS

## Your AI-powered command center for a one-person company

FounderOS brings the moving parts of a small business into one place: clients,
comms, social growth, finances, knowledge, and a roster of AI-assisted agents.
You set the direction; your departments and team work from the same operating
picture.

## Getting started

### Prerequisites

- Node.js 22 or newer
- npm

You do not need API keys or external accounts to try the demo.

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
