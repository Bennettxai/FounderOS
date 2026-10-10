<script lang="ts">
	import type { TerminalProvider } from '$lib/stores/terminal/terminalTypes';
	import { PROVIDER_CONFIGS } from '$lib/stores/terminal/terminalTypes';

	interface Props {
		provider: TerminalProvider;
		activeFocusMode: string;
		onFocusModeChange: (mode: string) => void;
	}

	let { provider, activeFocusMode, onFocusModeChange }: Props = $props();

	const FOCUS_PILLS = [
		{ id: 'research', label: 'Research' },
		{ id: 'analyze', label: 'Analyze' },
		{ id: 'write', label: 'Write' },
		{ id: 'build', label: 'Build' },
		{ id: 'generate', label: 'Generate' },
		{ id: 'general', label: 'General' }
	] as const;

	const providerColor = $derived(PROVIDER_CONFIGS.find(p => p.id === provider)?.color ?? '#8b5cf6');
	const isVisible = $derived(provider !== 'shell');
</script>

{#if isVisible}
	<div class="focus-bar">
		{#each FOCUS_PILLS as pill (pill.id)}
			<button
				class="focus-pill"
				class:active={activeFocusMode === pill.id}
				style="--accent: {providerColor}"
				onclick={() => onFocusModeChange(pill.id)}
			>
				{pill.label}
			</button>
		{/each}
	</div>
{/if}

<style>
	.focus-bar {
		display: flex;
		gap: 3px;
		padding: 3px 8px;
		background: #0a0a0a;
		border-bottom: 1px solid #1a1a1a;
		flex-shrink: 0;
		overflow-x: auto;
	}

	.focus-bar::-webkit-scrollbar {
		display: none;
	}

	.focus-pill {
		padding: 2px 8px;
		border-radius: 8px;
		border: 1px solid #2a2a2a;
		background: transparent;
		color: #666;
		font-family: 'SF Mono', monospace;
		font-size: 9px;
		font-weight: 500;
		cursor: pointer;
		transition: all 0.12s ease;
		white-space: nowrap;
		text-transform: uppercase;
		letter-spacing: 0.3px;
	}

	.focus-pill:hover {
		border-color: #444;
		color: #aaa;
	}

	.focus-pill.active {
		background: var(--accent, #8b5cf6);
		color: #000;
		border-color: var(--accent, #8b5cf6);
		font-weight: 700;
	}
</style>
