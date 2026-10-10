# Component Map

This page maps the runtime components to the architecture layers.

| Layer | Component responsibility | Current surface |
| --- | --- | --- |
| Workspace / Topology | Workspaces, Nodes, Node Types, relationships, memberships, setup/initiation. | `OptimalEngine.WorkspaceTopology`, topology modules, setup/initiate Mix tasks. |
| Source Intake | Preserve raw input before interpretation. | Memory Core source package services, asset store, connector preservation paths. |
| Signal Pipeline | Classify and parse input, create compatibility search rows. | Pipeline/parser/classifier modules. |
| Memory Core | Claims, Facts, Memory Objects, Relationship Edges, Derivation Ledger, validity, supersession. | `OptimalEngine.MemoryCore` modules. |
| Retrieval / Context | Query planning, context package assembly, stale refresh. | Retrieval coordinator and context package services. |
| Active Memory Pools | Task-scoped working memory, loaded context, observations. | Active pool services. |
| Workflow / Skill Runtime | Workflow traces, generalized workflows, procedures, skill packages. | Memory Core workflow/skill records and future runtime services. |
| Tool / Model Governance | Registered tools/models, permissions, calls, schemas, audit. | Tool/model definition and call-run tables. |
| Agent Collaboration | Remote agent definitions, Agent Cards, delegation records, returned artifacts, observations. | Tool/model governance spine now; dedicated A2A registry is a future hardening target. |
| Wiki / Export | Markdown/wiki/HTML/API/report projections. | Wiki service, workspace export, render/check Mix tasks. |
| Evaluation / Recovery | Reality checks, benchmarks, rebuild checks. | Evaluation modules and `mix optimal.reality_check`. |

## Component Rule

```text
Store modules persist.
Domain services decide lifecycle.
Projection services render.
Agents call public interfaces.
```

Avoid putting business meaning in low-level persistence adapters.
