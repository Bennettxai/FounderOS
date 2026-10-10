// Package notion provides the Notion integration.
// Database operations are split across:
//   - databases_types.go   — types and result structs
//   - databases_helpers.go — DatabaseService, save helpers, title extraction
//   - databases_sync.go    — SyncDatabases, SyncPages
//   - databases_crud.go    — GetDatabases, GetPages, GetPage, CreatePage, UpdatePage, GetToken
//   - databases_query.go   — ListDatabases, GetDatabase, QueryDatabase, Search
package notion
