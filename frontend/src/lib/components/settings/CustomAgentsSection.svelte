<script lang="ts">
  import AgentTestSandbox from '$lib/components/settings/AgentTestSandbox.svelte';
  import type { CustomAgent } from '$lib/stores/aiSettings';
  import { apiClient } from '$lib/api';

  interface Props {
    customAgents: CustomAgent[];
    loadingCustomAgents: boolean;
    onAgentUpdated: () => void;
  }

  let { customAgents, loadingCustomAgents, onAgentUpdated }: Props = $props();

  let showNewCustomAgent = $state(false);
  let editingCustomAgent = $state<CustomAgent | null>(null);
  let savingCustomAgent = $state(false);
  let testingAgentId = $state<string | null>(null);
  let newCustomAgent = $state({
    name: '',
    display_name: '',
    description: '',
    system_prompt: '',
    model_preference: '',
    temperature: 0.7,
    category: 'custom'
  });

  async function createCustomAgent() {
    if (!newCustomAgent.name.trim() || !newCustomAgent.system_prompt.trim()) return;
    savingCustomAgent = true;
    try {
      const res = await apiClient.post('/ai/custom-agents', newCustomAgent);
      if (res.ok) {
        showNewCustomAgent = false;
        newCustomAgent = {
          name: '',
          display_name: '',
          description: '',
          system_prompt: '',
          model_preference: '',
          temperature: 0.7,
          category: 'custom'
        };
        onAgentUpdated();
      }
    } catch {
      // silently fail
    } finally {
      savingCustomAgent = false;
    }
  }

  async function deleteCustomAgent(agentId: string) {
    if (!confirm('Delete this custom agent?')) return;
    try {
      const res = await apiClient.delete(`/ai/custom-agents/${agentId}`);
      if (res.ok) {
        onAgentUpdated();
      }
    } catch {
      // silently fail
    }
  }

  async function saveEditedCustomAgent() {
    if (!editingCustomAgent) return;
    savingCustomAgent = true;
    try {
      const res = await apiClient.put(`/ai/custom-agents/${editingCustomAgent.id}`, editingCustomAgent);
      if (res.ok) {
        editingCustomAgent = null;
        onAgentUpdated();
      }
    } catch {
      // silently fail
    } finally {
      savingCustomAgent = false;
    }
  }
</script>

<section class="section">
  <div class="section-header">
    <h2>Custom Agents</h2>
    <button class="btn-pill btn-pill-ghost add-btn" onclick={() => showNewCustomAgent = true}>
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M12 5v14M5 12h14"/></svg>
      New Agent
    </button>
  </div>
  <p class="section-subtitle">Create custom agents you can mention with @name in chat</p>

  {#if showNewCustomAgent}
    <div class="custom-agent-form">
      <div class="form-header">
        <h3>Create Custom Agent</h3>
        <button class="btn-pill btn-pill-ghost close-btn" onclick={() => showNewCustomAgent = false}>x</button>
      </div>
      <div class="form-grid">
        <div class="form-group">
          <label>Agent Name (for @mentions)</label>
          <div class="input-prefix">
            <span>@</span>
            <input type="text" bind:value={newCustomAgent.name} placeholder="coder" />
          </div>
          <span class="form-hint">Lowercase, no spaces (use hyphens)</span>
        </div>
        <div class="form-group">
          <label>Display Name</label>
          <input type="text" bind:value={newCustomAgent.display_name} placeholder="My Coding Assistant" />
        </div>
        <div class="form-group full-width">
          <label>Description</label>
          <input type="text" bind:value={newCustomAgent.description} placeholder="A helpful coding assistant..." />
        </div>
        <div class="form-group full-width">
          <label>System Prompt</label>
          <textarea bind:value={newCustomAgent.system_prompt} rows="8" placeholder="You are an expert programmer..."></textarea>
          <span class="form-hint">This prompt defines the agent's personality and behavior</span>
        </div>
        <div class="form-group">
          <label>Model (optional)</label>
          <select bind:value={newCustomAgent.model_preference}>
            <option value="">Use default</option>
            <option value="llama-3.3-70b-versatile">Llama 3.3 70B</option>
            <option value="llama-3.1-8b-instant">Llama 3.1 8B</option>
            <option value="claude-3-5-sonnet">Claude 3.5 Sonnet</option>
          </select>
        </div>
        <div class="form-group">
          <label>Temperature</label>
          <input type="number" bind:value={newCustomAgent.temperature} min="0" max="2" step="0.1" />
        </div>
      </div>
      <div class="form-actions">
        <button class="btn-pill btn-pill-ghost btn secondary" onclick={() => showNewCustomAgent = false}>Cancel</button>
        <button class="btn-pill btn-pill-ghost btn primary" onclick={createCustomAgent} disabled={savingCustomAgent}>
          {savingCustomAgent ? 'Creating...' : 'Create Agent'}
        </button>
      </div>
    </div>
  {/if}

  {#if loadingCustomAgents}
    <div class="loading">
      <div class="spinner"></div>
      <span>Loading custom agents...</span>
    </div>
  {:else if customAgents.length === 0}
    <div class="empty-state small">
      <p>No custom agents yet. Create one to mention with @name in chat!</p>
    </div>
  {:else}
    <div class="custom-agents-list">
      {#each customAgents as agent}
        <div class="custom-agent-card">
          <div class="agent-header">
            <div class="agent-info">
              <span class="agent-mention">@{agent.name}</span>
              <span class="agent-display-name">{agent.display_name}</span>
            </div>
            <div class="agent-actions">
              <span class="usage-badge">{agent.times_used} uses</span>
              <button class="btn-pill btn-pill-ghost icon-btn" onclick={() => testingAgentId = testingAgentId === agent.id ? null : agent.id} title="Test Agent">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2m-3 7h3m-3 4h3m-6-4h.01M9 16h.01"/></svg>
              </button>
              <button class="btn-pill btn-pill-ghost icon-btn" onclick={() => editingCustomAgent = {...agent}} title="Edit">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"/><path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"/></svg>
              </button>
              <button class="btn-pill btn-pill-ghost icon-btn danger" onclick={() => deleteCustomAgent(agent.id)} title="Delete">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M3 6h18M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/></svg>
              </button>
            </div>
          </div>
          {#if agent.description}
            <p class="agent-desc">{agent.description}</p>
          {/if}
          <div class="agent-prompt-preview">
            <span class="label">Prompt:</span>
            <span class="preview">{agent.system_prompt.slice(0, 100)}...</span>
          </div>

          {#if testingAgentId === agent.id}
            <div class="agent-test-section" style="margin-top: 1rem; padding-top: 1rem; border-top: 1px solid var(--color-border);">
              <h4 style="font-size: 0.875rem; font-weight: 600; color: var(--dt2); margin-bottom: 0.75rem;">Test Agent</h4>
              <AgentTestSandbox {agent} />
            </div>
          {/if}
        </div>
      {/each}
    </div>
  {/if}
</section>

<!-- Edit Custom Agent Modal -->
{#if editingCustomAgent}
  <div class="modal-overlay" onclick={() => editingCustomAgent = null}>
    <div class="modal" onclick={(e) => e.stopPropagation()}>
      <div class="form-header">
        <h3>Edit Agent</h3>
        <button class="btn-pill btn-pill-ghost close-btn" onclick={() => editingCustomAgent = null}>×</button>
      </div>
      <div class="form-grid">
        <div class="form-group">
          <label>Agent Name</label>
          <div class="input-prefix">
            <span>@</span>
            <input type="text" bind:value={editingCustomAgent.name} placeholder="coder" />
          </div>
        </div>
        <div class="form-group">
          <label>Display Name</label>
          <input type="text" bind:value={editingCustomAgent.display_name} placeholder="My Coding Assistant" />
        </div>
        <div class="form-group full-width">
          <label>Description</label>
          <input type="text" bind:value={editingCustomAgent.description} placeholder="A helpful coding assistant..." />
        </div>
        <div class="form-group full-width">
          <label>System Prompt</label>
          <textarea bind:value={editingCustomAgent.system_prompt} rows="8" placeholder="You are an expert programmer..."></textarea>
        </div>
      </div>
      <div class="form-actions">
        <button class="btn-pill btn-pill-ghost btn secondary" onclick={() => editingCustomAgent = null}>Cancel</button>
        <button class="btn-pill btn-pill-ghost btn primary" onclick={saveEditedCustomAgent} disabled={savingCustomAgent}>
          {savingCustomAgent ? 'Saving...' : 'Save Changes'}
        </button>
      </div>
    </div>
  </div>
{/if}

<style>
  .section {
    margin-bottom: 32px;
  }

  .section-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: 8px;
  }

  .section-header h2 {
    font-size: 16px;
    font-weight: 600;
    margin: 0;
  }

  .section-subtitle {
    color: var(--color-text-muted);
    font-size: 14px;
    margin: -8px 0 16px 0;
  }

  .add-btn {
    display: flex;
    align-items: center;
    gap: 6px;
    padding: 8px 16px;
    background: var(--color-primary);
    color: white;
    border: none;
    border-radius: 8px;
    font-size: 13px;
    font-weight: 500;
    cursor: pointer;
    transition: all 0.2s;
  }

  .add-btn:hover { opacity: 0.9; }
  .add-btn svg { width: 16px; height: 16px; }

  .custom-agent-form {
    background: var(--color-bg-elevated);
    border: 1px solid var(--color-border);
    border-radius: 12px;
    padding: 20px;
    margin-bottom: 20px;
  }

  .custom-agents-list {
    display: flex;
    flex-direction: column;
    gap: 12px;
  }

  .custom-agent-card {
    background: var(--color-bg);
    border: 1px solid var(--color-border);
    border-radius: 12px;
    padding: 16px;
  }

  .custom-agent-card:hover {
    border-color: var(--color-primary);
  }

  .custom-agent-card .agent-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 8px;
  }

  .custom-agent-card .agent-info {
    display: flex;
    align-items: center;
    gap: 12px;
  }

  .agent-mention {
    font-family: monospace;
    font-weight: 600;
    color: var(--color-primary);
    background: var(--color-primary-bg);
    padding: 4px 8px;
    border-radius: 6px;
    font-size: 14px;
  }

  .agent-display-name {
    font-weight: 500;
    color: var(--color-text);
  }

  .custom-agent-card .agent-actions {
    display: flex;
    align-items: center;
    gap: 8px;
  }

  .usage-badge {
    font-size: 12px;
    color: var(--color-text-muted);
    background: var(--color-bg-elevated);
    padding: 4px 8px;
    border-radius: 4px;
  }

  .custom-agent-card .agent-desc {
    font-size: 14px;
    color: var(--color-text-muted);
    margin-bottom: 8px;
  }

  .agent-prompt-preview {
    font-size: 13px;
    display: flex;
    gap: 8px;
  }

  .agent-prompt-preview .label {
    color: var(--color-text-muted);
    flex-shrink: 0;
  }

  .agent-prompt-preview .preview {
    color: var(--color-text-secondary);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .empty-state.small {
    padding: 24px;
    text-align: center;
    color: var(--color-text-muted);
    background: var(--color-bg-elevated);
    border-radius: 8px;
  }

  .icon-btn {
    width: 28px;
    height: 28px;
    border: none;
    background: var(--color-bg-tertiary);
    color: var(--color-text-secondary);
    border-radius: 6px;
    cursor: pointer;
    display: flex;
    align-items: center;
    justify-content: center;
    transition: all 0.2s;
  }

  .icon-btn svg {
    width: 14px;
    height: 14px;
  }

  .icon-btn:hover {
    background: var(--color-bg);
    color: var(--color-text);
  }

  .icon-btn.danger:hover {
    color: var(--color-error);
    background: rgba(239, 68, 68, 0.1);
    color: #ef4444;
  }

  .form-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 16px;
  }

  .form-header h3 {
    font-size: 16px;
    font-weight: 600;
    margin: 0;
  }

  .close-btn {
    width: 28px;
    height: 28px;
    border: none;
    background: var(--color-bg-tertiary);
    color: var(--color-text-secondary);
    border-radius: 6px;
    font-size: 18px;
    cursor: pointer;
    display: flex;
    align-items: center;
    justify-content: center;
  }

  .close-btn:hover {
    background: var(--color-bg);
    color: var(--color-text);
  }

  .form-grid {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 16px;
  }

  .form-group {
    display: flex;
    flex-direction: column;
    gap: 6px;
  }

  .form-group.full-width {
    grid-column: span 2;
  }

  .form-group label {
    font-size: 13px;
    font-weight: 500;
    color: var(--color-text-secondary);
  }

  .form-group input,
  .form-group textarea,
  .form-group select {
    padding: 10px 12px;
    border: 1px solid var(--color-border);
    border-radius: 8px;
    background: var(--color-bg);
    color: var(--color-text);
    font-size: 14px;
    transition: border-color 0.2s;
  }

  .form-group input:focus,
  .form-group textarea:focus,
  .form-group select:focus {
    outline: none;
    border-color: var(--color-primary);
  }

  .form-group textarea {
    resize: vertical;
    font-family: monospace;
    font-size: 13px;
  }

  .input-prefix {
    display: flex;
    align-items: center;
    border: 1px solid var(--color-border);
    border-radius: 8px;
    background: var(--color-bg);
    overflow: hidden;
  }

  .input-prefix span {
    padding: 10px 4px 10px 12px;
    color: var(--color-text-secondary);
    font-family: monospace;
  }

  .input-prefix input {
    border: none;
    padding-left: 0;
    flex: 1;
  }

  .form-hint {
    font-size: 11px;
    color: var(--color-text-tertiary);
  }

  .form-actions {
    display: flex;
    justify-content: flex-end;
    gap: 12px;
    margin-top: 20px;
    padding-top: 16px;
    border-top: 1px solid var(--color-border);
  }

  .btn {
    padding: 10px 20px;
    border: none;
    border-radius: 8px;
    font-size: 14px;
    font-weight: 500;
    cursor: pointer;
    transition: all 0.2s;
  }

  .btn.primary {
    background: var(--color-primary);
    color: white;
  }

  .btn.primary:hover { opacity: 0.9; }
  .btn.primary:disabled { opacity: 0.5; cursor: not-allowed; }

  .btn.secondary {
    background: var(--color-bg-tertiary);
    color: var(--color-text);
  }

  .btn.secondary:hover {
    background: var(--color-bg);
  }

  .modal-overlay {
    position: fixed;
    inset: 0;
    background: rgba(0, 0, 0, 0.5);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 100;
  }

  .modal {
    background: var(--color-bg-secondary);
    border: 1px solid var(--color-border);
    border-radius: 12px;
    padding: 20px;
    max-width: 600px;
    width: 90%;
    max-height: 85vh;
    overflow-y: auto;
  }

  .loading {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 24px;
    color: var(--color-text-muted);
  }

  .spinner {
    width: 20px;
    height: 20px;
    border: 2px solid var(--color-border);
    border-top-color: var(--color-primary);
    border-radius: 50%;
    animation: spin 0.8s linear infinite;
  }

  @keyframes spin {
    to { transform: rotate(360deg); }
  }

  :global(.dark) .icon-btn {
    background: var(--bos-settings-card-bg);
  }

  :global(.dark) .close-btn {
    background: var(--bos-settings-card-bg);
  }

  :global(.dark) .btn.secondary {
    background: var(--bos-settings-card-bg);
  }

  :global(.dark) .form-group input,
  :global(.dark) .form-group textarea,
  :global(.dark) .form-group select {
    background: var(--dbg);
    border-color: var(--dbd);
  }

  :global(.dark) .input-prefix {
    background: var(--dbg);
    border-color: var(--dbd);
  }

  :global(.dark) .modal {
    background: var(--dbg2);
    border-color: var(--dbd);
  }

  :global(.dark) .add-btn,
  :global(.dark) .btn.primary {
    background: var(--bos-nav-active);
    color: white;
  }
</style>
