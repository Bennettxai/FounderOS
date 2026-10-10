<!-- /os/skills (spec 6.19), FounderOS v1 app/skills/page.tsx (2026-09-24, the
     Brand Deals slab). Skills carry no dates, so the hero shows the catalog's
     shape (skills per source group) rather than a timeline it would invent.
     Every number comes from the backend's skills volume over the same rows the
     wall renders: the Claude Code skills on this host's ~/.claude (user +
     plugins; none when the host has none) and the operator skills table. -->
<script lang="ts">
	import { onMount } from 'svelte';
	import { founderosFetch } from '$lib/founderos/api';
	import { BigStat, Chip, DotMatrix, InsightCard, MeterStack, PILL, Slab, SlabCard, SlabTitle } from '$lib/founderos/kit';
	import SkillsGrid from '$lib/founderos/pages/skills/SkillsGrid.svelte';
	import type { SkillsBody } from '$lib/founderos/pages/skills/types';

	// FounderOS v1 globals.css --send-activity / --ramp-1 / --ramp-4
	const SEND_ACTIVITY = 'color-mix(in oklab, var(--bn-brain-2) 70%, var(--bn-text))';
	const RAMP_1 = 'var(--bn-brain-2)';
	const RAMP_4 = 'color-mix(in oklab, var(--bn-brain-1) 70%, var(--bn-accent))';

	let data = $state<SkillsBody | null>(null);
	let failure = $state<string | null>(null);

	onMount(() => {
		founderosFetch<SkillsBody>('/pages/skills')
			.then((b) => (data = b))
			.catch((e: unknown) => (failure = e instanceof Error ? e.message : 'unreachable'));
	});

	const v = $derived(data?.volume);
	const opKnown = $derived(data != null && data.operatorError == null);
	const meta = $derived(
		v
			? `${v.counts.claude} Claude Code skills on disk · ${opKnown ? `${v.counts.operator} operator skills` : 'operator skills unavailable'} · ${v.groupCount} groups`
			: failure
				? 'skills unreachable'
				: 'reading the catalog…'
	);
	const topCategory = $derived(v?.categories[0]);
</script>

<Slab>
	<SlabTitle eyebrow="capability library" title="Skills" {meta}>
		{#snippet right()}
			{#if v}
				<Chip tone={v.insight.value > 0 ? 'warn' : 'ok'}>{v.counts.live} live · {v.insight.value} drafts</Chip>
			{/if}
			<a href="/os/agents" data-lens="c" class={PILL}>Agents</a>
		{/snippet}
	</SlabTitle>

	{#if data && v}
		{#if data.operatorError}
			<p class="mb-6 font-mono text-[11px]" style="color: var(--bn-err)">{data.operatorError}</p>
		{/if}

		<!-- Hero row, Brand Deals' shape: the catalog's shape + the volume card -->
		<div class="grid grid-cols-[2fr_1fr] gap-6 max-[1200px]:grid-cols-1">
			<SlabCard i={1} title="Skill Library" sub="skills per source group">
				<!-- stacked: group names are long column labels -->
				<div class="flex flex-col gap-6 px-6 pb-6 pt-3">
					<BigStat
						size={30}
						value={v.groupCount}
						unit="groups"
						chips={[{ tone: 'accent', text: `${v.counts.plugin} from plugins` }, { text: `${v.counts.user} user` }]}
						caption={v.groups.length > 0 ? `largest · ${v.groups[0].label} with ${v.groups[0].count}` : 'no skills on this machine yet'}
					/>
					{#if v.groups.length > 0}<DotMatrix cols={v.groups} hue={SEND_ACTIVITY} />{/if}
				</div>
			</SlabCard>

			<SlabCard i={2} title="Skill Volume" class="flex flex-col">
				<div class="flex flex-1 flex-col px-6 pb-6 pt-3">
					<BigStat value={v.headline} chips={v.chips} caption={v.caption} />
					<MeterStack meters={v.meters} foot={v.foot} empty="no skills yet" />
				</div>
			</SlabCard>
		</div>

		<!-- Second row: operator categories, owners, THE gradient card -->
		<div class="mt-6 grid grid-cols-3 gap-6 max-[1200px]:grid-cols-1">
			<SlabCard i={3} title="Categories" sub="operator skills">
				<div class="flex flex-col gap-5 px-6 pb-6 pt-3">
					<BigStat
						size={30}
						value={opKnown ? v.categories.length : null}
						caption={!opKnown ? 'operator skills unavailable' : topCategory ? `most in ${topCategory.label} · ${topCategory.count}` : 'no operator skills yet'}
					/>
					{#if v.categories.length > 0}<DotMatrix cols={v.categories} hue={RAMP_1} />{/if}
				</div>
			</SlabCard>

			<SlabCard i={4} title="Owners" sub="agents wielding them">
				<div class="flex flex-col gap-5 px-6 pb-6 pt-3">
					<BigStat
						size={30}
						value={opKnown ? v.owners.length : null}
						chips={v.unassigned > 0 ? [{ tone: 'warn', text: `${v.unassigned} unassigned` }] : []}
						caption={!opKnown
							? 'operator skills unavailable'
							: v.owners.length > 0
								? `${v.owners[0].label} holds the most · ${v.owners[0].count}`
								: 'no operator skill has an owner yet'}
					/>
					{#if v.owners.length > 0}<DotMatrix cols={v.owners} hue={RAMP_4} />{/if}
				</div>
			</SlabCard>

			<InsightCard i={5} badge="Drafts" value={v.insight.value} headline={v.insight.headline} body={v.insight.body} frac={v.insight.frac} />
		</div>

		<SkillsGrid i={6} cards={data.cards} sourceNote={data.sourceNote} />
	{:else if failure}
		<p class="font-mono text-[12px]" style="color: var(--bn-err)">Skills could not be read: {failure}</p>
	{:else}
		<p class="bn-dim animate-pulse font-mono text-[12px]">loading the skill catalog…</p>
	{/if}
</Slab>
