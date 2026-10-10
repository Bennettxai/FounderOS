# Metrics Reference

## TODO: Complete this reference

This file should contain:

---

## Task Metrics

### tasks_completed
- **Calculation:** Count of tasks moved to "done" status
- **Period options:** today, week, month, quarter
- **Good value:** Consistent or trending up
- **Warning sign:** Sudden drop, or 0 for multiple days

### tasks_overdue
- **Calculation:** Count of tasks past due_date and not completed
- **Good value:** < 5% of active tasks
- **Warning sign:** > 20% or rapidly increasing

### completion_rate
- **Calculation:** tasks_completed / (tasks_completed + tasks_created) * 100
- **Good value:** > 80%
- **Warning sign:** < 50% (creating faster than completing)

### average_task_age
- **Calculation:** Average days from created_at to completed_at
- **Good value:** < 7 days for most task types
- **Warning sign:** > 14 days average

---

## Project Metrics

### project_progress
- **Calculation:** completed_tasks / total_tasks * 100
- **Good value:** On track with timeline
- **Warning sign:** Progress % < expected % based on deadline

### projects_at_risk
- **Calculation:** Projects where progress is significantly behind schedule
- **Threshold:** > 20% behind expected progress

---

## Productivity Metrics

### velocity
- **Calculation:** Tasks completed per week (rolling average)
- **Good value:** Stable or increasing
- **Warning sign:** Declining over 3+ weeks

### focus_time
- **Calculation:** Estimated hours of deep work (based on task completion patterns)
- **Note:** Requires time tracking integration

---

## Interpreting Trends

### Positive Indicators
- Completion rate > 80%
- Overdue count decreasing
- Velocity stable or increasing
- Average task age decreasing

### Warning Signs
- Overdue count increasing
- Completion rate < 50%
- Many tasks stuck in "In Progress" > 7 days
- New tasks created much faster than completed

### Recommendations by Pattern

| Pattern | Likely Cause | Suggestion |
|---------|--------------|------------|
| High overdue, low completion | Overcommitted | Review and reprioritize backlog |
| Tasks stuck in progress | Blockers or scope creep | Identify and resolve blockers |
| Velocity dropping | Context switching or burnout | Focus on fewer tasks at once |
| High creation, low completion | Planning without execution | Stop adding until caught up |
