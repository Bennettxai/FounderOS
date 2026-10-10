<script lang="ts">
  import type { EngineContext } from '../engine.js';
  import type { Memory } from '@optimal-engine/client';
  import { useMemories } from '../hooks/useMemories.svelte.js';

  interface Props {
    engine: EngineContext;
    audience?: string;
    limit?: number;
  }

  let { engine, audience, limit = 50 }: Props = $props();

  const mem = useMemories(engine, { audience, limit });

  let expandedId = $state<string | null>(null);
  let menuId = $state<string | null>(null);
  let forgetId = $state<string | null>(null);
  let forgetReason = $state('');
  let addFocused = $state(false);
  let addContent = $state('');
  let addLoading = $state(false);
  let showForgotten = $state(false);

  const REL_TYPES = ['updates', 'extends', 'derives', 'contradicts', 'cites'] as const;

  function relColor(type: string): string {
    if (type === 'updates') return 'oe-rel--updates';
    if (type === 'extends') return 'oe-rel--extends';
    if (type === 'derives') return 'oe-rel--derives';
    if (type === 'contradicts') return 'oe-rel--contradicts';
    if (type === 'cites') return 'oe-rel--cites';
    return '';
  }

  function preview(content: string): string {
    return content.length > 100 ? content.slice(0, 100) + '…' : content;
  }

  function fmtDate(iso?: string): string {
    if (!iso) return '';
    return new Date(iso).toLocaleDateString(undefined, { month: 'short', day: 'numeric', year: 'numeric' });
  }

  function relativeDate(iso?: string): string {
    if (!iso) return '';
    const diff = Date.now() - new Date(iso).getTime();
    const mins = Math.floor(diff / 60000);
    if (mins < 1) return 'just now';
    if (mins < 60) return `${mins}m ago`;
    const hrs = Math.floor(mins / 60);
    if (hrs < 24) return `${hrs}h ago`;
    return `${Math.floor(hrs / 24)}d ago`;
  }

  function toggleExpand(id: string) {
    expandedId = expandedId === id ? null : id;
    menuId = null;
  }

  function toggleMenu(e: MouseEvent, id: string) {
    e.stopPropagation();
    menuId = menuId === id ? null : id;
  }

  async function submitForget(id: string) {
    await mem.forget(id, forgetReason.trim() || undefined);
    forgetId = null;
    forgetReason = '';
    menuId = null;
  }

  async function submitAdd() {
    if (!addContent.trim()) return;
    addLoading = true;
    try {
      await mem.create(addContent.trim(), { audience });
      addContent = '';
      addFocused = false;
    } finally {
      addLoading = false;
    }
  }

  function copyContent(content: string) {
    navigator.clipboard.writeText(content).catch(() => {});
    menuId = null;
  }

  function onGlobalClick() {
    menuId = null;
  }

  const visible = $derived(
    mem.memories.filter((m: Memory) => showForgotten ? true : !m.forgotten_at)
  );
</script>

<!-- svelte-ignore a11y_click_events_have_key_events -->
<!-- svelte-ignore a11y_no_static_element_interactions -->
<div class="mb-root" onclick={onGlobalClick} role="presentation">
  <!-- Search bar -->
  <div class="oe-search mb-search">
    <svg width="14" height="14" viewBox="0 0 14 14" fill="none" aria-hidden="true">
      <circle cx="6" cy="6" r="4.5" stroke="currentColor" stroke-width="1.4"/>
      <path d="M9.5 9.5l2.5 2.5" stroke="currentColor" stroke-width="1.4" stroke-linecap="round"/>
    </svg>
    <input
      class="oe-search__input"
      type="search"
      placeholder="Search memories…"
      value={mem.query}
      oninput={(e) => { mem.query = (e.target as HTMLInputElement).value; }}
      aria-label="Search memories"
    />
    {#if mem.loading}
      <div class="oe-spinner mb-spinner"></div>
    {/if}
  </div>

  <!-- Forgotten toggle -->
  {#if mem.memories.some((m: Memory) => m.forgotten_at)}
    <div class="mb-toggle-row">
      <button class="mb-toggle-btn" onclick={() => showForgotten = !showForgotten}>
        {showForgotten ? 'Hide forgotten' : 'Show forgotten'}
      </button>
    </div>
  {/if}

  <!-- Memory list -->
  <div class="mb-list" role="list">
    {#if mem.loading && mem.memories.length === 0}
      <!-- Skeleton -->
      {#each [1, 2, 3] as n (n)}
        <div class="mb-skeleton">
          <div class="mb-skeleton__bullet"></div>
          <div class="mb-skeleton__lines">
            <div class="mb-skeleton__line mb-skeleton__line--wide"></div>
            <div class="mb-skeleton__line mb-skeleton__line--narrow"></div>
          </div>
        </div>
      {/each}
    {:else if visible.length === 0}
      <div class="oe-empty">
        <span class="oe-empty__icon">◦</span>
        <span>No memories in this workspace yet.</span>
      </div>
    {:else}
      {#each visible as memory (memory.id)}
        {@const isExpanded = expandedId === memory.id}
        {@const isForgetting = forgetId === memory.id}
        {@const isForgotten = !!memory.forgotten_at}
        {@const relations = (memory.relations as Array<{type: string; target_id: string}> | undefined) ?? []}

        <div
          class="oe-memory-row {isExpanded ? 'oe-memory-row--expanded' : ''} {isForgotten ? 'mb-forgotten' : ''}"
          role="listitem"
          onclick={() => toggleExpand(memory.id)}
          tabindex="0"
          onkeydown={(e) => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); toggleExpand(memory.id); }}}
          aria-expanded={isExpanded}
        >
          <div class="oe-memory-bullet" aria-hidden="true">
            {#if memory.is_static}
              <div class="mb-static-dot" title="Static memory"></div>
            {/if}
          </div>

          <div class="oe-memory-content">
            <!-- Collapsed preview -->
            <div class="mb-preview {isForgotten ? 'mb-forgotten__text' : ''}">
              {preview(memory.content)}
            </div>

            <!-- Chips row -->
            <div class="oe-memory-meta">
              {#if memory.is_static}
                <span class="oe-chip oe-chip--static" title="Static">STATIC</span>
              {/if}
              {#if memory.audience}
                <span class="oe-chip oe-chip--audience">{memory.audience}</span>
              {/if}
              {#if memory.version}
                <span class="oe-chip oe-chip--version">v{memory.version as string}</span>
              {/if}
            </div>

            <!-- Expanded content -->
            {#if isExpanded}
              <div class="oe-memory-expand" onclick={(e) => e.stopPropagation()} role="presentation">
                <div class="oe-memory-expand__body">{memory.content}</div>

                <div class="oe-memory-expand__footer">
                  {#if memory.citation_uri}
                    <a
                      href={memory.citation_uri}
                      target="_blank"
                      rel="noopener noreferrer"
                      class="oe-cite"
                      onclick={(e) => e.stopPropagation()}
                    >
                      <svg width="10" height="10" viewBox="0 0 10 10" fill="none" aria-hidden="true">
                        <path d="M4 2H2v6h6V6M6 1h3v3M9 1L5 5" stroke="currentColor" stroke-width="1.3" stroke-linecap="round"/>
                      </svg>
                      {memory.citation_uri}
                    </a>
                  {/if}

                  {#if memory.inserted_at}
                    <span class="mb-ts">Created {fmtDate(memory.inserted_at)} · {relativeDate(memory.inserted_at)}</span>
                  {/if}
                  {#if memory.updated_at}
                    <span class="mb-ts">Updated {relativeDate(memory.updated_at)}</span>
                  {/if}

                  <!-- Relation badges -->
                  {#each relations as rel}
                    <span class="oe-rel {relColor(rel.type)}" title="{rel.type} → {rel.target_id}">
                      {rel.type}
                    </span>
                  {/each}
                </div>

                <!-- Forget inline form -->
                {#if isForgetting}
                  <div class="mb-forget-form" onclick={(e) => e.stopPropagation()} role="presentation">
                    <input
                      class="mb-forget-input"
                      bind:value={forgetReason}
                      placeholder="Reason (optional)"
                      aria-label="Forget reason"
                      onkeydown={(e) => { if (e.key === 'Enter') submitForget(memory.id); if (e.key === 'Escape') { forgetId = null; forgetReason = ''; } }}
                    />
                    <button class="mb-forget-btn mb-forget-btn--confirm" onclick={() => submitForget(memory.id)}>
                      Confirm forget
                    </button>
                    <button class="mb-forget-btn" onclick={() => { forgetId = null; forgetReason = ''; }}>
                      Cancel
                    </button>
                  </div>
                {/if}
              </div>
            {/if}
          </div>

          <!-- 3-dot menu -->
          <div class="mb-menu-wrap" onclick={(e) => e.stopPropagation()} role="presentation">
            <button
              class="mb-menu-btn"
              onclick={(e) => toggleMenu(e, memory.id)}
              aria-label="Memory options"
              aria-haspopup="true"
            >
              <svg width="14" height="14" viewBox="0 0 14 14" fill="none" aria-hidden="true">
                <circle cx="7" cy="3" r="1.2" fill="currentColor"/>
                <circle cx="7" cy="7" r="1.2" fill="currentColor"/>
                <circle cx="7" cy="11" r="1.2" fill="currentColor"/>
              </svg>
            </button>

            {#if menuId === memory.id}
              <div class="mb-menu" role="menu">
                <button class="mb-menu__item mb-menu__item--danger" role="menuitem"
                  onclick={() => { forgetId = memory.id; expandedId = memory.id; menuId = null; }}>
                  Forget
                </button>
                <button class="mb-menu__item" role="menuitem"
                  onclick={() => copyContent(memory.content)}>
                  Copy
                </button>
                <button class="mb-menu__item" role="menuitem"
                  onclick={() => { expandedId = memory.id; menuId = null; }}>
                  View versions
                </button>
              </div>
            {/if}
          </div>
        </div>
      {/each}
    {/if}
  </div>

  <!-- Add memory row -->
  <div class="oe-add-row mb-add" onclick={() => { addFocused = true; }} role="presentation">
    <span class="mb-add-icon" aria-hidden="true">+</span>
    {#if addFocused}
      <div class="mb-add-expanded" onclick={(e) => e.stopPropagation()} role="presentation">
        <textarea
          class="mb-add-textarea"
          bind:value={addContent}
          placeholder="Add a memory…"
          rows={3}
          aria-label="New memory content"
          onkeydown={(e) => {
            if (e.key === 'Escape') { addFocused = false; addContent = ''; }
            if (e.key === 'Enter' && (e.metaKey || e.ctrlKey)) submitAdd();
          }}
        ></textarea>
        <div class="mb-add-actions">
          <button class="mb-add-submit" disabled={!addContent.trim() || addLoading} onclick={submitAdd}>
            {addLoading ? 'Saving…' : 'Save'}
          </button>
          <button class="mb-add-cancel" onclick={() => { addFocused = false; addContent = ''; }}>
            Cancel
          </button>
          <span class="mb-add-hint">⌘↵ to save</span>
        </div>
      </div>
    {:else}
      <input
        class="oe-add-input"
        type="text"
        placeholder="Add a memory…"
        onfocus={() => addFocused = true}
        aria-label="Add a memory"
        readonly
      />
    {/if}
  </div>

  {#if mem.error}
    <div class="mb-error">{mem.error}</div>
  {/if}
</div>

<style>
  .mb-root {
    display: flex;
    flex-direction: column;
    gap: 0.25rem;
  }

  .mb-search {
    margin-bottom: 0.5rem;
  }

  .mb-spinner {
    width: 14px;
    height: 14px;
    flex-shrink: 0;
  }

  .mb-toggle-row {
    margin-bottom: 0.25rem;
    padding: 0 0.6rem;
  }

  .mb-toggle-btn {
    background: none;
    border: none;
    font-size: 0.72rem;
    color: var(--dt4, #555);
    cursor: pointer;
    padding: 0;
    text-decoration: underline;
  }

  .mb-toggle-btn:hover {
    color: var(--dt3, #888);
  }

  .mb-list {
    display: flex;
    flex-direction: column;
    gap: 0.1rem;
  }

  /* Static dot overrides bullet */
  .oe-memory-bullet {
    position: relative;
  }

  .mb-static-dot {
    width: 6px;
    height: 6px;
    border-radius: 50%;
    background: #eab308;
  }

  .mb-forgotten {
    opacity: 0.45;
  }

  .mb-forgotten__text {
    text-decoration: line-through;
  }

  .mb-preview {
    font-size: 0.88rem;
    color: var(--dt, #f1f1f3);
    word-break: break-word;
  }

  .mb-ts {
    font-size: 0.68rem;
    color: var(--dt4, #555);
  }

  /* 3-dot menu */
  .mb-menu-wrap {
    position: relative;
    flex-shrink: 0;
  }

  .mb-menu-btn {
    background: none;
    border: none;
    padding: 3px 5px;
    color: var(--dt4, #555);
    cursor: pointer;
    border-radius: 4px;
    opacity: 0;
    transition: opacity 0.1s, background 0.1s;
  }

  .oe-memory-row:hover .mb-menu-btn {
    opacity: 1;
  }

  .mb-menu-btn:hover {
    background: var(--dbg2, #131820);
    color: var(--dt2, #ccc);
  }

  .mb-menu {
    position: absolute;
    right: 0;
    top: calc(100% + 2px);
    z-index: 50;
    background: var(--dbg, #0d1117);
    border: 1px solid var(--dbd, rgba(255,255,255,0.12));
    border-radius: 7px;
    min-width: 140px;
    box-shadow: 0 8px 24px -4px rgba(0,0,0,0.6);
    overflow: hidden;
  }

  .mb-menu__item {
    display: block;
    width: 100%;
    text-align: left;
    background: none;
    border: none;
    padding: 0.4rem 0.75rem;
    font-size: 0.84rem;
    color: var(--dt2, #ccc);
    cursor: pointer;
    transition: background 0.1s;
  }

  .mb-menu__item:hover {
    background: var(--dbg2, #131820);
  }

  .mb-menu__item--danger:hover {
    color: #f88;
  }

  /* Forget inline form */
  .mb-forget-form {
    display: flex;
    gap: 0.4rem;
    align-items: center;
    flex-wrap: wrap;
    margin-top: 0.5rem;
    padding: 0.4rem 0.5rem;
    background: rgba(248,136,136,0.06);
    border-radius: 5px;
    border: 1px solid rgba(248,136,136,0.15);
  }

  .mb-forget-input {
    flex: 1;
    min-width: 120px;
    background: none;
    border: 1px solid var(--dbd, rgba(255,255,255,0.1));
    border-radius: 4px;
    padding: 0.2rem 0.4rem;
    font-size: 0.82rem;
    color: var(--dt, #f1f1f3);
    font-family: inherit;
    outline: none;
  }

  .mb-forget-input:focus {
    border-color: #f88;
  }

  .mb-forget-btn {
    padding: 0.2rem 0.6rem;
    border-radius: 4px;
    border: 1px solid var(--dbd, rgba(255,255,255,0.1));
    background: none;
    font-size: 0.78rem;
    color: var(--dt3, #888);
    cursor: pointer;
    transition: background 0.1s;
  }

  .mb-forget-btn:hover {
    background: var(--dbg2, #131820);
  }

  .mb-forget-btn--confirm {
    border-color: rgba(248,136,136,0.3);
    color: #f88;
  }

  .mb-forget-btn--confirm:hover {
    background: rgba(248,136,136,0.1);
  }

  /* Add row expansion */
  .mb-add {
    margin-top: 0.25rem;
    cursor: text;
  }

  .mb-add-icon {
    font-size: 1rem;
    line-height: 1;
    color: var(--dt4, #555);
  }

  .mb-add-expanded {
    flex: 1;
    display: flex;
    flex-direction: column;
    gap: 0.4rem;
  }

  .mb-add-textarea {
    width: 100%;
    background: var(--dbg2, #131820);
    border: 1px solid var(--dbd, rgba(255,255,255,0.1));
    border-radius: 5px;
    padding: 0.4rem 0.5rem;
    font-size: 0.88rem;
    color: var(--dt, #f1f1f3);
    font-family: inherit;
    resize: vertical;
    outline: none;
    line-height: 1.55;
  }

  .mb-add-textarea:focus {
    border-color: var(--daccent, #7ea8ff);
  }

  .mb-add-actions {
    display: flex;
    gap: 0.4rem;
    align-items: center;
  }

  .mb-add-submit {
    padding: 0.25rem 0.7rem;
    border-radius: 5px;
    border: none;
    background: var(--daccent, #7ea8ff);
    color: #0d1117;
    font-size: 0.8rem;
    font-weight: 600;
    cursor: pointer;
    transition: opacity 0.1s;
  }

  .mb-add-submit:disabled {
    opacity: 0.4;
    cursor: not-allowed;
  }

  .mb-add-cancel {
    padding: 0.25rem 0.6rem;
    border-radius: 5px;
    border: 1px solid var(--dbd, rgba(255,255,255,0.1));
    background: none;
    color: var(--dt3, #888);
    font-size: 0.8rem;
    cursor: pointer;
    transition: background 0.1s;
  }

  .mb-add-cancel:hover {
    background: var(--dbg2, #131820);
  }

  .mb-add-hint {
    font-size: 0.68rem;
    color: var(--dt4, #555);
    margin-left: auto;
  }

  /* Skeleton */
  .mb-skeleton {
    display: flex;
    gap: 0.6rem;
    padding: 0.5rem 0.6rem;
    align-items: flex-start;
  }

  .mb-skeleton__bullet {
    width: 6px;
    height: 6px;
    border-radius: 50%;
    background: var(--dbg2, #131820);
    margin-top: 0.55rem;
    flex-shrink: 0;
  }

  .mb-skeleton__lines {
    flex: 1;
    display: flex;
    flex-direction: column;
    gap: 0.35rem;
  }

  .mb-skeleton__line {
    height: 10px;
    border-radius: 4px;
    background: var(--dbg2, #131820);
    animation: mb-pulse 1.4s ease-in-out infinite;
  }

  .mb-skeleton__line--wide { width: 80%; }
  .mb-skeleton__line--narrow { width: 45%; }

  @keyframes mb-pulse {
    0%, 100% { opacity: 0.4; }
    50% { opacity: 0.8; }
  }

  .mb-error {
    margin-top: 0.5rem;
    padding: 0.4rem 0.65rem;
    border-radius: 5px;
    background: rgba(248,136,136,0.08);
    border: 1px solid rgba(248,136,136,0.2);
    font-size: 0.82rem;
    color: #f88;
  }
</style>
