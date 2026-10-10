<!--
  VersionBadge.svelte

  Always-visible indicator showing current version status.

  States:
  - Normal: "v5 • Saved"
  - Unsaved: "v5 • Unsaved" (with dot indicator)
  - Previewing: "Viewing v3"
-->
<script lang="ts">
  import { Check, Circle, Eye } from 'lucide-svelte';

  interface Props {
    currentVersion: number;
    hasUnsavedChanges?: boolean;
    isPreviewingOldVersion?: boolean;
    previewingVersion?: number;
    onclick?: () => void;
    class?: string;
  }

  let {
    currentVersion,
    hasUnsavedChanges = false,
    isPreviewingOldVersion = false,
    previewingVersion,
    onclick,
    class: className = ''
  }: Props = $props();

  const isClickable = $derived(!!onclick);
</script>

<button
  type="button"
  class="btn-pill btn-pill-ghost version-badge {className}"
  class:clickable={isClickable}
  class:unsaved={hasUnsavedChanges}
  class:previewing={isPreviewingOldVersion}
  onclick={onclick}
  disabled={!isClickable}
>
  {#if isPreviewingOldVersion && previewingVersion}
    <Eye size={14} strokeWidth={2} class="icon" />
    <span class="label">Viewing v{previewingVersion}</span>
  {:else}
    <span class="version">v{currentVersion}</span>
    <span class="separator">•</span>
    {#if hasUnsavedChanges}
      <Circle size={8} fill="currentColor" class="dot unsaved" />
      <span class="status">Unsaved</span>
    {:else}
      <Check size={14} strokeWidth={2} class="icon saved" />
      <span class="status">Saved</span>
    {/if}
  {/if}
</button>

<style>
  .version-badge {
    display: inline-flex;
    align-items: center;
    gap: 6px;
    padding: 6px 10px;
    font-size: 13px;
    font-weight: 500;
    color: var(--bos-v2-text-secondary);
    background: var(--bos-v2-layer-background-secondary);
    border: 1px solid var(--bos-border-color);
    border-radius: 6px;
    font-family: inherit;
    cursor: default;
    transition: all 150ms ease;
  }

  .version-badge.clickable {
    cursor: pointer;
  }

  .version-badge.clickable:hover {
    background: var(--bos-v2-layer-background-tertiary);
    border-color: var(--bos-v2-layer-insideBorder-border);
  }

  .version-badge.unsaved {
    color: var(--bos-status-warning);
    background: var(--bos-status-warning-bg);
    border-color: color-mix(in srgb, var(--bos-status-warning) 40%, transparent);
  }

  :global(.dark) .version-badge.unsaved {
    color: var(--bos-status-warning);
    background: color-mix(in srgb, var(--bos-status-warning) 10%, transparent);
    border-color: color-mix(in srgb, var(--bos-status-warning) 30%, transparent);
  }

  .version-badge.previewing {
    color: var(--bos-nav-active);
    background: color-mix(in srgb, var(--bos-nav-active) 8%, var(--dbg));
    border-color: color-mix(in srgb, var(--bos-nav-active) 30%, transparent);
  }

  :global(.dark) .version-badge.previewing {
    color: var(--bos-nav-active);
    background: color-mix(in srgb, var(--bos-nav-active) 10%, transparent);
    border-color: color-mix(in srgb, var(--bos-nav-active) 30%, transparent);
  }

  .version {
    font-weight: 600;
    color: var(--bos-v2-text-primary);
  }

  .separator {
    color: var(--bos-border-color);
  }

  .icon {
    flex-shrink: 0;
  }

  .icon.saved {
    color: var(--bos-status-success);
  }

  .dot.unsaved {
    color: var(--bos-status-warning);
  }

  .status {
    font-weight: 500;
  }

  .label {
    font-weight: 500;
  }
</style>
