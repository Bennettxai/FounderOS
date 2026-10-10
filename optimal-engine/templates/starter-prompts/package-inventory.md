# Package Inventory Prompt

Use this when I need to define the recurring things my organization sends to
other people.

Packages are receiver/channel bundles. They may be PDFs, docs, markdown,
spreadsheets, HTML, zip files, or a folder of multiple files.

## Goal

Create a package inventory and package template plan for my Workspace.

## Examples To Look For

- proposals
- contracts
- SOP documents
- client requirements documents
- onboarding packets
- handoff packets
- board reports
- leadership updates
- evidence packets
- sales follow-up packets
- implementation plans
- training packets
- partner update bundles

## Your Job

For each package type, identify:

```text
package type
display name
receiver
channel
stage or situation
owning Node
source Nodes
required sections
delivery formats
review requirement
examples I already have
```

## Rules

- Keep the user's name for the package when it matters.
- Do not rename a proposal into generic jargon.
- Do not put Node-owned packages at the workspace root.
- Workspace-level packages are only for bundles that intentionally span multiple
  Nodes and list those source Nodes in the manifest.
- If the package was sent externally, preserve that sent artifact as Source
  Package evidence.

## Output

Return:

1. Package inventory table.
2. Recommended owner Node for each package.
3. Suggested `package.yaml` shape for each package type.
4. Open questions before generating any files.

