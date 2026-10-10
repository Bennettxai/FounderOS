<!--
	ModeIndicator.svelte
	Minimal badge showing the current active OSA mode.
	Used on each OSA message in ResponseStream.
	Colored dot + subtle label.
-->
<script lang="ts">
	import { osaStore, MODE_COLORS, type OsaMode } from '$lib/stores/osa';

	interface Props {
		mode?: OsaMode;
		confidence?: number;
		compact?: boolean;
	}

	let { mode, confidence, compact = false }: Props = $props();

	let activeMode = $derived(mode ?? $osaStore.activeMode);
	let activeConfidence = $derived(confidence ?? $osaStore.modeConfidence);
	let confidencePercent = $derived(Math.round(activeConfidence * 100));

	let dotColor = $derived(MODE_COLORS[activeMode] ?? MODE_COLORS.ASSIST);
	let label = $derived(
		compact || confidencePercent === 0
			? activeMode
			: `${activeMode} ${confidencePercent}%`
	);
	let ariaLabel = $derived(`${activeMode} mode, ${confidencePercent}% confidence`);
</script>

<span class="mode-indicator" role="status" aria-label={ariaLabel}>
	<span class="indicator-dot" style="background-color: {dotColor}"></span>
	<span class="indicator-label">{label}</span>
</span>

<style>
	.mode-indicator {
		display: inline-flex;
		align-items: center;
		gap: 5px;
	}

	.indicator-dot {
		width: 6px;
		height: 6px;
		border-radius: 50%;
		flex-shrink: 0;
	}

	.indicator-label {
		font-size: 10px;
		font-weight: 600;
		color: #8e8e93;
		letter-spacing: 0.04em;
		text-transform: uppercase;
	}

	:global(.dark) .indicator-label {
		color: #636366;
	}
</style>
