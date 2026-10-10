---
name: notification-management
description: >
  Configure notification preferences, set quiet hours, and manage alerts.
  Control how and when you receive notifications for tasks, mentions,
  deadlines, and team activity.
metadata:
  version: "1.0.0"
  author: businessos
  tools_used:
    - configure_notifications
  depends_on: []
  context_hints:
    - "User's current notification settings"
    - "Time zone"
  telemetry:
    track_events:
      - preferences_changed
      - quiet_hours_set
      - channel_configured
---

# Notification Management

## When to Use This Skill

Activate when user wants to:
- Change notification preferences
- Set quiet hours / do not disturb
- Mute specific channels or projects
- Configure notification frequency
- View notification settings

**Keywords:** notification, alert, mute, quiet, disturb, silence, snooze, preference, remind

**Do NOT use when:**
- User wants to see notifications list → show in UI
- User wants task reminders → use `task-management`
- User wants to @mention someone → handle in chat

---

## The Tool

**Name:** `configure_notifications`

**Actions:**

| Action | Purpose | Required Params |
|--------|---------|-----------------|
| `get_settings` | Get current settings | none |
| `update_settings` | Update preferences | settings object |
| `set_quiet_hours` | Set DND schedule | `start`, `end` |
| `mute_project` | Mute project notifications | `project_id`, `duration` |
| `unmute_project` | Unmute project | `project_id` |
| `mute_all` | Mute all notifications | `duration` |
| `unmute_all` | Unmute all | none |

---

## Request → Tool Mapping

| User Says | Action | Key Params |
|-----------|--------|------------|
| "Mute notifications" | `mute_all` | `duration: "1h"` (ask or default) |
| "Turn off notifications for 2 hours" | `mute_all` | `duration: "2h"` |
| "Set quiet hours from 6pm to 9am" | `set_quiet_hours` | `start: "18:00", end: "09:00"` |
| "Only notify me for urgent items" | `update_settings` | `priority_filter: "high"` |
| "Mute the Marketing project" | `mute_project` | `project: "Marketing"` |
| "What are my notification settings" | `get_settings` | none |
| "Unmute everything" | `unmute_all` | none |
| "Turn off email notifications" | `update_settings` | `email: false` |

---

## Examples

### Example 1: Quick Mute

**User:** "Mute notifications"

**Clarify duration first:**
"For how long? Options:
- 1 hour
- Until tomorrow
- Until I turn them back on"

**Then Tool Call:**
```json
{
  "action": "mute_all",
  "duration": "1h"
}
```

**Response:** "Notifications muted for 1 hour. I'll turn them back on at 3:00 PM."

---

### Example 2: Quiet Hours

**User:** "Set quiet hours from 6pm to 8am"

**Tool Call:**
```json
{
  "action": "set_quiet_hours",
  "start": "18:00",
  "end": "08:00"
}
```

**Response:** "Quiet hours set from 6:00 PM to 8:00 AM. You won't receive notifications during this time, but you can still check them manually."

---

### Example 3: Priority Filter

**User:** "Only notify me for important stuff"

**Tool Call:**
```json
{
  "action": "update_settings",
  "settings": {
    "priority_filter": "high",
    "mentions": true,
    "deadlines": true
  }
}
```

**Response:** "Updated! You'll only get notifications for:
- High priority items
- Direct @mentions
- Upcoming deadlines

Everything else will be silently logged."

---

## Notification Types

| Type | Description | Default |
|------|-------------|---------|
| `mentions` | When someone @mentions you | On |
| `assignments` | When assigned a task | On |
| `deadlines` | Upcoming due dates | On |
| `comments` | Comments on your tasks | On |
| `project_updates` | Project status changes | On |
| `team_activity` | Team member actions | Off |

---

## Error Handling

### Invalid Time Format

**Say:** "I need the time in a format like '6pm' or '18:00'. What time should quiet hours start?"

### Project Not Found

**Say:** "I couldn't find '{project_name}'. Your projects are:
- {list}

Which one would you like to mute?"

---

## Limitations

- Quiet hours are in user's local time zone
- Maximum mute duration: 7 days
- Some notifications (security alerts) cannot be muted

---

## Additional Resources

- `CHANNELS.md` - Notification delivery channels
- `EXAMPLES.md` - Extended conversation examples
