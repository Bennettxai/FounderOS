/**
 * Template Insights Generator
 * Generates personalized insights during onboarding based on user data
 */

export interface InsightData {
	role: string;
	businessType: string;
	mainFocus?: string;
	workStyle?: string;
}

/**
 * Generates 3 personalized insights based on user's onboarding data
 * @param data User's role, business type, main focus, and work style
 * @returns Array of 3 insight strings
 */
export function generateInsights(data: InsightData): string[] {
	const { role, businessType, mainFocus, workStyle } = data;

	// Priority 1: Role + Business Type combinations
	if (role === 'founder' && businessType === 'agency') {
		return [
			'Running a growing agency',
			'Client delivery is your priority',
			'Always optimizing workflows'
		];
	}

	if (role === 'founder' && businessType === 'startup') {
		return [
			'Building the next big thing',
			'Speed and iteration matter',
			'Product-market fit is the goal'
		];
	}

	if (role === 'founder' && businessType === 'saas') {
		return [
			'Building a SaaS business',
			'Recurring revenue is key',
			'Customer success drives growth'
		];
	}

	if (role === 'founder' && businessType === 'ecommerce') {
		return [
			'Running an online store',
			'Customer experience matters',
			'Sales and inventory are key'
		];
	}

	if (role === 'founder' && businessType === 'consulting') {
		return [
			'Leading a consulting practice',
			'Expertise drives value',
			'Client relationships are everything'
		];
	}

	// Priority 2: Role-based (when business type doesn't match)
	if (role === 'freelancer') {
		return [
			'Solo professional life',
			'Flexibility is everything',
			'Building your personal brand'
		];
	}

	if (role === 'consultant') {
		return [
			'Independent consultant',
			'Deep expertise matters',
			'Client relationships first'
		];
	}

	if (role === 'employee') {
		return [
			'Part of a great team',
			'Collaboration is key',
			'Getting work done efficiently'
		];
	}

	// Priority 3: Main Focus (fallback)
	if (mainFocus?.includes('product')) {
		return [
			'Building something new',
			'Product-focused mindset',
			'Shipping is the goal'
		];
	}

	if (mainFocus?.includes('Client work')) {
		return [
			'Client work is central',
			'Delivering quality consistently',
			'Managing expectations well'
		];
	}

	if (mainFocus?.includes('Sales')) {
		return [
			'Sales drive your business',
			'Pipeline is everything',
			'Building relationships'
		];
	}

	if (mainFocus?.includes('Marketing')) {
		return [
			'Growth is your focus',
			'Creative campaigns matter',
			'Data drives decisions'
		];
	}

	if (mainFocus?.includes('Operations')) {
		return [
			'Efficiency is your game',
			'Process optimization daily',
			'Keeping everything running'
		];
	}

	if (mainFocus?.includes('Creative')) {
		return [
			'Creativity is your craft',
			'Design and aesthetics matter',
			'Building beautiful things'
		];
	}

	// Priority 4: Work Style (fallback)
	if (workStyle?.includes('Deep focus')) {
		return [
			'Deep work is your strength',
			'Focused and intentional',
			'Minimal distractions'
		];
	}

	if (workStyle?.includes('meetings')) {
		return [
			'Collaboration is key',
			'Meetings drive alignment',
			'Team communication matters'
		];
	}

	if (workStyle?.includes('Async')) {
		return [
			'Async communication master',
			'Deep work over meetings',
			'Focused productivity'
		];
	}

	// Default fallback (if nothing matches)
	return [
		'Getting things done',
		'Streamlined workflows',
		'Ready to build'
	];
}
