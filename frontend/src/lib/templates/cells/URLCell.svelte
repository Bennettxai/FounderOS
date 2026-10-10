<script lang="ts">
	/**
	 * URLCell - Clickable external link
	 */

	interface Props {
		value: string | null | undefined;
		editable?: boolean;
		showFavicon?: boolean;
		truncate?: boolean;
		onchange?: (value: string) => void;
	}

	let {
		value,
		editable = false,
		showFavicon = true,
		truncate = true,
		onchange
	}: Props = $props();

	let editing = $state(false);
	let editValue = $state('');

	const displayUrl = $derived(() => {
		if (!value) return '';
		try {
			const url = new URL(value);
			return url.hostname + (url.pathname !== '/' ? url.pathname : '');
		} catch {
			return value;
		}
	});

	const faviconUrl = $derived(() => {
		if (!value || !showFavicon) return '';
		try {
			const url = new URL(value);
			return `https://www.google.com/s2/favicons?domain=${url.hostname}&sz=32`;
		} catch {
			return '';
		}
	});

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
			// Add https:// if no protocol
			let finalValue = editValue;
			if (finalValue && !finalValue.match(/^https?:\/\//)) {
				finalValue = 'https://' + finalValue;
			}
			onchange?.(finalValue);
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
		type="url"
		class="tpl-cell-edit"
		bind:value={editValue}
		placeholder="https://..."
		onblur={finishEdit}
		onkeydown={handleKeydown}
	/>
{:else if value}
	<a
		href={value}
		class="tpl-cell tpl-cell-url"
		class:tpl-cell-editable={editable}
		class:tpl-cell-truncate={truncate}
		target="_blank"
		rel="noopener noreferrer"
		ondblclick={startEdit}
	>
		{#if faviconUrl()}
			<img src={faviconUrl()} alt="" class="tpl-cell-favicon" />
		{:else}
			<svg class="tpl-cell-icon" viewBox="0 0 20 20" fill="currentColor">
				<path d="M12.586 4.586a2 2 0 112.828 2.828l-3 3a2 2 0 01-2.828 0 1 1 0 00-1.414 1.414 4 4 0 005.656 0l3-3a4 4 0 00-5.656-5.656l-1.5 1.5a1 1 0 101.414 1.414l1.5-1.5zm-5 5a2 2 0 012.828 0 1 1 0 101.414-1.414 4 4 0 00-5.656 0l-3 3a4 4 0 105.656 5.656l1.5-1.5a1 1 0 10-1.414-1.414l-1.5 1.5a2 2 0 11-2.828-2.828l3-3z" />
			</svg>
		{/if}
		<span>{displayUrl()}</span>
		<svg class="tpl-cell-external" viewBox="0 0 20 20" fill="currentColor">
			<path d="M11 3a1 1 0 100 2h2.586l-6.293 6.293a1 1 0 101.414 1.414L15 6.414V9a1 1 0 102 0V4a1 1 0 00-1-1h-5z" />
			<path d="M5 5a2 2 0 00-2 2v8a2 2 0 002 2h8a2 2 0 002-2v-3a1 1 0 10-2 0v3H5V7h3a1 1 0 000-2H5z" />
		</svg>
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

	.tpl-cell-url {
		color: var(--tpl-accent-primary);
	}

	.tpl-cell-url:hover {
		text-decoration: underline;
	}

	.tpl-cell-truncate {
		white-space: nowrap;
		overflow: hidden;
	}

	.tpl-cell-truncate span {
		overflow: hidden;
		text-overflow: ellipsis;
	}

	.tpl-cell-favicon {
		width: 14px;
		height: 14px;
		flex-shrink: 0;
		border-radius: 2px;
	}

	.tpl-cell-icon {
		width: 14px;
		height: 14px;
		flex-shrink: 0;
		opacity: 0.7;
	}

	.tpl-cell-external {
		width: 12px;
		height: 12px;
		flex-shrink: 0;
		opacity: 0.5;
		margin-left: auto;
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
