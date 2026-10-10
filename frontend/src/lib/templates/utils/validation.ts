/**
 * Field Validation Utilities for App Templates
 */

import type { Field } from '../types/field';

export interface ValidationRule {
	type: 'required' | 'min' | 'max' | 'minLength' | 'maxLength' | 'pattern' | 'email' | 'url' | 'custom';
	value?: unknown;
	message?: string;
}

export interface ValidationResult {
	valid: boolean;
	errors: string[];
}

export interface FieldValidation {
	fieldId: string;
	rules: ValidationRule[];
}

/**
 * Validate a single value against rules
 */
export function validateValue(value: unknown, rules: ValidationRule[], field?: Field): ValidationResult {
	const errors: string[] = [];

	for (const rule of rules) {
		const error = validateRule(value, rule, field);
		if (error) {
			errors.push(error);
		}
	}

	return {
		valid: errors.length === 0,
		errors
	};
}

/**
 * Validate a single rule
 */
function validateRule(value: unknown, rule: ValidationRule, field?: Field): string | null {
	const fieldLabel = getFieldLabel(field);

	switch (rule.type) {
		case 'required': {
			if (value === null || value === undefined || value === '' || (Array.isArray(value) && value.length === 0)) {
				return rule.message || `${fieldLabel} is required`;
			}
			break;
		}

		case 'min': {
			const numValue = Number(value);
			const minValue = Number(rule.value);
			if (!isNaN(numValue) && numValue < minValue) {
				return rule.message || `${fieldLabel} must be at least ${minValue}`;
			}
			break;
		}

		case 'max': {
			const numValue = Number(value);
			const maxValue = Number(rule.value);
			if (!isNaN(numValue) && numValue > maxValue) {
				return rule.message || `${fieldLabel} must be at most ${maxValue}`;
			}
			break;
		}

		case 'minLength': {
			const strValue = String(value || '');
			const minLen = Number(rule.value);
			if (strValue.length < minLen) {
				return rule.message || `${fieldLabel} must be at least ${minLen} characters`;
			}
			break;
		}

		case 'maxLength': {
			const strValue = String(value || '');
			const maxLen = Number(rule.value);
			if (strValue.length > maxLen) {
				return rule.message || `${fieldLabel} must be at most ${maxLen} characters`;
			}
			break;
		}

		case 'pattern': {
			const strValue = String(value || '');
			const pattern = rule.value instanceof RegExp ? rule.value : new RegExp(String(rule.value));
			if (strValue && !pattern.test(strValue)) {
				return rule.message || `${fieldLabel} has an invalid format`;
			}
			break;
		}

		case 'email': {
			const strValue = String(value || '');
			const emailPattern = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
			if (strValue && !emailPattern.test(strValue)) {
				return rule.message || `${fieldLabel} must be a valid email address`;
			}
			break;
		}

		case 'url': {
			const strValue = String(value || '');
			try {
				if (strValue) {
					new URL(strValue);
				}
			} catch {
				return rule.message || `${fieldLabel} must be a valid URL`;
			}
			break;
		}

		case 'custom': {
			if (typeof rule.value === 'function') {
				const customResult = rule.value(value);
				if (customResult !== true) {
					return rule.message || customResult || `${fieldLabel} is invalid`;
				}
			}
			break;
		}
	}

	return null;
}

/**
 * Validate an entire record
 */
export function validateRecord(
	record: Record<string, unknown>,
	validations: FieldValidation[],
	fields: Field[]
): Record<string, ValidationResult> {
	const results: Record<string, ValidationResult> = {};

	for (const validation of validations) {
		const field = fields.find(f => f.id === validation.fieldId);
		const value = record[validation.fieldId];
		results[validation.fieldId] = validateValue(value, validation.rules, field);
	}

	return results;
}

/**
 * Check if a record is valid
 */
export function isRecordValid(validationResults: Record<string, ValidationResult>): boolean {
	return Object.values(validationResults).every(result => result.valid);
}

/**
 * Get validation rules from field configuration
 */
export function getFieldValidationRules(field: Field): ValidationRule[] {
	const rules: ValidationRule[] = [];

	// Add rules based on field config
	if (field.required) {
		rules.push({ type: 'required' });
	}

	// Type-specific rules
	switch (field.type) {
		case 'email':
			rules.push({ type: 'email' });
			break;
		case 'url':
			rules.push({ type: 'url' });
			break;
		case 'number':
		case 'currency':
			if (getFieldConfig(field).min !== undefined) {
				rules.push({ type: 'min', value: getFieldConfig(field).min });
			}
			if (getFieldConfig(field).max !== undefined) {
				rules.push({ type: 'max', value: getFieldConfig(field).max });
			}
			break;
		case 'text':
			if (getFieldConfig(field).minLength !== undefined) {
				rules.push({ type: 'minLength', value: getFieldConfig(field).minLength });
			}
			if (getFieldConfig(field).maxLength !== undefined) {
				rules.push({ type: 'maxLength', value: getFieldConfig(field).maxLength });
			}
			if (getFieldConfig(field).pattern) {
				rules.push({ type: 'pattern', value: getFieldConfig(field).pattern });
			}
			break;
		case 'rating':
			rules.push({ type: 'min', value: 0 });
			rules.push({ type: 'max', value: getFieldConfig(field).max || 5 });
			break;
		case 'progress':
			rules.push({ type: 'min', value: 0 });
			rules.push({ type: 'max', value: 100 });
			break;
	}

	return rules;
}

/**
 * Create validation schema from fields
 */
export function createValidationSchema(fields: Field[]): FieldValidation[] {
	return fields
		.map(field => ({
			fieldId: field.id,
			rules: getFieldValidationRules(field)
		}))
		.filter(v => v.rules.length > 0);
}

function getFieldLabel(field?: Field): string {
	if (!field) return 'Field';
	return field.label ?? field.name ?? field.id;
}

function getFieldConfig(field: Field): {
	min?: number;
	max?: number;
	minLength?: number;
	maxLength?: number;
	pattern?: string | RegExp;
} {
	return field.config ?? {};
}
