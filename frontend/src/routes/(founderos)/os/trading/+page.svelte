<!-- /os/trading (spec 6.9): FounderOS v1 app/trading/page.tsx. Reads
     GET /api/founderos/pages/trading (the tradingPayload() equivalent) and
     re-reads it every 60s and on demand, like the source board. -->
<script lang="ts">
	import { onMount } from 'svelte';
	import { founderosFetch } from '$lib/founderos/api';
	import { Slab } from '$lib/founderos/kit';
	import TradingBoard from '$lib/founderos/pages/trading/TradingBoard.svelte';
	import type { TradingPayload } from '$lib/founderos/pages/trading/types';

	let data = $state<TradingPayload | null>(null);
	let error = $state<string | null>(null);
	let refreshError = $state<string | null>(null);
	let busy = $state(false);

	const reason = (e: unknown) => (e instanceof Error ? e.message : String(e));

	async function load() {
		if (busy) return;
		busy = true;
		try {
			data = await founderosFetch<TradingPayload>('/pages/trading');
			error = null;
			refreshError = null;
		} catch (e) {
			// Keep the last good board; say the refresh failed rather than blanking it.
			if (data) refreshError = reason(e);
			else error = reason(e);
		} finally {
			busy = false;
		}
	}

	onMount(() => {
		load();
		const t = setInterval(load, 60_000);
		return () => clearInterval(t);
	});
</script>

{#if data}
	{#if refreshError}
		<div class="mb-3 border px-4 py-2 font-mono text-[11px]" style="border-color: var(--bn-warn); color: var(--bn-warn)">
			refresh failed, showing the board from {data.at.slice(11, 16)} UTC: {refreshError}
		</div>
	{/if}
	<TradingBoard {data} {busy} onRefresh={load} />
{:else if error}
	<Slab>
		<div class="font-mono text-[10px] font-bold uppercase tracking-[0.26em]" style="color: var(--bn-err)">Trading unavailable</div>
		<p class="bn-muted mt-3 font-mono text-[12px]">{error}</p>
		<button type="button" class="bn-muted mt-4 border px-3 py-1 font-mono text-[11px]" style="border-color: var(--bn-border)" onclick={load}>retry</button>
	</Slab>
{:else}
	<Slab>
		<div class="bn-dim font-mono text-[11px]">loading the trading board…</div>
	</Slab>
{/if}
