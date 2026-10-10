<script lang="ts">
	import { onMount } from 'svelte';
	import { currentWorkspace } from '$lib/stores/workspaces';
	import {
		getDriveFiles,
		getDriveFolders,
		uploadDriveFile,
		updateDriveFile,
		deleteDriveFile,
		driveFileSrc,
		type DriveFile
	} from '$lib/api/drive';
	import {
		HardDrive,
		Upload,
		Loader2,
		X,
		Search,
		Trash2,
		Pencil,
		Folder,
		FolderOpen,
		Home,
		Download,
		Image as ImageIcon,
		FileText,
		Film,
		Music,
		File as FileIcon
	} from 'lucide-svelte';

	let files = $state<DriveFile[]>([]);
	let folders = $state<string[]>([]);
	let loading = $state(true);
	let error = $state<string | null>(null);
	let query = $state('');
	let activeFolder = $state<string>(''); // '' = root / all files

	let uploading = $state(false);
	let fileInput = $state<HTMLInputElement | null>(null);

	// Edit (rename / move) modal
	let showEdit = $state(false);
	let savingEdit = $state(false);
	let editing = $state<DriveFile | null>(null);
	let editForm = $state({ name: '', folder: '' });

	let wsId = $state<string | null | undefined>(undefined);
	$effect(() => {
		const id = $currentWorkspace?.id ?? null;
		if (id !== wsId) {
			wsId = id;
			activeFolder = '';
			query = '';
			load();
		}
	});

	onMount(load);

	// Breadcrumb segments for the active folder path.
	let crumbs = $derived(activeFolder ? activeFolder.split('/') : []);

	let searchTimer: ReturnType<typeof setTimeout>;
	function onSearch() {
		clearTimeout(searchTimer);
		searchTimer = setTimeout(loadFiles, 250);
	}
	function clearSearch() {
		query = '';
		clearTimeout(searchTimer);
		loadFiles();
	}

	// Full load: files + folder list.
	async function load() {
		loading = true;
		error = null;
		try {
			const [f, fo] = await Promise.all([
				getDriveFiles({
					folder: query.trim() ? undefined : activeFolder,
					q: query.trim() || undefined
				}),
				getDriveFolders()
			]);
			files = f.files;
			folders = fo.folders;
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to load drive';
		} finally {
			loading = false;
		}
	}

	// Refresh just the file list (search / folder switch). When searching, list
	// across all folders; otherwise scope to the active folder.
	async function loadFiles() {
		loading = true;
		error = null;
		try {
			const res = await getDriveFiles({
				folder: query.trim() ? undefined : activeFolder,
				q: query.trim() || undefined
			});
			files = res.files;
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to load files';
		} finally {
			loading = false;
		}
	}

	function selectFolder(f: string) {
		activeFolder = f;
		query = '';
		loadFiles();
	}

	async function onFilesPicked(e: Event) {
		const input = e.target as HTMLInputElement;
		const picked = input.files;
		if (!picked || picked.length === 0) return;
		uploading = true;
		error = null;
		try {
			for (const f of Array.from(picked)) {
				await uploadDriveFile(f, { name: f.name, folder: activeFolder || undefined });
			}
			await load();
		} catch (err) {
			error = err instanceof Error ? err.message : 'Upload failed';
		} finally {
			uploading = false;
			if (input) input.value = '';
		}
	}

	function openEdit(f: DriveFile) {
		editing = f;
		editForm = { name: f.name, folder: f.folder ?? '' };
		showEdit = true;
	}
	async function saveEdit(e: Event) {
		e.preventDefault();
		if (!editing) return;
		savingEdit = true;
		error = null;
		try {
			await updateDriveFile(editing.id, {
				name: editForm.name.trim(),
				folder: editForm.folder.trim()
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

	async function remove(f: DriveFile) {
		if (!confirm(`Delete "${f.name}"? This can't be undone.`)) return;
		try {
			await deleteDriveFile(f.id);
			files = files.filter((x) => x.id !== f.id);
			// A folder may have just emptied out — refresh the folder list.
			const fo = await getDriveFolders();
			folders = fo.folders;
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

	// Pick an icon component from the mime type.
	function iconFor(mime: string) {
		if (mime.startsWith('image/')) return ImageIcon;
		if (mime.startsWith('video/')) return Film;
		if (mime.startsWith('audio/')) return Music;
		if (
			mime.startsWith('application/pdf') ||
			mime.startsWith('application/msword') ||
			mime.startsWith('application/vnd') ||
			mime.startsWith('text/')
		)
			return FileText;
		return FileIcon;
	}
	const isImage = (f: DriveFile) => f.mime_type.startsWith('image/');
</script>

<svelte:head><title>Drive - BusinessOS</title></svelte:head>

<div class="page">
	<header class="page-header">
		<div class="page-icon"><HardDrive size={22} strokeWidth={1.8} /></div>
		<div class="head-text">
			<h1 class="page-title">Drive</h1>
			<p class="page-desc">Store and organize files into folders, shared across your workspace.</p>
		</div>
		<div class="head-actions">
			<button class="btn btn-primary" onclick={() => fileInput?.click()} disabled={uploading}>
				{#if uploading}<Loader2 size={15} class="spin" />{:else}<Upload size={15} />{/if}
				Upload
			</button>
			<input bind:this={fileInput} type="file" multiple class="hidden-input" onchange={onFilesPicked} />
		</div>
	</header>

	<div class="toolbar">
		<div class="search">
			<Search size={15} class="search-icon" />
			<input placeholder="Search files, folders…" bind:value={query} oninput={onSearch} />
			{#if query}<button class="search-clear" onclick={clearSearch}><X size={13} /></button>{/if}
		</div>
		<span class="count">{files.length} file{files.length === 1 ? '' : 's'}</span>
	</div>

	{#if !query}
		<div class="folder-bar">
			<button class="chip" class:active={activeFolder === ''} onclick={() => selectFolder('')}>
				<Home size={13} /> All files
			</button>
			{#each folders as f (f)}
				<button class="chip" class:active={activeFolder === f} onclick={() => selectFolder(f)}>
					{#if activeFolder === f}<FolderOpen size={13} />{:else}<Folder size={13} />{/if}
					{f}
				</button>
			{/each}
		</div>

		{#if activeFolder}
			<div class="breadcrumb">
				<button class="crumb" onclick={() => selectFolder('')}><Home size={13} /></button>
				{#each crumbs as seg, i}
					<span class="sep">/</span>
					<button class="crumb" onclick={() => selectFolder(crumbs.slice(0, i + 1).join('/'))}>{seg}</button>
				{/each}
			</div>
		{/if}
	{/if}

	{#if error}
		<div class="error-bar">{error}</div>
	{/if}

	{#if loading}
		<div class="loading"><Loader2 size={22} class="spin" /> Loading files…</div>
	{:else if files.length === 0}
		<div class="empty-state">
			<HardDrive size={40} strokeWidth={1.4} class="empty-icon" />
			<p class="empty-title">
				{#if query}No files match "{query}"{:else if activeFolder}No files in "{activeFolder}"{:else}No files yet{/if}
			</p>
			<p class="empty-body">
				Upload files to store them here. Organize them into folders like "Contracts" or "Contracts/2026" by moving files after upload.
			</p>
			<div class="empty-actions">
				<button class="btn btn-primary" onclick={() => fileInput?.click()}><Upload size={15} /> Upload a file</button>
			</div>
		</div>
	{:else}
		<div class="grid">
			{#each files as f (f.id)}
				{@const Icon = iconFor(f.mime_type)}
				<div class="card">
					<a class="thumb" href={driveFileSrc(f)} target="_blank" rel="noopener" title={f.name}>
						{#if isImage(f)}
							<img src={driveFileSrc(f)} alt={f.name} loading="lazy" />
						{:else}
							<Icon size={26} strokeWidth={1.5} />
						{/if}
					</a>
					<div class="card-actions">
						<a class="icon-btn" title="Download" href={driveFileSrc(f)} target="_blank" rel="noopener"><Download size={13} /></a>
						<button class="icon-btn" title="Rename / move" onclick={() => openEdit(f)}><Pencil size={13} /></button>
						<button class="icon-btn danger" title="Delete" onclick={() => remove(f)}><Trash2 size={13} /></button>
					</div>
					<div class="card-body">
						<a class="card-name" href={driveFileSrc(f)} target="_blank" rel="noopener" title={f.name}>{f.name}</a>
						<div class="card-meta">
							{#if f.folder}
								<button class="folder-tag" onclick={() => selectFolder(f.folder)} title="Open folder">
									<Folder size={11} /> {f.folder}
								</button>
							{/if}
							{#if f.size_bytes}<span class="size">{fmtSize(f.size_bytes)}</span>{/if}
						</div>
					</div>
				</div>
			{/each}
		</div>
	{/if}
</div>

{#if showEdit && editing}
	<div class="modal-overlay" role="button" tabindex="0" onclick={(e) => { if (e.target === e.currentTarget) showEdit = false; }} onkeydown={(e) => e.key === 'Escape' && (showEdit = false)}>
		<form class="modal" onsubmit={saveEdit}>
			<div class="modal-head">
				<h2>Rename / move file</h2>
				<button type="button" class="icon-btn" onclick={() => (showEdit = false)}><X size={16} /></button>
			</div>
			<label class="field">
				<span>Name</span>
				<input bind:value={editForm.name} required />
			</label>
			<label class="field">
				<span>Folder</span>
				<input bind:value={editForm.folder} placeholder="Contracts/2026 (blank = root)" />
			</label>
			<p class="hint">Use a slash-separated path to nest folders. Leave blank to move to the root.</p>
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

	.folder-bar { display: flex; flex-wrap: wrap; gap: 8px; }
	.chip { display: inline-flex; align-items: center; gap: 5px; padding: 6px 12px; border-radius: 20px; border: 1px solid var(--dbd); background: transparent; color: var(--dt2); font-size: 0.8rem; font-weight: 500; cursor: pointer; transition: background 0.12s, border-color 0.12s, color 0.12s; }
	.chip:hover { background: color-mix(in srgb, var(--dt) 5%, transparent); }
	.chip.active { background: var(--bos-v2-button-primary); border-color: var(--bos-v2-button-primary); color: var(--bos-v2-button-pureWhiteText); }

	.breadcrumb { display: flex; align-items: center; gap: 6px; font-size: 0.8rem; color: var(--dt3); }
	.breadcrumb .crumb { display: inline-flex; align-items: center; background: none; border: none; color: var(--dt2); cursor: pointer; padding: 0; font-size: 0.8rem; }
	.breadcrumb .crumb:hover { color: var(--dt); text-decoration: underline; }
	.breadcrumb .sep { color: var(--dt3); }

	.error-bar { padding: 10px 14px; border-radius: 8px; background: color-mix(in srgb, #ef4444 12%, transparent); color: #ef4444; font-size: 0.83rem; }
	.loading { display: flex; align-items: center; gap: 8px; padding: 48px; justify-content: center; color: var(--dt3); font-size: 0.9rem; }

	.grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(180px, 1fr)); gap: 16px; }
	.card { position: relative; border: 1px solid var(--dbd); border-radius: 12px; overflow: hidden; background: color-mix(in srgb, var(--dt) 2%, transparent); transition: border-color 0.12s; }
	.card:hover { border-color: color-mix(in srgb, var(--dt) 22%, transparent); }
	.thumb { position: relative; aspect-ratio: 4 / 3; display: flex; align-items: center; justify-content: center; background: color-mix(in srgb, var(--dt) 5%, transparent); color: var(--dt3); overflow: hidden; text-decoration: none; }
	.thumb img { width: 100%; height: 100%; object-fit: cover; }
	.card-actions { position: absolute; top: 8px; right: 8px; display: flex; gap: 4px; opacity: 0; transition: opacity 0.12s; }
	.card:hover .card-actions { opacity: 1; }
	.icon-btn { width: 26px; height: 26px; border-radius: 6px; border: 1px solid var(--dbd); background: var(--dbg); color: var(--dt2); display: flex; align-items: center; justify-content: center; cursor: pointer; text-decoration: none; }
	.icon-btn:hover { background: color-mix(in srgb, var(--dt) 8%, transparent); }
	.icon-btn.danger:hover { color: #ef4444; border-color: #ef4444; }
	.card-body { padding: 10px 12px; display: flex; flex-direction: column; gap: 6px; }
	.card-name { font-size: 0.85rem; font-weight: 550; color: var(--dt); text-decoration: none; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
	.card-name:hover { text-decoration: underline; }
	.card-meta { display: flex; align-items: center; gap: 8px; flex-wrap: wrap; }
	.folder-tag { display: inline-flex; align-items: center; gap: 4px; font-size: 0.68rem; padding: 2px 6px; border-radius: 4px; background: color-mix(in srgb, var(--dt) 8%, transparent); color: var(--dt2); border: none; cursor: pointer; max-width: 100%; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
	.folder-tag:hover { color: var(--dt); }
	.size { font-size: 0.72rem; color: var(--dt3); }

	.empty-state { flex: 1; display: flex; flex-direction: column; align-items: center; justify-content: center; gap: 10px; padding: 64px 24px; text-align: center; }
	.empty-state :global(.empty-icon) { color: var(--dt4, var(--dt3)); opacity: 0.5; }
	.empty-title { font-size: 0.95rem; font-weight: 600; color: var(--dt2); margin: 4px 0 0; }
	.empty-body { font-size: 0.84rem; color: var(--dt3); max-width: 380px; margin: 0; }
	.empty-actions { display: flex; gap: 8px; margin-top: 8px; }

	.modal-overlay { position: fixed; inset: 0; background: rgba(0,0,0,0.4); display: flex; align-items: center; justify-content: center; z-index: 100; padding: 20px; }
	.modal { width: 100%; max-width: 440px; background: var(--dbg); border: 1px solid var(--dbd); border-radius: 14px; padding: 20px; display: flex; flex-direction: column; gap: 14px; }
	.modal-head { display: flex; align-items: center; justify-content: space-between; }
	.modal-head h2 { font-size: 1rem; font-weight: 650; color: var(--dt); margin: 0; }
	.field { display: flex; flex-direction: column; gap: 5px; flex: 1; }
	.field span { font-size: 0.78rem; color: var(--dt2); font-weight: 550; }
	.field input { padding: 8px 10px; border-radius: 8px; border: 1px solid var(--dbd); background: transparent; color: var(--dt); font-size: 0.85rem; }
	.hint { font-size: 0.75rem; color: var(--dt3); margin: -4px 0 0; }
	.modal-foot { display: flex; justify-content: flex-end; gap: 8px; margin-top: 4px; }

	:global(.spin) { animation: spin 0.8s linear infinite; }
	@keyframes spin { to { transform: rotate(360deg); } }

	@media (max-width: 768px) { .page { padding: 16px 18px; } .head-actions { flex-wrap: wrap; } }
	@media (max-width: 480px) { .page { padding: 12px 14px; } }
</style>
