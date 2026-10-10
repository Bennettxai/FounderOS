---
name: task-management
description: >
  Create, update, complete, and delete tasks. Filter by status, project,
  priority, or due date. Bulk operations for efficiency. Assign tasks to
  team members and manage task dependencies.
metadata:
  version: "1.0.0"
  author: businessos
  tools_used:
    - manage_tasks
  depends_on: []
  context_hints:
    - "Current project context"
    - "User's assigned tasks"
    - "Recent task activity"
  telemetry:
    track_events:
      - task_created
      - task_completed
      - task_updated
      - bulk_operation
---

# Task Management

## When to Use This Skill

Activate when user wants to:
- Create new tasks
- Update task details (title, description, due date, priority)
- Complete or reopen tasks
- Delete tasks
- Filter or search tasks
- Bulk operations (complete all, move multiple)
- Assign tasks to team members

**Keywords:** task, create, add, complete, done, finish, update, change, delete, remove, assign, due, priority, todo

**Do NOT use when:**
- User wants to see tasks on dashboard → use `dashboard-management`
- User asks "why" questions → use `analytics-insights`
- User wants to manage project settings → use `project-management`

---

## The Tool

**Name:** `manage_tasks`

**Actions:**

| Action | Purpose | Required Params |
|--------|---------|-----------------|
| `create` | Create new task | `title` |
| `update` | Update task details | `task_id`, fields to update |
| `complete` | Mark task done | `task_id` |
| `reopen` | Reopen completed task | `task_id` |
| `delete` | Delete task | `task_id` |
| `list` | List/filter tasks | filters (optional) |
| `search` | Search tasks by text | `query` |
| `bulk_complete` | Complete multiple | `task_ids[]` or `filter` |
| `bulk_move` | Move multiple to project | `task_ids[]`, `project_id` |
| `assign` | Assign to user | `task_id`, `assignee_id` |

---

## Request → Tool Mapping

| User Says | Action | Key Params |
|-----------|--------|------------|
| "Create a task: Review PR" | `create` | `title: "Review PR"` |
| "Add a task to call John tomorrow" | `create` | `title: "Call John", due_date: tomorrow` |
| "Mark Review PR done" | `complete` | `task_id: (resolve from title)` |
| "Complete all tasks in Project Alpha" | `bulk_complete` | `filter: {project: "Alpha"}` |
| "What tasks do I have" | `list` | none |
| "Show me overdue tasks" | `list` | `filter: {status: "overdue"}` |
| "Change the priority to high" | `update` | `task_id, priority: "high"` |
| "Assign this to Sarah" | `assign` | `task_id, assignee: "Sarah"` |
| "Delete the old meeting task" | `delete` | `task_id` |

---

## Examples

### Example 1: Create Task

**User:** "Create a task to review the quarterly report by Friday"

**Tool Call:**
```json
{
  "action": "create",
  "title": "Review quarterly report",
  "due_date": "2026-01-17"
}
```

**Response:** "Created task 'Review quarterly report' due Friday, January 17th."

---

### Example 2: Quick Complete

**User:** "Done with Review PR"

**Tool Call:**
```json
{
  "action": "complete",
  "task_id": "(resolved from 'Review PR')"
}
```

**Response:** "Marked 'Review PR' as complete. Nice work!"

---

### Example 3: Bulk Operation

**User:** "Complete all tasks in the Website project"

**Tool Call:**
```json
{
  "action": "bulk_complete",
  "filter": {
    "project_name": "Website"
  }
}
```

**Response:** "Completed 8 tasks in the Website project."

---

### TODO: Add more examples
- List with filters
- Update multiple fields
- Search and filter
- Assignment

---

## Error Handling

### Task Not Found

**Say:** "I couldn't find a task called '{task_name}'. Did you mean one of these?
- {similar_task_1}
- {similar_task_2}

Or would you like to create it?"

### Permission Denied

**Say:** "You can't modify '{task_name}' - it's assigned to another team member. You can ask them to update it, or I can create a similar task for you."

---

## Limitations

- Cannot create tasks in projects user doesn't have access to
- Bulk operations limited to 50 tasks at once
- Cannot undo delete (soft delete retained for 30 days)

---

## Additional Resources

- `FILTERS.md` - All filter options and combinations
- `EXAMPLES.md` - Extended conversation examples
