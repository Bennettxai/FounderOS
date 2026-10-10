<script lang="ts">
	/**
	 * PhoneCell - Clickable phone link
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

	function formatPhone(phone: string): string {
		// Basic US phone formatting
		const cleaned = phone.replace(/\D/g, '');
		if (cleaned.length === 10) {
			return `(${cleaned.slice(0, 3)}) ${cleaned.slice(3, 6)}-${cleaned.slice(6)}`;
		}
		if (cleaned.length === 11 && cleaned[0] === '1') {
			return `+1 (${cleaned.slice(1, 4)}) ${cleaned.slice(4, 7)}-${cleaned.slice(7)}`;
		}
		return phone;
	}

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
		type="tel"
		class="tpl-cell-edit"
		bind:value={editValue}
		onblur={finishEdit}
		onkeydown={handleKeydown}
	/>
{:else if value}
	<a
		href="tel:{value.replace(/\D/g, '')}"
		class="tpl-cell tpl-cell-phone"
		class:tpl-cell-editable={editable}
		ondblclick={startEdit}
	>
		<svg class="tpl-cell-icon" viewBox="0 0 20 20" fill="currentColor">
			<path d="M2 3a1 1 0 011-1h2.153a1 1 0 01.986.836l.74 4.435a1 1 0 01-.54 1.06l-1.548.773a11.037 11.037 0 006.105 6.105l.774-1.548a1 1 0 011.059-.54l4.435.74a1 1 0 01.836.986V17a1 1 0 01-1 1h-2C7.82 18 2 12.18 2 5V3z" />
		</svg>
		<span>{formatPhone(value)}</span>
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

	.tpl-cell-phone {
		color: var(--tpl-accent-primary);
	}

	.tpl-cell-phone:hover {
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
