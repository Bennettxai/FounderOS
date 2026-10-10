<script lang="ts">
  import type { EngineContext } from '../engine.js';
  import { useWorkspace } from '../hooks/useWorkspace.svelte.js';

  interface Props {
    engine: EngineContext;
    onswitch?: (id: string) => void;
  }

  let { engine, onswitch }: Props = $props();

  const ws = useWorkspace(engine);

  let open = $state(false);
  let creatingNew = $state(false);
  let newSlug = $state('');
  let newName = $state('');

  function select(slug: string) {
    ws.switchTo(slug);
    onswitch?.(slug);
    open = false;
  }

  async function submitCreate() {
    if (!newSlug.trim() || !newName.trim()) return;
    const created = await ws.createWorkspace(newSlug.trim(), newName.trim());
    if (created) {
      select(created.slug);
      newSlug = '';
      newName = '';
      creatingNew = false;
    }
  }

  function onKeydown(e: KeyboardEvent) {
    if (e.key === 'Escape') { open = false; creatingNew = false; }
  }
</script>

<svelte:window onkeydown={onKeydown} />

<div class="wss-root">
  <button
    class="oe-ws-trigger"
    onclick={() => { open = !open; }}
    aria-haspopup="listbox"
    aria-expanded={open}
    aria-label="Switch workspace, current: {ws.current}"
  >
    <svg width="12" height="12" viewBox="0 0 12 12" fill="none" aria-hidden="true">
      <rect x="1" y="1" width="4" height="4" rx="1" stroke="currentColor" stroke-width="1.3"/>
      <rect x="7" y="1" width="4" height="4" rx="1" stroke="currentColor" stroke-width="1.3"/>
      <rect x="1" y="7" width="4" height="4" rx="1" stroke="currentColor" stroke-width="1.3"/>
      <rect x="7" y="7" width="4" height="4" rx="1" stroke="currentColor" stroke-width="1.3"/>
    </svg>
    <span>{ws.current}</span>
    <svg width="10" height="10" viewBox="0 0 10 10" fill="none" aria-hidden="true">
      <path d="M2 4l3 3 3-3" stroke="currentColor" stroke-width="1.3" stroke-linecap="round" stroke-linejoin="round"/>
    </svg>
  </button>

  {#if open}
    <!-- svelte-ignore a11y_click_events_have_key_events -->
    <!-- svelte-ignore a11y_no_static_element_interactions -->
    <div class="wss-backdrop" onclick={() => { open = false; creatingNew = false; }} aria-hidden="true"></div>

    <div class="oe-ws-dropdown" role="listbox" aria-label="Workspaces">
      {#if ws.loading}
        <div class="wss-loading"><div class="oe-spinner"></div></div>
      {:else}
        {#each ws.workspaces as workspace (workspace.id)}
          <div
            class="oe-ws-item {workspace.slug === ws.current ? 'oe-ws-item--active' : ''}"
            role="option"
            aria-selected={workspace.slug === ws.current}
            tabindex="0"
            onclick={() => select(workspace.slug)}
            onkeydown={(e) => { if (e.key === 'Enter') select(workspace.slug); }}
          >
            {workspace.name ?? workspace.slug}
            {#if workspace.description}
              <span class="wss-desc">{workspace.description}</span>
            {/if}
          </div>
        {/each}

        {#if ws.workspaces.length === 0}
          <div class="wss-empty">No workspaces yet</div>
        {/if}

        <div class="oe-ws-separator"></div>

        {#if creatingNew}
          <div class="wss-create-form">
            <input
              class="wss-input"
              bind:value={newSlug}
              placeholder="slug (e.g. crm)"
              aria-label="New workspace slug"
            />
            <input
              class="wss-input"
              bind:value={newName}
              placeholder="Display name"
              aria-label="New workspace name"
            />
            <div class="wss-create-actions">
              <button class="wss-btn wss-btn--create" onclick={submitCreate} disabled={!newSlug || !newName}>
                Create
              </button>
              <button class="wss-btn" onclick={() => creatingNew = false}>Cancel</button>
            </div>
          </div>
        {:else}
          <div
            class="oe-ws-item wss-new"
            role="button"
            tabindex="0"
            onclick={() => creatingNew = true}
            onkeydown={(e) => { if (e.key === 'Enter') creatingNew = true; }}
          >
            <span class="wss-plus">+</span> New workspace
          </div>
        {/if}
      {/if}
    </div>
  {/if}
</div>

<style>
  .wss-root {
    position: relative;
    display: inline-block;
  }

  .wss-backdrop {
    position: fixed;
    inset: 0;
    z-index: 99;
  }

  .oe-ws-dropdown {
    z-index: 100;
  }

  .wss-loading {
    padding: 0.75rem;
    display: flex;
    justify-content: center;
  }

  .wss-desc {
    font-size: 0.72rem;
    color: var(--dt4, #555);
    margin-left: auto;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    max-width: 120px;
  }

  .wss-empty {
    padding: 0.5rem 0.75rem;
    font-size: 0.82rem;
    color: var(--dt4, #555);
  }

  .wss-new {
    color: var(--dt3, #888);
  }

  .wss-plus {
    font-size: 1rem;
    line-height: 1;
    color: var(--daccent, #7ea8ff);
  }

  .wss-create-form {
    display: flex;
    flex-direction: column;
    gap: 0.4rem;
    padding: 0.5rem 0.75rem;
  }

  .wss-input {
    background: var(--dbg2, #131820);
    border: 1px solid var(--dbd, rgba(255, 255, 255, 0.1));
    border-radius: 5px;
    padding: 0.3rem 0.5rem;
    font-size: 0.84rem;
    color: var(--dt, #f1f1f3);
    font-family: inherit;
    outline: none;
    width: 100%;
  }

  .wss-input:focus {
    border-color: var(--daccent, #7ea8ff);
  }

  .wss-create-actions {
    display: flex;
    gap: 0.4rem;
    margin-top: 0.2rem;
  }

  .wss-btn {
    flex: 1;
    padding: 0.3rem 0;
    border-radius: 5px;
    border: 1px solid var(--dbd, rgba(255, 255, 255, 0.1));
    background: var(--dbg2, #131820);
    color: var(--dt2, #ccc);
    font-size: 0.8rem;
    cursor: pointer;
    transition: background 0.1s;
  }

  .wss-btn:hover:not(:disabled) {
    background: var(--dbg, #0d1117);
  }

  .wss-btn:disabled {
    opacity: 0.4;
    cursor: not-allowed;
  }

  .wss-btn--create {
    background: rgba(126, 168, 255, 0.12);
    color: var(--daccent, #7ea8ff);
    border-color: rgba(126, 168, 255, 0.2);
  }

  .wss-btn--create:hover:not(:disabled) {
    background: rgba(126, 168, 255, 0.22);
  }
</style>
