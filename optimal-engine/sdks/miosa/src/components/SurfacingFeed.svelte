<script lang="ts">
  import type { EngineContext } from '../engine.js';
  import type { SurfaceStream, SurfaceEvent } from '@optimal-engine/client';
  import { asSubscriptionId } from '@optimal-engine/client';
  import { onDestroy } from 'svelte';

  interface FeedEvent {
    id: string;
    type: string;
    category?: string;
    trigger?: string;
    slug?: string;
    score?: number;
    receivedAt: number;
    raw: SurfaceEvent;
  }

  interface Props {
    engine: EngineContext;
    subscriptionId?: string;
    maxEvents?: number;
  }

  let { engine, subscriptionId, maxEvents = 50 }: Props = $props();

  let events = $state<FeedEvent[]>([]);
  let connected = $state(false);
  let error = $state<string | null>(null);
  let listEl: HTMLDivElement;

  let stream: SurfaceStream | null = null;
  let counter = 0;

  function categoryColor(cat?: string): string {
    if (!cat) return '';
    const lower = cat.toLowerCase();
    if (lower.includes('update')) return 'oe-rel--updates';
    if (lower.includes('extend')) return 'oe-rel--extends';
    if (lower.includes('deriv')) return 'oe-rel--derives';
    if (lower.includes('contra')) return 'oe-rel--contradicts';
    return 'oe-rel--cites';
  }

  function triggerLabel(trigger?: string): string {
    if (!trigger) return '';
    return trigger.replace(/_/g, ' ');
  }

  function relativeTime(ts: number): string {
    const diff = Date.now() - ts;
    const secs = Math.floor(diff / 1000);
    if (secs < 60) return `${secs}s ago`;
    const mins = Math.floor(secs / 60);
    if (mins < 60) return `${mins}m ago`;
    return `${Math.floor(mins / 60)}h ago`;
  }

  function parseEvent(raw: SurfaceEvent): FeedEvent {
    const data = (raw.data as Record<string, unknown>) ?? {};
    return {
      id: `ev-${++counter}`,
      type: raw.type ?? 'message',
      category: data.category as string | undefined,
      trigger: data.trigger as string | undefined,
      slug: data.slug as string | undefined,
      score: data.score as number | undefined,
      receivedAt: Date.now(),
      raw,
    };
  }

  function openStream(subId: string) {
    if (stream) {
      stream.close();
      stream = null;
      connected = false;
    }
    error = null;
    const s = engine.client.surface.stream(asSubscriptionId(subId));
    s.on((ev: SurfaceEvent) => {
      const fe = parseEvent(ev);
      events = [fe, ...events].slice(0, maxEvents);
      // Auto-scroll to top (newest at top)
    }).onError((err: Error) => {
      error = err.message;
      connected = false;
    });
    stream = s;
    connected = true;
  }

  function dismiss(id: string) {
    events = events.filter((e) => e.id !== id);
  }

  // React to subscriptionId changes
  $effect(() => {
    if (subscriptionId) {
      openStream(subscriptionId);
    } else {
      if (stream) {
        stream.close();
        stream = null;
      }
      connected = false;
      events = [];
    }
  });

  onDestroy(() => {
    stream?.close();
  });
</script>

<div class="sf-root">
  {#if !subscriptionId}
    <div class="oe-empty">
      <span class="oe-empty__icon">◌</span>
      <span>No subscription active. Create one to start receiving pushes.</span>
    </div>
  {:else}
    <!-- Status bar -->
    <div class="sf-status">
      <div class="sf-dot {connected ? 'sf-dot--connected' : 'sf-dot--disconnected'}" aria-hidden="true"></div>
      <span class="sf-status-label">{connected ? 'Connected' : 'Disconnected'}</span>
      {#if events.length > 0}
        <span class="sf-count">{events.length} event{events.length !== 1 ? 's' : ''}</span>
        <button class="sf-clear" onclick={() => { events = []; }} aria-label="Clear all events">Clear</button>
      {/if}
    </div>

    {#if error}
      <div class="sf-error">{error}</div>
    {/if}

    <!-- Feed list -->
    <div class="oe-feed" bind:this={listEl} role="log" aria-live="polite" aria-label="Surface events feed">
      {#if events.length === 0}
        <div class="sf-waiting">
          <div class="sf-pulse" aria-hidden="true"></div>
          <span>Waiting for events…</span>
        </div>
      {:else}
        {#each events as ev (ev.id)}
          <div class="oe-feed-card sf-card">
            <div class="oe-feed-card__header">
              {#if ev.category}
                <span class="oe-feed-card__category oe-rel {categoryColor(ev.category)}">{ev.category}</span>
              {/if}
              {#if ev.trigger}
                <span class="sf-trigger">{triggerLabel(ev.trigger)}</span>
              {/if}
              <span class="oe-feed-card__time">{relativeTime(ev.receivedAt)}</span>
              <button class="sf-dismiss" onclick={() => dismiss(ev.id)} aria-label="Dismiss event">×</button>
            </div>

            <div class="oe-feed-card__body">
              {#if ev.slug}
                <code class="sf-slug">{ev.slug}</code>
              {:else}
                <span class="sf-type">{ev.type}</span>
              {/if}
            </div>

            {#if ev.score !== undefined}
              {#each [ev.score >= 0.7 ? 'high' : ev.score >= 0.4 ? 'mid' : 'low'] as lvl (lvl)}
                <div class="oe-score-bar sf-score-bar">
                  <div class="oe-score-bar__track">
                    <div class="oe-score-bar__fill oe-score-bar__fill--{lvl}" style="width: {Math.round(ev.score * 100)}%"></div>
                  </div>
                  <span class="oe-score-bar__label">{Math.round(ev.score * 100)}</span>
                </div>
              {/each}
            {/if}
          </div>
        {/each}
      {/if}
    </div>
  {/if}
</div>

<style>
  .sf-root {
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
  }

  .sf-status {
    display: flex;
    align-items: center;
    gap: 0.4rem;
    padding: 0.3rem 0.1rem;
  }

  .sf-dot {
    width: 7px;
    height: 7px;
    border-radius: 50%;
    flex-shrink: 0;
  }

  .sf-dot--connected {
    background: #7be3a3;
    box-shadow: 0 0 6px #7be3a3;
  }

  .sf-dot--disconnected {
    background: #f88;
  }

  .sf-status-label {
    font-size: 0.72rem;
    color: var(--dt3, #888);
  }

  .sf-count {
    font-size: 0.68rem;
    color: var(--dt4, #555);
    margin-left: auto;
  }

  .sf-clear {
    background: none;
    border: none;
    font-size: 0.68rem;
    color: var(--dt4, #555);
    cursor: pointer;
    padding: 0 0.2rem;
    text-decoration: underline;
    transition: color 0.1s;
  }

  .sf-clear:hover {
    color: var(--dt3, #888);
  }

  .sf-error {
    padding: 0.35rem 0.6rem;
    border-radius: 5px;
    background: rgba(248,136,136,0.08);
    border: 1px solid rgba(248,136,136,0.2);
    font-size: 0.78rem;
    color: #f88;
  }

  .sf-card {
    position: relative;
  }

  .sf-trigger {
    font-size: 0.65rem;
    font-weight: 600;
    padding: 1px 6px;
    border-radius: 999px;
    background: rgba(255,255,255,0.05);
    color: var(--dt3, #888);
    text-transform: capitalize;
  }

  .sf-slug {
    font-family: monospace;
    font-size: 0.82rem;
    color: var(--daccent, #7ea8ff);
    cursor: pointer;
    text-decoration: underline dashed;
    text-underline-offset: 2px;
  }

  .sf-type {
    font-size: 0.78rem;
    color: var(--dt3, #888);
    font-style: italic;
  }

  .sf-score-bar {
    margin-top: 0.35rem;
    max-width: 160px;
  }

  .sf-dismiss {
    background: none;
    border: none;
    color: var(--dt4, #555);
    font-size: 1rem;
    cursor: pointer;
    padding: 0 0.15rem;
    line-height: 1;
    margin-left: auto;
    transition: color 0.1s;
    display: none;
  }

  .oe-feed-card:hover .sf-dismiss {
    display: inline;
  }

  .sf-dismiss:hover {
    color: var(--dt2, #ccc);
  }

  /* Waiting pulse */
  .sf-waiting {
    display: flex;
    align-items: center;
    gap: 0.6rem;
    padding: 1rem 0.5rem;
    font-size: 0.82rem;
    color: var(--dt4, #555);
  }

  .sf-pulse {
    width: 8px;
    height: 8px;
    border-radius: 50%;
    background: var(--daccent, #7ea8ff);
    animation: sf-pulse 1.5s ease-in-out infinite;
  }

  @keyframes sf-pulse {
    0%, 100% { opacity: 0.2; transform: scale(0.8); }
    50% { opacity: 1; transform: scale(1.2); }
  }
</style>
