# Task Filters Reference

## TODO: Complete this reference

---

## Filter Options

### By Status
```json
{
  "filter": {
    "status": "todo"  // todo, in_progress, review, done, blocked
  }
}
```

### By Due Date
```json
{
  "filter": {
    "due": "today"  // today, tomorrow, this_week, overdue, no_date
  }
}
```

### By Priority
```json
{
  "filter": {
    "priority": "high"  // high, medium, low, none
  }
}
```

### By Project
```json
{
  "filter": {
    "project_id": "uuid",
    // OR
    "project_name": "Website Redesign"
  }
}
```

### By Assignee
```json
{
  "filter": {
    "assignee": "me",  // me, unassigned, or user_id
  }
}
```

### Combined Filters
```json
{
  "filter": {
    "status": "in_progress",
    "priority": "high",
    "due": "this_week"
  }
}
```

---

## Sort Options

| Sort By | Description |
|---------|-------------|
| `due_date` | By due date (default) |
| `priority` | High → Low |
| `created_at` | Newest first |
| `updated_at` | Recently modified |
| `title` | Alphabetical |

---

## Common Filter Combinations

| Intent | Filter |
|--------|--------|
| "What's urgent" | `{priority: "high", due: "this_week"}` |
| "What's overdue" | `{due: "overdue"}` |
| "My tasks today" | `{assignee: "me", due: "today"}` |
| "Blocked items" | `{status: "blocked"}` |
| "Unassigned in project X" | `{project_id: "X", assignee: "unassigned"}` |
