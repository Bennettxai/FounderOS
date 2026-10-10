<script lang="ts">
	import { onMount } from 'svelte';
	import { currentWorkspace } from '$lib/stores/workspaces';
	import {
		getBuilders,
		createBuilder,
		updateBuilder,
		deleteBuilder,
		type Builder,
		type BuilderKind,
		type BuilderStatus
	} from '$lib/api/builders';
	import {
		Wrench,
		Plus,
		Loader2,
		X,
		Search,
		Trash2,
		Pencil,
		FormInput,
		Workflow,
		Zap,
		LayoutGrid,
		Globe
	} from 'lucide-svelte';

	const KINDS: { value: BuilderKind; label: string; icon: typeof Wrench }[] = [
		{ value: 'form', label: 'Form', icon: FormInput },
		{ value: 'flow', label: 'Flow', icon: Workflow },
		{ value: 'automation', label: 'Automation', icon: Zap },
		{ value: 'app', label: 'App', icon: LayoutGrid },
		{ value: 'site', label: 'Site', icon: Globe }
	];
	const kindMeta = (k: string) => KINDS.find((x) => x.value === k) ?? KINDS[0];
	const kindLabel = (k: string) => kindMeta(k).label;

	const STATUSES: { value: BuilderStatus; label: string }[] = [
		{ value: 'draft', label: 'Draft' },
		{ value: 'active', label: 'Active' },
		{ value: 'archived', label: 'Archived' }
	];

	let builders = $state<Builder[]>([]);
	let loading = $state(true);
	let error = $state<string | null>(null);
	let query = $state('');
	let kindFilter = $state<BuilderKind | 'all'>('all');

	// Create / edit modal
	let showModal = $state(false);
	let saving = $state(false);
	let editing = $state<Builder | null>(null);
	let form = $state({
		name: '',
		kind: 'form' as BuilderKind,
		description: '',
		status: 'draft' as BuilderStatus
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
			const res = await getBuilders(query.trim() || undefined);
			builders = res.builders;
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to load builders';
		} finally {
			loading = false;
		}
	}

	const filtered = $derived(
		kindFilter === 'all' ? builders : builders.filter((b) => b.kind === kindFilter)
	);

	// Group the filtered builders by kind, preserving the KINDS order.
	const grouped = $derived(
		KINDS.map((k) => ({ ...k, items: filtered.filter((b) => b.kind === k.value) })).filter(
			(g) => g.items.length > 0
		)
	);

	function openCreate() {
		editing = null;
		form = { name: '', kind: 'form', description: '', status: 'draft' };
		showModal = true;
	}
	function openEdit(b: Builder) {
		editing = b;
		form = { name: b.name, kind: b.kind, description: b.description ?? '', status: b.status };
		showModal = true;
	}

	async function save(e: Event) {
		e.preventDefault();
		if (!form.name.trim()) return;
		saving = true;
		error = null;
		try {
			const payload = {
				name: form.name.trim(),
				kind: form.kind,
				description: form.description.trim(),
				status: form.status
			};
			if (editing) {
				await updateBuilder(editing.id, payload);
			} else {
				await createBuilder(payload);
			}
			showModal = false;
			editing = null;
			await load();
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to save builder';
		} finally {
			saving = false;
		}
	}

	async function remove(b: Builder) {
		if (!confirm(`Delete "${b.name}"? This can't be undone.`)) return;
		try {
			await deleteBuilder(b.id);
			builders = builders.filter((x) => x.id !== b.id);
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to delete';
		}
	}
</script>

<svelte:head><title>Builders - BusinessOS</title></svelte:head>

<div class="page">
	<header class="page-header">
		<div class="page-icon"><Wrench size={22} strokeWidth={1.8} /></div>
		<div class="head-text">
			<h1 class="page-title">Builders</h1>
			<p class="page-desc">The tools for creating forms, flows, automations, apps, and sites — a manageable registry of everything your workspace can build.</p>
		</div>
		<div class="head-actions">
			<button class="btn btn-primary" onclick={openCreate}><Plus size={15} /> New builder</button>
		</div>
	</header>

	<div class="toolbar">
		<div class="search">
			<Search size={15} class="search-icon" />
			<input placeholder="Search builders…" bind:value={query} oninput={onSearch} />
			{#if query}<button class="search-clear" onclick={clearSearch}><X size={13} /></button>{/if}
		</div>
		<div class="filters">
			<button class="chip" class:active={kindFilter === 'all'} onclick={() => (kindFilter = 'all')}>All</button>
			{#each KINDS as k}
				{@const Icon = k.icon}
				<button class="chip" class:active={kindFilter === k.value} onclick={() => (kindFilter = k.value)}>
					<Icon size={13} /> {k.label}
				</button>
			{/each}
		</div>
		<span class="count">{filtered.length} builder{filtered.length === 1 ? '' : 's'}</span>
	</div>

	{#if error}
		<div class="error-bar">{error}</div>
	{/if}

	{#if loading}
		<div class="loading"><Loader2 size={22} class="spin" /> Loading builders…</div>
	{:else if filtered.length === 0}
		<div class="empty-state">
			<Wrench size={40} strokeWidth={1.4} class="empty-icon" />
			<p class="empty-title">{builders.length === 0 ? 'No builders yet' : 'No builders match'}</p>
			<p class="empty-body">Builders are the tools for creating forms, flows, automations, apps, and sites. Register one to make it available across your workspace.</p>
			<div class="empty-actions">
				<button class="btn btn-primary" onclick={openCreate}><Plus size={15} /> New builder</button>
			</div>
		</div>
	{:else}
		{#each grouped as group (group.value)}
			{@const GroupIcon = group.icon}
			<section class="group">
				<div class="group-head">
					<GroupIcon size={15} strokeWidth={1.8} />
					<h2>{group.label}</h2>
					<span class="group-count">{group.items.length}</span>
				</div>
				<div class="grid">
					{#each group.items as b (b.id)}
						{@const Icon = kindMeta(b.kind).icon}
						<div class="card">
							<div class="card-top">
								<div class="card-icon"><Icon size={18} strokeWidth={1.7} /></div>
								<div class="card-actions">
									<button class="icon-btn" title="Edit" onclick={() => openEdit(b)}><Pencil size={13} /></button>
									<button class="icon-btn danger" title="Delete" onclick={() => remove(b)}><Trash2 size={13} /></button>
								</div>
							</div>
							<p class="card-name" title={b.name}>{b.name}</p>
							<div class="card-meta">
								<span class="kind-badge">{kindLabel(b.kind)}</span>
								<span class="status-badge status-{b.status}">{b.status}</span>
							</div>
							{#if b.description}
								<p class="card-desc">{b.description}</p>
							{/if}
						</div>
					{/each}
				</div>
			</section>
		{/each}
	{/if}
</div>

{#if showModal}
	<div class="modal-overlay" role="button" tabindex="0" onclick={(e) => { if (e.target === e.currentTarget) showModal = false; }} onkeydown={(e) => e.key === 'Escape' && (showModal = false)}>
		<form class="modal" onsubmit={save}>
			<div class="modal-head">
				<h2>{editing ? 'Edit builder' : 'New builder'}</h2>
				<button type="button" class="icon-btn" onclick={() => (showModal = false)}><X size={16} /></button>
			</div>
			<label class="field">
				<span>Name</span>
				<input bind:value={form.name} placeholder="Lead intake form" required />
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
			<label class="field">
				<span>Description</span>
				<textarea bind:value={form.description} rows="3" placeholder="What this builder creates and when to use it…"></textarea>
			</label>
			<div class="modal-foot">
				<button type="button" class="btn btn-ghost" onclick={() => (showModal = false)}>Cancel</button>
				<button type="submit" class="btn btn-primary" disabled={saving}>
					{#if saving}<Loader2 size={15} class="spin" />{:else if editing}<Pencil size={15} />{:else}<Plus size={15} />{/if}
					{editing ? 'Save' : 'Create builder'}
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
	.page-desc { font-size: 0.875rem; color: var(--dt3); margin: 2px 0 0; max-width: 640px; }
	.head-actions { display: flex; gap: 8px; flex-shrink: 0; }

	.btn { display: inline-flex; align-items: center; gap: 6px; padding: 8px 14px; border-radius: 8px; font-size: 0.84rem; font-weight: 550; cursor: pointer; border: 1px solid transparent; transition: background 0.12s, border-color 0.12s; }
	.btn-primary { background: var(--bos-v2-button-primary); color: var(--bos-v2-button-pureWhiteText); }
	.btn-primary:hover { opacity: 0.9; }
	.btn-ghost { background: transparent; border-color: var(--dbd); color: var(--dt2); }
	.btn-ghost:hover { background: color-mix(in srgb, var(--dt) 5%, transparent); }
	.btn:disabled { opacity: 0.6; cursor: default; }

	.toolbar { display: flex; align-items: center; gap: 12px; flex-wrap: wrap; }
	.search { position: relative; display: flex; align-items: center; flex: 1; min-width: 200px; max-width: 320px; }
	.search :global(.search-icon) { position: absolute; left: 10px; color: var(--dt3); }
	.search input { width: 100%; padding: 8px 30px; border-radius: 8px; border: 1px solid var(--dbd); background: transparent; color: var(--dt); font-size: 0.85rem; }
	.search-clear { position: absolute; right: 8px; background: none; border: none; color: var(--dt3); cursor: pointer; display: flex; }
	.filters { display: flex; align-items: center; gap: 6px; flex-wrap: wrap; }
	.chip { display: inline-flex; align-items: center; gap: 5px; padding: 6px 11px; border-radius: 999px; border: 1px solid var(--dbd); background: transparent; color: var(--dt2); font-size: 0.78rem; font-weight: 550; cursor: pointer; transition: background 0.12s, border-color 0.12s; }
	.chip:hover { background: color-mix(in srgb, var(--dt) 5%, transparent); }
	.chip.active { background: var(--bos-v2-button-primary); color: var(--bos-v2-button-pureWhiteText); border-color: var(--bos-v2-button-primary); }
	.count { font-size: 0.8rem; color: var(--dt3); margin-left: auto; }

	.error-bar { padding: 10px 14px; border-radius: 8px; background: color-mix(in srgb, #ef4444 12%, transparent); color: #ef4444; font-size: 0.83rem; }
	.loading { display: flex; align-items: center; gap: 8px; padding: 48px; justify-content: center; color: var(--dt3); font-size: 0.9rem; }

	.group { display: flex; flex-direction: column; gap: 12px; }
	.group-head { display: flex; align-items: center; gap: 8px; color: var(--dt2); }
	.group-head h2 { font-size: 0.9rem; font-weight: 600; color: var(--dt); margin: 0; }
	.group-count { font-size: 0.72rem; color: var(--dt3); padding: 1px 7px; border-radius: 999px; background: color-mix(in srgb, var(--dt) 8%, transparent); }

	.grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(220px, 1fr)); gap: 14px; }
	.card { border: 1px solid var(--dbd); border-radius: 12px; padding: 14px; background: color-mix(in srgb, var(--dt) 2%, transparent); display: flex; flex-direction: column; gap: 9px; transition: border-color 0.12s; }
	.card:hover { border-color: color-mix(in srgb, var(--dt) 22%, transparent); }
	.card-top { display: flex; align-items: flex-start; justify-content: space-between; }
	.card-icon { width: 34px; height: 34px; border-radius: 9px; border: 1px solid var(--dbd); background: color-mix(in srgb, var(--dt) 5%, transparent); color: var(--dt2); display: flex; align-items: center; justify-content: center; flex-shrink: 0; }
	.card-actions { display: flex; gap: 4px; opacity: 0; transition: opacity 0.12s; }
	.card:hover .card-actions { opacity: 1; }
	.icon-btn { width: 26px; height: 26px; border-radius: 6px; border: 1px solid var(--dbd); background: var(--dbg); color: var(--dt2); display: flex; align-items: center; justify-content: center; cursor: pointer; }
	.icon-btn:hover { background: color-mix(in srgb, var(--dt) 8%, transparent); }
	.icon-btn.danger:hover { color: #ef4444; border-color: #ef4444; }
	.card-name { font-size: 0.9rem; font-weight: 600; color: var(--dt); margin: 0; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
	.card-meta { display: flex; align-items: center; gap: 6px; }
	.kind-badge { font-size: 0.68rem; text-transform: uppercase; letter-spacing: 0.04em; padding: 2px 6px; border-radius: 4px; background: color-mix(in srgb, var(--dt) 8%, transparent); color: var(--dt2); }
	.status-badge { font-size: 0.68rem; text-transform: uppercase; letter-spacing: 0.04em; padding: 2px 6px; border-radius: 4px; border: 1px solid var(--dbd); color: var(--dt3); }
	.status-badge.status-active { color: #16a34a; border-color: color-mix(in srgb, #16a34a 40%, transparent); background: color-mix(in srgb, #16a34a 10%, transparent); }
	.status-badge.status-draft { color: var(--dt2); }
	.status-badge.status-archived { opacity: 0.7; }
	.card-desc { font-size: 0.8rem; color: var(--dt3); margin: 0; line-height: 1.4; display: -webkit-box; -webkit-line-clamp: 3; line-clamp: 3; -webkit-box-orient: vertical; overflow: hidden; }

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
	.field input, .field select, .field textarea { padding: 8px 10px; border-radius: 8px; border: 1px solid var(--dbd); background: transparent; color: var(--dt); font-size: 0.85rem; font-family: inherit; resize: vertical; }
	.field-row { display: flex; gap: 12px; }
	.modal-foot { display: flex; justify-content: flex-end; gap: 8px; margin-top: 4px; }

	:global(.spin) { animation: spin 0.8s linear infinite; }
	@keyframes spin { to { transform: rotate(360deg); } }

	@media (max-width: 768px) { .page { padding: 16px 18px; } .head-actions { flex-wrap: wrap; } }
	@media (max-width: 480px) { .page { padding: 12px 14px; } }
</style>
