<script lang="ts">
	import {
		desktopSettings,
		type AnimatedBackgroundEffect,
		type AnimatedBackgroundIntensity
	} from '$lib/stores/desktopStore';
	import AnimatedBackgroundSection from './AnimatedBackgroundSection.svelte';
	import WindowAnimationsSection from './WindowAnimationsSection.svelte';

	// Effects preview state - allows preview before applying
	let previewEffect = $state<AnimatedBackgroundEffect | null>(null);
	let previewIntensity = $state<AnimatedBackgroundIntensity | null>(null);
	let previewSpeed = $state<number | null>(null);
	let hasUnsavedEffectChanges = $state(false);

	// Get current or preview value for effects
	const effectiveEffect = $derived(previewEffect ?? $desktopSettings.animatedBackground.effect);
	const effectiveIntensity = $derived(previewIntensity ?? $desktopSettings.animatedBackground.intensity);
	const effectiveSpeed = $derived(previewSpeed ?? $desktopSettings.animatedBackground.speed);

	function previewEffectChange(effect: AnimatedBackgroundEffect) {
		previewEffect = effect;
		hasUnsavedEffectChanges = true;
	}

	function previewIntensityChange(intensity: AnimatedBackgroundIntensity) {
		previewIntensity = intensity;
		hasUnsavedEffectChanges = true;
	}

	function previewSpeedChange(speed: number) {
		previewSpeed = speed;
		hasUnsavedEffectChanges = true;
	}

	function applyEffectChanges() {
		const changes: Partial<typeof $desktopSettings.animatedBackground> = {};
		if (previewEffect !== null) changes.effect = previewEffect;
		if (previewIntensity !== null) changes.intensity = previewIntensity;
		if (previewSpeed !== null) changes.speed = previewSpeed;

		if (Object.keys(changes).length > 0) {
			desktopSettings.setAnimatedBackground(changes);
		}

		previewEffect = null;
		previewIntensity = null;
		previewSpeed = null;
		hasUnsavedEffectChanges = false;
	}

	function cancelEffectChanges() {
		previewEffect = null;
		previewIntensity = null;
		previewSpeed = null;
		hasUnsavedEffectChanges = false;
	}
</script>

<AnimatedBackgroundSection
	{previewEffect}
	{previewIntensity}
	{previewSpeed}
	{hasUnsavedEffectChanges}
	{effectiveEffect}
	{effectiveIntensity}
	{effectiveSpeed}
	onPreviewEffect={previewEffectChange}
	onPreviewIntensity={previewIntensityChange}
	onPreviewSpeed={previewSpeedChange}
	onApply={applyEffectChanges}
	onCancel={cancelEffectChanges}
/>

<WindowAnimationsSection />
