<script lang="ts">
  import '../app.css';
  import { onMount } from 'svelte';
  import { page } from '$app/stores';
  import { status, createWorkspace, type StatusPayload } from '$lib/api';
  import { theme, toggle } from '$lib/theme';
  import {
    organizations,
    workspaces,
    activeOrg,
    activeWorkspace,
    activeOrgId,
    activeWorkspaceId,
    bootstrap as bootstrapWorkspace,
    setActiveOrg,
    setActiveWorkspace,
    refreshWorkspaces
  } from '$lib/stores/workspace';

  let engineStatus = $state<StatusPayload['status']>('down');
  let { children } = $props<{ children?: () => any }>();

  let orgMenuOpen = $state(false);
  let wsMenuOpen = $state(false);
  let creating = $state(false);
  let createName = $state('');
  let createSlug = $state('');

  const FULL_BLEED = new Set(['/graph', '/workspace', '/activity']);
  let fullBleed = $derived(FULL_BLEED.has($page.url.pathname));

  onMount(async () => {
    try {
      const s = await status();
      engineStatus = s.status;
    } catch {
      engineStatus = 'down';
    }
    await bootstrapWorkspace();
  });

  function isActive(path: string): boolean {
    if (path === '/') return $page.url.pathname === '/';
    return $page.url.pathname.startsWith(path);
  }

  function closeMenus() {
    orgMenuOpen = false;
    wsMenuOpen = false;
    creating = false;
  }

  function openCreate() {
    wsMenuOpen = false;
    creating = true;
    createName = '';
    createSlug = '';
  }

  function slugify(s: string): string {
    return s.toLowerCase().trim().replace(/[^a-z0-9]+/g, '-').replace(/^-|-$/g, '');
  }

  async function submitCreate(e: Event) {
    e.preventDefault();
    if (!createName.trim()) return;
    const slug = createSlug.trim() || slugify(createName);
    try {
      await createWorkspace({ slug, name: createName.trim(), tenant: $activeOrgId ?? 'default' });
      await refreshWorkspaces();
      const created = $workspaces.find((w) => w.slug === slug);
      if (created) setActiveWorkspace(created.id);
      creating = false;
    } catch (err) {
      alert((err as Error).message);
    }
  }
</script>

<svelte:window onclick={closeMenus} />

<header class="app-header glass">
  <div class="app-header__inner">
    <a href="/" class="app-header__brand">
      <span class="app-header__mark">◇</span>
      <span class="app-header__name">Optimal Engine</span>
    </a>

    <!-- Org switcher -->
    <div class="switch" onclick={(e) => e.stopPropagation()} role="presentation">
      <button
        type="button"
        class="switch__btn"
        onclick={() => { orgMenuOpen = !orgMenuOpen; wsMenuOpen = false; creating = false; }}
        disabled={!$activeOrg}
      >
        <span class="switch__label">Org</span>
        <span class="switch__name">{$activeOrg?.name ?? '—'}</span>
        <span class="switch__caret">▾</span>
      </button>
      {#if orgMenuOpen}
        <div class="switch__menu">
          {#each $organizations as org}
            <button
              type="button"
              class="switch__item"
              class:active={org.id === $activeOrgId}
              onclick={() => { setActiveOrg(org.id); orgMenuOpen = false; }}
            >
              <span class="switch__item-name">{org.name}</span>
              <span class="switch__item-meta">{org.plan}</span>
            </button>
          {/each}
        </div>
      {/if}
    </div>

    <span class="switch__sep">/</span>

    <!-- Workspace switcher -->
    <div class="switch" onclick={(e) => e.stopPropagation()} role="presentation">
      <button
        type="button"
        class="switch__btn switch__btn--accent"
        onclick={() => { wsMenuOpen = !wsMenuOpen; orgMenuOpen = false; creating = false; }}
        disabled={!$activeWorkspace}
      >
        <span class="switch__label">Workspace</span>
        <span class="switch__name">{$activeWorkspace?.name ?? '—'}</span>
        <span class="switch__caret">▾</span>
      </button>
      {#if wsMenuOpen}
        <div class="switch__menu">
          {#each $workspaces as ws}
            <button
              type="button"
              class="switch__item"
              class:active={ws.id === $activeWorkspaceId}
              onclick={() => { setActiveWorkspace(ws.id); wsMenuOpen = false; }}
            >
              <span class="switch__item-name">{ws.name}</span>
              <span class="switch__item-meta">{ws.slug}</span>
            </button>
          {/each}
          <div class="switch__divider"></div>
          <button type="button" class="switch__item switch__item--cta" onclick={openCreate}>
            + New workspace
          </button>
        </div>
      {/if}
      {#if creating}
        <form class="switch__menu switch__create" onsubmit={submitCreate}>
          <label>
            <span>Name</span>
            <input type="text" bind:value={createName} placeholder="Engineering Brain" autofocus />
          </label>
          <label>
            <span>Slug <em>(optional)</em></span>
            <input type="text" bind:value={createSlug} placeholder={createName ? slugify(createName) : 'engineering'} />
          </label>
          <div class="switch__create-actions">
            <button type="button" class="btn btn--sm" onclick={() => (creating = false)}>Cancel</button>
            <button type="submit" class="btn btn--sm btn--primary" disabled={!createName.trim()}>Create</button>
          </div>
        </form>
      {/if}
    </div>

    <nav class="app-header__nav">
      <a href="/" class:active={isActive('/')}>Ask</a>
      <a href="/surface" class:active={isActive('/surface')}>Surface</a>
      <a href="/workspace" class:active={isActive('/workspace')}>Nodes</a>
      <a href="/graph" class:active={isActive('/graph')}>Graph</a>
      <a href="/timeline" class:active={isActive('/timeline')}>Timeline</a>
      <a href="/heatmap" class:active={isActive('/heatmap')}>Heatmap</a>
      <a href="/wiki" class:active={isActive('/wiki')}>Wiki</a>
      <a href="/memory" class:active={isActive('/memory')}>Memory</a>
      <a href="/architectures" class:active={isActive('/architectures')}>Architectures</a>
      <a href="/activity" class:active={isActive('/activity')}>Activity</a>
      <a href="/status" class:active={isActive('/status')}>Status</a>
    </nav>

    <div class="app-header__tools">
      <span class="chip chip--{engineStatus === 'up' ? 'good' : engineStatus === 'down' ? 'bad' : 'warn'}">
        engine · {engineStatus}
      </span>
      <button
        class="app-header__theme"
        onclick={toggle}
        aria-label="Toggle theme"
        title={$theme === 'dark' ? 'Switch to light' : 'Switch to dark'}
      >
        {#if $theme === 'dark'}☀︎{:else}☾{/if}
      </button>
    </div>
  </div>
</header>

<main class:main--full={fullBleed}>
  {@render children?.()}
</main>

<style>
  .app-header {
    position: sticky;
    top: 0;
    z-index: 50;
    border-radius: 0;
    border-left: none;
    border-right: none;
    border-top: none;
    border-bottom: 1px solid var(--border);
  }
  .app-header__inner {
    display: flex;
    align-items: center;
    gap: 1rem;
    padding: 0.7rem 1.5rem;
    height: 56px;
  }
  .app-header__brand {
    display: inline-flex;
    align-items: center;
    gap: 0.5rem;
    color: var(--text);
    font-weight: 600;
    text-decoration: none;
  }
  .app-header__brand:hover { text-decoration: none; }
  .app-header__mark {
    color: var(--accent);
    font-size: 1.05rem;
  }
  .app-header__name {
    font-size: 0.92rem;
    letter-spacing: 0.2px;
  }
  .app-header__nav {
    flex: 1;
    display: flex;
    gap: 0.15rem;
    margin-left: 0.85rem;
  }
  .app-header__nav a {
    color: var(--text-muted);
    text-decoration: none;
    font-size: 0.84rem;
    padding: 5px 10px;
    border-radius: 6px;
    transition: background 0.12s, color 0.12s;
  }
  .app-header__nav a:hover {
    color: var(--text);
    background: var(--bg-elevated-2);
    text-decoration: none;
  }
  .app-header__nav a.active {
    color: var(--accent);
    background: var(--accent-soft);
  }
  .app-header__tools {
    display: flex;
    align-items: center;
    gap: 0.5rem;
  }
  .app-header__theme {
    background: var(--bg-elevated);
    border: 1px solid var(--border);
    color: var(--text-muted);
    width: 30px;
    height: 30px;
    border-radius: 6px;
    cursor: pointer;
    font-size: 0.9rem;
    display: inline-flex;
    align-items: center;
    justify-content: center;
  }
  .app-header__theme:hover {
    color: var(--text);
    background: var(--bg-elevated-2);
  }

  main {
    padding: 1.5rem 1.5rem 3rem;
    max-width: 1080px;
    margin: 0 auto;
  }
  main.main--full {
    padding: 0;
    max-width: none;
    margin: 0;
    height: calc(100vh - 56px);
    width: 100%;
  }

  @media (max-width: 880px) {
    .app-header__nav { display: none; }
  }

  /* ── Switchers (Org + Workspace dropdowns) ── */
  .switch {
    position: relative;
    display: inline-flex;
  }
  .switch__btn {
    display: inline-flex;
    align-items: center;
    gap: 0.4rem;
    padding: 0.35rem 0.7rem 0.35rem 0.6rem;
    border: 1px solid var(--border);
    border-radius: var(--r-md);
    background: var(--bg-elevated);
    color: var(--text);
    font: inherit;
    font-size: 0.82rem;
    cursor: pointer;
    transition: background 0.12s, border-color 0.12s;
  }
  .switch__btn:hover:not(:disabled) {
    border-color: var(--accent);
    background: var(--bg-elevated-2);
  }
  .switch__btn:disabled { opacity: 0.5; cursor: not-allowed; }
  .switch__btn--accent {
    background: var(--accent-soft);
    border-color: color-mix(in srgb, var(--accent) 35%, transparent);
    color: var(--text);
  }
  .switch__btn--accent:hover:not(:disabled) {
    background: color-mix(in srgb, var(--accent) 18%, var(--bg-elevated));
  }
  .switch__label {
    color: var(--text-subtle);
    font-size: 0.65rem;
    font-weight: 700;
    text-transform: uppercase;
    letter-spacing: 0.08em;
  }
  .switch__name {
    color: var(--text);
    font-weight: 600;
    max-width: 16ch;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .switch__caret {
    color: var(--text-subtle);
    font-size: 0.7rem;
  }
  .switch__sep {
    color: var(--text-subtle);
    font-size: 1rem;
    margin: 0 0.05rem;
    user-select: none;
  }

  .switch__menu {
    position: absolute;
    top: calc(100% + 6px);
    left: 0;
    z-index: 60;
    min-width: 220px;
    padding: 0.35rem;
    background: var(--bg-elevated);
    border: 1px solid var(--border);
    border-radius: var(--r-md);
    box-shadow: 0 16px 40px -12px rgba(0, 0, 0, 0.55);
    display: flex;
    flex-direction: column;
    gap: 1px;
  }
  .switch__item {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 0.5rem;
    padding: 0.5rem 0.7rem;
    background: transparent;
    border: 1px solid transparent;
    border-radius: var(--r-sm);
    color: var(--text);
    font: inherit;
    font-size: 0.82rem;
    cursor: pointer;
    text-align: left;
  }
  .switch__item:hover { background: var(--bg-elevated-2); }
  .switch__item.active {
    background: var(--accent-soft);
    color: var(--accent);
  }
  .switch__item-name { font-weight: 500; }
  .switch__item-meta {
    color: var(--text-subtle);
    font-size: 0.7rem;
    font-family: var(--font-mono);
  }
  .switch__item--cta {
    color: var(--accent);
    font-weight: 600;
    justify-content: flex-start;
  }
  .switch__divider {
    height: 1px;
    background: var(--border-soft);
    margin: 0.3rem 0;
  }

  .switch__create {
    padding: 0.85rem;
    min-width: 280px;
    gap: 0.7rem;
  }
  .switch__create label {
    display: flex;
    flex-direction: column;
    gap: 0.3rem;
    font-size: 0.7rem;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.06em;
    color: var(--text-muted);
  }
  .switch__create label em {
    color: var(--text-subtle);
    font-style: normal;
    text-transform: none;
    letter-spacing: normal;
    font-weight: 400;
  }
  .switch__create input {
    background: var(--bg);
    border: 1px solid var(--border);
    border-radius: var(--r-sm);
    color: var(--text);
    padding: 0.45rem 0.65rem;
    font: inherit;
    font-size: 0.85rem;
    text-transform: none;
  }
  .switch__create input:focus {
    outline: none;
    border-color: var(--accent);
    box-shadow: 0 0 0 3px var(--accent-soft);
  }
  .switch__create-actions {
    display: flex;
    gap: 0.4rem;
    justify-content: flex-end;
  }
</style>
