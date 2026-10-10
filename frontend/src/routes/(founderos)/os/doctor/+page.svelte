<!-- /os/doctor (spec 6.19): FounderOS v1 app/doctor/page.tsx in the slab look,
     repurposed. Doctor was GBrain health; GBrain is retired on the bridge, so
     every panel now reports Optimal Engine health from
     GET /api/founderos/pages/doctor: the topology's engines, their workspaces,
     the store audit, claims vs facts, the memory agents' runs and the pillar
     radar. Rerun re-reads the engines (GETs only, nothing is written). -->
<script lang="ts">
	import { onMount } from 'svelte';
	import { doctorStatus } from '$lib/founderos/pages/doctor/status';
	import { founderosFetch } from '$lib/founderos/api';
	import { BigStat, Chip, Dot, InsightCard, MeterStack, PILL, Pressable, Slab, SlabCard, SlabTitle, StepLine, DotMatrix } from '$lib/founderos/kit';
	import DoctorChecks from '$lib/founderos/pages/doctor/DoctorChecks.svelte';
	import Arrow from '$lib/founderos/pages/doctor/Arrow.svelte';
	import DoctorCore from '$lib/founderos/pages/doctor/DoctorCore.svelte';
	import PillarRadar from '$lib/founderos/pages/doctor/PillarRadar.svelte';
	import { relativeTime } from '$lib/founderos/pages/doctor/radar';
	import type { DoctorBody, StorageLayer } from '$lib/founderos/pages/doctor/types';

	let data = $state<DoctorBody | null>(null);
	let failure = $state<string | null>(null);
	let busy = $state(true);

	async function load() {
		busy = true;
		try {
			data = await founderosFetch<DoctorBody>('/pages/doctor');
			failure = null;
		} catch (e) {
			failure = e instanceof Error ? e.message : 'unreachable';
		} finally {
			busy = false;
		}
	}
	onMount(load);

	// Production's DoctorRerun (AsyncButton, primary): the label says what it
	// runs, busy is a spinner + "checking" + elapsed, done is the score for a
	// beat. Here the run is the engine doctor: /api/health + /api/stores/audit
	// on every engine, GETs only.
	let phase = $state<'idle' | 'busy' | 'done'>('idle');
	let since = $state(0);
	let tick = $state(0);
	async function rerun() {
		if (phase !== 'idle') return;
		phase = 'busy';
		since = Date.now();
		const timer = setInterval(() => (tick += 1), 1000);
		try {
			await load();
		} finally {
			clearInterval(timer);
			phase = 'done';
			setTimeout(() => (phase = 'idle'), 1400);
		}
	}
	const elapsed = $derived.by(() => {
		void tick; // re-read each second while busy
		return Math.max(0, Math.floor((Date.now() - since) / 1000));
	});

	const v = $derived(data?.volume ?? null);
	const connected = $derived((data?.engines ?? []).some((e) => e.reachable));
	const status = $derived(doctorStatus(v?.counts ?? { ok: 0, warn: 0, fail: 0, total: 0 }, connected));
	const warnings = $derived(status.flagged);
	const layersLive = $derived((data?.layers ?? []).filter((l) => l.state === 'connected').length);
	const statusTone = $derived(status.tone);
	const statusText = $derived(status.text);
	const lastRun = $derived(data?.brain.last ? `last run ${relativeTime(data.brain.last.finishedAt)} · ${data.brain.last.agentId}` : 'no agent runs yet');
	const unreachableDetail = $derived(
		(data?.engines ?? [])
			.filter((e) => !e.reachable)
			.map((e) => `${e.name}: ${e.error ?? 'unreachable'}`)
			.join(' · ')
	);
	// v1: "<store> · N pages · last run 8d ago · data-agent"; the engines are the store now
	const meta = $derived(
		data
			? `${data.engines.map((e) => e.name).join(' + ') || 'no engines'} · ${data.volume.contexts ?? 'unknown'} pages · ${lastRun}`
			: 'reading the Optimal Engine topology…'
	);

	function pillStyle(state: StorageLayer['state']): string {
		const hue = state === 'connected' ? 'var(--bn-ok)' : state === 'error' ? 'var(--bn-err)' : 'var(--bn-warn)';
		return `background: color-mix(in oklab, ${hue} 16%, transparent); color: ${hue}`;
	}

	// the engine verbs, where prod lists the gbrain CLI's
	const COMMANDS = ['capture', 'retrieve', 'search', 'rerank', 'audit', 'health', 'stores', 'metrics'];
	// prod's DotMatrix carries seven short folder names; the table names run
	// longer, so six columns keep the matrix inside the card
	const storeCols = $derived((v?.store.cols ?? []).slice(0, 6));
	// prod's folder bars, on the engines' knowledge tables (alphabetical, like prod's folders)
	const tableBars = $derived([...(v?.store.cols ?? [])].sort((a, b) => a.label.localeCompare(b.label)));
	const maxBar = $derived(Math.max(1, ...tableBars.map((c) => c.count)));
</script>

{#snippet flow(title: string, detail: string)}
	<div class="flex-1 rounded-[var(--bn-r-panel)] border px-3 py-2.5" style="border-color: var(--bn-border); background: var(--bn-surface-2)">
		<div class="bn-text text-xs font-semibold">{title}</div>
		<div class="bn-dim mt-0.5 text-[11px] leading-relaxed">{detail}</div>
	</div>
{/snippet}

<Slab>
	<SlabTitle eyebrow="engine health" title="Doctor" {meta}>
		{#snippet right()}
			{#if data}
				<Chip tone={statusTone}>{statusText}</Chip>
			{/if}
			<a href="/os/brain" class={PILL}>Brain</a>
			<Pressable tone="primary" onclick={rerun} aria-busy={phase === 'busy'} disabled={phase !== 'idle'}>
				{#if phase === 'busy'}
					<span class="bn-doctor-spin inline-block h-[10px] w-[10px] rounded-full border-[1.5px]" style="border-color: var(--bn-accent-ink); border-right-color: transparent"></span>
					checking <span class="tabular-nums">{elapsed}s</span>
				{:else if phase === 'done'}
					✓ {data?.score != null ? `${data.score}/100` : 'checked'}
				{:else}
					▸ engine doctor
				{/if}
			</Pressable>
		{/snippet}
	</SlabTitle>

	{#if failure && !data}
		<SlabCard i={1} title="Doctor">
			<div class="px-6 pb-6 pt-3">
				<BigStat value={null} caption={`doctor unreachable: ${failure}`} />
			</div>
		</SlabCard>
	{:else if !data || !v}
		<SlabCard i={1} title="Doctor">
			<div class="px-6 pb-6 pt-3"><BigStat display="…" caption="reading the engines" /></div>
		</SlabCard>
	{:else}
		{#if failure}
			<div class="mb-4 font-mono text-[11px]" style="color: var(--bn-err)">rerun failed: {failure} · showing the previous read</div>
		{/if}

		<!-- Hero row: the pillar radar beside the Health Volume card -->
		<div class="grid grid-cols-[2fr_1fr] gap-6 max-[1200px]:grid-cols-1">
			<SlabCard i={1} title="Pillar Health" sub="live roster + runs + SOP coverage" class="flex flex-col overflow-hidden">
				<div aria-label="pillar health radar" class="flex min-h-[440px] flex-1 flex-col">
					{#if data.axes && data.axes.length > 0}
						<PillarRadar axes={data.axes} health={data.score} {warnings} />
					{:else}
						<div class="bn-dim grid flex-1 place-items-center px-6 font-mono text-[11px]">
							{data.relational.ok ? 'no departments seeded yet' : `pillars unknown · ${data.relational.error}`}
						</div>
					{/if}
				</div>
			</SlabCard>

			<SlabCard i={2} title="Health Volume" sub="out of 100" class="flex flex-col">
				<div class="flex flex-1 flex-col px-6 pb-6 pt-3">
					<BigStat value={v.headline} display={v.headline == null ? '—' : undefined} unit="/100" chips={v.chips} caption={v.caption} />
					<MeterStack meters={v.meters} foot={v.foot} />
				</div>
			</SlabCard>
		</div>

		<!-- Second row: memory runs, the store's shape, the one gradient card -->
		<div class="mt-6 grid grid-cols-3 gap-6 max-[1200px]:grid-cols-1">
			<SlabCard i={3} title="Brain Runs" sub={`${data.brain.agents.join(' + ')} · last ${data.windowDays} days`} class="pb-2">
				<div class="px-6 pb-2 pt-3">
					{#if data.relational.ok}
						<BigStat
							size={30}
							value={v.runsInWindow}
							chips={v.failedInWindow > 0 ? [{ tone: 'err', text: `${v.failedInWindow} failed` }] : []}
							caption={lastRun}
						/>
					{:else}
						<BigStat size={30} value={null} caption={`runs unknown · ${data.relational.error}`} />
					{/if}
				</div>
				{#if data.relational.ok}
					<StepLine series={v.series} hue="color-mix(in oklab, var(--bn-brain-2) 70%, var(--bn-text))" unit=" runs" empty={`No ${data.brain.agents.join(' + ')} runs in the last ${data.windowDays} days.`} />
				{/if}
			</SlabCard>

			<SlabCard i={4} title="Brain Store" sub={`${v.store.tables ?? v.store.cols.length} tables`}>
				<div class="flex flex-col gap-5 px-6 pb-6 pt-3">
					<BigStat
						size={30}
						value={v.store.total}
						unit="pages"
						caption={v.store.top ? `largest · ${v.store.top.name} · ${v.store.top.count.toLocaleString('en-US')}` : 'no engine reported its stores'}
					/>
					{#if storeCols.length > 0}
						<DotMatrix cols={storeCols} hue="var(--bn-brain-2)" />
					{/if}
				</div>
			</SlabCard>

			<InsightCard
				i={5}
				badge="Needs you"
				value={v.insight.value}
				display={v.insight.value == null ? '—' : undefined}
				headline={v.insight.headline}
				body={v.insight.body}
				frac={v.insight.frac}
			/>
		</div>

		<!-- Third row: the engine core beside the storage layers -->
		<div class="mt-6 grid grid-cols-1 gap-6 lg:grid-cols-2">
			<SlabCard i={6} title="Doctor Core" sub={connected ? (status.tone === 'err' ? 'failing' : warnings > 0 ? 'warnings' : 'ok') : 'unreachable'} class="flex flex-col overflow-hidden">
				<div class="dc-stage relative mt-4 flex min-h-[440px] flex-1 flex-col">
					<div class="bn-dim flex items-start justify-between px-6 pt-3.5 font-mono text-[10px] leading-normal">
						<span>{lastRun}</span>
						<div class="flex flex-col gap-1 text-right">
							<span><b class="bn-muted font-medium">hybrid search</b> {connected ? 'verified' : 'unreachable'}</span>
							<span>{data.engines.filter((e) => e.reachable).reduce((n, e) => n + (e.searches ?? 0), 0)} searches since boot</span>
						</div>
					</div>
					<div class="grid flex-1 place-items-center">
						<div class="w-full max-w-[540px]">
							<DoctorCore
								engines={data.engines}
								health={data.score}
								checks={data.checks}
								{connected}
								detail={unreachableDetail || 'no engine in the topology has an endpoint'}
							/>
						</div>
					</div>
					{#if unreachableDetail}
						<div class="px-6 pb-4 font-mono text-[10.5px]" style="color: var(--bn-err)">{unreachableDetail}</div>
					{/if}
				</div>
			</SlabCard>

			<SlabCard i={7} title="Storage layers" sub={`${layersLive}/${data.layers.length} live`} class="flex flex-col">
				<div class="mt-4 flex flex-1 flex-col border-t" style="border-color: var(--bn-border)">
					{#each data.layers as layer (layer.name)}
						<div data-lens="r" class="bn-pressable is-row flex flex-1 items-center gap-4 border-b px-6 py-4 last:border-0" style="border-color: var(--bn-border)">
							<Dot state={layer.state} pulse={layer.state === 'connected'} />
							<div class="min-w-0 flex-1">
								<div class="bn-text truncate text-[13.5px] font-medium">{layer.name}</div>
								<div class="bn-dim truncate font-mono text-[11px]" title={layer.sub}>{layer.sub}</div>
							</div>
							<span class="shrink-0 rounded-full px-2.5 py-0.5 font-mono text-[10.5px] uppercase tracking-[0.12em]" style={pillStyle(layer.state)}>{layer.val}</span>
						</div>
					{/each}
				</div>
				<div class="flex items-center justify-between border-t px-6 py-3.5 font-mono text-[11px]" style="border-color: var(--bn-border)">
					<span class="bn-dim"><b class="bn-muted font-medium">doctor</b> · health {data.score ?? '—'}/100</span>
					<span style="color: var(--bn-{status.tone})">
						{connected ? status.text : 'offline'}
					</span>
				</div>
			</SlabCard>
		</div>

		<!-- The pipeline: how knowledge becomes searchable in the engines -->
		<SlabCard i={8} title="Pipeline" sub={`${v.contexts ?? 'unknown'} pages across ${v.enginesUp} engine${v.enginesUp === 1 ? '' : 's'}`} class="mt-6">
			<div class="flex flex-col gap-2 px-6 pb-6 pt-4 xl:flex-row xl:items-stretch">
				<section class="min-w-0 flex-1 basis-0 rounded-[var(--bn-r-panel)] border p-5" style="border-color: var(--bn-border); background: var(--bn-surface-2)">
					<div class="flex items-center gap-3">
						<span class="grid h-7 w-7 shrink-0 place-items-center rounded-[var(--bn-r-ctl)] font-mono text-xs font-bold" style="background: var(--bn-accent); color: var(--bn-accent-ink)">1</span>
						<div>
							<h3 class="bn-text text-[14px] font-semibold">Intake</h3>
							<div class="bn-dim font-mono text-[10.5px]">memory.Capture → the workspace’s home engine</div>
						</div>
					</div>
					<div class="bn-muted mt-4 text-xs">
						Agents, pages and the Plaud ingest write through <code class="font-mono text-[11px]">memory.Router</code>, which sends each capture to its
						workspace’s one home engine. Personal and device workspaces never reach the hub.
					</div>
					<ul class="mt-3 space-y-1.5">
						{#each data.engines as e (e.name)}
							<li class="flex items-center gap-2 font-mono text-[11px]">
								<Dot state={e.reachable ? 'connected' : 'error'} />
								<span class="bn-muted w-20 shrink-0">{e.name}</span>
								<span class="bn-dim truncate">{e.reachable ? (e.workspaces ?? []).join(', ') || 'no workspaces' : 'unreachable'}</span>
							</li>
						{/each}
					</ul>
					{#if tableBars.length > 0}
						<ul class="mt-3 space-y-1.5" aria-label="knowledge tables">
							{#each tableBars as t (t.label)}
								<li class="flex items-center gap-2">
									<span class="bn-muted w-24 shrink-0 truncate font-mono text-[11px]">{t.label}</span>
									<span
										class="h-2 rounded-full"
										style="background: var(--bn-accent); width: {Math.max(6, (t.count / maxBar) * 100)}%; opacity: {0.25 + 0.55 * (t.count / maxBar)}"
									></span>
									<span class="bn-dim font-mono text-[11px]">{t.count.toLocaleString('en-US')}</span>
								</li>
							{/each}
						</ul>
					{/if}
				</section>

				<Arrow label="ingest · route" />

				<section class="min-w-0 flex-1 basis-0 rounded-[var(--bn-r-panel)] border p-5" style="border-color: var(--bn-border); background: var(--bn-surface-2)">
					<div class="flex items-center gap-3">
						<span class="grid h-7 w-7 shrink-0 place-items-center rounded-[var(--bn-r-ctl)] font-mono text-xs font-bold" style="background: var(--bn-accent); color: var(--bn-accent-ink)">2</span>
						<div>
							<h3 class="bn-text text-[14px] font-semibold">Store audit</h3>
							<div class="bn-dim font-mono text-[10.5px]">GET /api/health + /api/stores/audit, every engine</div>
						</div>
					</div>
					<div class="mt-4">
						<DoctorChecks checks={data.checks} healthScore={data.score} {connected} detail={unreachableDetail || 'no engine in the topology has an endpoint'} {busy} />
						<div class="mt-3 flex flex-wrap gap-1.5">
							{#each COMMANDS as cmd (cmd)}
								<span data-lens="c" class="bn-muted rounded-full border px-2.5 py-0.5 font-mono text-[10.5px]" style="border-color: var(--bn-border)">{cmd}</span>
							{/each}
						</div>
					</div>
				</section>

				<Arrow label="extract · promote" />

				<section class="min-w-0 flex-1 basis-0 rounded-[var(--bn-r-panel)] border p-5" style="border-color: var(--bn-border); background: var(--bn-surface-2)">
					<div class="flex items-center gap-3">
						<span class="grid h-7 w-7 shrink-0 place-items-center rounded-[var(--bn-r-ctl)] font-mono text-xs font-bold" style="background: var(--bn-accent); color: var(--bn-accent-ink)">3</span>
						<div>
							<h3 class="bn-text text-[14px] font-semibold">Claims → facts</h3>
							<div class="bn-dim font-mono text-[10.5px]">contexts, claims, facts, FTS + vectors</div>
						</div>
					</div>
					<div class="mt-4 grid grid-cols-2 gap-2">
						<div class="rounded-[var(--bn-r-panel)] border px-3 py-2.5" style="border-color: var(--bn-border); background: var(--bn-surface)">
							<div class="bn-text text-[30px] font-semibold leading-none tabular-nums">{v.claims ?? '—'}</div>
							<div class="bn-dim mt-1.5 font-mono text-[10px] uppercase tracking-wider">claims · rule-extracted</div>
						</div>
						<div class="rounded-[var(--bn-r-panel)] border px-3 py-2.5" style="border-color: var(--bn-border); background: var(--bn-surface)">
							<div class="bn-text text-[30px] font-semibold leading-none tabular-nums">{v.facts ?? '—'}</div>
							<div class="bn-dim mt-1.5 font-mono text-[10px] uppercase tracking-wider">facts · promoted</div>
						</div>
					</div>
					<div class="bn-muted mt-3 space-y-1.5 text-[11px] leading-relaxed">
						<p>
							Each capture becomes a context; claims are extracted from it by rule. A claim becomes a fact only once it is
							promoted, so facts trailing claims is the engine being careful, not losing data.
						</p>
						<p class="bn-dim">Contexts are indexed twice: FTS5 for exact text and chunk embeddings for cosine search.</p>
					</div>
				</section>
			</div>
		</SlabCard>

		<!-- How a query resolves -->
		<SlabCard i={9} title="Query path" sub="federated retrieval, honest failure" class="mt-6">
			<div class="px-6 pb-6 pt-3">
				<p class="bn-dim mb-4 text-[13px]">
					What happens when a page or an agent calls <code class="font-mono">memory.Retrieve</code>: every workspace’s home engine
					answers, and the pool is reranked.
				</p>
				<div class="flex flex-col gap-2 lg:flex-row lg:items-stretch">
					{@render flow('Question', 'Natural-language query from you, a page or an agent run.')}
					<Arrow label="route" />
					{@render flow('Fan out', 'memory.Router asks each workspace’s home engine (engine topology).')}
					<Arrow label="fan out" />
					<div class="flex flex-1 flex-col gap-2">
						{@render flow('Keyword search', 'SQLite FTS5 exact-text match over context text, per engine.')}
						{@render flow('Vector search', 'Cosine nearest-neighbour over chunk embeddings, per engine.')}
					</div>
					<Arrow label="merge" />
					{@render flow('Pool + rerank', 'Hits from every engine pool to 15; a cross-encoder reranks them.')}
					<Arrow label="answer" />
					{@render flow('Ranked snippets', 'The top 3 snippets, returned to the caller.')}
				</div>
				<div class="mt-2 rounded-[var(--bn-r-panel)] border border-dashed px-3 py-2.5" style="border-color: var(--bn-border)">
					<div class="bn-text text-xs font-semibold">When an engine is down</div>
					<div class="bn-dim mt-0.5 text-[11px] leading-relaxed">
						Its workspaces report unreachable instead of returning nothing, and no reranker means engine order, never a page error.
					</div>
				</div>
			</div>
		</SlabCard>
	{/if}
</Slab>

<style>
	.dc-stage {
		background:
			radial-gradient(circle at 50% 45%, color-mix(in oklab, var(--bn-brain-1) 10%, transparent), transparent 65%),
			var(--bn-surface);
	}
	.bn-doctor-spin {
		animation: bn-doctor-spin 0.8s linear infinite;
	}
	@keyframes bn-doctor-spin {
		to {
			transform: rotate(360deg);
		}
	}
</style>
