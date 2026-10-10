<script lang="ts">
  // "Currently surfacing" — proactive memory pushes from the engine.
  //
  // - Loads + creates subscriptions for the active workspace.
  // - Opens an SSE stream per active subscription.
  // - Renders incoming surface events as a feed (newest first).
  // - Lets the user trigger a synthetic push (Test) to verify the pipe.

  import { onMount, onDestroy } from 'svelte';
  import {
    listSubscriptions,
    createSubscription,
    deleteSubscription,
    openSurfaceStream,
    testSurface,
    type Subscription,
    type SurfaceEvent,
    type MemoryCategory,
    type SubscriptionScope
  } from '$lib/api';
  import { activeWorkspaceId } from '$lib/stores/workspace';

  let subscriptions = $state<Subscription[]>([]);
  let events = $state<SurfaceEvent[]>([]);
  let error = $state<string | null>(null);
  let loading = $state(false);

  // Stream lifecycle — one EventSource per subscription.
  const streams = new Map<string, EventSource>();

  // New-subscription form state
  let creating = $state(false);
  let newScope = $state<SubscriptionScope>('topic');
  let newScopeValue = $state('');
  let newCategories = $state<MemoryCategory[]>(['recent_actions', 'ownership']);

  // Engramme-derived enterprise category list (13 of their 18).
  const ALL_CATEGORIES: { id: MemoryCategory; label: string }[] = [
    { id: 'recent_actions', label: 'Recent actions' },
    { id: 'autobiographical_past', label: 'Past actions (older)' },
    { id: 'contacts', label: 'Contacts' },
    { id: 'schedules', label: 'Schedules' },
    { id: 'ownership', label: 'Ownership' },
    { id: 'open_tasks', label: 'Open tasks' },
    { id: 'tip_of_tongue', label: 'Tip-of-tongue' },
    { id: 'professional_knowledge', label: 'Professional knowledge' },
    { id: 'file_locations', label: 'File locations' },
    { id: 'procedures', label: 'Procedures' },
    { id: 'event_locations', label: 'Event locations' },
    { id: 'factual', label: 'Factual' },
    { id: 'unassigned', label: 'Unassigned' }
  ];

  $effect(() => {
    // Re-bootstrap whenever active workspace changes
    void bootstrap($activeWorkspaceId);
  });

  async function bootstrap(workspaceId: string | null) {
    closeAllStreams();
    error = null;
    if (!workspaceId) return;
    loading = true;
    try {
      const { subscriptions: subs } = await listSubscriptions(workspaceId);
      subscriptions = subs;
      // Open a stream per active subscription
      for (const s of subs) {
        if (s.status === 'active') openStream(s.id);
      }
    } catch (e) {
      error = (e as Error).message;
    } finally {
      loading = false;
    }
  }

  function openStream(subId: string) {
    if (streams.has(subId)) return;
    const es = openSurfaceStream(subId, (e) => {
      events = [e, ...events].slice(0, 50);
    });
    es.onerror = () => {
      // Engine restart, transient — drop and rely on the bootstrap on next load
    };
    streams.set(subId, es);
  }

  function closeStream(subId: string) {
    const es = streams.get(subId);
    if (es) {
      es.close();
      streams.delete(subId);
    }
  }

  function closeAllStreams() {
    for (const id of streams.keys()) closeStream(id);
  }

  onDestroy(closeAllStreams);

  async function submitCreate(e: Event) {
    e.preventDefault();
    if (newScope !== 'workspace' && !newScopeValue.trim()) return;
    try {
      const sub = await createSubscription({
        workspace: $activeWorkspaceId ?? 'default',
        scope: newScope,
        scope_value: newScope === 'workspace' ? null : newScopeValue.trim(),
        categories: newCategories
      });
      subscriptions = [sub, ...subscriptions];
      openStream(sub.id);
      creating = false;
      newScopeValue = '';
    } catch (err) {
      error = (err as Error).message;
    }
  }

  async function remove(sub: Subscription) {
    closeStream(sub.id);
    await deleteSubscription(sub.id);
    subscriptions = subscriptions.filter((s) => s.id !== sub.id);
  }

  async function test(sub: Subscription) {
    // Push a synthetic envelope with a real workspace slug so the category heuristic finds something
    await testSurface(sub.id, 'customer-portal-pricing-decision');
  }

  function toggleCategory(cat: MemoryCategory) {
    newCategories = newCategories.includes(cat)
      ? newCategories.filter((c) => c !== cat)
      : [...newCategories, cat];
  }

  function categoryColor(c: MemoryCategory): string {
    const map: Record<MemoryCategory, string> = {
      recent_actions: 'var(--accent)',
      autobiographical_past: 'var(--text-muted)',
      contacts: 'var(--cyan)',
      schedules: 'var(--purple)',
      ownership: 'var(--amber)',
      open_tasks: 'var(--good)',
      tip_of_tongue: 'var(--pink)',
      professional_knowledge: 'var(--accent)',
      file_locations: 'var(--text-muted)',
      procedures: 'var(--purple)',
      event_locations: 'var(--cyan)',
      factual: 'var(--text-muted)',
      unassigned: 'var(--text-subtle)'
    };
    return map[c] ?? 'var(--text-muted)';
  }

  function fmtTime(iso: string): string {
    return new Date(iso).toLocaleTimeString(undefined, {
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit'
    });
  }

  function categoryLabel(c: MemoryCategory): string {
    return ALL_CATEGORIES.find((x) => x.id === c)?.label ?? c;
  }
</script>

<section class="surf">
  <header class="surf__head">
    <div>
      <span class="eyebrow">Currently surfacing</span>
      <p class="surf__lede">Proactive memory pushes — what the engine thinks you need to remember right now.</p>
    </div>
    <button class="btn btn--sm" onclick={() => (creating = !creating)}>
      {creating ? 'Cancel' : '+ Subscribe'}
    </button>
  </header>

  {#if error}
    <pre class="error">{error}</pre>
  {/if}

  {#if creating}
    <form class="surf-form card" onsubmit={submitCreate}>
      <div class="surf-form__row">
        <label class="surf-form__field">
          <span>Scope</span>
          <select bind:value={newScope}>
            <option value="workspace">Workspace (everything)</option>
            <option value="node">Node</option>
            <option value="topic">Topic</option>
            <option value="audience">Audience</option>
          </select>
        </label>
        {#if newScope !== 'workspace'}
          <label class="surf-form__field">
            <span>{newScope === 'topic' ? 'Topic' : newScope === 'node' ? 'Node slug' : 'Audience'}</span>
            <input
              type="text"
              bind:value={newScopeValue}
              placeholder={newScope === 'topic' ? 'pricing' : newScope === 'node' ? 'product-customer-portal' : 'sales'}
              autofocus
            />
          </label>
        {/if}
      </div>
      <div class="surf-form__cats">
        <span class="eyebrow">Categories</span>
        <div class="surf-form__catgrid">
          {#each ALL_CATEGORIES as cat}
            <button
              type="button"
              class="cat-chip"
              class:active={newCategories.includes(cat.id)}
              style="--cat-color: {categoryColor(cat.id)}"
              onclick={() => toggleCategory(cat.id)}
            >
              {cat.label}
            </button>
          {/each}
        </div>
      </div>
      <div class="surf-form__actions">
        <button type="submit" class="btn btn--primary btn--sm">Subscribe</button>
      </div>
    </form>
  {/if}

  {#if subscriptions.length === 0 && !loading && !creating}
    <div class="surf-empty">
      <p class="muted">No subscriptions yet. Add one to start receiving proactive pushes.</p>
    </div>
  {:else}
    <ul class="surf-subs">
      {#each subscriptions as sub}
        <li class="surf-sub">
          <div class="surf-sub__head">
            <div class="surf-sub__scope">
              <span class="chip chip--accent">{sub.scope}</span>
              {#if sub.scope_value}<code>{sub.scope_value}</code>{/if}
            </div>
            <div class="surf-sub__actions">
              <button class="btn btn--sm btn--ghost" onclick={() => test(sub)}>Test push</button>
              <button class="btn btn--sm btn--ghost" onclick={() => remove(sub)}>Remove</button>
            </div>
          </div>
          <div class="surf-sub__cats">
            {#each sub.categories as c}
              <span class="cat-tag" style="--cat-color: {categoryColor(c)}">{categoryLabel(c)}</span>
            {/each}
          </div>
        </li>
      {/each}
    </ul>
  {/if}

  <div class="surf-feed">
    <header class="surf-feed__head">
      <span class="eyebrow">Live feed</span>
      <span class="chip">{events.length}</span>
    </header>
    {#if events.length === 0}
      <p class="muted surf-feed__empty">No events yet — surfacing is silent until something matches.</p>
    {:else}
      <ul class="surf-feed__list">
        {#each events as e}
          <li class="surf-event">
            <div class="surf-event__head">
              <span class="cat-tag" style="--cat-color: {categoryColor(e.category)}">{categoryLabel(e.category)}</span>
              <span class="chip">score {e.score.toFixed(2)}</span>
              <span class="surf-event__time">{fmtTime(e.pushed_at)}</span>
            </div>
            <div class="surf-event__body">
              <span class="muted">{e.trigger}</span>
              <code>{e.envelope.slug}</code>
              <span class="chip">{e.envelope.kind}</span>
              <span class="chip">aud: {e.envelope.audience}</span>
            </div>
          </li>
        {/each}
      </ul>
    {/if}
  </div>
</section>

<style>
  .surf {
    display: flex;
    flex-direction: column;
    gap: 1rem;
  }
  .surf__head {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    gap: 0.75rem;
  }
  .surf__lede {
    margin: 0.3rem 0 0;
    color: var(--text-muted);
    font-size: 0.85rem;
  }

  .surf-form { padding: 1rem 1.1rem 1.1rem; }
  .surf-form__row {
    display: flex;
    gap: 0.7rem;
    margin-bottom: 0.85rem;
  }
  .surf-form__field {
    display: flex;
    flex-direction: column;
    gap: 0.3rem;
    flex: 1;
    font-size: 0.7rem;
    text-transform: uppercase;
    letter-spacing: 0.06em;
    color: var(--text-muted);
    font-weight: 600;
  }
  .surf-form__field input,
  .surf-form__field select {
    background: var(--bg);
    border: 1px solid var(--border);
    border-radius: var(--r-sm);
    padding: 0.4rem 0.6rem;
    color: var(--text);
    font: inherit;
    font-size: 0.85rem;
    text-transform: none;
  }
  .surf-form__cats { margin-bottom: 0.7rem; }
  .surf-form__catgrid {
    display: flex;
    flex-wrap: wrap;
    gap: 0.3rem;
    margin-top: 0.4rem;
  }
  .cat-chip {
    background: var(--bg);
    border: 1px solid var(--border);
    color: var(--text-muted);
    border-radius: 999px;
    padding: 0.25rem 0.6rem;
    font: inherit;
    font-size: 0.72rem;
    cursor: pointer;
    transition: background 0.12s, border-color 0.12s, color 0.12s;
  }
  .cat-chip.active {
    background: color-mix(in srgb, var(--cat-color) 14%, var(--bg));
    border-color: var(--cat-color);
    color: var(--text);
  }
  .surf-form__actions {
    display: flex;
    justify-content: flex-end;
    gap: 0.4rem;
  }

  .surf-empty {
    background: var(--bg-elevated);
    border: 1px dashed var(--border);
    border-radius: var(--r-md);
    padding: 1rem 1.2rem;
  }

  .surf-subs {
    list-style: none;
    margin: 0;
    padding: 0;
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
  }
  .surf-sub {
    background: var(--bg-elevated);
    border: 1px solid var(--border);
    border-radius: var(--r-md);
    padding: 0.7rem 0.9rem;
  }
  .surf-sub__head {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 0.6rem;
    margin-bottom: 0.45rem;
  }
  .surf-sub__scope {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    font-size: 0.85rem;
  }
  .surf-sub__scope code {
    font-family: var(--font-mono);
    color: var(--text);
    font-size: 0.82rem;
  }
  .surf-sub__actions { display: flex; gap: 0.3rem; }
  .surf-sub__cats {
    display: flex;
    flex-wrap: wrap;
    gap: 0.3rem;
  }

  .cat-tag {
    display: inline-block;
    padding: 2px 8px;
    border-radius: 999px;
    background: color-mix(in srgb, var(--cat-color) 14%, transparent);
    color: var(--cat-color);
    font-size: 0.7rem;
    font-weight: 600;
    text-transform: lowercase;
    border: 1px solid color-mix(in srgb, var(--cat-color) 30%, transparent);
  }

  .surf-feed {
    background: var(--bg-elevated);
    border: 1px solid var(--border);
    border-radius: var(--r-md);
    padding: 0.85rem 1rem;
  }
  .surf-feed__head {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: 0.5rem;
  }
  .surf-feed__empty {
    margin: 0.4rem 0 0;
    font-size: 0.85rem;
  }
  .surf-feed__list {
    list-style: none;
    margin: 0;
    padding: 0;
    display: flex;
    flex-direction: column;
    gap: 0.4rem;
    max-height: 380px;
    overflow-y: auto;
  }
  .surf-event {
    border: 1px solid var(--border-soft);
    background: var(--bg);
    border-radius: var(--r-sm);
    padding: 0.5rem 0.7rem;
  }
  .surf-event__head {
    display: flex;
    align-items: center;
    gap: 0.4rem;
    margin-bottom: 0.3rem;
  }
  .surf-event__time {
    margin-left: auto;
    color: var(--text-subtle);
    font-size: 0.72rem;
    font-family: var(--font-mono);
  }
  .surf-event__body {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: 0.35rem;
    font-size: 0.78rem;
  }
  .surf-event__body code {
    font-family: var(--font-mono);
    color: var(--accent);
    font-size: 0.78rem;
  }
</style>
