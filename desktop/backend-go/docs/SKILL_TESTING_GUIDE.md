# Skill Testing Guide

A comprehensive test guide for all AI agent skills in BusinessOS.

---

## 1. TASK MANAGEMENT SKILL

| # | Prompt | Expected Behavior |
|---|--------|-------------------|
| 1.1 | `"Show me my tasks"` | Lists all tasks with names, priorities, status |
| 1.2 | `"Create a task called Review Q1 report with high priority"` | Creates task, confirms with details |
| 1.3 | `"Create 3 tasks: Design mockups, Write tests, Deploy app"` | Uses bulk_create_tasks, confirms all 3 |
| 1.4 | `"What tasks are due this week?"` | Filters by due date |
| 1.5 | `"Mark my first task as complete"` | Updates task status |
| 1.6 | `"Move the Review task to the Website project"` | Uses move_task |

---

## 2. PROJECT MANAGEMENT SKILL

| # | Prompt | Expected Behavior |
|---|--------|-------------------|
| 2.1 | `"What projects do I have?"` | Lists all projects |
| 2.2 | `"Create a project called Website Redesign"` | Creates project, confirms |
| 2.3 | `"Show me details of my Website project"` | Gets specific project |
| 2.4 | `"Update the Website project status to in progress"` | Uses update_project |

---

## 3. DASHBOARD MANAGEMENT SKILL

| # | Prompt | Expected Behavior |
|---|--------|-------------------|
| 3.1 | `"Create a dashboard called Sales Overview"` | Creates dashboard |
| 3.2 | `"
"` | Adds widget type "list" |
| 3.3 | `"Add a revenue chart to my Sales dashboard"` | Adds chart widget |
| 3.4 | `"Show me my dashboards"` | Lists all dashboards |
| 3.5 | `"Make the Sales dashboard my default"` | Sets as default |
| 3.6 | `"Create a productivity dashboard with tasks, calendar, and metrics widgets"` | Multiple widgets at once |
| 3.7 | `"Clone my Sales dashboard and call it Q2 Sales"` | Clones dashboard |
| 3.8 | `"Remove the chart widget from my dashboard"` | Removes widget |

---

## 4. ANALYTICS SKILL

| # | Prompt | Expected Behavior |
|---|--------|-------------------|
| 4.1 | `"Show me my productivity metrics"` | Uses query_metrics |
| 4.2 | `"How many tasks did I complete this month?"` | Shows completed count |
| 4.3 | `"What's my project progress?"` | Project completion % |
| 4.4 | `"Show me overdue tasks"` | Filters overdue items |
| 4.5 | `"Give me a summary of my workload"` | Overview metrics |

---

## 5. KNOWLEDGE BASE / SEARCH

| # | Prompt | Expected Behavior |
|---|--------|-------------------|
| 5.1 | `"Search my documents for meeting notes"` | Uses search_documents |
| 5.2 | `"Find information about the marketing strategy"` | Searches knowledge base |
| 5.3 | `"What do I know about client onboarding?"` | Uses tree_search |

---

## 6. COMBINED/COMPLEX QUERIES

| # | Prompt | Expected Behavior |
|---|--------|-------------------|
| 6.1 | `"Create a project called Q2 Launch with 3 tasks: Planning, Development, Testing"` | Creates project + 3 tasks |
| 6.2 | `"Show me my high priority tasks and add them to my dashboard"` | Lists tasks + configures dashboard |
| 6.3 | `"What's the status of my Website project and its tasks?"` | Gets project + lists related tasks |

---

## 7. CONVERSATIONAL (No Tools)

| # | Prompt | Expected Behavior |
|---|--------|-------------------|
| 7.1 | `"Hello!"` | Friendly greeting, no thinking panel |
| 7.2 | `"What can you help me with?"` | Explains capabilities |
| 7.3 | `"Thanks for your help"` | Polite response |

---

## What to Check for Each Test

1. **ThinkingPanel**: Shows user-friendly messages like "Checking your tasks..." NOT "Running tool: list_tasks"
2. **Response completeness**: Not cut off mid-sentence
3. **No technical leaks**: No UUIDs, no "Tool Result", no database terms
4. **Proper formatting**: Markdown lists, bold names, clean layout
5. **Accurate data**: Shows actual data from tools, not hallucinated

---

## Quick Test Sequence

Start with these in order:

```
1. "Show me my tasks"
2. "Create a task called Test Task with medium priority"
3. "Show me my tasks" (should show the new one)
4. "Create a dashboard called My Dashboard"
5. "Add a tasks widget to my dashboard"
6. "Show me my dashboards"
7. "What projects do I have?"
```

---

## Skill Tools Reference

### Task Management
- `list_tasks` - List all tasks with filters
- `get_task` - Get specific task details
- `create_task` - Create a single task
- `update_task` - Update task properties
- `bulk_create_tasks` - Create multiple tasks (max 50)
- `move_task` - Move task to different project
- `assign_task` - Assign task to team member

### Project Management
- `list_projects` - List all projects
- `get_project` - Get project details
- `create_project` - Create new project
- `update_project` - Update project properties

### Dashboard Management
- `configure_dashboard` - All dashboard operations:
  - `create_dashboard` - Create new dashboard
  - `add_widget` - Add widget to dashboard
  - `remove_widget` - Remove widget
  - `reorder_widgets` - Change widget order
  - `update_widget_config` - Modify widget settings
  - `list_dashboards` - Get all dashboards
  - `get_dashboard` - Get specific dashboard
  - `delete_dashboard` - Remove dashboard
  - `set_default` - Make dashboard default
  - `clone_dashboard` - Duplicate dashboard

### Analytics
- `query_metrics` - Get productivity metrics
  - Types: summary, tasks_completed, tasks_overdue, project_progress, workload

### Knowledge Base
- `search_documents` - Search documents by keyword
- `tree_search` - Semantic search in knowledge base
- `browse_tree` - Navigate knowledge structure
- `load_context` - Load relevant context

---

## Widget Types Available

| Type | Use Case |
|------|----------|
| `chart` | Line, bar, pie charts |
| `kpi` | Single metric display |
| `table` | Data tables |
| `list` | Task/item lists |
| `calendar` | Event calendar |
| `activity` | Activity feed |
| `progress` | Progress bars |
| `metric_card` | Metric with trend |

---

*Last Updated: January 2026*
