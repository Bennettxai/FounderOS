<!--
  VersionDropdown.svelte

  Quick access dropdown for version selection from the app toolbar.

  Features:
  - Shows recent versions (last 5)
  - Current version indicator
  - Labels for labeled versions
  - "View all versions" link
  - "Save Version" action
-->
<script lang="ts">
  import { ChevronDown, Check, History, Save, Sparkles, Pencil, Camera, RotateCcw } from 'lucide-svelte';
  import type { VersionSummary, VersionTrigger } from '$lib/types/versions';
  import { formatRelativeTime } from '$lib/types/versions';

  interface Props {
    currentVersion: number;
    versions: VersionSummary[];
    isLoading?: boolean;
    onVersionSelect: (version: VersionSummary) => void;
    onViewAll: () => void;
    onSaveVersion: () => void;
    class?: string;
  }

  let {
    currentVersion,
    versions,
    isLoading = false,
    onVersionSelect,
    onViewAll,
    onSaveVersion,
    class: className = ''
  }: Props = $props();

  let isOpen = $state(false);
  let buttonRef: HTMLButtonElement;

  // Show only last 5 versions in dropdown
  const displayVersions = $derived(versions.slice(0, 5));

  function getTriggerIcon(trigger: VersionTrigger) {
    switch (trigger) {
      case 'ai_generation':
        return Sparkles;
      case 'user_edit':
        return Pencil;
      case 'manual_snapshot':
        return Camera;
      case 'auto_snapshot':
        return Save;
      case 'restore':
        return RotateCcw;
      default:
        return History;
    }
  }

  function handleSelect(version: VersionSummary) {
    onVersionSelect(version);
    isOpen = false;
  }

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === 'Escape') {
      isOpen = false;
      buttonRef?.focus();
    }
  }

  function handleClickOutside(e: MouseEvent) {
    const target = e.target as HTMLElement;
    if (!target.closest('.version-dropdown')) {
      isOpen = false;
    }
  }

  $effect(() => {
    if (isOpen) {
      document.addEventListener('click', handleClickOutside);
      return () => document.removeEventListener('click', handleClickOutside);
    }
  });
</script>

<div class="version-dropdown {className}" onkeydown={handleKeydown}>
  <button
    bind:this={buttonRef}
    type="button"
    class="dropdown-trigger"
    class:open={isOpen}
    onclick={() => (isOpen = !isOpen)}
    aria-expanded={isOpen}
    aria-haspopup="listbox"
  >
    <span class="trigger-label">v{currentVersion}</span>
    <ChevronDown size={14} strokeWidth={2} class="chevron" />
  </button>

  {#if isOpen}
    <div class="dropdown-menu" role="listbox">
      {#if isLoading}
        <div class="loading-state">
          <div class="spinner"></div>
          <span>Loading versions...</span>
        </div>
      {:else if displayVersions.length === 0}
        <div class="empty-state">
          <History size={20} strokeWidth={1.5} />
          <span>No version history yet</span>
        </div>
      {:else}
        <div class="version-list">
          {#each displayVersions as version (version.id)}
            {@const TriggerIcon = getTriggerIcon(version.trigger)}
            <button
              type="button"
              class="btn-pill btn-pill-ghost version-item"
              class:current={version.isCurrent}
              onclick={() => handleSelect(version)}
              role="option"
              aria-selected={version.isCurrent}
            >
              <div class="version-main">
                <span class="version-number">v{version.versionNumber}</span>
                {#if version.isCurrent}
                  <Check size={14} strokeWidth={2} class="current-check" />
                {/if}
              </div>
              {#if version.label}
                <span class="version-label">{version.label}</span>
              {/if}
              <div class="version-meta">
                <TriggerIcon size={12} strokeWidth={2} />
                <span>{formatRelativeTime(version.createdAt)}</span>
              </div>
            </button>
          {/each}
        </div>

        <div class="dropdown-divider"></div>

        <button type="button" class="dropdown-action" onclick={() => { onViewAll(); isOpen = false; }}>
          <History size={14} strokeWidth={2} />
          <span>View all versions</span>
        </button>
      {/if}

      <div class="dropdown-divider"></div>

      <button type="button" class="dropdown-action primary" onclick={() => { onSaveVersion(); isOpen = false; }}>
        <Save size={14} strokeWidth={2} />
        <span>Save version</span>
      </button>
    </div>
  {/if}
</div>

<style>
  .version-dropdown {
    position: relative;
    display: inline-block;
  }

  .dropdown-trigger {
    display: inline-flex;
    align-items: center;
    gap: 4px;
    padding: 6px 10px;
    font-size: 13px;
    font-weight: 600;
    color: var(--bos-v2-text-primary);
    background: var(--bos-v2-layer-background-primary);
    border: 1px solid var(--bos-border-color);
    border-radius: 6px;
    cursor: pointer;
    font-family: inherit;
    transition: all 150ms ease;
  }

  .dropdown-trigger:hover {
    background: var(--bos-v2-layer-background-secondary);
    border-color: var(--bos-v2-layer-insideBorder-border);
  }

  .dropdown-trigger.open {
    background: var(--bos-v2-layer-background-secondary);
    border-color: var(--bos-v2-layer-insideBorder-border);
  }

  .chevron {
    color: var(--bos-v2-text-secondary);
    transition: transform 150ms ease;
  }

  .dropdown-trigger.open .chevron {
    transform: rotate(180deg);
  }

  .dropdown-menu {
    position: absolute;
    top: calc(100% + 4px);
    left: 0;
    min-width: 240px;
    background: var(--bos-v2-layer-background-overlayPanel);
    border: 1px solid var(--bos-border-color);
    border-radius: 8px;
    box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
    z-index: 50;
    overflow: hidden;
    animation: slideIn 150ms ease-out;
  }

  :global(.dark) .dropdown-menu {
    box-shadow: 0 4px 12px rgba(0, 0, 0, 0.3);
  }

  @keyframes slideIn {
    from {
      opacity: 0;
      transform: translateY(-4px);
    }
    to {
      opacity: 1;
      transform: translateY(0);
    }
  }

  .version-list {
    padding: 4px;
  }

  .version-item {
    display: flex;
    flex-direction: column;
    align-items: flex-start;
    gap: 2px;
    width: 100%;
    padding: 8px 10px;
    background: transparent;
    border: none;
    border-radius: 6px;
    cursor: pointer;
    font-family: inherit;
    text-align: left;
    transition: background 150ms ease;
  }

  .version-item:hover {
    background: var(--bos-v2-layer-background-secondary);
  }

  .version-item.current {
    background: color-mix(in srgb, var(--bos-status-success) 8%, transparent);
  }

  .version-main {
    display: flex;
    align-items: center;
    gap: 6px;
  }

  .version-number {
    font-size: 13px;
    font-weight: 600;
    color: var(--bos-v2-text-primary);
  }

  .current-check {
    color: var(--bos-status-success);
  }

  .version-label {
    font-size: 12px;
    color: var(--bos-v2-text-secondary);
    max-width: 100%;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .version-meta {
    display: flex;
    align-items: center;
    gap: 4px;
    font-size: 11px;
    color: var(--bos-v2-text-secondary);
  }

  .dropdown-divider {
    height: 1px;
    background: var(--bos-border-color);
    margin: 4px 0;
  }

  .dropdown-action {
    display: flex;
    align-items: center;
    gap: 8px;
    width: 100%;
    padding: 10px 14px;
    font-size: 13px;
    font-weight: 500;
    color: var(--bos-v2-text-primary);
    background: transparent;
    border: none;
    cursor: pointer;
    font-family: inherit;
    text-align: left;
    transition: background 150ms ease;
  }

  .dropdown-action:hover {
    background: var(--bos-v2-layer-background-secondary);
  }

  .dropdown-action.primary {
    color: var(--bos-nav-active);
  }

  :global(.dark) .dropdown-action.primary {
    color: var(--bos-nav-active);
  }

  .loading-state,
  .empty-state {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 8px;
    padding: 24px;
    color: var(--bos-v2-text-secondary);
    font-size: 13px;
  }

  .spinner {
    width: 20px;
    height: 20px;
    border: 2px solid var(--bos-border-color);
    border-top-color: var(--bos-nav-active);
    border-radius: 50%;
    animation: spin 600ms linear infinite;
  }

  @keyframes spin {
    to {
      transform: rotate(360deg);
    }
  }
</style>
