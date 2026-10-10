# Error Recovery Prompts

Standard prompts for handling errors gracefully.

---

## Widget Type Not Found

```
"{requested_type}" isn't an available widget. 

{if alternatives}
Similar options:
{foreach alternative}
- **{name}** - {description}
{/foreach}

Which would work for you?
{else}
Available widgets include: task summary, task list, burndown chart, deadlines, project progress, and metric cards.

What are you trying to see?
{/if}
```

---

## Dashboard Full

```
Your "{dashboard_name}" dashboard has {current}/{max} widgets (the maximum).

I can:
1. **Remove a widget** to make room
2. **Create a new dashboard** for additional widgets

What would you prefer?

{if list_widgets}
Current widgets:
{foreach widget}
- {type_display_name}
{/foreach}
{/if}
```

---

## Dashboard Not Found

```
I don't see a dashboard called "{requested_name}".

Your dashboards:
{foreach dashboard}
- **{name}**{if is_default} (default){/if}
{/foreach}

Would you like me to create "{requested_name}" as a new dashboard?
```

---

## Dashboard Name Exists

```
You already have a dashboard called "{name}".

Would you like to:
1. **Add widgets** to the existing "{name}" dashboard
2. **Create a new dashboard** with a different name

What works best?
```

---

## Widget Not Found on Dashboard

```
I couldn't find that widget on your dashboard.

"{dashboard_name}" currently has:
{foreach widget}
- **{type_display_name}**{if config} ({config_summary}){/if}
{/foreach}

Which one did you want to {action}?
```

---

## Invalid Config Value

```
"{value}" isn't a valid option for {field}.

{if allowed_values}
Valid options are:
{foreach option}
- **{option}**{if description} - {description}{/if}
{/foreach}
{/if}

{if range}
Value must be between {min} and {max}.
{/if}

What would you like to use?
```

---

## Project Not Found

```
I couldn't find "{project_name}" in your workspace.

Your active projects:
{foreach project}
- **{name}**
{/foreach}

Did you mean one of these, or would you like to create "{project_name}"?
```

---

## Permission Denied

```
{if workspace_template}
You can't modify the workspace template "{dashboard_name}" - that requires admin access.

I can create a personal copy that you can customize however you'd like. Want me to do that?
{else}
You don't have permission to {action}. 

{if alternative}
However, you can {alternative_action}.
{/if}
{/if}
```

---

## Rate Limited

```
You're making changes quickly! Let me catch up.

{if retry_after}
Try again in {retry_after} seconds.
{/if}

Tip: If you have multiple changes, tell me all at once and I'll do them together.
```

---

## Internal Error

```
Something went wrong on my end. Let me try that again.

{if error_id}
(Error reference: {error_id})
{/if}

{if retry_suggestion}
In the meantime, you can {retry_suggestion}.
{/if}
```

---

## Partial Success (Batch Operations)

```
I completed {success_count} of {total_count} changes:

✓ Completed:
{foreach success}
- {description}
{/foreach}

✗ Failed:
{foreach failure}
- {description}: {error_message}
{/foreach}

Would you like me to retry the failed {failure_type}?
```

---

## Generic Fallback

When error doesn't match specific patterns:

```
I ran into an issue: {error_message}

{if suggestion}
Try: {suggestion}
{/if}

{if can_retry}
Want me to try again?
{else}
Is there something else I can help you with?
{/if}
```
