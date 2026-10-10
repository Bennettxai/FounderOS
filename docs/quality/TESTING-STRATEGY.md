# Testing Strategy

> **Status:** ACTIVE
> **Owner:** Pedro (QA agent)
> **Priority:** P0

---

## Testing Pyramid

```
          ┌─────────┐
          │  E2E    │  Few, expensive, critical paths only
          ├─────────┤
          │ Integr. │  API endpoint tests, DB integration
          ├─────────┤
          │  Unit   │  Business logic, pure functions, validators
          └─────────┘
```

## Frameworks

| Layer | Backend (Go) | Frontend (SvelteKit) |
|-------|-------------|---------------------|
| Unit | `go test` (stdlib) | Vitest (configured, growing) |
| Integration | `go test` + PostgreSQL + Redis | `npm run build` (build verification) |
| E2E | Playwright (via CI) | Playwright (via CI) |
| Type checking | `go vet` | `npm run check` (svelte-check) |
| Linting | `golangci-lint` | ESLint |
| Security | Custom security test suite | npm audit |

## Running Tests

### Backend

```bash
cd desktop/backend-go

# All unit tests (skip integration)
go test -short ./...

# All tests including integration (requires DB + Redis)
go test ./...

# Specific package
go test ./internal/orchestration/
go test ./internal/services/

# With coverage
go test -cover ./...

# Verbose
go test -v ./internal/orchestration/

# With race detection
go test -race ./...

# Benchmarks
go test -bench=. ./internal/orchestration/
```

### Frontend

```bash
cd frontend

# Type-checking
npm run check

# Build verification (also serves as integration gate)
npm run build

# Unit tests (Vitest)
npm test

# Linting
npx eslint .
```

## Coverage Targets

| Component | Current | Target | Notes |
|-----------|---------|--------|-------|
| `internal/orchestration/` | ~80% | 80% | 56 tests + 36 tests |
| `internal/services/module_protection` | ~90% | 80% | 48 tests |
| `internal/services/` (other) | Measure with `go test -cover` | 70% | Priority: cover service layer |
| `internal/handlers/` | Measure with `go test -cover` | 60% | HTTP handlers, mock dependencies |
| Frontend components | Growing | 50% (Sprint 2) | Vitest + testing-library |
| Frontend stores | Growing | 70% (Sprint 3) | Critical state logic |

**How to measure current coverage:**
```bash
# Backend - summary
cd desktop/backend-go && go test -cover ./...

# Backend - detailed HTML report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html
```

## CI Test Requirements

All of the following must pass before a PR can merge:

| Gate | Command | Blocks Merge |
|------|---------|--------------|
| Backend build | `go build ./cmd/server` | Yes |
| Backend unit tests | `go test -short -race ./...` | Yes |
| Backend integration tests | `go test ./...` (with DB) | Yes |
| Frontend build | `npm run build` | Yes |
| Frontend type check | `npm run check` | Yes (deploy only) |
| Frontend tests | `npm test` (Vitest) | Yes |
| Security tests | Custom suite (SQL injection, XSS, CSRF, etc.) | Yes |
| E2E tests | Playwright suite | Yes (on main/release) |

See [CI-GATES.md](CI-GATES.md) for the full CI pipeline documentation.

## Test Database Setup

### Unit Tests (No DB Required)

Unit tests use `DATABASE_REQUIRED=false` and mock the repository layer. No database setup needed.

```go
// Example: mock repository for unit tests
type mockRepo struct{}
func (m *mockRepo) GetUser(ctx context.Context, id string) (*User, error) {
    return &User{ID: id, Name: "Test"}, nil
}
```

### Integration Tests (Full DB Required)

Integration tests run against a real PostgreSQL instance with pgvector:

```bash
# CI automatically provisions PostgreSQL + Redis via GitHub Actions services
# Local setup:
docker run -d --name test-db \
  -e POSTGRES_DB=businessos_test \
  -e POSTGRES_PASSWORD=testpass \
  -p 5433:5432 \
  pgvector/pgvector:pg15

# Apply migrations
psql "postgres://postgres:testpass@localhost:5433/businessos_test" \
  -f supabase/migrations/*.sql

# Run integration tests
DATABASE_URL="postgres://postgres:testpass@localhost:5433/businessos_test" \
  go test ./...
```

### Test Data Cleanup

Integration tests should clean up after themselves. Use `t.Cleanup()`:

```go
func TestCreateWorkspace(t *testing.T) {
    ws := createTestWorkspace(t, db)
    t.Cleanup(func() {
        db.Exec("DELETE FROM workspaces WHERE id = $1", ws.ID)
    })
    // ... test logic
}
```

## Writing New Tests

### Go Test Patterns

```go
// Follow existing patterns in the codebase:
func TestFunctionName_Scenario(t *testing.T) {
    // Arrange
    // Act
    // Assert
}

// Table-driven tests for multiple cases:
func TestFunctionName(t *testing.T) {
    tests := []struct {
        name     string
        input    Type
        expected Type
    }{
        {"case 1", input1, expected1},
        {"case 2", input2, expected2},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // test
        })
    }
}
```

### What Needs Tests (Priority Order)

1. **Orchestration layer** — COT pipeline, app generation, SSE streaming
2. **Service layer** — module protection, memory hierarchy, agent behavior
3. **Handler layer** — request validation, error responses, auth checks
4. **Frontend stores** — workspace state, auth state, app generation state

## E2E Test Plan

Critical E2E paths (implemented in Playwright, run in CI):

| Path | Priority | Status |
|------|----------|--------|
| Login → Create workspace → Generate app → View in editor | P0 | In CI |
| Login → Chat with OSA → Get response | P0 | In CI |
| Login → Execute SORX skill → See result | P1 | Sprint 2 target |
| App generation → SSE progress → Files persist → Open in editor | P0 | In CI |

E2E tests run against a full stack: backend (Go, port 8001), frontend (SvelteKit), PostgreSQL, and Playwright browser automation.

## Load Testing

Load testing is a **Sprint 5 deliverable**. Target metrics:

| Metric | Target |
|--------|--------|
| Concurrent users | 100 |
| API throughput | 50 RPS |
| Concurrent app generations | 10 |
| Tool | k6 (already configured in CI) |

See [LOAD-TESTING.md](LOAD-TESTING.md) for the full load testing plan.

---

**Last Updated:** 2026-02-23
