<script lang="ts">
  import { categoryLabels } from '$lib/stores/agents';

  interface Props {
    open: boolean;
    saving: boolean;
    name: string;
    handle: string;
    desc: string;
    category: string;
    model: string;
    prompt: string;
    active: boolean;
    error: string | null;
    categories: readonly string[];
    onclose: () => void;
    onsubmit: () => void;
    onNameInput: (e: Event) => void;
    onHandleInput: (e: Event) => void;
  }

  let {
    open,
    saving,
    name,
    handle,
    desc = $bindable(),
    category = $bindable(),
    model = $bindable(),
    prompt = $bindable(),
    active = $bindable(),
    error,
    categories,
    onclose,
    onsubmit,
    onNameInput,
    onHandleInput,
  }: Props = $props();
</script>

{#if open}
  <div
    class="ag-backdrop"
    onclick={onclose}
    onkeydown={(e) => e.key === 'Escape' && onclose()}
    role="presentation"
    aria-hidden="true"
  >
    <div
      class="ag-create-modal"
      onclick={(e) => e.stopPropagation()}
      onkeydown={(e) => e.stopPropagation()}
      role="dialog"
      aria-modal="true"
      aria-labelledby="ag-create-title"
      tabindex="-1"
    >
      <div class="ag-cm__header">
        <div class="ag-cm__header-icon" aria-hidden="true">
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <rect x="3" y="8" width="18" height="12" rx="3"/>
            <path d="M9 8V6a3 3 0 0 1 6 0v2"/>
            <circle cx="9" cy="13" r="1.5" fill="currentColor" stroke="none"/>
            <circle cx="15" cy="13" r="1.5" fill="currentColor" stroke="none"/>
            <path d="M9 17h6"/>
          </svg>
        </div>
        <div>
          <h2 class="ag-cm__title" id="ag-create-title">New Agent</h2>
          <p class="ag-cm__subtitle">Configure your custom AI agent</p>
        </div>
        <button class="ag-cm__close" onclick={onclose} aria-label="Close">
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" aria-hidden="true">
            <path d="M18 6 6 18M6 6l12 12"/>
          </svg>
        </button>
      </div>

      <div class="ag-cm__body">
        <div class="ag-cm__row">
          <div class="ag-cm__field ag-cm__field--grow">
            <label class="ag-cm__label" for="ag-create-name">Display Name <span class="ag-cm__req">*</span></label>
            <input
              id="ag-create-name"
              type="text"
              class="ag-cm__input"
              value={name}
              oninput={onNameInput}
              placeholder="e.g. Sales Assistant"
              autocomplete="off"
            />
          </div>
          <div class="ag-cm__field">
            <label class="ag-cm__label" for="ag-create-handle">Handle <span class="ag-cm__req">*</span></label>
            <div class="ag-cm__input-prefix">
              <span class="ag-cm__prefix">@</span>
              <input
                id="ag-create-handle"
                type="text"
                class="ag-cm__input ag-cm__input--prefixed"
                value={handle}
                oninput={onHandleInput}
                placeholder="sales_assistant"
                autocomplete="off"
                spellcheck="false"
              />
            </div>
          </div>
        </div>

        <div class="ag-cm__field">
          <label class="ag-cm__label" for="ag-create-desc">Description</label>
          <input
            id="ag-create-desc"
            type="text"
            class="ag-cm__input"
            bind:value={desc}
            placeholder="What does this agent do?"
            autocomplete="off"
          />
        </div>

        <div class="ag-cm__row">
          <div class="ag-cm__field ag-cm__field--grow">
            <label class="ag-cm__label" for="ag-create-category">Category</label>
            <select id="ag-create-category" class="ag-cm__select" bind:value={category}>
              {#each categories as cat}
                <option value={cat}>{categoryLabels[cat] || cat}</option>
              {/each}
            </select>
          </div>
          <div class="ag-cm__field ag-cm__field--grow">
            <label class="ag-cm__label" for="ag-create-model">Model</label>
            <select id="ag-create-model" class="ag-cm__select" bind:value={model}>
              <option value="claude-sonnet-4-6">Claude Sonnet 4.6</option>
              <option value="claude-opus-4-6">Claude Opus 4.6</option>
              <option value="claude-haiku-4-5-20251001">Claude Haiku 4.5</option>
            </select>
          </div>
        </div>

        <div class="ag-cm__field">
          <label class="ag-cm__label" for="ag-create-prompt">System Prompt</label>
          <textarea
            id="ag-create-prompt"
            class="ag-cm__textarea"
            bind:value={prompt}
            placeholder="You are a helpful assistant that..."
            rows="5"
          ></textarea>
          <p class="ag-cm__hint">Defines how your agent behaves. You can refine this later.</p>
        </div>

        <div class="ag-cm__toggle-row">
          <div>
            <p class="ag-cm__toggle-label">Active</p>
            <p class="ag-cm__toggle-hint">Inactive agents won't appear in chat</p>
          </div>
          <button
            class="ag-cm__toggle"
            class:ag-cm__toggle--on={active}
            onclick={() => active = !active}
            role="switch"
            aria-checked={active}
            aria-label="Toggle active status"
          >
            <span class="ag-cm__toggle-thumb"></span>
          </button>
        </div>

        {#if error}
          <p class="ag-cm__error" role="alert">{error}</p>
        {/if}
      </div>

      <div class="ag-cm__footer">
        <button onclick={onclose} class="btn-pill btn-pill-ghost btn-pill-sm" disabled={saving}>
          Cancel
        </button>
        <button onclick={onsubmit} class="btn-cta" disabled={saving || !name.trim() || !handle.trim()}>
          {saving ? 'Creating…' : 'Create Agent'}
        </button>
      </div>
    </div>
  </div>
{/if}

<style>
  .ag-backdrop {
    position: fixed;
    inset: 0;
    z-index: 50;
    display: flex;
    align-items: center;
    justify-content: center;
    background: rgba(0, 0, 0, 0.45);
    backdrop-filter: blur(2px);
  }

  .ag-create-modal {
    background: var(--dbg, #fff);
    border: 1px solid var(--dbd, #e0e0e0);
    border-radius: 16px;
    box-shadow: 0 20px 60px rgba(0,0,0,0.16), 0 4px 16px rgba(0,0,0,0.06);
    width: 560px;
    max-width: calc(100vw - 2rem);
    max-height: calc(100vh - 4rem);
    display: flex;
    flex-direction: column;
    overflow: hidden;
    animation: ag-cm-modal-in 0.18s cubic-bezier(0.16, 1, 0.3, 1);
  }
  @keyframes ag-cm-modal-in {
    from { opacity: 0; transform: scale(0.96) translateY(4px); }
    to   { opacity: 1; transform: scale(1)    translateY(0); }
  }

  .ag-cm__header {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    padding: 1.25rem 1.375rem 1rem;
    border-bottom: 1px solid var(--dbd2, #f0f0f0);
  }
  .ag-cm__header-icon {
    width: 2rem;
    height: 2rem;
    border-radius: 8px;
    background: var(--dbg2, #f5f5f5);
    border: 1px solid var(--dbd, #e0e0e0);
    display: flex;
    align-items: center;
    justify-content: center;
    color: var(--dt2, #555);
    flex-shrink: 0;
  }
  .ag-cm__title {
    font-size: 1rem;
    font-weight: 700;
    color: var(--dt, #111);
    margin: 0;
    letter-spacing: -0.01em;
  }
  .ag-cm__subtitle {
    font-size: 0.75rem;
    color: var(--dt3, #888);
    margin: 0.1rem 0 0;
  }
  .ag-cm__close {
    margin-left: auto;
    width: 28px;
    height: 28px;
    border-radius: 7px;
    border: none;
    background: transparent;
    color: var(--dt3, #888);
    cursor: pointer;
    display: flex;
    align-items: center;
    justify-content: center;
    transition: background 0.12s, color 0.12s;
    flex-shrink: 0;
  }
  .ag-cm__close:hover { background: var(--dbg2, #f5f5f5); color: var(--dt, #111); }

  .ag-cm__body {
    padding: 1.125rem 1.375rem;
    display: flex;
    flex-direction: column;
    gap: 1rem;
    overflow-y: auto;
    flex: 1;
  }
  .ag-cm__row {
    display: flex;
    gap: 0.75rem;
  }
  .ag-cm__field {
    display: flex;
    flex-direction: column;
    gap: 0.3rem;
  }
  .ag-cm__field--grow { flex: 1; min-width: 0; }
  .ag-cm__label {
    font-size: 0.75rem;
    font-weight: 600;
    color: var(--dt2, #444);
    letter-spacing: 0.01em;
  }
  .ag-cm__req { color: var(--bos-status-error, #ef4444); }
  .ag-cm__input,
  .ag-cm__select,
  .ag-cm__textarea {
    padding: 0.5625rem 0.75rem;
    font-size: 0.875rem;
    font-family: inherit;
    border: 1px solid var(--dbd, #e0e0e0);
    border-radius: 8px;
    background: var(--dbg, #fff);
    color: var(--dt, #111);
    outline: none;
    transition: border-color 0.15s, box-shadow 0.15s;
    width: 100%;
    box-sizing: border-box;
  }
  .ag-cm__input::placeholder,
  .ag-cm__textarea::placeholder { color: var(--dt4, #bbb); }
  .ag-cm__input:focus,
  .ag-cm__select:focus,
  .ag-cm__textarea:focus {
    border-color: var(--dt2, #555);
    box-shadow: 0 0 0 3px rgba(0,0,0,0.06);
  }
  .ag-cm__select {
    appearance: none;
    -webkit-appearance: none;
    background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='12' height='12' viewBox='0 0 24 24' fill='none' stroke='%23888888' stroke-width='2.5' stroke-linecap='round' stroke-linejoin='round'%3E%3Cpolyline points='6 9 12 15 18 9'/%3E%3C/svg%3E");
    background-repeat: no-repeat;
    background-position: right 0.625rem center;
    padding-right: 2rem;
    cursor: pointer;
  }
  .ag-cm__textarea {
    resize: vertical;
    min-height: 100px;
    line-height: 1.55;
  }
  .ag-cm__hint {
    font-size: 0.71875rem;
    color: var(--dt4, #bbb);
    margin: 0;
  }

  .ag-cm__input-prefix {
    position: relative;
    display: flex;
    align-items: center;
  }
  .ag-cm__prefix {
    position: absolute;
    left: 0.75rem;
    font-size: 0.875rem;
    color: var(--dt3, #888);
    pointer-events: none;
    user-select: none;
  }
  .ag-cm__input--prefixed { padding-left: 1.375rem; }

  .ag-cm__toggle-row {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 0.75rem 0.875rem;
    background: var(--dbg2, #f9f9f9);
    border: 1px solid var(--dbd2, #f0f0f0);
    border-radius: 8px;
  }
  .ag-cm__toggle-label {
    font-size: 0.8125rem;
    font-weight: 600;
    color: var(--dt, #111);
    margin: 0;
  }
  .ag-cm__toggle-hint {
    font-size: 0.71875rem;
    color: var(--dt3, #888);
    margin: 0.1rem 0 0;
  }
  .ag-cm__toggle {
    width: 36px;
    height: 20px;
    border-radius: 9999px;
    border: none;
    background: var(--dbd, #ddd);
    cursor: pointer;
    position: relative;
    transition: background 0.18s;
    flex-shrink: 0;
  }
  .ag-cm__toggle--on { background: var(--dt, #111); }
  .ag-cm__toggle-thumb {
    position: absolute;
    top: 2px;
    left: 2px;
    width: 16px;
    height: 16px;
    border-radius: 50%;
    background: #fff;
    box-shadow: 0 1px 3px rgba(0,0,0,0.18);
    transition: transform 0.18s;
  }
  .ag-cm__toggle--on .ag-cm__toggle-thumb { transform: translateX(16px); }

  .ag-cm__error {
    font-size: 0.8125rem;
    color: var(--bos-status-error, #ef4444);
    background: rgba(239, 68, 68, 0.06);
    border: 1px solid rgba(239, 68, 68, 0.18);
    border-radius: 7px;
    padding: 0.5rem 0.75rem;
    margin: 0;
  }

  .ag-cm__footer {
    display: flex;
    align-items: center;
    justify-content: flex-end;
    gap: 0.5rem;
    padding: 0.875rem 1.375rem;
    border-top: 1px solid var(--dbd2, #f0f0f0);
    background: var(--dbg2, #fafafa);
  }
  .ag-cm__footer :global(.btn-cta:disabled),
  .ag-cm__footer :global(.btn-pill:disabled) {
    opacity: 0.45;
    cursor: not-allowed;
    pointer-events: none;
  }
</style>
