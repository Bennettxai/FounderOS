<script lang="ts">
  import type { UsageStats } from '$lib/stores/aiSettings';

  interface Props {
    usageStats: UsageStats;
  }

  let { usageStats }: Props = $props();
</script>

<!-- Activity Timeline -->
<div class="activity-section">
  <div class="section-header">
    <h3>Activity Timeline</h3>
    {#if usageStats.recent.length > 0}
      <span class="section-hint">{usageStats.recent.length} days shown</span>
    {/if}
  </div>
  {#if usageStats.recent.length > 0}
    {@const maxRequests = Math.max(...usageStats.recent.map(d => d.requests), 1)}
    <div class="activity-chart">
      <div class="activity-area-chart">
        <svg viewBox="0 0 {usageStats.recent.length * 50} 120" preserveAspectRatio="none">
          <path
            d="M 0 120 {usageStats.recent.map((d, i) => `L ${i * 50 + 25} ${120 - (d.requests / maxRequests) * 100}`).join(' ')} L {(usageStats.recent.length - 1) * 50 + 25} 120 Z"
            fill="url(#areaGradient)"
            opacity="0.3"
          />
          <polyline
            fill="none"
            stroke="var(--color-primary)"
            stroke-width="2.5"
            stroke-linecap="round"
            stroke-linejoin="round"
            points="{usageStats.recent.map((d, i) => `${i * 50 + 25},${120 - (d.requests / maxRequests) * 100}`).join(' ')}"
          />
          {#each usageStats.recent as day, i}
            <circle
              cx="{i * 50 + 25}"
              cy="{120 - (day.requests / maxRequests) * 100}"
              r="4"
              fill="var(--color-bg)"
              stroke="var(--color-primary)"
              stroke-width="2"
            />
          {/each}
          <defs>
            <linearGradient id="areaGradient" x1="0" y1="0" x2="0" y2="1">
              <stop offset="0%" stop-color="var(--color-primary)"/>
              <stop offset="100%" stop-color="var(--color-primary)" stop-opacity="0"/>
            </linearGradient>
          </defs>
        </svg>
      </div>
      <div class="activity-labels">
        {#each usageStats.recent as day}
          <div class="activity-label-item">
            <span class="activity-day">{new Date(day.date).toLocaleDateString('en-US', { weekday: 'short' })}</span>
            <span class="activity-count">{day.requests}</span>
          </div>
        {/each}
      </div>
    </div>
  {:else}
    <div class="activity-empty">
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" width="40" height="40">
        <path d="M3 3v18h18"/>
        <path d="M18 17V9"/>
        <path d="M13 17V5"/>
        <path d="M8 17v-3"/>
      </svg>
      <p>Activity data will appear as you use AI features</p>
    </div>
  {/if}
</div>

<!-- Session Stats -->
<div class="session-stats">
  <div class="section-header">
    <h3>Session Statistics</h3>
  </div>
  <div class="session-grid">
    <div class="session-stat-card">
      <div class="session-stat-icon">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="18" height="18"><path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/><path d="M23 21v-2a4 4 0 0 0-3-3.87"/><path d="M16 3.13a4 4 0 0 1 0 7.75"/></svg>
      </div>
      <div class="session-stat-content">
        <span class="session-value">{usageStats.session_count}</span>
        <span class="session-label">Total Sessions</span>
      </div>
    </div>
    <div class="session-stat-card">
      <div class="session-stat-icon">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="18" height="18"><circle cx="12" cy="12" r="10"/><polyline points="12 6 12 12 16 14"/></svg>
      </div>
      <div class="session-stat-content">
        <span class="session-value">{usageStats.avg_session_duration_min}<span class="session-unit">min</span></span>
        <span class="session-label">Avg Duration</span>
      </div>
    </div>
    <div class="session-stat-card">
      <div class="session-stat-icon">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="18" height="18"><path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"/></svg>
      </div>
      <div class="session-stat-content">
        <span class="session-value">{usageStats.avg_requests_per_session}</span>
        <span class="session-label">Requests/Session</span>
      </div>
    </div>
    <div class="session-stat-card">
      <div class="session-stat-icon">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="18" height="18"><rect x="4" y="2" width="16" height="20" rx="2"/><line x1="8" y1="6" x2="16" y2="6"/><line x1="8" y1="10" x2="16" y2="10"/><line x1="8" y1="14" x2="12" y2="14"/></svg>
      </div>
      <div class="session-stat-content">
        <span class="session-value">{usageStats.total_requests > 0 ? Math.round(usageStats.total_tokens / usageStats.total_requests) : 0}</span>
        <span class="session-label">Tokens/Request</span>
      </div>
    </div>
  </div>
</div>

<style>
  /* Activity Chart */
  .activity-section {
    padding: 24px;
    background: var(--color-bg-secondary);
    border: 1px solid var(--color-border);
    border-radius: 16px;
  }

  .activity-section h3 { margin: 0; font-size: 15px; font-weight: 600; }

  .activity-chart { display: flex; flex-direction: column; gap: 12px; }

  .activity-area-chart { height: 120px; border-radius: 8px; overflow: hidden; }
  .activity-area-chart svg { width: 100%; height: 100%; }

  .activity-labels { display: flex; justify-content: space-between; }

  .activity-label-item {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 2px;
  }

  .activity-day { font-size: 10px; color: var(--color-text-muted); text-transform: uppercase; }
  .activity-count { font-size: 11px; font-weight: 600; color: var(--color-primary); }

  .activity-empty {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    padding: 40px 20px;
    gap: 12px;
    color: var(--color-text-muted);
  }

  .activity-empty p { font-size: 13px; margin: 0; }

  /* Session Stats */
  .session-stats {
    padding: 24px;
    background: var(--color-bg-secondary);
    border: 1px solid var(--color-border);
    border-radius: 16px;
  }

  .session-stats h3 { margin: 0; font-size: 15px; font-weight: 600; }

  .session-grid {
    display: grid;
    grid-template-columns: repeat(4, 1fr);
    gap: 16px;
  }

  .session-stat-card {
    display: flex;
    align-items: flex-start;
    gap: 14px;
    padding: 16px;
    background: var(--color-bg);
    border: 1px solid var(--color-border);
    border-radius: 12px;
    transition: all 0.2s ease;
  }

  .session-stat-card:hover { border-color: var(--color-text-muted); transform: translateY(-1px); }

  :global(.dark) .session-stat-card {
    background: rgba(255, 255, 255, 0.03);
    border-color: rgba(255, 255, 255, 0.08);
  }

  :global(.dark) .session-stat-card:hover {
    background: rgba(255, 255, 255, 0.06);
    border-color: rgba(255, 255, 255, 0.15);
  }

  .session-stat-icon {
    width: 36px;
    height: 36px;
    display: flex;
    align-items: center;
    justify-content: center;
    background: linear-gradient(135deg, rgba(59, 130, 246, 0.1), rgba(59, 130, 246, 0.02));
    border-radius: 10px;
    color: #3b82f6;
    flex-shrink: 0;
  }

  .session-stat-content { display: flex; flex-direction: column; gap: 2px; }

  .session-value {
    font-size: 22px;
    font-weight: 700;
    font-family: 'SF Mono', 'Menlo', monospace;
    color: var(--color-text);
    line-height: 1;
  }

  .session-unit { font-size: 14px; font-weight: 500; opacity: 0.7; }
  .session-label { font-size: 12px; color: var(--color-text-muted); font-weight: 500; }

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
    .session-grid { grid-template-columns: repeat(2, 1fr); }
  }
</style>
