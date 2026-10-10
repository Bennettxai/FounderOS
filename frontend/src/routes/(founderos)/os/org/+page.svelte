<!-- /os/org — the agent hierarchy board (FounderOS v1 app/org/page.tsx, spec 6.5). -->
<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/state';
	import { founderosFetch } from '$lib/founderos/api';
	import PageHeader from '$lib/founderos/kit/PageHeader.svelte';
	import OrgBoard from '$lib/founderos/pages/org/OrgBoard.svelte';
	import type { OrgView } from '$lib/founderos/pages/org/types';

	let view = $state<OrgView | null>(null);
	let error = $state<string | null>(null);
	const ventureId = $derived(page.url.searchParams.get('venture'));

	onMount(() => {
		founderosFetch<OrgView>('/pages/org')
			.then((v) => (view = v))
			.catch((err) => (error = err instanceof Error ? err.message : String(err)));
	});
</script>

<PageHeader title="Agent Hierarchy" />

{#if error}
	<p class="font-mono text-[11px]" style="color: var(--bn-err)">Could not load the org: {error}</p>
{:else if !view}
	<p class="bn-dim font-mono text-[10px]">loading the roster…</p>
{:else}
	<OrgBoard {view} {ventureId} />
{/if}
