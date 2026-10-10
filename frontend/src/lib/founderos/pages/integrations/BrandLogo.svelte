<!-- One brand mark on a tinted tile (FounderOS v1 lib/brand-logos.tsx BrandLogo):
     radius 28% and a 56% glyph of the tile size. The SVG markup is our own
     generated data (brand-marks.ts), never user input. -->
<script lang="ts">
	import { brandMark } from './brand-logos';

	let { slug, name, size = 40 }: { slug: string; name: string; size?: number } = $props();

	const mark = $derived(brandMark(slug, name));
	const radius = $derived(Math.round(size * 0.28));
	const glyph = $derived(Math.round(size * 0.56));
	const box = $derived(`width: ${size}px; height: ${size}px; border-radius: ${radius}px; background: ${mark.tile}`);
</script>

{#if mark.kind === 'handmade'}
	<span class="grid shrink-0 place-items-center" data-mark="handmade" style={box}>
		<svg width={glyph} height={glyph} viewBox={mark.viewBox} aria-hidden="true">{@html mark.svg}</svg>
	</span>
{:else if mark.kind === 'vector'}
	<span class="grid shrink-0 place-items-center" data-mark="vector" style="{box}{mark.mono ? '; color: #e6e7ea' : ''}">
		<svg width={glyph} height={glyph} viewBox={mark.viewBox} fill={mark.mono ? 'currentColor' : 'none'} aria-hidden="true"
			>{@html mark.svg}</svg
		>
	</span>
{:else if mark.kind === 'icon'}
	<span class="grid shrink-0 place-items-center" data-mark="icon" style={box}>
		<svg width={glyph} height={glyph} viewBox="0 0 24 24" aria-hidden="true"><path d={mark.path} fill={mark.fill} /></svg>
	</span>
{:else if mark.kind === 'raster'}
	<span class="grid shrink-0 place-items-center" data-mark="raster" style={box}>
		<img
			src={mark.src}
			alt=""
			width={glyph}
			height={glyph}
			loading="lazy"
			decoding="async"
			style="width: {glyph}px; height: {glyph}px; object-fit: contain"
			aria-hidden="true"
		/>
	</span>
{:else}
	<span
		class="grid shrink-0 place-items-center font-semibold"
		data-mark="lettermark"
		style="{box}; color: {mark.color}; font-size: {Math.round(size * 0.42)}px"
		aria-hidden="true">{mark.initial}</span
	>
{/if}
