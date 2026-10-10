<script lang="ts">
	import type { EditorBlock } from '$lib/stores/editor';

	interface Props {
		block: EditorBlock;
		readonly: boolean;
		isEmpty: boolean;
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

<div class="callout-block flex items-start gap-3 p-4 rounded-md bg-amber-50 dark:bg-[var(--bos-v2-layer-background-secondary)] border border-amber-200 dark:border-transparent">
	<button
		class="btn-pill btn-pill-ghost flex-shrink-0 w-6 h-6 flex items-center justify-center"
		tabindex="-1"
		title="Click to change icon"
	>
		{block.properties?.calloutIcon || '💡'}
	</button>
	<div
		use:bindElement
		contenteditable={!readonly}
		data-block-id={block.id}
		data-placeholder="Type something..."
		onfocus={onFocus}
		onblur={onBlur}
		oninput={onInput}
		onkeydown={onKeydown}
		class="flex-1 text-gray-800 dark:text-gray-200 outline-none min-h-[1.5em] block-editable"
		class:is-empty={isEmpty}
	></div>
</div>

<style>
	.block-editable {
		position: relative;
	}
	.block-editable:focus {
		outline: none;
		background-color: transparent;
	}
	.block-editable.is-empty:focus:before {
		content: attr(data-placeholder);
		color: var(--bos-status-neutral);
		pointer-events: none;
		position: absolute;
		font-style: normal;
		font-weight: normal;
	}
</style>
