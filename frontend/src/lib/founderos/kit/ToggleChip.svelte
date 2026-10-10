<!-- A 24px toggle pill (FounderOS v1 components/Pressable.tsx Chip): the palette
     scopes and any on/off filter. Solid when on, the dark lens when off. -->
<script lang="ts">
	import type { Snippet } from 'svelte';
	import './kit.css';

	let {
		on = false,
		class: className = '',
		children,
		...rest
	}: { on?: boolean; class?: string; children?: Snippet; [key: string]: unknown } = $props();
</script>

<button
	type="button"
	data-lens="c"
	data-on={on ? 'true' : 'false'}
	class="bn-pressable bn-toggle-chip inline-flex h-6 shrink-0 items-center whitespace-nowrap gap-1.5 rounded-full border px-2.5 font-mono text-[10px] font-semibold {on ? '' : 'is-dark'} {className}"
	{...rest}>{@render children?.()}</button
>
