<!-- 50px count-up headline + dot chips + caption (FounderOS v1 slab.tsx BigStat).
     Honest: with no value and no display it reads "unknown", never 0. -->
<script lang="ts">
	import type { Snippet } from 'svelte';
	import './kit.css';
	import Chip from './Chip.svelte';
	import CountUp from './CountUp.svelte';
	import Render from './Render.svelte';
	import { isKnown, type CountKind, type StatChip } from './format';

	let {
		value,
		kind = 'int',
		display,
		unit,
		chips = [],
		caption,
		size = 50
	}: {
		value?: number | null;
		kind?: CountKind;
		/** Preformatted headline when the count formats do not fit (4.2M, 68%). */
		display?: string;
		unit?: string;
		chips?: StatChip[];
		caption?: string | Snippet;
		size?: 50 | 30;
	} = $props();
</script>

<div>
	<div class="flex flex-wrap items-center gap-2.5">
		<span
			data-part="bigstat-value"
			class="bn-text {size === 50 ? 'text-[50px] tracking-[-0.035em]' : 'text-[30px] tracking-[-0.03em]'} font-semibold leading-none tabular-nums"
		>
			<!-- one line on purpose: a text space between the two would sit at 50px -->
			{#if display != null}{display}{:else if isKnown(value)}<CountUp {value} {kind} />{:else}<span data-unknown class="bn-dim">unknown</span>{/if}{#if unit}<span class="bn-dim ml-1.5 text-[15px] font-normal tracking-normal">{unit}</span>{/if}</span
		>
		{#each chips as c (c.text)}
			<Chip tone={c.tone}>{c.text}</Chip>
		{/each}
	</div>
	{#if caption}
		<div class="bn-dim mt-2 text-[13px]"><Render value={caption} /></div>
	{/if}
</div>
