<script lang="ts">
	interface SlashCommand {
		name: string;
		display_name: string;
		description: string;
		icon: string;
		category: string;
	}

	interface Props {
		filteredCommands: SlashCommand[];
		activeCommand: SlashCommand | null;
		commandDropdownIndex: number;
		onSelect: (cmd: SlashCommand) => void;
		onHover: (index: number) => void;
		onClearActive: () => void;
	}

	let {
		filteredCommands,
		activeCommand,
		commandDropdownIndex,
		onSelect,
		onHover,
		onClearActive
	}: Props = $props();
</script>

<!-- Command suggestions dropdown -->
{#if filteredCommands.length > 0}
	<div class="command-dropdown">
		<div class="command-dropdown-header">
			<span class="command-dropdown-title">Commands</span>
		</div>
		<div class="command-dropdown-list">
			{#each filteredCommands as cmd, i (cmd.name)}
				<button
					class="btn-pill btn-pill-ghost command-item"
					class:selected={commandDropdownIndex === i}
					onclick={() => onSelect(cmd)}
					onmouseenter={() => onHover(i)}
				>
					<div class="command-icon" class:selected={commandDropdownIndex === i}>
						<span>/</span>
					</div>
					<div class="command-info">
						<span class="command-display-name">{cmd.display_name}</span>
						<span class="command-desc">{cmd.description}</span>
					</div>
					<span class="command-shortcut">/{cmd.name}</span>
				</button>
			{/each}
		</div>
		<div class="command-dropdown-footer">
			↑↓ Navigate · Enter/Tab Select · Esc Cancel
		</div>
	</div>
{/if}

<!-- Active command chip -->
{#if activeCommand}
	<div class="active-command-chip">
		<div class="active-command-badge">
			<div class="active-command-icon"><span>/</span></div>
			<span class="active-command-name">{activeCommand.display_name}</span>
			<button onclick={onClearActive} class="btn-pill btn-pill-ghost active-command-clear" aria-label="Clear command">
				<svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" width="12" height="12">
					<path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12"/>
				</svg>
			</button>
		</div>
		<span class="active-command-desc">{activeCommand.description}</span>
	</div>
{/if}

<style>
	/* Command dropdown */
	.command-dropdown {
		background: var(--color-bg-secondary, #f9fafb);
		border: 1px solid var(--color-border, #e5e7eb);
		border-radius: 12px;
		overflow: hidden;
		box-shadow: 0 4px 12px rgba(0, 0, 0, 0.08);
	}

	:global(.dark) .command-dropdown {
		background: #2c2c2e;
		border-color: rgba(255, 255, 255, 0.12);
		box-shadow: 0 4px 16px rgba(0, 0, 0, 0.3);
	}

	.command-dropdown-header {
		padding: 8px 12px;
		border-bottom: 1px solid var(--color-border, #e5e7eb);
		background: var(--color-bg, white);
	}

	:global(.dark) .command-dropdown-header {
		background: #1c1c1e;
		border-color: rgba(255, 255, 255, 0.08);
	}

	.command-dropdown-title {
		font-size: 11px;
		font-weight: 600;
		text-transform: uppercase;
		letter-spacing: 0.5px;
		color: var(--color-text-secondary, #6b7280);
	}

	:global(.dark) .command-dropdown-title {
		color: #8e8e93;
	}

	.command-dropdown-list {
		max-height: 260px;
		overflow-y: auto;
	}

	.command-dropdown-footer {
		padding: 6px 12px;
		border-top: 1px solid var(--color-border, #e5e7eb);
		background: var(--color-bg-tertiary, #f3f4f6);
		font-size: 11px;
		color: var(--color-text-muted, #9ca3af);
	}

	:global(.dark) .command-dropdown-footer {
		background: #1c1c1e;
		border-color: rgba(255, 255, 255, 0.08);
		color: #6e6e73;
	}

	.command-item {
		display: flex;
		align-items: center;
		gap: 12px;
		width: 100%;
		padding: 10px 12px;
		background: transparent;
		border: none;
		cursor: pointer;
		text-align: left;
		transition: background 0.1s ease;
	}

	.command-item:hover,
	.command-item.selected {
		background: rgba(59, 130, 246, 0.08);
	}

	:global(.dark) .command-item:hover,
	:global(.dark) .command-item.selected {
		background: rgba(10, 132, 255, 0.15);
	}

	.command-item.selected {
		color: var(--color-primary, #3b82f6);
	}

	:global(.dark) .command-item.selected {
		color: #0A84FF;
	}

	.command-icon {
		width: 32px;
		height: 32px;
		border-radius: 8px;
		background: var(--color-bg-tertiary, #e5e7eb);
		color: var(--color-text-secondary, #6b7280);
		display: flex;
		align-items: center;
		justify-content: center;
		flex-shrink: 0;
		font-size: 14px;
		font-weight: 600;
	}

	.command-icon.selected {
		background: var(--color-primary, #3b82f6);
		color: white;
	}

	:global(.dark) .command-icon {
		background: #3a3a3c;
		color: #a1a1a6;
	}

	:global(.dark) .command-icon.selected {
		background: #0A84FF;
		color: white;
	}

	.command-info {
		flex: 1;
		min-width: 0;
		display: flex;
		flex-direction: column;
		gap: 2px;
	}

	.command-display-name {
		font-size: 14px;
		font-weight: 500;
		color: var(--color-text, #1f2937);
	}

	:global(.dark) .command-display-name {
		color: #f5f5f7;
	}

	.command-item.selected .command-display-name {
		color: var(--color-primary, #3b82f6);
	}

	:global(.dark) .command-item.selected .command-display-name {
		color: #0A84FF;
	}

	.command-desc {
		font-size: 12px;
		color: var(--color-text-muted, #6b7280);
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}

	:global(.dark) .command-desc {
		color: #8e8e93;
	}

	.command-shortcut {
		font-size: 12px;
		font-family: ui-monospace, SFMono-Regular, monospace;
		color: var(--color-text-muted, #9ca3af);
		flex-shrink: 0;
	}

	:global(.dark) .command-shortcut {
		color: #6e6e73;
	}

	/* Active command chip */
	.active-command-chip {
		display: flex;
		align-items: center;
		gap: 8px;
	}

	.active-command-badge {
		display: inline-flex;
		align-items: center;
		gap: 8px;
		padding: 6px 10px;
		background: rgba(59, 130, 246, 0.1);
		border: 1px solid rgba(59, 130, 246, 0.3);
		border-radius: 20px;
	}

	:global(.dark) .active-command-badge {
		background: rgba(10, 132, 255, 0.15);
		border-color: rgba(10, 132, 255, 0.3);
	}

	.active-command-icon {
		width: 20px;
		height: 20px;
		border-radius: 4px;
		background: var(--color-primary, #3b82f6);
		color: white;
		display: flex;
		align-items: center;
		justify-content: center;
		font-size: 12px;
		font-weight: 600;
	}

	:global(.dark) .active-command-icon {
		background: #0A84FF;
	}

	.active-command-name {
		font-size: 13px;
		font-weight: 500;
		color: var(--color-primary, #3b82f6);
	}

	:global(.dark) .active-command-name {
		color: #0A84FF;
	}

	.active-command-clear {
		width: 16px;
		height: 16px;
		border-radius: 50%;
		background: rgba(59, 130, 246, 0.2);
		border: none;
		cursor: pointer;
		display: flex;
		align-items: center;
		justify-content: center;
		color: var(--color-primary, #3b82f6);
		transition: all 0.15s ease;
	}

	.active-command-clear:hover {
		background: rgba(59, 130, 246, 0.3);
	}

	:global(.dark) .active-command-clear {
		background: rgba(10, 132, 255, 0.25);
		color: #0A84FF;
	}

	:global(.dark) .active-command-clear:hover {
		background: rgba(10, 132, 255, 0.4);
	}

	.active-command-desc {
		font-size: 12px;
		color: var(--color-text-muted, #6b7280);
	}

	:global(.dark) .active-command-desc {
		color: #8e8e93;
	}
</style>
