# Agent Dashboard Tool - Implementation Notes

## Overview

The `configure_dashboard` tool allows AI agents to create and manage user dashboards programmatically through natural language requests.

---

## Architecture

### Component Hierarchy

```
User Request ("Create a sales dashboard")
        ↓
    Chat Handler (chat_v2.go)
        ↓
    Agent (BaseAgentV2)
        ↓
    Tool Registry (agent_tools.go)
        ↓
    ConfigureDashboardTool (dashboard_tool.go)
        ↓
    SQLC Queries (dashboards.sql.go)
        ↓
    PostgreSQL Database
```

### Key Files

| File | Purpose |
|------|---------|
| `internal/tools/dashboard_tool.go` | Tool implementation with 10 actions |
| `internal/tools/agent_tools.go` | Tool registry - registers `configure_dashboard` |
| `internal/agents/base_agent_v2.go` | Base agent with skills prompt injection |
| `internal/handlers/chat_v2.go` | Chat endpoint - injects skills context |
| `internal/services/skills_loader.go` | Loads skill definitions from YAML |

---

## Tool Actions

The `configure_dashboard` tool supports 10 actions:

| Action | Description | Required Params |
|--------|-------------|-----------------|
| `create_dashboard` | Create new dashboard | `name` |
| `add_widget` | Add widget to dashboard | `dashboard_id`, `widget_type`, `title` |
| `remove_widget` | Remove widget | `dashboard_id`, `widget_id` |
| `reorder_widgets` | Change widget order | `dashboard_id`, `widget_ids` (array) |
| `update_widget_config` | Modify widget settings | `dashboard_id`, `widget_id`, `config` |
| `list_dashboards` | Get all user dashboards | (none) |
| `get_dashboard` | Get specific dashboard | `dashboard_id` |
| `delete_dashboard` | Remove dashboard | `dashboard_id` |
| `set_default` | Make dashboard default | `dashboard_id` |
| `clone_dashboard` | Duplicate dashboard | `dashboard_id`, `new_name` |

---

## How Users Interact

### Current: Natural Language via Any Agent

Users simply describe what they want in chat:

```
User: "Create a new dashboard called Sales Overview with a revenue chart"
Agent: [Calls configure_dashboard with action=create_dashboard]
Agent: [Calls configure_dashboard with action=add_widget]
Agent: "I've created your Sales Overview dashboard with a revenue chart widget."
```

### No Special @mention Required

The tool is available to any agent through the tool registry. The agent decides when to use it based on user intent.

### Future Options

1. **Dedicated Dashboard Agent** - `@dashboard create sales overview`
2. **Slash Commands** - `/dashboard create "name"`
3. **Dashboard Builder UI** - Visual drag-and-drop with AI assist

---

## System Prompt Injection

Skills context is injected into agent system prompts:

```go
// Order of prompt layers in buildSystemPromptWithThinking():
1. Role context (permissions)
2. Focus mode prompt
3. Output style prompt  
4. Memory context
5. Skills context ← Dashboard tool info here
6. Base system prompt
7. Personalization
8. Thinking instructions
```

---

## Database Schema

### user_dashboards table
```sql
- id (UUID, PK)
- user_id (UUID, FK)
- name (TEXT)
- description (TEXT)
- layout (JSONB) - widget positions/sizes
- is_default (BOOLEAN)
- is_shared (BOOLEAN)
- share_token (TEXT)
- created_at, updated_at
```

### Dashboard Layout JSON Structure
```json
{
  "widgets": [
    {
      "id": "widget-uuid",
      "type": "chart",
      "title": "Revenue",
      "x": 0, "y": 0,
      "w": 6, "h": 4,
      "config": {
        "chartType": "line",
        "dataSource": "revenue_monthly"
      }
    }
  ]
}
```

---

## Widget Types

Common widget types the tool can create:

| Type | Use Case |
|------|----------|
| `chart` | Line, bar, pie charts |
| `kpi` | Single metric display |
| `table` | Data tables |
| `list` | Task/item lists |
| `calendar` | Event calendar |
| `activity` | Activity feed |
| `progress` | Progress bars |
| `metric_card` | Metric with trend |

---

## Widget Ideas 🧩

### Productivity & Time

| Widget | Description | Size |
|--------|-------------|------|
| Pomodoro Timer | Focus timer with session tracking | Small |
| Time Tracker | Active timer for current task/project | Small |
| Weekly Time Summary | Bar chart of hours per project | Medium |
| Calendar Glance | Next 3-5 meetings from calendar | Medium |
| Deadlines | Countdown to upcoming due dates | Small |

### Financial

| Widget | Description | Size |
|--------|-------------|------|
| Revenue Tracker | Monthly revenue vs goal | Small |
| Outstanding Invoices | Count + total unpaid | Small |
| Expenses This Month | Spending by category donut | Medium |
| Cash Flow | Simple in/out sparkline | Medium |
| Top Clients | Revenue by client breakdown | Large |

### Team & Communication

| Widget | Description | Size |
|--------|-------------|------|
| Team Availability | Who's online/busy/away | Small |
| Unread Messages | Chat/email count with preview | Small |
| Mentions | @mentions across workspace | Medium |
| Team Workload | Task distribution heatmap | Large |

### Goals & Progress

| Widget | Description | Size |
|--------|-------------|------|
| OKR Progress | Key results with progress bars | Medium |
| Streak Tracker | Daily habits/streaks | Small |
| Weekly Goals | Checkbox list for the week | Medium |
| Milestone Timeline | Visual project milestones | Large |

### AI & Insights

| Widget | Description | Size |
|--------|-------------|------|
| AI Suggestions | "You might want to..." prompts | Medium |
| Daily Briefing | AI-generated summary of the day | Medium |
| Risk Alerts | Projects/tasks flagged as at-risk | Small |

---

## Testing

### API Test (curl)

```bash
# 1. Sign in
curl -X POST "http://localhost:8001/api/auth/sign-in/email" \
  -H "Content-Type: application/json" \
  -d '{"email":"test@test.com","password":"password123"}' \
  -c cookies.txt

# 2. Create dashboard via chat
curl -X POST "http://localhost:8001/api/chat/v2/message" \
  -H "Content-Type: application/json" \
  -b cookies.txt \
  -d '{
    "message": "Create a dashboard called Test Dashboard",
    "agent": "general"
  }'
```

### Direct Tool Test

```bash
# List dashboards via REST API
curl -X GET "http://localhost:8001/api/user-dashboards" \
  -H "Content-Type: application/json" \
  -b cookies.txt
```

---

## Bug Fixes Made

### 1. Deadlock in skills_loader.go

**Problem:** `GetSkillsPromptXML()` called `GetEnabledSkills()` while holding `RLock()` - would deadlock.

**Fix:** Inlined the sorting logic instead of calling the method.

```go
// Before (DEADLOCK):
func (l *SkillsLoader) GetSkillsPromptXML() string {
    l.mu.RLock()
    defer l.mu.RUnlock()
    skills := l.GetEnabledSkills() // DEADLOCK - tries to acquire RLock again
}

// After (FIXED):
func (l *SkillsLoader) GetSkillsPromptXML() string {
    l.mu.RLock()
    defer l.mu.RUnlock()
    // Inline the logic instead of calling method
    var skills []Skill
    for _, s := range l.skills {
        if s.Enabled { skills = append(skills, s) }
    }
}
```

### 2. Type Mismatches in dashboard_tool.go

**Problem:** SQLC uses `pgtype.UUID` for ID fields but `string` for UserID.

**Fix:** Added helper functions:

```go
func uuidToPgtype(id string) pgtype.UUID { ... }
func uuidToString(id pgtype.UUID) string { ... }
```

---

## What's Next

### Short-term
- [ ] Test dashboard creation via chat
- [ ] Add dashboard skill definition to skills.yaml
- [ ] Create widget templates for common use cases

### Medium-term
- [ ] Dedicated DashboardAgent with specialized prompt
- [ ] Dashboard sharing between workspace members
- [ ] AI-suggested widgets based on user data

### Long-term
- [ ] Multi-agent dashboard building (orchestration)
- [ ] Real-time collaborative dashboards
- [ ] Dashboard templates marketplace

---

## Related Documentation

- [API Documentation](./API_DOCUMENTATION_INDEX.md)
- [Agent Skills Overview](./AGENT_SKILLS_OVERVIEW.md)
- [Custom Agents Guide](./CUSTOM_AGENTS_METAS.md)
