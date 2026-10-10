---
name: analytics-insights
description: >
  Analyze productivity trends, explain metrics, and identify bottlenecks.
  Answer questions like "why are tasks overdue" or "how am I doing this week".
  Provide actionable insights based on workspace data.
metadata:
  version: "1.0.0"
  author: businessos
  tools_used:
    - query_analytics
  depends_on:
    - task-management   # Can suggest task actions based on insights
  context_hints:
    - "User's recent activity"
    - "Task completion patterns"
    - "Project deadlines and status"
  telemetry:
    track_events:
      - insight_requested
      - recommendation_given
      - trend_explained
---

# Analytics Insights

## When to Use This Skill

Activate when user asks:
- "Why" questions about their data
- "How am I doing" or productivity questions
- Trend analysis or comparisons
- Root cause analysis for problems
- Performance summaries

**Keywords:** why, analyze, trend, productivity, insight, explain, compare, performance, bottleneck, behind, overdue reason

**Do NOT use when:**
- User just wants to view tasks → use `task-management`
- User wants to add a chart to dashboard → use `dashboard-management`
- User wants to create/edit data → use appropriate management skill

---

## The Tool

**Name:** `query_analytics`

**Actions:**

| Action | Purpose | Required Params |
|--------|---------|-----------------|
| `get_summary` | Overview of productivity | `period` (optional) |
| `get_trends` | Show changes over time | `metric`, `period` |
| `analyze_overdue` | Why tasks are overdue | `project_id` (optional) |
| `analyze_blockers` | What's blocking progress | `project_id` (optional) |
| `compare_periods` | Compare two time periods | `period1`, `period2` |
| `get_workload` | Team/user capacity analysis | `user_id` (optional) |
| `explain_metric` | Explain a specific KPI | `metric` |

---

## Request → Tool Mapping

| User Says | Action | Key Params |
|-----------|--------|------------|
| "How am I doing this week" | `get_summary` | `period: "week"` |
| "Why are so many tasks overdue" | `analyze_overdue` | none |
| "What's blocking the Alpha project" | `analyze_blockers` | `project_id: "alpha"` |
| "Compare this month to last month" | `compare_periods` | `period1: "this_month", period2: "last_month"` |
| "Am I more productive than last week" | `compare_periods` | `period1: "this_week", period2: "last_week"` |
| "What does completion rate mean" | `explain_metric` | `metric: "completion_rate"` |
| "Show me productivity trends" | `get_trends` | `metric: "tasks_completed", period: "month"` |

---

## Response Style

**DON'T** just return numbers:
> "You have 12 overdue tasks."

**DO** explain and suggest:
> "You have 12 overdue tasks, mostly in the 'Website Redesign' project. 8 of them were due in the last 3 days—looks like a crunch period. The main causes are:
> 1. Blocked tasks waiting on design approval (4 tasks)
> 2. Underestimated complexity (3 tasks)
>
> Want me to help prioritize or reassign some of these?"

---

## Examples

### Example 1: Weekly Summary

**User:** "How am I doing this week?"

**Tool Call:**
```json
{
  "action": "get_summary",
  "period": "week"
}
```

**Response:** "Here's your week so far:
- **Completed:** 18 tasks ↑ (vs 12 last week)
- **In Progress:** 6 tasks
- **Overdue:** 2 tasks (down from 5)

You're ahead of last week's pace! The two overdue items are both in the Mobile App project—want me to look into what's blocking them?"

---

### Example 2: Overdue Analysis

**User:** "Why are my tasks overdue?"

**Tool Call:**
```json
{
  "action": "analyze_overdue"
}
```

**Response:** "You have 8 overdue tasks. Here's why:

**By Root Cause:**
- 4 tasks blocked by dependencies (waiting on others)
- 2 tasks had underestimated time
- 2 tasks were deprioritized but not rescheduled

**By Project:**
- Website Redesign: 5 tasks
- API Integration: 3 tasks

**Suggestion:** The 4 blocked tasks need unblocking first. Want me to show you which dependencies are holding things up?"

---

### TODO: Add more examples following this pattern
- Period comparison
- Blocker analysis
- Workload insights
- Metric explanation

---

## Error Handling

### Insufficient Data

**Tool Returns:** `{ "error": "insufficient_data", "reason": "less than 7 days of data" }`

**Say:** "I don't have enough data for a meaningful trend yet. Check back after you've been using BusinessOS for a week, and I'll have better insights for you."

### No Overdue Tasks

**Tool Returns:** `{ "overdue_count": 0 }`

**Say:** "Great news—you don't have any overdue tasks! You're on top of things. Want to see what's coming up this week instead?"

---

## Limitations

- Cannot modify data (read-only analysis)
- Trends require minimum 7 days of data
- Comparisons work best with similar-length periods
- Team insights require appropriate permissions

---

## Additional Resources

If you need more details, these references are available:
- `METRICS.md` - All metrics with calculations and interpretations
- `INSIGHTS.md` - Common insight patterns and recommendations
- `EXAMPLES.md` - Extended conversation examples
