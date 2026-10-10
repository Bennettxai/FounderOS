<script lang="ts">
	import { onMount } from 'svelte';
	import { api } from '$lib/api';
	import { currentWorkspace } from '$lib/stores/workspaces';
	import type { TeamMemberListResponse, TeamMemberStatus } from '$lib/api/team/types';
	import { UsersRound, Search, Loader2, Settings, Mail, FolderKanban, CheckCircle2 } from 'lucide-svelte';

	type StatusFilter = 'all' | TeamMemberStatus;

	const STATUS_META: Record<TeamMemberStatus, { label: string; color: string }> = {
		available: { label: 'Available', color: '#34d399' },
		busy: { label: 'Busy', color: '#facc15' },
		overloaded: { label: 'Overloaded', color: '#fb923c' },
		ooo: { label: 'Out of office', color: '#6b7280' }
	};
	const FILTERS: { id: StatusFilter; label: string }[] = [
		{ id: 'all', label: 'All' },
		{ id: 'available', label: 'Available' },
		{ id: 'busy', label: 'Busy' },
		{ id: 'overloaded', label: 'Overloaded' },
		{ id: 'ooo', label: 'OOO' }
	];

	let members = $state<TeamMemberListResponse[]>([]);
	let loading = $state(true);
	let error = $state<string | null>(null);
	let filter = $state<StatusFilter>('all');
	let query = $state('');

	// Reload when workspace changes
	let wsId = $state<string | null | undefined>(null);
	$effect(() => {
		const id = $currentWorkspace?.id ?? null;
		if (id !== wsId) { wsId = id; load(); }
	});

	onMount(load);

	async function load() {
		loading = true; error = null;
		try { members = await api.getTeamMembers(); }
		catch (e) { error = e instanceof Error ? e.message : 'Failed to load team members'; }
		finally { loading = false; }
	}

	function countFor(f: StatusFilter): number {
		if (f === 'all') return members.length;
		return members.filter((m) => m.status === f).length;
	}

	const filtered = $derived.by(() => {
		let list = filter === 'all' ? members : members.filter((m) => m.status === filter);
		const q = query.trim().toLowerCase();
		if (q) list = list.filter((m) => m.name.toLowerCase().includes(q));
		return list;
	});

	function initials(name: string): string {
		const parts = name.trim().split(/\s+/).filter(Boolean);
		if (parts.length === 0) return '?';
		if (parts.length === 1) return parts[0].slice(0, 2).toUpperCase();
		return (parts[0][0] + parts[parts.length - 1][0]).toUpperCase();
	}

	function statusMeta(s: string) {
		return STATUS_META[s as TeamMemberStatus] ?? { label: s, color: '#6b7280' };
	}
</script>

<svelte:head><title>Team - BusinessOS</title></svelte:head>

<div class="proj-root">
	<header class="topbar">
		<div class="title-wrap"><h1>Team</h1><span class="count">{filtered.length}</span></div>
		<div class="tools">
			<div class="search"><Search size={15} strokeWidth={2} /><input placeholder="Search by name" bind:value={query} /></div>
			<a class="btn btn--ghost" href="/settings"><Settings size={15} strokeWidth={2} />Manage members</a>
		</div>
	</header>

	{#if error}<div class="banner banner--error">{error}</div>{/if}

	{#if !loading && members.length > 0}
		<div class="filters">
			{#each FILTERS as f (f.id)}
				<button class="fchip" class:active={filter === f.id} onclick={() => (filter = f.id)}>
					{f.label}<span class="fchip-count">{countFor(f.id)}</span>
				</button>
			{/each}
		</div>
	{/if}

	{#if loading}
		<div class="loading"><Loader2 class="spin" size={20} /> Loading team…</div>
	{:else if members.length === 0}
		<div class="empty">
			<UsersRound size={28} strokeWidth={1.4} />
			<p class="empty-title">No team members yet</p>
			<p class="empty-body">Add people to your workspace under Settings → Members. They'll show up here with their role, status, and capacity.</p>
			<a class="btn btn--primary" href="/settings"><Settings size={15} strokeWidth={2} />Open Settings</a>
		</div>
	{:else if filtered.length === 0}
		<div class="empty">
			<UsersRound size={28} strokeWidth={1.4} />
			{#if query.trim()}
				<p class="empty-title">No members match "{query}"</p>
				<button class="btn btn--ghost" onclick={() => (query = '')}>Clear search</button>
			{:else}
				<p class="empty-title">No {filter === 'all' ? '' : statusMeta(filter).label.toLowerCase() + ' '}members</p>
				<button class="btn btn--ghost" onclick={() => (filter = 'all')}>Show all</button>
			{/if}
		</div>
	{:else}
		<div class="grid">
			{#each filtered as m (m.id)}
				<div class="card">
					<div class="card-top">
						<div class="avatar" aria-hidden="true">{initials(m.name)}</div>
						<div class="who">
							<span class="name">{m.name}</span>
							<span class="role">{m.role}</span>
						</div>
						<span class="status-chip" style="color:{statusMeta(m.status).color}; background: color-mix(in srgb, {statusMeta(m.status).color} 12%, transparent)">
							<span class="dot" style="background:{statusMeta(m.status).color}"></span>{statusMeta(m.status).label}
						</span>
					</div>
					<a class="email" href="mailto:{m.email}" title={m.email}><Mail size={12} strokeWidth={2} />{m.email}</a>
					<div class="cap-row">
						<span class="cap-label">Capacity</span>
						<div class="cap-bar"><div class="cap-fill" style="width:{Math.max(0, Math.min(100, m.capacity))}%"></div></div>
						<span class="cap-val">{m.capacity}%</span>
					</div>
					<div class="meta">
						<span class="chip" title="Active projects"><FolderKanban size={11} />{m.active_projects} project{m.active_projects === 1 ? '' : 's'}</span>
						<span class="chip" title="Open tasks"><CheckCircle2 size={11} />{m.open_tasks} open task{m.open_tasks === 1 ? '' : 's'}</span>
					</div>
				</div>
			{/each}
		</div>
	{/if}
</div>

<style>
	.proj-root { height: 100%; display: flex; flex-direction: column; background: var(--dbg); color: var(--dt); overflow: hidden; }
	.topbar { display: flex; align-items: center; justify-content: space-between; gap: 16px; padding: 18px 24px; border-bottom: 1px solid var(--dbd); flex-shrink: 0; }
	.title-wrap { display: flex; align-items: center; gap: 10px; }
	.title-wrap h1 { font-size: 1.15rem; font-weight: 680; letter-spacing: -0.02em; margin: 0; }
	.count { font-size: 0.74rem; color: var(--dt3); background: color-mix(in srgb, var(--dt) 8%, transparent); padding: 2px 9px; border-radius: 999px; }
	.tools { display: flex; align-items: center; gap: 10px; }
	.search { display: flex; align-items: center; gap: 7px; padding: 7px 11px; border: 1px solid var(--dbd); border-radius: 9px; color: var(--dt3); background: color-mix(in srgb, var(--dt) 3%, transparent); }
	.search input { border: 0; outline: 0; background: transparent; color: var(--dt); font-size: 0.82rem; width: 160px; }
	.btn { display: inline-flex; align-items: center; gap: 7px; padding: 8px 15px; border-radius: 9px; font-size: 0.83rem; font-weight: 580; cursor: pointer; border: 1px solid transparent; white-space: nowrap; text-decoration: none; }
	.btn--primary { background: var(--dt); color: var(--dbg); }
	.btn--ghost { background: transparent; border-color: var(--dbd); color: var(--dt2); }
	.filters { display: flex; align-items: center; gap: 8px; padding: 14px 24px 0; flex-wrap: wrap; flex-shrink: 0; }
	.fchip { display: inline-flex; align-items: center; gap: 6px; padding: 5px 12px; border-radius: 999px; border: 1px solid var(--dbd); background: transparent; color: var(--dt2); font-size: 0.78rem; font-weight: 540; cursor: pointer; }
	.fchip.active { background: color-mix(in srgb, var(--dt) 10%, transparent); color: var(--dt); border-color: color-mix(in srgb, var(--dt) 25%, transparent); }
	.fchip-count { font-size: 0.7rem; color: var(--dt3); }
	.fchip.active .fchip-count { color: var(--dt2); }
	.grid { flex: 1; overflow-y: auto; display: grid; grid-template-columns: repeat(auto-fill, minmax(280px, 1fr)); gap: 14px; padding: 18px 24px 24px; align-content: start; }
	.card { border: 1px solid var(--dbd); border-radius: 12px; padding: 16px; background: color-mix(in srgb, var(--dt) 2%, transparent); display: flex; flex-direction: column; gap: 12px; }
	.card-top { display: flex; align-items: flex-start; gap: 11px; }
	.avatar { width: 40px; height: 40px; border-radius: 10px; flex-shrink: 0; display: flex; align-items: center; justify-content: center; font-size: 0.85rem; font-weight: 640; letter-spacing: 0.02em; color: var(--dt); background: color-mix(in srgb, var(--dt) 9%, transparent); border: 1px solid var(--dbd); }
	.who { flex: 1; min-width: 0; display: flex; flex-direction: column; gap: 2px; }
	.name { font-size: 0.9rem; font-weight: 600; color: var(--dt); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
	.role { font-size: 0.76rem; color: var(--dt3); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
	.status-chip { display: inline-flex; align-items: center; gap: 5px; font-size: 0.68rem; font-weight: 580; padding: 3px 9px; border-radius: 999px; flex-shrink: 0; white-space: nowrap; }
	.dot { width: 6px; height: 6px; border-radius: 50%; }
	.email { display: inline-flex; align-items: center; gap: 6px; font-size: 0.78rem; color: var(--dt3); text-decoration: none; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; min-width: 0; }
	.email:hover { color: var(--dt2); }
	.cap-row { display: flex; align-items: center; gap: 9px; }
	.cap-label { font-size: 0.7rem; text-transform: uppercase; letter-spacing: 0.05em; color: var(--dt3); flex-shrink: 0; }
	.cap-bar { flex: 1; height: 5px; border-radius: 999px; background: color-mix(in srgb, var(--dt) 8%, transparent); overflow: hidden; }
	.cap-fill { height: 100%; border-radius: 999px; background: color-mix(in srgb, var(--dt) 55%, transparent); }
	.cap-val { font-size: 0.74rem; color: var(--dt2); font-variant-numeric: tabular-nums; flex-shrink: 0; }
	.meta { display: flex; flex-wrap: wrap; gap: 6px; }
	.chip { display: inline-flex; align-items: center; gap: 4px; font-size: 0.7rem; color: var(--dt3); background: color-mix(in srgb, var(--dt) 6%, transparent); padding: 3px 8px; border-radius: 6px; }
	.loading, .empty { flex: 1; display: flex; flex-direction: column; align-items: center; justify-content: center; gap: 12px; color: var(--dt3); padding: 24px; text-align: center; }
	.loading { flex-direction: row; gap: 8px; }
	.empty-title { font-size: 0.95rem; font-weight: 600; color: var(--dt2); margin: 0; }
	.empty-body { font-size: 0.83rem; color: var(--dt3); max-width: 380px; margin: 0; line-height: 1.5; }
	.banner { margin: 16px 24px 0; padding: 11px 14px; border-radius: 10px; font-size: 0.83rem; flex-shrink: 0; }
	.banner--error { background: color-mix(in srgb, #ef4444 12%, transparent); color: #ef4444; }
	:global(.spin) { animation: spin 0.9s linear infinite; }
	@keyframes spin { to { transform: rotate(360deg); } }

	@media (max-width: 768px) {
		.topbar { flex-wrap: wrap; gap: 10px; padding: 14px 16px; }
		.tools { flex-wrap: wrap; gap: 8px; width: 100%; }
		.search { flex: 1 1 auto; min-width: 0; }
		.search input { width: 100%; min-width: 0; }
		.filters { padding: 12px 16px 0; }
		.grid { padding: 14px 16px 20px; gap: 10px; }
		.banner { margin: 12px 16px 0; }
	}

	@media (max-width: 480px) {
		.topbar { padding: 12px 14px; }
		.tools { flex-direction: column; align-items: stretch; }
		.search { width: 100%; }
		.btn.btn--ghost { justify-content: center; }
		.grid { grid-template-columns: 1fr; padding: 12px 14px 18px; }
	}
</style>
