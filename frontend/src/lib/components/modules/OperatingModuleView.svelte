<script lang="ts">
	import type { CustomModule } from '$lib/types/modules';
	import { getTable, getTables, getRows, updateRow } from '$lib/api/tables';
	import { invalidateApiCache } from '$lib/api/base';
	import type { Row, Table, TableListItem } from '$lib/api/tables';
	import {
		Activity, AlertTriangle, CheckCircle2, ChevronRight, Database,
		FileCheck2, Loader2, Plug, RefreshCw, ShieldCheck
	} from 'lucide-svelte';

	interface ModuleManifest {
		module_code?: string;
		phase?: string;
		primary_primitive?: string;
		primitives?: string[];
		connected_systems?: string[];
		acceptance_criteria?: string[];
		mock_data_only?: boolean;
		expected_primitive_count?: number;
		expected_record_count?: number;
	}

	interface PrimitiveState {
		table: Table;
		rows: Row[];
	}

	let { mod }: { mod: CustomModule } = $props();

	const manifest = $derived((mod.manifest ?? {}) as ModuleManifest);
	const primitiveNames = $derived(Array.isArray(manifest.primitives) ? manifest.primitives : []);
	const acceptanceCriteria = $derived(
		Array.isArray(manifest.acceptance_criteria) ? manifest.acceptance_criteria : []
	);
	const connectedSystems = $derived(
		Array.isArray(manifest.connected_systems) ? manifest.connected_systems : []
	);

	let loading = $state(true);
	let error = $state<string | null>(null);
	let states = $state<PrimitiveState[]>([]);
	let tableSummaries = $state<TableListItem[]>([]);
	let activePrimitive = $state('Overview');
	let primitiveLoading = $state<string | null>(null);
	let decisionBusy = $state<string | null>(null);
	let loadGeneration = 0;
	const primitiveRequests = new Map<string, Promise<void>>();

	const loadedRecordCount = $derived(states.reduce((total, state) => total + state.rows.length, 0));
	const primitiveCount = $derived(states.length || manifest.expected_primitive_count || 0);
	const totalRecords = $derived(loadedRecordCount || manifest.expected_record_count || 0);
	const approvalState = $derived(states.find((state) => state.table.name === 'Approvals'));
	const alertState = $derived(states.find((state) => state.table.name === 'Alerts'));
	const evidenceState = $derived(states.find((state) => state.table.name === 'Evidence and Citations'));
	const primaryState = $derived(states.find((state) => state.table.name === manifest.primary_primitive));
	const overviewState = $derived(
		primaryState?.rows.length ? primaryState : states.find((state) => state.rows.length > 0)
	);
	const pendingApprovals = $derived(
		approvalState?.rows.filter((row) => String(fieldValue(approvalState, row, 'status')).toLowerCase() === 'pending').length ?? 0
	);
	const openAlerts = $derived(
		alertState?.rows.filter((row) => String(fieldValue(alertState, row, 'status')).toLowerCase() !== 'resolved').length ?? 0
	);
	const activeState = $derived(states.find((state) => state.table.name === activePrimitive));

	$effect(() => {
		const moduleId = mod.id;
		void loadData(moduleId);
	});

	async function loadData(expectedModuleId = mod.id) {
		const generation = ++loadGeneration;
		invalidateApiCache();
		loading = true;
		error = null;
		states = [];
		tableSummaries = [];
		primitiveRequests.clear();
		activePrimitive = 'Overview';
		try {
			const workspaceId = mod.workspace_id;
			const tables = await getTables(undefined, workspaceId);
			if (generation !== loadGeneration || mod.id !== expectedModuleId) return;
			tableSummaries = primitiveNames
				.map((name) => tables.find((table) => table.name === name))
				.filter((table): table is TableListItem => Boolean(table));
			const primarySummary = tableSummaries.find(
				(table) => table.name === manifest.primary_primitive
			);
			if (primarySummary) {
				await loadPrimitiveState(primarySummary, workspaceId, generation);
				void enrichOverview(primarySummary.id, workspaceId, generation);
			} else {
				void enrichOverview(undefined, workspaceId, generation);
			}
		} catch (cause) {
			error = cause instanceof Error ? cause.message : 'Failed to load module data';
		} finally {
			loading = false;
		}
	}

	async function enrichOverview(
		primaryTableId: string | undefined,
		workspaceId: string,
		generation: number
	): Promise<void> {
		const overviewPrimitiveNames = new Set(['Approvals', 'Alerts', 'Evidence and Citations']);
		const summaries = tableSummaries.filter(
			(table) => table.id !== primaryTableId && overviewPrimitiveNames.has(table.name)
		);
		for (const summary of summaries) {
			try {
				await loadPrimitiveState(summary, workspaceId, generation);
			} catch {
				// A single unavailable primitive must not block the rest of the module.
			}
		}
	}

	async function loadPrimitiveState(
		tableSummary: TableListItem,
		workspaceId = mod.workspace_id,
		generation = loadGeneration
	): Promise<void> {
		if (states.some((state) => state.table.id === tableSummary.id)) return;
		const existingRequest = primitiveRequests.get(tableSummary.id);
		if (existingRequest) return existingRequest;

		const request = (async () => {
			const table = await getTable(tableSummary.id, workspaceId);
			const rows = (await getRows(table.id, { page: 1, page_size: 100 }, workspaceId)).rows;
			if (generation !== loadGeneration) return;
			if (!states.some((state) => state.table.id === table.id)) {
				states = [...states, { table, rows }];
			}
		})();

		primitiveRequests.set(tableSummary.id, request);
		try {
			await request;
		} finally {
			if (primitiveRequests.get(tableSummary.id) === request) {
				primitiveRequests.delete(tableSummary.id);
			}
		}
	}

	async function selectPrimitive(name: string) {
		activePrimitive = name;
		if (name === 'Overview' || states.some((state) => state.table.name === name)) return;
		const summary = tableSummaries.find((table) => table.name === name);
		if (!summary) return;
		primitiveLoading = name;
		error = null;
		try {
			await loadPrimitiveState(summary);
		} catch (cause) {
			error = cause instanceof Error ? cause.message : `Failed to load ${name}`;
		} finally {
			primitiveLoading = null;
		}
	}

	function displayValue(value: unknown): string {
		if (value === null || value === undefined || value === '') return '-';
		if (typeof value === 'boolean') return value ? 'Yes' : 'No';
		if (typeof value === 'object') return JSON.stringify(value);
		return String(value);
	}

	function fieldValue(state: PrimitiveState, row: Row, fieldName: string): unknown {
		if (fieldName in row.data) return row.data[fieldName];
		const normalizedName = fieldName.toLowerCase();
		const column = state.table.columns.find(
			(candidate) => candidate.name.toLowerCase() === normalizedName
		);
		return column ? row.data[column.id] : undefined;
	}

	function recordTitle(state: PrimitiveState, row: Row): string {
		const titleFields = [
			'question', 'name', 'title', 'summary', 'answer_summary', 'request',
			'assertion', 'asset_name', 'event_type', 'meeting_name', 'deal_name'
		];
		const candidate = titleFields.map((name) => fieldValue(state, row, name)).find((value) => value != null && value !== '');
		if (candidate != null && candidate !== '') return displayValue(candidate);

		for (const column of state.table.columns.filter((item) => !item.is_hidden)) {
			const value = row.data[column.id] ?? row.data[column.name];
			if (value != null && value !== '' && typeof value !== 'object') return displayValue(value);
		}
		return '-';
	}

	async function setDemoApproval(rowId: string, status: 'Approved' | 'Returned') {
		const approval = states.find((state) => state.table.name === 'Approvals');
		if (!approval) return;
		const statusColumn = approval.table.columns.find((column) => column.name === 'status');
		if (!statusColumn) return;
		const row = approval.rows.find((candidate) => candidate.id === rowId);
		if (!row) return;
		decisionBusy = rowId;
		error = null;
		try {
			const data = { ...row.data, [statusColumn.id]: status };
			await updateRow(approval.table.id, rowId, { data });
			states = states.map((state) => state.table.id === approval.table.id
				? { ...state, rows: state.rows.map((candidate) => candidate.id === rowId ? { ...candidate, data } : candidate) }
				: state);
		} catch (cause) {
			error = cause instanceof Error ? cause.message : `Failed to mark approval ${status.toLowerCase()}`;
		} finally {
			decisionBusy = null;
		}
	}
</script>

<div class="operating-module-scroll" data-testid="operating-module-scroll">
	<div class="operating-module">
	<header class="module-header">
		<div>
			<div class="eyebrow">
				<span>{manifest.module_code ?? 'MODULE'}</span>
				<span>{manifest.phase ?? 'Phase 1'}</span>
				{#if manifest.mock_data_only}<span class="demo-label">Mock data</span>{/if}
			</div>
			<h1>{mod.name}</h1>
			<p>{mod.description}</p>
		</div>
		<button class="refresh" onclick={() => loadData()} disabled={loading} aria-label="Refresh module data">
			<RefreshCw size={16} class={loading ? 'spin' : ''} />
			<span>Refresh</span>
		</button>
	</header>

	{#if error}
		<div class="error-banner"><AlertTriangle size={17} />{error}</div>
	{/if}

	<section class="metrics" aria-label="Module status">
		<div class="metric">
			<Database size={18} />
			<div><strong>{primitiveCount}</strong><span>Connected primitives</span></div>
		</div>
		<div class="metric">
			<Activity size={18} />
			<div><strong>{totalRecords}</strong><span>Operating records</span></div>
		</div>
		<div class="metric" class:attention={pendingApprovals > 0}>
			<ShieldCheck size={18} />
			<div><strong>{pendingApprovals}</strong><span>Pending approvals</span></div>
		</div>
		<div class="metric" class:attention={openAlerts > 0}>
			<AlertTriangle size={18} />
			<div><strong>{openAlerts}</strong><span>Open alerts</span></div>
		</div>
	</section>

	{#if connectedSystems.length > 0}
		<section class="systems-strip" aria-label="Connected systems">
			<div class="systems-label"><Plug size={16} /><span>Connected systems</span></div>
			<div class="system-chips">
				{#each connectedSystems as system}<span>{system}</span>{/each}
			</div>
			<button class="systems-map" onclick={() => selectPrimitive('Systems and Integrations')}>Open systems map <ChevronRight size={14} /></button>
		</section>
	{/if}

	<nav class="tabs" aria-label="Module views">
		{#each ['Overview', ...primitiveNames] as tab}
			<button
				type="button"
				class:active={activePrimitive === tab}
				aria-pressed={activePrimitive === tab}
				onclick={() => selectPrimitive(tab)}
			>{tab}</button>
		{/each}
	</nav>

	{#if activePrimitive === 'Overview'}
		<div class="overview-grid">
			<section class="primary-panel">
				<div class="section-heading">
					<div><span>Primary operating view</span><h2>{overviewState?.table.name ?? manifest.primary_primitive ?? 'Module records'}</h2></div>
					<span class="count-badge">{overviewState?.rows.length ?? 0}</span>
				</div>
				{#if overviewState?.rows.length}
					<div class="record-list">
						{#each overviewState.rows.slice(0, 6) as row (row.id)}
							<button onclick={() => activePrimitive = overviewState.table.name}>
								<span>{recordTitle(overviewState, row)}</span><ChevronRight size={15} />
							</button>
						{/each}
					</div>
				{:else}
					<div class="empty-state">No operating records are available for this view.</div>
				{/if}
			</section>

			<aside class="acceptance-panel">
				<div class="section-heading">
					<div><span>RFP contract</span><h2>Acceptance criteria</h2></div>
					<FileCheck2 size={19} />
				</div>
				<ul>
					{#each acceptanceCriteria as criterion}
						<li><CheckCircle2 size={16} /><span>{criterion}</span></li>
					{/each}
				</ul>
			</aside>

			<section class="activity-panel">
				<div class="section-heading"><div><span>Evidence</span><h2>Recent cited assertions</h2></div></div>
				{#if evidenceState?.rows.length}
					<div class="compact-list">
						{#each evidenceState.rows.slice(0, 4) as row (row.id)}
							<div><strong>{displayValue(fieldValue(evidenceState, row, 'assertion'))}</strong><span>{displayValue(fieldValue(evidenceState, row, 'source_location'))}</span></div>
						{/each}
					</div>
				{:else}<div class="empty-state">No evidence records yet.</div>{/if}
			</section>

			<section class="activity-panel">
				<div class="section-heading"><div><span>Exceptions</span><h2>Human gates and alerts</h2></div></div>
				{#if approvalState?.rows.length}
					<div class="approval-list">
						{#each approvalState.rows.slice(0, 3) as row (row.id)}
							{@const status = String(fieldValue(approvalState, row, 'status'))}
							<div class="approval-item">
								<div>
									<strong>{displayValue(fieldValue(approvalState, row, 'request'))}</strong>
									<span>{displayValue(fieldValue(approvalState, row, 'policy_gate'))}</span>
								</div>
								{#if status.toLowerCase() === 'pending'}
									<div class="approval-actions">
										<button class="approve" disabled={decisionBusy === row.id} onclick={() => setDemoApproval(row.id, 'Approved')}>Approve</button>
										<button disabled={decisionBusy === row.id} onclick={() => setDemoApproval(row.id, 'Returned')}>Return</button>
									</div>
								{:else}<span class="status-pill">{status}</span>{/if}
							</div>
						{/each}
					</div>
				{:else}
					<div class="status-split">
						<div><strong>{pendingApprovals}</strong><span>Approvals waiting</span></div>
						<div><strong>{openAlerts}</strong><span>Alerts requiring attention</span></div>
					</div>
				{/if}
			</section>
		</div>
	{:else if primitiveLoading === activePrimitive}
		<div class="loading"><Loader2 size={20} class="spin" />Loading {activePrimitive}...</div>
	{:else if activeState}
		<section class="data-panel">
			<div class="section-heading">
				<div><span>Shared operating primitive</span><h2>{activeState.table.name}</h2><p>{activeState.table.description}</p></div>
				<span class="count-badge">{activeState.rows.length}</span>
			</div>
			{#if activeState.rows.length}
				<div class="data-table-wrap">
					<table>
						<thead><tr>{#each activeState.table.columns.filter((column) => !column.is_hidden) as column}<th>{column.name}</th>{/each}{#if activeState.table.name === 'Approvals'}<th>Decision</th>{/if}</tr></thead>
						<tbody>
							{#each activeState.rows as row (row.id)}
								{@const approvalStatus = String(fieldValue(activeState, row, 'status'))}
								<tr>
									{#each activeState.table.columns.filter((column) => !column.is_hidden) as column}<td>{displayValue(row.data[column.id] ?? row.data[column.name])}</td>{/each}
									{#if activeState.table.name === 'Approvals'}
										<td>
											{#if approvalStatus.toLowerCase() === 'pending'}
								<div class="approval-actions compact"><button class="approve" disabled={decisionBusy === row.id} onclick={() => setDemoApproval(row.id, 'Approved')}>Approve</button><button disabled={decisionBusy === row.id} onclick={() => setDemoApproval(row.id, 'Returned')}>Return</button></div>
											{:else}<span class="status-pill">{approvalStatus}</span>{/if}
										</td>
									{/if}
								</tr>
							{/each}
						</tbody>
					</table>
				</div>
			{:else}<div class="empty-state">This primitive is defined and ready for data.</div>{/if}
		</section>
	{/if}
	</div>
</div>

<style>
	.operating-module-scroll {
		flex: 1 1 auto;
		min-height: 0;
		width: 100%;
		overflow-x: hidden;
		overflow-y: auto;
		overscroll-behavior: contain;
		scrollbar-gutter: stable;
	}
	.operating-module {
		box-sizing: border-box;
		width: 100%;
		max-width: 1480px;
		margin: 0 auto;
		padding: 28px 32px 48px;
		color: var(--foreground, #171717);
	}
	.module-header { display: flex; justify-content: space-between; gap: 32px; align-items: flex-start; padding-bottom: 24px; border-bottom: 1px solid #e5e7eb; }
	.eyebrow { display: flex; gap: 8px; align-items: center; margin-bottom: 10px; }
	.eyebrow span { font-size: 11px; font-weight: 700; text-transform: uppercase; color: #5b6472; }
	.eyebrow span + span { border-left: 1px solid #d7dce2; padding-left: 8px; }
	.eyebrow .demo-label { color: #a16207; }
	h1 { margin: 0; font-size: clamp(28px, 3vw, 42px); line-height: 1.08; letter-spacing: 0; }
	.module-header p { max-width: 760px; color: #657080; margin: 10px 0 0; line-height: 1.55; }
	.refresh { display: inline-flex; gap: 8px; align-items: center; border: 1px solid #d9dee5; background: white; padding: 9px 12px; border-radius: 6px; font-weight: 600; white-space: nowrap; }
	.error-banner { display: flex; gap: 8px; align-items: center; margin-top: 16px; padding: 11px 13px; background: #fff1f2; color: #9f1239; border: 1px solid #fecdd3; border-radius: 6px; }
	.metrics { display: grid; grid-template-columns: repeat(4, minmax(0, 1fr)); border-bottom: 1px solid #e5e7eb; }
	.metric { display: flex; align-items: center; gap: 12px; min-height: 86px; padding: 16px 20px; border-right: 1px solid #e5e7eb; color: #4b5563; }
	.metric:first-child { padding-left: 0; }
	.metric:last-child { border-right: 0; }
	.metric div { display: flex; flex-direction: column; gap: 2px; }
	.metric strong { font-size: 24px; color: #111827; }
	.metric span { font-size: 12px; color: #6b7280; }
	.metric.attention strong, .metric.attention > :global(svg) { color: #b45309; }
	.systems-strip { display: flex; align-items: center; gap: 14px; min-height: 54px; margin: 14px 0 0; padding: 10px 12px; border: 1px solid #e2e6ea; border-radius: 6px; background: #f8fafc; }
	.systems-label { display: inline-flex; align-items: center; gap: 7px; flex: 0 0 auto; color: #475467; font-size: 12px; font-weight: 700; }
	.system-chips { display: flex; flex-wrap: wrap; gap: 6px; min-width: 0; }
	.system-chips span { padding: 5px 8px; border: 1px solid #d8dee7; border-radius: 4px; background: white; color: #344054; font-size: 11px; font-weight: 600; white-space: nowrap; }
	.systems-map { display: inline-flex; align-items: center; gap: 3px; margin-left: auto; padding: 4px 0; border: 0; background: transparent; color: #344054; font-size: 12px; font-weight: 700; white-space: nowrap; cursor: pointer; }
	.systems-map:hover { color: #111827; text-decoration: underline; }
	.tabs { display: flex; flex-wrap: wrap; gap: 4px; padding: 18px 0 14px; }
	.tabs button { border: 0; background: transparent; padding: 8px 11px; border-radius: 5px; color: #667085; font-size: 13px; white-space: nowrap; cursor: pointer; }
	.tabs button:hover { background: #f2f4f7; color: #344054; }
	.tabs button.active { background: #111827; color: white; }
	.tabs button:focus-visible, .systems-map:focus-visible, .record-list button:focus-visible, .approval-actions button:focus-visible, .refresh:focus-visible { outline: 2px solid #2563eb; outline-offset: 2px; }
	.loading { display: flex; align-items: center; justify-content: center; gap: 10px; min-height: 300px; color: #667085; }
	.overview-grid { display: grid; grid-template-columns: minmax(0, 1.55fr) minmax(300px, 0.9fr); gap: 1px; background: #dfe3e8; border: 1px solid #dfe3e8; }
	.primary-panel, .acceptance-panel, .activity-panel, .data-panel { background: white; padding: 22px; min-width: 0; }
	.section-heading { display: flex; justify-content: space-between; align-items: flex-start; gap: 20px; margin-bottom: 18px; }
	.section-heading span { display: block; text-transform: uppercase; font-size: 10px; font-weight: 700; color: #7a8492; margin-bottom: 5px; }
	.section-heading h2 { margin: 0; font-size: 18px; letter-spacing: 0; }
	.section-heading p { color: #667085; margin: 6px 0 0; font-size: 13px; }
	.count-badge { display: grid !important; place-items: center; min-width: 30px; height: 26px; padding: 0 8px; background: #f1f3f5; border-radius: 4px; color: #394150 !important; font-size: 12px !important; }
	.record-list { border-top: 1px solid #e6e9ed; }
	.record-list button { width: 100%; display: flex; justify-content: space-between; align-items: center; gap: 12px; padding: 13px 2px; border: 0; border-bottom: 1px solid #e6e9ed; background: white; text-align: left; color: #202938; }
	.acceptance-panel ul { list-style: none; padding: 0; margin: 0; display: grid; gap: 12px; }
	.acceptance-panel li { display: flex; gap: 9px; align-items: flex-start; color: #3f4856; font-size: 13px; line-height: 1.45; }
	.acceptance-panel li :global(svg) { flex: 0 0 auto; margin-top: 2px; color: #15803d; }
	.compact-list { display: grid; gap: 10px; }
	.compact-list div { display: grid; gap: 3px; padding-bottom: 10px; border-bottom: 1px solid #edf0f2; }
	.compact-list strong { font-size: 13px; font-weight: 600; }
	.compact-list span { font-size: 11px; color: #7a8492; }
	.approval-list { display: grid; gap: 1px; background: #e5e7eb; }
	.approval-item { display: flex; align-items: center; justify-content: space-between; gap: 16px; background: #fff; padding: 13px 14px; }
	.approval-item > div:first-child { display: grid; gap: 3px; min-width: 0; }
	.approval-item strong { font-size: 13px; color: #202938; }
	.approval-item span { font-size: 11px; color: #7a8492; }
	.approval-actions { display: flex; gap: 6px; flex: 0 0 auto; }
	.approval-actions button { border: 1px solid #d6dbe2; background: #fff; color: #344054; border-radius: 5px; padding: 6px 9px; font-size: 11px; font-weight: 700; }
	.approval-actions button.approve { border-color: #166534; background: #166534; color: #fff; }
	.approval-actions.compact button { padding: 5px 8px; }
	.status-pill { display: inline-flex; width: fit-content; padding: 5px 8px; border-radius: 999px; background: #ecfdf3; color: #166534 !important; font-size: 10px !important; font-weight: 700; text-transform: uppercase; }
	.status-split { display: grid; grid-template-columns: 1fr 1fr; gap: 1px; background: #e5e7eb; }
	.status-split div { display: grid; gap: 3px; background: #f8f9fa; padding: 18px; }
	.status-split strong { font-size: 26px; }
	.status-split span { color: #667085; font-size: 12px; }
	.data-panel { border: 1px solid #dfe3e8; }
	.data-table-wrap { width: 100%; max-width: 100%; overflow-x: auto; overflow-y: hidden; overscroll-behavior-inline: contain; scrollbar-gutter: stable; border: 1px solid #e2e6ea; }
	table { width: max-content; min-width: 100%; border-collapse: collapse; font-size: 12px; }
	th { padding: 10px 12px; background: #f6f7f8; color: #616b78; text-align: left; white-space: nowrap; font-size: 10px; text-transform: uppercase; }
	td { padding: 11px 12px; border-top: 1px solid #e8ebee; color: #303946; max-width: 360px; }
	.empty-state { padding: 40px 16px; color: #7a8492; text-align: center; border: 1px dashed #d8dde3; background: #fafbfc; }
	.spin { animation: spin 0.8s linear infinite; }
	@keyframes spin { to { transform: rotate(360deg); } }
	@media (max-width: 900px) {
		.operating-module { padding: 20px 16px 36px; }
		.metrics { grid-template-columns: 1fr 1fr; }
		.metric:nth-child(2) { border-right: 0; }
		.metric:first-child { padding-left: 20px; }
		.systems-strip { align-items: flex-start; flex-wrap: wrap; }
		.systems-map { width: 100%; margin-left: 23px; }
		.tabs { flex-wrap: nowrap; overflow-x: auto; scrollbar-gutter: stable; }
		.overview-grid { grid-template-columns: 1fr; }
	}
	@media (max-width: 560px) {
		.module-header { display: block; }
		.refresh { margin-top: 18px; }
		.metrics { grid-template-columns: 1fr; }
		.metric { border-right: 0; border-bottom: 1px solid #e5e7eb; }
	}
</style>
