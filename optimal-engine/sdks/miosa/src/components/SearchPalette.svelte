<script lang="ts">
  import type { EngineContext } from '../engine.js';
  import type { GrepHit } from '@optimal-engine/client';

  interface SelectPayload {
    type: 'memory' | 'wiki' | 'signal';
    id?: string;
    slug?: string;
  }

  interface GrepResultGroup {
    memories: GrepHit[];
    wiki: Array<{ slug: string; title?: string; snippet?: string; score?: number }>;
    signals: Array<{ id: string; title?: string; snippet?: string; score?: number }>;
  }

  interface Props {
    engine: EngineContext;
    open?: boolean;
    onclose?: () => void;
    onselect?: (payload: SelectPayload) => void;
  }

  let { engine, open = false, onclose, onselect }: Props = $props();

  let query = $state('');
  let results = $state<GrepResultGroup>({ memories: [], wiki: [], signals: [] });
  let loading = $state(false);
  let activeIndex = $state(-1);
  let debounceTimer: ReturnType<typeof setTimeout> | null = null;

  // Flat list for keyboard navigation
  type FlatItem = SelectPayload & { title: string; snippet?: string; score?: number };

  const flatItems = $derived<FlatItem[]>([
    ...results.memories.map((m) => ({
      type: 'memory' as const,
      id: m.id,
      title: m.content.slice(0, 60),
      snippet: m.content.slice(0, 200),
      score: m.score,
    })),
    ...results.wiki.map((p) => ({
      type: 'wiki' as const,
      slug: p.slug,
      title: p.title ?? p.slug,
      snippet: p.snippet,
      score: p.score,
    })),
    ...results.signals.map((s) => ({
      type: 'signal' as const,
      id: s.id,
      title: s.title ?? s.id,
      snippet: s.snippet,
      score: s.score,
    })),
  ]);

  $effect(() => {
    if (!open) {
      query = '';
      results = { memories: [], wiki: [], signals: [] };
      activeIndex = -1;
    }
  });

  function onInput(e: Event) {
    query = (e.target as HTMLInputElement).value;
    activeIndex = -1;
    if (debounceTimer) clearTimeout(debounceTimer);
    debounceTimer = setTimeout(() => void search(), 200);
  }

  async function search() {
    if (!query.trim()) {
      results = { memories: [], wiki: [], signals: [] };
      return;
    }
    loading = true;
    try {
      const ws = engine.getWorkspace();
      const raw = await engine.client.grep(query, { workspace: ws, limit: 15 });
      const hits = raw.results ?? [];
      // Grep returns a flat list; bucket by metadata.type if present
      const memories: GrepHit[] = [];
      const wiki: GrepResultGroup['wiki'] = [];
      const signals: GrepResultGroup['signals'] = [];
      for (const h of hits) {
        const t = (h.metadata as Record<string, string> | undefined)?.type;
        if (t === 'wiki') {
          wiki.push({ slug: (h.metadata as Record<string, string>)?.slug ?? h.id, snippet: h.content.slice(0, 200), score: h.score });
        } else if (t === 'signal') {
          signals.push({ id: h.id, title: h.content.slice(0, 60), snippet: h.content.slice(0, 200), score: h.score });
        } else {
          memories.push(h);
        }
      }
      results = { memories, wiki, signals };
    } catch {
      // Silently suppress — palette search errors shouldn't crash the UI
    } finally {
      loading = false;
    }
  }

  function selectItem(item: FlatItem) {
    const payload: SelectPayload = { type: item.type, id: item.id, slug: item.slug };
    onselect?.(payload);
    onclose?.();
  }

  function onKeydown(e: KeyboardEvent) {
    if (!open) return;
    if (e.key === 'Escape') { onclose?.(); return; }
    if (e.key === 'ArrowDown') {
      e.preventDefault();
      activeIndex = Math.min(activeIndex + 1, flatItems.length - 1);
    } else if (e.key === 'ArrowUp') {
      e.preventDefault();
      activeIndex = Math.max(activeIndex - 1, 0);
    } else if (e.key === 'Enter' && activeIndex >= 0) {
      e.preventDefault();
      const item = flatItems[activeIndex];
      if (item) selectItem(item);
    }
  }

  function scoreLevel(score?: number): 'high' | 'mid' | 'low' {
    if (score === undefined) return 'low';
    return score >= 0.7 ? 'high' : score >= 0.4 ? 'mid' : 'low';
  }

  // Running flat index for keyboard navigation mapping
  let runningIdx = 0;
  function nextIdx(): number { return runningIdx++; }

  $effect(() => {
    if (open) runningIdx = 0;
  });
</script>

<svelte:window onkeydown={onKeydown} />

{#if open}
  <!-- svelte-ignore a11y_click_events_have_key_events -->
  <!-- svelte-ignore a11y_no_static_element_interactions -->
  <div class="oe-palette-overlay" onclick={onclose} role="dialog" aria-modal="true" aria-label="Search palette">
    <!-- svelte-ignore a11y_click_events_have_key_events -->
    <!-- svelte-ignore a11y_no_static_element_interactions -->
    <div class="oe-palette" onclick={(e) => e.stopPropagation()} role="presentation">
      <!-- Input row -->
      <div class="oe-palette__input-row">
        <svg width="16" height="16" viewBox="0 0 16 16" fill="none" aria-hidden="true" style="color: var(--dt4, #555); flex-shrink: 0;">
          <circle cx="7" cy="7" r="5" stroke="currentColor" stroke-width="1.5"/>
          <path d="M11 11l3 3" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/>
        </svg>
        <input
          class="oe-palette__input"
          type="text"
          placeholder="Search memories, wiki, signals…"
          value={query}
          oninput={onInput}
          autofocus
          aria-label="Search"
          aria-autocomplete="list"
          role="combobox"
          aria-expanded={flatItems.length > 0}
        />
        {#if loading}
          <div class="oe-spinner sp-spinner"></div>
        {/if}
      </div>

      <!-- Results -->
      <div class="oe-palette__results" role="listbox">
        {#if !query.trim() && flatItems.length === 0}
          <div class="sp-hint">Type to search across memories, wiki pages, and signals.</div>
        {:else if query.trim() && flatItems.length === 0 && !loading}
          <div class="sp-hint">No results for "{query}"</div>
        {:else}
          {@const _ = (runningIdx = 0)}

          <!-- Memories section -->
          {#if results.memories.length > 0}
            <div class="oe-palette__section-label">Memories</div>
            {#each results.memories as hit (hit.id)}
              {@const idx = nextIdx()}
              {@const isActive = activeIndex === idx}
              {@const slvl = scoreLevel(hit.score)}
              <!-- svelte-ignore a11y_click_events_have_key_events -->
              <!-- svelte-ignore a11y_no_static_element_interactions -->
              <div
                class="oe-palette__result {isActive ? 'oe-palette__result--active' : ''}"
                role="option"
                aria-selected={isActive}
                onclick={() => selectItem({ type: 'memory', id: hit.id, title: hit.content.slice(0, 60), snippet: hit.content.slice(0, 200), score: hit.score })}
              >
                <div class="sp-result-body">
                  <div class="oe-palette__result-title">{hit.content.slice(0, 60)}</div>
                  <div class="oe-palette__result-snippet">{hit.content.slice(0, 200)}</div>
                </div>
                <div class="sp-result-right">
                  <span class="oe-chip sp-type-chip sp-type-memory">memory</span>
                  {#if hit.score !== undefined}
                    <div class="oe-score-bar sp-score">
                      <div class="oe-score-bar__track">
                        <div class="oe-score-bar__fill oe-score-bar__fill--{slvl}" style="width: {Math.round((hit.score ?? 0) * 100)}%"></div>
                      </div>
                    </div>
                  {/if}
                </div>
              </div>
            {/each}
          {/if}

          <!-- Wiki section -->
          {#if results.wiki.length > 0}
            <div class="oe-palette__section-label">Wiki Pages</div>
            {#each results.wiki as page (page.slug)}
              {@const idx = nextIdx()}
              {@const isActive = activeIndex === idx}
              {@const slvl = scoreLevel(page.score)}
              <!-- svelte-ignore a11y_click_events_have_key_events -->
              <!-- svelte-ignore a11y_no_static_element_interactions -->
              <div
                class="oe-palette__result {isActive ? 'oe-palette__result--active' : ''}"
                role="option"
                aria-selected={isActive}
                onclick={() => selectItem({ type: 'wiki', slug: page.slug, title: page.title ?? page.slug, snippet: page.snippet, score: page.score })}
              >
                <div class="sp-result-body">
                  <div class="oe-palette__result-title">{page.title ?? page.slug}</div>
                  {#if page.snippet}
                    <div class="oe-palette__result-snippet">{page.snippet}</div>
                  {/if}
                </div>
                <div class="sp-result-right">
                  <span class="oe-chip sp-type-chip sp-type-wiki">wiki</span>
                  {#if page.score !== undefined}
                    <div class="oe-score-bar sp-score">
                      <div class="oe-score-bar__track">
                        <div class="oe-score-bar__fill oe-score-bar__fill--{slvl}" style="width: {Math.round((page.score ?? 0) * 100)}%"></div>
                      </div>
                    </div>
                  {/if}
                </div>
              </div>
            {/each}
          {/if}

          <!-- Signals section -->
          {#if results.signals.length > 0}
            <div class="oe-palette__section-label">Signals</div>
            {#each results.signals as sig (sig.id)}
              {@const idx = nextIdx()}
              {@const isActive = activeIndex === idx}
              {@const slvl = scoreLevel(sig.score)}
              <!-- svelte-ignore a11y_click_events_have_key_events -->
              <!-- svelte-ignore a11y_no_static_element_interactions -->
              <div
                class="oe-palette__result {isActive ? 'oe-palette__result--active' : ''}"
                role="option"
                aria-selected={isActive}
                onclick={() => selectItem({ type: 'signal', id: sig.id, title: sig.title ?? sig.id, snippet: sig.snippet, score: sig.score })}
              >
                <div class="sp-result-body">
                  <div class="oe-palette__result-title">{sig.title ?? sig.id}</div>
                  {#if sig.snippet}
                    <div class="oe-palette__result-snippet">{sig.snippet}</div>
                  {/if}
                </div>
                <div class="sp-result-right">
                  <span class="oe-chip sp-type-chip sp-type-signal">signal</span>
                  {#if sig.score !== undefined}
                    <div class="oe-score-bar sp-score">
                      <div class="oe-score-bar__track">
                        <div class="oe-score-bar__fill oe-score-bar__fill--{slvl}" style="width: {Math.round((sig.score ?? 0) * 100)}%"></div>
                      </div>
                    </div>
                  {/if}
                </div>
              </div>
            {/each}
          {/if}
        {/if}
      </div>

      <!-- Footer hints -->
      <div class="oe-palette__footer">
        <span><kbd class="oe-palette__kbd">↑↓</kbd> navigate</span>
        <span><kbd class="oe-palette__kbd">↵</kbd> select</span>
        <span><kbd class="oe-palette__kbd">esc</kbd> close</span>
      </div>
    </div>
  </div>
{/if}

<style>
  .sp-spinner {
    width: 16px;
    height: 16px;
    flex-shrink: 0;
  }

  .sp-hint {
    padding: 1rem;
    text-align: center;
    font-size: 0.84rem;
    color: var(--dt4, #555);
  }

  .sp-result-body {
    flex: 1;
    min-width: 0;
    display: flex;
    flex-direction: column;
    gap: 0.15rem;
  }

  .sp-result-right {
    display: flex;
    flex-direction: column;
    align-items: flex-end;
    gap: 0.3rem;
    flex-shrink: 0;
  }

  .sp-score {
    width: 60px;
  }

  .sp-type-chip {
    font-size: 0.6rem;
    padding: 1px 5px;
  }

  .sp-type-memory {
    background: rgba(126, 168, 255, 0.1);
    color: var(--daccent, #7ea8ff);
    border: 1px solid rgba(126, 168, 255, 0.2);
  }

  .sp-type-wiki {
    background: rgba(123, 227, 163, 0.1);
    color: #7be3a3;
    border: 1px solid rgba(123, 227, 163, 0.2);
  }

  .sp-type-signal {
    background: rgba(187, 126, 255, 0.1);
    color: #bb7eff;
    border: 1px solid rgba(187, 126, 255, 0.2);
  }
</style>
