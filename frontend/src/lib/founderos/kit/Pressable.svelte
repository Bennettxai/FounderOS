<!-- The one clickable primitive of the interaction rebrand (FounderOS v1
     components/Pressable.tsx): hover lens + press sink + focus ring, themed
     per tone. Renders a link when given an href. -->
<script lang="ts">
	import type { Snippet } from 'svelte';
	import './kit.css';

	type Tone = 'primary' | 'secondary' | 'ghost' | 'row';

	const TONE: Record<Tone, string> = {
		primary: 'is-primary inline-flex h-[26px] items-center gap-1.5 rounded-[6px] border px-3 font-mono text-[10.5px] font-bold',
		secondary: 'is-dark inline-flex h-[26px] items-center gap-1.5 rounded-[6px] border px-2.5 font-mono text-[10.5px] font-semibold',
		ghost: 'is-dark inline-grid h-[26px] w-[26px] place-items-center rounded-[6px] border',
		row: 'is-row block'
	};

	let {
		tone = 'secondary',
		kind,
		href,
		class: className = '',
		children,
		...rest
	}: {
		tone?: Tone;
		/** Lens strength: "c" control (4px pull) or "r" row (2px). Rows default to r. */
		kind?: 'c' | 'r';
		href?: string;
		class?: string;
		children?: Snippet;
		[key: string]: unknown;
	} = $props();

	const lens = $derived(kind ?? (tone === 'row' ? 'r' : 'c'));
	const cls = $derived(`bn-pressable ${TONE[tone]} ${className}`.trim());
</script>

{#if href}
	<a {href} data-lens={lens} data-tone={tone} class={cls} {...rest}>{@render children?.()}</a>
{:else}
	<button type="button" data-lens={lens} data-tone={tone} class={cls} {...rest}>{@render children?.()}</button>
{/if}
