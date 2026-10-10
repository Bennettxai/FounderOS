<!-- /os/finances (spec 6.15): GET /api/founderos/pages/finances, rendered in
     the slab. An upload reloads the payload and moves the expense panel to the
     month it covered. -->
<script lang="ts">
	import { onMount } from 'svelte';
	import { founderosFetch } from '$lib/founderos/api';
	import { BigStat, Slab, SlabTitle } from '$lib/founderos/kit';
	import FinancesView from '$lib/founderos/pages/finances/FinancesView.svelte';
	import type { FinancesPayload } from '$lib/founderos/pages/finances/types';

	let data = $state<FinancesPayload | null>(null);
	let failure = $state<string | null>(null);
	let focus = $state<string | null>(null);

	async function load() {
		try {
			data = await founderosFetch<FinancesPayload>('/pages/finances');
			failure = null;
		} catch (e) {
			failure = e instanceof Error ? e.message : 'unreachable';
		}
	}

	async function uploaded(months: string[]) {
		await load();
		if (months.length > 0) focus = months[months.length - 1];
	}

	onMount(load);
</script>

{#if data}
	<FinancesView {data} {focus} onuploaded={uploaded} />
{:else}
	<Slab>
		<SlabTitle eyebrow="money · every processor, one view" title="Finances" />
		{#if failure}
			<BigStat value={null} caption={`finances unreachable: ${failure}`} />
		{:else}
			<BigStat display="…" caption="loading finances: processors, statements, bank income" />
		{/if}
	</Slab>
{/if}
