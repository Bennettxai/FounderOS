<script lang="ts">
	/**
	 * Table Header - Column header with resize and menu
	 */
	import { ChevronDown, ArrowUpDown, EyeOff, Trash2, Plus } from 'lucide-svelte';
	import { Menu, MenuItem, MenuSeparator, Tooltip } from '$lib/ui';
	import ColumnTypeIcon from './ColumnTypeIcon.svelte';
	import type { ColumnSchema, ColumnType } from '../../entities/block';

	interface Props {
		column: ColumnSchema;
		width?: number;
		sortDirection?: 'asc' | 'desc' | null;
		onSort?: () => void;
		onHide?: () => void;
		onDelete?: () => void;
		onRename?: (name: string) => void;
		onChangeType?: (type: ColumnType) => void;
		onResize?: (width: number) => void;
	}

	let {
		column,
		width = 180,
		sortDirection = null,
		onSort,
		onHide,
		onDelete,
		onRename,
		onChangeType,
		onResize
	}: Props = $props();

	let showMenu = $state(false);
	let isEditing = $state(false);
	let editValue = $state(column.name);
	let isResizing = $state(false);
	let startX = $state(0);
	let startWidth = $state(0);

	function handleDoubleClick() {
		isEditing = true;
		editValue = column.name;
	}

	function handleEditKeydown(e: KeyboardEvent) {
		if (e.key === 'Enter') {
			onRename?.(editValue);
			isEditing = false;
		} else if (e.key === 'Escape') {
			isEditing = false;
			editValue = column.name;
		}
	}

	function handleEditBlur() {
		if (editValue !== column.name) {
			onRename?.(editValue);
		}
		isEditing = false;
	}

	function handleResizeStart(e: MouseEvent) {
		e.preventDefault();
		isResizing = true;
		startX = e.clientX;
		startWidth = width;

		document.addEventListener('mousemove', handleResizeMove);
		document.addEventListener('mouseup', handleResizeEnd);
	}

	function handleResizeMove(e: MouseEvent) {
		if (!isResizing) return;
		const delta = e.clientX - startX;
		const newWidth = Math.max(80, startWidth + delta);
		onResize?.(newWidth);
	}

	function handleResizeEnd() {
		isResizing = false;
		document.removeEventListener('mousemove', handleResizeMove);
		document.removeEventListener('mouseup', handleResizeEnd);
	}

	const columnTypes: { type: ColumnType; label: string }[] = [
		{ type: 'text', label: 'Text' },
		{ type: 'number', label: 'Number' },
		{ type: 'select', label: 'Select' },
		{ type: 'multi-select', label: 'Multi-select' },
		{ type: 'date', label: 'Date' },
		{ type: 'checkbox', label: 'Checkbox' },
		{ type: 'url', label: 'URL' },
		{ type: 'email', label: 'Email' },
		{ type: 'phone', label: 'Phone' },
		{ type: 'person', label: 'Person' },
		{ type: 'file', label: 'File' },
		{ type: 'relation', label: 'Relation' }
	];
</script>

<th
	class="bos-table-header"
	class:bos-table-header--resizing={isResizing}
	style:width="{width}px"
	style:min-width="{width}px"
>
	<div class="bos-table-header__content">
		{#if isEditing}
			<input
				type="text"
				class="bos-table-header__input"
				bind:value={editValue}
				onkeydown={handleEditKeydown}
				onblur={handleEditBlur}
				autofocus
			/>
		{:else}
			<button
				class="btn-pill btn-pill-ghost bos-table-header__button"
				ondblclick={handleDoubleClick}
			>
				<ColumnTypeIcon type={column.type} />
				<span class="bos-table-header__name">{column.name}</span>
				{#if sortDirection}
					<span class="bos-table-header__sort" class:bos-table-header__sort--desc={sortDirection === 'desc'}>
						<ArrowUpDown />
					</span>
				{/if}
			</button>

			<Menu bind:open={showMenu}>
				{#snippet trigger()}
					<button class="bos-table-header__menu-trigger">
						<ChevronDown />
					</button>
				{/snippet}

				<MenuItem onSelect={onSort}>
					{#snippet prefix()}<ArrowUpDown />{/snippet}
					Sort {sortDirection === 'asc' ? 'descending' : 'ascending'}
				</MenuItem>

				<MenuItem onSelect={onHide}>
					{#snippet prefix()}<EyeOff />{/snippet}
					Hide column
				</MenuItem>

				<MenuSeparator />

				<div class="bos-table-header__type-section">
					<span class="bos-table-header__type-label">Column type</span>
					{#each columnTypes as ct}
						<button
							class="btn-pill btn-pill-ghost bos-table-header__type-option"
							class:bos-table-header__type-option--active={column.type === ct.type}
							onclick={() => {
								onChangeType?.(ct.type);
								showMenu = false;
							}}
						>
							<ColumnTypeIcon type={ct.type} />
							<span>{ct.label}</span>
						</button>
					{/each}
				</div>

				<MenuSeparator />

				<MenuItem destructive onSelect={onDelete}>
					{#snippet prefix()}<Trash2 />{/snippet}
					Delete column
				</MenuItem>
			</Menu>
		{/if}
	</div>

	<!-- Resize handle -->
	<div
		class="bos-table-header__resize"
		onmousedown={handleResizeStart}
		role="separator"
		aria-orientation="vertical"
		tabindex={-1}
	></div>
</th>

<style>
	.bos-table-header {
		position: relative;
		background: var(--bos-v2-layer-background-secondary, #f4f4f5);
		border-bottom: 1px solid var(--bos-v2-layer-insideBorder-border, rgba(0, 0, 0, 0.1));
		text-align: left;
		font-weight: 500;
		user-select: none;
	}

	.bos-table-header--resizing {
		cursor: ew-resize;
	}

	.bos-table-header__content {
		display: flex;
		align-items: center;
		height: 32px;
		padding: 0 4px;
		gap: 4px;
	}

	.bos-table-header__button {
		display: flex;
		align-items: center;
		flex: 1;
		gap: 6px;
		padding: 4px 8px;
		border: none;
		background: transparent;
		border-radius: 4px;
		cursor: pointer;
		text-align: left;
		min-width: 0;
	}

	.bos-table-header__button:hover {
		background: var(--bos-v2-layer-background-hoverOverlay, rgba(0, 0, 0, 0.04));
	}

	.bos-table-header__name {
		font-size: var(--bos-font-sm, 14px);
		font-weight: 500;
		color: var(--bos-v2-text-secondary, #8e8d91);
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}

	.bos-table-header__sort {
		display: flex;
		color: var(--bos-v2-icon-secondary, #a9a9ad);
	}

	.bos-table-header__sort :global(svg) {
		width: 12px;
		height: 12px;
	}

	.bos-table-header__sort--desc {
		transform: rotate(180deg);
	}

	.bos-table-header__menu-trigger {
		display: flex;
		align-items: center;
		justify-content: center;
		width: 20px;
		height: 20px;
		padding: 0;
		border: none;
		background: transparent;
		border-radius: 4px;
		color: var(--bos-v2-icon-secondary, #a9a9ad);
		cursor: pointer;
		opacity: 0;
		transition: opacity 0.15s;
	}

	.bos-table-header:hover .bos-table-header__menu-trigger {
		opacity: 1;
	}

	.bos-table-header__menu-trigger:hover {
		background: var(--bos-v2-layer-background-hoverOverlay, rgba(0, 0, 0, 0.04));
	}

	.bos-table-header__menu-trigger :global(svg) {
		width: 14px;
		height: 14px;
	}

	.bos-table-header__input {
		flex: 1;
		height: 24px;
		padding: 2px 8px;
		border: 1px solid var(--bos-brand-color, #1e96eb);
		border-radius: 4px;
		font-size: var(--bos-font-sm, 14px);
		font-weight: 500;
		background: var(--bos-v2-layer-background-primary, #ffffff);
		color: var(--bos-v2-text-primary, #121212);
		outline: none;
	}

	.bos-table-header__resize {
		position: absolute;
		top: 0;
		right: -2px;
		width: 4px;
		height: 100%;
		cursor: ew-resize;
		background: transparent;
		z-index: 1;
	}

	.bos-table-header__resize:hover,
	.bos-table-header--resizing .bos-table-header__resize {
		background: var(--bos-brand-color, #1e96eb);
	}

	.bos-table-header__type-section {
		padding: 4px;
	}

	.bos-table-header__type-label {
		display: block;
		padding: 4px 8px;
		font-size: 12px;
		font-weight: 600;
		color: var(--bos-v2-text-tertiary, #bfbfc3);
		text-transform: uppercase;
	}

	.bos-table-header__type-option {
		display: flex;
		align-items: center;
		gap: 8px;
		width: 100%;
		padding: 6px 8px;
		border: none;
		background: transparent;
		border-radius: 4px;
		font-size: var(--bos-font-sm, 14px);
		color: var(--bos-v2-text-primary, #121212);
		cursor: pointer;
		text-align: left;
	}

	.bos-table-header__type-option:hover {
		background: var(--bos-v2-layer-background-hoverOverlay, rgba(0, 0, 0, 0.04));
	}

	.bos-table-header__type-option--active {
		background: var(--bos-v2-layer-background-hoverOverlay, rgba(0, 0, 0, 0.04));
		color: var(--bos-brand-color, #1e96eb);
	}

	/* Dark mode */
	:global(.dark) .bos-table-header {
		background: var(--bos-v2-layer-background-secondary, #2c2c2c);
		border-color: var(--bos-v2-layer-insideBorder-border, rgba(255, 255, 255, 0.1));
	}

	:global(.dark) .bos-table-header__button:hover {
		background: var(--bos-v2-layer-background-hoverOverlay, rgba(255, 255, 255, 0.08));
	}

	:global(.dark) .bos-table-header__name {
		color: var(--bos-v2-text-secondary, #8e8d91);
	}

	:global(.dark) .bos-table-header__input {
		background: var(--bos-v2-layer-background-primary, #1e1e1e);
		color: var(--bos-v2-text-primary, #e6e6e6);
	}

	:global(.dark) .bos-table-header__type-option {
		color: var(--bos-v2-text-primary, #e6e6e6);
	}

	:global(.dark) .bos-table-header__type-option:hover,
	:global(.dark) .bos-table-header__type-option--active {
		background: var(--bos-v2-layer-background-hoverOverlay, rgba(255, 255, 255, 0.08));
	}
</style>
