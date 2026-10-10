# MIOSA Platform: Supabase Migration Plan

**Status:** Proposed
**Author:** Roberto
**Date:** December 28, 2025
**Team:** Roberto, Pedro, Nick, Abdul, Nejd/Javaris

---

## TL;DR

We're moving from local/Cloud SQL PostgreSQL to **Supabase** to:
- Cut database costs by 98% ($1,219/mo → $25/mo)
- Get built-in auth, real-time, and storage for free
- Simplify infrastructure (one dashboard vs. many services)
- Keep using the same PostgreSQL queries we already have

**No backend rewrite required.** Our Go code stays the same.

---

## Why Supabase Over Alternatives

| Criteria | Supabase | Neon | Convex |
|----------|----------|------|--------|
| PostgreSQL compatible | Yes | Yes | No (proprietary) |
| Works with Go backend | Yes | Yes | No (TypeScript only) |
| Monthly cost (Pro) | $25 | $1,219+ | $25/dev |
| Built-in Auth | Yes | No | Yes |
| Built-in Storage | Yes | No | Yes |
| Real-time subscriptions | Yes | No | Yes |
| Migration effort | Low | Low | **Full rewrite** |

**Decision:** Supabase gives us the most value with the least disruption.

---

## What We Get

### Included in Supabase Pro ($25/month)

| Feature | Limit | Our Current Usage |
|---------|-------|-------------------|
| Database size | 8 GB | ~500 MB |
| Monthly active users | 100,000 | ~100 (beta) |
| File storage | 100 GB | ~1 GB |
| Egress | 250 GB | ~10 GB |
| Realtime connections | Unlimited | N/A (using SSE) |
| Edge functions | 500K invocations | N/A |

We're way under limits. Room to grow 100x before hitting caps.

### Free Extras We Can Use

1. **Supabase Auth** - Replace our custom auth or use alongside
2. **Storage** - File uploads without S3 setup
3. **Realtime** - WebSocket subscriptions (alternative to SSE)
4. **Edge Functions** - Deno functions at the edge (if needed later)
5. **Database Branching** - Preview environments for PRs

---

## Migration Strategy

### Phase 1: Setup (Day 1)

```
1. Create Supabase project (us-central1 for Cloud Run proximity)
2. Get connection strings
3. Update environment variables
4. Test connection from local dev
```

### Phase 2: Schema Migration (Day 1-2)

```
1. Export current schema: pg_dump --schema-only
2. Import to Supabase: psql < schema.sql
3. Verify tables, indexes, constraints
4. Run our existing migrations
```

### Phase 3: Data Migration (Day 2-3)

```
1. Export data: pg_dump --data-only
2. Import to Supabase
3. Verify row counts match
4. Test critical queries
```

### Phase 4: Backend Update (Day 3-4)

```
1. Update DATABASE_URL in Cloud Run
2. Update connection pooling settings
3. Deploy and test
4. Monitor for issues
```

### Phase 5: Cutover (Day 5)

```
1. Final data sync
2. Switch production traffic
3. Monitor dashboards
4. Keep old DB as backup for 1 week
```

---

## Environment Variables

### Current (Cloud SQL)
```env
DATABASE_URL=postgres://user:pass@/cloudsql/project:region:instance/dbname
```

### New (Supabase)
```env
# Direct connection (for migrations, admin)
DATABASE_URL=postgres://postgres.[project-ref]:[password]@aws-0-us-central1.pooler.supabase.com:5432/postgres

# Pooled connection (for app - use this in production)
DATABASE_URL=postgres://postgres.[project-ref]:[password]@aws-0-us-central1.pooler.supabase.com:6543/postgres?pgbouncer=true

# Supabase-specific (optional, for using their SDK)
SUPABASE_URL=https://[project-ref].supabase.co
SUPABASE_ANON_KEY=eyJ...
SUPABASE_SERVICE_KEY=eyJ...
```

### Connection Pooling

Supabase uses **Supavisor** (their PgBouncer replacement):
- Port `5432` = Direct connection (use for migrations)
- Port `6543` = Pooled connection (use for app)

Our Go backend should use the **pooled connection** (port 6543) with `?pgbouncer=true`.

---

## Code Changes Required

### Minimal Changes

Our Go backend uses standard `database/sql` with `lib/pq`. **No code changes needed** for basic queries.

### If Using Prepared Statements

Supabase pooling uses transaction mode. If we have issues:

```go
// Before: Prepared statements might fail with pooling
stmt, err := db.Prepare("SELECT * FROM users WHERE id = $1")

// After: Use query directly (works fine)
rows, err := db.Query("SELECT * FROM users WHERE id = $1", userID)
```

### Connection String Update

```go
// internal/database/connection.go

func Connect() (*sql.DB, error) {
    // Just update the DATABASE_URL env var - no code change
    connStr := os.Getenv("DATABASE_URL")
    return sql.Open("postgres", connStr)
}
```

---

## What We Keep vs. Replace

| Component | Keep | Replace | Notes |
|-----------|------|---------|-------|
| Go backend | Yes | - | No changes |
| PostgreSQL queries | Yes | - | Same SQL |
| Redis cache | Yes | - | Still useful for sessions |
| SSE streaming | Yes | - | Works independently |
| Custom auth | Maybe | Supabase Auth | Evaluate later |
| File uploads | - | Supabase Storage | Optional |
| Cloud Run | Yes | - | Just update env vars |

---

## Cost Projection

### Current Estimated Costs
```
Cloud SQL (if we scaled):     $200-500/month
Separate auth service:        $0-100/month
File storage (S3):            $10-50/month
-----------------------------------
Total:                        $210-650/month
```

### Supabase Costs
```
Pro plan:                     $25/month
Extra storage (if needed):    $0.125/GB
Extra MAU (if needed):        $0.00325/user
-----------------------------------
Total (projected):            $25-50/month
```

### Growth Path
```
Beta (now):      Free tier     $0/month
Launch:          Pro tier      $25/month
Scale (10K MAU): Pro tier      $50-75/month
Enterprise:      Team tier     $599/month
```

---

## Security Considerations

### Row Level Security (RLS)

Supabase encourages RLS policies. We can add them but don't have to:

```sql
-- Optional: Enable RLS on tables
ALTER TABLE tasks ENABLE ROW LEVEL SECURITY;

-- Optional: Policy for user isolation
CREATE POLICY "Users can only see their own tasks"
ON tasks FOR SELECT
USING (user_id = auth.uid());
```

**Our approach:** Keep authorization in Go backend for now. Add RLS later if we adopt Supabase Auth.

### Connection Security

- All connections are SSL by default
- IP allowlisting available on Team tier
- Service role key should never be exposed to clients

---

## Rollback Plan

If something goes wrong:

1. **Immediate:** Switch `DATABASE_URL` back to Cloud SQL
2. **Data:** Old database still running, no data loss
3. **Timeline:** Keep Cloud SQL running for 1 week after migration
4. **Cost:** ~$50 extra for the overlap period

---

## Team Responsibilities

| Person | Task |
|--------|------|
| **Roberto** | Create Supabase project, coordinate migration |
| **Pedro** | Verify backend queries work, test AgentV2 |
| **Nick** | Update Cloud Run env vars, deployment |
| **Abdul** | Test E2B integration with new DB |
| **Nejd** | Test frontend functionality |

---

## Timeline

```
Day 1:  Setup + Schema migration
Day 2:  Data migration + Testing
Day 3:  Backend updates + Staging deploy
Day 4:  Team testing + Bug fixes
Day 5:  Production cutover
Day 6+: Monitor + Decommission old DB
```

**Total: 1 week**

---

## Decision Checklist

Before we proceed, confirm:

- [ ] Team agrees on Supabase choice
- [ ] Supabase project created (who has access?)
- [ ] Current database backed up
- [ ] Staging environment ready for testing
- [ ] Rollback plan understood by team

---

## Next Steps

1. **Roberto:** Create Supabase project, share credentials securely
2. **Pedro:** Review this doc, flag any backend concerns
3. **Team:** Async approval in Slack
4. **Schedule:** Pick migration date (suggest: next Monday)

---

## Questions?

Drop in #miosa-backend or ping Roberto directly.

---

**Approved by:**
- [ ] Roberto
- [ ] Pedro
- [ ] Nick
