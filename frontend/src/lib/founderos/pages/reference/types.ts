// GET /api/founderos/pages/reference (Go pages/reference): FounderOS v1 DomainSchema.
export type Domain = { id: string; number: number; title: string; color: string; items: string[] };

export type ReferenceBody = { domains: Domain[] };
