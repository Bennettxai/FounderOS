<!--
  VersionTimelinePanel.svelte

  Full version history in a slide-over panel.

  Features:
  - Vertical timeline with connecting lines
  - Version details (label, trigger, time)
  - Preview and Restore actions
  - Current version indicator
  - Compare versions button
-->
<script lang="ts">
  import {
    X,
    Eye,
    RotateCcw,
    Sparkles,
    Pencil,
    Camera,
    Save,
    Star,
    GitCompare
  } from 'lucide-svelte';
  import type { Version, VersionTrigger } from '$lib/types/versions';
  import { formatRelativeTime, getTriggerLabel } from '$lib/types/versions';

  interface Props {
    appId: string;
    versions: Version[];
    isOpen: boolean;
    isLoading?: boolean;
    onClose: () => void;
    onPreview: (version: Version) => void;
    onRestore: (version: Version) => void;
    onCompare?: () => void;
  }

  let {
    appId,
    versions,
    isOpen,
    isLoading = false,
    onClose,
    onPreview,
    onRestore,
    onCompare
  }: Props = $props();

  function getTriggerIcon(trigger: VersionTrigger) {
    switch (trigger) {
      case 'ai_generation':
        return Sparkles;
      case 'user_edit':
        return Pencil;
      case 'manual_snapshot':
        return Camera;
      case 'auto_snapshot':
        return Save;
      case 'restore':
        return RotateCcw;
    }
  }

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === 'Escape') {
      onClose();
    }
  }

  function handleBackdropClick(e: MouseEvent) {
    if ((e.target as HTMLElement).classList.contains('panel-backdrop')) {
      onClose();
    }
  }
</script>

{#if isOpen}
  <!-- svelte-ignore a11y_no_noninteractive_element_interactions -->
  <div
    class="panel-backdrop"
    role="dialog"
    aria-modal="true"
    aria-labelledby="panel-title"
    onclick={handleBackdropClick}
    onkeydown={handleKeydown}
  >
    <aside class="timeline-panel">
      <!-- Header -->
      <header class="panel-header">
        <h2 id="panel-title" class="panel-title">Version History</h2>
        <button type="button" class="btn-pill btn-pill-ghost close-btn" onclick={onClose} aria-label="Close panel">
          <X size={20} strokeWidth={2} />
        </button>
      </header>

      <!-- Content -->
      <div class="panel-content">
        {#if isLoading}
          <div class="loading-state">
            <div class="spinner"></div>
            <span>Loading history...</span>
          </div>
        {:else if versions.length === 0}
          <div class="empty-state">
            <div class="empty-icon">
              <Save size={32} strokeWidth={1.5} />
            </div>
            <h3>No versions yet</h3>
            <p>Versions are created when you or AI make changes to this app.</p>
          </div>
        {:else}
          <div class="timeline">
            {#each versions as version, index (version.id)}
              {@const TriggerIcon = getTriggerIcon(version.trigger)}
              {@const isLast = index === versions.length - 1}

              <div class="timeline-item" class:current={version.isCurrent}>
                <!-- Timeline dot and line -->
                <div class="timeline-track">
                  <div class="timeline-dot" class:current={version.isCurrent}>
                    {#if version.isCurrent}
                      <div class="dot-inner"></div>
                    {/if}
                  </div>
                  {#if !isLast}
                    <div class="timeline-line"></div>
                  {/if}
                </div>

                <!-- Version content -->
                <div class="timeline-content">
                  <div class="version-header">
                    <span class="version-number">v{version.versionNumber}</span>
                    {#if version.isCurrent}
                      <span class="current-badge">current</span>
                    {/if}
                    <span class="version-time">{formatRelativeTime(version.createdAt)}</span>
                  </div>

                  {#if version.label}
                    <p class="version-label">
                      {#if version.trigger === 'manual_snapshot'}
                        <Star size={12} strokeWidth={2} class="label-icon" />
                      {/if}
                      {version.label}
                    </p>
                  {/if}

                  <div class="version-meta">
                    <TriggerIcon size={12} strokeWidth={2} />
                    <span>{getTriggerLabel(version.trigger)}</span>
                    {#if version.createdByName}
                      <span class="separator">•</span>
                      <span>{version.createdByName}</span>
                    {/if}
                  </div>

                  {#if version.prompt}
                    <p class="version-prompt">"{version.prompt}"</p>
                  {/if}

                  <!-- Actions -->
                  <div class="version-actions">
                    <button
                      type="button"
                      class="btn-pill btn-pill-ghost action-btn"
                      onclick={() => onPreview(version)}
                    >
                      <Eye size={14} strokeWidth={2} />
                      Preview
                    </button>
                    {#if !version.isCurrent}
                      <button
                        type="button"
                        class="btn-pill btn-pill-ghost action-btn restore"
                        onclick={() => onRestore(version)}
                      >
                        <RotateCcw size={14} strokeWidth={2} />
                        Restore
                      </button>
                    {/if}
                  </div>
                </div>
              </div>
            {/each}
          </div>
        {/if}
      </div>

      <!-- Footer -->
      {#if versions.length > 1 && onCompare}
        <footer class="panel-footer">
          <button type="button" class="btn-pill btn-pill-ghost compare-btn" onclick={onCompare}>
            <GitCompare size={16} strokeWidth={2} />
            Compare Versions
          </button>
        </footer>
      {/if}
    </aside>
  </div>
{/if}

<style>
  .panel-backdrop {
    position: fixed;
    inset: 0;
    background: rgba(0, 0, 0, 0.3);
    z-index: 100;
    animation: fadeIn 150ms ease-out;
  }

  :global(.dark) .panel-backdrop {
    background: rgba(0, 0, 0, 0.5);
  }

  @keyframes fadeIn {
    from { opacity: 0; }
    to { opacity: 1; }
  }

  .timeline-panel {
    position: absolute;
    top: 0;
    right: 0;
    width: 400px;
    max-width: 100%;
    height: 100%;
    background: var(--bos-v2-layer-background-primary);
    border-left: 1px solid var(--bos-border-color);
    display: flex;
    flex-direction: column;
    animation: slideIn 200ms ease-out;
  }

  @keyframes slideIn {
    from {
      transform: translateX(100%);
    }
    to {
      transform: translateX(0);
    }
  }

  /* Header */
  .panel-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 16px 20px;
    border-bottom: 1px solid var(--bos-border-color);
    flex-shrink: 0;
  }

  .panel-title {
    font-size: 16px;
    font-weight: 600;
    color: var(--bos-v2-text-primary);
    margin: 0;
  }

  .close-btn {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 32px;
    height: 32px;
    color: var(--bos-v2-text-secondary);
    background: transparent;
    border: none;
    border-radius: 6px;
    cursor: pointer;
    transition: all 150ms ease;
  }

  .close-btn:hover {
    color: var(--bos-v2-text-primary);
    background: var(--bos-v2-layer-background-secondary);
  }

  /* Content */
  .panel-content {
    flex: 1;
    overflow-y: auto;
    padding: 20px;
  }

  /* Loading and empty states */
  .loading-state,
  .empty-state {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 12px;
    height: 200px;
    color: var(--bos-v2-text-secondary);
    text-align: center;
  }

  .spinner {
    width: 24px;
    height: 24px;
    border: 2px solid var(--bos-border-color);
    border-top-color: var(--bos-nav-active);
    border-radius: 50%;
    animation: spin 600ms linear infinite;
  }

  @keyframes spin {
    to { transform: rotate(360deg); }
  }

  .empty-icon {
    width: 64px;
    height: 64px;
    display: flex;
    align-items: center;
    justify-content: center;
    background: var(--bos-v2-layer-background-secondary);
    border-radius: 12px;
    color: var(--bos-v2-text-secondary);
  }

  .empty-state h3 {
    font-size: 14px;
    font-weight: 600;
    color: var(--bos-v2-text-primary);
    margin: 0;
  }

  .empty-state p {
    font-size: 13px;
    color: var(--bos-v2-text-secondary);
    margin: 0;
    max-width: 200px;
  }

  /* Timeline */
  .timeline {
    display: flex;
    flex-direction: column;
  }

  .timeline-item {
    display: flex;
    gap: 16px;
    padding-bottom: 24px;
  }

  .timeline-item:last-child {
    padding-bottom: 0;
  }

  .timeline-track {
    display: flex;
    flex-direction: column;
    align-items: center;
    width: 16px;
    flex-shrink: 0;
  }

  .timeline-dot {
    width: 12px;
    height: 12px;
    background: var(--bos-border-color);
    border: 2px solid var(--bos-v2-layer-background-primary);
    border-radius: 50%;
    flex-shrink: 0;
    box-shadow: 0 0 0 2px var(--bos-border-color);
  }

  .timeline-dot.current {
    background: var(--bos-nav-active);
    box-shadow: 0 0 0 2px color-mix(in srgb, var(--bos-nav-active) 30%, transparent);
  }

  :global(.dark) .timeline-dot.current {
    box-shadow: 0 0 0 2px color-mix(in srgb, var(--bos-nav-active) 30%, transparent);
  }

  .dot-inner {
    width: 100%;
    height: 100%;
    background: var(--bos-nav-active);
    border-radius: 50%;
    animation: pulse 2s ease-in-out infinite;
  }

  @keyframes pulse {
    0%, 100% { opacity: 1; }
    50% { opacity: 0.5; }
  }

  .timeline-line {
    width: 2px;
    flex: 1;
    background: var(--bos-border-color);
    margin-top: 4px;
  }

  .timeline-content {
    flex: 1;
    min-width: 0;
  }

  .version-header {
    display: flex;
    align-items: center;
    gap: 8px;
    margin-bottom: 4px;
  }

  .version-number {
    font-size: 14px;
    font-weight: 600;
    color: var(--bos-v2-text-primary);
  }

  .current-badge {
    font-size: 11px;
    font-weight: 500;
    color: var(--bos-status-success);
    background: color-mix(in srgb, var(--bos-status-success) 10%, var(--dbg));
    padding: 2px 6px;
    border-radius: 4px;
  }

  :global(.dark) .current-badge {
    color: var(--bos-status-success);
    background: color-mix(in srgb, var(--bos-status-success) 10%, transparent);
  }

  .version-time {
    font-size: 12px;
    color: var(--bos-v2-text-secondary);
    margin-left: auto;
  }

  .version-label {
    display: flex;
    align-items: center;
    gap: 4px;
    font-size: 13px;
    color: var(--bos-v2-text-primary);
    margin: 0 0 6px 0;
  }

  .label-icon {
    color: var(--bos-status-warning);
  }

  .version-meta {
    display: flex;
    align-items: center;
    gap: 4px;
    font-size: 12px;
    color: var(--bos-v2-text-secondary);
    margin-bottom: 6px;
  }

  .separator {
    color: var(--bos-border-color);
  }

  .version-prompt {
    font-size: 12px;
    font-style: italic;
    color: var(--bos-v2-text-secondary);
    margin: 0 0 8px 0;
    padding-left: 8px;
    border-left: 2px solid var(--bos-border-color);
  }

  .version-actions {
    display: flex;
    gap: 8px;
  }

  .action-btn {
    display: inline-flex;
    align-items: center;
    gap: 4px;
    padding: 4px 8px;
    font-size: 12px;
    font-weight: 500;
    color: var(--bos-v2-text-secondary);
    background: transparent;
    border: 1px solid var(--bos-border-color);
    border-radius: 4px;
    cursor: pointer;
    font-family: inherit;
    transition: all 150ms ease;
  }

  .action-btn:hover {
    color: var(--bos-v2-text-primary);
    background: var(--bos-v2-layer-background-secondary);
    border-color: var(--bos-v2-layer-insideBorder-border);
  }

  .action-btn.restore {
    color: var(--bos-nav-active);
    border-color: color-mix(in srgb, var(--bos-nav-active) 30%, transparent);
  }

  :global(.dark) .action-btn.restore {
    color: var(--bos-nav-active);
    border-color: color-mix(in srgb, var(--bos-nav-active) 30%, transparent);
  }

  .action-btn.restore:hover {
    background: color-mix(in srgb, var(--bos-nav-active) 8%, var(--dbg));
    border-color: color-mix(in srgb, var(--bos-nav-active) 50%, transparent);
  }

  :global(.dark) .action-btn.restore:hover {
    background: color-mix(in srgb, var(--bos-nav-active) 10%, transparent);
  }

  /* Footer */
  .panel-footer {
    padding: 16px 20px;
    border-top: 1px solid var(--bos-border-color);
    flex-shrink: 0;
  }

  .compare-btn {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 8px;
    width: 100%;
    padding: 10px 16px;
    font-size: 13px;
    font-weight: 500;
    color: var(--bos-v2-text-primary);
    background: var(--bos-v2-layer-background-secondary);
    border: 1px solid var(--bos-border-color);
    border-radius: 6px;
    cursor: pointer;
    font-family: inherit;
    transition: all 150ms ease;
  }

  .compare-btn:hover {
    background: var(--bos-v2-layer-background-tertiary);
    border-color: var(--bos-v2-layer-insideBorder-border);
  }

  /* Responsive */
  @media (max-width: 480px) {
    .timeline-panel {
      width: 100%;
    }
  }
</style>
