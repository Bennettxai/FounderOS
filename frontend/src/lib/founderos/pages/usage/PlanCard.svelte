<!-- One plan's card (UsageBoard.tsx PlanCard). A plan no machine pushed says so. -->
<script lang="ts">
	import { Badge, Dot, VolumeMeter } from '$lib/founderos/kit';
	import LaneBar from './LaneBar.svelte';
	import WindowStrip from './WindowStrip.svelte';
	import { age, limitMeter, totalBurn, type PlanUsage } from './usage';

	let { title, plan, selected, onselect, now }: { title: string; plan: PlanUsage | null; selected: boolean; onselect: () => void; now: number } =
		$props();

	const hasData = $derived(plan ? totalBurn(plan.days) > 0 || plan.lastActivity !== null : false);
	import { EMPTY_CARD, planCardClass } from './ui';
</script>

{#if !plan}
	<div data-part="plan-card" class={EMPTY_CARD}>
		<span class="bn-muted text-[15px] font-semibold">{title}</span>
		<p class="mt-2">no sessions found on any reporting machine</p>
	</div>
{:else}
	<button
		type="button"
		data-part="plan-card"
		onclick={onselect}
		aria-expanded={selected}
		data-lens="r"
		class={planCardClass(selected)}
	>
		<div class="flex items-center justify-between gap-2">
			<div class="flex items-center gap-2">
				<Dot state={hasData ? 'ok' : 'off'} />
				<span class="bn-text text-[15px] font-semibold tracking-[-0.01em]">{title}</span>
			</div>
			{#if plan.planConflict}
				<Badge tone="warn">plans differ</Badge>
			{:else}
				<Badge tone={plan.plan ? 'accent' : 'default'}>{plan.plan ? `${plan.plan} · active` : 'plan unknown'}</Badge>
			{/if}
		</div>
		{#if plan.planConflict}
			<p class="mt-2 font-mono text-[10px] leading-relaxed text-[color:var(--bn-warn)]">
				machines are logged into different plans: {plan.planConflict.join(' · ')}
			</p>
		{/if}

		{#if plan.official?.session || plan.official?.weekly}
			<div class="mt-4 space-y-4">
				{#if plan.official.session}<VolumeMeter {...limitMeter('Session limit', '5h', plan.official.session, now)} />{/if}
				{#if plan.official.weekly}<VolumeMeter {...limitMeter('Weekly limit', '7d', plan.official.weekly, now)} delay={150} />{/if}
			</div>
		{:else}
			<p class="bn-dim mt-4 text-[12px] leading-relaxed">no official gauge reported · burn below is measured from transcripts</p>
		{/if}

		<div class="mt-4"><WindowStrip {plan} /></div>

		<div class="mt-4 space-y-1.5">
			<LaneBar {plan} window="week" />
			<div class="bn-dim flex items-center justify-between font-mono text-[10px]">
				<span>{plan.machines.length} machine{plan.machines.length === 1 ? '' : 's'} · last activity {age(plan.lastActivity, now)}</span>
				<span>{selected ? 'breakdown ▾' : 'breakdown ▸'}</span>
			</div>
		</div>
	</button>
{/if}
