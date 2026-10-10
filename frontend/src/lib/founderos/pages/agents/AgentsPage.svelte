<!-- /os/agents (spec 6.4), FounderOS v1 app/agents/page.tsx: the live
     Paperclip board in the slab look. The volume rows lead (run activity,
     Agent Volume, runs by seat, task lanes, Needs you); the cockpit follows
     with tabs Roster (board strip + Conductor rail), Needs You, Deliverables,
     Hermes. The board is polled every 4s while the tab is visible and never
     stacks requests. -->
<script lang="ts">
	import { onDestroy, onMount } from 'svelte';
	import { ArrowUpRight } from '$lib/founderos/icons';
	import { founderosFetch } from '$lib/founderos/api';
	import Slab from '$lib/founderos/kit/Slab.svelte';
	import SlabCard from '$lib/founderos/kit/SlabCard.svelte';
	import SlabTitle from '$lib/founderos/kit/SlabTitle.svelte';
	import AgentsTabs from './AgentsTabs.svelte';
	import AgentsVolumePanel from './AgentsVolumePanel.svelte';
	import BoardLive from './BoardLive.svelte';
	import ConductorRail from './ConductorRail.svelte';
	import { parseAgentsTab } from './agents-tabs';
	import { PILL } from './ui';
	import type { AgentsView, BoardLive as LiveBody } from './types';

	const initialTab = parseAgentsTab(typeof window === 'undefined' ? null : new URLSearchParams(window.location.search).get('tab'));

	const POLL_MS = 4000;

	let view = $state<AgentsView | null>(null);
	let error = $state<string | null>(null);
	let inflight = false;
	let timer: ReturnType<typeof setInterval> | null = null;

	async function tick() {
		if (!view || inflight || (typeof document !== 'undefined' && document.hidden)) return;
		inflight = true;
		try {
			const live = await founderosFetch<LiveBody>('/pages/board/live');
			const { volume, stats, ...board } = live;
			view = { ...view, board, volume, stats };
		} catch {
			/* transient poll failure: keep the last snapshot on screen */
		} finally {
			inflight = false;
		}
	}

	onMount(() => {
		founderosFetch<AgentsView>('/pages/agents/view')
			.then((v) => (view = v))
			.catch((err) => (error = err instanceof Error ? err.message : String(err)));
		timer = setInterval(tick, POLL_MS);
	});
	onDestroy(() => timer && clearInterval(timer));

	const conductorModel = $derived(view?.board.agents.find((a) => a.name === 'Conductor')?.model ?? null);
</script>

<Slab>
	<SlabTitle eyebrow="runtime" title="Real Agents" meta="live Paperclip board · polls every 4s · Conductor on the rail">
		{#snippet right()}
			{#if view?.boardUrl}
				<a href={view.boardUrl} target="_blank" rel="noreferrer" class={PILL}>Open board <ArrowUpRight size={13} strokeWidth={1.7} /></a>
			{/if}
			<a href="/os/workflows" class={PILL}>Workflows</a>
		{/snippet}
	</SlabTitle>

	<!-- Hero + second row: run activity, Agent Volume, seats, lanes, insight.
	     Until the first board read lands they hold their place empty. -->
	{#if view}
		<AgentsVolumePanel volume={view.volume} connected={view.board.connected} />
	{/if}

	<!-- The cockpit owns one viewport of height; long lists scroll inside.
	     It renders at once like prod: Needs You / Deliverables read local files
	     and must not wait on the (possibly slow) board read. -->
	<SlabCard i={7} class="mt-6 p-5">
		<div class="flex flex-col xl:h-[calc(100dvh-12rem)]">
			<AgentsTabs hermesUrl={view?.hermesUrl ?? ''} boardUrl={view?.boardUrl ?? null} {initialTab} issues={view?.board.issues ?? []}>
				<div class="grid gap-6 xl:h-full xl:min-h-0 xl:grid-cols-[minmax(0,1fr)_400px]">
					<div class="order-2 min-h-0 min-w-0 xl:order-none xl:col-start-1 xl:row-start-1 xl:h-full">
						{#if error}
							<p class="font-mono text-[11px]" style:color="var(--bn-err)">Could not load the board: {error}</p>
						{:else if !view}
							<p class="bn-dim font-mono text-[10px]">reading the board…</p>
						{:else}
							<BoardLive board={view.board} stats={view.stats} boardUrl={view.boardUrl} />
						{/if}
					</div>
					<div class="order-1 min-h-0 xl:order-none xl:col-start-2 xl:row-start-1 xl:h-full">
						<ConductorRail model={conductorModel} />
					</div>
				</div>
			</AgentsTabs>
		</div>
	</SlabCard>
</Slab>

