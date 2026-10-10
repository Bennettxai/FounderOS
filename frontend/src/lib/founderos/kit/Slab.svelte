<!-- The floating surface every page sits on (FounderOS v1 .os-slab): 28px corners, hairline, a soft drop. -->
<script lang="ts">
	import type { Snippet } from 'svelte';
	import './kit.css';

	let { class: className = '', children }: { class?: string; children?: Snippet } = $props();
</script>

<div class="bn-slab {className}">{@render children?.()}</div>
