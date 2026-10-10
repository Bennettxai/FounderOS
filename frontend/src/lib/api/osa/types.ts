// OSA-5 API Types
// Matches backend types from internal/handlers/osa_api.go and internal/integrations/osa/client.go

export interface OSAHealthResponse {
	enabled: boolean;
	status: 'healthy' | 'unhealthy' | 'degraded';
	version: string;
	message?: string;
}

export interface GenerateAppRequest {
	app_name: string;
	description: string;
	template_id?: string;
	complexity?: 'simple' | 'standard' | 'complex';
	config?: Record<string, unknown>;
	generation_context?: Record<string, unknown>;
}

export interface GenerateAppResponse {
	queue_item_id: string;
	status: 'pending';
	message?: string;
}

export interface QueueItemStatus {
	queue_item_id: string;
	status: 'pending' | 'in_progress' | 'completed' | 'failed' | 'cancelled';
	progress_percent: number;
	current_phase: string;
	agent_statuses: {
		frontend: { status: string; progress: number };
		backend: { status: string; progress: number };
		database: { status: string; progress: number };
		deployment: { status: string; progress: number };
	};
	error_message?: string;
	started_at?: string;
	completed_at?: string;
	created_at: string;
	updated_at: string;
}

export interface AppGenerationStatus {
	app_id: string;
	status: 'queued' | 'processing' | 'completed' | 'failed';
	progress: number;
	phase: string;
	message?: string;
	created_at: string;
	updated_at: string;
	completed_at?: string;
	error?: string;
	result?: AppGenerationResult;
}

export interface AppGenerationResult {
	repository_url?: string;
	deployment_url?: string;
	files_generated: number;
	preview_available: boolean;
	artifacts: GeneratedArtifact[];
}

export interface GeneratedArtifact {
	type: 'code' | 'schema' | 'config' | 'documentation' | 'deployment';
	title: string;
	path: string;
	content?: string;
	language?: string;
}

export interface OSAWorkspace {
	id: string;
	name: string;
	description?: string;
	created_at: string;
	app_count: number;
}

export interface OSAWorkspacesResponse {
	workspaces: OSAWorkspace[];
	total: number;
}

// Event types for SSE streaming during generation
export type OSAGenerationEvent =
	| OSAProgressEvent
	| OSAPhaseEvent
	| OSAArtifactEvent
	| OSACompleteEvent
	| OSAErrorEvent;

export interface OSAProgressEvent {
	type: 'progress';
	app_id: string;
	progress: number;
	message: string;
}

export interface OSAPhaseEvent {
	type: 'phase';
	app_id: string;
	phase: string;
	description: string;
}

export interface OSAArtifactEvent {
	type: 'artifact';
	app_id: string;
	artifact: GeneratedArtifact;
}

export interface OSACompleteEvent {
	type: 'complete';
	app_id: string;
	result: AppGenerationResult;
}

export interface OSAErrorEvent {
	type: 'error';
	app_id: string;
	error: string;
}
