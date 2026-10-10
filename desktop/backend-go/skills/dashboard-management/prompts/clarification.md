# Clarification Prompts

Standard prompts for gathering missing information.

---

## Widget Type Ambiguous

When user says something like "the task widget" but multiple match:

```
I found a few task-related widgets on your dashboard:
- **Task Summary** (shows counts by status)
- **Task List** (scrollable list of tasks)
- **Task Burndown** (chart over time)

Which one did you mean?
```

---

## Dashboard Name Ambiguous

When user references a dashboard but multiple could match:

```
You have {count} dashboards that could match:
{foreach dashboard}
- **{name}** ({widget_count} widgets){if is_default} ← current default{/if}
{/foreach}

Which one would you like to use?
```

---

## Dashboard Not Specified

When adding widgets but user has multiple dashboards:

```
You have {count} dashboards. Where should I add this widget?
{foreach dashboard}
- **{name}**{if is_default} (default){/if}
{/foreach}

Or I can add it to your default dashboard ({default_name}).
```

---

## Widget Config Missing

When widget type requires config that wasn't provided:

### metric_card (needs metric)
```
Which metric would you like to see?
- **Tasks due today** - count of today's tasks
- **Overdue tasks** - tasks past their due date
- **Completed this week** - tasks finished this week
- **Active projects** - projects in progress
```

### task_summary (clarify grouping)
```
How would you like tasks grouped?
- **By status** - To Do, In Progress, Done, etc.
- **By project** - grouped by project name
- **By priority** - High, Medium, Low
```

---

## Action Unclear

When user intent isn't clear:

```
I can help you with dashboards! Here's what I can do:
- **Add widgets** like task summaries, charts, or deadlines
- **Create a new dashboard** for a specific purpose  
- **Modify existing widgets** on your current dashboard
- **Remove widgets** you no longer need

What would you like?
```

---

## Project Reference Ambiguous

When user mentions a project that can't be resolved:

```
I couldn't find "{project_name}". Your active projects are:
{foreach project}
- {name}
{/foreach}

Which one did you mean?
```

---

## Multiple Projects Match

When project name matches multiple:

```
I found {count} projects that match "{search}":
{foreach project}
- **{name}** - {task_count} tasks
{/foreach}

Which one should the widget show?
```

---

## Confirm Destructive Action

Before clearing or removing:

### Clear dashboard
```
This will remove all {count} widgets from "{dashboard_name}". 

Are you sure? Your widgets are:
{foreach widget}
- {type_display_name}
{/foreach}

Type "yes" to confirm, or tell me which specific widget to remove.
```

### Remove widget
```
Remove the {widget_type_display} from "{dashboard_name}"?
```

---

## Suggest After Action

After successfully completing an action, suggest next steps:

### After creating dashboard
```
Created "{dashboard_name}"! It's empty right now. Would you like me to add some widgets?

Popular choices:
- Task summary with deadlines
- Project progress overview
- Quick actions panel
```

### After adding first widget
```
Added {widget_type} to your dashboard. Want to add anything else?

Common additions:
- Upcoming deadlines
- Metric cards for quick stats
- Recent activity feed
```
