<script lang="ts">
	import { Pencil, Trash2, Check, X, Loader2, UserPlus, UserMinus } from 'lucide-svelte';
	import { adminControl } from '$lib/api/admin-control';
	import ConfirmDialog from './ConfirmDialog.svelte';
	import type { AdminWorkspaceDetail, AdminWorkspaceMember } from '$lib/api/admin';

	interface Props {
		workspace: AdminWorkspaceDetail;
		onDone: () => void;
	}

	let { workspace, onDone }: Props = $props();

	const PLANS = ['free', 'pro', 'team', 'enterprise'];
	const ROLES = ['member', 'admin', 'owner'];

	// shared error
	let actionError = $state<string | null>(null);

	// ── rename ───────────────────────────────────────────────────────────────
	let renaming = $state(false);
	let renameValue = $state(workspace.name);
	let renameBusy = $state(false);
	let renameError = $state<string | null>(null);

	function startRename() {
		renameValue = workspace.name;
		renameError = null;
		renaming = true;
	}
	function cancelRename() {
		renaming = false;
		renameError = null;
	}
	async function submitRename() {
		const name = renameValue.trim();
		if (!name || name === workspace.name) {
			cancelRename();
			return;
		}
		renameBusy = true;
		renameError = null;
		try {
			await adminControl.updateWorkspace(workspace.id, { name });
			renaming = false;
			onDone();
		} catch (e) {
			renameError = e instanceof Error ? e.message : 'Rename failed';
		} finally {
			renameBusy = false;
		}
	}

	// ── plan ─────────────────────────────────────────────────────────────────
	let planBusy = $state(false);

	async function changePlan(plan: string) {
		if (plan === workspace.plan_type) return;
		planBusy = true;
		actionError = null;
		try {
			await adminControl.updateWorkspace(workspace.id, { plan_type: plan });
			onDone();
		} catch (e) {
			actionError = e instanceof Error ? e.message : 'Plan update failed';
		} finally {
			planBusy = false;
		}
	}

	// ── delete ────────────────────────────────────────────────────────────────
	let confirmDelete = $state(false);
	let deleteBusy = $state(false);

	async function confirmDoDelete() {
		deleteBusy = true;
		actionError = null;
		try {
			await adminControl.deleteWorkspace(workspace.id);
			confirmDelete = false;
			onDone();
		} catch (e) {
			actionError = e instanceof Error ? e.message : 'Delete failed';
			confirmDelete = false;
		} finally {
			deleteBusy = false;
		}
	}

	// ── add member ────────────────────────────────────────────────────────────
	let addingMember = $state(false);
	let addEmail = $state('');
	let addRole = $state('member');
	let addBusy = $state(false);
	let addError = $state<string | null>(null);

	function startAddMember() {
		addEmail = '';
		addRole = 'member';
		addError = null;
		addingMember = true;
	}
	function cancelAddMember() {
		addingMember = false;
		addError = null;
	}
	async function submitAddMember() {
		const email = addEmail.trim();
		if (!email) {
			addError = 'Email is required';
			return;
		}
		addBusy = true;
		addError = null;
		try {
			await adminControl.addWorkspaceMember(workspace.id, email, addRole);
			addingMember = false;
			onDone();
		} catch (e) {
			addError = e instanceof Error ? e.message : 'Failed to add member';
		} finally {
			addBusy = false;
		}
	}

	// ── member actions ────────────────────────────────────────────────────────
	let busyMemberId = $state<string | null>(null);

	async function changeRole(m: AdminWorkspaceMember, role: string) {
		if (role === m.role) return;
		busyMemberId = m.user_id;
		actionError = null;
		try {
			await adminControl.setWorkspaceMemberRole(workspace.id, m.user_id, role);
			onDone();
		} catch (e) {
			actionError = e instanceof Error ? e.message : 'Role update failed';
		} finally {
			busyMemberId = null;
		}
	}

	let confirmRemoveMember = $state<AdminWorkspaceMember | null>(null);
	let removeBusy = $state(false);

	async function confirmDoRemove() {
		if (!confirmRemoveMember) return;
		removeBusy = true;
		actionError = null;
		try {
			await adminControl.removeWorkspaceMember(workspace.id, confirmRemoveMember.user_id);
			confirmRemoveMember = null;
			onDone();
		} catch (e) {
			actionError = e instanceof Error ? e.message : 'Remove failed';
			confirmRemoveMember = null;
		} finally {
			removeBusy = false;
		}
	}
</script>

<div class="ws-actions">
	{#if actionError}
		<div class="banner banner--error">{actionError}</div>
	{/if}

	<!-- rename + plan row -->
	<div class="row-label">Name</div>
	{#if renaming}
		<div class="inline-edit">
			<input
				bind:value={renameValue}
				onkeydown={(e) => {
					if (e.key === 'Enter') submitRename();
					if (e.key === 'Escape') cancelRename();
				}}
				disabled={renameBusy}
				aria-label="New workspace name"
				class="rename-input"
			/>
			<button class="icon-btn" onclick={submitRename} disabled={renameBusy} aria-label="Confirm rename">
				{#if renameBusy}<Loader2 size={14} class="spin" />{:else}<Check size={14} />{/if}
			</button>
			<button class="icon-btn" onclick={cancelRename} disabled={renameBusy} aria-label="Cancel">
				<X size={14} />
			</button>
			{#if renameError}<span class="inline-err">{renameError}</span>{/if}
		</div>
	{:else}
		<div class="field-row">
			<span class="field-val">{workspace.name}</span>
			<button class="action-btn" onclick={startRename} aria-label="Rename workspace">
				<Pencil size={12} /> Rename
			</button>
		</div>
	{/if}

	<div class="row-label" style="margin-top:14px">Plan</div>
	<div class="field-row">
		<select
			value={workspace.plan_type ?? 'free'}
			onchange={(e) => changePlan(e.currentTarget.value)}
			disabled={planBusy}
			class="plan-select"
			aria-label="Workspace plan"
		>
			{#each PLANS as p}<option value={p}>{p}</option>{/each}
		</select>
		{#if planBusy}<Loader2 size={14} class="spin muted" />{/if}
	</div>

	<!-- members section -->
	<div class="section-head">
		<span>Members ({workspace.members.length})</span>
		<button class="action-btn" onclick={startAddMember} aria-label="Add member">
			<UserPlus size={12} /> Add
		</button>
	</div>

	{#if addingMember}
		<div class="add-member-form">
			<input
				bind:value={addEmail}
				type="email"
				placeholder="email@example.com"
				disabled={addBusy}
				aria-label="Member email"
				class="rename-input"
				style="flex:1"
			/>
			<select bind:value={addRole} disabled={addBusy} class="plan-select" aria-label="Member role">
				{#each ROLES as r}<option value={r}>{r}</option>{/each}
			</select>
			<button class="icon-btn" onclick={submitAddMember} disabled={addBusy} aria-label="Confirm add">
				{#if addBusy}<Loader2 size={14} class="spin" />{:else}<Check size={14} />{/if}
			</button>
			<button class="icon-btn" onclick={cancelAddMember} disabled={addBusy} aria-label="Cancel">
				<X size={14} />
			</button>
			{#if addError}<span class="inline-err" style="width:100%">{addError}</span>{/if}
		</div>
	{/if}

	<div class="member-list">
		{#each workspace.members as m (m.user_id)}
			<div class="member-row" class:busy={busyMemberId === m.user_id}>
				<span class="m-email">{m.email}</span>
				<select
					value={m.role ?? 'member'}
					onchange={(e) => changeRole(m, e.currentTarget.value)}
					disabled={busyMemberId === m.user_id}
					class="plan-select sm"
					aria-label="Role for {m.email}"
				>
					{#each ROLES as r}<option value={r}>{r}</option>{/each}
				</select>
				<button
					class="icon-btn icon-btn--danger"
					onclick={() => (confirmRemoveMember = m)}
					disabled={busyMemberId === m.user_id}
					aria-label="Remove {m.email}"
				>
					<UserMinus size={13} />
				</button>
			</div>
		{/each}
		{#if workspace.members.length === 0}
			<span class="muted" style="font-size:0.82rem">No members.</span>
		{/if}
	</div>

	<!-- delete zone -->
	<div class="danger-zone">
		<button class="action-btn action-btn--danger" onclick={() => (confirmDelete = true)} disabled={deleteBusy} aria-label="Delete workspace">
			<Trash2 size={13} /> Delete workspace
		</button>
	</div>
</div>

{#if confirmDelete}
	<ConfirmDialog
		title="Delete workspace"
		message="Delete {workspace.name}? This cannot be undone. All workspace data will be removed."
		confirmLabel="Delete"
		danger={true}
		loading={deleteBusy}
		onConfirm={confirmDoDelete}
		onCancel={() => (confirmDelete = false)}
	/>
{/if}

{#if confirmRemoveMember}
	<ConfirmDialog
		title="Remove member"
		message="Remove {confirmRemoveMember.email} from {workspace.name}?"
		confirmLabel="Remove"
		danger={true}
		loading={removeBusy}
		onConfirm={confirmDoRemove}
		onCancel={() => (confirmRemoveMember = null)}
	/>
{/if}

<style>
	.ws-actions {
		display: flex;
		flex-direction: column;
		gap: 4px;
	}
	.banner--error {
		background: color-mix(in srgb, #ef4444 12%, transparent);
		color: #ef4444;
		padding: 8px 12px;
		border-radius: 9px;
		font-size: 0.8rem;
		margin-bottom: 4px;
	}
	.row-label {
		font-size: 0.72rem;
		text-transform: uppercase;
		letter-spacing: 0.05em;
		color: var(--dt3);
		font-weight: 600;
		margin-bottom: 4px;
	}
	.field-row {
		display: flex;
		align-items: center;
		gap: 8px;
	}
	.field-val {
		font-size: 0.85rem;
		color: var(--dt);
		flex: 1;
	}
	.section-head {
		display: flex;
		align-items: center;
		justify-content: space-between;
		margin-top: 18px;
		margin-bottom: 8px;
		font-size: 0.72rem;
		text-transform: uppercase;
		letter-spacing: 0.05em;
		color: var(--dt3);
		font-weight: 600;
	}
	.add-member-form {
		display: flex;
		align-items: center;
		gap: 6px;
		flex-wrap: wrap;
		margin-bottom: 8px;
	}
	.member-list {
		display: flex;
		flex-direction: column;
		gap: 2px;
	}
	.member-row {
		display: flex;
		align-items: center;
		gap: 8px;
		padding: 6px 0;
		border-bottom: 1px solid var(--dbd);
	}
	.member-row:last-child {
		border-bottom: none;
	}
	.member-row.busy {
		opacity: 0.5;
		pointer-events: none;
	}
	.m-email {
		flex: 1;
		font-size: 0.82rem;
		color: var(--dt);
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
		min-width: 0;
	}
	.danger-zone {
		margin-top: 20px;
		padding-top: 16px;
		border-top: 1px solid var(--dbd);
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
		min-width: 160px;
	}
	.rename-input:focus {
		border-color: #6366f1;
	}
	.plan-select {
		background: var(--dbg2, #111);
		border: 1px solid var(--dbd);
		border-radius: 8px;
		color: var(--dt);
		padding: 5px 8px;
		font-size: 0.8rem;
		cursor: pointer;
	}
	.plan-select.sm {
		font-size: 0.75rem;
		padding: 3px 6px;
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
	.icon-btn--danger {
		color: color-mix(in srgb, #ef4444 70%, var(--dt3));
		border-color: color-mix(in srgb, #ef4444 25%, var(--dbd));
	}
	.icon-btn--danger:hover {
		color: #ef4444;
	}
	.inline-err {
		font-size: 0.78rem;
		color: #ef4444;
	}
	.muted {
		color: var(--dt3);
	}
	:global(.spin) {
		animation: spin 0.9s linear infinite;
	}
	@keyframes spin {
		to {
			transform: rotate(360deg);
		}
	}
	@media (max-width: 768px) {
		.action-btn, .icon-btn { min-height: 40px; }
		.add-member-form { flex-direction: column; align-items: stretch; }
		.rename-input { min-width: 0; width: 100%; }
		.member-row { flex-wrap: wrap; }
		.plan-select { min-height: 40px; }
	}
</style>
