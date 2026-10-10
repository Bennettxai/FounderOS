<script lang="ts">
	import type { Block, BlockType, DividerStyle } from '../../entities/types';
	import { createBlock } from '../../services/documents.service';
	import SlashMenu from './SlashMenu.svelte';
	import FormatToolbar from './FormatToolbar.svelte';
	import BlockControls from './BlockControls.svelte';
	import BlockContentRenderer from './BlockContentRenderer.svelte';
	import { parseHTMLToRichText } from './blockUtils';
	import { extractRichTextPlainText, plainTextToRichText } from '../../utils/rich-text';
	import {
		createContenteditableSyncAction,
		createContenteditableSyncState,
		type ContenteditableSyncState
	} from './contenteditable-sync';
	import { detectMarkdownShortcut } from './markdown-shortcuts';
	import {
		allowBlockDrop,
		clearBlockDragOver,
		clearBlockDragState,
		createBlockDragState,
		getBlockDragOverPosition,
		getBlockDropMove,
		readBlockDragIndex,
		setBlockDragOver,
		startBlockDrag,
		writeBlockDragData
	} from './block-drag';
	import {
		applySelectionFormat,
		getHiddenSelectionToolbarState,
		getSelectionToolbarState,
		type SelectionFormat
	} from './selection-formatting';
	import {
		closeSlashMenu,
		createSlashMenuState,
		getSlashMenuStateForContentInput,
		selectSlashMenuItem
	} from './slash-menu-state';
	import {
		closeDividerPickerState,
		createDividerStyleBlock,
		isDividerPickerOutsideTarget,
		toggleDividerPickerState
	} from './divider-picker';

	interface Props {
		block: Block;
		index: number;
		readOnly?: boolean;
		shouldFocus?: boolean;
		isFirstBlock?: boolean;
		isOnlyEmptyBlock?: boolean;
		onBlockChange?: (block: Block) => void;
		onBlockDelete?: () => void;
		onBlockAdd?: (block: Block) => void;
		onFocused?: () => void;
		onBlockMove?: (fromIndex: number, toIndex: number) => void;
		totalBlocks?: number;
	}

	let {
		block,
		index,
		readOnly = false,
		shouldFocus = false,
		isFirstBlock = false,
		isOnlyEmptyBlock = false,
		onBlockChange,
		onBlockDelete,
		onBlockAdd,
		onFocused,
		onBlockMove,
		totalBlocks = 0
	}: Props = $props();

	// Determine placeholder text
	const getPlaceholder = (type: string): string => {
		if (isOnlyEmptyBlock && type === 'paragraph') {
			return "Press '/' for commands...";
		}
		switch (type) {
			case 'heading_1': return 'Heading 1';
			case 'heading_2': return 'Heading 2';
			case 'heading_3': return 'Heading 3';
			case 'bulleted_list': return 'List item';
			case 'numbered_list': return 'List item';
			case 'to_do': return 'To-do';
			case 'quote': return 'Quote';
			case 'code': return 'Code';
			case 'callout': return 'Callout';
			default: return '';
		}
	};

	let isHovered = $state(false);

	// Slash menu state
	let slashMenuState = $state(createSlashMenuState());
	let slashMenuRef: { handleKeydown: (e: KeyboardEvent) => void } | undefined;
	let blockElementRef: HTMLElement | null = $state(null);

	// Format toolbar state
	let showFormatToolbar = $state(false);
	let formatToolbarPosition = $state({ x: 0, y: 0 });
	let currentSelection = $state<ReturnType<typeof getSelectionToolbarState>['selection']>(null);

	// Divider style picker state
	let showDividerPicker = $state(false);

	// Drag and drop state
	let blockDragState = $state(createBlockDragState());

	// Track block ID to detect external changes (document switch)
	let contenteditableSyncState = $state<ContenteditableSyncState>(createContenteditableSyncState());

	// Initialize tracking state from block prop
	$effect(() => {
		if (contenteditableSyncState.lastBlockId === '') {
			const initialState = createContenteditableSyncState({
				blockId: block.id,
				content: block.content
			});
			contenteditableSyncState.lastBlockId = initialState.lastBlockId;
			contenteditableSyncState.lastContent = initialState.lastContent;
		}
	});

	function getPlainText(): string {
		return extractRichTextPlainText(block.content);
	}

	// Action for contenteditable elements - handles initial content and external updates
	const contenteditable = createContenteditableSyncAction({
		getSnapshot: () => ({ blockId: block.id, content: block.content }),
		state: contenteditableSyncState,
		onElementChange: (node) => {
			blockElementRef = node;
		}
	});

	$effect(() => {
		const content = getPlainText();
		if (block.id !== contenteditableSyncState.lastBlockId) {
			contenteditableSyncState.lastBlockId = block.id;
			contenteditableSyncState.lastContent = content;
		}
	});

	$effect(() => {
		if (shouldFocus && blockElementRef) {
			blockElementRef.focus();
			const selection = window.getSelection();
			if (selection) {
				const range = document.createRange();
				range.selectNodeContents(blockElementRef);
				range.collapse(false);
				selection.removeAllRanges();
				selection.addRange(range);
			}
			onFocused?.();
		}
	});

	function handleContentInput(e: Event) {
		const target = e.target as HTMLElement;
		const newContent = target.textContent || '';

		contenteditableSyncState.lastContent = newContent;

		slashMenuState = getSlashMenuStateForContentInput({
			content: newContent,
			state: slashMenuState,
			selection: window.getSelection(),
			targetRect: target.getBoundingClientRect()
		});

		onBlockChange?.({ ...block, content: parseHTMLToRichText(target.innerHTML, newContent) });
	}

	function applyMarkdownShortcut(type: BlockType, properties: Record<string, unknown> = {}) {
		if (blockElementRef) {
			blockElementRef.textContent = '';
		}
		contenteditableSyncState.lastContent = '';
		onBlockChange?.({
			...block,
			type,
			content: plainTextToRichText(''),
			properties: { ...block.properties, ...properties }
		});
	}

	function handleKeydown(e: KeyboardEvent) {
		if (slashMenuState.show) {
			if (e.key === 'ArrowDown' || e.key === 'ArrowUp' || e.key === 'Escape') {
				e.preventDefault();
				slashMenuRef?.handleKeydown(e);
				return;
			}
			if (e.key === 'Enter') {
				e.preventDefault();
				slashMenuRef?.handleKeydown(e);
				return;
			}
		}

		const isMac = navigator.platform.toUpperCase().indexOf('MAC') >= 0;
		const modifier = isMac ? e.metaKey : e.ctrlKey;

		if (modifier && !e.shiftKey) {
			switch (e.key.toLowerCase()) {
				case 'b':
					e.preventDefault();
					applySelectionKeyboardFormat('bold');
					return;
				case 'i':
					e.preventDefault();
					applySelectionKeyboardFormat('italic');
					return;
				case 'u':
					e.preventDefault();
					applySelectionKeyboardFormat('underline');
					return;
				case 'e':
					e.preventDefault();
					applySelectionKeyboardFormat('code');
					return;
				case 'k':
					e.preventDefault();
					applySelectionKeyboardFormat('link');
					return;
			}
		}

		if (modifier && e.shiftKey && e.key.toLowerCase() === 's') {
			e.preventDefault();
			applySelectionKeyboardFormat('strikethrough');
			return;
		}

		if (e.key === ' ') {
			const shortcut = detectMarkdownShortcut(blockElementRef?.textContent || getPlainText());
			if (shortcut) {
				e.preventDefault();
				applyMarkdownShortcut(shortcut.type, shortcut.properties);
				return;
			}
		}

		if (e.key === 'Escape' && showFormatToolbar) {
			hideFormatToolbar();
			return;
		}

		if (e.key === 'Enter' && !e.shiftKey) {
			e.preventDefault();
			onBlockAdd?.(createBlock('paragraph'));
		}
		if (e.key === 'Backspace' && getPlainText() === '') {
			e.preventDefault();
			onBlockDelete?.();
		}
	}

	function handleSlashMenuSelect(newBlock: Block) {
		const target = blockElementRef;
		if (target) {
			target.textContent = '';
		}
		onBlockChange?.({ ...block, content: plainTextToRichText('') });
		onBlockAdd?.(newBlock);
		slashMenuState = selectSlashMenuItem(slashMenuState);
	}

	function handleSlashMenuClose() {
		slashMenuState = closeSlashMenu(slashMenuState);
	}

	function showDividerStylePicker() {
		showDividerPicker = toggleDividerPickerState(showDividerPicker);
	}

	function closeDividerPicker() {
		showDividerPicker = closeDividerPickerState();
	}

	$effect(() => {
		if (showDividerPicker) {
			const handleClickOutside = (e: MouseEvent) => {
				if (isDividerPickerOutsideTarget(e.target)) {
					closeDividerPicker();
				}
			};
			document.addEventListener('click', handleClickOutside);
			return () => document.removeEventListener('click', handleClickOutside);
		}
	});

	function selectDividerStyle(style: string) {
		onBlockChange?.(createDividerStyleBlock(block, style as DividerStyle));
		showDividerPicker = closeDividerPickerState();
	}

	function handleSelectionChange() {
		const toolbarState = getSelectionToolbarState({ readOnly, blockElement: blockElementRef });
		showFormatToolbar = toolbarState.show;
		formatToolbarPosition = toolbarState.position;
		currentSelection = toolbarState.selection;
	}

	function hideFormatToolbar() {
		const toolbarState = getHiddenSelectionToolbarState();
		showFormatToolbar = toolbarState.show;
		formatToolbarPosition = toolbarState.position;
		currentSelection = toolbarState.selection;
	}

	function applySelectionKeyboardFormat(format: SelectionFormat) {
		const selection = window.getSelection();
		if (!selection || selection.isCollapsed || selection.rangeCount === 0) return;

		const selectedText = selection.toString();
		const formatSelection = currentSelection ?? {
			start: 0,
			end: selectedText.length,
			text: selectedText
		};

		if (applySelectionFormat(format, { currentSelection: formatSelection, blockElement: blockElementRef })) {
			updateBlockFromDOM();
			hideFormatToolbar();
		}
	}

	function handleFormat(format: SelectionFormat) {
		if (applySelectionFormat(format, { currentSelection, blockElement: blockElementRef })) {
			updateBlockFromDOM();
		}
		hideFormatToolbar();
	}

	function updateBlockFromDOM() {
		if (!blockElementRef) return;

		const html = blockElementRef.innerHTML;
		const text = blockElementRef.textContent || '';
		const richText = parseHTMLToRichText(html, text);

		contenteditableSyncState.lastContent = text;
		onBlockChange?.({ ...block, content: richText });
	}

	function handleMouseUp() {
		setTimeout(handleSelectionChange, 10);
	}

	function handleDragStart(e: DragEvent) {
		if (readOnly) return;

		if (e.dataTransfer) {
			writeBlockDragData(e.dataTransfer, { index, blockId: block.id });
		}
		blockDragState = startBlockDrag(blockDragState);
	}

	function handleDragEnd() {
		blockDragState = clearBlockDragState(blockDragState);
	}

	function handleDragOver(e: DragEvent) {
		if (readOnly) return;

		e.preventDefault();
		if (e.dataTransfer) {
			allowBlockDrop(e.dataTransfer);
		}

		const rect = (e.currentTarget as HTMLElement).getBoundingClientRect();
		blockDragState = setBlockDragOver(blockDragState, getBlockDragOverPosition(e.clientY, rect));
	}

	function handleDragLeave() {
		blockDragState = clearBlockDragOver(blockDragState);
	}

	function handleDrop(e: DragEvent) {
		e.preventDefault();
		const position = blockDragState.dragOverPosition;
		blockDragState = clearBlockDragOver(blockDragState);

		if (readOnly || !e.dataTransfer || !position) return;

		const fromIndex = readBlockDragIndex(e.dataTransfer);
		if (fromIndex === null) return;

		const move = getBlockDropMove({ fromIndex, targetIndex: index, position });
		if (move) {
			onBlockMove?.(move.fromIndex, move.toIndex);
		}
	}

	function handleCheckboxChange(e: Event) {
		const target = e.target as HTMLInputElement;
		onBlockChange?.({
			...block,
			properties: { ...block.properties, checked: target.checked }
		});
	}

	function handleToggleExpand() {
		onBlockChange?.({
			...block,
			properties: { ...block.properties, expanded: !block.properties.expanded }
		});
	}

	function addBlockOfType(type: BlockType) {
		onBlockAdd?.(createBlock(type));
	}
</script>

<!-- svelte-ignore a11y_no_static_element_interactions -->
<div
	class="block-wrapper"
	class:block-wrapper--hovered={isHovered}
	class:block-wrapper--dragging={blockDragState.isDragging}
	class:block-wrapper--drag-over={blockDragState.isDragOver}
	class:block-wrapper--drag-before={blockDragState.dragOverPosition === 'before'}
	class:block-wrapper--drag-after={blockDragState.dragOverPosition === 'after'}
	onmouseenter={() => (isHovered = true)}
	onmouseleave={() => (isHovered = false)}
	ondragover={handleDragOver}
	ondragleave={handleDragLeave}
	ondrop={handleDrop}
>
	{#if !readOnly}
		<BlockControls
			onAddBlock={addBlockOfType}
			onDragStart={handleDragStart}
			onDragEnd={handleDragEnd}
		/>
	{/if}

	<BlockContentRenderer
		{block}
		{index}
		{readOnly}
		contenteditableAction={contenteditable}
		onContentInput={handleContentInput}
		onKeydown={handleKeydown}
		onMouseUp={handleMouseUp}
		onCheckboxChange={handleCheckboxChange}
		onToggleExpand={handleToggleExpand}
		onDividerClick={showDividerStylePicker}
		onSelectDividerStyle={selectDividerStyle}
		onCloseDividerPicker={closeDividerPicker}
		onBlockChange={(updated) => onBlockChange?.(updated)}
		{showDividerPicker}
		{getPlaceholder}
	/>

	<!-- Slash command menu -->
	<SlashMenu
		visible={slashMenuState.show}
		position={slashMenuState.position}
		filter={slashMenuState.filter}
		onSelect={handleSlashMenuSelect}
		onClose={handleSlashMenuClose}
		bind:this={slashMenuRef}
	/>

	<!-- Format toolbar for text selection -->
	<FormatToolbar
		visible={showFormatToolbar}
		position={formatToolbarPosition}
		onFormat={handleFormat}
	/>
</div>

<style>
	.block-wrapper {
		position: relative;
		display: flex;
		align-items: flex-start;
		gap: 0.25rem;
		padding: 2px 0;
		padding-left: 52px;
		margin-left: -52px;
	}

	.block-wrapper--dragging {
		opacity: 0.5;
	}

	.block-wrapper--drag-over {
		position: relative;
	}

	.block-wrapper--drag-before::before {
		content: '';
		position: absolute;
		top: 0;
		left: 52px;
		right: 0;
		height: 2px;
		background-color: #1e96eb;
		border-radius: 1px;
	}

	.block-wrapper--drag-after::after {
		content: '';
		position: absolute;
		bottom: 0;
		left: 52px;
		right: 0;
		height: 2px;
		background-color: #1e96eb;
		border-radius: 1px;
	}
</style>
