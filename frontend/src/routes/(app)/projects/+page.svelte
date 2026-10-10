<script lang="ts">
	import { onMount } from 'svelte';
	import { api } from '$lib/api';
	import { currentWorkspace } from '$lib/stores/workspaces';
	import type { Project } from '$lib/api/projects/types';
	import {
		getProjectTemplates,
		createProjectFromTemplate,
		type ProjectTemplate
	} from '$lib/api/projects/templates';
	import {
		LayoutGrid, List, Plus, Loader2, X, Calendar, Building2, Trash2, Search,
		Sparkles, ListChecks, ChevronRight
	} from 'lucide-svelte';

	type Status = 'active' | 'paused' | 'completed' | 'archived';
	type Priority = 'critical' | 'high' | 'medium' | 'low';

	const STATUSES: { id: Status; label: string }[] = [
		{ id: 'active', label: 'Active' },
		{ id: 'paused', label: 'Paused' },
		{ id: 'completed', label: 'Completed' },
		{ id: 'archived', label: 'Archived' }
	];
	const PRIORITY_COLOR: Record<Priority, string> = {
		critical: '#f87171', high: '#fb923c', medium: '#facc15', low: '#6b7280'
	};

	let projects = $state<Project[]>([]);
	let loading = $state(true);
	let error = $state<string | null>(null);
	let view = $state<'board' | 'list'>('board');
	let query = $state('');
	let busyId = $state<string | null>(null);
	let draggedProject = $state<Project | null>(null);
	let dragOverCol = $state<string | null>(null);

	// Create modal
	let showCreate = $state(false);
	let creating = $state(false);
	let form = $state({ name: '', description: '', project_type: '', priority: 'medium' as Priority, client_name: '', due_date: '' });

	// Client-side starter presets: form prefills only, nothing persisted until the user submits.
	type Preset = { id: string; label: string; desc: string; project_type: string; priority: Priority };
	const PRESETS: Preset[] = [
		{ id: 'blank', label: 'Blank', desc: 'Start from scratch', project_type: '', priority: 'medium' },
		{ id: 'client', label: 'Client Work', desc: 'Client-facing delivery', project_type: 'client_work', priority: 'high' },
		{ id: 'internal', label: 'Internal', desc: 'Internal ops or build', project_type: 'internal', priority: 'medium' }
	];
	let activePreset = $state('blank');

	function applyPreset(ps: Preset) {
		activePreset = ps.id;
		form.project_type = ps.project_type;
		form.priority = ps.priority;
	}

	// Template flow: pick a delivery blueprint, then prefill phases/deliverables.
	let templates = $state<ProjectTemplate[]>([]);
	let templatesLoading = $state(false);
	let showTemplates = $state(false);
	let selectedTemplate = $state<ProjectTemplate | null>(null);
	let usingTemplate = $state(false);
	let tplForm = $state({ name: '', client_name: '', due_date: '', priority: 'medium' as Priority });

	// Reload when workspace changes
	let wsId = $state<string | null | undefined>(null);
	$effect(() => {
		const id = $currentWorkspace?.id ?? null;
		if (id !== wsId) { wsId = id; load(); }
	});

	onMount(load);

	async function load() {
		loading = true; error = null;
		try { projects = await api.getProjects(undefined, undefined, wsId ?? undefined); }
		catch (e) { error = e instanceof Error ? e.message : 'Failed to load projects'; }
		finally { loading = false; }
	}

	const filtered = $derived.by(() => {
		const q = query.trim().toLowerCase();
		if (!q) return projects;
		return projects.filter((p) => `${p.name} ${p.client_name ?? ''}`.toLowerCase().includes(q));
	});

	function byStatus(s: Status): Project[] {
		return filtered.filter((p) => (p.status ?? 'active') === s);
	}

	async function create(e: Event) {
		e.preventDefault();
		if (!form.name.trim()) return;
		creating = true; error = null;
		try {
			const created = await api.createProject({
				name: form.name.trim(),
				description: form.description.trim() || undefined,
				project_type: form.project_type || undefined,
				priority: form.priority,
				client_name: form.client_name.trim() || undefined,
				due_date: form.due_date || undefined
			});
			projects = [created, ...projects];
			showCreate = false;
			form = { name: '', description: '', project_type: '', priority: 'medium', client_name: '', due_date: '' };
			activePreset = 'blank';
		} catch (e) { error = e instanceof Error ? e.message : 'Failed to create project'; }
		finally { creating = false; }
	}

	async function openTemplates() {
		showTemplates = true;
		if (templates.length === 0) {
			templatesLoading = true;
			try { templates = await getProjectTemplates(); }
			catch (e) { error = e instanceof Error ? e.message : 'Failed to load templates'; }
			finally { templatesLoading = false; }
		}
	}

	function pickTemplate(t: ProjectTemplate) {
		selectedTemplate = t;
		tplForm = { name: '', client_name: '', due_date: '', priority: 'medium' };
	}

	function closeTemplates() {
		showTemplates = false;
		selectedTemplate = null;
	}

	async function useTemplate(e: Event) {
		e.preventDefault();
		if (!selectedTemplate) return;
		usingTemplate = true; error = null;
		try {
			const created = await createProjectFromTemplate(selectedTemplate.key, {
				name: tplForm.name.trim() || undefined,
				client_name: tplForm.client_name.trim() || undefined,
				due_date: tplForm.due_date || undefined,
				priority: tplForm.priority
			});
			projects = [created, ...projects];
			closeTemplates();
		} catch (e) { error = e instanceof Error ? e.message : 'Failed to create project from template'; }
		finally { usingTemplate = false; }
	}

	async function setStatus(p: Project, status: Status) {
		busyId = p.id; error = null;
		try {
			await api.updateProject(p.id, { status });
			projects = projects.map((x) => (x.id === p.id ? { ...x, status } : x));
		} catch (e) { error = e instanceof Error ? e.message : 'Failed to update'; }
		finally { busyId = null; }
	}

	async function remove(p: Project) {
		if (!confirm(`Delete "${p.name}"?`)) return;
		busyId = p.id;
		try { await api.deleteProject(p.id); projects = projects.filter((x) => x.id !== p.id); }
		catch (e) { error = e instanceof Error ? e.message : 'Failed to delete'; }
		finally { busyId = null; }
	}

	function fmtDate(d?: string | null): string {
		if (!d) return '';
		try { return new Date(d).toLocaleDateString(undefined, { month: 'short', day: 'numeric' }); } catch { return ''; }
	}

	function templateName(p: Project): string | null {
		const m = p.project_metadata as Record<string, unknown> | null;
		const n = m?.template_name;
		return typeof n === 'string' && n ? n : null;
	}

	function phaseCount(p: Project): number {
		const m = p.project_metadata as Record<string, unknown> | null;
		const ph = m?.phases;
		return Array.isArray(ph) ? ph.length : 0;
	}
</script>

<div class="proj-root">
	<header class="topbar">
		<div class="title-wrap">
			<h1>Projects</h1>
			<span class="count">{filtered.length}</span>
		</div>
		<div class="tools">
			<div class="search">
				<Search size={15} strokeWidth={2} />
				<input placeholder="Search projects" bind:value={query} />
			</div>
			<div class="seg">
				<button class:active={view === 'board'} onclick={() => (view = 'board')} title="Board" aria-label="Board view"><LayoutGrid size={16} /></button>
				<button class:active={view === 'list'} onclick={() => (view = 'list')} title="List" aria-label="List view"><List size={16} /></button>
			</div>
			<button class="btn btn--ghost" onclick={openTemplates}><Sparkles size={16} strokeWidth={2} />New from Growth Audit template</button>
			<button class="btn btn--primary" onclick={() => (showCreate = true)}><Plus size={16} strokeWidth={2.4} />New project</button>
		</div>
	</header>

	{#if error}<div class="banner banner--error">{error}</div>{/if}

	{#if loading}
		<div class="loading"><Loader2 class="spin" size={20} /> Loading projects…</div>
	{:else if projects.length === 0}
		<div class="empty">
			<LayoutGrid size={26} strokeWidth={1.4} />
			<p>No projects yet.</p>
			<div class="empty-actions">
				<button class="btn btn--ghost" onclick={openTemplates}><Sparkles size={16} />Start from Growth Audit template</button>
				<button class="btn btn--primary" onclick={() => (showCreate = true)}><Plus size={16} />Create your first project</button>
			</div>
		</div>
	{:else if filtered.length === 0}
		<div class="empty">
			<LayoutGrid size={26} strokeWidth={1.4} />
			<p>No projects match "{query}".</p>
			<button class="btn btn--ghost" onclick={() => (query = '')}>Clear search</button>
		</div>
	{:else if view === 'board'}
		<div class="board">
			{#each STATUSES as col}
				<div
					class="column {dragOverCol === col.id ? 'column--dragover' : ''}"
					role="group"
					aria-label="{col.label} column"
					ondragover={(e) => { e.preventDefault(); dragOverCol = col.id; }}
					ondragleave={() => { if (dragOverCol === col.id) dragOverCol = null; }}
					ondrop={() => { if (draggedProject && (draggedProject.status ?? 'ACTIVE') !== col.id) setStatus(draggedProject, col.id as Status); draggedProject = null; dragOverCol = null; }}
				>
					<div class="col-head"><span>{col.label}</span><span class="col-count">{byStatus(col.id).length}</span></div>
					<div class="col-body">
						{#each byStatus(col.id) as p (p.id)}
							<div class="card" draggable="true" ondragstart={() => (draggedProject = p)} ondragend={() => { draggedProject = null; dragOverCol = null; }}>
								<div class="card-top">
									<span class="prio-dot" style="background:{PRIORITY_COLOR[(p.priority ?? 'low') as Priority]}" title={p.priority}></span>
									<span class="card-name">{p.name}</span>
									<button class="card-x" title="Delete" onclick={() => remove(p)} disabled={busyId === p.id}><Trash2 size={13} /></button>
								</div>
								{#if p.description}<p class="card-desc">{p.description}</p>{/if}
								<div class="card-meta">
									{#if p.client_name}<span class="chip"><Building2 size={11} />{p.client_name}</span>{/if}
									{#if p.due_date}<span class="chip"><Calendar size={11} />{fmtDate(p.due_date)}</span>{/if}
									{#if templateName(p)}<span class="chip chip--tpl"><ListChecks size={11} />{templateName(p)}{#if phaseCount(p)} · {phaseCount(p)} phases{/if}</span>{/if}
								</div>
								<select class="status-sel" value={p.status} disabled={busyId === p.id} onchange={(e) => setStatus(p, (e.target as HTMLSelectElement).value as Status)} aria-label="Status">
									{#each STATUSES as s}<option value={s.id}>{s.label}</option>{/each}
								</select>
							</div>
						{/each}
					</div>
				</div>
			{/each}
		</div>
	{:else}
		<div class="list">
			<div class="list-head">
				<span class="lh-name">Name</span><span class="lh-col">Status</span><span class="lh-col">Priority</span><span class="lh-col">Client</span><span class="lh-col">Due</span><span class="lh-x"></span>
			</div>
			{#each filtered as p (p.id)}
				<div class="lrow">
					<span class="lr-name"><span class="prio-dot" style="background:{PRIORITY_COLOR[(p.priority ?? 'low') as Priority]}"></span>{p.name}</span>
					<span class="lr-col">
						<select class="status-sel" value={p.status} disabled={busyId === p.id} onchange={(e) => setStatus(p, (e.target as HTMLSelectElement).value as Status)} aria-label="Status">
							{#each STATUSES as s}<option value={s.id}>{s.label}</option>{/each}
						</select>
					</span>
					<span class="lr-col cap">{p.priority ?? '—'}</span>
					<span class="lr-col">{p.client_name ?? '—'}</span>
					<span class="lr-col">{fmtDate(p.due_date) || '—'}</span>
					<span class="lr-x"><button class="card-x" title="Delete" onclick={() => remove(p)} disabled={busyId === p.id}><Trash2 size={14} /></button></span>
				</div>
			{/each}
		</div>
	{/if}
</div>

{#if showCreate}
	<div class="overlay" role="button" tabindex="0" onclick={() => (showCreate = false)} onkeydown={(e) => e.key === 'Escape' && (showCreate = false)}>
		<div class="modal" role="dialog" tabindex="-1" onclick={(e) => e.stopPropagation()} onkeydown={() => {}}>
			<div class="modal-head"><h2>New project</h2><button class="card-x" onclick={() => (showCreate = false)} aria-label="Close"><X size={18} /></button></div>
			<form onsubmit={create}>
				<div class="presets" role="group" aria-label="Starter preset">
					{#each PRESETS as ps (ps.id)}
						<button type="button" class="preset" class:active={activePreset === ps.id} onclick={() => applyPreset(ps)}>
							<span class="preset-label">{ps.label}</span>
							<span class="preset-desc">{ps.desc}</span>
						</button>
					{/each}
				</div>
				<label class="field"><span>Name</span><input bind:value={form.name} placeholder="Project name" required /></label>
				<label class="field"><span>Description</span><textarea rows="2" bind:value={form.description} placeholder="What is this project?"></textarea></label>
				<div class="field-row">
					<label class="field"><span>Type</span>
						<select bind:value={form.project_type} onchange={() => (activePreset = '')}>
							<option value="">Default</option><option value="internal">Internal</option><option value="client_work">Client Work</option><option value="learning">Learning</option>
						</select>
					</label>
					<label class="field"><span>Priority</span>
						<select bind:value={form.priority}>
							<option value="critical">Critical</option><option value="high">High</option><option value="medium">Medium</option><option value="low">Low</option>
						</select>
					</label>
					<label class="field"><span>Due date</span><input type="date" bind:value={form.due_date} /></label>
				</div>
				<label class="field"><span>Client</span><input bind:value={form.client_name} placeholder="Client name (optional)" /></label>
				<div class="modal-actions">
					<button type="button" class="btn btn--ghost" onclick={() => (showCreate = false)}>Cancel</button>
					<button type="submit" class="btn btn--primary" disabled={creating || !form.name.trim()}>{#if creating}<Loader2 class="spin" size={15} />{/if}Create project</button>
				</div>
			</form>
		</div>
	</div>
{/if}

{#if showTemplates}
	<div class="overlay" role="button" tabindex="0" onclick={closeTemplates} onkeydown={(e) => e.key === 'Escape' && closeTemplates()}>
		<div class="modal modal--wide" role="dialog" tabindex="-1" onclick={(e) => e.stopPropagation()} onkeydown={() => {}}>
			{#if !selectedTemplate}
				<div class="modal-head"><h2>Start from a template</h2><button class="card-x" onclick={closeTemplates} aria-label="Close"><X size={18} /></button></div>
				{#if templatesLoading}
					<div class="tpl-loading"><Loader2 class="spin" size={18} /> Loading templates…</div>
				{:else if templates.length === 0}
					<p class="tpl-empty">No templates available in this workspace.</p>
				{:else}
					<div class="tpl-list">
						{#each templates as t (t.id)}
							<button class="tpl-card" onclick={() => pickTemplate(t)}>
								<div class="tpl-card-top">
									<span class="tpl-icon"><ListChecks size={16} /></span>
									<span class="tpl-name">{t.name}</span>
									{#if t.is_builtin}<span class="tpl-badge">Built-in</span>{/if}
									<ChevronRight class="tpl-chev" size={16} />
								</div>
								<p class="tpl-desc">{t.description}</p>
								<div class="tpl-meta">
									<span class="chip"><ListChecks size={11} />{t.phases.length} phases</span>
									<span class="chip"><Sparkles size={11} />{t.deliverables.length} deliverables</span>
								</div>
							</button>
						{/each}
					</div>
				{/if}
			{:else}
				<div class="modal-head">
					<button class="btn btn--ghost btn--back" onclick={() => (selectedTemplate = null)}>&larr; Templates</button>
					<button class="card-x" onclick={closeTemplates} aria-label="Close"><X size={18} /></button>
				</div>
				<h2 class="tpl-detail-title">{selectedTemplate.name}</h2>
				<p class="tpl-detail-desc">{selectedTemplate.description}</p>
				<form onsubmit={useTemplate}>
					<div class="field-row">
						<label class="field"><span>Project name</span><input bind:value={tplForm.name} placeholder={selectedTemplate.name} /></label>
						<label class="field"><span>Client</span><input bind:value={tplForm.client_name} placeholder="Client name (optional)" /></label>
					</div>
					<div class="field-row">
						<label class="field"><span>Priority</span>
							<select bind:value={tplForm.priority}>
								<option value="critical">Critical</option><option value="high">High</option><option value="medium">Medium</option><option value="low">Low</option>
							</select>
						</label>
						<label class="field"><span>Due date</span><input type="date" bind:value={tplForm.due_date} /></label>
					</div>

					<div class="phases">
						<div class="phases-head">Phases &amp; deliverables</div>
						{#each selectedTemplate.phases as phase, i}
							<div class="phase">
								<div class="phase-name"><span class="phase-num">{i + 1}</span>{phase.name}</div>
								{#if phase.deliverables && phase.deliverables.length}
									<div class="phase-delivs">{#each phase.deliverables as d}<span class="chip chip--tpl">{d}</span>{/each}</div>
								{/if}
							</div>
						{/each}
					</div>

					<div class="modal-actions">
						<button type="button" class="btn btn--ghost" onclick={() => (selectedTemplate = null)}>Back</button>
						<button type="submit" class="btn btn--primary" disabled={usingTemplate}>{#if usingTemplate}<Loader2 class="spin" size={15} />{/if}Create project</button>
					</div>
				</form>
			{/if}
		</div>
	</div>
{/if}

<style>
	.proj-root { height: 100%; display: flex; flex-direction: column; background: var(--dbg); color: var(--dt); overflow: hidden; }
	.topbar { display: flex; align-items: center; justify-content: space-between; gap: 16px; padding: 18px 24px; border-bottom: 1px solid var(--dbd); flex-shrink: 0; }
	.title-wrap { display: flex; align-items: center; gap: 10px; }
	.title-wrap h1 { font-size: 1.15rem; font-weight: 680; letter-spacing: -0.02em; margin: 0; }
	.count { font-size: 0.74rem; color: var(--dt3); background: color-mix(in srgb, var(--dt) 8%, transparent); padding: 2px 9px; border-radius: 999px; }
	.tools { display: flex; align-items: center; gap: 10px; }
	.search { display: flex; align-items: center; gap: 7px; padding: 7px 11px; border: 1px solid var(--dbd); border-radius: 9px; color: var(--dt3); background: color-mix(in srgb, var(--dt) 3%, transparent); }
	.search input { border: 0; outline: 0; background: transparent; color: var(--dt); font-size: 0.82rem; width: 150px; }
	.seg { display: flex; border: 1px solid var(--dbd); border-radius: 9px; overflow: hidden; }
	.seg button { display: flex; align-items: center; justify-content: center; width: 34px; height: 32px; background: transparent; border: none; color: var(--dt3); cursor: pointer; }
	.seg button.active { background: color-mix(in srgb, var(--dt) 10%, transparent); color: var(--dt); }
	.btn { display: inline-flex; align-items: center; gap: 7px; padding: 8px 15px; border-radius: 9px; font-size: 0.83rem; font-weight: 580; cursor: pointer; border: 1px solid transparent; white-space: nowrap; }
	.btn--primary { background: var(--dt); color: var(--dbg); }
	.btn--ghost { background: transparent; border-color: var(--dbd); color: var(--dt2); }
	.btn:disabled { opacity: 0.55; cursor: not-allowed; }

	.board { flex: 1; display: flex; gap: 14px; padding: 18px 24px; overflow-x: auto; }
	.column { flex: 1; min-width: 248px; display: flex; flex-direction: column; border-radius: 10px; transition: background 0.12s, box-shadow 0.12s; }
	.column--dragover { background: color-mix(in srgb, var(--accent, #16a34a) 10%, transparent); box-shadow: inset 0 0 0 2px var(--accent, #16a34a); }
	.card[draggable='true'] { cursor: grab; }
	.card[draggable='true']:active { cursor: grabbing; }
	.col-head { display: flex; align-items: center; gap: 8px; padding: 4px 4px 12px; font-size: 0.78rem; font-weight: 620; color: var(--dt2); text-transform: uppercase; letter-spacing: 0.05em; }
	.col-count { color: var(--dt3); font-weight: 500; }
	.col-body { display: flex; flex-direction: column; gap: 9px; overflow-y: auto; }
	.card { border: 1px solid var(--dbd); border-radius: 11px; padding: 12px; background: color-mix(in srgb, var(--dt) 2%, transparent); display: flex; flex-direction: column; gap: 8px; }
	.card-top { display: flex; align-items: center; gap: 8px; }
	.prio-dot { width: 8px; height: 8px; border-radius: 50%; flex-shrink: 0; }
	.card-name { font-size: 0.88rem; font-weight: 580; flex: 1; min-width: 0; }
	.card-x { display: inline-flex; align-items: center; justify-content: center; width: 26px; height: 26px; border-radius: 7px; border: none; background: transparent; color: var(--dt3); cursor: pointer; flex-shrink: 0; }
	.card-x:hover { background: color-mix(in srgb, #ef4444 12%, transparent); color: #ef4444; }
	.card-desc { font-size: 0.78rem; color: var(--dt3); margin: 0; line-height: 1.45; display: -webkit-box; -webkit-line-clamp: 2; line-clamp: 2; -webkit-box-orient: vertical; overflow: hidden; }
	.card-meta { display: flex; flex-wrap: wrap; gap: 6px; }
	.chip { display: inline-flex; align-items: center; gap: 4px; font-size: 0.7rem; color: var(--dt3); background: color-mix(in srgb, var(--dt) 6%, transparent); padding: 3px 8px; border-radius: 6px; }
	.status-sel { font-size: 0.74rem; padding: 5px 8px; border-radius: 7px; border: 1px solid var(--dbd); background: var(--dbg); color: var(--dt2); cursor: pointer; width: 100%; }

	.list { flex: 1; overflow-y: auto; padding: 8px 24px 24px; }
	.list-head, .lrow { display: grid; grid-template-columns: 2.4fr 1fr 0.8fr 1.2fr 0.7fr 40px; align-items: center; gap: 12px; }
	.list-head { padding: 10px 12px; font-size: 0.7rem; text-transform: uppercase; letter-spacing: 0.06em; color: var(--dt3); border-bottom: 1px solid var(--dbd); }
	.lrow { padding: 11px 12px; border-bottom: 1px solid var(--dbd); font-size: 0.85rem; }
	.lr-name { display: flex; align-items: center; gap: 9px; font-weight: 560; }
	.lr-col { color: var(--dt2); }
	.cap { text-transform: capitalize; }
	.lr-x { display: flex; justify-content: flex-end; }
	.lr-col .status-sel { width: auto; }

	.loading, .empty { flex: 1; display: flex; flex-direction: column; align-items: center; justify-content: center; gap: 14px; color: var(--dt3); }
	.loading { flex-direction: row; gap: 8px; }
	.banner { margin: 16px 24px 0; padding: 11px 14px; border-radius: 10px; font-size: 0.83rem; }
	.banner--error { background: color-mix(in srgb, #ef4444 12%, transparent); color: #ef4444; }

	.overlay { position: fixed; inset: 0; background: rgba(0,0,0,0.55); display: flex; align-items: center; justify-content: center; z-index: 100; padding: 20px; }
	.modal { width: 100%; max-width: 460px; background: var(--dbg); border: 1px solid var(--dbd); border-radius: 16px; padding: 22px; box-shadow: 0 24px 60px rgba(0,0,0,0.5); }
	.modal-head { display: flex; align-items: center; justify-content: space-between; margin-bottom: 18px; }
	.modal-head h2 { font-size: 1.05rem; font-weight: 640; margin: 0; }
	.field { display: flex; flex-direction: column; gap: 6px; margin-bottom: 14px; }
	.field span { font-size: 0.8rem; font-weight: 560; color: var(--dt2); }
	.field input, .field textarea, .field select { padding: 9px 12px; border-radius: 9px; border: 1px solid var(--dbd); background: var(--dbg); color: var(--dt); font-size: 0.86rem; outline: none; font-family: inherit; }
	.field input:focus, .field textarea:focus { border-color: color-mix(in srgb, var(--dt) 40%, transparent); }
	.field-row { display: flex; gap: 12px; }
	.field-row .field { flex: 1; }
	.modal-actions { display: flex; justify-content: flex-end; gap: 10px; margin-top: 6px; }

	/* --- create presets (client-side form prefills) --- */
	.presets { display: flex; gap: 8px; margin-bottom: 16px; }
	.preset { flex: 1; display: flex; flex-direction: column; gap: 3px; text-align: left; padding: 9px 11px; border: 1px solid var(--dbd); border-radius: 10px; background: color-mix(in srgb, var(--dt) 2%, transparent); cursor: pointer; }
	.preset:hover { border-color: color-mix(in srgb, var(--dt) 30%, transparent); }
	.preset.active { border-color: color-mix(in srgb, var(--dt) 45%, transparent); background: color-mix(in srgb, var(--dt) 7%, transparent); }
	.preset-label { font-size: 0.78rem; font-weight: 600; color: var(--dt); }
	.preset-desc { font-size: 0.68rem; color: var(--dt3); line-height: 1.35; }
	:global(.spin) { animation: spin 0.9s linear infinite; }
	@keyframes spin { to { transform: rotate(360deg); } }

	.empty-actions { display: flex; gap: 10px; flex-wrap: wrap; justify-content: center; }

	/* --- template chip --- */
	.chip--tpl { color: var(--dt2); background: color-mix(in srgb, var(--dt) 9%, transparent); }

	/* --- template picker modal --- */
	.modal--wide { max-width: 620px; max-height: 86vh; overflow-y: auto; }
	.btn--back { padding: 5px 10px; font-size: 0.8rem; }
	.tpl-loading { display: flex; align-items: center; gap: 8px; color: var(--dt3); padding: 24px 4px; font-size: 0.86rem; }
	.tpl-empty { color: var(--dt3); padding: 12px 4px; font-size: 0.86rem; }
	.tpl-list { display: flex; flex-direction: column; gap: 10px; }
	.tpl-card { text-align: left; border: 1px solid var(--dbd); border-radius: 12px; padding: 14px; background: color-mix(in srgb, var(--dt) 2%, transparent); cursor: pointer; display: flex; flex-direction: column; gap: 8px; }
	.tpl-card:hover { border-color: color-mix(in srgb, var(--dt) 30%, transparent); background: color-mix(in srgb, var(--dt) 5%, transparent); }
	.tpl-card-top { display: flex; align-items: center; gap: 9px; }
	.tpl-icon { display: inline-flex; align-items: center; justify-content: center; width: 28px; height: 28px; border-radius: 8px; background: color-mix(in srgb, var(--dt) 10%, transparent); color: var(--dt); flex-shrink: 0; }
	.tpl-name { font-size: 0.92rem; font-weight: 620; flex: 1; min-width: 0; }
	.tpl-badge { font-size: 0.66rem; text-transform: uppercase; letter-spacing: 0.05em; color: var(--dt3); border: 1px solid var(--dbd); padding: 2px 7px; border-radius: 999px; }
	:global(.tpl-chev) { color: var(--dt3); flex-shrink: 0; }
	.tpl-desc { font-size: 0.8rem; color: var(--dt3); margin: 0; line-height: 1.5; }
	.tpl-meta { display: flex; flex-wrap: wrap; gap: 6px; }
	.tpl-detail-title { font-size: 1.1rem; font-weight: 660; margin: 4px 0 6px; }
	.tpl-detail-desc { font-size: 0.84rem; color: var(--dt3); line-height: 1.55; margin: 0 0 18px; }
	.phases { border: 1px solid var(--dbd); border-radius: 12px; padding: 6px; margin: 4px 0 18px; display: flex; flex-direction: column; gap: 2px; max-height: 240px; overflow-y: auto; }
	.phases-head { font-size: 0.7rem; text-transform: uppercase; letter-spacing: 0.06em; color: var(--dt3); padding: 8px 10px 6px; }
	.phase { padding: 9px 10px; border-radius: 9px; }
	.phase + .phase { border-top: 1px solid color-mix(in srgb, var(--dt) 6%, transparent); }
	.phase-name { display: flex; align-items: center; gap: 9px; font-size: 0.84rem; font-weight: 560; }
	.phase-num { display: inline-flex; align-items: center; justify-content: center; width: 20px; height: 20px; border-radius: 6px; background: color-mix(in srgb, var(--dt) 10%, transparent); font-size: 0.72rem; font-weight: 640; flex-shrink: 0; }
	.phase-delivs { display: flex; flex-wrap: wrap; gap: 5px; margin: 7px 0 0 29px; }

	/* --- mobile: 768px --- */
	@media (max-width: 768px) {
		.topbar { flex-wrap: wrap; gap: 10px; padding: 14px 16px; }
		.tools { flex-wrap: wrap; gap: 8px; width: 100%; }
		.search { flex: 1 1 auto; min-width: 0; }
		.search input { width: 100%; min-width: 0; }
		.board { padding: 12px 16px; gap: 10px; }
		.column { min-width: 220px; }
		.list { padding: 6px 16px 20px; }
		.banner { margin: 12px 16px 0; }
		.modal { border-radius: 16px; padding: 18px; }
	}

	/* --- mobile: 480px (single-column board, stacked list) --- */
	@media (max-width: 480px) {
		.topbar { padding: 12px 14px; }
		.tools { flex-direction: column; align-items: stretch; }
		.search { width: 100%; }
		.seg { align-self: flex-start; }
		.btn.btn--primary { width: 100%; justify-content: center; min-height: 44px; }

		/* board: single column, no horizontal overflow */
		.board { flex-direction: column; overflow-x: visible; padding: 12px 14px; gap: 14px; }
		.column { min-width: 0; width: 100%; flex: none; }

		/* list: hide secondary columns, reflow to two-column (name + actions) */
		.list-head { display: none; }
		.lrow { display: flex; flex-wrap: wrap; align-items: center; gap: 8px; padding: 12px 10px; }
		.lr-name { flex: 1 1 100%; font-size: 0.9rem; }
		.lr-col { flex: 0 1 auto; font-size: 0.78rem; }
		.lr-x { margin-left: auto; }

		/* modal: bottom-sheet */
		.overlay { align-items: flex-end; padding: 0; }
		.modal { max-width: 100%; border-radius: 20px 20px 0 0; padding: 20px 16px 28px; }
		.field-row { flex-direction: column; gap: 0; }
	}
</style>
