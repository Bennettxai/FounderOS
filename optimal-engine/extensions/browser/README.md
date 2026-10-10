# Optimal Engine — Web Clipper

Chrome extension (Manifest V3) that lets you clip web pages, selections, and
URLs into an Optimal Engine workspace.

This is an optional surface. The extension does not own truth; it sends clipped
content to the backend, where the engine preserves source context and routes it
through the governed memory/intake flow.

## Features

- **Clip page** — saves the first 1000 characters of the active tab's body text
- **Clip selection** — saves highlighted text (shown when text is selected)
- **Search workspace context** — live search across any workspace from the popup
- **Context menu** — right-click a selection to "Save to Optimal Engine", or
  right-click a link to "Clip URL to Optimal Engine"
- **Workspace switcher** — change target workspace from the popup dropdown
- **Recent clips** — last 10 clips displayed in the popup
- **Options page** — configure engine URL, API key, default workspace, auto-tag

## Requirements

- Chrome 114+ (Manifest V3 service workers)
- A running [Optimal Engine](https://github.com/Miosa-osa/OptimalEngine) instance
- Node.js 20+

## Development

```bash
cd extensions/browser
npm install
npm run build        # produces dist/
npm run dev          # rebuild on file change (watch mode)
npm run typecheck    # tsc --noEmit
```

## Loading as an unpacked extension in Chrome

1. Run `npm run build` — this produces `extensions/browser/dist/`.
2. Open Chrome and navigate to `chrome://extensions`.
3. Enable **Developer mode** (toggle in the top-right corner).
4. Click **Load unpacked**.
5. Select the `extensions/browser/dist/` directory.
6. The "Optimal Engine — Web Clipper" extension will appear in your toolbar.

## Configuration

Open the extension options page (click **Settings** in the popup, or find the
extension in `chrome://extensions` and click **Details → Extension options**):

| Setting | Default | Description |
|---|---|---|
| Engine URL | `http://localhost:4200` | Base URL of your Optimal Engine instance |
| Default workspace | _(blank)_ | Workspace ID that clips are saved to |
| API key | _(blank)_ | Sent as `Authorization: Bearer ...` — leave blank for local engines |
| Auto-tag clips | off | Adds domain + date tags automatically |

## Project layout

```
extensions/browser/
├── manifest.json           Manifest V3 declaration
├── package.json
├── vite.config.ts          Vite + @crxjs/vite-plugin
├── tailwind.config.ts
├── tsconfig.json
├── public/
│   └── icons/              16/32/48/128 px PNG icons
└── src/
    ├── popup/              Popup UI (React 19)
    │   ├── App.tsx         search + clip + workspace selector
    │   ├── main.tsx
    │   ├── popup.html
    │   └── styles.css      Tailwind base + design tokens
    ├── options/            Options page
    │   ├── App.tsx
    │   ├── main.tsx
    │   └── options.html
    ├── content/
    │   └── content.ts      Captures page info + selection on request
    ├── background/
    │   └── service-worker.ts  Context-menu setup, message router
    └── lib/
        ├── client.ts       Engine HTTP client (clip, search, profile)
        ├── storage.ts      chrome.storage.local wrapper
        └── types.ts        Shared TypeScript types
```

## Engine API used

| Method | Path | Purpose |
|---|---|---|
| `POST` | `/api/memory` | Submit a clip through governed memory intake |
| `GET` | `/api/memory?workspace=&limit=` | List recent memory projections |
| `GET` | `/api/search?q=&workspace=&limit=` | Search workspace context |
| `GET` | `/api/profile?workspace=&bandwidth=l1` | Workspace profile/projection summary |
| `GET` | `/api/workspaces?tenant=` | List workspaces |

## Building for production

```bash
npm run build
```

The `dist/` directory is a self-contained unpacked extension — zip it and
upload to the Chrome Web Store if you want to distribute it.
