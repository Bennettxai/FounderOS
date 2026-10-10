# Company Wiki Import Prompt

Use this when I already have a wiki, docs folder, markdown vault, Notion export,
Confluence export, or old knowledge base.

## Goal

Turn the old knowledge system into an Optimal Engine import plan without
flattening everything into generic pages.

## Inputs

I may provide:

- exported markdown
- HTML pages
- PDFs
- document folders
- CSV exports
- a zip file
- links to docs
- a company wiki root folder

## Your Job

1. Inventory the top-level areas.
2. Decide which areas look like Workspaces, Nodes, packages, exports, or raw
   sources.
3. Identify pages that mix facts, tasks, decisions, and notes.
4. Identify durable decisions and open questions.
5. Identify recurring workflows or SOPs.
6. Identify package examples such as proposals, contracts, reports, onboarding
   packets, requirements documents, and handoff packets.
7. Propose a safe import order.

## Mapping Rules

```text
old wiki page
  -> Source Package
  -> Signal
  -> candidate Claims
  -> reviewed Facts / Memory Objects
  -> rebuilt wiki projection
```

```text
old folder
  -> candidate Workspace or Node
  -> review before creating durable topology
```

```text
old generated document
  -> Source Package if it is evidence
  -> package/export if it was a sent bundle or rendered view
```

## Output

Return:

1. Import summary.
2. Candidate Workspace and Node structure.
3. Pages/files that should become Source Packages.
4. Pages/files that look like package examples.
5. Pages/files that look like SOPs or workflows.
6. Open questions before import.
7. A staged import plan.

