<script lang="ts">
	import { onMount } from 'svelte';
	import { Building2, Loader2, Mail, Trash2, X, Plus, Check } from 'lucide-svelte';
	import {
		listMyOrganizations, createOrganization, updateOrganization,
		listOrgMembers, updateOrgMemberRole, removeOrgMember,
		listOrgInvites, createOrgInvite, revokeOrgInvite,
		type Organization, type OrgMember, type OrgInvite, type OrgRole
	} from '$lib/api/organizations';

	const ORG_ROLES: OrgRole[] = ['admin', 'member'];

	let orgs = $state<Organization[]>([]);
	let activeId = $state<string>('');
	let members = $state<OrgMember[]>([]);
	let invites = $state<OrgInvite[]>([]);
	let loading = $state(true);
	let error = $state<string | null>(null);
	let busyId = $state<string | null>(null);

	let name = $state('');
	let savingName = $state(false);
	let savedName = $state(false);

	let inviteEmail = $state('');
	let inviteRole = $state<'admin' | 'member'>('member');
	let inviting = $state(false);
	let inviteMsg = $state<string | null>(null);

	let newOrgName = $state('');
	let creating = $state(false);

	const active = $derived(orgs.find((o) => o.id === activeId));
	const role = $derived(active?.role ?? 'member');
	const canManage = $derived(['owner', 'admin'].includes(role));

	onMount(load);

	async function load() {
		loading = true;
		error = null;
		try {
			orgs = await listMyOrganizations();
			if (orgs.length && !activeId) activeId = orgs[0].id;
			if (activeId) await loadOrg();
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to load organizations';
		} finally {
			loading = false;
		}
	}

	async function loadOrg() {
		if (!activeId) return;
		name = active?.name ?? '';
		members = await listOrgMembers(activeId);
		if (canManage) {
			try { invites = (await listOrgInvites(activeId)).filter((i) => i.status === 'pending'); } catch { invites = []; }
		} else { invites = []; }
	}

	async function switchOrg(id: string) {
		activeId = id;
		await loadOrg();
	}

	async function saveName() {
		if (!canManage || !name.trim()) return;
		savingName = true; error = null; savedName = false;
		try {
			await updateOrganization(activeId, { name: name.trim() });
			orgs = orgs.map((o) => (o.id === activeId ? { ...o, name: name.trim() } : o));
			savedName = true; setTimeout(() => (savedName = false), 2000);
		} catch (e) { error = e instanceof Error ? e.message : 'Failed to save'; }
		finally { savingName = false; }
	}

	async function createOrg(e: Event) {
		e.preventDefault();
		if (!newOrgName.trim()) return;
		creating = true; error = null;
		try {
			const org = await createOrganization({ name: newOrgName.trim() });
			newOrgName = '';
			orgs = await listMyOrganizations();
			await switchOrg(org.id);
		} catch (e) { error = e instanceof Error ? e.message : 'Failed to create organization'; }
		finally { creating = false; }
	}

	async function sendInvite(e: Event) {
		e.preventDefault();
		if (!inviteEmail.trim()) return;
		inviting = true; inviteMsg = null; error = null;
		try {
			await createOrgInvite(activeId, inviteEmail.trim(), inviteRole);
			inviteMsg = `Invite sent to ${inviteEmail.trim()}`;
			inviteEmail = ''; inviteRole = 'member';
			await loadOrg();
			setTimeout(() => (inviteMsg = null), 3000);
		} catch (e) { error = e instanceof Error ? e.message : 'Failed to send invite'; }
		finally { inviting = false; }
	}

	async function changeRole(m: OrgMember, newRole: string) {
		busyId = m.user_id; error = null;
		try {
			await updateOrgMemberRole(activeId, m.user_id, newRole as OrgRole);
			members = members.map((x) => (x.user_id === m.user_id ? { ...x, role: newRole } : x));
		} catch (e) { error = e instanceof Error ? e.message : 'Failed to update role'; }
		finally { busyId = null; }
	}

	async function kick(m: OrgMember) {
		if (!confirm(`Remove ${m.name || m.email || m.user_id} from this organization?`)) return;
		busyId = m.user_id; error = null;
		try {
			await removeOrgMember(activeId, m.user_id);
			members = members.filter((x) => x.user_id !== m.user_id);
		} catch (e) { error = e instanceof Error ? e.message : 'Failed to remove'; }
		finally { busyId = null; }
	}

	async function pullInvite(inv: OrgInvite) {
		busyId = inv.id;
		try {
			await revokeOrgInvite(activeId, inv.id);
			invites = invites.filter((x) => x.id !== inv.id);
		} catch (e) { error = e instanceof Error ? e.message : 'Failed to revoke'; }
		finally { busyId = null; }
	}

	function initials(m: OrgMember): string {
		return (m.name || m.email || m.user_id).charAt(0).toUpperCase();
	}
</script>

<header class="sec-head">
	<div class="sec-icon"><Building2 size={20} strokeWidth={1.8} /></div>
	<div>
		<h2>Organization</h2>
		<p>Your account. Invite people to the org, then grant them access to specific workspaces.</p>
	</div>
</header>

{#if error}<div class="banner banner--error">{error}</div>{/if}

{#if loading}
	<div class="loading"><Loader2 class="spin" size={18} /> Loading…</div>
{:else}
	{#if orgs.length > 1}
		<div class="org-switch">
			{#each orgs as o}
				<button class="org-chip {o.id === activeId ? 'org-chip--active' : ''}" onclick={() => switchOrg(o.id)}>
					{o.name}{#if o.id === activeId}<Check size={13} strokeWidth={2.5} />{/if}
				</button>
			{/each}
		</div>
	{/if}

	<div class="card">
		<label class="field">
			<span class="field-label">Organization name</span>
			<div class="field-row">
				<input type="text" bind:value={name} disabled={!canManage} />
				{#if canManage}
					<button class="btn btn--primary" onclick={saveName} disabled={savingName || !name.trim()}>
						{#if savingName}<Loader2 class="spin" size={14} />{/if}{savedName ? 'Saved' : 'Save'}
					</button>
				{/if}
			</div>
		</label>
		<div class="meta">
			<div><span>Members</span><strong>{active?.member_count ?? members.length}</strong></div>
			<div><span>Workspaces</span><strong>{active?.workspace_count ?? '—'}</strong></div>
			<div><span>Your role</span><strong class="cap">{role}</strong></div>
		</div>
	</div>

	{#if canManage}
		<form class="invite-bar" onsubmit={sendInvite}>
			<div class="invite-input">
				<Mail size={15} class="invite-mail" />
				<input type="email" placeholder="person@company.com" bind:value={inviteEmail} required />
			</div>
			<select bind:value={inviteRole} aria-label="Org role">
				{#each ORG_ROLES as r}<option value={r}>{r}</option>{/each}
			</select>
			<button class="btn btn--primary" type="submit" disabled={inviting}>
				{#if inviting}<Loader2 class="spin" size={15} />{/if}Invite to org
			</button>
		</form>
		{#if inviteMsg}<div class="invite-ok">{inviteMsg}</div>{/if}
	{/if}

	<h3 class="subhead">People in this organization</h3>
	<div class="list">
		{#each members as m (m.user_id)}
			<div class="row">
				<div class="who">
					{#if m.image}<img src={m.image} alt="" class="avatar" />{:else}<div class="avatar avatar--fallback">{initials(m)}</div>{/if}
					<div class="who-text">
						<span class="who-name">{m.name || m.email || m.user_id}</span>
						{#if m.email && m.name}<span class="who-sub">{m.email}</span>{/if}
					</div>
				</div>
				<div class="row-actions">
					{#if canManage && m.role !== 'owner'}
						<select value={m.role} disabled={busyId === m.user_id} onchange={(e) => changeRole(m, (e.target as HTMLSelectElement).value)} aria-label="Role">
							{#each ORG_ROLES as r}<option value={r}>{r}</option>{/each}
							{#if !ORG_ROLES.includes(m.role as OrgRole)}<option value={m.role}>{m.role}</option>{/if}
						</select>
						<button class="icon-btn" title="Remove" onclick={() => kick(m)} disabled={busyId === m.user_id}><Trash2 size={15} /></button>
					{:else}
						<span class="role-tag">{m.role}</span>
					{/if}
				</div>
			</div>
		{/each}
	</div>

	{#if canManage && invites.length > 0}
		<h3 class="subhead">Pending invites</h3>
		<div class="list">
			{#each invites as inv (inv.id)}
				<div class="row">
					<div class="who">
						<div class="avatar avatar--invite"><Mail size={15} /></div>
						<div class="who-text"><span class="who-name">{inv.email}</span><span class="who-sub">invited as {inv.role}</span></div>
					</div>
					<button class="icon-btn" title="Revoke" onclick={() => pullInvite(inv)} disabled={busyId === inv.id}><X size={15} /></button>
				</div>
			{/each}
		</div>
	{/if}

	<form class="create-org" onsubmit={createOrg}>
		<input type="text" placeholder="New organization name" bind:value={newOrgName} />
		<button class="btn btn--ghost" type="submit" disabled={creating || !newOrgName.trim()}>
			{#if creating}<Loader2 class="spin" size={14} />{:else}<Plus size={14} />{/if}Create organization
		</button>
	</form>
{/if}

<style>
	.sec-head { display: flex; gap: 14px; align-items: flex-start; margin-bottom: 22px; }
	.sec-icon { width: 40px; height: 40px; border-radius: 11px; flex-shrink: 0; display: flex; align-items: center; justify-content: center; background: color-mix(in srgb, var(--dt) 8%, transparent); color: var(--dt); }
	.sec-head h2 { margin: 0; font-size: 1.15rem; font-weight: 660; letter-spacing: -0.02em; }
	.sec-head p { margin: 4px 0 0; font-size: 0.84rem; color: var(--dt3); max-width: 56ch; }
	.org-switch { display: flex; gap: 8px; flex-wrap: wrap; margin-bottom: 14px; }
	.org-chip { display: inline-flex; align-items: center; gap: 6px; padding: 6px 12px; border-radius: 8px; border: 1px solid var(--dbd); background: transparent; color: var(--dt2); font-size: 0.8rem; cursor: pointer; }
	.org-chip--active { background: color-mix(in srgb, var(--dt) 9%, transparent); color: var(--dt); }
	.card { border: 1px solid var(--dbd); border-radius: 14px; padding: 20px; display: flex; flex-direction: column; gap: 16px; background: color-mix(in srgb, var(--dt) 2%, transparent); margin-bottom: 18px; }
	.field { display: flex; flex-direction: column; gap: 6px; }
	.field-label { font-size: 0.82rem; font-weight: 580; }
	.field-row { display: flex; gap: 8px; }
	.field-row input { flex: 1; }
	input[type="text"], input[type="email"], select { padding: 9px 12px; border-radius: 9px; border: 1px solid var(--dbd); background: var(--dbg); color: var(--dt); font-size: 0.86rem; outline: none; }
	input:focus { border-color: color-mix(in srgb, var(--dt) 40%, transparent); }
	input:disabled { opacity: 0.6; }
	select { cursor: pointer; text-transform: capitalize; }
	.meta { display: flex; gap: 28px; flex-wrap: wrap; }
	.meta div { display: flex; flex-direction: column; gap: 2px; }
	.meta span { font-size: 0.7rem; color: var(--dt3); }
	.meta strong { font-size: 0.9rem; }
	.cap { text-transform: capitalize; }
	.invite-bar { display: flex; gap: 8px; margin-bottom: 8px; }
	.invite-input { position: relative; flex: 1; display: flex; align-items: center; }
	:global(.invite-mail) { position: absolute; left: 11px; color: var(--dt3); }
	.invite-input input { width: 100%; padding-left: 32px; }
	.invite-ok { font-size: 0.78rem; color: #16a34a; margin-bottom: 14px; }
	.btn { display: inline-flex; align-items: center; gap: 7px; padding: 9px 16px; border-radius: 9px; font-size: 0.84rem; font-weight: 560; cursor: pointer; border: 1px solid transparent; white-space: nowrap; }
	.btn--primary { background: var(--dt); color: var(--dbg); }
	.btn--ghost { background: transparent; border-color: var(--dbd); color: var(--dt2); }
	.btn:disabled { opacity: 0.55; cursor: not-allowed; }
	.subhead { font-size: 0.8rem; font-weight: 620; color: var(--dt3); text-transform: uppercase; letter-spacing: 0.06em; margin: 24px 0 10px; }
	.list { display: flex; flex-direction: column; border: 1px solid var(--dbd); border-radius: 12px; overflow: hidden; }
	.row { display: flex; align-items: center; justify-content: space-between; gap: 12px; padding: 11px 14px; }
	.row:not(:last-child) { border-bottom: 1px solid var(--dbd); }
	.who { display: flex; align-items: center; gap: 11px; min-width: 0; }
	.avatar { width: 34px; height: 34px; border-radius: 50%; object-fit: cover; flex-shrink: 0; }
	.avatar--fallback { display: flex; align-items: center; justify-content: center; background: linear-gradient(135deg, #6366f1, #8b5cf6); color: #fff; font-size: 0.82rem; font-weight: 650; }
	.avatar--invite { display: flex; align-items: center; justify-content: center; background: color-mix(in srgb, var(--dt) 8%, transparent); color: var(--dt3); }
	.who-text { display: flex; flex-direction: column; min-width: 0; }
	.who-name { font-size: 0.87rem; font-weight: 550; color: var(--dt); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
	.who-sub { font-size: 0.74rem; color: var(--dt3); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
	.row-actions { display: flex; align-items: center; gap: 8px; flex-shrink: 0; }
	.role-tag { font-size: 0.78rem; color: var(--dt2); text-transform: capitalize; padding: 4px 10px; border-radius: 7px; background: color-mix(in srgb, var(--dt) 7%, transparent); }
	.icon-btn { display: flex; align-items: center; justify-content: center; width: 30px; height: 30px; border-radius: 8px; border: none; background: transparent; color: var(--dt3); cursor: pointer; }
	.icon-btn:hover { background: color-mix(in srgb, #ef4444 12%, transparent); color: #ef4444; }
	.create-org { display: flex; gap: 8px; margin-top: 24px; padding-top: 18px; border-top: 1px solid var(--dbd); }
	.create-org input { flex: 1; }
	.banner { padding: 11px 14px; border-radius: 10px; font-size: 0.83rem; margin-bottom: 16px; }
	.banner--error { background: color-mix(in srgb, #ef4444 12%, transparent); color: #ef4444; }
	.loading { display: flex; align-items: center; gap: 8px; color: var(--dt3); font-size: 0.85rem; padding: 20px 0; }
	:global(.spin) { animation: spin 0.9s linear infinite; }
	@keyframes spin { to { transform: rotate(360deg); } }

	@media (max-width: 768px) {
		.field-row { flex-wrap: wrap; }
		.field-row input { width: 100%; flex-basis: 100%; }
		.field-row .btn { min-height: 40px; }

		.invite-bar { flex-wrap: wrap; }
		.invite-input { width: 100%; flex-basis: 100%; }
		.invite-bar select { flex: 1; min-width: 0; min-height: 40px; }
		.invite-bar .btn { flex: 1; justify-content: center; min-height: 40px; }

		.row { flex-wrap: wrap; align-items: flex-start; gap: 8px; }
		.who { flex: 1; min-width: 0; }
		.row-actions { width: 100%; justify-content: flex-end; }
		.row-actions select { min-height: 40px; flex: 1; min-width: 0; }
		.icon-btn { width: 40px; height: 40px; }

		.create-org { flex-wrap: wrap; }
		.create-org input { width: 100%; flex-basis: 100%; }
		.create-org .btn { min-height: 40px; }
	}

	@media (max-width: 480px) {
		.row { padding: 10px 12px; }
	}
</style>
