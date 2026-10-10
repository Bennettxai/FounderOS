<!-- /os/personas (spec 6.19): the platform variants, one template at a time
     (FounderOS v1 app/personas/page.tsx). Data: GET /api/founderos/pages/personas. -->
<script lang="ts">
	import { onMount } from 'svelte';
	import { founderosFetch } from '$lib/founderos/api';
	import { Badge, PageHeader, Slab } from '$lib/founderos/kit';
	import PersonasViewer from '$lib/founderos/pages/personas/PersonasViewer.svelte';
	import type { Persona, PersonasBody } from '$lib/founderos/pages/personas/types';

	let personas = $state<Persona[] | null>(null);
	let failure = $state<string | null>(null);

	onMount(() => {
		founderosFetch<PersonasBody>('/pages/personas')
			.then((b) => (personas = b.personas ?? []))
			.catch((e: unknown) => (failure = e instanceof Error ? e.message : 'unreachable'));
	});
</script>

<PageHeader eyebrow="platform variants" title="Personas">
	{#snippet right()}
		{#if personas}
			<Badge tone="accent">{personas.length} templates</Badge>
		{:else if failure}
			<Badge tone="err">unavailable</Badge>
		{:else}
			<Badge>loading</Badge>
		{/if}
	{/snippet}
</PageHeader>

<div class="bn-rise" style="--rise-i: 1">
	{#if personas && personas.length > 0}
		<PersonasViewer {personas} />
	{:else if personas}
		<Slab><p class="bn-dim font-mono text-[12px]">No personas yet. The persona templates table is empty.</p></Slab>
	{:else if failure}
		<Slab><p class="font-mono text-[12px]" style="color: var(--bn-err)">Personas could not be read: {failure}</p></Slab>
	{:else}
		<Slab><p class="bn-dim animate-pulse font-mono text-[12px]">loading personas…</p></Slab>
	{/if}
</div>
