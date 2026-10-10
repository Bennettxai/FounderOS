<!-- A card's small uppercase section label with its lucide icon. -->
<script lang="ts">
	import type { Snippet } from 'svelte';
	import type { IconComponent } from '$lib/founderos/icons';
	let { icon: Icon, children }: { icon: IconComponent; children: Snippet } = $props();
</script>

<div class="bn-dim mb-1.5 flex items-center gap-1.5 font-mono text-[9px] uppercase tracking-[0.16em]">
	<Icon size={12} />
	{@render children()}
</div>
