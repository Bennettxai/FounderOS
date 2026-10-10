<script lang="ts">
	import { onMount } from 'svelte';
	import { currentWorkspace } from '$lib/stores/workspaces';
	import {
		getAssets,
		uploadAsset,
		createAssetLink,
		updateAsset,
		deleteAsset,
		assetSrc,
		type Asset,
		type AssetKind
	} from '$lib/api/assets';
	import {
		Image as ImageIcon,
		Upload,
		Link2,
		Plus,
		Loader2,
		X,
		Search,
		Trash2,
		Pencil,
		FileText,
		Film,
		File as FileIcon
	} from 'lucide-svelte';

	const KINDS: { value: AssetKind; label: string }[] = [
		{ value: 'image', label: 'Image' },
		{ value: 'logo', label: 'Logo' },
		{ value: 'video', label: 'Video' },
		{ value: 'document', label: 'Document' },
		{ value: 'file', label: 'File' },
		{ value: 'link', label: 'Link' }
	];
	const kindLabel = (k: string) => KINDS.find((x) => x.value === k)?.label ?? k;

	let assets = $state<Asset[]>([]);
	let loading = $state(true);
	let error = $state<string | null>(null);
	let query = $state('');
	let uploading = $state(false);
	let fileInput = $state<HTMLInputElement | null>(null);

	// Link modal
	let showLink = $state(false);
	let savingLink = $state(false);
	let linkForm = $state({ name: '', url: '', kind: 'link' as AssetKind, tags: '' });

	// Edit modal
	let showEdit = $state(false);
	let savingEdit = $state(false);
	let editing = $state<Asset | null>(null);
	let editForm = $state({ name: '', tags: '', kind: 'file' as AssetKind });

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
			const res = await getAssets(query.trim() || undefined);
			assets = res.assets;
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to load assets';
		} finally {
			loading = false;
		}
	}

	async function onFilesPicked(e: Event) {
		const input = e.target as HTMLInputElement;
		const files = input.files;
		if (!files || files.length === 0) return;
		uploading = true;
		error = null;
		try {
			for (const f of Array.from(files)) {
				await uploadAsset(f, { name: f.name });
			}
			await load();
		} catch (err) {
			error = err instanceof Error ? err.message : 'Upload failed';
		} finally {
			uploading = false;
			if (input) input.value = '';
		}
	}

	function openLink() {
		linkForm = { name: '', url: '', kind: 'link', tags: '' };
		showLink = true;
	}
	async function saveLink(e: Event) {
		e.preventDefault();
		if (!linkForm.name.trim() || !linkForm.url.trim()) return;
		savingLink = true;
		error = null;
		try {
			await createAssetLink({
				name: linkForm.name.trim(),
				url: linkForm.url.trim(),
				kind: linkForm.kind,
				tags: linkForm.tags.trim()
			});
			showLink = false;
			await load();
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to add link';
		} finally {
			savingLink = false;
		}
	}

	function openEdit(a: Asset) {
		editing = a;
		editForm = { name: a.name, tags: a.tags ?? '', kind: a.kind };
		showEdit = true;
	}
	async function saveEdit(e: Event) {
		e.preventDefault();
		if (!editing) return;
		savingEdit = true;
		error = null;
		try {
			await updateAsset(editing.id, {
				name: editForm.name.trim(),
				tags: editForm.tags.trim(),
				kind: editForm.kind
			});
			showEdit = false;
			editing = null;
			await load();
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to save';
		} finally {
			savingEdit = false;
		}
	}

	async function remove(a: Asset) {
		if (!confirm(`Delete "${a.name}"? This can't be undone.`)) return;
		try {
			await deleteAsset(a.id);
			assets = assets.filter((x) => x.id !== a.id);
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to delete';
		}
	}

	function fmtSize(n: number): string {
		if (!n) return '';
		if (n < 1024) return `${n} B`;
		if (n < 1024 * 1024) return `${(n / 1024).toFixed(0)} KB`;
		return `${(n / 1024 / 1024).toFixed(1)} MB`;
	}
	function tagList(t: string): string[] {
		return (t || '')
			.split(',')
			.map((x) => x.trim())
			.filter(Boolean);
	}
	const isVisual = (a: Asset) => a.kind === 'image' || a.kind === 'logo';
</script>

<svelte:head><title>Assets - BusinessOS</title></svelte:head>

<div class="page">
	<header class="page-header">
		<div class="page-icon"><ImageIcon size={22} strokeWidth={1.8} /></div>
		<div class="head-text">
			<h1 class="page-title">Assets</h1>
			<p class="page-desc">Brand assets, images, logos, and media files shared across your workspace.</p>
		</div>
		<div class="head-actions">
			<button class="btn btn-ghost" onclick={openLink}><Link2 size={15} /> Add link</button>
			<button class="btn btn-primary" onclick={() => fileInput?.click()} disabled={uploading}>
				{#if uploading}<Loader2 size={15} class="spin" />{:else}<Upload size={15} />{/if}
				Upload
			</button>
			<input
				bind:this={fileInput}
				type="file"
				multiple
				class="hidden-input"
				onchange={onFilesPicked}
			/>
		</div>
	</header>

	<div class="toolbar">
		<div class="search">
			<Search size={15} class="search-icon" />
			<input placeholder="Search assets, tags…" bind:value={query} oninput={onSearch} />
			{#if query}<button class="search-clear" onclick={clearSearch}><X size={13} /></button>{/if}
		</div>
		<span class="count">{assets.length} asset{assets.length === 1 ? '' : 's'}</span>
	</div>

	{#if error}
		<div class="error-bar">{error}</div>
	{/if}

	{#if loading}
		<div class="loading"><Loader2 size={22} class="spin" /> Loading assets…</div>
	{:else if assets.length === 0}
		<div class="empty-state">
			<ImageIcon size={40} strokeWidth={1.4} class="empty-icon" />
			<p class="empty-title">No assets yet</p>
			<p class="empty-body">Upload brand assets and media, or add a link, to make them accessible across projects and campaigns.</p>
			<div class="empty-actions">
				<button class="btn btn-primary" onclick={() => fileInput?.click()}><Upload size={15} /> Upload a file</button>
				<button class="btn btn-ghost" onclick={openLink}><Link2 size={15} /> Add a link</button>
			</div>
		</div>
	{:else}
		<div class="grid">
			{#each assets as a (a.id)}
				<div class="card">
					<div class="thumb">
						{#if isVisual(a)}
							<img src={assetSrc(a)} alt={a.name} loading="lazy" />
						{:else if a.kind === 'video'}
							<Film size={26} strokeWidth={1.5} />
						{:else if a.kind === 'document'}
							<FileText size={26} strokeWidth={1.5} />
						{:else if a.kind === 'link'}
							<Link2 size={26} strokeWidth={1.5} />
						{:else}
							<FileIcon size={26} strokeWidth={1.5} />
						{/if}
						<div class="card-actions">
							<button class="icon-btn" title="Edit" onclick={() => openEdit(a)}><Pencil size={13} /></button>
							<button class="icon-btn danger" title="Delete" onclick={() => remove(a)}><Trash2 size={13} /></button>
						</div>
					</div>
					<div class="card-body">
						<a class="card-name" href={assetSrc(a)} target="_blank" rel="noopener" title={a.name}>{a.name}</a>
						<div class="card-meta">
							<span class="kind-badge">{kindLabel(a.kind)}</span>
							{#if a.size_bytes}<span class="size">{fmtSize(a.size_bytes)}</span>{/if}
						</div>
						{#if tagList(a.tags).length}
							<div class="tags">
								{#each tagList(a.tags) as t}<span class="tag">{t}</span>{/each}
							</div>
						{/if}
					</div>
				</div>
			{/each}
		</div>
	{/if}
</div>

{#if showLink}
	<div class="modal-overlay" role="button" tabindex="0" onclick={(e) => { if (e.target === e.currentTarget) showLink = false; }} onkeydown={(e) => e.key === 'Escape' && (showLink = false)}>
		<form class="modal" onsubmit={saveLink}>
			<div class="modal-head">
				<h2>Add a link asset</h2>
				<button type="button" class="icon-btn" onclick={() => (showLink = false)}><X size={16} /></button>
			</div>
			<label class="field">
				<span>Name</span>
				<input bind:value={linkForm.name} placeholder="Brand guidelines" required />
			</label>
			<label class="field">
				<span>URL</span>
				<input bind:value={linkForm.url} placeholder="https://…" required />
			</label>
			<div class="field-row">
				<label class="field">
					<span>Kind</span>
					<select bind:value={linkForm.kind}>
						{#each KINDS as k}<option value={k.value}>{k.label}</option>{/each}
					</select>
				</label>
				<label class="field">
					<span>Tags</span>
					<input bind:value={linkForm.tags} placeholder="logo, brand (comma-separated)" />
				</label>
			</div>
			<div class="modal-foot">
				<button type="button" class="btn btn-ghost" onclick={() => (showLink = false)}>Cancel</button>
				<button type="submit" class="btn btn-primary" disabled={savingLink}>
					{#if savingLink}<Loader2 size={15} class="spin" />{:else}<Plus size={15} />{/if} Add asset
				</button>
			</div>
		</form>
	</div>
{/if}

{#if showEdit && editing}
	<div class="modal-overlay" role="button" tabindex="0" onclick={(e) => { if (e.target === e.currentTarget) showEdit = false; }} onkeydown={(e) => e.key === 'Escape' && (showEdit = false)}>
		<form class="modal" onsubmit={saveEdit}>
			<div class="modal-head">
				<h2>Edit asset</h2>
				<button type="button" class="icon-btn" onclick={() => (showEdit = false)}><X size={16} /></button>
			</div>
			<label class="field">
				<span>Name</span>
				<input bind:value={editForm.name} required />
			</label>
			<div class="field-row">
				<label class="field">
					<span>Kind</span>
					<select bind:value={editForm.kind}>
						{#each KINDS as k}<option value={k.value}>{k.label}</option>{/each}
					</select>
				</label>
				<label class="field">
					<span>Tags</span>
					<input bind:value={editForm.tags} placeholder="comma-separated" />
				</label>
			</div>
			<div class="modal-foot">
				<button type="button" class="btn btn-ghost" onclick={() => (showEdit = false)}>Cancel</button>
				<button type="submit" class="btn btn-primary" disabled={savingEdit}>
					{#if savingEdit}<Loader2 size={15} class="spin" />{:else}<Pencil size={15} />{/if} Save
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
	.hidden-input { display: none; }

	.toolbar { display: flex; align-items: center; justify-content: space-between; gap: 12px; }
	.search { position: relative; display: flex; align-items: center; flex: 1; max-width: 360px; }
	.search :global(.search-icon) { position: absolute; left: 10px; color: var(--dt3); }
	.search input { width: 100%; padding: 8px 30px; border-radius: 8px; border: 1px solid var(--dbd); background: transparent; color: var(--dt); font-size: 0.85rem; }
	.search-clear { position: absolute; right: 8px; background: none; border: none; color: var(--dt3); cursor: pointer; display: flex; }
	.count { font-size: 0.8rem; color: var(--dt3); }

	.error-bar { padding: 10px 14px; border-radius: 8px; background: color-mix(in srgb, #ef4444 12%, transparent); color: #ef4444; font-size: 0.83rem; }
	.loading { display: flex; align-items: center; gap: 8px; padding: 48px; justify-content: center; color: var(--dt3); font-size: 0.9rem; }

	.grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(180px, 1fr)); gap: 16px; }
	.card { border: 1px solid var(--dbd); border-radius: 12px; overflow: hidden; background: color-mix(in srgb, var(--dt) 2%, transparent); transition: border-color 0.12s; }
	.card:hover { border-color: color-mix(in srgb, var(--dt) 22%, transparent); }
	.thumb { position: relative; aspect-ratio: 4 / 3; display: flex; align-items: center; justify-content: center; background: color-mix(in srgb, var(--dt) 5%, transparent); color: var(--dt3); overflow: hidden; }
	.thumb img { width: 100%; height: 100%; object-fit: cover; }
	.card-actions { position: absolute; top: 8px; right: 8px; display: flex; gap: 4px; opacity: 0; transition: opacity 0.12s; }
	.card:hover .card-actions { opacity: 1; }
	.icon-btn { width: 26px; height: 26px; border-radius: 6px; border: 1px solid var(--dbd); background: var(--dbg); color: var(--dt2); display: flex; align-items: center; justify-content: center; cursor: pointer; }
	.icon-btn:hover { background: color-mix(in srgb, var(--dt) 8%, transparent); }
	.icon-btn.danger:hover { color: #ef4444; border-color: #ef4444; }
	.card-body { padding: 10px 12px; display: flex; flex-direction: column; gap: 6px; }
	.card-name { font-size: 0.85rem; font-weight: 550; color: var(--dt); text-decoration: none; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
	.card-name:hover { text-decoration: underline; }
	.card-meta { display: flex; align-items: center; gap: 8px; }
	.kind-badge { font-size: 0.68rem; text-transform: uppercase; letter-spacing: 0.04em; padding: 2px 6px; border-radius: 4px; background: color-mix(in srgb, var(--dt) 8%, transparent); color: var(--dt2); }
	.size { font-size: 0.72rem; color: var(--dt3); }
	.tags { display: flex; flex-wrap: wrap; gap: 4px; }
	.tag { font-size: 0.68rem; padding: 2px 6px; border-radius: 4px; border: 1px solid var(--dbd); color: var(--dt3); }

	.empty-state { flex: 1; display: flex; flex-direction: column; align-items: center; justify-content: center; gap: 10px; padding: 64px 24px; text-align: center; }
	.empty-state :global(.empty-icon) { color: var(--dt4, var(--dt3)); opacity: 0.5; }
	.empty-title { font-size: 0.95rem; font-weight: 600; color: var(--dt2); margin: 4px 0 0; }
	.empty-body { font-size: 0.84rem; color: var(--dt3); max-width: 360px; margin: 0; }
	.empty-actions { display: flex; gap: 8px; margin-top: 8px; }

	.modal-overlay { position: fixed; inset: 0; background: rgba(0,0,0,0.4); display: flex; align-items: center; justify-content: center; z-index: 100; padding: 20px; }
	.modal { width: 100%; max-width: 440px; background: var(--dbg); border: 1px solid var(--dbd); border-radius: 14px; padding: 20px; display: flex; flex-direction: column; gap: 14px; }
	.modal-head { display: flex; align-items: center; justify-content: space-between; }
	.modal-head h2 { font-size: 1rem; font-weight: 650; color: var(--dt); margin: 0; }
	.field { display: flex; flex-direction: column; gap: 5px; flex: 1; }
	.field span { font-size: 0.78rem; color: var(--dt2); font-weight: 550; }
	.field input, .field select { padding: 8px 10px; border-radius: 8px; border: 1px solid var(--dbd); background: transparent; color: var(--dt); font-size: 0.85rem; }
	.field-row { display: flex; gap: 12px; }
	.modal-foot { display: flex; justify-content: flex-end; gap: 8px; margin-top: 4px; }

	:global(.spin) { animation: spin 0.8s linear infinite; }
	@keyframes spin { to { transform: rotate(360deg); } }

	@media (max-width: 768px) { .page { padding: 16px 18px; } .head-actions { flex-wrap: wrap; } }
	@media (max-width: 480px) { .page { padding: 12px 14px; } }
</style>
