<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import { getActivity, type ActivityEvent } from '$lib/api/workspace';
  import { activeWorkspaceId } from '$lib/stores/workspace';

  let events = $state<ActivityEvent[]>([]);
  let kind = $state('');
  let autoRefresh = $state(true);
  let error = $state<string | null>(null);
  let intervalId: ReturnType<typeof setInterval> | null = null;

  async function load() {
    try {
      events = await getActivity({
        limit: 200,
        kind: kind || undefined,
        workspace: $activeWorkspaceId,
      });
    } catch (e) {
      error = (e as Error).message;
    }
  }

  onMount(() => {
    load();
    intervalId = setInterval(() => autoRefresh && load(), 5000);
  });

  onDestroy(() => {
    if (intervalId) clearInterval(intervalId);
  });

  // Re-fetch when kind filter or active workspace changes
  $effect(() => {
    kind;
    $activeWorkspaceId;
    load();
  });

  function kindColor(k: string): string {
    if (k === 'ingest') return '#6f8';
    if (k === 'erasure') return '#f88';
    if (k === 'retention_action') return '#fc6';
    if (k === 'search') return '#8af';
    if (k === 'rag') return '#a8f';
    return '#ccc';
  }
</script>

<div class="activity">
  <header class="activity__header">
    <h2>Activity</h2>
    <label>
      kind:
      <select bind:value={kind}>
        <option value="">all</option>
        <option value="ingest">ingest</option>
        <option value="search">search</option>
        <option value="rag">rag</option>
        <option value="erasure">erasure</option>
        <option value="retention_action">retention_action</option>
      </select>
    </label>
    <label>
      <input type="checkbox" bind:checked={autoRefresh} />
      auto-refresh 5s
    </label>
    <span class="count">{events.length} events</span>
  </header>

  {#if error}
    <pre class="error">{error}</pre>
  {/if}

  <table class="activity__table">
    <thead>
      <tr>
        <th>ts</th>
        <th>kind</th>
        <th>principal</th>
        <th>target</th>
        <th>metadata</th>
      </tr>
    </thead>
    <tbody>
      {#each events as e (e.id)}
        <tr>
          <td class="mono">{e.ts}</td>
          <td>
            <span class="kind" style="color: {kindColor(e.kind)}">{e.kind}</span>
          </td>
          <td class="mono">{e.principal}</td>
          <td class="mono">{e.target_uri ?? ''}</td>
          <td class="meta">{Object.keys(e.metadata).length > 0 ? JSON.stringify(e.metadata) : ''}</td>
        </tr>
      {/each}
    </tbody>
  </table>
</div>

<style>
  .activity {
    padding: 1rem 1.5rem;
    color: #ddd;
    height: calc(100vh - 64px);
    overflow: auto;
    background: #0b0d10;
  }
  .activity__header {
    display: flex;
    align-items: center;
    gap: 1rem;
    margin-bottom: 1rem;
  }
  h2 {
    margin: 0;
    font-size: 1rem;
    color: #ccc;
  }
  label {
    display: flex;
    align-items: center;
    gap: 0.4rem;
    font-size: 0.85rem;
    color: #888;
  }
  select,
  input[type='text'] {
    background: #0e1116;
    color: #ddd;
    border: 1px solid #333;
    border-radius: 4px;
    padding: 3px 6px;
    font-size: 0.85rem;
  }
  .count {
    margin-left: auto;
    color: #666;
    font-size: 0.8rem;
  }
  .activity__table {
    width: 100%;
    border-collapse: collapse;
    font-size: 0.8rem;
  }
  th {
    text-align: left;
    padding: 0.4rem 0.6rem;
    color: #666;
    font-weight: 500;
    border-bottom: 1px solid #222;
    font-size: 0.7rem;
    text-transform: uppercase;
    letter-spacing: 0.5px;
  }
  td {
    padding: 0.45rem 0.6rem;
    border-bottom: 1px solid #141820;
    vertical-align: top;
  }
  .mono {
    font-family: 'SF Mono', Menlo, monospace;
    color: #bbb;
  }
  .kind {
    font-weight: 600;
    font-size: 0.8rem;
  }
  .meta {
    color: #888;
    max-width: 380px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .error {
    color: #f88;
    padding: 0.5rem;
    background: #311;
    border-radius: 4px;
  }
</style>
