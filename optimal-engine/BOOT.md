# Optimal Engine Boot Guide

This is the human and agent boot path for a fresh Optimal Engine checkout.

## Install

```bash
brew install snappy
make install
make bootstrap
```

## Start The Engine

```bash
make dev
```

The default local API is:

```text
http://localhost:4200
```

Verify it:

```bash
curl http://localhost:4200/api/health
bin/optimal doctor
bin/optimal reality-check
```

## First Workspace

Create a workspace when you know the shape:

```bash
bin/optimal setup my-workspace --name "My Workspace"
```

Create a workspace from messy context:

```bash
bin/optimal initiate my-workspace --name "My Workspace" --dump setup.md
```

Inspect it:

```bash
bin/optimal topology --workspace default:my-workspace
```

## Agent Boot Loop

Agents should use this loop before doing meaningful work:

```bash
bin/optimal boot
bin/optimal find "current state" --workspace default:my-workspace
bin/optimal rag "what context should I know?" --workspace default:my-workspace
```

Then capture important new evidence:

```bash
bin/optimal capture "raw signal or evidence" --workspace default:my-workspace
bin/optimal aware "important correction or durable signal" --workspace default:my-workspace
bin/optimal close "what changed and how verified"
```

## What Not To Commit

Never commit local runtime data:

```text
.optimal/
*.db
*.db-wal
*.db-shm
connector keys
private imported workspaces
private source packages
private cache data
```

Each clone gets its own local memory store.
The public repo ships the engine code and safe public examples, not Roberto's private memory.
