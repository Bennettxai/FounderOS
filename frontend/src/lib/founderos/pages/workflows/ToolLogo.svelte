<!-- A workflow tool's company logo (prod BrandLogo via lib/workflow-tool-brands):
     the Connections brand set first, then the few simple-icons glyphs only the
     process map needs (tool-marks.ts), else BrandLogo's lettermark. -->
<script lang="ts">
	import BrandLogo from '../integrations/BrandLogo.svelte';
	import { brandMarkKind } from '../integrations/brand-logos';
	import { TOOL_ICONS } from './tool-marks';

	let { slug, name, size = 12 }: { slug: string; name: string; size?: number } = $props();

	const local = $derived(brandMarkKind(slug) === 'none' || brandMarkKind(slug) === 'lettermark' ? TOOL_ICONS[slug] : undefined);
	// prod's rule: a near-black brand draws in the foreground ink on a faint tile
	const dark = $derived(local ? local.hex === '000000' : false);
	const glyph = $derived(Math.round(size * 0.56));
</script>

{#if local}
	<span
		class="grid shrink-0 place-items-center"
		data-mark="icon"
		style="width: {size}px; height: {size}px; border-radius: {Math.round(size * 0.28)}px; background: {dark ? 'rgba(255,255,255,0.06)' : `color-mix(in srgb, #${local.hex} 16%, transparent)`}"
	>
		<svg width={glyph} height={glyph} viewBox="0 0 24 24" aria-hidden="true"><path d={local.path} fill={dark ? '#e6e7ea' : `#${local.hex}`} /></svg>
	</span>
{:else}
	<BrandLogo {slug} {name} {size} />
{/if}
