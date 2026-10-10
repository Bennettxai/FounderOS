---
name: dashboard-management
description: >
  Create and configure custom dashboards with widgets. Add task summaries,
  burndown charts, project progress, upcoming deadlines, and metric cards.
  Group tasks by status, project, or priority. Set default dashboards.
metadata:
  version: "1.0.0"
  author: businessos
  tools_used:
    - configure_dashboard
  depends_on: []
  context_hints:
    - "User's current dashboard layout"
    - "User's workspace membership"
    - "Available projects for filtering"
  telemetry:
    track_events:
      - widget_added
      - widget_removed
      - dashboard_created
      - error_recovered
---

# Dashboard Management

## When to Use This Skill

Activate when user wants to:
- Create a new dashboard
- Add widgets to see their data visually
- Remove or modify existing widgets
- Set a default dashboard
- Customize their existing dashboard

**Keywords:** dashboard, widget, add, show, display, view, chart, summary, home, burndown, metrics, deadlines

**Do NOT use when:**
- User wants to create/edit tasks → use `task-management`
- User asks "why" questions about data → use `analytics-insights`
- User wants to manage notifications → use `notification-management`

---

## The Tool

**Name:** `configure_dashboard`

**Actions:**

| Action | Purpose | Required Params |
|--------|---------|-----------------|
| `list_dashboards` | Get user's dashboards | none |
| `list_widgets` | Get available widget types | none |
| `get_dashboard` | Get specific dashboard layout | `dashboard_id` or `dashboard_name` (optional, uses default) |
| `create_dashboard` | Create new dashboard | `name` |
| `add_widget` | Add single widget | `widget_type` |
| `add_widgets` | Add multiple widgets | `widgets[]` |
| `remove_widget` | Remove a widget | `widget_id` (or infer from context) |
| `update_widget` | Modify widget config | `widget_id`, `config` |
| `set_default` | Set default dashboard | `dashboard_id` or `dashboard_name` |

---

## Request → Tool Mapping

| User Says | Action | Key Params |
|-----------|--------|------------|
| "Show me my tasks" | `add_widget` | `widget_type: task_summary` |
| "Tasks grouped by project" | `add_widget` | `widget_type: task_summary, config: {group_by: "project"}` |
| "Tasks grouped by priority" | `add_widget` | `widget_type: task_summary, config: {group_by: "priority"}` |
| "What's due this week" | `add_widget` | `widget_type: upcoming_deadlines, config: {days_ahead: 7}` |
| "Show deadlines for next 2 weeks" | `add_widget` | `widget_type: upcoming_deadlines, config: {days_ahead: 14}` |
| "Add a burndown chart" | `add_widget` | `widget_type: task_burndown` |
| "Show project progress" | `add_widget` | `widget_type: project_progress` |
| "Create a new dashboard" | `create_dashboard` | `name: (ask user or infer)` |
| "Create a sales dashboard" | `create_dashboard` | `name: "Sales Dashboard"` |
| "Add tasks and deadlines" | `add_widgets` | `widgets: [{type: task_summary}, {type: upcoming_deadlines}]` |
| "Remove the burndown chart" | `remove_widget` | identify by type |
| "What dashboards do I have" | `list_dashboards` | none |
| "What widgets can I add" | `list_widgets` | none |
| "Make this my default" | `set_default` | current dashboard context |

---

## Widget Quick Reference

| Type | Best For | Common Config |
|------|----------|---------------|
| `task_summary` | Workload overview | `group_by`: status/project/priority |
| `task_list` | Detailed task view | `limit`, `filter`, `show_completed` |
| `upcoming_deadlines` | Planning ahead | `days_ahead`: 7, 14, 30 |
| `task_burndown` | Sprint tracking | `days`: 30, 90; `project_id` |
| `project_progress` | Project health | `project_id` (optional) |
| `metric_card` | Single KPI | `metric`: tasks_due_today, etc. |
| `recent_activity` | Activity feed | `limit`: 10, 20, 50 |
| `workload_heatmap` | Capacity planning | `date_range`: month, quarter |
| `client_overview` | Sales pipeline | `pipeline_stage` |
| `notes_pinned` | Quick reference | `limit` |
| `quick_actions` | Shortcuts | `actions[]` |
| `agent_shortcuts` | AI commands | `shortcuts[]` |

---

## Examples

### Simple: Add one widget

**User:** "I want to see my tasks on the dashboard"

**Tool Call:**
```json
{
  "action": "add_widget",
  "widget_type": "task_summary"
}
```

**Response:** "Added a task summary widget to your dashboard. It shows your tasks grouped by status."

---

### With Config: Grouped tasks

**User:** "Show me tasks grouped by project"

**Tool Call:**
```json
{
  "action": "add_widget",
  "widget_type": "task_summary",
  "config": {
    "group_by": "project"
  }
}
```

**Response:** "Added a task summary grouped by project to your dashboard."

---

### Multi-Widget: Dashboard setup

**User:** "Set up my dashboard with tasks, deadlines, and a burndown chart"

**Tool Call:**
```json
{
  "action": "add_widgets",
  "widgets": [
    {"type": "task_summary", "config": {"group_by": "status"}},
    {"type": "upcoming_deadlines", "config": {"days_ahead": 14}},
    {"type": "task_burndown", "config": {"days": 30}}
  ]
}
```

**Response:** "Added 3 widgets to your dashboard: task summary, 2-week deadlines, and 30-day burndown chart."

---

### Create Dashboard: New dashboard with widgets

**User:** "Create a project dashboard for Alpha with progress and tasks"

**Tool Call:**
```json
{
  "action": "create_dashboard",
  "name": "Alpha Project",
  "widgets": [
    {"type": "project_progress", "config": {"project_id": "alpha-uuid"}},
    {"type": "task_list", "config": {"project_id": "alpha-uuid", "limit": 20}}
  ]
}
```

**Response:** "Created 'Alpha Project' dashboard with progress tracking and task list."

---

### Remove Widget

**User:** "Remove the deadlines widget"

**Approach:**
1. If only one `upcoming_deadlines` widget exists → remove it
2. If multiple exist → ask which one
3. If none exist → inform user

**Tool Call (if unambiguous):**
```json
{
  "action": "remove_widget",
  "widget_id": "widget-uuid-here"
}
```

**Response:** "Removed the upcoming deadlines widget from your dashboard."

---

## Error Handling

### Widget Type Not Found

**Tool Returns:**
```json
{
  "success": false,
  "error": "widget_type_not_found",
  "requested": "team_capacity",
  "available_alternatives": ["workload_heatmap", "task_summary"]
}
```

**Say:** "Team capacity widget isn't available yet. I can add a workload heatmap that shows task density over time, or a task summary. Which would you prefer?"

---

### Dashboard Full

**Tool Returns:**
```json
{
  "success": false,
  "error": "dashboard_full",
  "current_count": 12,
  "max_count": 12
}
```

**Say:** "Your dashboard already has 12 widgets (the maximum). I can remove one to make room, or create a new dashboard. What would you prefer?"

---

### Ambiguous Widget Reference

When user says "remove the task widget" but multiple task widgets exist:

**Say:** "I found several task widgets on your dashboard:
- Task Summary (grouped by status)
- Task List (showing 20 items)
- Task Burndown (30-day chart)

Which one would you like to remove?"

---

## Additional Resources

If you need more details, these references are available:
- `WIDGETS.md` - Full widget catalog with visuals and all config options
- `CONFIGS.md` - Detailed configuration parameter documentation
- `ERRORS.md` - Complete error codes and recovery strategies
- `EXAMPLES.md` - 20+ example conversations

Request a reference when:
- User asks about a widget you're unsure about
- You encounter an error code not listed above
- User has a complex multi-step request

---

## Limitations

- **Cannot delete dashboards** - Only remove widgets (safety measure)
- **Cannot reposition widgets via chat** - Users drag-drop in UI
- **Max 12 widgets per dashboard** - Suggest new dashboard if full
- **Widget data is read-only** - Widgets display data, don't modify it
- **Workspace-scoped data** - Widgets only show data user has access to
