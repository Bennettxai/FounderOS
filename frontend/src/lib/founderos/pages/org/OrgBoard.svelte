<!-- The /org hierarchy board, 1:1 in structure with FounderOS v1 app/org/page.tsx
     (markup frozen there): board-live strip → venture switcher → focus panel →
     life-area legend → operator → AI Head row → department rail of crews. -->
<script lang="ts">
	import { page } from '$app/state';
	import { Users } from '$lib/founderos/icons';
	import { operatorName } from '$lib/founderos/operator';
	import OrgDot from './OrgDot.svelte';
	import AgentNodePill from './AgentNodePill.svelte';
	import ConductorCard from './ConductorCard.svelte';
	import LiveChip from './LiveChip.svelte';
	import SparkIcon from '$lib/founderos/kit/SparkIcon.svelte';
	import VentureDots from './VentureDots.svelte';
	import { dimFor, findVenture, liveDot, rosterDot } from './org';
	import type { OrgView } from './types';

	let { view, ventureId }: { view: OrgView; ventureId: string | null } = $props();
	const operator = $derived(operatorName(page.data?.user as { name?: unknown; email?: unknown } | undefined));

	const venture = $derived(findVenture(view.ventures, ventureId));
	const showStrip = $derived(view.live.connected && (view.live.conductor !== null || view.live.extras.length > 0));
	const rail = 'var(--bn-border-strong)';
</script>

{#snippet systemCard(href: string, title: string, caption: string)}
	<a href={href} data-part="system-card" data-lens="r" class="bn-pressable is-row bn-surface bn-border block w-44 rounded-[10px] border p-3 text-center">
		<div class="flex justify-center">
			<div class="flex h-10 w-10 items-center justify-center rounded-[8px] border" style="border-color: {rail}; background: var(--bn-bg)">
				<SparkIcon size={20} shade="#a3a3a3" />
			</div>
		</div>
		<div class="bn-text mt-1.5 text-xs font-bold">{title}</div>
		<div class="bn-dim text-[10px] leading-snug">{caption}</div>
	</a>
{/snippet}

<div data-part="org-board">
	<!-- Live board strip: only when the board answered -->
	{#if showStrip}
		<div class="bn-rise bn-surface bn-border mb-3 flex flex-wrap items-center gap-2 rounded-[10px] border px-3 py-2" style="--rise-i: 1">
			<span class="bn-muted flex items-center gap-1.5 text-[9px] uppercase tracking-[0.2em]">
				<OrgDot look={{ fill: 'var(--bn-ok)', pulse: true }} />
				Board live
			</span>
			{#each view.live.extras as extra (extra.id)}
				<span class="bn-muted flex items-center gap-1.5 font-mono text-[10px]">
					<OrgDot look={liveDot(extra.status)} />
					{extra.name}
				</span>
			{/each}
		</div>
	{/if}
	<!-- Board down = inert, exactly like prod: no strip, no chips, no banner. -->

	<!-- Venture switcher -->
	<div class="bn-rise mb-3 flex flex-wrap items-center gap-2" style="--rise-i: 2">
		<a
			href="/os/org"
			data-lens="c"
			class="bn-pressable rounded-[6px] border px-3 py-1.5 text-xs font-semibold"
			style={!venture
				? 'border-color: var(--bn-text); background: var(--bn-text); color: var(--bn-bg)'
				: 'border-color: var(--bn-border); background: var(--bn-surface); color: var(--bn-text-2)'}
		>
			All ventures
		</a>
		{#each view.ventures as v (v.id)}
			{@const active = venture?.id === v.id}
			<a
				href={`/os/org?venture=${v.id}`}
				data-lens="c"
				class="bn-pressable flex items-center gap-1.5 rounded-[6px] border px-3 py-1.5 text-xs font-semibold"
				style={active
					? `background: ${v.color}; border-color: ${v.color}; color: #000`
					: 'border-color: var(--bn-border); background: var(--bn-surface); color: var(--bn-text-2)'}
			>
				<span class="h-1.5 w-1.5 rounded-full" style="background: {active ? '#000000' : v.color}"></span>
				{v.label}
			</a>
		{/each}
		{#if venture}<span class="bn-dim text-[11px]">{venture.kind} · {venture.detail}</span>{/if}
	</div>

	{#if venture}
		<div
			class="bn-rise bn-surface mb-4 rounded-[10px] border px-4 py-3"
			style="--rise-i: 3; border-color: {venture.color}66; box-shadow: inset 3px 0 0 {venture.color}"
		>
			<div class="text-[9px] uppercase tracking-[0.2em]" style="color: {venture.color}">{venture.label} — executive focus</div>
			<ul class="mt-1.5 space-y-1">
				{#each venture.focus as f (f)}
					<li class="bn-muted flex items-start gap-2 text-[11px]">
						<span class="mt-1.5 h-1 w-1 shrink-0 rounded-full" style="background: {venture.color}"></span>
						{f}
					</li>
				{/each}
			</ul>
			<div class="bn-dim mt-2 font-mono text-[10px]">
				Brain tag: #{venture.brainTag} · {venture.agentIds.length} agents on this venture
			</div>
		</div>
	{/if}

	<!-- Life-area legend -->
	<div class="bn-rise bn-surface bn-border mb-6 flex flex-wrap items-center gap-4 rounded-[10px] border px-3 py-2" style="--rise-i: 4">
		<span class="bn-dim text-[9px] uppercase tracking-[0.2em]">Life areas</span>
		{#each view.lifeAreas as area (area.id)}
			<span class="bn-muted flex items-center gap-1.5 text-[10px]">
				<span class="h-2 w-2 rounded-full" style="background: {area.color}"></span>
				{area.label}
			</span>
		{/each}
	</div>

	<!-- Operator -->
	<div class="bn-rise flex flex-col items-center" style="--rise-i: 5">
		<Users class="bn-text h-7 w-7" />
		<div class="bn-text mt-1 text-base font-bold tracking-wide">{operator}</div>
		<div class="bn-dim text-[10px] uppercase tracking-[0.3em]">Operator</div>
		<div class="mt-2 h-6 w-px" style="background: {rail}"></div>
		<div class="flex items-center gap-2">
			<span class="bn-muted text-[10px] uppercase tracking-[0.2em]">Conductor (Super Agent)</span>
			{#if view.live.conductor}<LiveChip agent={view.live.conductor} />{/if}
		</div>
		<div class="h-3 w-px" style="background: {rail}"></div>
	</div>

	<!-- AI Head row: Optimal Engine ── Conductor ── Comms Feed -->
	<div class="bn-rise flex items-center justify-center" style="--rise-i: 6">
		{@render systemCard('/os/brain', 'Optimal Engine', 'governed memory · claims + chunks')}
		<div class="hidden h-px w-10 md:block" style="background: {rail}"></div>
		{#if view.conductor}
			<ConductorCard conductor={view.conductor} agentNames={view.agentNames} initialBroadcast={view.lastBroadcast} />
		{:else}
			<div class="bn-border bn-dim rounded-[6px] border border-dashed px-6 py-4 text-xs">conductor missing — run the founderos ETL</div>
		{/if}
		<div class="hidden h-px w-10 md:block" style="background: {rail}"></div>
		{@render systemCard('/os/comms', 'Comms Feed', 'Gmail · WhatsApp · Slack, unified')}
	</div>

	<div class="mx-auto h-10 w-px" style="background: {rail}"></div>

	<!-- Department crews -->
	<div class="bn-rise overflow-x-auto overflow-y-hidden pb-4" style="--rise-i: 7">
		<div class="mx-auto w-max">
			<div class="mx-36 h-px" style="background: {rail}"></div>
			<div class="flex gap-4 pt-4">
				{#each view.crews as crew (crew.department.id)}
					{@const tint = crew.area?.color ?? crew.department.color}
					<section data-part="crew" class="bn-org-connector flex w-72 shrink-0 flex-col items-center gap-2.5">
						<div class="flex items-center gap-2">
							<span class="bn-text text-xs font-bold">{crew.department.name}</span>
							{#if crew.live}<LiveChip agent={crew.live} />{/if}
						</div>
						{#if crew.area}
							<div class="flex items-center gap-1.5 text-[9px] uppercase tracking-[0.2em]" style="color: {crew.area.color}">
								<span class="h-1.5 w-1.5 rounded-full" style="background: {crew.area.color}"></span>
								{crew.area.label}
							</div>
						{/if}
						<div
							data-lens="r"
							class="bn-pressable is-row flex h-16 w-16 items-center justify-center rounded-[10px]"
							style="background: var(--bn-surface-2); border: 1px solid {crew.area?.color ?? '#333333'}55; box-shadow: 0 0 18px {crew.area?.color ?? '#000000'}22"
						>
							<SparkIcon size={34} shade={tint} />
						</div>

						{#if crew.leads.length > 0}
							<div class="w-full space-y-1.5">
								<div class="bn-dim text-center text-[9px] uppercase tracking-[0.2em]">{crew.department.name} crew</div>
								{#each crew.leads as lead (lead.id)}
									{@const dim = dimFor(venture, lead.id)}
									<div
										data-agent={lead.id}
										data-dim={dim}
										title={lead.description}
										data-lens="r"
										class="bn-pressable is-row bn-surface w-full rounded-[6px] border px-3 py-2"
										class:opacity-20={dim}
										style={venture && !dim
											? `border-color: ${venture.color}66; box-shadow: 0 0 12px ${venture.color}1a`
											: `border-color: ${rail}`}
									>
										<div class="flex items-center justify-between gap-2">
											<div class="flex min-w-0 items-center gap-1.5">
												<OrgDot look={rosterDot(lead.status)} />
												<span class="bn-text truncate text-xs font-bold">{lead.name}</span>
												<VentureDots ventures={view.ventures} agentId={lead.id} />
											</div>
											<span class="bn-dim shrink-0 rounded-[4px] px-1.5 py-0.5 font-mono text-[8px] uppercase tracking-wider" style="background: var(--bn-surface-2)">
												{lead.instance}
											</span>
										</div>
										<div class="bn-dim mt-0.5 truncate pl-3 text-[10px]">{lead.role}</div>
									</div>
								{/each}
							</div>
						{/if}

						{#if crew.pills.length > 0}
							<div class="grid w-full grid-cols-2 gap-1.5">
								{#each crew.pills as node (node.agent.id)}
									<AgentNodePill {node} {venture} ventures={view.ventures} />
								{/each}
							</div>
						{/if}

						{#if crew.empty}
							<div class="bn-border bn-dim w-full rounded-[6px] border border-dashed px-3 py-5 text-center text-[10px]">
								Agents land here as this department goes live
							</div>
						{/if}

						{#if crew.tools.length > 0}
							<div class="w-full">
								<div class="bn-muted rounded-[6px] px-2 py-1 text-center text-[8px] uppercase tracking-[0.2em]" style="background: var(--bn-surface-2)">Agent Tools</div>
								<div class="mt-1.5 flex flex-wrap justify-center gap-1">
									{#each crew.tools as tool (tool)}
										<span class="bn-border bn-muted rounded-[4px] border px-1.5 py-0.5 font-mono text-[9px]">{tool}</span>
									{/each}
								</div>
							</div>
						{/if}
					</section>
				{/each}
			</div>
		</div>
	</div>
</div>

<style>
	/* prod globals.css .org-connector: a 1px tick from the rail down to each crew */
	.bn-org-connector {
		position: relative;
	}
	.bn-org-connector::before {
		content: '';
		position: absolute;
		top: -16px;
		left: 50%;
		width: 1px;
		height: 16px;
		background: var(--bn-border-strong);
	}
</style>
