<script lang="ts">
  import type { LLMModel, AvailableModel, ModelCapability } from '$lib/stores/aiSettings';
  import { capabilityInfo, formatBytes } from '$lib/stores/aiSettings';

  interface Props {
    filteredModels: AvailableModel[];
    defaultModel: string;
    selectedVariants: Record<string, string>;
    isPulling: boolean;
    pullModelName: string;
    onSetDefaultModel: (modelId: string) => void;
    onSaveSettings: () => void;
    onPullModel: (modelId: string) => void;
    onDeleteModel: (modelId: string) => void;
    onSelectVariant: (modelId: string, variantId: string) => void;
  }

  let {
    filteredModels,
    defaultModel,
    selectedVariants,
    isPulling,
    pullModelName,
    onSetDefaultModel,
    onSaveSettings,
    onPullModel,
    onDeleteModel,
    onSelectVariant,
  }: Props = $props();

  function getSelectedVariant(model: AvailableModel): string {
    if (!model.variants || model.variants.length === 0) return model.id;
    return selectedVariants[model.id] || model.variants[0].id;
  }

  function getVariantInfo(model: AvailableModel) {
    if (!model.variants || model.variants.length === 0) return null;
    const selectedId = getSelectedVariant(model);
    return model.variants.find((v) => v.id === selectedId) || model.variants[0];
  }
</script>

{#if filteredModels.length === 0}
  <div class="empty-state">
    <div class="empty-icon">
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="11" cy="11" r="8"/><path d="m21 21-4.35-4.35"/></svg>
    </div>
    <h3>No models found</h3>
    <p>Try adjusting your search or filters</p>
  </div>
{:else}
  <div class="model-browser-grid">
    {#each filteredModels as model}
      {@const isDefault = defaultModel === model.id || defaultModel.startsWith(model.id.split(':')[0])}
      {@const variantInfo = getVariantInfo(model)}
      {@const displaySize = variantInfo ? variantInfo.size : model.size}
      {@const displayParams = variantInfo ? variantInfo.params : model.params}
      {@const pullId = getSelectedVariant(model)}
      <div class="browser-model-card" class:installed={model.isInstalled} class:default={isDefault}>
        <div class="bmc-header">
          <div class="bmc-title">
            <span class="bmc-name">{model.name}</span>
            {#if model.isInstalled}
              <span class="bmc-installed-badge">Installed</span>
            {/if}
          </div>
          <div class="bmc-meta">
            <span class="bmc-size">{displaySize}</span>
            <span class="bmc-params">{displayParams}</span>
          </div>
        </div>
        <p class="bmc-description">{model.description}</p>
        <div class="bmc-capabilities">
          {#each model.capabilities as cap}
            {@const info = capabilityInfo[cap]}
            <span class="cap-badge {cap}" title={info.label}>
              <svg class="cap-badge-icon-svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d={info.iconPath}/></svg>
              <span class="cap-badge-label">{info.label}</span>
            </span>
          {/each}
        </div>
        {#if model.variants && model.variants.length > 1}
          <div class="bmc-variants">
            <span class="variants-label">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="12" height="12"><path d="M12 2L2 7l10 5 10-5-10-5zM2 17l10 5 10-5M2 12l10 5 10-5"/></svg>
              Select size:
            </span>
            <div class="variant-buttons" class:many={model.variants.length > 4}>
              {#each model.variants as variant}
                <button
                  class="variant-btn"
                  class:selected={getSelectedVariant(model) === variant.id}
                  onclick={() => onSelectVariant(model.id, variant.id)}
                  title="{variant.params} parameters • {variant.size} download"
                >
                  <span class="variant-params">{variant.params}</span>
                  <span class="variant-size">{variant.size}</span>
                </button>
              {/each}
            </div>
          </div>
        {/if}
        <div class="bmc-footer">
          {#if model.downloads}
            <span class="bmc-downloads">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="12" height="12"><path d="M21 15v4a2 2 0 01-2 2H5a2 2 0 01-2-2v-4M7 10l5 5 5-5M12 15V3"/></svg>
              {model.downloads}
            </span>
          {/if}
          {#if model.isInstalled}
            <div class="bmc-actions">
              <button
                class="bmc-btn default-btn"
                class:is-default={isDefault}
                onclick={() => { onSetDefaultModel(model.id); onSaveSettings(); }}
              >
                {isDefault ? 'Default' : 'Set Default'}
              </button>
              <button class="bmc-btn delete-btn" onclick={() => onDeleteModel(model.id)} title="Delete model" aria-label="Delete model {model.name}">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M3 6h18M19 6v14a2 2 0 01-2 2H7a2 2 0 01-2-2V6m3 0V4a2 2 0 012-2h4a2 2 0 012 2v2"/></svg>
              </button>
            </div>
          {:else}
            <button class="bmc-btn pull-btn" onclick={() => onPullModel(pullId)} disabled={isPulling}>
              {#if isPulling && pullModelName === pullId}
                <div class="btn-spinner-small"></div>
                Pulling...
              {:else}
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M21 15v4a2 2 0 01-2 2H5a2 2 0 01-2-2v-4M7 10l5 5 5-5M12 15V3"/></svg>
                Pull Model
              {/if}
            </button>
          {/if}
        </div>
      </div>
    {/each}
  </div>
{/if}

<style>
  /* Empty State */
  .empty-state { text-align: center; padding: 48px; background: var(--color-bg); border: 1px dashed var(--color-border); border-radius: 12px; }
  .empty-icon { width: 48px; height: 48px; margin-bottom: 12px; color: var(--color-text-muted); }
  .empty-icon svg { width: 100%; height: 100%; }
  .empty-state h3 { margin: 0 0 8px; font-size: 16px; }
  .empty-state p { margin: 0; color: var(--color-text-muted); font-size: 14px; }

  /* Grid */
  .model-browser-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(280px, 1fr)); gap: 14px; }
  @keyframes fadeInUp { from { opacity: 0; transform: translateY(10px); } to { opacity: 1; transform: translateY(0); } }
  .model-browser-grid .browser-model-card { animation: fadeInUp 0.3s ease backwards; }
  .model-browser-grid .browser-model-card:nth-child(1) { animation-delay: 0.02s; }
  .model-browser-grid .browser-model-card:nth-child(2) { animation-delay: 0.04s; }
  .model-browser-grid .browser-model-card:nth-child(3) { animation-delay: 0.06s; }
  .model-browser-grid .browser-model-card:nth-child(4) { animation-delay: 0.08s; }
  .model-browser-grid .browser-model-card:nth-child(n+5) { animation-delay: 0.1s; }

  .browser-model-card { display: flex; flex-direction: column; gap: 10px; padding: 16px; background: var(--color-bg); border: 1px solid var(--color-border); border-radius: 14px; transition: all 0.25s ease; position: relative; overflow: hidden; }
  .browser-model-card::before { content: ''; position: absolute; top: 0; left: 0; right: 0; height: 3px; background: transparent; transition: background 0.25s ease; }
  .browser-model-card:hover { border-color: var(--color-border-hover); box-shadow: 0 8px 24px rgba(0,0,0,0.08); transform: translateY(-2px); }
  :global(.dark) .browser-model-card:hover { box-shadow: 0 8px 24px rgba(0,0,0,0.25); }
  .browser-model-card.installed { border-color: rgba(34,197,94,0.35); background: linear-gradient(135deg, rgba(34,197,94,0.04), transparent); }
  .browser-model-card.installed::before { background: linear-gradient(90deg, #22c55e, #16a34a); }
  .browser-model-card.default { border-color: rgba(59,130,246,0.4); background: linear-gradient(135deg, rgba(59,130,246,0.05), transparent); }
  .browser-model-card.default::before { background: linear-gradient(90deg, #3b82f6, #2563eb); }
  :global(.dark) .browser-model-card { background: rgba(255,255,255,0.02); border-color: var(--dbd2); }
  :global(.dark) .browser-model-card.installed { background: linear-gradient(135deg, rgba(34,197,94,0.08), transparent); border-color: rgba(34,197,94,0.4); }
  :global(.dark) .browser-model-card.default { background: linear-gradient(135deg, rgba(59,130,246,0.08), transparent); border-color: rgba(59,130,246,0.4); }

  .bmc-header { display: flex; justify-content: space-between; align-items: flex-start; gap: 12px; }
  .bmc-title { display: flex; align-items: center; gap: 8px; flex-wrap: wrap; }
  .bmc-name { font-weight: 700; font-size: 15px; color: var(--color-text); letter-spacing: -0.01em; }
  .bmc-installed-badge { font-size: 10px; font-weight: 600; padding: 3px 8px; background: rgba(34,197,94,0.15); color: #22c55e; border-radius: 6px; display: flex; align-items: center; gap: 4px; }
  .bmc-installed-badge::before { content: '✓'; font-size: 9px; }
  .bmc-meta { display: flex; gap: 6px; flex-shrink: 0; }
  .bmc-size { font-size: 11px; padding: 4px 8px; background: linear-gradient(135deg, rgba(59,130,246,0.1), rgba(59,130,246,0.05)); border-radius: 6px; color: #3b82f6; font-family: 'SF Mono', 'Menlo', monospace; font-weight: 600; }
  .bmc-params { font-size: 11px; padding: 4px 8px; background: linear-gradient(135deg, rgba(168,85,247,0.1), rgba(168,85,247,0.05)); border-radius: 6px; color: #a855f7; font-family: 'SF Mono', 'Menlo', monospace; font-weight: 600; }
  .bmc-description { font-size: 13px; color: var(--color-text-secondary); margin: 0; line-height: 1.4; }
  .bmc-capabilities { display: flex; flex-wrap: wrap; gap: 6px; }
  .cap-badge { display: flex; align-items: center; gap: 4px; padding: 4px 8px; border-radius: 6px; font-size: 11px; }
  .cap-badge.vision { background: rgba(139,92,246,0.12); color: #8b5cf6; }
  .cap-badge.tools { background: rgba(59,130,246,0.12); color: #3b82f6; }
  .cap-badge.coding { background: rgba(34,197,94,0.12); color: #22c55e; }
  .cap-badge.reasoning { background: rgba(249,115,22,0.12); color: #f97316; }
  .cap-badge.rag { background: rgba(6,182,212,0.12); color: #06b6d4; }
  .cap-badge.multilingual { background: rgba(236,72,153,0.12); color: #ec4899; }
  .cap-badge.fast { background: rgba(234,179,8,0.12); color: #eab308; }
  :global(.dark) .cap-badge.vision { background: color-mix(in srgb, var(--bos-category-ai) 20%, transparent); color: var(--bos-category-ai); }
  :global(.dark) .cap-badge.tools { background: color-mix(in srgb, var(--bos-status-info) 20%, transparent); color: var(--bos-status-info); }
  :global(.dark) .cap-badge.coding { background: color-mix(in srgb, var(--bos-status-success) 20%, transparent); color: var(--bos-status-success); }
  :global(.dark) .cap-badge.reasoning { background: color-mix(in srgb, var(--bos-priority-high) 20%, transparent); color: var(--bos-priority-high); }
  :global(.dark) .cap-badge.rag { background: color-mix(in srgb, var(--bos-category-automation) 20%, transparent); color: var(--bos-category-automation); }
  :global(.dark) .cap-badge.multilingual { background: color-mix(in srgb, var(--bos-category-communication) 20%, transparent); color: var(--bos-category-communication); }
  :global(.dark) .cap-badge.fast { background: color-mix(in srgb, var(--bos-status-warning) 20%, transparent); color: var(--bos-status-warning); }
  .cap-badge-icon-svg { width: 12px; height: 12px; flex-shrink: 0; }
  .cap-badge-label { font-weight: 500; }

  .bmc-variants { display: flex; align-items: flex-start; gap: 10px; padding-top: 10px; border-top: 1px solid var(--color-border); margin-top: 6px; }
  .variants-label { display: flex; align-items: center; gap: 5px; font-size: 11px; color: var(--color-text-muted); flex-shrink: 0; padding-top: 6px; }
  .variant-buttons { display: flex; flex-wrap: wrap; gap: 6px; }
  .variant-buttons.many { gap: 4px; }
  .variant-btn { display: flex; flex-direction: column; align-items: center; gap: 2px; padding: 8px 12px; background: var(--color-bg-secondary); border: 1px solid var(--color-border); border-radius: 8px; cursor: pointer; transition: all 0.2s ease; min-width: 52px; }
  .variant-btn:hover { border-color: var(--color-border-hover); background: var(--color-bg-tertiary); transform: translateY(-1px); }
  .variant-btn.selected { border-color: #3b82f6; background: linear-gradient(135deg, rgba(59,130,246,0.15), rgba(59,130,246,0.05)); box-shadow: 0 0 0 2px rgba(59,130,246,0.2); }
  :global(.dark) .variant-btn { background: rgba(255,255,255,0.03); border-color: var(--dbd2); }
  :global(.dark) .variant-btn:hover { background: rgba(255,255,255,0.06); }
  :global(.dark) .variant-btn.selected { background: linear-gradient(135deg, rgba(59,130,246,0.25), rgba(59,130,246,0.1)); box-shadow: 0 0 0 2px rgba(59,130,246,0.3); }
  .variant-params { font-size: 12px; font-weight: 700; color: var(--color-text); font-family: 'SF Mono','Menlo',monospace; }
  .variant-btn.selected .variant-params { color: #3b82f6; }
  .variant-size { font-size: 10px; color: var(--color-text-muted); font-family: 'SF Mono','Menlo',monospace; }
  .variant-btn.selected .variant-size { color: #60a5fa; }

  .bmc-footer { display: flex; align-items: center; justify-content: space-between; margin-top: auto; padding-top: 10px; border-top: 1px solid var(--color-border); }
  .bmc-downloads { display: flex; align-items: center; gap: 6px; font-size: 12px; color: var(--color-text-muted); font-weight: 500; }
  .bmc-actions { display: flex; gap: 8px; margin-left: auto; }
  .bmc-btn { display: flex; align-items: center; gap: 6px; padding: 8px 16px; border-radius: 8px; font-size: 13px; font-weight: 600; cursor: pointer; transition: all 0.2s ease; border: none; }
  .bmc-btn svg { width: 14px; height: 14px; }
  .bmc-btn.pull-btn { background: linear-gradient(135deg, #3b82f6, #2563eb); color: white; margin-left: auto; box-shadow: 0 2px 8px rgba(59,130,246,0.25); }
  .bmc-btn.pull-btn:hover { transform: translateY(-1px); box-shadow: 0 4px 12px rgba(59,130,246,0.35); }
  .bmc-btn.pull-btn:disabled { opacity: 0.6; cursor: not-allowed; transform: none; box-shadow: none; }
  .bmc-btn.default-btn { background: rgba(128,128,128,0.15); border: 1px solid var(--color-border); color: var(--color-text); }
  .bmc-btn.default-btn:hover { background: rgba(128,128,128,0.25); }
  .bmc-btn.default-btn.is-default { background: var(--color-primary); border-color: var(--color-primary); color: white; }
  :global(.dark) .bmc-btn.default-btn { background: rgba(255,255,255,0.1); border-color: var(--dbd); }
  :global(.dark) .bmc-btn.default-btn:hover { background: rgba(255,255,255,0.15); }
  .bmc-btn.delete-btn { background: transparent; color: var(--color-error); padding: 8px; border: 1px solid var(--color-border); }
  .bmc-btn.delete-btn:hover { background: rgba(220,38,38,0.1); border-color: var(--color-error); }

  .btn-spinner-small { width: 14px; height: 14px; border: 2px solid rgba(255,255,255,0.3); border-top-color: white; border-radius: 50%; animation: spin 0.8s linear infinite; }
  @keyframes spin { to { transform: rotate(360deg); } }
</style>
