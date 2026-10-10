<script lang="ts">
  import type { UsageStats } from '$lib/stores/aiSettings';
  import { formatTokens, formatDuration, getProviderIconPath, getProviderLabel, getCategoryIconPath } from '$lib/stores/aiSettings';

  interface Props {
    usageStats: UsageStats;
  }

  let { usageStats }: Props = $props();
</script>

<!-- Local vs Cloud Comparison -->
<div class="comparison-section">
  <div class="section-header">
    <h3>Local vs Cloud Usage</h3>
    {#if usageStats.total_requests > 0}
      <span class="section-hint">Based on {usageStats.total_requests} requests</span>
    {/if}
  </div>
  {#if usageStats.total_requests === 0}
    <div class="comparison-empty">
      <div class="comparison-empty-icon">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" width="48" height="48">
          <path d="M21.21 15.89A10 10 0 1 1 8 2.83"/>
          <path d="M22 12A10 10 0 0 0 12 2v10z"/>
        </svg>
      </div>
      <p class="comparison-empty-text">Start using AI features to see your local vs cloud breakdown</p>
      <p class="comparison-empty-hint">Local inference is free, cloud APIs are billed per token</p>
    </div>
  {:else}
    {@const localReqs = usageStats.by_provider['ollama_local']?.requests || 0}
    {@const cloudReqs = Object.entries(usageStats.by_provider).filter(([k]) => k !== 'ollama_local').reduce((sum, [, v]) => sum + v.requests, 0)}
    {@const localPct = (localReqs / usageStats.total_requests * 100)}
    {@const cloudPct = (cloudReqs / usageStats.total_requests * 100)}
    <div class="comparison-split-bar">
      <div class="split-bar-container">
        <div class="split-bar-fill local" style="width: {localPct}%">
          {#if localPct > 15}<span class="split-bar-label">{localPct.toFixed(0)}%</span>{/if}
        </div>
        <div class="split-bar-fill cloud" style="width: {cloudPct}%">
          {#if cloudPct > 15}<span class="split-bar-label">{cloudPct.toFixed(0)}%</span>{/if}
        </div>
      </div>
    </div>
    <div class="comparison-grid">
      <div class="comparison-card local">
        <div class="comp-header">
          <div class="comp-icon-wrapper local">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="20" height="20">
              <rect x="2" y="3" width="20" height="14" rx="2"/>
              <path d="M8 21h8"/>
              <path d="M12 17v4"/>
            </svg>
          </div>
          <div class="comp-title-group">
            <span class="comp-title">Local Inference</span>
            <span class="comp-subtitle">On-device processing</span>
          </div>
        </div>
        <div class="comp-stats">
          <div class="comp-stat">
            <span class="comp-value">{localReqs}</span>
            <span class="comp-label">Requests</span>
          </div>
          <div class="comp-stat">
            <span class="comp-value">{formatTokens(usageStats.by_provider['ollama_local']?.tokens || 0)}</span>
            <span class="comp-label">Tokens</span>
          </div>
          <div class="comp-stat">
            <span class="comp-value">${usageStats.local_power_cost_estimate.toFixed(2)}</span>
            <span class="comp-label" title="Estimated electricity cost based on average GPU power usage">Est. Power</span>
          </div>
        </div>
      </div>
      <div class="comparison-card cloud">
        <div class="comp-header">
          <div class="comp-icon-wrapper cloud">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="20" height="20">
              <path d="M18 10h-1.26A8 8 0 1 0 9 20h9a5 5 0 0 0 0-10z"/>
            </svg>
          </div>
          <div class="comp-title-group">
            <span class="comp-title">Cloud API</span>
            <span class="comp-subtitle">Remote processing</span>
          </div>
        </div>
        <div class="comp-stats">
          <div class="comp-stat">
            <span class="comp-value">{cloudReqs}</span>
            <span class="comp-label">Requests</span>
          </div>
          <div class="comp-stat">
            <span class="comp-value">{formatTokens(Object.entries(usageStats.by_provider).filter(([k]) => k !== 'ollama_local').reduce((sum, [, v]) => sum + v.tokens, 0))}</span>
            <span class="comp-label">Tokens</span>
          </div>
          <div class="comp-stat">
            <span class="comp-value">${usageStats.cloud_api_cost.toFixed(2)}</span>
            <span class="comp-label">API Cost</span>
          </div>
        </div>
      </div>
    </div>
  {/if}
</div>

<!-- Detailed Breakdowns -->
<div class="breakdowns-grid">
  <!-- Provider Breakdown -->
  <div class="breakdown-card">
    <div class="breakdown-header">
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="16" height="16"><path d="M2 20h.01M7 20v-4M12 20v-8M17 20V8M22 4v16"/></svg>
      <h4>By Provider</h4>
    </div>
    {#if Object.keys(usageStats.by_provider).length === 0}
      <div class="breakdown-empty">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" width="32" height="32"><circle cx="12" cy="12" r="10"/><path d="M8 12h8"/><path d="M12 8v8"/></svg>
        <span>No provider data yet</span>
      </div>
    {:else}
      <div class="breakdown-list">
        {#each Object.entries(usageStats.by_provider).sort((a, b) => b[1].requests - a[1].requests) as [provider, stats]}
          <div class="breakdown-row">
            <div class="breakdown-info">
              <span class="breakdown-icon provider">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d={getProviderIconPath(provider)}/></svg>
              </span>
              <span class="breakdown-name">{getProviderLabel(provider)}</span>
            </div>
            <div class="breakdown-values">
              <span class="breakdown-stat-value requests">{stats.requests}</span>
              <span class="breakdown-stat-value tokens">{formatTokens(stats.tokens)}</span>
              <span class="breakdown-stat-value cost">${stats.cost.toFixed(2)}</span>
            </div>
            <div class="breakdown-bar">
              <div class="breakdown-bar-fill provider" style="width: {(stats.requests / usageStats.total_requests * 100)}%"></div>
            </div>
          </div>
        {/each}
      </div>
    {/if}
  </div>

  <!-- Model Breakdown -->
  <div class="breakdown-card">
    <div class="breakdown-header">
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="16" height="16"><path d="M12 2L2 7l10 5 10-5-10-5zM2 17l10 5 10-5M2 12l10 5 10-5"/></svg>
      <h4>By Model</h4>
    </div>
    {#if Object.keys(usageStats.by_model).length === 0}
      <div class="breakdown-empty">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" width="32" height="32"><rect x="4" y="4" width="16" height="16" rx="2"/><path d="M9 9h6v6H9z"/></svg>
        <span>No model usage yet</span>
      </div>
    {:else}
      <div class="breakdown-list">
        {#each Object.entries(usageStats.by_model).sort((a, b) => b[1].requests - a[1].requests).slice(0, 5) as [model, stats]}
          <div class="breakdown-row">
            <div class="breakdown-info">
              <span class="breakdown-name truncate">{model}</span>
            </div>
            <div class="breakdown-values">
              <span class="breakdown-stat-value requests">{stats.requests}</span>
              <span class="breakdown-stat-value tokens">{formatTokens(stats.tokens)}</span>
              <span class="breakdown-stat-value latency">{formatDuration(stats.avg_latency_ms)}</span>
            </div>
            <div class="breakdown-bar">
              <div class="breakdown-bar-fill model" style="width: {(stats.requests / usageStats.total_requests * 100)}%"></div>
            </div>
          </div>
        {/each}
      </div>
    {/if}
  </div>

  <!-- Agent Breakdown -->
  <div class="breakdown-card">
    <div class="breakdown-header">
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="16" height="16"><path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"/><circle cx="12" cy="7" r="4"/></svg>
      <h4>By Agent</h4>
    </div>
    {#if Object.keys(usageStats.by_agent).length === 0}
      <div class="breakdown-empty">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" width="32" height="32"><circle cx="12" cy="12" r="3"/><path d="M12 1v4M12 19v4M4.22 4.22l2.83 2.83M16.95 16.95l2.83 2.83M1 12h4M19 12h4M4.22 19.78l2.83-2.83M16.95 7.05l2.83-2.83"/></svg>
        <span>No agent usage yet</span>
      </div>
    {:else}
      <div class="breakdown-list">
        {#each Object.entries(usageStats.by_agent).sort((a, b) => b[1].requests - a[1].requests).slice(0, 5) as [agent, stats]}
          <div class="breakdown-row">
            <div class="breakdown-info">
              <span class="breakdown-icon agent">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d={getCategoryIconPath(agent.toLowerCase())}/></svg>
              </span>
              <span class="breakdown-name">{agent}</span>
            </div>
            <div class="breakdown-values">
              <span class="breakdown-stat-value requests">{stats.requests}</span>
              <span class="breakdown-stat-value tokens">{formatTokens(stats.tokens)}</span>
            </div>
            <div class="breakdown-bar">
              <div class="breakdown-bar-fill agent" style="width: {(stats.requests / usageStats.total_requests * 100)}%"></div>
            </div>
          </div>
        {/each}
      </div>
    {/if}
  </div>
</div>

<style>
  /* Comparison Section */
  .comparison-section {
    background: var(--color-bg-secondary);
    border: 1px solid var(--color-border);
    border-radius: 16px;
    padding: 24px;
  }

  .comparison-section h3 { margin: 0; font-size: 15px; font-weight: 600; }

  .comparison-empty {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    padding: 48px 24px;
    text-align: center;
  }

  .comparison-empty-icon { color: var(--color-text-muted); margin-bottom: 16px; opacity: 0.5; }
  .comparison-empty-text { font-size: 14px; color: var(--color-text-muted); margin: 0 0 8px 0; }
  .comparison-empty-hint { font-size: 12px; color: var(--color-text-muted); opacity: 0.7; margin: 0; }

  .comparison-split-bar { margin-bottom: 20px; }

  .split-bar-container {
    display: flex;
    height: 32px;
    border-radius: 8px;
    overflow: hidden;
    background: var(--color-bg);
  }

  .split-bar-fill {
    display: flex;
    align-items: center;
    justify-content: center;
    min-width: 30px;
    transition: width 0.5s ease;
  }

  .split-bar-fill.local { background: linear-gradient(90deg, #22c55e, #16a34a); }
  .split-bar-fill.cloud { background: linear-gradient(90deg, #3b82f6, #2563eb); }
  .split-bar-label { font-size: 12px; font-weight: 600; color: white; text-shadow: 0 1px 2px rgba(0, 0, 0, 0.2); }

  .comparison-grid {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 16px;
  }

  .comparison-card {
    padding: 20px;
    background: var(--color-bg);
    border-radius: 14px;
    border: 1px solid var(--color-border);
    transition: all 0.2s ease;
  }

  .comparison-card:hover { border-color: var(--color-text-muted); }

  .comparison-card.local {
    border-color: rgba(34, 197, 94, 0.3);
    background: linear-gradient(135deg, rgba(34, 197, 94, 0.05), transparent);
  }

  .comparison-card.local:hover { border-color: rgba(34, 197, 94, 0.5); }

  .comparison-card.cloud {
    border-color: rgba(59, 130, 246, 0.3);
    background: linear-gradient(135deg, rgba(59, 130, 246, 0.05), transparent);
  }

  .comparison-card.cloud:hover { border-color: rgba(59, 130, 246, 0.5); }

  :global(.dark) .comparison-card.local {
    border-color: rgba(34, 197, 94, 0.4);
    background: rgba(34, 197, 94, 0.05);
  }

  :global(.dark) .comparison-card.cloud {
    border-color: rgba(59, 130, 246, 0.4);
    background: rgba(59, 130, 246, 0.05);
  }

  .comp-header { display: flex; align-items: center; gap: 12px; margin-bottom: 16px; }

  .comp-icon-wrapper {
    width: 40px;
    height: 40px;
    display: flex;
    align-items: center;
    justify-content: center;
    border-radius: 10px;
  }

  .comp-icon-wrapper.local { background: rgba(34, 197, 94, 0.15); color: #22c55e; }
  .comp-icon-wrapper.cloud { background: rgba(59, 130, 246, 0.15); color: #3b82f6; }

  .comp-title-group { display: flex; flex-direction: column; gap: 2px; }
  .comp-subtitle { font-size: 11px; color: var(--color-text-muted); }
  .comp-title { font-size: 15px; font-weight: 600; }
  .comp-stats { display: flex; gap: 24px; margin-bottom: 16px; }
  .comp-stat { display: flex; flex-direction: column; gap: 2px; }
  .comp-value { font-size: 20px; font-weight: 700; }
  .comp-label { font-size: 11px; color: var(--color-text-muted); }

  /* Breakdowns Grid */
  .breakdowns-grid {
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    gap: 16px;
  }

  .breakdown-card {
    padding: 20px;
    background: var(--color-bg-secondary);
    border: 1px solid var(--color-border);
    border-radius: 16px;
    min-height: 200px;
    display: flex;
    flex-direction: column;
  }

  .breakdown-header {
    display: flex;
    align-items: center;
    gap: 8px;
    margin-bottom: 16px;
  }

  .breakdown-header svg { color: var(--color-text-muted); }
  .breakdown-header h4 { margin: 0; font-size: 14px; font-weight: 600; }

  .breakdown-empty {
    flex: 1;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 12px;
    color: var(--color-text-muted);
    opacity: 0.6;
  }

  .breakdown-empty span { font-size: 13px; }

  .breakdown-list { display: flex; flex-direction: column; gap: 12px; }

  .breakdown-row {
    display: flex;
    flex-direction: column;
    gap: 6px;
    padding: 8px 10px;
    background: var(--color-bg);
    border-radius: 8px;
    transition: all 0.15s ease;
  }

  .breakdown-row:hover { background: var(--color-bg-tertiary); }

  :global(.dark) .breakdown-row { background: rgba(255, 255, 255, 0.03); }
  :global(.dark) .breakdown-row:hover { background: rgba(255, 255, 255, 0.06); }

  .breakdown-info { display: flex; align-items: center; gap: 8px; }

  .breakdown-icon {
    width: 24px;
    height: 24px;
    display: flex;
    align-items: center;
    justify-content: center;
    flex-shrink: 0;
    border-radius: 6px;
  }

  .breakdown-icon.provider { background: rgba(59, 130, 246, 0.1); color: #3b82f6; }
  .breakdown-icon.agent { background: rgba(168, 85, 247, 0.1); color: #a855f7; }
  .breakdown-icon svg { width: 14px; height: 14px; }

  .breakdown-name { font-size: 13px; font-weight: 500; }
  .breakdown-name.truncate { white-space: nowrap; overflow: hidden; text-overflow: ellipsis; max-width: 120px; }

  .breakdown-values { display: flex; gap: 12px; font-size: 11px; font-family: 'SF Mono', 'Menlo', monospace; }
  .breakdown-stat-value { font-weight: 500; }
  .breakdown-stat-value.requests { color: #3b82f6; }
  .breakdown-stat-value.tokens { color: #f97316; }
  .breakdown-stat-value.cost { color: #22c55e; }
  .breakdown-stat-value.latency { color: #a855f7; }

  .breakdown-bar { height: 4px; background: var(--color-border); border-radius: 2px; overflow: hidden; }
  .breakdown-bar-fill { height: 100%; border-radius: 2px; transition: width 0.5s ease; }
  .breakdown-bar-fill.provider { background: #3b82f6; }
  .breakdown-bar-fill.model { background: #f97316; }
  .breakdown-bar-fill.agent { background: #a855f7; }

  /* Section header */
  .section-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: 16px;
  }

  .section-header h3 { margin: 0; font-size: 15px; font-weight: 600; }
  .section-hint { font-size: 12px; color: var(--color-text-muted); }

  @media (max-width: 1200px) {
    .breakdowns-grid { grid-template-columns: 1fr; }
  }

  @media (max-width: 768px) {
    .comparison-grid { grid-template-columns: 1fr; }
  }
</style>
