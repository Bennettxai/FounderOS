# BOS Component Library — Synced from BusinessOS2

## Source of Truth
- **BusinessOS2:** `/Users/rhl/Desktop/MIOSA/_ACTIVE/BusinessOS2/frontend/src/`
- **BOS:** `/Users/rhl/Desktop/MIOSA/_ACTIVE/BOS/frontend/src/lib/ui/`

## Component Library (`src/lib/ui/`)

### Core Components (identical to BusinessOS2)
| Component | Path | Status |
|-----------|------|--------|
| Button | `/ui/button/Button.svelte` | Synced (data-attribute driven, 5 variants) |
| Input | `/ui/input/Input.svelte` | Synced |
| Loading | `/ui/loading/Loading.svelte` | Synced |
| Modal | `/ui/modal/Modal.svelte` | Synced (Bits UI Dialog wrapper) |
| Skeleton | `/ui/skeleton/Skeleton.svelte` | Synced |
| Separator | `/ui/separator/Separator.svelte` | Synced |
| Tooltip | `/ui/tooltip/Tooltip.svelte` | Synced |
| Popover | `/ui/popover/Popover.svelte` | Synced |
| Menu | `/ui/menu/` | Synced (Menu, MenuItem, MenuGroup, etc.) |
| Tabs | `/ui/tabs/` | Synced (Tabs, TabsList, TabsTrigger, TabsContent) |
| ScrollArea | `/ui/scroll-area/ScrollArea.svelte` | Synced |

### Pattern Components (NEW — extracted from BusinessOS2)
| Component | Path | Source Pattern |
|-----------|------|----------------|
| Badge | `/ui/badge/Badge.svelte` | PriorityBadge + StatusBadge patterns |
| Avatar | `/ui/avatar/Avatar.svelte` | MemberCard + ProjectMemberCard patterns |
| IconButton | `/ui/icon-button/IconButton.svelte` | Toolbar icon buttons across 20+ files |
| Card | `/ui/card/Card.svelte` | GlassCard + AppCard patterns |
| SidebarItem | `/ui/sidebar-item/SidebarItem.svelte` | KBSidebar + TablesSidebar patterns |
| Switch | `/ui/switch/Switch.svelte` | Toggle switch patterns |

### Global CSS Systems (in `app.css`)
| System | Status | Notes |
|--------|--------|-------|
| btn-pill | Complete | 10 variants, 5 sizes, 3 modifiers. For standalone action buttons ONLY. |
| badge-* | Complete | badge-active/paused/completed/archived + dark mode |
| priority-* | Complete | priority-critical/high/medium/low + dark mode |
| glass-card | Complete | Glassmorphism with dark mode |
| nav-pill | Complete | Navigation pill with gradient hover |
| Dark mode overrides | Complete | BOS has MORE dark mode CSS than BusinessOS2 |

---

## Usage Guide

### When to Use What

| Need | Use | NOT |
|------|-----|-----|
| Standalone action button (Save, Cancel, Submit) | `btn-pill btn-pill-primary` | raw Tailwind |
| Small toolbar icon button (close, settings, menu) | `<IconButton>` component | `btn-pill btn-pill-icon` |
| Sidebar navigation item | `<SidebarItem>` component | `btn-pill btn-pill-ghost` |
| Status label (Active, Paused) | `<Badge>` component | raw `<span>` with Tailwind |
| User avatar with fallback | `<Avatar>` component | manual `<img>` + `<div>` |
| Card container | `<Card>` component | raw `<div>` with Tailwind |
| Toggle on/off | `<Switch>` component | raw checkbox |
| Dropdown menu items | `<MenuItem>` from `/ui/menu/` | `btn-pill` on list items |
| Full-width list items | Raw Tailwind: `w-full text-left px-4 py-3 hover:bg-gray-50` | `btn-pill` |
| Backdrop overlay | `fixed inset-0 bg-transparent` | `btn-pill btn-pill-ghost` |

### btn-pill Quick Reference
```
Base:     btn-pill
Variants: btn-pill-primary | btn-pill-secondary | btn-pill-ghost | btn-pill-danger
          btn-pill-success | btn-pill-warning | btn-pill-outline | btn-pill-soft | btn-pill-link
Sizes:    btn-pill-xs | btn-pill-sm | (default md) | btn-pill-lg | btn-pill-xl
Modifiers: btn-pill-icon | btn-pill-block | btn-pill-loading
```

### Import
```svelte
import { Badge, Avatar, IconButton, Card, SidebarItem, Switch } from '$lib/ui';
```

---

## Known Anti-Patterns (DO NOT DO)

1. **btn-pill on sidebar items** — Sidebar items need full-width rectangular shape, not pills
2. **btn-pill on backdrop overlays** — Backdrops are invisible click-catchers, not buttons
3. **btn-pill on elements with scoped CSS** — If a Svelte component has its own `<style>` for a button class, don't add btn-pill
4. **btn-pill on dropdown items** — Dropdown items are list items, use `<MenuItem>` or raw Tailwind
5. **btn-pill-icon for toolbar buttons** — Use `<IconButton>` component instead

---

## Future Work
- [ ] Select — Dropdown select with search, multi-select
- [ ] Checkbox — Custom checkbox
- [ ] Radio — Radio button group
- [ ] Textarea — Auto-resize textarea with character count
- [ ] Progress — Progress bar
- [ ] Alert — Alert banner (info, warning, error, success)
- [ ] Toast — Toast notification system
- [ ] Migrate 42+ module files from raw Tailwind to shared components
