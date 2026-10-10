# BusinessOS Frontend

SvelteKit 2 + Svelte 5 frontend for the BusinessOS platform — a desktop-metaphor business OS with an AI-powered agent layer (OSA).

---

## Quick Start

```bash
pnpm install
pnpm dev
# open http://localhost:5173
```

**Requirements:** Node.js 22+, backend running on port 8001

The dev server proxies API calls to the backend via Vite config. No `.env` file is needed for local development.

---

## Tech Stack

| Dependency | Version | Purpose |
|------------|---------|---------|
| SvelteKit | 2.x | Meta-framework, file-based routing |
| Svelte | 5.x | UI framework (runes: `$state`, `$derived`, `$effect`) |
| TypeScript | 5.x | Strict mode throughout |
| TailwindCSS | 4.x | Utility-first styling |
| Vite | 7.x | Build tool and dev server |
| Three.js + Threlte | 0.182 / 8.x | 3D desktop effects |
| xterm.js | 6.x | Terminal emulator |
| Monaco Editor | 0.55 | In-app code editor |
| MediaPipe Hands | 0.4 | Gesture control input |
| Yjs + y-indexeddb | 13.x | Collaborative document editing |
| Zod | 4.x | Runtime schema validation |
| bits-ui | 2.x | Headless UI primitives |
| lucide-svelte | latest | Icon library |
| vitest | 4.x | Unit test runner |
| Playwright | 1.58 | E2E test runner |

---

## Project Structure

```
frontend/
├── src/
│   ├── routes/
│   │   ├── (app)/           # Authenticated app routes (26 modules)
│   │   ├── (auth)/          # Login, register, forgot/reset password
│   │   ├── window/          # Windowed views (OSA agent, popup chat)
│   │   ├── onboarding/      # User onboarding flow
│   │   ├── templates/       # Template browser and preview
│   │   ├── landing/         # Marketing landing page
│   │   └── +page.svelte     # Root redirect
│   │
│   ├── lib/
│   │   ├── api/             # Typed API client modules (one per domain)
│   │   ├── components/      # 588 Svelte components across 38 domains
│   │   ├── stores/          # Svelte 5 stores, domain-split with barrel re-exports
│   │   ├── services/        # Voice commands, permissions, push, terminal
│   │   ├── modules/         # Self-contained feature modules (knowledge base, etc.)
│   │   ├── hooks/           # Reusable Svelte hooks
│   │   ├── utils/           # Shared utilities
│   │   └── types/           # TypeScript type definitions
│   │
│   └── app.html             # HTML shell
│
└── static/                  # Static assets
```

---

## Key Modules

| Module | Route | Description |
|--------|-------|-------------|
| Desktop | `/(app)/` | Window manager, dock, file browser, spotlight, 3D effects |
| Chat | `/(app)/chat` | AI conversation with Focus Modes, hybrid search, artifacts |
| OSA Agent | `/window/osa/[id]` | AI agent terminal interface with streaming |
| Dashboard | `/(app)/dashboard` | Customizable widget layouts |
| Knowledge Base | `/(app)/knowledge` | Notion-like document editor (Yjs-backed) |
| Modules | `/(app)/modules` | Generated app viewer and management |
| CRM | `/(app)/crm` | Deals pipeline, clients, companies |
| Projects | `/(app)/projects` | Project tracking |
| Tasks | `/(app)/tasks` | Kanban, list, calendar views |
| Team | `/(app)/team` | Member management, roles, invites |
| Nodes | `/(app)/nodes/[id]` | Decision trees, delegation editor |
| Integrations | `/(app)/integrations` | OAuth connections (Google, Microsoft, Slack, etc.) |
| Settings | `/(app)/settings/*` | Workspace, AI models, desktop, usage |
| Usage | `/(app)/usage` | AI usage analytics |
| Terminal | `/(app)/terminal` | Docker workspace terminal |
| Voice Notes | `/(app)/voice-notes` | Voice recording and transcription |

---

## Component Architecture

Components are grouped by feature domain under `src/lib/components/`:

```
components/
├── chat/
│   ├── messages/        # AssistantMessage, UserMessage, ChatMessageList
│   ├── input/           # ChatInputBar, attachments, voice
│   ├── panels/          # ArtifactsPanel, HybridSearchPanel, ChatSidebar
│   ├── focus/           # FocusModeSelector, FocusInputArea, FocusCard
│   ├── conversations/   # ConversationListPanel, ConversationListItem
│   ├── modals/          # DocumentUploadModal, MemoryDetailModal
│   └── shared/          # MemoryCard, PermissionGate, RoleContextBadge
├── desktop/             # Desktop environment, icons, taskbar, spotlight
├── desktop3d/           # 3D WebGL desktop (Three.js via Threlte)
├── editor/              # Block-based rich text editor
├── settings/            # AIModelSettings, AIUsageStats, DesktopSettings
├── osa/                 # OSA agent UI (OsaOrb, BuildProgress)
├── ui/                  # Generic primitives (Button, Modal, Input)
├── workspace/           # Workspace settings, members, roles
└── ...                  # 27 more feature domains
```

### Store Architecture

Stores use Svelte 5 runes and are split by domain with barrel re-exports:

```
stores/
├── desktopStore.ts              # Re-exports: background, icons, animations, theme
├── windowStore.ts               # Re-exports: state, snap, icons, modules, serialization
├── chat/
│   ├── chatStore.svelte.ts      # Chat state, conversations
│   ├── chatUIStore.svelte.ts    # Panel visibility, focus mode
│   └── types.ts
├── auth.ts
├── projects.ts / tasks.ts / clients.ts / workspaces.ts
└── ...                          # 30+ store files total
```

---

## Conventions

### Svelte 5 Patterns

Use runes — no legacy `$:` reactivity, no `createEventDispatcher`:

```svelte
<script lang="ts">
  let { data, onsubmit, onclose }: {
    data: ItemData;
    onsubmit?: (item: Item) => void;
    onclose?: () => void;
  } = $props();

  let count = $state(0);
  let doubled = $derived(count * 2);

  $effect(() => {
    console.log('count changed:', count);
  });
</script>
```

### Event Handling

All components use callback props:

```svelte
<!-- Parent -->
<ChildComponent
  onselect={(item) => handleSelect(item)}
  onclose={() => (showModal = false)}
/>

<!-- Child -->
<script lang="ts">
  let { onselect, onclose }: {
    onselect?: (item: Item) => void;
    onclose?: () => void;
  } = $props();
</script>
<button onclick={() => onselect?.(item)}>Select</button>
```

### Console Logging

Wrap `console.log` in dev guards (stripped in production builds):

```typescript
if (import.meta.env.DEV) console.log('debug info', data);

// console.error and console.warn are NOT guarded (intentional)
console.error('Something failed:', error);
```

### File Size Targets

| Type | Target | Max |
|------|--------|-----|
| Svelte components | < 300 lines | 500 lines |
| TypeScript modules | < 300 lines | 400 lines |
| Store files | < 200 lines | 350 lines |

Split large files using barrel re-exports for backwards compatibility.

---

## Environment Variables

No `.env` is required for local development (Vite proxy handles API routing).

For production deployments, set the following:

```env
# Backend API URL (Cloud Run service)
VITE_API_URL=https://businessos-api-xxxxx.run.app/api

# Public URL (your deployment domain)
PUBLIC_URL=https://your-app.vercel.app
```

Copy `.env.production.example` as a reference for production configuration.

---

## Scripts

```bash
pnpm dev              # Dev server (port 5173)
pnpm build            # Production build
pnpm preview          # Preview production build locally
pnpm check            # Svelte type checking (svelte-check)
pnpm check:watch      # Type checking in watch mode

pnpm test             # Run unit tests (vitest)
pnpm test:watch       # Unit tests in watch mode
pnpm test:coverage    # Unit tests with coverage report
pnpm test:ci          # Unit tests for CI (no watch)

pnpm test:e2e         # Playwright E2E tests (all browsers)
pnpm test:e2e:ui      # Playwright with UI runner
pnpm test:e2e:chromium  # Chromium only
pnpm test:e2e:firefox   # Firefox only
pnpm test:e2e:webkit    # WebKit only
pnpm test:e2e:report    # Show last Playwright report

pnpm lint             # Lint with svelte-check
```

---

## Contributing

1. Branch from `main` with `feat/` or `fix/` prefix
2. Follow Svelte 5 patterns — runes, callback props, no `createEventDispatcher`
3. Keep components under 500 lines — split with barrel re-exports if larger
4. Wrap `console.log` in `import.meta.env.DEV` guards
5. Run `pnpm build && pnpm check` before committing
6. New UI logic should have unit tests under `src/lib/**/*.test.ts`
