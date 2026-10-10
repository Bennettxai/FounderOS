/**
 * OSA Onboarding API Client
 * Handles communication with the Groq-powered OSA Build onboarding backend
 * Backend: Go handlers with ProfileAnalyzerAgent, AppCustomizerAgent, EmailAnalyzerService
 */

import { request, getApiBaseUrl } from '../base';
import type {
	AnalyzeUserRequest,
	AnalyzeUserResponse,
	AnalysisProgressResponse,
	AnalysisStreamEvent,
	GenerateAppsRequest,
	GenerateAppsResponse
} from './types';

/**
 * Start AI-powered user analysis (Gmail + email metadata)
 * POST /api/v1/osa-onboarding/analyze
 *
 * This initiates async analysis:
 * 1. Extracts 50 recent Gmail emails
 * 2. Analyzes patterns (tools, topics, sender domains)
 * 3. Runs Groq AI (llama-3.3-70b-versatile) to generate insights
 * 4. Returns 3 conversational insight phrases
 *
 * @param userId - User ID from session
 * @param workspaceId - Workspace ID
 * @param maxEmails - Max emails to analyze (default: 50)
 * @returns analysis_id for polling/streaming progress
 */
export async function startAnalysis(
	userId: string,
	workspaceId: string,
	maxEmails = 50
): Promise<AnalyzeUserResponse> {
	return request<AnalyzeUserResponse>('/v1/osa-onboarding/analyze', {
		method: 'POST',
		body: {
			user_id: userId,
			workspace_id: workspaceId,
			max_emails: maxEmails
		} as AnalyzeUserRequest
	});
}

/**
 * Get current analysis progress (polling)
 * GET /api/v1/osa-onboarding/analyze/:analysis_id
 *
 * Check status of analysis:
 * - 'analyzing' - Still processing
 * - 'completed' - Ready with insights
 * - 'failed' - Error occurred
 *
 * @param analysisId - Analysis ID from startAnalysis()
 * @returns Current status + insights (if completed)
 */
export async function getAnalysisProgress(
	analysisId: string
): Promise<AnalysisProgressResponse> {
	return request<AnalysisProgressResponse>(`/v1/osa-onboarding/analyze/${analysisId}`);
}

/**
 * Stream analysis progress via Server-Sent Events (SSE)
 * GET /api/v1/osa-onboarding/analyze/:analysis_id/stream
 *
 * Returns a ReadableStream that emits progress events every second.
 * Use this for real-time updates instead of polling.
 *
 * @param analysisId - Analysis ID from startAnalysis()
 * @returns ReadableStream<Uint8Array> for chunked progress updates
 */
export async function streamAnalysisProgress(
	analysisId: string
): Promise<ReadableStream<Uint8Array> | null> {
	const response = await fetch(
		`${getApiBaseUrl()}/v1/osa-onboarding/analyze/${analysisId}/stream`,
		{
			method: 'GET',
			headers: { 'Content-Type': 'text/event-stream' },
			credentials: 'include'
		}
	);

	if (!response.ok) {
		const error = await response.json().catch(() => ({ detail: 'Stream failed' }));
		throw new Error(error.detail || 'Failed to stream analysis progress');
	}

	return response.body;
}

/**
 * Parse SSE stream events from analysis progress
 * Helper function to decode SSE events into AnalysisStreamEvent objects
 *
 * Usage:
 * ```typescript
 * const stream = await streamAnalysisProgress(analysisId);
 * const reader = stream.getReader();
 * const decoder = new TextDecoder();
 *
 * while (true) {
 *   const { done, value } = await reader.read();
 *   if (done) break;
 *   const chunk = decoder.decode(value);
 *   const event = parseSSEEvent(chunk);
 *   if (event) {
 *     console.log('Progress:', event);
 *   }
 * }
 * ```
 */
export function parseSSEEvent(chunk: string): AnalysisStreamEvent | null {
	try {
		// SSE format: event: type\ndata: json\n\n
		const lines = chunk.split('\n');
		let eventType: string | null = null;
		let eventData: string | null = null;

		for (const line of lines) {
			if (line.startsWith('event:')) {
				eventType = line.substring(6).trim();
			} else if (line.startsWith('data:')) {
				eventData = line.substring(5).trim();
			}
		}

		if (!eventType || !eventData) return null;

		const data = JSON.parse(eventData);

		return {
			type: eventType as 'progress' | 'done' | 'error',
			data: data.data,
			content: data.content
		};
	} catch (err) {
		console.warn('Failed to parse SSE event:', err);
		return null;
	}
}

/**
 * Generate personalized starter apps based on analysis
 * POST /api/v1/osa-onboarding/generate-apps
 *
 * Uses Groq AI (llama-3.3-70b-versatile) to recommend 3-4 apps:
 * - Based on user's interests, tools, and patterns
 * - Can customize existing core modules OR create new apps
 * - Returns apps with reasoning and customization prompts
 *
 * @param userId - User ID
 * @param workspaceId - Workspace ID
 * @param analysisId - Completed analysis ID
 * @returns Array of recommended starter apps
 */
export async function generateStarterApps(
	userId: string,
	workspaceId: string,
	analysisId: string
): Promise<GenerateAppsResponse> {
	return request<GenerateAppsResponse>('/v1/osa-onboarding/generate-apps', {
		method: 'POST',
		body: {
			user_id: userId,
			workspace_id: workspaceId,
			analysis_id: analysisId
		} as GenerateAppsRequest
	});
}

// Re-export types for convenience
export type * from './types';
