<script lang="ts">
  import { onMount } from 'svelte';
  import ActivityTimeline from '$lib/components/viz/ActivityTimeline.svelte';
  import { getActivity, listWorkspace, type ActivityEvent, type WorkspaceNode } from '$lib/api/workspace';
  import { listWiki, type WikiPageSummary } from '$lib/api';
  import { activeWorkspaceId, activeWorkspace } from '$lib/stores/workspace';

  // ── Config state ──────────────────────────────────────────────────────
  let windowDays = $state(30);
  let groupBy = $state<'intent' | 'node' | 'audience'>('intent');

  // ── Data state ────────────────────────────────────────────────────────
  let events = $state<ActivityEvent[]>([]);
  let wikiPages = $state<WikiPageSummary[]>([]);
  let loading = $state(false);
  let error = $state<string | null>(null);

  // ── Config (from /api/workspaces/:id/config, optional) ────────────────
  async function loadConfig(wsId: string | null) {
    if (!wsId) return;
    try {
      const res = await fetch(`/api/workspaces/${encodeURIComponent(wsId)}/config`);
      if (!res.ok) return; // 404 → use defaults silently
      const cfg = (await res.json()) as {
        visualizations?: {
          enabled?: string[];
          timeline?: {
            group_by?: string;
            default_window_days?: number;
          };
        };
      };
      const tlCfg = cfg.visualizations?.timeline;
      if (tlCfg?.default_window_days) windowDays = tlCfg.default_window_days;
      if (tlCfg?.group_by && ['intent', 'node', 'audience'].includes(tlCfg.group_by)) {
        groupBy = tlCfg.group_by as 'intent' | 'node' | 'audience';
      }
    } catch {
      // Config unavailable — silent, use defaults
    }
  }

  // ── Fetch data ─────────────────────────────────────────────────────────
  async function load(wsId: string | null) {
    loading = true;
    error = null;
    try {
      const [evts, wiki] = await Promise.allSettled([
        getActivity({ limit: 200, workspace: wsId }),
        listWiki({ workspace: wsId ?? 'default' }),
      ]);
      events = evts.status === 'fulfilled' ? evts.value : [];
      wikiPages = wiki.status === 'fulfilled' ? wiki.value.pages : [];
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

  function fmtCount(n: number, noun: string) {
    return `${n} ${noun}${n !== 1 ? 's' : ''}`;
  }
</script>

<div class="page page--wide">
  <div class="page-header">
    <h1>Timeline</h1>
    <p>
      Signal activity over time, color-coded by intent.
      {#if $activeWorkspace}
        <span class="muted">· workspace: <code>{$activeWorkspace.slug}</code></span>
      {/if}
    </p>

    <div class="tl-controls">
      <label class="ctrl-label">
        Window
        <select
          class="ctrl-select"
          bind:value={windowDays}
          onchange={() => load($activeWorkspaceId)}
        >
          <option value={7}>7 days</option>
          <option value={14}>14 days</option>
          <option value={30}>30 days</option>
          <option value={60}>60 days</option>
          <option value={90}>90 days</option>
        </select>
      </label>

      <label class="ctrl-label">
        Group by
        <select class="ctrl-select" bind:value={groupBy}>
          <option value="intent">intent</option>
          <option value="node">node</option>
          <option value="audience">audience</option>
        </select>
      </label>

      <button class="btn btn--sm" onclick={() => load($activeWorkspaceId)} disabled={loading}>
        {loading ? 'Loading…' : 'Refresh'}
      </button>

      <span class="ctrl-count muted">
        {fmtCount(events.length, 'event')} · {fmtCount(wikiPages.length, 'wiki page')}
      </span>
    </div>
  </div>

  {#if error}
    <pre class="error">{error}</pre>
  {/if}

  {#if loading && events.length === 0}
    <div class="tl-loading card">
      <div class="tl-loading__dots">
        <span></span><span></span><span></span>
      </div>
      <p class="muted">Loading activity…</p>
    </div>
  {:else if !loading && events.length === 0 && wikiPages.length === 0 && !error}
    <div class="tl-empty card">
      <div class="tl-empty__icon">◇</div>
      <h2>No activity yet</h2>
      <p>
        The timeline shows signal ingest events and wiki updates.
        Run <code>mix optimal.bootstrap</code> or ingest signals to see data here.
      </p>
    </div>
  {:else}
    <div class="card tl-card">
      <ActivityTimeline
        {events}
        {wikiPages}
        {windowDays}
        {groupBy}
      />
    </div>

    <div class="tl-stats">
      <div class="tl-stat card">
        <span class="tl-stat__value">{events.length}</span>
        <span class="tl-stat__label eyebrow">Total events</span>
      </div>
      <div class="tl-stat card">
        <span class="tl-stat__value">{wikiPages.length}</span>
        <span class="tl-stat__label eyebrow">Wiki pages</span>
      </div>
      <div class="tl-stat card">
        <span class="tl-stat__value">{windowDays}d</span>
        <span class="tl-stat__label eyebrow">Time window</span>
      </div>
      <div class="tl-stat card">
        {#if events.length > 0}
          {@const kinds = [...new Set(events.map((e) => e.kind))]}
          <span class="tl-stat__value">{kinds.length}</span>
          <span class="tl-stat__label eyebrow">Event kinds</span>
        {:else}
          <span class="tl-stat__value">—</span>
          <span class="tl-stat__label eyebrow">Event kinds</span>
        {/if}
      </div>
    </div>
  {/if}
</div>

<style>
  .tl-controls {
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

  .tl-card {
    padding: 1.25rem 1.4rem;
    overflow: hidden;
  }

  /* Loading skeleton */
  .tl-loading {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 1rem;
    padding: 3rem;
  }
  .tl-loading__dots {
    display: flex;
    gap: 0.5rem;
  }
  .tl-loading__dots span {
    width: 8px;
    height: 8px;
    border-radius: 50%;
    background: var(--accent);
    animation: bounce 1.2s infinite;
  }
  .tl-loading__dots span:nth-child(2) { animation-delay: 0.2s; }
  .tl-loading__dots span:nth-child(3) { animation-delay: 0.4s; }

  @keyframes bounce {
    0%, 80%, 100% { transform: scale(0.7); opacity: 0.5; }
    40%            { transform: scale(1);   opacity: 1; }
  }

  /* Empty state */
  .tl-empty {
    text-align: center;
    max-width: 560px;
    margin: 2rem auto;
    padding: 3rem 2rem;
  }
  .tl-empty__icon {
    color: var(--accent);
    font-size: 2rem;
    margin-bottom: 0.75rem;
  }
  .tl-empty h2 {
    margin: 0 0 0.5rem;
    font-size: 1.2rem;
    font-weight: 600;
  }
  .tl-empty p {
    color: var(--text-muted);
    font-size: 0.9rem;
    line-height: 1.55;
    margin: 0;
  }
  .tl-empty code {
    background: var(--bg-elevated-2);
    border: 1px solid var(--border);
    padding: 1px 6px;
    border-radius: 4px;
    font-size: 0.82em;
    color: var(--text);
  }

  /* Stats row */
  .tl-stats {
    display: grid;
    grid-template-columns: repeat(4, 1fr);
    gap: 0.75rem;
    margin-top: 1rem;
  }
  .tl-stat {
    padding: 0.85rem 1rem;
    display: flex;
    flex-direction: column;
    gap: 0.25rem;
  }
  .tl-stat__value {
    font-size: 1.5rem;
    font-weight: 700;
    color: var(--text);
    font-family: var(--font-mono);
    line-height: 1;
  }
  .tl-stat__label {
    font-size: 0.65rem;
  }

  @media (max-width: 680px) {
    .tl-stats { grid-template-columns: repeat(2, 1fr); }
    .tl-controls { flex-direction: column; align-items: flex-start; }
  }
</style>
