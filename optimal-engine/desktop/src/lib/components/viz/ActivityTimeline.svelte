<script lang="ts">
  /**
   * ActivityTimeline — SVG horizontal timeline of signal activity.
   *
   * Time axis: oldest on left, now on right.
   * Y bands:   one row per intent type (10 values of the intent enum).
   * Markers:
   *   - Signal events → small circle, radius proportional to sn_ratio
   *   - Wiki updates  → larger square in purple/cyan
   * Hover tooltip shows slug, intent, sn_ratio, citations.
   */

  import type { ActivityEvent } from '$lib/api/workspace';
  import type { WikiPageSummary } from '$lib/api';

  interface Props {
    events: ActivityEvent[];
    wikiPages: WikiPageSummary[];
    windowDays?: number;
    groupBy?: 'intent' | 'node' | 'audience';
  }

  let { events, wikiPages, windowDays = 30, groupBy = 'intent' }: Props = $props();

  // ── Intent enum (10 values) ────────────────────────────────────────────
  const INTENTS = [
    'decision',
    'action',
    'fact',
    'concern',
    'reference',
    'procedure',
    'observation',
    'commitment',
    'question',
    'other',
  ] as const;

  type IntentKey = (typeof INTENTS)[number];

  const INTENT_COLOR: Record<IntentKey, string> = {
    decision:    'var(--accent)',
    action:      'var(--amber)',
    fact:        'var(--cyan)',
    concern:     'var(--warn)',
    reference:   'var(--purple)',
    procedure:   'var(--good)',
    observation: '#a78bfa',   // violet
    commitment:  'var(--pink)',
    question:    '#34d399',   // emerald
    other:       'var(--text-muted)',
  };

  // ── Dimensions ─────────────────────────────────────────────────────────
  const MARGIN = { top: 16, right: 20, bottom: 36, left: 100 };
  const ROW_H = 40;
  const W = 880;
  const H = MARGIN.top + INTENTS.length * ROW_H + MARGIN.bottom;

  // ── Time domain ────────────────────────────────────────────────────────
  const now = $derived(Date.now());
  const windowMs = $derived(windowDays * 24 * 60 * 60 * 1000);
  const tMin = $derived(now - windowMs);
  const innerW = $derived(W - MARGIN.left - MARGIN.right);

  function xOf(ts: string | null): number | null {
    if (!ts) return null;
    const t = new Date(ts).getTime();
    if (t < tMin || t > now) return null;
    return MARGIN.left + ((t - tMin) / (now - tMin)) * innerW;
  }

  // ── Derive intent from metadata (graceful) ─────────────────────────────
  function eventIntent(e: ActivityEvent): IntentKey {
    const raw = (e.metadata?.intent as string | undefined) ?? '';
    return (INTENTS as readonly string[]).includes(raw) ? (raw as IntentKey) : 'other';
  }

  // ── Map events to viz markers ──────────────────────────────────────────
  interface Marker {
    x: number;
    y: number;
    intent: IntentKey;
    r: number;
    slug: string;
    snRatio: number;
    ts: string;
    citations: string;
    kind: 'signal';
  }

  interface WikiMarker {
    x: number;
    y: number;
    slug: string;
    ts: string;
    kind: 'wiki';
  }

  type AnyMarker = Marker | WikiMarker;

  const markers = $derived((): Marker[] => {
    return events.flatMap((e) => {
      const x = xOf(e.ts);
      if (x === null) return [];
      const intent = eventIntent(e);
      const idx = INTENTS.indexOf(intent);
      const y = MARGIN.top + idx * ROW_H + ROW_H / 2;
      const snRaw = e.metadata?.sn_ratio ?? e.metadata?.score ?? 0.5;
      const sn = typeof snRaw === 'number' ? Math.max(0.1, Math.min(1, snRaw)) : 0.5;
      const r = 3 + sn * 5;
      return [{
        x,
        y,
        intent,
        r,
        slug: (e.metadata?.slug as string | undefined) ?? e.target_uri ?? `event-${e.id}`,
        snRatio: sn,
        ts: e.ts,
        citations: String(e.metadata?.citations ?? 0),
        kind: 'signal' as const,
      }];
    });
  });

  const wikiMarkers = $derived((): WikiMarker[] => {
    return wikiPages.flatMap((p) => {
      const x = xOf(p.last_curated);
      if (x === null) return [];
      // Wiki markers float above the intent rows (y = row 0 top)
      const y = MARGIN.top + ROW_H / 2;
      return [{ x, y, slug: p.slug, ts: p.last_curated ?? '', kind: 'wiki' as const }];
    });
  });

  // ── Tooltip ────────────────────────────────────────────────────────────
  let tooltip = $state<{ x: number; y: number; content: string } | null>(null);

  function showTip(svgX: number, svgY: number, content: string) {
    tooltip = { x: svgX, y: svgY, content };
  }

  function hideTip() {
    tooltip = null;
  }

  // ── Tick marks along x axis ────────────────────────────────────────────
  const xTicks = $derived((): { x: number; label: string }[] => {
    const ticks: { x: number; label: string }[] = [];
    const step = windowDays <= 7 ? 1 : windowDays <= 14 ? 2 : 7;
    for (let d = 0; d <= windowDays; d += step) {
      const t = tMin + d * 24 * 60 * 60 * 1000;
      const x = MARGIN.left + (d / windowDays) * innerW;
      const date = new Date(t);
      const label = date.toLocaleDateString(undefined, { month: 'short', day: 'numeric' });
      ticks.push({ x, label });
    }
    return ticks;
  });
</script>

<div class="tl">
  <svg
    viewBox="0 0 {W} {H}"
    width="100%"
    style="max-width: {W}px; display: block;"
    role="img"
    aria-label="Signal activity timeline"
  >
    <!-- Grid lines + row bands -->
    {#each INTENTS as intent, i}
      {@const y = MARGIN.top + i * ROW_H}
      {@const isEven = i % 2 === 0}
      <rect
        x={MARGIN.left}
        {y}
        width={innerW}
        height={ROW_H}
        fill={isEven ? 'rgba(255,255,255,0.02)' : 'transparent'}
      />
      <!-- Row label -->
      <text
        x={MARGIN.left - 8}
        y={y + ROW_H / 2}
        dominant-baseline="middle"
        text-anchor="end"
        font-size="10"
        fill={INTENT_COLOR[intent]}
        font-family="var(--font-sans)"
        font-weight="600"
      >{intent}</text>
    {/each}

    <!-- Border box -->
    <rect
      x={MARGIN.left}
      y={MARGIN.top}
      width={innerW}
      height={INTENTS.length * ROW_H}
      fill="none"
      stroke="var(--border)"
      stroke-width="1"
      rx="4"
    />

    <!-- Vertical grid lines at ticks -->
    {#each xTicks() as tick}
      <line
        x1={tick.x}
        x2={tick.x}
        y1={MARGIN.top}
        y2={MARGIN.top + INTENTS.length * ROW_H}
        stroke="var(--border-soft)"
        stroke-width="0.5"
      />
    {/each}

    <!-- Wiki markers (squares) — rendered first so signals sit on top -->
    {#each wikiMarkers() as wm}
      <!-- svelte-ignore a11y_interactive_supports_focus -->
      <rect
        x={wm.x - 5}
        y={wm.y - 5}
        width="10"
        height="10"
        fill="var(--purple)"
        opacity="0.85"
        rx="2"
        stroke="var(--glass-border)"
        stroke-width="1"
        role="img"
        aria-label="Wiki: {wm.slug}"
        onmouseenter={() => showTip(wm.x, wm.y - 16, `wiki · ${wm.slug}\ncurated ${wm.ts.slice(0, 10)}`)}
        onmouseleave={hideTip}
        style="cursor: pointer;"
      />
    {/each}

    <!-- Signal markers (circles) -->
    {#each markers() as m}
      <!-- svelte-ignore a11y_interactive_supports_focus -->
      <circle
        cx={m.x}
        cy={m.y}
        r={m.r}
        fill={INTENT_COLOR[m.intent]}
        opacity="0.72"
        stroke="var(--glass-border)"
        stroke-width="0.5"
        role="img"
        aria-label="{m.intent}: {m.slug}"
        onmouseenter={() => showTip(m.x, m.y - m.r - 8, `${m.intent}\n${m.slug}\nsn_ratio: ${m.snRatio.toFixed(2)}\ncitations: ${m.citations}`)}
        onmouseleave={hideTip}
        style="cursor: pointer;"
      />
    {/each}

    <!-- X axis labels -->
    {#each xTicks() as tick}
      <text
        x={tick.x}
        y={MARGIN.top + INTENTS.length * ROW_H + 18}
        text-anchor="middle"
        font-size="9"
        fill="var(--text-subtle)"
        font-family="var(--font-sans)"
      >{tick.label}</text>
    {/each}

    <!-- "now" indicator -->
    <line
      x1={MARGIN.left + innerW}
      x2={MARGIN.left + innerW}
      y1={MARGIN.top}
      y2={MARGIN.top + INTENTS.length * ROW_H}
      stroke="var(--accent)"
      stroke-width="1.5"
      stroke-dasharray="4 2"
      opacity="0.5"
    />
    <text
      x={MARGIN.left + innerW}
      y={MARGIN.top + INTENTS.length * ROW_H + 30}
      text-anchor="end"
      font-size="9"
      fill="var(--accent)"
      font-family="var(--font-sans)"
    >now</text>

    <!-- Tooltip -->
    {#if tooltip}
      {@const lines = tooltip.content.split('\n')}
      {@const tipW = 200}
      {@const tipH = lines.length * 15 + 14}
      {@const tipX = Math.min(tooltip.x, W - tipW - 4)}
      {@const tipY = Math.max(MARGIN.top, tooltip.y - tipH)}
      <rect
        x={tipX}
        y={tipY}
        width={tipW}
        height={tipH}
        rx="6"
        fill="var(--bg-elevated)"
        stroke="var(--border)"
        stroke-width="1"
        filter="url(#shadow)"
      />
      {#each lines as line, li}
        <text
          x={tipX + 10}
          y={tipY + 14 + li * 15}
          font-size="10"
          fill={li === 0 ? 'var(--text)' : 'var(--text-muted)'}
          font-weight={li === 0 ? '600' : '400'}
          font-family="var(--font-sans)"
        >{line}</text>
      {/each}
    {/if}

    <defs>
      <filter id="shadow" x="-10%" y="-20%" width="120%" height="140%">
        <feDropShadow dx="0" dy="4" stdDeviation="6" flood-opacity="0.4" />
      </filter>
    </defs>
  </svg>

  <!-- Legend -->
  <div class="tl__legend">
    {#each INTENTS as intent}
      <span class="tl__legend-item">
        <span class="tl__legend-dot" style="background: {INTENT_COLOR[intent]};"></span>
        {intent}
      </span>
    {/each}
    <span class="tl__legend-item">
      <span class="tl__legend-wiki"></span>
      wiki update
    </span>
  </div>
</div>

<style>
  .tl {
    width: 100%;
  }

  .tl__legend {
    display: flex;
    flex-wrap: wrap;
    gap: 0.5rem 1rem;
    margin-top: 0.75rem;
    padding: 0.6rem 0.75rem;
    background: var(--bg-elevated);
    border: 1px solid var(--border);
    border-radius: var(--r-md);
  }

  .tl__legend-item {
    display: inline-flex;
    align-items: center;
    gap: 0.35rem;
    font-size: 0.72rem;
    color: var(--text-muted);
    font-weight: 500;
  }

  .tl__legend-dot {
    width: 8px;
    height: 8px;
    border-radius: 50%;
    flex-shrink: 0;
  }

  .tl__legend-wiki {
    width: 8px;
    height: 8px;
    border-radius: 2px;
    background: var(--purple);
    flex-shrink: 0;
  }
</style>
