<!--
  RestoreConfirmDialog.svelte

  Confirmation dialog before restoring to a previous version.

  Key messaging:
  - Creates a NEW version (non-destructive)
  - Copies the old config to the new version
  - Previous versions remain in history
  - Data (records) won't change
-->
<script lang="ts">
  import { RotateCcw, Info, X } from 'lucide-svelte';
  import type { Version } from '$lib/types/versions';

  interface Props {
    version: Version;
    currentVersion: number;
    isOpen: boolean;
    isRestoring?: boolean;
    onClose: () => void;
    onConfirm: () => void;
  }

  let {
    version,
    currentVersion,
    isOpen,
    isRestoring = false,
    onClose,
    onConfirm
  }: Props = $props();

  const newVersionNumber = $derived(currentVersion + 1);

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === 'Escape' && !isRestoring) {
      onClose();
    }
  }

  function handleBackdropClick(e: MouseEvent) {
    if ((e.target as HTMLElement).classList.contains('dialog-backdrop') && !isRestoring) {
      onClose();
    }
  }
</script>

{#if isOpen}
  <!-- svelte-ignore a11y_no_noninteractive_element_interactions -->
  <div
    class="dialog-backdrop"
    role="dialog"
    aria-modal="true"
    aria-labelledby="dialog-title"
    onclick={handleBackdropClick}
    onkeydown={handleKeydown}
  >
    <div class="dialog">
      <!-- Header -->
      <header class="dialog-header">
        <div class="dialog-icon">
          <RotateCcw size={20} strokeWidth={2} />
        </div>
        <div>
          <h2 id="dialog-title" class="dialog-title">Restore to v{version.versionNumber}?</h2>
          {#if version.label}
            <p class="dialog-subtitle">"{version.label}"</p>
          {/if}
        </div>
        <button
          type="button"
          class="btn-pill btn-pill-ghost close-btn"
          onclick={onClose}
          disabled={isRestoring}
          aria-label="Close dialog"
        >
          <X size={18} strokeWidth={2} />
        </button>
      </header>

      <!-- Content -->
      <div class="dialog-content">
        <p class="description">This will:</p>

        <ul class="checklist">
          <li>
            <span class="check">✓</span>
            <span>Create <strong>v{newVersionNumber}</strong> as a copy of v{version.versionNumber}</span>
          </li>
          <li>
            <span class="check">✓</span>
            <span>Make v{version.versionNumber}'s config the current state</span>
          </li>
          <li>
            <span class="check">✓</span>
            <span>Keep v{currentVersion} in history (you can go back)</span>
          </li>
        </ul>

        <div class="info-box">
          <Info size={16} strokeWidth={2} class="info-icon" />
          <p>Your data (records) won't change — only the app configuration will be restored.</p>
        </div>
      </div>

      <!-- Footer -->
      <footer class="dialog-footer">
        <button
          type="button"
          class="btn-pill btn-pill-ghost btn btn-secondary"
          onclick={onClose}
          disabled={isRestoring}
        >
          Cancel
        </button>
        <button
          type="button"
          class="btn-pill btn-pill-ghost btn btn-primary"
          onclick={onConfirm}
          disabled={isRestoring}
        >
          {#if isRestoring}
            <span class="spinner"></span>
            Restoring...
          {:else}
            <RotateCcw size={14} strokeWidth={2} />
            Restore to v{version.versionNumber}
          {/if}
        </button>
      </footer>
    </div>
  </div>
{/if}

<style>
  .dialog-backdrop {
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

  :global(.dark) .dialog-backdrop {
    background: rgba(0, 0, 0, 0.6);
  }

  @keyframes fadeIn {
    from { opacity: 0; }
    to { opacity: 1; }
  }

  .dialog {
    width: 100%;
    max-width: 420px;
    background: var(--bos-v2-layer-background-primary);
    border-radius: 12px;
    box-shadow: 0 8px 24px rgba(0, 0, 0, 0.15);
    animation: slideIn 200ms ease-out;
  }

  :global(.dark) .dialog {
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
  .dialog-header {
    display: flex;
    align-items: flex-start;
    gap: 12px;
    padding: 20px 20px 0;
  }

  .dialog-icon {
    width: 40px;
    height: 40px;
    display: flex;
    align-items: center;
    justify-content: center;
    background: var(--bos-tag-purple);
    color: var(--bos-nav-active);
    border-radius: 10px;
    flex-shrink: 0;
  }

  .dialog-title {
    font-size: 16px;
    font-weight: 600;
    color: var(--bos-v2-text-primary);
    margin: 0;
  }

  .dialog-subtitle {
    font-size: 13px;
    color: var(--bos-v2-text-secondary);
    margin: 2px 0 0 0;
  }

  .close-btn {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 28px;
    height: 28px;
    margin-left: auto;
    color: var(--bos-v2-text-secondary);
    background: transparent;
    border: none;
    border-radius: 6px;
    cursor: pointer;
    transition: all 150ms ease;
  }

  .close-btn:hover:not(:disabled) {
    color: var(--bos-v2-text-primary);
    background: var(--bos-v2-layer-background-secondary);
  }

  .close-btn:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  /* Content */
  .dialog-content {
    padding: 20px;
  }

  .description {
    font-size: 14px;
    color: var(--bos-v2-text-primary);
    margin: 0 0 12px 0;
  }

  .checklist {
    list-style: none;
    margin: 0 0 16px 0;
    padding: 0;
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .checklist li {
    display: flex;
    align-items: flex-start;
    gap: 8px;
    font-size: 13px;
    color: var(--bos-v2-text-primary);
  }

  .check {
    color: var(--bos-status-success);
    font-weight: 600;
    flex-shrink: 0;
  }

  .checklist strong {
    color: var(--bos-v2-text-primary);
  }

  .info-box {
    display: flex;
    gap: 10px;
    padding: 12px;
    background: color-mix(in srgb, var(--bos-status-success) 8%, transparent);
    border-radius: 8px;
  }

  .info-box :global(.info-icon) {
    color: var(--bos-status-success);
    flex-shrink: 0;
    margin-top: 1px;
  }

  .info-box p {
    font-size: 13px;
    color: var(--bos-v2-text-primary);
    margin: 0;
  }

  /* Footer */
  .dialog-footer {
    display: flex;
    justify-content: flex-end;
    gap: 8px;
    padding: 16px 20px;
    border-top: 1px solid var(--bos-border-color);
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

  .btn:disabled {
    opacity: 0.6;
    cursor: not-allowed;
  }

  .btn-secondary {
    color: var(--bos-v2-text-primary);
    background: var(--bos-v2-layer-background-primary);
    border: 1px solid var(--bos-border-color);
  }

  .btn-secondary:hover:not(:disabled) {
    background: var(--bos-v2-layer-background-secondary);
    border-color: var(--bos-v2-layer-insideBorder-border);
  }

  .btn-primary {
    color: var(--bos-surface-on-color);
    background: var(--bos-nav-active);
    border: 1px solid var(--bos-nav-active);
  }

  .btn-primary:hover:not(:disabled) {
    filter: brightness(0.9);
  }

  .spinner {
    width: 14px;
    height: 14px;
    border: 2px solid color-mix(in srgb, var(--bos-surface-on-color) 30%, transparent);
    border-top-color: var(--bos-surface-on-color);
    border-radius: 50%;
    animation: spin 600ms linear infinite;
  }

  @keyframes spin {
    to { transform: rotate(360deg); }
  }
</style>
