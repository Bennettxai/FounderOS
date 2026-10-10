<!-- Slab motion as a wrapper (FounderOS v1 components/motion.tsx Rise): rises in
     on the house ease, staggered by its index at 90ms a step. Reduced motion
     lands instantly (.bn-rise in kit.css). -->
<script lang="ts">
	import type { Snippet } from 'svelte';
	import './kit.css';

	let {
		i = 0,
		as = 'div',
		class: className = '',
		style = '',
		children,
		...rest
	}: {
		i?: number;
		as?: string;
		class?: string;
		style?: string;
		children?: Snippet;
		[key: string]: unknown;
	} = $props();

	const vars = $derived(`${style ? `${style.replace(/;?\s*$/, '')}; ` : ''}--rise-i: ${i}`);
</script>

<svelte:element this={as} class={`bn-rise ${className}`.trim()} style={vars} {...rest}>{@render children?.()}</svelte:element>
