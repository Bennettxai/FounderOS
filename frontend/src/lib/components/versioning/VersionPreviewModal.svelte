<!--
  VersionPreviewModal.svelte

  Modal showing a preview of the app at a specific version.

  Features:
  - Shows version details (number, label, trigger, time)
  - Displays config snapshot in a readable format
  - Restore button to apply this version
  - Close/dismiss actions
-->
<script lang="ts">
  import { X, RotateCcw, Clock, Sparkles, Pencil, Camera, Save, Eye } from 'lucide-svelte';
  import type { Version, VersionTrigger } from '$lib/types/versions';
  import { formatRelativeTime, getTriggerLabel } from '$lib/types/versions';

  interface Props {
    version: Version;
    isOpen: boolean;
    onClose: () => void;
    onRestore: (version: Version) => void;
  }

  let {
    version,
    isOpen,
    onClose,
    onRestore
  }: Props = $props();

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
    }
  }

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === 'Escape') {
      onClose();
    }
  }

  function handleBackdropClick(e: MouseEvent) {
    if ((e.target as HTMLElement).classList.contains('modal-backdrop')) {
      onClose();
    }
  }

  // Format config for display
  function formatConfig(config: Record<string, unknown>): string {
    return JSON.stringify(config, null, 2);
  }

  const TriggerIcon = $derived(getTriggerIcon(version.trigger));
</script>

{#if isOpen}
  <!-- svelte-ignore a11y_no_noninteractive_element_interactions -->
  <div
    class="modal-backdrop"
    role="dialog"
    aria-modal="true"
    aria-labelledby="modal-title"
    onclick={handleBackdropClick}
    onkeydown={handleKeydown}
  >
    <div class="modal">
      <!-- Header -->
      <header class="modal-header">
        <div class="header-left">
          <div class="preview-badge">
            <Eye size={14} strokeWidth={2} />
            Preview
          </div>
          <h2 id="modal-title" class="modal-title">Version {version.versionNumber}</h2>
          {#if version.label}
            <span class="version-label">{version.label}</span>
          {/if}
        </div>
        <button
          type="button"
          class="btn-pill btn-pill-ghost close-btn"
          onclick={onClose}
          aria-label="Close preview"
        >
          <X size={20} strokeWidth={2} />
        </button>
      </header>

      <!-- Version Meta -->
      <div class="version-meta">
        <div class="meta-item">
          <TriggerIcon size={14} strokeWidth={2} />
          <span>{getTriggerLabel(version.trigger)}</span>
        </div>
        <div class="meta-divider"></div>
        <div class="meta-item">
          <Clock size={14} strokeWidth={2} />
          <span>{formatRelativeTime(version.createdAt)}</span>
        </div>
        {#if version.createdByName}
          <div class="meta-divider"></div>
          <div class="meta-item">
            <span>by {version.createdByName}</span>
          </div>
        {/if}
      </div>

      {#if version.prompt}
        <div class="prompt-section">
          <span class="prompt-label">Prompt</span>
          <p class="prompt-text">"{version.prompt}"</p>
        </div>
      {/if}

      <!-- Content -->
      <div class="modal-content">
        <div class="config-section">
          <div class="config-header">
            <h3 class="config-title">Configuration Snapshot</h3>
            <span class="config-hint">Read-only preview of saved state</span>
          </div>
          <pre class="config-code">{formatConfig(version.configSnapshot)}</pre>
        </div>
      </div>

      <!-- Footer -->
      <footer class="modal-footer">
        <button
          type="button"
          class="btn-pill btn-pill-ghost btn btn-secondary"
          onclick={onClose}
        >
          Close
        </button>
        {#if !version.isCurrent}
          <button
            type="button"
            class="btn-pill btn-pill-ghost btn btn-primary"
            onclick={() => onRestore(version)}
          >
            <RotateCcw size={14} strokeWidth={2} />
            Restore this version
          </button>
        {:else}
          <span class="current-indicator">This is the current version</span>
        {/if}
      </footer>
    </div>
  </div>
{/if}

<style>
  .modal-backdrop {
    position: fixed;
    inset: 0;
    display: flex;
    align-items: center;
    justify-content: center;
    background: rgba(0, 0, 0, 0.4);
    z-index: 200;
    padding: 20px;
    animation: fadeIn 150ms ease-out;
  }

  :global(.dark) .modal-backdrop {
    background: rgba(0, 0, 0, 0.6);
  }

  @keyframes fadeIn {
    from { opacity: 0; }
    to { opacity: 1; }
  }

  .modal {
    width: 100%;
    max-width: 600px;
    max-height: 80vh;
    background: var(--bos-v2-layer-background-primary);
    border-radius: 12px;
    box-shadow: 0 8px 24px rgba(0, 0, 0, 0.15);
    display: flex;
    flex-direction: column;
    animation: slideIn 200ms ease-out;
  }

  :global(.dark) .modal {
    box-shadow: 0 8px 24px rgba(0, 0, 0, 0.4);
  }

  @keyframes slideIn {
    from {
      opacity: 0;
      transform: scale(0.95) translateY(-10px);
    }
    to {
      opacity: 1;
      transform: scale(1) translateY(0);
    }
  }

  /* Header */
  .modal-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 16px 20px;
    border-bottom: 1px solid var(--bos-border-color);
    flex-shrink: 0;
  }

  .header-left {
    display: flex;
    align-items: center;
    gap: 12px;
  }

  .preview-badge {
    display: inline-flex;
    align-items: center;
    gap: 4px;
    padding: 4px 8px;
    font-size: 11px;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.5px;
    color: var(--bos-nav-active);
    background: var(--bos-tag-purple);
    border-radius: 4px;
  }

  .modal-title {
    font-size: 16px;
    font-weight: 600;
    color: var(--bos-v2-text-primary);
    margin: 0;
  }

  .version-label {
    font-size: 13px;
    color: var(--bos-v2-text-secondary);
  }

  .close-btn {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 32px;
    height: 32px;
    color: var(--bos-v2-text-secondary);
    background: transparent;
    border: none;
    border-radius: 6px;
    cursor: pointer;
    transition: all 150ms ease;
  }

  .close-btn:hover {
    color: var(--bos-v2-text-primary);
    background: var(--bos-v2-layer-background-secondary);
  }

  /* Version Meta */
  .version-meta {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 12px 20px;
    background: var(--bos-v2-layer-background-secondary);
    border-bottom: 1px solid var(--bos-border-color);
  }

  .meta-item {
    display: flex;
    align-items: center;
    gap: 6px;
    font-size: 13px;
    color: var(--bos-v2-text-secondary);
  }

  .meta-divider {
    width: 1px;
    height: 16px;
    background: var(--bos-border-color);
  }

  /* Prompt Section */
  .prompt-section {
    padding: 12px 20px;
    border-bottom: 1px solid var(--bos-border-color);
  }

  .prompt-label {
    font-size: 11px;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.5px;
    color: var(--bos-v2-text-secondary);
    display: block;
    margin-bottom: 4px;
  }

  .prompt-text {
    font-size: 13px;
    font-style: italic;
    color: var(--bos-v2-text-primary);
    margin: 0;
  }

  /* Content */
  .modal-content {
    flex: 1;
    overflow-y: auto;
    padding: 20px;
  }

  .config-section {
    display: flex;
    flex-direction: column;
    gap: 12px;
  }

  .config-header {
    display: flex;
    align-items: baseline;
    justify-content: space-between;
  }

  .config-title {
    font-size: 13px;
    font-weight: 600;
    color: var(--bos-v2-text-primary);
    margin: 0;
  }

  .config-hint {
    font-size: 12px;
    color: var(--bos-v2-text-secondary);
  }

  .config-code {
    padding: 16px;
    font-size: 12px;
    font-family: 'SF Mono', 'Monaco', 'Inconsolata', 'Fira Code', monospace;
    color: var(--bos-v2-text-primary);
    background: var(--bos-v2-layer-background-secondary);
    border: 1px solid var(--bos-border-color);
    border-radius: 8px;
    overflow-x: auto;
    margin: 0;
    white-space: pre-wrap;
    word-break: break-word;
  }

  /* Footer */
  .modal-footer {
    display: flex;
    align-items: center;
    justify-content: flex-end;
    gap: 8px;
    padding: 16px 20px;
    border-top: 1px solid var(--bos-border-color);
    flex-shrink: 0;
  }

  .btn {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    gap: 6px;
    padding: 8px 16px;
    font-size: 13px;
    font-weight: 500;
    border-radius: 6px;
    cursor: pointer;
    font-family: inherit;
    transition: all 150ms ease;
  }

  .btn-secondary {
    color: var(--bos-v2-text-primary);
    background: var(--bos-v2-layer-background-primary);
    border: 1px solid var(--bos-border-color);
  }

  .btn-secondary:hover {
    background: var(--bos-v2-layer-background-secondary);
    border-color: var(--bos-v2-layer-insideBorder-border);
  }

  .btn-primary {
    color: var(--bos-surface-on-color);
    background: var(--bos-nav-active);
    border: 1px solid var(--bos-nav-active);
  }

  .btn-primary:hover {
    filter: brightness(0.9);
  }

  .current-indicator {
    font-size: 13px;
    color: var(--bos-status-success);
    font-weight: 500;
  }

  /* Responsive */
  @media (max-width: 640px) {
    .modal {
      max-height: 90vh;
    }

    .header-left {
      flex-wrap: wrap;
      gap: 8px;
    }

    .version-meta {
      flex-wrap: wrap;
    }
  }
</style>
