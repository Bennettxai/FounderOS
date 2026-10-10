# BusinessOS Local Runtime SOP

## Purpose

Use this procedure whenever BusinessOS must run locally in Electron for development, demos, or client work.

The success condition is not that a browser page or process appears open.

The success condition is that the database, Redis, backend, frontend, Electron, and Optimal Engine all pass their readiness checks together.

## Normal Start

For a clean checkout:

```bash
make onboard
```

For later starts:

```bash
make dev-local
make dev-local-verify
```

`make dev-local` uses `scripts/dev-local.sh` as the only supported local launcher.

The launcher initializes or verifies an isolated PostgreSQL cluster at port `25432`.

It creates the configured local database only when the local profile is new.

It also starts isolated Redis, uses an existing Optimal Engine on port `4200` when available, or starts the bundled clean engine.

The backend is the only owner of database initialization and migration state.

When the configured database is empty, the backend installs the canonical schema before establishing the historical migration baseline.

It then applies migrations newer than that baseline normally.

Developers must not perform this sequence manually.

`make dev-local-verify` is the acceptance gate.

It fails unless all of the following are true:

- PostgreSQL is reachable at the configured local URL.
- The configured local database exists.
- Backend `/readyz` confirms both database and Redis readiness.
- The current schema contract passes.
- The frontend is reachable.
- Electron is running.
- Optimal Engine is reachable on port `4200`.

## Expected Local Services

| Service | Expected endpoint or state |
| --- | --- |
| PostgreSQL | `localhost:25432`, default database `businessos_dev` |
| Redis | `localhost:26379` and backend `/readyz` reports ready |
| Backend | `http://localhost:8801/readyz` |
| Frontend | `http://localhost:5273/` |
| Electron | PID managed under `.run/electron.pid` |
| Optimal Engine | `http://localhost:4200/api/health` |

## Google Sign-In

If Google sign-in fails while the database is unavailable, do not reuse the callback page.

Google authorization codes are single-use.

First restore readiness with `make dev-local-verify`, then return to BusinessOS and start a fresh Google sign-in.

## Failure Procedure

1. Run `make dev-local-verify`.
2. If PostgreSQL is unavailable, review `.run/pgdev.log`.
3. If a previously initialized database is missing, stop and inspect `.run/pgdata` before creating or restoring anything.
4. If schema verification fails, inspect the backend migration error and take a database backup before any recovery write.
5. Never run `internal/database/schema.sql` against an existing populated local database.
6. Never apply a single migration manually through the shared `schema_migrations` ledger before backend startup.
7. After a repair, run `make dev-local-restart` followed by `make dev-local-verify`.
8. If modules disappear after a workspace switch, inspect the workspace `module_profile` and the schema contract before changing frontend routing.
9. Never infer that workspace data is missing from a hidden module or an HTTP 500.
10. Count the workspace-scoped records directly before any recovery write.

## Non-Negotiable Rules

- Use `make onboard` for the first run.
- Use `make dev-local` or `./run-local.sh start` for later runs.
- `run-local.sh` is now a compatibility shim that delegates to the canonical launcher.
- Do not use the retired hard-coded launcher implementation or a different PostgreSQL port.
- Treat `/healthz` as liveness only.
- Treat `/readyz` plus the schema contract as the proof that BusinessOS is usable.
- Do not reset, drop, or replace workspace data to resolve a startup failure.

## Logs And Recovery Evidence

```bash
make dev-local-status
make dev-local-logs
curl http://localhost:8801/readyz
```

Store any recovery backup in `.run/backups/` and record the root cause and verification in `NOTES.md` before closing the incident.

## Workspace Switch Acceptance

For a workspace-specific module incident, readiness alone is insufficient.

The acceptance check must:

1. Authenticate through a real browser session.
2. Start in a different accessible workspace.
3. Switch into the affected workspace from the workspace picker.
4. Confirm that the expected module profile is visible.
5. Open every enabled module route.
6. Fail on any HTTP 500 or browser page error.
7. Confirm the workspace-scoped record counts were preserved.
8. Confirm every module request carries the workspace ID currently shown in the picker.
9. Rapidly switch workspaces and verify that an older response cannot replace the active workspace's data or error state.

## Workspace Request Ordering

The selected workspace must be persisted before `currentWorkspace` is published to reactive module subscribers.

Modules that load immediately on workspace changes should pass the triggering workspace ID explicitly to their API functions.

Any module with overlapping asynchronous loads must use a request generation or cancellation mechanism so stale responses cannot update the current screen.

Do not combine a reactive workspace store for rendering with a delayed localStorage value for request scoping.
