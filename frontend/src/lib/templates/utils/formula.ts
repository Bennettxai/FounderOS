/**
 * Formula Engine for App Templates
 * Supports basic formulas for computed fields
 */

export type FormulaFunction =
	| 'SUM'
	| 'AVERAGE'
	| 'COUNT'
	| 'MIN'
	| 'MAX'
	| 'IF'
	| 'CONCAT'
	| 'UPPER'
	| 'LOWER'
	| 'TRIM'
	| 'LEN'
	| 'ROUND'
	| 'FLOOR'
	| 'CEIL'
	| 'ABS'
	| 'NOW'
	| 'TODAY'
	| 'YEAR'
	| 'MONTH'
	| 'DAY'
	| 'DAYS_BETWEEN'
	| 'FORMAT_DATE'
	| 'FORMAT_CURRENCY'
	| 'COALESCE';

export interface FormulaToken {
	type: 'function' | 'field' | 'number' | 'string' | 'operator' | 'paren';
	value: string | number;
}

export interface FormulaResult {
	value: unknown;
	error?: string;
}

/**
 * Parse a formula string into tokens
 */
function tokenize(formula: string): FormulaToken[] {
	const tokens: FormulaToken[] = [];
	let i = 0;

	while (i < formula.length) {
		const char = formula[i];

		// Skip whitespace
		if (/\s/.test(char)) {
			i++;
			continue;
		}

		// Numbers
		if (/\d/.test(char) || (char === '.' && /\d/.test(formula[i + 1]))) {
			let num = '';
			while (i < formula.length && (/\d/.test(formula[i]) || formula[i] === '.')) {
				num += formula[i];
				i++;
			}
			tokens.push({ type: 'number', value: parseFloat(num) });
			continue;
		}

		// Strings (double quotes)
		if (char === '"') {
			let str = '';
			i++; // Skip opening quote
			while (i < formula.length && formula[i] !== '"') {
				if (formula[i] === '\\' && formula[i + 1] === '"') {
					str += '"';
					i += 2;
				} else {
					str += formula[i];
					i++;
				}
			}
			i++; // Skip closing quote
			tokens.push({ type: 'string', value: str });
			continue;
		}

		// Field references (curly braces)
		if (char === '{') {
			let field = '';
			i++; // Skip opening brace
			while (i < formula.length && formula[i] !== '}') {
				field += formula[i];
				i++;
			}
			i++; // Skip closing brace
			tokens.push({ type: 'field', value: field });
			continue;
		}

		// Parentheses
		if (char === '(' || char === ')') {
			tokens.push({ type: 'paren', value: char });
			i++;
			continue;
		}

		// Operators
		if (['+', '-', '*', '/', '%', '>', '<', '=', '!', '&', '|', ','].includes(char)) {
			let op = char;
			// Check for multi-char operators
			if (i + 1 < formula.length) {
				const next = formula[i + 1];
				if ((char === '>' || char === '<' || char === '!' || char === '=') && next === '=') {
					op += next;
					i++;
				} else if ((char === '&' && next === '&') || (char === '|' && next === '|')) {
					op += next;
					i++;
				}
			}
			tokens.push({ type: 'operator', value: op });
			i++;
			continue;
		}

		// Function names / identifiers
		if (/[a-zA-Z_]/.test(char)) {
			let name = '';
			while (i < formula.length && /[a-zA-Z0-9_]/.test(formula[i])) {
				name += formula[i];
				i++;
			}
			tokens.push({ type: 'function', value: name.toUpperCase() });
			continue;
		}

		// Unknown character - skip
		i++;
	}

	return tokens;
}

/**
 * Evaluate a formula with a given record context
 */
export function evaluateFormula(
	formula: string,
	record: Record<string, unknown>,
	allRecords?: Record<string, unknown>[]
): FormulaResult {
	try {
		const tokens = tokenize(formula);
		const result = evaluate(tokens, record, allRecords || [record]);
		return { value: result };
	} catch (error) {
		return {
			value: null,
			error: error instanceof Error ? error.message : 'Unknown error'
		};
	}
}

/**
 * Recursive token evaluator
 */
function evaluate(
	tokens: FormulaToken[],
	record: Record<string, unknown>,
	allRecords: Record<string, unknown>[]
): unknown {
	if (tokens.length === 0) return null;

	// Simple single token
	if (tokens.length === 1) {
		const token = tokens[0];
		if (token.type === 'number' || token.type === 'string') {
			return token.value;
		}
		if (token.type === 'field') {
			return record[token.value as string];
		}
	}

	// Check for function call
	if (tokens[0].type === 'function' && tokens[1]?.type === 'paren' && tokens[1].value === '(') {
		const funcName = tokens[0].value as string;
		const args = extractFunctionArgs(tokens.slice(2), record, allRecords);
		return executeFunction(funcName, args, record, allRecords);
	}

	// Handle basic arithmetic expressions
	return evaluateExpression(tokens, record, allRecords);
}

/**
 * Extract function arguments from tokens
 */
function extractFunctionArgs(
	tokens: FormulaToken[],
	record: Record<string, unknown>,
	allRecords: Record<string, unknown>[]
): unknown[] {
	const args: unknown[] = [];
	let depth = 1;
	let currentArg: FormulaToken[] = [];

	for (let i = 0; i < tokens.length; i++) {
		const token = tokens[i];

		if (token.type === 'paren') {
			if (token.value === '(') {
				depth++;
				currentArg.push(token);
			} else if (token.value === ')') {
				depth--;
				if (depth === 0) {
					if (currentArg.length > 0) {
						args.push(evaluate(currentArg, record, allRecords));
					}
					break;
				}
				currentArg.push(token);
			}
		} else if (token.type === 'operator' && token.value === ',' && depth === 1) {
			args.push(evaluate(currentArg, record, allRecords));
			currentArg = [];
		} else {
			currentArg.push(token);
		}
	}

	return args;
}

/**
 * Execute a formula function
 */
function executeFunction(
	name: string,
	args: unknown[],
	record: Record<string, unknown>,
	allRecords: Record<string, unknown>[]
): unknown {
	switch (name) {
		case 'SUM':
			return args.reduce((sum: number, val) => sum + Number(val || 0), 0);

		case 'AVERAGE':
			if (args.length === 0) return 0;
			const total = args.reduce((sum: number, val) => sum + Number(val || 0), 0);
			return total / args.length;

		case 'COUNT':
			return args.filter(v => v !== null && v !== undefined && v !== '').length;

		case 'MIN':
			const nums = args.map(v => Number(v)).filter(n => !isNaN(n));
			return nums.length > 0 ? Math.min(...nums) : null;

		case 'MAX':
			const maxNums = args.map(v => Number(v)).filter(n => !isNaN(n));
			return maxNums.length > 0 ? Math.max(...maxNums) : null;

		case 'IF':
			return args[0] ? args[1] : args[2];

		case 'CONCAT':
			return args.map(v => String(v ?? '')).join('');

		case 'UPPER':
			return String(args[0] ?? '').toUpperCase();

		case 'LOWER':
			return String(args[0] ?? '').toLowerCase();

		case 'TRIM':
			return String(args[0] ?? '').trim();

		case 'LEN':
			return String(args[0] ?? '').length;

		case 'ROUND':
			const decimals = args[1] !== undefined ? Number(args[1]) : 0;
			return Number(Number(args[0]).toFixed(decimals));

		case 'FLOOR':
			return Math.floor(Number(args[0]));

		case 'CEIL':
			return Math.ceil(Number(args[0]));

		case 'ABS':
			return Math.abs(Number(args[0]));

		case 'NOW':
			return new Date().toISOString();

		case 'TODAY':
			return new Date().toISOString().split('T')[0];

		case 'YEAR':
			return new Date(args[0] as string).getFullYear();

		case 'MONTH':
			return new Date(args[0] as string).getMonth() + 1;

		case 'DAY':
			return new Date(args[0] as string).getDate();

		case 'DAYS_BETWEEN':
			const d1 = new Date(args[0] as string);
			const d2 = new Date(args[1] as string);
			return Math.floor((d2.getTime() - d1.getTime()) / (1000 * 60 * 60 * 24));

		case 'FORMAT_DATE':
			const date = new Date(args[0] as string);
			const format = String(args[1] || 'short');
			return date.toLocaleDateString('en-US', { dateStyle: format as 'short' | 'medium' | 'long' });

		case 'FORMAT_CURRENCY':
			const amount = Number(args[0]);
			const currency = String(args[1] || 'USD');
			return new Intl.NumberFormat('en-US', { style: 'currency', currency }).format(amount);

		case 'COALESCE':
			return args.find(v => v !== null && v !== undefined && v !== '') ?? null;

		default:
			throw new Error(`Unknown function: ${name}`);
	}
}

/**
 * Evaluate basic arithmetic expression
 */
function evaluateExpression(
	tokens: FormulaToken[],
	record: Record<string, unknown>,
	allRecords: Record<string, unknown>[]
): unknown {
	// Convert tokens to values and operators
	const values: unknown[] = [];
	const operators: string[] = [];

	for (let i = 0; i < tokens.length; i++) {
		const token = tokens[i];

		if (token.type === 'number' || token.type === 'string') {
			values.push(token.value);
		} else if (token.type === 'field') {
			values.push(record[token.value as string]);
		} else if (token.type === 'operator' && token.value !== ',') {
			operators.push(token.value as string);
		}
	}

	// Simple left-to-right evaluation (no operator precedence for now)
	if (values.length === 0) return null;

	let result = values[0];
	for (let i = 0; i < operators.length && i + 1 < values.length; i++) {
		const op = operators[i];
		const right = values[i + 1];

		switch (op) {
			case '+':
				result = Number(result) + Number(right);
				break;
			case '-':
				result = Number(result) - Number(right);
				break;
			case '*':
				result = Number(result) * Number(right);
				break;
			case '/':
				result = Number(right) !== 0 ? Number(result) / Number(right) : null;
				break;
			case '%':
				result = Number(result) % Number(right);
				break;
			case '>':
				result = Number(result) > Number(right);
				break;
			case '<':
				result = Number(result) < Number(right);
				break;
			case '>=':
				result = Number(result) >= Number(right);
				break;
			case '<=':
				result = Number(result) <= Number(right);
				break;
			case '==':
			case '=':
				result = result === right;
				break;
			case '!=':
				result = result !== right;
				break;
			case '&&':
				result = Boolean(result) && Boolean(right);
				break;
			case '||':
				result = Boolean(result) || Boolean(right);
				break;
		}
	}

	return result;
}

/**
 * Validate a formula string
 */
export function validateFormula(formula: string): { valid: boolean; error?: string } {
	try {
		tokenize(formula);
		return { valid: true };
	} catch (error) {
		return {
			valid: false,
			error: error instanceof Error ? error.message : 'Invalid formula'
		};
	}
}

/**
 * Get field references from a formula
 */
export function getFormulaFieldReferences(formula: string): string[] {
	const tokens = tokenize(formula);
	return tokens
		.filter(t => t.type === 'field')
		.map(t => t.value as string);
}
