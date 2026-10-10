/**
 * OSA Build Onboarding Types
 * Types for the OSA onboarding flow integration
 * Matches backend Groq-powered analysis API (Go handlers.OSAOnboardingHandler)
 */

/**
 * Analysis status from backend
 */
export type AnalysisStatus = 'analyzing' | 'completed' | 'failed';

/**
 * Request payload for starting user analysis
 * POST /api/v1/osa-onboarding/analyze
 */
export interface AnalyzeUserRequest {
	user_id: string;
	workspace_id: string;
	max_emails?: number; // Default: 50
}

/**
 * Response from analyze endpoint (starts async analysis)
 * Returns analysis_id for polling/streaming
 */
export interface AnalyzeUserResponse {
	analysis_id: string;
	status: AnalysisStatus; // Will be 'analyzing' initially
}

/**
 * Progress response from GET /api/v1/osa-onboarding/analyze/:id
 */
export interface AnalysisProgressResponse {
	analysis_id: string;
	status: AnalysisStatus;
	insights?: string[]; // 3 conversational insight phrases
	interests?: string[];
	tools_used?: string[];
	summary?: string;
	error?: string;
}

/**
 * SSE streaming event from analysis progress stream
 */
export interface AnalysisStreamEvent {
	type: 'progress' | 'done' | 'error';
	data?: {
		status: AnalysisStatus;
		insights?: string[];
		interests?: string[];
	};
	content?: string; // For error messages
}

/**
 * Starter app from backend
 */
export interface StarterApp {
	id: string;
	title: string;
	description: string;
	icon_emoji: string;
	category: string;
	reasoning: string;
	customization_prompt: string;
	based_on_interests: string[];
	based_on_tools: string[];
	base_module: string; // 'crm', 'tasks', 'projects', etc.
	module_customizations: Record<string, any>;
	generation_model: string; // 'llama-3.3-70b-versatile'
	ai_provider: string; // 'groq'
	display_order: number;
	status: string; // 'ready', 'pending', etc.
}

/**
 * Request payload for generating starter apps
 * POST /api/v1/osa-onboarding/generate-apps
 */
export interface GenerateAppsRequest {
	user_id: string;
	workspace_id: string;
	analysis_id: string;
}

/**
 * Response from generate-apps endpoint
 */
export interface GenerateAppsResponse {
	apps: StarterApp[];
	total_apps: number;
}
