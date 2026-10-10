<!-- The local kanban (FounderOS v1 components/TaskBoard.tsx): To do / In
     progress / In review / Done over founderos_agent_tasks. Drag a card or click
     ▸ to advance; the move PATCHes /pages/agents/work optimistically and a 6s
     poll reconciles with the server. -->
<script lang="ts">
	import { User } from '$lib/founderos/icons';
	import { founderosFetch, isGuardRefusal } from '$lib/founderos/api';
	import { SlabCard } from '$lib/founderos/kit';
	import { ADVANCE, COLUMNS, moveTask } from './tasks';
	import type { AgentTask, TaskStatus } from './types';

	let { initialTasks, agentNames, i = 8 }: { initialTasks: AgentTask[]; agentNames: Record<string, string>; i?: number } = $props();

	// svelte-ignore state_referenced_locally
	let tasks = $state<AgentTask[]>([...initialTasks]);
	let dragId = $state<string | null>(null);
	let overCol = $state<TaskStatus | null>(null);
	let pending = 0;
	// Bumped on every move: a poll that started before a move drops its answer.
	let moveGen = 0;
	let moveError = $state<string | null>(null);

	$effect(() => {
		const id = setInterval(async () => {
			if (pending > 0) return;
			const gen = moveGen;
			try {
				const body = await founderosFetch<{ tasks?: AgentTask[] }>('/pages/agents/work');
				if (gen === moveGen && pending === 0 && Array.isArray(body?.tasks)) tasks = body.tasks;
			} catch {
				/* keep the last good board */
			}
		}, 6000);
		return () => clearInterval(id);
	});

	async function move(id: string, status: TaskStatus) {
		const cur = tasks.find((t) => t.id === id);
		if (!cur || cur.status === status) return;
		const from = cur.status;
		tasks = moveTask(tasks, id, status);
		moveGen += 1;
		moveError = null;
		pending += 1;
		try {
			await founderosFetch('/pages/agents/work', { method: 'PATCH', json: { kind: 'task', id, status } });
		} catch (err) {
			// Snap back now and say why, instead of a silent revert on the next poll.
			tasks = moveTask(tasks, id, from);
			const body = (err as { body?: { error?: string } })?.body;
			moveError = isGuardRefusal(err)
				? `“${cur.title}” not moved · writes are off`
				: `“${cur.title}” not moved · ${body?.error ?? (err instanceof Error ? err.message : String(err))}`;
		} finally {
			pending -= 1;
		}
	}

	const labelOf = (s: TaskStatus) => COLUMNS.find((c) => c.status === s)?.label ?? s;
</script>

<SlabCard {i} class="mt-6" title="Local kanban" sub={String(tasks.length)}>
	{#snippet action()}
		<span class="bn-dim font-mono text-[11px]">drag between lanes · or click ▸ to advance</span>
	{/snippet}
	{#if moveError}<p role="alert" class="px-6 pt-3 font-mono text-[11px]" style:color="var(--bn-err)">{moveError}</p>{/if}
	<div class="grid gap-4 px-6 pb-6 pt-4 md:grid-cols-2 xl:grid-cols-4">
		{#each COLUMNS as col (col.status)}
			{@const colTasks = tasks.filter((t) => t.status === col.status)}
			<div
				role="list"
				data-lane={col.status}
				ondragover={(e) => {
					e.preventDefault();
					overCol = col.status;
				}}
				ondragleave={() => (overCol = overCol === col.status ? null : overCol)}
				ondrop={(e) => {
					e.preventDefault();
					overCol = null;
					if (dragId) void move(dragId, col.status);
					dragId = null;
				}}
				class="bn-state-fade flex min-h-[260px] flex-col gap-2.5 rounded-[10px] border p-3"
				style="border-color: {overCol === col.status ? 'var(--bn-accent)' : 'var(--bn-border)'}; background: {overCol === col.status
					? 'var(--bn-surface-2)'
					: 'var(--bn-bg)'}"
			>
				<div class="mb-1 flex items-center justify-between">
					<span class="flex items-center gap-2">
						<span class="h-2 w-2 rounded-full" style="background: {col.tone}"></span>
						<span class="bn-text text-[13px] font-semibold">{col.label}</span>
					</span>
					<span class="rounded-full px-2.5 py-0.5 font-mono text-[10.5px] tabular-nums" style="color: {col.tone}; background: color-mix(in oklab, {col.tone} 14%, transparent)"
						>{colTasks.length}</span
					>
				</div>
				{#each colTasks as task (task.id)}
					{@const next = ADVANCE[task.status]}
					<div
						role="listitem"
						draggable="true"
						ondragstart={() => (dragId = task.id)}
						ondragend={() => {
							dragId = null;
							overCol = null;
						}}
						class="cursor-grab rounded-[10px] border p-3 transition-opacity active:cursor-grabbing {dragId === task.id ? 'opacity-40' : ''}"
						style="border-color: var(--bn-border); background: var(--bn-surface)"
					>
						<div class="text-[12.5px] font-medium leading-snug {task.status === 'done' ? 'bn-dim line-through' : 'bn-text'}">{task.title}</div>
						<div class="bn-dim mt-2 flex items-center gap-1.5 font-mono text-[10px]">
							<User class="h-3 w-3" />
							<span class="min-w-0 flex-1 truncate">{agentNames[task.agentId] ?? task.agentId}</span>
							{#if next}
								<button
									type="button"
									onclick={() => void move(task.id, next)}
									title={`Advance to ${labelOf(next)}`}
									data-lens="c"
									class="bn-pressable bn-dim bn-advance shrink-0 rounded-full border px-2 py-0.5 text-[10px]"
									style="border-color: var(--bn-border)">▸</button
								>
							{/if}
						</div>
					</div>
				{/each}
				{#if colTasks.length === 0}
					<div class="bn-dim rounded-[10px] border border-dashed px-3 py-6 text-center font-mono text-[10.5px]" style="border-color: var(--bn-border)">drop here</div>
				{/if}
			</div>
		{/each}
	</div>
</SlabCard>

<style>
	.bn-advance:hover {
		color: var(--bn-text);
	}
</style>
