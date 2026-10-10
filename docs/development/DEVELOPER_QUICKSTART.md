# Developer Quick Start

This guide is the canonical path for running BusinessOS from source.

## First Run

Install the prerequisites listed in the root [README](../../README.md).

Then run:

```bash
git clone https://github.com/Miosa-osa/BusinessOS.git
cd BusinessOS
make onboard
```

Do not manually create a PostgreSQL cluster, database, Redis service, or migration ledger.
The onboarding command owns those steps and verifies their result.

## What Onboarding Creates

The command creates these ignored local files only when they are missing:

```text
.env.dev
desktop/backend-go/.env
frontend/.env.development.local
.run/
```

Existing environment files and data are never replaced.
On a new database, the backend installs the canonical schema, establishes the historical migration baseline, and then applies current repair migrations.
Developers must not perform this sequence manually.

The generated local profile uses:

```text
Backend         8801
Frontend        5273
PostgreSQL     25432
Redis          26379
Optimal Engine  4200
```

## Confirm the Result

```bash
make dev-local-status
make dev-local-verify
```

Verification fails unless all of these conditions are true:

- PostgreSQL accepts connections.
- Redis responds.
- The backend readiness endpoint passes.
- The complete schema contract passes.
- The frontend responds.
- Electron is running.
- Optimal Engine responds.

## Normal Workflow

```bash
git pull --ff-only
make dev-local-restart
make dev-local-verify
```

Use:

```bash
make dev-local-logs
```

Logs are stored under `.run/`.

## Environment Changes

Edit `.env.dev` for local ports.
Restart after changing it:

```bash
make dev-local-restart
```

Use `desktop/backend-go/.env` for optional credentials and provider configuration.
Never commit that file.

Google authentication and integrations require credentials from the team.
Local email and password authentication does not.
The application and its default local modules can be developed without Google credentials.

## Database Changes

Add schema changes as a new numbered migration under:

```text
desktop/backend-go/internal/database/migrations/
```

The backend applies migrations and records the ledger.
The launcher must not apply individual migrations.

Validate changes with:

```bash
make check-schema
make db-schema-check
cd desktop/backend-go && go test ./internal/database/... -count=1
```

Never pipe `schema.sql` into an existing developer database.

## Frontend Changes

The frontend uses pnpm.

```bash
cd frontend
pnpm check
pnpm test
pnpm build
```

Do not generate or commit an npm lockfile in `frontend/`.

## Backend Changes

```bash
cd desktop/backend-go
go test ./cmd/server ./internal/database ./internal/database/schemahealth -short -count=1
go test ./path/to/changed/package -short -count=1
go build ./cmd/server
```

Integration tests require an explicitly disposable PostgreSQL database:

```bash
TEST_DATABASE_URL='postgres://postgres:postgres@localhost:5432/businessos_test?sslmode=disable' \
  go test ./... -count=1
```

Never point `TEST_DATABASE_URL` at a developer or production database.
Some migration tests intentionally alter constraints and therefore require a database that can be discarded after the run.
The broad handler and workflow suites still contain legacy fixture assumptions and are not part of the onboarding acceptance gate.

The local launcher rebuilds the backend when Go files, dependency metadata, or embedded SQL migrations are newer than `.run/backend`.

## Desktop Changes

The Electron package intentionally uses npm.

```bash
cd desktop
npm install
npm test
```

Run `make dev-local-restart` to rebuild the Electron main and preload bundles.

## Troubleshooting

Start with:

```bash
make dev-local-status
make dev-local-verify
```

Then inspect:

```text
.run/backend.log
.run/frontend.log
.run/electron.log
.run/optimal-engine.log
.run/pgdev.log
.run/redis.log
```

Do not delete data or replay migrations before identifying the failed service.

See [Common Issues](COMMON_ISSUES.md) and the [Local Runtime SOP](../operations/businessos-local-runtime-sop.md) for recovery procedures.
