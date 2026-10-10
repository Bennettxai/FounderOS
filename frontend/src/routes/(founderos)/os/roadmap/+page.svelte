<!-- /os/roadmap: the build plan, phases + quarters (FounderOS v1
     app/roadmap/page.tsx). Data: GET /api/founderos/pages/roadmap. -->
<script lang="ts">
	import { onMount } from 'svelte';
	import { founderosFetch } from '$lib/founderos/api';
	import { PageHeader, Rise, Slab } from '$lib/founderos/kit';
	import RoadmapBoard from '$lib/founderos/pages/roadmap/RoadmapBoard.svelte';
	import type { RoadmapBody } from '$lib/founderos/pages/roadmap/types';

	let data = $state<RoadmapBody | null>(null);
	let failure = $state<string | null>(null);

	onMount(() => {
		founderosFetch<RoadmapBody>('/pages/roadmap')
			.then((b) => (data = b))
			.catch((e: unknown) => (failure = e instanceof Error ? e.message : 'unreachable'));
	});
</script>

<div>
	<PageHeader eyebrow="build plan" title="Roadmap">
		{#snippet right()}
			{#if data}
				<span class="bn-muted font-mono text-[10px]">{data.shipped}/{data.total} shipped</span>
			{/if}
		{/snippet}
	</PageHeader>
	<Rise i={1}>
		{#if data && (data.items.length > 0 || data.phases.length > 0)}
			<RoadmapBoard phases={data.phases} items={data.items} departments={data.departments} />
		{:else if data}
			<Slab><p class="bn-dim font-mono text-[12px]">No roadmap yet. The phases and roadmap tables are empty.</p></Slab>
		{:else if failure}
			<Slab><p class="font-mono text-[12px]" style="color: var(--bn-err)">The roadmap could not be read: {failure}</p></Slab>
		{:else}
			<Slab><p class="bn-dim animate-pulse font-mono text-[12px]">loading roadmap…</p></Slab>
		{/if}
	</Rise>
</div>
