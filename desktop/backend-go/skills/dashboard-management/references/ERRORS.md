# Error Recovery Guide

How to handle every error the dashboard tool can return.

---

## Error Response Format

All errors follow this structure:

```json
{
  "success": false,
  "error": "error_code",
  "message": "Human-readable message",
  "details": { },
  "suggestion": "What to try instead",
  "available_alternatives": ["option1", "option2"]
}
```

---

## Widget Errors

### widget_type_not_found

**Cause:** User requested a widget type that doesn't exist.

**Response:**
```json
{
  "error": "widget_type_not_found",
  "message": "Widget type 'team_capacity' is not available",
  "requested": "team_capacity",
  "available_alternatives": ["workload_heatmap", "task_summary"]
}
```

**Recovery:**
1. Acknowledge the widget doesn't exist
2. Suggest alternatives from `available_alternatives`
3. Ask which they'd prefer

**Say:** "Team capacity widget isn't available yet. I can add a workload heatmap that shows task density over time, or a task summary showing your workload. Which would you prefer?"

---

### widget_not_found

**Cause:** Trying to remove/update a widget that doesn't exist on the dashboard.

**Response:**
```json
{
  "error": "widget_not_found",
  "message": "Widget not found on dashboard",
  "widget_id": "requested-uuid",
  "current_widgets": [
    {"id": "w1", "type": "task_summary"},
    {"id": "w2", "type": "upcoming_deadlines"}
  ]
}
```

**Recovery:**
1. List the widgets that DO exist
2. Ask user to clarify

**Say:** "I couldn't find that widget. Your dashboard currently has:
- Task Summary
- Upcoming Deadlines

Which one did you want to modify?"

---

### duplicate_widget_type

**Cause:** Adding a widget type that already exists (when duplicates aren't allowed).

**Response:**
```json
{
  "error": "duplicate_widget_type",
  "message": "A task_summary widget already exists",
  "widget_type": "task_summary",
  "existing_widget_id": "w1"
}
```

**Recovery:**
1. Inform about existing widget
2. Offer to update it instead
3. Or allow duplicate with different config

**Say:** "You already have a task summary widget. Would you like me to update its settings, or add another one with different grouping?"

---

## Dashboard Errors

### dashboard_not_found

**Cause:** Referenced dashboard doesn't exist.

**Response:**
```json
{
  "error": "dashboard_not_found",
  "message": "Dashboard 'Sales View' not found",
  "requested": "Sales View",
  "available_dashboards": [
    {"id": "d1", "name": "My Day", "is_default": true},
    {"id": "d2", "name": "Project Overview"}
  ]
}
```

**Recovery:**
1. List available dashboards
2. Ask if they want to create the requested one

**Say:** "I don't see a dashboard called 'Sales View'. You have:
- **My Day** (default)
- **Project Overview**

Would you like me to create 'Sales View' as a new dashboard?"

---

### dashboard_full

**Cause:** Dashboard has reached maximum widget count (12).

**Response:**
```json
{
  "error": "dashboard_full",
  "message": "Dashboard has reached maximum widgets",
  "current_count": 12,
  "max_count": 12,
  "dashboard_name": "My Day"
}
```

**Recovery:**
1. Explain the limit
2. Offer to remove a widget
3. Or create a new dashboard

**Say:** "Your 'My Day' dashboard already has 12 widgets (the maximum). I can:
1. Remove a widget to make room
2. Create a new dashboard for additional widgets

What would you prefer?"

---

### dashboard_name_exists

**Cause:** Trying to create a dashboard with a name that already exists.

**Response:**
```json
{
  "error": "dashboard_name_exists",
  "message": "A dashboard named 'Project View' already exists",
  "existing_dashboard_id": "d2"
}
```

**Recovery:**
1. Inform about existing dashboard
2. Offer to use the existing one
3. Or suggest a different name

**Say:** "You already have a dashboard called 'Project View'. Would you like me to add widgets to that one, or create a new one with a different name?"

---

## Configuration Errors

### invalid_config

**Cause:** Widget config has invalid values.

**Response:**
```json
{
  "error": "invalid_config",
  "message": "Invalid configuration for task_summary",
  "field": "group_by",
  "value": "date",
  "allowed_values": ["status", "project", "priority"]
}
```

**Recovery:**
1. Explain what's wrong
2. Show valid options
3. Ask which they want

**Say:** "I can't group tasks by 'date'. The options are:
- **status** - To Do, In Progress, Done, etc.
- **project** - By project name
- **priority** - High, Medium, Low

Which grouping would work best?"

---

### config_out_of_range

**Cause:** Numeric config value is outside allowed range.

**Response:**
```json
{
  "error": "config_out_of_range",
  "message": "Limit must be between 5 and 50",
  "field": "limit",
  "value": 100,
  "min": 5,
  "max": 50
}
```

**Recovery:**
1. Explain the valid range
2. Suggest the closest valid value

**Say:** "The task list can show between 5 and 50 items. Would you like me to set it to 50 (the maximum)?"

---

### project_not_found

**Cause:** Referenced project doesn't exist or user doesn't have access.

**Response:**
```json
{
  "error": "project_not_found",
  "message": "Project not found",
  "project_id": "invalid-uuid",
  "available_projects": [
    {"id": "p1", "name": "Website Redesign"},
    {"id": "p2", "name": "Mobile App"}
  ]
}
```

**Recovery:**
1. List available projects
2. Ask which one they meant

**Say:** "I couldn't find that project. Your active projects are:
- Website Redesign
- Mobile App

Which one would you like to use?"

---

## Permission Errors

### permission_denied

**Cause:** User doesn't have access to perform the action.

**Response:**
```json
{
  "error": "permission_denied",
  "message": "Cannot modify workspace dashboard template",
  "required_role": "admin",
  "user_role": "member"
}
```

**Recovery:**
1. Explain why action isn't allowed
2. Suggest alternative (personal dashboard)

**Say:** "You can't modify the workspace template (that requires admin access). Would you like me to create a personal copy you can customize?"

---

### workspace_access_denied

**Cause:** User trying to access data from another workspace.

**Response:**
```json
{
  "error": "workspace_access_denied",
  "message": "Cannot access data from another workspace"
}
```

**Recovery:**
1. Explain the limitation
2. Confirm they're in the right workspace

**Say:** "I can only show data from your current workspace. Make sure you're viewing the right workspace in the sidebar."

---

## System Errors

### rate_limited

**Cause:** Too many requests in short time.

**Response:**
```json
{
  "error": "rate_limited",
  "message": "Too many dashboard changes",
  "retry_after_seconds": 30
}
```

**Recovery:**
1. Ask user to wait
2. Offer to batch multiple changes

**Say:** "You're making changes quickly! Give me 30 seconds, then I can continue. If you have multiple changes, tell me all at once and I'll do them together."

---

### internal_error

**Cause:** Unexpected server error.

**Response:**
```json
{
  "error": "internal_error",
  "message": "An unexpected error occurred",
  "error_id": "err-abc123"
}
```

**Recovery:**
1. Apologize
2. Suggest retry
3. Mention error ID for support

**Say:** "Something went wrong on my end (error: err-abc123). Try again in a moment, or if it keeps happening, our team can look into it."

---

## Error Recovery Patterns

### Pattern: Suggest Alternatives

When tool returns `available_alternatives`:

```
1. Acknowledge request can't be fulfilled
2. Present alternatives conversationally
3. Ask which they'd prefer
4. Don't just list - explain what each does
```

### Pattern: Clarify Ambiguity

When multiple items could match:

```
1. List the matching items
2. Add brief descriptions
3. Ask which one
4. Remember their choice for context
```

### Pattern: Offer Workarounds

When an action is blocked:

```
1. Explain why it's blocked
2. Offer alternatives that achieve similar goal
3. Let user choose
```
