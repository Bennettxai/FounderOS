import type { Meter, Tone } from '@/components/slab';
import type { SeriesPoint } from '@/components/slab-charts';
import type { BrainOverview } from '@/lib/connectors/gbrain';
import type { PillarAxis } from '@/lib/pillar-radar';

/** A storage layer no check covered on this read. */
const NOT_CHECKED = 'NOT CHECKED';

/**
 * /doctor in the Brand Deals look (Alex, 2026-09-24). One pure pass over
 * the page's own real reads (the gbrain doctor, the brain-store walk, the
 * pillar radar axes and the data-agent's run history) produces every number
 * the slab shows: the Health Volume headline, chips and meters, the brain-run
 * step line, the store dot matrix and the single "Needs you" insight. When the
 * doctor does not answer, the meters sit empty and the copy says so.
 */
export type DoctorVolumeInput = {
  doctor: BrainOverview['doctor'];
  store: BrainOverview['store'];
  axes: PillarAxis[];
  runs: { finishedAt: string; ok: boolean }[];
  now?: Date;
  days?: number;
};

export type StorageLayer = { name: string; sub: string; val: string; state: 'connected' | 'available' | 'error' };

export type DoctorVolume = {
  headline: number | null;
  counts: { ok: number; warn: number; fail: number; total: number };
  chips: Array<{ tone?: Tone; text: string }>;
  caption: string;
  meters: Meter[];
  foot: string;
  series: SeriesPoint[];
  runsInWindow: number;
  failedInWindow: number;
  store: { cols: SeriesPoint[]; total: number; top: { name: string; files: number } | null };
  insight: { value: number | null; headline: string; body: string; frac: number };
};

const frac = (n: number, d: number) => (d > 0 ? Math.max(0, Math.min(1, n / d)) : 0);
const pct = (f: number) => `${Math.round(f * 100)}%`;
const localDay = (d: Date) => `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`;
const dayLabel = (d: Date) => d.toLocaleDateString('en-US', { month: 'short', day: 'numeric' });
const clip = (s: string) => (s.length > 10 ? `${s.slice(0, 9)}…` : s);
const STORE_COLS = 7;

/**
 * The four storage layers the page lists, with the fallback rule it always
 * used: a failing Supabase/database check (or, with no such check, an
 * unreachable doctor) means searches are answered by local grep.
 */
export function doctorLayers(x: Pick<DoctorVolumeInput, 'doctor' | 'store'> & { storeShort: string }): {
  layers: StorageLayer[];
  fallbackActive: boolean;
} {
  const { doctor, store } = x;
  const supabaseCheck = doctor.checks.find((c) => /supabase|database/i.test(c.name));
  const embedCheck = doctor.checks.find((c) => /embed|ollama|zero/i.test(c.name));
  const fallbackActive = supabaseCheck ? supabaseCheck.status !== 'ok' : !doctor.connected;
  const layers: StorageLayer[] = [
    {
      name: 'gbrain CLI',
      sub: 'v0.47 · ~/.bun/bin/gbrain · doctor --fast',
      val: doctor.connected ? 'LIVE' : 'UNREACHABLE',
      state: doctor.connected ? 'connected' : 'error',
    },
    {
      name: 'brain-store/',
      sub: `${x.storeShort} · markdown knowledge`,
      val: `${store.totalFiles} pages`,
      state: store.totalFiles > 0 ? 'connected' : 'available',
    },
    {
      name: 'Ollama (bge-m3)',
      sub: 'hybrid-search embeddings · local, 1024d',
      // `doctor --fast` runs no embeddings check (and an unreachable doctor
      // runs none), so without one the layer is unverified, not LIVE.
      val: embedCheck ? (embedCheck.status === 'ok' ? 'LIVE' : embedCheck.status.toUpperCase()) : NOT_CHECKED,
      state: embedCheck && embedCheck.status === 'ok' ? 'connected' : 'available',
    },
    {
      name: 'Supabase Second Brain',
      sub: '918 pages / 11k chunks · free tier idle-pause',
      val: fallbackActive ? 'PAUSED' : 'LIVE',
      state: fallbackActive ? 'available' : 'connected',
    },
  ];
  return { layers, fallbackActive };
}

export function doctorVolume(x: DoctorVolumeInput): DoctorVolume {
  const now = x.now ?? new Date();
  const days = x.days ?? 14;
  const { doctor, store } = x;
  const live = doctor.connected;

  // Checks: ok, warn, and everything else (error, fail, unknown) as failing,
  // so the chips always add up to the checks run.
  const ok = doctor.checks.filter((c) => c.status === 'ok').length;
  const warn = doctor.checks.filter((c) => c.status === 'warn').length;
  const total = doctor.checks.length;
  const fail = total - ok - warn;
  const counts = { ok, warn, fail, total };

  const chips: DoctorVolume['chips'] = [];
  if (!live) chips.push({ tone: 'err', text: 'unreachable' });
  else {
    if (ok) chips.push({ tone: 'ok', text: `${ok} passing` });
    if (warn) chips.push({ tone: 'warn', text: `${warn} warn` });
    if (fail) chips.push({ tone: 'err', text: `${fail} failing` });
  }

  const score = live ? doctor.healthScore : null;
  const { layers } = doctorLayers({ doctor, store, storeShort: store.path });
  const layersLive = layers.filter((l) => l.state === 'connected').length;
  const layersChecked = layers.filter((l) => l.val !== NOT_CHECKED).length;
  const unchecked = layers.length - layersChecked;
  const pillarMean = x.axes.length ? Math.round(x.axes.reduce((n, a) => n + a.score, 0) / x.axes.length) : 0;

  const healthF = score == null ? 0 : frac(score, 100);
  const checksF = frac(ok, total);
  const layersF = frac(layersLive, layersChecked);
  const pillarF = frac(pillarMean, 100);
  const meters: Meter[] = [
    {
      label: `Engine health (${score ?? ' - '}/100)`,
      frac: healthF,
      display: score == null ? (live ? 'no score' : 'unreachable') : pct(healthF),
      hue: 'var(--accent)',
    },
    {
      label: `Checks passing (${ok}/${total})`,
      frac: checksF,
      display: total ? `${ok} ok · ${total - ok} flagged` : 'no checks run',
      hue: total > 0 && ok === total ? 'var(--ok)' : 'var(--warn)',
    },
    {
      label: unchecked
        ? `Storage layers live (${layersLive}/${layersChecked} checked · ${unchecked} unchecked)`
        : `Storage layers live (${layersLive}/${layers.length})`,
      frac: layersF,
      display: pct(layersF),
      hue: 'var(--ramp-1)',
    },
    {
      label: `Pillar health (${x.axes.length} pillars)`,
      frac: pillarF,
      display: x.axes.length ? `${pillarMean}/100` : 'no pillars',
      hue: 'var(--ramp-4)',
    },
  ];

  // data-agent runs bucketed by local day over the window, oldest first.
  const start = new Date(now);
  start.setHours(0, 0, 0, 0);
  start.setDate(start.getDate() - (days - 1));
  const buckets = new Map<string, number>();
  const labels: Array<[string, string]> = [];
  for (let i = 0; i < days; i++) {
    const d = new Date(start);
    d.setDate(start.getDate() + i);
    buckets.set(localDay(d), 0);
    labels.push([localDay(d), dayLabel(d)]);
  }
  let runsInWindow = 0;
  let failedInWindow = 0;
  for (const r of x.runs) {
    const t = new Date(r.finishedAt);
    if (!Number.isFinite(t.getTime()) || t < start || t > now) continue;
    const key = localDay(t);
    if (!buckets.has(key)) continue;
    buckets.set(key, (buckets.get(key) ?? 0) + 1);
    runsInWindow += 1;
    if (!r.ok) failedInWindow += 1;
  }
  const series = labels.map(([key, label]) => ({ label, count: buckets.get(key) ?? 0 }));

  // The store: its biggest folders, largest first.
  const biggest = [...store.folders].sort((a, b) => b.files - a.files || a.name.localeCompare(b.name));
  const storeCols = biggest.slice(0, STORE_COLS).map((f) => ({ label: clip(f.name), count: f.files }));

  // The one gradient card: the checks asking for attention.
  const flagged = doctor.checks.filter((c) => c.status !== 'ok');
  const insight: DoctorVolume['insight'] = !live
    ? {
        value: null,
        headline: 'The doctor is unreachable.',
        body: 'gbrain doctor did not answer; searches fall back to grepping the brain-store on disk.',
        frac: 0,
      }
    : {
        value: flagged.length,
        headline: flagged.length ? `${warn} warn · ${fail} failing.` : 'Every check passed.',
        body: flagged.length
          ? flagged
              .slice(0, 3)
              .map((c) => c.name)
              .join(' · ')
          : total
            ? `${total} checks green, nothing waiting on you.`
            : 'The doctor answered but ran no checks.',
        frac: frac(flagged.length, total),
      };

  return {
    headline: score,
    counts,
    chips,
    caption: live ? `${total} checks · ${store.totalFiles} pages on disk` : `gbrain unreachable · ${store.totalFiles} pages on disk, local grep answers`,
    meters,
    foot: `${store.folders.length} folders · ${x.axes.length} pillars · ${layersLive}/${layers.length} layers live`,
    series,
    runsInWindow,
    failedInWindow,
    store: { cols: storeCols, total: store.totalFiles, top: biggest[0] ? { name: biggest[0].name, files: biggest[0].files } : null },
    insight,
  };
}
