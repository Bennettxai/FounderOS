<!-- 900ms ease-out count-in (FounderOS v1 CountUp.tsx). The first paint counts from
     zero; a new target counts from the number already on screen; the last frame
     lands exactly; reduced motion lands instantly. -->
<script lang="ts">
	import { untrack } from 'svelte';
	import { countFrame, formatCount, prefersReducedMotion, type CountKind } from './format';

	let { value, kind = 'int', ms = 900 }: { value: number; kind?: CountKind; ms?: number } = $props();

	let shown = $state(untrack(() => (prefersReducedMotion() ? value : 0)));

	$effect(() => {
		const target = value;
		if (prefersReducedMotion() || typeof requestAnimationFrame !== 'function') {
			shown = target;
			return;
		}
		const from = untrack(() => shown);
		const start = performance.now();
		let raf = 0;
		const step = (now: number) => {
			const next = countFrame(from, target, (now - start) / ms);
			shown = next;
			if (next !== target) raf = requestAnimationFrame(step);
		};
		raf = requestAnimationFrame(step);
		return () => cancelAnimationFrame(raf);
	});
</script>

{formatCount(kind, shown)}
