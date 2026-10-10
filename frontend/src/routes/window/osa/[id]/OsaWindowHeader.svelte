<script lang="ts">
	import { goto } from '$app/navigation';
	import {
		ArrowLeft,
		Loader2,
		AlertCircle,
		CheckCircle,
		XCircle,
		Clock,
		Package,
		ExternalLink,
		Rocket
	} from 'lucide-svelte';

	interface Workflow {
		id: string;
		name: string;
		display_name?: string;
		description: string;
		status: string;
		created_at: string;
		generated_at?: string;
	}

	interface Props {
		workflow: Workflow;
		fileCount: number;
		installing: boolean;
		installSuccess: boolean;
		installError: string | null;
		deploying: boolean;
		deploymentUrl: string;
		deployError: string;
		onInstall: () => void;
		onDeploy: () => void;
	}

	let {
		workflow,
		fileCount,
		installing,
		installSuccess,
		installError,
		deploying,
		deploymentUrl,
		deployError,
		onInstall,
		onDeploy
	}: Props = $props();

	function getStatusIcon(status: string) {
		switch (status) {
			case 'completed': return CheckCircle;
			case 'failed': return XCircle;
			case 'processing': return Clock;
			default: return Clock;
		}
	}

	function getStatusColor(status: string) {
		switch (status) {
			case 'completed': return 'text-green-400';
			case 'failed': return 'text-red-400';
			case 'processing': return 'text-yellow-400';
			default: return 'text-gray-400';
		}
	}

	function formatDate(dateString: string): string {
		const date = new Date(dateString);
		return date.toLocaleString('en-US', {
			year: 'numeric',
			month: 'short',
			day: 'numeric',
			hour: '2-digit',
			minute: '2-digit'
		});
	}
</script>

<div class="workflow-header">
	<button class="back-btn" onclick={() => goto('/window')} aria-label="Back to Desktop">
		<ArrowLeft size={16} />
		<span>Back</span>
	</button>

	<div class="workflow-info">
		<div class="workflow-title-row">
			<h1 class="workflow-title">{workflow.display_name || workflow.name}</h1>
			<div class="status-badge" class:status-completed={workflow.status === 'completed'}>
				<svelte:component
					this={getStatusIcon(workflow.status)}
					size={16}
					class={getStatusColor(workflow.status)}
				/>
				<span>{workflow.status}</span>
			</div>
		</div>

		<p class="workflow-description">{workflow.description}</p>

		<div class="workflow-meta">
			<span>{fileCount} files</span>
			<span>•</span>
			<span>Created {formatDate(workflow.created_at)}</span>
			{#if workflow.generated_at}
				<span>•</span>
				<span>Generated {formatDate(workflow.generated_at)}</span>
			{/if}
		</div>
	</div>

	<button class="install-btn" onclick={onInstall} disabled={installing || installSuccess}>
		{#if installing}
			<Loader2 size={16} class="animate-spin" />
			<span>Installing...</span>
		{:else if installSuccess}
			<CheckCircle size={16} />
			<span>Installed!</span>
		{:else}
			<Package size={16} />
			<span>Install Module</span>
		{/if}
	</button>

	<button class="deploy-btn" onclick={onDeploy} disabled={deploying}>
		{#if deploying}
			<Loader2 size={16} class="animate-spin" />
			<span>Deploying...</span>
		{:else if deploymentUrl}
			<ExternalLink size={16} />
			<a href={deploymentUrl} target="_blank">Running</a>
		{:else}
			<Rocket size={16} />
			<span>Deploy & Run</span>
		{/if}
	</button>
</div>

{#if installError}
	<div class="install-error">
		<AlertCircle size={16} />
		<span>{installError}</span>
	</div>
{/if}

{#if deployError}
	<div class="deploy-error">
		<AlertCircle size={16} />
		<span>{deployError}</span>
	</div>
{/if}

<style>
	.workflow-header {
		display: flex;
		align-items: flex-start;
		gap: 20px;
		padding: 24px;
		background: #1e293b;
		border-bottom: 1px solid #334155;
	}

	.back-btn {
		display: flex;
		align-items: center;
		gap: 8px;
		padding: 8px 16px;
		background: #0f172a;
		border: 1px solid #334155;
		border-radius: 6px;
		color: #e2e8f0;
		font-size: 14px;
		cursor: pointer;
		transition: all 0.15s ease;
		white-space: nowrap;
	}

	.back-btn:hover {
		background: #1e293b;
		border-color: #60a5fa;
	}

	.workflow-info {
		flex: 1;
		min-width: 0;
	}

	.workflow-title-row {
		display: flex;
		align-items: center;
		gap: 12px;
		margin-bottom: 8px;
	}

	.workflow-title {
		font-size: 24px;
		font-weight: 600;
		color: #f1f5f9;
		margin: 0;
	}

	.status-badge {
		display: flex;
		align-items: center;
		gap: 6px;
		padding: 4px 12px;
		background: #1e293b;
		border: 1px solid #334155;
		border-radius: 12px;
		font-size: 12px;
		font-weight: 500;
		text-transform: capitalize;
	}

	.status-badge.status-completed {
		background: #064e3b;
		border-color: #059669;
	}

	.workflow-description {
		color: #94a3b8;
		font-size: 14px;
		margin: 0 0 12px 0;
		line-height: 1.5;
	}

	.workflow-meta {
		display: flex;
		align-items: center;
		gap: 8px;
		font-size: 13px;
		color: #64748b;
	}

	.install-btn {
		display: flex;
		align-items: center;
		gap: 8px;
		padding: 10px 20px;
		background: #3b82f6;
		border: none;
		border-radius: 6px;
		color: white;
		font-size: 14px;
		font-weight: 500;
		cursor: pointer;
		transition: all 0.15s ease;
		white-space: nowrap;
	}

	.install-btn:hover:not(:disabled) {
		background: #2563eb;
	}

	.install-btn:disabled {
		opacity: 0.6;
		cursor: not-allowed;
	}

	.deploy-btn {
		display: flex;
		align-items: center;
		gap: 8px;
		padding: 10px 20px;
		background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
		color: white;
		border: none;
		border-radius: 8px;
		font-size: 14px;
		font-weight: 500;
		cursor: pointer;
		transition: all 0.2s;
		white-space: nowrap;
	}

	.deploy-btn:hover:not(:disabled) {
		transform: translateY(-2px);
		box-shadow: 0 4px 12px rgba(102, 126, 234, 0.4);
	}

	.deploy-btn:disabled {
		opacity: 0.6;
		cursor: not-allowed;
	}

	.deploy-btn a {
		color: white;
		text-decoration: none;
	}

	.install-error,
	.deploy-error {
		display: flex;
		align-items: center;
		gap: 8px;
		padding: 12px 24px;
		background: #7f1d1d;
		border-bottom: 1px solid #991b1b;
		color: #fca5a5;
		font-size: 14px;
	}

	@media (max-width: 768px) {
		.workflow-header {
			flex-wrap: wrap;
			gap: 12px;
			padding: 16px;
		}
		.workflow-info {
			order: 1;
			width: 100%;
		}
		.back-btn {
			order: 0;
		}
		.install-btn,
		.deploy-btn {
			flex: 1;
			justify-content: center;
		}
		.workflow-title {
			font-size: 18px;
		}
		.workflow-title-row {
			flex-wrap: wrap;
			gap: 8px;
		}
		.workflow-meta {
			flex-wrap: wrap;
			gap: 4px;
		}
	}

	@media (max-width: 480px) {
		.workflow-header {
			padding: 12px;
		}
		.install-btn,
		.deploy-btn {
			width: 100%;
			justify-content: center;
		}
	}
</style>
