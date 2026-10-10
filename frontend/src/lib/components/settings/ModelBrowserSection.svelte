<script lang="ts">
  import { apiClient } from '$lib/api';
  import type {
    LLMModel,
    PullProgress,
    SystemInfo,
    AvailableModel,
    ModelCapability,
  } from '$lib/stores/aiSettings';
  import {
    availableModels,
    capabilityInfo,
    formatBytes,
  } from '$lib/stores/aiSettings';
  import ModelBrowserGrid from './ModelBrowserGrid.svelte';
  import ModelPullForm from './ModelPullForm.svelte';
  import ModelBrowserFilterBar from './ModelBrowserFilterBar.svelte';

  interface Props {
    models: LLMModel[];
    activeProvider: string;
    defaultModel: string;
    systemInfo: SystemInfo | null;
    onSetDefaultModel: (modelId: string) => void;
    onSaveSettings: () => void;
  }

  let {
    models,
    activeProvider,
    defaultModel,
    systemInfo,
    onSetDefaultModel,
    onSaveSettings,
  }: Props = $props();

  // Filter/sort state
  let modelSearchQuery = $state('');
  let selectedCapabilityFilters = $state<ModelCapability[]>([]);
  let selectedProviderFilter = $state<'all' | 'local' | 'cloud'>('all');
  let modelSortBy = $state<'recommended' | 'name' | 'size' | 'downloads'>('recommended');
  let showOnlyInstalled = $state(false);
  let selectedVariants = $state<Record<string, string>>({});

  // Pull model state
  let pullModelName = $state('');
  let isPulling = $state(false);
  let pullProgress = $state<PullProgress | null>(null);
  let pullError = $state('');
  let pullSpeed = $state<string>('');

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

  function getFilteredModels(): AvailableModel[] {
    let filtered = [...availableModels];
    const installedIds = new Set(models.map((m) => m.id.toLowerCase()));
    filtered = filtered.map((m) => ({
      ...m,
      isInstalled:
        installedIds.has(m.id.toLowerCase()) ||
        installedIds.has(m.id.split(':')[0].toLowerCase()),
    }));

    if (modelSearchQuery) {
      const query = modelSearchQuery.toLowerCase();
      filtered = filtered.filter(
        (m) =>
          m.name.toLowerCase().includes(query) ||
          m.description.toLowerCase().includes(query) ||
          m.id.toLowerCase().includes(query),
      );
    }
    if (selectedCapabilityFilters.length > 0) {
      filtered = filtered.filter((m) =>
        selectedCapabilityFilters.some((cap) => m.capabilities.includes(cap)),
      );
    }
    if (selectedProviderFilter !== 'all') {
      filtered = filtered.filter((m) => m.provider === selectedProviderFilter);
    }
    if (showOnlyInstalled) {
      filtered = filtered.filter((m) => m.isInstalled);
    }

    switch (modelSortBy) {
      case 'name':
        filtered.sort((a, b) => a.name.localeCompare(b.name));
        break;
      case 'size':
        filtered.sort((a, b) => (parseFloat(a.size) || 999) - (parseFloat(b.size) || 999));
        break;
      case 'downloads':
        filtered.sort((a, b) => parseDownloads(b.downloads) - parseDownloads(a.downloads));
        break;
      default:
        filtered.sort((a, b) => {
          if (a.isInstalled && !b.isInstalled) return -1;
          if (!a.isInstalled && b.isInstalled) return 1;
          return parseDownloads(b.downloads) - parseDownloads(a.downloads);
        });
    }
    return filtered;
  }

  function parseDownloads(d: string | undefined): number {
    if (!d) return 0;
    const num = parseFloat(d);
    if (d.includes('M')) return num * 1000000;
    if (d.includes('K')) return num * 1000;
    return num;
  }

  async function pullModel(modelId: string) {
    if (!modelId.trim() || isPulling) return;
    isPulling = true;
    pullError = '';
    pullProgress = { status: 'Connecting...' };
    const model = modelId.trim();

    try {
      const response = await fetch(`/api/ai/models/pull`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ model }),
        credentials: 'include',
      });
      if (!response.ok) {
        const data = await response.json();
        throw new Error(data.error || 'Failed to pull model');
      }
      const reader = response.body?.getReader();
      if (!reader) throw new Error('No response body');
      const decoder = new TextDecoder();
      let buffer = '';
      let lastCompleted = 0;
      let lastTime = Date.now();
      while (true) {
        const { done, value } = await reader.read();
        if (done) break;
        buffer += decoder.decode(value, { stream: true });
        const lines = buffer.split('\n');
        buffer = lines.pop() || '';
        for (const line of lines) {
          if (line.startsWith('data: ')) {
            try {
              const data = JSON.parse(line.slice(6));
              pullProgress = data;
              if (data.completed && data.completed > lastCompleted) {
                const now = Date.now();
                const timeDiff = (now - lastTime) / 1000;
                const bytesDiff = data.completed - lastCompleted;
                if (timeDiff > 0) pullSpeed = formatBytes(bytesDiff / timeDiff) + '/s';
                lastCompleted = data.completed;
                lastTime = now;
              }
              if (data.status === 'complete' || data.status === 'success') {
                pullProgress = { status: 'Complete!' };
                pullModelName = '';
              }
            } catch {}
          }
        }
      }
    } catch (err) {
      pullError = err instanceof Error ? err.message : 'Failed to pull model';
      pullProgress = null;
    } finally {
      isPulling = false;
      pullSpeed = '';
    }
  }

  async function deleteModel(modelId: string) {
    if (!confirm(`Delete model ${modelId}?`)) return;
    try {
      const res = await apiClient.delete(`/ai/models/${encodeURIComponent(modelId)}`);
      if (!res.ok) console.error('Failed to delete model');
    } catch (err) {
      console.error('Failed to delete model:', err);
    }
  }
</script>

<section class="section model-browser-section">
  <ModelBrowserFilterBar
    {modelSearchQuery}
    {selectedCapabilityFilters}
    {selectedProviderFilter}
    {modelSortBy}
    {showOnlyInstalled}
    onSearchChange={(q) => { modelSearchQuery = q; }}
    onCapabilityFiltersChange={(f) => { selectedCapabilityFilters = f; }}
    onProviderFilterChange={(f) => { selectedProviderFilter = f; }}
    onSortByChange={(s) => { modelSortBy = s; }}
    onShowOnlyInstalledChange={(v) => { showOnlyInstalled = v; }}
  />

  {#if systemInfo && systemInfo.recommended_models.length > 0 && activeProvider === 'ollama_local' && !modelSearchQuery && selectedCapabilityFilters.length === 0}
    <div class="recommended-banner">
      <div class="rec-banner-header">
        <h3>Recommended for You</h3>
        <span class="rec-badge-info">Based on {systemInfo.total_ram_gb}GB RAM</span>
      </div>
      <div class="rec-chips">
        {#each systemInfo.recommended_models.slice(0, 4) as model}
          {@const isInstalled = getLocalModels().some(m => m.id.startsWith(model.name.split(':')[0]))}
          <button
            class="btn-pill btn-pill-ghost rec-chip"
            class:installed={isInstalled}
            onclick={() => { if (!isInstalled) pullModel(model.name); }}
            disabled={isPulling}
          >
            <span class="chip-name">{model.name}</span>
            <span class="chip-meta">{model.speed} • {model.quality}</span>
            {#if isInstalled}
              <span class="chip-status installed">Installed</span>
            {:else}
              <span class="chip-status pull">Pull</span>
            {/if}
          </button>
        {/each}
      </div>
    </div>
  {/if}

  <div class="browser-content">
    <div class="browser-header">
      <h3>
        {#if showOnlyInstalled}
          Installed Models
        {:else if selectedCapabilityFilters.length === 1}
          {capabilityInfo[selectedCapabilityFilters[0]].label} Models
        {:else if selectedCapabilityFilters.length > 1}
          Filtered Models
        {:else}
          All Models
        {/if}
      </h3>
      <span class="model-count">{getFilteredModels().length} models</span>
    </div>

    <ModelBrowserGrid
      filteredModels={getFilteredModels()}
      {defaultModel}
      {selectedVariants}
      {isPulling}
      {pullModelName}
      {onSetDefaultModel}
      {onSaveSettings}
      onPullModel={pullModel}
      onDeleteModel={deleteModel}
      onSelectVariant={(modelId, variantId) => { selectedVariants[modelId] = variantId; }}
    />
  </div>

  <ModelPullForm
    {pullModelName}
    {isPulling}
    {pullProgress}
    {pullError}
    {pullSpeed}
    onPullModelNameChange={(name) => { pullModelName = name; }}
    onPull={() => pullModel(pullModelName)}
  />
</section>

<style>
  .section { margin-bottom: 32px; }
  .model-browser-section { display: flex; flex-direction: column; gap: 20px; }
  .recommended-banner { padding: 16px; background: linear-gradient(135deg, rgba(59,130,246,0.08), rgba(139,92,246,0.08)); border: 1px solid rgba(59,130,246,0.15); border-radius: 12px; }
  :global(.dark) .recommended-banner { background: linear-gradient(135deg, rgba(59,130,246,0.15), rgba(139,92,246,0.15)); border-color: rgba(59,130,246,0.25); }
  .rec-banner-header { display: flex; align-items: center; gap: 12px; margin-bottom: 12px; }
  .rec-banner-header h3 { margin: 0; font-size: 15px; font-weight: 600; }
  .rec-badge-info { font-size: 11px; padding: 3px 10px; background: rgba(59,130,246,0.12); border-radius: 10px; color: #3b82f6; }
  :global(.dark) .rec-badge-info { background: rgba(59,130,246,0.2); color: #60a5fa; }
  .rec-chips { display: flex; flex-wrap: wrap; gap: 10px; }
  .rec-chip { display: flex; flex-direction: column; gap: 4px; padding: 12px 16px; background: var(--color-bg); border: 1px solid var(--color-border); border-radius: 10px; cursor: pointer; transition: all 0.2s; min-width: 160px; }
  .rec-chip:hover { border-color: var(--color-primary); transform: translateY(-2px); }
  .rec-chip.installed { border-color: rgba(34,197,94,0.4); background: rgba(34,197,94,0.05); }
  .rec-chip:disabled { opacity: 0.7; cursor: not-allowed; }
  .chip-name { font-weight: 600; font-size: 13px; }
  .chip-meta { font-size: 11px; color: var(--color-text-muted); }
  .chip-status { font-size: 10px; padding: 2px 8px; border-radius: 4px; width: fit-content; }
  .chip-status.installed { background: rgba(34,197,94,0.1); color: var(--color-success); }
  .chip-status.pull { background: rgba(59,130,246,0.1); color: #3b82f6; }
  .browser-content { display: flex; flex-direction: column; gap: 16px; }
  .browser-header { display: flex; align-items: center; justify-content: space-between; }
  .browser-header h3 { margin: 0; font-size: 16px; font-weight: 600; }
  .model-count { font-size: 13px; color: var(--color-text-muted); }
</style>
