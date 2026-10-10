<!--
	ModeSelector.svelte
	Compact mode picker for the 5 OSA modes.
	Inline trigger with colored dot + glass dropdown.
-->
<script lang="ts">
	import { Hammer, MessageSquare, ChartBar, Zap, Wrench } from 'lucide-svelte';
	import { osaStore, MODE_COLORS, type OsaMode } from '$lib/stores/osa';
	import { onMount } from 'svelte';

	interface Props {
		compact?: boolean;
	}

	let { compact = false }: Props = $props();

	let isOpen = $state(false);
	let dropdownElement: HTMLDivElement | undefined = $state(undefined);

	let activeMode = $derived($osaStore.activeMode);
	let modes = $derived($osaStore.modes);

	// Load modes + health from API on first mount (falls back silently)
	onMount(() => { osaStore.loadModes(); osaStore.loadHealth(); });

	const MODE_ICONS = {
		BUILD: Hammer,
		ASSIST: MessageSquare,
		ANALYZE: ChartBar,
		EXECUTE: Zap,
		MAINTAIN: Wrench
	} as const;

	function selectMode(mode: OsaMode) {
		osaStore.setMode(mode);
		isOpen = false;
	}

	function handleKeyDown(e: KeyboardEvent) {
		if (e.key === 'Escape') isOpen = false;
	}

	function handleClickOutside(e: MouseEvent) {
		if (dropdownElement && !dropdownElement.contains(e.target as Node)) {
			isOpen = false;
		}
	}

	$effect(() => {
		if (isOpen) {
			document.addEventListener('click', handleClickOutside, true);
			return () => document.removeEventListener('click', handleClickOutside, true);
		}
	});

	function getModeIcon(mode: string) {
		return MODE_ICONS[mode as keyof typeof MODE_ICONS] ?? Wrench;
	}
</script>

<div
	bind:this={dropdownElement}
	class="mode-selector"
>
	<!-- Trigger: dot + label -->
	<button
		class="mode-trigger"
		class:compact
		role="combobox"
		aria-expanded={isOpen}
		aria-controls="osa-mode-listbox"
		aria-label="Select OSA mode: {activeMode}"
		aria-haspopup="listbox"
		title={compact ? activeMode : undefined}
		onclick={(e) => { e.stopPropagation(); isOpen = !isOpen; }}
		onkeydown={handleKeyDown}
	>
		<span class="mode-dot" style="background-color: {MODE_COLORS[activeMode]}"></span>
		{#if !compact}
			<span class="mode-label">{activeMode}</span>
			<svg
				class="mode-chevron"
				class:open={isOpen}
				viewBox="0 0 24 24"
				fill="none"
				stroke="currentColor"
				stroke-width="2.5"
			>
				<polyline points="6 9 12 15 18 9" />
			</svg>
		{/if}
	</button>

	<!-- Dropdown -->
	{#if isOpen}
		<div id="osa-mode-listbox" class="mode-dropdown" role="listbox" aria-label="OSA modes">
			{#each modes as m}
				{@const ModeIcon = getModeIcon(m.mode)}
				{@const isActive = m.mode === activeMode}
				<button
					class="mode-option"
					class:active={isActive}
					role="option"
					aria-selected={isActive}
					onclick={() => selectMode(m.mode)}
				>
					<span class="option-dot" style="background-color: {MODE_COLORS[m.mode]}"></span>
					<ModeIcon class="option-icon" aria-hidden="true" />
					<div class="option-text">
						<span class="option-label">{m.label}</span>
						{#if !compact}
							<span class="option-desc">{m.description}</span>
						{/if}
					</div>
					{#if isActive}
						<svg class="option-check" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
							<polyline points="20 6 9 17 4 12" />
						</svg>
					{/if}
				</button>
			{/each}
		</div>
	{/if}
</div>

<style>
	.mode-selector {
		position: relative;
		flex-shrink: 0;
	}

	/* ===== TRIGGER ===== */
	.mode-trigger {
		display: flex;
		align-items: center;
		gap: 6px;
		padding: 6px 10px;
		background: rgba(0, 0, 0, 0.04);
		border: 1px solid rgba(0, 0, 0, 0.06);
		border-radius: 10px;
		cursor: pointer;
		transition: all 0.15s ease;
	}

	.mode-trigger:hover {
		background: rgba(0, 0, 0, 0.07);
	}

	:global(.dark) .mode-trigger {
		background: rgba(255, 255, 255, 0.06);
		border-color: rgba(255, 255, 255, 0.08);
	}

	:global(.dark) .mode-trigger:hover {
		background: rgba(255, 255, 255, 0.1);
	}

	.mode-trigger.compact {
		padding: 6px;
		gap: 0;
		border-radius: 50%;
		width: 28px;
		height: 28px;
		justify-content: center;
	}

	.mode-dot {
		width: 8px;
		height: 8px;
		border-radius: 50%;
		flex-shrink: 0;
		box-shadow: 0 0 6px currentColor;
	}

	.compact .mode-dot {
		width: 10px;
		height: 10px;
	}

	.mode-label {
		font-size: 12px;
		font-weight: 600;
		color: #1c1c1e;
		letter-spacing: 0.02em;
	}

	:global(.dark) .mode-label {
		color: #f5f5f7;
	}

	.mode-chevron {
		width: 12px;
		height: 12px;
		color: #8e8e93;
		transition: transform 0.2s ease;
		flex-shrink: 0;
	}

	.mode-chevron.open {
		transform: rotate(180deg);
	}

	/* ===== DROPDOWN ===== */
	.mode-dropdown {
		position: absolute;
		bottom: calc(100% + 8px);
		left: 0;
		z-index: 10100;
		min-width: 240px;
		padding: 6px;
		background: rgba(255, 255, 255, 0.85);
		backdrop-filter: blur(32px) saturate(1.6);
		-webkit-backdrop-filter: blur(32px) saturate(1.6);
		border: 1px solid rgba(255, 255, 255, 0.7);
		border-radius: 16px;
		box-shadow:
			0 16px 48px rgba(0, 0, 0, 0.12),
			0 4px 12px rgba(0, 0, 0, 0.06),
			inset 0 1px 0 rgba(255, 255, 255, 0.8);
		animation: dropdownIn 0.15s ease-out;
	}

	:global(.dark) .mode-dropdown {
		background: rgba(36, 36, 38, 0.9);
		border-color: rgba(255, 255, 255, 0.1);
		box-shadow:
			0 16px 48px rgba(0, 0, 0, 0.5),
			0 4px 12px rgba(0, 0, 0, 0.3),
			inset 0 1px 0 rgba(255, 255, 255, 0.06);
	}

	@keyframes dropdownIn {
		from {
			opacity: 0;
			transform: translateY(4px) scale(0.97);
		}
		to {
			opacity: 1;
			transform: translateY(0) scale(1);
		}
	}

	/* ===== OPTIONS ===== */
	.mode-option {
		display: flex;
		align-items: center;
		gap: 10px;
		width: 100%;
		padding: 10px 12px;
		border-radius: 10px;
		border: none;
		background: transparent;
		cursor: pointer;
		transition: background 0.12s ease;
		text-align: left;
	}

	.mode-option:hover {
		background: rgba(0, 0, 0, 0.05);
	}

	.mode-option.active {
		background: rgba(0, 0, 0, 0.06);
	}

	:global(.dark) .mode-option:hover {
		background: rgba(255, 255, 255, 0.07);
	}

	:global(.dark) .mode-option.active {
		background: rgba(255, 255, 255, 0.08);
	}

	.option-dot {
		width: 6px;
		height: 6px;
		border-radius: 50%;
		flex-shrink: 0;
	}

	.mode-option :global(.option-icon) {
		width: 16px;
		height: 16px;
		color: #636366;
		flex-shrink: 0;
	}

	:global(.dark) .mode-option :global(.option-icon) {
		color: #98989d;
	}

	.option-text {
		display: flex;
		flex-direction: column;
		gap: 1px;
		flex: 1;
		min-width: 0;
	}

	.option-label {
		font-size: 13px;
		font-weight: 600;
		color: #1c1c1e;
	}

	:global(.dark) .option-label {
		color: #f5f5f7;
	}

	.option-desc {
		font-size: 11px;
		color: #8e8e93;
		line-height: 1.3;
	}

	:global(.dark) .option-desc {
		color: #636366;
	}

	.option-check {
		width: 16px;
		height: 16px;
		color: #007aff;
		flex-shrink: 0;
	}

	:global(.dark) .option-check {
		color: #0a84ff;
	}
</style>
