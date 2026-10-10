import { toast } from 'svelte-sonner';

/**
 * API Handler utility for consistent error and loading state management
 *
 * Usage:
 * ```typescript
 * const { data, loading, error, execute } = createApiHandler(async () => {
 *   return await fetchData();
 * });
 *
 * await execute(); // Triggers the API call
 * ```
 */

export interface ApiState<T> {
	data: T | null;
	loading: boolean;
	error: Error | null;
}

export interface ApiHandler<T> {
	data: T | null;
	loading: boolean;
	error: Error | null;
	execute: () => Promise<void>;
	reset: () => void;
}

export function createApiState<T>(): ApiState<T> {
	return {
		data: null,
		loading: false,
		error: null
	};
}

/**
 * Wrapper for API calls with automatic error handling and toast notifications
 */
export async function handleApiCall<T>(
	apiCall: () => Promise<T>,
	options: {
		showErrorToast?: boolean;
		showSuccessToast?: boolean;
		successMessage?: string;
		errorMessage?: string;
		onSuccess?: (data: T) => void;
		onError?: (error: Error) => void;
	} = {}
): Promise<{ data: T | null; error: Error | null }> {
	const {
		showErrorToast = true,
		showSuccessToast = false,
		successMessage = 'Operation completed successfully',
		errorMessage,
		onSuccess,
		onError
	} = options;

	try {
		const data = await apiCall();

		if (showSuccessToast) {
			toast.success(successMessage);
		}

		if (onSuccess) {
			onSuccess(data);
		}

		return { data, error: null };
	} catch (err) {
		const error = err instanceof Error ? err : new Error('Unknown error occurred');

		if (showErrorToast) {
			toast.error(errorMessage || error.message || 'An error occurred');
		}

		if (onError) {
			onError(error);
		}

		return { data: null, error };
	}
}

/**
 * Parse API error response and extract error message
 */
export function parseApiError(error: unknown): string {
	if (error instanceof Error) {
		return error.message;
	}

	if (typeof error === 'string') {
		return error;
	}

	if (error && typeof error === 'object' && 'message' in error) {
		return String(error.message);
	}

	if (error && typeof error === 'object' && 'detail' in error) {
		return String(error.detail);
	}

	return 'An unexpected error occurred';
}

/**
 * Retry API call with exponential backoff
 */
export async function retryApiCall<T>(
	apiCall: () => Promise<T>,
	options: {
		maxRetries?: number;
		baseDelay?: number;
		maxDelay?: number;
	} = {}
): Promise<T> {
	const { maxRetries = 3, baseDelay = 1000, maxDelay = 10000 } = options;

	let lastError: Error | null = null;

	for (let attempt = 0; attempt < maxRetries; attempt++) {
		try {
			return await apiCall();
		} catch (err) {
			lastError = err instanceof Error ? err : new Error('Unknown error');

			if (attempt < maxRetries - 1) {
				// Calculate exponential backoff delay
				const delay = Math.min(baseDelay * Math.pow(2, attempt), maxDelay);
				await new Promise((resolve) => setTimeout(resolve, delay));
			}
		}
	}

	throw lastError || new Error('Max retries exceeded');
}
