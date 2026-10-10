<!-- The Vantage emblem (components/SparkIcon.tsx): the PNG is a CSS mask over
     a solid color, so `shade` tints the brand silhouette any color. -->
<script lang="ts">
	import emblem from './vantage-emblem.png';

	let { size = 28, shade = 'var(--bn-accent)' }: { size?: number; shade?: string } = $props();

	const style = $derived(
		[
			`width: ${size}px`,
			`height: ${size}px`,
			`background-color: ${shade}`,
			`-webkit-mask-image: url(${emblem})`,
			`mask-image: url(${emblem})`,
			'-webkit-mask-repeat: no-repeat',
			'mask-repeat: no-repeat',
			'-webkit-mask-position: center',
			'mask-position: center',
			'-webkit-mask-size: contain',
			'mask-size: contain'
		].join('; ')
	);
</script>

<span role="img" aria-label="Vantage" class="inline-block shrink-0" {style}></span>
