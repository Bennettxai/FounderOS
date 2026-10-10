<script lang="ts">
	import { Database, LoaderCircle, Route, X } from 'lucide-svelte';
	import type {
		CommunicationRouteScope,
		CommsChannel,
	} from '$lib/api/comms';
	import type { Workspace } from '$lib/api/workspaces';

	interface Props {
		channel: CommsChannel;
		workspaces: Workspace[];
		isSaving?: boolean;
		onSave: (workspaceId: string, scope: CommunicationRouteScope) => void;
		onClear: (scope: CommunicationRouteScope) => void;
	}

	let {
		channel,
		workspaces,
		isSaving = false,
		onSave,
		onClear,
	}: Props = $props();

	let workspaceId = $state('');
	let scope = $state<CommunicationRouteScope>('conversation');

	$effect(() => {
		channel.id;
		workspaceId = channel.routed_workspace_id ?? '';
		scope = channel.routing_scope ?? 'conversation';
	});

	const isAssigned = $derived(Boolean(channel.routed_workspace_id));
	const routeLabel = $derived(
		isAssigned
			? `${channel.routed_workspace_name} via ${channel.routing_scope === 'account' ? 'provider default' : 'conversation route'}`
			: 'Unassigned - excluded from Optimal Engine',
	);
</script>

<div class="cm-route" class:cm-route--assigned={isAssigned}>
	<div class="cm-route__status">
		<span class="cm-route__icon" aria-hidden="true">
			{#if isAssigned}<Database size={15} />{:else}<Route size={15} />{/if}
		</span>
		<div class="cm-route__copy">
			<strong>Workspace route</strong>
			<span>{routeLabel}</span>
		</div>
	</div>

	<div class="cm-route__scope" aria-label="Routing scope">
		<button
			type="button"
			class:active={scope === 'conversation'}
			onclick={() => (scope = 'conversation')}
			disabled={isSaving}
		>
			This conversation
		</button>
		<button
			type="button"
			class:active={scope === 'account'}
			onclick={() => (scope = 'account')}
			disabled={isSaving || channel.provider === 'whatsapp'}
			title={channel.provider === 'whatsapp' ? 'Route WhatsApp conversations individually' : undefined}
		>
			All {channel.provider}
		</button>
	</div>

	<label class="cm-route__select-wrap">
		<span class="sr-only">Optimal Engine workspace</span>
		<select bind:value={workspaceId} disabled={isSaving}>
			<option value="">Choose workspace</option>
			{#each workspaces as workspace (workspace.id)}
				<option value={workspace.id}>{workspace.name}</option>
			{/each}
		</select>
	</label>

	<button
		type="button"
		class="btn-compact btn-compact-primary"
		disabled={!workspaceId || isSaving}
		onclick={() => onSave(workspaceId, scope)}
	>
		{#if isSaving}<LoaderCircle size={14} class="cm-route__spin" />{/if}
		Route
	</button>

	{#if isAssigned}
		<button
			type="button"
			class="btn-compact btn-compact-ghost btn-compact-icon"
			aria-label="Clear workspace route"
			title="Clear workspace route"
			disabled={isSaving}
			onclick={() => onClear(channel.routing_scope ?? scope)}
		>
			<X size={15} />
		</button>
	{/if}
</div>

<style>
	.cm-route {
		flex-shrink: 0;
		display: flex;
		align-items: center;
		gap: var(--space-2);
		min-height: 48px;
		padding: var(--space-2) var(--space-4);
		border-bottom: 1px solid var(--dbd);
		background: var(--dbg2);
	}

	.cm-route--assigned {
		background: color-mix(in srgb, var(--bos-status-success) 6%, var(--dbg));
	}

	.cm-route__status {
		display: flex;
		align-items: center;
		gap: var(--space-2);
		min-width: 220px;
		margin-right: auto;
	}

	.cm-route__icon {
		display: inline-flex;
		color: var(--dt3);
	}

	.cm-route--assigned .cm-route__icon {
		color: var(--bos-status-success);
	}

	.cm-route__copy {
		display: flex;
		flex-direction: column;
		min-width: 0;
	}

	.cm-route__copy strong,
	.cm-route__copy span {
		font-size: var(--text-xs);
		line-height: 1.35;
	}

	.cm-route__copy strong {
		color: var(--dt);
		font-weight: var(--font-semibold);
	}

	.cm-route__copy span {
		color: var(--dt3);
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}

	.cm-route__scope {
		display: inline-flex;
		align-items: center;
		padding: 2px;
		border: 1px solid var(--dbd);
		border-radius: var(--radius-md);
		background: var(--dbg);
	}

	.cm-route__scope button {
		min-height: 26px;
		padding: 0 var(--space-2);
		border: 0;
		border-radius: var(--radius-sm);
		background: transparent;
		color: var(--dt3);
		font: inherit;
		font-size: var(--text-xs);
		cursor: pointer;
	}

	.cm-route__scope button.active {
		background: var(--dbg3);
		color: var(--dt);
		box-shadow: inset 0 0 0 1px var(--dbd);
	}

	.cm-route__select-wrap select {
		height: 30px;
		max-width: 190px;
		padding: 0 28px 0 var(--space-2);
		border: 1px solid var(--dbd);
		border-radius: var(--radius-md);
		background: var(--dbg);
		color: var(--dt);
		font: inherit;
		font-size: var(--text-xs);
	}

	:global(.cm-route__spin) {
		animation: cm-route-spin 0.8s linear infinite;
	}

	@keyframes cm-route-spin {
		to { transform: rotate(360deg); }
	}

	@media (max-width: 980px) {
		.cm-route {
			align-items: stretch;
			flex-wrap: wrap;
		}

		.cm-route__status {
			width: 100%;
		}
	}

	@media (prefers-reduced-motion: reduce) {
		:global(.cm-route__spin) { animation: none; }
	}
</style>
