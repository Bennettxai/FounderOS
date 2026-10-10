<script lang="ts">
  import type { LLMProvider } from '$lib/stores/aiSettings';
  import { getProviderIconPath, getProviderLabel } from '$lib/stores/aiSettings';
  import { apiClient } from '$lib/api';

  interface Props {
    providers: LLMProvider[];
    activeProvider: string;
    billingMode: string;
    onSelectProvider: (id: string) => void;
    onSetBillingMode: (mode: string) => void | Promise<void>;
    onProviderSaved: (id?: string) => void | Promise<void>;
  }

  let { providers, activeProvider, billingMode, onSelectProvider, onSetBillingMode, onProviderSaved }: Props = $props();
  let activeProviderConfig = $derived(providers.find(p => p.id === activeProvider));

  let apiKeys = $state<Record<string, string>>({
    ollama_cloud: '',
    groq: '',
    anthropic: ''
  });
  let savingKey = $state<string | null>(null);
  let showApiKey = $state<Record<string, boolean>>({});

  async function saveAPIKey(providerId: string) {
    const key = apiKeys[providerId]?.trim();
    if (!key) return;
    savingKey = providerId;
    try {
      const res = await apiClient.post('/ai/api-keys', { provider: providerId, api_key: key });
      if (res.ok) {
        apiKeys[providerId] = '';
        await onProviderSaved(providerId);
      }
    } catch {
      // silently fail — parent will refresh providers list
    } finally {
      savingKey = null;
    }
  }
</script>

<!-- Providers Tab -->
<section class="section">
  <div class="section-header">
    <h2>AI Providers</h2>
    <span class="badge active">
      Using {getProviderLabel(activeProvider)}
    </span>
    {#if activeProviderConfig?.type === 'cloud'}
      <div class="access-toggle" aria-label="Provider access mode">
        <button
          class:active={billingMode === 'platform'}
          disabled={!activeProviderConfig.platform_configured}
          onclick={() => onSetBillingMode('platform')}
          title={activeProviderConfig.platform_configured ? 'Use BusinessOS platform credits' : 'No platform key configured'}
        >
          Platform credits
        </button>
        <button
          class:active={billingMode === 'byok'}
          disabled={!activeProviderConfig.byok_configured}
          onclick={() => onSetBillingMode('byok')}
          title={activeProviderConfig.byok_configured ? 'Use saved BYOK credential' : 'Save a provider API key first'}
        >
          BYOK
        </button>
      </div>
    {:else if activeProviderConfig}
      <span class="subtitle">Local runtime</span>
    {/if}
  </div>

  <div class="providers-grid">
    {#each providers as provider}
      <button
        class="btn-pill btn-pill-ghost provider-card"
        class:active={activeProvider === provider.id}
        class:disabled={!provider.configured && provider.type === 'cloud'}
        onclick={() => onSelectProvider(provider.id)}
      >
        <div class="provider-icon">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d={getProviderIconPath(provider.id)}/></svg>
        </div>
        <div class="provider-info">
          <span class="provider-name">{provider.name}</span>
          <span class="provider-desc">{provider.description}</span>
        </div>
        <div class="provider-status">
          {#if activeProvider === provider.id}
            <span class="status active">Active</span>
          {:else if provider.configured}
            <span class="status ready">Ready</span>
          {:else}
            <span class="status setup">Setup Required</span>
          {/if}
        </div>
        <span class="provider-badge" class:local={provider.type === 'local'}>
          {provider.type === 'local' ? 'Local' : 'Cloud'}
        </span>
      </button>
    {/each}
  </div>
</section>

<!-- API Keys -->
<section class="section">
  <div class="section-header">
    <h2>API Keys</h2>
    <span class="subtitle">Configure cloud provider access</span>
  </div>

  <div class="api-grid">
    {#each [
      { id: 'groq', name: 'Groq', url: 'https://console.groq.com' },
      { id: 'anthropic', name: 'Anthropic', url: 'https://console.anthropic.com' },
      { id: 'ollama_cloud', name: 'Ollama Cloud', url: 'https://ollama.com' }
    ] as provider}
      {@const configuredProvider = providers.find(p => p.id === provider.id)}
      {@const isConfigured = configuredProvider?.configured}
      {@const keySource = configuredProvider?.key_source || 'none'}
      <div class="api-card">
        <div class="api-header">
          <span class="api-icon">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d={getProviderIconPath(provider.id)}/></svg>
          </span>
          <span class="api-name">{provider.name}</span>
          {#if isConfigured}
            <span class="api-configured" title={keySource === 'byok' ? 'BYOK key saved' : 'Platform key available'}>
              {keySource === 'both' ? 'Platform + BYOK' : keySource === 'byok' ? 'BYOK' : 'Platform'}
            </span>
          {/if}
        </div>
        <div class="api-form">
          <div class="input-wrapper">
            <input
              type={showApiKey[provider.id] ? 'text' : 'password'}
              bind:value={apiKeys[provider.id]}
              placeholder={isConfigured ? '••••••••••••' : 'Enter API key'}
            />
            <button class="toggle-btn" onclick={() => showApiKey[provider.id] = !showApiKey[provider.id]}>
              {#if showApiKey[provider.id]}
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M17.94 17.94A10.07 10.07 0 0 1 12 20c-7 0-11-8-11-8a18.45 18.45 0 0 1 5.06-5.94M9.9 4.24A9.12 9.12 0 0 1 12 4c7 0 11 8 11 8a18.5 18.5 0 0 1-2.16 3.19m-6.72-1.07a3 3 0 1 1-4.24-4.24"/><line x1="1" y1="1" x2="23" y2="23"/></svg>
              {:else}
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z"/><circle cx="12" cy="12" r="3"/></svg>
              {/if}
            </button>
          </div>
          <button
            class="btn-pill btn-pill-ghost save-btn"
            onclick={() => saveAPIKey(provider.id)}
            disabled={!apiKeys[provider.id]?.trim() || savingKey === provider.id}
          >
            {savingKey === provider.id ? '...' : 'Save'}
          </button>
        </div>
        <a href={provider.url} target="_blank" class="api-link">Get API Key →</a>
      </div>
    {/each}
  </div>
</section>

<style>
  /* Providers Grid */
  .providers-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
    gap: 12px;
  }

  .provider-card {
    position: relative;
    display: flex;
    align-items: center;
    gap: 14px;
    padding: 18px;
    background: var(--color-bg);
    border: 1px solid var(--color-border);
    border-radius: 12px;
    cursor: pointer;
    transition: all 0.2s;
    text-align: left;
  }

  .provider-card:hover {
    border-color: var(--color-border-hover);
  }

  .provider-card.active {
    border-color: var(--color-primary);
    background: rgba(0, 0, 0, 0.02);
  }

  .provider-card.disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .provider-icon {
    width: 44px;
    height: 44px;
    display: flex;
    align-items: center;
    justify-content: center;
    background: rgba(255, 255, 255, 0.05);
    border-radius: 10px;
    flex-shrink: 0;
  }

  .provider-icon svg {
    width: 22px;
    height: 22px;
    color: var(--color-text-muted);
  }

  :global(.dark) .provider-icon {
    background: rgba(255, 255, 255, 0.05);
  }

  :global(:not(.dark)) .provider-icon {
    background: rgba(0, 0, 0, 0.03);
  }

  .provider-info {
    flex: 1;
    display: flex;
    flex-direction: column;
    gap: 2px;
  }

  .provider-name {
    font-weight: 600;
    font-size: 14px;
  }

  .provider-desc {
    font-size: 12px;
    color: var(--color-text-muted);
  }

  .provider-status .status {
    padding: 4px 10px;
    border-radius: 6px;
    font-size: 11px;
    font-weight: 500;
  }

  .status.active {
    background: rgba(34, 197, 94, 0.1);
    color: var(--color-success);
  }

  .status.ready {
    background: var(--color-bg-tertiary);
    color: var(--color-text-secondary);
  }

  .status.setup {
    background: rgba(245, 158, 11, 0.1);
    color: #f59e0b;
  }

  .provider-badge {
    position: absolute;
    top: 8px;
    right: 8px;
    padding: 2px 8px;
    font-size: 10px;
    font-weight: 600;
    border-radius: 4px;
    background: rgba(59, 130, 246, 0.1);
    color: #3b82f6;
  }

  .provider-badge.local {
    background: rgba(34, 197, 94, 0.1);
    color: var(--color-success);
  }

  .access-toggle {
    display: inline-flex;
    align-items: center;
    gap: 2px;
    padding: 2px;
    background: var(--color-bg-tertiary);
    border: 1px solid var(--color-border);
    border-radius: 8px;
  }

  .access-toggle button {
    border: 0;
    border-radius: 6px;
    background: transparent;
    color: var(--color-text-secondary);
    cursor: pointer;
    font-size: 12px;
    font-weight: 500;
    padding: 6px 10px;
  }

  .access-toggle button.active {
    background: var(--color-bg);
    color: var(--color-text);
    box-shadow: 0 1px 2px rgba(0, 0, 0, 0.08);
  }

  .access-toggle button:disabled {
    cursor: not-allowed;
    opacity: 0.45;
  }

  /* API Grid */
  .api-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
    gap: 12px;
  }

  .api-card {
    padding: 18px;
    background: var(--color-bg);
    border: 1px solid var(--color-border);
    border-radius: 12px;
  }

  .api-header {
    display: flex;
    align-items: center;
    gap: 10px;
    margin-bottom: 14px;
  }

  .api-icon {
    width: 38px;
    height: 38px;
    display: flex;
    align-items: center;
    justify-content: center;
    background: rgba(255, 255, 255, 0.05);
    border-radius: 8px;
    flex-shrink: 0;
  }

  .api-icon svg {
    width: 20px;
    height: 20px;
    color: var(--color-text-muted);
  }

  :global(.dark) .api-icon {
    background: rgba(255, 255, 255, 0.05);
  }

  :global(:not(.dark)) .api-icon {
    background: rgba(0, 0, 0, 0.03);
  }

  .api-name {
    font-weight: 600;
    flex: 1;
  }

  .api-configured {
    min-width: 22px;
    height: 22px;
    padding: 0 8px;
    display: flex;
    align-items: center;
    justify-content: center;
    background: rgba(34, 197, 94, 0.1);
    border-radius: 999px;
    color: var(--color-success);
    font-size: 10px;
    font-weight: 600;
  }

  .api-form {
    display: flex;
    gap: 8px;
    margin-bottom: 10px;
  }

  .input-wrapper {
    flex: 1;
    position: relative;
  }

  .input-wrapper input {
    width: 100%;
    padding: 10px 36px 10px 12px;
    background: var(--color-bg-secondary);
    border: 1px solid var(--color-border);
    border-radius: 8px;
    font-size: 13px;
    font-family: monospace;
    color: var(--color-text);
  }

  .input-wrapper input:focus {
    outline: none;
    border-color: var(--color-primary);
  }

  .input-wrapper input::placeholder { color: var(--color-text-muted); }

  .toggle-btn {
    position: absolute;
    right: 8px;
    top: 50%;
    transform: translateY(-50%);
    background: none;
    border: none;
    padding: 4px;
    cursor: pointer;
    color: var(--color-text-muted);
  }

  .toggle-btn:hover { color: var(--color-text); }
  .toggle-btn svg { width: 16px; height: 16px; }

  .save-btn {
    padding: 10px 16px;
    background: var(--color-primary);
    color: var(--color-bg);
    border: none;
    border-radius: 8px;
    font-size: 13px;
    font-weight: 500;
    cursor: pointer;
  }

  .save-btn:disabled { opacity: 0.5; cursor: not-allowed; }

  .api-link {
    font-size: 12px;
    color: var(--color-text-muted);
    text-decoration: none;
  }

  .api-link:hover { color: var(--color-primary); }

  /* Section layout */
  .section {
    margin-bottom: 32px;
  }

  .section-header {
    display: flex;
    align-items: center;
    gap: 12px;
    margin-bottom: 16px;
  }

  .section-header h2 {
    font-size: 16px;
    font-weight: 600;
    margin: 0;
  }

  .badge {
    padding: 4px 10px;
    border-radius: 6px;
    font-size: 12px;
    font-weight: 500;
    background: var(--color-bg-tertiary);
    color: var(--color-text-secondary);
  }

  .badge.active {
    background: rgba(34, 197, 94, 0.1);
    color: var(--color-success);
  }

  .subtitle {
    font-size: 13px;
    color: var(--color-text-muted);
  }

  /* Dark mode */
  :global(.dark) .provider-card,
  :global(.dark) .api-card {
    background: #2c2c2e;
    border-color: rgba(255, 255, 255, 0.1);
  }

  :global(.dark) .provider-card:hover,
  :global(.dark) .provider-card.active {
    border-color: rgba(255, 255, 255, 0.2);
  }

  :global(.dark) .provider-card.active,
  :global(.dark) .provider-card.expanded {
    border-color: #0A84FF;
    background: rgba(10, 132, 255, 0.1);
  }

  :global(.dark) .save-btn {
    background: #0A84FF;
    color: white;
  }
</style>
