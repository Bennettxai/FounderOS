<script lang="ts">
	import { FileText } from 'lucide-svelte';
	import PageListLayout from './PageListLayout.svelte';

	export interface RecentDocumentItem {
		id: string;
		title: string;
		openedAt: number;
	}

	interface Props {
		items: RecentDocumentItem[];
		formatTimeAgo: (timestamp: number) => string;
		onOpenDocument: (id: string) => void;
	}

	let { items, formatTimeAgo, onOpenDocument }: Props = $props();
</script>

<PageListLayout title="Recent">
	<div class="kb-recent">
		{#each items as item (item.id)}
			<button class="kb-recent__item" onclick={() => onOpenDocument(item.id)}>
				<span class="kb-recent__icon">
					<FileText size={16} />
				</span>
				<span class="kb-recent__title">{item.title}</span>
				<span class="kb-recent__node">{item.id.split('/')[0]}</span>
				<span class="kb-recent__time">{formatTimeAgo(item.openedAt)}</span>
			</button>
		{:else}
			<div class="kb-recent__empty">No recently opened documents</div>
		{/each}
	</div>
</PageListLayout>

<style>
	.kb-recent {
		display: flex;
		flex-direction: column;
		gap: 2px;
		padding: 0 16px 16px;
	}

	.kb-recent__item {
		display: flex;
		align-items: center;
		gap: 8px;
		width: 100%;
		padding: 7px 10px;
		border: none;
		background: transparent;
		color: var(--dt2, #666);
		font-size: 12px;
		cursor: pointer;
		border-radius: 5px;
		text-align: left;
	}

	.kb-recent__item:hover {
		background: var(--dbg2, rgba(0,0,0,0.04));
		color: var(--dt, inherit);
	}

	.kb-recent__icon {
		flex-shrink: 0;
		color: var(--dt3, #999);
	}

	.kb-recent__title {
		flex: 1;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	.kb-recent__node,
	.kb-recent__time {
		font-size: 10px;
		color: var(--dt4, #bbb);
		white-space: nowrap;
		flex-shrink: 0;
	}

	.kb-recent__time {
		color: var(--dt3, rgba(0,0,0,0.4));
	}

	.kb-recent__empty {
		padding: 64px 32px;
		text-align: center;
		color: var(--bos-v2-text-secondary, #8e8d91);
		font-size: 14px;
	}
</style>
