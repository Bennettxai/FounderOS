/**
 * Form utility functions and helpers
 * Provides reusable patterns for form handling, validation, and submission
 */

import type { FieldErrors, ValidationResult } from '$lib/types/forms';

/**
 * Validate a single field value against a pattern
 */
export function validateField(
	value: any,
	fieldName: string,
	rules: {
		required?: boolean;
		minLength?: number;
		maxLength?: number;
		min?: number;
		max?: number;
		pattern?: RegExp;
		custom?: (value: any) => string | null;
	}
): string | null {
	// Required check
	if (rules.required && !value) {
		return `${fieldName} is required`;
	}

	if (!value) return null; // Skip further validation if empty and not required

	// String length validation
	if (typeof value === 'string') {
		if (rules.minLength && value.length < rules.minLength) {
			return `${fieldName} must be at least ${rules.minLength} characters`;
		}
		if (rules.maxLength && value.length > rules.maxLength) {
			return `${fieldName} must not exceed ${rules.maxLength} characters`;
		}
		if (rules.pattern && !rules.pattern.test(value)) {
			return `${fieldName} format is invalid`;
		}
	}

	// Number validation
	if (typeof value === 'number') {
		if (rules.min !== undefined && value < rules.min) {
			return `${fieldName} must be at least ${rules.min}`;
		}
		if (rules.max !== undefined && value > rules.max) {
			return `${fieldName} must not exceed ${rules.max}`;
		}
	}

	// Custom validation
	if (rules.custom) {
		return rules.custom(value);
	}

	return null;
}

/**
 * Validate all form fields against rules
 */
export function validateFields(
	data: Record<string, any>,
	rules: Record<string, any>
): ValidationResult {
	const errors: FieldErrors = {};

	for (const [fieldName, fieldRules] of Object.entries(rules)) {
		const value = data[fieldName];
		const error = validateField(value, fieldName, fieldRules);
		if (error) {
			errors[fieldName] = error;
		}
	}

	return {
		valid: Object.keys(errors).length === 0,
		errors
	};
}

/**
 * Sanitize user input to prevent common security issues
 */
export function sanitizeInput(input: string): string {
	return input
		.trim()
		.replace(/[<>]/g, '') // Remove angle brackets
		.substring(0, 1000); // Limit length
}

/**
 * Format form data for API submission
 */
export function formatFormData<T extends Record<string, any>>(data: T): T {
	const formatted = { ...data } as Record<string, any>;

	for (const [key, value] of Object.entries(formatted)) {
		if (typeof value === 'string') {
			formatted[key] = sanitizeInput(value);
		}
	}

	return formatted as T;
}

/**
 * Toggle item in array (for multi-select fields)
 */
export function toggleArrayItem<T>(array: T[], item: T): T[] {
	if (array.includes(item)) {
		return array.filter(x => x !== item);
	}
	return [...array, item];
}

/**
 * Reset form to initial state
 */
export function resetForm<T extends Record<string, any>>(
	initial: T
): T {
	const reset: Record<string, any> = {};

	for (const [key, value] of Object.entries(initial)) {
		if (typeof value === 'string') {
			reset[key] = '';
		} else if (typeof value === 'number') {
			reset[key] = 0;
		} else if (typeof value === 'boolean') {
			reset[key] = false;
		} else if (Array.isArray(value)) {
			reset[key] = [];
		} else if (value === null || value === undefined) {
			reset[key] = null;
		} else {
			reset[key] = value;
		}
	}

	return reset as T;
}

/**
 * Compare two form objects to detect changes
 */
export function hasFormChanged<T extends Record<string, any>>(
	original: T,
	current: T
): boolean {
	for (const key of Object.keys(original)) {
		if (JSON.stringify(original[key]) !== JSON.stringify(current[key])) {
			return true;
		}
	}
	return false;
}

/**
 * Build form data object from FormData
 */
export function parseFormData(formData: FormData): Record<string, any> {
	const data: Record<string, any> = {};

	for (const [key, value] of formData.entries()) {
		if (key.endsWith('[]')) {
			// Handle arrays
			const arrayKey = key.slice(0, -2);
			if (!Array.isArray(data[arrayKey])) {
				data[arrayKey] = [];
			}
			data[arrayKey].push(value);
		} else {
			data[key] = value;
		}
	}

	return data;
}

/**
 * Create async form handler with common pattern
 */
export function createFormHandler<T extends Record<string, any>>(
	apiCall: (data: T) => Promise<Response>,
	options?: {
		onSuccess?: (response: any) => void;
		onError?: (error: string) => void;
		transformData?: (data: T) => T;
	}
) {
	return async (data: T) => {
		try {
			// Transform data if needed
			const finalData = options?.transformData ? options.transformData(data) : data;

			// Make API call
			const response = await apiCall(finalData);

			if (!response.ok) {
				const errorData = await response.json();
				const errorMessage = errorData.error || `HTTP ${response.status}`;
				throw new Error(errorMessage);
			}

			// Success callback
			if (options?.onSuccess) {
				const responseData = await response.json();
				options.onSuccess(responseData);
			}
		} catch (err) {
			// Error callback
			if (options?.onError) {
				const message = err instanceof Error ? err.message : 'An error occurred';
				options.onError(message);
			}
			throw err;
		}
	};
}

/**
 * Debounce form auto-save
 */
export function createAutoSaveDebounce<T extends Record<string, any>>(
	saveFunction: (data: T) => Promise<void>,
	delayMs: number = 1000
) {
	let timeout: NodeJS.Timeout;

	return (data: T) => {
		clearTimeout(timeout);
		timeout = setTimeout(() => {
			saveFunction(data).catch(err => console.error('Auto-save failed:', err));
		}, delayMs);
	};
}

/**
 * Create form submission handler with loading state management
 */
export function createFormSubmitHandler<T extends Record<string, any>>(options: {
	onValidate?: (data: T) => string | null; // Return error message or null
	onSubmit: (data: T) => Promise<void>;
	onSuccess?: (data: T) => void;
	onError?: (error: string) => void;
}) {
	return async (data: T) => {
		// Validate
		if (options.onValidate) {
			const validationError = options.onValidate(data);
			if (validationError) {
				options.onError?.(validationError);
				throw new Error(validationError);
			}
		}

		try {
			// Submit
			await options.onSubmit(data);

			// Success
			options.onSuccess?.(data);
		} catch (err) {
			const message = err instanceof Error ? err.message : 'An error occurred';
			options.onError?.(message);
			throw err;
		}
	};
}

/**
 * Regex patterns for common validations
 */
export const ValidationPatterns = {
	email: /^[^\s@]+@[^\s@]+\.[^\s@]+$/,
	url: /^https?:\/\/.+/,
	slug: /^[a-z0-9]+(?:-[a-z0-9]+)*$/,
	alphanumeric: /^[a-zA-Z0-9]+$/,
	phone: /^\d{10,}$/,
	password: /^(?=.*[a-z])(?=.*[A-Z])(?=.*\d).{8,}$/, // At least 8 chars, uppercase, lowercase, number
};

/**
 * Pre-built validation rules for common fields
 */
export const ValidationRules = {
	email: {
		required: true,
		pattern: ValidationPatterns.email
	},
	password: {
		required: true,
		minLength: 8,
		pattern: ValidationPatterns.password
	},
	agentName: {
		required: true,
		minLength: 1,
		maxLength: 100,
		pattern: /^[a-z0-9-]+$/ // lowercase, numbers, hyphens only
	},
	systemPrompt: {
		required: true,
		minLength: 10,
		maxLength: 5000
	},
	url: {
		pattern: ValidationPatterns.url
	}
};
