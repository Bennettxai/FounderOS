<script lang="ts">
	import { fade } from 'svelte/transition';
	import type { App } from '$lib/types/apps';
	import {
		Play,
		Terminal,
		Pin,
		PinOff,
		Copy,
		Trash2
	} from 'lucide-svelte';

	interface Props {
		app: App;
		x: number;
		y: number;
		onClose: () => void;
		onOpen?: (app: App) => void;
		onEdit?: (app: App) => void;
		onPin?: (app: App) => void;
		onDuplicate?: (app: App) => void;
		onDelete?: (app: App) => void;
	}

	let { app, x, y, onClose, onOpen, onEdit, onPin, onDuplicate, onDelete }: Props = $props();

	// Adjust position to keep menu in viewport
	let menuRef: HTMLDivElement | null = $state(null);
	let adjustedX = $state(x);
	let adjustedY = $state(y);

	$effect(() => {
		if (menuRef) {
			const rect = menuRef.getBoundingClientRect();
			const viewportWidth = window.innerWidth;
			const viewportHeight = window.innerHeight;

			// Adjust X if menu goes off right edge
			if (x + rect.width > viewportWidth - 16) {
				adjustedX = viewportWidth - rect.width - 16;
			} else {
				adjustedX = x;
			}

			// Adjust Y if menu goes off bottom edge
			if (y + rect.height > viewportHeight - 16) {
				adjustedY = viewportHeight - rect.height - 16;
			} else {
				adjustedY = y;
			}
		}
	});

	function handleAction(action: () => void) {
		action();
		onClose();
	}

	function handleKeydown(e: KeyboardEvent) {
		if (e.key === 'Escape') {
			onClose();
		}
	}

	function handleBackdropClick(e: MouseEvent) {
		if (e.target === e.currentTarget) {
			onClose();
		}
	}
</script>

<svelte:window onkeydown={handleKeydown} />

<!-- Backdrop (invisible, catches clicks outside menu) -->
<div
	class="fixed inset-0 z-50"
	onclick={handleBackdropClick}
	oncontextmenu={(e) => { e.preventDefault(); onClose(); }}
	role="presentation"
></div>

<!-- Context Menu -->
<div
	bind:this={menuRef}
	class="fixed z-50 min-w-48 py-1.5 bg-white dark:bg-gray-800 rounded-lg shadow-lg border border-gray-200 dark:border-gray-700"
	style="left: {adjustedX}px; top: {adjustedY}px;"
	transition:fade={{ duration: 100 }}
	role="menu"
	aria-label="App actions"
>
	<!-- App name header -->
	<div class="px-3 py-2 border-b border-gray-100 dark:border-gray-700">
		<p class="text-sm font-medium text-gray-900 dark:text-white truncate">{app.name}</p>
	</div>

	<!-- Actions -->
	<div class="py-1">
		{#if onOpen}
			<button
				onclick={() => handleAction(() => onOpen(app))}
				class="w-full flex items-center gap-3 px-3 py-2 text-sm text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 transition-colors"
				role="menuitem"
				aria-label="Open {app.name}"
			>
				<Play class="w-4 h-4" strokeWidth={2} aria-hidden="true" />
				<span>Open</span>
			</button>
		{/if}

		{#if onEdit}
			<button
				onclick={() => handleAction(() => onEdit(app))}
				class="w-full flex items-center gap-3 px-3 py-2 text-sm text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 transition-colors"
				role="menuitem"
				aria-label="Edit {app.name}"
			>
				<Terminal class="w-4 h-4" strokeWidth={2} aria-hidden="true" />
				<span>Edit</span>
			</button>
		{/if}

		{#if onPin}
			<button
				onclick={() => handleAction(() => onPin(app))}
				class="w-full flex items-center gap-3 px-3 py-2 text-sm text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 transition-colors"
				role="menuitem"
				aria-label="{app.isPinned ? 'Unpin' : 'Pin'} {app.name}"
			>
				{#if app.isPinned}
					<PinOff class="w-4 h-4" strokeWidth={2} aria-hidden="true" />
					<span>Unpin</span>
				{:else}
					<Pin class="w-4 h-4" strokeWidth={2} aria-hidden="true" />
					<span>Pin to top</span>
				{/if}
			</button>
		{/if}
	</div>

	<!-- Separator -->
	<div class="h-px bg-gray-100 dark:border-gray-700 my-1"></div>

	<div class="py-1">
		{#if onDuplicate}
			<button
				onclick={() => handleAction(() => onDuplicate(app))}
				class="w-full flex items-center gap-3 px-3 py-2 text-sm text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 transition-colors"
				role="menuitem"
				aria-label="Duplicate {app.name}"
			>
				<Copy class="w-4 h-4" strokeWidth={2} aria-hidden="true" />
				<span>Duplicate</span>
			</button>
		{/if}

		{#if onDelete}
			<button
				onclick={() => handleAction(() => onDelete(app))}
				class="w-full flex items-center gap-3 px-3 py-2 text-sm text-red-600 dark:text-red-400 hover:bg-red-50 dark:hover:bg-red-900/20 transition-colors"
				role="menuitem"
				aria-label="Delete {app.name}"
			>
				<Trash2 class="w-4 h-4" strokeWidth={2} aria-hidden="true" />
				<span>Delete</span>
			</button>
		{/if}
	</div>
</div>
