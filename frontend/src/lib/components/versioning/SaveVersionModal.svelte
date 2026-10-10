<!--
  SaveVersionModal.svelte

  Modal for creating a manual version snapshot.

  Features:
  - Optional label input
  - Shows what version number will be created
  - Quick save without label option
-->
<script lang="ts">
  import { Save, X } from 'lucide-svelte';

  interface Props {
    appId: string;
    currentVersion: number;
    isOpen: boolean;
    isSaving?: boolean;
    onClose: () => void;
    onSave: (label?: string) => void;
  }

  let {
    appId,
    currentVersion,
    isOpen,
    isSaving = false,
    onClose,
    onSave
  }: Props = $props();

  let label = $state('');
  let inputRef: HTMLInputElement;

  const newVersionNumber = $derived(currentVersion + 1);

  function handleSubmit(e: SubmitEvent) {
    e.preventDefault();
    onSave(label.trim() || undefined);
  }

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === 'Escape' && !isSaving) {
      onClose();
    }
  }

  function handleBackdropClick(e: MouseEvent) {
    if ((e.target as HTMLElement).classList.contains('dialog-backdrop') && !isSaving) {
      onClose();
    }
  }

  // Reset and focus on open
  $effect(() => {
    if (isOpen) {
      label = '';
      // Focus input after animation
      setTimeout(() => inputRef?.focus(), 100);
    }
  });
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
          <Save size={20} strokeWidth={2} />
        </div>
        <div>
          <h2 id="dialog-title" class="dialog-title">Save Version</h2>
          <p class="dialog-subtitle">Create v{newVersionNumber}</p>
        </div>
        <button
          type="button"
          class="btn-pill btn-pill-ghost close-btn"
          onclick={onClose}
          disabled={isSaving}
          aria-label="Close dialog"
        >
          <X size={18} strokeWidth={2} />
        </button>
      </header>

      <!-- Content -->
      <form onsubmit={handleSubmit}>
        <div class="dialog-content">
          <p class="description">
            Save the current state as a new version you can restore later.
          </p>

          <div class="form-group">
            <label for="version-label" class="label">
              Label <span class="optional">(optional)</span>
            </label>
            <input
              bind:this={inputRef}
              bind:value={label}
              type="text"
              id="version-label"
              class="input"
              placeholder="e.g., Pre-demo stable, Before refactor"
              maxlength="100"
              disabled={isSaving}
            />
            <p class="hint">A short description to help you remember this version</p>
          </div>
        </div>

        <!-- Footer -->
        <footer class="dialog-footer">
          <button
            type="button"
            class="btn-pill btn-pill-ghost btn btn-secondary"
            onclick={onClose}
            disabled={isSaving}
          >
            Cancel
          </button>
          <button
            type="submit"
            class="btn-pill btn-pill-ghost btn btn-primary"
            disabled={isSaving}
          >
            {#if isSaving}
              <span class="spinner"></span>
              Saving...
            {:else}
              <Save size={14} strokeWidth={2} />
              Save Version
            {/if}
          </button>
        </footer>
      </form>
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
    max-width: 400px;
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
    color: var(--bos-v2-text-secondary);
    margin: 0 0 16px 0;
  }

  .form-group {
    display: flex;
    flex-direction: column;
    gap: 6px;
  }

  .label {
    font-size: 13px;
    font-weight: 500;
    color: var(--bos-v2-text-primary);
  }

  .optional {
    font-weight: 400;
    color: var(--bos-v2-text-secondary);
  }

  .input {
    width: 100%;
    padding: 10px 12px;
    font-size: 14px;
    color: var(--bos-v2-text-primary);
    background: var(--bos-v2-layer-background-primary);
    border: 1px solid var(--bos-border-color);
    border-radius: 6px;
    font-family: inherit;
    transition: all 150ms ease;
  }

  .input:focus {
    outline: none;
    border-color: var(--bos-nav-active);
    box-shadow: 0 0 0 3px color-mix(in srgb, var(--bos-nav-active) 15%, transparent);
  }

  .input::placeholder {
    color: var(--bos-v2-text-secondary);
  }

  .input:disabled {
    opacity: 0.6;
    cursor: not-allowed;
  }

  .hint {
    font-size: 12px;
    color: var(--bos-v2-text-secondary);
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
