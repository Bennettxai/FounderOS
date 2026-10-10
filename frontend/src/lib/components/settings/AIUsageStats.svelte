<script lang="ts">
  import type { UsageStats, SystemInfo, LLMProvider } from '$lib/stores/aiSettings';
  import UsageSummaryCards from './UsageSummaryCards.svelte';
  import UsageBreakdown from './UsageBreakdown.svelte';
  import UsageTimeline from './UsageTimeline.svelte';

  interface Props {
    usageStats: UsageStats | null;
    loadingUsage: boolean;
    systemInfo: SystemInfo | null;
    providers: LLMProvider[];
    localModelsCount: number;
    onPeriodChange: (period: 'today' | 'week' | 'month' | 'all') => void;
    onRefresh: () => void;
  }

  let {
    usageStats,
    loadingUsage,
    systemInfo,
    providers,
    localModelsCount,
    onPeriodChange,
    onRefresh
  }: Props = $props();

  let usagePeriod = $state<'today' | 'week' | 'month' | 'all'>('month');

  function selectPeriod(period: 'today' | 'week' | 'month' | 'all') {
    usagePeriod = period;
    onPeriodChange(period);
  }
</script>

<div class="stats-page">
  <!-- Stats Header with Period Selector -->
  <div class="stats-header">
    <div class="stats-title-area">
      <span class="stats-eyebrow">Analytics</span>
      {#if usageStats}
        <button class="btn-pill btn-pill-ghost stats-date-picker">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="14" height="14"><rect x="3" y="4" width="18" height="18" rx="2" ry="2"/><line x1="16" y1="2" x2="16" y2="6"/><line x1="8" y1="2" x2="8" y2="6"/><line x1="3" y1="10" x2="21" y2="10"/></svg>
          <span>{usageStats.period_start} — {usageStats.period_end}</span>
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="12" height="12"><polyline points="6 9 12 15 18 9"/></svg>
        </button>
      {/if}
    </div>
    <div class="stats-controls">
      <div class="period-selector">
        <button class="btn-pill btn-pill-ghost period-btn" class:active={usagePeriod === 'today'} onclick={() => selectPeriod('today')}>Today</button>
        <button class="btn-pill btn-pill-ghost period-btn" class:active={usagePeriod === 'week'} onclick={() => selectPeriod('week')}>Week</button>
        <button class="btn-pill btn-pill-ghost period-btn" class:active={usagePeriod === 'month'} onclick={() => selectPeriod('month')}>Month</button>
        <button class="btn-pill btn-pill-ghost period-btn" class:active={usagePeriod === 'all'} onclick={() => selectPeriod('all')}>All Time</button>
      </div>
      <button class="btn-pill btn-pill-ghost refresh-btn" onclick={onRefresh} disabled={loadingUsage} title="Refresh stats" aria-label="Refresh stats">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class:spinning={loadingUsage}><path d="M23 4v6h-6M1 20v-6h6M3.51 9a9 9 0 0114.85-3.36L23 10M1 14l4.64 4.36A9 9 0 0020.49 15"/></svg>
      </button>
    </div>
  </div>

  {#if loadingUsage}
    <div class="loading">
      <div class="spinner"></div>
      <span>Loading analytics...</span>
    </div>
  {:else if usageStats}
    <UsageSummaryCards {usageStats} {systemInfo} {providers} {localModelsCount} />
    <UsageBreakdown {usageStats} />
    <UsageTimeline {usageStats} />
  {:else}
    <div class="empty-state">
      <div class="empty-icon">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="18" y1="20" x2="18" y2="10"/><line x1="12" y1="20" x2="12" y2="4"/><line x1="6" y1="20" x2="6" y2="14"/></svg>
      </div>
      <h3>No Usage Data</h3>
      <p>Usage statistics will appear here once you start using AI features</p>
    </div>
  {/if}
</div>

<style>
  .stats-page {
    display: flex;
    flex-direction: column;
    gap: 24px;
  }

  /* Header */
  .stats-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    flex-wrap: wrap;
    gap: 16px;
  }

  .stats-title-area {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .stats-eyebrow {
    font-size: 11px;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.5px;
    color: var(--color-text-muted);
  }

  .stats-date-picker {
    display: inline-flex;
    align-items: center;
    gap: 8px;
    padding: 8px 14px;
    background: var(--color-bg-secondary);
    border: 1px solid var(--color-border);
    border-radius: 10px;
    font-size: 13px;
    font-weight: 500;
    color: var(--color-text);
    cursor: pointer;
    transition: all 0.2s ease;
  }

  .stats-date-picker:hover {
    background: var(--color-bg-tertiary);
    border-color: var(--color-text-muted);
  }

  :global(.dark) .stats-date-picker {
    background: rgba(255, 255, 255, 0.05);
    border-color: rgba(255, 255, 255, 0.1);
  }

  :global(.dark) .stats-date-picker:hover {
    background: rgba(255, 255, 255, 0.1);
  }

  .stats-date-picker svg { color: var(--color-text-muted); }

  .stats-controls {
    display: flex;
    align-items: center;
    gap: 12px;
  }

  .period-selector {
    display: flex;
    gap: 4px;
    padding: 4px;
    background: rgba(0, 0, 0, 0.1);
    border-radius: 10px;
  }

  :global(.dark) .period-selector {
    background: rgba(255, 255, 255, 0.05);
  }

  .period-btn {
    padding: 8px 14px;
    background: transparent;
    border: none;
    border-radius: 8px;
    font-size: 13px;
    font-weight: 500;
    color: var(--color-text-secondary);
    cursor: pointer;
    transition: all 0.2s ease;
  }

  .period-btn:hover {
    color: var(--color-text);
    background: rgba(0, 0, 0, 0.05);
  }

  :global(.dark) .period-btn:hover {
    background: rgba(255, 255, 255, 0.1);
  }

  .period-btn.active {
    background: #3b82f6 !important;
    color: white !important;
  }

  /* Refresh button */
  .refresh-btn {
    display: flex;
    align-items: center;
    gap: 6px;
    padding: 8px 14px;
    background: var(--color-bg-tertiary);
    border: 1px solid var(--color-border);
    border-radius: 8px;
    font-size: 13px;
    color: var(--color-text);
    cursor: pointer;
    transition: all 0.2s;
  }

  .refresh-btn:hover { background: var(--color-bg-secondary); }
  .refresh-btn:disabled { opacity: 0.6; cursor: not-allowed; }
  .refresh-btn svg { width: 16px; height: 16px; }

  .refresh-btn svg.spinning {
    animation: spin 1s linear infinite;
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

  /* Empty state */
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

  .empty-icon svg { width: 32px; height: 32px; }
  .empty-state h3 { margin: 0; font-size: 16px; }
  .empty-state p { margin: 0; font-size: 14px; }

  @keyframes spin {
    to { transform: rotate(360deg); }
  }
</style>
