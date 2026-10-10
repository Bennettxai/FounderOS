<script lang="ts">
	import { FileText, FileJson } from 'lucide-svelte';
	import { exportDocumentAsMarkdown, exportDocumentAsJSON } from '../../services/documents.service';
	import type { Document } from '../../entities/types';

	interface Props {
		open: boolean;
		document: Document;
	}

	let { open = $bindable(false), document }: Props = $props();

	function handleExportMd() {
		exportDocumentAsMarkdown(document);
		open = false;
	}

	function handleExportJson() {
		exportDocumentAsJSON(document);
		open = false;
	}

	function handleClose() {
		open = false;
	}

	function handleBackdropClick(e: MouseEvent) {
		if (e.target === e.currentTarget) handleClose();
	}

	function handleKeydown(e: KeyboardEvent) {
		if (e.key === 'Escape') handleClose();
	}
</script>

{#if open}
	<!-- svelte-ignore a11y_no_noninteractive_element_interactions -->
	<div class="export-backdrop" onclick={handleBackdropClick} onkeydown={handleKeydown} role="dialog" aria-modal="true" aria-label="Export document">
		<div class="export-modal">
			<div class="export-modal__header">
				<h3 class="export-modal__title">Export</h3>
				<button class="export-modal__close" onclick={handleClose} aria-label="Close">×</button>
			</div>
			<div class="export-modal__body">
				<button class="export-modal__option" onclick={handleExportMd}>
					<FileText class="h-5 w-5" />
					<div>
						<span class="export-modal__option-title">Markdown</span>
						<span class="export-modal__option-desc">Export as .md file</span>
					</div>
				</button>
				<button class="export-modal__option" onclick={handleExportJson}>
					<FileJson class="h-5 w-5" />
					<div>
						<span class="export-modal__option-title">JSON</span>
						<span class="export-modal__option-desc">Export as structured .json file</span>
					</div>
				</button>
			</div>
		</div>
	</div>
{/if}

<style>
	.export-backdrop {
		position: fixed;
		inset: 0;
		z-index: 100;
		display: flex;
		align-items: center;
		justify-content: center;
		background: rgba(0, 0, 0, 0.5);
		animation: exportBackdropIn 0.15s ease;
	}

	@keyframes exportBackdropIn {
		from { opacity: 0; }
		to { opacity: 1; }
	}

	.export-modal {
		width: 340px;
		max-width: 90vw;
		background: var(--dbg);
		border: 1px solid var(--dbd);
		border-radius: 12px;
		box-shadow: 0 16px 48px rgba(0, 0, 0, 0.2);
		animation: exportModalIn 0.2s ease;
	}

	@keyframes exportModalIn {
		from { opacity: 0; transform: scale(0.95) translateY(8px); }
		to { opacity: 1; transform: scale(1) translateY(0); }
	}

	.export-modal__header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 14px 18px 10px;
		border-bottom: 1px solid var(--dbd);
	}

	.export-modal__title {
		font-size: 15px;
		font-weight: 600;
		color: var(--dt);
		margin: 0;
	}

	.export-modal__close {
		display: flex;
		align-items: center;
		justify-content: center;
		width: 28px;
		height: 28px;
		border: none;
		background: transparent;
		border-radius: 6px;
		color: var(--dt3);
		font-size: 18px;
		cursor: pointer;
	}

	.export-modal__close:hover {
		background: var(--dbg3);
		color: var(--dt);
	}

	.export-modal__body {
		padding: 8px;
		display: flex;
		flex-direction: column;
		gap: 2px;
	}

	.export-modal__option {
		display: flex;
		align-items: center;
		gap: 12px;
		padding: 12px 14px;
		border: none;
		background: transparent;
		border-radius: 8px;
		cursor: pointer;
		text-align: left;
		color: var(--dt3);
		transition: background 0.12s;
	}

	.export-modal__option:hover {
		background: var(--dbg2);
	}

	.export-modal__option-title {
		display: block;
		font-size: 14px;
		font-weight: 500;
		color: var(--dt);
	}

	.export-modal__option-desc {
		display: block;
		font-size: 12px;
		color: var(--dt3);
		margin-top: 1px;
	}
</style>
