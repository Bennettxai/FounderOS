<script lang="ts">
	import { onMount } from 'svelte';
	import { currentWorkspace } from '$lib/stores/workspaces';
	import {
		getGlossary,
		createGlossaryTerm,
		updateGlossaryTerm,
		deleteGlossaryTerm,
		type GlossaryTerm
	} from '$lib/api/glossary';
	import { BookA, Plus, Loader2, X, Search, Trash2, Pencil } from 'lucide-svelte';

	let terms = $state<GlossaryTerm[]>([]);
	let loading = $state(true);
	let error = $state<string | null>(null);
	let query = $state('');

	let showEdit = $state(false);
	let saving = $state(false);
	let editing = $state<GlossaryTerm | null>(null);
	let form = $state({ term: '', definition: '', category: '', aliases: '' });

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

	async function load() {
		loading = true; error = null;
		try {
			const res = await getGlossary(query.trim() || undefined);
			terms = res.terms;
		} catch (e) { error = e instanceof Error ? e.message : 'Failed to load glossary'; }
		finally { loading = false; }
	}

	// Group terms by category for display.
	const grouped = $derived.by(() => {
		const map = new Map<string, GlossaryTerm[]>();
		for (const t of terms) {
			const key = (t.category && t.category.trim()) || 'General';
			if (!map.has(key)) map.set(key, []);
			map.get(key)!.push(t);
		}
		return Array.from(map.entries()).sort((a, b) => a[0].localeCompare(b[0]));
	});

	function openNew() {
		editing = null;
		form = { term: '', definition: '', category: '', aliases: '' };
		showEdit = true;
	}
	function openEdit(t: GlossaryTerm) {
		editing = t;
		form = { term: t.term, definition: t.definition, category: t.category ?? '', aliases: t.aliases ?? '' };
		showEdit = true;
	}

	async function save(e: Event) {
		e.preventDefault();
		if (!form.term.trim()) return;
		saving = true; error = null;
		try {
			const body = {
				term: form.term.trim(),
				definition: form.definition.trim(),
				category: form.category.trim() || null,
				aliases: form.aliases.trim() || null
			};
			if (editing) await updateGlossaryTerm(editing.id, body);
			else await createGlossaryTerm(body);
			showEdit = false;
			await load();
		} catch (e) { error = e instanceof Error ? e.message : 'Failed to save term'; }
		finally { saving = false; }
	}

	async function remove(t: GlossaryTerm) {
		if (!confirm(`Delete "${t.term}"?`)) return;
		try { await deleteGlossaryTerm(t.id); terms = terms.filter((x) => x.id !== t.id); }
		catch (e) { error = e instanceof Error ? e.message : 'Failed to delete'; }
	}
</script>

<svelte:head><title>Glossary - BusinessOS</title></svelte:head>

<div class="gl-root">
	<header class="topbar">
		<div class="title-wrap"><h1>Glossary</h1><span class="count">{terms.length}</span></div>
		<div class="tools">
			<div class="search">
				<Search size={15} />
				<input bind:value={query} oninput={onSearch} placeholder="Search terms…" />
			</div>
			<button class="btn btn--primary" onclick={openNew}><Plus size={16} strokeWidth={2.4} />New term</button>
		</div>
	</header>

	{#if error}<div class="banner banner--error">{error}</div>{/if}

	{#if loading}
		<div class="loading"><Loader2 class="spin" size={20} /> Loading glossary…</div>
	{:else if terms.length === 0}
		<div class="empty">
			<BookA size={26} strokeWidth={1.4} />
			<p>No terms yet. Define your business's language so people and AI agents decode it the same way.</p>
			<button class="btn btn--primary" onclick={openNew}><Plus size={16} />Add your first term</button>
		</div>
	{:else}
		<div class="groups">
			{#each grouped as [cat, items] (cat)}
				<div class="cat-group">
					<div class="cat-head">{cat}<span class="cat-count">{items.length}</span></div>
					<div class="grid">
						{#each items as t (t.id)}
							<div class="card">
								<div class="card-head">
									<div class="card-title">
										<div class="card-name">{t.term}</div>
										{#if t.aliases}<div class="card-aliases">aka {t.aliases}</div>{/if}
									</div>
									<div class="card-actions">
										<button class="icon-btn" title="Edit" onclick={() => openEdit(t)}><Pencil size={14} /></button>
										<button class="icon-btn icon-btn--danger" title="Delete" onclick={() => remove(t)}><Trash2 size={14} /></button>
									</div>
								</div>
								<div class="card-def">{t.definition || '—'}</div>
							</div>
						{/each}
					</div>
				</div>
			{/each}
		</div>
	{/if}
</div>

{#if showEdit}
	<div class="overlay" role="button" tabindex="0" onclick={() => (showEdit = false)} onkeydown={(e) => e.key === 'Escape' && (showEdit = false)}>
		<div class="modal" role="dialog" tabindex="-1" onclick={(e) => e.stopPropagation()} onkeydown={() => {}}>
			<div class="modal-head"><h2>{editing ? 'Edit term' : 'New term'}</h2><button class="card-x" onclick={() => (showEdit = false)} aria-label="Close"><X size={18} /></button></div>
			<form onsubmit={save}>
				<label class="field"><span>Term</span><input bind:value={form.term} placeholder="e.g. Proposal" required /></label>
				<label class="field"><span>Definition</span><textarea bind:value={form.definition} rows="4" placeholder="What this means for our business…"></textarea></label>
				<div class="field-row">
					<label class="field"><span>Category</span><input bind:value={form.category} placeholder="e.g. Sales (optional)" /></label>
					<label class="field"><span>Also known as</span><input bind:value={form.aliases} placeholder="alt names (optional)" /></label>
				</div>
				<div class="modal-actions">
					<button type="button" class="btn btn--ghost" onclick={() => (showEdit = false)}>Cancel</button>
					<button type="submit" class="btn btn--primary" disabled={saving || !form.term.trim()}>{#if saving}<Loader2 class="spin" size={15} />{/if}{editing ? 'Save' : 'Add term'}</button>
				</div>
			</form>
		</div>
	</div>
{/if}

<style>
	.gl-root { height: 100%; display: flex; flex-direction: column; background: var(--dbg); color: var(--dt); overflow: hidden; }
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
	.groups { flex: 1; overflow-y: auto; padding: 18px 24px; }
	.cat-group { margin-bottom: 26px; }
	.cat-head { display: flex; align-items: center; gap: 8px; font-size: 0.74rem; font-weight: 640; text-transform: uppercase; letter-spacing: 0.06em; color: var(--dt3); margin-bottom: 12px; padding-bottom: 7px; border-bottom: 1px solid var(--dbd); }
	.cat-count { font-size: 0.68rem; font-weight: 600; color: var(--dt3); background: color-mix(in srgb, var(--dt) 8%, transparent); padding: 1px 7px; border-radius: 999px; }
	.grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(320px, 1fr)); gap: 16px; align-content: start; }
	.card { border: 1px solid var(--dbd); border-radius: 14px; padding: 16px 18px; background: color-mix(in srgb, var(--dt) 2%, transparent); display: flex; flex-direction: column; gap: 10px; transition: border-color 0.13s ease, background 0.13s ease; }
	.card:hover { border-color: color-mix(in srgb, var(--dt) 22%, transparent); background: color-mix(in srgb, var(--dt) 4%, transparent); }
	.card-head { display: flex; align-items: flex-start; justify-content: space-between; gap: 12px; }
	.card-title { min-width: 0; }
	.card-name { font-size: 0.98rem; font-weight: 640; letter-spacing: -0.01em; color: var(--dt); }
	.card-aliases { font-size: 0.72rem; color: var(--dt3); margin-top: 3px; }
	.card-def { font-size: 0.84rem; color: var(--dt2); line-height: 1.55; white-space: pre-wrap; }
	.card-actions { display: flex; gap: 4px; flex-shrink: 0; opacity: 0; transition: opacity 0.12s; }
	.card:hover .card-actions { opacity: 1; }
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
	.field input, .field textarea { padding: 9px 12px; border-radius: 9px; border: 1px solid var(--dbd); background: var(--dbg); color: var(--dt); font-size: 0.86rem; outline: none; font-family: inherit; resize: vertical; }
	.field input:focus, .field textarea:focus { border-color: color-mix(in srgb, var(--dt) 40%, transparent); }
	.field-row { display: flex; gap: 12px; }
	.field-row .field { flex: 1; }
	.modal-actions { display: flex; justify-content: flex-end; gap: 10px; margin-top: 6px; }
	:global(.spin) { animation: spin 0.9s linear infinite; }
	@keyframes spin { to { transform: rotate(360deg); } }
</style>
