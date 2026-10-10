<!--
  ToolPicker.svelte
  Multi-select grid for choosing integrations/tools
-->
<script lang="ts">
	import { CheckIcon } from './icons';

	interface Tool {
		id: string;
		name: string;
		icon: string;
		description?: string;
	}

	interface Props {
		tools: Tool[];
		selected?: string[];
		maxSelections?: number;
		columns?: 2 | 3 | 4;
		onSelectionChange?: (selected: string[]) => void;
		class?: string;
	}

	let {
		tools,
		selected = $bindable([]),
		maxSelections = Infinity,
		columns = 3,
		onSelectionChange,
		class: className = ''
	}: Props = $props();

	function toggleTool(toolId: string) {
		if (selected.includes(toolId)) {
			selected = selected.filter((id) => id !== toolId);
		} else if (selected.length < maxSelections) {
			selected = [...selected, toolId];
		}
		onSelectionChange?.(selected);
	}

	function isSelected(toolId: string) {
		return selected.includes(toolId);
	}
</script>

<div
	class="tool-picker {className}"
	style="--columns: {columns}"
>
	{#each tools as tool (tool.id)}
		<button
			type="button"
			class="btn-pill btn-pill-ghost tool-card"
			class:is-selected={isSelected(tool.id)}
			onclick={() => toggleTool(tool.id)}
		>
			<div class="tool-icon">
				{@html tool.icon}
			</div>
			<span class="tool-name">{tool.name}</span>
			{#if tool.description}
				<span class="tool-description">{tool.description}</span>
			{/if}
			{#if isSelected(tool.id)}
				<div class="check-badge">
					<CheckIcon size={14} />
				</div>
			{/if}
		</button>
	{/each}
</div>

<style>
	.tool-picker {
		display: grid;
		grid-template-columns: repeat(var(--columns, 3), 1fr);
		gap: 12px;
	}

	.tool-card {
		position: relative;
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 8px;
		padding: 20px 16px;
		border: 2px solid var(--border, #e5e7eb);
		border-radius: 12px;
		background-color: var(--background, #ffffff);
		cursor: pointer;
		transition: all 0.2s ease;
	}

	.tool-card:hover {
		border-color: var(--primary, #000000);
		background-color: var(--accent, #f3f4f6);
	}

	.tool-card.is-selected {
		border-color: var(--primary, #000000);
		background-color: var(--accent, #f3f4f6);
	}

	.tool-icon {
		width: 40px;
		height: 40px;
		display: flex;
		align-items: center;
		justify-content: center;
		color: var(--foreground, #1f2937);
	}

	.tool-icon :global(svg) {
		width: 32px;
		height: 32px;
	}

	.tool-icon :global(img) {
		width: 32px;
		height: 32px;
		object-fit: contain;
	}

	.tool-name {
		font-size: 14px;
		font-weight: 500;
		color: var(--foreground, #1f2937);
		text-align: center;
	}

	.tool-description {
		font-size: 12px;
		color: var(--muted-foreground, #6b7280);
		text-align: center;
	}

	.check-badge {
		position: absolute;
		top: 8px;
		right: 8px;
		width: 24px;
		height: 24px;
		border-radius: 50%;
		background-color: var(--primary, #000000);
		color: var(--primary-foreground, #ffffff);
		display: flex;
		align-items: center;
		justify-content: center;
	}

	/* Dark mode */
	:global(.dark) .tool-card {
		background-color: var(--background, #0a0a0a);
		border-color: var(--border, #2a2a2a);
	}

	:global(.dark) .tool-card:hover,
	:global(.dark) .tool-card.is-selected {
		border-color: var(--primary, #ffffff);
		background-color: var(--accent, #1a1a1a);
	}

	:global(.dark) .tool-icon {
		color: var(--foreground, #f9fafb);
	}

	:global(.dark) .tool-name {
		color: var(--foreground, #f9fafb);
	}

	:global(.dark) .check-badge {
		background-color: var(--primary, #ffffff);
		color: var(--primary-foreground, #000000);
	}

	/* Responsive */
	@media (max-width: 640px) {
		.tool-picker {
			grid-template-columns: repeat(2, 1fr);
		}
	}
</style>
