<!-- Renders a prop that may be plain text or a snippet (the React originals take ReactNode). -->
<script lang="ts">
	import type { Snippet } from 'svelte';
	let { value }: { value: string | number | Snippet | null | undefined } = $props();
</script>

{#if typeof value === 'function'}{@render value()}{:else if value != null}{value}{/if}
