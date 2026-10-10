<script lang="ts">
	import { X, Users, User, Check } from 'lucide-svelte';
	import type { SharePermission } from '$lib/types/modules';

	interface Props {
		moduleId: string;
		moduleName: string;
		isOpen: boolean;
		onClose: () => void;
		onShare: (data: { sharedWithUserId?: string; sharedWithWorkspaceId?: string; permissions: SharePermission[] }) => Promise<void>;
	}

	let { moduleId, moduleName, isOpen, onClose, onShare }: Props = $props();

	let shareType = $state<'user' | 'workspace'>('user');
	let userId = $state('');
	let permissions = $state<SharePermission[]>(['view', 'install']);
	let isSharing = $state(false);

	const allPermissions: Array<{ value: SharePermission; label: string; description: string }> = [
		{ value: 'view', label: 'View', description: 'Can view module details' },
		{ value: 'install', label: 'Install', description: 'Can install the module' },
		{ value: 'modify', label: 'Modify', description: 'Can edit the module' },
		{ value: 'reshare', label: 'Reshare', description: 'Can share with others' }
	];

	function togglePermission(permission: SharePermission) {
		if (permissions.includes(permission)) {
			permissions = permissions.filter(p => p !== permission);
		} else {
			permissions = [...permissions, permission];
		}
	}

	async function handleShare() {
		if (shareType === 'user' && !userId.trim()) {
			alert('Please enter a user ID or email');
			return;
		}

		if (permissions.length === 0) {
			alert('Please select at least one permission');
			return;
		}

		isSharing = true;
		try {
			await onShare({
				sharedWithUserId: shareType === 'user' ? userId : undefined,
				sharedWithWorkspaceId: shareType === 'workspace' ? 'current' : undefined,
				permissions
			});
			// Reset form
			userId = '';
			permissions = ['view', 'install'];
			onClose();
		} catch (error) {
			console.error('Failed to share module:', error);
		} finally {
			isSharing = false;
		}
	}
</script>

{#if isOpen}
	<div class="am-share-overlay">
		<!-- Backdrop -->
		<div
			class="am-share-backdrop"
			onclick={onClose}
			role="presentation"
		></div>

		<!-- Dialog -->
		<div class="am-share-dialog" role="dialog" aria-modal="true" aria-label="Share module">
			<!-- Header -->
			<div class="am-share-header">
				<h2 class="am-share-title">Share Module</h2>
				<button
					onclick={onClose}
					class="am-share-close"
					aria-label="Close dialog"
				>
					<X class="w-5 h-5" />
				</button>
			</div>

			<!-- Content -->
			<div class="am-share-content">
				<!-- Module Name -->
				<div>
					<p class="am-share-meta">
						Sharing: <span class="am-share-meta__name">{moduleName}</span>
					</p>
				</div>

				<!-- Share Type Tabs -->
				<div class="am-share-tabs">
					<button
						type="button"
						onclick={() => shareType = 'user'}
						class="am-share-tab {shareType === 'user' ? 'am-share-tab--active' : ''}"
						aria-label="Share with user"
					>
						<User class="w-4 h-4" />
						<span>User</span>
					</button>
					<button
						type="button"
						onclick={() => shareType = 'workspace'}
						class="am-share-tab {shareType === 'workspace' ? 'am-share-tab--active' : ''}"
						aria-label="Share with workspace"
					>
						<Users class="w-4 h-4" />
						<span>Workspace</span>
					</button>
				</div>

				<!-- User Input (if share type is user) -->
				{#if shareType === 'user'}
					<div class="am-share-field">
						<label class="am-share-label">
							User ID or Email <span class="am-share-req">*</span>
						</label>
						<input
							type="text"
							bind:value={userId}
							placeholder="user@example.com"
							class="am-share-input"
							aria-label="User ID or email"
						/>
					</div>
				{:else}
					<div class="am-share-info">
						<p>This module will be shared with your entire workspace.</p>
					</div>
				{/if}

				<!-- Permissions -->
				<div class="am-share-field">
					<label class="am-share-label">
						Permissions <span class="am-share-req">*</span>
					</label>
					<div class="am-perm-list">
						{#each allPermissions as perm}
							<button
								type="button"
								onclick={() => togglePermission(perm.value)}
								class="am-perm-card {permissions.includes(perm.value) ? 'am-perm-card--active' : ''}"
								aria-label="{perm.label} permission"
							>
								<div class="am-perm-check {permissions.includes(perm.value) ? 'am-perm-check--active' : ''}">
									{#if permissions.includes(perm.value)}
										<Check class="w-3.5 h-3.5" />
									{/if}
								</div>
								<div class="am-perm-info">
									<p class="am-perm-info__title">{perm.label}</p>
									<p class="am-perm-info__desc">{perm.description}</p>
								</div>
							</button>
						{/each}
					</div>
				</div>
			</div>

			<!-- Footer -->
			<div class="am-share-footer">
				<button
					type="button"
					onclick={onClose}
					class="btn-pill btn-pill-ghost"
					disabled={isSharing}
					aria-label="Cancel"
				>
					Cancel
				</button>
				<button
					type="button"
					onclick={handleShare}
					disabled={isSharing}
					class="btn-pill btn-pill-primary am-glow"
					aria-label="Share module"
				>
					{isSharing ? 'Sharing...' : 'Share Module'}
				</button>
			</div>
		</div>
	</div>
{/if}

<style>
	/* ══════════════════════════════════════════════════════════════ */
	/*  SHARE DIALOG (am-share-) — Foundation Tokens                */
	/* ══════════════════════════════════════════════════════════════ */
	.am-share-overlay {
		position: fixed;
		inset: 0;
		z-index: 50;
		display: flex;
		align-items: center;
		justify-content: center;
	}
	.am-share-backdrop {
		position: absolute;
		inset: 0;
		background: rgba(0, 0, 0, 0.5);
	}
	.am-share-dialog {
		position: relative;
		background: var(--dbg, #fff);
		border-radius: 16px;
		box-shadow: 0 20px 60px rgba(0,0,0,0.2);
		width: 100%;
		max-width: 480px;
		margin: 0 16px;
		overflow: hidden;
	}
	.am-share-header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 16px 24px;
		border-bottom: 1px solid var(--dbd2, #f0f0f0);
	}
	.am-share-title {
		font-size: 18px;
		font-weight: 600;
		color: var(--dt, #111);
	}
	.am-share-close {
		display: flex;
		align-items: center;
		justify-content: center;
		padding: 6px;
		border: none;
		background: none;
		color: var(--dt3, #888);
		cursor: pointer;
		border-radius: 8px;
		transition: background .15s;
	}
	.am-share-close:hover {
		background: var(--dbg3, #eee);
	}
	.am-share-content {
		padding: 20px 24px;
		display: flex;
		flex-direction: column;
		gap: 16px;
	}
	.am-share-meta {
		font-size: 13px;
		color: var(--dt3, #888);
	}
	.am-share-meta__name {
		font-weight: 500;
		color: var(--dt, #111);
	}

	/* Tabs */
	.am-share-tabs {
		display: flex;
		gap: 4px;
		padding: 4px;
		background: var(--dbg2, #f5f5f5);
		border-radius: 10px;
	}
	.am-share-tab {
		flex: 1;
		display: flex;
		align-items: center;
		justify-content: center;
		gap: 6px;
		padding: 8px 16px;
		border: none;
		border-radius: 8px;
		background: transparent;
		color: var(--dt3, #888);
		font-size: 13px;
		font-weight: 500;
		cursor: pointer;
		transition: all .15s;
	}
	.am-share-tab:hover {
		color: var(--dt, #111);
	}
	.am-share-tab--active {
		background: var(--dbg, #fff);
		color: var(--dt, #111);
		box-shadow: 0 1px 3px rgba(0,0,0,0.08);
	}

	/* Fields */
	.am-share-field {
		display: flex;
		flex-direction: column;
	}
	.am-share-label {
		font-size: 13px;
		font-weight: 500;
		color: var(--dt2, #555);
		margin-bottom: 8px;
	}
	.am-share-req {
		color: var(--color-error, #ef4444);
	}
	.am-share-input {
		width: 100%;
		padding: 8px 12px;
		border: 1px solid var(--dbd, #e0e0e0);
		border-radius: 8px;
		background: var(--dbg2, #f5f5f5);
		color: var(--dt, #111);
		font-size: 13px;
		outline: none;
		transition: border-color .15s;
	}
	.am-share-input:focus {
		border-color: var(--accent-blue, #3b82f6);
	}
	.am-share-info {
		background: rgba(59, 130, 246, 0.06);
		border: 1px solid rgba(59, 130, 246, 0.15);
		border-radius: 10px;
		padding: 14px 16px;
		font-size: 13px;
		color: var(--accent-blue, #3b82f6);
	}

	/* Permission cards */
	.am-perm-list {
		display: flex;
		flex-direction: column;
		gap: 8px;
	}
	.am-perm-card {
		width: 100%;
		display: flex;
		align-items: flex-start;
		gap: 12px;
		padding: 12px;
		border: 1px solid var(--dbd, #e0e0e0);
		border-radius: 10px;
		background: transparent;
		cursor: pointer;
		transition: all .15s;
		text-align: left;
	}
	.am-perm-card:hover {
		border-color: var(--dbd2, #f0f0f0);
	}
	.am-perm-card--active {
		border-color: var(--accent-blue, #3b82f6);
		background: rgba(59, 130, 246, 0.04);
	}
	.am-perm-check {
		flex-shrink: 0;
		width: 20px;
		height: 20px;
		border: 2px solid var(--dbd, #e0e0e0);
		border-radius: 4px;
		display: flex;
		align-items: center;
		justify-content: center;
		color: #fff;
		transition: all .15s;
	}
	.am-perm-check--active {
		border-color: var(--accent-blue, #3b82f6);
		background: var(--accent-blue, #3b82f6);
	}
	.am-perm-info__title {
		font-size: 13px;
		font-weight: 500;
		color: var(--dt, #111);
	}
	.am-perm-info__desc {
		font-size: 12px;
		color: var(--dt3, #888);
	}

	/* Footer */
	.am-share-footer {
		display: flex;
		align-items: center;
		justify-content: flex-end;
		gap: 10px;
		padding: 14px 24px;
		border-top: 1px solid var(--dbd2, #f0f0f0);
		background: var(--dbg2, #f5f5f5);
	}
	/* Foundation glow modifier for primary CTAs */
	.am-glow {
		box-shadow:
			0 1px 0 0 rgba(255, 255, 255, 0.1) inset,
			0 4px 16px 0 rgba(99, 102, 241, 0.25),
			0 8px 32px 0 rgba(99, 102, 241, 0.15);
	}
	.am-glow:hover:not(:disabled) {
		box-shadow:
			0 1px 0 0 rgba(255, 255, 255, 0.15) inset,
			0 6px 24px 0 rgba(99, 102, 241, 0.35),
			0 12px 40px 0 rgba(99, 102, 241, 0.2);
	}
</style>
