// OSA-5 File Explorer Types
// Extends types from $lib/api/osa/types

export type WorkflowStatus = 'queued' | 'processing' | 'completed' | 'failed';

export interface OSAFile {
	id: string;
	workflow_id: string;
	name: string;
	path: string;
	type: 'code' | 'schema' | 'config' | 'documentation' | 'deployment' | 'markdown' | 'yaml' | 'json' | 'text';
	language?: string;
	size: number;
	content?: string;
	created_at: string;
	updated_at: string;
}

export interface OSAWorkflow {
	id: string;
	app_id: string;
	name: string;
	description: string;
	status: WorkflowStatus;
	progress: number;
	phase: string;
	files: OSAFile[];
	created_at: string;
	updated_at: string;
	completed_at?: string;
	error?: string;
}

export interface OSAModule {
	id: string;
	workflow_id: string;
	name: string;
	description: string;
	version: string;
	installed: boolean;
	install_path?: string;
	dependencies: string[];
}

export interface FileTreeNode {
	id: string;
	name: string;
	path: string;
	type: 'file' | 'folder';
	children?: FileTreeNode[];
	file?: OSAFile;
	expanded?: boolean;
}

export interface FilePreviewState {
	file: OSAFile | null;
	content: string | null;
	loading: boolean;
	error: string | null;
}

export interface FilterOptions {
	status?: WorkflowStatus[];
	fileTypes?: string[];
	searchQuery?: string;
	dateFrom?: Date;
	dateTo?: Date;
}
