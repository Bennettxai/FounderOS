<script lang="ts">
	import { onMount } from 'svelte';
	import { createWorkspace } from '$lib/api/workspaces';
	import { switchWorkspace, initializeWorkspaces } from '$lib/stores/workspaces';
	import { listMyOrganizations } from '$lib/api/organizations';
	import type { Organization } from '$lib/api/organizations';
	import { X, AlertCircle, Loader2 } from 'lucide-svelte';

	interface Props {
		show: boolean;
		onClose: () => void;
	}

	let { show, onClose }: Props = $props();

	let name = $state('');
	let selectedOrgId = $state<string | null>(null);
	let orgs = $state<Organization[]>([]);
	let orgsLoading = $state(false);
	let loading = $state(false);
	let error = $state<string | null>(null);
	let nameError = $state<string | null>(null);

	// Auto-derive slug from name: lowercase, replace spaces/special chars with dashes
	const slug = $derived(
		name
			.trim()
			.toLowerCase()
			.replace(/[^a-z0-9]+/g, '-')
			.replace(/^-+|-+$/g, '')
	);

	function resetForm() {
		name = '';
		selectedOrgId = orgs.length === 1 ? orgs[0].id : null;
		error = null;
		nameError = null;
	}

	function validateName(): boolean {
		if (!name.trim()) {
			nameError = 'Workspace name is required';
			return false;
		}
		if (name.trim().length < 2) {
			nameError = 'Name must be at least 2 characters';
			return false;
		}
		if (name.trim().length > 50) {
			nameError = 'Name must be 50 characters or fewer';
			return false;
		}
		nameError = null;
		return true;
	}

	async function handleSubmit() {
		if (!validateName()) return;

		loading = true;
		error = null;

		try {
			const payload: Record<string, unknown> = { name: name.trim() };
			if (selectedOrgId) payload.organization_id = selectedOrgId;

			const workspace = await createWorkspace(payload as unknown as Parameters<typeof createWorkspace>[0]);

			await initializeWorkspaces();
			await switchWorkspace(workspace.id);

			resetForm();
			onClose();
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to create workspace';
		} finally {
			loading = false;
		}
	}

	function handleCancel() {
		resetForm();
		onClose();
	}

	function handleOverlayKeydown(e: KeyboardEvent) {
		if (e.key === 'Escape') handleCancel();
	}

	onMount(async () => {
		orgsLoading = true;
		try {
			orgs = await listMyOrganizations();
			// Auto-select if only one org
			if (orgs.length === 1) selectedOrgId = orgs[0].id;
		} catch {
			// Silently ignore - org picker is optional
		} finally {
			orgsLoading = false;
		}
	});
</script>

{#if show}
	<!-- svelte-ignore a11y_no_noninteractive_element_interactions -->
	<div
		class="modal-overlay"
		role="dialog"
		aria-modal="true"
		aria-label="Create workspace"
		onclick={handleCancel}
		onkeydown={handleOverlayKeydown}
	>
		<!-- svelte-ignore a11y_click_events_have_key_events -->
		<div class="modal" onclick={(e) => e.stopPropagation()}>
			<div class="modal-head">
				<h3 class="modal-title">New workspace</h3>
				<button class="close-btn" onclick={handleCancel} aria-label="Close">
					<X size={16} strokeWidth={2} />
				</button>
			</div>

			<div class="modal-body">
				<!-- Workspace name -->
				<div class="field">
					<label for="ws-name" class="field-label">
						Name <span class="required" aria-hidden="true">*</span>
					</label>
					<input
						id="ws-name"
						type="text"
						class="field-input"
						class:field-input--error={nameError}
						bind:value={name}
						onblur={validateName}
						placeholder="My Workspace"
						maxlength="50"
						autocomplete="off"
						autofocus
					/>
					{#if nameError}
						<span class="field-error">{nameError}</span>
					{:else if slug}
						<span class="field-hint">slug: {slug}</span>
					{/if}
				</div>

				<!-- Org picker - shown only when user belongs to 2+ orgs -->
				{#if orgsLoading}
					<div class="orgs-loading">
						<Loader2 size={13} class="animate-spin" />
						<span>Loading organizations...</span>
					</div>
				{:else if orgs.length > 1}
					<div class="field">
						<label for="ws-org" class="field-label">Organization</label>
						<select id="ws-org" class="field-select" bind:value={selectedOrgId}>
							<option value={null}>No organization</option>
							{#each orgs as org (org.id)}
								<option value={org.id}>{org.name}</option>
							{/each}
						</select>
						<span class="field-hint">Workspaces belong to an organization</span>
					</div>
				{/if}

				{#if error}
					<div class="error-banner">
						<AlertCircle size={14} />
						<span>{error}</span>
					</div>
				{/if}
			</div>

			<div class="modal-foot">
				<button
					class="btn btn-ghost"
					onclick={handleCancel}
					disabled={loading}
				>
					Cancel
				</button>
				<button
					class="btn btn-primary"
					onclick={handleSubmit}
					disabled={loading || !name.trim()}
				>
					{#if loading}
						<span class="spinner" aria-hidden="true"></span>
						Creating...
					{:else}
						Create workspace
					{/if}
				</button>
			</div>
		</div>
	</div>
{/if}

<style>
	.modal-overlay {
		position: fixed;
		inset: 0;
		background: rgb(0 0 0 / 0.55);
		display: flex;
		align-items: center;
		justify-content: center;
		z-index: 200;
		padding: 16px;
	}

	.modal {
		background: color-mix(in srgb, var(--dbg, #fff) 98%, #000 2%);
		border: 1px solid color-mix(in srgb, var(--dt, #111) 12%, transparent);
		border-radius: 12px;
		width: 100%;
		max-width: 420px;
		box-shadow: 0 24px 56px rgb(0 0 0 / 0.48), 0 0 0 1px rgb(255 255 255 / 0.03) inset;
		display: flex;
		flex-direction: column;
	}

	@media (max-width: 640px) {
		.modal-overlay {
			align-items: flex-end;
			padding: 0;
		}

		.modal {
			border-radius: 14px 14px 0 0;
			border-bottom: none;
			max-width: 100%;
			width: 100%;
			/* Safe area for home indicator */
			padding-bottom: env(safe-area-inset-bottom, 0px);
		}

		.modal-body {
			padding: 1rem 1rem 0.75rem;
		}

		.modal-foot {
			padding: 0.75rem 1rem calc(0.85rem + env(safe-area-inset-bottom, 0px));
		}

		.field-input,
		.field-select {
			height: 44px;
			font-size: 16px; /* prevents iOS auto-zoom on focus */
		}

		.btn {
			height: 44px;
			font-size: 0.875rem;
			flex: 1;
		}

		.modal-foot {
			gap: 0.75rem;
		}
	}

	@media (max-width: 320px) {
		.modal-head {
			padding: 0.75rem 0.75rem 0.5rem;
		}

		.modal-body {
			padding: 0.75rem;
		}

		.modal-foot {
			padding: 0.65rem 0.75rem calc(0.65rem + env(safe-area-inset-bottom, 0px));
		}
	}

	.modal-head {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 1rem 1rem 0.6rem;
		border-bottom: 1px solid color-mix(in srgb, var(--dt, #111) 9%, transparent);
	}

	.modal-title {
		margin: 0;
		font-size: 0.88rem;
		font-weight: 650;
		color: var(--dt, #111);
		line-height: 1.2;
	}

	.close-btn {
		display: flex;
		align-items: center;
		justify-content: center;
		width: 28px;
		height: 28px;
		border: none;
		background: transparent;
		color: var(--dt3, #888);
		border-radius: 6px;
		cursor: pointer;
		transition: background 120ms ease, color 120ms ease;
	}

	.close-btn:hover {
		background: color-mix(in srgb, var(--dt, #111) 8%, transparent);
		color: var(--dt, #111);
	}

	.modal-body {
		padding: 1rem;
		display: flex;
		flex-direction: column;
		gap: 0.9rem;
	}

	.field {
		display: flex;
		flex-direction: column;
		gap: 0.3rem;
	}

	.field-label {
		font-size: 0.72rem;
		font-weight: 600;
		color: var(--dt2, #444);
		line-height: 1.2;
	}

	.required {
		color: #f87171;
		margin-left: 2px;
	}

	.field-input,
	.field-select {
		height: 36px;
		padding: 0 0.62rem;
		background: color-mix(in srgb, var(--dt, #111) 4%, transparent);
		border: 1px solid color-mix(in srgb, var(--dt, #111) 12%, transparent);
		border-radius: 7px;
		color: var(--dt, #111);
		font-size: 0.8rem;
		font-family: inherit;
		outline: none;
		transition: border-color 120ms ease;
		width: 100%;
	}

	.field-input:focus,
	.field-select:focus {
		border-color: #6366f1;
	}

	.field-input--error {
		border-color: #f87171;
	}

	.field-input::placeholder {
		color: var(--dt3, #888);
	}

	.field-select {
		appearance: none;
		background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' fill='none' viewBox='0 0 24 24' stroke-width='2' stroke='%23888'%3E%3Cpath stroke-linecap='round' stroke-linejoin='round' d='m19.5 8.25-7.5 7.5-7.5-7.5'/%3E%3C/svg%3E");
		background-repeat: no-repeat;
		background-position: right 0.6rem center;
		background-size: 14px;
		padding-right: 2rem;
		cursor: pointer;
	}

	.field-hint {
		font-size: 0.64rem;
		color: var(--dt3, #888);
		line-height: 1.3;
	}

	.field-error {
		font-size: 0.64rem;
		color: #f87171;
		line-height: 1.3;
	}

	.orgs-loading {
		display: flex;
		align-items: center;
		gap: 0.4rem;
		font-size: 0.72rem;
		color: var(--dt3, #888);
	}

	.error-banner {
		display: flex;
		align-items: flex-start;
		gap: 0.4rem;
		padding: 0.55rem 0.65rem;
		background: rgb(220 38 38 / 0.08);
		border: 1px solid rgb(220 38 38 / 0.18);
		border-radius: 7px;
		color: #ef4444;
		font-size: 0.76rem;
		line-height: 1.4;
	}

	.modal-foot {
		display: flex;
		align-items: center;
		justify-content: flex-end;
		gap: 0.5rem;
		padding: 0.65rem 1rem 0.85rem;
		border-top: 1px solid color-mix(in srgb, var(--dt, #111) 9%, transparent);
	}

	.btn {
		display: inline-flex;
		align-items: center;
		gap: 0.4rem;
		padding: 0 0.85rem;
		height: 34px;
		border-radius: 7px;
		font-size: 0.78rem;
		font-weight: 600;
		font-family: inherit;
		cursor: pointer;
		transition: background 120ms ease, border-color 120ms ease, opacity 120ms ease;
		border: 1px solid transparent;
	}

	.btn:disabled {
		opacity: 0.5;
		cursor: not-allowed;
	}

	.btn-ghost {
		background: transparent;
		border-color: color-mix(in srgb, var(--dt, #111) 14%, transparent);
		color: var(--dt2, #444);
	}

	.btn-ghost:hover:not(:disabled) {
		background: color-mix(in srgb, var(--dt, #111) 6%, transparent);
		color: var(--dt, #111);
	}

	.btn-primary {
		background: #6366f1;
		border-color: #6366f1;
		color: #fff;
	}

	.btn-primary:hover:not(:disabled) {
		background: #4f46e5;
		border-color: #4f46e5;
	}

	.spinner {
		width: 13px;
		height: 13px;
		border: 2px solid rgb(255 255 255 / 0.3);
		border-top-color: #fff;
		border-radius: 50%;
		animation: spin 0.6s linear infinite;
		flex-shrink: 0;
	}

	@keyframes spin {
		to { transform: rotate(360deg); }
	}
</style>
