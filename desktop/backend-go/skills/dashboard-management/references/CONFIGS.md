# Configuration Reference

Complete documentation for all widget configuration options.

---

## Common Configuration Patterns

### Filtering by Project

Many widgets support `project_id` to filter data:

```json
{
  "widget_type": "task_summary",
  "config": {
    "project_id": "uuid-of-project"
  }
}
```

**Applies to:** task_summary, task_list, task_burndown, upcoming_deadlines, project_progress

---

### Time-Based Filtering

Widgets that show time-sensitive data:

| Widget | Option | Values | Example |
|--------|--------|--------|---------|
| upcoming_deadlines | `days_ahead` | 1-90 | 7 (one week) |
| task_burndown | `days` | 7-365 | 30 (one month) |
| workload_heatmap | `date_range` | month, quarter, year | "quarter" |

---

### Limiting Results

Most list widgets support `limit`:

```json
{
  "widget_type": "task_list",
  "config": {
    "limit": 20
  }
}
```

**Default limits by widget:**
| Widget | Default | Min | Max |
|--------|---------|-----|-----|
| task_list | 15 | 5 | 50 |
| upcoming_deadlines | 10 | 5 | 30 |
| recent_activity | 15 | 5 | 50 |
| project_progress | 5 | 1 | 10 |
| notes_pinned | 5 | 1 | 10 |
| client_overview | 10 | 5 | 20 |

---

## Configuration by Widget Type

### task_summary

```json
{
  "group_by": "status",        // status | project | priority
  "show_completed": false,     // Include completed tasks
  "project_id": null           // Filter to project (optional)
}
```

**group_by options:**
- `"status"` - Groups: To Do, In Progress, Review, Done, Blocked
- `"project"` - Groups by project name
- `"priority"` - Groups: High, Medium, Low, None

---

### task_list

```json
{
  "limit": 15,                 // 5-50
  "filter": "active",          // active | overdue | today | all
  "show_completed": false,     // Show completed tasks
  "project_id": null,          // Filter to project
  "sort_by": "due_date"        // due_date | priority | created_at
}
```

**filter options:**
- `"active"` - Not completed, not blocked
- `"overdue"` - Past due date
- `"today"` - Due today
- `"all"` - Everything including completed

---

### task_burndown

```json
{
  "days": 30,                  // 7-365
  "project_id": null,          // Filter to project
  "show_ideal": true           // Show ideal burndown line
}
```

---

### upcoming_deadlines

```json
{
  "days_ahead": 7,             // 1-90
  "show_overdue": true,        // Include overdue items
  "limit": 10,                 // 5-30
  "project_id": null           // Filter to project
}
```

---

### workload_heatmap

```json
{
  "date_range": "month",       // month | quarter | year
  "metric": "due"              // due | created | completed
}
```

---

### project_progress

```json
{
  "project_id": null,          // Specific project or all
  "limit": 5,                  // 1-10 projects
  "sort_by": "progress"        // progress | deadline | name
}
```

---

### metric_card

```json
{
  "metric": "tasks_due_today", // Required - see options below
  "show_trend": true           // Show comparison to last period
}
```

**metric options:**
| Value | Description |
|-------|-------------|
| `tasks_due_today` | Count of tasks due today |
| `tasks_overdue` | Count of overdue tasks |
| `tasks_completed_week` | Tasks completed this week |
| `tasks_completed_today` | Tasks completed today |
| `active_projects` | Projects in progress |
| `projects_at_risk` | Projects behind schedule |

---

### recent_activity

```json
{
  "limit": 15,                 // 5-50
  "filter": "all",             // all | tasks | projects | comments
  "user_filter": "all"         // all | me | others
}
```

---

### client_overview

```json
{
  "pipeline_stage": null,      // lead | prospect | active | churned (or null for all)
  "limit": 10                  // 5-20
}
```

---

### notes_pinned

```json
{
  "limit": 5,                  // 1-10
  "tag": null                  // Filter by tag (optional)
}
```

---

### quick_actions

```json
{
  "actions": ["new_task", "new_note", "new_project"]
}
```

**Available actions:**
- `new_task` - Create task button
- `new_project` - Create project button
- `new_note` - Create note button
- `new_client` - Add client button
- `start_timer` - Start time tracking button

---

### agent_shortcuts

```json
{
  "shortcuts": [
    "Summarize my day",
    "What's most urgent?",
    "Draft a status update"
  ]
}
```

Custom shortcuts are strings that become clickable commands.

---

## Default Configurations

When a user adds a widget without specifying config, these defaults apply:

```json
{
  "task_summary": {
    "group_by": "status",
    "show_completed": false
  },
  "task_list": {
    "limit": 15,
    "filter": "active",
    "sort_by": "due_date"
  },
  "task_burndown": {
    "days": 30,
    "show_ideal": true
  },
  "upcoming_deadlines": {
    "days_ahead": 7,
    "show_overdue": true,
    "limit": 10
  },
  "workload_heatmap": {
    "date_range": "month",
    "metric": "due"
  },
  "project_progress": {
    "limit": 5,
    "sort_by": "progress"
  },
  "metric_card": {
    "metric": "tasks_due_today",
    "show_trend": true
  },
  "recent_activity": {
    "limit": 15,
    "filter": "all",
    "user_filter": "all"
  },
  "client_overview": {
    "limit": 10
  },
  "notes_pinned": {
    "limit": 5
  },
  "quick_actions": {
    "actions": ["new_task", "new_note"]
  },
  "agent_shortcuts": {
    "shortcuts": ["Summarize my day", "What's most urgent?"]
  }
}
```

---

## Validation Rules

### Type Validation

| Type | Rule |
|------|------|
| `integer` | Must be whole number within min/max |
| `string` | Must be one of allowed values |
| `boolean` | Must be `true` or `false` |
| `array` | Must be array of allowed items |
| `uuid` | Must be valid UUID format |

### Common Errors

| Error | Cause | Fix |
|-------|-------|-----|
| `invalid_group_by` | Unknown grouping option | Use: status, project, priority |
| `limit_out_of_range` | Number too high/low | Check widget's min/max |
| `invalid_metric` | Unknown metric name | See metric_card options |
| `invalid_date_range` | Unknown range | Use: month, quarter, year |
| `project_not_found` | Bad project_id | Verify project UUID |
