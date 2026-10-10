<!-- The Deal Volume bar (FounderOS v1 VolumeMeter.tsx): a label/value row over a
     track whose hatched, glowing fill sweeps out from the left. Unknown reads
     "unknown" over an empty dashed track instead of a zero bar. -->
<script lang="ts">
	import './kit.css';
	import { meterWidth } from './format';

	const EASE = 'cubic-bezier(.2,.7,.2,1)';

	let {
		label,
		frac,
		display,
		hue,
		delay = 0,
		compact = false
	}: {
		label: string;
		frac: number | null;
		display: string;
		hue: string;
		delay?: number;
		/** A denser row (smaller type, thinner track) for long lists like /finances' categories. */
		compact?: boolean;
	} = $props();

	const w = $derived(meterWidth(frac));
	const fill = $derived(
		w === null
			? ''
			: [
					`width: ${Math.round(w * 1000) / 10}%`,
					`background: linear-gradient(90deg, transparent 72%, color-mix(in oklab, ${hue} 60%, var(--bn-text)) 100%), repeating-linear-gradient(45deg, ${hue}, ${hue} 6px, color-mix(in oklab, ${hue} 45%, transparent) 6px, color-mix(in oklab, ${hue} 45%, transparent) 12px)`,
					`box-shadow: 0 0 14px color-mix(in oklab, ${hue} 45%, transparent), inset 0 0 5px color-mix(in oklab, ${hue} 55%, transparent)`,
					`animation: bn-vol-meter-in 1.2s ${EASE} ${delay}ms both`
				].join('; ')
	);
</script>

<div>
	<div class="flex items-baseline justify-between gap-3">
		<span class="bn-muted {compact ? 'min-w-0 truncate text-[12.5px]' : 'text-[13.5px]'}">{label}</span>
		{#if w === null}
			<span class="bn-dim {compact ? 'shrink-0 text-[12.5px]' : 'text-[14px]'} font-semibold">unknown</span>
		{:else}
			<span class="bn-text {compact ? 'shrink-0 text-[12.5px]' : 'text-[14px]'} font-semibold tabular-nums">{display}</span>
		{/if}
	</div>
	<div class="bn-vol-track {compact ? 'mt-1 h-[6px]' : 'mt-2 h-[10px]'} overflow-hidden rounded-full" data-unknown={w === null ? '' : undefined}>
		{#if w !== null}
			<div class="bn-vol-fill h-full rounded-full" style={fill}></div>
		{/if}
	</div>
</div>
