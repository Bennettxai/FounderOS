<script lang="ts" generics="T extends MemoryListItem">
	import type { MemoryListItem } from '$lib/api/memory';
	import { fade, fly } from 'svelte/transition';

	interface Props {
		memory: T | null;
		onClose: () => void;
		onPin?: (memory: T) => void | Promise<void>;
		onDelete?: (memory: T) => void | Promise<void>;
	}

	let { memory, onClose, onPin, onDelete }: Props = $props();

	function handleBackdropClick(e: MouseEvent) {
		if (e.target === e.currentTarget) {
			onClose();
		}
	}

	function handlePin() {
		if (memory && onPin) {
			onPin(memory);
		}
	}

	function handleDelete() {
		if (memory && onDelete && confirm('Are you sure you want to delete this memory?')) {
			onDelete(memory);
			onClose();
		}
	}

	function formatDate(dateString: string) {
		const date = new Date(dateString);
		return new Intl.DateTimeFormat('en-US', {
			dateStyle: 'medium',
			timeStyle: 'short'
		}).format(date);
	}

	function getTypeColor(type: string) {
		const colors: Record<string, string> = {
			fact: '#3b82f6',
			preference: '#8b5cf6',
			decision: '#f59e0b',
			event: '#10b981',
			learning: '#ec4899',
			context: '#6366f1',
			relationship: '#14b8a6'
		};
		return colors[type] || '#6B7280';
	}

	function getTypeBadgeClass(type: string) {
		const classes: Record<string, string> = {
			fact: 'badge-blue',
			preference: 'badge-purple',
			decision: 'badge-amber',
			event: 'badge-green',
			learning: 'badge-pink',
			context: 'badge-indigo',
			relationship: 'badge-teal'
		};
		return classes[type] || 'badge-gray';
	}
</script>

{#if memory}
	<div class="modal-backdrop" onclick={handleBackdropClick} transition:fade={{ duration: 150 }}>
		<div class="modal-content" transition:fly={{ y: 20, duration: 200 }}>
			<!-- Header -->
			<div class="modal-header">
				<div class="header-left">
					<div class="type-badge {getTypeBadgeClass(memory.memory_type)}">
						{memory.memory_type}
					</div>
					{#if memory.is_pinned}
						<div class="pinned-badge">
							<svg xmlns="http://www.w3.org/2000/svg" fill="currentColor" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" width="14" height="14">
								<path stroke-linecap="round" stroke-linejoin="round" d="M16.5 3.75V16.5L12 14.25 7.5 16.5V3.75m9 0H18A2.25 2.25 0 0 1 20.25 6v12A2.25 2.25 0 0 1 18 20.25H6A2.25 2.25 0 0 1 3.75 18V6A2.25 2.25 0 0 1 6 3.75h1.5m9 0h-9" />
							</svg>
							Pinned
						</div>
					{/if}
				</div>
				<button class="btn-pill btn-pill-ghost close-btn" onclick={onClose} aria-label="Close modal">
					<svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" width="20" height="20">
						<path stroke-linecap="round" stroke-linejoin="round" d="M6 18 18 6M6 6l12 12" />
					</svg>
				</button>
			</div>

			<!-- Content -->
			<div class="modal-body">
				<div class="content-section">
					<h3 class="section-title">Content</h3>
					<div class="content-text">{memory.content}</div>
				</div>

				{#if memory.metadata && Object.keys(memory.metadata).length > 0}
					<div class="content-section">
						<h3 class="section-title">Metadata</h3>
						<div class="metadata-grid">
							{#each Object.entries(memory.metadata) as [key, value]}
								<div class="metadata-item">
									<span class="metadata-key">{key}:</span>
									<span class="metadata-value">{JSON.stringify(value)}</span>
								</div>
							{/each}
						</div>
					</div>
				{/if}

				<div class="info-grid">
					<div class="info-item">
						<div class="info-label">Importance</div>
						<div class="importance-bar">
							<div class="importance-fill" style="width: {memory.importance_score * 100}%"></div>
							<span class="importance-text">{Math.round(memory.importance_score * 100)}%</span>
						</div>
					</div>

					<div class="info-item">
						<div class="info-label">Created</div>
						<div class="info-value">{formatDate(memory.created_at)}</div>
					</div>

					<div class="info-item">
						<div class="info-label">Last Updated</div>
						<div class="info-value">{formatDate(memory.updated_at)}</div>
					</div>

					<div class="info-item">
						<div class="info-label">Access Count</div>
						<div class="info-value">{memory.access_count || 0} times</div>
					</div>

					{#if memory.last_accessed_at}
						<div class="info-item">
							<div class="info-label">Last Accessed</div>
							<div class="info-value">{formatDate(memory.last_accessed_at)}</div>
						</div>
					{/if}
				</div>

				{#if memory.tags && memory.tags.length > 0}
					<div class="content-section">
						<h3 class="section-title">Tags</h3>
						<div class="tags-list">
							{#each memory.tags as tag}
								<span class="tag">{tag}</span>
							{/each}
						</div>
					</div>
				{/if}
			</div>

			<!-- Footer -->
			<div class="modal-footer">
				<button class="btn-pill btn-pill-ghost action-btn secondary" onclick={handlePin}>
					<svg xmlns="http://www.w3.org/2000/svg" fill={memory.is_pinned ? 'currentColor' : 'none'} viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" width="16" height="16">
						<path stroke-linecap="round" stroke-linejoin="round" d="M16.5 3.75V16.5L12 14.25 7.5 16.5V3.75m9 0H18A2.25 2.25 0 0 1 20.25 6v12A2.25 2.25 0 0 1 18 20.25H6A2.25 2.25 0 0 1 3.75 18V6A2.25 2.25 0 0 1 6 3.75h1.5m9 0h-9" />
					</svg>
					{memory.is_pinned ? 'Unpin' : 'Pin'}
				</button>
				<button class="btn-pill btn-pill-ghost action-btn danger" onclick={handleDelete}>
					<svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" width="16" height="16">
						<path stroke-linecap="round" stroke-linejoin="round" d="m14.74 9-.346 9m-4.788 0L9.26 9m9.968-3.21c.342.052.682.107 1.022.166m-1.022-.165L18.16 19.673a2.25 2.25 0 0 1-2.244 2.077H8.084a2.25 2.25 0 0 1-2.244-2.077L4.772 5.79m14.456 0a48.108 48.108 0 0 0-3.478-.397m-12 .562c.34-.059.68-.114 1.022-.165m0 0a48.11 48.11 0 0 1 3.478-.397m7.5 0v-.916c0-1.18-.91-2.164-2.09-2.201a51.964 51.964 0 0 0-3.32 0c-1.18.037-2.09 1.022-2.09 2.201v.916m7.5 0a48.667 48.667 0 0 0-7.5 0" />
					</svg>
					Delete
				</button>
			</div>
		</div>
	</div>
{/if}

<style>
	.modal-backdrop {
		position: fixed;
		top: 0;
		left: 0;
		right: 0;
		bottom: 0;
		background: rgba(0, 0, 0, 0.5);
		display: flex;
		align-items: center;
		justify-content: center;
		z-index: 1000;
		padding: 20px;
	}

	.modal-content {
		background: var(--color-bg);
		border-radius: 12px;
		width: 100%;
		max-width: 600px;
		max-height: 90vh;
		display: flex;
		flex-direction: column;
		box-shadow: 0 20px 25px -5px rgba(0, 0, 0, 0.1), 0 10px 10px -5px rgba(0, 0, 0, 0.04);
	}

	:global(.dark) .modal-content {
		background: #1c1c1e;
		box-shadow: 0 20px 25px -5px rgba(0, 0, 0, 0.5), 0 10px 10px -5px rgba(0, 0, 0, 0.3);
	}

	.modal-header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 20px;
		border-bottom: 1px solid var(--color-border);
	}

	:global(.dark) .modal-header {
		border-bottom-color: rgba(255, 255, 255, 0.1);
	}

	.header-left {
		display: flex;
		align-items: center;
		gap: 8px;
	}

	.type-badge {
		padding: 4px 12px;
		font-size: 12px;
		font-weight: 600;
		border-radius: 6px;
		text-transform: capitalize;
	}

	.badge-blue { background: rgba(59, 130, 246, 0.1); color: #3b82f6; }
	.badge-purple { background: rgba(139, 92, 246, 0.1); color: #8b5cf6; }
	.badge-amber { background: rgba(245, 158, 11, 0.1); color: #f59e0b; }
	.badge-green { background: rgba(16, 185, 129, 0.1); color: #10b981; }
	.badge-pink { background: rgba(236, 72, 153, 0.1); color: #ec4899; }
	.badge-indigo { background: rgba(99, 102, 241, 0.1); color: #6366f1; }
	.badge-teal { background: rgba(20, 184, 166, 0.1); color: #14b8a6; }
	.badge-gray { background: rgba(107, 114, 128, 0.1); color: #6B7280; }

	.pinned-badge {
		display: flex;
		align-items: center;
		gap: 4px;
		padding: 4px 10px;
		font-size: 11px;
		font-weight: 600;
		color: #8b5cf6;
		background: rgba(139, 92, 246, 0.1);
		border-radius: 6px;
	}

	.close-btn {
		display: flex;
		align-items: center;
		justify-content: center;
		width: 32px;
		height: 32px;
		border: none;
		background: transparent;
		color: var(--color-text-muted);
		cursor: pointer;
		border-radius: 6px;
		transition: all 0.15s ease;
	}

	.close-btn:hover {
		background: var(--color-bg-secondary);
		color: var(--color-text);
	}

	:global(.dark) .close-btn:hover {
		background: #2c2c2e;
	}

	.modal-body {
		flex: 1;
		overflow-y: auto;
		padding: 20px;
		display: flex;
		flex-direction: column;
		gap: 20px;
	}

	.content-section {
		display: flex;
		flex-direction: column;
		gap: 8px;
	}

	.section-title {
		font-size: 13px;
		font-weight: 600;
		color: var(--color-text-muted);
		text-transform: uppercase;
		letter-spacing: 0.5px;
		margin: 0;
	}

	:global(.dark) .section-title {
		color: #6e6e73;
	}

	.content-text {
		font-size: 15px;
		line-height: 1.6;
		color: var(--color-text);
		white-space: pre-wrap;
	}

	:global(.dark) .content-text {
		color: #f5f5f7;
	}

	.metadata-grid {
		display: flex;
		flex-direction: column;
		gap: 6px;
	}

	.metadata-item {
		display: flex;
		gap: 8px;
		font-size: 13px;
	}

	.metadata-key {
		font-weight: 600;
		color: var(--color-text-muted);
		min-width: 100px;
	}

	:global(.dark) .metadata-key {
		color: #6e6e73;
	}

	.metadata-value {
		color: var(--color-text);
		flex: 1;
		word-break: break-word;
	}

	:global(.dark) .metadata-value {
		color: #f5f5f7;
	}

	.info-grid {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(140px, 1fr));
		gap: 16px;
	}

	.info-item {
		display: flex;
		flex-direction: column;
		gap: 6px;
	}

	.info-label {
		font-size: 12px;
		font-weight: 600;
		color: var(--color-text-muted);
		text-transform: uppercase;
		letter-spacing: 0.3px;
	}

	:global(.dark) .info-label {
		color: #6e6e73;
	}

	.info-value {
		font-size: 14px;
		font-weight: 500;
		color: var(--color-text);
	}

	:global(.dark) .info-value {
		color: #f5f5f7;
	}

	.importance-bar {
		position: relative;
		height: 24px;
		background: var(--color-bg-secondary);
		border-radius: 4px;
		overflow: hidden;
		display: flex;
		align-items: center;
		justify-content: center;
	}

	:global(.dark) .importance-bar {
		background: #2c2c2e;
	}

	.importance-fill {
		position: absolute;
		left: 0;
		top: 0;
		bottom: 0;
		background: linear-gradient(90deg, #3b82f6, #8b5cf6);
		transition: width 0.3s ease;
	}

	.importance-text {
		position: relative;
		z-index: 1;
		font-size: 12px;
		font-weight: 700;
		color: var(--color-text);
	}

	.tags-list {
		display: flex;
		flex-wrap: wrap;
		gap: 6px;
	}

	.tag {
		padding: 4px 10px;
		font-size: 12px;
		font-weight: 500;
		color: var(--color-text);
		background: var(--color-bg-secondary);
		border: 1px solid var(--color-border);
		border-radius: 6px;
	}

	:global(.dark) .tag {
		background: #2c2c2e;
		color: #f5f5f7;
		border-color: rgba(255, 255, 255, 0.1);
	}

	.modal-footer {
		display: flex;
		align-items: center;
		justify-content: flex-end;
		gap: 8px;
		padding: 16px 20px;
		border-top: 1px solid var(--color-border);
	}

	:global(.dark) .modal-footer {
		border-top-color: rgba(255, 255, 255, 0.1);
	}

	.action-btn {
		display: flex;
		align-items: center;
		gap: 6px;
		padding: 8px 16px;
		font-size: 13px;
		font-weight: 500;
		border: none;
		border-radius: 6px;
		cursor: pointer;
		transition: all 0.15s ease;
	}

	.action-btn.secondary {
		color: var(--color-text);
		background: var(--color-bg-secondary);
	}

	.action-btn.secondary:hover {
		background: var(--color-bg-tertiary);
	}

	:global(.dark) .action-btn.secondary {
		background: #2c2c2e;
		color: #f5f5f7;
	}

	:global(.dark) .action-btn.secondary:hover {
		background: #3a3a3c;
	}

	.action-btn.danger {
		color: white;
		background: #ef4444;
	}

	.action-btn.danger:hover {
		background: #dc2626;
	}

	@media (max-width: 600px) {
		.modal-content {
			max-width: 100%;
			max-height: 100vh;
			border-radius: 0;
		}

		.info-grid {
			grid-template-columns: 1fr;
		}
	}
</style>
