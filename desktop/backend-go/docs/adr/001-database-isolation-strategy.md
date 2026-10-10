# ADR-001: Database Isolation Strategy

## Status
Accepted

## Date
January 2026

## Context

BusinessOS is a multi-tenant business operating system where multiple organizations (workspaces) share a single application instance. Each workspace contains sensitive business data including:

- Projects, tasks, and clients
- AI conversations and agent memories
- User credentials and OAuth tokens
- Generated applications and templates
- Email metadata and calendar events
- Document embeddings and semantic search results

We needed to determine the appropriate database isolation strategy to ensure:

1. **Data Security:** Prevent unauthorized cross-tenant data access
2. **Performance:** Efficient queries without excessive overhead
3. **Scalability:** Support growing number of workspaces
4. **Maintainability:** Simple schema migrations and upgrades
5. **Cost Efficiency:** Reasonable infrastructure costs

### Options Considered

1. **Database Per Tenant:** Separate PostgreSQL database for each workspace
   - Pros: Maximum isolation, independent backups
   - Cons: High overhead, complex connection pooling, difficult migrations

2. **Schema Per Tenant:** Separate PostgreSQL schema for each workspace
   - Pros: Better isolation than shared tables, contained backups
   - Cons: Connection management complexity, schema proliferation

3. **Workspace-Level Row Isolation:** Single database with workspace_id column
   - Pros: Simple architecture, efficient queries, easy migrations
   - Cons: Requires careful query validation, risk of cross-tenant leaks

## Decision

**We chose Workspace-Level Row Isolation (Option 3)** with a hierarchical memory system.

### Isolation Model

All tenant data is stored in shared tables with a `workspace_id` foreign key:

```sql
-- Example: Projects table
CREATE TABLE projects (
    id UUID PRIMARY KEY,
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    ...
);

CREATE INDEX idx_projects_workspace ON projects(workspace_id);
```

### Memory Hierarchy

The system implements a 3-tier isolation model for AI agent memories:

1. **Workspace-Level (Shared):**
   - `visibility = 'workspace'`
   - Accessible to all active workspace members
   - Used for: Team decisions, processes, patterns, policies
   - Stored in: `workspace_memories` table

2. **User-Level (Private):**
   - `visibility = 'private'`
   - Accessible only to owner (`owner_user_id`)
   - Used for: Personal preferences, individual context
   - Stored in: `workspace_memories` table with owner reference

3. **Shared Private:**
   - `visibility = 'shared'`
   - Accessible to owner + specific users in `shared_with` array
   - Used for: Team-specific knowledge, cross-functional collaboration
   - Stored in: `workspace_memories` table with access list

### Access Control Functions

PostgreSQL functions enforce isolation at the database level:

```sql
-- Check if user can access a memory
CREATE FUNCTION can_access_memory(p_user_id TEXT, p_memory_id UUID)
RETURNS BOOLEAN AS $$
BEGIN
    -- Workspace memories: Check workspace membership
    -- Private memories: Check ownership
    -- Shared memories: Check ownership OR shared_with array
END;

-- Get workspace-level memories (team knowledge)
CREATE FUNCTION get_workspace_memories(
    p_workspace_id UUID,
    p_user_id TEXT,
    p_memory_type TEXT,
    p_limit INT
) RETURNS TABLE (...);

-- Get user's private and shared memories
CREATE FUNCTION get_user_memories(...) RETURNS TABLE (...);

-- Get all accessible memories (workspace + private + shared)
CREATE FUNCTION get_accessible_memories(...) RETURNS TABLE (...);
```

## Implementation

### Backend Enforcement

**1. Middleware Layer (Authentication):**
- `desktop/backend-go/internal/middleware/auth.go`
- Validates user session and workspace membership
- Injects `user_id` and `workspace_id` into request context

**2. Service Layer (Business Logic):**
- All queries filtered by `workspace_id` from authenticated context
- Example: `memory_hierarchy_service.go` enforces visibility rules
- Services validate workspace membership before data access

**3. Repository Layer (Database Access):**
- SQLC-generated queries with workspace_id parameters
- Composite indexes on `(workspace_id, ...)` for performance
- Cascading deletes via foreign keys

**4. Database Constraints:**
```sql
-- Enforce workspace isolation
ALTER TABLE workspace_memories
ADD CONSTRAINT check_private_owner
CHECK (
    (visibility = 'workspace' AND owner_user_id IS NULL) OR
    (visibility IN ('private', 'shared') AND owner_user_id IS NOT NULL)
);

-- Prevent orphaned records
FOREIGN KEY (workspace_id) REFERENCES workspaces(id) ON DELETE CASCADE
```

### Security Measures

1. **Row-Level Security (RLS):** NOT used (application-layer enforcement preferred)
2. **Query Validation:** All handlers use parameterized queries via SQLC
3. **Index Strategy:** Composite indexes starting with `workspace_id`
4. **Audit Logging:** `workspace_audit_logs` table tracks all access
5. **Session Cache:** Redis-backed session validation with HMAC-secured keys

### Files Involved

**Migrations:**
- `desktop/backend-go/supabase/migrations/028_workspaces.sql` - Workspace tables
- `desktop/backend-go/supabase/migrations/051_memory_hierarchy.sql` - Memory isolation

**Services:**
- `desktop/backend-go/internal/services/memory_hierarchy_service.go`
- `desktop/backend-go/internal/services/memory_hierarchy_service_test.go`

**Database Queries:**
- `desktop/backend-go/internal/database/queries/workspaces.sql` - Workspace CRUD
- SQLC generates type-safe Go code with workspace_id parameters

**Middleware:**
- `desktop/backend-go/internal/middleware/auth.go` - Session validation
- `desktop/backend-go/internal/middleware/redis_auth_test.go` - Tests

## Consequences

### Positive

1. **Simple Architecture:**
   - Single connection pool
   - Standard PostgreSQL queries
   - Easy to reason about and debug

2. **Excellent Performance:**
   - Efficient composite indexes: `(workspace_id, created_at)`
   - Query planner optimizations work well
   - Connection pooling overhead is minimal

3. **Easy Migrations:**
   - Single schema to update
   - Standard SQL migration tools (SQLC, Supabase)
   - No complex multi-database coordination

4. **Cost Efficient:**
   - Single PostgreSQL instance
   - Shared connection pool (pgxpool with 25 connections)
   - Reduced infrastructure overhead

5. **Flexible Memory Hierarchy:**
   - Three-tier visibility model
   - Granular sharing capabilities
   - User privacy controls

6. **Horizontal Scalability:**
   - Redis pub/sub for multi-instance coordination
   - Read replicas for query offloading
   - NATS JetStream for event streaming

### Negative

1. **Risk of Data Leaks:**
   - **Mitigation:** Comprehensive integration tests validate workspace isolation
   - **Testing:** `tests/security/access_control_test.go` with 90+ test cases
   - **Code Review:** All handlers reviewed for workspace_id filtering

2. **Query Complexity:**
   - Every query must include workspace_id
   - **Mitigation:** SQLC generates type-safe code, compile-time checks
   - **Convention:** `GetByWorkspace()` naming pattern enforced

3. **No Physical Isolation:**
   - Tenants share the same database instance
   - **Mitigation:** Encrypted at-rest storage (AES-256)
   - **Mitigation:** OAuth tokens encrypted with `TOKEN_ENCRYPTION_KEY`
   - **Mitigation:** Workspace audit logs track all access

4. **Backup Granularity:**
   - Cannot backup individual workspaces easily
   - **Mitigation:** Workspace version control system (semantic versioning)
   - **Mitigation:** Snapshot/restore API for workspace data export

5. **Resource Contention:**
   - One tenant's query load can affect others
   - **Mitigation:** Connection pooling limits per-request resources
   - **Mitigation:** Query timeouts prevent runaway queries
   - **Mitigation:** Rate limiting at API gateway level

### Monitoring

**Key Metrics:**
- Query performance by workspace_id (pg_stat_statements)
- Cross-workspace query attempts (audit logs)
- Memory access patterns (access_count, last_accessed_at)
- Failed authorization attempts (security_audit_test.go)

**Alerts:**
- Queries missing workspace_id filter
- Abnormal cross-workspace access patterns
- Failed workspace membership checks

## Related Decisions

- **ADR-002:** App Isolation Approach (Docker containers for user-generated apps)
- **CUS-25:** Memory Hierarchy System (task completed Q1 2026)
- **ISR-4:** File Browsing Endpoint (path traversal protection)

## References

- PostgreSQL Multi-Tenancy Best Practices: https://wiki.postgresql.org/wiki/Multi-tenancy
- Database Design Patterns: https://www.postgresql.org/docs/current/ddl-rowsecurity.html
- Migration 051: Memory Hierarchy: `desktop/backend-go/supabase/migrations/051_memory_hierarchy.sql`
- Security Audit (CUS-119): `desktop/backend-go/SECURITY_AUDIT_CUS-119.md`
