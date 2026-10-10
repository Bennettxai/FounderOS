<!-- Live connector map: a bar per connector, coloured by real state. An honest
     stand-in for a time series nobody stores (connector uptime has no history). -->
<script lang="ts">
	let { states }: { states: string[] } = $props();
	const W = 72;
	const H = 22;
	const GAP = 2;
	const items = $derived(states.slice(0, 16));
	const bw = $derived(Math.max(2, (W - GAP * (items.length - 1)) / Math.max(1, items.length)));
	const color = (s: string) => (s === 'connected' ? 'var(--bn-ok)' : s === 'error' ? 'var(--bn-err)' : 'var(--bn-text-3)');
	const barH = (s: string) => (s === 'connected' ? H - 4 : s === 'error' ? H - 9 : 6);
</script>

<svg width={W} height={H} viewBox="0 0 {W} {H}" aria-hidden="true">
	{#each items as s, i (i)}
		<rect x={(i * (bw + GAP)).toFixed(1)} y={(H - barH(s)).toFixed(1)} width={bw.toFixed(1)} height={barH(s)} fill={color(s)} opacity={s === 'connected' ? 1 : 0.75} />
	{/each}
</svg>
