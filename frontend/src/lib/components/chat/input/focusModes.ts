export interface FocusModeChoice {
	value: string;
	label: string;
	tooltip?: string;
}

export interface FocusModeOption {
	id: string;
	label: string;
	type: 'segment' | 'toggle' | 'select';
	choices?: FocusModeChoice[];
	default: string;
}

export interface FocusMode {
	id: string;
	name: string;
	icon: string;
	agent: 'analysis' | 'document' | 'planning' | 'orchestrator' | 'osa';
	options: FocusModeOption[];
}

export const FOCUS_MODES: FocusMode[] = [
	{
		id: 'deep',
		name: 'Deep Research',
		icon: 'globe-alt',
		agent: 'analysis',
		options: [
			{
				id: 'searchScope',
				label: 'Search scope',
				type: 'segment',
				choices: [
					{ value: 'web', label: 'Web', tooltip: 'Search the internet' },
					{ value: 'docs', label: 'Docs', tooltip: 'Search your documents' },
					{ value: 'all', label: 'All', tooltip: 'Search everything' }
				],
				default: 'web'
			},
			{
				id: 'depth',
				label: 'Depth',
				type: 'segment',
				choices: [
					{ value: 'quick', label: 'Quick', tooltip: '3 sources' },
					{ value: 'standard', label: 'Standard', tooltip: '5 sources' },
					{ value: 'deep', label: 'Deep', tooltip: '10+ sources' }
				],
				default: 'deep'
			},
			{
				id: 'output',
				label: 'Output',
				type: 'segment',
				choices: [
					{ value: 'summary', label: 'Summary', tooltip: 'Concise overview' },
					{ value: 'report', label: 'Report', tooltip: 'Detailed report with sources' }
				],
				default: 'report'
			}
		]
	},
	{
		id: 'research',
		name: 'Research',
		icon: 'magnifying-glass-chart',
		agent: 'analysis',
		options: [
			{
				id: 'searchScope',
				label: 'Search scope',
				type: 'segment',
				choices: [
					{ value: 'web', label: 'Web' },
					{ value: 'docs', label: 'Docs' },
					{ value: 'all', label: 'All' }
				],
				default: 'all'
			},
			{
				id: 'depth',
				label: 'Depth',
				type: 'segment',
				choices: [
					{ value: 'quick', label: 'Quick', tooltip: 'Fast overview' },
					{ value: 'thorough', label: 'Thorough', tooltip: 'Deep analysis' }
				],
				default: 'thorough'
			},
			{
				id: 'output',
				label: 'Output',
				type: 'segment',
				choices: [
					{ value: 'summary', label: 'Summary' },
					{ value: 'report', label: 'Report' }
				],
				default: 'summary'
			}
		]
	},
	{
		id: 'analyze',
		name: 'Analyze',
		icon: 'chart-bar',
		agent: 'analysis',
		options: [
			{
				id: 'approach',
				label: 'Approach',
				type: 'segment',
				choices: [
					{ value: 'validate', label: 'Validate', tooltip: 'Test significance' },
					{ value: 'compare', label: 'Compare' },
					{ value: 'forecast', label: 'Forecast' }
				],
				default: 'validate'
			},
			{
				id: 'depth',
				label: 'Depth',
				type: 'segment',
				choices: [
					{ value: 'quick', label: 'Quick' },
					{ value: 'thorough', label: 'Thorough' }
				],
				default: 'thorough'
			},
			{
				id: 'output',
				label: 'Output',
				type: 'segment',
				choices: [
					{ value: 'findings', label: 'Findings' },
					{ value: 'dashboard', label: 'Dashboard' }
				],
				default: 'findings'
			}
		]
	},
	{
		id: 'write',
		name: 'Write',
		icon: 'document-text',
		agent: 'document',
		options: [
			{
				id: 'format',
				label: 'Format',
				type: 'segment',
				choices: [
					{ value: 'doc', label: 'Doc' },
					{ value: 'slides', label: 'Slides' },
					{ value: 'spreadsheet', label: 'Spreadsheet' }
				],
				default: 'doc'
			},
			{
				id: 'writingMode',
				label: 'Writing mode',
				type: 'segment',
				choices: [
					{ value: 'stepByStep', label: 'Step by step' },
					{ value: 'firstDraft', label: 'First draft', tooltip: 'Get a first draft then ask for edits' }
				],
				default: 'firstDraft'
			},
			{
				id: 'citations',
				label: 'Citations',
				type: 'toggle',
				default: 'off'
			}
		]
	},
	{
		id: 'build',
		name: 'Build',
		icon: 'cube',
		agent: 'planning',
		options: [
			{
				id: 'artifactType',
				label: 'Create',
				type: 'segment',
				choices: [
					{ value: 'framework', label: 'Framework', tooltip: 'Strategic framework' },
					{ value: 'sop', label: 'SOP', tooltip: 'Standard Operating Procedure' },
					{ value: 'plan', label: 'Plan', tooltip: 'Project or action plan' }
				],
				default: 'framework'
			},
			{
				id: 'detail',
				label: 'Detail',
				type: 'segment',
				choices: [
					{ value: 'outline', label: 'Outline', tooltip: 'High-level structure' },
					{ value: 'detailed', label: 'Detailed', tooltip: 'Comprehensive content' }
				],
				default: 'detailed'
			}
		]
	},
	{
		id: 'general',
		name: 'Do more',
		icon: 'plus',
		agent: 'orchestrator',
		options: [
			{
				id: 'mode',
				label: 'Mode',
				type: 'segment',
				choices: [
					{ value: 'chat', label: 'Chat', tooltip: 'General conversation' },
					{ value: 'learn', label: 'Learn', tooltip: 'Educational mode' },
					{ value: 'brainstorm', label: 'Brainstorm', tooltip: 'Creative ideation' }
				],
				default: 'chat'
			}
		]
	},
	{
		id: 'generate',
		name: 'Generate App',
		icon: 'code-bracket',
		agent: 'osa',
		options: [
			{
				id: 'appType',
				label: 'App type',
				type: 'segment',
				choices: [
					{ value: 'web', label: 'Web', tooltip: 'Web application' },
					{ value: 'api', label: 'API', tooltip: 'REST/GraphQL API' },
					{ value: 'fullstack', label: 'Full-stack', tooltip: 'Complete application' },
					{ value: 'mobile', label: 'Mobile', tooltip: 'Mobile app' }
				],
				default: 'web'
			},
			{
				id: 'complexity',
				label: 'Complexity',
				type: 'segment',
				choices: [
					{ value: 'simple', label: 'Simple', tooltip: 'Basic CRUD app' },
					{ value: 'standard', label: 'Standard', tooltip: 'Multi-feature app' },
					{ value: 'advanced', label: 'Advanced', tooltip: 'Complex system' }
				],
				default: 'standard'
			},
			{
				id: 'deployment',
				label: 'Auto-deploy',
				type: 'toggle',
				default: 'off'
			}
		]
	}
];

// Helper to get default options for a focus mode
export function getDefaultOptions(mode: FocusMode): Record<string, string> {
	const defaults: Record<string, string> = {};
	for (const option of mode.options) {
		defaults[option.id] = option.default;
	}
	return defaults;
}

// Helper to get agent type for focus mode
export function getAgentForFocusMode(focusModeId: string): string {
	const mode = FOCUS_MODES.find(m => m.id === focusModeId);
	return mode?.agent || 'orchestrator';
}
