<script lang="ts">
  interface Props {
    id: string;
    title: string;
    genre?: string;
    intent?: string;
    node?: string;
    score?: number;
    createdAt?: string;
    updatedAt?: string;
    onselect?: (id: string) => void;
  }

  let {
    id,
    title,
    genre,
    intent,
    node,
    score,
    createdAt,
    updatedAt,
    onselect,
  }: Props = $props();

  const scoreLevel = $derived(
    score === undefined ? null : score >= 0.7 ? 'high' : score >= 0.4 ? 'mid' : 'low',
  );

  const scorePct = $derived(score !== undefined ? Math.round(score * 100) : null);

  function fmtDate(iso?: string): string {
    if (!iso) return '';
    return new Date(iso).toLocaleDateString(undefined, {
      month: 'short',
      day: 'numeric',
    });
  }
</script>

<!-- svelte-ignore a11y_click_events_have_key_events -->
<!-- svelte-ignore a11y_no_static_element_interactions -->
<div class="oe-signal-card" onclick={() => onselect?.(id)} role="button" tabindex="0"
  onkeydown={(e) => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); onselect?.(id); } }}
  aria-label="Signal: {title}">

  <div class="oe-signal-card__title">{title}</div>

  <div class="oe-signal-card__meta">
    {#if genre}
      <span class="oe-chip sc-genre">{genre}</span>
    {/if}
    {#if intent}
      <span class="oe-chip sc-intent">{intent}</span>
    {/if}
    {#if node}
      <span class="oe-signal-card__node">{node}</span>
    {/if}
  </div>

  {#if score !== undefined && scoreLevel}
    <div class="oe-score-bar" style="margin-bottom: 0.35rem;">
      <div class="oe-score-bar__track">
        <div
          class="oe-score-bar__fill oe-score-bar__fill--{scoreLevel}"
          style="width: {scorePct}%"
        ></div>
      </div>
      <span class="oe-score-bar__label">{scorePct}</span>
    </div>
  {/if}

  {#if createdAt || updatedAt}
    <div class="sc-dates">
      {#if createdAt}<span>created {fmtDate(createdAt)}</span>{/if}
      {#if updatedAt}<span>· updated {fmtDate(updatedAt)}</span>{/if}
    </div>
  {/if}
</div>

<style>
  .sc-genre {
    background: rgba(126, 168, 255, 0.1);
    color: var(--daccent, #7ea8ff);
    border: 1px solid rgba(126, 168, 255, 0.2);
  }

  .sc-intent {
    background: rgba(187, 126, 255, 0.1);
    color: #bb7eff;
    border: 1px solid rgba(187, 126, 255, 0.2);
  }

  .sc-dates {
    font-size: 0.68rem;
    color: var(--dt4, #555);
    display: flex;
    gap: 0.25rem;
  }
</style>
