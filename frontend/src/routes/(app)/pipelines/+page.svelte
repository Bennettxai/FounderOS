<script lang="ts">
	import { onMount } from 'svelte';
	import { currentWorkspace } from '$lib/stores/workspaces';
	import {
		getPipelines,
		createPipeline,
		updatePipeline,
		deletePipeline,
		getPipelineStages,
		createPipelineStage,
		updatePipelineStage,
		deletePipelineStage,
		type Pipeline,
		type PipelineStage,
		type PipelineType,
		type StageType
	} from '$lib/api/pipelines';
	import {
		GitBranch,
		Plus,
		Loader2,
		X,
		Trash2,
		Pencil,
		ChevronRight,
		Star,
		Layers
	} from 'lucide-svelte';

	const TYPES: { value: PipelineType; label: string }[] = [
		{ value: 'sales', label: 'Sales' },
		{ value: 'projects', label: 'Delivery / Projects' },
		{ value: 'hiring', label: 'Hiring' },
		{ value: 'custom', label: 'Custom' }
	];
	const typeLabel = (t?: string | null) => TYPES.find((x) => x.value === t)?.label ?? 'Custom';

	const STAGE_TYPES: { value: StageType; label: string }[] = [
		{ value: 'open', label: 'Open' },
		{ value: 'won', label: 'Won' },
		{ value: 'lost', label: 'Lost' }
	];

	let pipelines = $state<Pipeline[]>([]);
	let loading = $state(true);
	let error = $state<string | null>(null);

	// Lazily-loaded stages, keyed by pipeline id.
	let expanded = $state<string | null>(null);
	let stagesByPipeline = $state<Record<string, PipelineStage[]>>({});
	let stagesLoading = $state<string | null>(null);

	// Pipeline editor.
	let showEdit = $state(false);
	let saving = $state(false);
	let editing = $state<Pipeline | null>(null);
	let form = $state({
		name: '',
		pipeline_type: 'sales' as PipelineType,
		description: '',
		currency: 'USD',
		color: '',
		is_default: false,
		is_active: true
	});

	// Stage editor.
	let showStage = $state(false);
	let stageSaving = $state(false);
	let stagePipelineId = $state<string | null>(null);
	let editingStage = $state<PipelineStage | null>(null);
	let stageForm = $state({
		name: '',
		description: '',
		stage_type: 'open' as StageType,
		probability: '',
		rotting_days: '',
		color: ''
	});

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
			const res = await getPipelines();
			pipelines = res.pipelines ?? [];
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to load pipelines';
		} finally {
			loading = false;
		}
	}

	async function toggle(p: Pipeline) {
		if (expanded === p.id) {
			expanded = null;
			return;
		}
		expanded = p.id;
		if (!stagesByPipeline[p.id]) await loadStages(p.id);
	}

	async function loadStages(pipelineId: string) {
		stagesLoading = pipelineId;
		try {
			const res = await getPipelineStages(pipelineId);
			const stages = (res.stages ?? []).slice().sort((a, b) => a.position - b.position);
			stagesByPipeline = { ...stagesByPipeline, [pipelineId]: stages };
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to load stages';
		} finally {
			stagesLoading = null;
		}
	}

	// ---- Pipeline editor ----

	function openNew() {
		editing = null;
		form = {
			name: '',
			pipeline_type: 'sales',
			description: '',
			currency: 'USD',
			color: '',
			is_default: false,
			is_active: true
		};
		showEdit = true;
	}

	function openEdit(p: Pipeline) {
		editing = p;
		form = {
			name: p.name,
			pipeline_type: (p.pipeline_type as PipelineType) ?? 'custom',
			description: p.description ?? '',
			currency: p.currency ?? 'USD',
			color: p.color ?? '',
			is_default: p.is_default ?? false,
			is_active: p.is_active ?? true
		};
		showEdit = true;
	}

	async function savePipeline(e: Event) {
		e.preventDefault();
		if (!form.name.trim()) return;
		saving = true;
		error = null;
		try {
			if (editing) {
				await updatePipeline(editing.id, {
					name: form.name.trim(),
					description: form.description.trim(),
					currency: form.currency.trim(),
					color: form.color.trim(),
					is_active: form.is_active
				});
			} else {
				await createPipeline({
					name: form.name.trim(),
					pipeline_type: form.pipeline_type,
					description: form.description.trim(),
					currency: form.currency.trim(),
					color: form.color.trim(),
					is_default: form.is_default
				});
			}
			showEdit = false;
			await load();
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to save pipeline';
		} finally {
			saving = false;
		}
	}

	async function removePipeline(p: Pipeline) {
		if (!confirm(`Delete pipeline "${p.name}"? Its stages will be removed too.`)) return;
		try {
			await deletePipeline(p.id);
			pipelines = pipelines.filter((x) => x.id !== p.id);
			if (expanded === p.id) expanded = null;
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to delete pipeline';
		}
	}

	// ---- Stage editor ----

	function openNewStage(pipelineId: string) {
		stagePipelineId = pipelineId;
		editingStage = null;
		stageForm = {
			name: '',
			description: '',
			stage_type: 'open',
			probability: '',
			rotting_days: '',
			color: ''
		};
		showStage = true;
	}

	function openEditStage(pipelineId: string, s: PipelineStage) {
		stagePipelineId = pipelineId;
		editingStage = s;
		stageForm = {
			name: s.name,
			description: s.description ?? '',
			stage_type: (s.stage_type as StageType) ?? 'open',
			probability: s.probability != null ? String(s.probability) : '',
			rotting_days: s.rotting_days != null ? String(s.rotting_days) : '',
			color: s.color ?? ''
		};
		showStage = true;
	}

	function numOrUndef(s: string): number | undefined {
		const n = parseInt(s.trim(), 10);
		return Number.isFinite(n) ? n : undefined;
	}

	async function saveStage(e: Event) {
		e.preventDefault();
		if (!stagePipelineId || !stageForm.name.trim()) return;
		stageSaving = true;
		error = null;
		try {
			const pid = stagePipelineId;
			if (editingStage) {
				await updatePipelineStage(pid, editingStage.id, {
					name: stageForm.name.trim(),
					description: stageForm.description.trim(),
					probability: numOrUndef(stageForm.probability),
					rotting_days: numOrUndef(stageForm.rotting_days),
					color: stageForm.color.trim()
				});
			} else {
				const existing = stagesByPipeline[pid] ?? [];
				await createPipelineStage(pid, {
					name: stageForm.name.trim(),
					description: stageForm.description.trim(),
					stage_type: stageForm.stage_type,
					probability: numOrUndef(stageForm.probability),
					rotting_days: numOrUndef(stageForm.rotting_days),
					color: stageForm.color.trim(),
					position: existing.length
				});
			}
			showStage = false;
			await loadStages(pid);
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to save stage';
		} finally {
			stageSaving = false;
		}
	}

	async function removeStage(pipelineId: string, s: PipelineStage) {
		if (!confirm(`Delete stage "${s.name}"?`)) return;
		try {
			await deletePipelineStage(pipelineId, s.id);
			stagesByPipeline = {
				...stagesByPipeline,
				[pipelineId]: (stagesByPipeline[pipelineId] ?? []).filter((x) => x.id !== s.id)
			};
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to delete stage';
		}
	}
</script>

<svelte:head><title>Pipelines - BusinessOS</title></svelte:head>

<div class="pl-root">
	<header class="topbar">
		<div class="title-wrap">
			<h1>Pipelines</h1>
			<span class="count">{pipelines.length}</span>
		</div>
		<div class="tools">
			<button class="btn btn--primary" onclick={openNew}>
				<Plus size={16} strokeWidth={2.4} />New pipeline
			</button>
		</div>
	</header>

	{#if error}<div class="banner banner--error">{error}</div>{/if}

	{#if loading}
		<div class="state"><Loader2 class="spin" size={20} /> Loading pipelines…</div>
	{:else if pipelines.length === 0}
		<div class="state empty">
			<GitBranch size={28} strokeWidth={1.4} />
			<p>No pipelines yet. Create one to track deals, engagements, or delivery through named stages.</p>
			<button class="btn btn--primary" onclick={openNew}><Plus size={16} />Add your first pipeline</button>
		</div>
	{:else}
		<div class="list">
			{#each pipelines as p (p.id)}
				{@const isOpen = expanded === p.id}
				<div class="pipe" class:open={isOpen}>
					<div class="pipe-row">
						<button class="disclose" onclick={() => toggle(p)} aria-label={isOpen ? 'Collapse' : 'Expand'} aria-expanded={isOpen}>
							<span class="chev" class:rot={isOpen}><ChevronRight size={16} /></span>
							{#if p.color}<span class="dot" style="background:{p.color}"></span>{/if}
							<span class="pipe-name">{p.name}</span>
							{#if p.is_default}<span class="pill pill--default" title="Default pipeline"><Star size={11} strokeWidth={2.2} />Default</span>{/if}
							<span class="pill pill--type">{typeLabel(p.pipeline_type)}</span>
							{#if p.is_active === false}<span class="pill pill--inactive">Inactive</span>{/if}
						</button>
						<div class="pipe-actions">
							<button class="icon-btn" title="Edit pipeline" onclick={() => openEdit(p)}><Pencil size={14} /></button>
							<button class="icon-btn icon-btn--danger" title="Delete pipeline" onclick={() => removePipeline(p)}><Trash2 size={14} /></button>
						</div>
					</div>

					{#if p.description}<div class="pipe-desc">{p.description}</div>{/if}

					{#if isOpen}
						<div class="stages">
							{#if stagesLoading === p.id}
								<div class="stage-state"><Loader2 class="spin" size={15} /> Loading stages…</div>
							{:else}
								{@const stages = stagesByPipeline[p.id] ?? []}
								{#if stages.length === 0}
									<div class="stage-state stage-empty"><Layers size={16} strokeWidth={1.5} /> No stages yet.</div>
								{:else}
									<ol class="stage-list">
										{#each stages as s (s.id)}
											<li class="stage">
												<span class="stage-pos">{s.position + 1}</span>
												{#if s.color}<span class="dot dot--sm" style="background:{s.color}"></span>{/if}
												<span class="stage-name">{s.name}</span>
												{#if s.stage_type}<span class="tag tag--{s.stage_type}">{s.stage_type}</span>{/if}
												{#if s.probability != null}<span class="stage-meta">{s.probability}%</span>{/if}
												{#if s.rotting_days != null}<span class="stage-meta">rots {s.rotting_days}d</span>{/if}
												<span class="stage-spacer"></span>
												<button class="icon-btn icon-btn--sm" title="Edit stage" onclick={() => openEditStage(p.id, s)}><Pencil size={13} /></button>
												<button class="icon-btn icon-btn--sm icon-btn--danger" title="Delete stage" onclick={() => removeStage(p.id, s)}><Trash2 size={13} /></button>
											</li>
										{/each}
									</ol>
								{/if}
								<button class="btn btn--ghost btn--sm add-stage" onclick={() => openNewStage(p.id)}><Plus size={14} />Add stage</button>
							{/if}
						</div>
					{/if}
				</div>
			{/each}
		</div>
	{/if}
</div>

{#if showEdit}
	<div class="overlay" role="button" tabindex="0" onclick={() => (showEdit = false)} onkeydown={(e) => e.key === 'Escape' && (showEdit = false)}>
		<div class="modal" role="dialog" tabindex="-1" onclick={(e) => e.stopPropagation()} onkeydown={() => {}}>
			<div class="modal-head"><h2>{editing ? 'Edit pipeline' : 'New pipeline'}</h2><button class="card-x" onclick={() => (showEdit = false)} aria-label="Close"><X size={18} /></button></div>
			<form onsubmit={savePipeline}>
				<label class="field"><span>Name</span><input bind:value={form.name} placeholder="e.g. Sales Pipeline" required /></label>
				<div class="field-row">
					<label class="field"><span>Type</span>
						<select bind:value={form.pipeline_type} disabled={!!editing}>
							{#each TYPES as t}<option value={t.value}>{t.label}</option>{/each}
						</select>
					</label>
					<label class="field"><span>Currency</span><input bind:value={form.currency} placeholder="USD" /></label>
				</div>
				<label class="field"><span>Description</span><textarea bind:value={form.description} rows="3" placeholder="What this pipeline tracks…"></textarea></label>
				<div class="field-row">
					<label class="field"><span>Color</span><input bind:value={form.color} placeholder="#6366f1" /></label>
					{#if editing}
						<label class="field field--check"><input type="checkbox" bind:checked={form.is_active} /><span>Active</span></label>
					{:else}
						<label class="field field--check"><input type="checkbox" bind:checked={form.is_default} /><span>Set as default</span></label>
					{/if}
				</div>
				<div class="modal-actions">
					<button type="button" class="btn btn--ghost" onclick={() => (showEdit = false)}>Cancel</button>
					<button type="submit" class="btn btn--primary" disabled={saving || !form.name.trim()}>{#if saving}<Loader2 class="spin" size={15} />{/if}{editing ? 'Save' : 'Create pipeline'}</button>
				</div>
			</form>
		</div>
	</div>
{/if}

{#if showStage}
	<div class="overlay" role="button" tabindex="0" onclick={() => (showStage = false)} onkeydown={(e) => e.key === 'Escape' && (showStage = false)}>
		<div class="modal" role="dialog" tabindex="-1" onclick={(e) => e.stopPropagation()} onkeydown={() => {}}>
			<div class="modal-head"><h2>{editingStage ? 'Edit stage' : 'New stage'}</h2><button class="card-x" onclick={() => (showStage = false)} aria-label="Close"><X size={18} /></button></div>
			<form onsubmit={saveStage}>
				<label class="field"><span>Name</span><input bind:value={stageForm.name} placeholder="e.g. Qualified" required /></label>
				<div class="field-row">
					<label class="field"><span>Stage type</span>
						<select bind:value={stageForm.stage_type} disabled={!!editingStage}>
							{#each STAGE_TYPES as t}<option value={t.value}>{t.label}</option>{/each}
						</select>
					</label>
					<label class="field"><span>Win probability (%)</span><input bind:value={stageForm.probability} type="number" min="0" max="100" placeholder="e.g. 40" /></label>
				</div>
				<div class="field-row">
					<label class="field"><span>Rotting days</span><input bind:value={stageForm.rotting_days} type="number" min="0" placeholder="e.g. 14" /></label>
					<label class="field"><span>Color</span><input bind:value={stageForm.color} placeholder="#22c55e" /></label>
				</div>
				<label class="field"><span>Description</span><textarea bind:value={stageForm.description} rows="2" placeholder="What happens in this stage…"></textarea></label>
				<div class="modal-actions">
					<button type="button" class="btn btn--ghost" onclick={() => (showStage = false)}>Cancel</button>
					<button type="submit" class="btn btn--primary" disabled={stageSaving || !stageForm.name.trim()}>{#if stageSaving}<Loader2 class="spin" size={15} />{/if}{editingStage ? 'Save' : 'Add stage'}</button>
				</div>
			</form>
		</div>
	</div>
{/if}

<style>
	.pl-root { height: 100%; display: flex; flex-direction: column; background: var(--dbg); color: var(--dt); overflow: hidden; }
	.topbar { display: flex; align-items: center; justify-content: space-between; gap: 16px; padding: 16px 24px; border-bottom: 1px solid var(--dbd); flex-shrink: 0; }
	.title-wrap { display: flex; align-items: center; gap: 10px; }
	.title-wrap h1 { font-size: 1.15rem; font-weight: 680; letter-spacing: -0.02em; margin: 0; }
	.count { font-size: 0.74rem; color: var(--dt3); background: color-mix(in srgb, var(--dt) 8%, transparent); padding: 2px 9px; border-radius: 999px; }
	.tools { display: flex; align-items: center; gap: 10px; }
	.btn { display: inline-flex; align-items: center; gap: 7px; padding: 8px 15px; border-radius: 9px; font-size: 0.83rem; font-weight: 580; cursor: pointer; border: 1px solid transparent; white-space: nowrap; font-family: inherit; }
	.btn--primary { background: var(--dt); color: var(--dbg); }
	.btn--ghost { background: transparent; border-color: var(--dbd); color: var(--dt2); }
	.btn--sm { padding: 6px 11px; font-size: 0.78rem; }
	.btn:disabled { opacity: 0.55; cursor: not-allowed; }
	.state { flex: 1; display: flex; flex-direction: column; align-items: center; justify-content: center; gap: 14px; color: var(--dt3); text-align: center; padding: 0 24px; }
	.empty p { max-width: 420px; line-height: 1.5; }
	.banner { margin: 14px 24px 0; padding: 11px 14px; border-radius: 10px; font-size: 0.83rem; }
	.banner--error { background: color-mix(in srgb, #ef4444 12%, transparent); color: #ef4444; }
	.list { flex: 1; overflow-y: auto; padding: 18px 24px; max-width: 860px; width: 100%; margin: 0 auto; }
	.pipe { border: 1px solid var(--dbd); border-radius: 12px; margin-bottom: 12px; padding: 4px 6px 6px; }
	.pipe.open { background: color-mix(in srgb, var(--dt) 2%, transparent); }
	.pipe-row { display: flex; align-items: center; gap: 8px; padding: 8px 8px; }
	.disclose { flex: 1; min-width: 0; display: flex; align-items: center; gap: 9px; background: transparent; border: none; color: var(--dt); cursor: pointer; text-align: left; padding: 2px; font-family: inherit; }
	.chev { display: inline-flex; color: var(--dt3); transition: transform 0.15s; flex-shrink: 0; }
	.chev.rot { transform: rotate(90deg); }
	.dot { width: 9px; height: 9px; border-radius: 999px; flex-shrink: 0; }
	.dot--sm { width: 7px; height: 7px; }
	.pipe-name { font-size: 0.95rem; font-weight: 620; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
	.pill { display: inline-flex; align-items: center; gap: 4px; font-size: 0.68rem; font-weight: 620; text-transform: uppercase; letter-spacing: 0.04em; padding: 2px 8px; border-radius: 999px; flex-shrink: 0; }
	.pill--type { background: color-mix(in srgb, var(--dt) 9%, transparent); color: var(--dt3); }
	.pill--default { background: color-mix(in srgb, #f59e0b 16%, transparent); color: #f59e0b; }
	.pill--inactive { background: color-mix(in srgb, var(--dt) 8%, transparent); color: var(--dt3); }
	.pipe-actions { display: flex; gap: 4px; flex-shrink: 0; opacity: 0; transition: opacity 0.12s; }
	.pipe-row:hover .pipe-actions { opacity: 1; }
	.pipe-desc { font-size: 0.84rem; color: var(--dt2); line-height: 1.5; padding: 0 10px 8px 34px; }
	.stages { padding: 4px 10px 8px 34px; }
	.stage-state { display: flex; align-items: center; gap: 8px; font-size: 0.82rem; color: var(--dt3); padding: 6px 0; }
	.stage-empty { color: var(--dt3); }
	.stage-list { list-style: none; margin: 0 0 8px; padding: 0; display: flex; flex-direction: column; gap: 4px; }
	.stage { display: flex; align-items: center; gap: 9px; padding: 8px 10px; border: 1px solid var(--dbd); border-radius: 9px; }
	.stage-pos { display: inline-flex; align-items: center; justify-content: center; width: 20px; height: 20px; border-radius: 6px; background: color-mix(in srgb, var(--dt) 8%, transparent); color: var(--dt3); font-size: 0.72rem; font-weight: 640; flex-shrink: 0; }
	.stage-name { font-size: 0.86rem; font-weight: 560; }
	.stage-spacer { flex: 1; }
	.stage-meta { font-size: 0.74rem; color: var(--dt3); }
	.tag { font-size: 0.66rem; font-weight: 620; text-transform: uppercase; letter-spacing: 0.04em; padding: 1px 7px; border-radius: 999px; }
	.tag--open { background: color-mix(in srgb, var(--dt) 10%, transparent); color: var(--dt3); }
	.tag--won { background: color-mix(in srgb, #22c55e 16%, transparent); color: #22c55e; }
	.tag--lost { background: color-mix(in srgb, #ef4444 14%, transparent); color: #ef4444; }
	.add-stage { margin-top: 2px; }
	.icon-btn { display: inline-flex; align-items: center; justify-content: center; width: 28px; height: 28px; border-radius: 7px; border: none; background: transparent; color: var(--dt3); cursor: pointer; }
	.icon-btn--sm { width: 24px; height: 24px; }
	.icon-btn:hover { background: color-mix(in srgb, var(--dt) 8%, transparent); color: var(--dt); }
	.icon-btn--danger:hover { background: color-mix(in srgb, #ef4444 12%, transparent); color: #ef4444; }
	.overlay { position: fixed; inset: 0; background: rgba(0,0,0,0.55); display: flex; align-items: center; justify-content: center; z-index: 100; padding: 20px; }
	.modal { width: 100%; max-width: 480px; background: var(--dbg); border: 1px solid var(--dbd); border-radius: 16px; padding: 22px; box-shadow: 0 24px 60px rgba(0,0,0,0.5); }
	.modal-head { display: flex; align-items: center; justify-content: space-between; margin-bottom: 18px; }
	.modal-head h2 { font-size: 1.05rem; font-weight: 640; margin: 0; }
	.card-x { display: inline-flex; align-items: center; justify-content: center; width: 28px; height: 28px; border-radius: 7px; border: none; background: transparent; color: var(--dt3); cursor: pointer; }
	.card-x:hover { background: color-mix(in srgb, var(--dt) 8%, transparent); }
	.field { display: flex; flex-direction: column; gap: 6px; margin-bottom: 14px; }
	.field span { font-size: 0.8rem; font-weight: 560; color: var(--dt2); }
	.field input, .field textarea, .field select { padding: 9px 12px; border-radius: 9px; border: 1px solid var(--dbd); background: var(--dbg); color: var(--dt); font-size: 0.86rem; outline: none; font-family: inherit; resize: vertical; }
	.field input:focus, .field textarea:focus, .field select:focus { border-color: color-mix(in srgb, var(--dt) 40%, transparent); }
	.field select:disabled { opacity: 0.6; cursor: not-allowed; }
	.field--check { flex-direction: row; align-items: center; gap: 9px; }
	.field--check input { width: 16px; height: 16px; padding: 0; accent-color: var(--dt); }
	.field-row { display: flex; gap: 12px; }
	.field-row .field { flex: 1; }
	:global(.spin) { animation: spin 0.9s linear infinite; }
	@keyframes spin { to { transform: rotate(360deg); } }
	@media (max-width: 480px) { .list { padding: 14px; } }
</style>
