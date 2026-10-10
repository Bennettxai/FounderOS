<script lang="ts">
	import { onMount } from 'svelte';
	import { Boxes, Check, Loader2, Globe, Lock } from 'lucide-svelte';
	import {
		getWorkspaceModules,
		setModuleShareScope,
		type WorkspaceModuleItem,
		type ModuleShareScope
	} from '$lib/api/modules';
	import { getWorkspace, updateWorkspace } from '$lib/api/workspaces';
	import { currentWorkspace, workspaces } from '$lib/stores/workspaces';
	import {
		getEnabledModuleIds,
		getModuleCatalog,
		getWorkspaceModuleProfileOptions,
		resolveWorkspaceModuleProfile,
		type ModuleProfile
	} from '$lib/config/workspaceModules';

	let { workspaceId, canManage }: { workspaceId: string; canManage: boolean } = $props();

	let items = $state<WorkspaceModuleItem[]>([]);
	let loading = $state(true);
	let error = $state<string | null>(null);
	let busyId = $state<string | null>(null);
	let toastMsg = $state<string | null>(null);
	let workspaceSettings = $state<Record<string, unknown>>({});
	let selectedProfile = $state<ModuleProfile>('primitive');
	let enabledBuiltinIds = $state<string[]>([]);
	let savingBuiltins = $state(false);
	let builtinsDirty = $state(false);

	const moduleCatalog = getModuleCatalog();
	const profileOptions = getWorkspaceModuleProfileOptions();

	onMount(load);

	async function load() {
		loading = true;
		error = null;
		try {
			const [moduleItems, workspace] = await Promise.all([
				getWorkspaceModules(workspaceId),
				getWorkspace(workspaceId)
			]);
			items = moduleItems;
			workspaceSettings = workspace.settings ?? {};
			selectedProfile = resolveWorkspaceModuleProfile(workspaceSettings);
			enabledBuiltinIds = getEnabledModuleIds(workspaceSettings);
			builtinsDirty = false;
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to load modules';
		} finally {
			loading = false;
		}
	}

	function applyProfile(profile: ModuleProfile) {
		selectedProfile = profile;
		enabledBuiltinIds = getEnabledModuleIds({ module_profile: profile });
		builtinsDirty = true;
	}

	function toggleBuiltin(id: string) {
		if (!canManage || id === 'dashboard') return;
		enabledBuiltinIds = enabledBuiltinIds.includes(id)
			? enabledBuiltinIds.filter((moduleId) => moduleId !== id)
			: [...enabledBuiltinIds, id];
		builtinsDirty = true;
	}

	async function saveBuiltins() {
		if (!canManage || !builtinsDirty || savingBuiltins) return;
		savingBuiltins = true;
		error = null;
		try {
			const settings = {
				...workspaceSettings,
				module_profile: selectedProfile,
				enabled_builtin_modules: [...new Set(['dashboard', ...enabledBuiltinIds])]
			};
			const updated = await updateWorkspace(workspaceId, { settings });
			workspaceSettings = updated.settings ?? settings;
			enabledBuiltinIds = getEnabledModuleIds(workspaceSettings);
			currentWorkspace.update((workspace) => workspace?.id === updated.id ? updated : workspace);
			workspaces.update((all) => all.map((workspace) => workspace.id === updated.id ? updated : workspace));
			builtinsDirty = false;
			toastMsg = 'Workspace modules updated';
			setTimeout(() => (toastMsg = null), 3000);
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to update workspace modules';
		} finally {
			savingBuiltins = false;
		}
	}

	async function toggleScope(item: WorkspaceModuleItem) {
		if (!canManage || busyId) return;
		const next: ModuleShareScope =
			item.share_scope === 'organization' ? 'workspace' : 'organization';
		busyId = item.id;
		error = null;
		try {
			await setModuleShareScope(item.id, next);
			items = items.map((x) => (x.id === item.id ? { ...x, share_scope: next } : x));
			toastMsg = `${item.name} is now ${next === 'organization' ? 'shared across the organization' : 'private to this workspace'}`;
			setTimeout(() => (toastMsg = null), 3000);
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to update scope';
		} finally {
			busyId = null;
		}
	}

	const customModules = $derived(items.filter((m) => m.is_custom));
</script>

<header class="sec-head">
	<div class="sec-icon"><Boxes size={20} strokeWidth={1.8} /></div>
	<div>
		<h2>Modules</h2>
		<p>Choose the built-in operating modules for this workspace and manage custom module sharing.</p>
	</div>
</header>

{#if error}<div class="banner banner--error">{error}</div>{/if}
{#if toastMsg}<div class="banner banner--ok">{toastMsg}</div>{/if}

{#if loading}
	<div class="loading"><Loader2 class="spin" size={18} /> Loading modules…</div>
{:else}
	<div class="profile-block">
		<div class="profile-head">
			<div>
				<h3>Workspace profile</h3>
				<p>Start from a known operating model, then adjust individual modules below.</p>
			</div>
			{#if canManage}
				<button class="btn btn--primary" onclick={saveBuiltins} disabled={!builtinsDirty || savingBuiltins}>
					{#if savingBuiltins}<Loader2 class="spin" size={14} />{:else}<Check size={14} />{/if}
					Save modules
				</button>
			{/if}
		</div>
		<div class="profiles" role="radiogroup" aria-label="Workspace module profile">
			{#each profileOptions as profile}
				<button
					class="profile-option"
					class:profile-option--active={selectedProfile === profile.value}
					onclick={() => applyProfile(profile.value)}
					disabled={!canManage}
					role="radio"
					aria-checked={selectedProfile === profile.value}
				>
					<span>{profile.label}</span>
					<small>{profile.description}</small>
				</button>
			{/each}
		</div>
	</div>

	<h3 class="subhead">Built-in modules</h3>
	<div class="module-grid">
		{#each moduleCatalog as module (module.id)}
			{@const checked = enabledBuiltinIds.includes(module.id)}
			<button
				class="builtin-card"
				class:builtin-card--enabled={checked}
				onclick={() => toggleBuiltin(module.id)}
				disabled={!canManage || module.id === 'dashboard'}
				aria-pressed={checked}
			>
				<span class="builtin-check">{#if checked}<Check size={13} strokeWidth={2.5} />{/if}</span>
				<span class="builtin-copy"><strong>{module.label}</strong><small>{module.group}</small></span>
			</button>
		{/each}
	</div>

	{#if customModules.length > 0}
		<h3 class="subhead">Custom modules</h3>
		<div class="list">
			{#each customModules as m (m.id)}
				<div class="row">
					<div class="mod-info">
						<span class="mod-name">{m.name}</span>
						<span class="mod-key">{m.key}</span>
					</div>
					<div class="row-right">
						<div class="scope-label">
							{#if m.share_scope === 'organization'}
								<Globe size={13} strokeWidth={2} class="scope-icon scope-icon--org" />
								<span>Shared across organization</span>
							{:else}
								<Lock size={13} strokeWidth={2} class="scope-icon scope-icon--ws" />
								<span>Private to this workspace</span>
							{/if}
						</div>
						{#if canManage}
							<button
								class="btn btn--ghost btn--sm"
								onclick={() => toggleScope(m)}
								disabled={busyId === m.id}
								aria-label="Toggle sharing scope for {m.name}"
							>
								{#if busyId === m.id}
									<Loader2 class="spin" size={13} />
								{:else if m.share_scope === 'organization'}
									Make private
								{:else}
									Share with org
								{/if}
							</button>
						{/if}
					</div>
				</div>
			{/each}
		</div>
	{:else}
		<h3 class="subhead">Custom modules</h3>
		<div class="empty">No custom modules in this workspace yet.</div>
	{/if}

	{#if !canManage}
		<p class="readonly-note">Only owners, admins, and managers can change module sharing.</p>
	{/if}
{/if}

<style>
	.sec-head { display: flex; gap: 14px; align-items: flex-start; margin-bottom: 22px; }
	.sec-icon { width: 40px; height: 40px; border-radius: 11px; flex-shrink: 0; display: flex; align-items: center; justify-content: center; background: color-mix(in srgb, var(--dt) 8%, transparent); color: var(--dt); }
	.sec-head h2 { margin: 0; font-size: 1.15rem; font-weight: 660; letter-spacing: -0.02em; }
	.sec-head p { margin: 4px 0 0; font-size: 0.84rem; color: var(--dt3); max-width: 56ch; }

	.subhead { font-size: 0.8rem; font-weight: 620; color: var(--dt3); text-transform: uppercase; letter-spacing: 0.06em; margin: 24px 0 10px; }
	.subhead:first-of-type { margin-top: 0; }

	.list { display: flex; flex-direction: column; border: 1px solid var(--dbd); border-radius: 12px; overflow: hidden; }
	.row { display: flex; align-items: center; justify-content: space-between; gap: 12px; padding: 11px 14px; }
	.row:not(:last-child) { border-bottom: 1px solid var(--dbd); }
	.row--muted { opacity: 0.65; }

	.mod-info { display: flex; flex-direction: column; gap: 2px; min-width: 0; flex: 1; }
	.mod-name { font-size: 0.88rem; font-weight: 580; color: var(--dt); }
	.mod-key { font-size: 0.72rem; color: var(--dt3); font-family: monospace; }

	.row-right { display: flex; align-items: center; gap: 10px; flex-shrink: 0; }

	.scope-label { display: flex; align-items: center; gap: 5px; font-size: 0.78rem; color: var(--dt2); }
	:global(.scope-icon--org) { color: #6366f1; }
	:global(.scope-icon--ws) { color: var(--dt3); }

	.role-tag { font-size: 0.78rem; color: var(--dt2); text-transform: capitalize; padding: 4px 10px; border-radius: 7px; background: color-mix(in srgb, var(--dt) 7%, transparent); }

	.btn { display: inline-flex; align-items: center; gap: 7px; padding: 9px 16px; border-radius: 9px; font-size: 0.84rem; font-weight: 560; cursor: pointer; border: 1px solid transparent; white-space: nowrap; }
	.btn--ghost { background: transparent; border-color: var(--dbd); color: var(--dt2); }
	.btn--primary { background: var(--dt); color: var(--dbg); }
	.btn--sm { padding: 6px 11px; font-size: 0.78rem; }
	.btn:disabled { opacity: 0.55; cursor: not-allowed; }

	@media (max-width: 768px) {
		.row { flex-wrap: wrap; align-items: flex-start; gap: 8px; }
		.row-right { width: 100%; justify-content: space-between; }
		.btn--sm { min-height: 40px; }
	}

	@media (max-width: 480px) {
		.row { padding: 10px 12px; }
		.scope-label span { font-size: 0.74rem; }
	}

	.empty { font-size: 0.85rem; color: var(--dt3); padding: 20px; border: 1px dashed var(--dbd); border-radius: 12px; }
	.readonly-note { font-size: 0.74rem; color: var(--dt3); margin: 14px 0 0; text-align: right; }

	.banner { padding: 11px 14px; border-radius: 10px; font-size: 0.83rem; margin-bottom: 16px; }
	.banner--error { background: color-mix(in srgb, #ef4444 12%, transparent); color: #ef4444; border: 1px solid color-mix(in srgb, #ef4444 25%, transparent); }
	.banner--ok { background: color-mix(in srgb, #22c55e 12%, transparent); color: #16a34a; border: 1px solid color-mix(in srgb, #22c55e 25%, transparent); }

	.loading { display: flex; align-items: center; gap: 8px; color: var(--dt3); font-size: 0.85rem; padding: 20px 0; }
	.profile-block { padding: 16px; border: 1px solid var(--dbd); border-radius: 8px; background: color-mix(in srgb, var(--dt) 2%, transparent); }
	.profile-head { display: flex; align-items: flex-start; justify-content: space-between; gap: 16px; margin-bottom: 14px; }
	.profile-head h3 { margin: 0; font-size: 0.9rem; }
	.profile-head p { margin: 4px 0 0; color: var(--dt3); font-size: 0.78rem; }
	.profiles { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: 8px; }
	.profile-option { min-width: 0; padding: 11px; border: 1px solid var(--dbd); border-radius: 7px; background: var(--dbg); color: var(--dt); text-align: left; cursor: pointer; }
	.profile-option span { display: block; font-size: 0.82rem; font-weight: 620; }
	.profile-option small { display: block; margin-top: 4px; color: var(--dt3); font-size: 0.7rem; line-height: 1.35; }
	.profile-option--active { border-color: var(--dt); box-shadow: inset 0 0 0 1px var(--dt); }
	.module-grid { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: 7px; }
	.builtin-card { display: flex; align-items: center; gap: 9px; min-width: 0; padding: 9px 10px; border: 1px solid var(--dbd); border-radius: 7px; background: transparent; color: var(--dt); text-align: left; cursor: pointer; }
	.builtin-card--enabled { background: color-mix(in srgb, var(--dt) 5%, transparent); border-color: color-mix(in srgb, var(--dt) 35%, var(--dbd)); }
	.builtin-check { width: 17px; height: 17px; display: grid; place-items: center; flex: 0 0 17px; border: 1px solid var(--dbd); border-radius: 4px; }
	.builtin-card--enabled .builtin-check { background: var(--dt); color: var(--dbg); border-color: var(--dt); }
	.builtin-copy { min-width: 0; }
	.builtin-copy strong, .builtin-copy small { display: block; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
	.builtin-copy strong { font-size: 0.79rem; font-weight: 580; }
	.builtin-copy small { margin-top: 1px; color: var(--dt3); font-size: 0.67rem; }
	:global(.spin) { animation: spin 0.9s linear infinite; }
	@keyframes spin { to { transform: rotate(360deg); } }
	@media (max-width: 900px) { .profiles, .module-grid { grid-template-columns: repeat(2, minmax(0, 1fr)); } }
	@media (max-width: 560px) { .profiles, .module-grid { grid-template-columns: 1fr; } .profile-head { flex-direction: column; } }
</style>
