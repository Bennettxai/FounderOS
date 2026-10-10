# Changelog

All notable changes to BusinessOS. Versioning: `MAJOR.MINOR.PATCH` - patch bumps
per release from here (1.0.1 -> 1.0.2 -> ...), minor for feature waves, major
for breaking platform changes. Every release follows GO-LIVE-SOP.md and gets a
git tag `v<version>` on the release commit.

## v1.0.1 - 2026-07-03 (first public version)

The v1 line. Everything below is live: businessos.dev (web), Cloud Run backend,
signed desktop DMGs (arm64 + x64), and the team repo.

### Platform
- Workspace model end to end: clients, team, projects, tasks, personas,
  campaigns, content, offers, glossary all belong to one workspace
- Workspace-shared reads: active members see and work the workspace's CRM
  (clients + contacts + deals + interactions + board), team roster, and tasks
- Organizations -> workspaces -> teams hierarchy with invites that land members
  in the workspace they were invited to

### Modules
- Relationships (leads & clients, one record across the 14-stage pipeline)
  with client detail drawer + contacts
- Client Board: one composed surface per client (projects, tasks by status,
  team, deals, interactions)
- Boards: the composition layer - build boards from module views, filter to a
  client, pin to the sidebar
- Team, Projects (create + templates), Tasks, Calendar (Google auto-sync +
  connected indicator in Settings), Content pipeline (filters + stage moves),
  Offers, Personas, Campaigns, Glossary, Inbox, Finance, Pipelines,
  Deliverables, Knowledge (Notion-style, engine/cloud source tags)

### Knowledge & Optimal Engine
- Local-first engine knowledge with cloud sync (Sync to team), per-doc source
  tags (engine / synced / cloud), teammate pull endpoint

### Auth & Admin
- Google sign-in (web) live; email/password + workspace invites
- Platform Admin: overview, users, workspaces, organizations, per-user account
  connections, engine status + setup reminders

### Desktop
- Signed + notarized DMGs (arm64 + x64), desktop icon layer wired to all live
  modules, Boards + Team desktop icons

### Fixes of note
- Client deals now link to their client; JSON fields over the Supabase pooler
  (simple protocol) no longer 500; admin users list fixed; dead desktop icon
  routes mapped
