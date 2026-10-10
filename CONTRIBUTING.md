# Contributing to BusinessOS

## Set Up

Follow the [Developer Quick Start](docs/development/DEVELOPER_QUICKSTART.md).

```bash
make onboard
```

Do not create an alternative local stack or manually apply database migrations.

## Branches

Create a focused branch from the intended base branch.

```bash
git switch -c type/short-description
```

Keep changes scoped.
Do not include generated output, runtime data, environment files, or unrelated formatting.

## Validation

Run the checks that match the changed area.

```bash
make test-dev-local-launcher
make test-backend
make test-frontend
make check-schema
make dev-local-verify
```

Frontend:

```bash
cd frontend
pnpm check
pnpm test
pnpm build
```

Backend:

```bash
cd desktop/backend-go
go test ./... -count=1
```

Desktop:

```bash
cd desktop
npm test
```

## Database Rules

The backend owns migration execution and the migration ledger.

- Add a new numbered migration.
- Never edit an applied migration.
- Never replay `schema.sql` against an existing database.
- Verify both clean construction and runtime schema health.

## Package Managers

- `frontend/` uses pnpm and `pnpm-lock.yaml`.
- `desktop/` uses npm and `package-lock.json`.
- Optimal Engine subprojects use the package manager declared by that subproject.
- The repository root is not a JavaScript package.

## Commit Messages

Use a short imperative summary.

Examples:

```text
fix: preserve workspace selection during refresh
feat: add engine connection diagnostics
docs: replace manual local database setup
```

Do not add automated agent co-author lines.
Do not edit generated changelogs manually.

## Architecture

Read these before changing ownership boundaries:

- [BusinessOS Mission](MISSION.md)
- [Workspace and Engine Routing](docs/WORKSPACE_ENGINE_ROUTING.md)
- [Documentation Index](docs/README.md)
- [Optimal Engine Architecture](docs/architecture/OSA-ARCHITECTURE.md)

BusinessOS owns operational interfaces and workspace interaction.
Optimal Engine owns governed knowledge, memory, search, and context.

## Security

Never commit credentials, tokens, personal workspace data, database dumps, or local runtime state.

Private workspace and delivery material must remain excluded from the open-source mirror.
Review `.oss-exclude` when adding a new private top-level data area.
