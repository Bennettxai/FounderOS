import { beforeEach, describe, expect, test, vi } from 'vitest';
const mocks = vi.hoisted(() => ({ scan: vi.fn(), codex: vi.fn(), ollama: vi.fn(), db: vi.fn() }));
vi.mock('@/lib/connectors/claude-usage', () => ({ scanClaudeProjects: mocks.scan, defaultProjectsDir: () => '/private/transcripts', seatId: () => 'private-machine' }));
vi.mock('@/lib/connectors/codex-usage', () => ({ codexSeat: mocks.codex }));
vi.mock('@/lib/connectors/ollama-usage', () => ({ ollamaLane: mocks.ollama }));
vi.mock('@/lib/data', () => ({ getDb: mocks.db }));
import { GET } from '@/app/api/usage/route';
describe('public demo usage privacy', () => {
  beforeEach(() => vi.clearAllMocks());
  test('never reads private transcripts, host services or stored usage in demo mode', async () => {
    vi.stubEnv('DEMO_GATE', '1');
    try {
      const body = await (await GET()).json();
      expect(body.claude).toBeNull();
      expect(body.codex).toBeNull();
      expect(body.ollama.machines).toEqual([]);
      for (const fn of Object.values(mocks)) expect(fn).not.toHaveBeenCalled();
    } finally { vi.unstubAllEnvs(); }
  });
  test('adapts configured operator data to the new plan-shaped UI outside demo mode', async () => {
    vi.stubEnv('DEMO_GATE', '0');
    mocks.scan.mockResolvedValue(null);
    mocks.codex.mockResolvedValue(null);
    mocks.db.mockReturnValue({ usageSnapshots: { all: () => [] } });
    mocks.ollama.mockResolvedValue({ state: 'down', models: [], note: 'Not configured' });
    try {
      const body = await (await GET()).json();
      expect(body.claude).toBeNull();
      expect(body.codex).toBeNull();
      expect(body.ollama.requests).toBeNull();
      expect(body.ollama.machines[0].label).toBe('Local machine');
    } finally { vi.unstubAllEnvs(); }
  });
});
