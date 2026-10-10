<script lang="ts">
  import type { UsageStats, SystemInfo, LLMProvider } from '$lib/stores/aiSettings';
  import { formatTokens, formatDuration } from '$lib/stores/aiSettings';

  interface Props {
    usageStats: UsageStats;
    systemInfo: SystemInfo | null;
    providers: LLMProvider[];
    localModelsCount: number;
  }

  let { usageStats, systemInfo, providers, localModelsCount }: Props = $props();
</script>

<!-- System Overview Row -->
<div class="stats-system-row">
  <div class="system-card">
    <div class="system-platform-info">
      <div class="platform-icon-large">
        {#if systemInfo?.platform === 'darwin'}
          <svg viewBox="0 0 24 24" fill="currentColor"><path d="M18.71 19.5c-.83 1.24-1.71 2.45-3.05 2.47-1.34.03-1.77-.79-3.29-.79-1.53 0-2 .77-3.27.82-1.31.05-2.3-1.32-3.14-2.53C4.25 17 2.94 12.45 4.7 9.39c.87-1.52 2.43-2.48 4.12-2.51 1.28-.02 2.5.87 3.29.87.78 0 2.26-1.07 3.81-.91.65.03 2.47.26 3.64 1.98-.09.06-2.17 1.28-2.15 3.81.03 3.02 2.65 4.03 2.68 4.04-.03.07-.42 1.44-1.38 2.83M13 3.5c.73-.83 1.94-1.46 2.94-1.5.13 1.17-.34 2.35-1.04 3.19-.69.85-1.83 1.51-2.95 1.42-.15-1.15.41-2.35 1.05-3.11z"/></svg>
        {:else}
          <svg viewBox="0 0 24 24" fill="currentColor"><path d="M3,12V6.75L9,5.43V11.91L3,12M20,3V11.75L10,11.9V5.21L20,3M3,13L9,13.09V19.9L3,18.75V13M20,13.25V22L10,20.09V13.1L20,13.25Z"/></svg>
        {/if}
      </div>
      <div class="platform-text">
        <span class="platform-label">{systemInfo?.platform === 'darwin' ? 'macOS' : systemInfo?.platform || 'System'}</span>
        {#if systemInfo?.has_gpu}
          <span class="gpu-badge">{systemInfo.gpu_name || 'Apple Silicon / Metal'}</span>
        {/if}
      </div>
    </div>
    <div class="system-metrics">
      <div class="sys-metric">
        <span class="sys-metric-value">{localModelsCount}</span>
        <span class="sys-metric-label">Local Models</span>
      </div>
      <div class="sys-metric">
        <span class="sys-metric-value">{usageStats.local_model_storage_gb}GB</span>
        <span class="sys-metric-label">Storage Used</span>
      </div>
      <div class="sys-metric">
        <span class="sys-metric-value">{providers.filter(p => p.configured).length}</span>
        <span class="sys-metric-label">Providers</span>
      </div>
    </div>
  </div>
  {#if systemInfo}
    <div class="ram-gauge-card">
      <div class="ram-gauge-header">
        <span class="ram-title">Memory</span>
        <span class="ram-detail">{systemInfo.available_ram_gb.toFixed(0)}GB free / {systemInfo.total_ram_gb}GB</span>
      </div>
      <div class="ram-gauge-visual">
        <svg class="gauge-svg-large" viewBox="0 0 120 120">
          <circle class="gauge-bg-large" cx="60" cy="60" r="50" fill="none" />
          <circle
            class="gauge-fill-large"
            cx="60" cy="60" r="50"
            fill="none"
            stroke-dasharray="{(systemInfo.available_ram_gb / systemInfo.total_ram_gb) * 314} 314"
            transform="rotate(-90 60 60)"
          />
        </svg>
        <div class="gauge-center-text">
          <span class="gauge-percent">{Math.round((systemInfo.available_ram_gb / systemInfo.total_ram_gb) * 100)}%</span>
          <span class="gauge-subtitle">Free</span>
        </div>
      </div>
    </div>
  {/if}
</div>

<!-- Key Metrics Grid -->
<div class="key-metrics-grid">
  <div class="metric-card requests">
    <div class="metric-sparkline">
      {#if usageStats.recent.length > 0}
        {@const recent = usageStats.recent}
        <svg viewBox="0 0 100 30" preserveAspectRatio="none">
          <polyline
            fill="none"
            stroke="currentColor"
            stroke-width="2"
            stroke-linecap="round"
            stroke-linejoin="round"
            points="{recent.map((d, i) => `${i * (100 / (recent.length - 1 || 1))},${30 - (d.requests / Math.max(...recent.map(r => r.requests), 1)) * 25}`).join(' ')}"
          />
        </svg>
      {/if}
    </div>
    <div class="metric-header">
      <div class="metric-icon">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"/></svg>
      </div>
      <span class="metric-trend up">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" width="12" height="12"><polyline points="23 6 13.5 15.5 8.5 10.5 1 18"/><polyline points="17 6 23 6 23 12"/></svg>
      </span>
    </div>
    <div class="metric-content">
      <span class="metric-value animate-number">{usageStats.total_requests.toLocaleString()}</span>
      <span class="metric-label">Total Requests</span>
    </div>
    <div class="metric-sub">
      <span class="metric-badge">{usageStats.session_count} sessions</span>
    </div>
  </div>
  <div class="metric-card tokens">
    <div class="metric-sparkline">
      {#if usageStats.recent.length > 0}
        {@const recent = usageStats.recent}
        <svg viewBox="0 0 100 30" preserveAspectRatio="none">
          <polyline
            fill="none"
            stroke="currentColor"
            stroke-width="2"
            stroke-linecap="round"
            stroke-linejoin="round"
            points="{recent.map((d, i) => `${i * (100 / (recent.length - 1 || 1))},${30 - (d.tokens / Math.max(...recent.map(r => r.tokens), 1)) * 25}`).join(' ')}"
          />
        </svg>
      {/if}
    </div>
    <div class="metric-header">
      <div class="metric-icon">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="4" y="2" width="16" height="20" rx="2"/><line x1="8" y1="6" x2="16" y2="6"/><line x1="8" y1="10" x2="16" y2="10"/><line x1="8" y1="14" x2="12" y2="14"/></svg>
      </div>
    </div>
    <div class="metric-content">
      <span class="metric-value">{formatTokens(usageStats.total_tokens)}</span>
      <span class="metric-label">Total Tokens</span>
    </div>
    <div class="metric-breakdown">
      <span class="metric-in"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" width="10" height="10"><polyline points="19 12 12 19 5 12"/><line x1="12" y1="19" x2="12" y2="5"/></svg> {formatTokens(usageStats.input_tokens)}</span>
      <span class="metric-out"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" width="10" height="10"><polyline points="5 12 12 5 19 12"/><line x1="12" y1="5" x2="12" y2="19"/></svg> {formatTokens(usageStats.output_tokens)}</span>
    </div>
  </div>
  <div class="metric-card speed">
    <div class="metric-header">
      <div class="metric-icon">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polygon points="13 2 3 14 12 14 11 22 21 10 12 10 13 2"/></svg>
      </div>
      {#if usageStats.avg_response_time_ms < 500}
        <span class="metric-badge good">Fast</span>
      {:else if usageStats.avg_response_time_ms < 2000}
        <span class="metric-badge">Average</span>
      {:else}
        <span class="metric-badge slow">Slow</span>
      {/if}
    </div>
    <div class="metric-content">
      <span class="metric-value">{formatDuration(usageStats.avg_response_time_ms)}</span>
      <span class="metric-label">Avg Response</span>
    </div>
    <div class="metric-sub">
      <span class="metric-badge">{usageStats.avg_requests_per_session} req/session</span>
    </div>
  </div>
  <div class="metric-card cost">
    <div class="metric-header">
      <div class="metric-icon">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="12" y1="1" x2="12" y2="23"/><path d="M17 5H9.5a3.5 3.5 0 0 0 0 7h5a3.5 3.5 0 0 1 0 7H6"/></svg>
      </div>
    </div>
    <div class="metric-content">
      <span class="metric-value">${(usageStats.cloud_api_cost + usageStats.local_power_cost_estimate).toFixed(2)}</span>
      <span class="metric-label">Total Cost</span>
    </div>
    <div class="metric-breakdown cost-breakdown">
      <span class="metric-cloud" title="Cloud API Cost">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="12" height="12"><path d="M18 10h-1.26A8 8 0 1 0 9 20h9a5 5 0 0 0 0-10z"/></svg>
        ${usageStats.cloud_api_cost.toFixed(2)}
      </span>
      <span class="metric-local" title="Est. Power Cost">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="12" height="12"><rect x="2" y="3" width="20" height="14" rx="2"/><line x1="8" y1="21" x2="16" y2="21"/><line x1="12" y1="17" x2="12" y2="21"/></svg>
        ${usageStats.local_power_cost_estimate.toFixed(2)}
      </span>
    </div>
  </div>
</div>

<style>
  /* System Row */
  .stats-system-row {
    display: grid;
    grid-template-columns: 1fr auto;
    gap: 20px;
  }

  .system-card {
    padding: 20px;
    background: var(--color-bg-secondary);
    border: 1px solid var(--color-border);
    border-radius: 16px;
  }

  .system-platform-info {
    display: flex;
    align-items: center;
    gap: 16px;
    margin-bottom: 20px;
  }

  .platform-icon-large {
    width: 48px;
    height: 48px;
    display: flex;
    align-items: center;
    justify-content: center;
    background: var(--color-bg);
    border-radius: 12px;
  }

  .platform-icon-large svg {
    width: 28px;
    height: 28px;
    color: var(--color-text);
  }

  .platform-text {
    display: flex;
    flex-direction: column;
    gap: 4px;
  }

  .platform-label {
    font-size: 18px;
    font-weight: 600;
  }

  .gpu-badge {
    font-size: 12px;
    padding: 3px 10px;
    background: linear-gradient(135deg, rgba(34, 197, 94, 0.15), rgba(34, 197, 94, 0.05));
    color: #22c55e;
    border-radius: 20px;
    width: fit-content;
  }

  :global(.dark) .gpu-badge {
    background: linear-gradient(135deg, rgba(34, 197, 94, 0.2), rgba(34, 197, 94, 0.08));
  }

  .system-metrics {
    display: flex;
    gap: 32px;
  }

  .sys-metric {
    display: flex;
    flex-direction: column;
    gap: 2px;
  }

  .sys-metric-value {
    font-size: 24px;
    font-weight: 700;
    color: var(--color-text);
  }

  .sys-metric-label {
    font-size: 12px;
    color: var(--color-text-muted);
  }

  /* RAM Gauge */
  .ram-gauge-card {
    padding: 20px;
    background: var(--color-bg-secondary);
    border: 1px solid var(--color-border);
    border-radius: 16px;
    min-width: 180px;
  }

  .ram-gauge-header {
    display: flex;
    flex-direction: column;
    gap: 4px;
    margin-bottom: 12px;
  }

  .ram-title { font-size: 14px; font-weight: 600; }
  .ram-detail { font-size: 12px; color: var(--color-text-muted); }

  .ram-gauge-visual {
    position: relative;
    width: 120px;
    height: 120px;
    margin: 0 auto;
  }

  .gauge-svg-large { width: 100%; height: 100%; }
  .gauge-bg-large { stroke: var(--color-border); stroke-width: 8; }
  .gauge-fill-large {
    stroke: #22c55e;
    stroke-width: 8;
    stroke-linecap: round;
    transition: stroke-dasharray 0.5s ease;
  }

  .gauge-center-text {
    position: absolute;
    top: 50%;
    left: 50%;
    transform: translate(-50%, -50%);
    text-align: center;
  }

  .gauge-percent {
    display: block;
    font-size: 24px;
    font-weight: 700;
    color: #22c55e;
  }

  .gauge-subtitle { font-size: 11px; color: var(--color-text-muted); }

  /* Key Metrics Grid */
  .key-metrics-grid {
    display: grid;
    grid-template-columns: repeat(4, 1fr);
    gap: 16px;
  }

  .metric-card {
    position: relative;
    padding: 20px;
    background: var(--color-bg-secondary);
    border: 1px solid var(--color-border);
    border-radius: 16px;
    display: flex;
    flex-direction: column;
    gap: 8px;
    overflow: hidden;
    transition: all 0.2s ease;
  }

  .metric-card:hover {
    transform: translateY(-2px);
    box-shadow: 0 8px 24px rgba(0, 0, 0, 0.1);
  }

  :global(.dark) .metric-card:hover {
    box-shadow: 0 8px 24px rgba(0, 0, 0, 0.3);
  }

  .metric-card.requests {
    background: linear-gradient(135deg, rgba(59, 130, 246, 0.12), rgba(59, 130, 246, 0.02));
    border-color: rgba(59, 130, 246, 0.25);
  }
  .metric-card.requests .metric-icon svg { color: #3b82f6; }
  .metric-card.requests .metric-sparkline { color: rgba(59, 130, 246, 0.5); }

  .metric-card.tokens {
    background: linear-gradient(135deg, rgba(168, 85, 247, 0.12), rgba(168, 85, 247, 0.02));
    border-color: rgba(168, 85, 247, 0.25);
  }
  .metric-card.tokens .metric-icon svg { color: #a855f7; }
  .metric-card.tokens .metric-sparkline { color: rgba(168, 85, 247, 0.5); }

  .metric-card.speed {
    background: linear-gradient(135deg, rgba(34, 197, 94, 0.12), rgba(34, 197, 94, 0.02));
    border-color: rgba(34, 197, 94, 0.25);
  }
  .metric-card.speed .metric-icon svg { color: #22c55e; }

  .metric-card.cost {
    background: linear-gradient(135deg, rgba(249, 115, 22, 0.12), rgba(249, 115, 22, 0.02));
    border-color: rgba(249, 115, 22, 0.25);
  }
  .metric-card.cost .metric-icon svg { color: #f97316; }

  .metric-sparkline {
    position: absolute;
    top: 10px;
    right: 10px;
    width: 80px;
    height: 30px;
    opacity: 0.6;
  }

  .metric-sparkline svg { width: 100%; height: 100%; }

  .metric-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: 4px;
  }

  .metric-icon {
    width: 36px;
    height: 36px;
    display: flex;
    align-items: center;
    justify-content: center;
    background: rgba(255, 255, 255, 0.1);
    border-radius: 10px;
  }

  :global(.dark) .metric-icon { background: rgba(255, 255, 255, 0.05); }
  .metric-icon svg { width: 20px; height: 20px; }

  .metric-trend {
    display: flex;
    align-items: center;
    gap: 4px;
    font-size: 11px;
    font-weight: 600;
    padding: 4px 8px;
    border-radius: 20px;
  }

  .metric-trend.up { color: #22c55e; background: rgba(34, 197, 94, 0.15); }
  .metric-trend.down { color: #ef4444; background: rgba(239, 68, 68, 0.15); }

  .metric-content {
    display: flex;
    flex-direction: column;
    gap: 2px;
  }

  .metric-value {
    font-size: 32px;
    font-weight: 700;
    font-family: 'SF Mono', 'Menlo', 'Monaco', monospace;
    letter-spacing: -0.02em;
  }

  .metric-label {
    font-size: 13px;
    color: var(--color-text-muted);
    font-weight: 500;
  }

  .metric-sub { margin-top: 8px; }

  .metric-badge {
    display: inline-block;
    font-size: 11px;
    font-weight: 500;
    padding: 4px 10px;
    background: rgba(0, 0, 0, 0.05);
    border-radius: 20px;
    color: var(--color-text-muted);
  }

  :global(.dark) .metric-badge { background: rgba(255, 255, 255, 0.08); }
  .metric-badge.good { color: #22c55e; background: rgba(34, 197, 94, 0.12); }
  .metric-badge.slow { color: #ef4444; background: rgba(239, 68, 68, 0.12); }

  .metric-breakdown {
    display: flex;
    gap: 12px;
    font-size: 12px;
    font-weight: 500;
    margin-top: 8px;
  }

  .metric-in { display: flex; align-items: center; gap: 4px; color: #22c55e; }
  .metric-out { display: flex; align-items: center; gap: 4px; color: #f97316; }

  .cost-breakdown { flex-direction: column; gap: 6px; }
  .metric-cloud, .metric-local { display: inline-flex; align-items: center; gap: 6px; font-size: 12px; }
  .metric-cloud { color: #3b82f6; }
  .metric-local { color: #22c55e; }
  .metric-cloud svg, .metric-local svg { flex-shrink: 0; opacity: 0.7; }

  @media (max-width: 1200px) {
    .key-metrics-grid { grid-template-columns: repeat(2, 1fr); }
  }

  @media (max-width: 768px) {
    .stats-system-row { grid-template-columns: 1fr; }
    .key-metrics-grid { grid-template-columns: 1fr; }
  }
</style>
