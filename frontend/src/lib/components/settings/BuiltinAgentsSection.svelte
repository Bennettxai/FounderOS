<script lang="ts">
  import type { AgentInfo } from '$lib/stores/aiSettings';
  import { getCategoryIconPath, getCategoryLabel } from '$lib/stores/aiSettings';
  import { apiClient } from '$lib/api';

  interface Props {
    agents: AgentInfo[];
    loadingAgents: boolean;
    onAgentUpdated: () => void;
  }

  let { agents, loadingAgents, onAgentUpdated }: Props = $props();

  let expandedAgent = $state<string | null>(null);
  let editingAgent = $state<string | null>(null);
  let editedPrompt = $state<string>('');

  function toggleAgentExpand(agentId: string) {
    expandedAgent = expandedAgent === agentId ? null : agentId;
  }

  function startEditingAgent(agentId: string, currentPrompt: string) {
    editingAgent = agentId;
    editedPrompt = currentPrompt;
  }

  function cancelEditing() {
    editingAgent = null;
    editedPrompt = '';
  }

  async function saveAgentPrompt(agentId: string) {
    try {
      const res = await apiClient.put(`/ai/agents/${agentId}/prompt`, { prompt: editedPrompt });
      if (res.ok) {
        editingAgent = null;
        editedPrompt = '';
        onAgentUpdated();
      }
    } catch {
      // silently fail
    }
  }
</script>

<section class="section">
  <div class="section-header">
    <h2>Built-in Agents</h2>
    <span class="subtitle">View and customize agent prompts</span>
  </div>

  {#if loadingAgents}
    <div class="loading">
      <div class="spinner"></div>
      <span>Loading agents...</span>
    </div>
  {:else if agents.length === 0}
    <div class="empty-state">
      <div class="empty-icon">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="11" width="18" height="11" rx="2"/><circle cx="12" cy="5" r="2"/><path d="M12 7v4"/><path d="M7 22v-3"/><path d="M17 22v-3"/></svg>
      </div>
      <h3>No Agents Found</h3>
      <p>Agent configuration is not available</p>
    </div>
  {:else}
    <div class="agents-list">
      {#each agents as agent}
        <div class="agent-card" class:expanded={expandedAgent === agent.id}>
          <button class="btn-pill btn-pill-ghost agent-header-btn" onclick={() => toggleAgentExpand(agent.id)}>
            <div class="agent-info">
              <span class="agent-icon">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d={getCategoryIconPath(agent.category)}/></svg>
              </span>
              <div class="agent-text">
                <span class="agent-name">{agent.name}</span>
                <span class="agent-desc">{agent.description}</span>
              </div>
            </div>
            <div class="agent-meta">
              <span class="agent-category">{getCategoryLabel(agent.category)}</span>
              <svg class="agent-chevron" class:rotated={expandedAgent === agent.id} viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <polyline points="6 9 12 15 18 9"/>
              </svg>
            </div>
          </button>

          {#if expandedAgent === agent.id}
            <div class="agent-content">
              <div class="prompt-header">
                <h4>System Prompt</h4>
                {#if editingAgent !== agent.id}
                  <button class="btn-pill btn-pill-ghost prompt-edit-btn" onclick={() => startEditingAgent(agent.id, agent.prompt)}>
                    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"/><path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"/></svg>
                    Edit
                  </button>
                {/if}
              </div>

              {#if editingAgent === agent.id}
                <div class="prompt-editor">
                  <textarea
                    bind:value={editedPrompt}
                    rows="12"
                    placeholder="Enter system prompt..."
                  ></textarea>
                  <div class="editor-actions">
                    <button class="btn-pill btn-pill-ghost editor-btn cancel" onclick={cancelEditing}>Cancel</button>
                    <button class="btn-pill btn-pill-ghost editor-btn save" onclick={() => saveAgentPrompt(agent.id)}>Save Changes</button>
                  </div>
                </div>
              {:else}
                <div class="prompt-display">
                  <pre>{agent.prompt}</pre>
                </div>
              {/if}

              <div class="agent-stats">
                <div class="stat-item">
                  <span class="stat-label">ID</span>
                  <span class="stat-value-small">{agent.id}</span>
                </div>
                <div class="stat-item">
                  <span class="stat-label">Characters</span>
                  <span class="stat-value-small">{agent.prompt.length.toLocaleString()}</span>
                </div>
                <div class="stat-item">
                  <span class="stat-label">~Tokens</span>
                  <span class="stat-value-small">{Math.ceil(agent.prompt.length / 4).toLocaleString()}</span>
                </div>
              </div>
            </div>
          {/if}
        </div>
      {/each}
    </div>
  {/if}
</section>

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

  .subtitle {
    font-size: 13px;
    color: var(--color-text-muted);
  }

  .agents-list {
    display: flex;
    flex-direction: column;
    gap: 12px;
  }

  .agent-card {
    background: var(--color-bg);
    border: 1px solid var(--color-border);
    border-radius: 12px;
    overflow: hidden;
    transition: all 0.2s;
  }

  .agent-card:hover {
    border-color: var(--color-border-hover);
  }

  .agent-card.expanded {
    border-color: var(--color-primary);
  }

  .agent-header-btn {
    width: 100%;
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 16px 20px;
    background: transparent;
    border: none;
    cursor: pointer;
    text-align: left;
  }

  .agent-info {
    display: flex;
    align-items: center;
    gap: 14px;
  }

  .agent-icon {
    width: 48px;
    height: 48px;
    display: flex;
    align-items: center;
    justify-content: center;
    background: var(--color-bg-secondary);
    border-radius: 12px;
    flex-shrink: 0;
  }

  .agent-icon svg {
    width: 24px;
    height: 24px;
    color: var(--color-text-muted);
  }

  :global(.dark) .agent-icon {
    background: rgba(255, 255, 255, 0.05);
  }

  :global(:not(.dark)) .agent-icon {
    background: rgba(0, 0, 0, 0.03);
  }

  .agent-text {
    display: flex;
    flex-direction: column;
    gap: 4px;
  }

  .agent-name {
    font-weight: 600;
    font-size: 15px;
    color: var(--color-text);
  }

  .agent-desc {
    font-size: 13px;
    color: var(--color-text-muted);
  }

  .agent-meta {
    display: flex;
    align-items: center;
    gap: 12px;
  }

  .agent-category {
    font-size: 11px;
    padding: 4px 10px;
    background: var(--color-bg-tertiary);
    border-radius: 6px;
    color: var(--color-text-secondary);
    text-transform: uppercase;
    font-weight: 500;
  }

  .agent-chevron {
    width: 20px;
    height: 20px;
    color: var(--color-text-muted);
    transition: transform 0.2s;
  }

  .agent-chevron.rotated {
    transform: rotate(180deg);
  }

  .agent-content {
    padding: 0 20px 20px;
    border-top: 1px solid var(--color-border);
    background: var(--color-bg-secondary);
  }

  .prompt-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 16px 0 12px;
  }

  .prompt-header h4 {
    margin: 0;
    font-size: 14px;
    font-weight: 600;
  }

  .prompt-edit-btn {
    display: flex;
    align-items: center;
    gap: 6px;
    padding: 6px 12px;
    background: var(--color-bg);
    border: 1px solid var(--color-border);
    border-radius: 8px;
    font-size: 12px;
    color: var(--color-text-secondary);
    cursor: pointer;
    transition: all 0.2s;
  }

  .prompt-edit-btn:hover {
    background: var(--color-bg-tertiary);
    color: var(--color-text);
  }

  .prompt-edit-btn svg {
    width: 14px;
    height: 14px;
  }

  .prompt-display {
    background: var(--color-bg);
    border: 1px solid var(--color-border);
    border-radius: 10px;
    max-height: 400px;
    overflow-y: auto;
  }

  .prompt-display pre {
    margin: 0;
    padding: 16px;
    font-size: 12px;
    font-family: 'SF Mono', Monaco, 'Cascadia Code', 'Roboto Mono', monospace;
    line-height: 1.6;
    white-space: pre-wrap;
    word-break: break-word;
    color: var(--color-text-secondary);
  }

  .prompt-editor {
    display: flex;
    flex-direction: column;
    gap: 12px;
  }

  .prompt-editor textarea {
    width: 100%;
    padding: 16px;
    background: var(--color-bg);
    border: 1px solid var(--color-border);
    border-radius: 10px;
    font-size: 12px;
    font-family: 'SF Mono', Monaco, 'Cascadia Code', monospace;
    line-height: 1.6;
    color: var(--color-text);
    resize: vertical;
  }

  .prompt-editor textarea:focus {
    outline: none;
    border-color: var(--color-primary);
  }

  .editor-actions {
    display: flex;
    justify-content: flex-end;
    gap: 8px;
  }

  .editor-btn {
    padding: 8px 16px;
    border-radius: 8px;
    font-size: 13px;
    font-weight: 500;
    cursor: pointer;
    transition: all 0.2s;
  }

  .editor-btn.cancel {
    background: var(--color-bg);
    border: 1px solid var(--color-border);
    color: var(--color-text-secondary);
  }

  .editor-btn.cancel:hover {
    background: var(--color-bg-tertiary);
  }

  .editor-btn.save {
    background: var(--color-primary);
    border: none;
    color: var(--color-bg);
  }

  .editor-btn.save:hover {
    opacity: 0.9;
  }

  .agent-stats {
    display: flex;
    gap: 24px;
    padding-top: 16px;
    margin-top: 16px;
    border-top: 1px solid var(--color-border);
  }

  .stat-item {
    display: flex;
    flex-direction: column;
    gap: 2px;
  }

  .stat-item .stat-label {
    font-size: 10px;
    color: var(--color-text-muted);
    text-transform: uppercase;
    letter-spacing: 0.5px;
  }

  .stat-value-small {
    font-size: 13px;
    font-weight: 500;
    color: var(--color-text-secondary);
    font-family: monospace;
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

  .empty-state {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 12px;
    padding: 48px;
    text-align: center;
    color: var(--color-text-muted);
  }

  .empty-icon {
    width: 64px;
    height: 64px;
    display: flex;
    align-items: center;
    justify-content: center;
    background: var(--color-bg-tertiary);
    border-radius: 16px;
  }

  .empty-icon svg {
    width: 32px;
    height: 32px;
  }

  .empty-state h3 {
    margin: 0;
    font-size: 16px;
  }

  .empty-state p {
    margin: 0;
    font-size: 14px;
  }

  :global(.dark) .agent-card {
    background: #2c2c2e;
    border-color: rgba(255, 255, 255, 0.1);
  }

  :global(.dark) .agent-card:hover {
    border-color: rgba(255, 255, 255, 0.2);
  }

  :global(.dark) .agent-card.expanded {
    border-color: #0A84FF;
    background: rgba(10, 132, 255, 0.1);
  }

  :global(.dark) .agent-content {
    background: #1c1c1e;
  }

  :global(.dark) .prompt-display {
    background: #2c2c2e;
  }

  :global(.dark) .prompt-editor textarea {
    background: #2c2c2e;
  }

  :global(.dark) .editor-btn.save {
    background: #0A84FF;
    color: white;
  }

  :global(.dark) .agent-icon {
    background: #3a3a3c;
    border-color: rgba(255, 255, 255, 0.1);
  }
</style>
