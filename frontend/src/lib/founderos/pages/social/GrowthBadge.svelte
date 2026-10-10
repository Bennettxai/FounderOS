<!-- Growth chip (SocialStats.tsx GrowthBadge): an em dash when history is too
     short, never a fake 0. -->
<script lang="ts">
	import { Minus, TrendingDown, TrendingUp } from '$lib/founderos/icons';
	import { formatPct } from './lib';

	let { label, value }: { label: string; value: number | null } = $props();
	const Icon = $derived(value == null ? Minus : value >= 0 ? TrendingUp : TrendingDown);
</script>

<div class="soc-chip flex items-center gap-1.5 rounded-[6px] px-2 py-1" data-growth={label}>
	<Icon size={12} class={value == null ? 'bn-dim' : 'bn-text'} />
	<span class="text-xs font-semibold {value == null ? 'bn-dim' : 'bn-text'}">{formatPct(value)}</span>
	<span class="bn-dim text-[10px] uppercase tracking-wider">{label}</span>
</div>
