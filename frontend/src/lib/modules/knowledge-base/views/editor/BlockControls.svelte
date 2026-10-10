<script lang="ts">
	import type { BlockType } from '../../entities/types';
	import { GripVertical, Plus } from 'lucide-svelte';
	import { Menu, MenuItem, MenuLabel } from '$lib/ui';

	interface Props {
		onAddBlock: (type: BlockType) => void;
		onDragStart: (e: DragEvent) => void;
		onDragEnd: () => void;
	}

	let { onAddBlock, onDragStart, onDragEnd }: Props = $props();

	const blockTypes: { type: BlockType; label: string; icon: string }[] = [
		{ type: 'paragraph', label: 'Text', icon: 'T' },
		{ type: 'heading_1', label: 'Heading 1', icon: 'H1' },
		{ type: 'heading_2', label: 'Heading 2', icon: 'H2' },
		{ type: 'heading_3', label: 'Heading 3', icon: 'H3' },
		{ type: 'bulleted_list', label: 'Bulleted List', icon: '•' },
		{ type: 'numbered_list', label: 'Numbered List', icon: '1.' },
		{ type: 'to_do', label: 'To-do', icon: '[]' },
		{ type: 'toggle', label: 'Toggle', icon: '>' },
		{ type: 'quote', label: 'Quote', icon: '"' },
		{ type: 'divider', label: 'Divider', icon: '—' },
		{ type: 'code', label: 'Code', icon: '</>' },
		{ type: 'callout', label: 'Callout', icon: '!' },
		{ type: 'table', label: 'Table', icon: '#' }
	];

	let showAddMenu = $state(false);
</script>

<div class="block-wrapper__controls">
	<Menu bind:open={showAddMenu}>
		{#snippet trigger()}
			<button class="btn-pill btn-pill-ghost block-wrapper__btn" aria-label="Add block">
				<Plus class="h-3.5 w-3.5" />
			</button>
		{/snippet}
		<MenuLabel>Add block</MenuLabel>
		{#each blockTypes as bt}
			<MenuItem onSelect={() => onAddBlock(bt.type)}>
				<span class="block-type-icon">{bt.icon}</span>
				{bt.label}
			</MenuItem>
		{/each}
	</Menu>

	<button
		class="btn-pill btn-pill-ghost block-wrapper__btn block-wrapper__drag"
		aria-label="Drag block"
		draggable="true"
		ondragstart={onDragStart}
		ondragend={onDragEnd}
	>
		<GripVertical class="h-3.5 w-3.5" />
	</button>
</div>

<style>
	.block-wrapper__controls {
		position: absolute;
		left: 0;
		top: 2px;
		display: flex;
		align-items: center;
		gap: 2px;
		opacity: 0;
		transition: opacity 0.15s;
	}

	:global(.block-wrapper--hovered) .block-wrapper__controls {
		opacity: 1;
	}

	.block-wrapper__btn {
		display: flex;
		align-items: center;
		justify-content: center;
		width: 20px;
		height: 20px;
		padding: 0;
		background: transparent;
		border: none;
		border-radius: 0.25rem;
		color: var(--dt3);
		cursor: pointer;
		transition: background-color 0.1s;
	}

	.block-wrapper__btn:hover {
		background-color: var(--dbg2);
	}

	.block-wrapper__drag {
		cursor: grab;
	}

	.block-wrapper__drag:active {
		cursor: grabbing;
	}

	.block-type-icon {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		width: 20px;
		height: 20px;
		margin-right: 0.5rem;
		font-size: 0.75rem;
		font-weight: 500;
		color: var(--dt3);
	}
</style>
