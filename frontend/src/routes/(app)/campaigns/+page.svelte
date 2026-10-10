<script lang="ts">
	import { onMount } from 'svelte';
	import { currentWorkspace } from '$lib/stores/workspaces';
	import {
		getCampaigns,
		createCampaign,
		updateCampaign,
		deleteCampaign,
		type Campaign
	} from '$lib/api/campaigns';
	import { Megaphone, Plus, Loader2, X, Search, Trash2, Pencil } from 'lucide-svelte';

	let campaigns = $state<Campaign[]>([]);
	let loading = $state(true);
	let error = $state<string | null>(null);
	let query = $state('');

	let showEdit = $state(false);
	let saving = $state(false);
	let editing = $state<Campaign | null>(null);
	let form = $state({ name: '', channel: 'email', status: 'draft', hook: '', description: '', cta: 'Book a Solutions Call', start_date: '' });

	const CHANNELS = ['email', 'ads', 'sms', 'organic', 'other'];
	const STATUSES = ['draft', 'active', 'paused', 'done'];

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
			const res = await getCampaigns(query.trim() || undefined);
			campaigns = res.campaigns;
		} catch (e) { error = e instanceof Error ? e.message : 'Failed to load campaigns'; }
		finally { loading = false; }
	}

	// Group campaigns by channel, in the canonical channel order (email, ads, sms,
	// organic, other) with any unrecognized channels appended alphabetically.
	const grouped = $derived.by(() => {
		const map = new Map<string, Campaign[]>();
		for (const c of campaigns) {
			const key = (c.channel && c.channel.trim()) || 'other';
			if (!map.has(key)) map.set(key, []);
			map.get(key)!.push(c);
		}
		const known = CHANNELS.filter((ch) => map.has(ch));
		const extra = Array.from(map.keys())
			.filter((k) => !CHANNELS.includes(k))
			.sort((a, b) => a.localeCompare(b));
		return [...known, ...extra].map((ch) => [ch, map.get(ch)!] as [string, Campaign[]]);
	});

	function openNew() {
		editing = null;
		form = { name: '', channel: 'email', status: 'draft', hook: '', description: '', cta: 'Book a Solutions Call', start_date: '' };
		showEdit = true;
	}
	function openEdit(c: Campaign) {
		editing = c;
		form = {
			name: c.name,
			channel: c.channel || 'email',
			status: c.status || 'draft',
			hook: c.hook ?? '',
			description: c.description ?? '',
			cta: c.cta ?? 'Book a Solutions Call',
			start_date: c.start_date ? c.start_date.slice(0, 10) : ''
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
				channel: form.channel,
				status: form.status,
				hook: form.hook.trim(),
				description: form.description.trim(),
				cta: form.cta.trim(),
				start_date: form.start_date || null
			};
			if (editing) await updateCampaign(editing.id, body);
			else await createCampaign(body);
			showEdit = false;
			await load();
		} catch (e) { error = e instanceof Error ? e.message : 'Failed to save campaign'; }
		finally { saving = false; }
	}

	async function remove(c: Campaign) {
		if (!confirm(`Delete "${c.name}"?`)) return;
		try { await deleteCampaign(c.id); campaigns = campaigns.filter((x) => x.id !== c.id); }
		catch (e) { error = e instanceof Error ? e.message : 'Failed to delete'; }
	}

	function fmtDate(d: string | null): string {
		if (!d) return '';
		try { return new Date(d).toLocaleDateString(undefined, { month: 'short', day: 'numeric', year: 'numeric' }); }
		catch { return ''; }
	}
</script>

<svelte:head><title>Campaigns - BusinessOS</title></svelte:head>

<div class="gl-root">
	<header class="topbar">
		<div class="title-wrap"><h1>Campaigns</h1><span class="count">{campaigns.length}</span></div>
		<div class="tools">
			<div class="search">
				<Search size={15} />
				<input bind:value={query} oninput={onSearch} placeholder="Search campaigns…" aria-label="Search campaigns" />
			</div>
			<button class="btn btn--primary" onclick={openNew}><Plus size={16} strokeWidth={2.4} />New campaign</button>
		</div>
	</header>

	{#if error}<div class="banner banner--error">{error}</div>{/if}

	{#if loading}
		<div class="loading"><Loader2 class="spin" size={20} /> Loading campaigns…</div>
	{:else if campaigns.length === 0 && query.trim()}
		<div class="empty">
			<Search size={26} strokeWidth={1.4} />
			<p>No campaigns match "{query.trim()}".</p>
			<button class="btn btn--ghost" onclick={clearSearch}>Clear search</button>
		</div>
	{:else if campaigns.length === 0}
		<div class="empty">
			<Megaphone size={26} strokeWidth={1.4} />
			<p>No campaigns yet. Track your marketing and outreach across email, ads, sms, and organic in one place.</p>
			<button class="btn btn--primary" onclick={openNew}><Plus size={16} />Add your first campaign</button>
		</div>
	{:else}
		<div class="list">
			{#each grouped as [channel, items]}
				<div class="cat">
					<div class="cat-head">{channel}<span class="cat-count">{items.length}</span></div>
					{#each items as c (c.id)}
						<div class="term">
							<div class="term-main">
								<div class="term-name">
									{c.name}
									<span class="badge badge--{c.status}">{c.status}</span>
									{#if c.start_date}<span class="aliases">{fmtDate(c.start_date)}</span>{/if}
								</div>
								{#if c.hook}<div class="term-hook">{c.hook}</div>{/if}
								<div class="term-def">{c.description || '—'}</div>
								{#if c.cta}<div class="term-cta">{c.cta}</div>{/if}
							</div>
							<div class="term-actions">
								<button class="icon-btn" title="Edit" onclick={() => openEdit(c)}><Pencil size={14} /></button>
								<button class="icon-btn icon-btn--danger" title="Delete" onclick={() => remove(c)}><Trash2 size={14} /></button>
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
			<div class="modal-head"><h2>{editing ? 'Edit campaign' : 'New campaign'}</h2><button class="card-x" onclick={() => (showEdit = false)} aria-label="Close"><X size={18} /></button></div>
			<form onsubmit={save}>
				<label class="field"><span>Name</span><input bind:value={form.name} placeholder="e.g. Spring launch" required /></label>
				<div class="field-row">
					<label class="field"><span>Channel</span>
						<select bind:value={form.channel}>
							{#each CHANNELS as ch}<option value={ch}>{ch}</option>{/each}
						</select>
					</label>
					<label class="field"><span>Status</span>
						<select bind:value={form.status}>
							{#each STATUSES as st}<option value={st}>{st}</option>{/each}
						</select>
					</label>
				</div>
				<label class="field"><span>Hook</span><input bind:value={form.hook} placeholder="e.g. Your agency has an operating system problem" /></label>
				<label class="field"><span>Start date</span><input type="date" bind:value={form.start_date} /></label>
				<label class="field"><span>Description</span><textarea bind:value={form.description} rows="4" placeholder="What this campaign is about / the angle…"></textarea></label>
				<label class="field"><span>CTA</span><input bind:value={form.cta} placeholder="e.g. Book a Solutions Call" /></label>
				<div class="modal-actions">
					<button type="button" class="btn btn--ghost" onclick={() => (showEdit = false)}>Cancel</button>
					<button type="submit" class="btn btn--primary" disabled={saving || !form.name.trim()}>{#if saving}<Loader2 class="spin" size={15} />{/if}{editing ? 'Save' : 'Add campaign'}</button>
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
	.list { flex: 1; overflow-y: auto; padding: 18px 24px; max-width: 820px; width: 100%; margin: 0 auto; }
	.cat { margin-bottom: 24px; }
	.cat-head { display: flex; align-items: center; gap: 8px; font-size: 0.74rem; font-weight: 640; color: var(--dt3); text-transform: uppercase; letter-spacing: 0.06em; margin-bottom: 10px; padding-bottom: 7px; border-bottom: 1px solid var(--dbd); }
	.cat-count { font-size: 0.68rem; font-weight: 600; color: var(--dt4); letter-spacing: 0; background: color-mix(in srgb, var(--dt) 8%, transparent); padding: 1px 7px; border-radius: 999px; }
	.term { display: flex; align-items: flex-start; gap: 14px; padding: 12px 4px; }
	.term:not(:last-child) { border-bottom: 1px solid color-mix(in srgb, var(--dt) 5%, transparent); }
	.term-main { flex: 1; min-width: 0; }
	.term-name { font-size: 0.92rem; font-weight: 620; display: flex; align-items: baseline; gap: 8px; flex-wrap: wrap; }
	.aliases { font-size: 0.72rem; font-weight: 480; color: var(--dt4); }
	.term-hook { font-size: 0.85rem; font-weight: 560; color: var(--dt); line-height: 1.5; margin-top: 4px; }
	.term-def { font-size: 0.84rem; color: var(--dt2); line-height: 1.55; margin-top: 3px; white-space: pre-wrap; }
	.term-cta { font-size: 0.74rem; font-weight: 600; color: var(--dt3); margin-top: 6px; }
	.term-actions { display: flex; gap: 4px; flex-shrink: 0; opacity: 0; transition: opacity 0.12s; }
	.term:hover .term-actions { opacity: 1; }
	.icon-btn { display: inline-flex; align-items: center; justify-content: center; width: 28px; height: 28px; border-radius: 7px; border: none; background: transparent; color: var(--dt3); cursor: pointer; }
	.icon-btn:hover { background: color-mix(in srgb, var(--dt) 8%, transparent); color: var(--dt); }
	.icon-btn--danger:hover { background: color-mix(in srgb, #ef4444 12%, transparent); color: #ef4444; }
	.badge { font-size: 0.66rem; font-weight: 640; text-transform: uppercase; letter-spacing: 0.04em; padding: 2px 8px; border-radius: 999px; background: color-mix(in srgb, var(--dt) 10%, transparent); color: var(--dt3); }
	.badge--active { background: color-mix(in srgb, #22c55e 16%, transparent); color: #22c55e; }
	.badge--paused { background: color-mix(in srgb, #f59e0b 16%, transparent); color: #f59e0b; }
	.badge--done { background: color-mix(in srgb, #6366f1 16%, transparent); color: #818cf8; }
	.badge--draft { background: color-mix(in srgb, var(--dt) 10%, transparent); color: var(--dt3); }
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
