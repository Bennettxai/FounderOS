<!--
	GradientBackground.svelte
	Full-screen background that fills viewport edge-to-edge
	Uses solid light gray background like Wabi iOS app

	Usage:
	<GradientBackground>
		<YourContent />
	</GradientBackground>
-->
<script lang="ts">
	interface Props {
		variant?: string;
		fullScreen?: boolean;
		children?: import('svelte').Snippet;
	}

	let {
		variant = 'default',
		fullScreen = true,
		children
	}: Props = $props();
</script>

<div class="gradient-background" role="presentation">
	{#if children}
		{@render children()}
	{/if}
</div>

<style>
	.gradient-background {
		/* Full viewport coverage - edge to edge */
		min-height: 100vh;
		width: 100vw;

		/* Solid light gray background like Wabi */
		background: #EFEFEF;

		/* Ensure no overflow */
		overflow-x: hidden;

		/* Positioning */
		position: relative;
	}
</style>
