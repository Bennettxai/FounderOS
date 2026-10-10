<!-- The Pipeline hero (FounderOS v1 components/PipelineChart.tsx): column headers
     (42px count-up numeral + label + mono $ line, each a working filter
     button) above hatched step columns that fade out at the bottom, the
     active column solid, a tooltip pill over it, and an optional prompt bar
     melting out of the chart base. Rounded caps, pill and active column as on production. -->
<script lang="ts" generics="T extends { label: string; value: number; usd?: number; note?: string }">
	import type { Snippet } from 'svelte';
	import CountUp from '$lib/founderos/kit/CountUp.svelte';

	const EASE = 'cubic-bezier(.2,.7,.2,1)';

	let {
		stages,
		active,
		onSelect,
		search,
		svgHeight = 220
	}: {
		stages: T[];
		active: number;
		onSelect?: (stage: T, index: number) => void;
		search?: Snippet;
		svgHeight?: number;
	} = $props();

	const uid = $props.id();
	const hatch = `bn-pc-hatch-${uid}`;
	const solid = `bn-pc-solid-${uid}`;
	const fadeg = `bn-pc-fadeg-${uid}`;
	const fade = `bn-pc-fade-${uid}`;

	const W = 1000;
	const AXIS = 44;
	const H = $derived(svgHeight + 40);
	const max = $derived(Math.max(...stages.map((s) => s.value), 1));
	const colW = $derived((W - AXIS) / Math.max(stages.length, 1));
	const barW = $derived(colW * 0.62);
	const h = (v: number) => (v === 0 ? 10 : 34 + (v / max) * (H - 54));
	const x0 = (i: number) => AXIS + i * colW + 4;
	const ticks = $derived([0, 0.25, 0.5, 0.75, 1].map((f) => Math.round(max * f)));
	const activeStage = $derived(stages[active] as T | undefined);
	const tip = $derived.by(() => {
		if (!activeStage?.note) return null;
		const top = ((H - h(activeStage.value)) / H) * svgHeight;
		const centerPct = ((AXIS + active * colW + 4 + barW / 2) / W) * 100;
		const tx = centerPct < 15 ? '0%' : centerPct > 85 ? '-100%' : '-50%';
		return { left: centerPct, top: Math.max(6, top - 40), tx };
	});
	const usd = (v: number) => `$${Math.round(v).toLocaleString('en-US')}`;
</script>

<div data-part="pipeline">
	<div class="px-6 pt-4">
		<div class="grid" style="grid-template-columns: repeat({stages.length}, 1fr); margin-left: {(AXIS / W) * 100}%">
			{#each stages as s, i (s.label)}
				<button
					type="button"
					data-stage={i}
					aria-pressed={i === active}
					onclick={() => onSelect?.(s, i)}
					class="bn-pc-col bn-pressable px-5 pb-3 text-left {i > 0 ? 'bn-border border-l' : ''}"
				>
					<div class="text-[12.5px] {i === active ? 'bn-muted' : 'bn-dim'}">{s.label}</div>
					<div class="mt-0.5 text-[42px] font-semibold leading-none tabular-nums tracking-[-0.03em] {i === active ? 'bn-text' : 'bn-dim'}">
						<CountUp value={s.value} />
					</div>
					{#if s.usd !== undefined}
						<div class="mt-1 font-mono text-[11.5px] tabular-nums {i === active ? 'bn-accent' : 'bn-dim'}">{usd(s.usd)}</div>
					{/if}
				</button>
			{/each}
		</div>

		<div class="relative" style="mask-image: linear-gradient(to bottom, black 74%, transparent 100%); -webkit-mask-image: linear-gradient(to bottom, black 74%, transparent 100%)">
			<svg viewBox="0 0 {W} {H}" class="block w-full" style="height: {svgHeight}px" preserveAspectRatio="none" aria-hidden="true">
				<defs>
					<pattern id={hatch} width="9" height="9" patternUnits="userSpaceOnUse" patternTransform="rotate(45)">
						<rect width="9" height="9" style="fill: color-mix(in oklab, var(--bn-accent) 14%, transparent)" />
						<rect width="3.5" height="9" style="fill: color-mix(in oklab, var(--bn-accent) 42%, transparent)" />
					</pattern>
					<linearGradient id={solid} x1="0" y1="0" x2="0" y2="1">
						<stop offset="0" style="stop-color: var(--bn-accent)" />
						<!-- prod --pc-shade: color-mix(in oklab, var(--accent) 25%, var(--bg)) -->
						<stop offset="1" style="stop-color: color-mix(in oklab, var(--bn-accent) 55%, color-mix(in oklab, var(--bn-accent) 25%, var(--bn-bg)))" />
					</linearGradient>
					<linearGradient id={fadeg} x1="0" y1="0" x2="0" y2="1">
						<stop offset="0" stop-color="#fff" stop-opacity="0.95" />
						<stop offset="1" stop-color="#fff" stop-opacity="0.14" />
					</linearGradient>
					<mask id={fade}>
						<rect width={W} height={H} fill="url(#{fadeg})" />
					</mask>
				</defs>
				{#each ticks as t, i (i)}
					{@const y = H - h(t) + (t === 0 ? 10 : 0)}
					<g>
						<text x={AXIS - 10} y={y + 3} text-anchor="end" font-size="11" style="fill: var(--bn-text-3); font-variant-numeric: tabular-nums">{t}</text>
						<line x1={AXIS} x2={W} y1={y} y2={y} stroke-width="0.5" opacity="0.5" style="stroke: var(--bn-border)" />
					</g>
				{/each}
				{#each stages as s, i (s.label)}
					{@const isActive = i === active}
					{@const bx = x0(i)}
					{@const top = H - h(s.value)}
					{@const next = stages[i + 1]}
					{@const fill = isActive ? `url(#${solid})` : `url(#${hatch})`}
					<g mask={isActive ? undefined : `url(#${fade})`} data-bar={i} data-active={isActive ? '' : undefined}>
						<rect
							x={bx}
							y={top}
							width={barW}
							height={H - top}
							rx={isActive ? 3 : 0}
							{fill}
							style="transform-box: fill-box; transform-origin: bottom; animation: bn-pc-grow .8s {EASE} {200 + i * 110}ms both"
						/>
						{#if next}
							<polygon
								points="{bx + barW},{top} {x0(i + 1)},{H - h(next.value)} {x0(i + 1)},{H} {bx + barW},{H}"
								{fill}
								opacity={isActive ? 0.5 : 0.55}
								style="animation: bn-fade .9s {EASE} {380 + i * 110}ms both"
							/>
						{/if}
					</g>
				{/each}
				{#each stages as s, i (`cap-${s.label}`)}
					<rect
						data-cap={i}
						x={x0(i) + barW / 2 - 16}
						y={H - h(s.value) - 12}
						width="32"
						height="6"
						rx="3"
						style="fill: {i === active ? 'var(--bn-accent)' : 'color-mix(in oklab, var(--bn-text) 25%, transparent)'}; animation: bn-fade .5s {EASE} {700 + i * 90}ms both"
					/>
				{/each}
			</svg>
			{#if tip && activeStage}
				<div
					data-part="tooltip"
					class="bn-border pointer-events-none absolute flex items-center gap-2 whitespace-nowrap rounded-full border px-3.5 py-1.5 text-[12px] backdrop-blur"
					style="left: {tip.left}%; top: {tip.top}px; transform: translateX({tip.tx}); background: color-mix(in oklab, var(--bn-bg) 72%, transparent); box-shadow: 0 8px 24px -10px rgba(0,0,0,.6); animation: bn-fade .6s {EASE} 1100ms both"
				>
					<span class="bn-text font-semibold tabular-nums">{Math.round(activeStage.value)}</span>
					<span class="bn-muted">{activeStage.note}</span>
				</div>
			{/if}
		</div>
	</div>

	{#if search}
		<div class="relative z-[2] mx-6 -mt-12">
			<div
				class="rounded-[16px] p-3 backdrop-blur"
				style="background: linear-gradient(180deg, transparent, color-mix(in oklab, var(--bn-accent) 9%, transparent) 40%, color-mix(in oklab, var(--bn-accent) 12%, transparent))"
			>
				{@render search()}
			</div>
		</div>
	{/if}
</div>

<style>
	@keyframes -global-bn-pc-grow {
		from {
			transform: scaleY(0);
		}
		to {
			transform: scaleY(1);
		}
	}
	.bn-pc-col:hover {
		background: color-mix(in oklab, var(--bn-text) 4%, transparent);
	}

</style>
