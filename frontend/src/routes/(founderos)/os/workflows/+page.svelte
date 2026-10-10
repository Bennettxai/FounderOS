<!-- /os/workflows — scheduled tasks + the process map (FounderOS v1
     app/workflows/page.tsx, spec 6.17). -->
<script lang="ts">
	import { onMount } from 'svelte';
	import { founderosFetch } from '$lib/founderos/api';
	import WorkflowsPage from '$lib/founderos/pages/workflows/WorkflowsPage.svelte';
	import type { WorkflowsView } from '$lib/founderos/pages/workflows/types';

	let view = $state<WorkflowsView | null>(null);
	let error = $state<string | null>(null);

	async function load() {
		try {
			view = await founderosFetch<WorkflowsView>('/pages/workflows/view');
			error = null;
		} catch (err) {
			error = err instanceof Error ? err.message : String(err);
		}
	}

	onMount(load);
</script>

{#if error && !view}
	<p class="font-mono text-[11px]" style="color: var(--bn-err)">Could not load workflows: {error}</p>
{:else if !view}
	<p class="bn-dim font-mono text-[10px]">loading workflows…</p>
{:else}
	<WorkflowsPage {view} onchange={load} />
{/if}
