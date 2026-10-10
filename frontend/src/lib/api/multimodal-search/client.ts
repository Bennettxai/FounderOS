// Multimodal Search API Client
// Calls backend endpoints: /api/images and /api/search/multimodal

import { getApiBaseUrl } from '$lib/api/base';
import type {
	ImageUploadRequest,
	ImageUploadResponse,
	MultimodalSearchOptions,
	MultimodalSearchResponse,
	SimilarImagesRequest,
	SimilarImagesResponse,
	TextToImagesRequest,
	TextToImagesResponse,
	ImageEmbedding,
	SupportedModalities,
	MultimodalErrorResponse,
	UploadProgress
} from './types';

// Helper to convert File to base64
async function fileToBase64(file: File): Promise<string> {
	return new Promise((resolve, reject) => {
		const reader = new FileReader();
		reader.onload = () => {
			const base64 = reader.result as string;
			// Remove data:image/xxx;base64, prefix
			const base64Data = base64.split(',')[1];
			resolve(base64Data);
		};
		reader.onerror = reject;
		reader.readAsDataURL(file);
	});
}

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
		const error: MultimodalErrorResponse = await response.json().catch(() => ({
			error: `HTTP ${response.status}: ${response.statusText}`
		}));
		throw new Error(error.error || error.details || 'API request failed');
	}
	return response.json();
}

/**
 * Upload an image (base64 JSON)
 */
export async function uploadImage(
	request: ImageUploadRequest,
	serverUrl?: string
): Promise<ImageUploadResponse> {
	const baseUrl = serverUrl || getApiBaseUrl();
	const response = await fetch(`${baseUrl}/images/upload`, {
		method: 'POST',
		headers: getHeaders(),
		credentials: 'include',
		body: JSON.stringify(request)
	});

	return handleResponse<ImageUploadResponse>(response);
}

/**
 * Upload an image (multipart form)
 * Better for large images
 */
export async function uploadImageFile(
	file: File,
	caption?: string,
	description?: string,
	onProgress?: (progress: UploadProgress) => void,
	serverUrl?: string
): Promise<ImageUploadResponse> {
	const baseUrl = serverUrl || getApiBaseUrl();

	const formData = new FormData();
	formData.append('image', file);
	if (caption) formData.append('caption', caption);
	if (description) formData.append('description', description);

	return new Promise((resolve, reject) => {
		const xhr = new XMLHttpRequest();

		// Track upload progress
		xhr.upload.addEventListener('progress', (e) => {
			if (e.lengthComputable && onProgress) {
				onProgress({
					loaded: e.loaded,
					total: e.total,
					percentage: (e.loaded / e.total) * 100,
					status: 'uploading'
				});
			}
		});

		xhr.addEventListener('load', () => {
			if (xhr.status >= 200 && xhr.status < 300) {
				try {
					const result = JSON.parse(xhr.responseText);
					if (onProgress) {
						onProgress({
							loaded: 1,
							total: 1,
							percentage: 100,
							status: 'complete'
						});
					}
					resolve(result);
				} catch (error) {
					reject(new Error('Failed to parse response'));
				}
			} else {
				const error = JSON.parse(xhr.responseText).error || 'Upload failed';
				if (onProgress) {
					onProgress({
						loaded: 0,
						total: 1,
						percentage: 0,
						status: 'error',
						error
					});
				}
				reject(new Error(error));
			}
		});

		xhr.addEventListener('error', () => {
			const error = 'Network error';
			if (onProgress) {
				onProgress({
					loaded: 0,
					total: 1,
					percentage: 0,
					status: 'error',
					error
				});
			}
			reject(new Error(error));
		});

		xhr.open('POST', `${baseUrl}/images/upload-file`);
		xhr.withCredentials = true;
		xhr.send(formData);
	});
}

/**
 * Multimodal search (text + image combined)
 */
export async function multimodalSearch(
	options: MultimodalSearchOptions,
	serverUrl?: string
): Promise<MultimodalSearchResponse> {
	const baseUrl = serverUrl || getApiBaseUrl();

	// Convert File to base64 if needed
	let imageBase64: string | undefined;
	if (options.image) {
		if (options.image instanceof File) {
			imageBase64 = await fileToBase64(options.image);
		} else {
			imageBase64 = options.image;
		}
	}

	const requestBody = {
		query: options.query,
		image: imageBase64,
		max_results: options.max_results || 20,
		include_text: options.include_text ?? true,
		include_images: options.include_images ?? true,
		semantic_weight: options.semantic_weight || 0.4,
		keyword_weight: options.keyword_weight || 0.3,
		image_weight: options.image_weight || 0.3,
		rerank_enabled: options.rerank_enabled ?? true,
		context_ids: options.context_ids || []
	};

	const response = await fetch(`${baseUrl}/search/multimodal`, {
		method: 'POST',
		headers: getHeaders(),
		credentials: 'include',
		body: JSON.stringify(requestBody)
	});

	return handleResponse<MultimodalSearchResponse>(response);
}

/**
 * Search for similar images (image similarity)
 */
export async function searchSimilarImages(
	request: SimilarImagesRequest,
	serverUrl?: string
): Promise<SimilarImagesResponse> {
	const baseUrl = serverUrl || getApiBaseUrl();

	// Convert File to base64 if needed
	let imageBase64: string;
	if (request.image instanceof File) {
		imageBase64 = await fileToBase64(request.image);
	} else {
		imageBase64 = request.image;
	}

	const response = await fetch(`${baseUrl}/search/similar-images`, {
		method: 'POST',
		headers: getHeaders(),
		credentials: 'include',
		body: JSON.stringify({
			image: imageBase64,
			max_results: request.max_results || 10
		})
	});

	return handleResponse<SimilarImagesResponse>(response);
}

/**
 * Cross-modal search: Text query → Find images
 */
export async function searchImagesByText(
	request: TextToImagesRequest,
	serverUrl?: string
): Promise<TextToImagesResponse> {
	const baseUrl = serverUrl || getApiBaseUrl();

	const response = await fetch(`${baseUrl}/search/images-by-text`, {
		method: 'POST',
		headers: getHeaders(),
		credentials: 'include',
		body: JSON.stringify({
			query: request.query,
			max_results: request.max_results || 10
		})
	});

	return handleResponse<TextToImagesResponse>(response);
}

/**
 * Get image by ID
 */
export async function getImage(imageId: string, serverUrl?: string): Promise<ImageEmbedding> {
	const baseUrl = serverUrl || getApiBaseUrl();

	const response = await fetch(`${baseUrl}/images/${imageId}`, {
		method: 'GET',
		headers: getHeaders(),
		credentials: 'include'
	});

	return handleResponse<ImageEmbedding>(response);
}

/**
 * Get image data URL (for display)
 */
export function getImageDataUrl(imageId: string, serverUrl?: string): string {
	const baseUrl = serverUrl || getApiBaseUrl();
	return `${baseUrl}/images/${imageId}/data`;
}

/**
 * Delete image
 */
export async function deleteImage(imageId: string, serverUrl?: string): Promise<{ message: string }> {
	const baseUrl = serverUrl || getApiBaseUrl();

	const response = await fetch(`${baseUrl}/images/${imageId}`, {
		method: 'DELETE',
		headers: getHeaders(),
		credentials: 'include'
	});

	return handleResponse<{ message: string }>(response);
}

/**
 * Get supported modalities
 */
export async function getSupportedModalities(serverUrl?: string): Promise<SupportedModalities> {
	const baseUrl = serverUrl || getApiBaseUrl();

	const response = await fetch(`${baseUrl}/search/modalities`, {
		method: 'GET',
		headers: getHeaders(),
		credentials: 'include'
	});

	return handleResponse<SupportedModalities>(response);
}

/**
 * Check if multimodal search is available
 */
export async function isMultimodalAvailable(serverUrl?: string): Promise<boolean> {
	try {
		const modalities = await getSupportedModalities(serverUrl);
		return modalities.features.image_search && modalities.features.cross_modal;
	} catch {
		return false;
	}
}

// Helper to create image preview
export async function createImagePreview(file: File): Promise<{
	preview_url: string;
	base64: string;
	dimensions?: { width: number; height: number };
}> {
	const preview_url = URL.createObjectURL(file);
	const base64 = await fileToBase64(file);

	// Get image dimensions
	return new Promise((resolve) => {
		const img = new Image();
		img.onload = () => {
			resolve({
				preview_url,
				base64,
				dimensions: {
					width: img.width,
					height: img.height
				}
			});
		};
		img.onerror = () => {
			resolve({
				preview_url,
				base64
			});
		};
		img.src = preview_url;
	});
}

// Export all types
export type * from './types';
