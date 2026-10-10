// OSA-5 Files API Extension
// Additional API endpoints for file operations

import { request } from '../base';
import type { OSAFile, OSAWorkflow } from '$lib/components/osa/types';

/**
 * Get files for a specific workflow
 */
export async function getWorkflowFiles(workflowId: string): Promise<OSAFile[]> {
	const response = await request<{ files: OSAFile[] }>(`/osa/workflows/${workflowId}/files`);
	return response.files;
}

/**
 * Get file content by ID
 */
export async function getFileContent(fileId: string): Promise<{ content: string; file: OSAFile }> {
	return request<{ content: string; file: OSAFile }>(`/osa/files/${fileId}/content`);
}

/**
 * Download a file
 */
export async function downloadFile(fileId: string): Promise<Blob> {
	const response = await fetch(`/api/osa/files/${fileId}/download`, {
		method: 'GET',
		credentials: 'include'
	});

	if (!response.ok) {
		throw new Error(`Download failed: ${response.statusText}`);
	}

	return response.blob();
}

/**
 * Download multiple files as a ZIP archive
 */
export async function downloadFilesAsZip(fileIds: string[]): Promise<Blob> {
	const response = await fetch('/api/osa/files/download-zip', {
		method: 'POST',
		headers: {
			'Content-Type': 'application/json'
		},
		credentials: 'include',
		body: JSON.stringify({ file_ids: fileIds })
	});

	if (!response.ok) {
		throw new Error(`Download failed: ${response.statusText}`);
	}

	return response.blob();
}

/**
 * Install files as a module
 */
export async function installModule(
	workflowId: string,
	options: {
		module_name: string;
		install_path?: string;
		file_ids?: string[];
	}
): Promise<{ success: boolean; module_id: string; message: string }> {
	return request<{ success: boolean; module_id: string; message: string }>(
		'/osa/modules/install',
		{
			method: 'POST',
			body: {
				workflow_id: workflowId,
				...options
			}
		}
	);
}

/**
 * Get all workflows with files
 */
export async function getWorkflows(filters?: {
	status?: string[];
	search?: string;
}): Promise<OSAWorkflow[]> {
	const params = new URLSearchParams();

	if (filters?.status && filters.status.length > 0) {
		params.append('status', filters.status.join(','));
	}

	if (filters?.search) {
		params.append('search', filters.search);
	}

	const queryString = params.toString();
	const url = queryString ? `/osa/workflows?${queryString}` : '/osa/workflows';

	const response = await request<{ workflows: OSAWorkflow[] }>(url);
	return response.workflows;
}

/**
 * Search files across workflows
 */
export async function searchFiles(query: string, filters?: {
	workflow_id?: string;
	file_types?: string[];
}): Promise<OSAFile[]> {
	const params = new URLSearchParams({ q: query });

	if (filters?.workflow_id) {
		params.append('workflow_id', filters.workflow_id);
	}

	if (filters?.file_types && filters.file_types.length > 0) {
		params.append('types', filters.file_types.join(','));
	}

	const response = await request<{ files: OSAFile[] }>(`/osa/files/search?${params.toString()}`);
	return response.files;
}
