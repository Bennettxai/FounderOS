<script lang="ts">
  import type { PullProgress } from '$lib/stores/aiSettings';
  import { formatBytes } from '$lib/stores/aiSettings';

  interface Props {
    pullModelName: string;
    isPulling: boolean;
    pullProgress: PullProgress | null;
    pullError: string;
    pullSpeed: string;
    onPullModelNameChange: (name: string) => void;
    onPull: () => void;
  }

  let {
    pullModelName,
    isPulling,
    pullProgress,
    pullError,
    pullSpeed,
    onPullModelNameChange,
    onPull,
  }: Props = $props();

  function getProgressPercent(): number {
    if (!pullProgress?.total || !pullProgress?.completed) return 0;
    return Math.round((pullProgress.completed / pullProgress.total) * 100);
  }

  function getTimeRemaining(): string {
    if (!pullProgress?.total || !pullProgress?.completed || !pullSpeed) return '';
    const remaining = pullProgress.total - pullProgress.completed;
    const speedMatch = pullSpeed.match(/[\d.]+/);
    if (!speedMatch) return '';
    let speedBytes = parseFloat(speedMatch[0]);
    if (pullSpeed.includes('KB')) speedBytes *= 1024;
    if (pullSpeed.includes('MB')) speedBytes *= 1024 * 1024;
    if (pullSpeed.includes('GB')) speedBytes *= 1024 * 1024 * 1024;
    if (speedBytes <= 0) return '';
    const seconds = remaining / speedBytes;
    if (seconds < 60) return `~${Math.ceil(seconds)}s`;
    if (seconds < 3600) return `~${Math.ceil(seconds / 60)}m`;
    return `~${(seconds / 3600).toFixed(1)}h`;
  }
</script>

<div class="pull-card-compact">
  <div class="pull-compact-header">
    <h4>Pull Custom Model</h4>
    <a href="https://ollama.com/library" target="_blank" class="browse-link">
      Browse Ollama Library
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="14" height="14"><path d="M18 13v6a2 2 0 01-2 2H5a2 2 0 01-2-2V8a2 2 0 012-2h6M15 3h6v6M10 14L21 3"/></svg>
    </a>
  </div>
  <div class="pull-form-compact">
    <input
      type="text"
      value={pullModelName}
      oninput={(e) => onPullModelNameChange((e.target as HTMLInputElement).value)}
      placeholder="Enter model name (e.g., llama3.2:3b, phi3:medium)"
      disabled={isPulling}
    />
    <button class="pull-btn-compact" onclick={onPull} disabled={isPulling || !pullModelName.trim()}>
      {#if isPulling}
        <div class="btn-spinner-small"></div>
      {:else}
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M21 15v4a2 2 0 01-2 2H5a2 2 0 01-2-2v-4M7 10l5 5 5-5M12 15V3"/></svg>
      {/if}
      Pull
    </button>
  </div>
  {#if isPulling && pullProgress}
    <div class="pull-progress-compact">
      <div class="progress-info">
        <span class="progress-status">{pullProgress.status}</span>
        <span class="progress-percent">{getProgressPercent()}%</span>
      </div>
      <div class="progress-bar-compact">
        <div class="progress-fill" style="width: {getProgressPercent()}%"></div>
      </div>
      <div class="progress-details">
        {#if pullProgress.total && pullProgress.completed}
          <span>{formatBytes(pullProgress.completed)} / {formatBytes(pullProgress.total)}</span>
        {/if}
        {#if pullSpeed}<span class="speed">{pullSpeed}</span>{/if}
        {#if getTimeRemaining()}<span class="time">{getTimeRemaining()}</span>{/if}
      </div>
    </div>
  {/if}
  {#if pullError}
    <div class="pull-error-compact">{pullError}</div>
  {/if}
</div>

<style>
  .pull-card-compact { padding: 16px; background: var(--color-bg); border: 1px solid var(--color-border); border-radius: 12px; }
  .pull-compact-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 12px; }
  .pull-compact-header h4 { margin: 0; font-size: 14px; font-weight: 600; }
  .browse-link { display: flex; align-items: center; gap: 6px; font-size: 12px; color: var(--color-primary); text-decoration: none; }
  .browse-link:hover { text-decoration: underline; }
  .pull-form-compact { display: flex; gap: 8px; }
  .pull-form-compact input { flex: 1; padding: 10px 14px; background: var(--color-bg-secondary); border: 1px solid var(--color-border); border-radius: 8px; font-size: 13px; color: var(--color-text); }
  .pull-form-compact input:focus { outline: none; border-color: var(--color-primary); }
  .pull-form-compact input::placeholder { color: var(--color-text-muted); }
  .pull-btn-compact { display: flex; align-items: center; gap: 6px; padding: 10px 16px; background: var(--color-primary); color: white; border: none; border-radius: 8px; font-size: 13px; font-weight: 500; cursor: pointer; transition: all 0.2s; }
  .pull-btn-compact:hover { opacity: 0.9; }
  .pull-btn-compact:disabled { opacity: 0.6; cursor: not-allowed; }
  .pull-btn-compact svg { width: 16px; height: 16px; }
  .pull-progress-compact { margin-top: 12px; padding: 12px; background: var(--color-bg-secondary); border-radius: 8px; }
  .progress-info { display: flex; justify-content: space-between; margin-bottom: 8px; font-size: 12px; }
  .progress-percent { font-weight: 600; color: var(--color-primary); }
  .progress-bar-compact { height: 6px; background: var(--color-border); border-radius: 3px; overflow: hidden; }
  .progress-fill { height: 100%; background: var(--color-primary); transition: width 0.3s; }
  .progress-details { display: flex; justify-content: space-between; margin-top: 8px; font-size: 11px; color: var(--color-text-muted); }
  .pull-error-compact { margin-top: 12px; padding: 10px 14px; background: rgba(220,38,38,0.1); border-radius: 8px; color: var(--color-error); font-size: 13px; }
  .btn-spinner-small { width: 14px; height: 14px; border: 2px solid rgba(255,255,255,0.3); border-top-color: white; border-radius: 50%; animation: spin 0.8s linear infinite; }
  @keyframes spin { to { transform: rotate(360deg); } }
</style>
