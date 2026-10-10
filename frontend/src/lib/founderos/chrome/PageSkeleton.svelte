<!-- The instant-paint fallback a click lands on while the next view loads
     (FounderOS v1 components/PageSkeleton.tsx via app/loading.tsx). No
     page-specific content on purpose: the Brand Deals slab every page now
     lands in, in outline. Title row, a 2fr/1fr hero with a volume card's meter
     tracks, a row of three, then one wide block. -->
<script lang="ts">
	import '../kit/kit.css';
</script>

{#snippet block(cls: string)}
	<div data-skel class="bn-skel-block animate-pulse rounded-[12px] border {cls}"></div>
{/snippet}

<div data-part="page-skeleton" aria-busy="true" aria-label="Loading" class="bn-slab">
	<div class="mb-7 flex items-end justify-between gap-4">
		<div>
			{@render block('mb-3 h-2.5 w-32')}
			{@render block('h-11 w-64')}
			{@render block('mt-3 h-2.5 w-80 max-w-full')}
		</div>
		{@render block('h-9 w-32 !rounded-full')}
	</div>
	<div class="grid grid-cols-[2fr_1fr] gap-6 max-[1200px]:grid-cols-1">
		{@render block('h-72')}
		<!-- the volume card in outline: a big numeral, a caption, three meter tracks -->
		<div class="bn-skel-card flex h-72 flex-col rounded-[12px] border px-6 py-5">
			<div class="bn-skel-bar h-4 w-32 animate-pulse rounded-full"></div>
			<div class="bn-skel-bar mt-5 h-10 w-40 animate-pulse rounded-[8px]"></div>
			<div class="bn-skel-rule mt-6 flex flex-1 flex-col justify-around border-t pt-4">
				{#each [0.7, 0.45, 0.85] as w (w)}
					<div data-skel-meter>
						<div class="bn-skel-bar mb-2 h-2.5 w-24 animate-pulse rounded-full"></div>
						<div class="bn-skel-track h-[10px] w-full rounded-full">
							<div data-part="fill" class="bn-skel-bar h-full animate-pulse rounded-full" style="width: {w * 100}%"></div>
						</div>
					</div>
				{/each}
			</div>
		</div>
	</div>
	<div class="mt-6 grid grid-cols-3 gap-6 max-[1200px]:grid-cols-1">
		{@render block('h-44')}
		{@render block('h-44')}
		{@render block('h-44')}
	</div>
	{@render block('mt-6 h-48')}
</div>

<style>
	.bn-skel-block,
	.bn-skel-card {
		border-color: var(--bn-border);
		background: var(--bn-surface);
	}
	.bn-skel-rule {
		border-color: var(--bn-border);
	}
	.bn-skel-bar {
		background: var(--bn-border);
	}
	.bn-skel-track {
		background: color-mix(in oklab, var(--bn-text) 6%, transparent);
	}
</style>
