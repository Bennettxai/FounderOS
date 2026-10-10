<!--
	AgentRuntimeSelector.svelte
	Compact runtime picker chip for the dock chat.
	Trigger: chip showing active runtime name.
	Dropdown: opens upward, glassmorphism — mirrors ModelSelector exactly.
-->
<script lang="ts">
	import { osaStore, AGENT_RUNTIME_OPTIONS, type AgentRuntime } from '$lib/stores/osa';

	interface Props {
		class?: string;
		compact?: boolean;
	}

	let { class: className = '', compact = false }: Props = $props();

	// ─── State ───────────────────────────────────────────────────────────────────

	let isOpen = $state(false);
	let dropdownElement: HTMLDivElement | undefined = $state(undefined);

	// ─── Derived ─────────────────────────────────────────────────────────────────

	let activeRuntime = $derived($osaStore.activeRuntime);

	let activeOption = $derived(
		AGENT_RUNTIME_OPTIONS.find((r) => r.id === activeRuntime) ?? AGENT_RUNTIME_OPTIONS[0]
	);

	let displayLabel = $derived(truncate(activeOption.label, 12));

	// ─── Helpers ─────────────────────────────────────────────────────────────────

	function truncate(str: string, max: number): string {
		return str.length > max ? str.slice(0, max - 1) + '…' : str;
	}

	function selectRuntime(id: AgentRuntime) {
		osaStore.setRuntime(id);
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
</script>

<div bind:this={dropdownElement} class="ars-selector {className}">

	<!-- Trigger chip -->
	<button
		class="ars-trigger"
		class:has-runtime={true}
		role="combobox"
		aria-expanded={isOpen}
		aria-controls="osa-runtime-panel"
		aria-label="Select agent runtime: {activeOption.label}"
		aria-haspopup="dialog"
		title={activeOption.label}
		onclick={(e) => { e.stopPropagation(); isOpen = !isOpen; }}
		onkeydown={handleKeyDown}
	>
		<!-- Agent/circuit icon — visually distinct from ModelSelector's CPU chip -->
		<svg class="ars-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true">
			<circle cx="12" cy="12" r="3" />
			<path d="M12 2v3M12 19v3M2 12h3M19 12h3" />
			<path d="M4.93 4.93l2.12 2.12M16.95 16.95l2.12 2.12M4.93 19.07l2.12-2.12M16.95 7.05l2.12-2.12" />
		</svg>
		<!-- Colored status dot matching the active runtime's color -->
		<span
			class="ars-dot"
			style="background-color: {activeOption.color}"
			aria-hidden="true"
		></span>
		{#if !compact}
			<span class="ars-label">{displayLabel}</span>
			<svg
				class="ars-chevron"
				class:open={isOpen}
				viewBox="0 0 24 24"
				fill="none"
				stroke="currentColor"
				stroke-width="2.5"
				aria-hidden="true"
			>
				<polyline points="6 9 12 15 18 9" />
		</svg>
		{/if}
	</button>

	<!-- Dropdown — opens upward -->
	{#if isOpen}
		<div
			id="osa-runtime-panel"
			class="ars-dropdown"
			role="dialog"
			aria-label="Agent runtime selection"
		>
			<div class="ars-section-label">Agent Runtime</div>

			<div class="ars-options" role="listbox" aria-label="Available runtimes">
				{#each AGENT_RUNTIME_OPTIONS as option}
					<button
						class="ars-option"
						class:active={activeRuntime === option.id}
						role="option"
						aria-selected={activeRuntime === option.id}
						type="button"
						onclick={() => selectRuntime(option.id)}
					>
						<!-- Colored runtime dot -->
						<span
							class="ars-runtime-dot"
							style="background-color: {option.color}"
							aria-hidden="true"
						></span>

						<!-- Runtime name -->
						<span class="ars-runtime-name">{option.label}</span>

						<!-- Badge: Built-in vs CLI -->
						{#if option.id === 'osa'}
							<span class="ars-badge builtin">Built-in</span>
						{:else}
							<span class="ars-badge cli">CLI</span>
						{/if}

						<!-- Active checkmark -->
						{#if activeRuntime === option.id}
							<svg
								class="ars-check"
								viewBox="0 0 24 24"
								fill="none"
								stroke="currentColor"
								stroke-width="2.5"
								aria-hidden="true"
							>
								<polyline points="20 6 9 17 4 12" />
							</svg>
						{/if}
					</button>
				{/each}
			</div>
		</div>
	{/if}
</div>

<style>
	.ars-selector {
		position: relative;
		flex-shrink: 0;
	}

	/* ===== TRIGGER ===== */
	.ars-trigger {
		display: flex;
		align-items: center;
		gap: 4px;
		padding: 4px 8px 4px 6px;
		background: rgba(0, 0, 0, 0.04);
		border: 1px solid rgba(0, 0, 0, 0.06);
		border-radius: 8px;
		cursor: pointer;
		transition: all 0.15s ease;
		max-width: 160px;
	}

	.ars-trigger:hover {
		background: rgba(0, 0, 0, 0.07);
	}

	:global(.dark) .ars-trigger {
		background: rgba(255, 255, 255, 0.06);
		border-color: rgba(255, 255, 255, 0.08);
	}

	:global(.dark) .ars-trigger:hover {
		background: rgba(255, 255, 255, 0.1);
	}

	/* Agent icon — sunburst/circuit style, visually different from ModelSelector's CPU box */
	.ars-icon {
		width: 14px;
		height: 14px;
		color: #8e8e93;
		flex-shrink: 0;
	}

	:global(.dark) .ars-icon {
		color: #636366;
	}

	/* Small colored dot next to label — indicates active runtime at a glance */
	.ars-dot {
		width: 5px;
		height: 5px;
		border-radius: 50%;
		flex-shrink: 0;
	}

	.ars-label {
		font-size: 11px;
		font-weight: 500;
		font-family: ui-monospace, 'SF Mono', Menlo, monospace;
		color: #1c1c1e;
		letter-spacing: 0.01em;
		max-width: 80px;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	:global(.dark) .ars-label {
		color: #f5f5f7;
	}

	.ars-chevron {
		width: 10px;
		height: 10px;
		color: #aeaeb2;
		transition: transform 0.2s ease;
		flex-shrink: 0;
	}

	.ars-chevron.open {
		transform: rotate(180deg);
	}

	/* ===== DROPDOWN ===== */
	.ars-dropdown {
		position: absolute;
		bottom: calc(100% + 8px);
		left: 0;
		z-index: 10100;
		min-width: 220px;
		padding: 10px 8px 8px;
		background: rgba(255, 255, 255, 0.85);
		backdrop-filter: blur(32px) saturate(1.6);
		-webkit-backdrop-filter: blur(32px) saturate(1.6);
		border: 1px solid rgba(255, 255, 255, 0.7);
		border-radius: 16px;
		box-shadow:
			0 16px 48px rgba(0, 0, 0, 0.12),
			0 4px 12px rgba(0, 0, 0, 0.06),
			inset 0 1px 0 rgba(255, 255, 255, 0.8);
		animation: ars-dropdownIn 0.15s ease-out;
		display: flex;
		flex-direction: column;
		gap: 4px;
	}

	:global(.dark) .ars-dropdown {
		background: rgba(36, 36, 38, 0.9);
		border-color: rgba(255, 255, 255, 0.1);
		box-shadow:
			0 16px 48px rgba(0, 0, 0, 0.5),
			0 4px 12px rgba(0, 0, 0, 0.3),
			inset 0 1px 0 rgba(255, 255, 255, 0.06);
	}

	@keyframes ars-dropdownIn {
		from {
			opacity: 0;
			transform: translateY(4px) scale(0.97);
		}
		to {
			opacity: 1;
			transform: translateY(0) scale(1);
		}
	}

	/* ===== SECTION LABEL ===== */
	.ars-section-label {
		font-size: 10px;
		font-weight: 600;
		letter-spacing: 0.06em;
		text-transform: uppercase;
		color: #8e8e93;
		padding: 0 4px;
		margin-top: 2px;
	}

	:global(.dark) .ars-section-label {
		color: #636366;
	}

	/* ===== OPTIONS LIST ===== */
	.ars-options {
		display: flex;
		flex-direction: column;
		gap: 1px;
	}

	.ars-option {
		display: flex;
		align-items: center;
		gap: 8px;
		padding: 8px 10px;
		border-radius: 10px;
		cursor: pointer;
		transition: background 0.12s ease;
		user-select: none;
		background: transparent;
		border: none;
		width: 100%;
		text-align: left;
	}

	.ars-option:hover {
		background: rgba(0, 0, 0, 0.05);
	}

	.ars-option.active {
		background: rgba(0, 0, 0, 0.06);
	}

	:global(.dark) .ars-option:hover {
		background: rgba(255, 255, 255, 0.07);
	}

	:global(.dark) .ars-option.active {
		background: rgba(255, 255, 255, 0.08);
	}

	.ars-runtime-dot {
		width: 6px;
		height: 6px;
		border-radius: 50%;
		flex-shrink: 0;
	}

	.ars-runtime-name {
		font-size: 13px;
		font-weight: 600;
		color: #1c1c1e;
		flex: 1;
	}

	:global(.dark) .ars-runtime-name {
		color: #f5f5f7;
	}

	/* ===== BADGES ===== */
	.ars-badge {
		font-size: 9px;
		font-weight: 600;
		letter-spacing: 0.04em;
		padding: 2px 5px;
		border-radius: 4px;
		flex-shrink: 0;
	}

	.ars-badge.builtin {
		background: rgba(34, 197, 94, 0.1);
		color: #16a34a;
	}

	.ars-badge.cli {
		background: rgba(59, 130, 246, 0.1);
		color: #2563eb;
	}

	:global(.dark) .ars-badge.builtin {
		background: rgba(34, 197, 94, 0.15);
		color: #4ade80;
	}

	:global(.dark) .ars-badge.cli {
		background: rgba(59, 130, 246, 0.15);
		color: #60a5fa;
	}

	/* ===== CHECKMARK ===== */
	.ars-check {
		width: 14px;
		height: 14px;
		color: #007aff;
		flex-shrink: 0;
	}

	:global(.dark) .ars-check {
		color: #0a84ff;
	}
</style>
