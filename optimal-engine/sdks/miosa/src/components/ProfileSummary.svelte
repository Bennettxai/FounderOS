<script lang="ts">
  import type { EngineContext } from '../engine.js';
  import { useProfile } from '../hooks/useProfile.svelte.js';
  import type { Bandwidth } from '@optimal-engine/client';

  interface Props {
    engine: EngineContext;
    audience?: string;
    node?: string;
  }

  let { engine, audience, node }: Props = $props();

  const profile = useProfile(engine, { audience, node });

  const TIERS: { key: keyof typeof profile.profile; label: string }[] = [
    { key: 'static', label: 'Static' },
    { key: 'dynamic', label: 'Dynamic' },
    { key: 'curated', label: 'Curated' },
    { key: 'activity', label: 'Activity' },
  ];

  const BANDWIDTHS: Bandwidth[] = ['l0', 'l1', 'full'];

  let expanded = $state<Set<string>>(new Set());

  function toggle(key: string) {
    const next = new Set(expanded);
    if (next.has(key)) next.delete(key);
    else next.add(key);
    expanded = next;
  }

  function tierText(value: unknown): string {
    if (!value) return '';
    if (typeof value === 'string') return value;
    if (typeof value === 'object') return JSON.stringify(value, null, 2);
    return String(value);
  }

  function truncate(text: string, len = 200): string {
    return text.length > len ? text.slice(0, len) + '…' : text;
  }

  // Top entity tags from activity
  const entityTags = $derived.by(() => {
    const activity = profile.profile?.activity as Record<string, unknown> | undefined;
    if (!activity) return [];
    const entities = activity['top_entities'] ?? activity['entities'];
    if (!Array.isArray(entities)) return [];
    return (entities as { name: string; type: string }[]).slice(0, 10);
  });
</script>

<div class="ps-root">
  <!-- Bandwidth toggle -->
  <div class="ps-header">
    <span class="ps-title">Profile</span>
    <div class="oe-bw-toggle" role="group" aria-label="Bandwidth">
      {#each BANDWIDTHS as bw}
        <button
          class="oe-bw-toggle__btn {profile.bandwidth === bw ? 'oe-bw-toggle__btn--active' : ''}"
          onclick={() => { profile.bandwidth = bw; }}
          aria-pressed={profile.bandwidth === bw}
        >{bw}</button>
      {/each}
    </div>
  </div>

  {#if profile.loading}
    <div class="oe-empty"><div class="oe-spinner"></div></div>
  {:else if profile.error}
    <div class="oe-empty" role="alert">{profile.error}</div>
  {:else if profile.profile}
    <div class="oe-profile">
      {#each TIERS as tier}
        {@const raw = profile.profile[tier.key]}
        {@const text = tierText(raw)}
        {#if text}
          {@const isExpanded = expanded.has(tier.key)}
          {@const truncated = truncate(text)}
          <div class="oe-profile-tier">
            <div class="oe-profile-tier__label">{tier.label}</div>
            <div class="oe-profile-tier__text">
              {isExpanded ? text : truncated}
            </div>
            {#if text.length > 200}
              <span
                class="oe-profile-tier__more"
                role="button"
                tabindex="0"
                onclick={() => toggle(tier.key)}
                onkeydown={(e) => { if (e.key === 'Enter') toggle(tier.key); }}
                aria-expanded={isExpanded}
              >{isExpanded ? 'Show less' : 'Show more'}</span>
            {/if}
          </div>
        {/if}
      {/each}
    </div>

    {#if entityTags.length > 0}
      <div class="ps-tags" aria-label="Top entities">
        {#each entityTags as tag}
          <span class="ps-tag ps-tag--{tag.type ?? 'default'}">{tag.name}</span>
        {/each}
      </div>
    {/if}
  {:else}
    <div class="oe-empty">
      <span class="oe-empty__icon">◈</span>
      No profile data yet.
    </div>
  {/if}
</div>

<style>
  .ps-root {
    display: flex;
    flex-direction: column;
    gap: 0.75rem;
  }

  .ps-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
  }

  .ps-title {
    font-size: 0.75rem;
    font-weight: 700;
    text-transform: uppercase;
    letter-spacing: 0.06em;
    color: var(--dt4, #555);
  }

  .ps-tags {
    display: flex;
    flex-wrap: wrap;
    gap: 0.3rem;
    margin-top: 0.25rem;
  }

  .ps-tag {
    padding: 1px 8px;
    border-radius: 999px;
    font-size: 0.68rem;
    font-weight: 600;
    background: rgba(255, 255, 255, 0.06);
    color: var(--dt3, #888);
    border: 1px solid var(--dbd, rgba(255, 255, 255, 0.08));
  }

  .ps-tag--person { color: #7ea8ff; border-color: rgba(126, 168, 255, 0.2); background: rgba(126, 168, 255, 0.07); }
  .ps-tag--concept { color: #bb7eff; border-color: rgba(187, 126, 255, 0.2); background: rgba(187, 126, 255, 0.07); }
  .ps-tag--product { color: #eab308; border-color: rgba(234, 179, 8, 0.2); background: rgba(234, 179, 8, 0.07); }
</style>
