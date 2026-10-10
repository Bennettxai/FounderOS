<!-- /os/brand-deals (spec 6.12): FounderOS v1 app/brand-deals/page.tsx. The Notion
     Brand Deals Hub in the Deal Journeys slab. The slab owns its own title row,
     so the shared page header is skipped deliberately. Read-only: the board
     rereads GET /pages/brand-deals every 60s or on refresh, keeping the last
     good read when a refresh fails. -->
<script lang="ts">
	import { onMount } from 'svelte';
	import { founderosFetch, FounderosAuthError } from '$lib/founderos/api';
	import { Slab, SlabTitle } from '$lib/founderos/kit';
	import DealBoard from '$lib/founderos/pages/brand-deals/DealBoard.svelte';
	import type { BrandDealsBody } from '$lib/founderos/pages/brand-deals/view';

	const REFRESH_MS = 60_000;

	let body = $state<BrandDealsBody | null>(null);
	let loadError = $state<string | null>(null);
	let refreshError = $state<string | null>(null);
	let pending = false;

	async function load() {
		if (pending) return;
		pending = true;
		try {
			body = await founderosFetch<BrandDealsBody>('/pages/brand-deals');
			loadError = null;
			refreshError = null;
		} catch (e) {
			if (e instanceof FounderosAuthError) return;
			const msg = e instanceof Error ? e.message : String(e);
			if (body) refreshError = msg;
			else loadError = msg;
		} finally {
			pending = false;
		}
	}

	onMount(() => {
		load();
		const t = setInterval(load, REFRESH_MS);
		return () => clearInterval(t);
	});
</script>

{#if body}
	<DealBoard {body} onRefresh={load} {refreshError} />
{:else}
	<Slab>
		<SlabTitle eyebrow="sponsorships · notion brand deals hub" title="Brand Deals" meta={loadError ? 'the board could not be read' : 'reading the Brand Deals Hub'} />
		{#if loadError}
			<div data-part="load-error" class="font-mono text-[12px]">
				<span style="color: var(--bn-err)">Could not load brand deals: {loadError}</span>
				<button type="button" class="bn-border bn-muted ml-3 border px-3 py-1 hover:text-[var(--bn-text)]" onclick={load}>Retry</button>
			</div>
		{:else}
			<div data-part="loading" class="bn-dim font-mono text-[12px]">Loading deals from Notion...</div>
		{/if}
	</Slab>
{/if}
