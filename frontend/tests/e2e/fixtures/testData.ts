/**
 * Test Data Fixtures
 *
 * Seed data for E2E tests including workspaces, apps, and templates.
 */

export interface TestWorkspace {
	id?: string;
	name: string;
	description: string;
}

export interface TestApp {
	id?: string;
	name: string;
	description: string;
	status: 'pending' | 'building' | 'completed' | 'failed';
}

export interface TestTemplate {
	id?: string;
	name: string;
	description: string;
	category: string;
}

/**
 * Test Workspaces
 */
export const testWorkspaces: TestWorkspace[] = [
	{
		name: 'Test Workspace',
		description: 'Default test workspace for E2E tests'
	},
	{
		name: 'Development Workspace',
		description: 'Workspace for development testing'
	}
];

/**
 * Test Apps
 */
export const testApps: TestApp[] = [
	{
		name: 'Test CRM App',
		description: 'A customer relationship management system',
		status: 'completed'
	},
	{
		name: 'Test Project Manager',
		description: 'A project management tool',
		status: 'building'
	},
	{
		name: 'Test Todo App',
		description: 'A simple todo application',
		status: 'pending'
	}
];

/**
 * Test Templates
 */
export const testTemplates: TestTemplate[] = [
	{
		name: 'CRM Template',
		description: 'Customer relationship management system template',
		category: 'business'
	},
	{
		name: 'Project Management Template',
		description: 'Project management tool template',
		category: 'productivity'
	},
	{
		name: 'E-commerce Template',
		description: 'Online store template',
		category: 'business'
	}
];

/**
 * Test Chat Messages
 */
export const testChatMessages = {
	simple: 'Hello OSA!',
	appGeneration: 'Build a CRM app for managing customer relationships',
	question: 'What can you help me with?',
	complex: 'Create a project management system with task tracking, team collaboration, and deadline reminders'
};

/**
 * Test Agent Data
 */
export const testAgents = {
	custom: {
		name: 'Test Custom Agent',
		description: 'A custom agent for testing',
		systemPrompt: 'You are a helpful test agent.',
		temperature: 0.7
	}
};
