<!-- The OS mark (FounderOS v1 components/OsMark.tsx): the chrome swirl emblem on
     transparent, the OS logo only, never the "Founder" wordmark. -->
<script lang="ts">
	import emblem from './os-emblem.png';

	let { size = 34, class: className = '' }: { size?: number; class?: string } = $props();
</script>

<img src={emblem} alt="Founder OS" width={size} height={size} style="width: {size}px; height: {size}px; object-fit: contain" class={className} />
