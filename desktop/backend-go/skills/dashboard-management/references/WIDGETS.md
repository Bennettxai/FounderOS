# Widget Catalog

Complete reference for all available dashboard widgets.

---

## Task Widgets

### task_summary

**Description:** Shows task counts grouped by a category. Great for quick workload overview.

**Default Size:** 4 columns × 3 rows

**Config Options:**
| Option | Type | Default | Values | Description |
|--------|------|---------|--------|-------------|
| `group_by` | string | `"status"` | status, project, priority | How to group tasks |
| `show_completed` | boolean | `false` | true, false | Include completed tasks in counts |
| `project_id` | string | null | UUID | Filter to specific project |

**Visual:**
```
┌─────────────────────────────┐
│  Tasks by Status            │
├─────────────────────────────┤
│  ● To Do        12          │
│  ● In Progress   8          │
│  ● Review        3          │
│  ● Blocked       2          │
└─────────────────────────────┘
```

**Best For:** Daily standup view, workload distribution
**Often Paired With:** upcoming_deadlines, metric_card

---

### task_list

**Description:** Scrollable list of tasks with key details. Interactive - users can click to open tasks.

**Default Size:** 4 columns × 4 rows

**Config Options:**
| Option | Type | Default | Values | Description |
|--------|------|---------|--------|-------------|
| `limit` | integer | `15` | 5-50 | Max tasks to show |
| `filter` | string | `"active"` | active, overdue, today, all | Which tasks to include |
| `show_completed` | boolean | `false` | true, false | Show completed tasks |
| `project_id` | string | null | UUID | Filter to specific project |
| `sort_by` | string | `"due_date"` | due_date, priority, created_at | Sort order |

**Visual:**
```
┌─────────────────────────────────┐
│  My Tasks                    ↻  │
├─────────────────────────────────┤
│  ☐ Review PR #234       Today   │
│  ☐ Update docs          Tomorrow│
│  ☐ Fix login bug        Jan 15  │
│  ☐ Team sync prep       Jan 16  │
│  ☐ Client proposal      Jan 18  │
│         ▼ Show more             │
└─────────────────────────────────┘
```

**Best For:** Daily task management, focused work view
**Often Paired With:** task_summary, quick_actions

---

### task_burndown

**Description:** Line chart showing tasks completed vs remaining over time. Classic sprint tracking.

**Default Size:** 6 columns × 4 rows

**Config Options:**
| Option | Type | Default | Values | Description |
|--------|------|---------|--------|-------------|
| `days` | integer | `30` | 7-365 | Time period to show |
| `project_id` | string | null | UUID | Filter to specific project |
| `show_ideal` | boolean | `true` | true, false | Show ideal burndown line |

**Visual:**
```
┌─────────────────────────────────────────┐
│  Burndown - Last 30 Days                │
├─────────────────────────────────────────┤
│  50│ ╲                                  │
│    │   ╲___                             │
│  25│       ╲___                         │
│    │           ╲___●                    │
│   0│_______________╲____                │
│    Jan 1        Jan 15        Jan 30    │
│                                         │
│  ── Actual  - - Ideal                   │
└─────────────────────────────────────────┘
```

**Best For:** Sprint retrospectives, project health tracking
**Often Paired With:** project_progress, metric_card

---

## Deadline & Planning Widgets

### upcoming_deadlines

**Description:** Tasks due soon, sorted by date. Highlights overdue items.

**Default Size:** 4 columns × 3 rows

**Config Options:**
| Option | Type | Default | Values | Description |
|--------|------|---------|--------|-------------|
| `days_ahead` | integer | `7` | 1-90 | How far ahead to look |
| `show_overdue` | boolean | `true` | true, false | Include overdue tasks |
| `limit` | integer | `10` | 5-30 | Max items to show |
| `project_id` | string | null | UUID | Filter to specific project |

**Visual:**
```
┌─────────────────────────────┐
│  Upcoming Deadlines         │
├─────────────────────────────┤
│  🔴 Overdue (2)             │
│     • Client report         │
│     • Bug fix #123          │
│  📅 Today (3)               │
│     • Team standup          │
│     • Review designs        │
│  📅 Tomorrow (1)            │
│     • Submit proposal       │
└─────────────────────────────┘
```

**Best For:** Daily planning, deadline awareness
**Often Paired With:** task_summary, quick_actions

---

### workload_heatmap

**Description:** Calendar-style heatmap showing task density per day. Identifies busy periods.

**Default Size:** 6 columns × 3 rows

**Config Options:**
| Option | Type | Default | Values | Description |
|--------|------|---------|--------|-------------|
| `date_range` | string | `"month"` | month, quarter, year | Time period |
| `metric` | string | `"due"` | due, created, completed | What to count |

**Visual:**
```
┌─────────────────────────────────────────┐
│  Task Density - January                 │
├─────────────────────────────────────────┤
│  Mon Tue Wed Thu Fri Sat Sun            │
│   ░   ▒   ░   ▓   █   ░   ░   Week 1   │
│   ▒   ▒   ▓   ▓   █   ░   ░   Week 2   │
│   ░   ▓   ▓   █   █   ░   ░   Week 3   │
│   ▒   ▒   ░   ░   ░   ░   ░   Week 4   │
│                                         │
│  ░ Light  ▒ Medium  ▓ Heavy  █ Overload │
└─────────────────────────────────────────┘
```

**Best For:** Capacity planning, identifying crunch periods
**Often Paired With:** task_burndown, metric_card

---

## Project Widgets

### project_progress

**Description:** Progress bars for active projects based on task completion.

**Default Size:** 4 columns × 3 rows

**Config Options:**
| Option | Type | Default | Values | Description |
|--------|------|---------|--------|-------------|
| `project_id` | string | null | UUID | Specific project (or all) |
| `limit` | integer | `5` | 1-10 | Max projects to show |
| `sort_by` | string | `"progress"` | progress, deadline, name | Sort order |

**Visual:**
```
┌─────────────────────────────┐
│  Project Progress           │
├─────────────────────────────┤
│  Website Redesign           │
│  ████████████░░░░░░ 68%     │
│                             │
│  Mobile App v2              │
│  ██████░░░░░░░░░░░░ 35%     │
│                             │
│  API Integration            │
│  ████████████████░░ 89%     │
└─────────────────────────────┘
```

**Best For:** Portfolio view, stakeholder updates
**Often Paired With:** task_burndown, upcoming_deadlines

---

## Metrics & KPI Widgets

### metric_card

**Description:** Single large number with trend indicator. For key metrics.

**Default Size:** 2 columns × 2 rows

**Config Options:**
| Option | Type | Default | Values | Description |
|--------|------|---------|--------|-------------|
| `metric` | string | required | see below | Which metric to show |
| `show_trend` | boolean | `true` | true, false | Show vs last period |

**Available Metrics:**
- `tasks_due_today` - Tasks due today
- `tasks_overdue` - Overdue task count
- `tasks_completed_week` - Tasks completed this week
- `tasks_completed_today` - Tasks completed today
- `active_projects` - Projects in progress
- `projects_at_risk` - Projects behind schedule

**Visual:**
```
┌───────────────┐
│ Due Today     │
│     12        │
│   ▲ +3        │
└───────────────┘
```

**Best For:** KPI dashboards, quick status checks
**Often Paired With:** Other metric_cards, task_summary

---

### recent_activity

**Description:** Feed of recent actions across workspace.

**Default Size:** 4 columns × 4 rows

**Config Options:**
| Option | Type | Default | Values | Description |
|--------|------|---------|--------|-------------|
| `limit` | integer | `15` | 5-50 | Items to show |
| `filter` | string | `"all"` | all, tasks, projects, comments | Activity types |
| `user_filter` | string | `"all"` | all, me, others | Whose activity |

**Visual:**
```
┌─────────────────────────────────┐
│  Recent Activity                │
├─────────────────────────────────┤
│  ✓ You completed "Fix bug"      │
│    2 minutes ago                │
│                                 │
│  💬 Sarah commented on PR #234  │
│    15 minutes ago               │
│                                 │
│  📁 New project "Q1 Planning"   │
│    1 hour ago                   │
└─────────────────────────────────┘
```

**Best For:** Team awareness, catching up
**Often Paired With:** task_list, quick_actions

---

## Client & Sales Widgets

### client_overview

**Description:** Client pipeline summary by stage.

**Default Size:** 4 columns × 3 rows

**Config Options:**
| Option | Type | Default | Values | Description |
|--------|------|---------|--------|-------------|
| `pipeline_stage` | string | null | lead, prospect, active, churned | Filter by stage |
| `limit` | integer | `10` | 5-20 | Max clients to show |

**Visual:**
```
┌─────────────────────────────┐
│  Client Pipeline            │
├─────────────────────────────┤
│  Leads          8           │
│  Prospects      5           │
│  Active        23           │
│  At Risk        2           │
└─────────────────────────────┘
```

**Best For:** Sales dashboards, account management
**Often Paired With:** metric_card, recent_activity

---

## Utility Widgets

### notes_pinned

**Description:** Display pinned notes for quick reference.

**Default Size:** 4 columns × 3 rows

**Config Options:**
| Option | Type | Default | Values | Description |
|--------|------|---------|--------|-------------|
| `limit` | integer | `5` | 1-10 | Max notes to show |
| `tag` | string | null | any tag | Filter by tag |

**Visual:**
```
┌─────────────────────────────┐
│  📌 Pinned Notes            │
├─────────────────────────────┤
│  Team Standup Agenda        │
│  Daily at 9am, Zoom link... │
│  ─────────────────────────  │
│  Q1 Goals                   │
│  1. Launch mobile app...    │
└─────────────────────────────┘
```

---

### quick_actions

**Description:** Buttons for common actions.

**Default Size:** 2 columns × 2 rows

**Config Options:**
| Option | Type | Default | Values | Description |
|--------|------|---------|--------|-------------|
| `actions` | array | default set | see below | Which actions to show |

**Available Actions:**
- `new_task` - Create task
- `new_project` - Create project
- `new_note` - Create note
- `new_client` - Add client
- `start_timer` - Start time tracking

**Visual:**
```
┌───────────────┐
│  Quick Add    │
├───────────────┤
│  [+ Task]     │
│  [+ Note]     │
│  [+ Project]  │
└───────────────┘
```

---

### agent_shortcuts

**Description:** Pre-configured AI assistant commands.

**Default Size:** 2 columns × 2 rows

**Config Options:**
| Option | Type | Default | Values | Description |
|--------|------|---------|--------|-------------|
| `shortcuts` | array | default set | see below | Commands to show |

**Default Shortcuts:**
- "Summarize my day"
- "What's most urgent?"
- "Draft a status update"
- "Plan tomorrow"

**Visual:**
```
┌───────────────────┐
│  🤖 AI Assistant  │
├───────────────────┤
│  [Summarize day]  │
│  [Most urgent?]   │
│  [Plan tomorrow]  │
└───────────────────┘
```

---

## Widget Size Reference

| Widget | Min Size | Default | Max Size |
|--------|----------|---------|----------|
| task_summary | 3×2 | 4×3 | 6×4 |
| task_list | 3×3 | 4×4 | 6×6 |
| task_burndown | 4×3 | 6×4 | 12×6 |
| upcoming_deadlines | 3×2 | 4×3 | 6×4 |
| workload_heatmap | 4×2 | 6×3 | 12×4 |
| project_progress | 3×2 | 4×3 | 6×4 |
| metric_card | 2×2 | 2×2 | 3×3 |
| recent_activity | 3×3 | 4×4 | 6×6 |
| client_overview | 3×2 | 4×3 | 6×4 |
| notes_pinned | 3×2 | 4×3 | 6×4 |
| quick_actions | 2×2 | 2×2 | 3×3 |
| agent_shortcuts | 2×2 | 2×2 | 4×3 |

Grid is 12 columns wide. Height is unlimited (scrollable).
