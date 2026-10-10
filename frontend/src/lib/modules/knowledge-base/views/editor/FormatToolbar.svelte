<script lang="ts">
	/**
	 * FormatToolbar - Floating formatting toolbar for text selection
	 * Appears when user selects text in contenteditable elements
	 */
	import { Bold, Italic, Underline, Strikethrough, Code, Link } from 'lucide-svelte';

	interface Props {
		visible?: boolean;
		position?: { x: number; y: number };
		onFormat?: (format: 'bold' | 'italic' | 'underline' | 'strikethrough' | 'code' | 'link') => void;
	}

	let { visible = false, position = { x: 0, y: 0 }, onFormat }: Props = $props();

	const formats = [
		{ key: 'bold' as const, icon: Bold, label: 'Bold', shortcut: 'Cmd+B' },
		{ key: 'italic' as const, icon: Italic, label: 'Italic', shortcut: 'Cmd+I' },
		{ key: 'underline' as const, icon: Underline, label: 'Underline', shortcut: 'Cmd+U' },
		{ key: 'strikethrough' as const, icon: Strikethrough, label: 'Strikethrough', shortcut: 'Cmd+Shift+S' },
		{ key: 'code' as const, icon: Code, label: 'Code', shortcut: 'Cmd+E' },
		{ key: 'link' as const, icon: Link, label: 'Link', shortcut: 'Cmd+K' },
	];

	function handleFormat(format: typeof formats[number]['key']) {
		onFormat?.(format);
	}
</script>

{#if visible}
	<div
		class="format-toolbar"
		style:left="{position.x}px"
		style:top="{position.y}px"
		role="toolbar"
		aria-label="Text formatting"
	>
		{#each formats as format}
			<button
				class="btn-pill btn-pill-ghost format-toolbar__btn"
				onclick={() => handleFormat(format.key)}
				title="{format.label} ({format.shortcut})"
				aria-label={format.label}
			>
				<format.icon class="h-4 w-4" />
			</button>
		{/each}
	</div>
{/if}

<style>
	.format-toolbar {
		position: fixed;
		z-index: 100;
		display: flex;
		align-items: center;
		gap: 0.125rem;
		padding: 0.25rem;
		background-color: var(--dbg);
		border: 1px solid var(--dbd);
		border-radius: 0.5rem;
		box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
		transform: translateX(-50%) translateY(-100%);
		margin-top: -8px;
	}

	.format-toolbar__btn {
		display: flex;
		align-items: center;
		justify-content: center;
		width: 32px;
		height: 32px;
		padding: 0;
		background: transparent;
		border: none;
		border-radius: 0.375rem;
		color: var(--dt);
		cursor: pointer;
		transition: background-color 0.1s;
	}

	.format-toolbar__btn:hover {
		background-color: var(--dbg2);
	}

	.format-toolbar__btn:active {
		background-color: var(--dbg3);
	}
</style>
