<!-- /os/tasks — agent work: hero, cron rhythm, owners, then the cron strip,
     the board queue and the local kanban (FounderOS v1 app/tasks/page.tsx). -->
<script lang="ts">
	import { onMount } from 'svelte';
	import { founderosFetch } from '$lib/founderos/api';
	import TasksPage from '$lib/founderos/pages/tasks/TasksPage.svelte';
	import type { TasksView } from '$lib/founderos/pages/tasks/types';

	let view = $state<TasksView | null>(null);
	let error = $state<string | null>(null);

	onMount(() => {
		founderosFetch<TasksView>('/pages/tasks')
			.then((v) => (view = v))
			.catch((err) => (error = err instanceof Error ? err.message : String(err)));
	});
</script>

{#if error}
	<p class="font-mono text-[11px]" style="color: var(--bn-err)">Could not load tasks: {error}</p>
{:else if !view}
	<p class="bn-dim font-mono text-[10px]">loading the task queues…</p>
{:else}
	<TasksPage {view} />
{/if}
