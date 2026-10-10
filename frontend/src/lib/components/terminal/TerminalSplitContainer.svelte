<script lang="ts">
	import { onDestroy } from 'svelte';
	import type { EnvironmentInfo, PaneNode, TerminalConfig } from '$lib/stores/terminal/terminalTypes';
	import TerminalPane from './TerminalPane.svelte';
	import TerminalSplitContainer from './TerminalSplitContainer.svelte';
	import { terminalStore } from '$lib/stores/terminal';

	interface Props {
		node: PaneNode;
		config: TerminalConfig;
		activeFocusMode?: string;
		environmentInfo?: EnvironmentInfo;
		environmentMode?: string;
		onSessionCreated?: (paneId: string, sessionId: string) => void;
		onFocus?: (paneId: string) => void;
		onShellReady?: (paneId: string, write: (data: string) => void) => void;
	}

	let { node, config, activeFocusMode, environmentInfo, environmentMode = 'local', onSessionCreated, onFocus, onShellReady }: Props = $props();

	let containerEl = $state<HTMLDivElement | undefined>(undefined);
	let isDragging = $state(false);

	// Track active drag handlers for cleanup
	let activeMoveHandler: ((ev: MouseEvent) => void) | null = null;
	let activeUpHandler: (() => void) | null = null;

	function startDrag(e: MouseEvent) {
		if (node.type !== 'split' || !containerEl) return;
		e.preventDefault();
		isDragging = true;

		const startX = e.clientX;
		const startY = e.clientY;
		const startRatio = node.ratio;
		const rect = containerEl.getBoundingClientRect();

		function onMouseMove(ev: MouseEvent) {
			if (node.type !== 'split') return;

			let newRatio: number;
			if (node.direction === 'horizontal') {
				const deltaX = ev.clientX - startX;
				newRatio = startRatio + (deltaX / rect.width);
			} else {
				const deltaY = ev.clientY - startY;
				newRatio = startRatio + (deltaY / rect.height);
			}

			terminalStore.resizeSplit(node.id, newRatio);
		}

		function onMouseUp() {
			isDragging = false;
			window.removeEventListener('mousemove', onMouseMove);
			window.removeEventListener('mouseup', onMouseUp);
			document.body.style.cursor = '';
			document.body.style.userSelect = '';
			activeMoveHandler = null;
			activeUpHandler = null;
		}

		// Store references for cleanup
		activeMoveHandler = onMouseMove;
		activeUpHandler = onMouseUp;

		document.body.style.cursor = node.direction === 'horizontal' ? 'col-resize' : 'row-resize';
		document.body.style.userSelect = 'none';
		window.addEventListener('mousemove', onMouseMove);
		window.addEventListener('mouseup', onMouseUp);
	}

	function handleDividerKeydown(e: KeyboardEvent) {
		if (node.type !== 'split') return;

		const step = e.shiftKey ? 0.1 : 0.02;
		let handled = false;

		if (node.direction === 'horizontal') {
			if (e.key === 'ArrowLeft') {
				terminalStore.resizeSplit(node.id, node.ratio - step);
				handled = true;
			} else if (e.key === 'ArrowRight') {
				terminalStore.resizeSplit(node.id, node.ratio + step);
				handled = true;
			}
		} else {
			if (e.key === 'ArrowUp') {
				terminalStore.resizeSplit(node.id, node.ratio - step);
				handled = true;
			} else if (e.key === 'ArrowDown') {
				terminalStore.resizeSplit(node.id, node.ratio + step);
				handled = true;
			}
		}

		if (handled) e.preventDefault();
	}

	onDestroy(() => {
		// Clean up any in-flight drag listeners
		if (activeMoveHandler) {
			window.removeEventListener('mousemove', activeMoveHandler);
		}
		if (activeUpHandler) {
			window.removeEventListener('mouseup', activeUpHandler);
		}
		document.body.style.cursor = '';
		document.body.style.userSelect = '';
	});
</script>

{#if node.type === 'leaf'}
	<TerminalPane
		pane={node}
		{config}
		{activeFocusMode}
		{environmentInfo}
		{environmentMode}
		{onSessionCreated}
		{onFocus}
		{onShellReady}
	/>
{:else}
	<div
		class="split-container"
		class:horizontal={node.direction === 'horizontal'}
		class:vertical={node.direction === 'vertical'}
		bind:this={containerEl}
	>
		<div
			class="split-child"
			style="{node.direction === 'horizontal' ? 'width' : 'height'}: {node.ratio * 100}%"
		>
			<TerminalSplitContainer
				node={node.children[0]}
				{config}
				{activeFocusMode}
				{environmentInfo}
				{environmentMode}
				{onSessionCreated}
				{onFocus}
				{onShellReady}
			/>
		</div>

		<div
			class="split-divider"
			class:horizontal={node.direction === 'horizontal'}
			class:vertical={node.direction === 'vertical'}
			class:dragging={isDragging}
			onmousedown={startDrag}
			onkeydown={handleDividerKeydown}
			role="separator"
			aria-orientation={node.direction}
			aria-valuenow={Math.round(node.ratio * 100)}
			aria-valuemin={10}
			aria-valuemax={90}
			aria-label="Split pane divider"
			tabindex="0"
		></div>

		<div
			class="split-child"
			style="{node.direction === 'horizontal' ? 'width' : 'height'}: {(1 - node.ratio) * 100}%"
		>
			<TerminalSplitContainer
				node={node.children[1]}
				{config}
				{activeFocusMode}
				{environmentInfo}
				{environmentMode}
				{onSessionCreated}
				{onFocus}
				{onShellReady}
			/>
		</div>
	</div>
{/if}

<style>
	.split-container {
		display: flex;
		width: 100%;
		height: 100%;
		overflow: hidden;
	}

	.split-container.horizontal {
		flex-direction: row;
	}

	.split-container.vertical {
		flex-direction: column;
	}

	.split-child {
		overflow: hidden;
		position: relative;
	}

	.split-divider {
		flex-shrink: 0;
		background: #222;
		z-index: 5;
		transition: background 0.1s;
		outline: none;
	}

	.split-divider.horizontal {
		width: 4px;
		cursor: col-resize;
	}

	.split-divider.vertical {
		height: 4px;
		cursor: row-resize;
	}

	.split-divider:hover,
	.split-divider:focus-visible,
	.split-divider.dragging {
		background: #00ff00;
	}
</style>
