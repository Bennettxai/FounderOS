<!-- The board queue (FounderOS v1 components/BoardTasks.tsx), in the Tasks tab
     on /os/agents: the board's OPEN issues plus a composer that creates a REAL
     issue the Conductor routes.
     Creating is a guarded board write; with FOUNDEROS_WRITES off the refusal is
     shown, never a fake success. An unreachable board says so. -->
<script lang="ts">
	import { ArrowUpRight, LoaderCircle, Plus } from '$lib/founderos/icons';
	import { founderosFetch, isGuardRefusal } from '$lib/founderos/api';
	import { SlabCard } from '$lib/founderos/kit';
	import { chipClass, PILL } from '../agents/ui';
	import { age, issueBody, issueTone, ROUTES, type Route } from './tasks';
	import type { BoardIssue } from './types';

	let { initialIssues, boardUrl, i = 7 }: { initialIssues: BoardIssue[]; boardUrl: string | null; i?: number } = $props();

	// svelte-ignore state_referenced_locally
	let issues = $state<BoardIssue[]>([...initialIssues]);
	let title = $state('');
	let route = $state<Route>('Conductor routes');
	let sending = $state(false);
	let error = $state<string | null>(null);

	function refusal(err: unknown): string {
		if (isGuardRefusal(err)) {
			return 'Writes are off on the bridge (FOUNDEROS_WRITES=0): nothing was sent to the board.';
		}
		return `Board rejected it: ${err instanceof Error ? err.message : String(err)}`;
	}

	async function create() {
		const t = title.trim();
		if (!t || sending) return;
		sending = true;
		error = null;
		try {
			const res = await founderosFetch<{ issue?: BoardIssue }>('/pages/board/tasks', { method: 'POST', json: issueBody(t, route) });
			if (!res?.issue) throw new Error('the board returned no issue');
			issues = [res.issue, ...issues];
			title = '';
		} catch (err) {
			error = refusal(err);
		} finally {
			sending = false;
		}
	}
</script>

<SlabCard {i} class="mt-6" title="Board queue" sub={`${issues.length} open issues`}>
	{#snippet action()}
		{#if boardUrl}
			<a href={boardUrl} target="_blank" rel="noreferrer" class={PILL}>
				open board <ArrowUpRight class="h-3.5 w-3.5" />
			</a>
		{/if}
	{/snippet}
	<div class="px-6 pt-4">
		<div class="mb-3 flex items-center gap-2">
			<input
				bind:value={title}
				onkeydown={(e) => e.key === 'Enter' && create()}
				placeholder="Hand the company work · one line · the Conductor routes it"
				class="bn-text min-w-0 flex-1 rounded-full border px-4 py-2 text-[13px] focus:outline-none"
				style="background: var(--bn-bg); border-color: var(--bn-border)"
			/>
			<button
				type="button"
				onclick={create}
				disabled={sending || !title.trim()}
				class="flex shrink-0 items-center gap-1.5 rounded-full px-4 py-2 text-[13px] font-semibold disabled:opacity-40"
				style="background: var(--bn-accent); color: var(--bn-accent-ink)"
			>
				{#if sending}<LoaderCircle class="h-3.5 w-3.5 animate-spin" />{:else}<Plus class="h-3.5 w-3.5" />{/if}
				Create issue
			</button>
		</div>
		<div class="mb-4 flex flex-wrap items-center gap-2">
			{#each ROUTES as r (r)}
				<button
					type="button"
					onclick={() => (route = r)}
					aria-pressed={route === r}
					class={chipClass(route === r)}>{r}</button
				>
			{/each}
		</div>
		{#if error}<div class="mb-3 text-[12px]" style="color: var(--bn-err)">{error}</div>{/if}
	</div>

	{#if issues.length === 0}
		<div class="bn-dim border-t px-6 py-8 text-center text-[12.5px]" style="border-color: var(--bn-border)">Board unreachable or empty · the live queue shows here.</div>
	{:else}
		<ul class="border-t" style="border-color: var(--bn-border)">
			{#each issues.slice(0, 10) as issue (issue.id)}
				<li
					class="grid grid-cols-[1fr_auto_auto] items-center gap-4 border-b px-6 py-3.5 last:border-0 max-[700px]:grid-cols-[1fr_auto]"
					style="border-color: var(--bn-border)"
				>
					<div class="min-w-0">
						<div class="bn-text truncate text-[13.5px] font-medium">{issue.title}</div>
						<div class="bn-dim truncate font-mono text-[11px]">
							{issue.identifier}{issue.assigneeName ? ` · ${issue.assigneeName}` : ''}
						</div>
					</div>
					<span
						class="rounded-full px-2.5 py-0.5 font-mono text-[10.5px] uppercase tracking-[0.12em]"
						style="color: {issueTone(issue.status)}; background: color-mix(in oklab, {issueTone(issue.status)} 14%, transparent)"
						>{issue.status.replace(/_/g, ' ')}</span
					>
					<span class="bn-dim w-[40px] text-right font-mono text-[11px] max-[700px]:hidden">{age(issue.updatedAt)}</span>
				</li>
			{/each}
		</ul>
	{/if}
</SlabCard>
