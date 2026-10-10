<!-- Right glass panel: one tab per opened thing (last five), the active one
     showing what it is, its facts, what is inside it and every connection,
     each connection a click that jumps the map there. Port of FounderOS v1
     components/blueprint/HierarchyInspector.tsx. -->
<script lang="ts">
	import { Focus, Maximize2, Minimize2, PanelRightClose, PanelRightOpen, Scan, X } from '$lib/founderos/icons';
	import { KIND_ICON, KIND_LABEL, STATUS_LABEL, relationsOf, type HAny, type Hierarchy, type HierarchyIndex, type HRelationHit } from './hierarchy';
	import { iconFor } from './icons';

	let {
		h,
		idx,
		opened,
		active,
		expanded,
		minimized,
		onselect,
		onclosetab,
		onminimize,
		onfocus,
		onfit,
		ontoggle
	}: {
		h: Hierarchy;
		idx: HierarchyIndex;
		opened: string[];
		active: string | null;
		expanded: Set<string>;
		minimized: boolean;
		onselect: (id: string) => void;
		onclosetab: (id: string) => void;
		onminimize: (v: boolean) => void;
		onfocus: () => void;
		onfit: () => void;
		ontoggle: (id: string) => void;
	} = $props();

	const colorOf = (it: HAny) => `--c: var(--bh-k-${it.kind}, var(--text-2))`;

	const has = $derived(opened.length > 0);
	const current = $derived(active && opened.includes(active) ? active : opened[opened.length - 1]);
	const it = $derived(current ? idx.items.get(current) : undefined);
	const handleLabel = $derived((active && idx.items.get(active)?.name) || it?.name || 'Details');
	const rels = $derived(it ? relationsOf(h, idx, it.id) : []);
	const grouped = $derived.by(() => {
		const m = new Map<string, HRelationHit[]>();
		for (const r of rels) {
			const label = r.dir === 'out' ? r.via || r.kind : `${r.via || r.kind} ←`;
			m.set(label, [...(m.get(label) ?? []), r]);
		}
		return [...m.entries()];
	});
	const facts = $derived(it && (it.type === 'node' || it.type === 'container') ? Object.entries(it.facts) : []);
	const path = $derived(it ? idx.chain(it.id).slice(0, -1) : []);
	const members = $derived(it && it.type !== 'node' ? idx.descendants(it.id).filter((m) => m.type === 'node') : []);
	const Icon = $derived(it ? iconFor(it.icon || KIND_ICON[it.kind]) : null);
	const status = $derived(it ? (it.type === 'node' ? it.status : it.type === 'group' ? it.status : undefined) : undefined);
</script>

<button type="button" class="bn-pressable bh-insp-handle" title="Show details (i)" onclick={() => onminimize(false)} style="display: {has && minimized ? 'inline-flex' : 'none'}">
	<PanelRightOpen />
	<span>{handleLabel}</span>
</button>
<aside class="bh-inspector" aria-hidden={!has || minimized}>
	<div class="bh-insp-bar">
		<div class="bh-insp-tabs">
			{#each opened as id (id)}
				<button type="button" class="bn-pressable bh-itab {id === current ? 'is-on' : ''}" onclick={() => onselect(id)}>
					<span>{idx.items.get(id)?.name ?? id}</span>
					<!-- svelte-ignore a11y_click_events_have_key_events -->
					<span
						class="x"
						role="button"
						tabindex="-1"
						aria-label="Close"
						onclick={(e) => {
							e.stopPropagation();
							onclosetab(id);
						}}><X /></span
					>
				</button>
			{/each}
		</div>
		<button type="button" class="bn-pressable bh-insp-min" title="Hide panel (i)" onclick={() => onminimize(true)}><PanelRightClose /></button>
	</div>
	{#if it && Icon}
		{#key it.id}
			<div class="bh-insp-body">
				<div class="bh-insp-head">
					<div class="bh-tile" style={colorOf(it)}><Icon /></div>
					<div>
						<div class="t">{it.name}</div>
						<div class="s">
							<span class="bh-kchip" style={colorOf(it)}>{KIND_LABEL[it.kind]}</span>
							{#if status}
								<span class="bh-badge" data-s={status}><i></i>{STATUS_LABEL[status]}</span>
							{/if}
						</div>
					</div>
				</div>
				{#if path.length > 0}
					<div class="bh-insp-path">
						{#each path as p, i (p)}
							<span><b>{idx.items.get(p)?.name ?? p}</b>{i < path.length - 1 ? ' › ' : ''}</span>
						{/each}
					</div>
				{/if}
				{#if it.sub}
					<div class="bh-insp-sec">
						<div class="h">What it is</div>
						<div class="bh-insp-what">{it.sub}</div>
					</div>
				{/if}
				{#if facts.length > 0}
					<div class="bh-insp-sec">
						<div class="h">Facts</div>
						<div class="bh-kv">
							{#each facts as [k, v] (k)}
								<div class="k">{k}</div>
								<div class="v">{v === '' ? 'none' : v}</div>
							{/each}
						</div>
					</div>
				{/if}
				{#if members.length > 0}
					<div class="bh-insp-sec">
						<div class="h"><span>Inside</span><span>{members.length}</span></div>
						{#each members.slice(0, 14) as m (m.id)}
							{@const MI = iconFor(m.icon || KIND_ICON[m.kind])}
							<!-- svelte-ignore a11y_click_events_have_key_events -->
							<!-- svelte-ignore a11y_no_static_element_interactions -->
							<div class="bh-conn" onclick={() => onselect(m.id)}>
								<div class="bh-tile" style={colorOf(m)}><MI /></div>
								<div class="min-w-0">
									<div class="t">{m.name}</div>
									<div class="s">{m.sub}</div>
								</div>
								<span class="kind">{m.type === 'node' ? STATUS_LABEL[m.status] : ''}</span>
							</div>
						{/each}
						{#if members.length > 14}<div class="bh-insp-more">+{members.length - 14} more</div>{/if}
					</div>
				{/if}
				{#if rels.length > 0}
					<div class="bh-insp-sec">
						<div class="h"><span>Connections</span><span>{rels.length}</span></div>
						{#each grouped as [label, list] (label)}
							{#each list.slice(0, 10) as r (`${label}-${r.other}`)}
								{@const o = idx.items.get(r.other)}
								{#if o}
									{@const OI = iconFor(o.icon || KIND_ICON[o.kind])}
									<!-- svelte-ignore a11y_click_events_have_key_events -->
									<!-- svelte-ignore a11y_no_static_element_interactions -->
									<div class="bh-conn" onclick={() => onselect(r.other)}>
										<div class="bh-tile" style={colorOf(o)}><OI /></div>
										<div class="min-w-0">
											<div class="t">{o.name}</div>
											<div class="s">{o.sub}</div>
										</div>
										<span class="kind {r.dir === 'out' ? 'is-out' : ''}">{label}</span>
									</div>
								{/if}
							{/each}
						{/each}
					</div>
				{/if}
				<div class="bh-insp-actions">
					<button type="button" class="bn-pressable bh-btn" onclick={onfocus}><Focus /><span>Focus</span></button>
					{#if it.type === 'group'}
						{@const id = it.id}
						<button type="button" class="bn-pressable bh-btn" onclick={() => ontoggle(id)}>
							{#if expanded.has(id)}<Minimize2 />{:else}<Maximize2 />{/if}
							<span>{expanded.has(id) ? 'Collapse' : 'Expand'}</span>
						</button>
					{:else}
						<button type="button" class="bn-pressable bh-btn" onclick={onfit}><Scan /><span>Whole map</span></button>
					{/if}
				</div>
			</div>
		{/key}
	{/if}
</aside>
