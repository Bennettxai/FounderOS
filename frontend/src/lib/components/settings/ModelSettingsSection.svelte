<script lang="ts">
  import OutputStyleSelector from './OutputStyleSelector.svelte';
  import UserFactsPanel from './UserFactsPanel.svelte';
  import type {
    LLMModel,
    LLMProvider,
    OutputStyle,
    OutputPreference,
  } from '$lib/stores/aiSettings';
  import { cloudModels } from '$lib/stores/aiSettings';

  interface ModelSettings {
    temperature: number;
    maxTokens: number;
    contextWindow: number;
    topP: number;
    streamResponses: boolean;
    showUsageInChat: boolean;
  }

  interface Props {
    models: LLMModel[];
    providers: LLMProvider[];
    activeProvider: string;
    defaultModel: string;
    outputStyles: OutputStyle[];
    outputPreference: OutputPreference | null;
    selectedDefaultStyleId: string;
    loadingOutputStyles: boolean;
    savingOutputPreference: boolean;
    modelSettings: ModelSettings;
    isSaving: boolean;
    onSaveSettings: () => void;
    onSaveOutputPreference: () => void;
    onSelectDefaultStyleId: (id: string) => void;
    onUpdateModelSettings: (settings: ModelSettings) => void;
    onSetDefaultModel: (modelId: string) => void;
  }

  let {
    models,
    activeProvider,
    defaultModel,
    outputStyles,
    selectedDefaultStyleId,
    loadingOutputStyles,
    savingOutputPreference,
    modelSettings,
    isSaving,
    onSaveSettings,
    onSaveOutputPreference,
    onSelectDefaultStyleId,
    onUpdateModelSettings,
    onSetDefaultModel,
  }: Props = $props();

  function getLocalModels(): LLMModel[] {
    return (models || []).filter((m) => {
      const isLocalProvider = m.provider === 'ollama' || m.provider === 'ollama_local';
      const nameOrId = (m.id || '') + (m.name || '');
      const isCloudRef =
        nameOrId.toLowerCase().includes('cloud') &&
        (m.size === '< 1 KB' || m.size === '0 B' || !m.size);
      return isLocalProvider && !isCloudRef;
    });
  }
</script>

<!-- Settings Tab -->
<section class="section">
  <div class="section-header">
    <h2>Model Settings</h2>
    <span class="subtitle">Fine-tune AI behavior</span>
  </div>

  <div class="setting-card" style="margin-bottom: 16px;">
    <div class="setting-header">
      <label>Default Output Style</label>
    </div>
    <p class="setting-desc" style="margin-bottom: 1rem;">Choose how the AI formats its responses.</p>
    {#if loadingOutputStyles}
      <div class="loading" style="padding: 2rem; text-align: center;">
        <div class="spinner"></div>
        <span>Loading output styles...</span>
      </div>
    {:else}
      <OutputStyleSelector
        styles={outputStyles}
        selectedStyleId={selectedDefaultStyleId}
        onSelect={(styleId) => onSelectDefaultStyleId(styleId)}
        disabled={savingOutputPreference}
      />
      <button class="btn-pill btn-pill-ghost save-btn" style="margin-top: 1rem;" onclick={onSaveOutputPreference} disabled={savingOutputPreference}>
        {savingOutputPreference ? 'Saving...' : 'Save Default Style'}
      </button>
    {/if}
  </div>

  <div class="setting-card" style="margin-bottom: 16px; padding: 0; overflow: hidden; max-height: 600px;">
    <UserFactsPanel />
  </div>

  <div class="presets-row">
    <span class="presets-label">Quick Presets:</span>
    <button class="btn-pill btn-pill-ghost preset-btn" onclick={() => onUpdateModelSettings({ ...modelSettings, temperature: 0.3, maxTokens: 4096, topP: 0.8, contextWindow: 16384 })}>
      <svg class="preset-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polygon points="13 2 3 14 12 14 11 22 21 10 12 10 13 2"/></svg>
      Fast
    </button>
    <button class="btn-pill btn-pill-ghost preset-btn" onclick={() => onUpdateModelSettings({ ...modelSettings, temperature: 0.7, maxTokens: 8192, topP: 0.95, contextWindow: 32768 })}>
      <svg class="preset-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M12 2L2 7l10 5 10-5-10-5zM2 17l10 5 10-5M2 12l10 5 10-5"/></svg>
      Balanced
    </button>
    <button class="btn-pill btn-pill-ghost preset-btn" onclick={() => onUpdateModelSettings({ ...modelSettings, temperature: 0.9, maxTokens: 16384, topP: 1.0, contextWindow: 65536 })}>
      <svg class="preset-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"/><circle cx="12" cy="12" r="6"/><circle cx="12" cy="12" r="2"/></svg>
      Quality
    </button>
    <button class="btn-pill btn-pill-ghost preset-btn" onclick={() => onUpdateModelSettings({ ...modelSettings, temperature: 1.0, maxTokens: 32768, topP: 1.0, contextWindow: 131072 })}>
      <svg class="preset-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M12 2L2 19h20L12 2z"/><path d="M12 9v4"/><circle cx="12" cy="16" r="1"/></svg>
      Maximum
    </button>
  </div>

  <div class="settings-grid">
    <div class="setting-card">
      <div class="setting-header">
        <label for="contextWindow">Context Window</label>
        <span class="setting-value">{(modelSettings.contextWindow / 1000).toFixed(0)}K tokens</span>
      </div>
      <input type="range" id="contextWindow" value={modelSettings.contextWindow} oninput={(e) => onUpdateModelSettings({ ...modelSettings, contextWindow: Number((e.target as HTMLInputElement).value) })} min="4096" max="131072" step="4096" />
      <p class="setting-desc">How much conversation history to include</p>
    </div>
    <div class="setting-card">
      <div class="setting-header">
        <label for="maxTokens">Max Output Tokens</label>
        <span class="setting-value">{(modelSettings.maxTokens / 1000).toFixed(1)}K</span>
      </div>
      <input type="range" id="maxTokens" value={modelSettings.maxTokens} oninput={(e) => onUpdateModelSettings({ ...modelSettings, maxTokens: Number((e.target as HTMLInputElement).value) })} min="512" max="32768" step="512" />
      <p class="setting-desc">Maximum length of AI responses</p>
    </div>
    <div class="setting-card">
      <div class="setting-header">
        <label for="temperature">Temperature</label>
        <span class="setting-value">{modelSettings.temperature}</span>
      </div>
      <input type="range" id="temperature" value={modelSettings.temperature} oninput={(e) => onUpdateModelSettings({ ...modelSettings, temperature: Number((e.target as HTMLInputElement).value) })} min="0" max="2" step="0.1" />
      <p class="setting-desc">Lower = more focused, Higher = more creative</p>
    </div>
    <div class="setting-card">
      <div class="setting-header">
        <label for="topP">Top P (Nucleus Sampling)</label>
        <span class="setting-value">{modelSettings.topP}</span>
      </div>
      <input type="range" id="topP" value={modelSettings.topP} oninput={(e) => onUpdateModelSettings({ ...modelSettings, topP: Number((e.target as HTMLInputElement).value) })} min="0.1" max="1" step="0.05" />
      <p class="setting-desc">Controls diversity of responses</p>
    </div>
    <div class="setting-card toggle-card">
      <div class="setting-header">
        <label for="streaming">Stream Responses</label>
        <button class="btn-pill btn-pill-ghost toggle" class:on={modelSettings.streamResponses} onclick={() => onUpdateModelSettings({ ...modelSettings, streamResponses: !modelSettings.streamResponses })}>
          <span class="toggle-knob"></span>
        </button>
      </div>
      <p class="setting-desc">Show responses as they're generated</p>
    </div>
    <div class="setting-card toggle-card">
      <div class="setting-header">
        <label for="showUsage">Show Usage Stats</label>
        <button class="btn-pill btn-pill-ghost toggle" class:on={modelSettings.showUsageInChat} onclick={() => onUpdateModelSettings({ ...modelSettings, showUsageInChat: !modelSettings.showUsageInChat })}>
          <span class="toggle-knob"></span>
        </button>
      </div>
      <p class="setting-desc">Display tokens/second and token count after each response</p>
    </div>
  </div>

  <div class="settings-actions">
    <button class="btn-pill btn-pill-ghost action-btn primary" onclick={onSaveSettings} disabled={isSaving}>
      {isSaving ? 'Saving...' : 'Save Settings'}
    </button>
    <button class="btn-pill btn-pill-ghost action-btn" onclick={() => onUpdateModelSettings({ temperature: 0.7, maxTokens: 8192, topP: 0.95, contextWindow: 32768, streamResponses: true, showUsageInChat: true })}>
      Reset to Defaults
    </button>
  </div>
</section>

<!-- Default Model Selection -->
<section class="section">
  <div class="section-header">
    <h2>Default Model</h2>
    <span class="subtitle">Select your preferred model</span>
  </div>
  <div class="default-model-grid">
    {#if activeProvider === 'ollama_local'}
      {#each getLocalModels() as model}
        <button
          class="btn-pill btn-pill-ghost default-model-btn"
          class:selected={defaultModel === model.id}
          onclick={() => { onSetDefaultModel(model.id); onSaveSettings(); }}
        >
          <span class="dm-name">{model.name}</span>
          {#if model.size}<span class="dm-size">{model.size}</span>{/if}
          {#if defaultModel === model.id}
            <svg class="dm-check" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3"><polyline points="20 6 9 17 4 12"/></svg>
          {/if}
        </button>
      {/each}
    {:else}
      {#each cloudModels[activeProvider] || [] as model}
        <button
          class="btn-pill btn-pill-ghost default-model-btn"
          class:selected={defaultModel === model.id}
          onclick={() => { onSetDefaultModel(model.id); onSaveSettings(); }}
        >
          <span class="dm-name">{model.name}</span>
          <span class="dm-desc">{model.description}</span>
          {#if defaultModel === model.id}
            <svg class="dm-check" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3"><polyline points="20 6 9 17 4 12"/></svg>
          {/if}
        </button>
      {/each}
    {/if}
  </div>
</section>

<style>
  /* Loading */
  .loading {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 16px;
    color: var(--color-text-muted);
  }
  .spinner {
    width: 32px;
    height: 32px;
    border: 3px solid var(--color-border);
    border-top-color: var(--color-primary);
    border-radius: 50%;
    animation: spin 0.8s linear infinite;
  }
  @keyframes spin { to { transform: rotate(360deg); } }

  /* Sections */
  .section { margin-bottom: 32px; }
  .section-header { display: flex; align-items: center; gap: 12px; margin-bottom: 16px; }
  .section-header h2 { font-size: 18px; font-weight: 600; margin: 0; }
  .subtitle { font-size: 13px; color: var(--color-text-muted); }

  /* Settings Tab */
  .presets-row { display: flex; align-items: center; gap: 8px; margin-bottom: 16px; flex-wrap: wrap; }
  .presets-label { font-size: 13px; color: var(--color-text-tertiary); margin-right: 4px; }
  .preset-btn { display: flex; align-items: center; gap: 6px; padding: 8px 14px; font-size: 12px; font-weight: 500; border-radius: 8px; border: 1px solid var(--color-border); background: rgba(255,255,255,0.05); color: var(--color-text-secondary); cursor: pointer; transition: all 0.15s ease; }
  :global(.dark) .preset-btn { background: rgba(255,255,255,0.05); border-color: var(--dbd); }
  :global(:not(.dark)) .preset-btn { background: rgba(0,0,0,0.02); }
  .preset-btn:hover { background: rgba(255,255,255,0.1); border-color: var(--color-primary); color: var(--color-primary); }
  :global(:not(.dark)) .preset-btn:hover { background: rgba(0,0,0,0.05); }
  .preset-icon { width: 16px; height: 16px; flex-shrink: 0; }

  .settings-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(280px, 1fr)); gap: 12px; }
  .setting-card { padding: 18px; background: var(--bos-settings-card-bg); border: 1px solid var(--bos-settings-card-border); border-radius: var(--bos-settings-card-radius); }
  .setting-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 12px; }
  .setting-header label { font-weight: 500; font-size: 14px; }
  .setting-value { font-size: 14px; font-weight: 600; color: var(--color-primary); font-family: monospace; }
  .setting-card input[type="range"] { width: 100%; height: 6px; background: var(--color-border); border-radius: 3px; appearance: none; cursor: pointer; }
  .setting-card input[type="range"]::-webkit-slider-thumb { appearance: none; width: 18px; height: 18px; background: var(--color-primary); border-radius: 50%; cursor: pointer; }
  .setting-desc { margin: 10px 0 0; font-size: 12px; color: var(--color-text-muted); }
  .toggle-card .setting-header { margin-bottom: 0; }
  .toggle { width: 44px; height: 24px; background: var(--color-border); border: none; border-radius: 12px; cursor: pointer; position: relative; transition: all 0.2s; }
  .toggle.on { background: var(--color-success); }
  .toggle-knob { position: absolute; top: 2px; left: 2px; width: 20px; height: 20px; background: white; border-radius: 50%; transition: all 0.2s; }
  .toggle.on .toggle-knob { left: 22px; }
  .toggle-card .setting-desc { margin-top: 12px; }
  .settings-actions { display: flex; gap: 10px; margin-top: 20px; }
  .action-btn { padding: 12px 24px; background: var(--color-bg-tertiary); border: 1px solid var(--color-border); border-radius: 10px; font-size: 14px; font-weight: 500; cursor: pointer; transition: all 0.2s; color: var(--color-text); }
  .action-btn:hover { background: var(--color-bg-secondary); }
  .action-btn.primary { background: var(--color-primary); color: var(--color-bg); border-color: transparent; }
  .action-btn.primary:hover { opacity: 0.9; }
  .action-btn:disabled { opacity: 0.5; cursor: not-allowed; }
  .save-btn { padding: 10px 16px; background: var(--color-primary); color: var(--color-bg); border: none; border-radius: 8px; font-size: 13px; font-weight: 500; cursor: pointer; }
  .save-btn:disabled { opacity: 0.5; cursor: not-allowed; }
  :global(.dark) .setting-value { color: var(--bos-nav-active); }

  /* Default Model */
  .default-model-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(200px, 1fr)); gap: 10px; }
  .default-model-btn { position: relative; display: flex; flex-direction: column; gap: 4px; padding: 14px; background: var(--color-bg); border: 1px solid var(--color-border); border-radius: 10px; text-align: left; cursor: pointer; transition: all 0.2s; }
  .default-model-btn:hover { border-color: var(--color-border-hover); }
  .default-model-btn.selected { border-color: var(--color-primary); background: rgba(0,0,0,0.02); }
  .dm-name { font-weight: 600; font-size: 14px; }
  .dm-size, .dm-desc { font-size: 12px; color: var(--color-text-muted); }
  .dm-check { position: absolute; top: 10px; right: 10px; width: 18px; height: 18px; color: var(--color-success); }
  :global(.dark) .default-model-btn.selected { border-color: var(--bos-nav-active); background: rgba(10,132,255,0.1); }
</style>
