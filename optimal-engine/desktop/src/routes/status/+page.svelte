<script lang="ts">
  import { onMount } from 'svelte';
  import { status, type StatusPayload } from '$lib/api';

  let payload = $state<StatusPayload | null>(null);
  let error = $state<string | null>(null);
  let loading = $state(false);

  async function refresh() {
    loading = true;
    error = null;
    try {
      payload = await status();
    } catch (e) {
      error = (e as Error).message;
    } finally {
      loading = false;
    }
  }

  onMount(refresh);

  function checkChip(value: string): string {
    if (value === ':ok' || value === 'ok') return 'good';
    if (value.includes('warn') || value.includes('degraded')) return 'warn';
    if (value.includes('error') || value.includes('fail') || value.includes('down')) return 'bad';
    return '';
  }
</script>

<div class="page">
  <div class="page-header">
    <h1>Status</h1>
    <p>Engine readiness probes.</p>
    <div style="margin-left: auto;">
      <button class="btn btn--sm" onclick={refresh} disabled={loading}>
        {loading ? 'Checking…' : 'Refresh'}
      </button>
    </div>
  </div>

  {#if error}
    <pre class="error">{error}</pre>
  {:else if !payload}
    <p class="muted">Loading…</p>
  {:else}
    <section class="status-overall card">
      <div class="status-overall__row">
        <span class="eyebrow">Overall</span>
        <span class="chip chip--{payload.status === 'up' ? 'good' : payload.status === 'down' ? 'bad' : 'warn'}">
          {payload.status}
        </span>
      </div>
      {#if payload['ok?']}
        <p class="muted">All checks passing.</p>
      {/if}
      {#if payload.degraded.length > 0}
        <p class="degraded">Degraded: {payload.degraded.join(', ')}</p>
      {/if}
    </section>

    <section class="card status-checks">
      <div class="eyebrow" style="margin-bottom: 0.7rem;">Checks</div>
      <table>
        <thead>
          <tr>
            <th>Name</th>
            <th>Result</th>
          </tr>
        </thead>
        <tbody>
          {#each Object.entries(payload.checks) as [name, value]}
            <tr>
              <td><code>{name}</code></td>
              <td>
                {#if checkChip(value)}
                  <span class="chip chip--{checkChip(value)}">{value}</span>
                {:else}
                  <code>{value}</code>
                {/if}
              </td>
            </tr>
          {/each}
        </tbody>
      </table>
    </section>
  {/if}
</div>

<style>
  .status-overall { margin-bottom: 1rem; }
  .status-overall__row {
    display: flex;
    align-items: center;
    gap: 0.7rem;
    margin-bottom: 0.5rem;
  }
  .status-overall__row .chip {
    font-size: 0.78rem;
    padding: 3px 12px;
  }
  .status-overall p { margin: 0; font-size: 0.88rem; }
  .degraded { color: var(--warn); }

  .status-checks table {
    width: 100%;
    border-collapse: collapse;
    margin-top: 0.4rem;
  }
  .status-checks th, .status-checks td {
    text-align: left;
    padding: 0.55rem 0.75rem;
    border-bottom: 1px solid var(--border-soft);
    font-size: 0.85rem;
  }
  .status-checks th {
    color: var(--text-muted);
    font-weight: 600;
    font-size: 0.7rem;
    text-transform: uppercase;
    letter-spacing: 0.08em;
  }
  .status-checks code {
    color: var(--text);
    font-family: var(--font-mono);
    font-size: 0.82rem;
  }
</style>
