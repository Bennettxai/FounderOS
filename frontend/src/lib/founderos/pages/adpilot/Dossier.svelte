<!-- The ad dossier drawer: media, remake, save, track brand, copy hook,
     emotional drivers and a click-to-seek transcript. -->
<script lang="ts">
	import { safeHref } from '$lib/founderos/safe-href';
	import { goto } from '$app/navigation';
	import { Bookmark, ChevronRight, Copy, ExternalLink, Plus, X } from '$lib/founderos/icons';
	import { mmss } from './logic';
	import type { WallAd } from './types';

	let {
		ad,
		isSaved,
		onClose,
		onSave,
		onWatch
	}: { ad: WallAd; isSaved: boolean; onClose: () => void; onSave: (ad: WallAd) => void; onWatch: (ad: WallAd) => void } = $props();

	let video = $state<HTMLVideoElement | null>(null);
	let copied = $state(false);
	const topDrivers = $derived(Object.entries(ad.drivers).sort((a, b) => b[1] - a[1]).slice(0, 6));

	function remake() {
		const beats = ad.transcript.slice(0, 8).map((l) => l.s).join(' / ');
		const payload = {
			toolKey: 'video',
			title: `Remake: ${ad.brand}`,
			spec: {
				prompt: [
					`Remake this proven ad concept in the user's voice (source ran ${ad.daysRunning} days${ad.live ? ', still live' : ''}).`,
					ad.hook ? `Hook shape to model: "${ad.hook}"` : null,
					beats ? `Beat structure: ${beats}` : null
				]
					.filter(Boolean)
					.join('\n'),
				medias: ad.video ? [{ role: 'video_references', ref: ad.video, label: `${ad.brand} source ad` }] : []
			}
		};
		try {
			sessionStorage.setItem('founderos:remake', JSON.stringify(payload));
		} catch {
			// storage unavailable: the composer just opens blank
		}
		goto('/os/content?remake=1');
	}

	async function copyHook() {
		if (!ad.hook) return;
		try {
			await navigator.clipboard.writeText(ad.hook);
			copied = true;
			setTimeout(() => (copied = false), 1500);
		} catch {
			// clipboard blocked
		}
	}
</script>

<div class="fixed inset-0 z-50 flex justify-end bg-black/60" role="presentation" onclick={onClose}>
	<div
		class="ap-panel h-full w-full max-w-[460px] overflow-y-auto rounded-none border-y-0 border-r-0 p-5"
		role="dialog"
		aria-label="{ad.brand} ad dossier"
		tabindex="-1"
		onclick={(e) => e.stopPropagation()}
		onkeydown={(e) => e.key === 'Escape' && onClose()}
	>
		<div class="mb-3 flex items-start justify-between gap-3">
			<div>
				<div class="bn-text text-[15px]">{ad.brand}</div>
				<div class="bn-dim mt-0.5 text-[12px]">
					{ad.format ?? 'ad'} · {ad.daysRunning} days {ad.live ? '· live' : '· off'}{ad.ctaType ? ` · ${ad.ctaType.toLowerCase().replaceAll('_', ' ')}` : ''}
				</div>
			</div>
			<button type="button" onclick={onClose} aria-label="close" class="bn-pressable ap-btn p-1.5"><X class="h-4 w-4" strokeWidth={1.7} /></button>
		</div>

		{#if ad.video}
			<video bind:this={video} src={ad.video} controls playsinline class="w-full rounded-[10px] border border-[rgba(255,69,87,0.16)] bg-black"><track kind="captions" /></video>
		{:else if ad.thumbnail || ad.image}
			<img src={ad.thumbnail ?? ad.image ?? ''} alt="{ad.brand} creative" class="w-full rounded-[10px] border border-[rgba(255,69,87,0.16)]" />
		{/if}

		<div class="mt-3 flex flex-wrap gap-1.5">
			<button type="button" onclick={remake} class="bn-pressable ap-btn ap-btn-primary flex items-center gap-1.5 text-[12px]"><ChevronRight class="h-3.5 w-3.5" strokeWidth={1.7} /> Remake</button>
			<button type="button" onclick={() => onSave(ad)} class="bn-pressable ap-btn flex items-center gap-1.5 text-[12px]">
				<Bookmark class="h-3.5 w-3.5" strokeWidth={1.7} fill={isSaved ? 'currentColor' : 'none'} />
				{isSaved ? 'Saved' : 'Save'}
			</button>
			{#if ad.brandId}
				<button type="button" onclick={() => onWatch(ad)} class="bn-pressable ap-btn flex items-center gap-1.5 text-[12px]"><Plus class="h-3.5 w-3.5" strokeWidth={1.7} /> Track brand</button>
			{/if}
			{#if ad.hook}
				<button type="button" onclick={copyHook} class="bn-pressable ap-btn flex items-center gap-1.5 text-[12px]"><Copy class="h-3.5 w-3.5" strokeWidth={1.7} /> {copied ? 'Copied' : 'Copy hook'}</button>
			{/if}
			{#if safeHref(ad.linkUrl)}
				<a href={safeHref(ad.linkUrl)} target="_blank" rel="noreferrer" class="ap-btn flex items-center gap-1.5 text-[12px]"><ExternalLink class="h-3.5 w-3.5" strokeWidth={1.7} /> Landing page</a>
			{/if}
		</div>

		{#if ad.hook}
			<div class="mt-4">
				<div class="bn-dim text-[11px]">Hook ({ad.hookSource})</div>
				<p class="bn-text mt-1 text-[13.5px] leading-snug">“{ad.hook}”</p>
			</div>
		{/if}

		{#if topDrivers.length > 0}
			<div class="mt-4">
				<div class="bn-dim text-[11px]">Emotional drivers</div>
				<div class="mt-2 flex flex-col gap-1.5">
					{#each topDrivers as [axis, score] (axis)}
						<div class="flex items-center gap-2">
							<span class="bn-muted w-24 shrink-0 text-[12px]">{axis}</span>
							<div class="h-[7px] flex-1 overflow-hidden rounded-full bg-black/40">
								<div class="h-full rounded-full" style="width: {(score / 10) * 100}%; opacity: {0.4 + 0.06 * score}; background: var(--bn-accent)"></div>
							</div>
							<span class="bn-dim w-6 shrink-0 text-right text-[11.5px] tabular-nums">{score}</span>
						</div>
					{/each}
				</div>
			</div>
		{/if}

		{#if ad.transcript.length > 0}
			<div class="mt-4">
				<div class="bn-dim text-[11px]">Transcript: click a line to seek</div>
				<div class="mt-2 flex flex-col">
					{#each ad.transcript as line, i (i)}
						<button
							type="button"
							onclick={() => {
								if (video) {
									video.currentTime = line.t;
									video.play().catch(() => undefined);
								}
							}}
							class="bn-pressable group flex gap-2.5 rounded-[7px] px-2 py-1.5 text-left hover:bg-black/35"
						>
							<span class="bn-dim ap-group-accent shrink-0 text-[11px] tabular-nums">{mmss(line.t)}</span>
							<span class="bn-muted ap-group-text text-[12.5px] leading-snug">{line.s}</span>
						</button>
					{/each}
				</div>
			</div>
		{/if}
	</div>
</div>
