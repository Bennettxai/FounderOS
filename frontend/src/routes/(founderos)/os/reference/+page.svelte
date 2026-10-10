<!-- /os/reference: the reference model's operating domains (FounderOS v1
     app/reference/page.tsx). Data: GET /api/founderos/pages/reference. -->
<script lang="ts">
	import { onMount } from 'svelte';
	import { founderosFetch } from '$lib/founderos/api';
	import { PageHeader, Rise, Slab } from '$lib/founderos/kit';
	import DomainGrid from '$lib/founderos/pages/reference/DomainGrid.svelte';
	import type { Domain, ReferenceBody } from '$lib/founderos/pages/reference/types';

	let domains = $state<Domain[] | null>(null);
	let failure = $state<string | null>(null);

	onMount(() => {
		founderosFetch<ReferenceBody>('/pages/reference')
			.then((b) => (domains = b.domains ?? []))
			.catch((e: unknown) => (failure = e instanceof Error ? e.message : 'unreachable'));
	});
</script>

<div>
	<PageHeader eyebrow="operating domains" title="Reference Model" />
	<Rise i={1}>
		{#if domains && domains.length > 0}
			<DomainGrid {domains} />
		{:else if domains}
			<Slab><p class="bn-dim font-mono text-[12px]">No domains yet. The reference model table is empty.</p></Slab>
		{:else if failure}
			<Slab><p class="font-mono text-[12px]" style="color: var(--bn-err)">The reference model could not be read: {failure}</p></Slab>
		{:else}
			<Slab><p class="bn-dim animate-pulse font-mono text-[12px]">loading reference model…</p></Slab>
		{/if}
	</Rise>
</div>
