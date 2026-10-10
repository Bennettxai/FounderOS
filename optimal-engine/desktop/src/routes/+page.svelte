<script lang="ts">
  import { ask, type RagResult } from '$lib/api';
  import { activeWorkspaceId } from '$lib/stores/workspace';

  let query = $state('');
  let audience = $state('default');
  let loading = $state(false);
  let result = $state<RagResult | null>(null);
  let error = $state<string | null>(null);

  const seeds: { label: string; query: string }[] = [
    { label: 'Pricing decision',     query: 'customer portal pricing decision' },
    { label: 'Platform architecture', query: 'core platform architecture' },
    { label: 'Renewal pipeline',     query: 'renewal pipeline status' },
    { label: 'MicroVM spec',         query: 'microvm provisioning spec' }
  ];

  async function run(q: string) {
    query = q;
    loading = true;
    error = null;
    result = null;
    try {
      result = await ask(q, { audience, format: 'markdown', workspace: $activeWorkspaceId });
    } catch (e) {
      error = (e as Error).message;
    } finally {
      loading = false;
    }
  }

  async function submit(e: Event) {
    e.preventDefault();
    if (!query.trim()) return;
    await run(query);
  }

  // Clear result when the workspace switches so stale data is never shown
  $effect(() => {
    $activeWorkspaceId;
    result = null;
    error = null;
  });

  function sourceChip(s: string): string {
    if (s === 'wiki') return 'good';
    if (s === 'chunks') return 'accent';
    return 'warn';
  }
</script>

<div class="page">
  <div class="page-header">
    <h1>Ask the engine</h1>
    <p>Curated wiki first. Hybrid search second. ACL-scoped · audience-shaped · bandwidth-matched.</p>
  </div>

  <form onsubmit={submit} class="ask-form card">
    <div class="ask-form__row">
      <input
        type="text"
        bind:value={query}
        placeholder="e.g. customer portal pricing"
        disabled={loading}
        class="ask-form__input"
      />
      <select bind:value={audience} disabled={loading} class="ask-form__select">
        <option value="default">default</option>
        <option value="sales">sales</option>
        <option value="legal">legal</option>
        <option value="exec">exec</option>
        <option value="engineering">engineering</option>
      </select>
      <button type="submit" disabled={loading || !query.trim()} class="btn btn--primary">
        {loading ? 'Thinking…' : 'Ask'}
      </button>
    </div>

    <div class="ask-form__seeds">
      <span class="eyebrow">Try</span>
      {#each seeds as seed}
        <button type="button" class="seed-chip" onclick={() => run(seed.query)} disabled={loading}>
          {seed.label}
        </button>
      {/each}
    </div>
  </form>

  {#if error}
    <pre class="error" style="margin-top: 1rem;">{error}</pre>
  {/if}

  {#if result}
    <section class="ask-result">
      <div class="ask-trace">
        <span class="chip chip--{sourceChip(result.source)}">{result.source}</span>
        {#if result.trace['wiki_hit?']}
          <span class="chip chip--good">wiki hit</span>
        {/if}
        <span class="ask-stat"><small>candidates</small>{result.trace.n_candidates}</span>
        <span class="ask-stat"><small>delivered</small>{result.trace.n_delivered}</span>
        <span class="ask-stat"><small>elapsed</small>{result.trace.elapsed_ms} ms</span>
        {#if result.trace['truncated?']}
          <span class="chip chip--warn">truncated</span>
        {/if}
      </div>

      <article class="ask-envelope card">
        <header class="ask-envelope__head">
          <span class="eyebrow">Envelope · {result.envelope.format}</span>
        </header>
        <pre class="ask-envelope__body">{result.envelope.body}</pre>
      </article>

      {#if result.envelope.sources.length > 0}
        <section class="ask-sources card">
          <header class="ask-sources__head">
            <span class="eyebrow">Sources</span>
            <span class="chip">{result.envelope.sources.length}</span>
          </header>
          <ul>
            {#each result.envelope.sources as uri}
              <li><code>{uri}</code></li>
            {/each}
          </ul>
        </section>
      {/if}

      {#if result.envelope.warnings.length > 0}
        <section class="card" style="margin-top: 0.75rem;">
          <span class="eyebrow">Warnings</span>
          <ul style="margin: 0.4rem 0 0; padding-left: 1.2rem; color: var(--warn);">
            {#each result.envelope.warnings as w}
              <li style="font-size: 0.85rem;">{w}</li>
            {/each}
          </ul>
        </section>
      {/if}
    </section>
  {/if}
</div>

<style>
  .ask-form {
    padding: 1.1rem 1.2rem 0.85rem;
  }
  .ask-form__row {
    display: flex;
    gap: 0.5rem;
  }
  .ask-form__input,
  .ask-form__select {
    background: var(--bg);
    color: var(--text);
    border: 1px solid var(--border);
    border-radius: var(--r-md);
    padding: 0.55rem 0.8rem;
    font: inherit;
    font-size: 0.9rem;
  }
  .ask-form__input { flex: 1; }
  .ask-form__input:focus,
  .ask-form__select:focus {
    outline: none;
    border-color: var(--accent);
    box-shadow: 0 0 0 3px var(--accent-soft);
  }

  .ask-form__seeds {
    display: flex;
    align-items: center;
    flex-wrap: wrap;
    gap: 0.4rem;
    margin-top: 0.85rem;
  }
  .seed-chip {
    background: var(--bg);
    color: var(--text-muted);
    border: 1px solid var(--border);
    border-radius: 999px;
    padding: 0.3rem 0.7rem;
    font: inherit;
    font-size: 0.78rem;
    cursor: pointer;
    transition: background 0.12s, border-color 0.12s, color 0.12s;
  }
  .seed-chip:hover:not(:disabled) {
    background: var(--bg-elevated-2);
    border-color: var(--accent);
    color: var(--text);
  }
  .seed-chip:disabled { opacity: 0.4; cursor: not-allowed; }

  .ask-result { margin-top: 1.2rem; }
  .ask-trace {
    display: flex;
    gap: 0.5rem;
    align-items: center;
    flex-wrap: wrap;
    padding: 0.55rem 0.85rem;
    background: var(--bg-elevated);
    border: 1px solid var(--border);
    border-radius: var(--r-md);
    margin-bottom: 0.75rem;
  }
  .ask-stat {
    display: inline-flex;
    align-items: baseline;
    gap: 0.35rem;
    color: var(--text);
    font-size: 0.8rem;
  }
  .ask-stat small {
    color: var(--text-subtle);
    font-size: 0.65rem;
    text-transform: uppercase;
    letter-spacing: 0.08em;
    font-weight: 600;
  }

  .ask-envelope {
    padding: 0;
    overflow: hidden;
  }
  .ask-envelope__head {
    padding: 0.6rem 1rem;
    border-bottom: 1px solid var(--border-soft);
    background: var(--bg);
  }
  .ask-envelope__body {
    margin: 0;
    padding: 1.1rem 1.3rem;
    color: var(--text);
    font-family: var(--font-sans);
    font-size: 0.92rem;
    line-height: 1.65;
    white-space: pre-wrap;
  }

  .ask-sources { margin-top: 0.75rem; padding: 0; }
  .ask-sources__head {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 0.55rem 1rem;
    border-bottom: 1px solid var(--border-soft);
    background: var(--bg);
  }
  .ask-sources ul {
    list-style: none;
    margin: 0;
    padding: 0.5rem 1rem 0.7rem;
  }
  .ask-sources li {
    padding: 0.2rem 0;
  }
  .ask-sources code {
    color: var(--accent);
    font-family: var(--font-mono);
    font-size: 0.78rem;
    word-break: break-all;
  }
</style>
