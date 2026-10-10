<script lang="ts">
	import { onMount } from 'svelte';
	import { currentWorkspace } from '$lib/stores/workspaces';
	import {
		getPersonas,
		createPersona,
		updatePersona,
		deletePersona,
		type Persona,
		type PersonaFit
	} from '$lib/api/personas';
	import { Contact, Plus, Loader2, X, Search, Trash2, Pencil } from 'lucide-svelte';

	// best-fit vs poor-fit, straight from the offer one-pager's "Who it is for".
	const FITS: { value: PersonaFit; label: string; heading: string }[] = [
		{ value: 'best', label: 'Best fit', heading: 'Best fit' },
		{ value: 'poor', label: 'Poor fit', heading: 'Poor fit (disqualify)' }
	];

	let personas = $state<Persona[]>([]);
	let loading = $state(true);
	let error = $state<string | null>(null);
	let query = $state('');

	let showEdit = $state(false);
	let saving = $state(false);
	let editing = $state<Persona | null>(null);
	let form = $state({ name: '', segment: '', fit: 'best' as PersonaFit, pains: '', objections: '', language: '', notes: '' });

	// Group personas best-fit first, then poor-fit.
	const grouped = $derived.by(() =>
		FITS.map((f) => ({ fit: f, items: personas.filter((p) => (p.fit || 'best') === f.value) })).filter(
			(g) => g.items.length > 0
		)
	);

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
			const res = await getPersonas(query.trim() || undefined);
			personas = res.personas;
		} catch (e) { error = e instanceof Error ? e.message : 'Failed to load personas'; }
		finally { loading = false; }
	}

	function openNew() {
		editing = null;
		form = { name: '', segment: '', fit: 'best', pains: '', objections: '', language: '', notes: '' };
		showEdit = true;
	}
	function openEdit(p: Persona) {
		editing = p;
		form = {
			name: p.name,
			segment: p.segment ?? '',
			fit: p.fit ?? 'best',
			pains: p.pains ?? '',
			objections: p.objections ?? '',
			language: p.language ?? '',
			notes: p.notes ?? ''
		};
		showEdit = true;
	}

	async function save(e: Event) {
		e.preventDefault();
		if (!form.name.trim()) return;
		saving = true; error = null;
		try {
			const body = {
				name: form.name.trim(),
				segment: form.segment.trim(),
				fit: form.fit,
				pains: form.pains.trim(),
				objections: form.objections.trim(),
				language: form.language.trim(),
				notes: form.notes.trim()
			};
			if (editing) await updatePersona(editing.id, body);
			else await createPersona(body);
			showEdit = false;
			await load();
		} catch (e) { error = e instanceof Error ? e.message : 'Failed to save persona'; }
		finally { saving = false; }
	}

	async function remove(p: Persona) {
		if (!confirm(`Delete "${p.name}"?`)) return;
		try { await deletePersona(p.id); personas = personas.filter((x) => x.id !== p.id); }
		catch (e) { error = e instanceof Error ? e.message : 'Failed to delete'; }
	}
</script>

<svelte:head><title>Personas - BusinessOS</title></svelte:head>

<div class="pe-root">
	<header class="topbar">
		<div class="title-wrap"><h1>Personas</h1><span class="count">{personas.length}</span></div>
		<div class="tools">
			<div class="search">
				<Search size={15} />
				<input bind:value={query} oninput={onSearch} placeholder="Search personas…" aria-label="Search personas" />
			</div>
			<button class="btn btn--primary" onclick={openNew}><Plus size={16} strokeWidth={2.4} />New persona</button>
		</div>
	</header>

	{#if error}<div class="banner banner--error">{error}</div>{/if}

	{#if loading}
		<div class="loading"><Loader2 class="spin" size={20} /> Loading personas…</div>
	{:else if personas.length === 0 && query.trim()}
		<div class="empty">
			<Search size={26} strokeWidth={1.4} />
			<p>No personas match "{query.trim()}".</p>
			<button class="btn btn--ghost" onclick={clearSearch}>Clear search</button>
		</div>
	{:else if personas.length === 0}
		<div class="empty">
			<Contact size={26} strokeWidth={1.4} />
			<p>No personas yet. Define your ideal customer profiles so people and AI agents target messaging and offers the same way.</p>
			<button class="btn btn--primary" onclick={openNew}><Plus size={16} />Add your first persona</button>
		</div>
	{:else}
		<div class="groups">
			{#each grouped as g (g.fit.value)}
				<div class="fit-group">
					<div class="fit-head fit-head--{g.fit.value}">{g.fit.heading}<span class="fit-count">{g.items.length}</span></div>
					<div class="grid">
						{#each g.items as p (p.id)}
							<div class="card">
					<div class="card-head">
						<div class="card-title">
							<div class="card-name">{p.name}</div>
							{#if p.segment}<div class="card-segment">{p.segment}</div>{/if}
						</div>
						<div class="card-actions">
							<button class="icon-btn" title="Edit" onclick={() => openEdit(p)}><Pencil size={14} /></button>
							<button class="icon-btn icon-btn--danger" title="Delete" onclick={() => remove(p)}><Trash2 size={14} /></button>
						</div>
					</div>
					{#if p.pains}
						<div class="card-field"><div class="card-label">Pains</div><div class="card-body">{p.pains}</div></div>
					{/if}
					{#if p.objections}
						<div class="card-field"><div class="card-label">Objections</div><div class="card-body">{p.objections}</div></div>
					{/if}
					{#if p.language}
						<div class="card-field"><div class="card-label">Language</div><div class="card-body">{p.language}</div></div>
					{/if}
					{#if p.notes}
						<div class="card-field"><div class="card-label">Notes</div><div class="card-body">{p.notes}</div></div>
					{/if}
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
			<div class="modal-head"><h2>{editing ? 'Edit persona' : 'New persona'}</h2><button class="card-x" onclick={() => (showEdit = false)} aria-label="Close"><X size={18} /></button></div>
			<form onsubmit={save}>
				<div class="field-row">
					<label class="field"><span>Name</span><input bind:value={form.name} placeholder="e.g. Scaling agency owner" required /></label>
					<label class="field"><span>Segment</span><input bind:value={form.segment} placeholder="e.g. agencies on GHL/HubSpot" /></label>
				</div>
				<label class="field"><span>Fit</span>
					<select bind:value={form.fit}>
						{#each FITS as f}<option value={f.value}>{f.label}</option>{/each}
					</select>
				</label>
				<label class="field"><span>Pains</span><textarea bind:value={form.pains} rows="3" placeholder="What keeps them up at night…"></textarea></label>
				<label class="field"><span>Objections</span><textarea bind:value={form.objections} rows="3" placeholder="Why they hesitate to buy…"></textarea></label>
				<label class="field"><span>Language</span><textarea bind:value={form.language} rows="2" placeholder="Words and phrases they use…"></textarea></label>
				<label class="field"><span>Notes</span><textarea bind:value={form.notes} rows="2" placeholder="Anything else (optional)…"></textarea></label>
				<div class="modal-actions">
					<button type="button" class="btn btn--ghost" onclick={() => (showEdit = false)}>Cancel</button>
					<button type="submit" class="btn btn--primary" disabled={saving || !form.name.trim()}>{#if saving}<Loader2 class="spin" size={15} />{/if}{editing ? 'Save' : 'Add persona'}</button>
				</div>
			</form>
		</div>
	</div>
{/if}

<style>
	.pe-root { height: 100%; display: flex; flex-direction: column; background: var(--dbg); color: var(--dt); overflow: hidden; }
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
	.empty p { max-width: 440px; line-height: 1.5; }
	.banner { margin: 14px 24px 0; padding: 11px 14px; border-radius: 10px; font-size: 0.83rem; }
	.banner--error { background: color-mix(in srgb, #ef4444 12%, transparent); color: #ef4444; }
	.groups { flex: 1; overflow-y: auto; padding: 18px 24px; }
	.fit-group { margin-bottom: 26px; }
	.fit-head { display: flex; align-items: center; gap: 8px; font-size: 0.74rem; font-weight: 640; text-transform: uppercase; letter-spacing: 0.06em; color: var(--dt3); margin-bottom: 12px; padding-bottom: 7px; border-bottom: 1px solid var(--dbd); }
	.fit-head--best { color: #22c55e; }
	.fit-head--poor { color: #f59e0b; }
	.fit-count { font-size: 0.68rem; font-weight: 600; color: var(--dt3); background: color-mix(in srgb, var(--dt) 8%, transparent); padding: 1px 7px; border-radius: 999px; }
	.grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(320px, 1fr)); gap: 16px; align-content: start; }
	.card { border: 1px solid var(--dbd); border-radius: 14px; padding: 16px 18px; background: color-mix(in srgb, var(--dt) 2%, transparent); display: flex; flex-direction: column; gap: 12px; }
	.card-head { display: flex; align-items: flex-start; justify-content: space-between; gap: 12px; }
	.card-title { min-width: 0; }
	.card-name { font-size: 0.98rem; font-weight: 640; letter-spacing: -0.01em; }
	.card-segment { font-size: 0.76rem; color: var(--dt3); margin-top: 2px; }
	.card-actions { display: flex; gap: 4px; flex-shrink: 0; opacity: 0; transition: opacity 0.12s; }
	.card:hover .card-actions { opacity: 1; }
	.card-field { display: flex; flex-direction: column; gap: 3px; }
	.card-label { font-size: 0.68rem; font-weight: 640; color: var(--dt3); text-transform: uppercase; letter-spacing: 0.06em; }
	.card-body { font-size: 0.84rem; color: var(--dt2); line-height: 1.5; white-space: pre-wrap; }
	.icon-btn { display: inline-flex; align-items: center; justify-content: center; width: 28px; height: 28px; border-radius: 7px; border: none; background: transparent; color: var(--dt3); cursor: pointer; }
	.icon-btn:hover { background: color-mix(in srgb, var(--dt) 8%, transparent); color: var(--dt); }
	.icon-btn--danger:hover { background: color-mix(in srgb, #ef4444 12%, transparent); color: #ef4444; }
	.overlay { position: fixed; inset: 0; background: rgba(0,0,0,0.55); display: flex; align-items: center; justify-content: center; z-index: 100; padding: 20px; }
	.modal { width: 100%; max-width: 520px; max-height: 90vh; overflow-y: auto; background: var(--dbg); border: 1px solid var(--dbd); border-radius: 16px; padding: 22px; box-shadow: 0 24px 60px rgba(0,0,0,0.5); }
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
