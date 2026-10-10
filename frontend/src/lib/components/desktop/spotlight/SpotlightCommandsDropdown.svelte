<script lang="ts">
	import { fly } from 'svelte/transition';
	import type { SlashCommand } from './spotlightSearch.ts';

	interface Props {
		filteredCommands: SlashCommand[];
		show: boolean;
		highlightedIndex: number;
		onSelect: (cmd: SlashCommand) => void;
		onHover: (index: number) => void;
	}

	let { filteredCommands, show, highlightedIndex, onSelect, onHover }: Props = $props();
</script>

{#if show && filteredCommands.length > 0}
	<div class="commands-dropdown" transition:fly={{ y: 5, duration: 150 }}>
		{#each filteredCommands as cmd, i (cmd.id)}
			<button
				class="command-item"
				class:highlighted={highlightedIndex === i}
				onclick={() => onSelect(cmd)}
				onmouseenter={() => onHover(i)}
			>
				<span class="command-icon">{cmd.icon}</span>
				<div class="command-info">
					<span class="command-name">{cmd.name}</span>
					<span class="command-desc">{cmd.description}</span>
				</div>
			</button>
		{/each}
	</div>
{/if}

<style>
	.commands-dropdown {
		position: absolute;
		bottom: 100%;
		left: 16px;
		right: 16px;
		margin-bottom: 8px;
		background: white;
		border: 1px solid #e5e5e5;
		border-radius: 12px;
		box-shadow: 0 4px 16px rgba(0, 0, 0, 0.12);
		overflow: hidden;
		z-index: 100;
	}

	.command-item {
		width: 100%;
		padding: 10px 12px;
		border: none;
		background: none;
		text-align: left;
		cursor: pointer;
		display: flex;
		align-items: center;
		gap: 10px;
		transition: background 0.1s;
	}

	.command-item:hover,
	.command-item.highlighted {
		background: #f5f5f5;
	}

	.command-icon {
		font-size: 18px;
		width: 28px;
		text-align: center;
	}

	.command-info {
		flex: 1;
		display: flex;
		flex-direction: column;
	}

	.command-name {
		font-size: 13px;
		font-weight: 500;
		color: #333;
		font-family: monospace;
	}

	.command-desc {
		font-size: 11px;
		color: #888;
	}

	/* Dark mode */
	:global(.dark) .commands-dropdown {
		background: #2c2c2e;
		border-color: rgba(255, 255, 255, 0.12);
		box-shadow: 0 4px 16px rgba(0, 0, 0, 0.4);
	}

	:global(.dark) .command-item:hover,
	:global(.dark) .command-item.highlighted {
		background: #3a3a3c;
	}

	:global(.dark) .command-name {
		color: #f5f5f7;
	}

	:global(.dark) .command-desc {
		color: #6e6e73;
	}
</style>
