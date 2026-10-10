<script lang="ts">
  /**
   * ActivityHeatmap — node × week (or day) grid of ingest intensity.
   *
   * Rows:    nodes sorted by total activity desc.
   * Columns: weeks (last N) or days.
   * Color:   cold=bg → warm=accent-soft → hot=accent.
   * Click:   emits onDrillDown(nodeSlug, periodStart).
   */

  import type { ActivityEvent, WorkspaceNode } from '$lib/api/workspace';

  interface Props {
    events: ActivityEvent[];
    nodes: WorkspaceNode[];
    granularity?: 'week' | 'day';
    periods?: number;           // number of columns (weeks or days)
    onDrillDown?: (nodeSlug: string, periodStart: Date) => void;
  }

  let {
    events,
    nodes,
    granularity = 'week',
    periods = 12,
    onDrillDown,
  }: Props = $props();

  // ── Period buckets ─────────────────────────────────────────────────────

  /** Returns the floor of the period containing `d` (Mon for weeks, same day for days). */
  function periodFloor(d: Date): Date {
    if (granularity === 'week') {
      const copy = new Date(d);
      const day = copy.getUTCDay(); // 0=Sun
      const diff = (day + 6) % 7;  // Monday offset
      copy.setUTCDate(copy.getUTCDate() - diff);
      copy.setUTCHours(0, 0, 0, 0);
      return copy;
    }
    const copy = new Date(d);
    copy.setUTCHours(0, 0, 0, 0);
    return copy;
  }

  const msPerPeriod = $derived(granularity === 'week' ? 7 * 86400e3 : 86400e3);

  /** Sorted period starts, oldest first. */
  const periodStarts = $derived((): Date[] => {
    const now = periodFloor(new Date());
    const starts: Date[] = [];
    for (let i = periods - 1; i >= 0; i--) {
      starts.push(new Date(now.getTime() - i * msPerPeriod));
    }
    return starts;
  });

  /** Map periodKey → index */
  function periodKey(d: Date): string {
    return periodFloor(d).toISOString().slice(0, granularity === 'week' ? 10 : 10);
  }

  // ── Aggregate counts per (node slug) × period ──────────────────────────

  const countMap = $derived((): Map<string, Map<string, number>> => {
    const m = new Map<string, Map<string, number>>();
    for (const e of events) {
      // Extract node slug from target_uri: "optimal://contexts/<id>" or metadata.node
      const nodeSlug =
        (e.metadata?.node as string | undefined) ??
        (e.target_uri ? e.target_uri.split('/').slice(-2, -1)[0] ?? '' : '');
      if (!nodeSlug) continue;
      const pk = periodKey(new Date(e.ts));
      if (!m.has(nodeSlug)) m.set(nodeSlug, new Map());
      const inner = m.get(nodeSlug)!;
      inner.set(pk, (inner.get(pk) ?? 0) + 1);
    }
    return m;
  });

  /** Rows: nodes sorted by total activity desc. Fall back to all known nodes. */
  const rows = $derived((): WorkspaceNode[] => {
    const totals = new Map<string, number>();
    for (const [slug, periods] of countMap()) {
      let t = 0;
      for (const v of periods.values()) t += v;
      totals.set(slug, t);
    }

    // Merge event-seen nodes with workspace node list
    const slugSet = new Set([...nodes.map((n) => n.slug), ...totals.keys()]);

    const merged: WorkspaceNode[] = [];
    for (const slug of slugSet) {
      const known = nodes.find((n) => n.slug === slug);
      if (known) {
        merged.push(known);
      } else {
        // Synthetic node from activity events — node not yet in workspace list
        merged.push({
          id: slug,
          slug,
          name: slug,
          kind: 'unknown',
          parent_id: null,
          style: 'internal',
          status: 'active',
          signal_count: 0,
        });
      }
    }

    return merged.sort((a, b) => (totals.get(b.slug) ?? 0) - (totals.get(a.slug) ?? 0));
  });

  // ── Color scale ────────────────────────────────────────────────────────

  const maxCount = $derived((): number => {
    let m = 0;
    for (const inner of countMap().values()) {
      for (const v of inner.values()) if (v > m) m = v;
    }
    return m || 1;
  });

  function cellColor(count: number, max: number): string {
    if (count === 0) return 'var(--bg-elevated)';
    const t = count / max;
    if (t < 0.15) return 'var(--bg-elevated-2)';
    if (t < 0.35) return 'color-mix(in srgb, var(--accent) 18%, var(--bg-elevated))';
    if (t < 0.60) return 'color-mix(in srgb, var(--accent) 38%, var(--bg-elevated))';
    if (t < 0.80) return 'color-mix(in srgb, var(--accent) 62%, var(--bg-elevated))';
    return 'var(--accent)';
  }

  function cellTextColor(count: number, max: number): string {
    if (count === 0) return 'var(--text-subtle)';
    const t = count / max;
    return t >= 0.60 ? 'var(--bg)' : 'var(--text-muted)';
  }

  // ── Tooltip ────────────────────────────────────────────────────────────
  let tooltip = $state<{ x: number; y: number; content: string } | null>(null);

  function showCellTip(e: MouseEvent, nodeSlug: string, start: Date, count: number) {
    const rect = (e.currentTarget as HTMLElement).getBoundingClientRect();
    const label = granularity === 'week'
      ? `week of ${start.toLocaleDateString(undefined, { month: 'short', day: 'numeric' })}`
      : start.toLocaleDateString(undefined, { month: 'short', day: 'numeric' });
    tooltip = {
      x: rect.left + rect.width / 2,
      y: rect.top - 8,
      content: `${nodeSlug}\n${label}\n${count} signal${count !== 1 ? 's' : ''}`,
    };
  }

  function hideTip() {
    tooltip = null;
  }

  // ── Column header format ───────────────────────────────────────────────
  function colLabel(start: Date): string {
    if (granularity === 'week') {
      return start.toLocaleDateString(undefined, { month: 'short', day: 'numeric' });
    }
    return start.toLocaleDateString(undefined, { month: 'short', day: 'numeric' });
  }
</script>

<div class="hm">
  {#if rows().length === 0}
    <p class="hm__empty">No activity data for this period.</p>
  {:else}
    <div class="hm__scroll">
      <table class="hm__table">
        <thead>
          <tr>
            <th class="hm__th hm__th--node">Node</th>
            {#each periodStarts() as start}
              <th class="hm__th hm__th--col">{colLabel(start)}</th>
            {/each}
            <th class="hm__th hm__th--total">Total</th>
          </tr>
        </thead>
        <tbody>
          {#each rows() as node}
            {@const nodeMap = countMap().get(node.slug)}
            {@const rowTotal = nodeMap ? [...nodeMap.values()].reduce((a, b) => a + b, 0) : 0}
            <tr class="hm__row">
              <td class="hm__td-node">
                <span class="hm__node-name" title={node.slug}>{node.name}</span>
              </td>
              {#each periodStarts() as start}
                {@const pk = periodKey(start)}
                {@const count = nodeMap?.get(pk) ?? 0}
                <td
                  class="hm__cell"
                  style="background: {cellColor(count, maxCount())}; color: {cellTextColor(count, maxCount())};"
                  onmouseenter={(e) => showCellTip(e, node.slug, start, count)}
                  onmouseleave={hideTip}
                  onclick={() => onDrillDown?.(node.slug, start)}
                  role={onDrillDown ? 'button' : 'cell'}
                  tabindex={onDrillDown ? 0 : -1}
                  onkeydown={(e) => (e.key === 'Enter' || e.key === ' ') && onDrillDown?.(node.slug, start)}
                  aria-label="{node.name}: {count} signals in {colLabel(start)}"
                >
                  {count > 0 ? count : ''}
                </td>
              {/each}
              <td class="hm__td-total">{rowTotal}</td>
            </tr>
          {/each}
        </tbody>
      </table>
    </div>

    <!-- Color scale legend -->
    <div class="hm__scale">
      <span class="hm__scale-label">less</span>
      <div class="hm__scale-bar">
        {#each [0, 0.15, 0.35, 0.60, 0.80, 1.0] as t}
          <div
            class="hm__scale-seg"
            style="background: {cellColor(Math.round(t * 10), 10)};"
          ></div>
        {/each}
      </div>
      <span class="hm__scale-label">more</span>
    </div>
  {/if}
</div>

<!-- Fixed tooltip rendered outside SVG for correct screen positioning -->
{#if tooltip}
  <div
    class="hm__tooltip"
    style="left: {tooltip.x}px; top: {tooltip.y}px;"
    aria-hidden="true"
  >
    {#each tooltip.content.split('\n') as line, li}
      <div class:hm__tooltip-title={li === 0}>{line}</div>
    {/each}
  </div>
{/if}

<style>
  .hm {
    width: 100%;
    overflow: hidden;
  }

  .hm__empty {
    color: var(--text-muted);
    font-size: 0.9rem;
    padding: 2rem;
    text-align: center;
  }

  .hm__scroll {
    overflow-x: auto;
    border: 1px solid var(--border);
    border-radius: var(--r-md);
  }

  .hm__table {
    border-collapse: collapse;
    min-width: 100%;
    font-size: 0.78rem;
  }

  .hm__th {
    padding: 0.45rem 0.5rem;
    font-weight: 600;
    font-size: 0.68rem;
    text-transform: uppercase;
    letter-spacing: 0.06em;
    color: var(--text-muted);
    background: var(--bg-elevated);
    border-bottom: 1px solid var(--border);
    white-space: nowrap;
    text-align: center;
  }
  .hm__th--node {
    text-align: left;
    min-width: 140px;
    position: sticky;
    left: 0;
    z-index: 2;
    background: var(--bg-elevated);
    border-right: 1px solid var(--border);
  }
  .hm__th--col {
    min-width: 52px;
  }
  .hm__th--total {
    min-width: 52px;
    color: var(--text-subtle);
  }

  .hm__row:hover .hm__td-node {
    background: var(--bg-elevated-2);
  }

  .hm__td-node {
    padding: 0.35rem 0.6rem;
    background: var(--bg-elevated);
    border-right: 1px solid var(--border);
    border-bottom: 1px solid var(--border-soft);
    position: sticky;
    left: 0;
    z-index: 1;
    max-width: 160px;
    overflow: hidden;
    transition: background 0.1s;
  }

  .hm__node-name {
    display: block;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    color: var(--text);
    font-weight: 500;
    font-size: 0.8rem;
  }

  .hm__cell {
    width: 52px;
    height: 32px;
    text-align: center;
    font-size: 0.72rem;
    font-weight: 600;
    border-bottom: 1px solid var(--border-soft);
    border-right: 1px solid rgba(255, 255, 255, 0.04);
    transition: filter 0.1s, outline 0.1s;
    cursor: default;
    user-select: none;
  }
  .hm__cell[role='button'] {
    cursor: pointer;
  }
  .hm__cell[role='button']:hover {
    filter: brightness(1.18);
    outline: 2px solid var(--accent);
    outline-offset: -2px;
    z-index: 1;
    position: relative;
  }

  .hm__td-total {
    padding: 0.35rem 0.6rem;
    text-align: center;
    font-size: 0.75rem;
    color: var(--text-muted);
    font-weight: 600;
    border-bottom: 1px solid var(--border-soft);
    border-left: 1px solid var(--border-soft);
    background: var(--bg-elevated);
    font-family: var(--font-mono);
  }

  /* Color scale legend */
  .hm__scale {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    margin-top: 0.6rem;
    justify-content: flex-end;
  }
  .hm__scale-label {
    font-size: 0.7rem;
    color: var(--text-subtle);
  }
  .hm__scale-bar {
    display: flex;
    border-radius: 4px;
    overflow: hidden;
    border: 1px solid var(--border);
    height: 12px;
    width: 120px;
  }
  .hm__scale-seg {
    flex: 1;
    height: 100%;
  }

  /* Tooltip — fixed screen position */
  .hm__tooltip {
    position: fixed;
    transform: translate(-50%, -100%);
    background: var(--bg-elevated);
    border: 1px solid var(--border);
    border-radius: var(--r-md);
    padding: 0.45rem 0.7rem;
    font-size: 0.75rem;
    color: var(--text-muted);
    pointer-events: none;
    z-index: 100;
    box-shadow: 0 8px 24px -8px rgba(0, 0, 0, 0.5);
    white-space: nowrap;
    line-height: 1.5;
  }
  .hm__tooltip-title {
    color: var(--text);
    font-weight: 600;
    font-size: 0.8rem;
  }
</style>
