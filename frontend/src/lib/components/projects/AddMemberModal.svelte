<script lang="ts">
	import { Dialog } from 'bits-ui';
	import type { ProjectRole } from '$lib/api/projects/types';
	import RoleSelector from './RoleSelector.svelte';
	import { UserPlus, X, AlertCircle } from 'lucide-svelte';

	interface Props {
		open?: boolean;
		workspaceId: string;
		onClose?: () => void;
		onAdd?: (data: { user_id: string; role: ProjectRole; workspace_id: string }) => void;
	}

	let { open = $bindable(false), workspaceId, onClose, onAdd }: Props = $props();

	let userId = $state('');
	let userEmail = $state('');
	let selectedRole = $state<ProjectRole>('viewer');
	let error = $state('');

	function handleSubmit() {
		// Validate
		error = '';

		if (!userId.trim() && !userEmail.trim()) {
			error = 'Please enter a user ID or email address';
			return;
		}

		// In a real implementation, you might want to look up the user by email
		// For now, we'll use the userId if provided, otherwise use email as userId
		const finalUserId = userId.trim() || userEmail.trim();

		onAdd?.({
			user_id: finalUserId,
			role: selectedRole,
			workspace_id: workspaceId
		});

		resetForm();
		open = false;
	}

	function resetForm() {
		userId = '';
		userEmail = '';
		selectedRole = 'viewer';
		error = '';
	}

	function handleClose() {
		resetForm();
		open = false;
		onClose?.();
	}
</script>

<Dialog.Root bind:open>
	<Dialog.Portal>
		<Dialog.Overlay class="fixed inset-0 bg-black/50 z-50 animate-in fade-in-0" />
		<Dialog.Content
			class="fixed left-1/2 top-1/2 -translate-x-1/2 -translate-y-1/2 w-full bos-modal bos-modal--md animate-in fade-in-0 zoom-in-95"
		>
			<!-- Header -->
			<div class="bos-modal-header">
				<div class="flex items-center gap-3">
					<div class="p-2 rounded-lg prm-am__icon-bg">
						<UserPlus class="w-5 h-5 prm-am__icon-accent" />
					</div>
					<Dialog.Title class="bos-modal-title prm-am__modal-title">
						Add Project Member
					</Dialog.Title>
				</div>
				<Dialog.Close
					class="btn-pill btn-pill-ghost btn-pill-icon"
					onclick={handleClose}
				>
					<X class="w-5 h-5 prm-am__icon-muted" />
				</Dialog.Close>
			</div>

			<!-- Body -->
			<div class="px-6 py-4 space-y-4">
				{#if error}
					<div class="bos-error-banner flex items-start gap-2">
						<AlertCircle class="w-5 h-5 flex-shrink-0 mt-0.5 prm-am__icon-error" />
						<p class="text-sm">{error}</p>
					</div>
				{/if}

				<!-- User ID -->
				<div>
					<label for="user-id" class="bos-label">
						User ID <span class="prm-am__required">*</span>
					</label>
					<input
						id="user-id"
						type="text"
						bind:value={userId}
						placeholder="e.g., user_123abc or user@example.com"
						class="bos-input"
					/>
					<p class="prm-am__hint">
						Enter the user's ID from your workspace or their email address
					</p>
				</div>

				<!-- Email (optional alternative) -->
				<div>
					<label for="user-email" class="bos-label">
						Or Email Address
					</label>
					<input
						id="user-email"
						type="email"
						bind:value={userEmail}
						placeholder="user@example.com"
						disabled={userId.trim().length > 0}
						class="bos-input"
					/>
					<p class="prm-am__hint">
						Alternative: enter email if you don't know the user ID
					</p>
				</div>

				<!-- Role Selection -->
				<div>
					<label class="bos-label prm-am__role-label">
						Role <span class="prm-am__required">*</span>
					</label>
					<RoleSelector bind:value={selectedRole} />
					<p class="prm-am__hint prm-am__hint--role">
						{#if selectedRole === 'lead'}
							Full project control - can manage members, edit, and delete
						{:else if selectedRole === 'contributor'}
							Can edit and contribute to the project
						{:else if selectedRole === 'reviewer'}
							Can review and comment on the project
						{:else}
							Read-only access to the project
						{/if}
					</p>
				</div>

				<!-- Info Box -->
				<div class="p-3 rounded-lg prm-am__info-box">
					<div class="flex items-start gap-2">
						<svg
							class="w-5 h-5 flex-shrink-0 mt-0.5 prm-am__icon-accent"
							fill="currentColor"
							viewBox="0 0 20 20"
						>
							<path
								fill-rule="evenodd"
								d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7-4a1 1 0 11-2 0 1 1 0 012 0zM9 9a1 1 0 000 2v3a1 1 0 001 1h1a1 1 0 100-2v-3a1 1 0 00-1-1H9z"
								clip-rule="evenodd"
							/>
						</svg>
						<div class="text-xs prm-am__info-text">
							<p class="font-medium mb-1">About Project Roles</p>
							<ul class="space-y-1 list-disc list-inside">
								<li>Lead: Full control (edit, delete, invite members)</li>
								<li>Contributor: Can edit project content</li>
								<li>Reviewer: Can review and comment</li>
								<li>Viewer: Read-only access</li>
							</ul>
						</div>
					</div>
				</div>
			</div>

			<!-- Footer -->
			<div class="bos-modal-footer">
				<button
					onclick={handleClose}
					class="btn-pill btn-pill-ghost btn-pill-sm"
				>
					Cancel
				</button>
				<button
					onclick={handleSubmit}
					disabled={!userId.trim() && !userEmail.trim()}
					class="btn-pill btn-pill-primary btn-pill-sm flex items-center gap-2"
				>
					<UserPlus class="w-4 h-4" />
					Add Member
				</button>
			</div>
		</Dialog.Content>
	</Dialog.Portal>
</Dialog.Root>

<style>
	.prm-am__icon-bg {
		background: color-mix(in srgb, var(--bos-status-info) 12%, var(--dbg, #fff));
	}
	.prm-am__icon-accent {
		color: var(--bos-status-info);
	}
	.prm-am__icon-muted {
		color: var(--dt2, #555);
	}
	.prm-am__icon-error {
		color: var(--bos-status-error);
	}
	.prm-am__modal-title {
		margin-bottom: 0;
	}
	.prm-am__required {
		color: var(--bos-status-error);
	}
	.prm-am__hint {
		font-size: 0.75rem;
		color: var(--dt2, #555);
		margin-top: 0.25rem;
	}
	.prm-am__hint--role {
		margin-top: 0.5rem;
	}
	.prm-am__role-label {
		margin-bottom: 0.5rem;
	}
	.prm-am__info-box {
		background: var(--bos-status-info-bg);
		border: 1px solid color-mix(in srgb, var(--bos-status-info) 25%, var(--dbd, #e0e0e0));
		border-radius: 0.5rem;
	}
	.prm-am__info-text {
		color: var(--dt2, #555);
	}
</style>
