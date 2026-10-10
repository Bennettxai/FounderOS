// OSA API Types for TypeScript client

export interface OSAWorkflow {
	id: string;
	name: string;
	display_name: string;
	description: string;
	workflow_id: string;
	status: 'generated' | 'processing' | 'failed' | 'deployed';
	files_created: number;
	build_status: string | null;
	created_at: string;
	generated_at: string | null;
	deployed_at: string | null;
	workspace_name: string;
}

export interface OSAWorkflowDetail extends OSAWorkflow {
	metadata: {
		analysis?: string;
		architecture?: string;
		code?: string;
		quality?: string;
		deployment?: string;
		monitoring?: string;
		strategy?: string;
		recommendations?: string;
		discovered_at?: string;
	};
	error_message?: string;
	error_stack?: string;
	workspace_id: string;
}

export interface OSAWorkflowFile {
	type: 'analysis' | 'architecture' | 'code' | 'quality' | 'deployment' | 'monitoring' | 'strategy' | 'recommendations';
	content: string;
	size: number;
}

export interface OSAWorkflowsResponse {
	workflows: OSAWorkflow[];
	count: number;
}

export interface OSAWorkflowFilesResponse {
	workflow_id: string;
	files: OSAWorkflowFile[];
	count: number;
}

export interface OSAFileContentResponse {
	type: string;
	content: string;
	size: number;
}

export interface OSAWebhook {
	id: string;
	event_type: string;
	webhook_url: string;
	enabled: boolean;
	last_triggered_at: string | null;
	success_count: number;
	failure_count: number;
	created_at: string;
}

export interface OSAWebhooksResponse {
	webhooks: OSAWebhook[];
	count: number;
}

export const FILE_TYPE_ICONS: Record<OSAWorkflowFile['type'], string> = {
	analysis: '📊',
	architecture: '🏗️',
	code: '💻',
	quality: '✅',
	deployment: '🚀',
	monitoring: '📈',
	strategy: '🎯',
	recommendations: '💡'
};

export const FILE_TYPE_COLORS: Record<OSAWorkflowFile['type'], string> = {
	analysis: 'blue',
	architecture: 'purple',
	code: 'green',
	quality: 'emerald',
	deployment: 'orange',
	monitoring: 'cyan',
	strategy: 'violet',
	recommendations: 'yellow'
};

export const STATUS_COLORS: Record<OSAWorkflow['status'], string> = {
	generated: 'green',
	processing: 'blue',
	failed: 'red',
	deployed: 'purple'
};

export const STATUS_LABELS: Record<OSAWorkflow['status'], string> = {
	generated: 'Ready',
	processing: 'Processing',
	failed: 'Failed',
	deployed: 'Deployed'
};
