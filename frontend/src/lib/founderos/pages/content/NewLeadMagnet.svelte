<!-- Register a lead magnet from inside the OS (FounderOS v1 NewLeadMagnet).
     Posts to the page's own /pages/content/lead-magnets; the content-gen
     skill's POST /api/lead-magnets is the compat listener's job. -->
<script lang="ts">
	import { Plus, X } from '$lib/founderos/icons';
	import { founderosFetch } from '$lib/founderos/api';
	import { PILL, PILL_ACCENT } from '$lib/founderos/kit';
	import type { LeadMagnet } from './types';

	let { onCreated }: { onCreated?: () => void } = $props();

	const today = new Date().toISOString().slice(0, 10);
	const empty = () => ({
		name: '',
		url: '',
		offer: '',
		source: '',
		destination: 'Beehiiv · newsletter',
		captures: 'email',
		status: 'live',
		launchedAt: today,
		notes: ''
	});

	let open = $state(false);
	let busy = $state(false);
	let error = $state<string | null>(null);
	let done = $state<string | null>(null);
	let form = $state(empty());

	async function submit(e: SubmitEvent) {
		e.preventDefault();
		busy = true;
		error = null;
		try {
			const body = await founderosFetch<{ leadMagnet?: LeadMagnet }>('/pages/content/lead-magnets', { method: 'POST', json: { ...form } });
			done = body?.leadMagnet?.name ?? form.name;
			form = empty();
			open = false;
			onCreated?.();
		} catch (err) {
			error = err instanceof Error ? err.message : 'Check the name and the URL.';
		} finally {
			busy = false;
		}
	}

	const field = 'pc-input w-full rounded-[8px] px-2.5 py-2 text-[12px]';
	const label = 'bn-dim mb-1 block font-mono text-[9.5px] uppercase tracking-[0.16em]';
</script>

{#if !open}
	<div class="mb-4 flex items-center gap-3">
		<button type="button" onclick={() => ((done = null), (open = true))} data-lens="c" class="bn-pressable {PILL_ACCENT}">
			<Plus size={14} /> New lead magnet
		</button>
		{#if done}<span class="font-mono text-[11px]" style="color: var(--bn-ok)">Added {done} to the register.</span>{/if}
	</div>
{:else}
	<form onsubmit={submit} class="pc-panel mb-4 rounded-[10px] p-5">
		<div class="mb-3 flex items-center">
			<span class="bn-dim font-mono text-[10px] uppercase tracking-[0.16em]">Register a page</span>
			<button type="button" onclick={() => (open = false)} aria-label="Close" data-lens="c" class="bn-pressable lm-close bn-dim ml-auto"><X size={14} /></button>
		</div>
		<div class="grid gap-3 sm:grid-cols-2">
			<div>
				<label class={label} for="lm-name">Name</label>
				<input id="lm-name" class={field} bind:value={form.name} required placeholder="The Claude Trading Setup" />
			</div>
			<div>
				<label class={label} for="lm-url">Live URL</label>
				<input id="lm-url" class={field} bind:value={form.url} required type="url" placeholder="https://…" />
			</div>
			<div class="sm:col-span-2">
				<label class={label} for="lm-offer">What they get</label>
				<input id="lm-offer" class={field} bind:value={form.offer} placeholder="The setup, the prompt, and the guardrails" />
			</div>
			<div>
				<label class={label} for="lm-source">Campaign it was built for</label>
				<input id="lm-source" class={field} bind:value={form.source} placeholder="IG reel (comment TRADE)" />
			</div>
			<div>
				<label class={label} for="lm-dest">Where leads land</label>
				<input id="lm-dest" class={field} bind:value={form.destination} />
			</div>
			<div>
				<label class={label} for="lm-captures">Captures</label>
				<select id="lm-captures" class={field} bind:value={form.captures}>
					<option value="email">Email</option>
					<option value="booking">Booking</option>
					<option value="none">Nothing yet</option>
				</select>
			</div>
			<div>
				<label class={label} for="lm-launched">Went live</label>
				<input id="lm-launched" class={field} type="date" bind:value={form.launchedAt} />
			</div>
			<div>
				<label class={label} for="lm-status">Status</label>
				<select id="lm-status" class={field} bind:value={form.status}>
					<option value="live">Live</option>
					<option value="draft">Draft</option>
					<option value="paused">Paused</option>
					<option value="archived">Archived</option>
				</select>
			</div>
			<div class="sm:col-span-2">
				<label class={label} for="lm-notes">Notes</label>
				<textarea
					id="lm-notes"
					rows="2"
					class="{field} resize-y"
					bind:value={form.notes}
					placeholder="Anything future-you needs to know: what is gated, what still needs wiring, which automation feeds it"
				></textarea>
			</div>
		</div>
		{#if error}<p class="mt-2 font-mono text-[10.5px]" style="color: var(--bn-err)">{error}</p>{/if}
		<button type="submit" disabled={busy} data-lens="c" class="{PILL} mt-4 disabled:opacity-40">{busy ? 'saving…' : 'add to the register'}</button>
	</form>
{/if}

<style>
	.lm-close:hover {
		color: var(--bn-text);
	}
</style>
