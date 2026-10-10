/**
 * Maps file extensions to Monaco language IDs
 */
const EXTENSION_MAP: Record<string, string> = {
	// Web
	'.js': 'javascript',
	'.jsx': 'javascript',
	'.mjs': 'javascript',
	'.cjs': 'javascript',
	'.ts': 'typescript',
	'.tsx': 'typescript',
	'.svelte': 'html',
	'.vue': 'html',
	'.html': 'html',
	'.htm': 'html',
	'.css': 'css',
	'.scss': 'scss',
	'.less': 'less',
	'.json': 'json',
	'.jsonc': 'json',

	// Backend
	'.go': 'go',
	'.py': 'python',
	'.rb': 'ruby',
	'.rs': 'rust',
	'.java': 'java',
	'.php': 'php',
	'.cs': 'csharp',

	// Data/Config
	'.yaml': 'yaml',
	'.yml': 'yaml',
	'.toml': 'ini',
	'.ini': 'ini',
	'.env': 'plaintext',
	'.xml': 'xml',
	'.graphql': 'graphql',
	'.gql': 'graphql',

	// Database
	'.sql': 'sql',

	// Shell
	'.sh': 'shell',
	'.bash': 'shell',
	'.zsh': 'shell',
	'.fish': 'shell',
	'.bat': 'bat',
	'.cmd': 'bat',
	'.ps1': 'powershell',

	// Docs
	'.md': 'markdown',
	'.mdx': 'markdown',

	// DevOps
	'.dockerfile': 'dockerfile',
	'.tf': 'hcl',
};

/**
 * Filename-based detection (no extension)
 */
const FILENAME_MAP: Record<string, string> = {
	'Dockerfile': 'dockerfile',
	'Makefile': 'plaintext',
	'Rakefile': 'ruby',
	'Gemfile': 'ruby',
	'.gitignore': 'plaintext',
	'.dockerignore': 'plaintext',
	'.editorconfig': 'ini',
	'.prettierrc': 'json',
	'.eslintrc': 'json',
	'tsconfig.json': 'json',
	'package.json': 'json',
	'go.mod': 'go',
	'go.sum': 'plaintext',
};

/**
 * Detect Monaco language ID from a file path
 */
export function detectLanguage(filepath: string): string {
	const filename = filepath.split('/').pop() || filepath;

	// Check exact filename match first
	if (FILENAME_MAP[filename]) {
		return FILENAME_MAP[filename];
	}

	// Check extension
	const dotIndex = filename.lastIndexOf('.');
	if (dotIndex !== -1) {
		const ext = filename.slice(dotIndex).toLowerCase();
		if (EXTENSION_MAP[ext]) {
			return EXTENSION_MAP[ext];
		}
	}

	return 'plaintext';
}

/**
 * Get a display label for the language
 */
const LANGUAGE_LABELS: Record<string, string> = {
	javascript: 'JavaScript',
	typescript: 'TypeScript',
	html: 'HTML',
	css: 'CSS',
	scss: 'SCSS',
	less: 'LESS',
	json: 'JSON',
	go: 'Go',
	python: 'Python',
	ruby: 'Ruby',
	rust: 'Rust',
	java: 'Java',
	php: 'PHP',
	csharp: 'C#',
	yaml: 'YAML',
	xml: 'XML',
	sql: 'SQL',
	shell: 'Shell',
	bat: 'Batch',
	powershell: 'PowerShell',
	markdown: 'Markdown',
	dockerfile: 'Dockerfile',
	graphql: 'GraphQL',
	ini: 'INI',
	hcl: 'HCL',
	plaintext: 'Plain Text',
};

export function getLanguageLabel(languageId: string): string {
	return LANGUAGE_LABELS[languageId] || languageId;
}

/**
 * Get a file icon color based on language
 */
export function getLanguageColor(languageId: string): string {
	const colors: Record<string, string> = {
		javascript: '#fbbf24',
		typescript: '#60a5fa',
		html: '#f87171',
		css: '#a78bfa',
		scss: '#ec4899',
		json: '#fbbf24',
		go: '#22d3ee',
		python: '#34d399',
		ruby: '#f87171',
		rust: '#fb923c',
		sql: '#6366f1',
		shell: '#34d399',
		markdown: '#a1a1aa',
		dockerfile: '#60a5fa',
		yaml: '#f87171',
	};
	return colors[languageId] || '#a1a1aa';
}
