<script lang="ts">
	import { onMount } from 'svelte';
	import { Users, Loader2, Trash2, Mail, X, CloudUpload } from 'lucide-svelte';
	import {
		listMembers,
		listInvites,
		createInvite,
		revokeInvite,
		updateMemberRole,
		removeMember,
		syncWorkspaceToCloud,
		type WorkspaceMember,
		type WorkspaceInvite,
		type WorkspaceRoleName
	} from '$lib/api/workspace-admin';

	let { workspaceId, role }: { workspaceId: string; role: string } = $props();

	const ROLES: WorkspaceRoleName[] = ['admin', 'manager', 'member', 'viewer', 'guest'];

	const canInvite = $derived(['owner', 'admin', 'manager'].includes(role));
	const canEdit = $derived(['owner', 'admin'].includes(role));

	let members = $state<WorkspaceMember[]>([]);
	let invites = $state<WorkspaceInvite[]>([]);
	let loading = $state(true);
	let error = $state<string | null>(null);
	let busyId = $state<string | null>(null);

	// Cloud sync (owner/admin): push this workspace to the shared cloud so
	// teammates can work in it live and invite links resolve on businessos.dev.
	let syncingCloud = $state(false);
	let syncMsg = $state<string | null>(null);
	async function syncCloud() {
		syncingCloud = true;
		syncMsg = null;
		error = null;
		try {
			const r = await syncWorkspaceToCloud();
			const data = Object.values(r.tables ?? {}).reduce((a, b) => a + b, 0);
			let msg = `Synced to cloud: ${r.members} member(s), ${r.invites} invite(s), ${data} record(s).`;
			if (r.skipped_users?.length) msg += ` Waiting on cloud accounts for: ${r.skipped_users.join(', ')}.`;
			syncMsg = msg;
		} catch (e) {
			error = e instanceof Error ? e.message : 'Cloud sync failed';
		} finally {
			syncingCloud = false;
		}
	}

	// Invite form
	let inviteEmail = $state('');
	let inviteRole = $state<WorkspaceRoleName>('member');
	let inviting = $state(false);
	let inviteMsg = $state<string | null>(null);
	let inviteLink = $state<string | null>(null);
	let copied = $state(false);
	async function copyInviteLink() {
		if (!inviteLink) return;
		try {
			await navigator.clipboard.writeText(inviteLink);
			copied = true;
			setTimeout(() => (copied = false), 1500);
		} catch {
			/* clipboard blocked */
		}
	}

	onMount(load);

	async function load() {
		loading = true;
		error = null;
		try {
			members = await listMembers(workspaceId);
			if (canEdit) {
				try {
					invites = (await listInvites(workspaceId)).filter((i) => i.status === 'pending');
				} catch {
					invites = [];
				}
			}
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to load members';
		} finally {
			loading = false;
		}
	}

	async function sendInvite(e: Event) {
		e.preventDefault();
		if (!inviteEmail.trim()) return;
		inviting = true;
		inviteMsg = null;
		error = null;
		try {
			const invite = await createInvite(workspaceId, inviteEmail.trim(), inviteRole);
			// Email delivery isn't wired yet, so surface a copyable invite link the
			// owner can send directly. The invitee opens it, signs in, and joins.
			inviteLink = `${window.location.origin}/invite/${invite.token}`;
			inviteMsg = `Invite created for ${inviteEmail.trim()}. Copy this link and send it to them:`;
			inviteEmail = '';
			inviteRole = 'member';
			await load();
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to send invite';
		} finally {
			inviting = false;
		}
	}

	async function changeRole(m: WorkspaceMember, newRole: string) {
		busyId = m.user_id;
		error = null;
		try {
			await updateMemberRole(workspaceId, m.user_id, newRole as WorkspaceRoleName);
			members = members.map((x) => (x.user_id === m.user_id ? { ...x, role: newRole } : x));
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to update role';
		} finally {
			busyId = null;
		}
	}

	async function kick(m: WorkspaceMember) {
		if (!confirm(`Remove ${m.name || m.email || m.user_id} from this workspace?`)) return;
		busyId = m.user_id;
		error = null;
		try {
			await removeMember(workspaceId, m.user_id);
			members = members.filter((x) => x.user_id !== m.user_id);
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to remove member';
		} finally {
			busyId = null;
		}
	}

	async function pullInvite(inv: WorkspaceInvite) {
		busyId = inv.id;
		try {
			await revokeInvite(workspaceId, inv.id);
			invites = invites.filter((x) => x.id !== inv.id);
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to revoke';
		} finally {
			busyId = null;
		}
	}

	function initials(m: WorkspaceMember): string {
		const s = m.name || m.email || m.user_id;
		return s.charAt(0).toUpperCase();
	}
</script>

<header class="sec-head">
	<div class="sec-icon"><Users size={20} strokeWidth={1.8} /></div>
	<div class="sec-head-text">
		<h2>Members</h2>
		<p>Invite people and manage who can do what in this workspace.</p>
	</div>
	{#if canEdit}
		<button class="btn btn--ghost sync-btn" onclick={syncCloud} disabled={syncingCloud} title="Push this workspace to the shared cloud so teammates can join + work live">
			{#if syncingCloud}<Loader2 class="spin" size={15} />{:else}<CloudUpload size={15} />{/if}
			{syncingCloud ? 'Syncing…' : 'Sync to cloud'}
		</button>
	{/if}
</header>

{#if error}<div class="banner banner--error">{error}</div>{/if}
{#if syncMsg}<div class="invite-ok">{syncMsg}</div>{/if}

{#if canInvite}
	<form class="invite-bar" onsubmit={sendInvite}>
		<div class="invite-input">
			<Mail size={15} class="invite-mail" />
			<input type="email" placeholder="teammate@company.com" bind:value={inviteEmail} required />
		</div>
		<select bind:value={inviteRole} aria-label="Invite role">
			{#each ROLES as r}<option value={r}>{r}</option>{/each}
		</select>
		<button class="btn btn--primary" type="submit" disabled={inviting}>
			{#if inviting}<Loader2 class="spin" size={15} />{/if}
			Invite
		</button>
	</form>
	{#if inviteMsg}<div class="invite-ok">{inviteMsg}</div>{/if}
	{#if inviteLink}
		<div class="invite-link-row">
			<input class="invite-link-input" readonly value={inviteLink} onclick={(e) => e.currentTarget.select()} />
			<button type="button" class="btn btn--ghost" onclick={copyInviteLink}>{copied ? 'Copied' : 'Copy link'}</button>
		</div>
	{/if}
{/if}

{#if loading}
	<div class="loading"><Loader2 class="spin" size={18} /> Loading members…</div>
{:else}
	<div class="list">
		{#each members as m (m.user_id)}
			<div class="row">
				<div class="who">
					{#if m.image}
						<img src={m.image} alt="" class="avatar" />
					{:else}
						<div class="avatar avatar--fallback">{initials(m)}</div>
					{/if}
					<div class="who-text">
						<span class="who-name">{m.name || m.email || m.user_id}</span>
						{#if m.email && m.name}<span class="who-sub">{m.email}</span>{/if}
						{#if m.status && m.status !== 'active'}<span class="who-status">{m.status}</span>{/if}
					</div>
				</div>
				<div class="row-actions">
					{#if canEdit && m.role !== 'owner'}
						<select
							value={m.role}
							disabled={busyId === m.user_id}
							onchange={(e) => changeRole(m, (e.target as HTMLSelectElement).value)}
							aria-label="Member role"
						>
							{#each ROLES as r}<option value={r}>{r}</option>{/each}
							{#if !ROLES.includes(m.role as WorkspaceRoleName)}<option value={m.role}>{m.role}</option>{/if}
						</select>
						<button class="icon-btn" title="Remove" onclick={() => kick(m)} disabled={busyId === m.user_id}>
							<Trash2 size={15} />
						</button>
					{:else}
						<span class="role-tag">{m.role}</span>
					{/if}
				</div>
			</div>
		{/each}
	</div>

	{#if canEdit && invites.length > 0}
		<h3 class="subhead">Pending invites</h3>
		<div class="list">
			{#each invites as inv (inv.id)}
				<div class="row">
					<div class="who">
						<div class="avatar avatar--invite"><Mail size={15} /></div>
						<div class="who-text">
							<span class="who-name">{inv.email}</span>
							<span class="who-sub">invited as {inv.role}</span>
						</div>
					</div>
					<button class="icon-btn" title="Revoke invite" onclick={() => pullInvite(inv)} disabled={busyId === inv.id}>
						<X size={15} />
					</button>
				</div>
			{/each}
		</div>
	{/if}
{/if}

<style>
	.sec-head { display: flex; gap: 14px; align-items: flex-start; margin-bottom: 22px; }
	.sec-head-text { flex: 1; min-width: 0; }
	.sync-btn { flex-shrink: 0; align-self: center; }
	.sec-icon { width: 40px; height: 40px; border-radius: 11px; flex-shrink: 0; display: flex; align-items: center; justify-content: center; background: color-mix(in srgb, var(--dt) 8%, transparent); color: var(--dt); }
	.sec-head h2 { margin: 0; font-size: 1.15rem; font-weight: 660; letter-spacing: -0.02em; }
	.sec-head p { margin: 4px 0 0; font-size: 0.84rem; color: var(--dt3); }

	.invite-bar { display: flex; gap: 8px; margin-bottom: 8px; }
	.invite-input { position: relative; flex: 1; display: flex; align-items: center; }
	:global(.invite-mail) { position: absolute; left: 11px; color: var(--dt3); }
	.invite-input input {
		width: 100%; padding: 9px 12px 9px 32px; border-radius: 9px;
		border: 1px solid var(--dbd); background: var(--dbg); color: var(--dt); font-size: 0.86rem; outline: none;
	}
	.invite-input input:focus { border-color: color-mix(in srgb, var(--dt) 40%, transparent); }
	select {
		padding: 9px 10px; border-radius: 9px; border: 1px solid var(--dbd);
		background: var(--dbg); color: var(--dt); font-size: 0.83rem; cursor: pointer; text-transform: capitalize;
	}
	.invite-ok { font-size: 0.78rem; color: #16a34a; margin-bottom: 14px; }
	.invite-link-row { display: flex; gap: 8px; margin-bottom: 14px; }
	.invite-link-input { flex: 1; padding: 8px 11px; border-radius: 8px; border: 1px solid var(--dbd); background: color-mix(in srgb, var(--dt) 3%, transparent); color: var(--dt2); font-size: 0.78rem; font-family: ui-monospace, monospace; outline: none; }

	.btn { display: inline-flex; align-items: center; gap: 7px; padding: 9px 16px; border-radius: 9px; font-size: 0.84rem; font-weight: 560; cursor: pointer; border: 1px solid transparent; }
	.btn--primary { background: var(--dt); color: var(--dbg); }
	.btn:disabled { opacity: 0.55; cursor: not-allowed; }

	.list { display: flex; flex-direction: column; border: 1px solid var(--dbd); border-radius: 12px; overflow: hidden; }
	.row { display: flex; align-items: center; justify-content: space-between; gap: 12px; padding: 11px 14px; }
	.row:not(:last-child) { border-bottom: 1px solid var(--dbd); }
	.who { display: flex; align-items: center; gap: 11px; min-width: 0; }
	.avatar { width: 34px; height: 34px; border-radius: 50%; object-fit: cover; flex-shrink: 0; }
	.avatar--fallback { display: flex; align-items: center; justify-content: center; background: linear-gradient(135deg, #6366f1, #8b5cf6); color: #fff; font-size: 0.82rem; font-weight: 650; }
	.avatar--invite { display: flex; align-items: center; justify-content: center; background: color-mix(in srgb, var(--dt) 8%, transparent); color: var(--dt3); }
	.who-text { display: flex; flex-direction: column; min-width: 0; line-height: 1.25; }
	.who-name { font-size: 0.87rem; font-weight: 550; color: var(--dt); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
	.who-sub { font-size: 0.74rem; color: var(--dt3); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
	.who-status { font-size: 0.68rem; color: #f59e0b; text-transform: uppercase; letter-spacing: 0.04em; }
	.row-actions { display: flex; align-items: center; gap: 8px; flex-shrink: 0; }
	.role-tag { font-size: 0.78rem; color: var(--dt2); text-transform: capitalize; padding: 4px 10px; border-radius: 7px; background: color-mix(in srgb, var(--dt) 7%, transparent); }
	.icon-btn { display: flex; align-items: center; justify-content: center; width: 30px; height: 30px; border-radius: 8px; border: none; background: transparent; color: var(--dt3); cursor: pointer; }
	.icon-btn:hover { background: color-mix(in srgb, #ef4444 12%, transparent); color: #ef4444; }
	.icon-btn:disabled { opacity: 0.5; cursor: not-allowed; }

	@media (max-width: 768px) {
		.invite-bar { flex-wrap: wrap; }
		.invite-input { width: 100%; flex-basis: 100%; }
		.invite-bar select { flex: 1; min-width: 0; }
		.invite-bar .btn { flex: 1; justify-content: center; min-height: 40px; }

		.row { flex-wrap: wrap; align-items: flex-start; gap: 8px; }
		.who { flex: 1; min-width: 0; }
		.row-actions { width: 100%; justify-content: flex-end; }
		.row-actions select { min-height: 40px; flex: 1; min-width: 0; }
		.icon-btn { width: 40px; height: 40px; }
	}

	@media (max-width: 480px) {
		.row { padding: 10px 12px; }
		.invite-bar .btn { font-size: 0.82rem; }
	}

	.subhead { font-size: 0.8rem; font-weight: 620; color: var(--dt3); text-transform: uppercase; letter-spacing: 0.06em; margin: 24px 0 10px; }
	.banner { padding: 11px 14px; border-radius: 10px; font-size: 0.83rem; margin-bottom: 16px; }
	.banner--error { background: color-mix(in srgb, #ef4444 12%, transparent); color: #ef4444; }
	.loading { display: flex; align-items: center; gap: 8px; color: var(--dt3); font-size: 0.85rem; padding: 20px 0; }
	:global(.spin) { animation: spin 0.9s linear infinite; }
	@keyframes spin { to { transform: rotate(360deg); } }
</style>
