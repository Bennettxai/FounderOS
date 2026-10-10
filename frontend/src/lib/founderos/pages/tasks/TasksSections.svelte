<!-- The Tasks tab body on /os/agents (FounderOS v1 app/agents/page.tsx `tasks`,
     2026-09-24: "Remove the Tasks section entirely and bake it into the Agents
     tab"): the scheduled-jobs strip, the board's OPEN queue (finished work
     lives in Deliverables now) and the OS's own kanban. No hero: the retired
     /tasks page's dashboard did not come along. -->
<script lang="ts">
	import BoardQueue from './BoardQueue.svelte';
	import TaskCronStrip from './TaskCronStrip.svelte';
	import TaskKanban from './TaskKanban.svelte';
	import type { BoardIssue, TasksView } from './types';

	let { view, issues, boardUrl }: { view: TasksView; issues: BoardIssue[]; boardUrl: string | null } = $props();

	const openIssues = $derived(issues.filter((i) => i.status !== 'done'));
</script>

<div class="bn-tasks-tab">
	<TaskCronStrip i={2} jobs={view.jobs} />
	<BoardQueue i={3} initialIssues={openIssues} {boardUrl} />
	<TaskKanban i={4} initialTasks={view.tasks} agentNames={view.agentNames} />
</div>

<style>
	/* the tab bar already spaces the first card */
	.bn-tasks-tab > :global(:first-child) {
		margin-top: 0 !important;
	}
</style>
