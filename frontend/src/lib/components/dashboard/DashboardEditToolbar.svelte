<script lang="ts">
	import { fly } from 'svelte/transition';
	import { dashboardLayoutStore } from '$lib/stores/dashboard/dashboardLayoutStore.svelte';

	interface Props {
		onOpenWidgetPicker?: () => void;
	}

	let { onOpenWidgetPicker }: Props = $props();

	const layout = dashboardLayoutStore;

	const canUndo = $derived(layout.undoStack.length > 0);
</script>

<div
	class="dw-edit-toolbar"
	role="toolbar"
	aria-label="Dashboard edit mode toolbar"
	in:fly={{ y: -10, duration: 200 }}
>
	<!-- LEFT: Edit mode indicator -->
	<div class="dw-edit-toolbar__left">
		<div class="dw-edit-indicator" aria-hidden="true">
			<!-- Grid/pencil icon -->
			<svg
				class="dw-edit-indicator__icon"
				viewBox="0 0 16 16"
				fill="none"
				xmlns="http://www.w3.org/2000/svg"
				aria-hidden="true"
			>
				<path
					d="M11.013 1.427a1.75 1.75 0 012.474 2.474L5.61 11.778l-2.776.555.555-2.776 7.624-8.13z"
					stroke="currentColor"
					stroke-width="1.25"
					stroke-linecap="round"
					stroke-linejoin="round"
				/>
			</svg>
		</div>
		<div class="dw-edit-toolbar__mode-info">
			<span class="dw-edit-toolbar__mode-label">Edit Mode</span>
			<span class="dw-edit-toolbar__hint">Press Esc to exit</span>
		</div>
	</div>

	<!-- CENTER: Action buttons -->
	<div class="dw-edit-toolbar__center">
		<!-- Add Widget -->
		<button
			class="dw-btn-compact dw-btn-compact--solid"
			onclick={onOpenWidgetPicker}
			aria-label="Add a new widget to the dashboard"
		>
			<svg
				class="dw-btn-compact__icon"
				viewBox="0 0 16 16"
				fill="none"
				xmlns="http://www.w3.org/2000/svg"
				aria-hidden="true"
			>
				<path
					d="M8 3v10M3 8h10"
					stroke="currentColor"
					stroke-width="1.5"
					stroke-linecap="round"
				/>
			</svg>
			Add Widget
		</button>

		<!-- Undo -->
		<button
			class="dw-btn-compact dw-btn-compact--ghost"
			onclick={() => layout.undoRemove()}
			disabled={!canUndo}
			aria-label="Undo last widget removal"
			aria-disabled={!canUndo}
		>
			<svg
				class="dw-btn-compact__icon"
				viewBox="0 0 16 16"
				fill="none"
				xmlns="http://www.w3.org/2000/svg"
				aria-hidden="true"
			>
				<path
					d="M2.5 7.5A5.5 5.5 0 1 0 4 4L2 2"
					stroke="currentColor"
					stroke-width="1.5"
					stroke-linecap="round"
					stroke-linejoin="round"
				/>
				<path
					d="M2 5.5V2h3.5"
					stroke="currentColor"
					stroke-width="1.5"
					stroke-linecap="round"
					stroke-linejoin="round"
				/>
			</svg>
			Undo
		</button>

		<!-- Reset (placeholder — disabled) -->
		<button
			class="dw-btn-compact dw-btn-compact--ghost"
			disabled
			aria-label="Reset dashboard to default layout (coming soon)"
			aria-disabled="true"
			title="Reset to default layout — coming soon"
		>
			<svg
				class="dw-btn-compact__icon"
				viewBox="0 0 16 16"
				fill="none"
				xmlns="http://www.w3.org/2000/svg"
				aria-hidden="true"
			>
				<path
					d="M1.5 8A6.5 6.5 0 1 0 8 1.5"
					stroke="currentColor"
					stroke-width="1.5"
					stroke-linecap="round"
				/>
				<path
					d="M1.5 3.5v4H5.5"
					stroke="currentColor"
					stroke-width="1.5"
					stroke-linecap="round"
					stroke-linejoin="round"
				/>
			</svg>
			Reset
		</button>
	</div>

	<!-- RIGHT: Done button -->
	<div class="dw-edit-toolbar__right">
		<button
			class="dw-btn-compact dw-btn-compact--primary"
			onclick={() => layout.toggleEditMode()}
			aria-label="Exit edit mode and save dashboard layout"
		>
			Done
		</button>
	</div>
</div>

<style>
	/* ── Toolbar shell ─────────────────────────────────────────────────────────── */

	.dw-edit-toolbar {
		position: sticky;
		top: 0;
		z-index: 20;
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: var(--space-4);
		padding: var(--space-2) var(--space-4);
		border-radius: var(--radius-md);
		border: 1px solid var(--dbd);
		box-shadow: var(--shadow-md);

		/* Glass layer — light default */
		background: rgba(255, 255, 255, 0.85);
		backdrop-filter: blur(12px) saturate(160%);
		-webkit-backdrop-filter: blur(12px) saturate(160%);
	}

	/* Dark mode glass */
	:global(.dark) .dw-edit-toolbar {
		background: rgba(26, 26, 26, 0.85);
	}

	/* ── Sections ──────────────────────────────────────────────────────────────── */

	.dw-edit-toolbar__left {
		display: flex;
		align-items: center;
		gap: var(--space-2);
		min-width: 0;
	}

	.dw-edit-toolbar__center {
		display: flex;
		align-items: center;
		gap: var(--space-2);
		flex: 1;
		justify-content: center;
	}

	.dw-edit-toolbar__right {
		display: flex;
		align-items: center;
		gap: var(--space-2);
	}

	/* ── Edit mode indicator ───────────────────────────────────────────────────── */

	.dw-edit-indicator {
		display: flex;
		align-items: center;
		justify-content: center;
		width: 28px;
		height: 28px;
		border-radius: var(--radius-sm);
		background: color-mix(in srgb, var(--dt) 8%, transparent);
		flex-shrink: 0;
	}

	.dw-edit-indicator__icon {
		width: var(--icon-xs);
		height: var(--icon-xs);
		color: var(--dt2);
	}

	.dw-edit-toolbar__mode-info {
		display: flex;
		flex-direction: column;
		gap: 0;
	}

	.dw-edit-toolbar__mode-label {
		font-size: var(--text-sm);
		font-weight: var(--font-semibold);
		color: var(--dt);
		line-height: 1.2;
		white-space: nowrap;
	}

	.dw-edit-toolbar__hint {
		font-size: var(--text-xs);
		color: var(--dt3);
		line-height: 1.2;
		white-space: nowrap;
	}

	/* ── Compact button base ───────────────────────────────────────────────────── */

	.dw-btn-compact {
		display: inline-flex;
		align-items: center;
		gap: var(--space-1);
		height: var(--btn-h-sm);
		padding: 0 var(--btn-px-sm);
		border-radius: var(--radius-sm);
		font-size: var(--btn-font-sm);
		font-weight: var(--font-medium);
		line-height: 1;
		cursor: pointer;
		border: 1px solid transparent;
		transition:
			background 150ms ease,
			color 150ms ease,
			border-color 150ms ease,
			opacity 150ms ease,
			box-shadow 150ms ease;
		white-space: nowrap;
		user-select: none;
	}

	.dw-btn-compact:disabled {
		opacity: 0.4;
		cursor: not-allowed;
		pointer-events: none;
	}

	.dw-btn-compact__icon {
		width: var(--icon-xs);
		height: var(--icon-xs);
		flex-shrink: 0;
	}

	/* Solid variant — "Add Widget" */
	.dw-btn-compact--solid {
		background: color-mix(in srgb, var(--dt) 8%, transparent);
		color: var(--dt);
		border-color: var(--dbd);
	}

	.dw-btn-compact--solid:hover:not(:disabled) {
		background: color-mix(in srgb, var(--dt) 13%, transparent);
		border-color: var(--dbd);
	}

	.dw-btn-compact--solid:active:not(:disabled) {
		background: color-mix(in srgb, var(--dt) 18%, transparent);
	}

	/* Ghost variant — "Undo", "Reset" */
	.dw-btn-compact--ghost {
		background: transparent;
		color: var(--dt2);
		border-color: transparent;
	}

	.dw-btn-compact--ghost:hover:not(:disabled) {
		background: color-mix(in srgb, var(--dt) 6%, transparent);
		color: var(--dt);
	}

	.dw-btn-compact--ghost:active:not(:disabled) {
		background: color-mix(in srgb, var(--dt) 10%, transparent);
	}

	/* Primary variant — "Done" */
	.dw-btn-compact--primary {
		background: var(--dt);
		color: var(--dbg);
		border-color: var(--dt);
		padding-left: var(--btn-px-default);
		padding-right: var(--btn-px-default);
	}

	.dw-btn-compact--primary:hover:not(:disabled) {
		background: var(--dt2);
		border-color: var(--dt2);
	}

	.dw-btn-compact--primary:active:not(:disabled) {
		background: var(--dt3);
		border-color: var(--dt3);
	}

	/* ── Focus rings (keyboard accessibility) ──────────────────────────────────── */

	.dw-btn-compact:focus-visible {
		outline: 2px solid var(--dt);
		outline-offset: 2px;
	}
</style>
