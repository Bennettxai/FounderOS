<script lang="ts">
	/**
	 * SlashMenu - Command palette for inserting blocks
	 * Shows when user types "/" in a block
	 */
	import { createBlock } from '../../services/documents.service';
	import type { Block, BlockType } from '../../entities/types';
	import {
		Type,
		Heading1,
		Heading2,
		Heading3,
		List,
		ListOrdered,
		CheckSquare,
		ChevronRight,
		Code,
		Quote,
		Lightbulb,
		Minus,
		Image,
		Link,
		Table,
		Database
	} from 'lucide-svelte';

	interface Props {
		visible: boolean;
		position: { x: number; y: number };
		filter?: string;
		onSelect: (block: Block) => void;
		onClose: () => void;
	}

	let { visible, position, filter = '', onSelect, onClose }: Props = $props();

	let menuRef: HTMLDivElement | null = $state(null);

	// Click-outside handler
	function handleClickOutside(e: MouseEvent) {
		if (menuRef && !menuRef.contains(e.target as Node)) {
			onClose();
		}
	}

	// Add/remove event listener based on visibility
	$effect(() => {
		if (visible) {
			// Use setTimeout to avoid immediate close from the same click
			setTimeout(() => {
				document.addEventListener('click', handleClickOutside);
			}, 0);
		}
		return () => {
			document.removeEventListener('click', handleClickOutside);
		};
	});

	// Command definitions with categories
	const commands: {
		category: string;
		items: { type: BlockType; label: string; description: string; icon: typeof Type; keywords: string[]; shortcut?: string }[];
	}[] = [
		{
			category: 'Basic blocks',
			items: [
				{
					type: 'paragraph',
					label: 'Text',
					description: 'Just start writing with plain text.',
					icon: Type,
					keywords: ['text', 'paragraph', 'plain']
				},
				{
					type: 'heading_1',
					label: 'Heading 1',
					description: 'Big section heading.',
					icon: Heading1,
					keywords: ['h1', 'heading', 'title', 'header'],
					shortcut: '#'
				},
				{
					type: 'heading_2',
					label: 'Heading 2',
					description: 'Medium section heading.',
					icon: Heading2,
					keywords: ['h2', 'heading', 'subtitle'],
					shortcut: '##'
				},
				{
					type: 'heading_3',
					label: 'Heading 3',
					description: 'Small section heading.',
					icon: Heading3,
					keywords: ['h3', 'heading', 'subheading'],
					shortcut: '###'
				}
			]
		},
		{
			category: 'Lists',
			items: [
				{
					type: 'bulleted_list',
					label: 'Bulleted list',
					description: 'Create a simple bulleted list.',
					icon: List,
					keywords: ['bullet', 'list', 'unordered', 'ul'],
					shortcut: '-'
				},
				{
					type: 'numbered_list',
					label: 'Numbered list',
					description: 'Create a list with numbering.',
					icon: ListOrdered,
					keywords: ['number', 'list', 'ordered', 'ol'],
					shortcut: '1.'
				},
				{
					type: 'to_do',
					label: 'To-do list',
					description: 'Track tasks with a to-do list.',
					icon: CheckSquare,
					keywords: ['todo', 'task', 'checkbox', 'check'],
					shortcut: '[]'
				},
				{
					type: 'toggle',
					label: 'Toggle list',
					description: 'Toggles can hide and show content.',
					icon: ChevronRight,
					keywords: ['toggle', 'collapse', 'expand', 'accordion'],
					shortcut: '>'
				}
			]
		},
		{
			category: 'Advanced blocks',
			items: [
				{
					type: 'code',
					label: 'Code',
					description: 'Capture a code snippet.',
					icon: Code,
					keywords: ['code', 'snippet', 'programming', 'syntax'],
					shortcut: '```'
				},
				{
					type: 'quote',
					label: 'Quote',
					description: 'Capture a quote.',
					icon: Quote,
					keywords: ['quote', 'blockquote', 'citation'],
					shortcut: '"'
				},
				{
					type: 'callout',
					label: 'Callout',
					description: 'Make writing stand out.',
					icon: Lightbulb,
					keywords: ['callout', 'note', 'tip', 'warning', 'info']
				},
				{
					type: 'divider',
					label: 'Divider',
					description: 'Visually divide blocks.',
					icon: Minus,
					keywords: ['divider', 'separator', 'line', 'hr'],
					shortcut: '---'
				}
			]
		},
		{
			category: 'Media',
			items: [
				{
					type: 'image',
					label: 'Image',
					description: 'Upload or embed an image.',
					icon: Image,
					keywords: ['image', 'photo', 'picture', 'img']
				},
				{
					type: 'bookmark',
					label: 'Bookmark',
					description: 'Save a link as a visual bookmark.',
					icon: Link,
					keywords: ['bookmark', 'link', 'url', 'embed']
				}
			]
		},
		{
			category: 'Database',
			items: [
				{
					type: 'table',
					label: 'Table',
					description: 'Add a simple table.',
					icon: Table,
					keywords: ['table', 'grid', 'spreadsheet']
				},
				{
					type: 'database_view',
					label: 'Database',
					description: 'Create a linked database view.',
					icon: Database,
					keywords: ['database', 'db', 'notion', 'view']
				}
			]
		}
	];

	// Filter commands based on search
	const filteredCommands = $derived(() => {
		if (!filter) return commands;

		const searchLower = filter.toLowerCase();
		return commands
			.map((category) => ({
				...category,
				items: category.items.filter(
					(item) =>
						item.label.toLowerCase().includes(searchLower) ||
						item.description.toLowerCase().includes(searchLower) ||
						item.keywords.some((k) => k.includes(searchLower))
				)
			}))
			.filter((category) => category.items.length > 0);
	});

	// Flatten for keyboard navigation
	const flatItems = $derived(() => {
		return filteredCommands().flatMap((cat) => cat.items);
	});

	let selectedIndex = $state(0);

	// Reset selection when filter changes
	$effect(() => {
		filter; // dependency
		selectedIndex = 0;
	});

	function handleKeydown(e: KeyboardEvent) {
		const items = flatItems();
		if (!items.length) return;

		if (e.key === 'ArrowDown') {
			e.preventDefault();
			selectedIndex = (selectedIndex + 1) % items.length;
		} else if (e.key === 'ArrowUp') {
			e.preventDefault();
			selectedIndex = (selectedIndex - 1 + items.length) % items.length;
		} else if (e.key === 'Enter') {
			e.preventDefault();
			const selected = items[selectedIndex];
			if (selected) {
				onSelect(createBlock(selected.type));
			}
		} else if (e.key === 'Escape') {
			e.preventDefault();
			onClose();
		}
	}

	function handleItemClick(type: BlockType) {
		onSelect(createBlock(type));
	}

	// Track item index across categories
	function getGlobalIndex(categoryIndex: number, itemIndex: number): number {
		let index = 0;
		for (let i = 0; i < categoryIndex; i++) {
			index += filteredCommands()[i]?.items.length || 0;
		}
		return index + itemIndex;
	}

	// Export keyboard handler for parent to attach
	export { handleKeydown };
</script>

{#if visible}
	<!-- svelte-ignore a11y_no_static_element_interactions -->
	<div
		class="slash-menu"
		style="left: {position.x}px; top: {position.y}px;"
		onkeydown={handleKeydown}
		bind:this={menuRef}
	>
		<div class="slash-menu__header">
			{#if filter}
				<span class="slash-menu__filter">/{filter}</span>
			{:else}
				<span class="slash-menu__label">Type to filter...</span>
			{/if}
		</div>

		<div class="slash-menu__content">
			{#each filteredCommands() as category, categoryIndex}
				<div class="slash-menu__category">
					{#if categoryIndex > 0}
						<div class="slash-menu__category-label">{category.category}</div>
					{/if}
					{#each category.items as item, itemIndex}
						{@const globalIndex = getGlobalIndex(categoryIndex, itemIndex)}
						<button
							class="btn-pill btn-pill-ghost slash-menu__item"
							class:slash-menu__item--selected={selectedIndex === globalIndex}
							onclick={() => handleItemClick(item.type)}
							onmouseenter={() => (selectedIndex = globalIndex)}
						>
							<div class="slash-menu__item-icon">
								<item.icon class="h-5 w-5" />
							</div>
							<div class="slash-menu__item-content">
								<div class="slash-menu__item-label">{item.label}</div>
								<div class="slash-menu__item-description">{item.description}</div>
							</div>
							{#if item.shortcut}
								<div class="slash-menu__item-shortcut">{item.shortcut}</div>
							{/if}
						</button>
					{/each}
				</div>
			{/each}

			{#if flatItems().length === 0}
				<div class="slash-menu__empty">
					No results for "{filter}"
				</div>
			{/if}
		</div>
	</div>
{/if}

<style>
	.slash-menu {
		position: fixed;
		z-index: 100;
		width: 320px;
		max-height: 400px;
		background: hsl(var(--background));
		border: 1px solid hsl(var(--border));
		border-radius: 8px;
		box-shadow: 0 4px 16px hsl(var(--foreground) / 0.12);
		overflow: hidden;
	}

	.slash-menu__header {
		padding: 8px 12px;
		border-bottom: 1px solid hsl(var(--border));
	}

	.slash-menu__label {
		font-size: 12px;
		font-weight: 500;
		color: hsl(var(--muted-foreground));
	}

	.slash-menu__filter {
		font-family: ui-monospace, monospace;
		font-size: 14px;
		color: hsl(var(--foreground));
	}

	.slash-menu__content {
		max-height: 350px;
		overflow-y: auto;
		padding: 4px;
	}

	.slash-menu__category {
		padding: 4px 0;
	}

	.slash-menu__category-label {
		padding: 8px 8px 4px;
		font-size: 11px;
		font-weight: 600;
		color: hsl(var(--muted-foreground));
		text-transform: uppercase;
		letter-spacing: 0.5px;
	}

	.slash-menu__item {
		display: flex;
		align-items: flex-start;
		gap: 12px;
		width: 100%;
		padding: 8px;
		background: transparent;
		border: none;
		border-radius: 6px;
		cursor: pointer;
		text-align: left;
		transition: background-color 0.1s;
	}

	.slash-menu__item:hover,
	.slash-menu__item--selected {
		background: hsl(var(--muted));
	}

	.slash-menu__item-icon {
		display: flex;
		align-items: center;
		justify-content: center;
		width: 40px;
		height: 40px;
		background: hsl(var(--muted));
		border: 1px solid hsl(var(--border));
		border-radius: 6px;
		color: hsl(var(--foreground));
		flex-shrink: 0;
	}

	.slash-menu__item--selected .slash-menu__item-icon {
		background: hsl(var(--primary) / 0.1);
		border-color: hsl(var(--primary) / 0.3);
		color: hsl(var(--primary));
	}

	.slash-menu__item-content {
		flex: 1;
		min-width: 0;
		padding-top: 2px;
	}

	.slash-menu__item-label {
		font-size: 14px;
		font-weight: 500;
		color: hsl(var(--foreground));
		line-height: 1.3;
	}

	.slash-menu__item-description {
		font-size: 12px;
		color: hsl(var(--muted-foreground));
		line-height: 1.4;
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}

	.slash-menu__item-shortcut {
		font-family: ui-monospace, monospace;
		font-size: 12px;
		color: hsl(var(--muted-foreground));
		padding: 2px 6px;
		background: hsl(var(--muted));
		border-radius: 4px;
		flex-shrink: 0;
		margin-left: auto;
		align-self: center;
	}

	.slash-menu__empty {
		padding: 24px 16px;
		text-align: center;
		color: hsl(var(--muted-foreground));
		font-size: 13px;
	}
</style>
