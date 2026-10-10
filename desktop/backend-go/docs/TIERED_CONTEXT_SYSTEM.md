# Tiered Context Loading System

## Overview

The Tiered Context Loading System enhances BusinessOS's RAG (Retrieval-Augmented Generation) capabilities by providing **scoped, hierarchical context** to AI queries based on user selections in the chat interface.

Instead of searching ALL documents for every query, the system now:
1. Focuses on **selected items** with full detail (Level 1)
2. Provides **awareness** of related items via summaries (Level 2)
3. Enables **on-demand fetching** via AI tool calls (Level 3)

## Architecture

```
┌────────────────────────────────────────────────────────────────┐
│ LEVEL 1: FULL CONTEXT (selected items)                         │
│ • Selected Project → all tasks, descriptions, full content     │
│ • Selected Contexts → all document blocks, embeddings searched │
│ • Direct relationships → linked clients, assigned team         │
├────────────────────────────────────────────────────────────────┤
│ LEVEL 2: AWARENESS (titles/summaries only)                     │
│ • Other projects in scope → just titles, status                │
│ • Sibling contexts → just names, types                         │
│ • Related entities → just names (not full details)             │
├────────────────────────────────────────────────────────────────┤
│ LEVEL 3: ON-DEMAND (AI fetches via tool call)                  │
│ • Tool: get_entity_context(type, id) → returns full details    │
│ • Used when user mentions something AI only has awareness of   │
└────────────────────────────────────────────────────────────────┘
```

## Key Components

### 1. TieredContextService (`internal/services/tiered_context.go`)

The core service that builds hierarchical context for AI queries.

```go
type TieredContextRequest struct {
    UserID     string
    ContextIDs []uuid.UUID  // Selected documents
    ProjectID  *uuid.UUID   // Selected project
    NodeID     *uuid.UUID   // Business node context
}

type TieredContext struct {
    Level1 *FullContext      // Full details for selected items
    Level2 *AwarenessContext // Summaries of related items
    Level3 *OnDemandRegistry // Registry of fetchable entities
}
```

**Key Methods:**
- `BuildTieredContext()` - Constructs the full 3-tier context
- `ScopedRAGSearch()` - Performs embedding search ONLY within selected contexts
- `FormatForAI()` - Formats context as structured system prompt

### 2. Scoped RAG Search (`internal/services/embedding.go`)

New method for context-limited vector search:

```go
func (s *EmbeddingService) ScopedSimilaritySearch(
    ctx context.Context,
    query string,
    contextIDs []uuid.UUID,  // Only search these documents
    userID string,
    limit int,
) ([]RelevantBlock, error)
```

Uses SQL filter: `c.id = ANY($3)` to restrict search scope.

### 3. GetEntityContextTool (`internal/tools/context_tools.go`)

On-demand context fetching tool for AI:

```go
type GetEntityContextTool struct {
    pool   *pgxpool.Pool
    userID string
}

// Supports entity types:
// - project, context, task, client, team_member, node
func (t *GetEntityContextTool) Execute(ctx context.Context, input GetEntityContextInput) GetEntityContextOutput
```

The AI can call this tool when it needs more details about an entity it only has awareness of.

### 4. Updated Chat Handler (`internal/handlers/chat.go`)

New request fields:
```go
type SendMessageRequest struct {
    // ... existing fields ...
    ContextIDs []string  `json:"context_ids"`  // Multiple contexts
    NodeID     *string   `json:"node_id"`      // Business node
}
```

The handler now:
1. Parses ContextIDs and NodeID from requests
2. Uses tiered context when selections exist
3. Falls back to legacy global RAG otherwise

## Data Flow

```
Frontend Chat UI
    │
    │ POST /api/chat/message
    │ {
    │   "message": "What's the status of our project?",
    │   "context_ids": ["doc-uuid-1", "doc-uuid-2"],
    │   "project_id": "project-uuid",
    │   "node_id": "node-uuid"
    │ }
    │
    ▼
SendMessage Handler
    │
    ├── Parse IDs (ContextIDs, ProjectID, NodeID)
    │
    ├── Build Tiered Context
    │   ├── Level 1: Full project + tasks + selected docs
    │   ├── Level 2: Other projects, sibling docs, clients
    │   └── Level 3: Registry of fetchable entities
    │
    ├── Scoped RAG Search (only within selected contexts)
    │
    ├── Format for AI (structured system prompt)
    │
    └── Stream LLM Response
```

## Context Formatting

The `FormatForAI()` method produces structured context like:

```markdown
## Context Overview

### Primary Focus (Full Details)

**Active Project: Client Portal Redesign**
- Status: ACTIVE | Priority: HIGH
- Description: Complete redesign of the client-facing portal

**Project Tasks:**
- [TODO] Design new dashboard layout (HIGH) - Due: 2024-01-15
- [DONE] Create wireframes (MEDIUM)

**Selected Documents:**
- **Client Requirements** (DOCUMENT, 2450 words)
  Content:
  > The client requires a modern, responsive design...

**Relevant Knowledge (from selected documents):**
1. From "Client Requirements" (87% match):
   > Users should be able to customize their dashboard...

### Context Awareness (Summaries Only)

**Business Node: Q1 Client Projects** (BUSINESS)
- Purpose: All client-facing development work for Q1

**Other Projects in Scope:** API Integration, Mobile App v2

**Related Documents:** Design System, Brand Guidelines

### On-Demand Context
You can use the `get_entity_context` tool to retrieve full details for any entity mentioned above.
```

## Selection Scenarios

| Scenario | Level 1 | Level 2 | Behavior |
|----------|---------|---------|----------|
| Project + Node | Selected project + its tasks | Node's other projects/contexts | Focus on project, aware of node |
| Just Node | Node overview | All node's projects/contexts | AI chooses what to expand |
| Multiple Contexts | All selected docs | Siblings under same parent | Search across all selected |
| Cross-node (Project A, Node B) | Project A details | Node B's items | AI understands overlap |

## Configuration

The tiered context service is initialized in `cmd/server/main.go`:

```go
if embeddingService.HealthCheck(ctx) {
    contextBuilder = services.NewContextBuilder(pool, embeddingService)
    tieredContextService = services.NewTieredContextService(pool, embeddingService)
    log.Printf("Tiered context service enabled")
}
```

## Dependencies

- PostgreSQL with pgvector extension (for embeddings)
- Ollama with `nomic-embed-text` model (for generating embeddings)

## Future Enhancements

1. **Node-Entity Relationships**: Add `node_id` column to projects and contexts tables for direct node-based queries
2. **Conversation Memory**: Include relevant past conversation context
3. **Dynamic Tool Registration**: Register the `get_entity_context` tool with the LLM provider
4. **Caching**: Cache frequently accessed Level 1 context

## Files Modified/Created

| File | Change Type | Description |
|------|-------------|-------------|
| `internal/services/tiered_context.go` | NEW | Core tiered context service |
| `internal/services/embedding.go` | MODIFIED | Added ScopedSimilaritySearch |
| `internal/tools/context_tools.go` | NEW | On-demand context fetching tool |
| `internal/handlers/chat.go` | MODIFIED | Tiered context integration |
| `internal/handlers/handlers.go` | MODIFIED | Added TieredContextService field |
| `cmd/server/main.go` | MODIFIED | Wire up TieredContextService |
