# Architecture Decision Records (ADRs)

## About

This directory contains Architecture Decision Records (ADRs) for BusinessOS. ADRs document significant architectural and design decisions made during the development of the system.

## What is an ADR?

An Architecture Decision Record (ADR) is a document that captures:
- **Context:** What situation led to this decision
- **Options Considered:** Alternative approaches evaluated
- **Decision:** What was chosen and why
- **Consequences:** Trade-offs, benefits, and risks

## Why ADRs?

- **Documentation:** Preserve reasoning behind key decisions
- **Onboarding:** Help new team members understand system design
- **Accountability:** Create a paper trail for design choices
- **Learning:** Review past decisions and their outcomes
- **Preventing Re-litigation:** Avoid revisiting settled decisions

## ADR Format

Each ADR follows this structure:

1. **Status:** Proposed | Accepted | Deprecated | Superseded
2. **Date:** When the decision was made
3. **Context:** Problem statement and background
4. **Options Considered:** Alternative approaches
5. **Decision:** What was chosen
6. **Implementation:** How it's implemented
7. **Consequences:** Positive and negative outcomes
8. **Related Decisions:** Links to other ADRs

## Current ADRs

| ADR | Title | Status | Date | Topics |
|-----|-------|--------|------|--------|
| [001](001-database-isolation-strategy.md) | Database Isolation Strategy | Accepted | Jan 2026 | Multi-tenancy, Workspace isolation, Memory hierarchy |
| [002](002-app-isolation-approach.md) | App Isolation Approach | Accepted | Jan 2026 | Container security, Docker, User-generated apps |
| [003](../CARRIER-DEPLOYMENT-PLAN.md#13-adr-003-rabbitmq-broker-selection-for-carrier) | RabbitMQ Broker Selection for CARRIER | Accepted | Feb 2026 | AMQP, RabbitMQ, CARRIER bridge, SorxMain, messaging |

## Key Decisions by Area

### Security & Isolation
- **ADR-001:** Workspace-level row isolation for multi-tenancy
- **ADR-002:** Docker containers for app execution sandboxing

### Data Architecture
- **ADR-001:** 3-tier memory hierarchy (workspace/private/shared)

### Infrastructure
- **ADR-002:** Self-hosted Docker over cloud sandboxing (E2B)

## How to Create a New ADR

1. **Create a new file:** `003-your-decision-title.md`
2. **Use the template below:**

```markdown
# ADR-XXX: [Title]

## Status
[Proposed | Accepted | Deprecated | Superseded]

## Date
[YYYY-MM-DD]

## Context
[What is the problem/situation?]

## Options Considered

### Option 1: [Name]
- **Pros:**
- **Cons:**

### Option 2: [Name]
- **Pros:**
- **Cons:**

## Decision
[What was chosen and why?]

## Implementation
[How is it implemented? Which files?]

## Consequences

### Positive
- [Benefit 1]
- [Benefit 2]

### Negative
- [Trade-off 1]
- [Trade-off 2]

## Related Decisions
- ADR-XXX: [Title]

## References
- [Link 1]
- [Link 2]
```

3. **Update this README** with the new ADR in the table above

## Best Practices

### When to Write an ADR

Create an ADR for decisions that:
- **Impact system architecture:** Database design, service boundaries, API contracts
- **Affect scalability:** Caching strategy, data partitioning, load balancing
- **Have security implications:** Authentication, authorization, encryption
- **Introduce new technologies:** New frameworks, databases, or tools
- **Change existing patterns:** Refactoring major components

### When NOT to Write an ADR

Skip ADRs for:
- **Routine code changes:** Bug fixes, minor refactoring
- **UI polish:** Color schemes, button placement
- **Obvious decisions:** Using JSON for API responses
- **Temporary solutions:** Quick hacks or POCs

### ADR Lifecycle

```
Proposed → Accepted → [Implemented] → [Superseded/Deprecated]
```

- **Proposed:** Under discussion, not yet approved
- **Accepted:** Decision made, ready to implement
- **Deprecated:** No longer recommended, but still in use
- **Superseded:** Replaced by a newer ADR

### Updating ADRs

- **Never delete ADRs** (preserve history)
- **Mark as Superseded** if replaced by a new decision
- **Link to the new ADR** in the Superseded notice
- **Add post-implementation notes** in a "Lessons Learned" section

## Additional Resources

- [ADR GitHub Organization](https://adr.github.io/)
- [Documenting Architecture Decisions](https://cognitect.com/blog/2011/11/15/documenting-architecture-decisions)
- [Architecture Decision Records Template](https://github.com/joelparkerhenderson/architecture-decision-record)

## Contact

For questions about ADRs or architectural decisions, please:
- Open a GitHub issue
- Discuss in architecture review meetings
- Contact the backend team lead

---

**Last Updated:** January 2026
**Maintained By:** BusinessOS Backend Team
