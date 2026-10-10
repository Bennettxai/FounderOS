# Pages (Knowledge Base) - Complete Architecture Map

## Overview
A Notion/AFFiNE-like document system with block-based editing, hierarchical pages, and graph visualization.

---

## 1. COMPONENT ARCHITECTURE

### 1.1 Page Layout (`knowledge-v2/+page.svelte`)
```
┌─────────────────────────────────────────────────────────────────┐
│                         PAGES APP                                │
├──────────────┬──────────────────────────────────────────────────┤
│   SIDEBAR    │                  MAIN CONTENT                    │
│   (280px)    │                                                  │
│              │  ┌────────────────────────────────────────────┐  │
│  ┌────────┐  │  │ VIEW ROUTER (based on sidebarStore.view)   │  │
│  │ Header │  │  │                                            │  │
│  │ Search │  │  │  - 'all'/'favorites'/'recent'/'trash'     │  │
│  │ +New   │  │  │    → DocumentEditor (if doc selected)      │  │
│  └────────┘  │  │    → EmptyState (if no doc)                │  │
│              │  │                                            │  │
│  ┌────────┐  │  │  - 'graph'                                 │  │
│  │ Views  │  │  │    → GraphView (3D force-directed)         │  │
│  │ - All  │  │  │                                            │  │
│  │ - Fav  │  │  │  - 'knowledge-graph'                       │  │
│  │ - Rec  │  │  │    → KnowledgeGraph (3D sphere)            │  │
│  │ - Graph│  │  │    + KnowledgeChatPanel (optional)         │  │
│  │ - KG   │  │  │    + KnowledgeDocumentPanel (on select)    │  │
│  │ - Trash│  │  │                                            │  │
│  └────────┘  │  └────────────────────────────────────────────┘  │
│              │                                                  │
│  ┌────────┐  │                                                  │
│  │ Tree   │  │                                                  │
│  │ PAGES  │  │                                                  │
│  │ └ Doc1 │  │                                                  │
│  │ └ Doc2 │  │                                                  │
│  │   └Sub │  │                                                  │
│  └────────┘  │                                                  │
├──────────────┴──────────────────────────────────────────────────┤
│                     QuickSearch Modal (⌘K)                      │
└─────────────────────────────────────────────────────────────────┘
```

### 1.2 Sidebar Components
```
KBSidebar.svelte
├── SidebarHeader.svelte
│   ├── Logo/Title ("Pages")
│   ├── Search input (⌘K trigger)
│   ├── New page button (+)
│   └── Settings button
├── Navigation (View Options)
│   ├── All Pages
│   ├── Favorites
│   ├── Recent
│   ├── Graph View
│   ├── Knowledge Graph
│   └── Trash
├── SidebarSection.svelte ("PAGES")
│   └── RecursiveTreeItem.svelte (for each root)
│       └── SidebarTreeItem.svelte
│           ├── Chevron (expand/collapse)
│           ├── Icon (emoji/default)
│           ├── Title
│           ├── Favorite star
│           └── Actions (on hover)
│               ├── Add subpage
│               └── More menu (Favorite, Duplicate, Delete)
└── Resize handle (drag to resize)
    └── Collapse toggle button
```

### 1.3 Editor Components
```
DocumentEditor.svelte
├── EditorToolbar.svelte
│   ├── Close button (X)
│   ├── Save status ("Saved just now")
│   ├── Share button
│   ├── Favorite button
│   └── More menu (...)
├── EditorHeader.svelte
│   ├── Cover image (optional)
│   ├── Icon (emoji picker)
│   └── Title (contenteditable)
└── Block list
    └── BlockRenderer.svelte (for each block)
        ├── Drag handle (grip)
        ├── Add block button (+)
        ├── Block content (by type):
        │   ├── paragraph
        │   ├── heading_1/2/3
        │   ├── bulleted_list
        │   ├── numbered_list
        │   ├── to_do (checkbox)
        │   ├── toggle
        │   ├── quote
        │   ├── divider
        │   ├── code
        │   └── callout
        └── SlashMenu.svelte (on "/" command)
```

### 1.4 Graph Components
```
GraphView.svelte (Force-directed 3D)
├── Three.js scene
├── Node spheres (by document type)
├── Edge lines (parent-child)
├── Labels (floating)
├── Tooltip (on hover)
├── Controls (reset, zoom in/out)
└── Legend

KnowledgeGraph.svelte (3D Sphere)
├── Threlte Canvas
├── KnowledgeScene.svelte
│   └── MemoryBubble.svelte (for each memory)
├── GraphControls.svelte
└── Panels:
    ├── KnowledgeChatPanel.svelte
    └── KnowledgeDocumentPanel.svelte
```

---

## 2. DATA FLOW

### 2.1 Stores
```typescript
// Document data
documentsStore: {
  documents: Map<string, Document>      // Full documents (with content)
  documentMetas: Map<string, DocumentMeta>  // List items (for sidebar)
  loading: boolean
  error: string | null
}

// Active document selection
activeDocumentStore: {
  id: string | null
  loading: boolean
  saving: boolean
  error: string | null
  lastSaved: string | null
}

// Tree expansion state
treeStore: {
  expandedIds: Set<string>
  loadingIds: Set<string>
}

// Sidebar UI state
sidebarStore: {
  view: SidebarView  // 'all' | 'favorites' | 'recent' | 'graph' | 'knowledge-graph' | 'trash'
  searchQuery: string
  width: number
  collapsed: boolean
}
```

### 2.2 Derived Stores
```typescript
activeDocument      // Current full document (from documentsStore + activeDocumentStore)
rootDocuments       // Documents with parent_id = null
favoriteDocuments   // Documents with is_favorite = true
recentDocuments     // Documents sorted by updated_at (limit 10)
documentMetas       // All non-archived documents
documentTree        // Hierarchical tree structure for sidebar
```

### 2.3 Services
```typescript
// CRUD Operations
fetchDocuments(parentId?)   → Load all documents (or children)
fetchDocument(id)           → Load single document with content
createDocument(params)      → Create new document
updateDocument(id, updates) → Update document
deleteDocument(id, perm?)   → Archive or permanently delete
restoreDocument(id)         → Unarchive document
moveDocument(id, newParent) → Change parent
duplicateDocument(id)       → Copy document
toggleFavorite(id)          → Toggle is_favorite

// Navigation
openDocument(id)            → Set active document
closeDocument()             → Clear active document
openAndFetchDocument(id)    → Set active + fetch content

// Search
searchDocuments(query)      → Full-text search
```

---

## 3. USER INTERACTIONS

### 3.1 Sidebar Interactions
| Action | Trigger | Handler |
|--------|---------|---------|
| Search | Click search / ⌘K | `showQuickSearch = true` |
| New page | Click + | `createDocument()` → `openAndFetchDocument()` |
| Change view | Click nav item | `sidebarStore.setView(view)` |
| Select page | Click tree item | `openAndFetchDocument(id)` |
| Expand/collapse | Click chevron | `treeStore.toggleExpanded(id)` |
| Add subpage | Click + on item | `createDocument({ parent_id })` |
| Toggle favorite | Menu → Favorite | `toggleFavorite(id)` |
| Duplicate | Menu → Duplicate | `duplicateDocument(id)` |
| Delete | Menu → Delete | `deleteDocument(id)` |
| Resize sidebar | Drag edge | `sidebarStore.setWidth(w)` |
| Collapse sidebar | Click toggle | `sidebarStore.toggleCollapsed()` |

### 3.2 Editor Interactions
| Action | Trigger | Handler |
|--------|---------|---------|
| Edit title | Type in title | `handleTitleChange()` → debounced save |
| Change icon | Click icon | Emoji picker → `updateDocument({ icon })` |
| Add cover | Click "Add cover" | Image picker → `updateDocument({ cover })` |
| Edit block | Type in block | `handleBlockChange()` → debounced save |
| New block | Enter at end | `handleBlockAdd(createBlock('paragraph'))` |
| Delete block | Backspace on empty | `handleBlockDelete()` |
| Slash command | Type "/" | Show SlashMenu |
| Select block type | Click in SlashMenu | Transform to new block type |
| Close document | Click X | `handleCloseDocument()` |
| Toggle favorite | Click star | `toggleFavorite(doc.id)` |

### 3.3 Graph Interactions
| Action | Trigger | Handler |
|--------|---------|---------|
| Rotate view | Drag | OrbitControls |
| Zoom | Scroll / buttons | Camera position |
| Select node | Click | `onSelect(doc)` |
| Open node | Double-click | `onNavigate(doc)` → open in editor |
| View tooltip | Hover | Show node details |
| Reset view | Click reset button | Reset camera |

---

## 4. API ENDPOINTS (Backend)

```
GET    /contexts                    → List all contexts
GET    /contexts?parent_id=xxx      → List children
GET    /contexts/:id                → Get single context with blocks
POST   /contexts                    → Create context
PUT    /contexts/:id                → Update context
PATCH  /contexts/:id/blocks         → Update blocks only
PATCH  /contexts/:id/archive        → Archive context
PATCH  /contexts/:id/unarchive      → Unarchive context
DELETE /contexts/:id                → Permanently delete
POST   /contexts/:id/duplicate      → Duplicate context
POST   /contexts/:id/share          → Enable sharing
DELETE /contexts/:id/share          → Disable sharing
GET    /contexts/public/:shareId    → Get public context
POST   /contexts/aggregate          → Aggregate multiple contexts
```

---

## 5. BLOCK TYPES

| Type | Render | Props |
|------|--------|-------|
| `paragraph` | `<p>` contenteditable | - |
| `heading_1` | `<h1>` contenteditable | - |
| `heading_2` | `<h2>` contenteditable | - |
| `heading_3` | `<h3>` contenteditable | - |
| `bulleted_list` | `• ` + content | - |
| `numbered_list` | `1. ` + content | - |
| `to_do` | checkbox + content | `checked: boolean` |
| `toggle` | collapsible | `collapsed: boolean` |
| `quote` | blockquote | - |
| `divider` | `<hr>` | - |
| `code` | `<pre><code>` | `language: string` |
| `callout` | icon + content | `icon: string`, `color: string` |
| `image` | `<img>` | `url: string`, `caption: string` |
| `bookmark` | link preview | `url: string` |
| `embed` | iframe | `url: string` |

---

## 6. KEYBOARD SHORTCUTS

| Shortcut | Action |
|----------|--------|
| `⌘K` | Open quick search |
| `Enter` | Create new paragraph block |
| `Shift+Enter` | Line break within block |
| `Backspace` | Delete empty block |
| `/` | Open slash command menu |
| `Esc` | Close menus/modals |
| `↑/↓` | Navigate slash menu |

---

## 7. CURRENT STATUS

### Working ✅
- [x] Sidebar layout and navigation
- [x] View switching (All, Favorites, Recent, Graph, KG, Trash)
- [x] Document tree rendering
- [x] Tree expand/collapse
- [x] Document editor display
- [x] Title editing
- [x] Icon display
- [x] Block rendering (basic)
- [x] Slash command menu
- [x] Auto-save with debounce
- [x] GraphView (3D force-directed)
- [x] KnowledgeGraph (3D sphere)

### Needs Testing 🔍
- [ ] Create new document
- [ ] Delete document
- [ ] Add subpage
- [ ] Duplicate document
- [ ] Move document (drag-drop)
- [ ] Quick search
- [ ] Favorites toggle
- [ ] Block type conversion
- [ ] Cover image
- [ ] Graph node selection → open document

### Known Issues 🐛
- [ ] Type mismatches between Context API and Document types
- [ ] Missing children_count calculation
- [ ] Icon type handling (string vs object)

---

## 8. FILES REFERENCE

```
src/lib/modules/knowledge-base/
├── entities/
│   ├── types.ts          # Core type definitions
│   ├── block.ts          # Block-specific types
│   └── schemas.ts        # Zod validation schemas
├── stores/
│   ├── documents.ts      # Document stores
│   ├── yjs-block-store.ts # CRDT store (future)
│   └── database-store.ts  # Database view store
├── services/
│   ├── documents.service.ts   # Document CRUD
│   ├── page-adapter.ts        # Context↔Document mapping
│   ├── knowledge-base.service.ts # Graph service
│   └── ai-integration.service.ts # AI features
├── views/
│   ├── sidebar/
│   │   ├── KBSidebar.svelte
│   │   ├── SidebarHeader.svelte
│   │   ├── SidebarSection.svelte
│   │   ├── SidebarTreeItem.svelte
│   │   ├── RecursiveTreeItem.svelte
│   │   └── QuickSearch.svelte
│   ├── editor/
│   │   ├── DocumentEditor.svelte
│   │   ├── EditorHeader.svelte
│   │   ├── EditorToolbar.svelte
│   │   ├── BlockRenderer.svelte
│   │   └── SlashMenu.svelte
│   ├── graph/
│   │   └── GraphView.svelte
│   └── database/
│       ├── Database.svelte
│       ├── DatabaseTable.svelte
│       └── ...
└── index.ts              # Module exports

src/lib/components/knowledge/
├── KnowledgeGraph.svelte
├── KnowledgeScene.svelte
├── MemoryBubble.svelte
├── KnowledgeChatPanel.svelte
├── KnowledgeDocumentPanel.svelte
└── GraphControls.svelte

src/routes/(app)/knowledge-v2/
└── +page.svelte          # Main page orchestrator
```
