# Extended Examples

20+ conversation examples for dashboard management scenarios.

---

## Basic Widget Operations

### Example 1: Simple Widget Add

**User:** "Add a task widget to my dashboard"

**Analysis:**
- Intent: Add widget
- Widget type: task-related → default to `task_summary`
- Config: None specified → use defaults

**Tool Call:**
```json
{
  "action": "add_widget",
  "widget_type": "task_summary"
}
```

**Response:** "Added a task summary to your dashboard. It shows your tasks grouped by status (To Do, In Progress, Done, etc.)."

---

### Example 2: Widget with Specific Config

**User:** "I want to see my tasks organized by priority"

**Analysis:**
- Intent: Add widget (or update existing)
- Widget type: `task_summary`
- Config: `group_by: "priority"`

**Tool Call:**
```json
{
  "action": "add_widget",
  "widget_type": "task_summary",
  "config": {
    "group_by": "priority"
  }
}
```

**Response:** "Added a task summary grouped by priority. You'll see tasks organized as High, Medium, Low, and None."

---

### Example 3: Deadline Widget with Time Range

**User:** "Show me what's due in the next 2 weeks"

**Analysis:**
- Intent: See upcoming deadlines
- Widget type: `upcoming_deadlines`
- Config: `days_ahead: 14`

**Tool Call:**
```json
{
  "action": "add_widget",
  "widget_type": "upcoming_deadlines",
  "config": {
    "days_ahead": 14
  }
}
```

**Response:** "Added a deadlines widget showing everything due in the next 14 days, including any overdue items."

---

### Example 4: Burndown Chart for Project

**User:** "Add a burndown chart for the website project"

**Analysis:**
- Intent: Add burndown
- Widget type: `task_burndown`
- Config: Need to resolve "website project" to project_id

**Pre-step:** Look up project ID for "website project"

**Tool Call:**
```json
{
  "action": "add_widget",
  "widget_type": "task_burndown",
  "config": {
    "project_id": "uuid-of-website-project",
    "days": 30
  }
}
```

**Response:** "Added a 30-day burndown chart for the Website project. It shows task completion progress with an ideal pace line."

---

### Example 5: KPI Metric Card

**User:** "I want to see how many tasks I've completed this week"

**Analysis:**
- Intent: Single metric display
- Widget type: `metric_card`
- Config: `metric: "tasks_completed_week"`

**Tool Call:**
```json
{
  "action": "add_widget",
  "widget_type": "metric_card",
  "config": {
    "metric": "tasks_completed_week",
    "show_trend": true
  }
}
```

**Response:** "Added a metric card showing tasks completed this week, with a comparison to last week."

---

## Multi-Widget Operations

### Example 6: Two Related Widgets

**User:** "Add tasks and deadlines to my dashboard"

**Analysis:**
- Intent: Add multiple widgets
- Widgets: `task_summary` and `upcoming_deadlines`

**Tool Call:**
```json
{
  "action": "add_widgets",
  "widgets": [
    {"type": "task_summary"},
    {"type": "upcoming_deadlines"}
  ]
}
```

**Response:** "Added 2 widgets: a task summary and upcoming deadlines. They're placed side by side on your dashboard."

---

### Example 7: Full Dashboard Setup

**User:** "Set up my dashboard with everything I need for daily work - tasks, deadlines, and a quick way to see what's overdue"

**Analysis:**
- Intent: Comprehensive setup
- Widgets: task_summary, upcoming_deadlines, metric_card (overdue)

**Tool Call:**
```json
{
  "action": "add_widgets",
  "widgets": [
    {"type": "task_summary", "config": {"group_by": "status"}},
    {"type": "upcoming_deadlines", "config": {"days_ahead": 7}},
    {"type": "metric_card", "config": {"metric": "tasks_overdue"}}
  ]
}
```

**Response:** "Set up your dashboard with:
- **Task Summary** - Your tasks by status
- **Upcoming Deadlines** - Next 7 days
- **Overdue Counter** - Quick view of overdue items

You're all set for daily planning!"

---

### Example 8: Project-Focused Dashboard

**User:** "Create a dashboard for tracking the Alpha project with progress and tasks"

**Analysis:**
- Intent: Create new dashboard + add widgets
- Dashboard name: "Alpha Project" (inferred)
- Widgets: project_progress, task_list (filtered to project)

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

**Response:** "Created 'Alpha Project' dashboard with project progress and task list. Both are filtered to show only Alpha project data."

---

## Dashboard Management

### Example 9: List Dashboards

**User:** "What dashboards do I have?"

**Tool Call:**
```json
{
  "action": "list_dashboards"
}
```

**Tool Response:**
```json
{
  "success": true,
  "dashboards": [
    {"id": "d1", "name": "My Day", "is_default": true, "widget_count": 4},
    {"id": "d2", "name": "Project Overview", "is_default": false, "widget_count": 6}
  ]
}
```

**Response:** "You have 2 dashboards:
- **My Day** (default) - 4 widgets
- **Project Overview** - 6 widgets

Would you like to see what's on either one?"

---

### Example 10: Create New Dashboard

**User:** "Create a new dashboard called Weekly Review"

**Tool Call:**
```json
{
  "action": "create_dashboard",
  "name": "Weekly Review"
}
```

**Response:** "Created 'Weekly Review' dashboard. It's empty right now - what widgets would you like to add?"

---

### Example 11: Set Default Dashboard

**User:** "Make Project Overview my default dashboard"

**Tool Call:**
```json
{
  "action": "set_default",
  "dashboard_name": "Project Overview"
}
```

**Response:** "Done! 'Project Overview' is now your default dashboard. It'll be the first thing you see when you log in."

---

## Widget Modification

### Example 12: Update Widget Config

**User:** "Change my task summary to group by project instead"

**Analysis:**
- Find task_summary widget
- Update its group_by config

**Tool Call:**
```json
{
  "action": "update_widget",
  "widget_id": "w1",
  "config": {
    "group_by": "project"
  }
}
```

**Response:** "Updated your task summary to group by project."

---

### Example 13: Remove Widget

**User:** "Remove the burndown chart"

**Analysis:**
- Find task_burndown widget
- Remove it

**Tool Call:**
```json
{
  "action": "remove_widget",
  "widget_id": "w3"
}
```

**Response:** "Removed the burndown chart from your dashboard."

---

### Example 14: Ambiguous Remove Request

**User:** "Remove the task widget"

**Analysis:**
- Multiple task widgets might exist
- Need to clarify

**Check current widgets first, then respond:**

"I found a few task-related widgets:
- **Task Summary** (grouped by status)
- **Task List** (showing 15 items)

Which one would you like to remove?"

---

## Error Handling Examples

### Example 15: Unknown Widget Type

**User:** "Add a team velocity widget"

**Tool Response:**
```json
{
  "success": false,
  "error": "widget_type_not_found",
  "requested": "team_velocity",
  "available_alternatives": ["task_burndown", "workload_heatmap"]
}
```

**Response:** "Team velocity widget isn't available yet. For tracking pace over time, I can add:
- **Task Burndown** - Shows completion progress over time
- **Workload Heatmap** - Shows task density by day

Which would work for you?"

---

### Example 16: Dashboard Full

**User:** "Add a client overview widget"

**Tool Response:**
```json
{
  "success": false,
  "error": "dashboard_full",
  "current_count": 12,
  "max_count": 12
}
```

**Response:** "Your dashboard is full (12 widgets max). I can:
1. **Remove a widget** to make room - which one can go?
2. **Create a new dashboard** for additional widgets

What would you prefer?"

---

### Example 17: Project Not Found

**User:** "Show tasks for the Beta project"

**Tool Response:**
```json
{
  "success": false,
  "error": "project_not_found",
  "available_projects": [
    {"id": "p1", "name": "Alpha"},
    {"id": "p2", "name": "Website Redesign"}
  ]
}
```

**Response:** "I couldn't find a 'Beta' project. Your active projects are:
- Alpha
- Website Redesign

Did you mean one of these, or would you like to create a Beta project?"

---

## Complex Scenarios

### Example 18: Dashboard for Specific Role

**User:** "Set up a dashboard for managing clients"

**Analysis:**
- Sales/account management focus
- Widgets: client_overview, recent_activity, metric_card

**Tool Call:**
```json
{
  "action": "create_dashboard",
  "name": "Client Management",
  "widgets": [
    {"type": "client_overview"},
    {"type": "recent_activity", "config": {"filter": "all"}},
    {"type": "metric_card", "config": {"metric": "active_projects"}},
    {"type": "task_list", "config": {"filter": "today", "limit": 10}}
  ]
}
```

**Response:** "Created 'Client Management' dashboard with:
- Client pipeline overview
- Recent activity feed
- Active projects count
- Today's tasks

This should give you a good view for managing client work!"

---

### Example 19: Replace Widget

**User:** "Replace the task summary with a task list"

**Analysis:**
- Remove task_summary
- Add task_list

**Tool Calls (or single batch):**
```json
{
  "action": "remove_widget",
  "widget_type": "task_summary"
}
```
then:
```json
{
  "action": "add_widget",
  "widget_type": "task_list"
}
```

**Response:** "Replaced the task summary with a task list. You can now see individual tasks and click them to open details."

---

### Example 20: Copy Widget Config

**User:** "Add another task summary but group it by priority"

**Analysis:**
- User wants second task_summary
- Different config than existing

**Tool Call:**
```json
{
  "action": "add_widget",
  "widget_type": "task_summary",
  "config": {
    "group_by": "priority"
  }
}
```

**Response:** "Added a second task summary, this one grouped by priority. Now you can see tasks by both status and priority."

---

### Example 21: Clear and Rebuild

**User:** "Clear my dashboard and start fresh with just tasks and deadlines"

**Analysis:**
- Need to identify approach:
  - Option A: Remove all widgets, add new ones
  - Option B: Create new dashboard

**Clarify first:**

"Would you like me to:
1. **Clear this dashboard** and add fresh widgets
2. **Create a new dashboard** and keep this one as backup

Either way, I'll add task summary and deadlines."

**If user chooses option 1:**
```json
{
  "action": "add_widgets",
  "clear_existing": true,
  "widgets": [
    {"type": "task_summary"},
    {"type": "upcoming_deadlines"}
  ]
}
```

---

### Example 22: Investigate Dashboard

**User:** "What's on my dashboard right now?"

**Tool Call:**
```json
{
  "action": "get_dashboard"
}
```

**Tool Response:**
```json
{
  "success": true,
  "dashboard": {
    "name": "My Day",
    "widgets": [
      {"type": "task_summary", "config": {"group_by": "status"}},
      {"type": "upcoming_deadlines", "config": {"days_ahead": 7}},
      {"type": "metric_card", "config": {"metric": "tasks_due_today"}}
    ]
  }
}
```

**Response:** "Your 'My Day' dashboard has 3 widgets:
- **Task Summary** - Grouped by status
- **Upcoming Deadlines** - Next 7 days
- **Due Today** - Count of today's tasks

Want to add, remove, or change anything?"
