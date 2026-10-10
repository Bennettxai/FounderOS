<script lang="ts">
	// Board renderer (Views & Boards, Phase B).
	// One getBoardData(id) load resolves every view server-side; this page just
	// lays the sections out in layout order with the house dark style.
	// Read-only surface: the only mutation is the pin toggle (updateBoard).
	import { onMount } from 'svelte';
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import { getBoardData, updateBoard, type BoardData, type BoardView } from '$lib/api/boards';
	import { currentWorkspace } from '$lib/stores/workspaces';
	import {
		ChevronLeft,
		Pin,
		PinOff,
		Loader2,
		Inbox,
		ArrowUpRight,
		Filter,
		CalendarClock
	} from 'lucide-svelte';

	const boardId = $derived($page.params.id ?? '');

	// Item shapes as returned by GET /api/boards/:id/data (boards.go view resolvers).
	interface ProjectItem {
		id: string;
		name: string;
		description: string | null;
		status: string | null;
		priority: string | null;
		client_id: string | null;
		client_name: string | null;
		start_date: string | null;
		due_date: string | null;
	}
	interface TaskItem {
		id: string;
		title: string;
		status: string | null;
		priority: string | null;
		due_date: string | null;
		project_id: string | null;
		project_name: string | null;
	}
	interface TeamItem {
		id: string;
		name: string;
		email: string;
		role: string;
		status: string | null;
	}
	interface DealItem {
		id: string;
		name: string;
		amount: number | null;
		currency: string | null;
		status: string | null;
		stage_name: string | null;
		expected_close_date: string | null;
	}
	interface ClientItem {
		id: string;
		name: string;
		status: string | null;
		type: string | null;
	}

	// Sections may carry a per-view resolution error (invalid filter, unknown view).
	interface SectionWithError {
		view: BoardView;
		items: unknown[];
		count: number;
		error?: string;
	}

	let data = $state<BoardData | null>(null);
	let loading = $state(true);
	let error = $state<string | null>(null);
	let notMigrated = $state(false);
	let notFound = $state(false);
	let pinning = $state(false);

	// Reload when the board id or the active workspace changes.
	let loadedKey = $state<string | null>(null);
	$effect(() => {
		const key = `${boardId}::${$currentWorkspace?.id ?? ''}`;
		if (key !== loadedKey) {
			loadedKey = key;
			load();
		}
	});

	onMount(() => {
		if (data === null && !loading) load();
	});

	async function load() {
		if (!boardId) return;
		loading = true;
		error = null;
		notMigrated = false;
		notFound = false;
		try {
			data = await getBoardData(boardId);
		} catch (e) {
			const msg = e instanceof Error ? e.message : 'Failed to load board';
			const m = /\(HTTP (\d{3})\)/.exec(msg);
			const status = m ? Number(m[1]) : 0;
			if (status === 503) {
				notMigrated = true;
			} else if (status === 404 || status === 400) {
				notFound = true;
			} else {
				error = msg;
			}
		} finally {
			loading = false;
		}
	}

	// Pin toggle goes through PUT /boards/:id (updateBoard) - the pinBoard()
	// helper in boards.ts targets a route shape the backend does not expose.
	async function togglePin() {
		if (!data || pinning) return;
		pinning = true;
		error = null;
		try {
			const updated = await updateBoard(data.board.id, { is_pinned: !data.board.is_pinned });
			data = { ...data, board: updated };
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to update pin';
		} finally {
			pinning = false;
		}
	}

	const sections = $derived((data?.sections ?? []) as SectionWithError[]);
	const totalItems = $derived(sections.reduce((sum, s) => sum + (s.count ?? 0), 0));

	// Client context: a board (or any of its views) filtered to one client.
	const contextClientId = $derived.by<string | null>(() => {
		if (!data) return null;
		if (data.board.subject_type === 'client' && data.board.subject_id) {
			return data.board.subject_id;
		}
		const layout = Array.isArray(data.board.layout) ? data.board.layout : [];
		for (const entry of layout) {
			if (entry?.filters?.client_id) return entry.filters.client_id;
		}
		return null;
	});

	// Resolve the client's name from the sections we already have - never a
	// second request. If nothing in the data names the client, no chip.
	const contextClientName = $derived.by<string | null>(() => {
		if (!contextClientId || !data) return null;
		for (const s of sections) {
			if (s.view === 'clients') {
				const hit = (s.items as ClientItem[]).find((c) => c.id === contextClientId);
				if (hit?.name) return hit.name;
			}
			if (s.view === 'projects') {
				const hit = (s.items as ProjectItem[]).find(
					(p) => p.client_id === contextClientId && p.client_name
				);
				if (hit?.client_name) return hit.client_name;
			}
		}
		return null;
	});

	// ── labels / links / formatting ──────────────────────────────────────────

	const VIEW_LABEL: Record<string, string> = {
		projects: 'Projects',
		tasks: 'Tasks',
		team: 'Team',
		deals: 'Deals',
		clients: 'Clients'
	};
	const VIEW_HREF: Record<string, string> = {
		projects: '/projects',
		tasks: '/tasks',
		team: '/team',
		deals: '/relationships',
		clients: '/relationships'
	};

	function viewLabel(view: string): string {
		return VIEW_LABEL[view] ?? pretty(view);
	}

	function pretty(raw: string | null | undefined): string {
		if (!raw) return 'None';
		const s = raw.replace(/[_-]+/g, ' ').trim();
		return s.charAt(0).toUpperCase() + s.slice(1);
	}

	function fmtDate(raw: string | null | undefined): string {
		if (!raw) return '';
		const d = new Date(raw);
		if (Number.isNaN(d.getTime())) return String(raw);
		return d.toLocaleDateString(undefined, { month: 'short', day: 'numeric', year: 'numeric' });
	}

	function dealValue(d: DealItem): string {
		if (d.amount === null || d.amount === undefined) return '';
		try {
			return new Intl.NumberFormat(undefined, {
				style: 'currency',
				currency: (d.currency || 'USD').toUpperCase(),
				maximumFractionDigits: 0
			}).format(d.amount);
		} catch {
			return d.amount.toLocaleString();
		}
	}

	// ── tasks grouped by status ──────────────────────────────────────────────

	const TASK_STATUS_ORDER = [
		'todo',
		'in_progress',
		'in-progress',
		'review',
		'blocked',
		'done',
		'completed',
		'cancelled'
	];

	function groupTasks(items: TaskItem[]): { status: string; tasks: TaskItem[] }[] {
		const map = new Map<string, TaskItem[]>();
		for (const t of items) {
			const key = t.status ?? 'no status';
			const list = map.get(key);
			if (list) list.push(t);
			else map.set(key, [t]);
		}
		return [...map.entries()]
			.map(([status, tasks]) => ({ status, tasks }))
			.sort((a, b) => {
				const ia = TASK_STATUS_ORDER.indexOf(a.status);
				const ib = TASK_STATUS_ORDER.indexOf(b.status);
				return (ia === -1 ? 99 : ia) - (ib === -1 ? 99 : ib);
			});
	}
</script>

<svelte:head>
	<title>{data?.board.name ?? 'Board'} - BusinessOS</title>
</svelte:head>

<div class="page">
	<!-- Header -->
	<header class="topbar">
		<button class="back-btn" onclick={() => goto('/boards')} aria-label="Back to boards">
			<ChevronLeft size={18} />
		</button>
		<div class="title-wrap">
			<h1>{data?.board.name ?? (loading ? '...' : 'Board')}</h1>
			{#if contextClientName}
				<span class="chip chip--ctx"><Filter size={11} />Filtered to {contextClientName}</span>
			{/if}
		</div>
		{#if data}
			<button
				class="btn btn--ghost"
				onclick={togglePin}
				disabled={pinning}
				aria-label={data.board.is_pinned ? 'Unpin board from sidebar' : 'Pin board to sidebar'}
			>
				{#if pinning}
					<Loader2 class="spin" size={14} />
				{:else if data.board.is_pinned}
					<PinOff size={14} />
				{:else}
					<Pin size={14} />
				{/if}
				{data.board.is_pinned ? 'Pinned' : 'Pin'}
			</button>
		{/if}
	</header>

	{#if error}
		<div class="banner banner--error">
			{error}
			<button class="btn btn--ghost btn--sm" onclick={load}>Retry</button>
		</div>
	{/if}

	{#if loading}
		<div class="center"><Loader2 class="spin" size={20} />Loading board...</div>
	{:else if notMigrated}
		<div class="empty">
			<span class="empty-icon"><Inbox size={22} strokeWidth={1.6} /></span>
			<p class="empty-title">Boards are not available yet</p>
			<p class="empty-body">
				The boards table has not been migrated on this backend. Apply the boards migration, then
				reload.
			</p>
			<button class="btn btn--ghost" onclick={load}>Retry</button>
		</div>
	{:else if notFound}
		<div class="empty">
			<span class="empty-icon"><Inbox size={22} strokeWidth={1.6} /></span>
			<p class="empty-title">Board not found</p>
			<p class="empty-body">
				This board does not exist in the current workspace. It may have been deleted, or it belongs
				to a different workspace.
			</p>
			<a class="btn btn--ghost" href="/boards">Back to boards</a>
		</div>
	{:else if data}
		<div class="body">
			<!-- Counts strip -->
			<div class="counts">
				{#each sections as s (s.view)}
					<span class="chip">{viewLabel(s.view)} <strong>{s.count}</strong></span>
				{/each}
				{#if sections.length > 0}
					<span class="chip chip--total">{totalItems} item{totalItems === 1 ? '' : 's'} total</span>
				{/if}
			</div>

			{#if sections.length === 0}
				<div class="empty">
					<span class="empty-icon"><Inbox size={22} strokeWidth={1.6} /></span>
					<p class="empty-title">No views on this board</p>
					<p class="empty-body">
						This board has no views configured yet. Open the boards page to add views to its
						layout.
					</p>
					<a class="btn btn--ghost" href="/boards">Back to boards</a>
				</div>
			{:else}
				<div class="grid">
					{#each sections as s, i (`${s.view}-${i}`)}
						<section class="sec" class:sec--wide={s.view === 'tasks'}>
							<div class="d-sec d-sec--row">
								<span>{viewLabel(s.view)} <span class="c-count">{s.count}</span></span>
								<a class="lnk" href={VIEW_HREF[s.view] ?? '/dashboard'}>
									Open module <ArrowUpRight size={12} />
								</a>
							</div>

							{#if s.error}
								<p class="sec-note">This view could not be resolved: {s.error}.</p>
							{:else if s.view === 'projects'}
								{@const projects = s.items as ProjectItem[]}
								{#if projects.length === 0}
									<p class="sec-empty">
										{#if contextClientId}
											No projects are linked to this client yet.
										{:else}
											No projects yet.
										{/if}
									</p>
								{:else}
									<div class="cards">
										{#each projects as p (p.id)}
											<div class="pcard">
												<div class="pcard-top">
													<span class="pcard-name">{p.name}</span>
													{#if p.status}<span class="chip">{pretty(p.status)}</span>{/if}
												</div>
												{#if p.due_date}
													<span class="meta"><CalendarClock size={11} />Due {fmtDate(p.due_date)}</span>
												{:else}
													<span class="meta meta--dim">No due date</span>
												{/if}
											</div>
										{/each}
									</div>
								{/if}
							{:else if s.view === 'tasks'}
								{@const tasks = s.items as TaskItem[]}
								{#if tasks.length === 0}
									<p class="sec-empty">
										{#if contextClientId}
											No tasks. Tasks attach to a client through its projects, so this stays empty
											until a project linked to this client has tasks.
										{:else}
											No tasks yet.
										{/if}
									</p>
								{:else}
									<div class="tgroups">
										{#each groupTasks(tasks) as g (g.status)}
											<div class="tgroup">
												<div class="tgroup-head">
													{pretty(g.status)} <span class="c-count">{g.tasks.length}</span>
												</div>
												{#each g.tasks as t (t.id)}
													<div class="row">
														<span class="row-name">{t.title}</span>
														<span class="row-col">{t.project_name ?? 'No project'}</span>
														<span class="row-col row-col--end">
															{#if t.due_date}{fmtDate(t.due_date)}{/if}
														</span>
													</div>
												{/each}
											</div>
										{/each}
									</div>
								{/if}
							{:else if s.view === 'team'}
								{@const team = s.items as TeamItem[]}
								{#if team.length === 0}
									<p class="sec-empty">
										{#if contextClientId}
											No team members are assigned to this client's projects yet.
										{:else}
											No team members are assigned to any project yet.
										{/if}
									</p>
								{:else}
									<div class="rows">
										{#each team as m (m.id)}
											<div class="row">
												<span class="row-name">{m.name}</span>
												<span class="row-col">{m.role || 'No role'}</span>
												<span class="row-col row-col--end">{m.email}</span>
											</div>
										{/each}
									</div>
								{/if}
							{:else if s.view === 'deals'}
								{@const deals = s.items as DealItem[]}
								{#if deals.length === 0}
									<p class="sec-empty">
										{#if contextClientId}
											No deals are linked to this client yet.
										{:else}
											No deals yet.
										{/if}
									</p>
								{:else}
									<div class="rows">
										{#each deals as d (d.id)}
											<div class="row">
												<span class="row-name">{d.name}</span>
												<span class="row-col row-col--num">{dealValue(d) || 'No value'}</span>
												<span class="row-col row-col--end">{d.stage_name ?? pretty(d.status)}</span>
											</div>
										{/each}
									</div>
								{/if}
							{:else if s.view === 'clients'}
								{@const clients = s.items as ClientItem[]}
								{#if clients.length === 0}
									<p class="sec-empty">No clients yet.</p>
								{:else}
									<div class="rows">
										{#each clients as c (c.id)}
											<div class="row row--two">
												<span class="row-name">{c.name}</span>
												<span class="row-col row-col--end">
													{#if c.status}<span class="chip">{pretty(c.status)}</span>{:else}<span
															class="meta--dim">No status</span
														>{/if}
												</span>
											</div>
										{/each}
									</div>
								{/if}
							{:else}
								<p class="sec-note">Unknown view type "{s.view}".</p>
							{/if}
						</section>
					{/each}
				</div>
			{/if}
		</div>
	{/if}
</div>

<style>
	.page { height: 100%; display: flex; flex-direction: column; background: var(--dbg); color: var(--dt); overflow: hidden; }

	/* topbar */
	.topbar { display: flex; align-items: center; gap: 12px; padding: 14px 20px; border-bottom: 1px solid var(--dbd); flex-shrink: 0; }
	.back-btn { display: inline-flex; align-items: center; justify-content: center; width: 34px; height: 34px; border-radius: 8px; border: 1px solid var(--dbd); background: transparent; color: var(--dt2); cursor: pointer; flex-shrink: 0; }
	.back-btn:hover { background: color-mix(in srgb, var(--dt) 6%, transparent); }
	.title-wrap { display: flex; align-items: center; gap: 10px; flex: 1; min-width: 0; flex-wrap: wrap; }
	.title-wrap h1 { font-size: 1.1rem; font-weight: 680; letter-spacing: -0.02em; margin: 0; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }

	/* body */
	.body { flex: 1; overflow-y: auto; padding: 18px 24px 40px; }
	.counts { display: flex; flex-wrap: wrap; gap: 6px; margin-bottom: 6px; }

	/* chips */
	.chip { display: inline-flex; align-items: center; gap: 4px; font-size: 0.7rem; color: var(--dt3); background: color-mix(in srgb, var(--dt) 6%, transparent); padding: 3px 8px; border-radius: 6px; white-space: nowrap; }
	.chip strong { color: var(--dt2); font-weight: 640; }
	.chip--total { color: var(--dt2); border: 1px solid var(--dbd); background: transparent; }
	.chip--ctx { color: #818cf8; background: color-mix(in srgb, #818cf8 12%, transparent); }

	/* section grid */
	.grid { display: grid; grid-template-columns: 1fr 1fr; gap: 8px 28px; align-items: start; }
	.sec { min-width: 0; }
	.sec--wide { grid-column: 1 / -1; }

	/* section headers (house .d-sec system) */
	.d-sec { font-size: 0.7rem; text-transform: uppercase; letter-spacing: 0.06em; color: var(--dt3); font-weight: 640; margin: 18px 0 8px; padding-bottom: 6px; border-bottom: 1px solid var(--dbd); }
	.d-sec--row { display: flex; align-items: center; justify-content: space-between; gap: 8px; }
	.d-sec--row > span { display: inline-flex; align-items: center; gap: 6px; }
	.c-count { color: var(--dt3); font-weight: 500; text-transform: none; letter-spacing: 0; }
	.lnk { display: inline-flex; align-items: center; gap: 4px; font-size: 0.72rem; font-weight: 560; color: #818cf8; text-decoration: none; text-transform: none; letter-spacing: 0; }
	.lnk:hover { text-decoration: underline; }

	.sec-empty { font-size: 0.82rem; color: var(--dt3); padding: 6px 0; margin: 0; line-height: 1.5; }
	.sec-note { font-size: 0.8rem; color: #f59e0b; padding: 6px 0; margin: 0; }

	/* project cards */
	.cards { display: grid; grid-template-columns: repeat(auto-fill, minmax(220px, 1fr)); gap: 9px; }
	.pcard { border: 1px solid var(--dbd); border-radius: 11px; padding: 12px; background: color-mix(in srgb, var(--dt) 2%, transparent); display: flex; flex-direction: column; gap: 8px; min-width: 0; }
	.pcard:hover { border-color: color-mix(in srgb, var(--dt) 22%, transparent); }
	.pcard-top { display: flex; align-items: flex-start; justify-content: space-between; gap: 8px; }
	.pcard-name { font-size: 0.88rem; font-weight: 580; min-width: 0; overflow-wrap: anywhere; }
	.meta { display: inline-flex; align-items: center; gap: 5px; font-size: 0.74rem; color: var(--dt3); }
	.meta--dim { opacity: 0.7; font-size: 0.74rem; color: var(--dt3); }

	/* generic rows (team / deals / clients / tasks) */
	.rows { display: flex; flex-direction: column; }
	.row { display: grid; grid-template-columns: 1.6fr 1fr 1fr; align-items: center; gap: 12px; padding: 9px 4px; border-bottom: 1px solid var(--dbd); font-size: 0.84rem; }
	.row--two { grid-template-columns: 1fr auto; }
	.row:last-child { border-bottom: none; }
	.row:hover { background: color-mix(in srgb, var(--dt) 2%, transparent); }
	.row-name { font-weight: 560; color: var(--dt); min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
	.row-col { color: var(--dt2); font-size: 0.79rem; min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
	.row-col--end { text-align: right; }
	.row-col--num { font-variant-numeric: tabular-nums; }

	/* task groups */
	.tgroups { display: flex; flex-direction: column; gap: 14px; }
	.tgroup-head { font-size: 0.72rem; font-weight: 620; color: var(--dt2); text-transform: uppercase; letter-spacing: 0.04em; padding: 2px 4px 6px; display: flex; align-items: center; gap: 6px; }

	/* states */
	.center { flex: 1; display: flex; align-items: center; justify-content: center; gap: 8px; color: var(--dt3); font-size: 0.88rem; }
	.empty { flex: 1; display: flex; flex-direction: column; align-items: center; justify-content: center; gap: 10px; text-align: center; color: var(--dt3); padding: 60px 24px; }
	.empty-icon { display: inline-flex; align-items: center; justify-content: center; width: 44px; height: 44px; border-radius: 12px; border: 1px solid var(--dbd); background: color-mix(in srgb, var(--dt) 4%, transparent); color: var(--dt3); }
	.empty-title { font-size: 0.95rem; font-weight: 600; color: var(--dt2); margin: 4px 0 0; }
	.empty-body { font-size: 0.84rem; max-width: 380px; margin: 0; line-height: 1.5; }
	.banner { display: flex; align-items: center; justify-content: space-between; gap: 12px; margin: 12px 20px 0; padding: 10px 14px; border-radius: 10px; font-size: 0.82rem; flex-shrink: 0; }
	.banner--error { background: color-mix(in srgb, #ef4444 12%, transparent); color: #ef4444; }

	/* buttons */
	.btn { display: inline-flex; align-items: center; gap: 7px; padding: 8px 15px; border-radius: 9px; font-size: 0.83rem; font-weight: 580; cursor: pointer; border: 1px solid transparent; white-space: nowrap; font-family: inherit; text-decoration: none; }
	.btn--ghost { background: transparent; border-color: var(--dbd); color: var(--dt2); }
	.btn--ghost:hover { background: color-mix(in srgb, var(--dt) 6%, transparent); }
	.btn--sm { padding: 5px 10px; font-size: 0.76rem; }
	.btn:disabled { opacity: 0.5; cursor: not-allowed; }

	:global(.spin) { animation: spin 0.9s linear infinite; }
	@keyframes spin { to { transform: rotate(360deg); } }

	/* responsive - single column under 900px */
	@media (max-width: 900px) {
		.grid { grid-template-columns: 1fr; }
		.sec--wide { grid-column: auto; }
		.body { padding: 14px 16px 32px; }
		.topbar { padding: 12px 16px; }
	}
	@media (max-width: 480px) {
		.topbar { padding: 10px 12px; }
		.title-wrap h1 { font-size: 1rem; }
		.row { grid-template-columns: 1fr; gap: 2px; padding: 10px 4px; }
		.row--two { grid-template-columns: 1fr auto; }
		.row-col--end { text-align: left; }
		.cards { grid-template-columns: 1fr; }
	}
</style>
