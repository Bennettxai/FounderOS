# Operations Spec

Operations describe how work moves through Optimal Engine.

## Work Setup

```text
Create or initiate Workspace
  -> define Nodes and relationships
  -> grant humans/agents/tools
  -> configure routing rules and policies
  -> render initial projections
```

## Intake Operation

```text
input
  -> preserve Source Package
  -> classify Signal
  -> route to workspace/node
  -> extract Claims
  -> write derivation ledger
  -> queue review
  -> update projections/indexes
```

## Recall Operation

```text
request
  -> authenticate actor
  -> build authorization envelope
  -> plan retrieval
  -> assemble Context Package
  -> return source-linked context
  -> record audit
```

## Agent Action Operation

```text
task
  -> open or load Active Memory Pool
  -> load Context Package
  -> request governed tool/model call
  -> validate input/output
  -> record run
  -> publish observation
  -> create pending Claim when useful
```

## Agent Collaboration Operation

Use agent-to-agent collaboration when another agent is the right actor.

```text
task
  -> load Context Package
  -> resolve remote agent definition and Agent Card
  -> check delegation policy and partition grants
  -> send A2A task request
  -> receive progress, response, or artifact
  -> preserve useful output as Source Package or observation
  -> create pending Claims when knowledge-bearing
  -> record delegation audit
```

## Workflow Promotion

```text
repeated episodes
  -> Workflow Traces
  -> Generalized Workflow
  -> Procedural Memory Object
  -> Skill Package
  -> execution and validation records
```

## Review Gates

| Gate | Required before |
| --- | --- |
| Topology review | Agent-proposed Nodes/relationships become durable topology. |
| Claim review | Extracted assertions become Facts. |
| Tool grant review | Agents call external tools with write capability. |
| Agent delegation review | Agents delegate work to another agent or accept returned artifacts. |
| Workflow review | Repeated traces become reusable procedures. |
| Skill enablement review | Skill Packages become executable guidance. |
