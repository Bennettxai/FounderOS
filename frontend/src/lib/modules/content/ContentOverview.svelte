<script lang="ts">
	import type { ContentItem } from '$lib/api/content';
	import { CalendarDays, CheckCircle2, Clapperboard, FileText, FolderKanban, Plus } from 'lucide-svelte';
	import { contentProfile, contentWorkstream, normalizeStatus, unique } from './model';

	type Props = {
		items: ContentItem[];
		workspaceName: string;
		knownProfiles?: string[];
		scope?: 'my' | 'clients';
		onNew: () => void;
		onOpenPipeline: () => void;
		onEdit: (item: ContentItem) => void;
	};

	let { items, workspaceName, knownProfiles = [], scope = 'my', onNew, onOpenPipeline, onEdit }: Props = $props();
	let lastCreateActionAt = 0;

	const profiles = $derived(unique([...knownProfiles, ...items.map(contentProfile)]));
	const workstreams = $derived(unique(items.map(contentWorkstream)));
	const production = $derived(items.filter((item) => ['scripting', 'to_film', 'to_edit'].includes(normalizeStatus(item.status))));
	const review = $derived(items.filter((item) => ['client_review', 'approved'].includes(normalizeStatus(item.status))));
	const ready = $derived(items.filter((item) => normalizeStatus(item.status) === 'to_post'));
	const upcoming = $derived(
		items
			.filter((item) => item.publish_date)
			.sort((a, b) => a.publish_date.localeCompare(b.publish_date))
			.slice(0, 5)
	);

	function triggerNew(event: Event) {
		event.preventDefault();
		event.stopPropagation();
		const now = Date.now();
		if (now - lastCreateActionAt < 180) return;
		lastCreateActionAt = now;
		onNew();
	}
</script>

<section class="workspace-hero">
	<div>
		<span>{scope === 'clients' ? 'Client content' : 'Workspace content'}</span>
		<h2>{scope === 'clients' ? 'Client Content' : workspaceName}</h2>
		<p>{scope === 'clients' ? 'Plan, produce, review, and publish content for client profiles.' : 'Plan, produce, review, and publish every content record owned by this workspace.'}</p>
	</div>
	<div class="actions">
		<button class="primary" type="button" onpointerdown={triggerNew} onmousedown={triggerNew} onclick={triggerNew}><Plus size={16} />New content</button>
		<button onclick={onOpenPipeline}><FolderKanban size={16} />Open pipeline</button>
	</div>
</section>

{#if items.length === 0}
	<section class="empty">
		<FileText size={28} strokeWidth={1.4} />
		<h3>{scope === 'clients' ? 'No client content yet' : 'No content in this workspace'}</h3>
		<p>{scope === 'clients' ? `Create the first record for ${profiles[0] || 'the first client'}.` : 'Create the first record. Profiles and workstreams will appear from the content your team adds.'}</p>
		{#if scope === 'clients' && profiles.length}
			<div class="inventory-group empty-profiles"><span>Client profiles</span><div>{#each profiles as profile}<b>{profile}</b>{/each}</div></div>
		{/if}
		<button class="primary" type="button" onpointerdown={triggerNew} onmousedown={triggerNew} onclick={triggerNew}><Plus size={16} />Add content</button>
	</section>
{:else}
	<section class="metrics" aria-label="Workspace content summary">
		<div><FileText size={16} /><span>Total</span><strong>{items.length}</strong></div>
		<div><Clapperboard size={16} /><span>In production</span><strong>{production.length}</strong></div>
		<div><CheckCircle2 size={16} /><span>Ready to post</span><strong>{ready.length}</strong></div>
		<div><CalendarDays size={16} /><span>Profiles</span><strong>{profiles.length}</strong></div>
	</section>

	<div class="overview-grid">
		<section class="panel">
			<header><div><span>Production</span><h3>Work in motion</h3></div><strong>{production.length}</strong></header>
			{#if production.length}
				<div class="rows">
					{#each production.slice(0, 5) as item (item.id)}
						<button onclick={() => onEdit(item)}>
							<span class="stage">{normalizeStatus(item.status).replaceAll('_', ' ')}</span>
							<strong>{item.title}</strong>
							<small>{contentProfile(item)} · {contentWorkstream(item)}</small>
						</button>
					{/each}
				</div>
			{:else}<p class="quiet">Nothing is in production right now.</p>{/if}
		</section>

		<section class="panel">
			<header><div><span>Approvals</span><h3>Review queue</h3></div><strong>{review.length}</strong></header>
			{#if review.length}
				<div class="rows">
					{#each review.slice(0, 5) as item (item.id)}
						<button onclick={() => onEdit(item)}>
							<span class="stage">{normalizeStatus(item.status).replaceAll('_', ' ')}</span>
							<strong>{item.title}</strong>
							<small>{item.editor || item.owner || 'Unassigned'}</small>
						</button>
					{/each}
				</div>
			{:else}<p class="quiet">No content is waiting for review.</p>{/if}
		</section>

		<section class="panel">
			<header><div><span>Schedule</span><h3>Upcoming posts</h3></div><strong>{upcoming.length}</strong></header>
			{#if upcoming.length}
				<div class="rows">
					{#each upcoming as item (item.id)}
						<button onclick={() => onEdit(item)}>
							<span class="date">{item.publish_date}</span>
							<strong>{item.title}</strong>
							<small>{item.channel || contentWorkstream(item)}</small>
						</button>
					{/each}
				</div>
			{:else}<p class="quiet">No publish dates have been scheduled.</p>{/if}
		</section>

		<section class="panel inventory">
			<header><div><span>Workspace structure</span><h3>Content inventory</h3></div></header>
			<div class="inventory-group"><span>Profiles</span><div>{#each profiles as profile}<b>{profile}</b>{/each}</div></div>
			<div class="inventory-group"><span>Workstreams</span><div>{#each workstreams as workstream}<b>{workstream}</b>{/each}</div></div>
		</section>
	</div>
{/if}

<style>
	.workspace-hero { display: flex; align-items: flex-start; justify-content: space-between; gap: 24px; padding: 20px; border-bottom: 1px solid var(--dbd); }
	.workspace-hero span, .panel header span, .inventory-group > span { color: var(--dt3); font-size: .7rem; font-weight: 750; text-transform: uppercase; }
	h2 { margin: 4px 0 7px; color: var(--dt); font-size: 1.45rem; letter-spacing: 0; }
	.workspace-hero p, .quiet { margin: 0; color: var(--dt2); font-size: .85rem; line-height: 1.5; }
	.actions { display: flex; gap: 8px; flex-wrap: wrap; }
	button { display: inline-flex; align-items: center; justify-content: center; gap: 7px; min-height: 34px; padding: 7px 11px; border: 1px solid var(--dbd); border-radius: 7px; background: transparent; color: var(--dt2); font: inherit; font-size: .8rem; font-weight: 680; cursor: pointer; }
	button { -webkit-user-select: none; user-select: none; }
	button:hover { background: color-mix(in srgb, var(--dt) 5%, transparent); color: var(--dt); }
	button.primary { border-color: var(--dt); background: var(--dt); color: var(--dbg); }
	.metrics { display: grid; grid-template-columns: repeat(4, minmax(0, 1fr)); border-bottom: 1px solid var(--dbd); }
	.metrics div { display: grid; grid-template-columns: auto 1fr auto; align-items: center; gap: 8px; min-height: 54px; padding: 10px 16px; border-right: 1px solid var(--dbd); color: var(--dt3); }
	.metrics div:last-child { border-right: 0; }
	.metrics span { font-size: .76rem; }
	.metrics strong { color: var(--dt); font-size: .95rem; }
	.overview-grid { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 12px; padding: 16px; }
	.panel { min-width: 0; border: 1px solid var(--dbd); border-radius: 8px; background: color-mix(in srgb, var(--dt) 2%, var(--dbg)); overflow: hidden; }
	.panel header { display: flex; justify-content: space-between; gap: 12px; padding: 14px; border-bottom: 1px solid var(--dbd); }
	.panel h3 { margin: 3px 0 0; color: var(--dt); font-size: .95rem; letter-spacing: 0; }
	.panel header > strong { color: var(--dt); font-size: 1.05rem; }
	.rows { display: grid; }
	.rows button { display: grid; justify-content: stretch; gap: 4px; min-width: 0; padding: 11px 14px; border: 0; border-bottom: 1px solid var(--dbd); border-radius: 0; text-align: left; }
	.rows button:last-child { border-bottom: 0; }
	.rows strong { overflow: hidden; color: var(--dt); font-size: .82rem; text-overflow: ellipsis; white-space: nowrap; }
	.rows small { overflow: hidden; color: var(--dt3); font-size: .72rem; text-overflow: ellipsis; white-space: nowrap; }
	.stage, .date { color: #0f766e; font-size: .66rem; font-weight: 750; text-transform: capitalize; }
	.quiet { padding: 18px 14px; }
	.inventory { padding-bottom: 14px; }
	.inventory-group { display: grid; gap: 8px; padding: 13px 14px 0; }
	.inventory-group div { display: flex; flex-wrap: wrap; gap: 6px; }
	.inventory-group b { padding: 5px 8px; border: 1px solid var(--dbd); border-radius: 6px; color: var(--dt2); font-size: .72rem; font-weight: 650; }
	.empty { display: grid; justify-items: center; gap: 9px; padding: 76px 24px; color: var(--dt3); text-align: center; }
	.empty h3 { margin: 0; color: var(--dt); font-size: 1rem; }
	.empty p { max-width: 460px; margin: 0 0 6px; color: var(--dt2); font-size: .83rem; line-height: 1.5; }
	@media (max-width: 800px) { .workspace-hero { flex-direction: column; } .metrics { grid-template-columns: repeat(2, 1fr); } .metrics div:nth-child(2) { border-right: 0; } .metrics div:nth-child(-n+2) { border-bottom: 1px solid var(--dbd); } .overview-grid { grid-template-columns: 1fr; } }
</style>
