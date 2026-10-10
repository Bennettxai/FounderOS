<script lang="ts">
	/**
	 * DateCell - Display and edit date values
	 */

	interface Props {
		value: string | Date | null | undefined;
		editable?: boolean;
		format?: 'short' | 'medium' | 'long' | 'relative';
		includeTime?: boolean;
		locale?: string;
		onchange?: (value: string) => void;
	}

	let {
		value,
		editable = false,
		format = 'medium',
		includeTime = false,
		locale = 'en-US',
		onchange
	}: Props = $props();

	let editing = $state(false);
	let editValue = $state('');

	const dateValue = $derived(() => {
		if (!value) return null;
		return typeof value === 'string' ? new Date(value) : value;
	});

	const formattedValue = $derived(() => {
		const date = dateValue();
		if (!date || isNaN(date.getTime())) return '—';

		if (format === 'relative') {
			return formatRelative(date);
		}

		const options: Intl.DateTimeFormatOptions = {};

		switch (format) {
			case 'short':
				options.dateStyle = 'short';
				break;
			case 'medium':
				options.dateStyle = 'medium';
				break;
			case 'long':
				options.dateStyle = 'long';
				break;
		}

		if (includeTime) {
			options.timeStyle = 'short';
		}

		return new Intl.DateTimeFormat(locale, options).format(date);
	});

	function formatRelative(date: Date): string {
		const now = new Date();
		const diff = now.getTime() - date.getTime();
		const seconds = Math.floor(diff / 1000);
		const minutes = Math.floor(seconds / 60);
		const hours = Math.floor(minutes / 60);
		const days = Math.floor(hours / 24);

		if (days > 30) {
			return new Intl.DateTimeFormat(locale, { dateStyle: 'medium' }).format(date);
		}
		if (days > 0) return `${days}d ago`;
		if (hours > 0) return `${hours}h ago`;
		if (minutes > 0) return `${minutes}m ago`;
		return 'Just now';
	}

	function startEdit() {
		if (editable) {
			const date = dateValue();
			if (date) {
				editValue = includeTime
					? date.toISOString().slice(0, 16)
					: date.toISOString().slice(0, 10);
			} else {
				editValue = '';
			}
			editing = true;
		}
	}

	function finishEdit() {
		editing = false;
		if (editValue) {
			const newDate = new Date(editValue);
			if (!isNaN(newDate.getTime())) {
				onchange?.(newDate.toISOString());
			}
		}
	}

	function handleKeydown(e: KeyboardEvent) {
		if (e.key === 'Enter') {
			finishEdit();
		}
		if (e.key === 'Escape') {
			editing = false;
		}
	}
</script>

{#if editing}
	<input
		type={includeTime ? 'datetime-local' : 'date'}
		class="tpl-cell-edit tpl-cell-date-edit"
		bind:value={editValue}
		onblur={finishEdit}
		onkeydown={handleKeydown}
	/>
{:else}
	<button
		type="button"
		class="tpl-cell tpl-cell-date"
		class:tpl-cell-editable={editable}
		class:tpl-cell-empty={!dateValue()}
		ondblclick={startEdit}
		disabled={!editable}
	>
		<svg class="tpl-cell-date-icon" viewBox="0 0 20 20" fill="currentColor">
			<path fill-rule="evenodd" d="M6 2a1 1 0 00-1 1v1H4a2 2 0 00-2 2v10a2 2 0 002 2h12a2 2 0 002-2V6a2 2 0 00-2-2h-1V3a1 1 0 10-2 0v1H7V3a1 1 0 00-1-1zm0 5a1 1 0 000 2h8a1 1 0 100-2H6z" clip-rule="evenodd" />
		</svg>
		<span>{formattedValue()}</span>
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
		color: var(--tpl-text-primary);
		text-align: left;
		cursor: default;
	}

	.tpl-cell-date-icon {
		width: 14px;
		height: 14px;
		color: var(--tpl-text-muted);
		flex-shrink: 0;
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
		background: var(--tpl-bg-primary);
		border: 2px solid var(--tpl-accent-primary);
		border-radius: var(--tpl-radius-sm);
		font-family: var(--tpl-font-sans);
		font-size: var(--tpl-text-sm);
		color: var(--tpl-text-primary);
		outline: none;
		box-shadow: var(--tpl-shadow-focus);
	}
</style>
