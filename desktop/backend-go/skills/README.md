# Agent Skills System

This directory contains all agent skills for BusinessOS.

## What is a Skill?

A skill is a **set of instructions** (not code) that teaches the agent WHEN and HOW to use a specific tool. Skills are written in Markdown and loaded on-demand to save context tokens.

## Architecture

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                        SKILLS VS TOOLS                                       │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│   SKILL (Markdown)                      TOOL (Go Code)                       │
│   ────────────────                      ──────────────                       │
│   skills/dashboard-management/          internal/tools/dashboard_tool.go     │
│   └── SKILL.md                          └── ConfigureDashboardTool           │
│                                                                              │
│   Teaches WHEN/HOW to use tool          Actually DOES the work               │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

## Directory Structure

```
skills/
├── skills.yaml                      # Global configuration
├── README.md                        # This file
│
├── dashboard-management/            # Full implementation
│   ├── SKILL.md                     # Core instructions
│   ├── references/                  # Detailed documentation
│   │   ├── WIDGETS.md              
│   │   ├── CONFIGS.md              
│   │   ├── ERRORS.md               
│   │   └── EXAMPLES.md             
│   ├── schemas/                     # JSON Schema validation
│   │   ├── input.schema.json       
│   │   └── widget_configs/         
│   ├── tests/                       # Skill validation
│   │   ├── matching.yaml           
│   │   ├── tool_calls.yaml         
│   │   └── edge_cases.yaml         
│   └── prompts/                     # Response templates
│       ├── clarification.md        
│       └── error_recovery.md       
│
├── analytics-insights/              # Scaffolded
│   ├── SKILL.md
│   ├── references/
│   └── tests/
│
├── task-management/                 # Scaffolded
│   ├── SKILL.md
│   ├── references/
│   └── tests/
│
├── project-management/              # Scaffolded
│   ├── SKILL.md
│   └── tests/
│
└── notification-management/         # Scaffolded
    ├── SKILL.md
    └── tests/
```

## Skill File Purposes

| File | Purpose | When Loaded |
|------|---------|-------------|
| `SKILL.md` | Core instructions for agent | On skill match |
| `references/*.md` | Detailed documentation | On-demand |
| `schemas/*.json` | Input validation | Every tool call |
| `tests/*.yaml` | Automated validation | CI/CD |
| `prompts/*.md` | Response templates | As needed |

## Progressive Loading

To save context tokens, skills use three levels:

1. **Discovery** (~50 tokens/skill) - Always in system prompt
2. **Activation** (~1500 tokens) - Full SKILL.md on match
3. **References** (~500 tokens each) - On-demand details

## Adding a New Skill

1. Create folder: `skills/my-skill/`
2. Add `SKILL.md` with frontmatter and instructions
3. Add to `skills.yaml`
4. Create matching tool in `internal/tools/`
5. Add tests in `tests/matching.yaml`

## Testing Skills

```bash
# Validate all skills
go run cmd/validate-skills/main.go

# Test specific skill
go run cmd/validate-skills/main.go --skill dashboard-management
```

## Related Documentation

- `docs/AGENT_SKILLS_OVERVIEW.md` - Architecture details
- `docs/AGENT_SKILLS_TASK_LIST.md` - Implementation checklist
- `docs/DASHBOARD_AGENT_TOOL.md` - Dashboard tool specification
