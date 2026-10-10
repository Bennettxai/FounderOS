<script lang="ts">
  import {
    listMemories,
    memoryRelations,
    forgetMemory,
    memoryVersions,
    type Memory,
    type MemoryRelation,
    type MemoryRelationType,
  } from '$lib/api';
  import { activeWorkspaceId, activeWorkspace } from '$lib/stores/workspace';
  import MemoryGraph from '$lib/components/memory/MemoryGraph.svelte';
  import AddMemoryModal from '$lib/components/memory/AddMemoryModal.svelte';

  // ── Filters ──────────────────────────────────────────────────────────────
  let audience = $state('');
  let showForgotten = $state(false);
  let showOldVersions = $state(false);
  let sortBy = $state<'recency' | 'audience' | 'static'>('recency');

  // ── Data state ───────────────────────────────────────────────────────────
  let memories = $state<Memory[]>([]);
  let allRelations = $state<MemoryRelation[]>([]);
  let loading = $state(false);
  let error = $state<string | null>(null);

  // ── Selected memory detail ───────────────────────────────────────────────
  let selectedId = $state<string | null>(null);
  let selectedRelations = $state<{ inbound: MemoryRelation[]; outbound: MemoryRelation[] } | null>(null);
  let selectedVersions = $state<Memory[]>([]);
  let detailLoading = $state(false);

  // ── Modal state ──────────────────────────────────────────────────────────
  let showAddModal = $state(false);

  // ── Derived sorted list ──────────────────────────────────────────────────
  let sortedMemories = $derived(() => {
    const list = [...memories];
    if (sortBy === 'recency') {
      list.sort((a, b) => b.created_at.localeCompare(a.created_at));
    } else if (sortBy === 'audience') {
      list.sort((a, b) => a.audience.localeCompare(b.audience));
    } else if (sortBy === 'static') {
      list.sort((a, b) => Number(b.is_static) - Number(a.is_static));
    }
    return list;
  });

  let selectedMemory = $derived(memories.find((m) => m.id === selectedId) ?? null);

  // ── Load memories ────────────────────────────────────────────────────────
  async function load(wsId: string | null) {
    loading = true;
    error = null;
    try {
      const res = await listMemories({
        workspace: wsId ?? undefined,
        audience: audience || undefined,
        includeForgotten: showForgotten,
        includeOldVersions: showOldVersions,
        limit: 200,
      });
      memories = res.memories;
      // Gather all relations for graph edges
      await loadAllRelations(res.memories);
    } catch (e) {
      error = (e as Error).message;
      memories = [];
      allRelations = [];
    } finally {
      loading = false;
    }
  }

  async function loadAllRelations(mems: Memory[]) {
    if (mems.length === 0) { allRelations = []; return; }
    try {
      const results = await Promise.allSettled(
        mems.map((m) => memoryRelations(m.id))
      );
      const seen = new Set<string>();
      const flat: MemoryRelation[] = [];
      for (const r of results) {
        if (r.status !== 'fulfilled') continue;
        for (const rel of [...r.value.inbound, ...r.value.outbound]) {
          const key = `${rel.source_memory_id}-${rel.target_memory_id}-${rel.relation}`;
          if (!seen.has(key)) {
            seen.add(key);
            flat.push(rel);
          }
        }
      }
      allRelations = flat;
    } catch {
      allRelations = [];
    }
  }

  // ── Load detail for selected memory ─────────────────────────────────────
  async function loadDetail(id: string) {
    detailLoading = true;
    selectedRelations = null;
    selectedVersions = [];
    try {
      const [rels, vers] = await Promise.allSettled([
        memoryRelations(id),
        memoryVersions(id),
      ]);
      selectedRelations = rels.status === 'fulfilled'
        ? { inbound: rels.value.inbound, outbound: rels.value.outbound }
        : { inbound: [], outbound: [] };
      selectedVersions = vers.status === 'fulfilled' ? vers.value.versions : [];
    } finally {
      detailLoading = false;
    }
  }

  function selectMemory(id: string) {
    if (selectedId === id) {
      selectedId = null;
      selectedRelations = null;
      selectedVersions = [];
      return;
    }
    selectedId = id;
    void loadDetail(id);
  }

  // ── Forget a memory ──────────────────────────────────────────────────────
  async function handleForget(id: string) {
    if (!confirm('Mark this memory as forgotten?')) return;
    try {
      await forgetMemory(id, { reason: 'user request' });
      await load($activeWorkspaceId);
    } catch (e) {
      alert((e as Error).message);
    }
  }

  // ── Reactive workspace switch (also runs on mount) ───────────────────────
  $effect(() => {
    const wsId = $activeWorkspaceId;
    selectedId = null;
    selectedRelations = null;
    selectedVersions = [];
    void load(wsId);
  });

  // ── Helpers ──────────────────────────────────────────────────────────────
  const RELATION_LABEL: Record<MemoryRelationType, string> = {
    updates:     'updates',
    extends:     'extends',
    derives:     'derives',
    contradicts: 'contradicts',
    cites:       'cites',
  };

  function snip(text: string, len = 90): string {
    return text.length > len ? text.slice(0, len) + '…' : text;
  }

  function fmtDate(iso: string | null): string {
    if (!iso) return '—';
    return new Date(iso).toLocaleDateString(undefined, {
      month: 'short',
      day: 'numeric',
      year: '2-digit',
    });
  }

  function fmtDateLong(iso: string | null): string {
    if (!iso) return '—';
    return new Date(iso).toLocaleString(undefined, {
      month: 'short',
      day: 'numeric',
      year: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
    });
  }

  function relColor(rel: MemoryRelationType): string {
    const map: Record<MemoryRelationType, string> = {
      updates:     'accent',
      extends:     'good',
      derives:     'purple',
      contradicts: 'bad',
      cites:       'muted',
    };
    return map[rel] ?? 'muted';
  }

  function onMemoryCreated(m: Memory) {
    showAddModal = false;
    memories = [m, ...memories];
    selectMemory(m.id);
    void loadAllRelations(memories);
  }
</script>

<div class="page page--wide mem-page">
  <!-- Header -->
  <div class="page-header mem-header">
    <h1>Memory</h1>
    {#if $activeWorkspace}
      <span class="chip chip--accent">{$activeWorkspace.slug}</span>
    {/if}
    <p class="muted">
      Versioned memories with typed relations.
    </p>

    <!-- Controls -->
    <div class="mem-controls">
      <label class="ctrl-label">
        Audience
        <select
          class="ctrl-select"
          bind:value={audience}
          onchange={() => load($activeWorkspaceId)}
        >
          <option value="">all</option>
          <option value="default">default</option>
          <option value="sales">sales</option>
          <option value="legal">legal</option>
          <option value="exec">exec</option>
          <option value="engineering">engineering</option>
        </select>
      </label>

      <label class="ctrl-label ctrl-label--check">
        <input type="checkbox" bind:checked={showForgotten} onchange={() => load($activeWorkspaceId)} />
        Show forgotten
      </label>

      <label class="ctrl-label ctrl-label--check">
        <input type="checkbox" bind:checked={showOldVersions} onchange={() => load($activeWorkspaceId)} />
        All versions
      </label>

      <button
        class="btn btn--sm"
        onclick={() => load($activeWorkspaceId)}
        disabled={loading}
      >
        {loading ? 'Loading…' : 'Refresh'}
      </button>

      <button
        class="btn btn--sm btn--primary"
        onclick={() => (showAddModal = true)}
      >
        + Add memory
      </button>
    </div>
  </div>

  {#if error}
    <pre class="error">{error}</pre>
  {/if}

  {#if loading && memories.length === 0}
    <!-- Loading state -->
    <div class="mem-loading card">
      <div class="mem-dots"><span></span><span></span><span></span></div>
      <p class="muted">Loading memories…</p>
    </div>
  {:else if !loading && memories.length === 0 && !error}
    <!-- Empty state -->
    <div class="mem-empty card">
      <div class="mem-empty__icon">◈</div>
      <h2>No memories yet</h2>
      <p>
        Memories are versioned, typed knowledge entries attached to a workspace.
        Click <strong>+ Add memory</strong> to create the first one, or ingest signals
        to have the engine populate them automatically.
      </p>
    </div>
  {:else}
    <!-- Two-column layout -->
    <div class="mem-layout">
      <!-- LEFT: memory list -->
      <aside class="mem-list">
        <header class="mem-list__head">
          <span class="eyebrow">Memories</span>
          <span class="chip">{memories.length}</span>
          <select class="ctrl-select mem-list__sort" bind:value={sortBy}>
            <option value="recency">Newest</option>
            <option value="audience">Audience</option>
            <option value="static">Static first</option>
          </select>
        </header>

        <ul class="mem-list__ul">
          {#each sortedMemories() as m (m.id)}
            <li>
              <button
                class="mem-item"
                class:mem-item--selected={selectedId === m.id}
                class:mem-item--forgotten={m.is_forgotten}
                onclick={() => selectMemory(m.id)}
              >
                <div class="mem-item__top">
                  {#if m.is_static}
                    <span class="mem-item__static" title="Static memory">◆</span>
                  {/if}
                  <span class="mem-item__snippet">{snip(m.content)}</span>
                </div>
                <div class="mem-item__meta">
                  <span class="chip chip--accent">v{m.version}</span>
                  <span class="chip">{m.audience}</span>
                  {#if m.is_forgotten}
                    <span class="chip chip--bad">forgotten</span>
                  {/if}
                  {#if !m.is_latest}
                    <span class="chip">old</span>
                  {/if}
                  <span class="mem-item__date muted">{fmtDate(m.created_at)}</span>
                </div>
              </button>
            </li>
          {/each}
        </ul>
      </aside>

      <!-- RIGHT: SVG graph -->
      <div class="mem-graph card">
        <MemoryGraph
          {memories}
          relations={allRelations}
          {selectedId}
          onSelect={selectMemory}
        />
      </div>
    </div>

    <!-- BOTTOM: detail drawer for selected memory -->
    {#if selectedMemory}
      <section class="mem-detail card">
        <header class="mem-detail__header">
          <div class="mem-detail__chips">
            <span class="chip chip--accent">v{selectedMemory.version}</span>
            <span class="chip">{selectedMemory.audience}</span>
            {#if selectedMemory.is_static}
              <span class="chip" style="color: var(--amber); border-color: var(--amber)">static</span>
            {/if}
            {#if selectedMemory.is_forgotten}
              <span class="chip chip--bad">forgotten</span>
            {/if}
            {#if !selectedMemory.is_latest}
              <span class="chip">old version</span>
            {/if}
          </div>
          <div class="mem-detail__actions">
            {#if !selectedMemory.is_forgotten}
              <button
                class="btn btn--sm btn--ghost"
                style="color: var(--bad);"
                onclick={() => handleForget(selectedMemory!.id)}
              >
                Forget
              </button>
            {/if}
            <button
              class="btn btn--sm btn--ghost"
              onclick={() => { selectedId = null; selectedRelations = null; selectedVersions = []; }}
              aria-label="Close detail"
            >
              ×
            </button>
          </div>
        </header>

        <div class="mem-detail__body">
          <!-- Full content -->
          <div class="mem-detail__content">{selectedMemory.content}</div>

          <div class="mem-detail__cols">
            <!-- Metadata -->
            <div class="mem-detail__section">
              <h3 class="eyebrow">Info</h3>
              <table class="mem-table">
                <tbody>
                  <tr>
                    <td class="mem-table__key">ID</td>
                    <td><code class="mono">{selectedMemory.id}</code></td>
                  </tr>
                  <tr>
                    <td class="mem-table__key">Created</td>
                    <td>{fmtDateLong(selectedMemory.created_at)}</td>
                  </tr>
                  <tr>
                    <td class="mem-table__key">Updated</td>
                    <td>{fmtDateLong(selectedMemory.updated_at)}</td>
                  </tr>
                  {#if selectedMemory.citation_uri}
                    <tr>
                      <td class="mem-table__key">Citation</td>
                      <td><a href={selectedMemory.citation_uri} target="_blank" rel="noreferrer">{selectedMemory.citation_uri}</a></td>
                    </tr>
                  {/if}
                  {#if selectedMemory.is_forgotten && selectedMemory.forget_reason}
                    <tr>
                      <td class="mem-table__key">Forget reason</td>
                      <td>{selectedMemory.forget_reason}</td>
                    </tr>
                  {/if}
                  {#if selectedMemory.is_forgotten && selectedMemory.forget_after}
                    <tr>
                      <td class="mem-table__key">Forget after</td>
                      <td>{fmtDate(selectedMemory.forget_after)}</td>
                    </tr>
                  {/if}
                  {#if selectedMemory.root_memory_id}
                    <tr>
                      <td class="mem-table__key">Root ID</td>
                      <td><code class="mono">{selectedMemory.root_memory_id}</code></td>
                    </tr>
                  {/if}
                </tbody>
              </table>
            </div>

            <!-- Version chain -->
            {#if detailLoading}
              <div class="mem-detail__section">
                <h3 class="eyebrow">Versions</h3>
                <p class="muted" style="font-size: 0.8rem;">Loading…</p>
              </div>
            {:else if selectedVersions.length > 0}
              <div class="mem-detail__section">
                <h3 class="eyebrow">Version chain ({selectedVersions.length})</h3>
                <div class="mem-versions">
                  {#each selectedVersions as v (v.id)}
                    <div
                      class="mem-ver"
                      class:mem-ver--current={v.id === selectedId}
                    >
                      <span class="chip chip--accent">v{v.version}</span>
                      <span class="mem-ver__date muted">{fmtDate(v.created_at)}</span>
                      {#if v.is_latest}<span class="chip chip--good">latest</span>{/if}
                      {#if v.is_forgotten}<span class="chip chip--bad">forgotten</span>{/if}
                      <button
                        class="mem-ver__btn btn btn--sm btn--ghost"
                        onclick={() => selectMemory(v.id)}
                        disabled={v.id === selectedId}
                      >view</button>
                    </div>
                  {/each}
                </div>
              </div>
            {/if}

            <!-- Relations -->
            {#if !detailLoading && selectedRelations}
              {#if selectedRelations.outbound.length > 0 || selectedRelations.inbound.length > 0}
                <div class="mem-detail__section">
                  <h3 class="eyebrow">Relations</h3>
                  {#if selectedRelations.outbound.length > 0}
                    <p class="mem-rel__heading">Outbound</p>
                    {#each selectedRelations.outbound as rel (rel.target_memory_id + rel.relation)}
                      {@const target = memories.find((m) => m.id === rel.target_memory_id)}
                      <div class="mem-rel">
                        <span class="mem-rel__badge mem-rel__badge--{relColor(rel.relation)}">{RELATION_LABEL[rel.relation]}</span>
                        <button
                          class="mem-rel__target btn btn--ghost btn--sm"
                          onclick={() => selectMemory(rel.target_memory_id)}
                          title={target?.content ?? rel.target_memory_id}
                        >
                          {target ? snip(target.content, 50) : rel.target_memory_id}
                        </button>
                      </div>
                    {/each}
                  {/if}
                  {#if selectedRelations.inbound.length > 0}
                    <p class="mem-rel__heading">Inbound</p>
                    {#each selectedRelations.inbound as rel (rel.source_memory_id + rel.relation)}
                      {@const source = memories.find((m) => m.id === rel.source_memory_id)}
                      <div class="mem-rel">
                        <span class="mem-rel__badge mem-rel__badge--{relColor(rel.relation)}">{RELATION_LABEL[rel.relation]}</span>
                        <button
                          class="mem-rel__target btn btn--ghost btn--sm"
                          onclick={() => selectMemory(rel.source_memory_id)}
                          title={source?.content ?? rel.source_memory_id}
                        >
                          {source ? snip(source.content, 50) : rel.source_memory_id}
                        </button>
                      </div>
                    {/each}
                  {/if}
                </div>
              {/if}
            {/if}
          </div>
        </div>
      </section>
    {/if}
  {/if}
</div>

<!-- Add memory modal -->
{#if showAddModal}
  <AddMemoryModal
    workspace={$activeWorkspaceId}
    onClose={() => (showAddModal = false)}
    onCreated={onMemoryCreated}
  />
{/if}

<style>
  /* ── Page layout ─────────────────────────────────────────────────── */
  .mem-page {
    display: flex;
    flex-direction: column;
    gap: 1rem;
  }

  .mem-header {
    flex-wrap: wrap;
    gap: 0.6rem;
  }

  .mem-controls {
    display: flex;
    align-items: center;
    gap: 0.6rem;
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

  .ctrl-label--check {
    cursor: pointer;
    gap: 0.3rem;
  }
  .ctrl-label--check input { cursor: pointer; }

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

  /* ── Loading / empty ─────────────────────────────────────────────── */
  .mem-loading {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 1rem;
    padding: 3rem;
  }
  .mem-dots {
    display: flex;
    gap: 0.5rem;
  }
  .mem-dots span {
    width: 8px;
    height: 8px;
    border-radius: 50%;
    background: var(--accent);
    animation: mem-bounce 1.2s infinite;
  }
  .mem-dots span:nth-child(2) { animation-delay: 0.2s; }
  .mem-dots span:nth-child(3) { animation-delay: 0.4s; }

  @keyframes mem-bounce {
    0%, 80%, 100% { transform: scale(0.7); opacity: 0.5; }
    40%            { transform: scale(1);   opacity: 1; }
  }

  .mem-empty {
    text-align: center;
    max-width: 560px;
    margin: 2rem auto;
    padding: 3rem 2rem;
  }
  .mem-empty__icon {
    font-size: 2rem;
    color: var(--accent);
    margin-bottom: 0.6rem;
  }
  .mem-empty h2 {
    margin: 0 0 0.5rem;
    font-size: 1.2rem;
    font-weight: 600;
  }
  .mem-empty p {
    color: var(--text-muted);
    font-size: 0.9rem;
    line-height: 1.55;
    margin: 0;
  }
  .mem-empty p strong { color: var(--text); }

  /* ── Two-column layout ───────────────────────────────────────────── */
  .mem-layout {
    display: grid;
    grid-template-columns: 320px 1fr;
    gap: 1rem;
    align-items: start;
    min-height: 460px;
  }

  /* ── Memory list (left) ──────────────────────────────────────────── */
  .mem-list {
    background: var(--bg-elevated);
    border: 1px solid var(--border);
    border-radius: var(--r-md);
    padding: 0.65rem;
    position: sticky;
    top: 72px;
    max-height: calc(100vh - 200px);
    overflow-y: auto;
  }

  .mem-list__head {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    padding: 0.2rem 0.4rem 0.6rem;
  }

  .mem-list__sort {
    margin-left: auto;
    font-size: 0.72rem;
    padding: 0.15rem 0.4rem;
  }

  .mem-list__ul {
    list-style: none;
    padding: 0;
    margin: 0;
    display: flex;
    flex-direction: column;
    gap: 0.2rem;
  }

  .mem-item {
    width: 100%;
    background: transparent;
    border: 1px solid transparent;
    border-radius: var(--r-sm);
    color: var(--text);
    text-align: left;
    padding: 0.5rem 0.65rem;
    cursor: pointer;
    font: inherit;
    transition: background 0.12s, border-color 0.12s;
  }
  .mem-item:hover { background: var(--bg-elevated-2); }
  .mem-item--selected {
    background: var(--accent-soft);
    border-color: var(--accent);
  }
  .mem-item--forgotten {
    opacity: 0.55;
  }

  .mem-item__top {
    display: flex;
    align-items: flex-start;
    gap: 0.3rem;
    margin-bottom: 0.3rem;
  }

  .mem-item__static {
    color: var(--amber);
    font-size: 0.7rem;
    flex-shrink: 0;
    margin-top: 1px;
  }

  .mem-item__snippet {
    font-size: 0.82rem;
    color: var(--text);
    line-height: 1.4;
    word-break: break-word;
  }

  .mem-item__meta {
    display: flex;
    gap: 0.25rem;
    flex-wrap: wrap;
    align-items: center;
  }

  .mem-item__date {
    font-size: 0.68rem;
    font-family: var(--font-mono);
    margin-left: auto;
  }

  /* ── Graph (right) ───────────────────────────────────────────────── */
  .mem-graph {
    padding: 0;
    overflow: hidden;
    height: 460px;
  }

  /* ── Detail drawer (bottom) ──────────────────────────────────────── */
  .mem-detail {
    padding: 0;
    overflow: hidden;
  }

  .mem-detail__header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 0.5rem;
    padding: 0.75rem 1.1rem;
    border-bottom: 1px solid var(--border-soft);
    background: var(--bg);
  }

  .mem-detail__chips {
    display: flex;
    gap: 0.35rem;
    flex-wrap: wrap;
  }

  .mem-detail__actions {
    display: flex;
    gap: 0.35rem;
    align-items: center;
    margin-left: auto;
  }

  .mem-detail__body {
    padding: 1rem 1.25rem 1.25rem;
    display: flex;
    flex-direction: column;
    gap: 1rem;
  }

  .mem-detail__content {
    font-size: 0.9rem;
    color: var(--text);
    line-height: 1.6;
    white-space: pre-wrap;
    padding: 0.75rem;
    background: var(--bg);
    border: 1px solid var(--border-soft);
    border-radius: var(--r-sm);
  }

  .mem-detail__cols {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
    gap: 1.25rem;
    align-items: start;
  }

  .mem-detail__section h3 {
    margin: 0 0 0.5rem;
  }

  /* ── Info table ──────────────────────────────────────────────────── */
  .mem-table {
    border-collapse: collapse;
    width: 100%;
    font-size: 0.8rem;
  }
  .mem-table td {
    padding: 0.25rem 0.4rem;
    color: var(--text);
    vertical-align: top;
    border-bottom: 1px solid var(--border-soft);
  }
  .mem-table tr:last-child td { border-bottom: none; }
  .mem-table__key {
    color: var(--text-muted) !important;
    white-space: nowrap;
    font-weight: 500;
    width: 40%;
  }

  .mono {
    font-family: var(--font-mono);
    font-size: 0.75em;
    color: var(--text-muted);
    word-break: break-all;
  }

  /* ── Version chain ───────────────────────────────────────────────── */
  .mem-versions {
    display: flex;
    flex-direction: column;
    gap: 0.3rem;
  }

  .mem-ver {
    display: flex;
    align-items: center;
    gap: 0.4rem;
    padding: 0.3rem 0.4rem;
    border-radius: var(--r-sm);
    background: var(--bg);
    border: 1px solid var(--border-soft);
    font-size: 0.78rem;
  }

  .mem-ver--current {
    background: var(--accent-soft);
    border-color: var(--accent);
  }

  .mem-ver__date {
    font-size: 0.7rem;
    font-family: var(--font-mono);
  }

  .mem-ver__btn {
    margin-left: auto;
  }

  /* ── Relations ───────────────────────────────────────────────────── */
  .mem-rel__heading {
    font-size: 0.68rem;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.07em;
    color: var(--text-subtle);
    margin: 0.4rem 0 0.2rem;
  }

  .mem-rel {
    display: flex;
    align-items: center;
    gap: 0.4rem;
    margin-bottom: 0.25rem;
  }

  .mem-rel__badge {
    display: inline-flex;
    padding: 1px 7px;
    border-radius: 999px;
    font-size: 0.65rem;
    font-weight: 600;
    text-transform: lowercase;
    white-space: nowrap;
    flex-shrink: 0;
  }
  .mem-rel__badge--accent     { background: var(--accent-soft);         color: var(--accent); }
  .mem-rel__badge--good       { background: var(--good-bg);             color: var(--good); }
  .mem-rel__badge--purple     { background: rgba(187,126,255,0.12);     color: var(--purple); }
  .mem-rel__badge--bad        { background: var(--bad-bg);              color: var(--bad); }
  .mem-rel__badge--muted      { background: var(--bg-elevated-2);       color: var(--text-muted); }

  .mem-rel__target {
    text-align: left;
    font-size: 0.78rem;
    color: var(--text-muted);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    max-width: 280px;
    cursor: pointer;
  }
  .mem-rel__target:hover { color: var(--accent); }

  @media (max-width: 880px) {
    .mem-layout {
      grid-template-columns: 1fr;
    }
    .mem-list {
      position: static;
      max-height: 300px;
    }
    .mem-graph {
      height: 340px;
    }
    .mem-controls {
      margin-left: 0;
    }
  }
</style>
