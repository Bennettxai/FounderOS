<script lang="ts">
	import { desktop3dLayoutStore } from '$lib/stores/desktop3dLayoutStore';

	interface Props {
		onClose: () => void;
		onSaved: () => void;
	}

	let { onClose, onSaved }: Props = $props();

	let layoutNameInput = $state('');

	function handleSave() {
		const name = layoutNameInput.trim();
		if (!name) return;
		desktop3dLayoutStore.saveLayout(name);
		desktop3dLayoutStore.exitEditMode();
		layoutNameInput = '';
		onSaved();
	}

	function handleCancel() {
		layoutNameInput = '';
		onClose();
	}

	function handleKeydown(e: KeyboardEvent) {
		if (e.key === 'Enter' && layoutNameInput.trim()) {
			handleSave();
		}
		if (e.key === 'Escape') {
			handleCancel();
		}
	}
</script>

<!-- svelte-ignore a11y_no_noninteractive_element_interactions -->
<div
	class="modal-overlay"
	role="dialog"
	aria-modal="true"
	aria-labelledby="save-layout-title"
	onclick={handleCancel}
	onkeydown={(e) => e.key === 'Escape' && handleCancel()}
>
	<!-- svelte-ignore a11y_no_noninteractive_element_interactions -->
	<div
		class="modal-content"
		role="document"
		onclick={(e) => e.stopPropagation()}
		onkeydown={() => {}}
	>
		<div class="modal-header">
			<h3 id="save-layout-title">Save Layout</h3>
			<button class="close-btn" onclick={handleCancel} aria-label="Close dialog">
				<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true">
					<line x1="18" y1="6" x2="6" y2="18" />
					<line x1="6" y1="6" x2="18" y2="18" />
				</svg>
			</button>
		</div>

		<div class="modal-body">
			<p class="modal-description">Enter a name for your custom layout:</p>
			<input
				type="text"
				class="layout-name-input"
				placeholder="e.g., My Workspace, Development Setup"
				bind:value={layoutNameInput}
				maxlength="255"
				onkeydown={handleKeydown}
				aria-label="Layout name"
			/>
		</div>

		<div class="modal-footer">
			<button class="btn btn-secondary" onclick={handleCancel}>
				Cancel
			</button>
			<button
				class="btn btn-primary"
				disabled={!layoutNameInput.trim()}
				onclick={handleSave}
			>
				Save
			</button>
		</div>
	</div>
</div>

<style>
	.modal-overlay {
		position: fixed;
		top: 0;
		left: 0;
		right: 0;
		bottom: 0;
		background: rgba(0, 0, 0, 0.5);
		backdrop-filter: blur(4px);
		display: flex;
		align-items: center;
		justify-content: center;
		z-index: 20000;
		animation: fadeIn 0.2s ease-out;
	}

	@keyframes fadeIn {
		from { opacity: 0; }
		to { opacity: 1; }
	}

	.modal-content {
		background: white;
		border-radius: 12px;
		width: 90%;
		max-width: 400px;
		box-shadow: 0 20px 60px rgba(0, 0, 0, 0.3);
		animation: slideUp 0.3s ease-out;
	}

	@keyframes slideUp {
		from {
			opacity: 0;
			transform: translateY(20px);
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
		padding: 20px;
		border-bottom: 1px solid #e2e8f0;
	}

	.modal-header h3 {
		margin: 0;
		font-size: 18px;
		font-weight: 600;
		color: #1e293b;
	}

	.close-btn {
		background: none;
		border: none;
		padding: 4px;
		cursor: pointer;
		color: #64748b;
		transition: color 0.2s;
		display: flex;
		align-items: center;
	}

	.close-btn:hover {
		color: #1e293b;
	}

	.modal-body {
		padding: 20px;
	}

	.modal-description {
		margin: 0 0 12px 0;
		color: #64748b;
		font-size: 14px;
	}

	.layout-name-input {
		width: 100%;
		padding: 10px 14px;
		border: 2px solid #e2e8f0;
		border-radius: 6px;
		font-size: 14px;
		transition: all 0.2s;
		box-sizing: border-box;
	}

	.layout-name-input:focus {
		outline: none;
		border-color: #0066FF;
		box-shadow: 0 0 0 3px rgba(0, 102, 255, 0.1);
	}

	.modal-footer {
		display: flex;
		justify-content: flex-end;
		gap: 10px;
		padding: 20px;
		border-top: 1px solid #e2e8f0;
	}

	.btn {
		display: flex;
		align-items: center;
		gap: 6px;
		padding: 8px 16px;
		border: none;
		border-radius: 6px;
		font-size: 14px;
		font-weight: 500;
		cursor: pointer;
		transition: all 0.2s;
	}

	.btn:disabled {
		opacity: 0.5;
		cursor: not-allowed;
	}

	.btn-primary {
		background: #0066FF;
		color: white;
	}

	.btn-primary:hover:not(:disabled) {
		background: #0055DD;
	}

	.btn-secondary {
		background: white;
		color: #64748b;
		border: 1px solid #e2e8f0;
	}

	.btn-secondary:hover:not(:disabled) {
		background: #f8fafc;
		border-color: #cbd5e1;
	}

	/* Dark mode */
	:global(.dark) .modal-content {
		background: #1e293b;
		border: 1px solid rgba(255, 255, 255, 0.1);
	}

	:global(.dark) .modal-header h3 {
		color: #f5f5f7;
	}

	:global(.dark) .modal-header {
		border-bottom-color: rgba(255, 255, 255, 0.1);
	}

	:global(.dark) .close-btn {
		color: #a1a1a6;
	}

	:global(.dark) .close-btn:hover {
		color: #f5f5f7;
	}

	:global(.dark) .modal-description {
		color: #a1a1a6;
	}

	:global(.dark) .layout-name-input {
		background: #0f172a;
		border-color: rgba(255, 255, 255, 0.1);
		color: #f5f5f7;
	}

	:global(.dark) .layout-name-input:focus {
		border-color: #0A84FF;
		box-shadow: 0 0 0 3px rgba(10, 132, 255, 0.2);
	}

	:global(.dark) .modal-footer {
		border-top-color: rgba(255, 255, 255, 0.1);
	}

	:global(.dark) .btn-primary {
		background: #0A84FF;
	}

	:global(.dark) .btn-primary:hover:not(:disabled) {
		background: #0077FF;
	}

	:global(.dark) .btn-secondary {
		background: #334155;
		border-color: rgba(255, 255, 255, 0.1);
		color: #f5f5f7;
	}

	:global(.dark) .btn-secondary:hover:not(:disabled) {
		background: #475569;
		border-color: rgba(255, 255, 255, 0.15);
	}
</style>
