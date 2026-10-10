<!-- Scheduled jobs on /tasks (FounderOS v1 components/TaskCronStrip.tsx): a
     compact card per cron with a status dot, cadence + next fire and its real
     ok/runs count off founderos_cron_runs. A job that never ran reads zero. -->
<script lang="ts">
	import { TriangleAlert } from '$lib/founderos/icons';
	import { Dot, SlabCard } from '$lib/founderos/kit';
	import { PILL } from '../agents/ui';
	import { ago, inNext, jobFailing } from './tasks';
	import type { JobRow } from './types';

	let { jobs, i = 6 }: { jobs: JobRow[]; i?: number } = $props();
	const enabled = $derived(jobs.filter((r) => r.enabled).length);
</script>

{#if jobs.length > 0}
	<SlabCard {i} class="mt-6" title="Scheduled jobs" sub={`${enabled} of ${jobs.length} enabled`}>
		{#snippet action()}
			<a href="/os/workflows" class={PILL}>full panel →</a>
		{/snippet}
		<div class="grid gap-3 px-6 pb-6 pt-4 sm:grid-cols-2 xl:grid-cols-3">
			{#each jobs as r (r.id)}
				{@const failing = jobFailing(r)}
				{@const next = r.enabled ? inNext(r.nextRunAt) : null}
				<div
					data-job={r.id}
					class="rounded-[10px] border px-4 py-3.5 {r.enabled ? '' : 'opacity-60'}"
					style="background: var(--bn-bg); border-color: {failing ? 'color-mix(in oklab, var(--bn-err) 50%, transparent)' : 'var(--bn-border)'}"
				>
					<div class="flex items-center gap-2">
						<Dot state={!r.enabled ? 'off' : failing ? 'error' : 'connected'} />
						<span class="bn-text min-w-0 flex-1 truncate text-[13.5px] font-medium">{r.description}</span>
						<span
							class="shrink-0 rounded-full px-2.5 py-0.5 font-mono text-[10.5px] tabular-nums"
							style="color: {failing ? 'var(--bn-err)' : 'var(--bn-text-2)'}; background: color-mix(in oklab, {failing ? 'var(--bn-err) 14%' : 'var(--bn-text) 6%'}, transparent)"
						>{r.ok}/{r.runs}</span>
					</div>
					<div class="bn-dim mt-2 flex items-center gap-2 font-mono text-[11px]">
						<span class="min-w-0 truncate">{r.scheduleLabel}</span>
						{#if next}<span class="shrink-0">· next {next}</span>{/if}
						{#if !r.enabled}<span class="shrink-0">· paused</span>{/if}
					</div>
					<div class="mt-1 flex items-center gap-1.5 font-mono text-[11px]">
						<span class="bn-dim min-w-0 truncate">{r.agentName}</span>
						<span class="ml-auto shrink-0" style="color: {r.lastOk === false ? 'var(--bn-err)' : 'var(--bn-text-3)'}">
							{r.lastOk === false ? `last failed · ${ago(r.lastRunAt)}` : ago(r.lastRunAt)}
						</span>
					</div>
					{#if r.overdue || r.unknownAgent}
						<div class="mt-1.5 flex items-center gap-1 font-mono text-[9.5px]" style="color: var(--bn-err)">
							<TriangleAlert class="h-3 w-3" />
							<span>{r.unknownAgent ? 'agent missing from the runtime' : 'overdue · slot passed with no run'}</span>
						</div>
					{/if}
				</div>
			{/each}
		</div>
	</SlabCard>
{/if}
