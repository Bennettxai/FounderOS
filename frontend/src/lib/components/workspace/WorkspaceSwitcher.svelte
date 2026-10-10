<script lang="ts">
  import { onMount } from 'svelte';
  import type { Workspace } from '$lib/api/workspaces';
  import {
    workspaces,
    currentWorkspace,
    currentUserRole,
    workspaceLoading,
    workspaceError,
    switchWorkspace,
  } from '$lib/stores/workspaces';
  import { ChevronDown, Building2, Loader2, AlertCircle, Search, Check, HardDrive, Cloud, Plus } from 'lucide-svelte';
  import WorkspaceCreateModal from './WorkspaceCreateModal.svelte';

  let isOpen = $state(false);
  let query = $state('');
  let showCreate = $state(false);
  let dropdownRef: HTMLDivElement;
  let triggerRef: HTMLButtonElement;
  // The dropdown is position:fixed (so it escapes the sidebar's overflow:hidden,
  // which was clipping it under the main panel). Positioned from the trigger rect.
  let menuTop = $state(0);
  let menuLeft = $state(0);
  let menuWidth = $state(300);

  const filteredWorkspaces = $derived.by(() => {
    const q = query.trim().toLowerCase();
    if (!q) return $workspaces;
    return $workspaces.filter((workspace) => {
      return `${workspace.name} ${workspace.slug}`.toLowerCase().includes(q);
    });
  });

  const localWorkspaceCount = $derived(
    $workspaces.filter((workspace) => isLocalKnowledgeWorkspace(workspace)).length,
  );

  function isLocalKnowledgeWorkspace(workspace: Workspace): boolean {
    return workspace.id.startsWith('kb:') || workspace.settings?.source === 'knowledge';
  }

  function workspaceKind(workspace: Workspace): string {
    return isLocalKnowledgeWorkspace(workspace) ? 'Local knowledge' : 'Workspace';
  }

  function workspaceInitials(name: string): string {
    const parts = name.trim().split(/\s+/).filter(Boolean);
    if (parts.length === 0) return '?';
    if (parts.length === 1) return parts[0].slice(0, 2).toUpperCase();
    return `${parts[0][0]}${parts[1][0]}`.toUpperCase();
  }

  function avatarTone(workspace: Workspace): string {
    const tones = ['tone-blue', 'tone-violet', 'tone-teal', 'tone-amber', 'tone-rose'];
    const seed = workspace.slug.split('').reduce((sum, char) => sum + char.charCodeAt(0), 0);
    return tones[seed % tones.length];
  }

  onMount(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (dropdownRef && !dropdownRef.contains(event.target as Node)) {
        isOpen = false;
      }
    };

    const handleEscape = (event: KeyboardEvent) => {
      if (event.key === 'Escape' && isOpen) {
        isOpen = false;
      }
    };

    const handleReposition = () => {
      if (isOpen) positionMenu();
    };

    document.addEventListener('click', handleClickOutside);
    document.addEventListener('keydown', handleEscape);
    window.addEventListener('resize', handleReposition);
    window.addEventListener('scroll', handleReposition, true);
    return () => {
      document.removeEventListener('click', handleClickOutside);
      document.removeEventListener('keydown', handleEscape);
      window.removeEventListener('resize', handleReposition);
      window.removeEventListener('scroll', handleReposition, true);
    };
  });

  async function handleWorkspaceSelect(workspaceId: string) {
    if ($currentWorkspace?.id === workspaceId) {
      isOpen = false;
      return;
    }

    try {
      await switchWorkspace(workspaceId);
      isOpen = false;
      query = '';
    } catch (error) {
      console.error('Failed to switch workspace:', error);
    }
  }

  function positionMenu() {
    if (!triggerRef) return;
    const r = triggerRef.getBoundingClientRect();
    menuTop = r.bottom + 6;
    menuLeft = r.left;
    menuWidth = Math.min(300, window.innerWidth - 24);
  }

  function toggleDropdown() {
    isOpen = !isOpen;
    if (isOpen) positionMenu();
  }

  function openCreate() {
    isOpen = false;
    query = '';
    showCreate = true;
  }
</script>

<div class="workspace-switcher" bind:this={dropdownRef}>
  <!-- Trigger Button -->
  <button
    bind:this={triggerRef}
    class="btn-pill btn-pill-ghost workspace-trigger"
    class:loading={$workspaceLoading.switching}
    disabled={$workspaceLoading.switching}
    aria-label="Switch workspace"
    aria-expanded={isOpen}
    onclick={toggleDropdown}
  >
    {#if $workspaceLoading.switching}
      <Loader2 class="w-4 h-4 animate-spin text-gray-400" />
    {:else}
      <Building2 class="w-4 h-4 text-gray-400" />
    {/if}

    <div class="workspace-info">
      {#if $currentWorkspace}
        <span class="workspace-name">{$currentWorkspace.name}</span>
        {#if $currentUserRole}
          <span class="workspace-role">{$currentUserRole}</span>
        {/if}
      {:else}
        <span class="workspace-name text-gray-400">Select Workspace</span>
      {/if}
    </div>

    <div class="workspace-chevron" class:rotate-180={isOpen}>
      <ChevronDown class="w-4 h-4 text-gray-400" />
    </div>
  </button>

  {#if isOpen}
    <div
      class="workspace-dropdown"
      style="--ws-top:{menuTop}px; --ws-left:{menuLeft}px; --ws-width:{menuWidth}px;"
    >
      <div class="workspace-dropdown-head">
        <div>
          <div class="workspace-dropdown-title">Switch workspace</div>
          <div class="workspace-dropdown-meta">
            {$workspaces.length} total
            {#if localWorkspaceCount > 0}
              / {localWorkspaceCount} local
            {/if}
          </div>
        </div>
      </div>

      <label class="workspace-search">
        <Search size={14} strokeWidth={2} />
        <input bind:value={query} placeholder="Find workspace..." autocomplete="off" />
      </label>

      {#if $workspaceError}
        <div class="error-message">
          <AlertCircle class="w-4 h-4" />
          <span>{$workspaceError}</span>
        </div>
      {/if}

      {#if $workspaceLoading.workspaces}
        <div class="loading-state">
          <Loader2 class="w-5 h-5 animate-spin" />
          <span>Loading workspaces...</span>
        </div>
      {:else if $workspaces.length === 0}
        <div class="empty-state">
          <Building2 class="w-8 h-8 text-gray-300 mb-2" />
          <p class="text-sm text-gray-500">No workspaces available</p>
        </div>
      {:else if filteredWorkspaces.length === 0}
        <div class="empty-state">
          <Search class="w-6 h-6 text-gray-400 mb-2" />
          <p class="text-sm text-gray-500">No matching workspaces</p>
        </div>
      {:else}
        <div class="workspace-list">
          {#each filteredWorkspaces as workspace (workspace.id)}
            <button
              class="workspace-item"
              class:active={$currentWorkspace?.id === workspace.id}
              onclick={() => handleWorkspaceSelect(workspace.id)}
            >
              <div class="workspace-item-icon">
                {#if workspace.logo_url}
                  <img src={workspace.logo_url} alt={workspace.name} />
                {:else}
                  <div class="workspace-avatar {avatarTone(workspace)}">
                    {workspaceInitials(workspace.name)}
                  </div>
                {/if}
              </div>

              <div class="workspace-item-info">
                <span class="workspace-item-name">{workspace.name}</span>
                <span class="workspace-item-slug">
                  {#if isLocalKnowledgeWorkspace(workspace)}
                    <HardDrive size={11} strokeWidth={2} />
                  {:else}
                    <Cloud size={11} strokeWidth={2} />
                  {/if}
                  {workspaceKind(workspace)}
                  <span class="workspace-slug-dot">/</span>
                  {workspace.slug}
                </span>
              </div>

              {#if $currentWorkspace?.id === workspace.id}
                <div class="workspace-item-check">
                  <Check size={15} strokeWidth={2.4} />
                </div>
              {/if}
            </button>
          {/each}
        </div>
      {/if}

      <div class="workspace-dropdown-foot">
        <button class="workspace-create-btn" onclick={openCreate}>
          <Plus size={15} strokeWidth={2.2} />
          <span>New workspace</span>
        </button>
        <p class="workspace-org-hint">Workspaces belong to an organization</p>
      </div>
    </div>
  {/if}
</div>

<WorkspaceCreateModal show={showCreate} onClose={() => (showCreate = false)} />

<style>
  .workspace-switcher {
    position: relative;
    width: 100%;
  }

  .workspace-trigger {
    display: flex;
    align-items: center;
    gap: 0.55rem;
    padding: 0.45rem 0.55rem;
    width: 100%;
    height: 44px;
    background: color-mix(in srgb, var(--dt, #111) 4%, transparent);
    border: 1px solid color-mix(in srgb, var(--dt, #111) 9%, transparent);
    border-radius: var(--radius-sm, 8px);
    cursor: pointer;
    transition: background 160ms ease, border-color 160ms ease;
    min-width: 0;
  }

  .workspace-trigger:hover:not(:disabled) {
    background: color-mix(in srgb, var(--dt, #111) 7%, transparent);
    border-color: color-mix(in srgb, var(--dt, #111) 16%, transparent);
  }

  .workspace-trigger:disabled {
    opacity: 0.6;
    cursor: not-allowed;
  }

  .workspace-info {
    flex: 1;
    display: flex;
    flex-direction: column;
    align-items: flex-start;
    gap: 1px;
    min-width: 0;
  }

  .workspace-name {
    font-size: 0.78rem;
    font-weight: 650;
    color: var(--dt, #111);
    line-height: 1.15;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    max-width: 100%;
  }

  .workspace-role {
    font-size: 0.65rem;
    color: var(--dt3, #888);
    line-height: 1.15;
    text-transform: capitalize;
  }

  .workspace-chevron {
    display: flex;
    align-items: center;
    justify-content: center;
    transition: transform 160ms ease;
  }

  .workspace-dropdown {
    /* Fixed (not absolute) so it escapes the sidebar's overflow:hidden and is
       never clipped under the main panel. Positioned from the trigger rect via
       the --ws-* custom properties set inline. */
    position: fixed;
    top: var(--ws-top, calc(100% + 6px));
    left: var(--ws-left, 0);
    width: var(--ws-width, min(300px, calc(100vw - 24px)));
    max-height: min(420px, calc(100vh - 96px));
    overflow: hidden;
    background: color-mix(in srgb, var(--dbg, #fff) 96%, #000 4%);
    border: 1px solid color-mix(in srgb, var(--dt, #111) 12%, transparent);
    border-radius: 10px;
    box-shadow: 0 18px 48px rgb(0 0 0 / 0.42), 0 0 0 1px rgb(255 255 255 / 0.03) inset;
    z-index: 1000;
  }

  @media (max-width: 640px) {
    .workspace-dropdown {
      /* Bottom-sheet on mobile: fixed to viewport bottom, full width.
         These explicit values override the --ws-* desktop positioning above. */
      position: fixed;
      top: auto;
      bottom: 0;
      left: 0;
      right: 0;
      width: 100%;
      max-width: 100%;
      max-height: 70vh;
      border-radius: 14px 14px 0 0;
      border-bottom: none;
      overflow-y: auto;
      /* Safe area for home indicator */
      padding-bottom: env(safe-area-inset-bottom, 0px);
    }

    .workspace-list {
      max-height: none;
    }

    .workspace-item {
      min-height: 52px;
      padding: 0.6rem 0.75rem;
    }

    .workspace-create-btn {
      min-height: 44px;
      padding: 0.6rem 0.75rem;
    }
  }

  .workspace-dropdown-head {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
    padding: 0.7rem 0.75rem 0.45rem;
  }

  .workspace-dropdown-title {
    font-size: 0.76rem;
    font-weight: 650;
    color: var(--dt, #111);
    line-height: 1.15;
  }

  .workspace-dropdown-meta {
    margin-top: 2px;
    font-size: 0.64rem;
    color: var(--dt3, #888);
    line-height: 1.2;
  }

  .workspace-search {
    display: flex;
    align-items: center;
    gap: 7px;
    margin: 0 0.5rem 0.45rem;
    padding: 0.42rem 0.5rem;
    border: 1px solid color-mix(in srgb, var(--dt, #111) 10%, transparent);
    border-radius: 7px;
    background: color-mix(in srgb, var(--dt, #111) 4%, transparent);
    color: var(--dt3, #888);
  }

  .workspace-search input {
    flex: 1;
    min-width: 0;
    border: 0;
    outline: 0;
    background: transparent;
    color: var(--dt, #111);
    font-size: 0.74rem;
    line-height: 1.2;
  }

  .workspace-search input::placeholder {
    color: var(--dt3, #888);
  }

  .error-message {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    padding: 0.625rem 0.75rem;
    background: rgba(220, 38, 38, 0.08);
    border-bottom: 1px solid rgba(220, 38, 38, 0.15);
    color: var(--color-error, #dc2626);
    font-size: 0.8rem;
  }

  .loading-state,
  .empty-state {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    padding: 1.5rem;
    color: var(--dt3, #888);
  }

  .loading-state {
    gap: 0.5rem;
  }

  .workspace-list {
    max-height: 286px;
    overflow-y: auto;
    padding: 0.25rem;
    scrollbar-width: thin;
  }

  .workspace-item {
    display: flex;
    align-items: center;
    gap: 0.58rem;
    width: 100%;
    min-height: 48px;
    padding: 0.45rem 0.48rem;
    border-radius: 7px;
    cursor: pointer;
    transition: background 120ms ease, color 120ms ease;
    border: 1px solid transparent;
    background: transparent;
    text-align: left;
    color: var(--dt2, #444);
  }

  .workspace-item:hover {
    background: color-mix(in srgb, var(--dt, #111) 6%, transparent);
  }

  .workspace-item.active {
    background: color-mix(in srgb, #6366f1 14%, transparent);
    border-color: color-mix(in srgb, #6366f1 26%, transparent);
  }

  .workspace-item-icon {
    flex-shrink: 0;
  }

  .workspace-avatar {
    width: 1.78rem;
    height: 1.78rem;
    display: flex;
    align-items: center;
    justify-content: center;
    color: white;
    font-weight: 650;
    font-size: 0.68rem;
    border-radius: 6px;
    letter-spacing: 0;
  }

  .workspace-item-icon img {
    width: 1.78rem;
    height: 1.78rem;
    border-radius: 6px;
    object-fit: cover;
  }

  .tone-blue { background: linear-gradient(135deg, #2563eb, #4f46e5); }
  .tone-violet { background: linear-gradient(135deg, #7c3aed, #a855f7); }
  .tone-teal { background: linear-gradient(135deg, #0f766e, #14b8a6); }
  .tone-amber { background: linear-gradient(135deg, #b45309, #f59e0b); }
  .tone-rose { background: linear-gradient(135deg, #be123c, #f43f5e); }

  .workspace-item.active .workspace-avatar {
    box-shadow: 0 0 0 1px rgb(255 255 255 / 0.2) inset;
  }

  .workspace-item-info {
    flex: 1;
    display: flex;
    flex-direction: column;
    gap: 1px;
    min-width: 0;
  }

  .workspace-item-name {
    font-size: 0.78rem;
    font-weight: 600;
    color: var(--dt, #111);
    line-height: 1.18;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .workspace-item-slug {
    display: flex;
    align-items: center;
    gap: 4px;
    min-width: 0;
    font-size: 0.62rem;
    color: var(--dt3, #888);
    line-height: 1.15;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .workspace-slug-dot {
    opacity: 0.55;
  }

  .workspace-item-check {
    flex-shrink: 0;
    color: #818cf8;
  }

  .workspace-dropdown-foot {
    padding: 0.35rem 0.5rem 0.5rem;
    border-top: 1px solid color-mix(in srgb, var(--dt, #111) 10%, transparent);
  }

  .workspace-create-btn {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    width: 100%;
    padding: 0.5rem 0.55rem;
    border-radius: 7px;
    border: 1px dashed color-mix(in srgb, var(--dt, #111) 20%, transparent);
    background: transparent;
    color: var(--dt2, #444);
    font-size: 0.76rem;
    font-weight: 600;
    cursor: pointer;
    transition: background 120ms ease, border-color 120ms ease, color 120ms ease;
  }

  .workspace-create-btn:hover {
    background: color-mix(in srgb, var(--dt, #111) 6%, transparent);
    border-color: color-mix(in srgb, var(--dt, #111) 32%, transparent);
    color: var(--dt, #111);
  }

  .workspace-org-hint {
    margin: 0.3rem 0 0;
    font-size: 0.62rem;
    color: var(--dt3, #888);
    text-align: center;
    line-height: 1.3;
  }
</style>
