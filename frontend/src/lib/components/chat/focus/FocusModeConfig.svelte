<script lang="ts">
	import type { FocusMode } from '../input/focusModes';

	interface Props {
		mode: FocusMode;
		options: Record<string, string>;
		onOptionChange: (optionId: string, value: string) => void;
	}

	let { mode, options, onOptionChange }: Props = $props();
</script>

<div class="options-panel">
	<div class="options-header">
		<span class="options-title">{mode.name} options</span>
	</div>
	<div class="options-grid">
		{#each mode.options as option}
			<div class="option-item">
				<span class="option-label">{option.label}</span>
				{#if option.type === 'segment'}
					<div class="segment-control">
						{#each option.choices || [] as choice}
							<button
								class="btn-pill btn-pill-ghost segment-btn"
								class:active={options[option.id] === choice.value}
								onclick={() => onOptionChange(option.id, choice.value)}
								title={choice.tooltip || ''}
								type="button"
							>
								{choice.label}
							</button>
						{/each}
					</div>
				{:else if option.type === 'toggle'}
					<button
						class="toggle-btn"
						class:active={options[option.id] === 'on'}
						onclick={() => onOptionChange(option.id, options[option.id] === 'on' ? 'off' : 'on')}
						type="button"
						aria-label="Toggle {option.label}"
					>
						<span class="toggle-track">
							<span class="toggle-thumb"></span>
						</span>
					</button>
				{/if}
			</div>
		{/each}
	</div>
</div>

<style>
	.options-panel {
		width: 100%;
		max-width: 600px;
		background: var(--color-bg-secondary);
		border: 1px solid var(--color-border);
		border-radius: 16px;
		padding: 16px 20px;
	}

	:global(.dark) .options-panel {
		background: #2c2c2e;
		border-color: rgba(255, 255, 255, 0.12);
	}

	.options-header {
		margin-bottom: 16px;
	}

	.options-title {
		font-size: 13px;
		font-weight: 600;
		color: var(--color-text-secondary);
		text-transform: uppercase;
		letter-spacing: 0.5px;
	}

	:global(.dark) .options-title {
		color: #a1a1a6;
	}

	.options-grid {
		display: flex;
		flex-wrap: wrap;
		gap: 16px;
	}

	.option-item {
		display: flex;
		align-items: center;
		gap: 12px;
	}

	.option-label {
		font-size: 14px;
		color: var(--color-text);
		font-weight: 500;
		min-width: 80px;
	}

	:global(.dark) .option-label {
		color: #f5f5f7;
	}

	/* Segment control */
	.segment-control {
		display: flex;
		background: var(--color-bg-tertiary);
		border-radius: 8px;
		padding: 3px;
	}

	:global(.dark) .segment-control {
		background: #1c1c1e;
	}

	.segment-btn {
		padding: 6px 12px;
		font-size: 13px;
		font-weight: 500;
		border: none;
		background: transparent;
		color: var(--color-text-secondary);
		cursor: pointer;
		border-radius: 6px;
		transition: all 0.15s ease;
		white-space: nowrap;
	}

	.segment-btn:hover:not(.active) {
		color: var(--color-text);
	}

	.segment-btn.active {
		background: var(--color-bg);
		color: var(--color-text);
		box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
	}

	:global(.dark) .segment-btn {
		color: #8e8e93;
	}

	:global(.dark) .segment-btn:hover:not(.active) {
		color: #f5f5f7;
	}

	:global(.dark) .segment-btn.active {
		background: #3a3a3c;
		color: #f5f5f7;
		box-shadow: 0 1px 3px rgba(0, 0, 0, 0.3);
	}

	/* Toggle */
	.toggle-btn {
		background: transparent;
		border: none;
		cursor: pointer;
		padding: 0;
	}

	.toggle-track {
		display: block;
		width: 44px;
		height: 24px;
		background: var(--color-bg-tertiary);
		border-radius: 12px;
		position: relative;
		transition: background 0.2s ease;
	}

	.toggle-btn.active .toggle-track {
		background: #34c759;
	}

	:global(.dark) .toggle-track {
		background: #3a3a3c;
	}

	:global(.dark) .toggle-btn.active .toggle-track {
		background: #30d158;
	}

	.toggle-thumb {
		position: absolute;
		top: 2px;
		left: 2px;
		width: 20px;
		height: 20px;
		background: white;
		border-radius: 50%;
		box-shadow: 0 1px 3px rgba(0, 0, 0, 0.2);
		transition: transform 0.2s ease;
	}

	.toggle-btn.active .toggle-thumb {
		transform: translateX(20px);
	}
</style>
