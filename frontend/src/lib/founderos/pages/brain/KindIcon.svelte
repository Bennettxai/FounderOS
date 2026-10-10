<!-- One knowledge-graph kind's glyph (FounderOS v1 KnowledgeGraph CAT icons):
     lucide glyphs for pillars, SOPs, humans, tools and the core; the OS mark
     for board seats and the Vantage mark for AI agents, flattened to white
     like production's .mark-adaptive. Works inside the graph's SVG (pass x/y)
     and in HTML chrome (legend, directory). Size is set inline as well as by
     attribute: app.css's unlayered `svg { height: auto }` would otherwise
     stretch a nested <svg>. -->
<script lang="ts">
	import { ClipboardList, Sparkles, UserRound, Users, Wrench } from '$lib/founderos/icons';
	import vantageMark from './vantage-mark.png';
	import osMark from '$lib/founderos/kit/os-emblem.png';
	import type { KGNode } from './types';

	let {
		kind,
		size,
		x = undefined,
		y = undefined,
		strokeWidth = 2
	}: { kind: KGNode['kind']; size: number; x?: number; y?: number; strokeWidth?: number } = $props();

	const LUCIDE = { self: Sparkles, team: Users, task: ClipboardList, person: UserRound, tool: Wrench } as const;
	const style = $derived(`width: ${size}px; height: ${size}px; max-width: none`);
</script>

{#if kind === 'board' || kind === 'employee'}
	<svg viewBox="0 0 24 24" fill="none" aria-hidden="true" data-icon={kind} {x} {y} width={size} height={size} {style}>
		<image
			href={kind === 'board' ? osMark : vantageMark}
			x={kind === 'board' ? 1 : 1.5}
			y={kind === 'board' ? 1 : 1.5}
			width={kind === 'board' ? 22 : 21}
			height={kind === 'board' ? 22 : 21}
			preserveAspectRatio="xMidYMid meet"
			style="filter: brightness(0) invert(1)"
		/>
	</svg>
{:else}
	{@const Icon = LUCIDE[kind]}
	<Icon data-icon={kind} {x} {y} size={size} {strokeWidth} {style} aria-hidden="true" />
{/if}
