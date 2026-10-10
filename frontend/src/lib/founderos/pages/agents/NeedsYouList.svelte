<!-- Needs you (FounderOS v1 components/NeedsYouList.tsx): the queue of agent
     work waiting on the operator, mounted by the Agents tab (AgentsTabs) and by Home
     (console/NeedsYou, with `fill`). It reads the SAME payload the Deliverables
     tab uses (one fetch, two views). The bridge ranks it (people first, then
     deadlines) and takes out what he already decided; his own view is layered
     on with the needs-queue module: a decision in flight, the two-hour snooze
     and the per-row unseen dot. Results confirm on the OS-wide toast stack
     (chrome/toast, mounted by the /os layout) with an undo, and a row opens
     the slide-in review panel.

     The white button is the outward call (send it, go ahead, publish): it goes
     through the bridge guard, so with FOUNDEROS_WRITES=0 it is held, says so, and
     the row stays. Dismiss is his queue's own state. Snooze holds his view in
     this browser only; no agent is told. Bulk is dismiss-only. -->
<script lang="ts">
	import { onMount } from 'svelte';
	import { ExternalLink, RefreshCw, X } from '$lib/founderos/icons';
	import { founderosFetch, isGuardRefusal } from '$lib/founderos/api';
	import Label from '$lib/founderos/kit/Label.svelte';
	import Pressable from '$lib/founderos/kit/Pressable.svelte';
	import ReviewPanel from './ReviewPanel.svelte';
	import { toast } from '$lib/founderos/chrome/toast';
	import { writeFailure } from './runerror';
	import type { Classified, Decision, DeliverableItem, DeliverablesBody } from './types';
	import {
		SNOOZE_KEY,
		ago,
		parseSnoozeMap,
		primaryOf,
		pruneSnoozed,
		queueView,
		readStore,
		snoozeItem,
		subject,
		unseenStateOf,
		unsnooze,
		writeStore,
		type OpenedMap,
		type SnoozeMap
	} from '../console/needs-queue';

	let {
		data,
		error = null,
		boardUrl = null,
		refreshing = false,
		opened,
		onOpen,
		onReload,
		fill = false
	}: {
		data: DeliverablesBody | null;
		error?: string | null;
		boardUrl?: string | null;
		refreshing?: boolean;
		/** The per-row opened store, shared with the Deliverables tab. */
		opened: OpenedMap | null;
		/** Opening a row is what clears ITS dot. */
		onOpen: (item: DeliverableItem) => void;
		onReload: () => Promise<void> | void;
		/** Home: fill the parent's box and scroll the rows inside it, so the
		    queue ends exactly where its row-mate ends (side by side only). */
		fill?: boolean;
	} = $props();

	type RowState = 'busy' | 'held';
	let local = $state<Record<string, 'approved' | 'dismissed'>>({});
	let snoozed = $state<SnoozeMap>({});
	let now = $state(Date.now());
	let confirming = $state(false);
	let failure = $state<string | null>(null);
	let rowState = $state<Record<string, RowState>>({});
	let rowError = $state<Record<string, string>>({});
	let reviewing = $state<Classified | null>(null);

	const view = $derived(queueView(data, { local, snoozed, opened, now }));
	const textOf = (c: DeliverableItem) => c.title || subject(c.name);
	const decisionOf = (id: string): Decision | undefined => data?.decisions?.find((d) => d.id === id);
	// Prod writes these borders as border-os-ok/40 etc.; Tailwind v3 cannot
	// put an opacity on a var() colour, so the class never exists and the
	// border falls back to preflight's gray-200. That light edge is what prod
	// shows, so the port draws it too (likewise no tint on overdue rows).
	const PREFLIGHT_BORDER = 'rgb(229 231 235)';
	const GLYPH_TONE: Record<string, string> = { ok: 'var(--bn-ok)', warn: 'var(--bn-warn)', err: 'var(--bn-err)', dim: 'var(--bn-text-3)' };

	onMount(() => {
		snoozed = pruneSnoozed(parseSnoozeMap(readStore(SNOOZE_KEY)));
		// Held rows come back by themselves, so the queue re-reads the clock.
		const tick = setInterval(() => (now = Date.now()), 60_000);
		return () => clearInterval(tick);
	});

	async function refresh() {
		await onReload();
		// What the reload reports as decided no longer needs the local hold.
		local = {};
	}

	/** Every handled row leaves with a toast he can take back (prod settle()). */
	function showReceipt(text: string, undo: () => void) {
		toast.ok(text, undo);
	}

	async function post(json: unknown) {
		await founderosFetch('/pages/board/deliverables/decision', { method: 'POST', json });
	}

	async function undoDecision(c: Classified) {
		try {
			await post({ id: c.id, decision: null });
		} catch (e) {
			rowError = { ...rowError, [c.id]: writeFailure(e, 'undo') };
		}
		await refresh();
	}

	/** The outward call. Not optimistic: it leaves the queue only once the
	    bridge accepted it, because a guard refusal must leave it in place. */
	async function approve(c: Classified) {
		if (rowState[c.id]) return;
		const words = primaryOf(c.ask);
		rowState = { ...rowState, [c.id]: 'busy' };
		rowError = { ...rowError, [c.id]: '' };
		try {
			await post({ id: c.id, decision: 'approved', decidedRevision: c.revision });
			local = { ...local, [c.id]: 'approved' };
			const { [c.id]: _gone, ...rest } = rowState;
			rowState = rest;
			showReceipt(`${words.done} · ${textOf(c)}`, () => void undoDecision(c));
			void refresh();
		} catch (e) {
			if (isGuardRefusal(e)) {
				rowState = { ...rowState, [c.id]: 'held' };
				toast.warn(`held · writes are off · ${textOf(c)}`);
			} else {
				const { [c.id]: _gone, ...rest } = rowState;
				rowState = rest;
				rowError = { ...rowError, [c.id]: writeFailure(e, words.label) };
			}
		}
	}

	/** Dismiss is his own queue's state: optimistic, with a receipt he can take back. */
	async function dismiss(c: Classified) {
		local = { ...local, [c.id]: 'dismissed' };
		showReceipt(`dismissed · ${textOf(c)}`, () => void undoDecision(c));
		try {
			await post({ id: c.id, decision: 'dismissed', decidedRevision: c.revision });
		} catch (e) {
			const { [c.id]: _gone, ...rest } = local;
			local = rest;
			rowError = { ...rowError, [c.id]: writeFailure(e, 'dismiss') };
			return;
		}
		void refresh();
	}

	/** "Not now" is not a decision: a two-hour hold on his own view. */
	function snooze(c: Classified) {
		snoozed = pruneSnoozed(snoozeItem(snoozed, c.id));
		writeStore(SNOOZE_KEY, snoozed);
		showReceipt(`snoozed 2h · ${textOf(c)}`, () => {
			snoozed = unsnooze(snoozed, c.id);
			writeStore(SNOOZE_KEY, snoozed);
		});
	}

	async function dismissAll(items: Classified[]) {
		confirming = false;
		const next = { ...local };
		for (const c of items) next[c.id] = 'dismissed';
		local = next;
		try {
			await post({ decision: 'dismissed', items: items.map((c) => ({ id: c.id, decidedRevision: c.revision })) });
		} catch (e) {
			failure = writeFailure(e, 'bulk dismiss');
		}
		void refresh();
	}

	function open(c: Classified) {
		onOpen(c);
		reviewing = c;
	}

	function decideFromPanel(item: DeliverableItem, kind: Decision['decision'] | null) {
		const c = reviewing;
		if (!c || c.id !== item.id) return;
		if (kind === 'approved') void approve(c);
		else if (kind === 'dismissed') {
			reviewing = null;
			void dismiss(c);
		} else void undoDecision(c);
	}
</script>

<section
	data-part="card"
	data-card="needsyou"
	class="bn-card {fill ? 'flex h-full min-h-0 flex-col' : ''}"
>
	<div data-part="card-head" class="ny-line flex flex-wrap items-center gap-x-3 gap-y-1 border-b px-4 py-2.5">
		<Label>Needs you</Label>
		<span class="bn-dim font-mono text-[10px]">
			{#if view}
				{view.queue.length} of {view.files} agent files are waiting on you{#if view.overdue > 0}<span style:color="var(--bn-err)">{` · ${view.overdue} past due`}</span>{/if}{#if view.unread > 0}<span style:color="var(--bn-accent)">{` · ${view.unread} unread`}</span>{/if}{#if view.held > 0}<span>{` · ${view.held} snoozed`}</span>{/if}{#if view.handled > 0}<span>{` · ${view.handled} handled`}</span>{/if}
			{:else if error}
				queue unreachable
			{:else}
				reading the board…
			{/if}
		</span>
		{#if boardUrl}
			<a href={boardUrl} target="_blank" rel="noreferrer" class="bn-dim bn-linky ml-auto flex items-center gap-1 font-mono text-[10px]">open board <ExternalLink class="h-3 w-3" /></a>
		{/if}
		<!-- Bulk clear is DISMISS only: on a staged item approve means SEND IT, so
		     there is no bulk approve. Two clicks, because 45 dismissals is not
		     undoable in one gesture even though each row is. -->
		{#if view && view.queue.length > 0}
			{#if confirming}
				<span class="flex shrink-0 items-center gap-1.5">
					<button type="button" class="bn-pressable ny-danger flex items-center gap-1 border px-2 py-1 font-mono text-[10px] font-bold uppercase tracking-[0.12em]" onclick={() => view && void dismissAll(view.queue)}>
						<X class="h-3 w-3" /> dismiss all {view.queue.length}
					</button>
					<button type="button" class="bn-pressable bn-dim ny-hover font-mono text-[10px] uppercase tracking-[0.12em]" onclick={() => (confirming = false)}>cancel</button>
				</span>
			{:else}
				<button type="button" class="bn-pressable bn-dim ny-hover flex shrink-0 items-center gap-1 font-mono text-[10px] uppercase tracking-[0.12em]" title="Dismiss everything shown. Nothing is sent." onclick={() => (confirming = true)}>
					<X class="h-3 w-3" /> clear all
				</button>
			{/if}
		{/if}
		<button type="button" class="bn-pressable bn-dim ny-hover flex shrink-0 items-center gap-1 font-mono text-[10px] uppercase tracking-[0.12em] disabled:opacity-40" disabled={refreshing} onclick={() => void refresh()}>
			<RefreshCw class="h-3 w-3 {refreshing ? 'animate-spin' : ''}" /> refresh
		</button>
	</div>

	<!-- The queue says who ranked it, because the order is a claim. -->
	<div class="flex items-center gap-2 px-4 pt-2">
		<span class="ny-rule h-px flex-1"></span>
		<span class="bn-dim font-mono text-[9px] uppercase tracking-[0.1em]">ranked by the Conductor · people first, then deadlines</span>
		<span class="ny-rule h-px flex-1"></span>
	</div>

	{#if failure}<p class="px-4 pt-2 font-mono text-[10.5px]" style:color="var(--bn-err)">{failure}</p>{/if}

	{#if error && !view}
		<p class="px-4 py-3 font-mono text-[11px]" style:color="var(--bn-err)">deliverables unreachable: {error}</p>
	{:else if !view}
		<p class="bn-dim px-4 py-3 font-mono text-[10.5px]">reading the board…</p>
	{:else if view.queue.length === 0}
		<div class="ny-empty m-4 grid place-items-center gap-1 border border-dashed px-4 py-8 text-center">
			<span class="text-[18px]" style:color="var(--bn-ok)">✓</span>
			<p class="bn-text text-[12.5px] font-semibold">Nothing needs you.</p>
			<p class="bn-dim font-mono text-[10px]">
				the Conductor will surface the next thing here{#if view.held > 0} · {view.held} snoozed{/if}{#if view.handled > 0} · you handled {view.handled}{/if}
			</p>
		</div>
	{:else}
		<div
			data-part="queue"
			class="mt-2 max-h-[calc(100dvh-20rem)] overflow-y-auto overscroll-contain {fill ? 'min-[1201px]:max-h-none min-[1201px]:min-h-0 min-[1201px]:flex-1' : ''}"
		>
			{#each view.queue as c (c.id)}
				{@const words = primaryOf(c.ask)}
				{@const unseen = unseenStateOf(opened, c.id, c.revision)}
				{@const state = rowState[c.id]}
				<div
					role="button"
					tabindex="0"
					data-lens="r"
					data-part="row"
					class="bn-pressable is-row ny-line grid w-full cursor-pointer grid-cols-[22px_minmax(0,1fr)] gap-2.5 border-b px-3 py-2.5 text-left last:border-b-0"
					onclick={() => open(c)}
					onkeydown={(e) => (e.key === 'Enter' || e.key === ' ') && (e.preventDefault(), open(c))}
				>
					<!-- The kind, as one character. Colour on it is status, never decoration. -->
					<span
						aria-label={c.label}
						title={c.label}
						class="ny-glyph mt-px grid h-[22px] w-[22px] place-items-center border font-mono text-[11px] font-bold"
						style:color={GLYPH_TONE[c.glyphTone] ?? GLYPH_TONE.dim}
						style:border-color={c.glyphTone === 'dim' ? 'var(--bn-border)' : PREFLIGHT_BORDER}
					>
						{c.glyph}
					</span>

					<div class="min-w-0">
						<div class="bn-dim flex items-center gap-2 font-mono text-[9.5px] uppercase tracking-[0.1em]">
							{#if unseen}
								<span
									class="h-1.5 w-1.5 shrink-0 animate-pulse"
									title={unseen === 'new' ? 'New since you last looked' : 'Rewritten since you read it'}
									aria-label={unseen === 'new' ? 'new' : 'updated'}
									style:background={unseen === 'new' ? 'var(--bn-accent)' : 'var(--bn-warn)'}
								></span>
							{/if}
							<span class="truncate">{c.meta || 'board'}</span>
							<span class="ml-auto shrink-0">{ago(c.modifiedAt, now)}</span>
						</div>

						<!-- The document's own title, not the filename. -->
						<div class="bn-text mt-0.5 text-[12.5px] font-semibold leading-snug" title={c.name}>{textOf(c)}</div>

						{#if c.summary}
							<p class="bn-muted mt-1 line-clamp-2 text-[11.5px] leading-snug">{c.summary}</p>
						{/if}

						<!-- The Conductor shows its working. -->
						<p class="bn-dim mt-1 font-mono text-[10px]">
							why here · {c.why}{#if c.deadline}<span class={c.overdue ? 'font-bold' : ''} style:color={c.overdue ? 'var(--bn-err)' : 'var(--bn-warn)'}>{c.overdue ? ' · past due ' : ' · due '}{new Date(c.deadline).toISOString().slice(11, 16)}Z</span>{/if}
						</p>

						<!-- Answerable in place; the controls stop the click before it opens the review. -->
						<!-- svelte-ignore a11y_no_static_element_interactions, a11y_click_events_have_key_events -->
						<div class="mt-2 flex flex-wrap items-center gap-1.5" onclick={(e) => e.stopPropagation()} onkeydown={(e) => e.stopPropagation()}>
							{#if state === 'held'}
								<button
									type="button"
									class="ny-held inline-flex h-[26px] items-center gap-1.5 border px-3 font-mono text-[10.5px] font-bold"
									disabled
									title="FOUNDEROS_WRITES=0: the bridge guard held this. Nothing was sent or recorded.">held: writes off</button
								>
							{:else}
								<Pressable tone="primary" disabled={!!state} aria-busy={state === 'busy'} onclick={() => void approve(c)}>
									{#if state === 'busy'}<span class="ny-spin inline-block h-[10px] w-[10px] rounded-full border-[1.5px]"></span>{words.busy}{:else}{words.label}{/if}
								</Pressable>
							{/if}
							<Pressable tone="secondary" onclick={() => void dismiss(c)}>dismiss</Pressable>
							<button
								type="button"
								data-lens="c"
								class="bn-pressable is-dark ny-snooze inline-flex h-[26px] items-center border px-2.5 font-mono text-[10.5px] font-semibold"
								title="Hold it for two hours. Nothing is sent and no agent is told."
								onclick={() => snooze(c)}>snooze</button
							>
							<span class="bn-dim ml-auto font-mono text-[9.5px] uppercase tracking-[0.1em]">
								{c.label}{#if c.alsoAs && c.alsoAs.length > 0}{` · also .${c.alsoAs.join(', .')}`}{/if}
							</span>
						</div>
						{#if rowError[c.id]}
							<p class="mt-1 font-mono text-[10px]" style:color="var(--bn-err)">{rowError[c.id]}</p>
						{/if}
					</div>
				</div>
			{/each}
		</div>
	{/if}
</section>

<!-- Read the work, then approve or dismiss it, without leaving the page. -->
<ReviewPanel item={reviewing} decision={reviewing ? decisionOf(reviewing.id) : undefined} onDecide={decideFromPanel} onClose={() => (reviewing = null)} />

<style>
	.ny-line {
		border-color: var(--bn-border);
	}
	.ny-hover:hover:not(:disabled) {
		color: var(--bn-text);
	}
	/* border-os-err/50 + hover:bg-os-err/10 in prod: both no-ops (see PREFLIGHT_BORDER) */
	.ny-danger {
		border-color: rgb(229 231 235);
		border-radius: var(--bn-r-ctl, 6px);
		color: var(--bn-err);
	}
	.ny-rule {
		background: var(--bn-border);
	}
	.ny-empty {
		border-color: var(--bn-border);
		border-radius: var(--bn-r-panel, 10px);
	}
	.ny-glyph {
		border-radius: var(--bn-r-ctl, 6px);
	}
	.ny-snooze {
		border-color: var(--bn-border);
		border-radius: var(--bn-r-ctl, 6px);
		background: var(--bn-bg);
		color: var(--bn-text-2);
	}
	.ny-held {
		border-color: color-mix(in oklab, var(--bn-warn) 50%, transparent);
		border-radius: var(--bn-r-ctl, 6px);
		background: color-mix(in oklab, var(--bn-warn) 10%, transparent);
		color: var(--bn-warn);
		cursor: not-allowed;
	}
	.ny-spin {
		border-color: var(--bn-accent-ink);
		border-right-color: transparent;
		animation: bn-om-spin 0.8s linear infinite;
	}
</style>
