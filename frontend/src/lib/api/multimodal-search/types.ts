// Multimodal Search Types for BusinessOS Frontend
// Integrates with backend /api/images and /api/search/multimodal endpoints

export type SearchModality = 'text' | 'image' | 'hybrid' | 'cross_modal';
export type ImageFormat = 'image/png' | 'image/jpeg' | 'image/webp' | 'image/gif';

// Image Metadata
export interface ImageMetadata {
	caption?: string;
	description?: string;
	tags?: string[];
	context_id?: string;
	project_id?: string;
	conversation_id?: string;
	detected_objects?: string[];
	ocr_text?: string;
	width?: number;
	height?: number;
}

// Image Embedding Result (from backend)
export interface ImageEmbedding {
	id: string;
	user_id: string;
	image_url?: string;
	image_data?: string; // Base64
	caption?: string;
	description?: string;
	metadata?: ImageMetadata;
	created_at: string;
	updated_at?: string;
}

// Upload Response
export interface ImageUploadResponse {
	id: string;
	user_id: string;
	filename?: string;
	size?: number;
	created_at: string;
	metadata?: ImageMetadata;
	message: string;
}

// Multimodal Search Options
export interface MultimodalSearchOptions {
	// Query
	query?: string; // Text query (optional)
	image?: File | string; // File object or base64 string

	// Weights (must sum to 1.0)
	semantic_weight?: number; // Default: 0.4
	keyword_weight?: number; // Default: 0.3
	image_weight?: number; // Default: 0.3

	// Behavior
	max_results?: number; // Default: 20
	include_text?: boolean; // Include text results
	include_images?: boolean; // Include image results
	rerank_enabled?: boolean; // Apply re-ranking

	// Filters
	context_ids?: string[];
	project_ids?: string[];
	min_similarity?: number;
}

// Search Result (can be text or image)
export interface MultimodalSearchResult {
	id: string;
	type: 'text' | 'image' | 'hybrid';
	score: number;
	similarity: number;

	// Text result fields
	context_id?: string;
	content?: string;
	title?: string;

	// Image result fields
	image_id?: string;
	image_url?: string;
	image_caption?: string;
	image_data?: string; // Base64 for small images

	// Common
	user_id: string;
	metadata?: Record<string, any>;
	source: string; // 'semantic', 'keyword', 'image', 'cross_modal', 'hybrid'
	created_at?: string;
}

// Search Response
export interface MultimodalSearchResponse {
	results: MultimodalSearchResult[];
	count: number;
	query?: string;
	options: Partial<MultimodalSearchOptions>;
	modalities_used: string[];
}

// Similar Images Request
export interface SimilarImagesRequest {
	image: File | string; // Base64
	max_results?: number;
}

// Similar Images Response
export interface SimilarImagesResponse {
	results: ImageEmbedding[];
	count: number;
}

// Text-to-Images Request (Cross-modal)
export interface TextToImagesRequest {
	query: string;
	max_results?: number;
	context_ids?: string[];
	project_ids?: string[];
}

// Text-to-Images Response
export interface TextToImagesResponse {
	results: MultimodalSearchResult[];
	count: number;
	query: string;
}

// Image Upload Request (JSON)
export interface ImageUploadRequest {
	image: string; // Base64
	caption?: string;
	description?: string;
	tags?: string[];
	context_id?: string;
	project_id?: string;
	metadata?: Record<string, any>;
}

// Supported Modalities Response
export interface SupportedModalities {
	modalities: string[];
	features: {
		text_search: boolean;
		semantic_search: boolean;
		keyword_search: boolean;
		image_search: boolean;
		cross_modal: boolean;
		hybrid_search: boolean;
		reranking: boolean;
	};
}

// Image Collection (for organizing images)
export interface ImageCollection {
	id: string;
	user_id: string;
	name: string;
	description?: string;
	thumbnail_image_id?: string;
	image_count?: number;
	is_public: boolean;
	created_at: string;
	updated_at: string;
}

// Image Tag
export interface ImageTag {
	id: string;
	image_id: string;
	tag: string;
	confidence?: number; // 0.0-1.0 for auto-generated tags
	source: 'user' | 'auto' | 'ai';
	created_at: string;
}

// Error Response
export interface MultimodalErrorResponse {
	error: string;
	details?: string;
	code?: string;
}

// Utility type for file upload progress
export interface UploadProgress {
	loaded: number;
	total: number;
	percentage: number;
	status: 'idle' | 'uploading' | 'processing' | 'complete' | 'error';
	error?: string;
}

// Image Preview (for UI)
export interface ImagePreview {
	file: File;
	preview_url: string; // Object URL
	base64?: string;
	name: string;
	size: number;
	type: string;
	width?: number;
	height?: number;
}
