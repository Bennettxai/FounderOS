<script lang="ts">
	import { onMount } from 'svelte';
	import { currentWorkspace } from '$lib/stores/workspaces';
	import {
		getSites,
		createSite,
		updateSite,
		deleteSite,
		type Site,
		type SiteKind
	} from '$lib/api/sites';
	import { Globe, Plus, Loader2, X, Search, Trash2, Pencil, ExternalLink } from 'lucide-svelte';

	let sites = $state<Site[]>([]);
	let loading = $state(true);
	let error = $state<string | null>(null);
	let query = $state('');

	let showEdit = $state(false);
	let saving = $state(false);
	let editing = $state<Site | null>(null);
	let form = $state({ name: '', kind: 'page' as SiteKind, url: '', status: 'live', cta: 'Book a Solutions Call', notes: '' });

	const STATUSES = ['live', 'draft', 'building', 'archived'];
	const KINDS: SiteKind[] = ['funnel', 'page', 'form', 'site', 'app'];

	let wsId = $state<string | null | undefined>(undefined);
	$effect(() => {
		const id = $currentWorkspace?.id ?? null;
		if (id !== wsId) { wsId = id; load(); }
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
		loading = true; error = null;
		try {
			const res = await getSites(query.trim() || undefined);
			sites = res.sites;
		} catch (e) { error = e instanceof Error ? e.message : 'Failed to load sites'; }
		finally { loading = false; }
	}

	// Group sites by status for display.
	const grouped = $derived.by(() => {
		const order = ['live', 'building', 'draft', 'archived'];
		const map = new Map<string, Site[]>();
		for (const s of sites) {
			const key = (s.status && s.status.trim()) || 'live';
			if (!map.has(key)) map.set(key, []);
			map.get(key)!.push(s);
		}
		return Array.from(map.entries()).sort((a, b) => {
			const ai = order.indexOf(a[0]); const bi = order.indexOf(b[0]);
			return (ai === -1 ? 99 : ai) - (bi === -1 ? 99 : bi) || a[0].localeCompare(b[0]);
		});
	});

	function displayUrl(u: string) {
		return u.replace(/^https?:\/\//, '').replace(/\/$/, '');
	}
	function hrefFor(u: string) {
		if (!u) return '';
		return /^https?:\/\//.test(u) ? u : `https://${u}`;
	}

	function openNew() {
		editing = null;
		form = { name: '', kind: 'page', url: '', status: 'live', cta: 'Book a Solutions Call', notes: '' };
		showEdit = true;
	}
	function openEdit(s: Site) {
		editing = s;
		form = { name: s.name, kind: s.kind ?? 'page', url: s.url ?? '', status: s.status || 'live', cta: s.cta ?? 'Book a Solutions Call', notes: s.notes ?? '' };
		showEdit = true;
	}

	async function save(e: Event) {
		e.preventDefault();
		if (!form.name.trim()) return;
		saving = true; error = null;
		try {
			const body = {
				name: form.name.trim(),
				kind: form.kind,
				url: form.url.trim(),
				status: form.status || 'live',
				cta: form.cta.trim(),
				notes: form.notes.trim()
			};
			if (editing) await updateSite(editing.id, body);
			else await createSite(body);
			showEdit = false;
			await load();
		} catch (e) { error = e instanceof Error ? e.message : 'Failed to save site'; }
		finally { saving = false; }
	}

	async function remove(s: Site) {
		if (!confirm(`Delete "${s.name}"?`)) return;
		try { await deleteSite(s.id); sites = sites.filter((x) => x.id !== s.id); }
		catch (e) { error = e instanceof Error ? e.message : 'Failed to delete'; }
	}
</script>

<svelte:head>
	<title>Sites - BusinessOS</title>
</svelte:head>

<div class="st-root">
	<header class="topbar">
		<div class="title-wrap"><h1>Sites</h1><span class="count">{sites.length}</span></div>
		<div class="tools">
			<div class="search">
				<Search size={15} />
				<input bind:value={query} oninput={onSearch} placeholder="Search sites…" aria-label="Search sites" />
			</div>
			<button class="btn btn--primary" onclick={openNew}><Plus size={16} strokeWidth={2.4} />New site</button>
		</div>
	</header>

	{#if error}<div class="banner banner--error">{error}</div>{/if}

	{#if loading}
		<div class="loading"><Loader2 class="spin" size={20} /> Loading sites…</div>
	{:else if sites.length === 0 && query.trim()}
		<div class="empty">
			<Search size={26} strokeWidth={1.4} />
			<p>No sites match "{query.trim()}".</p>
			<button class="btn btn--ghost" onclick={clearSearch}>Clear search</button>
		</div>
	{:else if sites.length === 0}
		<div class="empty">
			<Globe size={26} strokeWidth={1.4} />
			<p>No sites yet. Track your business's web properties so people and AI agents know where you live online.</p>
			<button class="btn btn--primary" onclick={openNew}><Plus size={16} />Add your first site</button>
		</div>
	{:else}
		<div class="list">
			{#each grouped as [status, items]}
				<div class="cat">
					<div class="cat-head">{status}</div>
					{#each items as s (s.id)}
						<div class="site">
							<div class="site-main">
								<div class="site-name">
									{s.name}
									{#if s.kind}<span class="kind">{s.kind}</span>{/if}
									<span class="status status--{s.status}">{s.status}</span>
								</div>
								{#if s.url}
									<a class="site-url" href={hrefFor(s.url)} target="_blank" rel="noopener noreferrer">
										{displayUrl(s.url)}<ExternalLink size={12} />
									</a>
								{/if}
								{#if s.notes}<div class="site-notes">{s.notes}</div>{/if}
								{#if s.cta}<div class="site-cta">{s.cta}</div>{/if}
							</div>
							<div class="site-actions">
								<button class="icon-btn" title="Edit" onclick={() => openEdit(s)}><Pencil size={14} /></button>
								<button class="icon-btn icon-btn--danger" title="Delete" onclick={() => remove(s)}><Trash2 size={14} /></button>
							</div>
						</div>
					{/each}
				</div>
			{/each}
		</div>
	{/if}
</div>

{#if showEdit}
	<div class="overlay" role="button" tabindex="0" onclick={() => (showEdit = false)} onkeydown={(e) => e.key === 'Escape' && (showEdit = false)}>
		<div class="modal" role="dialog" tabindex="-1" onclick={(e) => e.stopPropagation()} onkeydown={() => {}}>
			<div class="modal-head"><h2>{editing ? 'Edit site' : 'New site'}</h2><button class="card-x" onclick={() => (showEdit = false)} aria-label="Close"><X size={18} /></button></div>
			<form onsubmit={save}>
				<label class="field"><span>Name</span><input bind:value={form.name} placeholder="e.g. Solutions Call funnel" required /></label>
				<label class="field"><span>URL</span><input bind:value={form.url} placeholder="e.g. https://agencymiosa.com" /></label>
				<div class="field-row">
					<label class="field"><span>Kind</span>
						<select bind:value={form.kind}>
							{#each KINDS as k}<option value={k}>{k}</option>{/each}
						</select>
					</label>
					<label class="field"><span>Status</span>
						<select bind:value={form.status}>
							{#each STATUSES as st}<option value={st}>{st}</option>{/each}
						</select>
					</label>
				</div>
				<label class="field"><span>CTA</span><input bind:value={form.cta} placeholder="e.g. Book a Solutions Call" /></label>
				<label class="field"><span>Notes</span><textarea bind:value={form.notes} rows="3" placeholder="What this property is for (optional)…"></textarea></label>
				<div class="modal-actions">
					<button type="button" class="btn btn--ghost" onclick={() => (showEdit = false)}>Cancel</button>
					<button type="submit" class="btn btn--primary" disabled={saving || !form.name.trim()}>{#if saving}<Loader2 class="spin" size={15} />{/if}{editing ? 'Save' : 'Add site'}</button>
				</div>
			</form>
		</div>
	</div>
{/if}

<style>
	.st-root { height: 100%; display: flex; flex-direction: column; background: var(--dbg); color: var(--dt); overflow: hidden; }
	.topbar { display: flex; align-items: center; justify-content: space-between; gap: 16px; padding: 16px 24px; border-bottom: 1px solid var(--dbd); flex-shrink: 0; }
	.title-wrap { display: flex; align-items: center; gap: 10px; }
	.title-wrap h1 { font-size: 1.15rem; font-weight: 680; letter-spacing: -0.02em; margin: 0; }
	.count { font-size: 0.74rem; color: var(--dt3); background: color-mix(in srgb, var(--dt) 8%, transparent); padding: 2px 9px; border-radius: 999px; }
	.tools { display: flex; align-items: center; gap: 10px; }
	.search { display: flex; align-items: center; gap: 7px; padding: 7px 11px; border: 1px solid var(--dbd); border-radius: 9px; color: var(--dt3); }
	.search input { background: transparent; border: none; outline: none; color: var(--dt); font-size: 0.84rem; width: 180px; font-family: inherit; }
	.btn { display: inline-flex; align-items: center; gap: 7px; padding: 8px 15px; border-radius: 9px; font-size: 0.83rem; font-weight: 580; cursor: pointer; border: 1px solid transparent; white-space: nowrap; }
	.btn--primary { background: var(--dt); color: var(--dbg); }
	.btn--ghost { background: transparent; border-color: var(--dbd); color: var(--dt2); }
	.btn:disabled { opacity: 0.55; cursor: not-allowed; }
	.loading, .empty { flex: 1; display: flex; flex-direction: column; align-items: center; justify-content: center; gap: 14px; color: var(--dt3); text-align: center; padding: 0 24px; }
	.empty p { max-width: 420px; line-height: 1.5; }
	.banner { margin: 14px 24px 0; padding: 11px 14px; border-radius: 10px; font-size: 0.83rem; }
	.banner--error { background: color-mix(in srgb, #ef4444 12%, transparent); color: #ef4444; }
	.list { flex: 1; overflow-y: auto; padding: 18px 24px; max-width: 820px; width: 100%; margin: 0 auto; }
	.cat { margin-bottom: 24px; }
	.cat-head { font-size: 0.74rem; font-weight: 640; color: var(--dt3); text-transform: uppercase; letter-spacing: 0.06em; margin-bottom: 10px; padding-bottom: 7px; border-bottom: 1px solid var(--dbd); }
	.site { display: flex; align-items: flex-start; gap: 14px; padding: 12px 4px; }
	.site:not(:last-child) { border-bottom: 1px solid color-mix(in srgb, var(--dt) 5%, transparent); }
	.site-main { flex: 1; min-width: 0; }
	.site-name { font-size: 0.92rem; font-weight: 620; display: flex; align-items: center; gap: 8px; flex-wrap: wrap; }
	.status { font-size: 0.66rem; font-weight: 620; text-transform: uppercase; letter-spacing: 0.05em; padding: 2px 7px; border-radius: 999px; background: color-mix(in srgb, var(--dt) 8%, transparent); color: var(--dt3); }
	.status--live { background: color-mix(in srgb, #22c55e 16%, transparent); color: #22c55e; }
	.status--building { background: color-mix(in srgb, #f59e0b 16%, transparent); color: #f59e0b; }
	.status--draft { background: color-mix(in srgb, #60a5fa 16%, transparent); color: #60a5fa; }
	.status--archived { background: color-mix(in srgb, var(--dt) 8%, transparent); color: var(--dt4); }
	.kind { font-size: 0.64rem; font-weight: 640; text-transform: uppercase; letter-spacing: 0.05em; padding: 2px 7px; border-radius: 999px; background: color-mix(in srgb, #818cf8 16%, transparent); color: #818cf8; }
	.site-cta { font-size: 0.74rem; font-weight: 600; color: var(--dt3); margin-top: 6px; }
	.site-url { display: inline-flex; align-items: center; gap: 5px; font-size: 0.82rem; color: var(--dt2); text-decoration: none; margin-top: 4px; }
	.site-url:hover { color: var(--dt); text-decoration: underline; }
	.site-notes { font-size: 0.82rem; color: var(--dt3); line-height: 1.5; margin-top: 4px; white-space: pre-wrap; }
	.site-actions { display: flex; gap: 4px; flex-shrink: 0; opacity: 0; transition: opacity 0.12s; }
	.site:hover .site-actions { opacity: 1; }
	.icon-btn { display: inline-flex; align-items: center; justify-content: center; width: 28px; height: 28px; border-radius: 7px; border: none; background: transparent; color: var(--dt3); cursor: pointer; }
	.icon-btn:hover { background: color-mix(in srgb, var(--dt) 8%, transparent); color: var(--dt); }
	.icon-btn--danger:hover { background: color-mix(in srgb, #ef4444 12%, transparent); color: #ef4444; }
	.overlay { position: fixed; inset: 0; background: rgba(0,0,0,0.55); display: flex; align-items: center; justify-content: center; z-index: 100; padding: 20px; }
	.modal { width: 100%; max-width: 480px; background: var(--dbg); border: 1px solid var(--dbd); border-radius: 16px; padding: 22px; box-shadow: 0 24px 60px rgba(0,0,0,0.5); }
	.modal-head { display: flex; align-items: center; justify-content: space-between; margin-bottom: 18px; }
	.modal-head h2 { font-size: 1.05rem; font-weight: 640; margin: 0; }
	.card-x { display: inline-flex; align-items: center; justify-content: center; width: 28px; height: 28px; border-radius: 7px; border: none; background: transparent; color: var(--dt3); cursor: pointer; }
	.card-x:hover { background: color-mix(in srgb, var(--dt) 8%, transparent); }
	.field { display: flex; flex-direction: column; gap: 6px; margin-bottom: 14px; }
	.field span { font-size: 0.8rem; font-weight: 560; color: var(--dt2); }
	.field input, .field textarea, .field select { padding: 9px 12px; border-radius: 9px; border: 1px solid var(--dbd); background: var(--dbg); color: var(--dt); font-size: 0.86rem; outline: none; font-family: inherit; resize: vertical; }
	.field input:focus, .field textarea:focus, .field select:focus { border-color: color-mix(in srgb, var(--dt) 40%, transparent); }
	.field-row { display: flex; gap: 12px; }
	.field-row .field { flex: 1; }
	.modal-actions { display: flex; justify-content: flex-end; gap: 10px; margin-top: 6px; }
	:global(.spin) { animation: spin 0.9s linear infinite; }
	@keyframes spin { to { transform: rotate(360deg); } }
</style>
