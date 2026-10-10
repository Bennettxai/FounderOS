<script lang="ts">
	import { Pencil, Trash2, Check, X, Loader2 } from 'lucide-svelte';
	import { adminControl } from '$lib/api/admin-control';
	import ConfirmDialog from './ConfirmDialog.svelte';
	import type { AdminOrg } from '$lib/api/admin';

	interface Props {
		org: AdminOrg;
		onDone: () => void;
	}

	let { org, onDone }: Props = $props();

	// rename state
	let renaming = $state(false);
	let renameValue = $state(org.name);
	let renameBusy = $state(false);
	let renameError = $state<string | null>(null);

	// delete state
	let confirmDelete = $state(false);
	let deleteBusy = $state(false);

	// shared error banner
	let actionError = $state<string | null>(null);

	function startRename() {
		renameValue = org.name;
		renameError = null;
		renaming = true;
	}

	function cancelRename() {
		renaming = false;
		renameError = null;
	}

	async function submitRename() {
		const name = renameValue.trim();
		if (!name || name === org.name) {
			cancelRename();
			return;
		}
		renameBusy = true;
		renameError = null;
		try {
			await adminControl.renameOrganization(org.id, name);
			renaming = false;
			onDone();
		} catch (e) {
			renameError = e instanceof Error ? e.message : 'Rename failed';
		} finally {
			renameBusy = false;
		}
	}

	async function confirmDoDelete() {
		deleteBusy = true;
		actionError = null;
		try {
			await adminControl.deleteOrganization(org.id);
			confirmDelete = false;
			onDone();
		} catch (e) {
			actionError = e instanceof Error ? e.message : 'Delete failed';
			confirmDelete = false;
		} finally {
			deleteBusy = false;
		}
	}
</script>

<div class="org-actions">
	{#if actionError}
		<div class="banner banner--error">{actionError}</div>
	{/if}

	{#if renaming}
		<div class="inline-edit">
			<input
				bind:value={renameValue}
				onkeydown={(e) => {
					if (e.key === 'Enter') submitRename();
					if (e.key === 'Escape') cancelRename();
				}}
				disabled={renameBusy}
				aria-label="New organization name"
				class="rename-input"
			/>
			<button class="icon-btn" onclick={submitRename} disabled={renameBusy} aria-label="Confirm rename">
				{#if renameBusy}<Loader2 size={14} class="spin" />{:else}<Check size={14} />{/if}
			</button>
			<button class="icon-btn" onclick={cancelRename} disabled={renameBusy} aria-label="Cancel rename">
				<X size={14} />
			</button>
			{#if renameError}<span class="inline-err">{renameError}</span>{/if}
		</div>
	{:else}
		<div class="action-row">
			<button class="action-btn" onclick={startRename} aria-label="Rename organization">
				<Pencil size={13} /> Rename
			</button>
			<button class="action-btn action-btn--danger" onclick={() => (confirmDelete = true)} disabled={deleteBusy} aria-label="Delete organization">
				<Trash2 size={13} /> Delete
			</button>
		</div>
	{/if}
</div>

{#if confirmDelete}
	<ConfirmDialog
		title="Delete organization"
		message="Delete {org.name}? This cannot be undone. All workspaces and members will be removed."
		confirmLabel="Delete"
		danger={true}
		loading={deleteBusy}
		onConfirm={confirmDoDelete}
		onCancel={() => (confirmDelete = false)}
	/>
{/if}

<style>
	.org-actions {
		display: flex;
		flex-direction: column;
		gap: 8px;
	}
	.banner--error {
		background: color-mix(in srgb, #ef4444 12%, transparent);
		color: #ef4444;
		padding: 8px 12px;
		border-radius: 9px;
		font-size: 0.8rem;
	}
	.action-row {
		display: flex;
		align-items: center;
		gap: 6px;
		flex-wrap: wrap;
	}
	.action-btn {
		display: inline-flex;
		align-items: center;
		gap: 5px;
		padding: 5px 11px;
		border-radius: 8px;
		font-size: 0.8rem;
		font-weight: 520;
		cursor: pointer;
		background: transparent;
		border: 1px solid var(--dbd);
		color: var(--dt3);
		transition: color 0.1s, border-color 0.1s;
	}
	.action-btn:hover {
		color: var(--dt);
		border-color: var(--dt3);
	}
	.action-btn:disabled {
		opacity: 0.45;
		pointer-events: none;
	}
	.action-btn--danger {
		color: color-mix(in srgb, #ef4444 80%, var(--dt3));
		border-color: color-mix(in srgb, #ef4444 30%, var(--dbd));
	}
	.action-btn--danger:hover {
		color: #ef4444;
		border-color: color-mix(in srgb, #ef4444 60%, transparent);
	}
	.inline-edit {
		display: flex;
		align-items: center;
		gap: 6px;
		flex-wrap: wrap;
	}
	.rename-input {
		background: var(--dbg2, #111);
		border: 1px solid var(--dbd);
		border-radius: 8px;
		color: var(--dt);
		padding: 5px 9px;
		font-size: 0.84rem;
		outline: none;
		min-width: 180px;
	}
	.rename-input:focus {
		border-color: #6366f1;
	}
	.icon-btn {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		background: transparent;
		border: 1px solid var(--dbd);
		border-radius: 7px;
		color: var(--dt3);
		padding: 5px;
		cursor: pointer;
	}
	.icon-btn:hover {
		color: var(--dt);
	}
	.icon-btn:disabled {
		opacity: 0.45;
		pointer-events: none;
	}
	.inline-err {
		font-size: 0.78rem;
		color: #ef4444;
	}
	:global(.spin) {
		animation: spin 0.9s linear infinite;
	}
	@keyframes spin {
		to {
			transform: rotate(360deg);
		}
	}
</style>
