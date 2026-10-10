<script lang="ts">
	import { onMount } from 'svelte';
	import { notificationStore, type Notification } from '$lib/stores/notifications';
	import { currentWorkspace } from '$lib/stores/workspaces';
	import { goto } from '$app/navigation';
	import { Inbox, Loader2, Check, CheckCheck, Trash2, ShieldCheck } from 'lucide-svelte';

	const notifications = notificationStore.notifications;
	const connected = notificationStore.isConnected;
	const connError = notificationStore.connectionError;

	let loading = $state(true);
	let error = $state<string | null>(null);
	let filter = $state<'all' | 'unread'>('all');
	let busyId = $state<string | null>(null);

	// Reload notifications when workspace changes
	let wsId = $state<string | null | undefined>(null);
	$effect(() => {
		const id = $currentWorkspace?.id ?? null;
		if (id !== wsId) { wsId = id; fetchInbox(); }
	});

	onMount(fetchInbox);

	async function fetchInbox() {
		loading = true; error = null;
		try {
			await notificationStore.fetchNotifications(50, 0);
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to load inbox';
		} finally {
			loading = false;
		}
	}

	const items = $derived.by(() => {
		const list = ($notifications ?? []).filter((n) => !n.workspace_id || n.workspace_id === wsId);
		return filter === 'unread' ? list.filter((n) => !n.is_read) : list;
	});
	const unread = $derived(items.filter((n) => !n.is_read).length);

	async function open(n: Notification) {
		if (!n.is_read) { await notificationStore.markAsRead(n.id); }
		const link = routeFor(n);
		if (link) goto(link);
	}
	function routeFor(n: Notification): string | null {
		const href = n.metadata?.href;
		if (typeof href === 'string' && href.startsWith('/')) return href;
		if (!n.entity_type) return null;
		switch (n.entity_type) {
			case 'project': return '/projects';
			case 'task': return '/tasks';
			case 'context': case 'knowledge': return '/knowledge';
			case 'calendar_event': return '/calendar';
			case 'client': return '/relationships';
			default: return null;
		}
	}
	function actionLabel(n: Notification): string | null {
		const label = n.metadata?.action_label;
		return typeof label === 'string' ? label : null;
	}
	async function markRead(n: Notification, e: Event) {
		e.stopPropagation(); busyId = n.id;
		try { await notificationStore.markAsRead(n.id); } finally { busyId = null; }
	}
	async function remove(n: Notification, e: Event) {
		e.stopPropagation(); busyId = n.id;
		try { await notificationStore.deleteNotification(n.id); } finally { busyId = null; }
	}
	async function markAll() {
		try { await notificationStore.markAllAsRead(); } catch (e) { error = e instanceof Error ? e.message : 'Failed'; }
	}
	function initials(n: Notification): string { return (n.sender_name || n.title || '?').charAt(0).toUpperCase(); }
	function ago(s: string): string {
		try {
			const diff = (Date.now() - new Date(s).getTime()) / 1000;
			if (diff < 60) return 'just now';
			if (diff < 3600) return `${Math.floor(diff / 60)}m`;
			if (diff < 86400) return `${Math.floor(diff / 3600)}h`;
			return new Date(s).toLocaleDateString(undefined, { month: 'short', day: 'numeric' });
		} catch { return ''; }
	}
	const PRIORITY: Record<string, string> = { urgent: '#f87171', high: '#fb923c', normal: '', low: '' };
</script>

<svelte:head><title>Inbox - BusinessOS</title></svelte:head>

<div class="ibx-root">
	<header class="topbar">
		<div class="title-wrap">
			<h1>Inbox</h1>
			{#if unread > 0}<span class="count">{unread}</span>{/if}
			{#if $connected}
				<span class="conn conn--live" title="Live updates connected"><span class="conn-dot"></span>Live</span>
			{:else if $connError}
				<span class="conn conn--off" title={$connError}><span class="conn-dot"></span>Reconnecting</span>
			{/if}
		</div>
		<div class="tools">
			<div class="seg">
				<button class:active={filter === 'all'} onclick={() => (filter = 'all')}>All</button>
				<button class:active={filter === 'unread'} onclick={() => (filter = 'unread')}>Unread</button>
			</div>
			<button class="btn btn--ghost" onclick={markAll} disabled={unread === 0}><CheckCheck size={15} />Mark all read</button>
		</div>
	</header>

	{#if error}<div class="banner banner--error">{error}</div>{/if}

	{#if loading}
		<div class="loading"><Loader2 class="spin" size={20} /> Loading inbox…</div>
	{:else if items.length === 0}
		<div class="empty"><Inbox size={28} strokeWidth={1.4} /><p>{filter === 'unread' ? 'No unread items.' : 'Inbox zero. Nothing needs your attention.'}</p></div>
	{:else}
		<div class="list">
			{#each items as n (n.id)}
				<div class="row" class:unread={!n.is_read} role="button" tabindex="0" onclick={() => open(n)} onkeydown={(e) => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); open(n); } }}>
					{#if !n.is_read}<span class="dot" style={PRIORITY[n.priority] ? `background:${PRIORITY[n.priority]}` : ''}></span>{:else}<span class="dot dot--read"></span>{/if}
					{#if n.sender_avatar_url}<img src={n.sender_avatar_url} alt="" class="avatar" />{:else}<div class="avatar avatar--fb">{initials(n)}</div>{/if}
					<div class="body">
						<div class="line1"><span class="title" class:strong={!n.is_read}>{n.title}</span><span class="time">{ago(n.created_at)}</span></div>
						{#if n.body}<p class="snippet">{n.body}</p>{/if}
					</div>
					<div class="actions">
						{#if actionLabel(n)}<button class="review-btn" onclick={(e) => { e.stopPropagation(); open(n); }}><ShieldCheck size={14} />{actionLabel(n)}</button>{/if}
						{#if !n.is_read}<span class="icon-btn" role="button" tabindex="0" title="Mark read" onclick={(e) => markRead(n, e)} onkeydown={(e) => e.key === 'Enter' && markRead(n, e)}><Check size={15} /></span>{/if}
						<span class="icon-btn" role="button" tabindex="0" title="Delete" onclick={(e) => remove(n, e)} onkeydown={(e) => e.key === 'Enter' && remove(n, e)}><Trash2 size={14} /></span>
					</div>
				</div>
			{/each}
		</div>
	{/if}
</div>

<style>
	.ibx-root { height: 100%; display: flex; flex-direction: column; background: var(--dbg); color: var(--dt); overflow: hidden; }
	.topbar { display: flex; align-items: center; justify-content: space-between; gap: 16px; padding: 18px 24px; border-bottom: 1px solid var(--dbd); flex-shrink: 0; }
	.title-wrap { display: flex; align-items: center; gap: 10px; }
	.title-wrap h1 { font-size: 1.15rem; font-weight: 680; letter-spacing: -0.02em; margin: 0; }
	.count { font-size: 0.74rem; color: var(--dbg); background: var(--dt); padding: 2px 9px; border-radius: 999px; font-weight: 650; }
	.conn { display: inline-flex; align-items: center; gap: 5px; font-size: 0.68rem; font-weight: 560; padding: 2px 8px 2px 7px; border-radius: 999px; border: 1px solid var(--dbd); color: var(--dt3); letter-spacing: 0.01em; }
	.conn-dot { width: 6px; height: 6px; border-radius: 50%; background: currentColor; flex-shrink: 0; }
	.conn--live { color: #34d399; border-color: color-mix(in srgb, #34d399 30%, transparent); }
	.conn--live .conn-dot { box-shadow: 0 0 0 0 color-mix(in srgb, #34d399 60%, transparent); animation: pulse 2s ease-out infinite; }
	.conn--off { color: #fbbf24; border-color: color-mix(in srgb, #fbbf24 30%, transparent); }
	@keyframes pulse { 0% { box-shadow: 0 0 0 0 color-mix(in srgb, #34d399 55%, transparent); } 70% { box-shadow: 0 0 0 5px transparent; } 100% { box-shadow: 0 0 0 0 transparent; } }
	.tools { display: flex; align-items: center; gap: 10px; }
	.seg { display: flex; border: 1px solid var(--dbd); border-radius: 9px; overflow: hidden; }
	.seg button { padding: 7px 14px; background: transparent; border: none; color: var(--dt3); cursor: pointer; font-size: 0.82rem; }
	.seg button.active { background: color-mix(in srgb, var(--dt) 10%, transparent); color: var(--dt); }
	.btn { display: inline-flex; align-items: center; gap: 7px; padding: 8px 14px; border-radius: 9px; font-size: 0.82rem; font-weight: 560; cursor: pointer; border: 1px solid var(--dbd); background: transparent; color: var(--dt2); }
	.btn:disabled { opacity: 0.5; cursor: not-allowed; }
	.list { flex: 1; overflow-y: auto; }
	.row { display: flex; align-items: flex-start; gap: 12px; width: 100%; text-align: left; padding: 14px 24px; background: transparent; border: none; border-bottom: 1px solid var(--dbd); cursor: pointer; }
	.row:hover { background: color-mix(in srgb, var(--dt) 4%, transparent); }
	.row:focus-visible { outline: none; background: color-mix(in srgb, var(--dt) 6%, transparent); box-shadow: inset 2px 0 0 var(--dt2); }
	.row.unread { background: color-mix(in srgb, var(--dt) 3%, transparent); }
	.dot { width: 8px; height: 8px; border-radius: 50%; background: #6366f1; margin-top: 6px; flex-shrink: 0; }
	.dot--read { background: transparent; }
	.avatar { width: 34px; height: 34px; border-radius: 50%; object-fit: cover; flex-shrink: 0; }
	.avatar--fb { display: flex; align-items: center; justify-content: center; background: linear-gradient(135deg,#6366f1,#8b5cf6); color: #fff; font-size: 0.8rem; font-weight: 650; }
	.body { flex: 1; min-width: 0; }
	.line1 { display: flex; align-items: baseline; justify-content: space-between; gap: 10px; }
	.title { font-size: 0.88rem; color: var(--dt2); }
	.title.strong { color: var(--dt); font-weight: 600; }
	.time { font-size: 0.72rem; color: var(--dt3); flex-shrink: 0; }
	.snippet { margin: 3px 0 0; font-size: 0.8rem; color: var(--dt3); line-height: 1.45; display: -webkit-box; -webkit-line-clamp: 2; line-clamp: 2; -webkit-box-orient: vertical; overflow: hidden; }
	.actions { display: flex; align-items: center; gap: 4px; flex-shrink: 0; }
	.review-btn { display: inline-flex; align-items: center; gap: 6px; border: 1px solid var(--dbd); border-radius: 7px; padding: 7px 10px; background: var(--dt); color: var(--dbg); font-size: 0.72rem; font-weight: 650; cursor: pointer; white-space: nowrap; }
	.icon-btn { display: inline-flex; align-items: center; justify-content: center; width: 28px; height: 28px; border-radius: 7px; color: var(--dt3); cursor: pointer; }
	.icon-btn:hover { background: color-mix(in srgb, var(--dt) 10%, transparent); color: var(--dt); }
	.loading, .empty { flex: 1; display: flex; flex-direction: column; align-items: center; justify-content: center; gap: 12px; color: var(--dt3); }
	.loading { flex-direction: row; gap: 8px; }
	.banner { margin: 16px 24px 0; padding: 11px 14px; border-radius: 10px; font-size: 0.83rem; }
	.banner--error { background: color-mix(in srgb, #ef4444 12%, transparent); color: #ef4444; }
	:global(.spin) { animation: spin 0.9s linear infinite; }
	@keyframes spin { to { transform: rotate(360deg); } }
	@media (prefers-reduced-motion: reduce) {
		.conn--live .conn-dot { animation: none; }
		:global(.spin) { animation: none; }
	}

	/* --- mobile: 768px --- */
	@media (max-width: 768px) {
		.topbar { flex-wrap: wrap; gap: 10px; padding: 14px 16px; }
		.tools { flex-wrap: wrap; gap: 8px; }
		.banner { margin: 12px 16px 0; }
		/* rows: full-width, ensure 44px tap target */
		.row { padding: 12px 16px; min-height: 44px; }
	}

	/* --- mobile: 480px --- */
	@media (max-width: 480px) {
		.topbar { padding: 12px 14px; }
		.tools { flex-direction: column; align-items: stretch; }
		.seg { align-self: flex-start; }
		.btn { width: 100%; justify-content: center; min-height: 44px; }

		/* rows: full-width tappable, tighter padding */
		.row { padding: 12px 14px; min-height: 48px; gap: 10px; }
		/* prevent snippet from squashing on small screens */
		.body { max-width: calc(100vw - 110px); }
	}
</style>
