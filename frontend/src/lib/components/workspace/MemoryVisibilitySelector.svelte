<script lang="ts">
	import type { MemoryVisibility } from '$lib/api/workspaces/memory';

	interface Props {
		selected: MemoryVisibility | 'all';
		onChange: (visibility: MemoryVisibility | 'all') => void;
		label?: string;
	}

	let { selected = 'all', onChange, label = 'Visibility' }: Props = $props();

	const visibilityOptions: { value: MemoryVisibility | 'all'; label: string; icon: string }[] = [
		{
			value: 'all',
			label: 'All Memories',
			icon: `<path stroke-linecap="round" stroke-linejoin="round" d="M2.036 12.322a1.012 1.012 0 0 1 0-.639C3.423 7.51 7.36 4.5 12 4.5c4.638 0 8.573 3.007 9.963 7.178.07.207.07.431 0 .639C20.577 16.49 16.64 19.5 12 19.5c-4.638 0-8.573-3.007-9.963-7.178Z" /><path stroke-linecap="round" stroke-linejoin="round" d="M15 12a3 3 0 1 1-6 0 3 3 0 0 1 6 0Z" />`
		},
		{
			value: 'workspace',
			label: 'Workspace',
			icon: `<path stroke-linecap="round" stroke-linejoin="round" d="M18 18.72a9.094 9.094 0 0 0 3.741-.479 3 3 0 0 0-4.682-2.72m.94 3.198.001.031c0 .225-.012.447-.037.666A11.944 11.944 0 0 1 12 21c-2.17 0-4.207-.576-5.963-1.584A6.062 6.062 0 0 1 6 18.719m12 0a5.971 5.971 0 0 0-.941-3.197m0 0A5.995 5.995 0 0 0 12 12.75a5.995 5.995 0 0 0-5.058 2.772m0 0a3 3 0 0 0-4.681 2.72 8.986 8.986 0 0 0 3.74.477m.94-3.197a5.971 5.971 0 0 0-.94 3.197M15 6.75a3 3 0 1 1-6 0 3 3 0 0 1 6 0Zm6 3a2.25 2.25 0 1 1-4.5 0 2.25 2.25 0 0 1 4.5 0Zm-13.5 0a2.25 2.25 0 1 1-4.5 0 2.25 2.25 0 0 1 4.5 0Z" />`
		},
		{
			value: 'private',
			label: 'Private',
			icon: `<path stroke-linecap="round" stroke-linejoin="round" d="M16.5 10.5V6.75a4.5 4.5 0 1 0-9 0v3.75m-.75 11.25h10.5a2.25 2.25 0 0 0 2.25-2.25v-6.75a2.25 2.25 0 0 0-2.25-2.25H6.75a2.25 2.25 0 0 0-2.25 2.25v6.75a2.25 2.25 0 0 0 2.25 2.25Z" />`
		},
		{
			value: 'shared',
			label: 'Shared',
			icon: `<path stroke-linecap="round" stroke-linejoin="round" d="M7.217 10.907a2.25 2.25 0 1 0 0 2.186m0-2.186c.18.324.283.696.283 1.093s-.103.77-.283 1.093m0-2.186 9.566-5.314m-9.566 7.5 9.566 5.314m0 0a2.25 2.25 0 1 0 3.935 2.186 2.25 2.25 0 0 0-3.935-2.186Zm0-12.814a2.25 2.25 0 1 0 3.933-2.185 2.25 2.25 0 0 0-3.933 2.185Z" />`
		}
	];

	function handleChange(value: MemoryVisibility | 'all') {
		onChange(value);
	}
</script>

<div class="visibility-selector">
	<label class="selector-label">{label}</label>
	<div class="visibility-options">
		{#each visibilityOptions as option}
			<button
				class="btn-pill btn-pill-ghost visibility-option"
				class:selected={selected === option.value}
				onclick={() => handleChange(option.value)}
				aria-label={`Filter by ${option.label}`}
			>
				<svg
					xmlns="http://www.w3.org/2000/svg"
					fill="none"
					viewBox="0 0 24 24"
					stroke-width="1.5"
					stroke="currentColor"
					width="14"
					height="14"
				>
					{@html option.icon}
				</svg>
				<span>{option.label}</span>
			</button>
		{/each}
	</div>
</div>

<style>
	.visibility-selector {
		display: flex;
		flex-direction: column;
		gap: 8px;
	}

	.selector-label {
		font-size: 11px;
		font-weight: 600;
		text-transform: uppercase;
		letter-spacing: 0.5px;
		color: var(--color-text-muted);
	}

	:global(.dark) .selector-label {
		color: #a1a1a6;
	}

	.visibility-options {
		display: flex;
		gap: 6px;
		flex-wrap: wrap;
	}

	.visibility-option {
		display: flex;
		align-items: center;
		gap: 6px;
		padding: 6px 10px;
		font-size: 12px;
		font-weight: 500;
		color: var(--color-text-muted);
		background: transparent;
		border: 1px solid var(--color-border);
		border-radius: 6px;
		cursor: pointer;
		transition: all 0.15s ease;
	}

	.visibility-option:hover {
		color: var(--color-text);
		background: var(--color-bg-secondary);
		border-color: var(--color-border);
	}

	.visibility-option.selected {
		color: #3b82f6;
		background: rgba(59, 130, 246, 0.1);
		border-color: #3b82f6;
	}

	:global(.dark) .visibility-option {
		color: #a1a1a6;
		border-color: rgba(255, 255, 255, 0.1);
	}

	:global(.dark) .visibility-option:hover {
		color: #f5f5f7;
		background: #3a3a3c;
		border-color: rgba(255, 255, 255, 0.15);
	}

	:global(.dark) .visibility-option.selected {
		color: #3b82f6;
		background: rgba(59, 130, 246, 0.15);
		border-color: #3b82f6;
	}
</style>
