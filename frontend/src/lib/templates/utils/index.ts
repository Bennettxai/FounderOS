/**
 * Template Utilities - Central Export
 */

// Keyboard Shortcuts
export {
	createKeyboardHandler,
	formatShortcut,
	useKeyboardShortcuts,
	defaultViewShortcuts,
	type KeyboardShortcut,
	type KeyboardShortcutGroup
} from './keyboard';

// Validation
export {
	validateValue,
	validateRecord,
	isRecordValid,
	getFieldValidationRules,
	createValidationSchema,
	type ValidationRule,
	type ValidationResult,
	type FieldValidation
} from './validation';

// Conditional Formatting
export {
	getRowStyles,
	getCellStyles,
	formatStyleToCSS,
	cssToStyleString,
	createConditionalFormat,
	formatPresets,
	type ComparisonOperator,
	type FormatCondition,
	type FormatStyle,
	type ConditionalFormat
} from './conditional-formatting';

// Formula Engine
export {
	evaluateFormula,
	validateFormula,
	getFormulaFieldReferences,
	type FormulaFunction,
	type FormulaResult
} from './formula';
