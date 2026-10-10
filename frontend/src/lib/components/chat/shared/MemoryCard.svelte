<script lang="ts" generics="T extends MemoryListItem">
	import type { MemoryListItem, MemoryType } from '$lib/api/memory';
	import { api } from '$lib/api';

	interface Props {
		memory: T;
		onPin?: (memory: T) => void | Promise<void>;
		onClick?: (memory: T) => void;
		onDelete?: (memory: T) => void | Promise<void>;
		isSelected?: boolean;
	}

	let { memory, onPin, onClick, onDelete, isSelected = false }: Props = $props();
	let pinLoading = $state(false);

	function getMemoryIcon(type: MemoryType): string {
		switch (type) {
			case 'fact':
				return `<path stroke-linecap="round" stroke-linejoin="round" d="M9.879 7.519c1.171-1.025 3.071-1.025 4.242 0 1.172 1.025 1.172 2.687 0 3.712-.203.179-.43.326-.67.442-.745.361-1.45.999-1.45 1.827v.75M21 12a9 9 0 1 1-18 0 9 9 0 0 1 18 0Zm-9 5.25h.008v.008H12v-.008Z" />`;
			case 'preference':
				return `<path stroke-linecap="round" stroke-linejoin="round" d="M10.5 6h9.75M10.5 6a1.5 1.5 0 1 1-3 0m3 0a1.5 1.5 0 1 0-3 0M3.75 6H7.5m3 12h9.75m-9.75 0a1.5 1.5 0 0 1-3 0m3 0a1.5 1.5 0 0 0-3 0m-3.75 0H7.5m9-6h3.75m-3.75 0a1.5 1.5 0 0 1-3 0m3 0a1.5 1.5 0 0 0-3 0m-9.75 0h9.75" />`;
			case 'decision':
				return `<path stroke-linecap="round" stroke-linejoin="round" d="M3.75 13.5l10.5-11.25L12 10.5h8.25L9.75 21.75 12 13.5H3.75z" />`;
			case 'event':
				return `<path stroke-linecap="round" stroke-linejoin="round" d="M6.75 3v2.25M17.25 3v2.25M3 18.75V7.5a2.25 2.25 0 0 1 2.25-2.25h13.5A2.25 2.25 0 0 1 21 7.5v11.25m-18 0A2.25 2.25 0 0 0 5.25 21h13.5A2.25 2.25 0 0 0 21 18.75m-18 0v-7.5A2.25 2.25 0 0 1 5.25 9h13.5A2.25 2.25 0 0 1 21 11.25v7.5" />`;
			case 'learning':
				return `<path stroke-linecap="round" stroke-linejoin="round" d="M4.26 10.147a60.438 60.438 0 0 0-.491 6.347A48.62 48.62 0 0 1 12 20.904a48.62 48.62 0 0 1 8.232-4.41 60.46 60.46 0 0 0-.491-6.347m-15.482 0a50.636 50.636 0 0 0-2.658-.813A59.906 59.906 0 0 1 12 3.493a59.903 59.903 0 0 1 10.399 5.84c-.896.248-1.783.52-2.658.814m-15.482 0A50.717 50.717 0 0 1 12 13.489a50.702 50.702 0 0 1 7.74-3.342M6.75 15a.75.75 0 1 0 0-1.5.75.75 0 0 0 0 1.5Zm0 0v-3.675A55.378 55.378 0 0 1 12 8.443m-7.007 11.55A5.981 5.981 0 0 0 6.75 15.75v-1.5" />`;
			case 'context':
				return `<path stroke-linecap="round" stroke-linejoin="round" d="M6.429 9.75 2.25 12l4.179 2.25m0-4.5 5.571 3 5.571-3m-11.142 0L2.25 7.5 12 2.25l9.75 5.25-4.179 2.25m0 0L21.75 12l-4.179 2.25m0 0 4.179 2.25L12 21.75 2.25 16.5l4.179-2.25m11.142 0-5.571 3-5.571-3" />`;
			case 'relationship':
				return `<path stroke-linecap="round" stroke-linejoin="round" d="M15 19.128a9.38 9.38 0 0 0 2.625.372 9.337 9.337 0 0 0 4.121-.952 4.125 4.125 0 0 0-7.533-2.493M15 19.128v-.003c0-1.113-.285-2.16-.786-3.07M15 19.128v.106A12.318 12.318 0 0 1 8.624 21c-2.331 0-4.512-.645-6.374-1.766l-.001-.109a6.375 6.375 0 0 1 11.964-3.07M12 6.375a3.375 3.375 0 1 1-6.75 0 3.375 3.375 0 0 1 6.75 0Zm8.25 2.25a2.625 2.625 0 1 1-5.25 0 2.625 2.625 0 0 1 5.25 0Z" />`;
			default:
				return `<path stroke-linecap="round" stroke-linejoin="round" d="M9.879 7.519c1.171-1.025 3.071-1.025 4.242 0 1.172 1.025 1.172 2.687 0 3.712-.203.179-.43.326-.67.442-.745.361-1.45.999-1.45 1.827v.75M21 12a9 9 0 1 1-18 0 9 9 0 0 1 18 0Zm-9 5.25h.008v.008H12v-.008Z" />`;
		}
	}

	function getMemoryColor(type: MemoryType): string {
		switch (type) {
			case 'fact': return '#3b82f6';
			case 'preference': return '#8b5cf6';
			case 'decision': return '#f59e0b';
			case 'event': return '#22c55e';
			case 'learning': return '#ec4899';
			case 'context': return '#06b6d4';
			case 'relationship': return '#6366f1';
			default: return '#6b7280';
		}
	}

	function formatDate(dateStr: string): string {
		const date = new Date(dateStr);
		const now = new Date();
		const diff = now.getTime() - date.getTime();
		const days = Math.floor(diff / (1000 * 60 * 60 * 24));

		if (days === 0) return 'Today';
		if (days === 1) return 'Yesterday';
		if (days < 7) return `${days} days ago`;
		return date.toLocaleDateString();
	}

	async function handlePin(e: Event) {
		e.stopPropagation();
		if (pinLoading) return;

		pinLoading = true;
		try {
			await api.pinMemory(memory.id, !memory.is_pinned);
			onPin?.(memory);
		} catch (error) {
			console.error('Failed to pin memory:', error);
		} finally {
			pinLoading = false;
		}
	}

	function handleDelete(e: Event) {
		e.stopPropagation();
		onDelete?.(memory);
	}
</script>

<div
	class="memory-card"
	class:pinned={memory.is_pinned}
	class:selected={isSelected}
	onclick={() => onClick?.(memory)}
	onkeydown={(e) => e.key === 'Enter' && onClick?.(memory)}
	role="button"
	tabindex="0"
>
	<div class="memory-header">
		<div class="memory-checkbox {isSelected ? 'checked' : ''}">
			{#if isSelected}
				<svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="2.5" stroke="currentColor" width="12" height="12">
					<path stroke-linecap="round" stroke-linejoin="round" d="m4.5 12.75 6 6 9-13.5" />
				</svg>
			{/if}
		</div>
		<div class="memory-icon" style="color: {getMemoryColor(memory.memory_type)}">
			<svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" width="16" height="16">
				{@html getMemoryIcon(memory.memory_type)}
			</svg>
		</div>
		<div class="memory-meta">
			<span class="memory-type">{memory.memory_type}</span>
			<span class="memory-date">{formatDate(memory.created_at)}</span>
		</div>
		<div class="memory-actions">
			<button
				class="btn-pill btn-pill-ghost action-btn"
				class:active={memory.is_pinned}
				onclick={handlePin}
				disabled={pinLoading}
				aria-label={memory.is_pinned ? 'Unpin memory' : 'Pin memory'}
			>
				<svg xmlns="http://www.w3.org/2000/svg" fill={memory.is_pinned ? 'currentColor' : 'none'} viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" width="14" height="14">
					<path stroke-linecap="round" stroke-linejoin="round" d="M16.5 3.75V16.5L12 14.25 7.5 16.5V3.75m9 0H18A2.25 2.25 0 0 1 20.25 6v12A2.25 2.25 0 0 1 18 20.25H6A2.25 2.25 0 0 1 3.75 18V6A2.25 2.25 0 0 1 6 3.75h1.5m9 0h-9" />
				</svg>
			</button>
			{#if onDelete}
				<button
					class="btn-pill btn-pill-ghost action-btn delete"
					onclick={handleDelete}
					aria-label="Delete memory"
				>
					<svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" width="14" height="14">
						<path stroke-linecap="round" stroke-linejoin="round" d="m14.74 9-.346 9m-4.788 0L9.26 9m9.968-3.21c.342.052.682.107 1.022.166m-1.022-.165L18.16 19.673a2.25 2.25 0 0 1-2.244 2.077H8.084a2.25 2.25 0 0 1-2.244-2.077L4.772 5.79m14.456 0a48.108 48.108 0 0 0-3.478-.397m-12 .562c.34-.059.68-.114 1.022-.165m0 0a48.11 48.11 0 0 1 3.478-.397m7.5 0v-.916c0-1.18-.91-2.164-2.09-2.201a51.964 51.964 0 0 0-3.32 0c-1.18.037-2.09 1.022-2.09 2.201v.916m7.5 0a48.667 48.667 0 0 0-7.5 0" />
					</svg>
				</button>
			{/if}
		</div>
	</div>

	<div class="memory-content">
		<h4 class="memory-title">{memory.title}</h4>
		{#if memory.summary}
			<p class="memory-summary">{memory.summary}</p>
		{/if}
	</div>

	{#if memory.tags && memory.tags.length > 0}
		<div class="memory-tags">
			{#each memory.tags.slice(0, 3) as tag}
				<span class="tag">{tag}</span>
			{/each}
			{#if memory.tags.length > 3}
				<span class="tag more">+{memory.tags.length - 3}</span>
			{/if}
		</div>
	{/if}

	<div class="memory-footer">
		<span class="importance" title="Importance score">
			<svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" width="12" height="12">
				<path stroke-linecap="round" stroke-linejoin="round" d="M3 13.125C3 12.504 3.504 12 4.125 12h2.25c.621 0 1.125.504 1.125 1.125v6.75C7.5 20.496 6.996 21 6.375 21h-2.25A1.125 1.125 0 0 1 3 19.875v-6.75ZM9.75 8.625c0-.621.504-1.125 1.125-1.125h2.25c.621 0 1.125.504 1.125 1.125v11.25c0 .621-.504 1.125-1.125 1.125h-2.25a1.125 1.125 0 0 1-1.125-1.125V8.625ZM16.5 4.125c0-.621.504-1.125 1.125-1.125h2.25C20.496 3 21 3.504 21 4.125v15.75c0 .621-.504 1.125-1.125 1.125h-2.25a1.125 1.125 0 0 1-1.125-1.125V4.125Z" />
			</svg>
			{(memory.importance_score * 100).toFixed(0)}%
		</span>
		{#if memory.access_count > 0}
			<span class="access-count" title="Times accessed">
				<svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" width="12" height="12">
					<path stroke-linecap="round" stroke-linejoin="round" d="M2.036 12.322a1.012 1.012 0 0 1 0-.639C3.423 7.51 7.36 4.5 12 4.5c4.638 0 8.573 3.007 9.963 7.178.07.207.07.431 0 .639C20.577 16.49 16.64 19.5 12 19.5c-4.638 0-8.573-3.007-9.963-7.178Z" />
					<path stroke-linecap="round" stroke-linejoin="round" d="M15 12a3 3 0 1 1-6 0 3 3 0 0 1 6 0Z" />
				</svg>
				{memory.access_count}
			</span>
		{/if}
	</div>
</div>

<style>
	.memory-card {
		display: flex;
		flex-direction: column;
		gap: 8px;
		width: 100%;
		padding: 12px;
		background: var(--color-bg);
		border: 1px solid var(--color-border);
		border-radius: 10px;
		cursor: pointer;
		text-align: left;
		transition: all 0.15s ease;
	}

	.memory-card:hover {
		background: var(--color-bg-secondary);
		border-color: var(--color-border);
	}

	.memory-card.pinned {
		border-color: rgba(59, 130, 246, 0.4);
		background: rgba(59, 130, 246, 0.05);
	}

	.memory-card.selected {
		border-color: rgba(34, 197, 94, 0.4);
		background: rgba(34, 197, 94, 0.08);
	}

	:global(.dark) .memory-card {
		background: #2c2c2e;
		border-color: rgba(255, 255, 255, 0.1);
	}

	:global(.dark) .memory-card:hover {
		background: #3a3a3c;
	}

	:global(.dark) .memory-card.pinned {
		border-color: rgba(59, 130, 246, 0.5);
		background: rgba(59, 130, 246, 0.1);
	}

	:global(.dark) .memory-card.selected {
		border-color: rgba(34, 197, 94, 0.5);
		background: rgba(34, 197, 94, 0.15);
	}

	.memory-header {
		display: flex;
		align-items: center;
		gap: 8px;
	}

	.memory-checkbox {
		display: flex;
		align-items: center;
		justify-content: center;
		width: 18px;
		height: 18px;
		border: 2px solid var(--color-border);
		border-radius: 4px;
		flex-shrink: 0;
		transition: all 0.15s ease;
	}

	.memory-checkbox.checked {
		background: #22c55e;
		border-color: #22c55e;
		color: white;
	}

	:global(.dark) .memory-checkbox {
		border-color: rgba(255, 255, 255, 0.3);
	}

	.memory-icon {
		display: flex;
		align-items: center;
		justify-content: center;
		width: 28px;
		height: 28px;
		background: var(--color-bg-tertiary);
		border-radius: 6px;
		flex-shrink: 0;
	}

	:global(.dark) .memory-icon {
		background: #3a3a3c;
	}

	.memory-meta {
		flex: 1;
		display: flex;
		flex-direction: column;
		gap: 1px;
		min-width: 0;
	}

	.memory-type {
		font-size: 10px;
		font-weight: 600;
		text-transform: uppercase;
		letter-spacing: 0.5px;
		color: var(--color-text-muted);
	}

	:global(.dark) .memory-type {
		color: #a1a1a6;
	}

	.memory-date {
		font-size: 11px;
		color: var(--color-text-muted);
		opacity: 0.7;
	}

	:global(.dark) .memory-date {
		color: #6e6e73;
	}

	.memory-actions {
		display: flex;
		gap: 4px;
	}

	.action-btn {
		display: flex;
		align-items: center;
		justify-content: center;
		width: 24px;
		height: 24px;
		border: none;
		background: transparent;
		color: var(--color-text-muted);
		cursor: pointer;
		border-radius: 4px;
		transition: all 0.15s ease;
	}

	.action-btn:hover {
		background: var(--color-bg-secondary);
		color: var(--color-text);
	}

	.action-btn.active {
		color: #3b82f6;
	}

	.action-btn.delete:hover {
		color: #ef4444;
	}

	.action-btn:disabled {
		opacity: 0.5;
		cursor: not-allowed;
	}

	:global(.dark) .action-btn {
		color: #6e6e73;
	}

	:global(.dark) .action-btn:hover {
		background: #4a4a4c;
		color: #f5f5f7;
	}

	.memory-content {
		display: flex;
		flex-direction: column;
		gap: 4px;
	}

	.memory-title {
		font-size: 13px;
		font-weight: 600;
		color: var(--color-text);
		margin: 0;
		line-height: 1.4;
	}

	:global(.dark) .memory-title {
		color: #f5f5f7;
	}

	.memory-summary {
		font-size: 12px;
		color: var(--color-text-muted);
		margin: 0;
		line-height: 1.5;
		display: -webkit-box;
		-webkit-line-clamp: 2;
		-webkit-box-orient: vertical;
		overflow: hidden;
	}

	:global(.dark) .memory-summary {
		color: #a1a1a6;
	}

	.memory-tags {
		display: flex;
		flex-wrap: wrap;
		gap: 4px;
	}

	.tag {
		font-size: 10px;
		padding: 2px 6px;
		background: var(--color-bg-tertiary);
		color: var(--color-text-muted);
		border-radius: 4px;
	}

	.tag.more {
		background: transparent;
		color: var(--color-text-muted);
		opacity: 0.7;
	}

	:global(.dark) .tag {
		background: #3a3a3c;
		color: #a1a1a6;
	}

	.memory-footer {
		display: flex;
		align-items: center;
		gap: 12px;
		padding-top: 4px;
		border-top: 1px solid var(--color-border);
	}

	:global(.dark) .memory-footer {
		border-top-color: rgba(255, 255, 255, 0.06);
	}

	.importance,
	.access-count {
		display: flex;
		align-items: center;
		gap: 4px;
		font-size: 11px;
		color: var(--color-text-muted);
	}

	:global(.dark) .importance,
	:global(.dark) .access-count {
		color: #6e6e73;
	}
</style>
