<script lang="ts">
	import OrgNode from './OrgNode.svelte';

	type Status = 'available' | 'busy' | 'overloaded' | 'ooo';

	interface OrgMember {
		id: string;
		name: string;
		role: string;
		avatar?: string;
		status: Status;
		managerId?: string | null;
	}

	interface Props {
		members: OrgMember[];
		onMemberClick?: (memberId: string) => void;
	}

	let { members, onMemberClick }: Props = $props();

	let scale = $state(1);
	let panX = $state(0);
	let panY = $state(0);
	let isDragging = $state(false);
	let startX = 0;
	let startY = 0;

	// Build org tree structure
	interface OrgTreeNode extends OrgMember {
		children: OrgTreeNode[];
		depth: number;
	}

	const orgTree = $derived((): OrgTreeNode[] => {
		const memberMap = new Map<string, OrgTreeNode>();
		const roots: OrgTreeNode[] = [];

		// Initialize all members
		members.forEach(m => {
			memberMap.set(m.id, { ...m, children: [], depth: 0 });
		});

		// Build tree
		memberMap.forEach((node) => {
			if (node.managerId && memberMap.has(node.managerId)) {
				const parent = memberMap.get(node.managerId)!;
				node.depth = parent.depth + 1;
				parent.children.push(node);
			} else {
				roots.push(node);
			}
		});

		// Update depths recursively
		function updateDepths(node: OrgTreeNode, depth: number) {
			node.depth = depth;
			node.children.forEach(child => updateDepths(child, depth + 1));
		}
		roots.forEach(root => updateDepths(root, 0));

		return roots;
	});

	function zoomIn() {
		scale = Math.min(scale + 0.2, 2);
	}

	function zoomOut() {
		scale = Math.max(scale - 0.2, 0.5);
	}

	function resetView() {
		scale = 1;
		panX = 0;
		panY = 0;
	}

	function handleMouseDown(e: MouseEvent) {
		if (e.button === 0) {
			isDragging = true;
			startX = e.clientX - panX;
			startY = e.clientY - panY;
		}
	}

	function handleMouseMove(e: MouseEvent) {
		if (isDragging) {
			panX = e.clientX - startX;
			panY = e.clientY - startY;
		}
	}

	function handleMouseUp() {
		isDragging = false;
	}

	function handleWheel(e: WheelEvent) {
		e.preventDefault();
		const delta = e.deltaY > 0 ? -0.1 : 0.1;
		scale = Math.min(Math.max(scale + delta, 0.5), 2);
	}
</script>

<div class="td-orgchart">
	<!-- Org Chart Canvas -->
	<div
		class="td-orgchart__canvas"
		onmousedown={handleMouseDown}
		onmousemove={handleMouseMove}
		onmouseup={handleMouseUp}
		onmouseleave={handleMouseUp}
		onwheel={handleWheel}
		role="application"
		aria-label="Organization chart"
	>
		<div
			class="td-orgchart__inner"
			style="transform: translate({panX}px, {panY}px) scale({scale}); transform-origin: center top;"
		>
			{#if orgTree().length === 0}
				<div class="td-orgchart__empty">
					<div class="td-orgchart__empty-icon">
						<svg class="w-8 h-8" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M19 21V5a2 2 0 00-2-2H7a2 2 0 00-2 2v16m14 0h2m-2 0h-5m-9 0H3m2 0h5M9 7h1m-1 4h1m4-4h1m-1 4h1m-5 10v-5a1 1 0 011-1h2a1 1 0 011 1v5m-4 0h4" />
						</svg>
					</div>
					<h3 class="td-orgchart__empty-title">No org structure</h3>
					<p class="td-orgchart__empty-text">Add team members and set reporting relationships</p>
				</div>
			{:else}
				<!-- Render org tree -->
				<div class="td-orgchart__tree">
					{#each orgTree() as root (root.id)}
						{@render orgBranch(root)}
					{/each}
				</div>
			{/if}
		</div>
	</div>

	<!-- Controls -->
	<div class="td-orgchart__controls">
		<div class="flex items-center gap-2">
			<button
				onclick={zoomIn}
				class="btn-pill btn-pill-ghost w-8 h-8 flex items-center justify-center"
				title="Zoom in"
				aria-label="Zoom in"
			>
				<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0zM10 7v3m0 0v3m0-3h3m-3 0H7" />
				</svg>
			</button>
			<button
				onclick={zoomOut}
				class="btn-pill btn-pill-ghost w-8 h-8 flex items-center justify-center"
				title="Zoom out"
				aria-label="Zoom out"
			>
				<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0zM13 10H7" />
				</svg>
			</button>
			<button
				onclick={resetView}
				class="btn-pill btn-pill-ghost btn-pill-sm"
				aria-label="Reset zoom"
			>
				Reset
			</button>
			<span class="td-orgchart__scale">{Math.round(scale * 100)}%</span>
		</div>

		<button
			class="btn-pill btn-pill-ghost btn-pill-sm flex items-center gap-1"
		>
			<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 8V4m0 0h4M4 4l5 5m11-1V4m0 0h-4m4 0l-5 5M4 16v4m0 0h4m-4 0l5-5m11 5l-5-5m5 5v-4m0 4h-4" />
			</svg>
			Full Screen
		</button>
	</div>
</div>

{#snippet orgBranch(node: OrgTreeNode)}
	<div class="td-org__branch">
		<OrgNode
			id={node.id}
			name={node.name}
			role={node.role}
			avatar={node.avatar}
			status={node.status}
			depth={node.depth}
			onClick={() => onMemberClick?.(node.id)}
		/>

		{#if node.children.length > 0}
			<div class="td-org__v-line"></div>

			{#if node.children.length > 1}
				<div class="td-org__h-line"></div>
			{/if}

			<div class="td-org__level">
				{#each node.children as child (child.id)}
					<div class="td-org__branch">
						{#if node.children.length > 1}
							<div class="td-org__v-line"></div>
						{/if}
						{@render orgBranch(child)}
					</div>
				{/each}
			</div>
		{/if}
	</div>
{/snippet}

<style>
	/* ── Org Chart Layout ── */
	.td-orgchart {
		flex: 1;
		display: flex;
		flex-direction: column;
		overflow: hidden;
	}
	.td-orgchart__canvas {
		flex: 1;
		overflow: auto;
		background: var(--dbg2);
		cursor: grab;
		padding: 2rem;
	}
	.td-orgchart__canvas:active {
		cursor: grabbing;
	}
	.td-orgchart__inner {
		display: flex;
		justify-content: center;
		min-width: max-content;
		transition: transform 100ms ease;
	}
	.td-orgchart__tree {
		display: flex;
		flex-direction: column;
		align-items: center;
	}

	/* ── Controls Bar ── */
	.td-orgchart__controls {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 0.75rem 1rem;
		background: var(--dbg);
		border-top: 1px solid var(--dbd);
		color: var(--dt3);
	}
	.td-orgchart__scale {
		font-size: 0.875rem;
		color: var(--dt4);
		margin-left: 0.5rem;
	}

	/* ── Empty State ── */
	.td-orgchart__empty {
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		padding: 4rem 0;
	}
	.td-orgchart__empty-icon {
		width: 4rem;
		height: 4rem;
		border-radius: 50%;
		background: var(--dbg3);
		display: flex;
		align-items: center;
		justify-content: center;
		margin-bottom: 1rem;
		color: var(--dt4);
	}
	.td-orgchart__empty-title {
		font-size: 1.125rem;
		font-weight: 600;
		color: var(--dt);
		margin: 0 0 0.25rem;
	}
	.td-orgchart__empty-text {
		font-size: 0.875rem;
		color: var(--dt3);
		margin: 0;
	}

	/* ── Org Tree Lines ── */
	.td-org__branch {
		display: flex;
		flex-direction: column;
		align-items: center;
		flex: 1;
		min-width: 150px;
	}
	.td-org__v-line {
		width: 2px;
		height: 18px;
		background: var(--dbd2, #f0f0f0);
		margin: 0 auto;
	}
	.td-org__h-line {
		width: 65%;
		height: 2px;
		background: var(--dbd2, #f0f0f0);
		margin: 0 auto;
	}
	.td-org__level {
		display: flex;
		align-items: flex-start;
		justify-content: center;
		gap: 12px;
		width: 100%;
	}
</style>
