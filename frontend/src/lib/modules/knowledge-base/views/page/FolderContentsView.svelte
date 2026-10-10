<script lang="ts">
	import { FileText, Folder } from 'lucide-svelte';
	import type { DocumentMeta } from '../../entities/types';
	import PageListLayout from './PageListLayout.svelte';

	interface Props {
		title: string;
		children: DocumentMeta[];
		onOpenDocument: (id: string) => void;
	}

	let { title, children, onOpenDocument }: Props = $props();
</script>

<PageListLayout {title} meta={`${children.length} items`}>
	{#if children.length === 0}
		<div class="kb-folder__empty">This folder is empty</div>
	{:else}
		<div class="kb-folder__grid">
			{#each children as child (child.id)}
				<button class="kb-folder__item" onclick={() => onOpenDocument(child.id)}>
					<span class="kb-folder__icon">
						{#if child.type === 'folder'}
							<Folder size={18} />
						{:else}
							<FileText size={18} />
						{/if}
					</span>
					<span class="kb-folder__body">
						<span class="kb-folder__title">{child.title || 'Untitled'}</span>
						{#if child.type === 'folder'}
							<span class="kb-folder__meta">{child.children_count} items</span>
						{/if}
					</span>
				</button>
			{/each}
		</div>
	{/if}
</PageListLayout>

<style>
	.kb-folder__grid {
		display: flex;
		flex-direction: column;
		gap: 2px;
	}

	.kb-folder__item {
		display: flex;
		align-items: center;
		gap: 12px;
		width: 100%;
		padding: 10px 14px;
		border-radius: 8px;
		border: none;
		background: transparent;
		color: inherit;
		cursor: pointer;
		text-align: left;
	}

	.kb-folder__item:hover {
		background: var(--bos-v2-layer-background-hoverOverlay, rgba(0, 0, 0, 0.04));
	}

	.kb-folder__icon {
		display: flex;
		align-items: center;
		justify-content: center;
		width: 28px;
		height: 28px;
		flex-shrink: 0;
		color: var(--bos-v2-icon-secondary, #a9a9ad);
	}

	.kb-folder__body {
		flex: 1;
		min-width: 0;
		display: flex;
		flex-direction: column;
		gap: 2px;
	}

	.kb-folder__title {
		font-size: 14px;
		font-weight: 500;
		color: var(--bos-v2-text-primary, #121212);
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}

	.kb-folder__meta,
	.kb-folder__empty {
		font-size: 12px;
		color: var(--bos-v2-text-tertiary, #bfbfc3);
	}

	.kb-folder__empty {
		display: flex;
		justify-content: center;
		padding: 64px 32px;
	}
</style>
