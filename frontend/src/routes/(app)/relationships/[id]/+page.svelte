<script lang="ts">
	import { page } from '$app/stores';
	import { api } from '$lib/api';
	import type {
		ClientBoardResponse,
		BoardProject,
		BoardTask,
		BoardTeamMember,
		DealResponse,
		InteractionResponse
	} from '$lib/api/clients/types';
	import type { TaskStatus } from '$lib/api/dashboard/types';
	import {
		ChevronLeft,
		Loader2,
		Building2,
		FolderKanban,
		ListChecks,
		Users,
		Handshake,
		MessageSquare,
		CalendarClock,
		SearchX
	} from 'lucide-svelte';

	// The backend board endpoint returns a couple of fields the shared types
	// do not declare yet (projects.progress, team.team_member_id/user_id).
	// Extend locally instead of touching the shared types file.
	type ProjectRow = BoardProject & { progress?: number | null };
	type TeamRow = BoardTeamMember & {
		team_member_id?: string | null;
		user_id?: string | null;
	};

	const clientId = $derived($page.params.id ?? '');

	let board = $state<ClientBoardResponse | null>(null);
	let tasks = $state<BoardTask[]>([]);
	let loading = $state(true);
	let notFound = $state(false);
	let error = $state<string | null>(null);
	let taskBusyId = $state<string | null>(null);

	// Re-load whenever the route param changes (also covers first mount).
	let loadedId = $state<string | null>(null);
	$effect(() => {
		if (clientId && clientId !== loadedId) {
			loadedId = clientId;
			load(clientId);
		}
	});

	async function load(id: string) {
		loading = true;
		notFound = false;
		error = null;
		board = null;
		tasks = [];
		try {
			const b = await api.getClientBoard(id);
			board = b;
			tasks = b.tasks ?? [];
		} catch (e) {
			const msg = e instanceof Error ? e.message : 'Failed to load client board';
			if (msg.includes('HTTP 404')) {
				notFound = true;
			} else {
				error = msg;
			}
		} finally {
			loading = false;
		}
	}

	const projects = $derived((board?.projects ?? []) as ProjectRow[]);
	const team = $derived((board?.team ?? []) as TeamRow[]);
	const deals = $derived((board?.deals ?? []) as DealResponse[]);
	const interactions = $derived((board?.interactions ?? []) as InteractionResponse[]);

	// ---- tasks grouped by status (inline status editing keeps this live) ----
	const TASK_GROUPS: { id: TaskStatus; label: string }[] = [
		{ id: 'todo', label: 'To do' },
		{ id: 'in_progress', label: 'In progress' },
		{ id: 'done', label: 'Done' },
		{ id: 'cancelled', label: 'Cancelled' }
	];

	function normStatus(s: string | null | undefined): TaskStatus {
		const v = (s ?? '').toLowerCase();
		if (v === 'in_progress') return 'in_progress';
		if (v === 'done' || v === 'completed') return 'done';
		if (v === 'cancelled') return 'cancelled';
		return 'todo';
	}

	function tasksByStatus(s: TaskStatus): BoardTask[] {
		return tasks.filter((t) => normStatus(t.status) === s);
	}

	// Task counts derive from live task state so the strip stays honest after
	// inline status changes; the other counts come from the server aggregate.
	const tasksOpen = $derived(
		tasks.filter((t) => {
			const s = normStatus(t.status);
			return s === 'todo' || s === 'in_progress';
		}).length
	);
	const tasksDone = $derived(tasks.filter((t) => normStatus(t.status) === 'done').length);

	async function setTaskStatus(t: BoardTask, status: TaskStatus) {
		taskBusyId = t.id;
		error = null;
		try {
			await api.updateTask(t.id, { status });
			tasks = tasks.map((x) => (x.id === t.id ? { ...x, status } : x));
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to update task';
		} finally {
			taskBusyId = null;
		}
	}

	function projectName(id: string | null): string {
		if (!id) return '';
		return projects.find((p) => p.id === id)?.name ?? '';
	}

	function teamKey(m: TeamRow, i: number): string {
		return m.team_member_id || m.user_id || m.email || `row-${i}`;
	}

	// ---- formatting ----
	function fmtDate(iso: string | null | undefined): string {
		if (!iso) return '';
		const d = new Date(iso);
		if (Number.isNaN(d.getTime())) return '';
		return d.toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric' });
	}

	const money = new Intl.NumberFormat('en-US', {
		style: 'currency',
		currency: 'USD',
		maximumFractionDigits: 0
	});

	function stageLabel(s: string): string {
		return s.replace(/_/g, ' ');
	}

	function initial(name: string): string {
		return (name || '?').charAt(0).toUpperCase();
	}
</script>

<div class="board-root">
	<header class="topbar">
		<a class="back" href="/relationships"><ChevronLeft size={16} />Relationships</a>
	</header>

	{#if error}<div class="banner banner--error">{error}</div>{/if}

	{#if loading}
		<div class="center"><Loader2 class="spin" size={20} /> Loading client board…</div>
	{:else if notFound}
		<div class="center center--col">
			<SearchX size={26} strokeWidth={1.4} />
			<p>Client not found.</p>
			<a class="btn btn--ghost" href="/relationships">Back to Relationships</a>
		</div>
	{:else if board}
		{@const client = board.client}
		<div class="body">
			<!-- Client identity -->
			<div class="client-head">
				<div class="avatar avatar--lg">{initial(client.name)}</div>
				<div class="ch-name">
					<h1>{client.name}</h1>
					<div class="ch-chips">
						<span class="chip chip--status chip--{client.status}">{client.status}</span>
						<span class="chip"><Building2 size={11} />{client.type}</span>
						{#if client.city || client.state}
							<span class="chip">{[client.city, client.state].filter(Boolean).join(', ')}</span>
						{/if}
					</div>
				</div>
			</div>

			<!-- Counts strip -->
			<div class="counts">
				<div class="stat"><span class="stat-n">{board.counts.projects}</span><span class="stat-l">Projects</span></div>
				<div class="stat"><span class="stat-n">{tasksOpen}</span><span class="stat-l">Open tasks</span></div>
				<div class="stat"><span class="stat-n">{tasksDone}</span><span class="stat-l">Done</span></div>
				<div class="stat"><span class="stat-n">{board.counts.team}</span><span class="stat-l">Team</span></div>
				<div class="stat"><span class="stat-n">{board.counts.deals}</span><span class="stat-l">Deals</span></div>
			</div>

			<!-- Projects -->
			<div class="d-sec d-sec--row">
				<span><FolderKanban size={13} /> Projects <span class="c-count">{projects.length}</span></span>
				<a class="sec-link" href="/projects">Projects module</a>
			</div>
			{#if projects.length === 0}
				<p class="sec-empty">
					No projects linked to this client yet. Tasks and team reach a client board through its
					projects, so those sections are empty too. Link a project to this client in the Projects
					module to populate the board.
				</p>
			{:else}
				<div class="proj-grid">
					{#each projects as p (p.id)}
						<div class="card">
							<div class="card-top">
								<span class="card-name">{p.name}</span>
								{#if p.status}<span class="chip chip--pstatus chip--{p.status}">{p.status}</span>{/if}
							</div>
							<div class="card-meta">
								{#if p.priority}<span class="chip">{p.priority}</span>{/if}
								{#if p.due_date}<span class="chip"><CalendarClock size={11} />{fmtDate(p.due_date)}</span>{/if}
							</div>
							<div class="progress" role="progressbar" aria-valuenow={p.progress ?? 0} aria-valuemin={0} aria-valuemax={100} aria-label="Project progress">
								<div class="progress-bar" style={`width:${Math.min(100, Math.max(0, p.progress ?? 0))}%`}></div>
							</div>
							<span class="progress-n">{p.progress ?? 0}% of tasks done</span>
						</div>
					{/each}
				</div>
			{/if}

			<!-- Tasks -->
			<div class="d-sec d-sec--row">
				<span><ListChecks size={13} /> Tasks <span class="c-count">{tasks.length}</span></span>
				<a class="sec-link" href="/tasks">Tasks module</a>
			</div>
			{#if tasks.length === 0}
				{#if projects.length === 0}
					<p class="sec-empty">No tasks. This client has no projects, and tasks only appear here through a client's projects.</p>
				{:else}
					<p class="sec-empty">
						No tasks in this client's projects yet. Note: tasks without a project do not appear on
						client boards.
					</p>
				{/if}
			{:else}
				<div class="task-groups">
					{#each TASK_GROUPS as g (g.id)}
						{@const group = tasksByStatus(g.id)}
						{#if group.length > 0}
							<div class="task-group">
								<div class="tg-head">{g.label} <span class="c-count">{group.length}</span></div>
								{#each group as t (t.id)}
									<div class="trow">
										<div class="t-main">
											<span class="t-title" class:t-done={g.id === 'done' || g.id === 'cancelled'}>{t.title}</span>
											<div class="t-meta">
												{#if projectName(t.project_id)}<span class="chip">{projectName(t.project_id)}</span>{/if}
												{#if t.priority}<span class="chip">{t.priority}</span>{/if}
												{#if t.due_date}<span class="chip"><CalendarClock size={11} />{fmtDate(t.due_date)}</span>{/if}
											</div>
										</div>
										<select
											class="stage-sel"
											value={normStatus(t.status)}
											disabled={taskBusyId === t.id}
											onchange={(e) => setTaskStatus(t, (e.target as HTMLSelectElement).value as TaskStatus)}
											aria-label="Task status"
										>
											{#each TASK_GROUPS as s}<option value={s.id}>{s.label}</option>{/each}
										</select>
									</div>
								{/each}
							</div>
						{/if}
					{/each}
				</div>
				<p class="sec-note">Tasks without a project do not appear on client boards.</p>
			{/if}

			<!-- Team -->
			<div class="d-sec d-sec--row">
				<span><Users size={13} /> Team <span class="c-count">{team.length}</span></span>
				<a class="sec-link" href="/team">Team module</a>
			</div>
			{#if team.length === 0}
				{#if projects.length === 0}
					<p class="sec-empty">No team members. People appear here when they are assigned to this client's projects, and this client has no projects yet.</p>
				{:else}
					<p class="sec-empty">No one is assigned to this client's projects yet. Add project members in the Projects module.</p>
				{/if}
			{:else}
				<div class="rows">
					{#each team as m, i (teamKey(m, i))}
						<div class="row">
							<div class="avatar avatar--sm">{initial(m.name ?? '')}</div>
							<div class="row-main">
								<span class="row-title">{m.name || 'Unnamed'}</span>
								{#if m.email}<span class="row-sub">{m.email}</span>{/if}
							</div>
							{#if m.role}<span class="chip">{m.role}</span>{/if}
						</div>
					{/each}
				</div>
			{/if}

			<!-- Deals -->
			<div class="d-sec d-sec--row">
				<span><Handshake size={13} /> Deals <span class="c-count">{deals.length}</span></span>
				<a class="sec-link" href="/relationships">Relationships module</a>
			</div>
			{#if deals.length === 0}
				<p class="sec-empty">No deals for this client yet. Add deals from the client's record in Relationships.</p>
			{:else}
				<div class="rows">
					{#each deals as d (d.id)}
						<div class="row">
							<div class="row-main">
								<span class="row-title">{d.name}</span>
								{#if d.expected_close_date}<span class="row-sub">Expected close {fmtDate(d.expected_close_date)}</span>{/if}
							</div>
							<span class="deal-value">{money.format(d.value ?? 0)}</span>
							<span class="chip chip--stage chip--{d.stage}">{stageLabel(d.stage)}</span>
						</div>
					{/each}
				</div>
			{/if}

			<!-- Interactions -->
			<div class="d-sec d-sec--row">
				<span><MessageSquare size={13} /> Interactions <span class="c-count">{interactions.length}</span></span>
				<a class="sec-link" href="/relationships">Relationships module</a>
			</div>
			{#if interactions.length === 0}
				<p class="sec-empty">No interactions logged for this client yet.</p>
			{:else}
				<div class="rows rows--last">
					{#each interactions as it (it.id)}
						<div class="row">
							<span class="chip">{it.type}</span>
							<div class="row-main">
								<span class="row-title">{it.subject}</span>
								{#if it.description}<span class="row-sub">{it.description}</span>{/if}
							</div>
							<span class="row-date">{fmtDate(it.occurred_at)}</span>
						</div>
					{/each}
				</div>
			{/if}
		</div>
	{/if}
</div>

<style>
	.board-root { height: 100%; display: flex; flex-direction: column; background: var(--dbg); color: var(--dt); overflow: hidden; }
	.topbar { display: flex; align-items: center; gap: 14px; padding: 14px 24px; border-bottom: 1px solid var(--dbd); flex-shrink: 0; }
	.back { display: inline-flex; align-items: center; gap: 4px; font-size: 0.82rem; color: var(--dt3); text-decoration: none; }
	.back:hover { color: var(--dt); }

	.center { flex: 1; display: flex; align-items: center; justify-content: center; gap: 8px; color: var(--dt3); }
	.center--col { flex-direction: column; gap: 14px; }
	.banner { margin: 16px 24px 0; padding: 11px 14px; border-radius: 10px; font-size: 0.83rem; }
	.banner--error { background: color-mix(in srgb, #ef4444 12%, transparent); color: #ef4444; }
	.btn { display: inline-flex; align-items: center; gap: 7px; padding: 8px 15px; border-radius: 9px; font-size: 0.83rem; font-weight: 580; cursor: pointer; border: 1px solid transparent; white-space: nowrap; text-decoration: none; }
	.btn--ghost { background: transparent; border-color: var(--dbd); color: var(--dt2); }

	.body { flex: 1; overflow-y: auto; padding: 22px 24px 48px; max-width: 1080px; width: 100%; margin: 0 auto; }

	/* ---- client identity ---- */
	.client-head { display: flex; align-items: center; gap: 14px; }
	.avatar { width: 30px; height: 30px; border-radius: 50%; background: linear-gradient(135deg, #6366f1, #8b5cf6); color: #fff; font-size: 0.78rem; font-weight: 650; display: flex; align-items: center; justify-content: center; flex-shrink: 0; }
	.avatar--lg { width: 46px; height: 46px; font-size: 1.05rem; }
	.avatar--sm { width: 26px; height: 26px; font-size: 0.72rem; }
	.ch-name h1 { font-size: 1.25rem; font-weight: 680; letter-spacing: -0.02em; margin: 0; }
	.ch-chips { display: flex; flex-wrap: wrap; gap: 6px; margin-top: 5px; }

	.chip { display: inline-flex; align-items: center; gap: 4px; font-size: 0.7rem; color: var(--dt3); background: color-mix(in srgb, var(--dt) 6%, transparent); padding: 3px 8px; border-radius: 6px; text-transform: capitalize; }
	.chip--status.chip--active, .chip--pstatus.chip--active, .chip--pstatus.chip--completed, .chip--stage.chip--closed_won { color: #34d399; background: color-mix(in srgb, #34d399 12%, transparent); }
	.chip--status.chip--lead, .chip--status.chip--prospect { color: #60a5fa; background: color-mix(in srgb, #60a5fa 12%, transparent); }
	.chip--status.chip--churned, .chip--stage.chip--closed_lost { color: #ef4444; background: color-mix(in srgb, #ef4444 12%, transparent); }
	.chip--pstatus.chip--paused { color: #f59e0b; background: color-mix(in srgb, #f59e0b 12%, transparent); }

	/* ---- counts strip ---- */
	.counts { display: flex; gap: 10px; margin: 18px 0 6px; flex-wrap: wrap; }
	.stat { flex: 1; min-width: 110px; display: flex; flex-direction: column; gap: 2px; border: 1px solid var(--dbd); border-radius: 11px; padding: 12px 14px; background: color-mix(in srgb, var(--dt) 2%, transparent); }
	.stat-n { font-size: 1.15rem; font-weight: 680; letter-spacing: -0.02em; }
	.stat-l { font-size: 0.7rem; text-transform: uppercase; letter-spacing: 0.05em; color: var(--dt3); }

	/* ---- section headers (matches relationships drawer conventions) ---- */
	.d-sec { font-size: 0.7rem; text-transform: uppercase; letter-spacing: 0.06em; color: var(--dt3); font-weight: 640; margin: 26px 0 10px; padding-bottom: 6px; border-bottom: 1px solid var(--dbd); }
	.d-sec--row { display: flex; align-items: center; justify-content: space-between; }
	.d-sec--row span { display: inline-flex; align-items: center; gap: 6px; }
	.c-count { color: var(--dt3); font-weight: 500; text-transform: none; letter-spacing: 0; }
	.sec-link { font-size: 0.72rem; color: #818cf8; text-decoration: none; text-transform: none; letter-spacing: 0; font-weight: 520; }
	.sec-link:hover { text-decoration: underline; }
	.sec-empty { font-size: 0.82rem; color: var(--dt3); padding: 4px 0; margin: 0; line-height: 1.5; }
	.sec-note { font-size: 0.72rem; color: var(--dt3); margin: 8px 0 0; }

	/* ---- projects ---- */
	.proj-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(260px, 1fr)); gap: 10px; }
	.card { border: 1px solid var(--dbd); border-radius: 11px; padding: 12px; background: color-mix(in srgb, var(--dt) 2%, transparent); display: flex; flex-direction: column; gap: 8px; }
	.card-top { display: flex; align-items: flex-start; justify-content: space-between; gap: 9px; }
	.card-name { font-size: 0.88rem; font-weight: 580; min-width: 0; overflow-wrap: anywhere; }
	.card-meta { display: flex; flex-wrap: wrap; gap: 6px; }
	.progress { height: 5px; border-radius: 999px; background: color-mix(in srgb, var(--dt) 8%, transparent); overflow: hidden; }
	.progress-bar { height: 100%; border-radius: 999px; background: linear-gradient(90deg, #6366f1, #8b5cf6); }
	.progress-n { font-size: 0.68rem; color: var(--dt3); }

	/* ---- tasks ---- */
	.task-groups { display: flex; flex-direction: column; gap: 14px; }
	.tg-head { font-size: 0.72rem; font-weight: 620; color: var(--dt2); text-transform: uppercase; letter-spacing: 0.04em; margin-bottom: 7px; display: inline-flex; gap: 6px; }
	.trow { display: flex; align-items: center; justify-content: space-between; gap: 12px; border: 1px solid var(--dbd); border-radius: 11px; padding: 10px 12px; margin-bottom: 7px; }
	.t-main { min-width: 0; display: flex; flex-direction: column; gap: 5px; }
	.t-title { font-size: 0.85rem; font-weight: 560; overflow-wrap: anywhere; }
	.t-done { color: var(--dt3); text-decoration: line-through; }
	.t-meta { display: flex; flex-wrap: wrap; gap: 6px; }
	.stage-sel { font-size: 0.74rem; padding: 5px 8px; border-radius: 7px; border: 1px solid var(--dbd); background: var(--dbg); color: var(--dt2); cursor: pointer; flex-shrink: 0; }
	.stage-sel:disabled { opacity: 0.55; cursor: not-allowed; }

	/* ---- generic rows (team / deals / interactions) ---- */
	.rows { display: flex; flex-direction: column; gap: 8px; }
	.rows--last { margin-bottom: 8px; }
	.row { display: flex; align-items: center; gap: 11px; border: 1px solid var(--dbd); border-radius: 11px; padding: 11px 12px; }
	.row-main { min-width: 0; flex: 1; display: flex; flex-direction: column; gap: 3px; }
	.row-title { font-size: 0.86rem; font-weight: 570; overflow-wrap: anywhere; }
	.row-sub { font-size: 0.76rem; color: var(--dt3); overflow-wrap: anywhere; }
	.row-date { font-size: 0.76rem; color: var(--dt3); flex-shrink: 0; }
	.deal-value { font-size: 0.86rem; font-weight: 620; flex-shrink: 0; }

	:global(.spin) { animation: spin 0.9s linear infinite; }
	@keyframes spin { to { transform: rotate(360deg); } }

	@media (max-width: 768px) {
		.topbar { padding: 12px 16px; }
		.body { padding: 16px 16px 40px; }
		.banner { margin: 12px 16px 0; }
		.proj-grid { grid-template-columns: 1fr; }
		.counts { gap: 8px; }
		.stat { min-width: calc(50% - 8px); flex: 1 1 calc(50% - 8px); }
		.trow { flex-wrap: wrap; }
		.trow .stage-sel { width: 100%; }
		.row { flex-wrap: wrap; }
	}
</style>
