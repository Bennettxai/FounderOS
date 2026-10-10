<script lang="ts">
	/**
	 * EmailCell - Clickable email link
	 */

	interface Props {
		value: string | null | undefined;
		editable?: boolean;
		onchange?: (value: string) => void;
	}

	let {
		value,
		editable = false,
		onchange
	}: Props = $props();

	let editing = $state(false);
	let editValue = $state('');

	function startEdit(e: MouseEvent) {
		if (editable) {
			e.preventDefault();
			editValue = value ?? '';
			editing = true;
		}
	}

	function finishEdit() {
		editing = false;
		if (editValue !== value) {
			onchange?.(editValue);
		}
	}

	function handleKeydown(e: KeyboardEvent) {
		if (e.key === 'Enter') {
			finishEdit();
		}
		if (e.key === 'Escape') {
			editValue = value ?? '';
			editing = false;
		}
	}
</script>

{#if editing}
	<input
		type="email"
		class="tpl-cell-edit"
		bind:value={editValue}
		onblur={finishEdit}
		onkeydown={handleKeydown}
	/>
{:else if value}
	<a
		href="mailto:{value}"
		class="tpl-cell tpl-cell-email"
		class:tpl-cell-editable={editable}
		ondblclick={startEdit}
	>
		<svg class="tpl-cell-icon" viewBox="0 0 20 20" fill="currentColor">
			<path d="M2.003 5.884L10 9.882l7.997-3.998A2 2 0 0016 4H4a2 2 0 00-1.997 1.884z" />
			<path d="M18 8.118l-8 4-8-4V14a2 2 0 002 2h12a2 2 0 002-2V8.118z" />
		</svg>
		<span>{value}</span>
	</a>
{:else}
	<button
		type="button"
		class="tpl-cell tpl-cell-empty"
		class:tpl-cell-editable={editable}
		ondblclick={startEdit}
		disabled={!editable}
	>
		—
	</button>
{/if}

<style>
	.tpl-cell {
		display: flex;
		align-items: center;
		gap: var(--tpl-space-2);
		width: 100%;
		padding: var(--tpl-space-2) var(--tpl-space-3);
		background: transparent;
		border: none;
		font-family: var(--tpl-font-sans);
		font-size: var(--tpl-text-sm);
		text-align: left;
		cursor: default;
		text-decoration: none;
	}

	.tpl-cell-email {
		color: var(--tpl-accent-primary);
	}

	.tpl-cell-email:hover {
		text-decoration: underline;
	}

	.tpl-cell-icon {
		width: 14px;
		height: 14px;
		flex-shrink: 0;
		opacity: 0.7;
	}

	.tpl-cell-empty {
		color: var(--tpl-text-muted);
	}

	.tpl-cell-editable {
		cursor: text;
		border-radius: var(--tpl-radius-sm);
		transition: background var(--tpl-transition-fast);
	}

	.tpl-cell-editable:hover {
		background: var(--tpl-bg-hover);
	}

	.tpl-cell-edit {
		width: 100%;
		padding: var(--tpl-space-2) var(--tpl-space-3);
		margin: -1px;
		background: var(--tpl-bg-primary);
		border: 1px solid var(--tpl-accent-primary);
		border-radius: var(--tpl-radius-sm);
		font-family: var(--tpl-font-sans);
		font-size: var(--tpl-text-sm);
		color: var(--tpl-text-primary);
		outline: none;
		box-shadow: var(--tpl-shadow-focus);
	}
</style>
