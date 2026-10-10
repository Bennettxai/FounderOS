<script lang="ts">
	import { onMount } from 'svelte';
	import { UsersRound, Loader2, Plus, Trash2, UserPlus, ArrowRightLeft } from 'lucide-svelte';
	import {
		listTeams, createTeam, deleteTeam, listTeamMembers,
		addTeamMember, removeTeamMember, moveTeamMember,
		type Team, type TeamMember
	} from '$lib/api/teams';
	import { listMembers, type WorkspaceMember } from '$lib/api/workspace-admin';

	let { workspaceId, canManage }: { workspaceId: string; canManage: boolean } = $props();

	let teams = $state<Team[]>([]);
	let wsMembers = $state<WorkspaceMember[]>([]);
	let teamMembers = $state<Record<string, TeamMember[]>>({});
	let expanded = $state<string | null>(null);
	let loading = $state(true);
	let error = $state<string | null>(null);
	let busy = $state<string | null>(null);

	let newName = $state('');
	let creating = $state(false);
	let addPick = $state<Record<string, string>>({});

	onMount(load);

	async function load() {
		loading = true; error = null;
		try {
			[teams, wsMembers] = await Promise.all([listTeams(workspaceId), listMembers(workspaceId)]);
		} catch (e) { error = e instanceof Error ? e.message : 'Failed to load teams'; }
		finally { loading = false; }
	}

	async function toggle(t: Team) {
		if (expanded === t.id) { expanded = null; return; }
		expanded = t.id;
		if (!teamMembers[t.id]) {
			try { teamMembers = { ...teamMembers, [t.id]: await listTeamMembers(t.id) }; }
			catch (e) { error = e instanceof Error ? e.message : 'Failed to load members'; }
		}
	}

	async function refreshTeam(teamId: string) {
		teamMembers = { ...teamMembers, [teamId]: await listTeamMembers(teamId) };
		teams = await listTeams(workspaceId);
	}

	async function create(e: Event) {
		e.preventDefault();
		if (!newName.trim()) return;
		creating = true; error = null;
		try {
			await createTeam(workspaceId, { name: newName.trim() });
			newName = '';
			teams = await listTeams(workspaceId);
		} catch (e) { error = e instanceof Error ? e.message : 'Failed to create team'; }
		finally { creating = false; }
	}

	async function removeTeam(t: Team) {
		if (!confirm(`Delete the "${t.name}" team? Members stay in the workspace.`)) return;
		busy = t.id;
		try { await deleteTeam(t.id); teams = teams.filter((x) => x.id !== t.id); if (expanded === t.id) expanded = null; }
		catch (e) { error = e instanceof Error ? e.message : 'Failed to delete'; }
		finally { busy = null; }
	}

	async function add(teamId: string) {
		const uid = addPick[teamId];
		if (!uid) return;
		busy = teamId;
		try { await addTeamMember(teamId, uid); addPick = { ...addPick, [teamId]: '' }; await refreshTeam(teamId); }
		catch (e) { error = e instanceof Error ? e.message : 'Failed to add'; }
		finally { busy = null; }
	}

	async function remove(teamId: string, uid: string) {
		busy = teamId + uid;
		try { await removeTeamMember(teamId, uid); await refreshTeam(teamId); }
		catch (e) { error = e instanceof Error ? e.message : 'Failed to remove'; }
		finally { busy = null; }
	}

	async function move(fromTeam: string, uid: string, toTeam: string) {
		if (!toTeam) return;
		busy = fromTeam + uid;
		try { await moveTeamMember(fromTeam, uid, toTeam); await refreshTeam(fromTeam); await refreshTeam(toTeam); }
		catch (e) { error = e instanceof Error ? e.message : 'Failed to move'; }
		finally { busy = null; }
	}

	function nameFor(uid: string): string {
		const m = wsMembers.find((x) => x.user_id === uid);
		return m?.name || m?.email || uid;
	}
	function available(teamId: string): WorkspaceMember[] {
		const inTeam = new Set((teamMembers[teamId] ?? []).map((m) => m.user_id));
		return wsMembers.filter((m) => !inTeam.has(m.user_id));
	}
</script>

<header class="sec-head">
	<div class="sec-icon"><UsersRound size={20} strokeWidth={1.8} /></div>
	<div>
		<h2>Teams</h2>
		<p>Group people inside this workspace. Add workspace members to teams and move them between teams.</p>
	</div>
</header>

{#if error}<div class="banner banner--error">{error}</div>{/if}

{#if loading}
	<div class="loading"><Loader2 class="spin" size={18} /> Loading teams…</div>
{:else}
	{#if canManage}
		<form class="create-bar" onsubmit={create}>
			<input type="text" placeholder="New team name (e.g. Design, Sales)" bind:value={newName} />
			<button class="btn btn--primary" type="submit" disabled={creating || !newName.trim()}>
				{#if creating}<Loader2 class="spin" size={15} />{:else}<Plus size={15} />{/if}Create team
			</button>
		</form>
	{/if}

	{#if teams.length === 0}
		<div class="empty">No teams yet. {#if canManage}Create one above to start organizing people.{/if}</div>
	{:else}
		<div class="teams">
			{#each teams as t (t.id)}
				<div class="team">
					<button class="team-head" onclick={() => toggle(t)}>
						<div class="team-dot" style={t.color ? `background:${t.color}` : ''}></div>
						<span class="team-name">{t.name}</span>
						<span class="team-count">{t.member_count ?? 0}</span>
						{#if canManage}
							<span
								class="icon-btn"
								role="button"
								tabindex="0"
								title="Delete team"
								onclick={(e) => { e.stopPropagation(); removeTeam(t); }}
								onkeydown={(e) => { if (e.key === 'Enter') { e.stopPropagation(); removeTeam(t); } }}
							><Trash2 size={14} /></span>
						{/if}
					</button>

					{#if expanded === t.id}
						<div class="team-body">
							{#each (teamMembers[t.id] ?? []) as m (m.user_id)}
								<div class="trow">
									<span class="tname">{m.name || m.email || m.user_id}</span>
									{#if canManage}
										<div class="trow-actions">
											<select aria-label="Move to team" onchange={(e) => { move(t.id, m.user_id, (e.target as HTMLSelectElement).value); (e.target as HTMLSelectElement).value=''; }}>
												<option value="">Move to…</option>
												{#each teams.filter((x) => x.id !== t.id) as dest}<option value={dest.id}>{dest.name}</option>{/each}
											</select>
											<button class="icon-btn" title="Remove from team" onclick={() => remove(t.id, m.user_id)} disabled={busy === t.id + m.user_id}><Trash2 size={14} /></button>
										</div>
									{/if}
								</div>
							{:else}
								<p class="team-empty">No one in this team yet.</p>
							{/each}

							{#if canManage && available(t.id).length > 0}
								<div class="add-row">
									<select bind:value={addPick[t.id]} aria-label="Add member">
										<option value="">Add a workspace member…</option>
										{#each available(t.id) as m}<option value={m.user_id}>{m.name || m.email || m.user_id}</option>{/each}
									</select>
									<button class="btn btn--ghost btn--sm" onclick={() => add(t.id)} disabled={!addPick[t.id] || busy === t.id}><UserPlus size={14} />Add</button>
								</div>
							{/if}
						</div>
					{/if}
				</div>
			{/each}
		</div>
	{/if}
{/if}

<style>
	.sec-head { display: flex; gap: 14px; align-items: flex-start; margin-bottom: 22px; }
	.sec-icon { width: 40px; height: 40px; border-radius: 11px; flex-shrink: 0; display: flex; align-items: center; justify-content: center; background: color-mix(in srgb, var(--dt) 8%, transparent); color: var(--dt); }
	.sec-head h2 { margin: 0; font-size: 1.15rem; font-weight: 660; letter-spacing: -0.02em; }
	.sec-head p { margin: 4px 0 0; font-size: 0.84rem; color: var(--dt3); max-width: 56ch; }
	.create-bar { display: flex; gap: 8px; margin-bottom: 16px; }
	.create-bar input { flex: 1; }
	input[type="text"], select { padding: 9px 12px; border-radius: 9px; border: 1px solid var(--dbd); background: var(--dbg); color: var(--dt); font-size: 0.85rem; outline: none; }
	input:focus { border-color: color-mix(in srgb, var(--dt) 40%, transparent); }
	select { cursor: pointer; }
	.btn { display: inline-flex; align-items: center; gap: 7px; padding: 9px 16px; border-radius: 9px; font-size: 0.84rem; font-weight: 560; cursor: pointer; border: 1px solid transparent; white-space: nowrap; }
	.btn--primary { background: var(--dt); color: var(--dbg); }
	.btn--ghost { background: transparent; border-color: var(--dbd); color: var(--dt2); }
	.btn--sm { padding: 7px 12px; font-size: 0.8rem; }
	.btn:disabled { opacity: 0.55; cursor: not-allowed; }
	.empty { font-size: 0.85rem; color: var(--dt3); padding: 20px; border: 1px dashed var(--dbd); border-radius: 12px; }
	.teams { display: flex; flex-direction: column; gap: 10px; }
	.team { border: 1px solid var(--dbd); border-radius: 12px; overflow: hidden; }
	.team-head { display: flex; align-items: center; gap: 10px; width: 100%; padding: 12px 14px; background: transparent; border: none; cursor: pointer; text-align: left; color: var(--dt); }
	.team-head:hover { background: color-mix(in srgb, var(--dt) 4%, transparent); }
	.team-dot { width: 9px; height: 9px; border-radius: 50%; background: color-mix(in srgb, var(--dt) 35%, transparent); flex-shrink: 0; }
	.team-name { font-size: 0.9rem; font-weight: 600; flex: 1; }
	.team-count { font-size: 0.74rem; color: var(--dt3); background: color-mix(in srgb, var(--dt) 8%, transparent); padding: 2px 9px; border-radius: 999px; }
	.team-body { padding: 6px 14px 14px; border-top: 1px solid var(--dbd); display: flex; flex-direction: column; gap: 6px; }
	.trow { display: flex; align-items: center; justify-content: space-between; gap: 10px; padding: 7px 0; }
	.tname { font-size: 0.85rem; color: var(--dt); min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
	.trow-actions { display: flex; align-items: center; gap: 8px; flex-shrink: 0; }
	.trow-actions select { font-size: 0.78rem; padding: 5px 8px; }
	.team-empty { font-size: 0.8rem; color: var(--dt3); margin: 6px 0; }
	.add-row { display: flex; gap: 8px; margin-top: 8px; padding-top: 10px; border-top: 1px solid var(--dbd); }
	.add-row select { flex: 1; }
	.icon-btn { display: inline-flex; align-items: center; justify-content: center; width: 28px; height: 28px; border-radius: 7px; border: none; background: transparent; color: var(--dt3); cursor: pointer; }
	.icon-btn:hover { background: color-mix(in srgb, #ef4444 12%, transparent); color: #ef4444; }
	.banner { padding: 11px 14px; border-radius: 10px; font-size: 0.83rem; margin-bottom: 16px; }
	.banner--error { background: color-mix(in srgb, #ef4444 12%, transparent); color: #ef4444; }
	.loading { display: flex; align-items: center; gap: 8px; color: var(--dt3); font-size: 0.85rem; padding: 20px 0; }
	:global(.spin) { animation: spin 0.9s linear infinite; }
	@keyframes spin { to { transform: rotate(360deg); } }

	@media (max-width: 768px) {
		.create-bar { flex-wrap: wrap; }
		.create-bar input { width: 100%; flex-basis: 100%; }
		.create-bar .btn { min-height: 40px; }

		/* Team member rows: name on top, actions below */
		.trow { flex-wrap: wrap; align-items: flex-start; }
		.tname { flex-basis: 100%; }
		.trow-actions { width: 100%; justify-content: flex-end; }
		.trow-actions select { flex: 1; min-width: 0; min-height: 40px; font-size: 0.83rem; padding: 8px 10px; }
		.icon-btn { width: 40px; height: 40px; }

		.add-row { flex-wrap: wrap; }
		.add-row select { width: 100%; flex-basis: 100%; min-height: 40px; }
		.add-row .btn { min-height: 40px; }
	}

	@media (max-width: 480px) {
		.team-body { padding: 6px 12px 12px; }
	}
</style>
