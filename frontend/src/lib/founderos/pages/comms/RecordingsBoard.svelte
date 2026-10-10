<!-- The Recordings tab (FounderOS v1 RecordingsBoard.tsx): Plaud (the recorder
     in the room) and Fathom (the notetaker on calls) newest first, each
     recorder's honest connector state on top. A recorder that failed to
     answer is named; its rows are missing, not "no recordings". -->
<script lang="ts">
	import { ExternalLink, Mic, Video } from '$lib/founderos/icons';
	import { Dot } from '$lib/founderos/kit';
	import { formatDuration, isPlaudSample, relativeAge, type ConnectorStatus, type Recording } from './model';

	let { recordings, sources, errors = [], nowMs }: { recordings: Recording[]; sources: ConnectorStatus[]; errors?: string[]; nowMs: number } = $props();
</script>

<div data-part="recordings" class="flex flex-col gap-4">
	<div class="grid gap-2.5 sm:grid-cols-2">
		{#each sources as s (s.id)}
			<div class="pc-surface flex items-start gap-2.5 rounded-[8px] px-3 py-2.5">
				<Dot state={s.state} />
				<div class="min-w-0">
					<div class="font-mono text-[11.5px] font-semibold">
						{s.name}
						<span class="bn-dim ml-2 text-[9.5px] uppercase tracking-[0.15em]">{s.id === 'plaud' ? 'in the room' : 'on calls'}</span>
					</div>
					<p class="bn-dim mt-0.5 truncate text-[10px] leading-snug" title={s.detail}>{s.detail}</p>
				</div>
			</div>
		{/each}
	</div>

	{#each errors as e (e)}
		<p class="font-mono text-[10.5px]" style="color: var(--bn-err)">{e}</p>
	{/each}

	{#if recordings.length === 0}
		<p class="pc-hair bn-dim rounded-[8px] border border-dashed px-3 py-3 font-mono text-[10.5px]">
			{#if errors.length > 0}
				No recordings could be read right now; see the recorder errors above.
			{:else}
				No recordings yet — connect Plaud (PLAUD_REFRESH_TOKEN) and Fathom (FATHOM_API_KEY) to list every recorded conversation here.
			{/if}
		</p>
	{:else}
		<ul class="pc-surface">
			{#each recordings as r (r.id)}
				{@const t = Date.parse(r.at)}
				{@const Icon = r.source === 'plaud' ? Mic : Video}
				<li data-part="recording" data-lens="r" class="bn-pressable is-row pc-hair flex items-center gap-3 border-b px-3 py-2.5 last:border-b-0">
					<Icon size={14} class="bn-accent shrink-0" />
					<div class="min-w-0 flex-1">
						<div class="truncate font-mono text-[11.5px] font-semibold">{r.title}</div>
						<div class="bn-dim mt-0.5 font-mono text-[9.5px] uppercase tracking-[0.15em]">
							{r.source} · {Number.isFinite(t) ? r.at.slice(0, 10) : 'undated'}{Number.isFinite(t) && relativeAge(nowMs - t) ? ` · ${relativeAge(nowMs - t)}` : ''}
						</div>
					</div>
					{#if r.source === 'plaud'}
						<span
							class="shrink-0 rounded-full border px-1.5 py-px font-mono text-[9px] uppercase tracking-[0.15em]"
							style="border-color: {r.brain ? 'var(--bn-ok)' : 'var(--bn-border)'}; color: {r.brain ? 'var(--bn-ok)' : 'var(--bn-text-3)'}"
							title={r.brain
								? `filed in the knowledge base via ${r.brain}`
								: isPlaudSample(r.title)
									? "one of Plaud's bundled sample recordings; never filed"
									: 'not in the knowledge base yet (waiting for Plaud to transcribe, or for the next ingest pass)'}
						>
							{r.brain ? 'in brain' : isPlaudSample(r.title) ? 'sample' : 'not filed'}
						</span>
					{/if}
					<span class="bn-muted shrink-0 font-mono text-[10.5px]">{formatDuration(r.durationMinutes)}</span>
					{#if r.url}
						<a href={r.url} target="_blank" rel="noreferrer" class="bn-dim shrink-0 hover:text-[color:var(--bn-accent)]" aria-label="open recording"><ExternalLink size={14} /></a>
					{/if}
				</li>
			{/each}
		</ul>
	{/if}
</div>
