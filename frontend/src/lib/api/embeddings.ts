// Embedding API client for semantic search and RAG
import { getApiBaseUrl } from './base';

// API Base URL helper - uses centralized config from base.ts
function getApiBase(): string {
	if (typeof window === 'undefined') {
		return import.meta.env.VITE_API_URL || '/api';
	}
	return getApiBaseUrl();
}

export interface Block {
	id: string;
	type: string;
	content: string;
}

export interface SearchResult {
	context_id: string;
	block_id: string;
	block_type: string;
	content: string;
	context_name: string;
	context_type: string;
	parent_id: string | null;
	similarity: number;
}

export interface ProfileContext {
	id: string;
	name: string;
	type: string;
	system_prompt?: string;
	content?: string;
}

export interface RelevantBlock {
	context_id: string;
	document_name: string;
	block_content: string;
	block_type: string;
	similarity: number;
}

export interface RelatedDoc {
	id: string;
	name: string;
	type: string;
}

export interface HierarchicalContext {
	query: string;
	profile_context?: ProfileContext;
	relevant_blocks: RelevantBlock[];
	related_docs?: RelatedDoc[];
	sibling_docs?: RelatedDoc[];
}

export interface EmbeddingStats {
	total_documents: number;
	indexed_documents: number;
	total_blocks: number;
	model: string;
	dimensions: number;
}

class EmbeddingsApi {
	private async request<T>(endpoint: string, options: RequestInit = {}): Promise<T> {
		const baseUrl = getApiBase();
		const response = await fetch(`${baseUrl}${endpoint}`, {
			...options,
			credentials: 'include',
			headers: {
				'Content-Type': 'application/json',
				...options.headers
			}
		});

		if (!response.ok) {
			const error = await response.json().catch(() => ({ error: 'Request failed' }));
			throw new Error(error.error || 'Request failed');
		}

		return response.json();
	}

	/**
	 * Index a document's blocks for semantic search
	 * Called automatically after document save
	 */
	async indexDocument(contextId: string, blocks: Block[]): Promise<{ status: string; blocks_count: number }> {
		return this.request(`/embeddings/index/${contextId}`, {
			method: 'POST',
			body: JSON.stringify({ blocks })
		});
	}

	/**
	 * Perform semantic search across all user documents
	 */
	async search(query: string, limit = 10): Promise<{ query: string; results: SearchResult[]; count: number }> {
		return this.request('/embeddings/search', {
			method: 'POST',
			body: JSON.stringify({ query, limit })
		});
	}

	/**
	 * Build hierarchical context for AI queries (RAG)
	 */
	async buildAIContext(query: string, limit = 5): Promise<{ context: HierarchicalContext; formatted: string }> {
		return this.request('/embeddings/context', {
			method: 'POST',
			body: JSON.stringify({ query, limit })
		});
	}

	/**
	 * Get context for a specific document
	 */
	async getDocumentContext(contextId: string): Promise<{ context: HierarchicalContext; formatted: string }> {
		return this.request(`/embeddings/context/${contextId}`, {
			method: 'GET'
		});
	}

	/**
	 * Get embedding statistics for the current user
	 */
	async getStats(): Promise<EmbeddingStats> {
		return this.request('/embeddings/stats', {
			method: 'GET'
		});
	}

	/**
	 * Check if embedding service is healthy
	 */
	async healthCheck(): Promise<{ status: string; service: string; model: string }> {
		return this.request('/embeddings/health', {
			method: 'GET'
		});
	}
}

export const embeddings = new EmbeddingsApi();
