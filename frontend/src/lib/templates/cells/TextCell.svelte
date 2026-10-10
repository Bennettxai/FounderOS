<script lang="ts">
	/**
	 * TextCell - Display and edit text values
	 */

	interface Props {
		value: string | null | undefined;
		editable?: boolean;
		placeholder?: string;
		maxLength?: number;
		multiline?: boolean;
		truncate?: boolean;
		onchange?: (value: string) => void;
	}

	let {
		value,
		editable = false,
		placeholder = '',
		maxLength,
		multiline = false,
		truncate = true,
		onchange
	}: Props = $props();

	let editing = $state(false);
	let editValue = $state('');

	function startEdit() {
		if (editable) {
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
		if (e.key === 'Enter' && !multiline) {
			finishEdit();
		}
		if (e.key === 'Escape') {
			editValue = value ?? '';
			editing = false;
		}
	}
</script>

{#if editing}
	{#if multiline}
		<textarea
			class="tpl-cell-edit tpl-cell-textarea"
			bind:value={editValue}
			maxlength={maxLength}
			onblur={finishEdit}
			onkeydown={handleKeydown}
		></textarea>
	{:else}
		<input
			type="text"
			class="tpl-cell-edit"
			bind:value={editValue}
			maxlength={maxLength}
			onblur={finishEdit}
			onkeydown={handleKeydown}
		/>
	{/if}
{:else}
	<button
		type="button"
		class="tpl-cell tpl-cell-text"
		class:tpl-cell-editable={editable}
		class:tpl-cell-truncate={truncate}
		class:tpl-cell-empty={!value}
		ondblclick={startEdit}
		disabled={!editable}
	>
		{value || placeholder || '—'}
	</button>
{/if}

<style>
	.tpl-cell {
		display: block;
		width: 100%;
		padding: var(--tpl-space-2) var(--tpl-space-3);
		background: transparent;
		border: none;
		font-family: var(--tpl-font-sans);
		font-size: var(--tpl-text-sm);
		color: var(--tpl-text-primary);
		text-align: left;
		cursor: default;
	}

	.tpl-cell-text {
		line-height: var(--tpl-leading-normal);
	}

	.tpl-cell-truncate {
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
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

	.tpl-cell-textarea {
		min-height: 60px;
		resize: vertical;
	}
</style>
