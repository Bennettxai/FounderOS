# ADR-001: CRM and Clients Module Separation

## Status
Accepted

## Date
2026-01-11

## Context
BusinessOS has an existing `clients` module for basic contact management (list view, detail view, contact interactions). A comprehensive CRM schema was added in migration 041 that includes:
- `companies` table (organizations with lifecycle stages, health scores)
- `pipelines` and `pipeline_stages` tables
- `deals` table (opportunities with amounts, probabilities)
- `crm_activities` table (calls, emails, meetings, demos, etc.)
- `deal_stage_history` for analytics
- `contact_company_relations` (M:N junction table)

The question arose: Should CRM functionality be merged into the Clients module or kept as a separate module?

## Decision
Keep CRM and Clients as **separate modules** with clear boundaries and a relationship link between them via the `contact_company_relations` junction table.

### Module Definitions

**Clients Module** (Contact Management)
- Purpose: Personal/business contact directory
- Entities: contacts, contact_interactions, contact_tags
- Features: Contact list, detail view, interaction history, quick actions
- Integrations: Google Contacts, HubSpot Contacts, Gmail

**CRM Module** (Sales Pipeline)
- Purpose: Sales pipeline and deal management
- Entities: companies, pipelines, pipeline_stages, deals, crm_activities, deal_stage_history
- Features: Pipeline Kanban board, company profiles, deal tracking, analytics
- Integrations: HubSpot Companies/Deals, future Salesforce/Pipedrive

### Data Model Relationship
```
CLIENTS                              CRM
┌─────────┐                    ┌─────────────┐
│ Contact │◄──────────────────►│   Company   │
│  (Jane) │  M:N via           │  (Acme Inc) │
└─────────┘  contact_company_  └─────────────┘
             relations              │
                                    │
                                    ▼
                              ┌─────────────┐
                              │    Deal     │
                              │ ($50K Q1)   │
                              └─────────────┘
```

## Consequences

### Positive
- Clean separation of concerns - each module has a focused purpose
- Each module stays simple and maintainable
- Matches user mental models from other tools (Contacts app vs Salesforce)
- Easier to build and test independently
- Users can have Clients without CRM (simpler use case)
- Users can have CRM without detailed contact management

### Negative
- Need to build linking UI between modules
- Slightly more complex navigation (two places to look)
- Need to handle data consistency across modules

### Neutral
- Companies can exist without associated contacts
- Contacts can exist without company association
- Integration sync logic routes to appropriate module based on data type

## Alternatives Considered

1. **Merge CRM into Clients module**
   - Rejected: Would make Clients overly complex, mixing individual contacts with organizational pipelines
   - Users who just want contact management would see unnecessary CRM complexity

2. **Rename Clients to CRM and expand it**
   - Rejected: Loses the simple contact management use case
   - "CRM" implies sales focus which isn't always needed

3. **Create unified "Relationships" module**
   - Rejected: Too abstract, doesn't match user expectations
   - Harder to understand and navigate

## Implementation Status

| Component | Status |
|-----------|--------|
| CRM Database Schema | 100% (migration 041) |
| CRM SQLC Queries | 100% |
| CRM Backend Handlers | ~30% |
| CRM Frontend UI | 0% |
| Clients Module | 100% |
| Module Linking UI | 0% |

## References
- Migration 041: `/desktop/backend-go/internal/database/migrations/041_crm.sql`
- CRM Queries: `/desktop/backend-go/internal/database/queries/crm.sql`
- Existing Deals Handler: `/desktop/backend-go/internal/handlers/deals.go`
- Integration Mapping: `/docs/architecture/INTEGRATION_MODULE_MAPPING.md`

---
*Decision made by: Roberto*
*Reviewed by: Claude Code Architect Agent*
