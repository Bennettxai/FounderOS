<!-- Stand-in for a /os view that has not been ported yet, so navigation never
     404s. Page agents replace the route's +page.svelte with the real view. -->
<script lang="ts">
	import PageHeader from '../kit/PageHeader.svelte';
	import Slab from '../kit/Slab.svelte';
	import { NAV_GROUPS } from '../nav';

	let { href }: { href: string } = $props();

	const group = $derived(NAV_GROUPS.find((g) => g.items.some((i) => i.href === href)));
	const label = $derived(group?.items.find((i) => i.href === href)?.label ?? href);
</script>

<PageHeader eyebrow={group?.title.toLowerCase()} title={label} />
<Slab>
	<div class="font-mono text-[10px] font-bold uppercase tracking-[0.26em] bn-dim">Porting in progress</div>
	<p class="bn-muted mt-3 font-mono text-[12px]">This view is being ported from FounderOS v1 ({href.replace(/^\/os/, '') || '/'}).</p>
</Slab>
