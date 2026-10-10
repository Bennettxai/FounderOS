<script lang="ts">
  import type { CustomAgent } from '$lib/api/ai/types';
  import { categoryLabels } from '$lib/stores/agents';

  interface Props {
    agent: CustomAgent;
    onSelect?: (agent: CustomAgent) => void;
    onEdit?: (agent: CustomAgent) => void;
    onDelete?: (agent: CustomAgent) => void;
    onChat?: (agent: CustomAgent) => void;
    variant?: 'default' | 'compact';
  }

  let { agent, onSelect, onEdit, onDelete, onChat, variant = 'default' }: Props = $props();

  let showMenu = $state(false);
  let confirmingDelete = $state(false);

  function getInitials(name: string | undefined): string {
    if (!name) return '??';
    const words = name.trim().split(/\s+/).filter(Boolean);
    if (words.length === 0) return '??';
    if (words.length === 1) return words[0].slice(0, 2).toUpperCase();
    return (words[0][0] + words[1][0]).toUpperCase();
  }

  function formatUsage(n: number): string {
    if (n >= 1000) return `${(n / 1000).toFixed(1)}k`;
    return String(n);
  }

  function formatModel(model: string): string {
    return model;
  }

  function formatCategory(category: string): string {
    const label = categoryLabels[category];
    if (label) return label;
    return category.charAt(0).toUpperCase() + category.slice(1);
  }

  function handleSelect() {
    onSelect?.(agent);
  }

  function handleEdit(e: MouseEvent) {
    e.stopPropagation();
    showMenu = false;
    onEdit?.(agent);
  }

  function handleDelete(e: MouseEvent) {
    e.stopPropagation();
    if (!confirmingDelete) {
      confirmingDelete = true;
      return;
    }
    showMenu = false;
    confirmingDelete = false;
    onDelete?.(agent);
  }

  function cancelDelete(e: MouseEvent) {
    e.stopPropagation();
    confirmingDelete = false;
  }

  function toggleMenu(e: MouseEvent) {
    e.stopPropagation();
    showMenu = !showMenu;
    confirmingDelete = false;
  }

  $effect(() => {
    if (!showMenu) return;
    function close() { showMenu = false; }
    function onKey(e: KeyboardEvent) { if (e.key === 'Escape') showMenu = false; }
    const frame = requestAnimationFrame(() => {
      document.addEventListener('click', close);
      document.addEventListener('keydown', onKey);
    });
    return () => {
      cancelAnimationFrame(frame);
      document.removeEventListener('click', close);
      document.removeEventListener('keydown', onKey);
    };
  });
</script>

<div
  class="ac"
  class:ac--featured={agent.is_featured}
  class:ac--inactive={!agent.is_active}
  class:ac--compact={variant === 'compact'}
  class:compact={variant === 'compact'}
>
  <!-- Featured accent bar -->
  {#if agent.is_featured}
    <div class="ac__featured-bar" aria-hidden="true"></div>
  {/if}

  <!-- Top row -->
  <div class="ac__top">
    <div class="ac__avatar" aria-hidden="true">
      {#if agent.avatar}
        <img class="ac__avatar-img" src={agent.avatar} alt={agent.display_name} />
      {:else}
        <span class="ac__initials">{getInitials(agent.display_name)}</span>
      {/if}
      {#if agent.is_active}
        <span class="ac__dot ac__dot--active" title="Active"></span>
        <span class="ac__sr-only">Active</span>
      {:else}
        <span class="ac__dot ac__dot--inactive" title="Inactive"></span>
        <span class="ac__sr-only">Inactive</span>
      {/if}
    </div>

    <div class="ac__info">
      <h3 class="ac__name">{agent.display_name}</h3>
      <span class="ac__handle">@{agent.name}</span>
    </div>

    {#if onEdit || onDelete}
      <div class="ac__menu-wrap">
        <button
          class="ac__menu-btn"
          onclick={toggleMenu}
          aria-label="More actions"
          aria-expanded={showMenu}
          aria-haspopup="menu"
        >
          <svg width="15" height="15" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
            <circle cx="12" cy="5" r="1.5"/>
            <circle cx="12" cy="12" r="1.5"/>
            <circle cx="12" cy="19" r="1.5"/>
          </svg>
        </button>
        {#if showMenu}
          <div class="ac__menu" role="menu" tabindex="-1">
            {#if onEdit}
              <button class="ac__menu-item" role="menuitem" onclick={handleEdit}>
                <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true">
                  <path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"/>
                  <path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"/>
                </svg>
                Edit
              </button>
            {/if}
            {#if onDelete}
              {#if confirmingDelete}
                <div class="ac__confirm" role="group" aria-label="Confirm delete">
                  <span>Are you sure? This cannot be undone.</span>
                  <div class="ac__confirm-actions">
                    <button class="ac__menu-item" role="menuitem" onclick={cancelDelete}>Cancel</button>
                    <button class="ac__menu-item ac__menu-item--danger" role="menuitem" onclick={handleDelete}>Delete</button>
                  </div>
                </div>
              {:else}
                <button class="ac__menu-item ac__menu-item--danger" role="menuitem" onclick={handleDelete}>
                  <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true">
                    <polyline points="3 6 5 6 21 6"/>
                    <path d="M19 6l-1 14a2 2 0 0 1-2 2H8a2 2 0 0 1-2-2L5 6"/>
                  </svg>
                  Delete
                </button>
              {/if}
            {/if}
          </div>
        {/if}
      </div>
    {/if}
  </div>

  <!-- Description -->
  <p class="ac__desc line-clamp-2" title={agent.description || 'No description provided.'}>{agent.description || 'No description provided.'}</p>

  <!-- Tools row -->
  {#if agent.tools_enabled && agent.tools_enabled.length > 0}
    <div class="ac__tools" aria-label="Enabled tools">
      {#each agent.tools_enabled.slice(0, 3) as tool}
        <span class="ac__tool">{tool.replace(/_/g, ' ')}</span>
      {/each}
      {#if agent.tools_enabled.length > 3}
        <span class="ac__tool ac__tool--more">+{agent.tools_enabled.length - 3}</span>
      {/if}
    </div>
  {/if}

  <!-- Quick-launch chat button -->
  {#if onChat}
    <button
      class="ac__chat-btn"
      onclick={(e) => { e.stopPropagation(); onChat?.(agent); }}
      aria-label="Chat with {agent.display_name}"
    >
      <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
        <path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"/>
      </svg>
      Chat
    </button>
  {/if}

  <!-- Footer -->
  <div class="ac__footer">
    <div class="ac__footer-left">
      {#if agent.category}
        <span class="ac__badge">{formatCategory(agent.category)}</span>
      {/if}
      {#if agent.model_preference}
        <span class="ac__model">{formatModel(agent.model_preference)}</span>
      {/if}
    </div>
    {#if agent.times_used && agent.times_used > 0}
      <span class="ac__usage" title="{agent.times_used} uses">{formatUsage(agent.times_used)} uses</span>
    {/if}
  </div>
  {#if onSelect}
    <button class="ac__select" onclick={(e) => { e.stopPropagation(); handleSelect(); }} aria-label="Select {agent.display_name}">
      Select
    </button>
  {/if}
</div>

<style>
  /* ═══ AgentCard — BOS Tokens ═══ */
  .ac {
    background: var(--dbg, #fff);
    border: 1px solid var(--dbd, #e0e0e0);
    border-radius: 12px;
    padding: 1.125rem;
    display: flex;
    flex-direction: column;
    gap: 0.75rem;
    transition: border-color 0.18s, box-shadow 0.18s, transform 0.18s;
    position: relative;
    overflow: hidden;
    outline: none;
  }
  .ac--inactive { opacity: 0.55; }
  .ac--inactive:hover { opacity: 0.85; }

  /* Featured accent */
  .ac__featured-bar {
    position: absolute;
    top: 0;
    left: 0;
    right: 0;
    height: 2px;
    background: var(--bos-accent-blue, #3b82f6);
  }

  /* Top row */
  .ac__top {
    display: flex;
    align-items: center;
    gap: 0.75rem;
  }

  .ac__avatar {
    width: 2.5rem;
    height: 2.5rem;
    border-radius: 10px;
    background: var(--dbg3, #eee);
    display: flex;
    align-items: center;
    justify-content: center;
    flex-shrink: 0;
    position: relative;
    overflow: visible;
  }
  .ac__avatar-img {
    width: 100%;
    height: 100%;
    object-fit: cover;
    border-radius: inherit;
  }
  .ac__initials {
    font-size: 0.8125rem;
    font-weight: 700;
    color: var(--dt2, #555);
    letter-spacing: 0.02em;
    user-select: none;
  }
  .ac__dot {
    position: absolute;
    bottom: -1px;
    right: -1px;
    width: 8px;
    height: 8px;
    border-radius: 50%;
    border: 2px solid var(--dbg, #fff);
  }
  .ac__dot--active { background: var(--bos-status-success, #22c55e); }
  .ac__dot--inactive { background: var(--dt4, #ccc); }
  .ac__sr-only {
    position: absolute;
    width: 1px;
    height: 1px;
    padding: 0;
    margin: -1px;
    overflow: hidden;
    clip: rect(0, 0, 0, 0);
    white-space: nowrap;
    border: 0;
  }

  .ac__info {
    flex: 1;
    min-width: 0;
  }
  .ac__name {
    font-size: 0.9375rem;
    font-weight: 600;
    color: var(--dt, #111);
    margin: 0;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  .ac__handle {
    font-size: 0.6875rem;
    color: var(--dt3, #888);
    font-family: var(--bos-font-code-family, monospace);
  }

  /* Menu */
  .ac__menu-wrap { position: relative; flex-shrink: 0; }
  .ac__menu-btn {
    width: 28px;
    height: 28px;
    border-radius: 6px;
    border: none;
    background: transparent;
    color: var(--dt3, #888);
    cursor: pointer;
    display: flex;
    align-items: center;
    justify-content: center;
    transition: background 0.12s, color 0.12s;
  }
  .ac__menu-btn:hover { background: var(--dbg2, #f5f5f5); color: var(--dt, #111); }
  .ac__menu {
    position: absolute;
    right: 0;
    top: calc(100% + 4px);
    background: var(--dbg, #fff);
    border: 1px solid var(--dbd, #e0e0e0);
    border-radius: 8px;
    box-shadow: 0 8px 24px rgba(0,0,0,0.12);
    z-index: 20;
    min-width: 128px;
    padding: 4px;
  }
  .ac__menu-item {
    display: flex;
    align-items: center;
    gap: 0.4375rem;
    width: 100%;
    text-align: left;
    padding: 0.375rem 0.625rem;
    font-size: 0.8125rem;
    color: var(--dt, #111);
    background: none;
    border: none;
    border-radius: 5px;
    cursor: pointer;
    transition: background 0.1s;
  }
  .ac__menu-item:hover { background: var(--dbg2, #f5f5f5); }
  .ac__menu-item--danger { color: var(--bos-status-error, #ef4444); }
  .ac__menu-item--danger:hover { background: rgba(239, 68, 68, 0.07); }
  .ac__confirm {
    display: flex;
    flex-direction: column;
    gap: 0.375rem;
    padding: 0.375rem;
    min-width: 190px;
    font-size: 0.75rem;
    color: var(--dt2, #555);
  }
  .ac__confirm-actions {
    display: flex;
    gap: 0.25rem;
  }
  .ac__confirm-actions .ac__menu-item {
    justify-content: center;
  }

  /* Description */
  .ac__desc {
    font-size: 0.8125rem;
    line-height: 1.55;
    color: var(--dt2, #555);
    margin: 0;
    display: -webkit-box;
    -webkit-line-clamp: 2;
    line-clamp: 2;
    -webkit-box-orient: vertical;
    overflow: hidden;
  }

  /* Tools */
  .ac__tools {
    display: flex;
    flex-wrap: wrap;
    gap: 0.3125rem;
  }
  .ac__tool {
    font-size: 0.6875rem;
    padding: 0.125rem 0.4375rem;
    border-radius: 4px;
    background: var(--dbg2, #f5f5f5);
    color: var(--dt3, #888);
    white-space: nowrap;
    text-transform: capitalize;
  }
  .ac__tool--more {
    background: transparent;
    color: var(--dt4, #bbb);
  }

  /* Footer */
  .ac__footer {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 0.5rem;
    padding-top: 0.625rem;
    border-top: 1px solid var(--dbd2, #f0f0f0);
    margin-top: auto;
  }
  .ac__footer-left {
    display: flex;
    align-items: center;
    gap: 0.375rem;
    min-width: 0;
    overflow: hidden;
  }
  .ac__badge {
    font-size: 0.6875rem;
    font-weight: 600;
    color: var(--dt2, #555);
    background: var(--dbg2, #f5f5f5);
    padding: 0.125rem 0.5rem;
    border-radius: 4px;
    text-transform: capitalize;
    white-space: nowrap;
  }
  .ac__model {
    font-size: 0.6875rem;
    color: var(--dt3, #888);
    font-family: var(--bos-font-code-family, monospace);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  .ac__usage {
    font-size: 0.6875rem;
    color: var(--dt3, #888);
    font-family: var(--bos-font-number-family, monospace);
    white-space: nowrap;
    flex-shrink: 0;
  }
  .ac__select {
    width: 100%;
    min-height: 2rem;
    border: 1px solid var(--dt, #111);
    border-radius: 7px;
    background: var(--dt, #111);
    color: var(--dbg, #fff);
    font-size: 0.8125rem;
    font-weight: 600;
    cursor: pointer;
    transition: opacity 0.15s;
  }
  .ac__select:hover { opacity: 0.9; }

  /* Chat button */
  .ac__chat-btn {
    width: 100%;
    padding: 0.4375rem;
    font-size: 0.75rem;
    font-weight: 600;
    border: 1px solid var(--dbd, #e0e0e0);
    border-radius: 7px;
    background: var(--dbg2, #f5f5f5);
    color: var(--dt2, #555);
    cursor: pointer;
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 0.375rem;
    transition: all 0.15s;
    opacity: 0;
    transform: translateY(4px);
    pointer-events: none;
  }
  .ac:hover .ac__chat-btn,
  .ac:focus-within .ac__chat-btn {
    opacity: 1;
    transform: translateY(0);
    pointer-events: auto;
  }
  .ac__chat-btn:hover {
    background: var(--dt, #111);
    color: var(--dbg, #fff);
    border-color: var(--dt, #111);
  }

  /* Compact variant */
  .ac--compact { padding: 0.875rem; gap: 0.5rem; }
  .ac--compact .ac__avatar { width: 2rem; height: 2rem; border-radius: 8px; }
  .ac--compact .ac__initials { font-size: 0.6875rem; }
  .ac--compact .ac__name { font-size: 0.8125rem; }
  .ac--compact .ac__desc { font-size: 0.75rem; -webkit-line-clamp: 1; line-clamp: 1; }
  .ac--compact .ac__tools { display: none; }
  .ac--compact .ac__chat-btn { display: none; }
</style>
