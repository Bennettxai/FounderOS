<script lang="ts">
	import { onMount } from 'svelte';
	import { currentWorkspace } from '$lib/stores/workspaces';
	import {
		getOffers,
		createOffer,
		updateOffer,
		deleteOffer,
		type Offer,
		type OfferStatus,
		type OfferTier
	} from '$lib/api/offers';
	import { Tag, Plus, Loader2, X, Search, Trash2, Pencil } from 'lucide-svelte';

	// The Growth System Audit offer ladder. Order here is the buyer journey.
	const TIERS: { value: OfferTier; label: string }[] = [
		{ value: 'audit', label: 'Entry Audit' },
		{ value: 'phase-1', label: 'Phase 1 Build' },
		{ value: 'phase-2', label: 'Phase 2 Retainer' },
		{ value: 'lane', label: 'Lanes' }
	];
	const tierLabel = (t: string) => TIERS.find((x) => x.value === t)?.label ?? t;

	let offers = $state<Offer[]>([]);
	let loading = $state(true);
	let error = $state<string | null>(null);
	let query = $state('');

	let showEdit = $state(false);
	let saving = $state(false);
	let editing = $state<Offer | null>(null);
	let form = $state({
		name: '',
		tier: 'audit' as OfferTier,
		price: '',
		status: 'active' as OfferStatus,
		promise: '',
		description: '',
		includes: '',
		cta: 'Book a Solutions Call'
	});

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
			const res = await getOffers(query.trim() || undefined);
			offers = res.offers;
		} catch (e) { error = e instanceof Error ? e.message : 'Failed to load offers'; }
		finally { loading = false; }
	}

	// Group offers by ladder tier in the canonical buyer-journey order.
	const grouped = $derived.by(() =>
		TIERS.map((t) => ({ tier: t, items: offers.filter((o) => (o.tier || 'audit') === t.value) })).filter(
			(g) => g.items.length > 0
		)
	);

	function openNew() {
		editing = null;
		form = {
			name: '',
			tier: 'audit',
			price: '',
			status: 'active',
			promise: '',
			description: '',
			includes: '',
			cta: 'Book a Solutions Call'
		};
		showEdit = true;
	}
	function openEdit(o: Offer) {
		editing = o;
		form = {
			name: o.name,
			tier: o.tier ?? 'audit',
			price: o.price ?? '',
			status: o.status ?? 'active',
			promise: o.promise ?? '',
			description: o.description ?? '',
			includes: o.includes ?? '',
			cta: o.cta ?? 'Book a Solutions Call'
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
				tier: form.tier,
				price: form.price.trim(),
				status: form.status,
				promise: form.promise.trim(),
				description: form.description.trim(),
				includes: form.includes.trim(),
				cta: form.cta.trim()
			};
			if (editing) await updateOffer(editing.id, body);
			else await createOffer(body);
			showEdit = false;
			await load();
		} catch (e) { error = e instanceof Error ? e.message : 'Failed to save offer'; }
		finally { saving = false; }
	}

	async function remove(o: Offer) {
		if (!confirm(`Delete "${o.name}"?`)) return;
		try { await deleteOffer(o.id); offers = offers.filter((x) => x.id !== o.id); }
		catch (e) { error = e instanceof Error ? e.message : 'Failed to delete'; }
	}

	function includesList(s: string): string[] {
		return s.split(/\r?\n|,/).map((x) => x.trim()).filter(Boolean);
	}
</script>

<svelte:head><title>Offers - BusinessOS</title></svelte:head>

<div class="of-root">
	<header class="topbar">
		<div class="title-wrap"><h1>Offers</h1><span class="count">{offers.length}</span></div>
		<div class="tools">
			<div class="search">
				<Search size={15} />
				<input bind:value={query} oninput={onSearch} placeholder="Search offers…" />
			</div>
			<button class="btn btn--primary" onclick={openNew}><Plus size={16} strokeWidth={2.4} />New offer</button>
		</div>
	</header>

	{#if error}<div class="banner banner--error">{error}</div>{/if}

	{#if loading}
		<div class="loading"><Loader2 class="spin" size={20} /> Loading offers…</div>
	{:else if offers.length === 0 && query.trim()}
		<div class="empty">
			<Search size={26} strokeWidth={1.4} />
			<p>No offers match "{query.trim()}".</p>
			<button class="btn btn--ghost" onclick={clearSearch}>Clear search</button>
		</div>
	{:else if offers.length === 0}
		<div class="empty">
			<Tag size={26} strokeWidth={1.4} />
			<p>No offers yet. Define what your business sells so people and AI agents know your offer stack.</p>
			<button class="btn btn--primary" onclick={openNew}><Plus size={16} />Add your first offer</button>
		</div>
	{:else}
		<div class="list">
			{#each grouped as g (g.tier.value)}
				<div class="cat">
					<div class="cat-head">{g.tier.label}<span class="cat-count">{g.items.length}</span></div>
					{#each g.items as o (o.id)}
						<div class="offer">
							<div class="offer-main">
								<div class="offer-head">
									<span class="offer-name">{o.name}</span>
									<span class="status status--{o.status}">{o.status}</span>
									{#if o.price}<span class="offer-price">{o.price}</span>{/if}
								</div>
								{#if o.promise}<div class="offer-promise">{o.promise}</div>{/if}
								{#if o.description}<div class="offer-desc">{o.description}</div>{/if}
								{#if o.includes}
									<ul class="includes">
										{#each includesList(o.includes) as item}<li>{item}</li>{/each}
									</ul>
								{/if}
								{#if o.cta}<div class="offer-cta">{o.cta}</div>{/if}
							</div>
							<div class="offer-actions">
								<button class="icon-btn" title="Edit" onclick={() => openEdit(o)}><Pencil size={14} /></button>
								<button class="icon-btn icon-btn--danger" title="Delete" onclick={() => remove(o)}><Trash2 size={14} /></button>
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
			<div class="modal-head"><h2>{editing ? 'Edit offer' : 'New offer'}</h2><button class="card-x" onclick={() => (showEdit = false)} aria-label="Close"><X size={18} /></button></div>
			<form onsubmit={save}>
				<label class="field"><span>Name</span><input bind:value={form.name} placeholder="e.g. Growth System Audit" required /></label>
				<div class="field-row">
					<label class="field"><span>Tier</span>
						<select bind:value={form.tier}>
							{#each TIERS as t}<option value={t.value}>{t.label}</option>{/each}
						</select>
					</label>
					<label class="field"><span>Status</span>
						<select bind:value={form.status}>
							<option value="active">Active</option>
							<option value="draft">Draft</option>
							<option value="archived">Archived</option>
						</select>
					</label>
				</div>
				<label class="field"><span>Price</span><input bind:value={form.price} placeholder="e.g. $10k+ depending on scope" /></label>
				<label class="field"><span>Promise</span><input bind:value={form.promise} placeholder="The outcome the buyer is paying for…" /></label>
				<label class="field"><span>Description</span><textarea bind:value={form.description} rows="3" placeholder="What this offer is…"></textarea></label>
				<label class="field"><span>What's included</span><textarea bind:value={form.includes} rows="4" placeholder="One per line, or comma-separated…"></textarea></label>
				<label class="field"><span>CTA</span><input bind:value={form.cta} placeholder="e.g. Book a Solutions Call" /></label>
				<div class="modal-actions">
					<button type="button" class="btn btn--ghost" onclick={() => (showEdit = false)}>Cancel</button>
					<button type="submit" class="btn btn--primary" disabled={saving || !form.name.trim()}>{#if saving}<Loader2 class="spin" size={15} />{/if}{editing ? 'Save' : 'Add offer'}</button>
				</div>
			</form>
		</div>
	</div>
{/if}

<style>
	.of-root { height: 100%; display: flex; flex-direction: column; background: var(--dbg); color: var(--dt); overflow: hidden; }
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
	.cat-head { display: flex; align-items: center; gap: 8px; font-size: 0.74rem; font-weight: 640; color: var(--dt3); text-transform: uppercase; letter-spacing: 0.06em; margin-bottom: 10px; padding-bottom: 7px; border-bottom: 1px solid var(--dbd); }
	.cat-count { font-size: 0.68rem; font-weight: 600; color: var(--dt3); background: color-mix(in srgb, var(--dt) 8%, transparent); padding: 1px 7px; border-radius: 999px; text-transform: none; letter-spacing: 0; }
	.offer { display: flex; align-items: flex-start; gap: 14px; padding: 16px; border: 1px solid var(--dbd); border-radius: 12px; margin-bottom: 12px; }
	.offer-promise { font-size: 0.86rem; font-weight: 540; color: var(--dt); line-height: 1.5; margin-top: 7px; }
	.offer-cta { font-size: 0.76rem; font-weight: 600; color: var(--dt3); margin-top: 10px; }
	.offer-main { flex: 1; min-width: 0; }
	.offer-head { display: flex; align-items: center; gap: 10px; flex-wrap: wrap; }
	.offer-name { font-size: 0.96rem; font-weight: 640; }
	.offer-price { font-size: 0.86rem; font-weight: 580; color: var(--dt2); margin-left: auto; }
	.status { font-size: 0.68rem; font-weight: 620; text-transform: uppercase; letter-spacing: 0.05em; padding: 2px 8px; border-radius: 999px; }
	.status--active { background: color-mix(in srgb, #22c55e 16%, transparent); color: #22c55e; }
	.status--draft { background: color-mix(in srgb, var(--dt) 10%, transparent); color: var(--dt3); }
	.status--archived { background: color-mix(in srgb, #f59e0b 14%, transparent); color: #f59e0b; }
	.offer-desc { font-size: 0.84rem; color: var(--dt2); line-height: 1.55; margin-top: 7px; white-space: pre-wrap; }
	.includes { margin: 10px 0 0; padding-left: 18px; display: flex; flex-direction: column; gap: 4px; }
	.includes li { font-size: 0.82rem; color: var(--dt2); line-height: 1.45; }
	.offer-actions { display: flex; gap: 4px; flex-shrink: 0; opacity: 0; transition: opacity 0.12s; }
	.offer:hover .offer-actions { opacity: 1; }
	.icon-btn { display: inline-flex; align-items: center; justify-content: center; width: 28px; height: 28px; border-radius: 7px; border: none; background: transparent; color: var(--dt3); cursor: pointer; }
	.icon-btn:hover { background: color-mix(in srgb, var(--dt) 8%, transparent); color: var(--dt); }
	.icon-btn--danger:hover { background: color-mix(in srgb, #ef4444 12%, transparent); color: #ef4444; }
	.overlay { position: fixed; inset: 0; background: rgba(0,0,0,0.55); display: flex; align-items: center; justify-content: center; z-index: 100; padding: 20px; }
	.modal { width: 100%; max-width: 480px; max-height: 90vh; overflow-y: auto; background: var(--dbg); border: 1px solid var(--dbd); border-radius: 16px; padding: 22px; box-shadow: 0 24px 60px rgba(0,0,0,0.5); }
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
