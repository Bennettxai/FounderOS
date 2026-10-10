<!-- The 9am report (FounderOS v1 components/CommsDigestPanel.tsx). Tiers are his
     stated priority: calls first, then clients, people, brand deals, group
     chats, companies last with an unsubscribe list. Rendered from the STORED
     digest; "run now" regenerates it (POST /pages/comms/digest). Ticks are
     optimistic and persist through /pages/comms/digest/read, keyed to the
     message, so tomorrow's rebuild does not resurrect a cleared row. -->
<script lang="ts">
	import { BellOff, Check, ChevronDown, ChevronRight, ExternalLink, Hash, Mail, MessageSquare, RefreshCw, Undo2 } from '$lib/founderos/icons';
	import { onMount, untrack } from 'svelte';
	import { founderosFetch } from '$lib/founderos/api';
	import { Label } from '$lib/founderos/kit';
	import { agoLong } from '$lib/founderos/pages/pc/ui';
	import { entryKey, replyTarget, TIER_ORDER, type CommsDigest, type DigestBody, type DigestEntry, type DigestSourceState, type DigestTier } from './model';

	let { initial, initialRead = [], nowMs = Date.now() }: { initial: DigestBody; initialRead?: string[]; nowMs?: number } = $props();

	const TIER_LABEL: Record<DigestTier, string> = {
		call: 'You have a call with them',
		client: 'Clients and deals',
		people: 'People · students, family, cohort',
		branddeal: 'Brand deals',
		group: 'Group chats',
		noise: 'Companies and software'
	};
	const TIER_TONE: Record<DigestTier, string> = {
		call: 'var(--bn-err)',
		client: 'var(--bn-warn)',
		people: 'var(--bn-ok)',
		branddeal: 'var(--bn-accent)',
		group: 'var(--bn-text-2)',
		noise: 'var(--bn-text-3)'
	};
	const ICON = { email: Mail, whatsapp: MessageSquare, slack: Hash } as Record<string, typeof Mail>;

	let digest = $state<CommsDigest | null>(untrack(() => initial.digest));
	let sources = $state<DigestSourceState[]>(untrack(() => initial.sources ?? []));
	let at = $state<string | null>(untrack(() => initial.generatedAt));
	let loadError = $state<string | null>(untrack(() => initial.error ?? null));
	let runError = $state<string | null>(null);
	let busy = $state(false);
	let open = $state(true);
	// Collapsed stays collapsed across reloads (prod 02f00b3: collapse the
	// report and the Inbox is on the first screen). Per-browser convenience
	// only; storage can throw in private windows, so it is best-effort.
	const OPEN_KEY = 'founderos-os:comms:morning-report-open';
	onMount(() => {
		try {
			if (window.localStorage.getItem(OPEN_KEY) === '0') open = false;
		} catch {
			/* default open */
		}
	});
	function toggleOpen() {
		open = !open;
		try {
			window.localStorage.setItem(OPEN_KEY, open ? '1' : '0');
		} catch {
			/* ignore */
		}
	}
	let showNoise = $state(false);
	let hideRead = $state(true);
	let read = $state<Set<string>>(untrack(() => new Set(initialRead)));
	let writeError = $state<string | null>(null);

	function toggle(key: string) {
		const next = new Set(read);
		const was = next.has(key);
		if (was) next.delete(key);
		else next.add(key);
		read = next;
		// Optimistic: the tick lands now and the write follows. A failed write
		// is said out loud (the row may come back tomorrow), never swallowed.
		founderosFetch('/pages/comms/digest/read', { method: was ? 'DELETE' : 'POST', json: { key } }).catch((e: unknown) => {
			writeError = `read state not saved: ${e instanceof Error ? e.message : 'unreachable'}`;
		});
	}

	function clearTier(rows: DigestEntry[]) {
		for (const r of rows) if (!read.has(entryKey(r))) toggle(entryKey(r));
	}

	async function run() {
		busy = true;
		runError = null;
		try {
			const b = await founderosFetch<DigestBody>('/pages/comms/digest', { method: 'POST' });
			digest = b.digest;
			sources = b.sources ?? [];
			at = b.generatedAt;
			loadError = null;
		} catch (e) {
			// leave the last report on screen and say the run failed
			runError = e instanceof Error ? e.message : 'run failed';
		} finally {
			busy = false;
		}
	}

	const tiers = $derived<DigestTier[]>(showNoise ? TIER_ORDER : TIER_ORDER.filter((t) => t !== 'noise'));
	const dead = $derived(sources.filter((s) => !s.ok));
	const left = $derived(digest ? digest.entries.filter((e) => e.tier !== 'noise' && !read.has(entryKey(e))).length : 0);
	const held = $derived(digest ? digest.entries.filter((e) => e.carried).length : 0);
</script>

<section data-part="digest" class="pc-surface rounded-2xl">
	<div class="flex flex-wrap items-center gap-x-3 gap-y-1 px-3 py-1.5 {open ? 'pc-hair border-b' : ''}">
		<button
			type="button"
			onclick={toggleOpen}
			class="bn-pressable bn-dim hover:text-[color:var(--bn-text)]"
			aria-expanded={open}
			aria-label={open ? 'Collapse morning report' : 'Expand morning report'}
			data-testid="morning-report-toggle"
		>
			{#if open}<ChevronDown size={14} />{:else}<ChevronRight size={14} />{/if}
		</button>
		<Label>Morning report</Label>
		{#if digest}
			<span data-part="digest-meta" class="bn-dim font-mono text-[10px]">
				{left} left to clear · {held ? `${digest.total - held} new · ${held} held over` : `${digest.total} in ${digest.windowHours}h`}{at ? ` · ${agoLong(at, nowMs)}` : ''}
			</span>
		{:else if loadError}
			<span class="font-mono text-[10px]" style="color: var(--bn-err)">report unreadable: {loadError}</span>
		{:else}
			<span class="bn-dim font-mono text-[10px]">no report yet — runs daily at 09:00</span>
		{/if}
		{#if dead.length > 0}
			<span class="font-mono text-[10px]" style="color: var(--bn-err)">{dead.map((d) => d.source).join(', ')} unavailable</span>
		{/if}
		{#if runError}
			<span class="font-mono text-[10px]" style="color: var(--bn-err)">run failed: {runError}</span>
		{/if}
		{#if digest}
			<button
				type="button"
				onclick={() => (hideRead = !hideRead)}
				class="bn-pressable bn-dim ml-auto flex shrink-0 items-center gap-1 font-mono text-[10px] uppercase tracking-[0.12em] hover:text-[color:var(--bn-text)]"
				title={hideRead ? 'Show what you already cleared' : 'Hide what you cleared'}
			>
				<Undo2 size={12} />
				{hideRead ? 'show cleared' : 'hide cleared'}
			</button>
		{/if}
		<button
			type="button"
			onclick={run}
			disabled={busy}
			class="bn-pressable bn-dim flex shrink-0 items-center gap-1 font-mono text-[10px] uppercase tracking-[0.12em] hover:text-[color:var(--bn-text)] disabled:opacity-40"
		>
			<RefreshCw size={12} class={busy ? 'animate-spin' : ''} />
			{busy ? 'scraping' : 'run now'}
		</button>
	</div>
	{#if writeError}
		<p class="px-4 pt-2 font-mono text-[10px]" style="color: var(--bn-warn)">{writeError}</p>
	{/if}

	{#if open}
		{#if digest === null || digest.total === 0}
			<p class="bn-dim px-4 py-3 font-mono text-[10.5px]">
				Nothing in the last 24 hours{digest ? '' : ' yet'}. The 09:00 job writes this report; hit run now to build it on demand.
			</p>
		{:else}
			<div class="p-2">
				<div class="mb-2 grid grid-cols-3 gap-2 sm:grid-cols-6">
					{#each TIER_ORDER as t (t)}
						<div data-part="tier-tile" class="pc-panel rounded-xl px-2 py-1">
							<div class="font-mono text-[14px] font-semibold leading-none" style="color: {TIER_TONE[t]}">{digest.counts[t] ?? 0}</div>
							<div class="bn-dim mt-1 truncate font-mono text-[8.5px] uppercase tracking-[0.1em]">{t}</div>
						</div>
					{/each}
				</div>

				<div class="space-y-2">
					{#each tiers as t (t)}
						{@const all = digest.entries.filter((e) => e.tier === t)}
						{@const rows = hideRead ? all.filter((e) => !read.has(entryKey(e))) : all}
						{@const remaining = all.filter((e) => !read.has(entryKey(e))).length}
						{#if all.length > 0}
							<div data-tier={t} class="pc-panel rounded-xl">
								<div class="pc-hair flex items-center gap-2 border-b px-3 py-1.5">
									<span data-part="tier-dot" class="h-1.5 w-1.5 shrink-0 rounded-full" style="background: {TIER_TONE[t]}"></span>
									<span class="bn-muted font-mono text-[9.5px] uppercase tracking-[0.16em]">{TIER_LABEL[t]}</span>
									<span class="bn-dim ml-auto font-mono text-[9.5px]">{remaining === 0 ? 'cleared' : `${remaining} left`}</span>
									{#if remaining > 0}
										<button type="button" onclick={() => clearTier(all)} title="Mark this whole group read" class="bn-pressable bn-dim shrink-0 font-mono text-[9px] uppercase tracking-[0.1em] hover:text-[color:var(--bn-ok)]">clear</button>
									{/if}
								</div>
								{#if rows.length === 0}
									<p class="bn-dim px-3 py-2 font-mono text-[10px]">all clear</p>
								{:else}
									{#each rows.slice(0, 12) as e, i (`${e.sender}-${e.ts}-${i}`)}
										{@const key = entryKey(e)}
										{@const isRead = read.has(key)}
										{@const target = replyTarget(e)}
										{@const Icon = ICON[e.source] ?? Mail}
										<div data-part="digest-row" class="pc-hair flex items-start gap-2 border-b px-3 py-1.5 transition-opacity last:border-b-0 {isRead ? 'opacity-40' : ''}">
											<button
												type="button"
												onclick={() => toggle(key)}
												title={isRead ? 'Mark unread' : 'Mark read'}
												aria-label={isRead ? `Mark ${e.sender} unread` : `Mark ${e.sender} read`}
												class="bn-pressable mt-0.5 flex h-4 w-4 shrink-0 items-center justify-center rounded-[6px] border"
												style="border-color: {isRead ? 'var(--bn-ok)' : 'var(--bn-border)'}; color: {isRead ? 'var(--bn-ok)' : 'transparent'}"
											>
												<Check size={10} />
											</button>
											<Icon size={12} class="bn-dim mt-0.5 shrink-0" />
											<div class="min-w-0 flex-1">
												<div class="flex items-baseline gap-2">
													<span class="truncate text-[12px] font-semibold {isRead ? 'line-through' : ''}">{e.sender}</span>
													<span class="bn-dim shrink-0 font-mono text-[9px]">{agoLong(e.ts, nowMs)}</span>
													{#if e.carried}
														<span
															class="shrink-0 rounded-md border px-1 font-mono text-[8.5px] uppercase tracking-[0.1em]"
															style="border-color: color-mix(in oklab, var(--bn-warn) 40%, transparent); color: var(--bn-warn)"
															title={`Still open since ${new Date(e.firstSeenAt ?? e.ts).toLocaleDateString()}`}
														>
															held {agoLong(e.firstSeenAt ?? e.ts, nowMs).replace(' ago', '')}
														</span>
													{/if}
												</div>
												<!-- title and preview share one line (condensed, prod 02f00b3) -->
												<div data-part="digest-line" class="bn-muted truncate text-[11px]">{e.title}{#if e.preview}<span class="bn-dim">{` · ${e.preview}`}</span>{/if}</div>
											</div>
											<div class="flex shrink-0 items-center gap-2">
												{#if target.href}
													<a
														href={target.href}
														target={target.kind === 'whatsapp' ? '_blank' : undefined}
														rel="noreferrer"
														data-lens="c"
														class="bn-pressable is-dark bn-border bn-dim flex items-center gap-1 rounded-full border px-1.5 py-0.5 font-mono text-[9px] uppercase tracking-[0.1em]"
													>
														{target.label}
														<ExternalLink size={10} />
													</a>
												{/if}
												<span class="bn-dim hidden font-mono text-[9px] uppercase tracking-[0.12em] sm:inline">{e.reason}</span>
											</div>
										</div>
									{/each}
								{/if}
							</div>
						{/if}
					{/each}
				</div>

				{#if digest.unsubscribes.length > 0}
					<div class="pc-panel mt-2 rounded-xl">
						<div class="pc-hair flex items-center gap-2 border-b px-3 py-1.5">
							<BellOff size={12} class="bn-dim shrink-0" />
							<span class="bn-muted font-mono text-[9.5px] uppercase tracking-[0.16em]">Unsubscribe candidates</span>
							<span class="bn-dim ml-auto font-mono text-[9.5px]">{digest.unsubscribes.length}</span>
						</div>
						<div class="flex flex-wrap gap-1.5 p-2.5">
							{#each digest.unsubscribes.slice(0, 24) as u (u.sender)}
								<span title={u.reason} class="pc-hair bn-dim rounded-lg border px-2 py-0.5 font-mono text-[9.5px]">
									{u.sender}
									{#if u.count > 1}<span style="color: var(--bn-warn)">×{u.count}</span>{/if}
								</span>
							{/each}
						</div>
					</div>
				{/if}

				<button type="button" onclick={() => (showNoise = !showNoise)} class="bn-pressable bn-dim mt-2 font-mono text-[9.5px] uppercase tracking-[0.12em] hover:text-[color:var(--bn-text)]">
					{showNoise ? 'hide' : 'show'} companies and software ({digest.counts.noise ?? 0})
				</button>
			</div>
		{/if}
	{/if}
</section>

