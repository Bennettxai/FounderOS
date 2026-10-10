# Optimal Engine Docs Site

SvelteKit static documentation surface for
[Optimal Engine](https://github.com/Miosa-osa/OptimalEngine).

This app is a projection surface. It should explain the backend runtime:
workspace topology, source-first intake, Signal classification, Memory Core,
Context Packages, Active Memory Pools, workflow/Skill Packages, tool/model
governance, integrations, and wiki/export projections. It is not the canonical
store.

## Run locally

```bash
cd apps/docs
npm install
npm run dev        # dev server at http://localhost:1422
```

## Build (static)

```bash
npm run build      # produces build/
```

The output is a fully static site. Every route is pre-rendered at build time.
Serve with any static host.

## Type check

```bash
npm run check
```

## Deploy

```bash
# Netlify
netlify deploy --dir build --prod

# Cloudflare Pages
wrangler pages publish build

# nginx — point root at build/
```

## Structure

```
src/
├── routes/
│   ├── +layout.svelte           Header shell
│   └── +page.svelte             Current docs landing page
└── lib/
    ├── components/
    │   ├── CodeBlock.svelte
    │   └── EngineThemeToggle.svelte
```

## Content Rule

The canonical markdown docs live under `docs/`. Keep product/architecture truth
there first, then mirror or render it here. Do not introduce separate
architecture claims in this app that disagree with `docs/README.md`.

## Tech

- SvelteKit 2 + Svelte 5 (runes)
- adapter-static, prerender: true
- No external UI dependencies — all styles are hand-written CSS vars from the Foundation token system
- Port 1422 (dev + preview)
