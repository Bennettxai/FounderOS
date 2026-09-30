import { readFileSync } from 'node:fs';
import { join } from 'node:path';
import { describe, expect, test } from 'vitest';
import { tradingVolume } from '@/lib/trading-view';
import type { TradingAccountSnapshot, TradingPosition } from '@/lib/schemas';

/**
 * /trading already sat in the Brand Deals mould (2026-09-18). On 2026-09-24
 * it moved onto the shared slab kit so its Accounts card reads exactly like
 * Deal Volume: a count-up headline, dot chips, and the kit's sweeping meters,
 * each an honest share of everything held. The numbers come from here.
 */

const snap = (over: Partial<TradingAccountSnapshot>): TradingAccountSnapshot => ({
  capturedAt: '2026-09-24T14:30:00.000Z',
  accountId: 'agentic',
  accountLabel: 'Agentic',
  accountValueUsd: 600,
  buyingPowerUsd: 500,
  cashUsd: 500,
  dayPnlUsd: 0,
  totalPnlUsd: 0,
  source: 'robinhood',
  ...over,
});
const pos = (symbol: string, marketValueUsd: number): TradingPosition => ({
  capturedAt: '2026-09-24T14:30:00.000Z',
  accountId: 'agentic',
  symbol,
  quantity: 1,
  avgCostUsd: marketValueUsd,
  marketValueUsd,
  unrealizedPnlUsd: 0,
});

describe('tradingVolume', () => {
  const accounts = [snap({ dayPnlUsd: 4.5 }), snap({ accountId: 'individual', accountLabel: 'Individual', accountValueUsd: 1200, dayPnlUsd: -1.25 })];
  const phantom = { address: '68SHabcdefghBST2', sol: 2, usdPerSol: 100, usdValue: 200, fetchedAt: '2026-09-24T14:30:00.000Z' };
  const v = tradingVolume({ accounts, positions: [pos('QQQ', 90), pos('SPY', 10)], phantom });

  test('the headline is the brokerage total, with the day and the book as chips', () => {
    expect(v.headline).toBe(1800);
    expect(v.chips).toEqual([
      { tone: 'ok', text: '+$3.25 today' },
      { tone: 'accent', text: '$100 in 2 positions' },
    ]);
    expect(v.caption).toBe('brokerage, 2 accounts · $2,000.00 with the wallet');
  });

  test('one meter per account plus the wallet, each its share of everything held, naming who may trade it', () => {
    expect(v.meters.map((m) => m.label)).toEqual(['Agentic · agent may trade', 'Individual · read-only to agents', 'Phantom · 2 SOL · 68SH…BST2']);
    expect(v.meters[0].frac).toBeCloseTo(0.3);
    expect(v.meters[1].frac).toBeCloseTo(0.6);
    expect(v.meters[2].frac).toBeCloseTo(0.1);
    expect(v.meters.map((m) => m.display)).toEqual(['$600.00', '$1,200.00', '$200.00']);
    expect(v.meters.map((m) => m.hue)).toEqual(['var(--accent)', 'var(--ramp-1)', 'var(--ramp-4)']);
    expect(v.foot).toBe('only the agentic sleeve can be traded by an agent · $100.00 / SOL');
  });

  test('a losing day is a red chip; an unreachable wallet is an empty meter that says so', () => {
    const r = tradingVolume({ accounts: [snap({ dayPnlUsd: -3 })], positions: [], phantom: null });
    expect(r.chips[0]).toEqual({ tone: 'err', text: '-$3.00 today' });
    expect(r.chips[1]).toEqual({ tone: 'accent', text: 'all cash' });
    expect(r.meters.at(-1)).toMatchObject({ label: 'Phantom · wallet not reachable', frac: 0, display: '--' });
    expect(r.foot).toBe('only the agentic sleeve can be traded by an agent · wallet read from a public RPC');
  });

  test('a wallet with no price reads unavailable, not $0', () => {
    const r = tradingVolume({ accounts: [snap({})], positions: [], phantom: { ...phantom, usdValue: null, usdPerSol: null } });
    expect(r.meters.at(-1)).toMatchObject({ frac: 0, display: 'price unavailable' });
  });

  test('nothing fed yet: empty meters, never fabricated fills', () => {
    const e = tradingVolume({ accounts: [], positions: [], phantom: null });
    expect(e.headline).toBe(0);
    expect(e.meters.every((m) => m.frac === 0)).toBe(true);
  });
});

describe('the trading board on the slab kit', () => {
  const board = readFileSync(join(process.cwd(), 'components/trading/TradingBoard.tsx'), 'utf8');

  test('uses the kit slab, title row, count-up headline, meters, insight card and dot matrix', () => {
    expect(board).toMatch(/from '@\/components\/slab'/);
    expect(board).toMatch(/from '@\/components\/slab-charts'/);
    for (const piece of ['<Slab>', '<SlabTitle', '<BigStat', '<MeterStack', '<InsightCard', '<DotMatrix']) expect(board, piece).toContain(piece);
    expect(board).toMatch(/tradingVolume\(/);
    expect((board.match(/<InsightCard/g) ?? []).length).toBe(1);
  });

  test('no hand-rolled copies of what the kit already does', () => {
    expect(board).not.toMatch(/^function Meter\b/m);
    expect(board).not.toMatch(/^function DotMatrix\b/m);
    expect(board).not.toContain('tb-meter-in');
    expect(board).not.toContain('tile-glow-a');
    expect(board).not.toMatch(/transition-(colors|all)\b/);
  });
});
