<!-- Ask Adscout: a question grounded in the synced store, answered by the
     model behind POST /pages/adscout/ask. Only a submit calls the model. -->
<script lang="ts">
	import { LoaderCircle, Send, X } from '$lib/founderos/icons';
	import { founderosFetch } from '$lib/founderos/api';
	import { relatedAds } from './logic';
	import type { WallAd } from './types';

	let { wall, onOpen }: { wall: WallAd[]; onOpen: (ad: WallAd) => void } = $props();

	let question = $state('');
	let phase = $state<'idle' | 'running' | 'done' | 'error'>('idle');
	let text = $state('');
	let asked = $state('');

	const examples = $derived(phase === 'done' && text ? relatedAds(`${asked} ${text}`, wall) : []);

	async function ask() {
		const q = question.trim();
		if (q.length < 3 || phase === 'running') return;
		phase = 'running';
		asked = q;
		try {
			const body = await founderosFetch<{ ok: boolean; answer: string }>('/pages/adscout/ask', { method: 'POST', json: { question: q } });
			text = body.answer;
			phase = 'done';
		} catch (err) {
			text = err instanceof Error ? err.message : 'ask failed';
			phase = 'error';
		}
	}
</script>

<div class="relative mx-auto max-w-3xl overflow-hidden rounded-[20px] border border-[rgba(255,120,132,0.28)]" data-part="ask">
	<div class="ap-orbit" aria-hidden="true"></div>
	<div class="absolute inset-0 bg-[#0a0608]/60 backdrop-blur-2xl" aria-hidden="true"></div>
	<div class="relative px-6 py-5">
		<div class="flex items-center gap-3">
			<span class="shrink-0 text-[14px] font-medium" style="color: var(--ap-accent-2)">Ask Adscout</span>
			<input
				bind:value={question}
				onkeydown={(e) => e.key === 'Enter' && ask()}
				placeholder="what's working in my niche right now?"
				aria-label="question for Adscout"
				class="bn-text ap-ph min-w-0 flex-1 bg-transparent text-[14.5px] outline-none"
			/>
			<button type="button" onclick={ask} disabled={phase === 'running'} aria-label="ask Adscout" class="bn-pressable ap-btn px-3 py-2">
				{#if phase === 'running'}<LoaderCircle class="ap-spin h-4 w-4" strokeWidth={1.7} />{:else}<Send class="h-4 w-4" strokeWidth={1.7} />{/if}
			</button>
		</div>
		{#if phase === 'running'}
			<p class="bn-muted mt-3 text-[13px]">Checking the library: hooks, longevity, drivers, signals…</p>
		{:else if phase === 'error'}
			<p class="mt-3 text-[13px] text-[#ff8790]" data-part="ask-error">{text}</p>
		{:else if phase === 'done'}
			<div class="mt-4 border-t border-[rgba(255,120,132,0.16)] pt-4">
				<div class="flex items-start gap-3">
					<p class="bn-text flex-1 whitespace-pre-wrap text-[13.5px] leading-relaxed" data-part="answer">{text}</p>
					<button type="button" onclick={() => (phase = 'idle')} aria-label="dismiss answer" class="bn-pressable bn-dim ap-hover-text shrink-0">
						<X class="h-4 w-4" strokeWidth={1.7} />
					</button>
				</div>
				{#if examples.length > 0}
					<div class="mt-4">
						<div class="bn-dim text-[12px]">From the library</div>
						<div class="mt-2 grid gap-2 sm:grid-cols-3">
							{#each examples as ad (ad.id)}
								<button type="button" onclick={() => onOpen(ad)} class="bn-pressable flex items-center gap-2.5 rounded-[10px] border border-[rgba(255,120,132,0.18)] bg-black/35 p-2 text-left hover:bg-black/55">
									{#if ad.thumbnail || ad.image}
										<img src={ad.thumbnail ?? ad.image ?? ''} alt="" class="h-11 w-14 shrink-0 rounded-[7px] object-cover" />
									{:else}
										<span class="bn-dim flex h-11 w-14 shrink-0 items-center justify-center rounded-[7px] bg-black/50 text-[10px]">{ad.format ?? 'ad'}</span>
									{/if}
									<span class="min-w-0">
										<span class="bn-text block truncate text-[12px]">{ad.brand}</span>
										<span class="bn-dim block text-[11px] tabular-nums">{ad.daysRunning}d {ad.live ? 'live' : 'off'}</span>
									</span>
								</button>
							{/each}
						</div>
					</div>
				{/if}
			</div>
		{/if}
	</div>
</div>
