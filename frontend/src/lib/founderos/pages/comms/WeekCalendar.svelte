<!-- 7-day week grid (FounderOS v1 WeekCalendar.tsx): days across the top, hours
     down the side, meetings at their times colored per account; events with a
     video link join on click. Local time. An unreachable calendar says so
     instead of drawing an empty week. -->
<script lang="ts">
	import { Video } from '$lib/founderos/icons';
	import { assignLanes, weekLayout, type CalEvent } from './model';

	let { events, accounts, nowISO, error }: { events: CalEvent[]; accounts: Array<{ name: string; color: string }>; nowISO: string; error?: string } = $props();

	const HOUR_PX = 46;
	const GUTTER = 50;
	const gridCols = `${GUTTER}px repeat(7, minmax(96px, 1fr))`;

	const w = $derived(weekLayout(events, nowISO));
	const hours = $derived(Array.from({ length: w.hiHour - w.loHour }, (_, i) => w.loHour + i));
	const bodyHeight = $derived((w.hiHour - w.loHour) * HOUR_PX);
	const nowTop = $derived((w.nowMin / 60 - w.loHour) * HOUR_PX);
	const showNow = $derived(w.nowMin / 60 >= w.loHour && w.nowMin / 60 <= w.hiHour);
	const hasAllDay = $derived(w.allDay.some((d) => d.length > 0));

	const fmtHour = (h: number) => `${h % 12 === 0 ? 12 : h % 12} ${h < 12 || h === 24 ? 'AM' : 'PM'}`;
	const fmtTime = (iso: string) => new Date(iso).toLocaleTimeString([], { hour: 'numeric', minute: '2-digit' });
</script>

<div data-part="week">
	{#if error}
		<p data-part="calendar-error" class="mb-3 rounded-[8px] border px-3 py-2 font-mono text-[10.5px]" style="border-color: color-mix(in oklab, var(--bn-err) 40%, transparent); color: var(--bn-err)">
			Calendar unreachable: {error}. Meetings for the week are unknown.
		</p>
	{/if}
	<div class="mb-3 flex flex-wrap items-center gap-x-4 gap-y-1.5">
		{#each accounts as a (a.name)}
			<div class="bn-muted flex items-center gap-1.5 font-mono text-[10.5px]">
				<span class="h-2.5 w-2.5 rounded-[2px]" style="background: {a.color}"></span>
				{a.name}
			</div>
		{/each}
	</div>

	<div class="pc-surface overflow-x-auto rounded-[10px]">
		<div class="pc-hair grid border-b" style="grid-template-columns: {gridCols}">
			<div></div>
			{#each w.columns as c, i (c.key)}
				<div class="pc-hair border-l px-2 py-2 text-center" style={i === 0 ? 'background: var(--bn-accent-soft)' : ''}>
					<div class="font-mono text-[9.5px] uppercase tracking-[0.12em] {i === 0 ? 'bn-accent' : 'bn-dim'}">{c.date.toLocaleDateString([], { weekday: 'short' })}</div>
					<div class="text-[15px] font-semibold {i === 0 ? 'bn-accent' : 'bn-text'}">{c.date.getDate()}</div>
				</div>
			{/each}
		</div>

		{#if hasAllDay}
			<div class="pc-hair grid border-b" style="grid-template-columns: {gridCols}">
				<div class="bn-dim flex items-center justify-end pr-2 font-mono text-[9px] uppercase tracking-[0.1em]">all-day</div>
				{#each w.allDay as list, i (i)}
					<div class="pc-hair flex flex-col gap-1 border-l p-1">
						{#each list as ev (ev.id)}
							<svelte:element
								this={ev.joinUrl ? 'a' : 'div'}
								href={ev.joinUrl ?? undefined}
								target={ev.joinUrl ? '_blank' : undefined}
								rel={ev.joinUrl ? 'noopener noreferrer' : undefined}
								class="block truncate rounded-[3px] px-1.5 py-0.5 text-[10px] font-semibold"
								style="background: {ev.color}; color: var(--bn-bg)"
								title={`${ev.title} · ${ev.account}`}>{ev.title}</svelte:element
							>
						{/each}
					</div>
				{/each}
			</div>
		{/if}

		<div class="grid" style="grid-template-columns: {gridCols}">
			<div class="relative" style="height: {bodyHeight}px">
				{#each hours as h, i (h)}
					<div class="bn-dim absolute right-2 -translate-y-1/2 font-mono text-[9px]" style="top: {i * HOUR_PX}px">{i === 0 ? '' : fmtHour(h)}</div>
				{/each}
			</div>
			{#each w.timed as dayEvents, i (w.columns[i].key)}
				{@const lanes = assignLanes(dayEvents.map((p) => ({ startMin: p.startMin, endMin: p.endMin })))}
				<div class="pc-hair relative border-l" style="height: {bodyHeight}px">
					{#each hours as h, hi (h)}
						<!-- prod writes border-os-border/50, which Tailwind cannot apply to its var() colour, so the hour rules fall back to currentColor: full-ink lines. Drawn the same here. -->
						<div data-part="hour-rule" class="absolute inset-x-0 border-t" style="top: {hi * HOUR_PX}px; border-color: var(--bn-text)"></div>
					{/each}
					{#if i === 0 && showNow}
						<div class="absolute inset-x-0 z-10 border-t-2" style="top: {nowTop}px; border-color: var(--bn-accent)">
							<span class="absolute -left-1 -top-[3px] h-1.5 w-1.5 rounded-full" style="background: var(--bn-accent)"></span>
						</div>
					{/if}
					{#each dayEvents as p, j (p.ev.id)}
						{@const top = (p.startMin / 60 - w.loHour) * HOUR_PX}
						{@const height = Math.max(22, ((p.endMin - p.startMin) / 60) * HOUR_PX - 3)}
						{@const widthPct = 100 / lanes[j].lanes}
						<svelte:element
							this={p.ev.joinUrl ? 'a' : 'div'}
							data-part="event"
							href={p.ev.joinUrl ?? undefined}
							target={p.ev.joinUrl ? '_blank' : undefined}
							rel={p.ev.joinUrl ? 'noopener noreferrer' : undefined}
							class="group absolute overflow-hidden rounded-[4px] border-l-[3px] px-1.5 py-1 {p.ev.joinUrl ? 'cursor-pointer' : ''}"
							style="top: {top}px; height: {height}px; left: calc({lanes[j].lane * widthPct}% + 2px); width: calc({widthPct}% - 4px); border-left-color: {p.ev.color}; background: color-mix(in srgb, {p.ev.color} 16%, var(--bn-surface))"
							title={`${p.ev.title} · ${p.ev.account}${p.ev.joinUrl ? ' · click to join' : ''}`}
						>
							<div class="flex items-center gap-1">
								<span class="bn-text truncate text-[11px] font-semibold leading-tight">{p.ev.title}</span>
								{#if p.ev.joinUrl}<Video size={12} class="bn-accent ml-auto shrink-0" />{/if}
							</div>
							{#if height >= 38}
								<div class="bn-dim mt-0.5 truncate font-mono text-[9px]">{fmtTime(p.ev.start)} · {p.ev.account}</div>
							{/if}
						</svelte:element>
					{/each}
				</div>
			{/each}
		</div>
	</div>
</div>
