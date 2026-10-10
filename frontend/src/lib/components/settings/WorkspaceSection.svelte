<script lang="ts">
	import { onMount } from 'svelte';
	import { Building2, Loader2 } from 'lucide-svelte';
	import { getWorkspace, updateWorkspace, type Workspace } from '$lib/api/workspace-admin';
	import { currentWorkspace } from '$lib/stores/workspaces';

	let { workspaceId, canManage }: { workspaceId: string; canManage: boolean } = $props();

	let ws = $state<Workspace | null>(null);
	let name = $state('');
	let description = $state('');
	let loading = $state(true);
	let saving = $state(false);
	let saved = $state(false);
	let error = $state<string | null>(null);

	onMount(async () => {
		try {
			ws = await getWorkspace(workspaceId);
			name = ws.name ?? '';
			description = ws.description ?? '';
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to load workspace';
		} finally {
			loading = false;
		}
	});

	async function save() {
		if (!canManage) return;
		saving = true;
		error = null;
		saved = false;
		try {
			const updated = await updateWorkspace(workspaceId, {
				name: name.trim(),
				description: description.trim()
			});
			ws = updated;
			// Keep the sidebar/switcher label in sync.
			currentWorkspace.update((w) => (w ? { ...w, name: updated.name } : w));
			saved = true;
			setTimeout(() => (saved = false), 2500);
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to save';
		} finally {
			saving = false;
		}
	}
</script>

<header class="sec-head">
	<div class="sec-icon"><Building2 size={20} strokeWidth={1.8} /></div>
	<div>
		<h2>Workspace</h2>
		<p>The shared container for your team's data, members, and engine.</p>
	</div>
</header>

{#if error}<div class="banner banner--error">{error}</div>{/if}

{#if loading}
	<div class="loading"><Loader2 class="spin" size={18} /> Loading…</div>
{:else if ws}
	<div class="card">
		<label class="field">
			<span class="field-label">Workspace name</span>
			<input type="text" bind:value={name} disabled={!canManage} />
		</label>
		<label class="field">
			<span class="field-label">Description</span>
			<textarea rows="3" bind:value={description} disabled={!canManage} placeholder="What this workspace is for"></textarea>
		</label>
		<div class="meta">
			<div><span>Slug</span><code>{ws.slug}</code></div>
			<div><span>Plan</span><code>{ws.plan_type}</code></div>
		</div>
		{#if canManage}
			<div class="actions">
				<button class="btn btn--primary" onclick={save} disabled={saving || !name.trim()}>
					{#if saving}<Loader2 class="spin" size={15} />{/if}
					{saved ? 'Saved' : 'Save changes'}
				</button>
			</div>
		{:else}
			<p class="readonly-note">Only owners and admins can edit the workspace.</p>
		{/if}
	</div>
{/if}

<style>
	.sec-head { display: flex; gap: 14px; align-items: flex-start; margin-bottom: 22px; }
	.sec-icon { width: 40px; height: 40px; border-radius: 11px; flex-shrink: 0; display: flex; align-items: center; justify-content: center; background: color-mix(in srgb, var(--dt) 8%, transparent); color: var(--dt); }
	.sec-head h2 { margin: 0; font-size: 1.15rem; font-weight: 660; letter-spacing: -0.02em; }
	.sec-head p { margin: 4px 0 0; font-size: 0.84rem; color: var(--dt3); }
	.card { border: 1px solid var(--dbd); border-radius: 14px; padding: 20px; display: flex; flex-direction: column; gap: 18px; background: color-mix(in srgb, var(--dt) 2%, transparent); }
	.field { display: flex; flex-direction: column; gap: 6px; }
	.field-label { font-size: 0.82rem; font-weight: 580; }
	.field input, .field textarea { width: 100%; padding: 9px 12px; border-radius: 9px; border: 1px solid var(--dbd); background: var(--dbg); color: var(--dt); font-size: 0.86rem; outline: none; font-family: inherit; resize: vertical; }
	.field input:focus, .field textarea:focus { border-color: color-mix(in srgb, var(--dt) 40%, transparent); }
	.field input:disabled, .field textarea:disabled { opacity: 0.6; }
	.meta { display: flex; gap: 28px; flex-wrap: wrap; }
	.meta div { display: flex; flex-direction: column; gap: 3px; }
	.meta span { font-size: 0.72rem; color: var(--dt3); }
	.meta code { font-size: 0.8rem; color: var(--dt2); }
	.actions { display: flex; justify-content: flex-end; }
	.btn { display: inline-flex; align-items: center; gap: 7px; padding: 9px 16px; border-radius: 9px; font-size: 0.84rem; font-weight: 560; cursor: pointer; border: 1px solid transparent; }
	.btn--primary { background: var(--dt); color: var(--dbg); }
	.btn:disabled { opacity: 0.55; cursor: not-allowed; }
	.readonly-note { font-size: 0.74rem; color: var(--dt3); margin: 0; text-align: right; }
	.banner { padding: 11px 14px; border-radius: 10px; font-size: 0.83rem; margin-bottom: 16px; }
	.banner--error { background: color-mix(in srgb, #ef4444 12%, transparent); color: #ef4444; }
	.loading { display: flex; align-items: center; gap: 8px; color: var(--dt3); font-size: 0.85rem; padding: 20px 0; }
	:global(.spin) { animation: spin 0.9s linear infinite; }
	@keyframes spin { to { transform: rotate(360deg); } }

	@media (max-width: 768px) {
		.card { padding: 16px; }
		.actions .btn { width: 100%; justify-content: center; min-height: 40px; }
		.actions { justify-content: stretch; }
		.field input, .field textarea { font-size: 1rem; /* prevent iOS zoom */ }
		.field input { min-height: 40px; }
	}
</style>
