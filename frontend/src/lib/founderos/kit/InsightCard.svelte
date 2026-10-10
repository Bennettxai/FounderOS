<!-- The ONE insight card a page may carry (FounderOS v1 slab.tsx InsightCard):
     drifting glow, grain, a glass shard, a big number in the insight ink, 8 progress ticks.
     Built from theme tokens only; unknown reads unknown. `compact` is the
     shorter tile for a strip inside another card (/workflows). -->
<script lang="ts">
	import type { Snippet } from 'svelte';
	import { Lightbulb } from '$lib/founderos/icons';
	import './kit.css';
	import CountUp from './CountUp.svelte';
	import Render from './Render.svelte';
	import { insightTicks, isKnown, type CountKind } from './format';

	let {
		badge,
		icon,
		value,
		kind = 'int',
		display,
		headline,
		body,
		frac = 0,
		i = 5,
		compact = false,
		class: className = ''
	}: {
		badge: string;
		icon?: Snippet;
		value?: number | null;
		kind?: CountKind;
		display?: string;
		headline: string | Snippet;
		body?: string | Snippet;
		frac?: number | null;
		i?: number;
		/** A shorter tile: 32px number beside the badge, tighter padding, a smaller shard. */
		compact?: boolean;
		class?: string;
	} = $props();

	const lit = $derived(insightTicks(frac));

	const GRAIN =
		"url(\"data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='160' height='160'%3E%3Cfilter id='n'%3E%3CfeTurbulence type='fractalNoise' baseFrequency='0.9' numOctaves='2'/%3E%3C/filter%3E%3Crect width='160' height='160' filter='url(%23n)' opacity='0.55'/%3E%3C/svg%3E\")";

	const style = $derived(
		[
			`--rise-i: ${i}`,
			`background: ${[
				'radial-gradient(120% 90% at 85% 8%, color-mix(in oklab, var(--bn-glow-a) 55%, transparent), transparent 60%)',
				'radial-gradient(130% 110% at 12% 92%, color-mix(in oklab, var(--bn-glow-b) 55%, transparent), transparent 62%)',
				'radial-gradient(110% 110% at 55% 55%, color-mix(in oklab, var(--bn-glow-c) 45%, transparent), transparent 70%)',
				'var(--bn-insight-base)'
			].join(', ')}`,
			'background-size: 160% 160%',
			// the rise beat and the slow glow drift share one animation list
			'animation: bn-rise .6s var(--bn-ease) calc(var(--rise-i) * 90ms) backwards, bn-drift 14s ease-in-out 1s infinite alternate'
		].join('; ')
	);
</script>

<div class="bn-rise bn-rise-card bn-insight relative min-w-0 overflow-hidden rounded-[12px] {className}" {style}>
	<div class="pointer-events-none absolute inset-0" style="background-image: {GRAIN}; mix-blend-mode: overlay; opacity: 0.85"></div>
	<div
		class="bn-insight-shard pointer-events-none absolute rotate-[24deg] {compact
			? '-right-10 -top-12 h-32 w-32 rounded-[24px]'
			: '-right-14 -top-16 h-56 w-56 rounded-[36px]'}"
	></div>
	<div data-part="insight-inner" class="relative flex h-full flex-col {compact ? 'px-4 py-3' : 'px-6 py-5'}">
		<!-- compact: the badge and the number share one line -->
		<div data-part="insight-top" class={compact ? 'flex items-center justify-between gap-3' : 'contents'}>
			<span class="bn-insight-pill inline-flex w-fit items-center gap-1.5 rounded-full px-3 py-1 text-[12px] backdrop-blur">
				{#if icon}{@render icon()}{:else}<Lightbulb size={13} strokeWidth={1.7} />{/if}
				{badge}
			</span>
			<div data-part="insight-value" class="{compact ? 'text-[32px]' : 'mt-4 text-[64px]'} font-semibold leading-none tracking-[-0.03em] tabular-nums">
				{#if display != null}{display}{:else if isKnown(value)}<CountUp {value} {kind} />{:else}<span data-unknown class="bn-dim">unknown</span>{/if}
			</div>
		</div>
		<div class="{compact ? 'mt-2 text-[13px]' : 'mt-2 text-[16px]'} font-semibold leading-snug"><Render value={headline} /></div>
		{#if body}<div class="bn-insight-body {compact ? 'mt-1 text-[11.5px] leading-snug' : 'mt-1.5 text-[12.5px] leading-relaxed'}"><Render value={body} /></div>{/if}
		<div class="mt-auto flex gap-1.5 {compact ? 'pt-2.5' : 'pt-4'}">
			{#each { length: 8 } as _, t (t)}
				<span data-tick={t < lit ? 'on' : 'off'} class="h-[3px] flex-1 rounded-full"></span>
			{/each}
		</div>
	</div>
</div>
