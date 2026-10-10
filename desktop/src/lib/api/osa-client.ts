// OSA API Client
import type {
	OSAWorkflow,
	OSAWorkflowDetail,
	OSAWorkflowsResponse,
	OSAWorkflowFilesResponse,
	OSAFileContentResponse,
	OSAWebhooksResponse
} from './osa-types';

const API_BASE = '/api/osa';

/**
 * Fetch all workflows for the current user
 */
export async function fetchWorkflows(): Promise<OSAWorkflow[]> {
	const response = await fetch(`${API_BASE}/workflows`, {
		headers: {
			'Content-Type': 'application/json'
		},
		credentials: 'include'
	});

	if (!response.ok) {
		throw new Error(`Failed to fetch workflows: ${response.statusText}`);
	}

	const data = (await response.json()) as OSAWorkflowsResponse;
	return data.workflows;
}

/**
 * Fetch a single workflow by ID
 */
export async function fetchWorkflow(id: string): Promise<OSAWorkflowDetail> {
	const response = await fetch(`${API_BASE}/workflows/${id}`, {
		headers: {
			'Content-Type': 'application/json'
		},
		credentials: 'include'
	});

	if (!response.ok) {
		throw new Error(`Failed to fetch workflow: ${response.statusText}`);
	}

	return (await response.json()) as OSAWorkflowDetail;
}

/**
 * Fetch all files for a workflow
 */
export async function fetchWorkflowFiles(id: string): Promise<OSAWorkflowFilesResponse> {
	const response = await fetch(`${API_BASE}/workflows/${id}/files`, {
		headers: {
			'Content-Type': 'application/json'
		},
		credentials: 'include'
	});

	if (!response.ok) {
		throw new Error(`Failed to fetch workflow files: ${response.statusText}`);
	}

	return (await response.json()) as OSAWorkflowFilesResponse;
}

/**
 * Fetch a specific file content from a workflow
 */
export async function fetchFileContent(
	workflowId: string,
	fileType: string
): Promise<OSAFileContentResponse> {
	const response = await fetch(`${API_BASE}/workflows/${workflowId}/files/${fileType}`, {
		headers: {
			'Content-Type': 'application/json'
		},
		credentials: 'include'
	});

	if (!response.ok) {
		throw new Error(`Failed to fetch file content: ${response.statusText}`);
	}

	return (await response.json()) as OSAFileContentResponse;
}

/**
 * Trigger a manual sync from OSA-5 workspace
 */
export async function triggerSync(): Promise<void> {
	const response = await fetch(`${API_BASE}/sync/trigger`, {
		method: 'POST',
		headers: {
			'Content-Type': 'application/json'
		},
		credentials: 'include'
	});

	if (!response.ok) {
		throw new Error(`Failed to trigger sync: ${response.statusText}`);
	}
}

/**
 * Fetch all webhooks for the current user
 */
export async function fetchWebhooks(): Promise<OSAWebhooksResponse> {
	const response = await fetch(`${API_BASE}/webhooks`, {
		headers: {
			'Content-Type': 'application/json'
		},
		credentials: 'include'
	});

	if (!response.ok) {
		throw new Error(`Failed to fetch webhooks: ${response.statusText}`);
	}

	return (await response.json()) as OSAWebhooksResponse;
}

/**
 * Format a timestamp as a relative time string
 */
export function formatRelativeTime(timestamp: string | null): string {
	if (!timestamp) return 'Never';

	const date = new Date(timestamp);
	const now = new Date();
	const diffMs = now.getTime() - date.getTime();
	const diffSec = Math.floor(diffMs / 1000);
	const diffMin = Math.floor(diffSec / 60);
	const diffHour = Math.floor(diffMin / 60);
	const diffDay = Math.floor(diffHour / 24);

	if (diffSec < 60) return `${diffSec}s ago`;
	if (diffMin < 60) return `${diffMin}m ago`;
	if (diffHour < 24) return `${diffHour}h ago`;
	if (diffDay < 7) return `${diffDay}d ago`;

	return date.toLocaleDateString();
}

/**
 * Format file size in human-readable format
 */
export function formatFileSize(bytes: number): string {
	const units = ['B', 'KB', 'MB', 'GB'];
	let size = bytes;
	let unitIndex = 0;

	while (size >= 1024 && unitIndex < units.length - 1) {
		size /= 1024;
		unitIndex++;
	}

	return `${size.toFixed(1)} ${units[unitIndex]}`;
}
