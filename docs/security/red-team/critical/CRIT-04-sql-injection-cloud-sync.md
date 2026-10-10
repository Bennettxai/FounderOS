# CRIT-4: SQL Injection via Cloud Sync Column Name Injection

**Severity:** CRITICAL | **Exploitability:** 8/10 | **CVSS:** 9.1

## Location
`desktop/backend-go/internal/handlers/cloud_sync.go:490-511` (`buildInsertParts`, `buildSetParts`)

## Description
The `SyncChange.Data` field is a `json.RawMessage` from the client. Its keys become column names embedded directly into SQL strings without sanitization.

## PoC
```json
POST /api/sync/push
{
  "table": "projects",
  "record_id": "any-uuid",
  "action": "update",
  "data": {
    "role = 'admin', name": "anything"
  }
}
```
Generated SQL: `UPDATE projects SET role = 'admin', name = $1 WHERE id = $2`

An attacker writes to arbitrary columns in any allowlisted table — including ownership fields, status flags, and foreign keys.

## Fix
```go
var allowedColumns = map[string]map[string]bool{
    "projects": {"name": true, "description": true, "status": true},
    "tasks":    {"title": true, "description": true, "status": true},
}

for col := range data {
    if !allowedColumns[table][col] {
        return fmt.Errorf("column %q not permitted for table %q", col, table)
    }
}
```
