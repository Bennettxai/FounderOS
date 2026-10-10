<script lang="ts">
	import { fly, fade } from 'svelte/transition';

	interface Props {
		userName: string;
		energyLevel?: number | null;
		onEnergySet?: (level: number) => void;
	}

	let { userName, energyLevel = null, onEnergySet }: Props = $props();

	let showEnergyCheck = $state(energyLevel === null);
	let sliderValue = $state(energyLevel ?? 5);

	// Dynamic greeting based on time
	const getGreeting = () => {
		const hour = new Date().getHours();
		if (hour >= 5 && hour < 12) return 'Good morning';
		if (hour >= 12 && hour < 17) return 'Good afternoon';
		if (hour >= 17 && hour < 21) return 'Good evening';
		return 'Working late';
	};

	// Format date — compact, no year
	const formatDate = () => {
		return new Date().toLocaleDateString('en-US', {
			weekday: 'long',
			month: 'long',
			day: 'numeric'
		});
	};

	// Energy level labels
	const getEnergyLabel = (level: number): string => {
		if (level <= 2) return 'Low';
		if (level <= 4) return 'Below Average';
		if (level <= 6) return 'Average';
		if (level <= 8) return 'Good';
		return 'Excellent';
	};

	// Energy level colors — returns hex for inline style (no Tailwind dark: needed)
	const getEnergyHex = (level: number): string => {
		if (level <= 2) return '#ef4444';
		if (level <= 4) return '#f97316';
		if (level <= 6) return '#eab308';
		if (level <= 8) return '#84cc16';
		return '#22c55e';
	};

	function handleSliderChange(event: Event) {
		const target = event.target as HTMLInputElement;
		sliderValue = parseInt(target.value, 10);
	}

	function handleEnergySubmit() {
		onEnergySet?.(sliderValue);
		showEnergyCheck = false;
	}

	function dismissEnergyCheck() {
		showEnergyCheck = false;
	}
	
	// Get a contextual message based on time
	const getContextMessage = () => {
		const hour = new Date().getHours();
		if (hour >= 5 && hour < 9) return "Start the day strong";
		if (hour >= 9 && hour < 12) return "Peak productivity hours";
		if (hour >= 12 && hour < 14) return "Focused work time";
		if (hour >= 14 && hour < 17) return "Keep the momentum";
		if (hour >= 17 && hour < 21) return "Wrapping up";
		return "Late night session";
	};
</script>

<header class="dw-header" in:fade={{ duration: 300 }}>
	<div class="dw-header-content">
		<div class="flex items-start justify-between">
			<div>
				<h1
					class="dw-header-title"
					in:fly={{ y: -10, duration: 400, delay: 100 }}
				>
					{getGreeting()}, {userName}
				</h1>
				<p class="dw-header-subtitle" in:fly={{ y: -10, duration: 400, delay: 200 }}>
					<span>{formatDate()}</span>
					<span class="dw-header-dot">&#8226;</span>
					<span class="dw-header-context">{getContextMessage()}</span>
				</p>
			</div>
		</div>

		{#if showEnergyCheck}
			<div
				class="dw-energy-card"
				in:fly={{ y: 10, duration: 400, delay: 300 }}
			>
				<div class="dw-energy-row">
					<p class="dw-energy-label">Energy</p>
					<div class="dw-energy-slider-wrap">
						<span class="dw-energy-bound">1</span>
						<input
							type="range"
							min="1"
							max="10"
							value={sliderValue}
							oninput={handleSliderChange}
							class="dw-energy-slider"
							aria-label="Energy level slider"
						/>
						<span class="dw-energy-bound">10</span>
					</div>
					<div class="dw-energy-dot" style="background: {getEnergyHex(sliderValue)}"></div>
					<span class="dw-energy-value">{sliderValue}</span>
					<span class="dw-energy-text">{getEnergyLabel(sliderValue)}</span>
					<button
						onclick={handleEnergySubmit}
						class="btn-pill btn-pill-primary btn-pill-sm"
					>
						Log
					</button>
					<button
						onclick={dismissEnergyCheck}
						class="btn-pill btn-pill-ghost btn-pill-icon btn-pill-xs"
						aria-label="Dismiss energy check"
					>
						<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path
								stroke-linecap="round"
								stroke-linejoin="round"
								stroke-width="2"
								d="M6 18L18 6M6 6l12 12"
							/>
						</svg>
					</button>
				</div>
			</div>
		{/if}
	</div>
</header>

<style>
	.dw-header {
		position: relative;
		overflow: hidden;
	}

	.dw-header-content {
		position: relative;
		padding: 1rem 1.5rem;
	}

	.dw-header-title {
		font-size: 1.5rem;
		font-weight: 700;
		color: var(--dt, #111);
		letter-spacing: -0.015em;
	}

	.dw-header-subtitle {
		font-size: 0.875rem;
		color: var(--dt3, #888);
		margin-top: 0.25rem;
		display: flex;
		align-items: center;
		gap: 0.5rem;
	}

	.dw-header-dot {
		color: var(--dbd, #e0e0e0);
	}

	.dw-header-context {
		color: var(--dt2, #555);
	}

	.dw-energy-card {
		margin-top: 0.75rem;
		padding: 0.625rem 0.875rem;
		background: color-mix(in srgb, var(--dbg2, #f5f5f5) 80%, transparent);
		backdrop-filter: blur(12px);
		-webkit-backdrop-filter: blur(12px);
		border-radius: var(--radius-md, 12px);
		border: 1px solid var(--dbd2, #f0f0f0);
		box-shadow: var(--shadow-sm, 0 1px 3px rgba(0,0,0,0.08));
		transition: all 300ms ease;
	}

	.dw-energy-row {
		display: flex;
		align-items: center;
		gap: 0.5rem;
	}

	.dw-energy-slider-wrap {
		display: flex;
		align-items: center;
		gap: 0.375rem;
		flex: 1;
		min-width: 0;
	}

	.dw-energy-label {
		font-size: 0.78rem;
		font-weight: 500;
		color: var(--dt3, #888);
		white-space: nowrap;
		flex-shrink: 0;
	}

	.dw-energy-bound {
		font-size: 0.7rem;
		color: var(--dt4, #aaa);
		flex-shrink: 0;
	}

	.dw-energy-slider {
		flex: 1;
		height: 0.375rem;
		background: var(--dbd, #e0e0e0);
		border-radius: 0.5rem;
		appearance: none;
		cursor: pointer;
		accent-color: var(--dt, #111);
		min-width: 60px;
	}

	.dw-energy-dot {
		width: 0.75rem;
		height: 0.75rem;
		border-radius: 50%;
		transition: background 0.2s;
	}

	.dw-energy-value {
		font-size: 0.875rem;
		font-weight: 500;
		color: var(--dt, #111);
	}

	.dw-energy-text {
		font-size: 0.875rem;
		color: var(--dt3, #888);
	}
</style>
