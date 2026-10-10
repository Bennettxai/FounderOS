# Vocabulary

Use product terms consistently.

| Term | Meaning |
| --- | --- |
| Tenant / Organization | Governance boundary for one person, team, or company. |
| Workspace | Bounded operating area inside an organization. |
| Node | Governed unit of context, purpose, relationships, state, and activity. |
| Source Package | Preserved raw input with metadata, hash, scope, and policy. |
| Signal | Classified interpretation of source material. |
| Claim | Unaccepted assertion extracted from a source or observation. |
| Fact | Reviewed/policy-accepted assertion. |
| Memory Object | Source-backed meaning around Facts, Claims, sources, and relationships. |
| Context Package | Authorized bundle of context for a query or task. |
| Active Memory Pool | Task-scoped working memory shared by humans and agents. |
| Workflow Trace | Evidence-linked trace of repeated or meaningful work. |
| Skill Package | Governed reusable procedure with policy, tools, checks, and audit. |
| Delivery Package | Receiver/channel bundle assembled from multiple files or objects, often zipped for delivery. |
| Projection | Rebuildable surface such as markdown, wiki, HTML, app view, API response, or agent prompt. |
| Alias | Scoped user-facing name that resolves to a canonical engine object. |
| Display Label | User or domain wording for a canonical type, such as calling a Project Node an initiative, campaign, deal, or case. |
| Stable ID | Machine-readable identity that survives renames and alias changes. |

## Domain Translation

Different users may prefer different surface words:

| Engine term | Business-facing wording | Personal-facing wording | Technical wording |
| --- | --- | --- | --- |
| Node | team, project, account, function | life area, project, person | namespace, module, scope |
| Signal | update, note, event | thought, note, observation | event, payload |
| Memory Object | company memory | remembered context | durable context object |
| Context Package | briefing packet | prep packet | authorized context bundle |
| Skill Package | SOP/playbook | routine | executable procedure |
| Delivery Package | client packet, handoff, board packet | trip packet, prep bundle | export bundle, zip artifact |

The engine can render friendly language, but internal architecture should use
the canonical terms above.

## Naming Rules

```text
Organization owns governance.
Workspace owns an operating context.
Node owns a unit of context/activity inside a workspace.
Project is a Node type.
Folder is a projection of a Node.
```

Different users can use different labels:

| User word | Canonical engine mapping |
| --- | --- |
| initiative | Project Node, unless the workspace defines a more specific type. |
| campaign | Project Node or custom campaign subtype. |
| deal | Project Node, account Node, or opportunity subtype depending on workspace vocabulary. |
| case | Project Node, support Node, legal Node, or custom subtype depending on workspace vocabulary. |
| account | Customer/account Node, or Project Node when the workspace defines accounts as project-like. |
| area | Workspace or Node depending on whether it has its own node graph and policy boundary. |

The engine should preserve the user's word as an alias or display label while
linking it to a stable canonical object.

## Terms To Keep Separate

| Pair | Distinction |
| --- | --- |
| Source Package / Signal | Source is preserved evidence. Signal is classification and interpretation. |
| Claim / Fact | Claim is unaccepted. Fact is reviewed or policy-accepted. |
| Memory Object / Context Package | Memory is durable meaning. Context Package is task/query packaging. |
| Genre / Format | Genre is the expected kind of communication. Format is the container. |
| Format / Structure | Format carries content. Structure is the internal skeleton. |
| Active Memory Pool / Workspace | Pool is temporary task working state. Workspace is durable operating topology. |
| Delivery Package / Skill Package | Delivery Package is sent to a receiver/channel. Skill Package is an approved reusable procedure. |
