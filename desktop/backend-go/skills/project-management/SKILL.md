---
name: project-management
description: >
  Create and manage projects. Add team members, set deadlines, track status,
  and organize work. Configure project settings, templates, and workflows.
metadata:
  version: "1.0.0"
  author: businessos
  tools_used:
    - manage_projects
  depends_on:
    - task-management  # Projects contain tasks
  context_hints:
    - "User's projects"
    - "Team members in workspace"
    - "Project templates available"
  telemetry:
    track_events:
      - project_created
      - member_added
      - status_changed
---

# Project Management

## When to Use This Skill

Activate when user wants to:
- Create new projects
- Update project details (name, deadline, description)
- Add or remove team members
- Change project status
- Archive or delete projects
- View project summary

**Keywords:** project, create project, add member, team, deadline, archive, status, template

**Do NOT use when:**
- User wants to manage tasks → use `task-management`
- User asks about project progress metrics → use `analytics-insights`
- User wants to see project on dashboard → use `dashboard-management`

---

## The Tool

**Name:** `manage_projects`

**Actions:**

| Action | Purpose | Required Params |
|--------|---------|-----------------|
| `create` | Create new project | `name` |
| `update` | Update project details | `project_id`, fields to update |
| `add_member` | Add team member | `project_id`, `user_id` or `email` |
| `remove_member` | Remove team member | `project_id`, `user_id` |
| `change_status` | Update project status | `project_id`, `status` |
| `archive` | Archive project | `project_id` |
| `delete` | Delete project | `project_id` |
| `list` | List projects | filters (optional) |
| `get` | Get project details | `project_id` |

---

## Request → Tool Mapping

| User Says | Action | Key Params |
|-----------|--------|------------|
| "Create a project called Q1 Planning" | `create` | `name: "Q1 Planning"` |
| "Add Sarah to the Website project" | `add_member` | `project, user: "Sarah"` |
| "Remove John from Project Alpha" | `remove_member` | `project, user: "John"` |
| "Mark Website project as complete" | `change_status` | `status: "completed"` |
| "Archive the old Marketing project" | `archive` | `project_id` |
| "What projects do I have" | `list` | none |
| "Show me the Alpha project details" | `get` | `project_id` |
| "Change deadline to end of month" | `update` | `deadline: end_of_month` |

---

## Examples

### Example 1: Create Project

**User:** "Create a new project for the mobile app launch"

**Tool Call:**
```json
{
  "action": "create",
  "name": "Mobile App Launch"
}
```

**Response:** "Created project 'Mobile App Launch'. Would you like to add team members or set a deadline?"

---

### Example 2: Add Team Member

**User:** "Add Sarah to the Website project"

**Tool Call:**
```json
{
  "action": "add_member",
  "project_name": "Website",
  "user_identifier": "Sarah"
}
```

**Response:** "Added Sarah to the Website project. She'll now see all project tasks and updates."

---

### TODO: Add more examples
- Update project details
- Change status
- Archive project
- List with filters

---

## Project Statuses

| Status | Description |
|--------|-------------|
| `planning` | Not yet started |
| `active` | In progress |
| `on_hold` | Temporarily paused |
| `completed` | Finished |
| `archived` | Hidden from active view |

---

## Error Handling

### Project Not Found

**Say:** "I couldn't find a project called '{name}'. Your projects are:
- {project_1}
- {project_2}

Did you mean one of these, or would you like to create '{name}'?"

### User Not Found

**Say:** "I couldn't find '{user_name}' in your workspace. Would you like to invite them?"

### Permission Denied

**Say:** "Only project owners can {action}. You can ask {owner_name} to make this change."

---

## Limitations

- Cannot delete projects with active tasks (archive instead)
- Project owners can add members; members cannot
- Maximum 50 members per project

---

## Additional Resources

- `STATUSES.md` - Project lifecycle and status transitions
- `PERMISSIONS.md` - Who can do what
- `EXAMPLES.md` - Extended conversation examples
