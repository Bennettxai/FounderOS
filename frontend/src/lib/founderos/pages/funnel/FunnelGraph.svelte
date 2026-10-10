<!-- Lazy shell for the two funnel engines, the heaviest code on /os/funnel
     (FounderOS v1 components/FunnelGraphsLazy.tsx, the next/dynamic ssr:false
     contract): each engine is a dynamic import, and until it lands a skeleton
     with the same aspect ratio (radial 1100/680, space 1100/460) holds the
     space, so nothing shifts when it arrives. -->
<script lang="ts">
	import type { Component } from 'svelte';
	import type { FunnelNode, Segment, Summary } from './types';

	let {
		layout,
		nodes,
		segments,
		summary,
		stages,
		stageLabels,
		constants,
		initialLeadId = null
	}: {
		layout: 'flow' | 'radial';
		nodes: FunnelNode[];
		segments: Segment[];
		summary: Summary;
		stages: { id: string; label: string }[];
		stageLabels: Record<string, string>;
		constants: { stallDays: number; decayDays: number; decayFadeStart: number };
		initialLeadId?: string | null;
	} = $props();

	// eslint-disable-next-line @typescript-eslint/no-explicit-any
	let Engine = $state<Component<any> | null>(null);
	let failed = $state<string | null>(null);

	const loaders = {
		flow: () => import('./FunnelSpace.svelte'),
		radial: () => import('./FunnelRadial.svelte')
	};

	$effect(() => {
		const which = layout;
		let alive = true;
		Engine = null;
		failed = null;
		loaders[which]()
			.then((m) => {
				if (alive) Engine = m.default;
			})
			.catch((e: unknown) => {
				if (alive) failed = e instanceof Error ? e.message : 'failed to load';
			});
		return () => {
			alive = false;
		};
	});
</script>

{#if Engine}
	<Engine {nodes} {segments} {summary} {stages} {stageLabels} {constants} {initialLeadId} />
{:else if failed}
	<p class="py-6 text-center font-mono text-[11.5px]" style="color: var(--bn-err)">graph failed to load: {failed}</p>
{:else}
	<div
		data-testid="funnel-graph-skeleton"
		class="w-full animate-pulse overflow-hidden border {layout === 'radial' ? 'fn-radial-canvas' : ''}"
		style="aspect-ratio: {layout === 'radial' ? '1100 / 680' : '1100 / 460'}; border-color: var(--bn-border); background: var(--bn-surface)"
	></div>
{/if}
