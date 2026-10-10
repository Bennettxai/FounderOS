<script lang="ts">
  import { createMemory, type Memory } from '$lib/api';

  // ── Props ────────────────────────────────────────────────────────────────
  interface Props {
    workspace: string | null;
    onClose: () => void;
    onCreated: (m: Memory) => void;
  }
  let { workspace, onClose, onCreated }: Props = $props();

  // ── Form state ───────────────────────────────────────────────────────────
  let content = $state('');
  let isStatic = $state(false);
  let audience = $state('default');
  let citationUri = $state('');

  // Metadata key-value pairs
  type KVPair = { key: string; value: string };
  let metaPairs = $state<KVPair[]>([]);

  // ── UI state ─────────────────────────────────────────────────────────────
  let submitting = $state(false);
  let error = $state<string | null>(null);

  // ── Helpers ──────────────────────────────────────────────────────────────
  function addPair() {
    metaPairs = [...metaPairs, { key: '', value: '' }];
  }

  function removePair(i: number) {
    metaPairs = metaPairs.filter((_, idx) => idx !== i);
  }

  function updatePairKey(i: number, key: string) {
    metaPairs = metaPairs.map((p, idx) => (idx === i ? { ...p, key } : p));
  }

  function updatePairValue(i: number, value: string) {
    metaPairs = metaPairs.map((p, idx) => (idx === i ? { ...p, value } : p));
  }

  function buildMetadata(): Record<string, unknown> | undefined {
    const valid = metaPairs.filter((p) => p.key.trim());
    if (valid.length === 0) return undefined;
    return Object.fromEntries(valid.map((p) => [p.key.trim(), p.value]));
  }

  // ── Submit ───────────────────────────────────────────────────────────────
  async function submit(e: Event) {
    e.preventDefault();
    if (!content.trim()) return;
    submitting = true;
    error = null;
    try {
      const m = await createMemory({
        content: content.trim(),
        workspace: workspace ?? undefined,
        isStatic,
        audience: audience || undefined,
        citationUri: citationUri.trim() || undefined,
        metadata: buildMetadata(),
      });
      onCreated(m);
    } catch (err) {
      error = (err as Error).message;
    } finally {
      submitting = false;
    }
  }

  // Close on backdrop click
  function onBackdrop(e: MouseEvent) {
    if (e.target === e.currentTarget) onClose();
  }

  // Close on Escape
  function onKeydown(e: KeyboardEvent) {
    if (e.key === 'Escape') onClose();
  }
</script>

<svelte:window onkeydown={onKeydown} />

<!-- svelte-ignore a11y_click_events_have_key_events -->
<!-- svelte-ignore a11y_no_static_element_interactions -->
<div class="amm__backdrop" onclick={onBackdrop} role="dialog" aria-modal="true" aria-label="Add memory" tabindex="-1">
  <div class="amm__panel card">
    <header class="amm__header">
      <h2 class="amm__title">Add Memory</h2>
      <button class="amm__close" onclick={onClose} aria-label="Close">×</button>
    </header>

    {#if error}
      <p class="amm__error error">{error}</p>
    {/if}

    <form class="amm__form" onsubmit={submit}>
      <!-- Content -->
      <label class="amm__label">
        Content <span class="amm__required">*</span>
        <textarea
          class="amm__textarea"
          bind:value={content}
          rows={5}
          placeholder="Enter the memory content…"
          required
          disabled={submitting}
          aria-required="true"
        ></textarea>
      </label>

      <!-- Row: audience + is_static -->
      <div class="amm__row">
        <label class="amm__label amm__label--inline">
          Audience
          <select class="amm__select" bind:value={audience} disabled={submitting}>
            <option value="default">default</option>
            <option value="sales">sales</option>
            <option value="legal">legal</option>
            <option value="exec">exec</option>
            <option value="engineering">engineering</option>
          </select>
        </label>

        <label class="amm__label amm__label--check">
          <input
            type="checkbox"
            bind:checked={isStatic}
            disabled={submitting}
          />
          <span>Static (pinned)</span>
        </label>
      </div>

      <!-- Citation URI -->
      <label class="amm__label">
        Citation URI <span class="amm__optional">(optional)</span>
        <input
          type="text"
          class="amm__input"
          bind:value={citationUri}
          placeholder="https://example.com/source"
          disabled={submitting}
        />
      </label>

      <!-- Metadata key-value -->
      <div class="amm__meta-section">
        <div class="amm__meta-header">
          <span class="eyebrow">Metadata</span>
          <span class="amm__optional">(optional)</span>
          <button type="button" class="btn btn--sm btn--ghost amm__meta-add" onclick={addPair} disabled={submitting}>
            + Add field
          </button>
        </div>

        {#each metaPairs as pair, i}
          <div class="amm__kv">
            <input
              class="amm__input amm__kv-key"
              type="text"
              placeholder="key"
              value={pair.key}
              oninput={(e) => updatePairKey(i, (e.currentTarget as HTMLInputElement).value)}
              disabled={submitting}
              aria-label="Metadata key {i + 1}"
            />
            <input
              class="amm__input amm__kv-val"
              type="text"
              placeholder="value"
              value={pair.value}
              oninput={(e) => updatePairValue(i, (e.currentTarget as HTMLInputElement).value)}
              disabled={submitting}
              aria-label="Metadata value {i + 1}"
            />
            <button
              type="button"
              class="amm__kv-remove"
              onclick={() => removePair(i)}
              disabled={submitting}
              aria-label="Remove field {i + 1}"
            >×</button>
          </div>
        {/each}
      </div>

      <!-- Actions -->
      <div class="amm__actions">
        <button type="button" class="btn btn--ghost btn--sm" onclick={onClose} disabled={submitting}>
          Cancel
        </button>
        <button
          type="submit"
          class="btn btn--primary btn--sm"
          disabled={submitting || !content.trim()}
        >
          {submitting ? 'Creating…' : 'Create memory'}
        </button>
      </div>
    </form>
  </div>
</div>

<style>
  .amm__backdrop {
    position: fixed;
    inset: 0;
    z-index: 100;
    background: rgba(0, 0, 0, 0.55);
    backdrop-filter: blur(4px);
    display: flex;
    align-items: center;
    justify-content: center;
    padding: 1rem;
  }

  .amm__panel {
    width: 100%;
    max-width: 520px;
    display: flex;
    flex-direction: column;
    gap: 0;
    padding: 0;
    background: var(--bg-elevated);
    border: 1px solid var(--border);
    border-radius: 14px;
    box-shadow: 0 24px 60px -10px rgba(0,0,0,0.65);
    overflow: hidden;
  }

  .amm__header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 1rem 1.25rem 0.75rem;
    border-bottom: 1px solid var(--border-soft);
  }

  .amm__title {
    margin: 0;
    font-size: 1rem;
    font-weight: 600;
    color: var(--text);
  }

  .amm__close {
    background: none;
    border: none;
    color: var(--text-muted);
    font-size: 1.4rem;
    cursor: pointer;
    padding: 0 0.3rem;
    line-height: 1;
    transition: color 0.1s;
  }
  .amm__close:hover { color: var(--text); }

  .amm__error {
    margin: 0.75rem 1.25rem 0;
    padding: 0.5rem 0.75rem;
  }

  .amm__form {
    display: flex;
    flex-direction: column;
    gap: 0.85rem;
    padding: 1rem 1.25rem 1.25rem;
  }

  .amm__label {
    display: flex;
    flex-direction: column;
    gap: 0.3rem;
    font-size: 0.72rem;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.07em;
    color: var(--text-muted);
  }

  .amm__label--inline {
    flex-direction: row;
    align-items: center;
    gap: 0.5rem;
    text-transform: none;
    font-weight: 500;
    font-size: 0.82rem;
  }

  .amm__label--check {
    flex-direction: row;
    align-items: center;
    gap: 0.45rem;
    text-transform: none;
    font-size: 0.82rem;
    font-weight: 500;
    cursor: pointer;
  }
  .amm__label--check input { cursor: pointer; }

  .amm__required { color: var(--bad); }
  .amm__optional {
    color: var(--text-subtle);
    font-size: 0.68rem;
    font-weight: 400;
    text-transform: none;
    letter-spacing: normal;
  }

  .amm__textarea,
  .amm__input {
    background: var(--bg);
    border: 1px solid var(--border);
    border-radius: var(--r-sm);
    color: var(--text);
    padding: 0.45rem 0.65rem;
    font: inherit;
    font-size: 0.875rem;
    resize: vertical;
    transition: border-color 0.12s, box-shadow 0.12s;
  }
  .amm__textarea:focus,
  .amm__input:focus {
    outline: none;
    border-color: var(--accent);
    box-shadow: 0 0 0 3px var(--accent-soft);
  }
  .amm__textarea:disabled,
  .amm__input:disabled {
    opacity: 0.6;
  }

  .amm__select {
    background: var(--bg);
    border: 1px solid var(--border);
    border-radius: var(--r-sm);
    color: var(--text);
    padding: 0.3rem 0.5rem;
    font: inherit;
    font-size: 0.82rem;
    cursor: pointer;
  }
  .amm__select:focus {
    outline: none;
    border-color: var(--accent);
  }

  .amm__row {
    display: flex;
    gap: 1rem;
    align-items: center;
    flex-wrap: wrap;
  }

  /* Metadata section */
  .amm__meta-section {
    display: flex;
    flex-direction: column;
    gap: 0.4rem;
  }

  .amm__meta-header {
    display: flex;
    align-items: center;
    gap: 0.5rem;
  }

  .amm__meta-add {
    margin-left: auto;
  }

  .amm__kv {
    display: flex;
    gap: 0.4rem;
    align-items: center;
  }

  .amm__kv-key {
    flex: 0 0 40%;
  }

  .amm__kv-val {
    flex: 1;
  }

  .amm__kv-remove {
    background: none;
    border: 1px solid var(--border);
    color: var(--text-muted);
    border-radius: var(--r-sm);
    width: 26px;
    height: 26px;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    cursor: pointer;
    font-size: 1rem;
    flex-shrink: 0;
    transition: border-color 0.1s, color 0.1s;
  }
  .amm__kv-remove:hover {
    border-color: var(--bad);
    color: var(--bad);
  }

  .amm__actions {
    display: flex;
    justify-content: flex-end;
    gap: 0.5rem;
    padding-top: 0.25rem;
    border-top: 1px solid var(--border-soft);
    margin-top: 0.25rem;
  }
</style>
