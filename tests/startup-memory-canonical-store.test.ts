import {
  afterEach,
  describe,
  expect,
  test,
} from 'vitest';

import {
  openDb,
  type FounderDb,
} from '@/lib/db';
import { seedDatabase } from '@/lib/seed';
import {
  buildCanonicalStartupMemory,
} from '@/lib/brain/memory/canonical-store';
import {
  searchCanonicalStartupMemory,
} from '@/lib/brain/memory/selective-retrieval';

let db: FounderDb | null = null;

function createMemory() {
  db = openDb(':memory:');
  seedDatabase(db);

  return buildCanonicalStartupMemory({
    db,
    rootDir: process.cwd(),
    generatedAt: '2026-01-01T00:00:00.000Z',
  });
}

afterEach(() => {
  db?.close();
  db = null;
});

describe('Canonical Startup Memory', () => {
  test('contains structural records with atomic governance and no operational experience', () => {
    const memory = createMemory();

    expect(memory.records.length).toBeGreaterThan(25);
    expect(memory.records.filter((record) => record.kind === 'agent')).toHaveLength(19);
    expect(memory.records.filter((record) => record.kind === 'department')).toHaveLength(3);
    expect(memory.records.some((record) => record.id === 'company:identity')).toBe(true);
    expect(memory.records.some((record) => record.id === 'governance:financial_execution_policy')).toBe(true);
    expect(memory.records.some((record) => record.id === 'governance:separation_of_duties')).toBe(true);
    expect(memory.records.some((record) => record.id === 'company:memory-policy')).toBe(true);
  });

  test('agent memory combines organization, runtime profile and current model', () => {
    const memory = createMemory();
    const lauti = memory.records.find((record) => record.id === 'agent:lauti');

    expect(lauti).toBeDefined();
    expect(lauti?.agentId).toBe('lauti');
    expect(lauti?.currentModel).toBe('unassigned');
    expect(lauti?.content).toContain('approve_own_proposals');
    expect(lauti?.content).toContain('financial-statements');
    expect(lauti?.provenance.map((item) => item.type)).toEqual(
      expect.arrayContaining([
        'canonical_yaml',
        'startup_db',
        'runtime_profile',
        'model_assignment_history',
      ]),
    );
  });

  test('selectively retrieves Lauti and governance for an execution-permission question', () => {
    const memory = createMemory();

    const results = searchCanonicalStartupMemory(memory, {
      query: 'Can Lauti execute trades?',
      requester: {
        kind: 'human',
        id: 'ceo',
      },
      limit: 5,
      kinds: [],
    });

    const ids = results.map((result) => result.record.id);

    expect(ids[0]).toBe('agent:lauti');
    expect(ids).toContain('governance:financial_execution_policy');
    expect(ids).toContain('governance:separation_of_duties');
    expect(ids.filter((id) => id.startsWith('agent:'))).toEqual(['agent:lauti']);
    expect(results.length).toBeLessThanOrEqual(5);
  });

  test('never turns a search into an unrestricted full dump', () => {
    const memory = createMemory();

    const results = searchCanonicalStartupMemory(memory, {
      query: 'startup company agent governance memory',
      requester: {
        kind: 'agent',
        id: 'lauti',
      },
      limit: 4,
      kinds: [],
    });

    expect(memory.records.length).toBeGreaterThan(25);
    expect(results.length).toBeLessThanOrEqual(4);
  });

  test('returns no records when nothing is relevant', () => {
    const memory = createMemory();

    const results = searchCanonicalStartupMemory(memory, {
      query: 'xylophone nebula archaeology',
      requester: {
        kind: 'human',
        id: 'ceo',
      },
      limit: 8,
      kinds: [],
    });

    expect(results).toEqual([]);
  });
test('prioritizes execution governance and suppresses departments for named-agent permission queries', () => {
    const memory = createMemory();

    const results = searchCanonicalStartupMemory(memory, {
      query: 'Can Lauti execute trades?',
      requester: {
        kind: 'human',
        id: 'ceo',
      },
      limit: 5,
      kinds: [],
    });

    const ids = results.map((result) => result.record.id);

    expect(ids[0]).toBe('agent:lauti');
    expect(ids[1]).toBe('governance:financial_execution_policy');
    expect(ids[2]).toBe('governance:separation_of_duties');
    expect(ids.some((id) => id.startsWith('department:'))).toBe(false);
    expect(ids.filter((id) => id.startsWith('agent:'))).toEqual(['agent:lauti']);
  });
});

