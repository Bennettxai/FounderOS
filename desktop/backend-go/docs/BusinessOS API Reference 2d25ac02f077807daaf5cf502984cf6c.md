# BusinessOS API Reference

Owner: Nicholas Del Negro

# Complete REST API documentation for the BusinessOS Go backend.

**Base URL:** `http://localhost:8001/api`

**Authentication:** Session cookie-based via Better Auth. All protected routes require valid session.

---

## Authentication

| Method | Endpoint | Auth | Description |
| --- | --- | --- | --- |
| POST | `/auth/sign-up/email` | No | Register with email/password |
| POST | `/auth/sign-in/email` | No | Login with email/password |
| GET | `/auth/google` | No | Initiate Google OAuth |
| GET | `/auth/google/callback/login` | No | Google OAuth callback |
| GET | `/auth/session` | No | Get current session |
| GET | `/auth/get-session` | No | Get session (Better Auth alias) |
| POST | `/auth/logout` | No | End session |
| POST | `/auth/sign-out` | No | End session (Better Auth alias) |

---

## Terminal (WebSocket + PTY)

| Method | Endpoint | Auth | Description |
| --- | --- | --- | --- |
| GET | `/terminal/ws` | Yes | WebSocket connection for PTY |
| GET | `/terminal/sessions` | Yes | List active sessions |
| DELETE | `/terminal/sessions/:id` | Yes | Close specific session |

### WebSocket Parameters

| Parameter | Type | Default | Description |
| --- | --- | --- | --- |
| cols | int | 80 | Terminal columns |
| rows | int | 24 | Terminal rows |
| shell | string | zsh | Shell to spawn |
| cwd | string | ~ | Working directory |

### Security Features

| Feature | Implementation |
| --- | --- |
| Rate Limiting | 100 msg/sec, 20 burst, 5 connections/user |
| Input Sanitization | 28+ dangerous patterns blocked |
| Session Security | IP binding, 8h max + 30min idle timeout |
| Container Isolation | Read-only root, Seccomp profile, dropped capabilities |

---

## Chat and AI

| Method | Endpoint | Auth | Description |
| --- | --- | --- | --- |
| GET | `/chat/conversations` | Yes | List conversations |
| POST | `/chat/conversations` | Yes | Create conversation |
| GET | `/chat/conversations/:id` | Yes | Get conversation |
| PUT | `/chat/conversations/:id` | Yes | Update conversation |
| DELETE | `/chat/conversations/:id` | Yes | Delete conversation |
| POST | `/chat/message` | Yes | Send message |
| GET | `/chat/search` | Yes | Search conversations |
| POST | `/chat/ai/document` | Yes | Document with AI |
| POST | `/chat/ai/analyze` | Yes | Analyze content |
| POST | `/chat/ai/extract-tasks` | Yes | Extract tasks from text |
| POST | `/chat/ai/plan` | Yes | Create AI plan |

---

## AI Configuration

| Method | Endpoint | Auth | Description |
| --- | --- | --- | --- |
| GET | `/ai/providers` | Yes | List LLM providers |
| GET | `/ai/models` | Yes | Get all models |
| GET | `/ai/models/local` | Yes | Get local (Ollama) models |
| POST | `/ai/models/pull` | Yes | Pull model from Ollama |
| POST | `/ai/models/warmup` | Yes | Warmup model |
| GET | `/ai/system` | Yes | System info |
| POST | `/ai/api-key` | Yes | Save API key |
| PUT | `/ai/provider` | Yes | Update AI provider |
| GET | `/ai/agents` | Yes | List agent prompts |
| GET | `/ai/agents/:id` | Yes | Get agent prompt |
| GET | `/ai/commands` | Yes | List slash commands |
| POST | `/ai/commands` | Yes | Create custom command |
| GET | `/ai/commands/:id` | Yes | Get command |
| PUT | `/ai/commands/:id` | Yes | Update command |
| DELETE | `/ai/commands/:id` | Yes | Delete command |

---

## Projects

| Method | Endpoint | Auth | Description |
| --- | --- | --- | --- |
| GET | `/projects` | Yes | List projects |
| POST | `/projects` | Yes | Create project |
| GET | `/projects/:id` | Yes | Get project |
| PUT | `/projects/:id` | Yes | Update project |
| DELETE | `/projects/:id` | Yes | Delete project |
| POST | `/projects/:id/notes` | Yes | Add project note |

---

## Clients (CRM)

| Method | Endpoint | Auth | Description |
| --- | --- | --- | --- |
| GET | `/clients` | Yes | List clients |
| POST | `/clients` | Yes | Create client |
| GET | `/clients/:id` | Yes | Get client |
| PUT | `/clients/:id` | Yes | Update client |
| PATCH | `/clients/:id/status` | Yes | Update client status |
| DELETE | `/clients/:id` | Yes | Delete client |

### Contacts

| Method | Endpoint | Auth | Description |
| --- | --- | --- | --- |
| GET | `/clients/:id/contacts` | Yes | List contacts |
| POST | `/clients/:id/contacts` | Yes | Create contact |
| PUT | `/clients/:id/contacts/:contactId` | Yes | Update contact |
| DELETE | `/clients/:id/contacts/:contactId` | Yes | Delete contact |

### Interactions

| Method | Endpoint | Auth | Description |
| --- | --- | --- | --- |
| GET | `/clients/:id/interactions` | Yes | List interactions |
| POST | `/clients/:id/interactions` | Yes | Create interaction |

### Deals (Client-scoped)

| Method | Endpoint | Auth | Description |
| --- | --- | --- | --- |
| GET | `/clients/:id/deals` | Yes | List client deals |
| POST | `/clients/:id/deals` | Yes | Create deal |
| PUT | `/clients/:id/deals/:dealId` | Yes | Update deal |

---

## Deals (Global)

| Method | Endpoint | Auth | Description |
| --- | --- | --- | --- |
| GET | `/deals` | Yes | List all deals |
| PATCH | `/deals/:id/stage` | Yes | Update deal stage |

---

## Dashboard

| Method | Endpoint | Auth | Description |
| --- | --- | --- | --- |
| GET | `/dashboard/summary` | Yes | Get dashboard summary |
| GET | `/dashboard/focus` | Yes | List focus items |
| POST | `/dashboard/focus` | Yes | Create focus item |
| PUT | `/dashboard/focus/:id` | Yes | Update focus item |
| DELETE | `/dashboard/focus/:id` | Yes | Delete focus item |
| GET | `/dashboard/tasks` | Yes | List tasks |
| POST | `/dashboard/tasks` | Yes | Create task |
| PUT | `/dashboard/tasks/:id` | Yes | Update task |
| POST | `/dashboard/tasks/:id/toggle` | Yes | Toggle task completion |
| DELETE | `/dashboard/tasks/:id` | Yes | Delete task |

---

## Team

| Method | Endpoint | Auth | Description |
| --- | --- | --- | --- |
| GET | `/team` | Yes | List team members |
| POST | `/team` | Yes | Create team member |
| GET | `/team/:id` | Yes | Get team member |
| PUT | `/team/:id` | Yes | Update team member |
| PATCH | `/team/:id/status` | Yes | Update status |
| PATCH | `/team/:id/capacity` | Yes | Update capacity |
| POST | `/team/:id/activity` | Yes | Add activity |
| DELETE | `/team/:id` | Yes | Delete team member |

---

## Nodes (Business Structure)

| Method | Endpoint | Auth | Description |
| --- | --- | --- | --- |
| GET | `/nodes` | Yes | List nodes |
| GET | `/nodes/tree` | Yes | Get full tree |
| GET | `/nodes/active` | Yes | Get active node |
| POST | `/nodes` | Yes | Create node |
| GET | `/nodes/:id` | Yes | Get node |
| PATCH | `/nodes/:id` | Yes | Update node |
| POST | `/nodes/:id/activate` | Yes | Activate node |
| POST | `/nodes/:id/deactivate` | Yes | Deactivate node |
| DELETE | `/nodes/:id` | Yes | Delete node |
| GET | `/nodes/:id/children` | Yes | Get children |
| POST | `/nodes/:id/reorder` | Yes | Reorder nodes |

---

## Contexts (Documents)

| Method | Endpoint | Auth | Description |
| --- | --- | --- | --- |
| GET | `/contexts/public/:shareId` | No | Get public context |
| GET | `/contexts` | Yes | List contexts |
| POST | `/contexts` | Yes | Create context |
| GET | `/contexts/:id` | Yes | Get context |
| PUT | `/contexts/:id` | Yes | Update context |
| PATCH | `/contexts/:id/blocks` | Yes | Update blocks |
| POST | `/contexts/:id/share` | Yes | Share context |
| DELETE | `/contexts/:id/share` | Yes | Unshare context |
| POST | `/contexts/:id/duplicate` | Yes | Duplicate context |
| PATCH | `/contexts/:id/archive` | Yes | Archive context |
| PATCH | `/contexts/:id/unarchive` | Yes | Unarchive context |
| DELETE | `/contexts/:id` | Yes | Delete context |
| POST | `/contexts/aggregate` | Yes | Aggregate contexts |

---

## Artifacts

| Method | Endpoint | Auth | Description |
| --- | --- | --- | --- |
| GET | `/artifacts` | Yes | List artifacts |
| POST | `/artifacts` | Yes | Create artifact |
| GET | `/artifacts/:id` | Yes | Get artifact |
| PATCH | `/artifacts/:id` | Yes | Update artifact |
| PATCH | `/artifacts/:id/link` | Yes | Link artifact |
| DELETE | `/artifacts/:id` | Yes | Delete artifact |

---

## Daily Logs

| Method | Endpoint | Auth | Description |
| --- | --- | --- | --- |
| GET | `/daily/logs` | Yes | List logs |
| GET | `/daily/logs/today` | Yes | Get today's log |
| GET | `/daily/logs/:date` | Yes | Get log by date |
| POST | `/daily/logs` | Yes | Create/update log |
| PUT | `/daily/logs/:id` | Yes | Update log |
| DELETE | `/daily/logs/:id` | Yes | Delete log |

---

## Calendar

| Method | Endpoint | Auth | Description |
| --- | --- | --- | --- |
| GET | `/calendar/events` | Yes | List events |
| GET | `/calendar/events/:id` | Yes | Get event |
| POST | `/calendar/events` | Yes | Create event |
| PUT | `/calendar/events/:id` | Yes | Update event |
| DELETE | `/calendar/events/:id` | Yes | Delete event |
| POST | `/calendar/sync` | Yes | Sync with Google |
| GET | `/calendar/today` | Yes | Today's events |
| GET | `/calendar/upcoming` | Yes | Upcoming events |

---

## Google Integration

| Method | Endpoint | Auth | Description |
| --- | --- | --- | --- |
| GET | `/integrations/google/auth` | Yes | Initiate Google auth |
| GET | `/integrations/google/callback` | No | OAuth callback |
| GET | `/integrations/google/status` | Yes | Connection status |
| DELETE | `/integrations/google` | Yes | Disconnect Google |

---

## Filesystem

| Method | Endpoint | Auth | Description |
| --- | --- | --- | --- |
| GET | `/filesystem/list` | Yes | List directory |
| GET | `/filesystem/read` | Yes | Read file |
| GET | `/filesystem/download` | Yes | Download file |
| GET | `/filesystem/info` | Yes | Get file info |
| GET | `/filesystem/quick-access` | Yes | Quick access paths |
| POST | `/filesystem/mkdir` | Yes | Create directory |
| POST | `/filesystem/upload` | Yes | Upload file |
| DELETE | `/filesystem/delete` | Yes | Delete file/dir |

---

## Usage Analytics

| Method | Endpoint | Auth | Description |
| --- | --- | --- | --- |
| GET | `/usage/summary` | Yes | Usage summary |
| GET | `/usage/providers` | Yes | Usage by provider |
| GET | `/usage/models` | Yes | Usage by model |
| GET | `/usage/agents` | Yes | Usage by agent |
| GET | `/usage/trend` | Yes | Usage trend |
| GET | `/usage/recent` | Yes | Recent AI usage |
| GET | `/usage/mcp` | Yes | MCP usage |

---

## MCP (Model Context Protocol)

| Method | Endpoint | Auth | Description |
| --- | --- | --- | --- |
| GET | `/mcp/tools` | Yes | List MCP tools |
| POST | `/mcp/execute` | Yes | Execute MCP tool |
| GET | `/mcp/health` | Yes | MCP health check |

---

## Transcription

| Method | Endpoint | Auth | Description |
| --- | --- | --- | --- |
| POST | `/transcribe` | Yes | Transcribe audio |
| GET | `/transcribe/status` | Yes | Transcription status |

---

## Voice Notes

| Method | Endpoint | Auth | Description |
| --- | --- | --- | --- |
| GET | `/voice-notes` | Yes | List voice notes |
| POST | `/voice-notes` | Yes | Upload voice note |
| GET | `/voice-notes/stats` | Yes | Voice note stats |
| GET | `/voice-notes/:id` | Yes | Get voice note |
| DELETE | `/voice-notes/:id` | Yes | Delete voice note |
| POST | `/voice-notes/:id/retranscribe` | Yes | Re-transcribe |

---

## Profile

| Method | Endpoint | Auth | Description |
| --- | --- | --- | --- |
| PUT | `/profile` | Yes | Update profile |
| POST | `/profile/photo` | Yes | Upload photo |
| DELETE | `/profile/photo` | Yes | Delete photo |
| POST | `/profile/background` | Yes | Upload background |
| GET | `/profile/background/:filename` | Yes | Get background |
| DELETE | `/profile/background` | Yes | Delete background |

---

## Settings

| Method | Endpoint | Auth | Description |
| --- | --- | --- | --- |
| GET | `/settings` | Yes | Get settings |
| PUT | `/settings` | Yes | Update settings |
| GET | `/settings/system` | Yes | Get system settings |

---

## Summary

| Domain | Endpoint Count |
| --- | --- |
| Authentication | 8 |
| Terminal | 3 |
| Chat/AI | 12 |
| AI Config | 15 |
| Projects | 6 |
| Clients | 16 |
| Deals | 2 |
| Dashboard | 10 |
| Team | 8 |
| Nodes | 11 |
| Contexts | 13 |
| Artifacts | 6 |
| Daily Logs | 6 |
| Calendar | 8 |
| Google Integration | 4 |
| Filesystem | 8 |
| Usage | 7 |
| MCP | 3 |
| Transcription | 2 |
| Voice Notes | 6 |
| Profile | 6 |
| Settings | 3 |
| **Total** | **~145** |