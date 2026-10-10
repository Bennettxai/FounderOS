# Optimal Engine — Raycast Extension

Search, ask, and submit context to
[Optimal Engine](https://github.com/Miosa-osa/OptimalEngine) directly from
Raycast.

This is an optional agent/user surface over the backend runtime. Raycast does
not own workspace truth; it calls the engine API for search, retrieval, and
governed memory intake.

## Commands

| Command | Description |
|---------|-------------|
| **Search Memory** | Search governed memory/context projections in your active workspace |
| **Add Memory** | Submit a fact, decision, observation, or note through memory intake |
| **Ask Engine** | Ask a workspace-scoped question using governed retrieval/context |

## Setup

### 1. Install the Raycast app

Download from [raycast.com](https://raycast.com) and install.

### 2. Install this extension (development mode)

```bash
cd extensions/raycast
npm install
npm run dev
```

Raycast will open and import the extension automatically.

### 3. Configure preferences

Open Raycast → Extension Preferences → Optimal Engine and set:

| Preference | Default | Description |
|-----------|---------|-------------|
| Engine URL | `http://localhost:4200` | Base URL of your running Optimal Engine instance |
| Default Workspace | `default` | Workspace slug to search / write to |
| API Key | _(empty)_ | Bearer token — only required when `auth_required=true` |

## Usage

### Search Memory

Type any term. Results stream in with a 200 ms debounce. The detail panel on the
right shows the content projection, audience, relevance score, and metadata.

Actions available on each result:
- **Copy Content** — copy the full memory text to clipboard
- **Copy Slug** — copy the memory slug
- **Open Source URL** — open `citation_uri` in the browser (when present)

### Add Memory

Fill in the form:
- **Content** (required) — the fact, decision, observation, or source-backed note
- **Audience** — `general | technical | executive | internal`
- **Static** — pin the memory so it is never evicted by the engine
- **Source URL** — optional provenance link

On success a toast displays the new memory ID and the form closes automatically.

### Ask Engine

Type a natural-language question and press **Enter**. The engine answers using
workspace-scoped retrieval/context. The detail view renders the answer as
Markdown with sources listed below when the API returns them.

Actions available on each answer:
- **Copy Answer** — copy the full response body
- **Copy Sources** — copy the source slugs / URLs
- **Ask Follow-up** — pre-fills the search bar with the previous query
- **Open: \<slug\>** — opens the citation URL for any source that has one

## Development

```bash
# Type-check only (no Raycast CLI required)
npx tsc --noEmit

# Local source verification
npm run lint

# Strict store/publisher lint; requires package.json author to be a real
# Raycast publisher account.
npm run lint:publish

# TypeScript build check
npm run build

# Raycast bundle build; requires Raycast app and a valid publisher account
npm run build:raycast

# Live-reload dev mode
npm run dev
```

## File Structure

```
src/
├── search-memory.tsx       Search command
├── add-memory.tsx          Add command
├── ask-engine.tsx          Ask command
└── lib/
    ├── client.ts           Engine HTTP wrapper (native fetch, no deps)
    ├── preferences.ts      Typed preference accessor
    └── types.ts            All shared TypeScript types
└── hooks/
    └── use-workspaces.ts   Cached workspace list hook
```

## Metadata / Screenshots

Place Raycast Store screenshots (1280×800 PNG) in `metadata/`. See the
[Raycast developer docs](https://developers.raycast.com/basics/prepare-an-extension-for-store#screenshots)
for naming conventions.
