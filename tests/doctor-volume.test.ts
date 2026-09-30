import { describe, expect, test } from 'vitest';
import { doctorVolume, doctorLayers, type DoctorVolumeInput } from '@/lib/doctor-volume';

/**
 * /doctor in the Brand Deals look (Alex, 2026-09-24: "rebuild the entire OS
 * in that light"). The Health Volume card, the brain-run step line, the store
 * dot matrix and the one "Needs you" card all come from this pure view-model,
 * fed with the page's own real reads: the gbrain doctor, the brain-store walk,
 * the pillar radar axes and the data-agent's run history. Offline means empty
 * meters and honest copy, never a number the doctor did not say.
 */
const NOW = new Date('2026-09-24T18:00:00');

const axis = (label: string, score: number) => ({ id: label.toLowerCase(), label, color: 'var(--ramp-1)', score, roster: 0, freshness: 0, sop: 0 });

const LIVE: DoctorVolumeInput = {
  doctor: {
    connected: true,
    status: 'warnings',
    healthScore: 82,
    detail: 'gbrain warnings · health 82/100',
    checks: [
      { name: 'database', status: 'ok', message: 'reachable' },
      { name: 'embeddings', status: 'ok', message: 'ollama bge-m3' },
      { name: 'stale pages', status: 'warn', message: '12 pages not re-embedded' },
      { name: 'orphans', status: 'error', message: '3 orphan chunks' },
    ],
  },
  store: {
    path: '/tmp/demo-brain-store',
    totalFiles: 60,
    folders: [
      { name: 'meetings', files: 30 },
      { name: 'people', files: 20 },
      { name: 'launchpad-cohorts-remotion', files: 10 },
    ],
  },
  axes: [axis('Sales', 80), axis('Marketing', 60), axis('Tech', 100)],
  runs: [
    { finishedAt: '2026-09-24T09:00:00', ok: true },
    { finishedAt: '2026-09-24T12:00:00', ok: false },
    { finishedAt: '2026-09-22T12:00:00', ok: true },
    { finishedAt: '2026-08-01T12:00:00', ok: true }, // outside the window
  ],
  now: NOW,
  days: 14,
};

const OFFLINE: DoctorVolumeInput = {
  ...LIVE,
  doctor: { connected: false, status: 'unreachable', healthScore: null, detail: 'gbrain not reachable', checks: [] },
  runs: [],
};

describe('doctorLayers: the four storage layers, from the real reads', () => {
  test('live doctor, reachable database: every layer connected', () => {
    const { layers, fallbackActive } = doctorLayers({ ...LIVE, storeShort: '~/brain-store' });
    expect(fallbackActive).toBe(false);
    expect(layers.map((l) => l.name)).toEqual(['gbrain CLI', 'brain-store/', 'Ollama (bge-m3)', 'Supabase Second Brain']);
    expect(layers[1].val).toBe('60 pages');
    // the embeddings check is ok, so Ollama reads LIVE
    expect(layers[2]).toMatchObject({ val: 'LIVE', state: 'connected' });
    expect(layers.every((l) => l.state === 'connected')).toBe(true);
  });

  test('a failing database check turns on the local fallback and pauses Supabase', () => {
    const doctor = { ...LIVE.doctor, checks: [{ name: 'supabase database', status: 'error', message: 'paused' }] };
    const { layers, fallbackActive } = doctorLayers({ ...LIVE, doctor, storeShort: '~' });
    expect(fallbackActive).toBe(true);
    expect(layers[3]).toMatchObject({ val: 'PAUSED', state: 'available' });
  });

  test('unreachable doctor: the CLI is an error and the fallback is active', () => {
    const { layers, fallbackActive } = doctorLayers({ ...OFFLINE, storeShort: '~' });
    expect(fallbackActive).toBe(true);
    expect(layers[0]).toMatchObject({ val: 'UNREACHABLE', state: 'error' });
  });
});

describe('doctorVolume: the Health Volume card', () => {
  const v = doctorVolume(LIVE);

  test('the headline is the doctor health score, chips split the checks and add up', () => {
    expect(v.headline).toBe(82);
    expect(v.counts).toEqual({ ok: 2, warn: 1, fail: 1, total: 4 });
    expect(v.chips).toEqual([
      { tone: 'ok', text: '2 passing' },
      { tone: 'warn', text: '1 warn' },
      { tone: 'err', text: '1 failing' },
    ]);
    expect(v.caption).toContain('4 checks');
    expect(v.caption).toContain('60 pages');
  });

  test('meters are honest fractions of a real whole', () => {
    const byLabel = Object.fromEntries(v.meters.map((m) => [m.label.split(' (')[0], m]));
    expect(byLabel['Engine health'].frac).toBeCloseTo(0.82);
    expect(byLabel['Checks passing'].frac).toBeCloseTo(0.5);
    expect(byLabel['Checks passing'].label).toContain('2/4');
    expect(byLabel['Storage layers live'].frac).toBeCloseTo(1);
    // pillar health is the mean of the radar's own scores, over 100
    expect(byLabel['Pillar health'].frac).toBeCloseTo(0.8);
    expect(byLabel['Pillar health'].display).toBe('80/100');
    for (const m of v.meters) {
      expect(m.frac).toBeGreaterThanOrEqual(0);
      expect(m.frac).toBeLessThanOrEqual(1);
      expect(m.hue).toMatch(/^var\(--/);
    }
  });

  test('the foot names the store and the pillars behind the numbers', () => {
    expect(v.foot).toContain('3 folders');
    expect(v.foot).toContain('3 pillars');
  });
});

describe('doctorVolume: offline is honest', () => {
  const v = doctorVolume(OFFLINE);

  test('no score is invented and the doctor meters sit empty', () => {
    expect(v.headline).toBeNull();
    expect(v.chips).toEqual([{ tone: 'err', text: 'unreachable' }]);
    const health = v.meters.find((m) => m.label.startsWith('Engine health'))!;
    const checks = v.meters.find((m) => m.label.startsWith('Checks passing'))!;
    expect(health.frac).toBe(0);
    expect(health.display).toBe('unreachable');
    expect(checks.frac).toBe(0);
    expect(checks.display).toBe('no checks run');
    expect(v.caption).toMatch(/unreachable/);
  });

  test('the insight does not claim zero warnings when the doctor never answered', () => {
    expect(v.insight.value).toBeNull();
    expect(v.insight.headline).toMatch(/unreachable/i);
    expect(v.insight.frac).toBe(0);
  });

  test('no pillars, no store: empty meters rather than fabricated ones', () => {
    const empty = doctorVolume({ ...OFFLINE, axes: [], store: { path: '/x', totalFiles: 0, folders: [] } });
    const pillar = empty.meters.find((m) => m.label.startsWith('Pillar health'))!;
    expect(pillar.frac).toBe(0);
    expect(pillar.display).toBe('no pillars');
    expect(empty.store.cols).toEqual([]);
  });
});

describe('doctorVolume: the second row', () => {
  const v = doctorVolume(LIVE);

  test('the step line buckets real data-agent runs by local day, oldest first', () => {
    expect(v.series).toHaveLength(14);
    expect(v.series[13].count).toBe(2);
    expect(v.series[11].count).toBe(1);
    expect(v.series.reduce((n, s) => n + s.count, 0)).toBe(3);
    expect(v.runsInWindow).toBe(3);
    expect(v.failedInWindow).toBe(1);
  });

  test('the dot matrix is the biggest brain-store folders, short labels, real counts', () => {
    expect(v.store.cols.map((c) => c.count)).toEqual([30, 20, 10]);
    expect(v.store.cols[0].label).toBe('meetings');
    // a long folder name is clipped so the column stays narrow
    expect(v.store.cols[2].label.length).toBeLessThanOrEqual(10);
    expect(v.store.total).toBe(60);
    expect(v.store.top).toEqual({ name: 'meetings', files: 30 });
  });

  test('only the biggest seven folders make the matrix', () => {
    const folders = Array.from({ length: 10 }, (_, i) => ({ name: `f${i}`, files: i + 1 }));
    const many = doctorVolume({ ...LIVE, store: { path: '/x', totalFiles: 55, folders } });
    expect(many.store.cols.map((c) => c.count)).toEqual([10, 9, 8, 7, 6, 5, 4]);
  });

  test('the one insight card counts the checks that need attention', () => {
    expect(v.insight.value).toBe(2);
    expect(v.insight.headline).toContain('1 warn');
    expect(v.insight.headline).toContain('1 failing');
    expect(v.insight.body).toContain('stale pages');
    expect(v.insight.body).toContain('orphans');
    expect(v.insight.frac).toBeCloseTo(0.5);
  });

  test('all green says so', () => {
    const green = doctorVolume({ ...LIVE, doctor: { ...LIVE.doctor, checks: LIVE.doctor.checks.slice(0, 2) } });
    expect(green.insight.value).toBe(0);
    expect(green.insight.headline).toMatch(/every check passed/i);
  });
});

describe('an unchecked layer is not a live layer (review 2026-09-24)', () => {
  // `doctor --fast` runs no embeddings check, and an unreachable doctor runs
  // none at all; Ollama used to default to LIVE either way.
  const noEmbed = { ...LIVE.doctor, checks: LIVE.doctor.checks.filter((c) => c.name !== 'embeddings') };
  test('with no embeddings check Ollama reads NOT CHECKED and is not counted live', () => {
    const { layers } = doctorLayers({ ...LIVE, doctor: noEmbed, storeShort: '~' });
    expect(layers[2]).toMatchObject({ val: 'NOT CHECKED', state: 'available' });
  });
  test('the storage meter counts live layers over the layers actually checked', () => {
    const v = doctorVolume({ ...LIVE, doctor: noEmbed });
    const m = v.meters.find((x) => x.label.startsWith('Storage layers live'))!;
    expect(m.label).toBe('Storage layers live (3/3 checked · 1 unchecked)');
    expect(m.frac).toBeCloseTo(1);
  });
  test('an unreachable doctor never reads a layer as live on a default', () => {
    const v = doctorVolume(OFFLINE);
    const m = v.meters.find((x) => x.label.startsWith('Storage layers live'))!;
    // brain-store on disk is still a real read; the CLI is down; Ollama unchecked
    expect(m.label).toMatch(/^Storage layers live \(1\//);
  });
});
