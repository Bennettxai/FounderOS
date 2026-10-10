<script lang="ts">
	import { onMount } from 'svelte';
	import { currentWorkspace } from '$lib/stores/workspaces';
	import {
		getDeliverables,
		createDeliverable,
		updateDeliverable,
		deleteDeliverable,
		type Deliverable,
		type DeliverableStatus,
		type DeliverableKind
	} from '$lib/api/deliverables';
	import {
		Package,
		Plus,
		Loader2,
		X,
		Search,
		Trash2,
		Pencil,
		ExternalLink
	} from 'lucide-svelte';

	// Production lifecycle. Order here is the flow a deliverable moves through.
	const STATUSES: { value: DeliverableStatus; label: string }[] = [
		{ value: 'draft', label: 'Draft' },
		{ value: 'in_progress', label: 'In Progress' },
		{ value: 'delivered', label: 'Delivered' }
	];

	const KINDS: { value: DeliverableKind; label: string }[] = [
		{ value: 'package', label: 'Package' },
		{ value: 'document', label: 'Document' },
		{ value: 'deck', label: 'Deck' },
		{ value: 'script', label: 'Script' },
		{ value: 'report', label: 'Report' },
		{ value: 'video', label: 'Video' },
		{ value: 'other', label: 'Other' }
	];
	const kindLabel = (k: string) => KINDS.find((x) => x.value === k)?.label ?? k;

	let deliverables = $state<Deliverable[]>([]);
	let loading = $state(true);
	let error = $state<string | null>(null);
	let query = $state('');

	let showEdit = $state(false);
	let saving = $state(false);
	let editing = $state<Deliverable | null>(null);
	let form = $state({
		title: '',
		kind: 'package' as DeliverableKind,
		status: 'draft' as DeliverableStatus,
		client: '',
		project: '',
		link: '',
		description: ''
	});

	let wsId = $state<string | null | undefined>(undefined);
	$effect(() => {
		const id = $currentWorkspace?.id ?? null;
		if (id !== wsId) {
			wsId = id;
			load();
		}
	});

	onMount(load);

	let searchTimer: ReturnType<typeof setTimeout>;
	function onSearch() {
		clearTimeout(searchTimer);
		searchTimer = setTimeout(load, 250);
	}

	function clearSearch() {
		query = '';
		clearTimeout(searchTimer);
		load();
	}

	async function load() {
		loading = true;
		error = null;
		try {
			const res = await getDeliverables(query.trim() || undefined);
			deliverables = res.deliverables ?? [];
		} catch (e) {
			// The deliverables backend endpoint is not built yet. Treat a missing
			// endpoint as an empty catalog rather than surfacing a hard error, so
			// the module reads clean until the API lands. Real failures still show.
			const msg = e instanceof Error ? e.message : 'Failed to load deliverables';
			if (msg.includes('HTTP 404') || msg.includes('HTTP 501')) {
				deliverables = [];
			} else {
				error = msg;
			}
		} finally {
			loading = false;
		}
	}

	// Group deliverables by lifecycle status in canonical order.
	const grouped = $derived.by(() =>
		STATUSES.map((s) => ({
			status: s,
			items: deliverables.filter((d) => (d.status || 'draft') === s.value)
		})).filter((g) => g.items.length > 0)
	);

	function openNew() {
		editing = null;
		form = {
			title: '',
			kind: 'package',
			status: 'draft',
			client: '',
			project: '',
			link: '',
			description: ''
		};
		showEdit = true;
	}

	function openEdit(d: Deliverable) {
		editing = d;
		form = {
			title: d.title,
			kind: d.kind ?? 'package',
			status: d.status ?? 'draft',
			client: d.client ?? '',
			project: d.project ?? '',
			link: d.link ?? '',
			description: d.description ?? ''
		};
		showEdit = true;
	}

	async function save(e: Event) {
		e.preventDefault();
		if (!form.title.trim()) return;
		saving = true;
		error = null;
		try {
			const body = {
				title: form.title.trim(),
				kind: form.kind,
				status: form.status,
				client: form.client.trim(),
				project: form.project.trim(),
				link: form.link.trim(),
				description: form.description.trim()
			};
			if (editing) await updateDeliverable(editing.id, body);
			else await createDeliverable(body);
			showEdit = false;
			await load();
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to save deliverable';
		} finally {
			saving = false;
		}
	}

	async function remove(d: Deliverable) {
		if (!confirm(`Delete "${d.title}"?`)) return;
		try {
			await deleteDeliverable(d.id);
			deliverables = deliverables.filter((x) => x.id !== d.id);
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to delete';
		}
	}
</script>

<svelte:head><title>Deliverables - BusinessOS</title></svelte:head>

<div class="dl-root">
	<header class="topbar">
		<div class="title-wrap">
			<h1>Deliverables</h1>
			<span class="count">{deliverables.length}</span>
		</div>
		<div class="tools">
			<div class="search">
				<Search size={15} />
				<input bind:value={query} oninput={onSearch} placeholder="Search deliverables…" />
			</div>
			<button class="btn btn--primary" onclick={openNew}>
				<Plus size={16} strokeWidth={2.4} />New deliverable
			</button>
		</div>
	</header>

	{#if error}<div class="banner banner--error">{error}</div>{/if}

	{#if loading}
		<div class="loading"><Loader2 class="spin" size={20} /> Loading deliverables…</div>
	{:else if deliverables.length === 0 && query.trim()}
		<div class="empty">
			<Search size={26} strokeWidth={1.4} />
			<p>No deliverables match "{query.trim()}".</p>
			<button class="btn btn--ghost" onclick={clearSearch}>Clear search</button>
		</div>
	{:else if deliverables.length === 0}
		<div class="empty">
			<Package size={26} strokeWidth={1.4} />
			<p>
				No deliverables yet. Track the documents, decks, and packages you produce and send so
				every client engagement has a record of what was delivered.
			</p>
			<button class="btn btn--primary" onclick={openNew}>
				<Plus size={16} />Add your first deliverable
			</button>
		</div>
	{:else}
		<div class="list">
			{#each grouped as g (g.status.value)}
				<div class="cat">
					<div class="cat-head">
						{g.status.label}<span class="cat-count">{g.items.length}</span>
					</div>
					{#each g.items as d (d.id)}
						<div class="deliverable">
							<div class="deliverable-main">
								<div class="deliverable-head">
									<span class="deliverable-title">{d.title}</span>
									<span class="kind">{kindLabel(d.kind)}</span>
									<span class="status status--{d.status}">
										{STATUSES.find((s) => s.value === d.status)?.label ?? d.status}
									</span>
								</div>
								{#if d.client || d.project}
									<div class="meta">
										{#if d.client}<span class="meta-item">{d.client}</span>{/if}
										{#if d.client && d.project}<span class="meta-sep">·</span>{/if}
										{#if d.project}<span class="meta-item meta-item--muted">{d.project}</span>{/if}
									</div>
								{/if}
								{#if d.description}<div class="deliverable-desc">{d.description}</div>{/if}
								{#if d.link}
									<a class="deliverable-link" href={d.link} target="_blank" rel="noopener noreferrer">
										<ExternalLink size={13} />Open
									</a>
								{/if}
							</div>
							<div class="deliverable-actions">
								<button class="icon-btn" title="Edit" onclick={() => openEdit(d)}>
									<Pencil size={14} />
								</button>
								<button class="icon-btn icon-btn--danger" title="Delete" onclick={() => remove(d)}>
									<Trash2 size={14} />
								</button>
							</div>
						</div>
					{/each}
				</div>
			{/each}
		</div>
	{/if}
</div>

{#if showEdit}
	<div
		class="overlay"
		role="button"
		tabindex="0"
		onclick={() => (showEdit = false)}
		onkeydown={(e) => e.key === 'Escape' && (showEdit = false)}
	>
		<div
			class="modal"
			role="dialog"
			tabindex="-1"
			onclick={(e) => e.stopPropagation()}
			onkeydown={() => {}}
		>
			<div class="modal-head">
				<h2>{editing ? 'Edit deliverable' : 'New deliverable'}</h2>
				<button class="card-x" onclick={() => (showEdit = false)} aria-label="Close">
					<X size={18} />
				</button>
			</div>
			<form onsubmit={save}>
				<label class="field">
					<span>Title</span>
					<input bind:value={form.title} placeholder="e.g. Q3 Growth Audit Package" required />
				</label>
				<div class="field-row">
					<label class="field">
						<span>Kind</span>
						<select bind:value={form.kind}>
							{#each KINDS as k}<option value={k.value}>{k.label}</option>{/each}
						</select>
					</label>
					<label class="field">
						<span>Status</span>
						<select bind:value={form.status}>
							{#each STATUSES as s}<option value={s.value}>{s.label}</option>{/each}
						</select>
					</label>
				</div>
				<div class="field-row">
					<label class="field">
						<span>Client</span>
						<input bind:value={form.client} placeholder="e.g. Better Stem" />
					</label>
					<label class="field">
						<span>Project</span>
						<input bind:value={form.project} placeholder="e.g. Phase 2 Retainer" />
					</label>
				</div>
				<label class="field">
					<span>Link</span>
					<input bind:value={form.link} placeholder="Drive folder, doc, or deployed page URL" />
				</label>
				<label class="field">
					<span>Description</span>
					<textarea bind:value={form.description} rows="3" placeholder="What this deliverable is…"
					></textarea>
				</label>
				<div class="modal-actions">
					<button type="button" class="btn btn--ghost" onclick={() => (showEdit = false)}>
						Cancel
					</button>
					<button type="submit" class="btn btn--primary" disabled={saving || !form.title.trim()}>
						{#if saving}<Loader2 class="spin" size={15} />{/if}{editing ? 'Save' : 'Add deliverable'}
					</button>
				</div>
			</form>
		</div>
	</div>
{/if}

<style>
	.dl-root {
		height: 100%;
		display: flex;
		flex-direction: column;
		background: var(--dbg);
		color: var(--dt);
		overflow: hidden;
	}
	.topbar {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 16px;
		padding: 16px 24px;
		border-bottom: 1px solid var(--dbd);
		flex-shrink: 0;
	}
	.title-wrap {
		display: flex;
		align-items: center;
		gap: 10px;
	}
	.title-wrap h1 {
		font-size: 1.15rem;
		font-weight: 680;
		letter-spacing: -0.02em;
		margin: 0;
	}
	.count {
		font-size: 0.74rem;
		color: var(--dt3);
		background: color-mix(in srgb, var(--dt) 8%, transparent);
		padding: 2px 9px;
		border-radius: 999px;
	}
	.tools {
		display: flex;
		align-items: center;
		gap: 10px;
	}
	.search {
		display: flex;
		align-items: center;
		gap: 7px;
		padding: 7px 11px;
		border: 1px solid var(--dbd);
		border-radius: 9px;
		color: var(--dt3);
	}
	.search input {
		background: transparent;
		border: none;
		outline: none;
		color: var(--dt);
		font-size: 0.84rem;
		width: 180px;
		font-family: inherit;
	}
	.btn {
		display: inline-flex;
		align-items: center;
		gap: 7px;
		padding: 8px 15px;
		border-radius: 9px;
		font-size: 0.83rem;
		font-weight: 580;
		cursor: pointer;
		border: 1px solid transparent;
		white-space: nowrap;
	}
	.btn--primary {
		background: var(--dt);
		color: var(--dbg);
	}
	.btn--ghost {
		background: transparent;
		border-color: var(--dbd);
		color: var(--dt2);
	}
	.btn:disabled {
		opacity: 0.55;
		cursor: not-allowed;
	}
	.loading,
	.empty {
		flex: 1;
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		gap: 14px;
		color: var(--dt3);
		text-align: center;
		padding: 0 24px;
	}
	.empty p {
		max-width: 420px;
		line-height: 1.5;
	}
	.banner {
		margin: 14px 24px 0;
		padding: 11px 14px;
		border-radius: 10px;
		font-size: 0.83rem;
	}
	.banner--error {
		background: color-mix(in srgb, #ef4444 12%, transparent);
		color: #ef4444;
	}
	.list {
		flex: 1;
		overflow-y: auto;
		padding: 18px 24px;
		max-width: 820px;
		width: 100%;
		margin: 0 auto;
	}
	.cat {
		margin-bottom: 24px;
	}
	.cat-head {
		display: flex;
		align-items: center;
		gap: 8px;
		font-size: 0.74rem;
		font-weight: 640;
		color: var(--dt3);
		text-transform: uppercase;
		letter-spacing: 0.06em;
		margin-bottom: 10px;
		padding-bottom: 7px;
		border-bottom: 1px solid var(--dbd);
	}
	.cat-count {
		font-size: 0.68rem;
		font-weight: 600;
		color: var(--dt3);
		background: color-mix(in srgb, var(--dt) 8%, transparent);
		padding: 1px 7px;
		border-radius: 999px;
		text-transform: none;
		letter-spacing: 0;
	}
	.deliverable {
		display: flex;
		align-items: flex-start;
		gap: 14px;
		padding: 16px;
		border: 1px solid var(--dbd);
		border-radius: 12px;
		margin-bottom: 12px;
	}
	.deliverable-main {
		flex: 1;
		min-width: 0;
	}
	.deliverable-head {
		display: flex;
		align-items: center;
		gap: 10px;
		flex-wrap: wrap;
	}
	.deliverable-title {
		font-size: 0.96rem;
		font-weight: 640;
	}
	.kind {
		font-size: 0.68rem;
		font-weight: 600;
		color: var(--dt3);
		background: color-mix(in srgb, var(--dt) 8%, transparent);
		padding: 2px 8px;
		border-radius: 999px;
	}
	.status {
		font-size: 0.68rem;
		font-weight: 620;
		text-transform: uppercase;
		letter-spacing: 0.05em;
		padding: 2px 8px;
		border-radius: 999px;
	}
	.status--delivered {
		background: color-mix(in srgb, #22c55e 16%, transparent);
		color: #22c55e;
	}
	.status--in_progress {
		background: color-mix(in srgb, #3b82f6 15%, transparent);
		color: #3b82f6;
	}
	.status--draft {
		background: color-mix(in srgb, var(--dt) 10%, transparent);
		color: var(--dt3);
	}
	.meta {
		display: flex;
		align-items: center;
		gap: 7px;
		font-size: 0.8rem;
		color: var(--dt2);
		margin-top: 7px;
		flex-wrap: wrap;
	}
	.meta-item--muted {
		color: var(--dt3);
	}
	.meta-sep {
		color: var(--dt3);
	}
	.deliverable-desc {
		font-size: 0.84rem;
		color: var(--dt2);
		line-height: 1.55;
		margin-top: 7px;
		white-space: pre-wrap;
	}
	.deliverable-link {
		display: inline-flex;
		align-items: center;
		gap: 5px;
		font-size: 0.78rem;
		font-weight: 560;
		color: var(--dt2);
		text-decoration: none;
		margin-top: 10px;
	}
	.deliverable-link:hover {
		color: var(--dt);
	}
	.deliverable-actions {
		display: flex;
		gap: 4px;
		flex-shrink: 0;
		opacity: 0;
		transition: opacity 0.12s;
	}
	.deliverable:hover .deliverable-actions {
		opacity: 1;
	}
	.icon-btn {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		width: 28px;
		height: 28px;
		border-radius: 7px;
		border: none;
		background: transparent;
		color: var(--dt3);
		cursor: pointer;
	}
	.icon-btn:hover {
		background: color-mix(in srgb, var(--dt) 8%, transparent);
		color: var(--dt);
	}
	.icon-btn--danger:hover {
		background: color-mix(in srgb, #ef4444 12%, transparent);
		color: #ef4444;
	}
	.overlay {
		position: fixed;
		inset: 0;
		background: rgba(0, 0, 0, 0.55);
		display: flex;
		align-items: center;
		justify-content: center;
		z-index: 100;
		padding: 20px;
	}
	.modal {
		width: 100%;
		max-width: 480px;
		max-height: 90vh;
		overflow-y: auto;
		background: var(--dbg);
		border: 1px solid var(--dbd);
		border-radius: 16px;
		padding: 22px;
		box-shadow: 0 24px 60px rgba(0, 0, 0, 0.5);
	}
	.modal-head {
		display: flex;
		align-items: center;
		justify-content: space-between;
		margin-bottom: 18px;
	}
	.modal-head h2 {
		font-size: 1.05rem;
		font-weight: 640;
		margin: 0;
	}
	.card-x {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		width: 28px;
		height: 28px;
		border-radius: 7px;
		border: none;
		background: transparent;
		color: var(--dt3);
		cursor: pointer;
	}
	.card-x:hover {
		background: color-mix(in srgb, var(--dt) 8%, transparent);
	}
	.field {
		display: flex;
		flex-direction: column;
		gap: 6px;
		margin-bottom: 14px;
	}
	.field span {
		font-size: 0.8rem;
		font-weight: 560;
		color: var(--dt2);
	}
	.field input,
	.field textarea,
	.field select {
		padding: 9px 12px;
		border-radius: 9px;
		border: 1px solid var(--dbd);
		background: var(--dbg);
		color: var(--dt);
		font-size: 0.86rem;
		outline: none;
		font-family: inherit;
		resize: vertical;
	}
	.field input:focus,
	.field textarea:focus,
	.field select:focus {
		border-color: color-mix(in srgb, var(--dt) 40%, transparent);
	}
	.field-row {
		display: flex;
		gap: 12px;
	}
	.field-row .field {
		flex: 1;
	}
	.modal-actions {
		display: flex;
		justify-content: flex-end;
		gap: 10px;
		margin-top: 6px;
	}
	:global(.spin) {
		animation: spin 0.9s linear infinite;
	}
	@keyframes spin {
		to {
			transform: rotate(360deg);
		}
	}
	@media (max-width: 768px) {
		.topbar {
			padding: 12px 16px;
		}
		.list {
			padding: 14px 16px;
		}
	}
</style>
