<script lang="ts">
  import { onMount } from 'svelte';
  import ActivityHeatmap from '$lib/components/viz/ActivityHeatmap.svelte';
  import { getActivity, listWorkspace, type ActivityEvent, type WorkspaceNode } from '$lib/api/workspace';
  import { activeWorkspaceId, activeWorkspace } from '$lib/stores/workspace';

  // ── Config state ──────────────────────────────────────────────────────
  let granularity = $state<'week' | 'day'>('week');
  let periods = $state(12);

  // ── Data state ────────────────────────────────────────────────────────
  let events = $state<ActivityEvent[]>([]);
  let nodes = $state<WorkspaceNode[]>([]);
  let loading = $state(false);
  let error = $state<string | null>(null);

  // ── Drill-down state ──────────────────────────────────────────────────
  let drillNode = $state<string | null>(null);
  let drillPeriod = $state<Date | null>(null);
  let drillEvents = $state<ActivityEvent[]>([]);

  // ── Config (from /api/workspaces/:id/config, optional) ────────────────
  async function loadConfig(wsId: string | null) {
    if (!wsId) return;
    try {
      const res = await fetch(`/api/workspaces/${encodeURIComponent(wsId)}/config`);
      if (!res.ok) return;
      const cfg = (await res.json()) as {
        visualizations?: {
          enabled?: string[];
          heatmap?: {
            granularity?: string;
          };
        };
      };
      const hmCfg = cfg.visualizations?.heatmap;
      if (hmCfg?.granularity === 'day') granularity = 'day';
      else if (hmCfg?.granularity === 'week') granularity = 'week';
    } catch {
      // Config unavailable — silent
    }
  }

  // ── Fetch ─────────────────────────────────────────────────────────────
  async function load(wsId: string | null) {
    loading = true;
    error = null;
    drillNode = null;
    drillPeriod = null;
    drillEvents = [];
    try {
      const [evts, wsList] = await Promise.allSettled([
        getActivity({ limit: 500, workspace: wsId }),
        listWorkspace(wsId),
      ]);
      events = evts.status === 'fulfilled' ? evts.value : [];
      nodes = wsList.status === 'fulfilled' ? wsList.value : [];
    } catch (e) {
      error = (e as Error).message;
    } finally {
      loading = false;
    }
  }

  onMount(async () => {
    await loadConfig($activeWorkspaceId);
    await load($activeWorkspaceId);
  });

  // Re-fetch when workspace switches
  $effect(() => {
    const wsId = $activeWorkspaceId;
    void loadConfig(wsId).then(() => load(wsId));
  });

  // ── Drill-down handler ────────────────────────────────────────────────
  function handleDrillDown(nodeSlug: string, periodStart: Date) {
    drillNode = nodeSlug;
    drillPeriod = periodStart;

    const msPerPeriod = granularity === 'week' ? 7 * 86400e3 : 86400e3;
    const periodEnd = new Date(periodStart.getTime() + msPerPeriod);

    drillEvents = events.filter((e) => {
      const t = new Date(e.ts).getTime();
      const eNode =
        (e.metadata?.node as string | undefined) ??
        (e.target_uri ? e.target_uri.split('/').slice(-2, -1)[0] ?? '' : '');
      return eNode === nodeSlug && t >= periodStart.getTime() && t < periodEnd.getTime();
    });
  }

  function closeDrill() {
    drillNode = null;
    drillPeriod = null;
    drillEvents = [];
  }

  function fmtPeriod(d: Date | null): string {
    if (!d) return '';
    return d.toLocaleDateString(undefined, { year: 'numeric', month: 'short', day: 'numeric' });
  }
</script>

<div class="page page--wide">
  <div class="page-header">
    <h1>Heatmap</h1>
    <p>
      Node x {granularity} ingest intensity.
      {#if $activeWorkspace}
        <span class="muted">· workspace: <code>{$activeWorkspace.slug}</code></span>
      {/if}
    </p>

    <div class="hm-controls">
      <label class="ctrl-label">
        Granularity
        <select
          class="ctrl-select"
          bind:value={granularity}
          onchange={() => { closeDrill(); periods = granularity === 'week' ? 12 : 30; }}
        >
          <option value="week">Week</option>
          <option value="day">Day</option>
        </select>
      </label>

      <label class="ctrl-label">
        Periods
        <select class="ctrl-select" bind:value={periods}>
          {#if granularity === 'week'}
            <option value={6}>6 weeks</option>
            <option value={12}>12 weeks</option>
            <option value={24}>24 weeks</option>
          {:else}
            <option value={14}>14 days</option>
            <option value={30}>30 days</option>
            <option value={60}>60 days</option>
          {/if}
        </select>
      </label>

      <button class="btn btn--sm" onclick={() => load($activeWorkspaceId)} disabled={loading}>
        {loading ? 'Loading…' : 'Refresh'}
      </button>

      <span class="ctrl-count muted">
        {events.length} events · {nodes.length} nodes
      </span>
    </div>
  </div>

  {#if error}
    <pre class="error">{error}</pre>
  {/if}

  {#if loading && events.length === 0}
    <div class="hm-loading card">
      <div class="hm-loading__dots">
        <span></span><span></span><span></span>
      </div>
      <p class="muted">Loading activity…</p>
    </div>
  {:else if !loading && events.length === 0 && !error}
    <div class="hm-empty card">
      <div class="hm-empty__icon">◇</div>
      <h2>No activity data</h2>
      <p>
        The heatmap aggregates signal ingest events by node and {granularity}.
        Run <code>mix optimal.bootstrap</code> or ingest signals to populate it.
      </p>
    </div>
  {:else}
    <div class="card hm-card">
      <ActivityHeatmap
        {events}
        {nodes}
        {granularity}
        {periods}
        onDrillDown={handleDrillDown}
      />
    </div>

    <!-- Drill-down panel -->
    {#if drillNode && drillPeriod}
      <div class="drill card">
        <div class="drill__header">
          <div>
            <span class="eyebrow">Drill-down</span>
            <h3 class="drill__title">
              {drillNode}
              <span class="muted">·</span>
              {fmtPeriod(drillPeriod)}
            </h3>
          </div>
          <button class="btn btn--sm btn--ghost" onclick={closeDrill} aria-label="Close drill-down">
            ✕ Close
          </button>
        </div>

        {#if drillEvents.length === 0}
          <p class="muted" style="padding: 0.5rem 0;">No matching events.</p>
        {:else}
          <div class="drill__count">
            <span class="chip chip--accent">{drillEvents.length} event{drillEvents.length !== 1 ? 's' : ''}</span>
          </div>
          <table class="drill__table">
            <thead>
              <tr>
                <th>Time</th>
                <th>Kind</th>
                <th>Principal</th>
                <th>Target</th>
              </tr>
            </thead>
            <tbody>
              {#each drillEvents.slice(0, 50) as e (e.id)}
                <tr>
                  <td class="mono">{new Date(e.ts).toLocaleString()}</td>
                  <td><span class="drill__kind">{e.kind}</span></td>
                  <td class="mono muted">{e.principal}</td>
                  <td class="mono muted">{e.target_uri ?? '—'}</td>
                </tr>
              {/each}
            </tbody>
          </table>
          {#if drillEvents.length > 50}
            <p class="muted" style="font-size: 0.8rem; margin-top: 0.5rem;">
              Showing 50 of {drillEvents.length} events.
            </p>
          {/if}
        {/if}
      </div>
    {/if}
  {/if}
</div>

<style>
  .hm-controls {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    flex-wrap: wrap;
    margin-left: auto;
  }

  .ctrl-label {
    display: inline-flex;
    align-items: center;
    gap: 0.4rem;
    font-size: 0.78rem;
    color: var(--text-muted);
    font-weight: 500;
  }

  .ctrl-select {
    background: var(--bg-elevated);
    color: var(--text);
    border: 1px solid var(--border);
    border-radius: var(--r-md);
    padding: 0.25rem 0.5rem;
    font: inherit;
    font-size: 0.8rem;
    cursor: pointer;
  }
  .ctrl-select:focus {
    outline: none;
    border-color: var(--accent);
    box-shadow: 0 0 0 3px var(--accent-soft);
  }

  .ctrl-count {
    font-size: 0.75rem;
    font-family: var(--font-mono);
  }

  .hm-card {
    padding: 1.25rem 1.4rem;
  }

  /* Loading */
  .hm-loading {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 1rem;
    padding: 3rem;
  }
  .hm-loading__dots {
    display: flex;
    gap: 0.5rem;
  }
  .hm-loading__dots span {
    width: 8px;
    height: 8px;
    border-radius: 50%;
    background: var(--accent);
    animation: bounce 1.2s infinite;
  }
  .hm-loading__dots span:nth-child(2) { animation-delay: 0.2s; }
  .hm-loading__dots span:nth-child(3) { animation-delay: 0.4s; }
  @keyframes bounce {
    0%, 80%, 100% { transform: scale(0.7); opacity: 0.5; }
    40%            { transform: scale(1);   opacity: 1; }
  }

  /* Empty */
  .hm-empty {
    text-align: center;
    max-width: 560px;
    margin: 2rem auto;
    padding: 3rem 2rem;
  }
  .hm-empty__icon {
    color: var(--accent);
    font-size: 2rem;
    margin-bottom: 0.75rem;
  }
  .hm-empty h2 {
    margin: 0 0 0.5rem;
    font-size: 1.2rem;
    font-weight: 600;
  }
  .hm-empty p {
    color: var(--text-muted);
    font-size: 0.9rem;
    line-height: 1.55;
    margin: 0;
  }
  .hm-empty code {
    background: var(--bg-elevated-2);
    border: 1px solid var(--border);
    padding: 1px 6px;
    border-radius: 4px;
    font-size: 0.82em;
    color: var(--text);
  }

  /* Drill-down panel */
  .drill {
    margin-top: 1rem;
    padding: 1.1rem 1.25rem;
  }
  .drill__header {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    margin-bottom: 0.85rem;
    gap: 1rem;
  }
  .drill__title {
    margin: 0.2rem 0 0;
    font-size: 1rem;
    font-weight: 600;
    color: var(--text);
  }
  .drill__count {
    margin-bottom: 0.6rem;
  }
  .drill__table {
    width: 100%;
    border-collapse: collapse;
    font-size: 0.78rem;
  }
  .drill__table th {
    text-align: left;
    padding: 0.35rem 0.6rem;
    color: var(--text-subtle);
    font-weight: 600;
    font-size: 0.68rem;
    text-transform: uppercase;
    letter-spacing: 0.06em;
    border-bottom: 1px solid var(--border);
  }
  .drill__table td {
    padding: 0.35rem 0.6rem;
    border-bottom: 1px solid var(--border-soft);
    vertical-align: top;
  }
  .drill__kind {
    font-weight: 600;
    font-size: 0.78rem;
    color: var(--accent);
  }
  .mono {
    font-family: var(--font-mono);
    font-size: 0.75rem;
  }
</style>
