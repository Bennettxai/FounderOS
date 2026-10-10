<!-- /os/clients (spec 6.13), ported from FounderOS v1 app/clients/page.tsx. -->
<script lang="ts">
	import { onMount } from 'svelte';
	import { founderosFetch, FounderosAuthError } from '$lib/founderos/api';
	import { Slab, SlabTitle } from '$lib/founderos/kit';
	import ClientsView from '$lib/founderos/pages/clients/ClientsView.svelte';
	import type { ClientsPayload } from '$lib/founderos/pages/clients/types';

	let data = $state<ClientsPayload | null>(null);
	let error = $state<string | null>(null);
	let loading = $state(true);

	async function load() {
		try {
			data = await founderosFetch<ClientsPayload>('/pages/clients');
			error = null;
		} catch (e) {
			if (e instanceof FounderosAuthError) return; // already sent to /login
			error = e instanceof Error ? e.message : String(e);
		} finally {
			loading = false;
		}
	}

	onMount(() => {
		void load();
	});
</script>

{#if data}
	<ClientsView {data} onchange={load} />
{:else}
	<Slab>
		<SlabTitle eyebrow="client work · draft first" title="Clients" />
		{#if loading}
			<p data-state="loading" class="bn-dim font-mono text-[12px]">Loading client work…</p>
		{:else if error}
			<p data-state="error" role="alert" class="font-mono text-[12px]" style="color: var(--bn-err)">
				Client work is unreachable: {error}. Nothing is shown rather than an empty board.
			</p>
		{/if}
	</Slab>
{/if}
