import { describe, expect, test, beforeEach } from 'vitest';
import { openDb, type FounderDb } from '@/lib/db';

/**
 * The slab pages chart a window of runs (14 or 30 days). They read it with a
 * date bound in SQL instead of pulling thousands of rows and filtering in JS
 * (review 2026-09-24: recent(2000) on Home, recent(5000) on /tasks and
 * /workflows, an unbounded byAgent per crew member on /content).
 */
let db: FounderDb;
beforeEach(() => {
  db = openDb(':memory:');
});

const agentRun = (id: string, agentId: string, startedAt: string) => ({ id, agentId, startedAt, finishedAt: startedAt, ok: true, summary: 's' });
const cronRun = (id: string, startedAt: string) => ({ id, cronId: 'c1', agentId: 'a', startedAt, finishedAt: startedAt, ok: true, summary: 's' });

describe('runs since a date', () => {
  test('agentRuns.since returns only runs at or after the bound, newest first', () => {
    const before = db.agentRuns.since('2026-09-10T00:00:00.000Z').length;
    db.agentRuns.insert(agentRun('old', 'x', '2026-09-01T00:00:00.000Z'));
    db.agentRuns.insert(agentRun('new1', 'x', '2026-09-20T00:00:00.000Z'));
    db.agentRuns.insert(agentRun('new2', 'y', '2026-09-22T00:00:00.000Z'));
    const got = db.agentRuns.since('2026-09-10T00:00:00.000Z').slice(0, 2);
    expect(got.map((r) => r.id)).toEqual(['new2', 'new1']);
    expect(db.agentRuns.since('2026-09-10T00:00:00.000Z').length).toBe(before + 2);
  });
  test('agentRuns.since can narrow to some agents', () => {
    db.agentRuns.insert(agentRun('a1', 'x', '2026-09-20T00:00:00.000Z'));
    db.agentRuns.insert(agentRun('b1', 'y', '2026-09-21T00:00:00.000Z'));
    expect(db.agentRuns.since('2026-09-10T00:00:00.000Z', ['x']).map((r) => r.id)).toEqual(['a1']);
    expect(db.agentRuns.since('2026-09-10T00:00:00.000Z', [])).toEqual([]);
  });
  test('cronRuns.since bounds by date too', () => {
    db.cronRuns.insert(cronRun('c-old', '2026-09-01T00:00:00.000Z'));
    db.cronRuns.insert(cronRun('c-new', '2026-09-20T00:00:00.000Z'));
    expect(db.cronRuns.since('2026-09-10T00:00:00.000Z').map((r) => r.id)).toContain('c-new');
    expect(db.cronRuns.since('2026-09-10T00:00:00.000Z').map((r) => r.id)).not.toContain('c-old');
  });
});
