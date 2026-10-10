<script lang="ts">
	import { onMount } from 'svelte';
	import {
		adminApi,
		type AdminMiosaQuotaInput,
		type AdminMiosaTenantStatus,
		type AdminSettings
	} from '$lib/api/admin';
	import { CheckCircle2, KeyRound, Link2, Loader2, RefreshCw, Server, TriangleAlert } from 'lucide-svelte';

	let { onForbidden }: { onForbidden?: () => void } = $props();

	let loading = $state(true);
	let savingKey = $state(false);
	let savingQuotaFor = $state<string | null>(null);
	let linkingWorkspaceFor = $state<string | null>(null);
	let savingEntitlementFor = $state<string | null>(null);
	let error = $state<string | null>(null);
	let success = $state<string | null>(null);
	let settings = $state<AdminSettings | null>(null);
	let tenant = $state<AdminMiosaTenantStatus | null>(null);
	let apiKey = $state('');
	let selectedWorkspaceId = $state<string | null>(null);
	let quota = $state({
		max_sandboxes: '5',
		max_concurrent: '2',
		max_storage_gb: '10',
		max_credit_cents: ''
	});

	onMount(load);

	async function load() {
		loading = true;
		error = null;
		try {
			const [settingsResult, tenantResult] = await Promise.all([
				adminApi.settings(),
				adminApi.miosaTenant()
			]);
			settings = settingsResult;
			tenant = tenantResult;
			selectedWorkspaceId = tenantResult.external_workspaces[0]?.id ?? null;
		} catch (e) {
			const msg = e instanceof Error ? e.message : 'Failed to load platform settings';
			if (msg.includes('403') || msg.toLowerCase().includes('forbidden')) onForbidden?.();
			else error = msg;
		} finally {
			loading = false;
		}
	}

	async function saveKey() {
		const trimmed = apiKey.trim();
		if (!trimmed) {
			error = 'Enter a MIOSA API key first';
			return;
		}
		savingKey = true;
		error = null;
		success = null;
		try {
			await adminApi.saveMiosaTenantKey(trimmed);
			apiKey = '';
			success = 'BusinessOS MIOSA tenant key saved';
			tenant = await adminApi.miosaTenant();
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to save MIOSA key';
		} finally {
			savingKey = false;
		}
	}

	async function saveQuota(workspaceId: string) {
		savingQuotaFor = workspaceId;
		error = null;
		success = null;
		try {
			await adminApi.setMiosaExternalWorkspaceQuota(workspaceId, quotaPayload());
			success = 'MIOSA external workspace quota saved';
			tenant = await adminApi.miosaTenant();
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to save quota';
		} finally {
			savingQuotaFor = null;
		}
	}

	async function linkWorkspace(workspaceId: string) {
		linkingWorkspaceFor = workspaceId;
		error = null;
		success = null;
		try {
			await adminApi.ensureMiosaExternalWorkspace(workspaceId);
			success = 'MIOSA workspace linked';
			tenant = await adminApi.miosaTenant();
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to link MIOSA workspace';
		} finally {
			linkingWorkspaceFor = null;
		}
	}

	async function setWorkspaceEntitlement(
		workspaceId: string,
		entitlement: { sandbox_enabled?: boolean; computer_enabled?: boolean; desktop_enabled?: boolean },
		message: string
	) {
		savingEntitlementFor = workspaceId;
		error = null;
		success = null;
		try {
			await adminApi.setMiosaExternalWorkspaceEntitlement(workspaceId, entitlement);
			success = message;
			tenant = await adminApi.miosaTenant();
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to update MIOSA access';
		} finally {
			savingEntitlementFor = null;
		}
	}

	function quotaPayload(): AdminMiosaQuotaInput {
		return {
			max_sandboxes: parseNullableInt(quota.max_sandboxes),
			max_concurrent: parseNullableInt(quota.max_concurrent),
			max_storage_gb: parseNullableInt(quota.max_storage_gb),
			max_credit_cents: parseNullableInt(quota.max_credit_cents)
		};
	}

	function parseNullableInt(value: string): number | null {
		const trimmed = value.trim();
		if (!trimmed) return null;
		const parsed = Number.parseInt(trimmed, 10);
		return Number.isFinite(parsed) ? parsed : null;
	}

	function formatNumber(value: number | undefined): string {
		return typeof value === 'number' ? value.toLocaleString() : '0';
	}

	function rollupFor(workspaceId: string) {
		return tenant?.usage_rollup.find((row) => row.external_user_id === workspaceId);
	}
</script>

{#if error}<div class="banner banner--error">{error}</div>{/if}
{#if success}<div class="banner banner--success">{success}</div>{/if}

{#if loading}
	<div class="loading"><Loader2 class="spin" size={20} /> Loading platform config...</div>
{:else}
	<div class="platform-grid">
		<section class="panel span-2">
			<div class="panel-head">
				<div>
					<p class="eyebrow">BusinessOS MIOSA tenant</p>
					<h2>Platform capacity account</h2>
				</div>
				{#if tenant?.configured}
					<span class="status status--ok"><CheckCircle2 size={14} />Configured</span>
				{:else}
					<span class="status"><TriangleAlert size={14} />No key</span>
				{/if}
			</div>

			<div class="key-row">
				<div class="key-status">
					<KeyRound size={16} />
					<div>
						<strong>{tenant?.key_prefix ? `${tenant.key_prefix}...` : 'No tenant key saved'}</strong>
						<span>Raw keys stay server-side in the credential vault.</span>
					</div>
				</div>
				<div class="key-form">
					<input
						type="password"
						bind:value={apiKey}
						placeholder="msk_p_..."
						autocomplete="off"
						aria-label="BusinessOS MIOSA tenant API key"
					/>
					<button class="btn primary" onclick={saveKey} disabled={savingKey}>
						{#if savingKey}<Loader2 class="spin" size={14} />Saving{:else}<KeyRound size={14} />Save key{/if}
					</button>
					<button class="icon-btn" onclick={load} title="Refresh MIOSA status" aria-label="Refresh MIOSA status">
						<RefreshCw size={15} />
					</button>
				</div>
			</div>
		</section>

		<section class="metric">
			<p>Credits</p>
			<strong>{formatNumber(tenant?.credits?.balance)}</strong>
			<span>{tenant?.credit_usage?.total_credits ?? 0} used this period</span>
		</section>

		<section class="metric">
			<p>Tenant</p>
			<strong>{tenant?.tenant?.plan ?? settings?.deployment_mode ?? 'local'}</strong>
			<span>{tenant?.tenant?.status ?? tenant?.capacity_provider ?? 'local'}</span>
		</section>

		<section class="metric">
			<p>External workspaces</p>
			<strong>{tenant?.external_workspaces.length ?? 0}</strong>
			<span>{tenant?.external_workspaces.filter((workspace) => workspace.sandbox_enabled).length ?? 0} enabled for sandboxes</span>
		</section>

		<section class="metric">
			<p>Recent sandboxes</p>
			<strong>{tenant?.recent_sandboxes?.length ?? 0}</strong>
			<span>Created through BusinessOS tenant auth</span>
		</section>
	</div>

	<section class="panel quota-panel">
		<div class="panel-head">
			<div>
				<p class="eyebrow">External workspace controls</p>
				<h2>Workspace sandbox quotas</h2>
			</div>
			<Server size={17} />
		</div>

		<div class="quota-controls">
			<label>
				<span>Max sandboxes</span>
				<input type="number" min="0" bind:value={quota.max_sandboxes} />
			</label>
			<label>
				<span>Max concurrent</span>
				<input type="number" min="0" bind:value={quota.max_concurrent} />
			</label>
			<label>
				<span>Storage GB</span>
				<input type="number" min="0" bind:value={quota.max_storage_gb} />
			</label>
			<label>
				<span>Credit cents</span>
				<input type="number" min="0" bind:value={quota.max_credit_cents} placeholder="optional" />
			</label>
		</div>

		<div class="table-wrap">
			<table>
				<thead>
					<tr>
						<th>Workspace</th>
						<th>Organization</th>
						<th>MIOSA access</th>
						<th>MIOSA workspace</th>
						<th>External ID</th>
						<th>Owner</th>
						<th>Members</th>
						<th>30d usage</th>
						<th></th>
					</tr>
				</thead>
				<tbody>
					{#each tenant?.external_workspaces ?? [] as workspace}
						{@const usage = rollupFor(workspace.id)}
						<tr class:selected={selectedWorkspaceId === workspace.id}>
							<td>
								<strong>{workspace.name}</strong>
								<span>{workspace.slug} - {workspace.plan_type}</span>
							</td>
							<td>
								<strong>{workspace.organization_name || 'Personal'}</strong>
								<span>{workspace.organization_id || 'No organization'}</span>
							</td>
							<td>
								<div class="access-stack">
									<button
										class:primary={workspace.sandbox_enabled}
										class="btn"
										onclick={() =>
											setWorkspaceEntitlement(
												workspace.id,
												{ sandbox_enabled: !workspace.sandbox_enabled },
												workspace.sandbox_enabled ? 'Sandbox terminal access disabled' : 'Sandbox terminal access enabled'
											)}
										disabled={savingEntitlementFor === workspace.id}
									>
										{#if savingEntitlementFor === workspace.id}<Loader2 class="spin" size={13} />Saving{:else}Sandbox{/if}
									</button>
									<button
										class:primary={workspace.computer_enabled}
										class="btn"
										onclick={() =>
											setWorkspaceEntitlement(
												workspace.id,
												{ computer_enabled: !workspace.computer_enabled },
												workspace.computer_enabled ? 'Computer capacity disabled' : 'Computer capacity enabled'
											)}
										disabled={savingEntitlementFor === workspace.id}
									>
										Computer
									</button>
									<button
										class:primary={workspace.desktop_enabled}
										class="btn"
										onclick={() =>
											setWorkspaceEntitlement(
												workspace.id,
												{ desktop_enabled: !workspace.desktop_enabled },
												workspace.desktop_enabled ? 'Desktop control disabled' : 'Desktop control enabled'
											)}
										disabled={savingEntitlementFor === workspace.id}
									>
										Desktop
									</button>
								</div>
							</td>
							<td>
								{#if workspace.miosa_workspace_id}
									<code>{workspace.miosa_workspace_id}</code>
									<span>{workspace.miosa_status || 'linked'}</span>
								{:else}
									<button
										class="btn"
										onclick={() => linkWorkspace(workspace.id)}
										disabled={linkingWorkspaceFor === workspace.id || !tenant?.configured}
									>
										{#if linkingWorkspaceFor === workspace.id}<Loader2 class="spin" size={13} />Linking{:else}<Link2 size={13} />Link{/if}
									</button>
								{/if}
							</td>
							<td>
								<code>{workspace.external_user_id || workspace.id}</code>
								<span>{workspace.external_workspace_id || workspace.id}</span>
							</td>
							<td>{workspace.owner_email || 'Unknown'}</td>
							<td>{workspace.member_count}</td>
							<td>
								{formatNumber(usage?.sandbox_seconds)}s sandbox
								<span>{formatNumber(Number(usage?.credit_cents ?? 0) / 100)} credits</span>
							</td>
							<td>
								<button class="btn" onclick={() => saveQuota(workspace.id)} disabled={savingQuotaFor === workspace.id || !tenant?.configured}>
									{#if savingQuotaFor === workspace.id}<Loader2 class="spin" size={13} />Saving{:else}Apply quota{/if}
								</button>
							</td>
						</tr>
					{/each}
				</tbody>
			</table>
		</div>
	</section>

	<section class="panel quota-panel">
		<div class="panel-head">
			<div>
				<p class="eyebrow">Tenant activity</p>
				<h2>Recent MIOSA sandboxes</h2>
			</div>
			<Server size={17} />
		</div>

		<div class="table-wrap">
			<table>
				<thead>
					<tr>
						<th>Sandbox</th>
						<th>Workspace</th>
						<th>User</th>
						<th>Status</th>
						<th>Preview</th>
						<th>Created</th>
					</tr>
				</thead>
				<tbody>
					{#each tenant?.recent_sandboxes ?? [] as sandbox}
						<tr>
							<td>
								<code>{sandbox.miosa_sandbox_id}</code>
								<span>{sandbox.terminal_session_id || 'No terminal session'}</span>
							</td>
							<td>
								<strong>{sandbox.workspace_name || 'Workspace'}</strong>
								<span>{sandbox.external_workspace_id}</span>
							</td>
							<td>{sandbox.user_email || sandbox.external_user_id}</td>
							<td>{sandbox.status}</td>
							<td>
								{#if sandbox.preview_url}
									<a href={sandbox.preview_url} target="_blank" rel="noreferrer">Open</a>
								{:else}
									<span>None</span>
								{/if}
							</td>
							<td>{new Date(sandbox.created_at).toLocaleString()}</td>
						</tr>
					{:else}
						<tr>
							<td colspan="6"><span>No BusinessOS MIOSA sandboxes created yet.</span></td>
						</tr>
					{/each}
				</tbody>
			</table>
		</div>
	</section>
{/if}

<style>
	.banner { padding: 11px 14px; border-radius: 8px; font-size: 0.83rem; margin-bottom: 12px; }
	.banner--error { background: color-mix(in srgb, #ef4444 12%, transparent); color: #ef4444; }
	.banner--success { background: color-mix(in srgb, #22c55e 12%, transparent); color: #16a34a; }
	.loading { display: flex; align-items: center; justify-content: center; gap: 8px; color: var(--dt3); padding: 40px; }
	.platform-grid { display: grid; grid-template-columns: repeat(4, minmax(0, 1fr)); gap: 12px; margin-bottom: 12px; }
	.span-2 { grid-column: span 4; }
	.panel, .metric { border: 1px solid var(--dbd); border-radius: 8px; background: color-mix(in srgb, var(--dbg2) 74%, transparent); }
	.panel { padding: 16px; }
	.panel-head { display: flex; align-items: flex-start; justify-content: space-between; gap: 16px; margin-bottom: 14px; }
	.eyebrow { margin: 0 0 4px; color: var(--dt3); font-size: 0.68rem; text-transform: uppercase; letter-spacing: 0.08em; font-weight: 700; }
	h2 { margin: 0; font-size: 1rem; font-weight: 680; color: var(--dt); }
	.status { display: inline-flex; align-items: center; gap: 5px; border: 1px solid var(--dbd); color: var(--dt3); padding: 4px 8px; border-radius: 999px; font-size: 0.72rem; font-weight: 650; }
	.status--ok { color: #16a34a; border-color: color-mix(in srgb, #16a34a 35%, transparent); }
	.key-row { display: grid; grid-template-columns: minmax(220px, 1fr) minmax(320px, 1.2fr); gap: 14px; align-items: center; }
	.key-status { display: flex; align-items: center; gap: 10px; color: var(--dt2); }
	.key-status strong { display: block; color: var(--dt); font-size: 0.9rem; }
	.key-status span { display: block; color: var(--dt3); font-size: 0.76rem; margin-top: 2px; }
	.key-form { display: grid; grid-template-columns: 1fr auto auto; gap: 8px; }
	input { width: 100%; border: 1px solid var(--dbd); border-radius: 7px; background: var(--dbg); color: var(--dt); padding: 9px 10px; font-size: 0.84rem; min-width: 0; }
	.btn, .icon-btn { display: inline-flex; align-items: center; justify-content: center; gap: 6px; border: 1px solid var(--dbd); border-radius: 7px; background: var(--dbg); color: var(--dt2); min-height: 36px; padding: 0 11px; cursor: pointer; font-size: 0.8rem; font-weight: 650; white-space: nowrap; }
	.btn.primary { background: var(--dt); color: var(--dbg); border-color: var(--dt); }
	.btn:disabled { opacity: 0.5; cursor: not-allowed; }
	.icon-btn { width: 36px; padding: 0; }
	.access-stack { display: flex; flex-wrap: wrap; gap: 6px; min-width: 210px; }
	.access-stack .btn { min-height: 30px; padding: 0 8px; font-size: 0.74rem; }
	.metric { padding: 14px; min-height: 104px; }
	.metric p { margin: 0 0 8px; color: var(--dt3); font-size: 0.72rem; text-transform: uppercase; letter-spacing: 0.06em; font-weight: 700; }
	.metric strong { display: block; font-size: 1.45rem; color: var(--dt); line-height: 1.1; }
	.metric span { display: block; margin-top: 7px; color: var(--dt3); font-size: 0.77rem; }
	.quota-panel { margin-top: 12px; }
	.quota-controls { display: grid; grid-template-columns: repeat(4, minmax(0, 1fr)); gap: 10px; margin-bottom: 14px; }
	label span { display: block; color: var(--dt3); font-size: 0.72rem; font-weight: 650; margin-bottom: 5px; }
	.table-wrap { overflow-x: auto; border: 1px solid var(--dbd); border-radius: 8px; }
	table { width: 100%; border-collapse: collapse; min-width: 980px; }
	th, td { padding: 11px 12px; border-bottom: 1px solid var(--dbd); text-align: left; font-size: 0.8rem; vertical-align: middle; }
	th { color: var(--dt3); font-size: 0.68rem; text-transform: uppercase; letter-spacing: 0.06em; font-weight: 750; background: color-mix(in srgb, var(--dbg3) 55%, transparent); }
	td strong { display: block; color: var(--dt); font-size: 0.84rem; }
	td span { display: block; color: var(--dt3); margin-top: 2px; font-size: 0.74rem; }
	code { font-family: ui-monospace, SFMono-Regular, Menlo, monospace; color: var(--dt2); word-break: break-all; }
	a { color: #0284c7; font-weight: 650; text-decoration: none; }
	a:hover { text-decoration: underline; }
	tr:last-child td { border-bottom: none; }
	tr.selected td { background: color-mix(in srgb, #38bdf8 5%, transparent); }
	:global(.spin) { animation: spin 0.9s linear infinite; }
	@keyframes spin { to { transform: rotate(360deg); } }
	@media (max-width: 920px) {
		.platform-grid { grid-template-columns: 1fr; }
		.span-2 { grid-column: auto; }
		.key-row { grid-template-columns: 1fr; }
		.quota-controls { grid-template-columns: repeat(2, minmax(0, 1fr)); }
	}
	@media (max-width: 560px) {
		.key-form { grid-template-columns: 1fr; }
		.icon-btn { width: 100%; }
		.quota-controls { grid-template-columns: 1fr; }
	}
</style>
