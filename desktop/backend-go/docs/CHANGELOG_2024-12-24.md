# BusinessOS Backend - Knowledge Update (December 24, 2024)

## Summary

This update introduces the **Knowledge Base Enhancement** - a major upgrade to the AI context system that enables scoped, intelligent context loading for chat interactions.

## What's New

### Tiered Context Loading System

Previously, the RAG system searched ALL user documents for every query. Now, the system intelligently scopes context based on user selections:

- **Scoped RAG Search**: Only searches within selected documents
- **Multi-Context Support**: Handle multiple selected contexts simultaneously
- **Project Context**: Full project details including tasks, client, and team
- **Business Node Awareness**: Understand relationships within business nodes
- **On-Demand Fetching**: AI can request additional context via tool calls

### New Files

| File | Lines | Purpose |
|------|-------|---------|
| `internal/services/tiered_context.go` | ~720 | Core tiered context service with Level 1/2/3 hierarchy |
| `internal/tools/context_tools.go` | ~410 | GetEntityContextTool for AI on-demand context fetching |
| `docs/TIERED_CONTEXT_SYSTEM.md` | - | Comprehensive documentation |

### Modified Files

| File | Changes |
|------|---------|
| `internal/services/embedding.go` | Added `ScopedSimilaritySearch()` method |
| `internal/handlers/chat.go` | Added `ContextIDs`, `NodeID` fields; integrated tiered context |
| `internal/handlers/handlers.go` | Added `TieredContextService` to Handlers struct |
| `cmd/server/main.go` | Wired up `TieredContextService` initialization |

## Technical Details

### New Request Fields

```json
{
  "message": "What's the project status?",
  "context_ids": ["uuid1", "uuid2"],
  "project_id": "uuid3",
  "node_id": "uuid4"
}
```

### Context Hierarchy

```
Level 1 (Full): Selected project + tasks + documents + scoped RAG results
Level 2 (Awareness): Other projects, sibling docs, related clients (titles only)
Level 3 (On-Demand): Registry of all entities AI can request details for
```

### API Compatibility

- **Backward Compatible**: Legacy `context_id` field still works
- **Graceful Fallback**: Falls back to global RAG if no selections
- **No Schema Changes**: Works with existing database schema

## Performance Improvements

- **Reduced Search Scope**: Only searches selected contexts vs. all documents
- **Efficient Queries**: Raw SQL with proper indexing for context building
- **Minimal Overhead**: Tiered context only built when selections exist

## Dependencies

No new dependencies added. Uses existing:
- pgvector for embeddings
- Ollama with nomic-embed-text model

## Testing Notes

To test the tiered context system:

1. Start the backend server
2. In the chat UI, select a project and/or multiple contexts
3. Send a message - observe scoped RAG in server logs:
   ```
   [Chat] Tiered context built: Level1(project=true, contexts=2), Level2(projects=3, siblings=5)
   [Chat] Tiered context: scoped RAG found 4 relevant blocks
   ```

## Future Work

1. Add `node_id` foreign key to projects/contexts tables
2. Implement conversation memory integration
3. Register get_entity_context tool with LLM providers
4. Add caching for frequently accessed context

---

*Branch: knowledge*
*Author: Claude Code + Roberto*
*Date: December 24, 2024*
