<script lang="ts">
  import type { AgentPreset } from '$lib/api/ai/types';

  interface Props {
    preset: AgentPreset;
    onUse: (preset: AgentPreset) => void;
  }

  let { preset, onUse }: Props = $props();

  function getInitials(name: string | undefined): string {
    if (!name) return 'AI';
    const words = name.trim().split(/\s+/).filter(Boolean);
    if (words.length === 0) return 'AI';
    if (words.length === 1) return words[0].slice(0, 2).toUpperCase();
    return (words[0][0] + words[1][0]).toUpperCase();
  }

  const displayName = $derived(preset.display_name ?? preset.name);
  const tags = $derived(
    'capabilities' in preset ? (preset.capabilities as string[] | undefined) ?? [] :
    'tags' in preset ? (preset as unknown as { tags?: string[] }).tags ?? [] :
    []
  );
</script>

<div class="agpc-card">
  <!-- Top row: avatar + name + category -->
  <div class="agpc-top">
    <div class="agpc-avatar" aria-hidden="true">
      <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" aria-hidden="true">
        <rect x="3" y="11" width="18" height="11" rx="2" ry="2"/>
        <path d="M7 11V7a5 5 0 0 1 10 0v4"/>
      </svg>
    </div>

    <div class="agpc-info">
      <h3 class="agpc-name">{displayName}</h3>
      {#if preset.name && preset.name !== displayName}
        <span class="agpc-role">{preset.name}</span>
      {/if}
    </div>

    {#if preset.category}
      <span class="agpc-cat">{preset.category}</span>
    {/if}
  </div>

  <!-- Description -->
  <p class="agpc-desc">{preset.description || 'No description provided.'}</p>

  <!-- Tags row -->
  {#if tags.length > 0}
    <div class="agpc-tags">
      {#each tags.slice(0, 3) as tag}
        <span class="agpc-tag">{tag}</span>
      {/each}
      {#if tags.length > 3}
        <span class="agpc-tag agpc-tag--more">+{tags.length - 3}</span>
      {/if}
    </div>
  {/if}

  <!-- Footer: model -->
  {#if preset.model_preference}
    <div class="agpc-footer">
      <span class="agpc-model">
        <svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true">
          <circle cx="12" cy="12" r="3"/><path d="M12 1v4M12 19v4M4.22 4.22l2.83 2.83M16.95 16.95l2.83 2.83M1 12h4M19 12h4M4.22 19.78l2.83-2.83M16.95 7.05l2.83-2.83"/>
        </svg>
        {preset.model_preference.replace('claude-', '').replace('-4', ' 4')}
      </span>
    </div>
  {/if}

  <!-- Use button -->
  <button
    onclick={() => onUse(preset)}
    class="btn-cta agpc-use-btn"
    aria-label="Use {displayName} template"
  >
    Use Template
  </button>
</div>

<style>
  /* ═══ PresetCard — BOS Tokens ═══ */

  .agpc-card {
    background: var(--dbg, #fff);
    border: 1px solid var(--dbd, #e0e0e0);
    border-radius: 12px;
    padding: 1.125rem;
    display: flex;
    flex-direction: column;
    gap: 0.75rem;
    height: 100%;
    transition: border-color 0.18s, box-shadow 0.18s, transform 0.18s;
  }
  .agpc-card:hover {
    border-color: var(--bos-accent-blue, #3b82f6);
    box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.07), 0 4px 16px rgba(0,0,0,0.05);
    transform: translateY(-1px);
  }

  /* Top row */
  .agpc-top {
    display: flex;
    align-items: flex-start;
    gap: 0.625rem;
  }

  .agpc-avatar {
    width: 2.5rem;
    height: 2.5rem;
    border-radius: 10px;
    background: var(--dbg3, #eee);
    display: flex;
    align-items: center;
    justify-content: center;
    flex-shrink: 0;
    color: var(--dt2, #555);
  }

  .agpc-info {
    flex: 1;
    min-width: 0;
  }
  .agpc-name {
    font-size: 0.9375rem;
    font-weight: 600;
    color: var(--dt, #111);
    margin: 0;
    line-height: 1.3;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  .agpc-role {
    font-size: 0.6875rem;
    color: var(--dt3, #888);
    font-family: var(--bos-font-code-family, monospace);
    display: block;
    margin-top: 0.125rem;
  }

  .agpc-cat {
    font-size: 0.6875rem;
    font-weight: 600;
    padding: 0.1875rem 0.5rem;
    border-radius: 4px;
    background: var(--dbg2, #f5f5f5);
    color: var(--dt2, #555);
    text-transform: capitalize;
    white-space: nowrap;
    flex-shrink: 0;
    align-self: flex-start;
    margin-top: 0.125rem;
  }

  /* Description */
  .agpc-desc {
    font-size: 0.8125rem;
    line-height: 1.55;
    color: var(--dt2, #555);
    margin: 0;
    flex-grow: 1;
    display: -webkit-box;
    -webkit-line-clamp: 3;
    line-clamp: 3;
    -webkit-box-orient: vertical;
    overflow: hidden;
  }

  /* Tags */
  .agpc-tags {
    display: flex;
    flex-wrap: wrap;
    gap: 0.3125rem;
  }
  .agpc-tag {
    font-size: 0.6875rem;
    padding: 0.125rem 0.4375rem;
    border-radius: 4px;
    background: var(--dbg2, #f5f5f5);
    color: var(--dt3, #888);
    white-space: nowrap;
  }
  .agpc-tag--more {
    background: transparent;
    color: var(--dt4, #bbb);
  }

  /* Footer */
  .agpc-footer {
    padding-top: 0.5rem;
    border-top: 1px solid var(--dbd2, #f0f0f0);
  }
  .agpc-model {
    display: inline-flex;
    align-items: center;
    gap: 0.3125rem;
    font-size: 0.6875rem;
    color: var(--dt3, #888);
    font-family: var(--bos-font-code-family, monospace);
  }

  /* Use button */
  .agpc-use-btn {
    width: 100%;
    justify-content: center;
    margin-top: auto;
  }
</style>
