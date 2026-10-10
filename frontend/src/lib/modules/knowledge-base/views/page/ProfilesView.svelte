<script lang="ts">
	import { Building2, FileText, Folder, FolderKanban, Users } from 'lucide-svelte';
	import type { DocumentMeta, SidebarView } from '../../entities/types';
	import PageListLayout from './PageListLayout.svelte';

	export interface PagesNodeInfo {
		slug: string;
		name: string;
		type: string;
		signal_count: number;
		has_signals?: boolean;
	}

	interface Props {
		view: SidebarView;
		nodes: PagesNodeInfo[];
		documents: DocumentMeta[];
		onOpenDocument: (id: string) => void;
	}

	let { view, nodes, documents, onOpenDocument }: Props = $props();

	const title = $derived.by(() => {
		if (view === 'profiles-business') return 'Businesses';
		if (view === 'profiles-project') return 'Projects';
		return 'People';
	});

	function profileKind(node: PagesNodeInfo): 'business' | 'person' | 'other' {
		const type = node.type.toLowerCase();
		if (['business', 'company', 'organization', 'organisation'].includes(type)) return 'business';
		if (['person', 'people', 'team', 'member'].includes(type)) return 'person';
		return 'other';
	}

	const visibleNodes = $derived.by(() => {
		if (view === 'profiles-business') {
			return nodes.filter((node) => profileKind(node) === 'business');
		}
		if (view === 'profiles-project') {
			return nodes;
		}
		return nodes.filter((node) => profileKind(node) === 'person');
	});

	function getProjectItems(node: PagesNodeInfo) {
		return documents.filter((doc) =>
			doc.id.startsWith(`${node.slug}/`) && (
				doc.id.toLowerCase().includes('project') ||
				doc.id.toLowerCase().includes('autonomo') ||
				doc.id.toLowerCase().includes('launch') ||
				doc.id.toLowerCase().includes('pipeline')
			)
		);
	}

	function getNodeChildren(node: PagesNodeInfo) {
		return documents.filter((doc) => {
			if (view === 'profiles' || view === 'profiles-person') {
				return doc.parent_id === node.slug && (doc.type === 'folder' || doc.id.endsWith('.md'));
			}
			return doc.parent_id === node.slug;
		});
	}
</script>

<PageListLayout {title}>
	<div class="kb-profiles">
		{#each visibleNodes as node (node.slug)}
			{@const projectItems = view === 'profiles-project' ? getProjectItems(node) : []}
			{#if view !== 'profiles-project' || projectItems.length > 0}
				<section class="kb-profiles__section">
					{#if view === 'profiles-project'}
						<div class="kb-profiles__node-header">
							<div class="kb-profiles__node-info">
								<span class="kb-profiles__node-slug">{node.slug.split('-')[0]}</span>
								<span class="kb-profiles__node-name">{node.name}</span>
							</div>
							<span class="kb-profiles__count">{projectItems.length}</span>
						</div>
					{:else}
						<button class="kb-profiles__node-header" onclick={() => onOpenDocument(`${node.slug}/context.md`)}>
							<div class="kb-profiles__node-info">
								<span class="kb-profiles__node-slug">{node.slug.split('-')[0]}</span>
								<span class="kb-profiles__node-name">{node.name}</span>
							</div>
							{#if view === 'profiles-business'}
								<span class="kb-profiles__type">{node.type}</span>
							{/if}
						</button>
					{/if}

					<div class="kb-profiles__children">
						{#each view === 'profiles-project' ? projectItems : getNodeChildren(node) as child (child.id)}
							<button
								class="kb-profiles__child"
								class:kb-profiles__child--folder={child.type === 'folder'}
								onclick={() => onOpenDocument(child.id)}
							>
								<span class="kb-profiles__icon">
									{#if child.type === 'folder'}
										{#if view === 'profiles-business'}
											<Building2 size={15} />
										{:else if view === 'profiles-project'}
											<FolderKanban size={15} />
										{:else}
											<Users size={15} />
										{/if}
									{:else}
										<FileText size={15} />
									{/if}
								</span>
								<span class="kb-profiles__child-title">{child.title}</span>
								{#if child.children_count > 0}
									<span class="kb-profiles__count">{child.children_count}</span>
								{/if}
							</button>
						{/each}
					</div>
				</section>
			{/if}
		{:else}
			<div class="kb-profiles__empty">No profiles found</div>
		{/each}
	</div>
</PageListLayout>

<style>
	.kb-profiles {
		display: flex;
		flex-direction: column;
		gap: 8px;
	}

	.kb-profiles__section {
		border-radius: 8px;
		overflow: hidden;
	}

	.kb-profiles__node-header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		width: 100%;
		padding: 10px 12px;
		border: none;
		background: var(--dbg2, rgba(0,0,0,0.03));
		border-left: 3px solid var(--dbd, rgba(0,0,0,0.1));
		color: inherit;
		text-align: left;
	}

	button.kb-profiles__node-header {
		cursor: pointer;
	}

	button.kb-profiles__node-header:hover {
		background: var(--dbg3, rgba(0,0,0,0.06));
		border-left-color: rgba(59,130,246,0.5);
	}

	.kb-profiles__node-info {
		display: flex;
		align-items: center;
		gap: 10px;
		min-width: 0;
	}

	.kb-profiles__node-slug {
		font-size: 11px;
		font-family: ui-monospace, monospace;
		color: var(--dt4, #aaa);
		min-width: 20px;
	}

	.kb-profiles__node-name {
		font-size: 13px;
		font-weight: 500;
		color: var(--dt, inherit);
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	.kb-profiles__children {
		display: flex;
		flex-direction: column;
		padding-left: 32px;
		gap: 1px;
	}

	.kb-profiles__child {
		display: flex;
		align-items: center;
		gap: 8px;
		width: 100%;
		padding: 6px 10px;
		border: none;
		background: transparent;
		color: var(--dt2, #666);
		font-size: 12px;
		cursor: pointer;
		border-radius: 4px;
		text-align: left;
	}

	.kb-profiles__child:hover {
		background: var(--dbg2, rgba(0,0,0,0.04));
		color: var(--dt, inherit);
	}

	.kb-profiles__child--folder {
		color: var(--dt2, #555);
	}

	.kb-profiles__icon {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		width: 18px;
		flex-shrink: 0;
		color: var(--dt3, #999);
	}

	.kb-profiles__child-title {
		flex: 1;
		min-width: 0;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	.kb-profiles__count,
	.kb-profiles__type {
		font-size: 10px;
		color: var(--dt3, #999);
		background: var(--dbg2, rgba(0,0,0,0.05));
		padding: 2px 6px;
		border-radius: 4px;
		white-space: nowrap;
	}

	.kb-profiles__empty {
		padding: 64px 32px;
		text-align: center;
		color: var(--bos-v2-text-secondary, #8e8d91);
		font-size: 14px;
	}
</style>
