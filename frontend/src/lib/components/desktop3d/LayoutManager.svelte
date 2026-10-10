<script lang="ts">
	import { desktop3dLayoutStore, type Layout } from '$lib/stores/desktop3dLayoutStore';

	interface Props {
		show: boolean;
		onClose: () => void;
	}

	let { show, onClose }: Props = $props();

	// Local state for UI interactions only
	let deleteConfirmId = $state<string | null>(null);
	let deleting = $state(false);

	async function handleLoadLayout(layoutId: string) {
		if (layoutId === $desktop3dLayoutStore.activeLayoutId) return;

		if (import.meta.env.DEV) console.log('[LayoutManager] Loading layout:', layoutId);
		await desktop3dLayoutStore.loadLayout(layoutId);
	}

	function handleOpenDeleteConfirm(layoutId: string) {
		if (layoutId === 'default') return;
		deleteConfirmId = layoutId;
	}

	function handleCloseDeleteConfirm() {
		deleteConfirmId = null;
	}

	async function handleConfirmDelete() {
		if (!deleteConfirmId) return;

		deleting = true;
		const success = await desktop3dLayoutStore.deleteLayout(deleteConfirmId);

		if (success) {
			if (import.meta.env.DEV) console.log('[LayoutManager] ✅ Layout deleted:', deleteConfirmId);
			handleCloseDeleteConfirm();
		} else {
			console.error('[LayoutManager] Failed to delete layout');
		}

		deleting = false;
	}

	function formatDate(date: Date | string): string {
		const d = typeof date === 'string' ? new Date(date) : date;
		return d.toLocaleDateString('en-US', {
			month: 'short',
			day: 'numeric',
			year: 'numeric',
			hour: 'numeric',
			minute: '2-digit'
		});
	}
</script>

{#if show}
	<!-- Modal Overlay -->
	<div class="modal-overlay" role="presentation" onclick={onClose}>
		<div class="modal-content" role="dialog" aria-modal="true" tabindex="-1" onclick={(e) => e.stopPropagation()}>
			<!-- Header -->
			<div class="modal-header">
				<div class="header-content">
					<svg
						class="header-icon"
						width="24"
						height="24"
						viewBox="0 0 24 24"
						fill="none"
						stroke="currentColor"
						stroke-width="2"
					>
						<rect x="3" y="3" width="7" height="7" />
						<rect x="14" y="3" width="7" height="7" />
						<rect x="14" y="14" width="7" height="7" />
						<rect x="3" y="14" width="7" height="7" />
					</svg>
					<h2>Layout Manager</h2>
				</div>

				<button class="btn-pill btn-pill-ghost close-btn" onclick={onClose} aria-label="Close layout manager">
					<svg
						width="20"
						height="20"
						viewBox="0 0 24 24"
						fill="none"
						stroke="currentColor"
						stroke-width="2"
					>
						<line x1="18" y1="6" x2="6" y2="18" />
						<line x1="6" y1="6" x2="18" y2="18" />
					</svg>
				</button>
			</div>

			<!-- Body -->
			<div class="modal-body">
				{#if $desktop3dLayoutStore.loading}
					<div class="loading-state">
						<div class="spinner large"></div>
						<p>Loading layouts...</p>
					</div>
				{:else if $desktop3dLayoutStore.error}
					<div class="error-state">
						<svg
							width="48"
							height="48"
							viewBox="0 0 24 24"
							fill="none"
							stroke="currentColor"
							stroke-width="2"
						>
							<circle cx="12" cy="12" r="10" />
							<line x1="12" y1="8" x2="12" y2="12" />
							<line x1="12" y1="16" x2="12.01" y2="16" />
						</svg>
						<p>{$desktop3dLayoutStore.error}</p>
						<button class="btn-pill btn-pill-ghost btn btn-primary" onclick={() => desktop3dLayoutStore.loadLayouts()}>
							Retry
						</button>
					</div>
				{:else if $desktop3dLayoutStore.layouts.length === 0}
					<div class="empty-state">
						<svg
							width="64"
							height="64"
							viewBox="0 0 24 24"
							fill="none"
							stroke="currentColor"
							stroke-width="2"
						>
							<rect x="3" y="3" width="7" height="7" />
							<rect x="14" y="3" width="7" height="7" />
							<rect x="14" y="14" width="7" height="7" />
							<rect x="3" y="14" width="7" height="7" />
						</svg>
						<p>No layouts found</p>
						<p class="empty-hint">Create a custom layout by entering edit mode and saving your setup</p>
					</div>
				{:else}
					<div class="layouts-grid">
						{#each $desktop3dLayoutStore.layouts as layout (layout.id)}
							<div
								class="layout-card"
								class:active={layout.id === $desktop3dLayoutStore.activeLayoutId}
								class:default={layout.type === 'default'}
							>
								<!-- Card Header -->
								<div class="card-header">
									<div class="card-title">
										<div class="title-row">
											<h3>{layout.name}</h3>
											{#if layout.type === 'default'}
												<span class="badge badge-default">Default</span>
											{/if}
											{#if layout.id === $desktop3dLayoutStore.activeLayoutId}
												<span class="badge badge-active">Active</span>
											{/if}
										</div>
										<p class="card-date">{formatDate(layout.created_at)}</p>
									</div>

									{#if layout.type === 'custom'}
										<button
											class="btn-pill btn-pill-ghost delete-btn"
											onclick={() => handleOpenDeleteConfirm(layout.id)}
											title="Delete layout"
										>
											<svg
												width="18"
												height="18"
												viewBox="0 0 24 24"
												fill="none"
												stroke="currentColor"
												stroke-width="2"
											>
												<polyline points="3 6 5 6 21 6" />
												<path
													d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"
												/>
											</svg>
										</button>
									{/if}
								</div>

								<!-- Card Body -->
								<div class="card-body">
									<div class="stat">
										<svg
											width="16"
											height="16"
											viewBox="0 0 24 24"
											fill="none"
											stroke="currentColor"
											stroke-width="2"
										>
											<rect x="3" y="3" width="7" height="7" />
											<rect x="14" y="3" width="7" height="7" />
											<rect x="14" y="14" width="7" height="7" />
											<rect x="3" y="14" width="7" height="7" />
										</svg>
										<span>{layout.modules.length} modules</span>
									</div>
								</div>

								<!-- Card Actions -->
								<div class="card-actions">
									{#if layout.id === $desktop3dLayoutStore.activeLayoutId}
										<button class="btn-pill btn-pill-ghost btn btn-secondary" disabled>
											<svg
												width="16"
												height="16"
												viewBox="0 0 24 24"
												fill="none"
												stroke="currentColor"
												stroke-width="2"
											>
												<polyline points="20 6 9 17 4 12" />
											</svg>
											Current Layout
										</button>
									{:else}
										<button class="btn-pill btn-pill-ghost btn btn-primary" onclick={() => handleLoadLayout(layout.id)}>
											Load Layout
										</button>
									{/if}
								</div>
							</div>
						{/each}
					</div>
				{/if}
			</div>
		</div>
	</div>

	<!-- Delete Confirmation Modal -->
	{#if deleteConfirmId}
		{@const layoutToDelete = $desktop3dLayoutStore.layouts.find((l) => l.id === deleteConfirmId)}
		<div class="confirm-overlay" role="presentation" onclick={handleCloseDeleteConfirm}>
			<div class="confirm-content" role="alertdialog" aria-modal="true" tabindex="-1" onclick={(e) => e.stopPropagation()}>
				<div class="confirm-icon">
					<svg
						width="48"
						height="48"
						viewBox="0 0 24 24"
						fill="none"
						stroke="currentColor"
						stroke-width="2"
					>
						<circle cx="12" cy="12" r="10" />
						<line x1="12" y1="8" x2="12" y2="12" />
						<line x1="12" y1="16" x2="12.01" y2="16" />
					</svg>
				</div>

				<h3>Delete Layout?</h3>
				<p>
					Are you sure you want to delete "<strong>{layoutToDelete?.name}</strong>"? This action
					cannot be undone.
				</p>

				<div class="confirm-actions">
					<button class="btn-pill btn-pill-ghost btn btn-secondary" onclick={handleCloseDeleteConfirm} disabled={deleting}>
						Cancel
					</button>
					<button class="btn-pill btn-pill-ghost btn btn-danger" onclick={handleConfirmDelete} disabled={deleting}>
						{#if deleting}
							<span class="spinner"></span>
							Deleting...
						{:else}
							Delete
						{/if}
					</button>
				</div>
			</div>
		</div>
	{/if}
{/if}

<style>
	.modal-overlay {
		position: fixed;
		top: 0;
		left: 0;
		right: 0;
		bottom: 0;
		background: rgba(0, 0, 0, 0.6);
		backdrop-filter: blur(4px);
		display: flex;
		align-items: center;
		justify-content: center;
		z-index: 3000;
		animation: fadeIn 0.2s ease-out;
	}

	@keyframes fadeIn {
		from {
			opacity: 0;
		}
		to {
			opacity: 1;
		}
	}

	.modal-content {
		background: var(--dbg);
		border-radius: 20px;
		width: 90%;
		max-width: 900px;
		max-height: 85vh;
		display: flex;
		flex-direction: column;
		box-shadow: 0 25px 80px rgba(0, 0, 0, 0.4);
		animation: slideUp 0.3s ease-out;
	}

	@keyframes slideUp {
		from {
			opacity: 0;
			transform: translateY(30px);
		}
		to {
			opacity: 1;
			transform: translateY(0);
		}
	}

	.modal-header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 28px 32px;
		border-bottom: 1px solid var(--dbd);
	}

	.header-content {
		display: flex;
		align-items: center;
		gap: 12px;
	}

	.header-icon {
		color: var(--bos-status-info);
	}

	.modal-header h2 {
		margin: 0;
		font-size: 24px;
		font-weight: 600;
		color: var(--dt);
	}

	.close-btn {
		background: none;
		border: none;
		padding: 8px;
		cursor: pointer;
		color: var(--dt2);
		transition: all 0.2s;
		border-radius: 8px;
	}

	.close-btn:hover {
		background: var(--dbg2);
		color: var(--dt);
	}

	.modal-body {
		padding: 24px 32px;
		overflow-y: auto;
		flex: 1;
	}

	.loading-state,
	.error-state,
	.empty-state {
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		padding: 60px 20px;
		text-align: center;
	}

	.loading-state svg,
	.error-state svg,
	.empty-state svg {
		color: #94a3b8;
		margin-bottom: 16px;
	}

	.loading-state p,
	.error-state p,
	.empty-state p {
		color: #64748b;
		font-size: 16px;
		margin: 0;
	}

	.empty-hint {
		font-size: 14px !important;
		color: #94a3b8 !important;
		max-width: 400px;
		margin-top: 8px !important;
	}

	.spinner {
		display: inline-block;
		width: 16px;
		height: 16px;
		border: 2px solid rgba(255, 255, 255, 0.3);
		border-top-color: white;
		border-radius: 50%;
		animation: spin 0.6s linear infinite;
	}

	.spinner.large {
		width: 48px;
		height: 48px;
		border-width: 4px;
		border-color: #e2e8f0;
		border-top-color: #3b82f6;
	}

	@keyframes spin {
		to {
			transform: rotate(360deg);
		}
	}

	.layouts-grid {
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
		gap: 20px;
	}

	.layout-card {
		background: #f8fafc;
		border: 2px solid #e2e8f0;
		border-radius: 16px;
		padding: 20px;
		transition: all 0.2s;
		display: flex;
		flex-direction: column;
		gap: 16px;
	}

	.layout-card:hover {
		border-color: #cbd5e1;
		transform: translateY(-2px);
		box-shadow: 0 8px 24px rgba(0, 0, 0, 0.1);
	}

	.layout-card.active {
		background: linear-gradient(135deg, #eff6ff, #dbeafe);
		border-color: #3b82f6;
	}

	.layout-card.default {
		background: linear-gradient(135deg, #fef3c7, #fde68a);
		border-color: #f59e0b;
	}

	.card-header {
		display: flex;
		justify-content: space-between;
		align-items: flex-start;
	}

	.card-title {
		flex: 1;
	}

	.title-row {
		display: flex;
		align-items: center;
		gap: 8px;
		flex-wrap: wrap;
	}

	.card-title h3 {
		margin: 0;
		font-size: 18px;
		font-weight: 600;
		color: var(--dt);
	}

	.card-date {
		margin: 4px 0 0 0;
		font-size: 13px;
		color: var(--dt2);
	}

	.badge {
		padding: 4px 10px;
		border-radius: 6px;
		font-size: 11px;
		font-weight: 600;
		text-transform: uppercase;
		letter-spacing: 0.5px;
	}

	.badge-default {
		background: #fbbf24;
		color: #78350f;
	}

	.badge-active {
		background: #3b82f6;
		color: var(--bos-surface-on-color);
	}

	.delete-btn {
		background: none;
		border: none;
		padding: 8px;
		cursor: pointer;
		color: var(--dt2);
		transition: all 0.2s;
		border-radius: 6px;
	}

	.delete-btn:hover {
		background: #fee2e2;
		color: #dc2626;
	}

	.card-body {
		flex: 1;
	}

	.stat {
		display: flex;
		align-items: center;
		gap: 8px;
		color: var(--dt2);
		font-size: 14px;
	}

	.stat svg {
		flex-shrink: 0;
	}

	.card-actions {
		display: flex;
		gap: 8px;
	}

	.btn {
		display: flex;
		align-items: center;
		justify-content: center;
		gap: 6px;
		padding: 10px 18px;
		border: none;
		border-radius: 10px;
		font-size: 14px;
		font-weight: 600;
		cursor: pointer;
		transition: all 0.2s;
		flex: 1;
	}

	.btn:disabled {
		opacity: 0.6;
		cursor: not-allowed;
	}

	.btn-primary {
		background: linear-gradient(135deg, #3b82f6, #2563eb);
		color: var(--bos-surface-on-color);
	}

	.btn-primary:hover:not(:disabled) {
		transform: translateY(-1px);
		box-shadow: 0 4px 12px rgba(59, 130, 246, 0.4);
	}

	.btn-secondary {
		background: var(--dbg);
		color: #475569;
		border: 2px solid var(--dbd);
	}

	.btn-secondary:hover:not(:disabled) {
		background: var(--dbg2);
		border-color: #cbd5e1;
	}

	.btn-danger {
		background: linear-gradient(135deg, #ef4444, #dc2626);
		color: var(--bos-surface-on-color);
	}

	.btn-danger:hover:not(:disabled) {
		transform: translateY(-1px);
		box-shadow: 0 4px 12px rgba(239, 68, 68, 0.4);
	}

	/* Delete Confirmation Modal */
	.confirm-overlay {
		position: fixed;
		top: 0;
		left: 0;
		right: 0;
		bottom: 0;
		background: rgba(0, 0, 0, 0.7);
		backdrop-filter: blur(4px);
		display: flex;
		align-items: center;
		justify-content: center;
		z-index: 4000;
		animation: fadeIn 0.2s ease-out;
	}

	.confirm-content {
		background: var(--dbg);
		border-radius: 16px;
		width: 90%;
		max-width: 420px;
		padding: 32px;
		text-align: center;
		box-shadow: 0 20px 60px rgba(0, 0, 0, 0.3);
		animation: slideUp 0.3s ease-out;
	}

	.confirm-icon {
		margin-bottom: 16px;
	}

	.confirm-icon svg {
		color: #ef4444;
	}

	.confirm-content h3 {
		margin: 0 0 12px 0;
		font-size: 20px;
		font-weight: 600;
		color: var(--dt);
	}

	.confirm-content p {
		margin: 0 0 24px 0;
		color: var(--dt2);
		font-size: 15px;
		line-height: 1.6;
	}

	.confirm-content strong {
		color: var(--dt);
		font-weight: 600;
	}

	.confirm-actions {
		display: flex;
		gap: 12px;
	}
</style>
