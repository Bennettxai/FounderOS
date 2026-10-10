<!-- /os/adpilot: the paid-media deck (FounderOS v1 app/adpilot/page.tsx).
     Sanctioned theme exception: this page runs the RED accent, scoped to
     its own wrapper by overriding the --bn-accent* tokens there (nothing
     outside inherits it), over a black ground with a dot grid that fades
     toward the edges. Top half: the campaign deck (campaigns only from the
     ad-account sync). Bottom half: the Ad library on the Foreplay store. -->
<script lang="ts">
	import { onMount } from 'svelte';
	import { founderosFetch } from '$lib/founderos/api';
	import PageHeader from '$lib/founderos/kit/PageHeader.svelte';
	import SectionHead from '$lib/founderos/kit/SectionHead.svelte';
	import AdLibrary from './AdLibrary.svelte';
	import AdPilotDeck from './AdPilotDeck.svelte';
	import './adpilot.css';
	import type { AdpilotPayload } from './types';

	const RED_THEME = [
		'--bn-accent: #ff4557',
		'--ap-accent-2: #ff7582',
		'--bn-accent-ink: #2a060b',
		'--bn-accent-soft: rgba(255, 69, 87, 0.10)',
		'--bn-accent-line: rgba(255, 69, 87, 0.42)',
		'--bn-surface-2: #170b0f',
		'--bn-hairline: #2a141a'
	].join('; ');

	let st = $state<{ phase: 'loading' } | { phase: 'error'; message: string } | { phase: 'ready'; data: AdpilotPayload }>({ phase: 'loading' });
	let version = $state(0);

	async function load() {
		try {
			const data = await founderosFetch<AdpilotPayload>('/pages/adpilot');
			st = { phase: 'ready', data };
			version += 1;
		} catch (err) {
			st = { phase: 'error', message: err instanceof Error ? err.message : 'could not load AdPilot' };
		}
	}

	onMount(load);
</script>

<div class="relative" style={RED_THEME} data-adpilot>
	<div aria-hidden="true" class="pointer-events-none absolute -inset-x-6 -inset-y-6" style="background: #04070a"></div>
	<div
		aria-hidden="true"
		class="pointer-events-none absolute -inset-x-6 -inset-y-6"
		style="background-image: radial-gradient(rgba(160,172,168,0.12) 1px, transparent 1.4px); background-size: 26px 26px; mask-image: linear-gradient(90deg, transparent, #000 16%, #000 84%, transparent); -webkit-mask-image: linear-gradient(90deg, transparent, #000 16%, #000 84%, transparent)"
	></div>

	<div class="relative">
		<PageHeader eyebrow="Paid media" title="AdPilot" />

		{#if st.phase === 'loading'}
			<div class="ap-panel bn-dim px-5 py-10 text-center font-mono text-[12px]" data-part="loading">Loading the deck and the ad library...</div>
		{:else if st.phase === 'error'}
			<div class="ap-panel px-5 py-8 text-center" data-part="error">
				<p class="bn-text text-[13px]">AdPilot could not load.</p>
				<p class="bn-dim mt-1 font-mono text-[12px]">{st.message}</p>
				<button type="button" class="bn-pressable ap-btn mt-4 text-[12px]" onclick={() => ((st = { phase: 'loading' }), load())}>Retry</button>
			</div>
		{:else}
			{@const data = st.data}
			{#if data.errors.campaigns}
				<p class="mb-4 font-mono text-[11.5px] text-[#ff8790]" data-part="campaigns-error">Campaign file unreadable: {data.errors.campaigns}</p>
			{/if}
			<AdPilotDeck deck={data.deck} />

			<section class="mt-12">
				<SectionHead label="Ad intelligence" count={`${data.library.wall.length} tracked ads`} />
				{#if data.errors.store}
					<p class="mb-4 font-mono text-[11.5px] text-[#ff8790]" data-part="store-error">Ad store unreadable: {data.errors.store}</p>
				{/if}
				{#key version}
					<AdLibrary library={data.library} onRefreshed={load} />
				{/key}
			</section>
		{/if}
	</div>
</div>
