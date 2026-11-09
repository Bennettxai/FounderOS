import { z } from 'zod';

export const StartupMemoryRecordKindSchema = z.enum([
  'company',
  'governance',
  'memory_policy',
  'department',
  'agent',
]);

export const StartupMemoryProvenanceTypeSchema = z.enum([
  'canonical_yaml',
  'startup_db',
  'runtime_profile',
  'model_assignment_history',
]);

export const StartupMemoryProvenanceSchema = z.object({
  source: z.string().min(1),
  type: StartupMemoryProvenanceTypeSchema,
});

export const StartupMemoryRecordSchema = z.object({
  id: z.string().min(1),
  kind: StartupMemoryRecordKindSchema,
  title: z.string().min(1),
  summary: z.string().default(''),
  content: z.string().min(1),
  keywords: z.array(z.string().min(1)).default([]),
  departmentId: z.string().min(1).optional(),
  agentId: z.string().min(1).optional(),
  currentModel: z.string().optional(),
  version: z.number().int().positive().default(1),
  provenance: z.array(StartupMemoryProvenanceSchema).min(1),
  metadata: z.record(z.unknown()).default({}),
});

export const StartupMemorySnapshotSchema = z.object({
  schemaVersion: z.literal(1),
  generatedAt: z.string().datetime({ offset: true }),
  records: z.array(StartupMemoryRecordSchema),
});

export const StartupMemoryRequesterSchema = z.object({
  kind: z.enum(['agent', 'human']),
  id: z.string().min(1),
});

export const StartupMemorySearchRequestSchema = z.object({
  query: z.string().trim().min(2).max(4000),
  requester: StartupMemoryRequesterSchema,
  limit: z.number().int().min(1).max(30).default(8),
  kinds: z.array(StartupMemoryRecordKindSchema).default([]),
});

export type StartupMemoryRecordKind = z.infer<
  typeof StartupMemoryRecordKindSchema
>;

export type StartupMemoryRecord = z.infer<
  typeof StartupMemoryRecordSchema
>;

export type StartupMemorySnapshot = z.infer<
  typeof StartupMemorySnapshotSchema
>;

export type StartupMemorySearchRequest = z.infer<
  typeof StartupMemorySearchRequestSchema
>;
