<script lang="ts">
	import { onMount } from 'svelte';
	import { api } from '$lib/api';
	import { currentWorkspace } from '$lib/stores/workspaces';
	import { useSession } from '$lib/auth-client';
	import type { Task } from '$lib/api/dashboard/types';
	import type { TeamMemberListResponse } from '$lib/api/team';
	import { LayoutGrid, List, Plus, Loader2, X, Calendar, Trash2, Search, CheckCircle2, Circle, UserRound } from 'lucide-svelte';

	type Status = 'todo' | 'in_progress' | 'done' | 'cancelled';
	type Priority = 'critical' | 'high' | 'medium' | 'low';
	type Quick = 'all' | 'mine' | 'overdue' | 'today';

	const STATUSES: { id: Status; label: string }[] = [
		{ id: 'todo', label: 'To do' },
		{ id: 'in_progress', label: 'In progress' },
		{ id: 'done', label: 'Done' },
		{ id: 'cancelled', label: 'Cancelled' }
	];
	const PRIORITY_COLOR: Record<Priority, string> = { critical: '#f87171', high: '#fb923c', medium: '#facc15', low: '#6b7280' };
	const QUICKS: { id: Quick; label: string }[] = [
		{ id: 'all', label: 'All' },
		{ id: 'mine', label: 'My tasks' },
		{ id: 'overdue', label: 'Overdue' },
		{ id: 'today', label: 'Due today' }
	];

	const session = useSession();

	let tasks = $state<Task[]>([]);
	let members = $state<TeamMemberListResponse[]>([]);
	let loading = $state(true);
	let error = $state<string | null>(null);
	let view = $state<'board' | 'list'>('board');
	let query = $state('');
	let quick = $state<Quick>('all');
	let busyId = $state<string | null>(null);
	let projectFilter = $state<string>('all');
	let projects = $state<{ id: string; name: string; client_name: string | null }[]>([]);
	let draggedTask = $state<Task | null>(null);
	let dragOverCol = $state<string | null>(null);

	let showCreate = $state(false);
	let creating = $state(false);
	let form = $state({ title: '', description: '', priority: 'medium' as Priority, due_date: '', assignee_id: '' });

	// Reload when workspace changes
	let wsId = $state<string | null | undefined>(null);
	$effect(() => {
		const id = $currentWorkspace?.id ?? null;
		if (id !== wsId) { wsId = id; load(); }
	});

	onMount(load);

	async function load() {
		loading = true; error = null;
		try { tasks = await api.getTasks(); }
		catch (e) { error = e instanceof Error ? e.message : 'Failed to load tasks'; }
		finally { loading = false; }
		// Assignee names are enrichment only; tasks still render if this fails.
		try { members = await api.getTeamMembers(); } catch { members = []; }
		// Projects enable grouping by project / company (via project.client_name).
		try { projects = (await api.getProjects()) as typeof projects; } catch { projects = []; }
	}

	// "Me" = the team member(s) whose email matches the signed-in session user.
	// tasks.assignee_id points at team_members.id, so both the pill count and the
	// filter below derive from this ONE set. Never a hardcoded id.
	const sessionEmail = $derived(($session.data?.user?.email ?? '').trim().toLowerCase());
	const myMemberIds = $derived.by(() => {
		if (!sessionEmail) return new Set<string>();
		return new Set(members.filter((m) => (m.email ?? '').trim().toLowerCase() === sessionEmail).map((m) => m.id));
	});
	const memberById = $derived(new Map(members.map((m) => [m.id, m.name])));

	function assigneeName(t: Task): string | null {
		return t.assignee_id ? (memberById.get(t.assignee_id) ?? null) : null;
	}
	function dayOf(iso: string): number | null {
		const d = new Date(iso);
		if (isNaN(d.getTime())) return null;
		return new Date(d.getFullYear(), d.getMonth(), d.getDate()).getTime();
	}
	function today(): number { const n = new Date(); return new Date(n.getFullYear(), n.getMonth(), n.getDate()).getTime(); }
	function isOpen(t: Task): boolean { const s = t.status ?? 'todo'; return s !== 'done' && s !== 'cancelled'; }
	function isMine(t: Task): boolean { return !!t.assignee_id && myMemberIds.has(t.assignee_id); }
	function isOverdue(t: Task): boolean {
		if (!t.due_date || !isOpen(t)) return false;
		const d = dayOf(t.due_date); return d !== null && d < today();
	}
	function isDueToday(t: Task): boolean {
		if (!t.due_date) return false;
		const d = dayOf(t.due_date); return d !== null && d === today();
	}
	function matchesQuick(t: Task): boolean {
		if (quick === 'mine') return isMine(t);
		if (quick === 'overdue') return isOverdue(t);
		if (quick === 'today') return isDueToday(t);
		return true;
	}

	const filtered = $derived.by(() => {
		const q = query.trim().toLowerCase();
		return tasks.filter((t) =>
			(projectFilter === 'all' || t.project_id === projectFilter) &&
			(!q || `${t.title} ${t.description ?? ''}`.toLowerCase().includes(q))
		);
	});
	// Counts and the visible set share the same predicates over the same base set,
	// so a pill's count always equals what clicking it shows.
	const quickCounts = $derived.by(() => {
		const c: Record<Quick, number> = { all: filtered.length, mine: 0, overdue: 0, today: 0 };
		for (const t of filtered) {
			if (isMine(t)) c.mine++;
			if (isOverdue(t)) c.overdue++;
			if (isDueToday(t)) c.today++;
		}
		return c;
	});
	const visible = $derived(filtered.filter(matchesQuick));

	// Columns are ALWAYS status (To do / In progress / Done / Cancelled). The
	// dropdown is a PROJECT FILTER — it scopes which project's tasks show; it does
	// not restructure the board. "All projects" shows everything.
	function stripSop(n: string): string { return n.replace(/^\[SOP\]\s*/, ''); }
	const columns = $derived(
		STATUSES.map((s) => ({ key: s.id, label: s.label, tasks: visible.filter((t) => (t.status ?? 'todo') === s.id) }))
	);

	async function create(e: Event) {
		e.preventDefault();
		if (!form.title.trim()) return;
		creating = true; error = null;
		try {
			const created = await api.createTask({
				title: form.title.trim(),
				description: form.description.trim() || undefined,
				priority: form.priority,
				due_date: form.due_date || undefined,
				assignee_id: form.assignee_id || undefined
			});
			tasks = [created, ...tasks];
			showCreate = false;
			form = { title: '', description: '', priority: 'medium', due_date: '', assignee_id: '' };
		} catch (e) { error = e instanceof Error ? e.message : 'Failed to create task'; }
		finally { creating = false; }
	}

	async function setStatus(t: Task, status: Status) {
		busyId = t.id; error = null;
		try { await api.updateTask(t.id, { status }); tasks = tasks.map((x) => (x.id === t.id ? { ...x, status } : x)); }
		catch (e) { error = e instanceof Error ? e.message : 'Failed to update'; }
		finally { busyId = null; }
	}
	async function toggleDone(t: Task) {
		await setStatus(t, t.status === 'done' ? 'todo' : 'done');
	}
	async function remove(t: Task) {
		if (!confirm(`Delete "${t.title}"?`)) return;
		busyId = t.id;
		try { await api.deleteTask(t.id); tasks = tasks.filter((x) => x.id !== t.id); }
		catch (e) { error = e instanceof Error ? e.message : 'Failed to delete'; }
		finally { busyId = null; }
	}
	function fmtDate(d?: string | null): string {
		if (!d) return '';
		try { return new Date(d).toLocaleDateString(undefined, { month: 'short', day: 'numeric' }); } catch { return ''; }
	}
</script>

<div class="proj-root">
	<header class="topbar">
		<div class="title-wrap"><h1>Tasks</h1><span class="count">{visible.length}</span></div>
		<div class="tools">
			<div class="search"><Search size={15} strokeWidth={2} /><input placeholder="Search tasks" bind:value={query} /></div>
			<select class="group-sel" bind:value={projectFilter} aria-label="Filter by project">
				<option value="all">All projects</option>
				{#each projects as p (p.id)}<option value={p.id}>{stripSop(p.name)}</option>{/each}
			</select>
			<div class="seg">
				<button class:active={view === 'board'} onclick={() => (view = 'board')} aria-label="Board view"><LayoutGrid size={16} /></button>
				<button class:active={view === 'list'} onclick={() => (view = 'list')} aria-label="List view"><List size={16} /></button>
			</div>
			<button class="btn btn--primary" onclick={() => (showCreate = true)}><Plus size={16} strokeWidth={2.4} />New task</button>
		</div>
	</header>

	{#if error}<div class="banner banner--error">{error}</div>{/if}

	{#if !loading && tasks.length > 0}
		<div class="pills" role="tablist" aria-label="Quick filters">
			{#each QUICKS as f (f.id)}
				<button class="pill" class:active={quick === f.id} role="tab" aria-selected={quick === f.id} onclick={() => (quick = f.id)}>
					{f.label}<span class="pill-count">{quickCounts[f.id]}</span>
				</button>
			{/each}
		</div>
	{/if}

	{#if loading}
		<div class="loading"><Loader2 class="spin" size={20} /> Loading tasks…</div>
	{:else if tasks.length === 0}
		<div class="empty"><CheckCircle2 size={26} strokeWidth={1.4} /><p>No tasks yet.</p>
			<button class="btn btn--primary" onclick={() => (showCreate = true)}><Plus size={16} />Create your first task</button>
		</div>
	{:else if visible.length === 0}
		<div class="empty"><CheckCircle2 size={26} strokeWidth={1.4} />
			{#if quick === 'mine' && myMemberIds.size === 0}
				<p>No team member matches your account{sessionEmail ? ` (${sessionEmail})` : ''}, so no tasks can be assigned to you yet.</p>
			{:else if query.trim()}
				<p>No tasks match "{query}".</p>
			{:else if quick === 'mine'}
				<p>No tasks assigned to you.</p>
			{:else if quick === 'overdue'}
				<p>No overdue tasks.</p>
			{:else if quick === 'today'}
				<p>Nothing due today.</p>
			{:else}
				<p>No tasks to show.</p>
			{/if}
			<button class="btn btn--ghost" onclick={() => { query = ''; quick = 'all'; }}>Clear filters</button>
		</div>
	{:else if view === 'board'}
		<div class="board">
			{#each columns as col (col.key)}
				<div
					class="column {dragOverCol === col.key ? 'column--dragover' : ''}"
					role="group"
					aria-label="{col.label} column"
					ondragover={(e) => { e.preventDefault(); dragOverCol = col.key; }}
					ondragleave={() => { if (dragOverCol === col.key) dragOverCol = null; }}
					ondrop={() => { if (draggedTask && (draggedTask.status ?? 'todo') !== col.key) setStatus(draggedTask, col.key as Status); draggedTask = null; dragOverCol = null; }}
				>
					<div class="col-head"><span>{col.label}</span><span class="col-count">{col.tasks.length}</span></div>
					<div class="col-body">
						{#each col.tasks as t (t.id)}
							<div class="card" draggable="true" ondragstart={() => (draggedTask = t)} ondragend={() => { draggedTask = null; dragOverCol = null; }}>
								<div class="card-top">
									<span class="prio-dot" style="background:{PRIORITY_COLOR[(t.priority ?? 'low') as Priority]}" title={t.priority}></span>
									<span class="card-name" class:done={t.status === 'done'}>{t.title}</span>
									<button class="card-x" title="Delete" onclick={() => remove(t)} disabled={busyId === t.id}><Trash2 size={13} /></button>
								</div>
								{#if t.description}<p class="card-desc">{t.description}</p>{/if}
								{#if t.due_date || assigneeName(t)}
									<div class="card-meta">
										{#if t.due_date}<span class="chip" class:due-over={isOverdue(t)}><Calendar size={11} />{fmtDate(t.due_date)}</span>{/if}
										{#if assigneeName(t)}<span class="chip"><UserRound size={11} />{assigneeName(t)}</span>{/if}
									</div>
								{/if}
								<select class="status-sel" value={t.status} disabled={busyId === t.id} onchange={(e) => setStatus(t, (e.target as HTMLSelectElement).value as Status)} aria-label="Status">
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
			<div class="list-head"><span class="lh-name">Task</span><span class="lh-col">Status</span><span class="lh-col">Priority</span><span class="lh-col">Assignee</span><span class="lh-col">Due</span><span class="lh-x"></span></div>
			{#each visible as t (t.id)}
				<div class="lrow">
					<span class="lr-name">
						<button class="check" onclick={() => toggleDone(t)} aria-label="Toggle done">
							{#if t.status === 'done'}<CheckCircle2 size={16} />{:else}<Circle size={16} />{/if}
						</button>
						<span class:done={t.status === 'done'}>{t.title}</span>
					</span>
					<span class="lr-col">
						<select class="status-sel" value={t.status} disabled={busyId === t.id} onchange={(e) => setStatus(t, (e.target as HTMLSelectElement).value as Status)} aria-label="Status">
							{#each STATUSES as s}<option value={s.id}>{s.label}</option>{/each}
						</select>
					</span>
					<span class="lr-col cap lr-prio"><span class="prio-dot" style="background:{PRIORITY_COLOR[(t.priority ?? 'low') as Priority]}"></span>{t.priority ?? 'low'}</span>
					<span class="lr-col" class:muted={!assigneeName(t)}>{assigneeName(t) ?? 'Unassigned'}</span>
					<span class="lr-col" class:due-over={isOverdue(t)}>{fmtDate(t.due_date) || '—'}</span>
					<span class="lr-x"><button class="card-x" title="Delete" onclick={() => remove(t)} disabled={busyId === t.id}><Trash2 size={14} /></button></span>
				</div>
			{/each}
		</div>
	{/if}
</div>

{#if showCreate}
	<div class="overlay" role="button" tabindex="0" onclick={() => (showCreate = false)} onkeydown={(e) => e.key === 'Escape' && (showCreate = false)}>
		<div class="modal" role="dialog" tabindex="-1" onclick={(e) => e.stopPropagation()} onkeydown={() => {}}>
			<div class="modal-head"><h2>New task</h2><button class="card-x" onclick={() => (showCreate = false)} aria-label="Close"><X size={18} /></button></div>
			<form onsubmit={create}>
				<label class="field"><span>Title</span><input bind:value={form.title} placeholder="What needs to happen?" required /></label>
				<label class="field"><span>Description</span><textarea rows="2" bind:value={form.description} placeholder="Details (optional)"></textarea></label>
				<div class="field-row">
					<label class="field"><span>Priority</span>
						<select bind:value={form.priority}><option value="critical">Critical</option><option value="high">High</option><option value="medium">Medium</option><option value="low">Low</option></select>
					</label>
					<label class="field"><span>Due date</span><input type="date" bind:value={form.due_date} /></label>
				</div>
				{#if members.length > 0}
					<label class="field"><span>Assignee</span>
						<select bind:value={form.assignee_id}>
							<option value="">Unassigned</option>
							{#each members as m (m.id)}<option value={m.id}>{m.name}</option>{/each}
						</select>
					</label>
				{/if}
				<div class="modal-actions">
					<button type="button" class="btn btn--ghost" onclick={() => (showCreate = false)}>Cancel</button>
					<button type="submit" class="btn btn--primary" disabled={creating || !form.title.trim()}>{#if creating}<Loader2 class="spin" size={15} />{/if}Create task</button>
				</div>
			</form>
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
	.group-sel { font-size: 0.8rem; padding: 7px 10px; border: 1px solid var(--dbd); border-radius: 9px; background: var(--dbg); color: var(--dt2); cursor: pointer; }
	.seg button { display: flex; align-items: center; justify-content: center; width: 34px; height: 32px; background: transparent; border: none; color: var(--dt3); cursor: pointer; }
	.seg button.active { background: color-mix(in srgb, var(--dt) 10%, transparent); color: var(--dt); }
	.btn { display: inline-flex; align-items: center; gap: 7px; padding: 8px 15px; border-radius: 9px; font-size: 0.83rem; font-weight: 580; cursor: pointer; border: 1px solid transparent; white-space: nowrap; }
	.btn--primary { background: var(--dt); color: var(--dbg); }
	.btn--ghost { background: transparent; border-color: var(--dbd); color: var(--dt2); }
	.btn:disabled { opacity: 0.55; cursor: not-allowed; }
	.pills { display: flex; align-items: center; gap: 6px; padding: 12px 24px 0; flex-wrap: wrap; flex-shrink: 0; }
	.pill { display: inline-flex; align-items: center; gap: 6px; padding: 5px 11px; border-radius: 999px; border: 1px solid var(--dbd); background: transparent; color: var(--dt3); font-size: 0.78rem; font-weight: 560; cursor: pointer; white-space: nowrap; }
	.pill:hover { color: var(--dt); }
	.pill.active { background: color-mix(in srgb, var(--dt) 10%, transparent); color: var(--dt); border-color: transparent; }
	.pill-count { font-size: 0.7rem; color: var(--dt3); }
	.pill.active .pill-count { color: var(--dt2); }
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
	.card-name.done, .lr-name .done { text-decoration: line-through; color: var(--dt3); }
	.card-x { display: inline-flex; align-items: center; justify-content: center; width: 26px; height: 26px; border-radius: 7px; border: none; background: transparent; color: var(--dt3); cursor: pointer; flex-shrink: 0; }
	.card-x:hover { background: color-mix(in srgb, #ef4444 12%, transparent); color: #ef4444; }
	.card-desc { font-size: 0.78rem; color: var(--dt3); margin: 0; line-height: 1.45; display: -webkit-box; -webkit-line-clamp: 2; line-clamp: 2; -webkit-box-orient: vertical; overflow: hidden; }
	.card-meta { display: flex; flex-wrap: wrap; gap: 6px; }
	.chip { display: inline-flex; align-items: center; gap: 4px; font-size: 0.7rem; color: var(--dt3); background: color-mix(in srgb, var(--dt) 6%, transparent); padding: 3px 8px; border-radius: 6px; }
	.status-sel { font-size: 0.74rem; padding: 5px 8px; border-radius: 7px; border: 1px solid var(--dbd); background: var(--dbg); color: var(--dt2); cursor: pointer; width: 100%; }
	.list { flex: 1; overflow-y: auto; padding: 8px 24px 24px; }
	.list-head, .lrow { display: grid; grid-template-columns: 2.4fr 1.1fr 0.8fr 1fr 0.7fr 40px; align-items: center; gap: 12px; }
	.list-head { padding: 10px 12px; font-size: 0.7rem; text-transform: uppercase; letter-spacing: 0.06em; color: var(--dt3); border-bottom: 1px solid var(--dbd); }
	.lrow { padding: 11px 12px; border-bottom: 1px solid var(--dbd); font-size: 0.85rem; }
	.lr-name { display: flex; align-items: center; gap: 9px; font-weight: 560; }
	.check { background: transparent; border: none; color: var(--dt3); cursor: pointer; display: inline-flex; padding: 0; }
	.check:hover { color: var(--dt); }
	.lr-col { color: var(--dt2); }
	.lr-prio { display: inline-flex; align-items: center; gap: 7px; }
	.muted { color: var(--dt3); }
	.due-over { color: #f87171; }
	.chip.due-over { background: color-mix(in srgb, #f87171 12%, transparent); color: #f87171; }
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
	:global(.spin) { animation: spin 0.9s linear infinite; }
	@keyframes spin { to { transform: rotate(360deg); } }

	/* --- mobile: 768px --- */
	@media (max-width: 768px) {
		.topbar { flex-wrap: wrap; gap: 10px; padding: 14px 16px; }
		.tools { flex-wrap: wrap; gap: 8px; width: 100%; }
		.search { flex: 1 1 auto; min-width: 0; }
		.search input { width: 100%; min-width: 0; }
		.pills { padding: 10px 16px 0; }
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

		/* list: hide header row, reflow each row */
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
