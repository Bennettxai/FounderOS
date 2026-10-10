<script lang="ts">
  import { onMount } from 'svelte';
  import { listWiki, getWiki, type WikiPageSummary } from '$lib/api';
  import { activeWorkspaceId, activeWorkspace } from '$lib/stores/workspace';

  let pages = $state<WikiPageSummary[]>([]);
  let selected = $state<string | null>(null);
  let body = $state('');
  let loading = $state(false);
  let error = $state<string | null>(null);
  let audience = $state('default');

  async function refresh(workspace: string | null) {
    if (!workspace) return;
    error = null;
    loading = true;
    selected = null;
    body = '';
    try {
      const res = await listWiki({ workspace });
      pages = res.pages;
      if (pages.length > 0) {
        await open(pages[0].slug);
      }
    } catch (e) {
      error = (e as Error).message;
    } finally {
      loading = false;
    }
  }

  onMount(() => refresh($activeWorkspaceId));

  // Refresh when workspace switches
  $effect(() => {
    void refresh($activeWorkspaceId);
  });

  async function open(slug: string) {
    selected = slug;
    body = '';
    error = null;
    loading = true;
    try {
      const res = await getWiki(slug, {
        workspace: $activeWorkspaceId ?? 'default',
        audience
      });
      body = res.body;
    } catch (e) {
      error = (e as Error).message;
    } finally {
      loading = false;
    }
  }

  // Highlight {{cite: ...}} directives in the rendered body
  function renderBody(text: string): string {
    return text
      .replace(/\{\{cite:\s*([^}]+)\}\}/g, '<span class="cite">{{cite: $1}}</span>')
      .replace(/\{\{(include|expand|search|table|trace|recent):\s*([^}]+)\}\}/g, '<span class="directive directive--$1">{{$1: $2}}</span>')
      .replace(/\[\[([^\]]+)\]\]/g, '<span class="wikilink">[[$1]]</span>');
  }

  function fmtDate(iso: string | null): string {
    if (!iso) return '—';
    return new Date(iso).toLocaleDateString(undefined, { year: 'numeric', month: 'short', day: 'numeric' });
  }
</script>

<div class="page page--wide">
  <div class="page-header">
    <h1>Wiki</h1>
    <p>
      Tier 3 — LLM-curated pages with hot citations.
      {#if $activeWorkspace}
        <span class="muted">· workspace: <code>{$activeWorkspace.slug}</code></span>
      {/if}
    </p>
    <div style="margin-left: auto; display: flex; gap: 0.5rem; align-items: center;">
      <label class="audience-label">
        Audience
        <select class="audience-select" bind:value={audience} onchange={() => selected && open(selected)}>
          <option value="default">default</option>
          <option value="sales">sales</option>
          <option value="legal">legal</option>
          <option value="exec">exec</option>
          <option value="engineering">engineering</option>
        </select>
      </label>
      <button class="btn btn--sm" onclick={() => refresh($activeWorkspaceId)}>Refresh</button>
    </div>
  </div>

  {#if error}
    <pre class="error">{error}</pre>
  {/if}

  {#if !loading && pages.length === 0 && !error}
    <!-- Empty state — explain what's needed -->
    <div class="empty">
      <div class="empty__icon">◇</div>
      <h2>No wiki pages yet.</h2>
      <p>
        The wiki is built by <strong>Stage 9 · Curate</strong> after signals are ingested and clustered.
        With the engine running on a fresh database, there's nothing to curate yet.
      </p>
      <h3>Bootstrap from the sample workspace</h3>
      <p>Stop the engine and run:</p>
      <pre class="code-block">make bootstrap</pre>
      <p>or:</p>
      <pre class="code-block">mix optimal.bootstrap</pre>
      <p>
        That ingests <code>sample-workspace/</code> (6 nodes, 13 signals, 2 curated wiki pages) through the
        full 9-stage pipeline. Then come back here and click Refresh.
      </p>
    </div>
  {:else}
    <div class="wiki">
      <aside class="wiki__list">
        <header class="wiki__list-head">
          <span class="eyebrow">Pages</span>
          <span class="chip">{pages.length}</span>
        </header>
        <ul>
          {#each pages as p}
            <li class:selected={selected === p.slug}>
              <button onclick={() => open(p.slug)}>
                <div class="wiki__list-slug">{p.slug}</div>
                <div class="wiki__list-meta">
                  <span class="chip chip--accent">v{p.version}</span>
                  <span class="chip">{p.audience}</span>
                  <span class="muted">{Math.round(p.size_bytes / 1024)}kb</span>
                </div>
                <div class="wiki__list-date">curated {fmtDate(p.last_curated)}</div>
              </button>
            </li>
          {/each}
        </ul>
      </aside>

      <article class="wiki__page card">
        {#if !selected}
          <p class="muted">Pick a page to render.</p>
        {:else if loading}
          <p class="muted">Loading {selected}…</p>
        {:else if body}
          <header class="wiki__page-head">
            <code class="wiki__page-slug">{selected}</code>
            <div class="wiki__page-chips">
              <span class="chip chip--accent">audience: {audience}</span>
              <span class="chip chip--good">curated</span>
            </div>
          </header>
          <div class="wiki__body">{@html renderBody(body)}</div>
        {/if}
      </article>
    </div>
  {/if}
</div>

<style>
  .audience-label {
    display: inline-flex;
    align-items: center;
    gap: 0.4rem;
    font-size: 0.78rem;
    color: var(--text-muted);
  }
  .audience-select {
    background: var(--bg-elevated);
    color: var(--text);
    border: 1px solid var(--border);
    border-radius: var(--r-md);
    padding: 0.25rem 0.5rem;
    font: inherit;
    font-size: 0.8rem;
  }

  /* Empty state */
  .empty {
    text-align: center;
    background: var(--bg-elevated);
    border: 1px solid var(--border);
    border-radius: 16px;
    padding: 3.5rem 2rem 3rem;
    max-width: 640px;
    margin: 2rem auto;
  }
  .empty__icon {
    color: var(--accent);
    font-size: 2rem;
    line-height: 1;
    margin-bottom: 0.6rem;
  }
  .empty h2 {
    margin: 0 0 0.5rem;
    font-size: 1.25rem;
    font-weight: 600;
  }
  .empty h3 {
    margin: 1.6rem 0 0.5rem;
    font-size: 0.95rem;
    color: var(--text);
    font-weight: 600;
  }
  .empty p {
    color: var(--text-muted);
    margin: 0 0 0.6rem;
    font-size: 0.9rem;
    line-height: 1.55;
  }
  .empty p strong { color: var(--text); }
  .empty code {
    background: var(--bg-elevated-2);
    border: 1px solid var(--border);
    padding: 1px 6px;
    border-radius: 4px;
    color: var(--text);
    font-size: 0.82em;
  }
  .code-block {
    background: var(--bg);
    border: 1px solid var(--border);
    border-radius: var(--r-md);
    padding: 0.6rem 0.85rem;
    margin: 0.4rem auto 0.7rem;
    color: var(--text);
    font-family: var(--font-mono);
    font-size: 0.85rem;
    text-align: left;
    max-width: 360px;
  }

  /* Two-column wiki layout */
  .wiki {
    display: grid;
    grid-template-columns: 280px 1fr;
    gap: 1rem;
    align-items: start;
  }

  .wiki__list {
    background: var(--bg-elevated);
    border: 1px solid var(--border);
    border-radius: var(--r-md);
    padding: 0.75rem;
    position: sticky;
    top: 72px;
  }
  .wiki__list-head {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 0.2rem 0.4rem 0.6rem;
  }
  .wiki__list ul {
    list-style: none;
    padding: 0;
    margin: 0;
    display: flex;
    flex-direction: column;
    gap: 0.25rem;
  }
  .wiki__list li button {
    width: 100%;
    background: transparent;
    border: 1px solid transparent;
    color: var(--text);
    text-align: left;
    padding: 0.55rem 0.65rem;
    border-radius: var(--r-sm);
    cursor: pointer;
    font: inherit;
    transition: background 0.12s, border-color 0.12s;
  }
  .wiki__list li button:hover {
    background: var(--bg-elevated-2);
  }
  .wiki__list li.selected button {
    background: var(--accent-soft);
    border-color: var(--accent);
  }
  .wiki__list-slug {
    font-size: 0.85rem;
    font-weight: 500;
    color: var(--text);
    margin-bottom: 0.3rem;
    word-break: break-word;
  }
  .wiki__list-meta {
    display: flex;
    gap: 0.3rem;
    align-items: center;
    margin-bottom: 0.2rem;
    flex-wrap: wrap;
  }
  .wiki__list-date {
    color: var(--text-subtle);
    font-size: 0.72rem;
  }

  .wiki__page {
    padding: 0;
    overflow: hidden;
  }
  .wiki__page-head {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 0.75rem;
    padding: 0.85rem 1.2rem;
    border-bottom: 1px solid var(--border-soft);
    background: var(--bg);
  }
  .wiki__page-slug {
    font-family: var(--font-mono);
    font-size: 0.82rem;
    color: var(--text);
  }
  .wiki__page-chips {
    display: flex;
    gap: 0.35rem;
  }
  .wiki__body {
    padding: 1.5rem 1.7rem;
    color: var(--text);
    font-size: 0.92rem;
    line-height: 1.7;
    white-space: pre-wrap;
    font-family: var(--font-sans);
  }
  /* Heading-ish detection inside the rendered prose */
  .wiki__body :global(.cite) {
    color: var(--accent);
    background: var(--accent-soft);
    padding: 1px 6px;
    border-radius: 4px;
    font-family: var(--font-mono);
    font-size: 0.78rem;
  }
  .wiki__body :global(.directive) {
    background: rgba(187, 126, 255, 0.1);
    color: var(--purple);
    padding: 1px 6px;
    border-radius: 4px;
    font-family: var(--font-mono);
    font-size: 0.78rem;
  }
  .wiki__body :global(.directive--include) { background: rgba(95, 207, 212, 0.1); color: var(--cyan); }
  .wiki__body :global(.directive--expand)  { background: rgba(252, 158, 108, 0.1); color: var(--amber); }
  .wiki__body :global(.wikilink) {
    color: var(--accent);
    border-bottom: 1px dashed var(--accent);
  }

  @media (max-width: 880px) {
    .wiki {
      grid-template-columns: 1fr;
    }
    .wiki__list {
      position: static;
    }
  }
</style>
