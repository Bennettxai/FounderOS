/**
 * Conditional Formatting Utilities for App Templates
 */

export type ComparisonOperator =
	| 'equals'
	| 'not_equals'
	| 'contains'
	| 'not_contains'
	| 'greater_than'
	| 'less_than'
	| 'greater_or_equal'
	| 'less_or_equal'
	| 'between'
	| 'is_empty'
	| 'is_not_empty';

export interface FormatCondition {
	fieldId: string;
	operator: ComparisonOperator;
	value?: unknown;
	value2?: unknown; // For 'between' operator
}

export interface FormatStyle {
	backgroundColor?: string;
	textColor?: string;
	fontWeight?: 'normal' | 'medium' | 'semibold' | 'bold';
	fontStyle?: 'normal' | 'italic';
	textDecoration?: 'none' | 'underline' | 'line-through';
	borderColor?: string;
	icon?: string;
	badge?: {
		text: string;
		color: string;
	};
}

export interface ConditionalFormat {
	id: string;
	name?: string;
	conditions: FormatCondition[];
	conditionOperator: 'and' | 'or';
	style: FormatStyle;
	priority: number;
	applyTo: 'row' | 'cell';
	targetFieldId?: string; // If applyTo is 'cell'
}

/**
 * Preset format styles
 */
export const formatPresets: Record<string, FormatStyle> = {
	success: {
		backgroundColor: 'var(--tpl-status-success-bg)',
		textColor: 'var(--tpl-status-success-text)'
	},
	warning: {
		backgroundColor: 'var(--tpl-status-warning-bg)',
		textColor: 'var(--tpl-status-warning-text)'
	},
	error: {
		backgroundColor: 'var(--tpl-status-error-bg)',
		textColor: 'var(--tpl-status-error-text)'
	},
	info: {
		backgroundColor: 'var(--tpl-status-info-bg)',
		textColor: 'var(--tpl-status-info-text)'
	},
	highlight: {
		backgroundColor: 'var(--tpl-accent-primary-light)',
		textColor: 'var(--tpl-accent-primary)'
	},
	muted: {
		textColor: 'var(--tpl-text-muted)',
		fontStyle: 'italic'
	},
	bold: {
		fontWeight: 'bold'
	},
	strikethrough: {
		textDecoration: 'line-through',
		textColor: 'var(--tpl-text-muted)'
	}
};

/**
 * Evaluate a single condition
 */
function evaluateCondition(condition: FormatCondition, record: Record<string, unknown>): boolean {
	const value = record[condition.fieldId];

	switch (condition.operator) {
		case 'equals':
			return value === condition.value;

		case 'not_equals':
			return value !== condition.value;

		case 'contains':
			return String(value || '').toLowerCase().includes(String(condition.value || '').toLowerCase());

		case 'not_contains':
			return !String(value || '').toLowerCase().includes(String(condition.value || '').toLowerCase());

		case 'greater_than':
			return Number(value) > Number(condition.value);

		case 'less_than':
			return Number(value) < Number(condition.value);

		case 'greater_or_equal':
			return Number(value) >= Number(condition.value);

		case 'less_or_equal':
			return Number(value) <= Number(condition.value);

		case 'between':
			const num = Number(value);
			return num >= Number(condition.value) && num <= Number(condition.value2);

		case 'is_empty':
			return value === null || value === undefined || value === '' || (Array.isArray(value) && value.length === 0);

		case 'is_not_empty':
			return value !== null && value !== undefined && value !== '' && !(Array.isArray(value) && value.length === 0);

		default:
			return false;
	}
}

/**
 * Evaluate all conditions for a format rule
 */
function evaluateConditions(
	format: ConditionalFormat,
	record: Record<string, unknown>
): boolean {
	if (format.conditions.length === 0) return false;

	if (format.conditionOperator === 'and') {
		return format.conditions.every(c => evaluateCondition(c, record));
	} else {
		return format.conditions.some(c => evaluateCondition(c, record));
	}
}

/**
 * Get applicable format styles for a row
 */
export function getRowStyles(
	record: Record<string, unknown>,
	formats: ConditionalFormat[]
): FormatStyle | null {
	// Filter to row formats, sort by priority, and find first matching
	const rowFormats = formats
		.filter(f => f.applyTo === 'row')
		.sort((a, b) => a.priority - b.priority);

	for (const format of rowFormats) {
		if (evaluateConditions(format, record)) {
			return format.style;
		}
	}

	return null;
}

/**
 * Get applicable format styles for a specific cell
 */
export function getCellStyles(
	record: Record<string, unknown>,
	fieldId: string,
	formats: ConditionalFormat[]
): FormatStyle | null {
	// Filter to cell formats for this field, sort by priority, and find first matching
	const cellFormats = formats
		.filter(f => f.applyTo === 'cell' && f.targetFieldId === fieldId)
		.sort((a, b) => a.priority - b.priority);

	for (const format of cellFormats) {
		if (evaluateConditions(format, record)) {
			return format.style;
		}
	}

	return null;
}

/**
 * Convert FormatStyle to CSS style object
 */
export function formatStyleToCSS(style: FormatStyle): Record<string, string> {
	const css: Record<string, string> = {};

	if (style.backgroundColor) {
		css.backgroundColor = style.backgroundColor;
	}
	if (style.textColor) {
		css.color = style.textColor;
	}
	if (style.fontWeight) {
		const weights = { normal: '400', medium: '500', semibold: '600', bold: '700' };
		css.fontWeight = weights[style.fontWeight];
	}
	if (style.fontStyle) {
		css.fontStyle = style.fontStyle;
	}
	if (style.textDecoration) {
		css.textDecoration = style.textDecoration;
	}
	if (style.borderColor) {
		css.borderColor = style.borderColor;
	}

	return css;
}

/**
 * Convert CSS style object to inline style string
 */
export function cssToStyleString(css: Record<string, string>): string {
	return Object.entries(css)
		.map(([key, value]) => `${key.replace(/([A-Z])/g, '-$1').toLowerCase()}: ${value}`)
		.join('; ');
}

/**
 * Create a conditional format builder helper
 */
export function createConditionalFormat(
	options: Partial<ConditionalFormat> & { conditions: FormatCondition[] }
): ConditionalFormat {
	return {
		id: Math.random().toString(36).substring(2, 9),
		name: options.name,
		conditions: options.conditions,
		conditionOperator: options.conditionOperator || 'and',
		style: options.style || {},
		priority: options.priority || 0,
		applyTo: options.applyTo || 'row',
		targetFieldId: options.targetFieldId
	};
}
