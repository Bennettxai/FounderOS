<script lang="ts">
	import { editor, type EditorBlock, type BlockType, type EditorState } from '$lib/stores/editor';
	import Block from './Block.svelte';

	interface Props {
		block: EditorBlock;
		readonly: boolean;
		isEmpty: boolean;
		parentContextId: string | undefined;
		onPageClick: ((pageId: string) => void) | undefined;
		onFocus: () => void;
		onBlur: (e: FocusEvent) => void;
		onInput: (e: Event) => void;
		onKeydown: (e: KeyboardEvent) => void;
		onBindElement: (el: HTMLElement | null) => void;
	}

	let {
		block,
		readonly,
		isEmpty,
		parentContextId,
		onPageClick,
		onFocus,
		onBlur,
		onInput,
		onKeydown,
		onBindElement,
	}: Props = $props();

	function bindElement(node: HTMLElement) {
		onBindElement(node);
		return {
			destroy() {
				onBindElement(null);
			}
		};
	}
</script>

<div class="flex items-start gap-1">
	<button
		onclick={() => editor.toggleToggleBlock(block.id)}
		class="w-6 h-6 flex items-center justify-center rounded hover:bg-gray-200 dark:hover:bg-gray-700 text-gray-500 dark:text-gray-400 transition-transform flex-shrink-0"
		class:rotate-90={block.properties?.expanded}
		tabindex="-1"
		aria-label="Toggle expand"
	>
		<svg class="w-4 h-4" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24">
			<path stroke-linecap="round" stroke-linejoin="round" d="M9 5l7 7-7 7" />
		</svg>
	</button>
	<div class="flex-1">
		<div
			use:bindElement
			contenteditable={!readonly}
			data-block-id={block.id}
			data-placeholder="Toggle header..."
			onfocus={onFocus}
			onblur={onBlur}
			oninput={onInput}
			onkeydown={onKeydown}
			class="text-gray-900 dark:text-gray-100 font-medium outline-none min-h-[1.5em] block-editable"
			class:is-empty={isEmpty}
		></div>
		{#if block.properties?.expanded}
			<div class="pl-6 mt-1 space-y-0.5">
				{#if block.children?.length}
					{#each block.children as childBlock, childIdx}
						<Block block={childBlock} index={childIdx} {readonly} {parentContextId} {onPageClick} />
					{/each}
				{/if}
				<!-- Empty block placeholder to add content -->
				{#if !readonly}
					<div
						contenteditable="true"
						data-placeholder="Type inside toggle..."
						class="text-gray-700 dark:text-gray-300 outline-none min-h-[1.5em] empty:before:content-[attr(data-placeholder)] empty:before:text-gray-400 dark:empty:before:text-gray-500"
						onkeydown={(e) => {
							if (e.key === 'Enter' && !e.shiftKey) {
								e.preventDefault();
								const target = e.currentTarget as HTMLElement;
								const content = target.innerText || '';
								if (content.trim()) {
									editor.update((s: EditorState) => ({
										...s,
										blocks: s.blocks.map((b: EditorBlock) =>
											b.id === block.id
												? {
													...b,
													children: [...(b.children || []), {
														id: Math.random().toString(36).substring(2, 11),
														type: 'paragraph' as BlockType,
														content: content.trim(),
														properties: {}
													}]
												}
												: b
										),
										isDirty: true
									}));
									target.innerText = '';
								}
							}
						}}
					></div>
				{/if}
			</div>
		{/if}
	</div>
</div>
