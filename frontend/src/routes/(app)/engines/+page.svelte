<script lang="ts">
	import { onMount } from 'svelte';
	import { currentWorkspace } from '$lib/stores/workspaces';
	import {
		getWorkflows,
		createWorkflow,
		updateWorkflow,
		deleteWorkflow,
		runWorkflow,
		getRuns,
		type Workflow,
		type WorkflowInput,
		type StepType,
		type WorkflowTrigger,
		type EngineRun
	} from '$lib/api/engines';
	import {
		Cpu,
		Plus,
		Loader2,
		X,
		Play,
		Trash2,
		Pencil,
		History,
		Sparkles,
		StickyNote,
		Globe,
		GripVertical,
		Clock,
		Zap,
		MousePointerClick
	} from 'lucide-svelte';

	const TRIGGERS: { value: WorkflowTrigger; label: string }[] = [
		{ value: 'manual', label: 'Manual' },
		{ value: 'scheduled', label: 'Scheduled' },
		{ value: 'event', label: 'Event' }
	];
	const STEP_TYPES: { value: StepType; label: string }[] = [
		{ value: 'ai', label: 'AI' },
		{ value: 'note', label: 'Note' },
		{ value: 'http', label: 'HTTP' }
	];

	let workflows = $state<Workflow[]>([]);
	let loading = $state(true);
	let error = $state<string | null>(null);

	// Create/edit modal
	let showEdit = $state(false);
	let saving = $state(false);
	let editing = $state<Workflow | null>(null);
	let form = $state<WorkflowInput>(blankForm());

	// Run output panel
	let runningId = $state<string | null>(null);
	let latestRun = $state<EngineRun | null>(null);
	let latestRunName = $state('');

	// Runs history modal
	let showHistory = $state(false);
	let historyLoading = $state(false);
	let historyRuns = $state<EngineRun[]>([]);
	let historyName = $state('');

	function blankForm(): WorkflowInput {
		return {
			name: '',
			description: '',
			trigger: 'manual',
			status: 'active',
			steps: [{ type: 'ai', label: '', prompt: '', config: '' }]
		};
	}

	let wsId = $state<string | null | undefined>(undefined);
	$effect(() => {
		const id = $currentWorkspace?.id ?? null;
		if (id !== wsId) {
			wsId = id;
			load();
		}
	});

	onMount(load);

	async function load() {
		loading = true;
		error = null;
		try {
			const res = await getWorkflows();
			workflows = res.workflows;
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to load engines';
		} finally {
			loading = false;
		}
	}

	function openCreate() {
		editing = null;
		form = blankForm();
		showEdit = true;
	}
	function openEdit(w: Workflow) {
		editing = w;
		form = {
			name: w.name,
			description: w.description,
			trigger: w.trigger,
			status: w.status,
			steps: w.steps.length
				? w.steps.map((s) => ({ ...s }))
				: [{ type: 'ai', label: '', prompt: '', config: '' }]
		};
		showEdit = true;
	}

	function addStep() {
		form.steps = [...form.steps, { type: 'note', label: '', prompt: '', config: '' }];
	}
	function removeStep(i: number) {
		form.steps = form.steps.filter((_, idx) => idx !== i);
	}

	async function save(e: Event) {
		e.preventDefault();
		if (!form.name.trim()) return;
		saving = true;
		error = null;
		try {
			const payload: WorkflowInput = {
				...form,
				name: form.name.trim(),
				steps: form.steps.filter((s) => s.label.trim() || s.prompt.trim() || s.config.trim())
			};
			if (editing) {
				await updateWorkflow(editing.id, payload);
			} else {
				await createWorkflow(payload);
			}
			showEdit = false;
			editing = null;
			await load();
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to save workflow';
		} finally {
			saving = false;
		}
	}

	async function remove(w: Workflow) {
		if (!confirm(`Delete "${w.name}"? This also removes its run history and can't be undone.`)) return;
		try {
			await deleteWorkflow(w.id);
			workflows = workflows.filter((x) => x.id !== w.id);
			if (latestRun?.workflow_id === w.id) latestRun = null;
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to delete';
		}
	}

	async function run(w: Workflow) {
		runningId = w.id;
		error = null;
		try {
			const res = await runWorkflow(w.id);
			latestRun = res;
			latestRunName = w.name;
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to run workflow';
		} finally {
			runningId = null;
		}
	}

	async function openHistory(w: Workflow) {
		historyName = w.name;
		showHistory = true;
		historyLoading = true;
		historyRuns = [];
		try {
			const res = await getRuns(w.id);
			historyRuns = res.runs;
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to load runs';
		} finally {
			historyLoading = false;
		}
	}

	const triggerLabel = (t: string) => TRIGGERS.find((x) => x.value === t)?.label ?? t;
	function fmtDate(s: string): string {
		try {
			return new Date(s).toLocaleString(undefined, {
				month: 'short',
				day: 'numeric',
				hour: '2-digit',
				minute: '2-digit'
			});
		} catch {
			return s;
		}
	}
</script>

<svelte:head><title>Engines - BusinessOS</title></svelte:head>

<div class="page">
	<header class="page-header">
		<div class="page-icon"><Cpu size={22} strokeWidth={1.8} /></div>
		<div class="head-text">
			<h1 class="page-title">Engines</h1>
			<p class="page-desc">Sequenced and multi-agent workflows that automate work across your workspace. Define steps, then run them.</p>
		</div>
		<div class="head-actions">
			<button class="btn btn-primary" onclick={openCreate}><Plus size={15} /> New workflow</button>
		</div>
	</header>

	{#if error}
		<div class="error-bar">{error}</div>
	{/if}

	{#if latestRun}
		<div class="run-panel">
			<div class="run-head">
				<div class="run-title">
					<span class="run-badge {latestRun.status}">{latestRun.status}</span>
					<strong>{latestRunName}</strong>
					<span class="run-time">{fmtDate(latestRun.created_at)}</span>
				</div>
				<button class="icon-btn" title="Dismiss" onclick={() => (latestRun = null)}><X size={14} /></button>
			</div>
			<pre class="run-output">{latestRun.output}</pre>
		</div>
	{/if}

	{#if loading}
		<div class="loading"><Loader2 size={22} class="spin" /> Loading engines…</div>
	{:else if workflows.length === 0}
		<div class="empty-state">
			<Cpu size={40} strokeWidth={1.4} class="empty-icon" />
			<p class="empty-title">No workflows yet</p>
			<p class="empty-body">Build an internal engine: a sequence of AI, note, and HTTP steps you can run on demand to automate repeatable work.</p>
			<div class="empty-actions">
				<button class="btn btn-primary" onclick={openCreate}><Plus size={15} /> Create a workflow</button>
			</div>
		</div>
	{:else}
		<div class="grid">
			{#each workflows as w (w.id)}
				<div class="card">
					<div class="card-top">
						<div class="wf-icon">
							{#if w.trigger === 'scheduled'}<Clock size={16} />{:else if w.trigger === 'event'}<Zap size={16} />{:else}<MousePointerClick size={16} />{/if}
						</div>
						<div class="card-heading">
							<h3 class="card-name" title={w.name}>{w.name}</h3>
							{#if w.description}<p class="card-desc" title={w.description}>{w.description}</p>{/if}
						</div>
						<div class="card-actions">
							<button class="icon-btn" title="Edit" onclick={() => openEdit(w)}><Pencil size={13} /></button>
							<button class="icon-btn danger" title="Delete" onclick={() => remove(w)}><Trash2 size={13} /></button>
						</div>
					</div>
					<div class="card-meta">
						<span class="meta-badge">{w.steps.length} step{w.steps.length === 1 ? '' : 's'}</span>
						<span class="meta-badge">{triggerLabel(w.trigger)}</span>
						{#if w.status !== 'active'}<span class="meta-badge muted">{w.status}</span>{/if}
					</div>
					<div class="card-foot">
						<button class="btn btn-primary btn-sm" onclick={() => run(w)} disabled={runningId === w.id}>
							{#if runningId === w.id}<Loader2 size={14} class="spin" />{:else}<Play size={14} />{/if}
							Run
						</button>
						<button class="btn btn-ghost btn-sm" onclick={() => openHistory(w)}><History size={14} /> Runs</button>
					</div>
				</div>
			{/each}
		</div>
	{/if}
</div>

{#if showEdit}
	<div class="modal-overlay" role="button" tabindex="0" onclick={(e) => { if (e.target === e.currentTarget) showEdit = false; }} onkeydown={(e) => e.key === 'Escape' && (showEdit = false)}>
		<form class="modal modal-lg" onsubmit={save}>
			<div class="modal-head">
				<h2>{editing ? 'Edit workflow' : 'New workflow'}</h2>
				<button type="button" class="icon-btn" onclick={() => (showEdit = false)}><X size={16} /></button>
			</div>

			<label class="field">
				<span>Name</span>
				<input bind:value={form.name} placeholder="Weekly client digest" required />
			</label>
			<label class="field">
				<span>Description</span>
				<input bind:value={form.description} placeholder="What this workflow does" />
			</label>
			<label class="field">
				<span>Trigger</span>
				<select bind:value={form.trigger}>
					{#each TRIGGERS as t}<option value={t.value}>{t.label}</option>{/each}
				</select>
			</label>

			<div class="steps">
				<div class="steps-head">
					<span>Steps</span>
					<button type="button" class="btn btn-ghost btn-sm" onclick={addStep}><Plus size={13} /> Add step</button>
				</div>
				{#if form.steps.length === 0}
					<p class="steps-empty">No steps yet. Add one to define what this workflow does.</p>
				{/if}
				{#each form.steps as step, i (i)}
					<div class="step">
						<div class="step-top">
							<span class="step-drag"><GripVertical size={14} /></span>
							<span class="step-num">{i + 1}</span>
							<select class="step-type" bind:value={step.type}>
								{#each STEP_TYPES as st}<option value={st.value}>{st.label}</option>{/each}
							</select>
							<input class="step-label" bind:value={step.label} placeholder="Step label" />
							<button type="button" class="icon-btn danger" title="Remove step" onclick={() => removeStep(i)}><X size={13} /></button>
						</div>
						{#if step.type === 'ai'}
							<div class="step-body">
								<span class="step-icon"><Sparkles size={13} /></span>
								<textarea bind:value={step.prompt} rows="3" placeholder="Prompt for the AI to run…"></textarea>
							</div>
						{:else if step.type === 'http'}
							<div class="step-body">
								<span class="step-icon"><Globe size={13} /></span>
								<textarea bind:value={step.config} rows="2" placeholder="HTTP config (URL / method / notes)…"></textarea>
							</div>
						{:else}
							<div class="step-body">
								<span class="step-icon"><StickyNote size={13} /></span>
								<textarea bind:value={step.config} rows="2" placeholder="Note text to record in the run…"></textarea>
							</div>
						{/if}
					</div>
				{/each}
			</div>

			<div class="modal-foot">
				<button type="button" class="btn btn-ghost" onclick={() => (showEdit = false)}>Cancel</button>
				<button type="submit" class="btn btn-primary" disabled={saving}>
					{#if saving}<Loader2 size={15} class="spin" />{:else}<Plus size={15} />{/if}
					{editing ? 'Save changes' : 'Create workflow'}
				</button>
			</div>
		</form>
	</div>
{/if}

{#if showHistory}
	<div class="modal-overlay" role="button" tabindex="0" onclick={(e) => { if (e.target === e.currentTarget) showHistory = false; }} onkeydown={(e) => e.key === 'Escape' && (showHistory = false)}>
		<div class="modal modal-lg">
			<div class="modal-head">
				<h2>Run history — {historyName}</h2>
				<button type="button" class="icon-btn" onclick={() => (showHistory = false)}><X size={16} /></button>
			</div>
			{#if historyLoading}
				<div class="loading"><Loader2 size={20} class="spin" /> Loading runs…</div>
			{:else if historyRuns.length === 0}
				<p class="steps-empty">No runs yet. Run this workflow to see its history here.</p>
			{:else}
				<div class="runs-list">
					{#each historyRuns as r (r.id)}
						<div class="run-item">
							<div class="run-item-head">
								<span class="run-badge {r.status}">{r.status}</span>
								<span class="run-time">{fmtDate(r.created_at)}</span>
							</div>
							<pre class="run-output">{r.output}</pre>
						</div>
					{/each}
				</div>
			{/if}
			<div class="modal-foot">
				<button type="button" class="btn btn-ghost" onclick={() => (showHistory = false)}>Close</button>
			</div>
		</div>
	</div>
{/if}

<style>
	.page { display: flex; flex-direction: column; gap: 20px; padding: 28px 32px; height: 100%; overflow-y: auto; }
	.page-header { display: flex; align-items: flex-start; gap: 14px; }
	.page-icon { width: 42px; height: 42px; border-radius: 10px; border: 1px solid var(--dbd); background: color-mix(in srgb, var(--dt) 5%, transparent); color: var(--dt2); display: flex; align-items: center; justify-content: center; flex-shrink: 0; }
	.head-text { flex: 1; min-width: 0; }
	.page-title { font-size: 1.25rem; font-weight: 650; color: var(--dt); letter-spacing: -0.02em; margin: 0; }
	.page-desc { font-size: 0.875rem; color: var(--dt3); margin: 2px 0 0; }
	.head-actions { display: flex; gap: 8px; flex-shrink: 0; }

	.btn { display: inline-flex; align-items: center; gap: 6px; padding: 8px 14px; border-radius: 8px; font-size: 0.84rem; font-weight: 550; cursor: pointer; border: 1px solid transparent; transition: background 0.12s, border-color 0.12s; }
	.btn-primary { background: var(--bos-v2-button-primary); color: var(--bos-v2-button-pureWhiteText); }
	.btn-primary:hover { opacity: 0.9; }
	.btn-ghost { background: transparent; border-color: var(--dbd); color: var(--dt2); }
	.btn-ghost:hover { background: color-mix(in srgb, var(--dt) 5%, transparent); }
	.btn-sm { padding: 6px 10px; font-size: 0.8rem; }
	.btn:disabled { opacity: 0.6; cursor: default; }

	.error-bar { padding: 10px 14px; border-radius: 8px; background: color-mix(in srgb, #ef4444 12%, transparent); color: #ef4444; font-size: 0.83rem; }
	.loading { display: flex; align-items: center; gap: 8px; padding: 48px; justify-content: center; color: var(--dt3); font-size: 0.9rem; }

	.grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(280px, 1fr)); gap: 16px; }
	.card { border: 1px solid var(--dbd); border-radius: 12px; padding: 14px; background: color-mix(in srgb, var(--dt) 2%, transparent); display: flex; flex-direction: column; gap: 12px; transition: border-color 0.12s; }
	.card:hover { border-color: color-mix(in srgb, var(--dt) 22%, transparent); }
	.card-top { display: flex; align-items: flex-start; gap: 10px; }
	.wf-icon { width: 30px; height: 30px; border-radius: 8px; border: 1px solid var(--dbd); background: color-mix(in srgb, var(--dt) 5%, transparent); color: var(--dt2); display: flex; align-items: center; justify-content: center; flex-shrink: 0; }
	.card-heading { flex: 1; min-width: 0; }
	.card-name { font-size: 0.92rem; font-weight: 600; color: var(--dt); margin: 0; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
	.card-desc { font-size: 0.8rem; color: var(--dt3); margin: 2px 0 0; display: -webkit-box; -webkit-line-clamp: 2; line-clamp: 2; -webkit-box-orient: vertical; overflow: hidden; }
	.card-actions { display: flex; gap: 4px; flex-shrink: 0; }
	.card-meta { display: flex; flex-wrap: wrap; gap: 6px; }
	.meta-badge { font-size: 0.68rem; text-transform: uppercase; letter-spacing: 0.04em; padding: 2px 7px; border-radius: 4px; background: color-mix(in srgb, var(--dt) 8%, transparent); color: var(--dt2); }
	.meta-badge.muted { background: transparent; border: 1px solid var(--dbd); color: var(--dt3); }
	.card-foot { display: flex; gap: 8px; margin-top: 2px; }

	.icon-btn { width: 26px; height: 26px; border-radius: 6px; border: 1px solid var(--dbd); background: var(--dbg); color: var(--dt2); display: flex; align-items: center; justify-content: center; cursor: pointer; flex-shrink: 0; }
	.icon-btn:hover { background: color-mix(in srgb, var(--dt) 8%, transparent); }
	.icon-btn.danger:hover { color: #ef4444; border-color: #ef4444; }

	.run-panel { border: 1px solid var(--dbd); border-radius: 12px; background: color-mix(in srgb, var(--dt) 3%, transparent); display: flex; flex-direction: column; }
	.run-head { display: flex; align-items: center; justify-content: space-between; gap: 10px; padding: 10px 14px; border-bottom: 1px solid var(--dbd); }
	.run-title { display: flex; align-items: center; gap: 8px; font-size: 0.85rem; color: var(--dt); min-width: 0; }
	.run-time { font-size: 0.72rem; color: var(--dt3); }
	.run-badge { font-size: 0.68rem; text-transform: uppercase; letter-spacing: 0.04em; padding: 2px 7px; border-radius: 4px; font-weight: 600; }
	.run-badge.done { background: color-mix(in srgb, #22c55e 16%, transparent); color: #16a34a; }
	.run-badge.error { background: color-mix(in srgb, #ef4444 16%, transparent); color: #ef4444; }
	.run-badge.running { background: color-mix(in srgb, var(--dt) 12%, transparent); color: var(--dt2); }
	.run-output { margin: 0; padding: 14px; font-size: 0.8rem; line-height: 1.5; color: var(--dt2); white-space: pre-wrap; word-break: break-word; font-family: ui-monospace, SFMono-Regular, Menlo, monospace; max-height: 340px; overflow-y: auto; }

	.empty-state { flex: 1; display: flex; flex-direction: column; align-items: center; justify-content: center; gap: 10px; padding: 64px 24px; text-align: center; }
	.empty-state :global(.empty-icon) { color: var(--dt4, var(--dt3)); opacity: 0.5; }
	.empty-title { font-size: 0.95rem; font-weight: 600; color: var(--dt2); margin: 4px 0 0; }
	.empty-body { font-size: 0.84rem; color: var(--dt3); max-width: 380px; margin: 0; }
	.empty-actions { display: flex; gap: 8px; margin-top: 8px; }

	.modal-overlay { position: fixed; inset: 0; background: rgba(0,0,0,0.4); display: flex; align-items: center; justify-content: center; z-index: 100; padding: 20px; }
	.modal { width: 100%; max-width: 440px; background: var(--dbg); border: 1px solid var(--dbd); border-radius: 14px; padding: 20px; display: flex; flex-direction: column; gap: 14px; max-height: 90vh; overflow-y: auto; }
	.modal-lg { max-width: 620px; }
	.modal-head { display: flex; align-items: center; justify-content: space-between; }
	.modal-head h2 { font-size: 1rem; font-weight: 650; color: var(--dt); margin: 0; }
	.field { display: flex; flex-direction: column; gap: 5px; flex: 1; }
	.field span { font-size: 0.78rem; color: var(--dt2); font-weight: 550; }
	.field input, .field select { padding: 8px 10px; border-radius: 8px; border: 1px solid var(--dbd); background: transparent; color: var(--dt); font-size: 0.85rem; }
	.modal-foot { display: flex; justify-content: flex-end; gap: 8px; margin-top: 4px; }

	.steps { display: flex; flex-direction: column; gap: 10px; border-top: 1px solid var(--dbd); padding-top: 14px; }
	.steps-head { display: flex; align-items: center; justify-content: space-between; }
	.steps-head span { font-size: 0.78rem; color: var(--dt2); font-weight: 550; }
	.steps-empty { font-size: 0.82rem; color: var(--dt3); margin: 0; }
	.step { border: 1px solid var(--dbd); border-radius: 10px; padding: 10px; display: flex; flex-direction: column; gap: 8px; background: color-mix(in srgb, var(--dt) 2%, transparent); }
	.step-top { display: flex; align-items: center; gap: 8px; }
	.step-drag { color: var(--dt3); display: flex; }
	.step-num { width: 20px; height: 20px; border-radius: 5px; background: color-mix(in srgb, var(--dt) 8%, transparent); color: var(--dt2); font-size: 0.72rem; font-weight: 600; display: flex; align-items: center; justify-content: center; flex-shrink: 0; }
	.step-type { padding: 6px 8px; border-radius: 7px; border: 1px solid var(--dbd); background: transparent; color: var(--dt); font-size: 0.8rem; flex-shrink: 0; }
	.step-label { flex: 1; min-width: 0; padding: 6px 8px; border-radius: 7px; border: 1px solid var(--dbd); background: transparent; color: var(--dt); font-size: 0.82rem; }
	.step-body { display: flex; gap: 8px; align-items: flex-start; }
	.step-icon { color: var(--dt3); display: flex; padding-top: 8px; flex-shrink: 0; }
	.step-body textarea { flex: 1; padding: 8px 10px; border-radius: 8px; border: 1px solid var(--dbd); background: transparent; color: var(--dt); font-size: 0.82rem; line-height: 1.4; resize: vertical; font-family: inherit; }

	.runs-list { display: flex; flex-direction: column; gap: 12px; }
	.run-item { border: 1px solid var(--dbd); border-radius: 10px; overflow: hidden; }
	.run-item-head { display: flex; align-items: center; gap: 8px; padding: 8px 12px; border-bottom: 1px solid var(--dbd); background: color-mix(in srgb, var(--dt) 3%, transparent); }
	.run-item .run-output { max-height: 240px; }

	:global(.spin) { animation: spin 0.8s linear infinite; }
	@keyframes spin { to { transform: rotate(360deg); } }

	@media (max-width: 768px) { .page { padding: 16px 18px; } .head-actions { flex-wrap: wrap; } }
	@media (max-width: 480px) { .page { padding: 12px 14px; } }
</style>
