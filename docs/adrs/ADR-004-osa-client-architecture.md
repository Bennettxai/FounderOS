# ADR-004: OSA Client Architecture

## Status
Accepted

## Date
2026-01-24

## Context
BusinessOS needs to integrate with OSA-5, the 21-agent orchestration system that generates complete applications. OSA-5 provides a REST API for triggering app generation, checking status, and managing workspaces.

The question arose: Should we use the official **Claude Agent SDK** (TypeScript/Python) or build a **custom Go HTTP client** for OSA integration?

### Reactive vs Proactive Agent Distinction

This ADR was written before the formal reactive/proactive architecture split was established.
That split is now material to understanding how the OSA client fits in the system:

**Reactive agents** (BusinessOS, this repository) respond to user messages. The custom Go
HTTP client described in this ADR is the mechanism by which the reactive BOS backend calls
the OSA-5 generation API — a classic synchronous request/response interaction. The client
lives at `internal/integrations/osa/client.go` and is appropriate for reactive use: user
triggers generation → BOS calls OSA-5 → result returned and streamed.

**Proactive agents** (OSA engine, `osa-engine/`, premium Miosa.ai) initiate actions without
user input. A proactive agent does not use a request/response HTTP client as its primary
interface — it monitors signals and emits events. When a proactive agent needs to delegate
work to the OSA-5 generation system, it may use a similar client pattern, but the invocation
is triggered by the agent's own reasoning, not by a user message.

The decision to build a custom Go HTTP client (rather than use the Claude Agent SDK) remains
correct for the reactive use case. For the proactive layer, additional client patterns may
be needed that are designed for event-driven, long-running agent processes rather than
synchronous API calls.

**See also:** [ARCHITECTURE-SEPARATION.md](../ARCHITECTURE-SEPARATION.md)

Key considerations:
- OSA-5 is a separate service with its own REST API
- BusinessOS backend is Go-based (not TypeScript/Python)
- Need resilient communication (retry logic, circuit breakers)
- Must integrate cleanly with existing Go architecture
- Claude Agent SDK is designed for autonomous agents (not API client libraries)

## Decision
Build a **custom Go HTTP client library** specifically for OSA-5 integration, NOT use Claude Agent SDK.

### Implementation Details

**Custom OSA Client:**
- Location: `/desktop/backend-go/internal/integrations/osa/client.go`
- Size: 220 lines (production-ready)
- Language: Go (matches BusinessOS backend)

**Features:**
```go
type Client struct {
    config     *Config
    httpClient *http.Client
}

// Core Methods
func (c *Client) Orchestrate(ctx, req) (*OrchestrateResponse, error)
func (c *Client) GetAppStatus(ctx, appID, userID) (*AppStatusResponse, error)
func (c *Client) GetWorkspaces(ctx, userID) (*WorkspacesResponse, error)
func (c *Client) HealthCheck(ctx) (*HealthResponse, error)
```

**Resilience:**
- Retry logic with configurable max retries
- Configurable retry delay
- Context timeout support
- Circuit breaker pattern (via resilient_client.go)
- Graceful error handling

**Authentication:**
- JWT token generation with shared secret
- User ID and Workspace ID in headers
- `Authorization: Bearer <token>` pattern

**API Endpoints:**
```
POST /api/orchestrate          - Generate applications with 21-agent system
GET  /api/apps/:id/status      - Check generation status
GET  /api/workspaces           - List user workspaces
GET  /health                   - Health check
```

### Why NOT Claude Agent SDK

**1. Language Mismatch**
- Claude Agent SDK: TypeScript (official) or Python (official)
- BusinessOS Backend: Go
- Community Go SDK exists but is unofficial and less mature

**2. Wrong Use Case**
- Agent SDK is for building autonomous agents with tool loops
- BusinessOS needs a simple HTTP client to call OSA API
- OSA-5 itself uses Agent SDK internally - BusinessOS just calls its API

**3. Architectural Complexity**
Using Agent SDK would require either:
- **Option A**: TypeScript microservice (adds Node.js dependency)
- **Option B**: Community Go SDK (unofficial, maintenance burden)
- **Option C**: Direct API calls with custom client (CHOSEN)

**4. Performance**
- Custom HTTP client: Direct connection, minimal overhead
- Agent SDK: Additional abstraction layer, slower
- Latency matters: App generation API is user-facing

**5. Control**
- Custom client: Full control over retry logic, timeouts, error handling
- Agent SDK: Opinionated design may not match our needs

## Consequences

### Positive
- **Go-native**: No cross-language overhead, matches tech stack
- **Simple**: Straightforward HTTP client, easy to understand
- **Performant**: Direct HTTP calls, no SDK abstraction
- **Maintainable**: 220 lines of code, fully under our control
- **Flexible**: Can customize retry logic, timeouts, headers
- **Testable**: Easy to mock HTTP client for tests
- **Production-ready**: Includes resilience patterns (retry, circuit breaker)

### Negative
- **Manual maintenance**: Must update client when OSA API changes
- **No built-in features**: Don't get Agent SDK's session management, hooks, MCP integration
- **Documentation responsibility**: Must document client API ourselves

### Neutral
- Custom client is specific to OSA-5 API (not a general-purpose agent framework)
- Authentication uses JWT tokens with shared secret
- File sync via polling (webhook support could be added later)

## Alternatives Considered

### 1. Claude Agent SDK (TypeScript)
**Description:** Use `@anthropic-ai/claude-agent-sdk` with TypeScript microservice.

**Rejected because:**
- Adds Node.js/TypeScript dependency to Go-only stack
- Microservices complexity (deployment, monitoring, inter-service communication)
- Additional latency (~100ms) for cross-language RPC
- Over-engineering: OSA API is simple REST, doesn't need Agent SDK features
- Agent SDK designed for autonomous agents, not API clients

### 2. Community Go Agent SDK
**Description:** Use `github.com/schlunsen/claude-agent-sdk-go` (unofficial port).

**Rejected because:**
- Unofficial/community-maintained (less stable than Anthropic's official SDKs)
- Designed for building agents, not calling external APIs
- Adds unnecessary dependencies for simple HTTP client needs
- Our custom client is simpler and more focused

### 3. Third-Party HTTP Client Library
**Description:** Use Resty, Go-Requests, or similar HTTP wrapper.

**Rejected because:**
- Standard library `net/http` is sufficient and battle-tested
- Third-party libraries add dependencies without significant benefit
- Custom client gives exact behavior we need

### 4. gRPC Integration
**Description:** Use gRPC instead of REST for OSA communication.

**Deferred because:**
- OSA-5 currently exposes REST API, not gRPC
- Would require OSA-5 backend changes
- REST is sufficient for current scale
- Could migrate to gRPC in future if performance requires

## Implementation Status

| Component | Status |
|-----------|--------|
| OSA Client (client.go) | 100% (220 lines) |
| Resilient Client wrapper | 100% |
| Authentication (auth.go) | 100% |
| Configuration (config.go) | 100% |
| Type definitions (types.go) | 100% |
| Integration tests | 100% |
| Unit tests | 100% |
| Documentation | 100% |

## References
- `/desktop/backend-go/internal/integrations/osa/client.go` - Custom OSA HTTP client (220 lines)
- `/desktop/backend-go/internal/integrations/osa/resilient_client.go` - Circuit breaker pattern
- `/desktop/backend-go/internal/integrations/osa/auth.go` - JWT authentication
- `/desktop/backend-go/internal/integrations/osa/types.go` - Request/response types
- `/CLAUDE_AGENT_SDK_COMPREHENSIVE_RESEARCH_2026-01-24.md` - Comprehensive SDK research (127KB)
- Claude Agent SDK (TypeScript): https://github.com/anthropics/anthropic-agent-sdk
- Claude Agent SDK (Python): https://pypi.org/project/claude-agent-sdk/

---
*Decision made by: Nick*
*Reviewed by: Claude Code Architecture Agent*
*Research by: 6-agent parallel research team (2026-01-24)*
*Last Updated: 2026-02-26 — Reactive vs proactive agent distinction added*
