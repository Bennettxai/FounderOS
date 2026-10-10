<script lang="ts">
  import type { CommandInfo } from '$lib/stores/aiSettings';
  import { contextSourceOptions, toggleContextSource } from '$lib/stores/aiSettings';
  import { apiClient } from '$lib/api';

  interface Props {
    commands: CommandInfo[];
    loadingCommands: boolean;
    onCommandsChanged: () => void;
  }

  let { commands, loadingCommands, onCommandsChanged }: Props = $props();

  let showNewCommand = $state(false);
  let editingCommand = $state<CommandInfo | null>(null);
  let expandedCommand = $state<string | null>(null);
  let savingCommand = $state(false);
  let newCommand = $state<Partial<CommandInfo>>({
    name: '',
    display_name: '',
    description: '',
    icon: '',
    system_prompt: '',
    context_sources: []
  });

  async function saveNewCommand() {
    if (!newCommand.name?.trim()) return;
    savingCommand = true;
    try {
      const res = await apiClient.post('/ai/commands', newCommand);
      if (res.ok) {
        showNewCommand = false;
        newCommand = {
          name: '',
          display_name: '',
          description: '',
          icon: '',
          system_prompt: '',
          context_sources: []
        };
        onCommandsChanged();
      }
    } catch {
      // silently fail
    } finally {
      savingCommand = false;
    }
  }

  async function updateCommand(cmd: CommandInfo) {
    savingCommand = true;
    try {
      const endpoint = cmd.id
        ? `/ai/commands/${cmd.id}`
        : `/ai/commands/${cmd.name}/override`;
      const res = await apiClient.put(endpoint, cmd);
      if (res.ok) {
        editingCommand = null;
        onCommandsChanged();
      }
    } catch {
      // silently fail
    } finally {
      savingCommand = false;
    }
  }

  async function deleteCommand(cmdId: string) {
    if (!confirm('Delete this command?')) return;
    try {
      const res = await apiClient.delete(`/ai/commands/${cmdId}`);
      if (res.ok) {
        onCommandsChanged();
      }
    } catch {
      // silently fail
    }
  }

  function toggleBuiltInCommand(name: string) {
    expandedCommand = expandedCommand === name ? null : name;
  }

  function handleCommandCardKeydown(event: KeyboardEvent, name: string) {
    if (event.key === 'Enter' || event.key === ' ') {
      event.preventDefault();
      toggleBuiltInCommand(name);
    }
  }
</script>

<!-- Commands Tab -->
<section class="section">
  <div class="section-header">
    <h2>Slash Commands</h2>
    <button class="btn-pill btn-pill-ghost add-btn" onclick={() => showNewCommand = true}>
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M12 5v14M5 12h14"/></svg>
      New Command
    </button>
  </div>

  {#if loadingCommands}
    <div class="loading">
      <div class="spinner"></div>
      <span>Loading commands...</span>
    </div>
  {:else}
    <!-- New Command Form -->
    {#if showNewCommand}
      <div class="command-form">
        <div class="form-header">
          <h3>Create Custom Command</h3>
          <button class="btn-pill btn-pill-ghost close-btn" onclick={() => showNewCommand = false}>×</button>
        </div>
        <div class="form-grid">
          <div class="form-group">
            <label for="new-command-name">Command Name</label>
            <div class="input-prefix">
              <span>/</span>
              <input id="new-command-name" type="text" bind:value={newCommand.name} placeholder="my-command" />
            </div>
            <span class="form-hint">Lowercase, no spaces (use hyphens)</span>
          </div>
          <div class="form-group">
            <label for="new-command-display-name">Display Name</label>
            <input id="new-command-display-name" type="text" bind:value={newCommand.display_name} placeholder="My Command" />
          </div>
          <div class="form-group full-width">
            <label for="new-command-description">Description</label>
            <input id="new-command-description" type="text" bind:value={newCommand.description} placeholder="What this command does..." />
          </div>
          <div class="form-group">
            <label for="new-command-icon">Icon (Emoji)</label>
            <input id="new-command-icon" type="text" bind:value={newCommand.icon} placeholder="✨" maxlength="4" />
          </div>
          <div class="form-group full-width">
            <label for="new-command-system-prompt">System Prompt</label>
            <textarea id="new-command-system-prompt" bind:value={newCommand.system_prompt} rows="6" placeholder="You are an AI assistant that..."></textarea>
            <span class="form-hint">This prompt will be used when the command is executed</span>
          </div>
          <div class="form-group full-width">
            <span class="field-label">Context Sources</span>
            <div class="context-sources">
              {#each contextSourceOptions as opt}
                <button
                  class="btn-pill btn-pill-ghost context-chip"
                  class:active={newCommand.context_sources?.includes(opt.id)}
                  onclick={() => newCommand.context_sources = toggleContextSource(newCommand.context_sources, opt.id)}
                  title={opt.desc}
                >
                  {opt.label}
                </button>
              {/each}
            </div>
            <span class="form-hint">Select what data to automatically include when command runs</span>
          </div>
        </div>
        <div class="form-actions">
          <button class="btn-pill btn-pill-ghost btn secondary" onclick={() => showNewCommand = false}>Cancel</button>
          <button class="btn-pill btn-pill-ghost btn primary" onclick={saveNewCommand} disabled={savingCommand}>
            {savingCommand ? 'Creating...' : 'Create Command'}
          </button>
        </div>
      </div>
    {/if}

    <!-- Built-in Commands -->
    <div class="commands-section">
      <h4>Built-in Commands ({commands.filter(c => !c.is_custom).length})</h4>
      <div class="commands-grid">
        {#each commands.filter(c => !c.is_custom) as cmd}
          <div
            class="command-card clickable"
            class:expanded={expandedCommand === cmd.name}
            role="button"
            tabindex="0"
            onclick={() => toggleBuiltInCommand(cmd.name)}
            onkeydown={(event) => handleCommandCardKeydown(event, cmd.name)}
          >
            <div class="command-header">
              <span class="command-icon">{cmd.icon || '⚡'}</span>
              <span class="command-name">/{cmd.name}</span>
              <span class="command-category">{cmd.category}</span>
              <button
                class="btn-pill btn-pill-ghost icon-btn edit-builtin"
                onclick={(e) => { e.stopPropagation(); editingCommand = {...cmd, is_builtin_override: true}; }}
                title="Customize this command"
                aria-label="Customize command"
              >
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"/><path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"/></svg>
              </button>
            </div>
            <p class="command-display-name">{cmd.display_name}</p>
            <p class="command-desc">{cmd.description}</p>
            {#if cmd.context_sources?.length > 0}
              <div class="command-sources">
                {#each cmd.context_sources as source}
                  <span class="source-tag">{source}</span>
                {/each}
              </div>
            {/if}
            {#if expandedCommand === cmd.name}
              <div class="command-details">
                <div class="detail-section">
                  <h5>System Prompt</h5>
                  <p class="detail-hint">This is the prompt that guides the AI when using this command</p>
                  <pre class="prompt-preview">{cmd.system_prompt || 'Built-in prompt (customize to view and edit)'}</pre>
                </div>
                <div class="detail-actions">
                  <button class="btn-pill btn-pill-ghost btn primary small" onclick={(e) => { e.stopPropagation(); editingCommand = {...cmd, is_builtin_override: true}; }}>
                    Customize Command
                  </button>
                </div>
              </div>
            {/if}
          </div>
        {/each}
      </div>
    </div>

    <!-- Custom Commands -->
    {#if commands.filter(c => c.is_custom).length > 0}
      <div class="commands-section">
        <h4>Your Custom Commands</h4>
        <div class="commands-grid">
          {#each commands.filter(c => c.is_custom) as cmd}
            <div class="command-card custom">
              <div class="command-header">
                <span class="command-icon">{cmd.icon || '✨'}</span>
                <span class="command-name">/{cmd.name}</span>
                <div class="command-actions">
                  <button class="btn-pill btn-pill-ghost icon-btn" onclick={() => editingCommand = {...cmd}} title="Edit" aria-label="Edit command">
                    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"/><path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"/></svg>
                  </button>
                  <button class="btn-pill btn-pill-ghost icon-btn danger" onclick={() => cmd.id && deleteCommand(cmd.id)} title="Delete" aria-label="Delete command">
                    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="3 6 5 6 21 6"/><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/></svg>
                  </button>
                </div>
              </div>
              <p class="command-display-name">{cmd.display_name}</p>
              <p class="command-desc">{cmd.description}</p>
              {#if cmd.context_sources?.length > 0}
                <div class="command-sources">
                  {#each cmd.context_sources as source}
                    <span class="source-tag">{source}</span>
                  {/each}
                </div>
              {/if}
            </div>
          {/each}
        </div>
      </div>
    {/if}

    <!-- Edit Command Modal -->
    {#if editingCommand}
      <div class="modal-overlay" role="presentation" onclick={() => editingCommand = null}>
        <div class="modal" role="dialog" aria-modal="true" tabindex="-1" onclick={(e) => e.stopPropagation()}>
          <div class="form-header">
            <h3>Edit Command</h3>
            <button class="btn-pill btn-pill-ghost close-btn" onclick={() => editingCommand = null}>×</button>
          </div>
          <div class="form-grid">
            <div class="form-group">
              <label for="edit-command-name">Command Name</label>
              <div class="input-prefix">
                <span>/</span>
                <input id="edit-command-name" type="text" bind:value={editingCommand.name} placeholder="my-command" />
              </div>
            </div>
            <div class="form-group">
              <label for="edit-command-display-name">Display Name</label>
              <input id="edit-command-display-name" type="text" bind:value={editingCommand.display_name} placeholder="My Command" />
            </div>
            <div class="form-group full-width">
              <label for="edit-command-description">Description</label>
              <input id="edit-command-description" type="text" bind:value={editingCommand.description} placeholder="What this command does..." />
            </div>
            <div class="form-group">
              <label for="edit-command-icon">Icon (Emoji)</label>
              <input id="edit-command-icon" type="text" bind:value={editingCommand.icon} placeholder="✨" maxlength="4" />
            </div>
            <div class="form-group full-width">
              <label for="edit-command-system-prompt">System Prompt</label>
              <textarea id="edit-command-system-prompt" bind:value={editingCommand.system_prompt} rows="6" placeholder="You are an AI assistant that..."></textarea>
            </div>
            <div class="form-group full-width">
              <span class="field-label">Context Sources</span>
              <div class="context-sources">
                {#each contextSourceOptions as opt}
                  <button
                    class="btn-pill btn-pill-ghost context-chip"
                    class:active={editingCommand.context_sources?.includes(opt.id)}
                    onclick={() => editingCommand && (editingCommand.context_sources = toggleContextSource(editingCommand.context_sources, opt.id))}
                    title={opt.desc}
                  >
                    {opt.label}
                  </button>
                {/each}
              </div>
            </div>
          </div>
          <div class="form-actions">
            <button class="btn-pill btn-pill-ghost btn secondary" onclick={() => editingCommand = null}>Cancel</button>
            <button class="btn-pill btn-pill-ghost btn primary" onclick={() => editingCommand && updateCommand(editingCommand)} disabled={savingCommand}>
              {savingCommand ? 'Saving...' : 'Save Changes'}
            </button>
          </div>
        </div>
      </div>
    {/if}
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
    margin-bottom: 16px;
  }

  .section-header h2 {
    font-size: 16px;
    font-weight: 600;
    margin: 0;
  }

  /* Add Button */
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

  /* Command form */
  .command-form {
    background: var(--color-bg-secondary);
    border: 1px solid var(--color-border);
    border-radius: 12px;
    padding: 20px;
    margin-bottom: 24px;
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

  .form-group label,
  .field-label {
    font-size: 13px;
    font-weight: 500;
    color: var(--color-text-secondary);
  }

  .form-group input, .form-group textarea {
    padding: 10px 12px;
    border: 1px solid var(--color-border);
    border-radius: 8px;
    background: var(--color-bg);
    color: var(--color-text);
    font-size: 14px;
    transition: border-color 0.2s;
  }

  .form-group input:focus, .form-group textarea:focus {
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

  .context-sources {
    display: flex;
    flex-wrap: wrap;
    gap: 8px;
  }

  .context-chip {
    padding: 6px 12px;
    border: 1px solid var(--color-border);
    border-radius: 20px;
    background: var(--color-bg);
    color: var(--color-text-secondary);
    font-size: 12px;
    cursor: pointer;
    transition: all 0.2s;
  }

  .context-chip:hover {
    border-color: var(--color-primary);
  }

  .context-chip.active {
    background: var(--color-primary);
    color: white;
    border-color: var(--color-primary);
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

  .btn.small {
    padding: 6px 12px;
    font-size: 12px;
  }

  /* Commands sections */
  .commands-section {
    margin-bottom: 24px;
  }

  .commands-section h4 {
    font-size: 14px;
    font-weight: 600;
    margin: 0 0 12px;
    color: var(--color-text-secondary);
  }

  .commands-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
    gap: 12px;
  }

  .command-card {
    background: var(--color-bg);
    border: 1px solid var(--color-border);
    border-radius: 10px;
    padding: 14px;
    transition: all 0.2s;
  }

  .command-card:hover {
    border-color: var(--color-border-hover);
  }

  .command-card.custom {
    border-color: rgba(99, 102, 241, 0.3);
    background: rgba(99, 102, 241, 0.03);
  }

  .command-card.clickable {
    cursor: pointer;
    transition: all 0.2s;
  }

  .command-card.clickable:hover {
    border-color: var(--color-primary);
    transform: translateY(-1px);
  }

  .command-card.expanded {
    border-color: var(--color-primary);
    background: var(--color-bg-secondary);
  }

  .command-header {
    display: flex;
    align-items: center;
    gap: 8px;
    margin-bottom: 8px;
  }

  .command-icon {
    font-size: 18px;
  }

  .command-name {
    font-family: monospace;
    font-size: 14px;
    font-weight: 600;
    color: var(--color-primary);
  }

  .command-category {
    margin-left: auto;
    font-size: 11px;
    padding: 2px 8px;
    background: var(--color-bg-tertiary);
    border-radius: 4px;
    color: var(--color-text-tertiary);
    text-transform: capitalize;
  }

  .command-actions {
    margin-left: auto;
    display: flex;
    gap: 4px;
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
    background: rgba(239, 68, 68, 0.1);
    color: #ef4444;
  }

  .command-header .edit-builtin {
    opacity: 0;
    transition: opacity 0.2s;
  }

  .command-card:hover .edit-builtin {
    opacity: 1;
  }

  .command-display-name {
    font-size: 14px;
    font-weight: 500;
    margin: 0 0 4px;
    color: var(--color-text);
  }

  .command-desc {
    font-size: 12px;
    color: var(--color-text-secondary);
    margin: 0;
    line-height: 1.4;
  }

  .command-sources {
    display: flex;
    flex-wrap: wrap;
    gap: 4px;
    margin-top: 10px;
  }

  .source-tag {
    font-size: 10px;
    padding: 2px 6px;
    background: var(--color-bg-tertiary);
    border-radius: 4px;
    color: var(--color-text-tertiary);
  }

  /* Command details expanded */
  .command-details {
    margin-top: 16px;
    padding-top: 16px;
    border-top: 1px solid var(--color-border);
  }

  .detail-section h5 {
    font-size: 12px;
    font-weight: 600;
    color: var(--color-text-secondary);
    margin: 0 0 4px;
    text-transform: uppercase;
    letter-spacing: 0.5px;
  }

  .detail-hint {
    font-size: 11px;
    color: var(--color-text-tertiary);
    margin: 0 0 8px;
  }

  .prompt-preview {
    font-size: 12px;
    font-family: monospace;
    background: var(--color-bg-tertiary);
    padding: 12px;
    border-radius: 8px;
    white-space: pre-wrap;
    word-break: break-word;
    max-height: 200px;
    overflow-y: auto;
    color: var(--color-text-secondary);
    line-height: 1.5;
    margin: 0;
  }

  .detail-actions {
    margin-top: 12px;
    display: flex;
    justify-content: flex-end;
  }

  /* Modal */
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

  /* Loading */
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

  /* Dark mode */
  :global(.dark) .command-form,
  :global(.dark) .modal {
    background: #2c2c2e;
    border-color: rgba(255, 255, 255, 0.1);
  }

  :global(.dark) .command-card {
    background: #1c1c1e;
    border-color: rgba(255, 255, 255, 0.1);
  }

  :global(.dark) .command-card.custom {
    background: rgba(99, 102, 241, 0.1);
    border-color: rgba(99, 102, 241, 0.3);
  }

  :global(.dark) .context-chip {
    background: #3a3a3c;
    border-color: rgba(255, 255, 255, 0.1);
  }

  :global(.dark) .context-chip.active {
    background: #0A84FF;
    border-color: #0A84FF;
  }

  :global(.dark) .close-btn,
  :global(.dark) .icon-btn {
    background: #3a3a3c;
  }

  :global(.dark) .btn.secondary {
    background: #3a3a3c;
  }

  :global(.dark) .input-prefix {
    background: #1c1c1e;
    border-color: rgba(255, 255, 255, 0.1);
  }

  :global(.dark) .form-group input,
  :global(.dark) .form-group textarea {
    background: #1c1c1e;
    border-color: rgba(255, 255, 255, 0.1);
  }

  :global(.dark) .form-group input:focus,
  :global(.dark) .form-group textarea:focus {
    border-color: #0A84FF;
  }

  :global(.dark) .command-name {
    color: #0A84FF;
  }

  :global(.dark) .command-card.expanded {
    background: #2c2c2e;
    border-color: #0A84FF;
  }

  :global(.dark) .command-card.clickable:hover {
    border-color: #0A84FF;
  }

  :global(.dark) .command-details {
    border-top-color: rgba(255, 255, 255, 0.1);
  }

  :global(.dark) .prompt-preview {
    background: #1c1c1e;
    color: #98989d;
  }

  :global(.dark) .source-tag {
    background: #3a3a3c;
    color: #98989d;
  }

  :global(.dark) .add-btn,
  :global(.dark) .btn.primary {
    background: #0A84FF;
    color: white;
  }
</style>
