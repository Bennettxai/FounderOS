<script lang="ts">
	import { onMount } from 'svelte';
	import { api } from '$lib/api';
	import { currentWorkspace } from '$lib/stores/workspaces';
	import type {
		ClientResponse,
		ClientDetailResponse,
		CreateClientData,
		ContactResponse,
		CreateContactData
	} from '$lib/api/clients/types';
	import {
		LayoutGrid,
		List,
		Plus,
		Loader2,
		X,
		Trash2,
		Search,
		Building2,
		MapPin,
		Star,
		CalendarClock,
		Pencil,
		Mail,
		Phone,
		Globe,
		Users,
		UserPlus,
		BadgeCheck,
		Link2
	} from 'lucide-svelte';
	import {
		PIPELINE_STAGES,
		DEFAULT_STAGE,
		AGENCY_TYPES,
		PHYSICAL_OFFICE_STATUS,
		OUTREACH_STATUS,
		MEETING_PREFERENCE,
		PAIN_CATEGORY,
		OFFER_FIT,
		FIT_SCORES,
		TOOL_STACK_FIELDS,
		CF,
		optionLabel,
		type PipelineStage
	} from './field-pack';

	// A lead row is the API client record; agency fields ride in custom_fields.
	type Lead = ClientResponse & { custom_fields: Record<string, unknown> };

	let leads = $state<Lead[]>([]);
	let loading = $state(true);
	let error = $state<string | null>(null);
	let view = $state<'board' | 'list'>('board');
	let query = $state('');
	let agencyFilter = $state<string>('');
	let busyId = $state<string | null>(null);
	let draggedLead = $state<Lead | null>(null);
	let dragOverCol = $state<string | null>(null);

	// ---- create / edit modal state ----
	let showForm = $state(false);
	let saving = $state(false);
	let editingId = $state<string | null>(null);

	// ---- detail drawer state ----
	let showDetail = $state(false);
	let detailLoading = $state(false);
	let detailLead = $state<(ClientDetailResponse & { custom_fields: Record<string, unknown> }) | null>(
		null
	);
	let contacts = $state<ContactResponse[]>([]);

	// ---- contact form state (inside detail drawer) ----
	let showContactForm = $state(false);
	let contactSaving = $state(false);
	let contactEditingId = $state<string | null>(null);
	let contactBusyId = $state<string | null>(null);

	type ContactForm = {
		name: string;
		role: string;
		email: string;
		phone: string;
		is_primary: boolean;
		notes: string;
	};
	function emptyContact(): ContactForm {
		return { name: '', role: '', email: '', phone: '', is_primary: false, notes: '' };
	}
	let contactForm = $state<ContactForm>(emptyContact());

	type FormState = {
		// core columns
		name: string;
		website: string;
		city: string;
		state: string;
		email: string;
		phone: string;
		source: string;
		notes: string;
		// agency custom fields
		agency_type: string;
		pipeline_stage: PipelineStage;
		physical_office_status: string;
		outreach_status: string;
		meeting_preference: string;
		offer_fit: string;
		fit_score: string;
		fit_score_n: number | '';
		google_maps_url: string;
		linkedin_url: string;
		owner_name: string;
		operator_name: string;
		lead_owner: string;
		next_step_date: string;
		pain_category: string[];
		tool_stack: Record<string, string>;
		// required notes block
		who_they_serve: string;
		why_they_care: string;
		proof_of_activity: string;
		likely_pain: string;
		next_action: string;
	};

	function emptyForm(): FormState {
		return {
			name: '',
			website: '',
			city: '',
			state: '',
			email: '',
			phone: '',
			source: '',
			notes: '',
			agency_type: '',
			pipeline_stage: DEFAULT_STAGE,
			physical_office_status: '',
			outreach_status: '',
			meeting_preference: '',
			offer_fit: '',
			fit_score: '',
			fit_score_n: '',
			google_maps_url: '',
			linkedin_url: '',
			owner_name: '',
			operator_name: '',
			lead_owner: '',
			next_step_date: '',
			pain_category: [],
			tool_stack: {},
			who_they_serve: '',
			why_they_care: '',
			proof_of_activity: '',
			likely_pain: '',
			next_action: ''
		};
	}

	let form = $state<FormState>(emptyForm());

	// Reload when workspace changes.
	let wsId = $state<string | null | undefined>(null);
	$effect(() => {
		const id = $currentWorkspace?.id ?? null;
		if (id !== wsId) {
			wsId = id;
			load();
		}
	});

	onMount(load);

	function cf(l: Lead, key: string): string {
		const v = l.custom_fields?.[key];
		return v == null ? '' : String(v);
	}
	function cfArr(l: Lead, key: string): string[] {
		const v = l.custom_fields?.[key];
		return Array.isArray(v) ? (v as string[]) : [];
	}
	function leadStage(l: Lead): PipelineStage {
		const s = cf(l, CF.pipelineStage);
		return (PIPELINE_STAGES.find((x) => x.id === s)?.id ?? DEFAULT_STAGE) as PipelineStage;
	}
	function fitScore(l: Lead): number {
		const n = Number(l.custom_fields?.[CF.fitScore]);
		return Number.isFinite(n) ? n : 0;
	}

	async function load() {
		loading = true;
		error = null;
		try {
			leads = (await api.getClients()) as unknown as Lead[];
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to load leads';
		} finally {
			loading = false;
		}
	}

	const filtered = $derived.by(() => {
		const q = query.trim().toLowerCase();
		return leads.filter((l) => {
			if (agencyFilter && cf(l, CF.agencyType) !== agencyFilter) return false;
			if (!q) return true;
			const hay = `${l.name} ${l.email ?? ''} ${l.city ?? ''} ${l.state ?? ''} ${cf(l, CF.ownerName)} ${cf(l, CF.operatorName)}`.toLowerCase();
			return hay.includes(q);
		});
	});

	function byStage(s: PipelineStage): Lead[] {
		return filtered.filter((l) => leadStage(l) === s);
	}

	// Open the read-only detail drawer; loads the full record (incl. contacts).
	async function openDetail(id: string) {
		showDetail = true;
		detailLoading = true;
		showContactForm = false;
		contactEditingId = null;
		error = null;
		try {
			const full = (await api.getClient(id)) as unknown as ClientDetailResponse & {
				custom_fields: Record<string, unknown>;
			};
			detailLead = full;
			contacts = full.contacts ?? [];
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to load lead';
			showDetail = false;
		} finally {
			detailLoading = false;
		}
	}

	function closeDetail() {
		showDetail = false;
		detailLead = null;
		contacts = [];
		showContactForm = false;
		contactEditingId = null;
	}

	// Populate the edit form from an already-loaded record.
	function populateForm(full: Lead) {
		const c = full.custom_fields ?? {};
		form = {
			name: full.name ?? '',
			website: full.website ?? '',
			city: full.city ?? '',
			state: full.state ?? '',
			email: full.email ?? '',
			phone: full.phone ?? '',
			source: full.source ?? '',
			notes: full.notes ?? '',
			agency_type: String(c[CF.agencyType] ?? ''),
			pipeline_stage: (PIPELINE_STAGES.find((x) => x.id === c[CF.pipelineStage])?.id ??
				DEFAULT_STAGE) as PipelineStage,
			physical_office_status: String(c[CF.physicalOfficeStatus] ?? ''),
			outreach_status: String(c[CF.outreachStatus] ?? ''),
			meeting_preference: String(c[CF.meetingPreference] ?? ''),
			offer_fit: String(c[CF.offerFit] ?? ''),
			fit_score: String(c[CF.fitScore] ?? ''),
			fit_score_n: c[CF.fitScore] != null && c[CF.fitScore] !== '' ? Number(c[CF.fitScore]) : '',
			google_maps_url: String(c[CF.googleMapsUrl] ?? ''),
			linkedin_url: String(c[CF.linkedinUrl] ?? ''),
			owner_name: String(c[CF.ownerName] ?? ''),
			operator_name: String(c[CF.operatorName] ?? ''),
			lead_owner: String(c[CF.leadOwner] ?? ''),
			next_step_date: String(c[CF.nextStepDate] ?? ''),
			pain_category: Array.isArray(c[CF.painCategory]) ? (c[CF.painCategory] as string[]) : [],
			tool_stack: (c[CF.toolStack] as Record<string, string>) ?? {},
			who_they_serve: String(c[CF.whoTheyServe] ?? ''),
			why_they_care: String(c[CF.whyTheyCare] ?? ''),
			proof_of_activity: String(c[CF.proofOfActivity] ?? ''),
			likely_pain: String(c[CF.likelyPain] ?? ''),
			next_action: String(c[CF.nextAction] ?? '')
		};
	}

	// Edit from within the detail drawer (record already loaded).
	function editFromDetail() {
		if (!detailLead) return;
		populateForm(detailLead as unknown as Lead);
		editingId = detailLead.id;
		showForm = true;
	}

	function openCreate() {
		form = emptyForm();
		editingId = null;
		showForm = true;
	}

	function buildPayload(): CreateClientData {
		const tool_stack: Record<string, string> = {};
		for (const f of TOOL_STACK_FIELDS) {
			const v = (form.tool_stack[f.key] ?? '').trim();
			if (v) tool_stack[f.key] = v;
		}
		const custom_fields: Record<string, unknown> = {
			[CF.agencyType]: form.agency_type || undefined,
			[CF.pipelineStage]: form.pipeline_stage,
			[CF.physicalOfficeStatus]: form.physical_office_status || undefined,
			[CF.outreachStatus]: form.outreach_status || undefined,
			[CF.meetingPreference]: form.meeting_preference || undefined,
			[CF.offerFit]: form.offer_fit || undefined,
			[CF.fitScore]: form.fit_score_n === '' ? undefined : Number(form.fit_score_n),
			[CF.googleMapsUrl]: form.google_maps_url || undefined,
			[CF.linkedinUrl]: form.linkedin_url || undefined,
			[CF.ownerName]: form.owner_name || undefined,
			[CF.operatorName]: form.operator_name || undefined,
			[CF.leadOwner]: form.lead_owner || undefined,
			[CF.nextStepDate]: form.next_step_date || undefined,
			[CF.painCategory]: form.pain_category.length ? form.pain_category : undefined,
			[CF.toolStack]: Object.keys(tool_stack).length ? tool_stack : undefined,
			[CF.whoTheyServe]: form.who_they_serve || undefined,
			[CF.whyTheyCare]: form.why_they_care || undefined,
			[CF.proofOfActivity]: form.proof_of_activity || undefined,
			[CF.likelyPain]: form.likely_pain || undefined,
			[CF.nextAction]: form.next_action || undefined
		};
		// strip undefined so we store a clean JSONB object
		for (const k of Object.keys(custom_fields)) {
			if (custom_fields[k] === undefined) delete custom_fields[k];
		}
		return {
			name: form.name.trim(),
			type: 'company',
			website: form.website.trim() || undefined,
			city: form.city.trim() || undefined,
			state: form.state.trim() || undefined,
			email: form.email.trim() || undefined,
			phone: form.phone.trim() || undefined,
			source: form.source.trim() || undefined,
			notes: form.notes.trim() || undefined,
			custom_fields
		};
	}

	async function save(e: Event) {
		e.preventDefault();
		if (!form.name.trim()) return;
		saving = true;
		error = null;
		try {
			const payload = buildPayload();
			const savedEditId = editingId;
			if (editingId) {
				await api.updateClient(editingId, payload);
			} else {
				await api.createClient(payload);
			}
			showForm = false;
			await load();
			// keep an open detail drawer in sync with the edit
			if (savedEditId && showDetail && detailLead?.id === savedEditId) {
				await openDetail(savedEditId);
			}
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to save';
		} finally {
			saving = false;
		}
	}

	// Move a lead to a new stage (drag-free: select on card).
	async function setStage(l: Lead, stage: PipelineStage) {
		busyId = l.id;
		error = null;
		const next = { ...(l.custom_fields ?? {}), [CF.pipelineStage]: stage };
		try {
			await api.updateClient(l.id, { custom_fields: next });
			leads = leads.map((x) => (x.id === l.id ? { ...x, custom_fields: next } : x));
			if (detailLead?.id === l.id) detailLead = { ...detailLead, custom_fields: next };
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to move lead';
		} finally {
			busyId = null;
		}
	}

	async function remove(l: Lead) {
		if (!confirm(`Delete "${l.name}"?`)) return;
		busyId = l.id;
		try {
			await api.deleteClient(l.id);
			leads = leads.filter((x) => x.id !== l.id);
			if (detailLead?.id === l.id) closeDetail();
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to delete';
		} finally {
			busyId = null;
		}
	}

	// ---- Contacts (inside the detail drawer) ----
	function openContactCreate() {
		contactForm = emptyContact();
		contactEditingId = null;
		showContactForm = true;
	}

	function openContactEdit(ct: ContactResponse) {
		contactForm = {
			name: ct.name ?? '',
			role: ct.role ?? '',
			email: ct.email ?? '',
			phone: ct.phone ?? '',
			is_primary: ct.is_primary ?? false,
			notes: ct.notes ?? ''
		};
		contactEditingId = ct.id;
		showContactForm = true;
	}

	async function saveContact(e: Event) {
		e.preventDefault();
		if (!detailLead || !contactForm.name.trim()) return;
		contactSaving = true;
		error = null;
		const payload: CreateContactData = {
			name: contactForm.name.trim(),
			role: contactForm.role.trim() || undefined,
			email: contactForm.email.trim() || undefined,
			phone: contactForm.phone.trim() || undefined,
			is_primary: contactForm.is_primary,
			notes: contactForm.notes.trim() || undefined
		};
		try {
			if (contactEditingId) {
				await api.updateContact(detailLead.id, contactEditingId, payload);
			} else {
				await api.createContact(detailLead.id, payload);
			}
			contacts = await api.getClientContacts(detailLead.id);
			showContactForm = false;
			contactEditingId = null;
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to save contact';
		} finally {
			contactSaving = false;
		}
	}

	async function removeContact(ct: ContactResponse) {
		if (!detailLead) return;
		if (!confirm(`Remove contact "${ct.name}"?`)) return;
		contactBusyId = ct.id;
		error = null;
		try {
			await api.deleteContact(detailLead.id, ct.id);
			contacts = contacts.filter((x) => x.id !== ct.id);
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to remove contact';
		} finally {
			contactBusyId = null;
		}
	}

	function togglePain(id: string) {
		form.pain_category = form.pain_category.includes(id)
			? form.pain_category.filter((p) => p !== id)
			: [...form.pain_category, id];
	}

	function initials(l: Lead): string {
		return (l.name || '?').charAt(0).toUpperCase();
	}
</script>

<div class="proj-root">
	<header class="topbar">
		<div class="title-wrap">
			<h1>Relationships</h1>
			<span class="count">{filtered.length}</span>
			<span class="sub">Leads &amp; clients</span>
		</div>
		<div class="tools">
			<div class="search">
				<Search size={15} strokeWidth={2} />
				<input placeholder="Search company, owner, city" bind:value={query} />
			</div>
			<select class="filter-sel" bind:value={agencyFilter} aria-label="Filter by agency type">
				<option value="">All agency types</option>
				{#each AGENCY_TYPES as t}<option value={t.id}>{t.label}</option>{/each}
			</select>
			<div class="seg">
				<button class:active={view === 'board'} onclick={() => (view = 'board')} aria-label="Board view"><LayoutGrid size={16} /></button>
				<button class:active={view === 'list'} onclick={() => (view = 'list')} aria-label="List view"><List size={16} /></button>
			</div>
			<button class="btn btn--primary" onclick={openCreate}><Plus size={16} strokeWidth={2.4} />New lead</button>
		</div>
	</header>

	{#if error}<div class="banner banner--error">{error}</div>{/if}

	{#if loading}
		<div class="loading"><Loader2 class="spin" size={20} /> Loading…</div>
	{:else if leads.length === 0}
		<div class="empty">
			<Building2 size={26} strokeWidth={1.4} />
			<p>No relationships yet. Every lead and client lives here - one record from first touch to retained client.</p>
			<button class="btn btn--primary" onclick={openCreate}><Plus size={16} />Add your first lead or client</button>
		</div>
	{:else if filtered.length === 0}
		<div class="empty">
			<Building2 size={26} strokeWidth={1.4} />
			<p>No leads match your filters.</p>
			<button class="btn btn--ghost" onclick={() => { query = ''; agencyFilter = ''; }}>Clear filters</button>
		</div>
	{:else if view === 'board'}
		<div class="board">
			{#each PIPELINE_STAGES as col}
				<div
					class="column {dragOverCol === col.id ? 'column--dragover' : ''}"
					role="group"
					aria-label="{col.label} column"
					ondragover={(e) => { e.preventDefault(); dragOverCol = col.id; }}
					ondragleave={() => { if (dragOverCol === col.id) dragOverCol = null; }}
					ondrop={() => { if (draggedLead && leadStage(draggedLead) !== col.id) setStage(draggedLead, col.id); draggedLead = null; dragOverCol = null; }}
				>
					<div class="col-head"><span>{col.label}</span><span class="col-count">{byStage(col.id).length}</span></div>
					<div class="col-body">
						{#each byStage(col.id) as l (l.id)}
							<div class="card" role="button" tabindex="0" draggable="true" ondragstart={() => (draggedLead = l)} ondragend={() => { draggedLead = null; dragOverCol = null; }} onclick={() => openDetail(l.id)} onkeydown={(e) => e.key === 'Enter' && openDetail(l.id)}>
								<div class="card-top">
									<div class="avatar">{initials(l)}</div>
									<div class="card-name-wrap">
										<span class="card-name">{l.name}</span>
										{#if cf(l, CF.agencyType)}<span class="card-type">{optionLabel(AGENCY_TYPES, cf(l, CF.agencyType))}</span>{/if}
									</div>
									<button class="card-x" title="Delete" onclick={(e) => { e.stopPropagation(); remove(l); }} disabled={busyId === l.id}><Trash2 size={13} /></button>
								</div>
								<div class="card-meta">
									{#if l.city || l.state}<span class="chip"><MapPin size={11} />{[l.city, l.state].filter(Boolean).join(', ')}</span>{/if}
									{#if fitScore(l) > 0}<span class="chip chip--fit"><Star size={11} />{fitScore(l)}/5</span>{/if}
									{#if cf(l, CF.nextStepDate)}<span class="chip"><CalendarClock size={11} />{cf(l, CF.nextStepDate)}</span>{/if}
								</div>
								{#if cf(l, CF.outreachStatus)}<span class="outreach">{optionLabel(OUTREACH_STATUS, cf(l, CF.outreachStatus))}</span>{/if}
								<select class="stage-sel" value={leadStage(l)} disabled={busyId === l.id} onclick={(e) => e.stopPropagation()} onchange={(e) => setStage(l, (e.target as HTMLSelectElement).value as PipelineStage)} aria-label="Pipeline stage">
									{#each PIPELINE_STAGES as s}<option value={s.id}>{s.label}</option>{/each}
								</select>
							</div>
						{/each}
					</div>
				</div>
			{/each}
		</div>
	{:else}
		<div class="list">
			<div class="list-head">
				<span>Company</span><span>Agency type</span><span>Location</span><span>Outreach</span><span>Fit</span><span>Stage</span><span></span>
			</div>
			{#each filtered as l (l.id)}
				<div class="lrow" role="button" tabindex="0" onclick={() => openDetail(l.id)} onkeydown={(e) => e.key === 'Enter' && openDetail(l.id)}>
					<span class="lr-name"><div class="avatar avatar--sm">{initials(l)}</div>{l.name}</span>
					<span class="lr-col">{optionLabel(AGENCY_TYPES, cf(l, CF.agencyType)) || '—'}</span>
					<span class="lr-col">{[l.city, l.state].filter(Boolean).join(', ') || '—'}</span>
					<span class="lr-col">{optionLabel(OUTREACH_STATUS, cf(l, CF.outreachStatus)) || '—'}</span>
					<span class="lr-col">{fitScore(l) > 0 ? `${fitScore(l)}/5` : '—'}</span>
					<span class="lr-col" onclick={(e) => e.stopPropagation()} role="presentation">
						<select class="stage-sel" value={leadStage(l)} disabled={busyId === l.id} onchange={(e) => setStage(l, (e.target as HTMLSelectElement).value as PipelineStage)} aria-label="Pipeline stage">
							{#each PIPELINE_STAGES as s}<option value={s.id}>{s.label}</option>{/each}
						</select>
					</span>
					<span class="lr-x"><button class="card-x" title="Delete" onclick={(e) => { e.stopPropagation(); remove(l); }} disabled={busyId === l.id}><Trash2 size={14} /></button></span>
				</div>
			{/each}
		</div>
	{/if}
</div>

{#if showForm}
	<div class="overlay" role="button" tabindex="0" onclick={() => (showForm = false)} onkeydown={(e) => e.key === 'Escape' && (showForm = false)}>
		<div class="modal" role="dialog" tabindex="-1" onclick={(e) => e.stopPropagation()} onkeydown={() => {}}>
			<div class="modal-head">
				<h2>{editingId ? 'Edit lead' : 'New agency lead'}</h2>
				<button class="card-x" onclick={() => (showForm = false)} aria-label="Close"><X size={18} /></button>
			</div>
			<form onsubmit={save}>
				<!-- Identity -->
				<div class="sec-label">Company</div>
				<label class="field"><span>Company name *</span><input bind:value={form.name} placeholder="Acme Marketing Co." required /></label>
				<div class="field-row">
					<label class="field"><span>Agency type</span>
						<select bind:value={form.agency_type}><option value="">—</option>{#each AGENCY_TYPES as t}<option value={t.id}>{t.label}</option>{/each}</select>
					</label>
					<label class="field"><span>Website</span><input bind:value={form.website} placeholder="https://" /></label>
				</div>
				<div class="field-row">
					<label class="field"><span>City</span><input bind:value={form.city} /></label>
					<label class="field"><span>State</span><input bind:value={form.state} /></label>
				</div>
				<div class="field-row">
					<label class="field"><span>Physical office status</span>
						<select bind:value={form.physical_office_status}><option value="">—</option>{#each PHYSICAL_OFFICE_STATUS as o}<option value={o.id}>{o.label}</option>{/each}</select>
					</label>
					<label class="field"><span>Source</span><input bind:value={form.source} placeholder="Field visit, referral…" /></label>
				</div>
				<div class="field-row">
					<label class="field"><span>Google Maps URL</span><input bind:value={form.google_maps_url} placeholder="https://maps…" /></label>
					<label class="field"><span>LinkedIn URL</span><input bind:value={form.linkedin_url} placeholder="https://linkedin.com/…" /></label>
				</div>

				<!-- People & contact -->
				<div class="sec-label">People &amp; contact</div>
				<div class="field-row">
					<label class="field"><span>Owner name</span><input bind:value={form.owner_name} /></label>
					<label class="field"><span>Operator name</span><input bind:value={form.operator_name} /></label>
				</div>
				<div class="field-row">
					<label class="field"><span>Email</span><input type="email" bind:value={form.email} /></label>
					<label class="field"><span>Phone</span><input bind:value={form.phone} /></label>
				</div>
				<div class="field-row">
					<label class="field"><span>Lead owner</span><input bind:value={form.lead_owner} placeholder="Who owns this lead" /></label>
					<label class="field"><span>Next step date</span><input type="date" bind:value={form.next_step_date} /></label>
				</div>

				<!-- Qualification -->
				<div class="sec-label">Qualification</div>
				<div class="field-row">
					<label class="field"><span>Pipeline stage</span>
						<select bind:value={form.pipeline_stage}>{#each PIPELINE_STAGES as s}<option value={s.id}>{s.label}</option>{/each}</select>
					</label>
					<label class="field"><span>Outreach status</span>
						<select bind:value={form.outreach_status}><option value="">—</option>{#each OUTREACH_STATUS as o}<option value={o.id}>{o.label}</option>{/each}</select>
					</label>
				</div>
				<div class="field-row">
					<label class="field"><span>Meeting preference</span>
						<select bind:value={form.meeting_preference}><option value="">—</option>{#each MEETING_PREFERENCE as o}<option value={o.id}>{o.label}</option>{/each}</select>
					</label>
					<label class="field"><span>Fit score</span>
						<select bind:value={form.fit_score_n}><option value="">—</option>{#each FIT_SCORES as f}<option value={f.id}>{f.label}</option>{/each}</select>
					</label>
				</div>
				<label class="field"><span>Offer fit</span>
					<select bind:value={form.offer_fit}><option value="">—</option>{#each OFFER_FIT as o}<option value={o.id}>{o.label}</option>{/each}</select>
				</label>
				<div class="field">
					<span>Pain category</span>
					<div class="chips-pick">
						{#each PAIN_CATEGORY as p}
							<button type="button" class="pick" class:on={form.pain_category.includes(p.id)} onclick={() => togglePain(p.id)}>{p.label}</button>
						{/each}
					</div>
				</div>

				<!-- Tool stack -->
				<div class="sec-label">Tool stack</div>
				<div class="tool-grid">
					{#each TOOL_STACK_FIELDS as t}
						<label class="field"><span>{t.label}</span><input value={form.tool_stack[t.key] ?? ''} oninput={(e) => (form.tool_stack[t.key] = (e.target as HTMLInputElement).value)} /></label>
					{/each}
				</div>

				<!-- Required notes -->
				<div class="sec-label">Required notes</div>
				<label class="field"><span>Who they serve</span><input bind:value={form.who_they_serve} /></label>
				<label class="field"><span>Why they might care</span><input bind:value={form.why_they_care} /></label>
				<label class="field"><span>Visible proof of activity</span><input bind:value={form.proof_of_activity} /></label>
				<label class="field"><span>Likely pain point</span><input bind:value={form.likely_pain} /></label>
				<label class="field"><span>Next action</span><input bind:value={form.next_action} /></label>
				<label class="field"><span>Notes</span><textarea bind:value={form.notes} rows="3" placeholder="Anything else"></textarea></label>

				<div class="modal-actions">
					<button type="button" class="btn btn--ghost" onclick={() => (showForm = false)}>Cancel</button>
					<button type="submit" class="btn btn--primary" disabled={saving || !form.name.trim()}>
						{#if saving}<Loader2 class="spin" size={15} />{/if}{editingId ? 'Save' : 'Add lead'}
					</button>
				</div>
			</form>
		</div>
	</div>
{/if}

{#if showDetail}
	<div class="overlay overlay--right" role="button" tabindex="0" onclick={closeDetail} onkeydown={(e) => e.key === 'Escape' && closeDetail()}>
		<div class="drawer" role="dialog" tabindex="-1" onclick={(e) => e.stopPropagation()} onkeydown={() => {}}>
			{#if detailLoading || !detailLead}
				<div class="drawer-loading"><Loader2 class="spin" size={20} /> Loading…</div>
			{:else}
				{@const l = detailLead as unknown as Lead}
				<div class="drawer-head">
					<div class="dh-id">
						<div class="avatar avatar--lg">{initials(l)}</div>
						<div class="dh-name-wrap">
							<h2>{l.name}</h2>
							<div class="dh-sub">
								{#if cf(l, CF.agencyType)}<span>{optionLabel(AGENCY_TYPES, cf(l, CF.agencyType))}</span>{/if}
								{#if l.city || l.state}<span class="dot">·</span><span>{[l.city, l.state].filter(Boolean).join(', ')}</span>{/if}
							</div>
						</div>
					</div>
					<button class="card-x" onclick={closeDetail} aria-label="Close"><X size={18} /></button>
				</div>

				<div class="drawer-actions">
					<select class="stage-sel stage-sel--wide" value={leadStage(l)} disabled={busyId === l.id} onchange={(e) => setStage(l, (e.target as HTMLSelectElement).value as PipelineStage)} aria-label="Pipeline stage">
						{#each PIPELINE_STAGES as s}<option value={s.id}>{s.label}</option>{/each}
					</select>
					<a class="btn btn--ghost" style="text-decoration:none" href={`/relationships/${l.id}`}><LayoutGrid size={14} />Open client board</a>
					<button class="btn btn--ghost" onclick={editFromDetail}><Pencil size={14} />Edit</button>
					<button class="btn btn--danger" onclick={() => remove(l)} disabled={busyId === l.id}><Trash2 size={14} />Delete</button>
				</div>

				<div class="drawer-body">
					<div class="meta-chips">
						{#if fitScore(l) > 0}<span class="chip chip--fit"><Star size={11} />Fit {fitScore(l)}/5</span>{/if}
						{#if cf(l, CF.outreachStatus)}<span class="chip">{optionLabel(OUTREACH_STATUS, cf(l, CF.outreachStatus))}</span>{/if}
						{#if cf(l, CF.nextStepDate)}<span class="chip"><CalendarClock size={11} />{cf(l, CF.nextStepDate)}</span>{/if}
					</div>

					{#if l.website || cf(l, CF.googleMapsUrl) || cf(l, CF.linkedinUrl)}
						<div class="link-row">
							{#if l.website}<a class="lnk" href={l.website} target="_blank" rel="noopener noreferrer"><Globe size={13} />Website</a>{/if}
							{#if cf(l, CF.googleMapsUrl)}<a class="lnk" href={cf(l, CF.googleMapsUrl)} target="_blank" rel="noopener noreferrer"><MapPin size={13} />Maps</a>{/if}
							{#if cf(l, CF.linkedinUrl)}<a class="lnk" href={cf(l, CF.linkedinUrl)} target="_blank" rel="noopener noreferrer"><Link2 size={13} />LinkedIn</a>{/if}
						</div>
					{/if}

					<!-- Primary contact info on the company record -->
					<div class="d-sec">Contact</div>
					<dl class="dl">
						{#if l.email}<div><dt>Email</dt><dd><a class="lnk" href={`mailto:${l.email}`}>{l.email}</a></dd></div>{/if}
						{#if l.phone}<div><dt>Phone</dt><dd><a class="lnk" href={`tel:${l.phone}`}>{l.phone}</a></dd></div>{/if}
						{#if cf(l, CF.ownerName)}<div><dt>Owner</dt><dd>{cf(l, CF.ownerName)}</dd></div>{/if}
						{#if cf(l, CF.operatorName)}<div><dt>Operator</dt><dd>{cf(l, CF.operatorName)}</dd></div>{/if}
						{#if cf(l, CF.leadOwner)}<div><dt>Lead owner</dt><dd>{cf(l, CF.leadOwner)}</dd></div>{/if}
						{#if l.source}<div><dt>Source</dt><dd>{l.source}</dd></div>{/if}
						{#if cf(l, CF.physicalOfficeStatus)}<div><dt>Office</dt><dd>{optionLabel(PHYSICAL_OFFICE_STATUS, cf(l, CF.physicalOfficeStatus))}</dd></div>{/if}
						{#if cf(l, CF.meetingPreference)}<div><dt>Meeting</dt><dd>{optionLabel(MEETING_PREFERENCE, cf(l, CF.meetingPreference))}</dd></div>{/if}
						{#if cf(l, CF.offerFit)}<div><dt>Offer fit</dt><dd>{optionLabel(OFFER_FIT, cf(l, CF.offerFit))}</dd></div>{/if}
					</dl>

					{#if cfArr(l, CF.painCategory).length}
						<div class="d-sec">Pain categories</div>
						<div class="tag-wrap">
							{#each cfArr(l, CF.painCategory) as p}<span class="tag">{optionLabel(PAIN_CATEGORY, p)}</span>{/each}
						</div>
					{/if}

					{#if TOOL_STACK_FIELDS.some((t) => cf(l, CF.toolStack) && (l.custom_fields?.[CF.toolStack] as Record<string, string>)?.[t.key])}
						<div class="d-sec">Tool stack</div>
						<dl class="dl">
							{#each TOOL_STACK_FIELDS as t}
								{@const v = (l.custom_fields?.[CF.toolStack] as Record<string, string>)?.[t.key]}
								{#if v}<div><dt>{t.label}</dt><dd>{v}</dd></div>{/if}
							{/each}
						</dl>
					{/if}

					{#if cf(l, CF.whoTheyServe) || cf(l, CF.whyTheyCare) || cf(l, CF.proofOfActivity) || cf(l, CF.likelyPain) || cf(l, CF.nextAction) || l.notes}
						<div class="d-sec">Notes</div>
						<dl class="dl dl--stack">
							{#if cf(l, CF.whoTheyServe)}<div><dt>Who they serve</dt><dd>{cf(l, CF.whoTheyServe)}</dd></div>{/if}
							{#if cf(l, CF.whyTheyCare)}<div><dt>Why they care</dt><dd>{cf(l, CF.whyTheyCare)}</dd></div>{/if}
							{#if cf(l, CF.proofOfActivity)}<div><dt>Proof of activity</dt><dd>{cf(l, CF.proofOfActivity)}</dd></div>{/if}
							{#if cf(l, CF.likelyPain)}<div><dt>Likely pain</dt><dd>{cf(l, CF.likelyPain)}</dd></div>{/if}
							{#if cf(l, CF.nextAction)}<div><dt>Next action</dt><dd>{cf(l, CF.nextAction)}</dd></div>{/if}
							{#if l.notes}<div><dt>Notes</dt><dd>{l.notes}</dd></div>{/if}
						</dl>
					{/if}

					<!-- Contacts (people at this company) -->
					<div class="d-sec d-sec--row">
						<span><Users size={13} /> Contacts <span class="c-count">{contacts.length}</span></span>
						{#if !showContactForm}<button class="btn btn--ghost btn--sm" onclick={openContactCreate}><UserPlus size={13} />Add</button>{/if}
					</div>

					{#if showContactForm}
						<form class="contact-form" onsubmit={saveContact}>
							<div class="field-row">
								<label class="field"><span>Name *</span><input bind:value={contactForm.name} placeholder="Jane Doe" required /></label>
								<label class="field"><span>Role</span><input bind:value={contactForm.role} placeholder="Owner, Ops…" /></label>
							</div>
							<div class="field-row">
								<label class="field"><span>Email</span><input type="email" bind:value={contactForm.email} /></label>
								<label class="field"><span>Phone</span><input bind:value={contactForm.phone} /></label>
							</div>
							<label class="field"><span>Notes</span><textarea bind:value={contactForm.notes} rows="2"></textarea></label>
							<label class="check"><input type="checkbox" bind:checked={contactForm.is_primary} /><span>Primary contact</span></label>
							<div class="cf-actions">
								<button type="button" class="btn btn--ghost btn--sm" onclick={() => { showContactForm = false; contactEditingId = null; }}>Cancel</button>
								<button type="submit" class="btn btn--primary btn--sm" disabled={contactSaving || !contactForm.name.trim()}>
									{#if contactSaving}<Loader2 class="spin" size={13} />{/if}{contactEditingId ? 'Save contact' : 'Add contact'}
								</button>
							</div>
						</form>
					{/if}

					{#if contacts.length === 0 && !showContactForm}
						<p class="c-empty">No contacts yet.</p>
					{:else}
						<div class="c-list">
							{#each contacts as ct (ct.id)}
								<div class="c-row">
									<div class="c-main">
										<div class="c-name">
											{ct.name}
											{#if ct.is_primary}<span class="c-primary" title="Primary contact"><BadgeCheck size={12} />Primary</span>{/if}
										</div>
										<div class="c-meta">
											{#if ct.role}<span>{ct.role}</span>{/if}
											{#if ct.email}<a class="lnk" href={`mailto:${ct.email}`}><Mail size={11} />{ct.email}</a>{/if}
											{#if ct.phone}<a class="lnk" href={`tel:${ct.phone}`}><Phone size={11} />{ct.phone}</a>{/if}
										</div>
										{#if ct.notes}<div class="c-notes">{ct.notes}</div>{/if}
									</div>
									<div class="c-acts">
										<button class="card-x" title="Edit contact" onclick={() => openContactEdit(ct)}><Pencil size={13} /></button>
										<button class="card-x" title="Remove contact" onclick={() => removeContact(ct)} disabled={contactBusyId === ct.id}><Trash2 size={13} /></button>
									</div>
								</div>
							{/each}
						</div>
					{/if}
				</div>
			{/if}
		</div>
	</div>
{/if}

<style>
	.proj-root { height: 100%; display: flex; flex-direction: column; background: var(--dbg); color: var(--dt); overflow: hidden; }
	.topbar { display: flex; align-items: center; justify-content: space-between; gap: 16px; padding: 18px 24px; border-bottom: 1px solid var(--dbd); flex-shrink: 0; }
	.title-wrap { display: flex; align-items: baseline; gap: 10px; }
	.title-wrap h1 { font-size: 1.15rem; font-weight: 680; letter-spacing: -0.02em; margin: 0; }
	.sub { font-size: 0.74rem; color: var(--dt3); }
	.count { font-size: 0.74rem; color: var(--dt3); background: color-mix(in srgb, var(--dt) 8%, transparent); padding: 2px 9px; border-radius: 999px; }
	.tools { display: flex; align-items: center; gap: 10px; }
	.search { display: flex; align-items: center; gap: 7px; padding: 7px 11px; border: 1px solid var(--dbd); border-radius: 9px; color: var(--dt3); background: color-mix(in srgb, var(--dt) 3%, transparent); }
	.search input { border: 0; outline: 0; background: transparent; color: var(--dt); font-size: 0.82rem; width: 170px; }
	.filter-sel { font-size: 0.8rem; padding: 8px 11px; border-radius: 9px; border: 1px solid var(--dbd); background: var(--dbg2, var(--dbg)); color: var(--dt2); cursor: pointer; }
	.seg { display: flex; border: 1px solid var(--dbd); border-radius: 9px; overflow: hidden; }
	.seg button { display: flex; align-items: center; justify-content: center; width: 34px; height: 32px; background: transparent; border: none; color: var(--dt3); cursor: pointer; }
	.seg button.active { background: color-mix(in srgb, var(--dt) 10%, transparent); color: var(--dt); }
	.btn { display: inline-flex; align-items: center; gap: 7px; padding: 8px 15px; border-radius: 9px; font-size: 0.83rem; font-weight: 580; cursor: pointer; border: 1px solid transparent; white-space: nowrap; }
	.btn--primary { background: var(--dt); color: var(--dbg); }
	.btn--ghost { background: transparent; border-color: var(--dbd); color: var(--dt2); }
	.btn:disabled { opacity: 0.55; cursor: not-allowed; }
	.board { flex: 1; display: flex; gap: 14px; padding: 18px 24px; overflow-x: auto; }
	.column { flex: 0 0 250px; min-width: 250px; display: flex; flex-direction: column; border-radius: 10px; transition: background 0.12s, box-shadow 0.12s; }
	.column--dragover { background: color-mix(in srgb, var(--accent, #16a34a) 10%, transparent); box-shadow: inset 0 0 0 2px var(--accent, #16a34a); }
	.card[draggable='true'] { cursor: grab; }
	.card[draggable='true']:active { cursor: grabbing; }
	.col-head { display: flex; align-items: center; gap: 8px; padding: 4px 4px 12px; font-size: 0.72rem; font-weight: 620; color: var(--dt2); text-transform: uppercase; letter-spacing: 0.04em; }
	.col-count { color: var(--dt3); font-weight: 500; }
	.col-body { display: flex; flex-direction: column; gap: 9px; overflow-y: auto; }
	.card { border: 1px solid var(--dbd); border-radius: 11px; padding: 12px; background: color-mix(in srgb, var(--dt) 2%, transparent); display: flex; flex-direction: column; gap: 8px; cursor: pointer; text-align: left; }
	.card:hover { border-color: color-mix(in srgb, var(--dt) 22%, transparent); }
	.card-top { display: flex; align-items: flex-start; gap: 9px; }
	.card-name-wrap { display: flex; flex-direction: column; gap: 2px; flex: 1; min-width: 0; }
	.avatar { width: 30px; height: 30px; border-radius: 50%; background: linear-gradient(135deg,#6366f1,#8b5cf6); color: #fff; font-size: 0.78rem; font-weight: 650; display: flex; align-items: center; justify-content: center; flex-shrink: 0; }
	.avatar--sm { width: 26px; height: 26px; font-size: 0.72rem; }
	.card-name { font-size: 0.88rem; font-weight: 580; }
	.card-type { font-size: 0.7rem; color: var(--dt3); }
	.card-meta { display: flex; flex-wrap: wrap; gap: 6px; }
	.chip { display: inline-flex; align-items: center; gap: 4px; font-size: 0.7rem; color: var(--dt3); background: color-mix(in srgb, var(--dt) 6%, transparent); padding: 3px 8px; border-radius: 6px; }
	.chip--fit { color: #f59e0b; background: color-mix(in srgb, #f59e0b 12%, transparent); }
	.outreach { font-size: 0.7rem; color: var(--dt2); }
	.stage-sel { font-size: 0.74rem; padding: 5px 8px; border-radius: 7px; border: 1px solid var(--dbd); background: var(--dbg); color: var(--dt2); cursor: pointer; width: 100%; }
	.card-x { display: inline-flex; align-items: center; justify-content: center; width: 26px; height: 26px; border-radius: 7px; border: none; background: transparent; color: var(--dt3); cursor: pointer; flex-shrink: 0; }
	.card-x:hover { background: color-mix(in srgb, #ef4444 12%, transparent); color: #ef4444; }
	.list { flex: 1; overflow-y: auto; padding: 8px 24px 24px; }
	.list-head, .lrow { display: grid; grid-template-columns: 2.2fr 1.4fr 1.3fr 1.2fr 0.6fr 1.4fr 40px; align-items: center; gap: 12px; }
	.list-head { padding: 10px 12px; font-size: 0.7rem; text-transform: uppercase; letter-spacing: 0.06em; color: var(--dt3); border-bottom: 1px solid var(--dbd); }
	.lrow { padding: 11px 12px; border-bottom: 1px solid var(--dbd); font-size: 0.85rem; cursor: pointer; }
	.lrow:hover { background: color-mix(in srgb, var(--dt) 3%, transparent); }
	.lr-name { display: flex; align-items: center; gap: 9px; font-weight: 560; }
	.lr-col { color: var(--dt2); }
	.lr-col .stage-sel { width: auto; }
	.lr-x { display: flex; justify-content: flex-end; }
	.loading, .empty { flex: 1; display: flex; flex-direction: column; align-items: center; justify-content: center; gap: 14px; color: var(--dt3); }
	.loading { flex-direction: row; gap: 8px; }
	.banner { margin: 16px 24px 0; padding: 11px 14px; border-radius: 10px; font-size: 0.83rem; }
	.banner--error { background: color-mix(in srgb, #ef4444 12%, transparent); color: #ef4444; }
	.overlay { position: fixed; inset: 0; background: rgba(0,0,0,0.55); display: flex; align-items: center; justify-content: center; z-index: 100; padding: 20px; }
	.modal { width: 100%; max-width: 640px; max-height: 88vh; overflow-y: auto; background: var(--dbg); border: 1px solid var(--dbd); border-radius: 16px; padding: 22px; box-shadow: 0 24px 60px rgba(0,0,0,0.5); }
	.modal-head { display: flex; align-items: center; justify-content: space-between; margin-bottom: 16px; }
	.modal-head h2 { font-size: 1.05rem; font-weight: 640; margin: 0; }
	.sec-label { font-size: 0.7rem; text-transform: uppercase; letter-spacing: 0.06em; color: var(--dt3); font-weight: 640; margin: 16px 0 10px; padding-bottom: 6px; border-bottom: 1px solid var(--dbd); }
	.sec-label:first-of-type { margin-top: 0; }
	.field { display: flex; flex-direction: column; gap: 6px; margin-bottom: 12px; }
	.field span { font-size: 0.78rem; font-weight: 560; color: var(--dt2); }
	.field input, .field select, .field textarea { padding: 9px 12px; border-radius: 9px; border: 1px solid var(--dbd); background: var(--dbg); color: var(--dt); font-size: 0.86rem; outline: none; font-family: inherit; }
	.field textarea { resize: vertical; }
	.field input:focus, .field select:focus, .field textarea:focus { border-color: color-mix(in srgb, var(--dt) 40%, transparent); }
	.field-row { display: flex; gap: 12px; }
	.field-row .field { flex: 1; min-width: 0; }
	.tool-grid { display: grid; grid-template-columns: 1fr 1fr 1fr; gap: 0 12px; }
	.chips-pick { display: flex; flex-wrap: wrap; gap: 6px; }
	.pick { font-size: 0.74rem; padding: 5px 10px; border-radius: 999px; border: 1px solid var(--dbd); background: transparent; color: var(--dt2); cursor: pointer; }
	.pick.on { background: var(--dt); color: var(--dbg); border-color: var(--dt); }
	.modal-actions { display: flex; justify-content: flex-end; gap: 10px; margin-top: 10px; }
	.btn--danger { background: transparent; border-color: color-mix(in srgb, #ef4444 40%, transparent); color: #ef4444; }
	.btn--danger:hover { background: color-mix(in srgb, #ef4444 12%, transparent); }
	.btn--sm { padding: 6px 11px; font-size: 0.78rem; }

	/* ---- detail drawer ---- */
	.overlay--right { justify-content: flex-end; align-items: stretch; padding: 0; }
	.drawer { width: 100%; max-width: 460px; height: 100%; overflow-y: auto; background: var(--dbg); border-left: 1px solid var(--dbd); box-shadow: -24px 0 60px rgba(0,0,0,0.4); display: flex; flex-direction: column; }
	.drawer-loading { flex: 1; display: flex; align-items: center; justify-content: center; gap: 8px; color: var(--dt3); }
	.drawer-head { display: flex; align-items: flex-start; justify-content: space-between; gap: 12px; padding: 20px 22px 14px; border-bottom: 1px solid var(--dbd); }
	.dh-id { display: flex; align-items: center; gap: 12px; min-width: 0; }
	.avatar--lg { width: 42px; height: 42px; font-size: 1rem; }
	.dh-name-wrap { min-width: 0; }
	.dh-name-wrap h2 { font-size: 1.1rem; font-weight: 650; margin: 0; letter-spacing: -0.01em; }
	.dh-sub { display: flex; flex-wrap: wrap; align-items: center; gap: 6px; font-size: 0.78rem; color: var(--dt3); margin-top: 3px; }
	.dh-sub .dot { opacity: 0.5; }
	.drawer-actions { display: flex; align-items: center; gap: 8px; padding: 14px 22px; border-bottom: 1px solid var(--dbd); }
	.stage-sel--wide { flex: 1; }
	.drawer-body { padding: 18px 22px 40px; display: flex; flex-direction: column; gap: 6px; }
	.meta-chips { display: flex; flex-wrap: wrap; gap: 6px; margin-bottom: 6px; }
	.link-row { display: flex; flex-wrap: wrap; gap: 8px; margin-bottom: 8px; }
	.lnk { display: inline-flex; align-items: center; gap: 5px; font-size: 0.8rem; color: #818cf8; text-decoration: none; }
	.lnk:hover { text-decoration: underline; }
	.d-sec { font-size: 0.7rem; text-transform: uppercase; letter-spacing: 0.06em; color: var(--dt3); font-weight: 640; margin: 18px 0 8px; padding-bottom: 6px; border-bottom: 1px solid var(--dbd); }
	.d-sec--row { display: flex; align-items: center; justify-content: space-between; }
	.d-sec--row span { display: inline-flex; align-items: center; gap: 6px; }
	.c-count { color: var(--dt3); font-weight: 500; text-transform: none; letter-spacing: 0; }
	.dl { display: flex; flex-direction: column; gap: 8px; margin: 0; }
	.dl > div { display: grid; grid-template-columns: 120px 1fr; gap: 10px; align-items: baseline; }
	.dl--stack > div { grid-template-columns: 1fr; gap: 3px; }
	.dl dt { font-size: 0.76rem; color: var(--dt3); }
	.dl dd { font-size: 0.84rem; color: var(--dt); margin: 0; overflow-wrap: anywhere; }
	.tag-wrap { display: flex; flex-wrap: wrap; gap: 6px; }
	.tag { font-size: 0.74rem; padding: 4px 9px; border-radius: 999px; border: 1px solid var(--dbd); color: var(--dt2); background: color-mix(in srgb, var(--dt) 4%, transparent); }
	.contact-form { border: 1px solid var(--dbd); border-radius: 12px; padding: 14px; margin-bottom: 12px; background: color-mix(in srgb, var(--dt) 2%, transparent); }
	.check { display: flex; align-items: center; gap: 8px; font-size: 0.82rem; color: var(--dt2); margin-bottom: 12px; cursor: pointer; }
	.check input { width: 15px; height: 15px; accent-color: var(--dt); }
	.cf-actions { display: flex; justify-content: flex-end; gap: 8px; }
	.c-empty { font-size: 0.82rem; color: var(--dt3); padding: 6px 0; }
	.c-list { display: flex; flex-direction: column; gap: 8px; }
	.c-row { display: flex; align-items: flex-start; justify-content: space-between; gap: 10px; border: 1px solid var(--dbd); border-radius: 11px; padding: 11px 12px; }
	.c-main { min-width: 0; display: flex; flex-direction: column; gap: 4px; }
	.c-name { display: flex; align-items: center; gap: 8px; font-size: 0.86rem; font-weight: 570; }
	.c-primary { display: inline-flex; align-items: center; gap: 3px; font-size: 0.68rem; font-weight: 560; color: #34d399; background: color-mix(in srgb, #34d399 12%, transparent); padding: 2px 7px; border-radius: 999px; }
	.c-meta { display: flex; flex-wrap: wrap; gap: 10px; font-size: 0.78rem; color: var(--dt3); }
	.c-notes { font-size: 0.78rem; color: var(--dt2); }
	.c-acts { display: flex; gap: 2px; flex-shrink: 0; }
	:global(.spin) { animation: spin 0.9s linear infinite; }
	@keyframes spin { to { transform: rotate(360deg); } }

	@media (max-width: 768px) {
		.topbar { flex-wrap: wrap; gap: 10px; padding: 14px 16px; }
		.tools { flex-wrap: wrap; gap: 8px; width: 100%; }
		.search { flex: 1 1 auto; min-width: 0; }
		.search input { width: 100%; min-width: 0; }
		.board { padding: 12px 16px; gap: 10px; }
		.list { padding: 6px 16px 20px; }
		.banner { margin: 12px 16px 0; }
		.tool-grid { grid-template-columns: 1fr 1fr; }
	}
	@media (max-width: 480px) {
		.topbar { padding: 12px 14px; }
		.tools { flex-direction: column; align-items: stretch; }
		.filter-sel, .search { width: 100%; }
		.seg { align-self: flex-start; }
		.btn.btn--primary { width: 100%; justify-content: center; min-height: 44px; }
		.board { flex-direction: column; overflow-x: visible; padding: 12px 14px; gap: 14px; }
		.column { min-width: 0; width: 100%; flex: none; }
		.list-head { display: none; }
		.lrow { display: flex; flex-wrap: wrap; align-items: center; gap: 8px; padding: 12px 10px; }
		.lr-name { flex: 1 1 100%; font-size: 0.9rem; }
		.lr-col { flex: 0 1 auto; font-size: 0.78rem; }
		.lr-x { margin-left: auto; }
		.overlay { align-items: flex-end; padding: 0; }
		.modal { max-width: 100%; max-height: 92vh; border-radius: 20px 20px 0 0; padding: 20px 16px 28px; }
		.field-row { flex-direction: column; gap: 0; }
		.tool-grid { grid-template-columns: 1fr; }
		.overlay--right { align-items: stretch; }
		.drawer { max-width: 100%; }
		.drawer-head, .drawer-actions { padding-left: 16px; padding-right: 16px; }
		.drawer-body { padding: 16px 16px 40px; }
		.dl > div { grid-template-columns: 100px 1fr; }
	}
</style>
