<script lang="ts">
  import { AlertTriangle, X } from 'lucide-svelte';

  interface Props {
    title: string;
    message: string;
    confirmText?: string;
    cancelText?: string;
    onconfirm?: () => void;
    oncancel?: () => void;
  }

  let {
    title,
    message,
    confirmText = 'Delete',
    cancelText = 'Cancel',
    onconfirm,
    oncancel,
  }: Props = $props();

  let confirmInput = $state('');
  const requiredText = 'DELETE';

  function handleConfirm() {
    if (confirmInput === requiredText) {
      onconfirm?.();
    }
  }

  function handleCancel() {
    oncancel?.();
  }

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === 'Escape') {
      handleCancel();
    }
  }
</script>

<svelte:window onkeydown={handleKeydown} />

<div class="bos-modal-overlay" onclick={handleCancel}>
  <div class="bos-modal bos-modal--sm" onclick={(e) => e.stopPropagation()}>
    <div class="bos-modal-header">
      <div style="display:flex;align-items:center;justify-content:center;width:40px;height:40px;background:var(--bos-background-error-color);color:var(--bos-error-color);border-radius:50%;">
        <AlertTriangle class="w-5 h-5" />
      </div>
      <h2>{title}</h2>
      <button class="bos-modal-close" onclick={handleCancel} type="button" aria-label="Close modal">
        <X class="w-5 h-5" />
      </button>
    </div>

    <div class="bos-modal-body">
      <p>{message}</p>

      <div style="display:flex;flex-direction:column;gap:0.5rem;">
        <label class="bos-label" for="confirm-input">
          Type <strong style="font-weight:600;color:var(--bos-error-color);">{requiredText}</strong> to confirm
        </label>
        <input
          id="confirm-input"
          class="bos-input"
          type="text"
          bind:value={confirmInput}
          placeholder={requiredText}
          autocomplete="off"
        />
      </div>
    </div>

    <div class="bos-modal-footer">
      <button
        class="btn-pill btn-pill-ghost btn-pill-sm"
        style="flex:1"
        onclick={handleCancel}
        type="button"
      >
        {cancelText}
      </button>
      <button
        class="btn-pill btn-pill-sm"
        style="flex:1; background: var(--bos-error-color); color: white;"
        onclick={handleConfirm}
        disabled={confirmInput !== requiredText}
        type="button"
      >
        {confirmText}
      </button>
    </div>
  </div>
</div>
