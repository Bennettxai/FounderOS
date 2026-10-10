<!-- Scheduled tasks: the functional half of /workflows (FounderOS v1
     components/ScheduledTasks.tsx). Add a task, run it now, pause it, delete
     it, in place. Cron CRUD goes through /pages/agents/work; "run now" is
     /pages/cron/run, recorded exactly like a tick. -->
<script lang="ts">
	import { Check, Clock, Plus, Trash2, X } from '$lib/founderos/icons';
	import { founderosFetch } from '$lib/founderos/api';
	import Dot from '$lib/founderos/kit/Dot.svelte';
	import Label from '$lib/founderos/kit/Label.svelte';
	import type { JobRow, SimpleAgent } from './types';
	import { ago, CRON_PRESETS, jobState, untilLabel } from './workflows';

	let { jobs, agents, onchange }: { jobs: JobRow[]; agents: SimpleAgent[]; onchange?: () => void } = $props();

	let adding = $state(false);
	let busy = $state<string | null>(null);
	let error = $state<string | null>(null);
	let ranMsg = $state<string | null>(null);
	let agentId = $state('');
	let schedule = $state('0 9 * * *');
	let description = $state('');

	$effect(() => {
		if (!agentId && agents.length) agentId = agents[0].id;
	});

	const late = $derived(jobs.filter((j) => j.overdue).length);
	const countdown = $derived(
		untilLabel(
			jobs
				.filter((j) => j.enabled && j.nextRunAt)
				.map((j) => j.nextRunAt as string)
				.sort()[0] ?? null
		)
	);

	async function call<T>(path: string, method: string, json: unknown, label: string): Promise<T | null> {
		busy = label;
		error = null;
		try {
			return await founderosFetch<T>(path, { method, json });
		} catch (err) {
			error = err instanceof Error ? err.message : String(err);
			return null;
		} finally {
			busy = null;
		}
	}

	async function add() {
		if (!agentId || !description.trim()) {
			error = 'pick an agent and describe the task';
			return;
		}
		const ok = await call('/pages/agents/work', 'POST', { kind: 'cron', agentId, schedule, description: description.trim() }, 'add');
		if (ok) {
			description = '';
			adding = false;
			onchange?.();
		}
	}

	async function runNow(id: string) {
		const body = await call<{ ok: boolean; summary: string }>('/pages/cron/run', 'POST', { cronId: id }, id);
		if (body?.summary) ranMsg = body.summary.slice(0, 220);
		onchange?.();
	}

	async function toggle(id: string, enabled: boolean) {
		await call('/pages/agents/work', 'PATCH', { kind: 'cron', id, enabled }, id);
		onchange?.();
	}

	async function remove(id: string) {
		await call('/pages/agents/work', 'DELETE', { kind: 'cron', id }, id);
		onchange?.();
	}

	const ctl = 'bn-pressable bn-border rounded-[5px] border px-2 py-0.5 font-mono text-[9.5px] disabled:opacity-40';
</script>

<section data-part="scheduled" class="mb-6">
	<div class="mb-3 flex flex-wrap items-baseline gap-x-3 gap-y-1">
		<div class="min-w-0 flex-1"><Label count={jobs.length} rule>Scheduled tasks</Label></div>
		{#if countdown}<span class="bn-muted flex shrink-0 items-center gap-1.5 font-mono text-[10.5px]"><Clock size={12} class="bn-dim" /> next fires in {countdown}</span>{/if}
		{#if late > 0}<span class="shrink-0 font-mono text-[11px]" style="color: var(--bn-warn)">{late} overdue</span>{/if}
		<button type="button" data-part="new-task" class="bn-pressable bn-border bn-muted flex shrink-0 items-center gap-1 rounded-[5px] border px-2 py-1 font-mono text-[10px] uppercase tracking-[0.12em]" onclick={() => (adding = !adding)}>
			{#if adding}<X size={12} />{:else}<Plus size={12} />{/if}
			{adding ? 'cancel' : 'new task'}
		</button>
	</div>

	{#if adding}
		<div data-part="add-form" class="bn-surface mb-3 rounded-lg border p-3" style="border-color: var(--bn-border-strong)">
			<div class="grid gap-2 sm:grid-cols-[minmax(0,1fr)_11rem]">
				<input bind:value={description} placeholder="What should run? e.g. Sweep unpaid invoices" class="bn-text rounded-[5px] border px-2.5 py-1.5 text-[12px] outline-none" style="border-color: var(--bn-border); background: var(--bn-bg)" />
				<select bind:value={agentId} aria-label="Agent" class="bn-muted rounded-[5px] border px-2 py-1.5 font-mono text-[11px] outline-none" style="border-color: var(--bn-border); background: var(--bn-bg)">
					{#each agents as a (a.id)}<option value={a.id}>{a.name}</option>{/each}
				</select>
			</div>
			<div class="mt-2 flex flex-wrap items-center gap-1.5">
				{#each CRON_PRESETS as p (p.expr)}
					<button type="button" class="bn-pressable rounded-[5px] border px-2 py-0.5 font-mono text-[9.5px] {schedule === p.expr ? 'bn-text' : 'bn-dim'}" style="border-color: {schedule === p.expr ? 'var(--bn-accent)' : 'var(--bn-border)'}" onclick={() => (schedule = p.expr)}>{p.label}</button>
				{/each}
				<input bind:value={schedule} spellcheck="false" aria-label="Cron schedule" class="bn-muted ml-auto w-32 rounded-[5px] border px-2 py-1 text-right font-mono text-[10.5px] outline-none" style="border-color: var(--bn-border); background: var(--bn-bg)" />
				<button type="button" class="bn-pressable bn-text flex items-center gap-1 rounded-[5px] border px-2.5 py-1 font-mono text-[10px] uppercase tracking-[0.12em] disabled:opacity-40" style="border-color: var(--bn-accent)" disabled={busy === 'add'} onclick={add}><Check size={12} /> {busy === 'add' ? 'adding' : 'add'}</button>
			</div>
			<p class="bn-dim mt-2 font-mono text-[9.5px]">Runs in the backend's local time. The runner ticks every minute (only when FOUNDEROS_CRONS=1) and catches up a missed slot.</p>
		</div>
	{/if}

	{#if error}<p data-part="error" class="mb-2 font-mono text-[10.5px]" style="color: var(--bn-err)">⚠ {error}</p>{/if}
	{#if ranMsg}<p class="bn-muted mb-2 font-mono text-[10.5px]">ran: {ranMsg}</p>{/if}

	<div data-part="job-list" class="bn-surface rounded-lg border" style="border-color: var(--bn-border)">
		{#if jobs.length === 0}
			<p class="bn-dim px-3 py-4 font-mono text-[11px]">No scheduled tasks yet. Hit <span class="bn-muted">new task</span> to add one.</p>
		{:else}
			{#each jobs as job (job.id)}
				<div data-job={job.id} class="grid grid-cols-[auto_minmax(0,1fr)_auto] items-center gap-x-3 gap-y-1 border-b px-3 py-2.5 last:border-b-0 sm:grid-cols-[auto_minmax(0,1fr)_8.5rem_5rem_7.5rem_5.5rem_auto] {job.enabled ? '' : 'opacity-45'}" style="border-color: var(--bn-border)">
					<Dot state={jobState(job)} />
					<div class="min-w-0">
						<div class="bn-text truncate text-[12px] font-medium">{job.description}</div>
						<div class="bn-dim truncate font-mono text-[10px]">
							{job.agentName}{#if job.unknownAgent}<span style="color: var(--bn-err)">{' · no such agent'}</span>{/if}
						</div>
					</div>
					<div class="bn-muted font-mono text-[10.5px]">
						{job.scheduleLabel}
						<div class="bn-dim text-[9.5px]">{job.schedule}</div>
					</div>
					<div class="bn-muted font-mono text-[10.5px]">
						{untilLabel(job.nextRunAt) ? `in ${untilLabel(job.nextRunAt)}` : job.enabled ? 'due' : 'paused'}
					</div>
					<div class="min-w-0 font-mono text-[10.5px]">
						{#if job.overdue}
							<span style="color: var(--bn-warn)">overdue</span>
						{:else if job.lastOk === false}
							<span class="block truncate" style="color: var(--bn-err)" title={job.lastSummary ?? undefined}>failed · {job.lastSummary ? job.lastSummary.slice(0, 32) : ago(job.lastRunAt)}</span>
						{:else}
							<span class="bn-muted">{ago(job.lastRunAt)}</span>
						{/if}
					</div>
					<div class="justify-self-end">
						{#if job.history.length === 0}
							<span class="bn-dim font-mono text-[10.5px]">·</span>
						{:else}
							<div data-part="history" class="flex items-end gap-[2px]" title="last {job.history.length} runs">
								{#each job.history as ok, i (i)}
									<span class="w-[4px]" style="height: {ok ? 10 : 12}px; background: {ok ? 'color-mix(in oklab, var(--bn-ok) 70%, transparent)' : 'var(--bn-err)'}"></span>
								{/each}
							</div>
						{/if}
						{#if job.runs - job.ok > 0}<div class="text-right font-mono text-[9.5px]" style="color: var(--bn-err)">{job.runs - job.ok} failed</div>{/if}
					</div>
					<div class="flex shrink-0 items-center gap-1">
						<button type="button" class="{ctl} bn-muted whitespace-nowrap" disabled={busy === job.id} onclick={() => runNow(job.id)}>▸ run now</button>
						<button type="button" class="font-mono text-[9px] uppercase tracking-[0.1em] disabled:opacity-40" style="color: {job.enabled ? 'var(--bn-text-3)' : 'var(--bn-warn)'}" title={job.enabled ? 'Pause' : 'Enable'} disabled={busy === job.id} onclick={() => toggle(job.id, !job.enabled)}>{job.enabled ? 'on' : 'off'}</button>
						<button type="button" class="bn-dim disabled:opacity-40" title="Delete" aria-label="Delete {job.description}" disabled={busy === job.id} onclick={() => remove(job.id)}><Trash2 size={12} /></button>
					</div>
				</div>
			{/each}
		{/if}
	</div>
</section>
