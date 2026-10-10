<script lang="ts">
	import type { FocusMode, FocusModeOption } from '../input/focusModes';

	interface Props {
		mode: FocusMode;
		isSelected: boolean;
		onSelect: () => void;
		onDeselect: () => void;
	}

	let { mode, isSelected, onSelect, onDeselect }: Props = $props();

	function handleClick() {
		if (isSelected) {
			onDeselect();
		} else {
			onSelect();
		}
	}

	// Get icon SVG based on mode
	function getIcon(iconName: string): string {
		const icons: Record<string, string> = {
			'magnifying-glass-chart': `<path stroke-linecap="round" stroke-linejoin="round" d="m21 21-5.197-5.197m0 0A7.5 7.5 0 1 0 5.196 5.196a7.5 7.5 0 0 0 10.607 10.607Z" />`,
			'chart-bar': `<path stroke-linecap="round" stroke-linejoin="round" d="M3 13.125C3 12.504 3.504 12 4.125 12h2.25c.621 0 1.125.504 1.125 1.125v6.75C7.5 20.496 6.996 21 6.375 21h-2.25A1.125 1.125 0 0 1 3 19.875v-6.75ZM9.75 8.625c0-.621.504-1.125 1.125-1.125h2.25c.621 0 1.125.504 1.125 1.125v11.25c0 .621-.504 1.125-1.125 1.125h-2.25a1.125 1.125 0 0 1-1.125-1.125V8.625ZM16.5 4.125c0-.621.504-1.125 1.125-1.125h2.25C20.496 3 21 3.504 21 4.125v15.75c0 .621-.504 1.125-1.125 1.125h-2.25a1.125 1.125 0 0 1-1.125-1.125V4.125Z" />`,
			'document-text': `<path stroke-linecap="round" stroke-linejoin="round" d="M19.5 14.25v-2.625a3.375 3.375 0 0 0-3.375-3.375h-1.5A1.125 1.125 0 0 1 13.5 7.125v-1.5a3.375 3.375 0 0 0-3.375-3.375H8.25m0 12.75h7.5m-7.5 3H12M10.5 2.25H5.625c-.621 0-1.125.504-1.125 1.125v17.25c0 .621.504 1.125 1.125 1.125h12.75c.621 0 1.125-.504 1.125-1.125V11.25a9 9 0 0 0-9-9Z" />`,
			'cube': `<path stroke-linecap="round" stroke-linejoin="round" d="m21 7.5-9-5.25L3 7.5m18 0-9 5.25m9-5.25v9l-9 5.25M3 7.5l9 5.25M3 7.5v9l9 5.25m0-9v9" />`,
			'plus': `<path stroke-linecap="round" stroke-linejoin="round" d="M12 4.5v15m7.5-7.5h-15" />`
		};
		return icons[iconName] || icons['plus'];
	}
</script>

<button
	class="btn-pill btn-pill-sm focus-card"
	class:selected={isSelected}
	onclick={handleClick}
	type="button"
>
	<svg
		class="card-icon"
		xmlns="http://www.w3.org/2000/svg"
		fill="none"
		viewBox="0 0 24 24"
		stroke-width="1.5"
		stroke="currentColor"
	>
		{@html getIcon(mode.icon)}
	</svg>
	<span class="card-name">{mode.name}</span>
	{#if isSelected}
		<svg class="check-icon" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor">
			<path fill-rule="evenodd" d="M16.704 4.153a.75.75 0 0 1 .143 1.052l-8 10.5a.75.75 0 0 1-1.127.075l-4.5-4.5a.75.75 0 0 1 1.06-1.06l3.894 3.893 7.48-9.817a.75.75 0 0 1 1.05-.143Z" clip-rule="evenodd" />
		</svg>
	{/if}
</button>

<style>
	.focus-card {
		display: flex;
		align-items: center;
		gap: 7px;
		padding: 8px 14px;
		background: var(--dbg2);
		border: 1px solid var(--dbd);
		border-radius: 24px;
		cursor: pointer;
		transition: all 0.15s ease;
		white-space: nowrap;
	}

	.focus-card:hover:not(.selected) {
		background: var(--dbg3);
		border-color: var(--dt3);
	}

	.focus-card.selected {
		background: var(--dt);
		border-color: var(--dt);
	}

	.card-icon {
		width: 15px;
		height: 15px;
		color: var(--dt2);
		flex-shrink: 0;
	}

	.selected .card-icon {
		color: var(--dbg);
	}

	.card-name {
		font-size: 13px;
		font-weight: 500;
		color: var(--dt2);
	}

	.selected .card-name {
		color: var(--dbg);
	}

	.check-icon {
		width: 13px;
		height: 13px;
		color: var(--dbg);
		flex-shrink: 0;
	}
</style>
