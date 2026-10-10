<script lang="ts">
	import { onMount } from 'svelte';
	import { currentWorkspace } from '$lib/stores/workspaces';
	import {
		getResources,
		createResource,
		updateResource,
		deleteResource,
		type Resource
	} from '$lib/api/resources';
	import {
		Library,
		Plus,
		Loader2,
		X,
		Search,
		Trash2,
		Pencil,
		ExternalLink,
		Link2
	} from 'lucide-svelte';

	let resources = $state<Resource[]>([]);
	let loading = $state(true);
	let error = $state<string | null>(null);
	let query = $state('');

	// Add / edit modal (shared form).
	let showForm = $state(false);
	let saving = $state(false);
	let editing = $state<Resource | null>(null);
	let form = $state({ title: '', url: '', category: '', notes: '' });

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
			const res = await getResources(query.trim() || undefined);
			resources = res.resources;
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to load resources';
		} finally {
			loading = false;
		}
	}

	function openAdd() {
		editing = null;
		form = { title: '', url: '', category: '', notes: '' };
		showForm = true;
	}
	function openEdit(r: Resource) {
		editing = r;
		form = { title: r.title, url: r.url, category: r.category, notes: r.notes };
		showForm = true;
	}

	async function save(e: Event) {
		e.preventDefault();
		if (!form.title.trim()) return;
		saving = true;
		error = null;
		try {
			const payload = {
				title: form.title.trim(),
				url: form.url.trim(),
				category: form.category.trim(),
				notes: form.notes
			};
			if (editing) {
				await updateResource(editing.id, payload);
			} else {
				await createResource(payload);
			}
			showForm = false;
			editing = null;
			await load();
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to save resource';
		} finally {
			saving = false;
		}
	}

	async function remove(r: Resource) {
		if (!confirm(`Delete "${r.title}"? This can't be undone.`)) return;
		try {
			await deleteResource(r.id);
			resources = resources.filter((x) => x.id !== r.id);
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to delete';
		}
	}

	function hostOf(url: string): string {
		try {
			return new URL(url).hostname.replace(/^www\./, '');
		} catch {
			return url;
		}
	}
</script>

<svelte:head><title>Resources - BusinessOS</title></svelte:head>

<div class="page">
	<header class="page-header">
		<div class="page-icon"><Library size={22} strokeWidth={1.8} /></div>
		<div class="head-text">
			<h1 class="page-title">Resources</h1>
			<p class="page-desc">A library of SOPs, guides, templates, and links your team relies on.</p>
		</div>
		<div class="head-actions">
			<button class="btn btn-primary" onclick={openAdd}><Plus size={15} /> Add resource</button>
		</div>
	</header>

	<div class="toolbar">
		<div class="search">
			<Search size={15} class="search-icon" />
			<input placeholder="Search title, url, category…" bind:value={query} oninput={onSearch} />
			{#if query}<button class="search-clear" onclick={clearSearch}><X size={13} /></button>{/if}
		</div>
		<span class="count">{resources.length} resource{resources.length === 1 ? '' : 's'}</span>
	</div>

	{#if error}<div class="error-bar">{error}</div>{/if}

	{#if loading}
		<div class="loading"><Loader2 size={22} class="spin" /> Loading resources…</div>
	{:else if resources.length === 0}
		<div class="empty-state">
			<Library size={40} strokeWidth={1.4} class="empty-icon" />
			<p class="empty-title">No resources saved</p>
			<p class="empty-body">Save SOPs, how-to guides, and reference links so your team can find them fast.</p>
			<div class="empty-actions">
				<button class="btn btn-primary" onclick={openAdd}><Plus size={15} /> Add a resource</button>
			</div>
		</div>
	{:else}
		<div class="grid">
			{#each resources as r (r.id)}
				<div class="card">
					<div class="card-head">
						<span class="card-title" title={r.title}>{r.title}</span>
						<div class="card-actions">
							<button class="icon-btn" title="Edit" onclick={() => openEdit(r)}><Pencil size={13} /></button>
							<button class="icon-btn danger" title="Delete" onclick={() => remove(r)}><Trash2 size={13} /></button>
						</div>
					</div>
					{#if r.category}<span class="cat-badge">{r.category}</span>{/if}
					{#if r.notes}<p class="card-notes">{r.notes}</p>{/if}
					{#if r.url}
						<a class="card-link" href={r.url} target="_blank" rel="noopener noreferrer">
							<Link2 size={13} /> {hostOf(r.url)} <ExternalLink size={12} />
						</a>
					{/if}
				</div>
			{/each}
		</div>
	{/if}
</div>

{#if showForm}
	<div class="modal-overlay" role="button" tabindex="0" onclick={(e) => { if (e.target === e.currentTarget) showForm = false; }} onkeydown={(e) => e.key === 'Escape' && (showForm = false)}>
		<form class="modal" onsubmit={save}>
			<div class="modal-head">
				<h2>{editing ? 'Edit resource' : 'Add a resource'}</h2>
				<button type="button" class="icon-btn" onclick={() => (showForm = false)}><X size={16} /></button>
			</div>
			<label class="field">
				<span>Title</span>
				<input bind:value={form.title} placeholder="Onboarding SOP" required />
			</label>
			<div class="field-row">
				<label class="field">
					<span>URL</span>
					<input bind:value={form.url} placeholder="https://…" />
				</label>
				<label class="field">
					<span>Category</span>
					<input bind:value={form.category} placeholder="SOP, guide, tool…" />
				</label>
			</div>
			<label class="field">
				<span>Notes</span>
				<textarea bind:value={form.notes} rows="3" placeholder="What this is for, who owns it…"></textarea>
			</label>
			<div class="modal-foot">
				<button type="button" class="btn btn-ghost" onclick={() => (showForm = false)}>Cancel</button>
				<button type="submit" class="btn btn-primary" disabled={saving}>
					{#if saving}<Loader2 size={15} class="spin" />{:else if editing}<Pencil size={15} />{:else}<Plus size={15} />{/if}
					{editing ? 'Save' : 'Add resource'}
				</button>
			</div>
		</form>
	</div>
{/if}

<style>
	.page { display: flex; flex-direction: column; gap: 20px; padding: 28px 32px; height: 100%; overflow-y: auto; }
	.page-header { display: flex; align-items: flex-start; gap: 14px; }
	.page-icon { width: 42px; height: 42px; border-radius: 10px; border: 1px solid var(--dbd); background: color-mix(in srgb, var(--dt) 5%, transparent); color: var(--dt2); display: flex; align-items: center; justify-content: center; flex-shrink: 0; }
	.head-text { flex: 1; min-width: 0; }
	.page-title { font-size: 1.25rem; font-weight: 650; color: var(--dt); letter-spacing: -0.02em; margin: 0; }
	.page-desc { font-size: 0.875rem; color: var(--dt3); margin: 2px 0 0; }
	.head-actions { display: flex; gap: 8px; flex-shrink: 0; }

	.btn { display: inline-flex; align-items: center; gap: 6px; padding: 8px 14px; border-radius: 8px; font-size: 0.84rem; font-weight: 550; cursor: pointer; border: 1px solid transparent; transition: background 0.12s, border-color 0.12s; }
	.btn-primary { background: var(--bos-v2-button-primary); color: var(--bos-v2-button-pureWhiteText); }
	.btn-primary:hover { opacity: 0.9; }
	.btn-ghost { background: transparent; border-color: var(--dbd); color: var(--dt2); }
	.btn-ghost:hover { background: color-mix(in srgb, var(--dt) 5%, transparent); }
	.btn:disabled { opacity: 0.6; cursor: default; }

	.toolbar { display: flex; align-items: center; justify-content: space-between; gap: 12px; }
	.search { position: relative; display: flex; align-items: center; flex: 1; max-width: 360px; }
	.search :global(.search-icon) { position: absolute; left: 10px; color: var(--dt3); }
	.search input { width: 100%; padding: 8px 30px; border-radius: 8px; border: 1px solid var(--dbd); background: transparent; color: var(--dt); font-size: 0.85rem; }
	.search-clear { position: absolute; right: 8px; background: none; border: none; color: var(--dt3); cursor: pointer; display: flex; }
	.count { font-size: 0.8rem; color: var(--dt3); }

	.error-bar { padding: 10px 14px; border-radius: 8px; background: color-mix(in srgb, #ef4444 12%, transparent); color: #ef4444; font-size: 0.83rem; }
	.loading { display: flex; align-items: center; gap: 8px; padding: 48px; justify-content: center; color: var(--dt3); font-size: 0.9rem; }

	.grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(260px, 1fr)); gap: 14px; }
	.card { border: 1px solid var(--dbd); border-radius: 12px; padding: 14px 16px; display: flex; flex-direction: column; gap: 8px; background: color-mix(in srgb, var(--dt) 2%, transparent); transition: border-color 0.12s; }
	.card:hover { border-color: color-mix(in srgb, var(--dt) 22%, transparent); }
	.card-head { display: flex; align-items: flex-start; gap: 8px; }
	.card-title { flex: 1; font-size: 0.9rem; font-weight: 600; color: var(--dt); line-height: 1.3; }
	.card-actions { display: flex; gap: 4px; opacity: 0; transition: opacity 0.12s; }
	.card:hover .card-actions { opacity: 1; }
	.icon-btn { width: 26px; height: 26px; border-radius: 6px; border: 1px solid var(--dbd); background: var(--dbg); color: var(--dt2); display: flex; align-items: center; justify-content: center; cursor: pointer; flex-shrink: 0; }
	.icon-btn:hover { background: color-mix(in srgb, var(--dt) 8%, transparent); }
	.icon-btn.danger:hover { color: #ef4444; border-color: #ef4444; }
	.cat-badge { align-self: flex-start; font-size: 0.68rem; text-transform: uppercase; letter-spacing: 0.04em; padding: 2px 7px; border-radius: 4px; background: color-mix(in srgb, var(--dt) 8%, transparent); color: var(--dt2); }
	.card-notes { font-size: 0.82rem; color: var(--dt2); margin: 0; line-height: 1.45; }
	.card-link { display: inline-flex; align-items: center; gap: 5px; font-size: 0.8rem; color: var(--dt3); text-decoration: none; margin-top: auto; }
	.card-link:hover { color: var(--dt); text-decoration: underline; }

	.empty-state { flex: 1; display: flex; flex-direction: column; align-items: center; justify-content: center; gap: 10px; padding: 64px 24px; text-align: center; }
	.empty-state :global(.empty-icon) { color: var(--dt4, var(--dt3)); opacity: 0.5; }
	.empty-title { font-size: 0.95rem; font-weight: 600; color: var(--dt2); margin: 4px 0 0; }
	.empty-body { font-size: 0.84rem; color: var(--dt3); max-width: 360px; margin: 0; }
	.empty-actions { display: flex; gap: 8px; margin-top: 8px; }

	.modal-overlay { position: fixed; inset: 0; background: rgba(0,0,0,0.4); display: flex; align-items: center; justify-content: center; z-index: 100; padding: 20px; }
	.modal { width: 100%; max-width: 460px; background: var(--dbg); border: 1px solid var(--dbd); border-radius: 14px; padding: 20px; display: flex; flex-direction: column; gap: 14px; }
	.modal-head { display: flex; align-items: center; justify-content: space-between; }
	.modal-head h2 { font-size: 1rem; font-weight: 650; color: var(--dt); margin: 0; }
	.field { display: flex; flex-direction: column; gap: 5px; flex: 1; }
	.field span { font-size: 0.78rem; color: var(--dt2); font-weight: 550; }
	.field input, .field textarea { padding: 8px 10px; border-radius: 8px; border: 1px solid var(--dbd); background: transparent; color: var(--dt); font-size: 0.85rem; font-family: inherit; resize: vertical; }
	.field-row { display: flex; gap: 12px; }
	.modal-foot { display: flex; justify-content: flex-end; gap: 8px; margin-top: 4px; }

	:global(.spin) { animation: spin 0.8s linear infinite; }
	@keyframes spin { to { transform: rotate(360deg); } }
	@media (max-width: 768px) { .page { padding: 16px 18px; } .head-actions { flex-wrap: wrap; } .grid { grid-template-columns: 1fr; } }
	@media (max-width: 480px) { .page { padding: 12px 14px; } }
</style>
