// RAG Search API Client
// Calls backend endpoints: /api/rag/*

import { getApiBaseUrl } from '$lib/api/base';
import type {
	HybridSearchRequest,
	HybridSearchResponse,
	SearchExplanationResponse,
	AgenticRAGRequest,
	AgenticRAGResponse,
	ReRankRequest,
	ReRankResponse,
	CreateMemoryRequest,
	RAGMemory,
	ListMemoriesRequest,
	ListMemoriesResponse,
	RAGErrorResponse
} from './types';

// Helper to get auth headers
function getHeaders(): HeadersInit {
	return {
		'Content-Type': 'application/json'
		// Session cookie is automatically included
	};
}

// Helper to handle API errors
async function handleResponse<T>(response: Response): Promise<T> {
	if (!response.ok) {
		const error: RAGErrorResponse = await response.json().catch(() => ({
			error: `HTTP ${response.status}: ${response.statusText}`
		}));
		throw new Error(error.error || error.details || 'RAG API request failed');
	}
	return response.json();
}

/**
 * Hybrid Search - Combines semantic and keyword search
 * POST /api/rag/search/hybrid
 */
export async function hybridSearch(
	request: HybridSearchRequest,
	serverUrl?: string
): Promise<HybridSearchResponse> {
	const baseUrl = serverUrl || getApiBaseUrl();
	const response = await fetch(`${baseUrl}/rag/search/hybrid`, {
		method: 'POST',
		headers: getHeaders(),
		credentials: 'include',
		body: JSON.stringify({
			query: request.query,
			semantic_weight: request.semantic_weight ?? 0.7,
			keyword_weight: request.keyword_weight ?? 0.3,
			max_results: request.max_results ?? 10,
			min_similarity: request.min_similarity ?? 0.3
		})
	});

	return handleResponse<HybridSearchResponse>(response);
}

/**
 * Get search explanation and breakdown
 * POST /api/rag/search/hybrid/explain
 */
export async function getSearchExplanation(
	request: HybridSearchRequest,
	serverUrl?: string
): Promise<SearchExplanationResponse> {
	const baseUrl = serverUrl || getApiBaseUrl();
	const response = await fetch(`${baseUrl}/rag/search/hybrid/explain`, {
		method: 'POST',
		headers: getHeaders(),
		credentials: 'include',
		body: JSON.stringify({
			query: request.query,
			semantic_weight: request.semantic_weight ?? 0.7,
			keyword_weight: request.keyword_weight ?? 0.3,
			max_results: request.max_results ?? 10,
			min_similarity: request.min_similarity ?? 0.3
		})
	});

	return handleResponse<SearchExplanationResponse>(response);
}

/**
 * Agentic RAG - Intelligent adaptive retrieval
 * POST /api/rag/retrieve
 */
export async function agenticRAG(
	request: AgenticRAGRequest,
	serverUrl?: string
): Promise<AgenticRAGResponse> {
	const baseUrl = serverUrl || getApiBaseUrl();
	const response = await fetch(`${baseUrl}/rag/retrieve`, {
		method: 'POST',
		headers: getHeaders(),
		credentials: 'include',
		body: JSON.stringify({
			query: request.query,
			max_results: request.max_results ?? 10,
			min_quality_score: request.min_quality_score ?? 0.5,
			project_id: request.project_id,
			task_id: request.task_id,
			use_personalization: request.use_personalization ?? false
		})
	});

	return handleResponse<AgenticRAGResponse>(response);
}

/**
 * Re-rank search results
 * POST /api/rag/search/rerank
 */
export async function reRankResults(
	request: ReRankRequest,
	serverUrl?: string
): Promise<ReRankResponse> {
	const baseUrl = serverUrl || getApiBaseUrl();
	const response = await fetch(`${baseUrl}/rag/search/rerank`, {
		method: 'POST',
		headers: getHeaders(),
		credentials: 'include',
		body: JSON.stringify({
			results: request.results,
			query: request.query,
			project_id: request.project_id,
			task_id: request.task_id
		})
	});

	return handleResponse<ReRankResponse>(response);
}

/**
 * List memories
 * GET /api/rag/memories
 */
export async function listMemories(
	params?: ListMemoriesRequest,
	serverUrl?: string
): Promise<ListMemoriesResponse> {
	const baseUrl = serverUrl || getApiBaseUrl();
	const searchParams = new URLSearchParams();
	if (params?.type) searchParams.set('type', params.type);
	if (params?.limit) searchParams.set('limit', params.limit.toString());
	if (params?.offset) searchParams.set('offset', params.offset.toString());

	const url = `${baseUrl}/rag/memories${searchParams.toString() ? '?' + searchParams.toString() : ''}`;
	const response = await fetch(url, {
		method: 'GET',
		headers: getHeaders(),
		credentials: 'include'
	});

	return handleResponse<ListMemoriesResponse>(response);
}

/**
 * Get specific memory
 * GET /api/rag/memories/:id
 */
export async function getMemory(memoryId: string, serverUrl?: string): Promise<RAGMemory> {
	const baseUrl = serverUrl || getApiBaseUrl();
	const response = await fetch(`${baseUrl}/rag/memories/${memoryId}`, {
		method: 'GET',
		headers: getHeaders(),
		credentials: 'include'
	});

	return handleResponse<RAGMemory>(response);
}

/**
 * Create memory
 * POST /api/rag/memories
 */
export async function createMemory(
	request: CreateMemoryRequest,
	serverUrl?: string
): Promise<RAGMemory> {
	const baseUrl = serverUrl || getApiBaseUrl();
	const response = await fetch(`${baseUrl}/rag/memories`, {
		method: 'POST',
		headers: getHeaders(),
		credentials: 'include',
		body: JSON.stringify({
			title: request.title,
			summary: request.summary,
			content: request.content,
			memory_type: request.memory_type,
			category: request.category,
			source_type: request.source_type,
			source_id: request.source_id,
			project_id: request.project_id,
			node_id: request.node_id,
			importance_score: request.importance_score ?? 0.5,
			tags: request.tags || []
		})
	});

	return handleResponse<RAGMemory>(response);
}

/**
 * Update memory
 * PUT /api/rag/memories/:id
 */
export async function updateMemory(
	memoryId: string,
	updates: Partial<CreateMemoryRequest>,
	serverUrl?: string
): Promise<RAGMemory> {
	const baseUrl = serverUrl || getApiBaseUrl();
	const response = await fetch(`${baseUrl}/rag/memories/${memoryId}`, {
		method: 'PUT',
		headers: getHeaders(),
		credentials: 'include',
		body: JSON.stringify(updates)
	});

	return handleResponse<RAGMemory>(response);
}

/**
 * Delete memory
 * DELETE /api/rag/memories/:id
 */
export async function deleteMemory(
	memoryId: string,
	serverUrl?: string
): Promise<{ message: string }> {
	const baseUrl = serverUrl || getApiBaseUrl();
	const response = await fetch(`${baseUrl}/rag/memories/${memoryId}`, {
		method: 'DELETE',
		headers: getHeaders(),
		credentials: 'include'
	});

	return handleResponse<{ message: string }>(response);
}

/**
 * Check if RAG services are available
 */
export async function isRAGAvailable(serverUrl?: string): Promise<boolean> {
	try {
		const baseUrl = serverUrl || getApiBaseUrl();
		const response = await fetch(`${baseUrl}/rag/health`, {
			method: 'GET',
			headers: getHeaders(),
			credentials: 'include'
		});
		return response.ok;
	} catch {
		return false;
	}
}

// Export all types
export type * from './types';
