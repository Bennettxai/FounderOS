<!-- FounderOS v1 Sidebar on the bridge: fixed, grouped nav from nav.ts, active
     state, the live systems count, and three remembered shapes: expanded
     (drag the right edge to resize, double-click resets), the icon rail, and
     hidden (drag past the edge; the far-left strip peeks it back, ⌘\ docks). -->
<script lang="ts">
	import { onMount, type Snippet } from 'svelte';
	import { PanelLeft } from '$lib/founderos/icons';
	import '../kit/kit.css';
	import OsMark from '../kit/OsMark.svelte';
	import { NAV_BUSINESSOS, NAV_GROUPS, isActive, type NavGroup } from '../nav';
	import { RAIL_W, loadKey, saveKey } from './chrome';
	import { SIDEBAR, SIDEBAR_KEYS, dragTo, inHotZone, peekShouldEnd, readStoredLayout, shellOffset } from './sidebar-layout';

	let {
		pathname,
		live = null,
		host = null,
		workspace,
		trafficLights = false,
		visibleHrefs = null
	}: {
		pathname: string;
		/** Connected / total connectors; null while unknown. */
		live?: { up: number; total: number } | null;
		host?: string | null;
		/** The BusinessOS workspace switcher, shown under the header when expanded. */
		workspace?: Snippet;
		/** Electron on macOS: the window's traffic lights sit over the top-left
		 *  corner, so reserve a draggable strip for them (as the (app) shell does). */
		trafficLights?: boolean;
		/** The /os pages switched on for the current workspace; null shows all. */
		visibleHrefs?: Set<string> | null;
	} = $props();

	const groups: NavGroup[] = $derived([
		...NAV_GROUPS.map((g) => ({ ...g, items: visibleHrefs ? g.items.filter((i) => visibleHrefs.has(i.href)) : g.items })).filter(
			(g) => g.items.length > 0
		),
		{ title: 'BusinessOS', sub: 'BusinessOS', items: NAV_BUSINESSOS }
	]);

	// The first render is the expanded default; the saved shape lands on mount.
	let collapsed = $state(false); // the icon rail
	let hidden = $state(false); // gone completely (0px)
	let peek = $state(false); // hidden, but revealed as an overlay
	let width = $state<number>(SIDEBAR.DEFAULT_W);
	let restored = false;
	let dragging = false;
	// The nav scrolls, and a scrolling ancestor clips an absolutely positioned
	// child, so the rail's label is rendered fixed, outside that box.
	let tip = $state<{ label: string; y: number } | null>(null);

	onMount(() => {
		const saved = readStoredLayout(loadKey);
		width = saved.width;
		collapsed = saved.rail && !saved.hidden;
		hidden = saved.hidden;
		restored = true;

		// Hidden: the far-left strip reveals it as an overlay, and it goes away
		// once the mouse is clear of it. No click needed either way.
		const onMove = (ev: MouseEvent) => {
			if (!hidden || dragging) return;
			if (inHotZone(ev.clientX)) peek = true;
			else if (peekShouldEnd(ev.clientX, width)) peek = false;
		};
		const onLeave = () => {
			if (hidden && !dragging) peek = false;
		};
		// ⌘\ / Ctrl+\ hides or docks it, so a hidden sidebar is never mouse-only.
		const onKey = (e: KeyboardEvent) => {
			if ((e.metaKey || e.ctrlKey) && e.key === '\\') {
				e.preventDefault();
				collapsed = false;
				hidden = !hidden;
			}
		};
		window.addEventListener('mousemove', onMove);
		window.addEventListener('keydown', onKey);
		document.documentElement.addEventListener('mouseleave', onLeave);
		return () => {
			window.removeEventListener('mousemove', onMove);
			window.removeEventListener('keydown', onKey);
			document.documentElement.removeEventListener('mouseleave', onLeave);
		};
	});

	// The shell's left margin reads this, so the page follows the sidebar.
	$effect(() => {
		document.documentElement.style.setProperty('--bn-sidebar-w', `${shellOffset({ hidden, rail: collapsed, width })}px`);
	});

	$effect(() => {
		const rail = collapsed ? '1' : '0';
		const gone = hidden ? '1' : '0';
		if (!restored) return;
		saveKey(SIDEBAR_KEYS.rail, rail);
		saveKey(SIDEBAR_KEYS.hidden, gone);
	});

	$effect(() => {
		if (!hidden) peek = false;
	});

	// A stale label must never hang around after the rail expands.
	$effect(() => {
		if (!collapsed) tip = null;
	});

	function toggle() {
		restored = true;
		if (hidden) {
			// the button on a peeking overlay docks it, expanded
			hidden = false;
			collapsed = false;
			return;
		}
		collapsed = !collapsed;
	}

	// Drag the right edge (or, while hidden, out from the screen edge): past
	// HIDE_AT it hides completely, above it it docks at that width.
	function onDragStart(e: MouseEvent) {
		if (collapsed) return;
		e.preventDefault();
		dragging = true;
		restored = true;
		// A drag that ends hidden must not keep the width it passed through.
		const startWidth = width;
		document.body.style.cursor = 'col-resize';
		document.body.style.userSelect = 'none';
		const onMove = (ev: MouseEvent) => {
			if (!dragging) return;
			const next = dragTo(ev.clientX);
			hidden = next.hidden;
			if (next.hidden) {
				peek = false;
				width = startWidth;
			} else width = next.width;
		};
		const onUp = () => {
			dragging = false;
			document.body.style.cursor = '';
			document.body.style.userSelect = '';
			window.removeEventListener('mousemove', onMove);
			window.removeEventListener('mouseup', onUp);
			saveKey(SIDEBAR_KEYS.width, String(width));
		};
		window.addEventListener('mousemove', onMove);
		window.addEventListener('mouseup', onUp);
	}

	function resetWidth() {
		width = SIDEBAR.DEFAULT_W;
		saveKey(SIDEBAR_KEYS.width, String(width));
	}
</script>

{#if hidden && !peek}
	<!-- Hidden: a thin strip on the far-left edge. Touching it reveals the
	     sidebar; dragging out from it docks it. -->
	<!-- svelte-ignore a11y_interactive_supports_focus -->
	<div
		role="button"
		aria-label="Reveal sidebar"
		title={'Reveal sidebar · drag out to dock · ⌘\\ or Ctrl+\\'}
		onmouseenter={() => (peek = true)}
		onmousedown={onDragStart}
		class="bn-hot fixed inset-y-0 left-0 z-40 cursor-col-resize border-r"
		style="width: {SIDEBAR.HOT_ZONE}px"
	></div>
{/if}

<!-- svelte-ignore a11y_no_noninteractive_element_interactions -->
<aside
	class="bn-sidebar fixed inset-y-0 left-0 flex flex-col border-r {hidden ? 'z-40' : 'z-20'}"
	data-hidden={hidden}
	data-peek={hidden && peek}
	style:width="{collapsed ? RAIL_W : width}px"
	style:transform={hidden && !peek ? 'translateX(-100%)' : undefined}
	style:visibility={hidden && !peek ? 'hidden' : undefined}
	onmouseleave={(e) => {
		if (hidden && !dragging && peekShouldEnd(e.clientX, width)) peek = false;
	}}
>
	{#if trafficLights}
		<div data-part="traffic-lights" class="h-10 shrink-0" style="-webkit-app-region: drag;"></div>
	{/if}
	<div class="flex pb-[18px] {trafficLights ? 'pt-1' : 'pt-5'} {collapsed ? 'flex-col items-center gap-2 px-0' : 'items-center justify-between px-[18px]'}">
		<!-- The emblem is the OS logo. Collapsed, it IS the identity: the wordmark
		     is gone and the toggle stacks underneath. -->
		{#if collapsed}
			<OsMark size={30} class="shrink-0" />
		{:else}
			<div class="flex items-center gap-[11px]">
				<OsMark size={34} class="shrink-0" />
				<div>
					<div class="bn-text text-[13px] font-bold tracking-[0.14em]">FOUNDER OS</div>
					<div class="bn-dim mt-[3px] whitespace-nowrap font-mono text-[9px] uppercase tracking-[0.16em]">v3 · Operator Mode</div>
				</div>
			</div>
		{/if}
		<button
			type="button"
			onclick={toggle}
			title={hidden ? 'Dock sidebar' : collapsed ? 'Expand sidebar' : 'Collapse sidebar'}
			aria-label={collapsed ? 'Expand sidebar' : 'Collapse sidebar'}
			aria-expanded={!collapsed}
			data-lens="c"
			class="bn-pressable is-dark bn-ctl bn-dim flex h-7 w-7 shrink-0 items-center justify-center rounded-[6px] border"
		>
			<PanelLeft size={15} strokeWidth={1.7} />
		</button>
	</div>

	{#if workspace && !collapsed}
		<div class="bn-ws px-2.5 pb-2">{@render workspace()}</div>
	{/if}

	<nav class="flex flex-1 flex-col gap-0.5 overflow-y-auto px-2.5 pb-2">
		{#each groups as group (group.title)}
			{#if collapsed}
				<div class="bn-rule mx-2 my-1.5 border-t" aria-label={group.title}></div>
			{:else}
				<div data-part="group-title" class="bn-dim px-2.5 pb-1.5 pt-3.5 font-mono text-[9px] uppercase tracking-[0.18em]">{group.title}</div>
			{/if}
			{#each group.items as item (item.href)}
				{@const active = isActive(item.href, pathname)}
				{@const Icon = item.icon}
				<a
					href={item.href}
					aria-label={collapsed ? item.label : undefined}
					onmouseenter={(e) => {
						if (collapsed) tip = { label: item.label, y: e.currentTarget.getBoundingClientRect().top };
					}}
					onmouseleave={() => {
						if (collapsed) tip = null;
					}}
					aria-current={active ? 'page' : undefined}
					data-active={active ? 'true' : 'false'}
					data-lens="r"
					class="bn-pressable is-dark bn-nav-item group relative flex items-center rounded-[6px] border text-[13.5px] font-medium {collapsed
						? 'justify-center px-0 py-[9px]'
						: 'gap-2.5 px-2.5 py-[7px]'}"
				>
					<Icon size={15} strokeWidth={1.7} class="shrink-0 opacity-85" />
					{#if !collapsed}{item.label}{/if}
				</a>
			{/each}
		{/each}
	</nav>

	<div class="bn-rule flex flex-col gap-2 border-t py-3.5 {collapsed ? 'items-center px-0' : 'px-[18px]'}">
		<div class="bn-muted flex items-center gap-2 whitespace-nowrap font-mono text-[10px]">
			<span class="bn-dot {live ? 'ok pulse' : 'off'}" aria-hidden="true"></span>
			{#if !collapsed}{live ? `${live.up}/${live.total}` : '—/—'} systems live{/if}
		</div>
		{#if !collapsed}
			<div class="bn-dim break-words font-mono text-[10px] leading-relaxed">{host ?? '…'} · postgres · optimal engine</div>
		{/if}
	</div>

	{#if collapsed && tip}
		<div
			data-part="rail-tip"
			class="bn-tip pointer-events-none fixed z-50 -translate-y-1/2 whitespace-nowrap rounded-[5px] border px-2 py-1 font-mono text-[10px] uppercase tracking-[0.14em]"
			style="left: {RAIL_W + 8}px; top: {tip.y + 15}px"
		>{tip.label}</div>
	{/if}

	{#if !collapsed}
		<!-- the right edge: a wide invisible hit area on the border itself -->
		<!-- svelte-ignore a11y_no_noninteractive_element_interactions -->
		<div
			role="separator"
			aria-orientation="vertical"
			aria-label="Resize sidebar"
			title="Drag to resize · drag to the edge to hide · double-click to reset"
			onmousedown={onDragStart}
			ondblclick={resetWidth}
			class="bn-edge absolute inset-y-0 -right-1 z-30 w-2 cursor-col-resize"
		></div>
	{/if}
</aside>

<style>
	.bn-sidebar {
		background: var(--bn-bg-2);
		border-color: var(--bn-border);
	}
	.bn-rule {
		border-color: var(--bn-border);
	}
	/* WorkspaceSwitcher's BusinessOS text tokens, mapped onto the operator's. */
	.bn-ws {
		--dt: var(--bn-text);
		--dt2: var(--bn-text-2);
		--dt3: var(--bn-text-3);
	}
	.bn-tip {
		border-color: var(--bn-border-strong);
		background: var(--bn-surface);
		color: var(--bn-text);
	}
	.bn-hot {
		border-color: transparent;
	}
	.bn-hot:hover {
		border-color: var(--bn-border-strong);
	}
	.bn-edge:hover {
		background: var(--bn-accent-soft);
	}
	/* Rest colors only: hover, press and focus are the .bn-pressable lens
	   (kit.css), exactly as Sidebar.tsx pairs `pressable is-dark` with them. */
	.bn-ctl {
		border-color: transparent;
	}
	.bn-nav-item {
		border-color: transparent;
		color: var(--bn-text-2);
	}
	.bn-nav-item:where([data-active='true']) {
		border-color: var(--bn-accent-line);
		background: var(--bn-accent-soft);
		color: var(--bn-accent);
	}
</style>
