<script lang="ts">
	import { onMount } from 'svelte';
	import { Shield, Loader2 } from 'lucide-svelte';
	import { listRoles, type WorkspaceRole } from '$lib/api/workspace-admin';

	let { workspaceId }: { workspaceId: string } = $props();

	let roles = $state<WorkspaceRole[]>([]);
	let loading = $state(true);
	let error = $state<string | null>(null);

	onMount(async () => {
		try {
			roles = (await listRoles(workspaceId)).sort(
				(a, b) => (b.hierarchy_level ?? 0) - (a.hierarchy_level ?? 0)
			);
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to load roles';
		} finally {
			loading = false;
		}
	});

	function permCount(r: WorkspaceRole): number {
		const p = r.permissions || {};
		return Object.values(p).filter((v) => v === true).length;
	}
</script>

<header class="sec-head">
	<div class="sec-icon"><Shield size={20} strokeWidth={1.8} /></div>
	<div>
		<h2>Roles &amp; permissions</h2>
		<p>The role hierarchy for this workspace. Assign roles to people in the Members tab.</p>
	</div>
</header>

{#if error}<div class="banner banner--error">{error}</div>{/if}

{#if loading}
	<div class="loading"><Loader2 class="spin" size={18} /> Loading roles…</div>
{:else if roles.length === 0}
	<div class="empty">No roles defined yet. Default roles are seeded when the workspace is created.</div>
{:else}
	<div class="grid">
		{#each roles as r (r.id)}
			<div class="role-card" style={r.color ? `--accent:${r.color}` : ''}>
				<div class="role-top">
					<span class="role-name">{r.display_name || r.name}</span>
					{#if r.is_default}<span class="chip">default</span>{/if}
					{#if r.is_system}<span class="chip chip--muted">system</span>{/if}
				</div>
				{#if r.description}<p class="role-desc">{r.description}</p>{/if}
				<div class="role-foot">
					<span class="level">Level {r.hierarchy_level}</span>
					<span class="perms">{permCount(r)} permissions</span>
				</div>
			</div>
		{/each}
	</div>
{/if}

<style>
	.sec-head { display: flex; gap: 14px; align-items: flex-start; margin-bottom: 22px; }
	.sec-icon { width: 40px; height: 40px; border-radius: 11px; flex-shrink: 0; display: flex; align-items: center; justify-content: center; background: color-mix(in srgb, var(--dt) 8%, transparent); color: var(--dt); }
	.sec-head h2 { margin: 0; font-size: 1.15rem; font-weight: 660; letter-spacing: -0.02em; }
	.sec-head p { margin: 4px 0 0; font-size: 0.84rem; color: var(--dt3); }
	.grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(220px, 1fr)); gap: 12px; }

	@media (max-width: 480px) {
		.grid { grid-template-columns: 1fr; }
	}
	.role-card { border: 1px solid var(--dbd); border-radius: 12px; padding: 14px 16px; border-left: 3px solid var(--accent, color-mix(in srgb, var(--dt) 30%, transparent)); }
	.role-top { display: flex; align-items: center; gap: 8px; }
	.role-name { font-size: 0.92rem; font-weight: 620; text-transform: capitalize; }
	.chip { font-size: 0.62rem; text-transform: uppercase; letter-spacing: 0.05em; padding: 2px 7px; border-radius: 6px; background: color-mix(in srgb, #22c55e 16%, transparent); color: #16a34a; font-weight: 650; }
	.chip--muted { background: color-mix(in srgb, var(--dt) 10%, transparent); color: var(--dt3); }
	.role-desc { margin: 8px 0 0; font-size: 0.78rem; color: var(--dt3); line-height: 1.45; }
	.role-foot { display: flex; justify-content: space-between; margin-top: 12px; font-size: 0.72rem; color: var(--dt3); }
	.empty { font-size: 0.85rem; color: var(--dt3); padding: 20px; border: 1px dashed var(--dbd); border-radius: 12px; }
	.banner { padding: 11px 14px; border-radius: 10px; font-size: 0.83rem; margin-bottom: 16px; }
	.banner--error { background: color-mix(in srgb, #ef4444 12%, transparent); color: #ef4444; }
	.loading { display: flex; align-items: center; gap: 8px; color: var(--dt3); font-size: 0.85rem; padding: 20px 0; }
	:global(.spin) { animation: spin 0.9s linear infinite; }
	@keyframes spin { to { transform: rotate(360deg); } }
</style>
