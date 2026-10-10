<script lang="ts">
  import type { EngineContext } from '../engine.js';
  import type { RecallItem } from '@optimal-engine/client';
  import SignalCard from './SignalCard.svelte';

  type Tab = 'actions' | 'who' | 'when' | 'where' | 'owns';

  interface Props {
    engine: EngineContext;
    defaultTab?: Tab;
  }

  let { engine, defaultTab = 'actions' }: Props = $props();

  const TABS: { id: Tab; label: string }[] = [
    { id: 'actions', label: 'Actions' },
    { id: 'who', label: 'Who' },
    { id: 'when', label: 'When' },
    { id: 'where', label: 'Where' },
    { id: 'owns', label: 'Owns' },
  ];

  let activeTab = $state<Tab>(defaultTab);
  let loading = $state(false);
  let error = $state<string | null>(null);
  let results = $state<RecallItem[]>([]);

  // Per-tab field state
  let actorsTopic = $state('');
  let actionsActor = $state('');
  let actionsSince = $state('');
  let whoTopic = $state('');
  let whoRole = $state('');
  let whenEvent = $state('');
  let whereThing = $state('');
  let ownsActor = $state('');

  async function submit() {
    const ws = engine.getWorkspace();
    loading = true;
    error = null;
    results = [];
    try {
      let res;
      switch (activeTab) {
        case 'actions':
          res = await engine.client.recall.actions({
            topic: actorsTopic,
            actor: actionsActor || undefined,
            since: actionsSince || undefined,
            workspace: ws,
          });
          break;
        case 'who':
          res = await engine.client.recall.who({
            topic: whoTopic,
            role: whoRole || undefined,
            workspace: ws,
          });
          break;
        case 'when':
          res = await engine.client.recall.when({
            event: whenEvent,
            workspace: ws,
          });
          break;
        case 'where':
          res = await engine.client.recall.where({
            thing: whereThing,
            workspace: ws,
          });
          break;
        case 'owns':
          res = await engine.client.recall.owns({
            actor: ownsActor,
            workspace: ws,
          });
          break;
      }
      results = res?.items ?? [];
    } catch (e) {
      error = (e as Error).message;
    } finally {
      loading = false;
    }
  }

  function onTabChange(tab: Tab) {
    activeTab = tab;
    results = [];
    error = null;
  }

  function onKeydown(e: KeyboardEvent) {
    if (e.key === 'Enter' && (e.metaKey || e.ctrlKey || e.target instanceof HTMLInputElement)) {
      void submit();
    }
  }

  function isSubmittable(): boolean {
    switch (activeTab) {
      case 'actions': return !!actorsTopic.trim();
      case 'who': return !!whoTopic.trim();
      case 'when': return !!whenEvent.trim();
      case 'where': return !!whereThing.trim();
      case 'owns': return !!ownsActor.trim();
    }
  }
</script>

<div class="rw-root">
  <!-- Tab pills -->
  <div class="rw-tabs" role="tablist" aria-label="Recall type">
    {#each TABS as tab (tab.id)}
      <button
        class="rw-tab {activeTab === tab.id ? 'rw-tab--active' : ''}"
        role="tab"
        aria-selected={activeTab === tab.id}
        onclick={() => onTabChange(tab.id)}
      >
        {tab.label}
      </button>
    {/each}
  </div>

  <!-- Input area — changes per tab -->
  <div class="rw-inputs" role="tabpanel" onkeydown={onKeydown}>
    {#if activeTab === 'actions'}
      <div class="rw-row">
        <input class="rw-input rw-input--main" bind:value={actorsTopic} placeholder="Topic…" aria-label="Topic"/>
        <input class="rw-input" bind:value={actionsActor} placeholder="Actor (opt.)" aria-label="Actor"/>
        <input class="rw-input" bind:value={actionsSince} placeholder="Since (opt.)" aria-label="Since"/>
      </div>
    {:else if activeTab === 'who'}
      <div class="rw-row">
        <input class="rw-input rw-input--main" bind:value={whoTopic} placeholder="Topic…" aria-label="Topic"/>
        <input class="rw-input" bind:value={whoRole} placeholder="Role (opt.)" aria-label="Role"/>
      </div>
    {:else if activeTab === 'when'}
      <div class="rw-row">
        <input class="rw-input rw-input--main" bind:value={whenEvent} placeholder="Event…" aria-label="Event"/>
      </div>
    {:else if activeTab === 'where'}
      <div class="rw-row">
        <input class="rw-input rw-input--main" bind:value={whereThing} placeholder="Thing…" aria-label="Thing"/>
      </div>
    {:else if activeTab === 'owns'}
      <div class="rw-row">
        <input class="rw-input rw-input--main" bind:value={ownsActor} placeholder="Actor…" aria-label="Actor"/>
      </div>
    {/if}

    <button
      class="rw-submit"
      disabled={!isSubmittable() || loading}
      onclick={submit}
      aria-label="Run recall"
    >
      {#if loading}
        <div class="oe-spinner rw-spinner"></div>
        Searching…
      {:else}
        Recall
      {/if}
    </button>
  </div>

  <!-- Error -->
  {#if error}
    <div class="rw-error">{error}</div>
  {/if}

  <!-- Results -->
  <div class="rw-results" role="region" aria-label="Recall results" aria-live="polite">
    {#if !loading && results.length === 0 && !error}
      <div class="oe-empty rw-empty">
        <span>No results for this recall.</span>
      </div>
    {:else}
      {#each results as item (item.id ?? item.content ?? Math.random())}
        <SignalCard
          id={item.id ?? ''}
          title={item.content?.slice(0, 80) ?? '(no content)'}
          createdAt={item.timestamp}
        />
      {/each}
    {/if}
  </div>
</div>

<style>
  .rw-root {
    display: flex;
    flex-direction: column;
    gap: 0.6rem;
  }

  /* Tab pills */
  .rw-tabs {
    display: flex;
    gap: 0.2rem;
    flex-wrap: wrap;
  }

  .rw-tab {
    padding: 0.25rem 0.7rem;
    border-radius: 999px;
    border: 1px solid var(--dbd, rgba(255,255,255,0.1));
    background: none;
    font-size: 0.78rem;
    font-weight: 500;
    color: var(--dt3, #888);
    cursor: pointer;
    transition: background 0.1s, color 0.1s, border-color 0.1s;
  }

  .rw-tab:hover {
    background: var(--dbg2, #131820);
    color: var(--dt2, #ccc);
  }

  .rw-tab--active {
    background: rgba(126, 168, 255, 0.12);
    color: var(--daccent, #7ea8ff);
    border-color: rgba(126, 168, 255, 0.25);
  }

  /* Input row */
  .rw-inputs {
    display: flex;
    flex-direction: column;
    gap: 0.4rem;
  }

  .rw-row {
    display: flex;
    gap: 0.4rem;
    flex-wrap: wrap;
  }

  .rw-input {
    flex: 1;
    min-width: 80px;
    background: var(--dbg2, #131820);
    border: 1px solid var(--dbd, rgba(255,255,255,0.1));
    border-radius: 6px;
    padding: 0.35rem 0.6rem;
    font-size: 0.84rem;
    color: var(--dt, #f1f1f3);
    font-family: inherit;
    outline: none;
    transition: border-color 0.15s;
  }

  .rw-input:focus {
    border-color: var(--daccent, #7ea8ff);
  }

  .rw-input::placeholder {
    color: var(--dt4, #555);
  }

  .rw-input--main {
    flex: 2;
  }

  .rw-submit {
    align-self: flex-start;
    display: inline-flex;
    align-items: center;
    gap: 0.4rem;
    padding: 0.35rem 1rem;
    border-radius: 6px;
    border: none;
    background: var(--daccent, #7ea8ff);
    color: #0d1117;
    font-size: 0.84rem;
    font-weight: 600;
    cursor: pointer;
    transition: opacity 0.1s;
  }

  .rw-submit:disabled {
    opacity: 0.4;
    cursor: not-allowed;
  }

  .rw-spinner {
    width: 14px;
    height: 14px;
    border-top-color: #0d1117;
  }

  /* Error */
  .rw-error {
    padding: 0.35rem 0.6rem;
    border-radius: 5px;
    background: rgba(248,136,136,0.08);
    border: 1px solid rgba(248,136,136,0.2);
    font-size: 0.78rem;
    color: #f88;
  }

  /* Results */
  .rw-results {
    display: flex;
    flex-direction: column;
    gap: 0.3rem;
    min-height: 1px;
  }

  .rw-empty {
    padding: 1.5rem 0.5rem;
    font-size: 0.82rem;
    justify-content: flex-start;
  }
</style>
