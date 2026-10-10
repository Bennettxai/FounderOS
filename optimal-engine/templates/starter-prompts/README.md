# Starter Prompts

These prompts are for humans using Codex, Claude Code, ChatGPT, an MCP client,
or another agent with access to their files.

Use them when a person wants to turn an existing messy workspace into an Optimal
Engine workspace.

## How To Use

1. Pick the prompt that matches the job.
2. Paste it into the agent.
3. Attach or point the agent at the relevant files/folders/exports.
4. Let the agent create a setup dump or import plan.
5. Run `mix optimal.initiate` or `mix optimal.setup`.
6. Review proposed Nodes, integrations, packages, and open questions before
   accepting durable structure.

The agent should preserve evidence first, then propose structure. It should not
silently rewrite the user's workspace.

## Prompts

| Prompt | Use when |
| --- | --- |
| `workspace-initiation.md` | The user wants to start from a messy explanation, pasted notes, or a folder of files. |
| `company-wiki-import.md` | The user already has a company wiki, Notion/Confluence export, Obsidian vault, markdown folder, or document dump. |
| `package-inventory.md` | The user wants to define recurring things they send out, such as proposals, contracts, SOPs, reports, and client requirement documents. |
| `agentic-loop-design.md` | The user wants a repeatable loop with phases, validation gates, stop conditions, and memory/audit outputs. |
| `youtube-learning-import.md` | The user wants to save a video, transcript, lecture, podcast, or media source into a learning/research workspace. |
| `interface-and-publishing-plan.md` | The user wants an app, dashboard, client portal, static page, public link, package delivery flow, or deployment surface on top of the engine. |
